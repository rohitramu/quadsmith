package auth

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math"
	"net"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	connect "connectrpc.com/connect"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/protobuf/types/known/timestamppb"

	pb "quadsmith/shared/gen/quadsmith/api"
	"quadsmith/shared/gen/quadsmith/api/apiconnect"
	config "quadsmith/shared/gen/quadsmith/config"
	"quadsmith/shared/pkg/email"
)

// ---------------------------------------------------------
// AuthService
// ---------------------------------------------------------

type AuthConfig struct {
	Secret           string
	DevPrintOTP      bool
	SecureCookies    bool
	TrustedProxyHops int
	AccessTokenTTL   time.Duration
	RefreshTokenTTL  time.Duration
	OtpTTL           time.Duration
	MagicLinkTTL     time.Duration
	EmailCooldown    time.Duration
	Email            *config.EmailConfig
}

func FromProtoAuthConfig(p *config.AuthConfig) *AuthConfig {
	if p == nil {
		return nil
	}
	return &AuthConfig{
		Secret:           p.GetSecret(),
		DevPrintOTP:      p.GetDevPrintOtp(),
		SecureCookies:    p.GetSecureCookies(),
		TrustedProxyHops: int(p.GetTrustedProxyHops()),
		AccessTokenTTL:   p.GetAccessTokenTtl().AsDuration(),
		RefreshTokenTTL:  p.GetRefreshTokenTtl().AsDuration(),
		OtpTTL:           p.GetOtpTtl().AsDuration(),
		MagicLinkTTL:     p.GetMagicLinkTtl().AsDuration(),
		EmailCooldown:    p.GetEmailCooldown().AsDuration(),
		Email:            p.GetEmail(),
	}
}

type AuthService struct {
	apiconnect.UnimplementedAuthServiceHandler
	db          *pgxpool.Pool
	cfg         *AuthConfig
	emailSender email.EmailSender
}

func NewAuthService(db *pgxpool.Pool, protoCfg ...*config.AuthConfig) *AuthService {
	var authCfg *AuthConfig
	if len(protoCfg) > 0 && protoCfg[0] != nil {
		authCfg = FromProtoAuthConfig(protoCfg[0])
	} else {
		authCfg = &AuthConfig{
			Secret:           string(getMasterSecret()),
			DevPrintOTP:      os.Getenv("QS_DEV_PRINT_OTP") == "1" || os.Getenv("ENV") != "production",
			SecureCookies:    os.Getenv("ENV") == "production" || os.Getenv("SECURE_COOKIES") == "true",
			TrustedProxyHops: 0,
			AccessTokenTTL:   AccessTokenTTL,
			RefreshTokenTTL:  RefreshTokenTTL,
			OtpTTL:           OtpTTL,
			MagicLinkTTL:     MagicLinkTTL,
			EmailCooldown:    EmailCooldown,
		}
	}
	if authCfg.Secret != "" {
		SetMasterSecret(authCfg.Secret)
	}
	if authCfg.AccessTokenTTL > 0 || authCfg.RefreshTokenTTL > 0 {
		SetTokenTTLs(authCfg.AccessTokenTTL, authCfg.RefreshTokenTTL)
	}

	var sender email.EmailSender
	if authCfg.Email != nil {
		sender = email.NewEmailSender(authCfg.Email)
	} else {
		sender = email.NewConsoleEmailSender()
	}

	return &AuthService{
		db:          db,
		cfg:         authCfg,
		emailSender: sender,
	}
}

// SetEmailSender overrides the active EmailSender implementation (useful for tests).
func (s *AuthService) SetEmailSender(sender email.EmailSender) {
	s.emailSender = sender
}

func (s *AuthService) SendEmailOtp(ctx context.Context, req *connect.Request[pb.SendEmailOtpRequest]) (*connect.Response[pb.SendEmailOtpResponse], error) {
	email := strings.ToLower(strings.TrimSpace(req.Msg.GetEmail()))
	emailRegex := regexp.MustCompile(`^[a-z0-9._%+\-]+@[a-z0-9.\-]+\.[a-z]{2,}$`)
	if !emailRegex.MatchString(email) {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("please enter a valid email address"))
	}

	clientIP := getClientIP(req)
	if err := CheckIPRateLimit(clientIP); err != nil {
		return nil, connect.NewError(connect.CodeResourceExhausted, err)
	}

	cooldown, err := CheckEmailCooldown(ctx, s.db, email, s.cfg.EmailCooldown)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if cooldown > 0 {
		connErr := connect.NewError(connect.CodeResourceExhausted, fmt.Errorf("please wait %d seconds before requesting another code", cooldown))
		if detail, detailErr := connect.NewErrorDetail(&pb.OtpVerificationError{
			Message:           fmt.Sprintf("Please wait %d seconds before requesting another code", cooldown),
			RemainingAttempts: 0,
			MaxAttempts:       MaxOtpAttempts,
			IsExhausted:       false,
			CooldownSeconds:   cooldown,
		}); detailErr == nil {
			connErr.AddDetail(detail)
		}
		return nil, connErr
	}

	// Invalidate any existing active challenges for this email so previous codes cannot be used
	_, _ = s.db.Exec(ctx, `UPDATE email_otps SET used_at = NOW() WHERE email = $1 AND used_at IS NULL`, email)

	otpCode, codeHash, err := GenerateOTP()
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to generate otp: %w", err))
	}

	challengeToken, challengeHash, err := GenerateChallengeToken()
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to generate challenge token: %w", err))
	}

	otpUUID := NewUUIDv7()
	now := time.Now()
	expiresAt := now.Add(s.cfg.MagicLinkTTL)
	userAgent := req.Header().Get("User-Agent")
	locationHint := extractLocationHint(req.Header(), clientIP)

	_, err = s.db.Exec(ctx, `
		INSERT INTO email_otps (uuid, email, challenge_id_hash, code_hash, purpose, expires_at, created_at, failed_attempts, ip_address, user_agent, location_hint, status)
		VALUES ($1, $2, $3, $4, 'login', $5, $6, 0, $7, $8, $9, 'pending')`,
		otpUUID, email, challengeHash, codeHash, expiresAt, now, clientIP, userAgent, locationHint)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to save otp challenge: %w", err))
	}

	magicToken, err := SignMagicLinkToken(otpUUID, email, "login", "", expiresAt)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to generate magic link token: %w", err))
	}

	baseURL := s.getBaseURL(req)
	magicLinkURL := fmt.Sprintf("%s/login/verify?token=%s", baseURL, magicToken)

	ctxTimeout, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if s.emailSender != nil {
		if err := s.emailSender.SendVerification(ctxTimeout, email, otpCode, magicLinkURL, "login", s.cfg.OtpTTL); err != nil {
			return nil, connect.NewError(connect.CodeUnavailable, fmt.Errorf("unable to deliver verification email: %w", err))
		}
	} else if s.cfg.DevPrintOTP {
		fmt.Printf("\n=======================================================\n")
		fmt.Printf("[AUTH] Email OTP for %s: %s (valid for %v)\n", email, otpCode, s.cfg.OtpTTL)
		fmt.Printf("[AUTH] Challenge Token: %s\n", challengeToken)
		fmt.Printf("[AUTH] Magic Link: %s (valid for %v)\n", magicLinkURL, s.cfg.MagicLinkTTL)
		fmt.Printf("=======================================================\n\n")
	}

	exists, err := s.userExistsForEmail(ctx, email)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	res := connect.NewResponse(&pb.SendEmailOtpResponse{
		Success:           true,
		Message:           "Verification code sent to your email",
		CooldownSeconds:   int32(s.cfg.EmailCooldown.Seconds()),
		ChallengeId:       &challengeToken,
		IsExistingAccount: exists,
	})
	s.setChallengeCookie(res.Header(), challengeToken)
	return res, nil
}

func (s *AuthService) VerifyEmailOtp(ctx context.Context, req *connect.Request[pb.VerifyEmailOtpRequest]) (*connect.Response[pb.AuthResponse], error) {
	email := strings.ToLower(strings.TrimSpace(req.Msg.GetEmail()))
	code := strings.TrimSpace(req.Msg.GetCode())
	if email == "" || code == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("email and 6-digit code are required"))
	}

	clientIP := getClientIP(req)
	if err := CheckIPRateLimit(clientIP); err != nil {
		connErr := connect.NewError(connect.CodeResourceExhausted, err)
		if detail, detailErr := connect.NewErrorDetail(&pb.OtpVerificationError{
			Message:           err.Error(),
			RemainingAttempts: 0,
			MaxAttempts:       MaxOtpAttempts,
			IsExhausted:       true,
			CooldownSeconds:   int32(IPLockoutDuration.Seconds()),
		}); detailErr == nil {
			connErr.AddDetail(detail)
		}
		return nil, connErr
	}

	challengeID := extractChallengeID(req.Msg.GetChallengeId(), req.Header())
	if challengeID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("challenge session expired or missing. Please request a new verification code"))
	}
	challengeHash := HashToken(challengeID)

	// Atomic attempt reservation in PostgreSQL (eliminates concurrency race conditions)
	var otpUUID string
	var codeHash string
	var failedAttempts int32
	var createdAt time.Time

	err := s.db.QueryRow(ctx, `
		UPDATE email_otps
		SET failed_attempts = failed_attempts + 1
		WHERE email = $1 AND challenge_id_hash = $2 AND purpose = 'login' AND used_at IS NULL AND failed_attempts < $3
		RETURNING uuid, code_hash, failed_attempts, created_at`,
		email, challengeHash, MaxOtpAttempts).Scan(&otpUUID, &codeHash, &failedAttempts, &createdAt)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// Query existing challenge row to return helpful status
			var existingUUID string
			var existingFailures int32
			var existingUsedAt *time.Time
			checkErr := s.db.QueryRow(ctx, `
				SELECT uuid, failed_attempts, used_at
				FROM email_otps
				WHERE email = $1 AND challenge_id_hash = $2
				ORDER BY created_at DESC
				LIMIT 1`, email, challengeHash).Scan(&existingUUID, &existingFailures, &existingUsedAt)

			if checkErr != nil {
				return nil, connect.NewError(connect.CodeNotFound, errors.New("no active verification challenge found. Please request a new code"))
			}

			if existingUsedAt != nil {
				return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("this verification challenge has already been used or expired. Please request a new code"))
			}

			if existingFailures >= MaxOtpAttempts {
				_, _ = s.db.Exec(ctx, `UPDATE email_otps SET used_at = NOW() WHERE uuid = $1`, existingUUID)
				if err := CheckIPRateLimit(clientIP); err != nil {
					connErr := connect.NewError(connect.CodeResourceExhausted, err)
					if detail, detailErr := connect.NewErrorDetail(&pb.OtpVerificationError{
						Message:           err.Error(),
						RemainingAttempts: 0,
						MaxAttempts:       MaxOtpAttempts,
						IsExhausted:       true,
						CooldownSeconds:   int32(IPLockoutDuration.Seconds()),
					}); detailErr == nil {
						connErr.AddDetail(detail)
					}
					return nil, connErr
				}
				return s.returnOtpError("Verification challenge exhausted. Please request a new code after the cooldown period.", 0, true, int32(ExhaustedCooldown.Seconds()))
			}

			return nil, connect.NewError(connect.CodeNotFound, errors.New("no active verification challenge found. Please request a new code"))
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	// Verify OTP lifespan
	if time.Since(createdAt) > s.cfg.OtpTTL {
		_, _ = s.db.Exec(ctx, `UPDATE email_otps SET used_at = NOW() WHERE uuid = $1`, otpUUID)
		return nil, connect.NewError(connect.CodeDeadlineExceeded, fmt.Errorf("verification code has expired (%v limit). Please request a new code", s.cfg.OtpTTL))
	}

	if !VerifyOTP(code, codeHash) {
		remaining := int32(MaxOtpAttempts - failedAttempts)
		if remaining <= 0 {
			_, _ = s.db.Exec(ctx, `UPDATE email_otps SET used_at = NOW() WHERE uuid = $1`, otpUUID)
			RecordIPFailure(clientIP)
			if err := CheckIPRateLimit(clientIP); err != nil {
				connErr := connect.NewError(connect.CodeResourceExhausted, err)
				if detail, detailErr := connect.NewErrorDetail(&pb.OtpVerificationError{
					Message:           err.Error(),
					RemainingAttempts: 0,
					MaxAttempts:       MaxOtpAttempts,
					IsExhausted:       true,
					CooldownSeconds:   int32(IPLockoutDuration.Seconds()),
				}); detailErr == nil {
					connErr.AddDetail(detail)
				}
				return nil, connErr
			}
			return s.returnOtpError("Incorrect code. You have 0 attempts left.\nPlease request a new code after the cooldown period.", 0, true, int32(ExhaustedCooldown.Seconds()))
		}
		msg := fmt.Sprintf("Incorrect code. You have %d %s left", remaining, pluralAttempts(remaining))
		return s.returnOtpError(msg, remaining, false, 0)
	}

	// Verification succeeded: mark challenge consumed
	RecordIPSuccess(clientIP)
	if err := CheckAndRecordIPSignIn(clientIP, email); err != nil {
		return nil, connect.NewError(connect.CodeResourceExhausted, err)
	}
	_, _ = s.db.Exec(ctx, `UPDATE email_otps SET used_at = NOW() WHERE uuid = $1`, otpUUID)

	exists, err := s.userExistsForEmail(ctx, email)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if !exists {
		setupToken, err := SignSetupToken(email)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to sign setup token: %w", err))
		}
		return connect.NewResponse(&pb.AuthResponse{
			AccountSetupRequired: true,
			SetupToken:           setupToken,
		}), nil
	}

	return s.issueAuthSession(ctx, email, req.Header().Get("User-Agent"))
}

func (s *AuthService) VerifyMagicLink(ctx context.Context, req *connect.Request[pb.VerifyMagicLinkRequest]) (*connect.Response[pb.AuthResponse], error) {
	token := strings.TrimSpace(req.Msg.GetToken())
	if token == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("magic link token required"))
	}

	claims, err := VerifyMagicLinkToken(token)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid or expired magic link: %w", err))
	}

	var createdAt time.Time
	var usedAt *time.Time
	var purpose string
	err = s.db.QueryRow(ctx, `SELECT created_at, used_at, purpose FROM email_otps WHERE uuid = $1`, claims.OtpUUID).Scan(&createdAt, &usedAt, &purpose)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, connect.NewError(connect.CodeNotFound, errors.New("magic link challenge not found"))
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	if usedAt != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("this magic link has already been used. Please request a new code"))
	}

	if claims.Purpose != "" && purpose != claims.Purpose {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("invalid magic link purpose"))
	}

	if time.Since(createdAt) > s.cfg.MagicLinkTTL {
		_, _ = s.db.Exec(ctx, `UPDATE email_otps SET used_at = NOW() WHERE uuid = $1`, claims.OtpUUID)
		return nil, connect.NewError(connect.CodeDeadlineExceeded, errors.New("magic link has expired. Please request a new code"))
	}

	_, _ = s.db.Exec(ctx, `UPDATE email_otps SET used_at = NOW() WHERE uuid = $1`, claims.OtpUUID)

	clientIP := getClientIP(req)
	RecordIPSuccess(clientIP)
	if err := CheckAndRecordIPSignIn(clientIP, claims.Email); err != nil {
		return nil, connect.NewError(connect.CodeResourceExhausted, err)
	}

	email := claims.Email
	exists, err := s.userExistsForEmail(ctx, email)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if !exists {
		setupToken, err := SignSetupToken(email)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to sign setup token: %w", err))
		}
		return connect.NewResponse(&pb.AuthResponse{
			AccountSetupRequired: true,
			SetupToken:           setupToken,
		}), nil
	}

	return s.issueAuthSession(ctx, email, req.Header().Get("User-Agent"))
}

func (s *AuthService) GetLoginChallengeDetails(ctx context.Context, req *connect.Request[pb.GetLoginChallengeDetailsRequest]) (*connect.Response[pb.GetLoginChallengeDetailsResponse], error) {
	token := strings.TrimSpace(req.Msg.GetToken())
	if token == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("magic link token required"))
	}

	claims, err := VerifyMagicLinkToken(token)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid or expired magic link: %w", err))
	}

	var email, ipAddress, userAgent, locationHint, statusStr string
	var createdAt, expiresAt time.Time
	var usedAt *time.Time

	err = s.db.QueryRow(ctx, `
		SELECT email, ip_address, user_agent, location_hint, status, created_at, expires_at, used_at
		FROM email_otps
		WHERE uuid = $1`, claims.OtpUUID).Scan(&email, &ipAddress, &userAgent, &locationHint, &statusStr, &createdAt, &expiresAt, &usedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, connect.NewError(connect.CodeNotFound, errors.New("login challenge not found"))
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	status := pb.LoginChallengeStatus_LOGIN_CHALLENGE_STATUS_PENDING
	if usedAt != nil || time.Now().After(expiresAt) {
		status = pb.LoginChallengeStatus_LOGIN_CHALLENGE_STATUS_EXPIRED
	} else {
		switch statusStr {
		case "approved":
			status = pb.LoginChallengeStatus_LOGIN_CHALLENGE_STATUS_APPROVED
		case "denied":
			status = pb.LoginChallengeStatus_LOGIN_CHALLENGE_STATUS_DENIED
		default:
			status = pb.LoginChallengeStatus_LOGIN_CHALLENGE_STATUS_PENDING
		}
	}

	deviceHint := parseDeviceHint(userAgent)

	return connect.NewResponse(&pb.GetLoginChallengeDetailsResponse{
		Email:        email,
		IpAddress:    ipAddress,
		LocationHint: locationHint,
		DeviceHint:   deviceHint,
		CreatedAt:    timestamppb.New(createdAt),
		ExpiresAt:    timestamppb.New(expiresAt),
		Status:       status,
	}), nil
}

func (s *AuthService) ApproveLoginChallenge(ctx context.Context, req *connect.Request[pb.ApproveLoginChallengeRequest]) (*connect.Response[pb.ApproveLoginChallengeResponse], error) {
	token := strings.TrimSpace(req.Msg.GetToken())
	if token == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("magic link token required"))
	}

	claims, err := VerifyMagicLinkToken(token)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid or expired magic link: %w", err))
	}

	tag, err := s.db.Exec(ctx, `
		UPDATE email_otps
		SET status = 'approved', approved_at = NOW()
		WHERE uuid = $1 AND status = 'pending' AND used_at IS NULL AND expires_at > NOW()`, claims.OtpUUID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to approve login: %w", err))
	}
	if tag.RowsAffected() == 0 {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("login challenge is no longer valid, expired, or already answered"))
	}

	return connect.NewResponse(&pb.ApproveLoginChallengeResponse{
		Success: true,
		Message: "Login approved",
	}), nil
}

func (s *AuthService) DenyLoginChallenge(ctx context.Context, req *connect.Request[pb.DenyLoginChallengeRequest]) (*connect.Response[pb.DenyLoginChallengeResponse], error) {
	token := strings.TrimSpace(req.Msg.GetToken())
	if token == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("magic link token required"))
	}

	claims, err := VerifyMagicLinkToken(token)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid or expired magic link: %w", err))
	}

	tag, err := s.db.Exec(ctx, `
		UPDATE email_otps
		SET status = 'denied', denied_at = NOW()
		WHERE uuid = $1 AND status = 'pending' AND used_at IS NULL`, claims.OtpUUID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to deny login: %w", err))
	}
	if tag.RowsAffected() == 0 {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("login challenge is no longer valid or already answered"))
	}

	return connect.NewResponse(&pb.DenyLoginChallengeResponse{
		Success: true,
		Message: "Login request denied",
	}), nil
}

func (s *AuthService) GetLoginChallengeStatus(ctx context.Context, req *connect.Request[pb.GetLoginChallengeStatusRequest]) (*connect.Response[pb.GetLoginChallengeStatusResponse], error) {
	challengeID := extractChallengeID(req.Msg.GetChallengeId(), req.Header())
	if challengeID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("challenge session expired or missing"))
	}
	challengeHash := HashToken(challengeID)

	var otpUUID, email, statusStr string
	var expiresAt time.Time
	var usedAt *time.Time

	err := s.db.QueryRow(ctx, `
		SELECT uuid, email, status, expires_at, used_at
		FROM email_otps
		WHERE challenge_id_hash = $1 AND purpose = 'login'
		ORDER BY created_at DESC
		LIMIT 1`, challengeHash).Scan(&otpUUID, &email, &statusStr, &expiresAt, &usedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return connect.NewResponse(&pb.GetLoginChallengeStatusResponse{
				Status: pb.LoginChallengeStatus_LOGIN_CHALLENGE_STATUS_EXPIRED,
			}), nil
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	if time.Now().After(expiresAt) {
		return connect.NewResponse(&pb.GetLoginChallengeStatusResponse{
			Status: pb.LoginChallengeStatus_LOGIN_CHALLENGE_STATUS_EXPIRED,
		}), nil
	}

	switch statusStr {
	case "denied":
		return connect.NewResponse(&pb.GetLoginChallengeStatusResponse{
			Status: pb.LoginChallengeStatus_LOGIN_CHALLENGE_STATUS_DENIED,
		}), nil
	case "approved":
		tag, err := s.db.Exec(ctx, `UPDATE email_otps SET used_at = NOW() WHERE uuid = $1 AND used_at IS NULL`, otpUUID)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		if tag.RowsAffected() == 0 {
			// Already consumed by another request
			return connect.NewResponse(&pb.GetLoginChallengeStatusResponse{
				Status: pb.LoginChallengeStatus_LOGIN_CHALLENGE_STATUS_EXPIRED,
			}), nil
		}

		clientIP := getClientIP(req)
		RecordIPSuccess(clientIP)
		if err := CheckAndRecordIPSignIn(clientIP, email); err != nil {
			return nil, connect.NewError(connect.CodeResourceExhausted, err)
		}

		exists, err := s.userExistsForEmail(ctx, email)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		if !exists {
			setupToken, err := SignSetupToken(email)
			if err != nil {
				return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to sign setup token: %w", err))
			}
			return connect.NewResponse(&pb.GetLoginChallengeStatusResponse{
				Status: pb.LoginChallengeStatus_LOGIN_CHALLENGE_STATUS_APPROVED,
				AuthResponse: &pb.AuthResponse{
					AccountSetupRequired: true,
					SetupToken:           setupToken,
				},
			}), nil
		}

		authRes, err := s.issueAuthSession(ctx, email, req.Header().Get("User-Agent"))
		if err != nil {
			return nil, err
		}

		res := connect.NewResponse(&pb.GetLoginChallengeStatusResponse{
			Status:       pb.LoginChallengeStatus_LOGIN_CHALLENGE_STATUS_APPROVED,
			AuthResponse: authRes.Msg,
		})
		for _, cookie := range authRes.Header().Values("Set-Cookie") {
			res.Header().Add("Set-Cookie", cookie)
		}
		return res, nil

	default:
		return connect.NewResponse(&pb.GetLoginChallengeStatusResponse{
			Status: pb.LoginChallengeStatus_LOGIN_CHALLENGE_STATUS_PENDING,
		}), nil
	}
}

func (s *AuthService) CompleteAccountSetup(ctx context.Context, req *connect.Request[pb.CompleteAccountSetupRequest]) (*connect.Response[pb.AuthResponse], error) {
	setupToken := strings.TrimSpace(req.Msg.GetSetupToken())
	if setupToken == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("setup token required"))
	}
	email, err := VerifySetupToken(setupToken)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("invalid or expired setup token: %w", err))
	}

	username := strings.ToLower(strings.TrimSpace(req.Msg.GetUsername()))
	if IsReservedUsername(username) {
		return nil, connect.NewError(connect.CodeAlreadyExists, errors.New("This username is already taken"))
	}
	if err := ValidateUsername(username); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	// Check if active user already has this username (or is in 30-day deletion grace period)
	var activeOwnerUUID string
	err = s.db.QueryRow(ctx, `SELECT uuid FROM users WHERE LOWER(id) = $1 AND (deleted_at IS NULL OR deleted_at + INTERVAL '30 days' > NOW())`, username).Scan(&activeOwnerUUID)
	if err == nil {
		return nil, connect.NewError(connect.CodeAlreadyExists, errors.New("This username is already taken"))
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	// Check if held in username_histories within 30-day window
	var historyOwnerUUID string
	err = s.db.QueryRow(ctx, `
		SELECT user_uuid
		FROM username_histories
		WHERE LOWER(old_username) = $1 AND COALESCE(first_archived_at, renamed_at) + INTERVAL '30 days' > NOW()
		ORDER BY COALESCE(first_archived_at, renamed_at) DESC
		LIMIT 1`, username).Scan(&historyOwnerUUID)
	if err == nil {
		return nil, connect.NewError(connect.CodeAlreadyExists, errors.New("This username is already taken"))
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	// Check if user already exists for this email
	exists, err := s.userExistsForEmail(ctx, email)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if exists {
		return nil, connect.NewError(connect.CodeAlreadyExists, errors.New("an account with this email already exists"))
	}

	userUUID := NewUUIDv7()
	now := time.Now()
	role := "USER"

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, `
		INSERT INTO users (uuid, id, email, role, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $5)`,
		userUUID, username, email, role, now)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to create user: %w", err))
	}

	profileUUID := NewUUIDv7()
	avatarURL := strings.TrimSpace(req.Msg.GetAvatarUrl())
	_, err = tx.Exec(ctx, `
		INSERT INTO user_profiles (uuid, user_uuid, avatar_url, bio, updated_at)
		VALUES ($1, $2, $3, '', $4)`,
		profileUUID, userUUID, avatarURL, now)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to create profile: %w", err))
	}

	emailUUID := NewUUIDv7()
	_, err = tx.Exec(ctx, `
		INSERT INTO user_emails (uuid, user_uuid, email, is_primary, verified_via_otp, created_at)
		VALUES ($1, $2, $3, true, true, $4)`,
		emailUUID, userUUID, email, now)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to create user email: %w", err))
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to commit account creation: %w", err))
	}

	clientIP := getClientIP(req)
	if err := CheckAndRecordIPSignIn(clientIP, email); err != nil {
		return nil, connect.NewError(connect.CodeResourceExhausted, err)
	}

	return s.issueAuthSession(ctx, email, req.Header().Get("User-Agent"))
}

func (s *AuthService) userExistsForEmail(ctx context.Context, email string) (bool, error) {
	var count int
	err := s.db.QueryRow(ctx, `
		SELECT COUNT(*) FROM (
			SELECT 1 FROM user_emails ue
			JOIN users u ON u.uuid = ue.user_uuid
			WHERE LOWER(ue.email) = LOWER($1) AND (u.deleted_at IS NULL OR u.deleted_at + INTERVAL '30 days' > NOW())
			UNION
			SELECT 1 FROM users u
			WHERE LOWER(u.email) = LOWER($1) AND (u.deleted_at IS NULL OR u.deleted_at + INTERVAL '30 days' > NOW())
		) sub`, email).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (s *AuthService) RefreshToken(ctx context.Context, req *connect.Request[pb.RefreshTokenRequest]) (*connect.Response[pb.AuthResponse], error) {
	token := req.Msg.GetRefreshToken()
	if token == "" {
		if cookieHeader := req.Header().Get("Cookie"); cookieHeader != "" {
			header := http.Header{}
			header.Add("Cookie", cookieHeader)
			reqHTTP := &http.Request{Header: header}
			if c, err := reqHTTP.Cookie("qs_session"); err == nil && c.Value != "" {
				token = c.Value
			}
		}
	}
	if token == "" {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("no refresh token provided"))
	}

	tokenHash := HashToken(token)
	var sessionUUID, userUUID string
	var expiresAt time.Time

	err := s.db.QueryRow(ctx, `
		SELECT uuid, user_uuid, expires_at
		FROM refresh_tokens
		WHERE token_hash = $1`, tokenHash).Scan(&sessionUUID, &userUUID, &expiresAt)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("invalid or expired session"))
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	if time.Now().After(expiresAt) {
		_, _ = s.db.Exec(ctx, `DELETE FROM refresh_tokens WHERE uuid = $1`, sessionUUID)
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("session has expired. Please sign in again"))
	}

	// sliding window: roll expiration forward to NOW() + RefreshTokenTTL
	newExpiresAt := time.Now().Add(s.cfg.RefreshTokenTTL)
	_, err = s.db.Exec(ctx, `
		UPDATE refresh_tokens
		SET expires_at = $1, last_used_at = NOW()
		WHERE uuid = $2`, newExpiresAt, sessionUUID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	// Fetch User
	var user pb.User
	var username, email, role string
	var lastRenamedAt, deletedAt, bannedAt, createdAt, updatedAt *time.Time
	err = s.db.QueryRow(ctx, `
		SELECT uuid, id, email, role, last_renamed_at, deleted_at, banned_at, created_at, updated_at
		FROM users WHERE uuid = $1`, userUUID).Scan(
		&user.Uuid, &username, &email, &role,
		&lastRenamedAt, &deletedAt, &bannedAt, &createdAt, &updatedAt)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	user.Id = username
	user.Email = email
	user.Role = role
	if createdAt != nil {
		user.CreatedAt = timestamppb.New(*createdAt)
	}
	if updatedAt != nil {
		user.UpdatedAt = timestamppb.New(*updatedAt)
	}

	// Fetch Profile
	var profile pb.UserProfile
	var profileUpdatedAt *time.Time
	_ = s.db.QueryRow(ctx, `
		SELECT uuid, user_uuid, avatar_url, bio, updated_at
		FROM user_profiles WHERE user_uuid = $1`, userUUID).Scan(
		&profile.Uuid, &profile.UserUuid, &profile.AvatarUrl, &profile.Bio, &profileUpdatedAt)
	if profileUpdatedAt != nil {
		profile.UpdatedAt = timestamppb.New(*profileUpdatedAt)
	}

	accessToken, err := SignAccessToken(userUUID, username, email, role, sessionUUID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	res := connect.NewResponse(&pb.AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: token,
		User:         &user,
		Profile:      &profile,
	})

	cookie := &http.Cookie{
		Name:     "qs_session",
		Value:    token,
		Path:     "/",
		MaxAge:   int(s.cfg.RefreshTokenTTL.Seconds()),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   s.cfg.SecureCookies,
	}
	res.Header().Set("Set-Cookie", cookie.String())

	return res, nil
}

func (s *AuthService) GetCurrentUser(ctx context.Context, _ *connect.Request[pb.GetCurrentUserRequest]) (*connect.Response[pb.User], error) {
	userCtx := UserContextFromContext(ctx)
	if userCtx == nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("authentication required"))
	}

	var user pb.User
	var lastRenamedAt, deletedAt, bannedAt, createdAt, updatedAt *time.Time
	err := s.db.QueryRow(ctx, `
		SELECT uuid, id, email, role, last_renamed_at, deleted_at, banned_at, created_at, updated_at
		FROM users WHERE uuid = $1`, userCtx.UserUUID).Scan(
		&user.Uuid, &user.Id, &user.Email, &user.Role,
		&lastRenamedAt, &deletedAt, &bannedAt, &createdAt, &updatedAt)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("user not found"))
	}
	if lastRenamedAt != nil {
		user.LastRenamedAt = timestamppb.New(*lastRenamedAt)
	}
	if deletedAt != nil {
		user.DeletedAt = timestamppb.New(*deletedAt)
	}
	if createdAt != nil {
		user.CreatedAt = timestamppb.New(*createdAt)
	}
	if updatedAt != nil {
		user.UpdatedAt = timestamppb.New(*updatedAt)
	}

	return connect.NewResponse(&user), nil
}

func (s *AuthService) UpdateProfile(ctx context.Context, req *connect.Request[pb.UpdateProfileRequest]) (*connect.Response[pb.UserProfile], error) {
	userCtx := UserContextFromContext(ctx)
	if userCtx == nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("authentication required"))
	}

	// Ensure profile exists
	var profileUUID string
	err := s.db.QueryRow(ctx, `SELECT uuid FROM user_profiles WHERE user_uuid = $1`, userCtx.UserUUID).Scan(&profileUUID)
	if errors.Is(err, pgx.ErrNoRows) {
		profileUUID = NewUUIDv7()
		_, err = s.db.Exec(ctx, `
			INSERT INTO user_profiles (uuid, user_uuid, avatar_url, bio, updated_at)
			VALUES ($1, $2, '', '', NOW())`,
			profileUUID, userCtx.UserUUID)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
	}

	if req.Msg.AvatarUrl != nil {
		_, _ = s.db.Exec(ctx, `UPDATE user_profiles SET avatar_url = $1, updated_at = NOW() WHERE user_uuid = $2`, *req.Msg.AvatarUrl, userCtx.UserUUID)
	}
	if req.Msg.Bio != nil {
		_, _ = s.db.Exec(ctx, `UPDATE user_profiles SET bio = $1, updated_at = NOW() WHERE user_uuid = $2`, *req.Msg.Bio, userCtx.UserUUID)
	}

	var profile pb.UserProfile
	var updatedAt *time.Time
	err = s.db.QueryRow(ctx, `
		SELECT uuid, user_uuid, avatar_url, bio, updated_at
		FROM user_profiles WHERE user_uuid = $1`, userCtx.UserUUID).Scan(
		&profile.Uuid, &profile.UserUuid, &profile.AvatarUrl, &profile.Bio, &updatedAt)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if updatedAt != nil {
		profile.UpdatedAt = timestamppb.New(*updatedAt)
	}

	return connect.NewResponse(&profile), nil
}

func (s *AuthService) Logout(ctx context.Context, req *connect.Request[pb.LogoutRequest]) (*connect.Response[pb.LogoutResponse], error) {
	token := req.Msg.GetRefreshToken()
	if token != "" {
		_, _ = s.db.Exec(ctx, `DELETE FROM refresh_tokens WHERE token_hash = $1`, HashToken(token))
	} else if userCtx := UserContextFromContext(ctx); userCtx != nil && userCtx.SessionUUID != "" {
		_, _ = s.db.Exec(ctx, `DELETE FROM refresh_tokens WHERE uuid = $1`, userCtx.SessionUUID)
	}

	res := connect.NewResponse(&pb.LogoutResponse{Success: true})
	clearCookie := &http.Cookie{
		Name:     "qs_session",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	}
	res.Header().Set("Set-Cookie", clearCookie.String())

	return res, nil
}

func (s *AuthService) ListUserEmails(ctx context.Context, req *connect.Request[pb.ListUserEmailsRequest]) (*connect.Response[pb.ListUserEmailsResponse], error) {
	userCtx := UserContextFromContext(ctx)
	if userCtx == nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("authentication required"))
	}

	// Self-healing backfill for user's primary email
	if userCtx.Email != "" {
		emailUUID := NewUUIDv7()
		_, _ = s.db.Exec(ctx, `
			INSERT INTO user_emails (uuid, user_uuid, email, is_primary, verified_via_otp, created_at)
			VALUES ($1, $2, $3, true, true, NOW())
			ON CONFLICT (email) DO NOTHING`,
			emailUUID, userCtx.UserUUID, userCtx.Email)
	}

	rows, err := s.db.Query(ctx, `
		SELECT ue.uuid, ue.user_uuid, ue.email, ue.is_primary, ue.created_at, ue.verified_via_otp,
		       COALESCE(ARRAY_AGG(DISTINCT oa.provider) FILTER (WHERE oa.provider IS NOT NULL), '{}') AS oauth_providers
		FROM user_emails ue
		LEFT JOIN user_email_oauth_accounts uea ON ue.uuid = uea.user_email_uuid
		LEFT JOIN oauth_accounts oa ON uea.oauth_account_uuid = oa.uuid
		WHERE ue.user_uuid = $1
		GROUP BY ue.uuid, ue.user_uuid, ue.email, ue.is_primary, ue.created_at, ue.verified_via_otp
		ORDER BY ue.is_primary DESC, ue.created_at ASC`, userCtx.UserUUID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to list user emails: %w", err))
	}
	defer rows.Close()

	var emails []*pb.UserEmail
	for rows.Next() {
		var em pb.UserEmail
		var createdAt *time.Time
		var providers []string
		if err := rows.Scan(&em.Uuid, &em.UserUuid, &em.Email, &em.IsPrimary, &createdAt, &em.VerifiedViaOtp, &providers); err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to scan email: %w", err))
		}
		if createdAt != nil {
			em.CreatedAt = timestamppb.New(*createdAt)
		}
		em.OauthProviders = providers
		emails = append(emails, &em)
	}

	return connect.NewResponse(&pb.ListUserEmailsResponse{Emails: emails}), nil
}

func (s *AuthService) SendAddEmailOtp(ctx context.Context, req *connect.Request[pb.SendAddEmailOtpRequest]) (*connect.Response[pb.SendEmailOtpResponse], error) {
	userCtx := UserContextFromContext(ctx)
	if userCtx == nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("authentication required"))
	}

	email := strings.ToLower(strings.TrimSpace(req.Msg.GetEmail()))
	if email == "" || !strings.Contains(email, "@") {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("valid email address required"))
	}

	// Check if already associated with CURRENT user
	var alreadyAssociated bool
	err := s.db.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM user_emails
			WHERE user_uuid = $1 AND LOWER(email) = LOWER($2)
		)`, userCtx.UserUUID, email).Scan(&alreadyAssociated)
	if err == nil && alreadyAssociated {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("this email address is already associated with your account"))
	}

	// Rate limit per IP
	clientIP := getClientIP(req)
	if err := CheckIPRateLimit(clientIP); err != nil {
		return nil, connect.NewError(connect.CodeResourceExhausted, err)
	}

	cooldown, err := CheckEmailCooldown(ctx, s.db, email)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if cooldown > 0 {
		connErr := connect.NewError(connect.CodeResourceExhausted, fmt.Errorf("please wait %d seconds before requesting another code", cooldown))
		if detail, detailErr := connect.NewErrorDetail(&pb.OtpVerificationError{
			Message:           fmt.Sprintf("Please wait %d seconds before requesting another code", cooldown),
			RemainingAttempts: 0,
			MaxAttempts:       MaxOtpAttempts,
			IsExhausted:       false,
			CooldownSeconds:   cooldown,
		}); detailErr == nil {
			connErr.AddDetail(detail)
		}
		return nil, connErr
	}

	// Invalidate previous unconsumed OTP challenges for this email
	_, _ = s.db.Exec(ctx, `UPDATE email_otps SET used_at = NOW() WHERE email = $1 AND used_at IS NULL`, email)

	otpCode, codeHash, err := GenerateOTP()
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to generate otp: %w", err))
	}

	challengeToken, challengeHash, err := GenerateChallengeToken()
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to generate challenge token: %w", err))
	}

	otpUUID := NewUUIDv7()
	now := time.Now()
	expiresAt := now.Add(s.cfg.MagicLinkTTL)

	_, err = s.db.Exec(ctx, `
		INSERT INTO email_otps (uuid, email, challenge_id_hash, code_hash, purpose, requested_by_user_uuid, expires_at, created_at, failed_attempts)
		VALUES ($1, $2, $3, $4, 'add_email', $5, $6, $7, 0)`,
		otpUUID, email, challengeHash, codeHash, userCtx.UserUUID, expiresAt, now)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to save otp challenge: %w", err))
	}

	magicToken, err := SignMagicLinkToken(otpUUID, email, "add_email", userCtx.UserUUID, expiresAt)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to generate magic link token: %w", err))
	}

	baseURL := s.getBaseURL(req)
	magicLinkURL := fmt.Sprintf("%s/account?add_email_token=%s", baseURL, magicToken)

	ctxTimeout, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if s.emailSender != nil {
		if err := s.emailSender.SendVerification(ctxTimeout, email, otpCode, magicLinkURL, "add_email", s.cfg.OtpTTL); err != nil {
			return nil, connect.NewError(connect.CodeUnavailable, fmt.Errorf("unable to deliver verification email: %w", err))
		}
	} else if s.cfg.DevPrintOTP {
		fmt.Printf("\n=======================================================\n")
		fmt.Printf("[AUTH] Add Email OTP for %s: %s (valid for %v)\n", email, otpCode, s.cfg.OtpTTL)
		fmt.Printf("[AUTH] Challenge Token: %s\n", challengeToken)
		fmt.Printf("[AUTH] Add Email Magic Link: %s (valid for %v)\n", magicLinkURL, s.cfg.MagicLinkTTL)
		fmt.Printf("=======================================================\n\n")
	}

	res := connect.NewResponse(&pb.SendEmailOtpResponse{
		Success:           true,
		Message:           "Verification code sent to email",
		CooldownSeconds:   int32(s.cfg.EmailCooldown.Seconds()),
		IsExistingAccount: false,
		ChallengeId:       &challengeToken,
	})
	s.setChallengeCookie(res.Header(), challengeToken)
	return res, nil
}

func (s *AuthService) VerifyAddEmailOtp(ctx context.Context, req *connect.Request[pb.VerifyAddEmailOtpRequest]) (*connect.Response[pb.VerifyAddEmailOtpResponse], error) {
	userCtx := UserContextFromContext(ctx)
	if userCtx == nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("authentication required"))
	}

	email := strings.ToLower(strings.TrimSpace(req.Msg.GetEmail()))
	code := strings.TrimSpace(req.Msg.GetCode())
	token := strings.TrimSpace(req.Msg.GetToken())

	clientIP := getClientIP(req)
	if err := CheckIPRateLimit(clientIP); err != nil {
		return nil, connect.NewError(connect.CodeResourceExhausted, err)
	}

	var otpUUID string
	if token != "" {
		claims, err := VerifyMagicLinkToken(token)
		if err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid or expired magic link: %w", err))
		}
		if claims.Purpose != "add_email" {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("invalid magic link purpose"))
		}
		if claims.RequesterUserUUID != userCtx.UserUUID {
			return nil, connect.NewError(connect.CodePermissionDenied, errors.New("this magic link was issued to a different account session"))
		}
		if email == "" {
			email = claims.Email
		} else if strings.ToLower(email) != strings.ToLower(claims.Email) {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("magic link token email mismatch"))
		}
		otpUUID = claims.OtpUUID

		var createdAt time.Time
		var usedAt *time.Time
		err = s.db.QueryRow(ctx, `SELECT created_at, used_at FROM email_otps WHERE uuid = $1`, otpUUID).Scan(&createdAt, &usedAt)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil, connect.NewError(connect.CodeNotFound, errors.New("magic link challenge not found"))
			}
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		if usedAt != nil {
			return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("this magic link has already been used"))
		}
		if time.Since(createdAt) > s.cfg.MagicLinkTTL {
			_, _ = s.db.Exec(ctx, `UPDATE email_otps SET used_at = NOW() WHERE uuid = $1`, otpUUID)
			return nil, connect.NewError(connect.CodeDeadlineExceeded, errors.New("magic link has expired"))
		}
	} else {
		if email == "" || code == "" {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("email and 6-digit code (or magic link token) are required"))
		}

		challengeID := extractChallengeID(req.Msg.GetChallengeId(), req.Header())
		if challengeID == "" {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("challenge session expired or missing. Please request a new verification code"))
		}
		challengeHash := HashToken(challengeID)

		var codeHash string
		var failedAttempts int32
		var createdAt time.Time

		err := s.db.QueryRow(ctx, `
			UPDATE email_otps
			SET failed_attempts = failed_attempts + 1
			WHERE email = $1 AND challenge_id_hash = $2 AND purpose = 'add_email' AND requested_by_user_uuid = $3 AND used_at IS NULL AND failed_attempts < $4
			RETURNING uuid, code_hash, failed_attempts, created_at`,
			email, challengeHash, userCtx.UserUUID, MaxOtpAttempts).Scan(&otpUUID, &codeHash, &failedAttempts, &createdAt)

		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil, connect.NewError(connect.CodeNotFound, errors.New("no active verification challenge found. Please request a new code"))
			}
			return nil, connect.NewError(connect.CodeInternal, err)
		}

		if time.Since(createdAt) > s.cfg.OtpTTL {
			_, _ = s.db.Exec(ctx, `UPDATE email_otps SET used_at = NOW() WHERE uuid = $1`, otpUUID)
			return nil, connect.NewError(connect.CodeDeadlineExceeded, errors.New("verification code has expired"))
		}

		if !VerifyOTP(code, codeHash) {
			remaining := int32(MaxOtpAttempts - failedAttempts)
			if remaining <= 0 {
				_, _ = s.db.Exec(ctx, `UPDATE email_otps SET used_at = NOW() WHERE uuid = $1`, otpUUID)
				RecordIPFailure(clientIP)
				if err := CheckIPRateLimit(clientIP); err != nil {
					return nil, connect.NewError(connect.CodeResourceExhausted, err)
				}
				return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("incorrect code. You have 0 attempts left.\nPlease request a new code after the cooldown period."))
			}
			return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("incorrect code. %d %s left", remaining, pluralAttempts(remaining)))
		}
	}

	RecordIPSuccess(clientIP)
	_, _ = s.db.Exec(ctx, `UPDATE email_otps SET used_at = NOW() WHERE uuid = $1`, otpUUID)

	// Check if this email is registered to ANOTHER account
	var otherUserUUID, otherUsername string
	err := s.db.QueryRow(ctx, `
		SELECT u.uuid, u.id
		FROM users u
		WHERE LOWER(u.email) = LOWER($1) AND u.uuid != $2 AND u.deleted_at IS NULL
		UNION
		SELECT u.uuid, u.id
		FROM user_emails ue
		JOIN users u ON u.uuid = ue.user_uuid
		WHERE LOWER(ue.email) = LOWER($1) AND u.uuid != $2 AND u.deleted_at IS NULL
		LIMIT 1`, email, userCtx.UserUUID).Scan(&otherUserUUID, &otherUsername)

	if err == nil && otherUserUUID != "" {
		// Another account was found! Issue signed merge ticket
		mergeTicket, err := SignMergeTicket(userCtx.UserUUID, otherUserUUID, email, time.Now().Add(15*time.Minute))
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to sign merge ticket: %w", err))
		}

		var buildCount int
		_ = s.db.QueryRow(ctx, `SELECT COUNT(*) FROM builds WHERE user_uuid = $1`, otherUserUUID).Scan(&buildCount)

		var buildNames []string
		rows, err := s.db.Query(ctx, `SELECT name FROM builds WHERE user_uuid = $1 ORDER BY name ASC LIMIT 10`, otherUserUUID)
		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var bName string
				if err := rows.Scan(&bName); err == nil {
					buildNames = append(buildNames, bName)
				}
			}
		}

		var otherEmails []string
		mailRows, err := s.db.Query(ctx, `
			SELECT DISTINCT email FROM (
				SELECT email FROM user_emails WHERE user_uuid = $1
				UNION
				SELECT email FROM users WHERE uuid = $1
			) sub ORDER BY email ASC`, otherUserUUID)
		if err == nil {
			defer mailRows.Close()
			for mailRows.Next() {
				var em string
				if err := mailRows.Scan(&em); err == nil && em != "" {
					otherEmails = append(otherEmails, em)
				}
			}
		}
		if email != "" && !slices.Contains(otherEmails, email) {
			otherEmails = append(otherEmails, email)
		}

		var oauthProviders []string
		oaRows, err := s.db.Query(ctx, `SELECT provider FROM oauth_accounts WHERE user_uuid = $1 ORDER BY provider ASC`, otherUserUUID)
		if err == nil {
			defer oaRows.Close()
			for oaRows.Next() {
				var prov string
				if err := oaRows.Scan(&prov); err == nil {
					oauthProviders = append(oauthProviders, prov)
				}
			}
		}

		return connect.NewResponse(&pb.VerifyAddEmailOtpResponse{
			Success:             true,
			MergeRequired:       true,
			OtherUserUsername:   otherUsername,
			OtherBuildCount:     int32(buildCount),
			OtherBuildNames:     buildNames,
			OtherEmails:         otherEmails,
			OtherOauthProviders: oauthProviders,
			MergeTicket:         &mergeTicket,
			Message:             fmt.Sprintf("This email belongs to account @%s (%d builds). Merging required to transfer data.", otherUsername, buildCount),
		}), nil
	}

	// No other account found -> Add directly to current user's emails
	newUUID := NewUUIDv7()
	_, err = s.db.Exec(ctx, `
		INSERT INTO user_emails (uuid, user_uuid, email, is_primary, verified_via_otp, created_at)
		VALUES ($1, $2, $3, false, true, NOW())
		ON CONFLICT (email) DO UPDATE SET user_uuid = $2, is_primary = false, verified_via_otp = true`,
		newUUID, userCtx.UserUUID, email)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to add email: %w", err))
	}

	return connect.NewResponse(&pb.VerifyAddEmailOtpResponse{
		Success:       true,
		MergeRequired: false,
		Message:       "Email successfully added to your account",
	}), nil
}

func (s *AuthService) MergeAccount(ctx context.Context, req *connect.Request[pb.MergeAccountRequest]) (*connect.Response[pb.MergeAccountResponse], error) {
	userCtx := UserContextFromContext(ctx)
	if userCtx == nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("authentication required"))
	}

	email := strings.ToLower(strings.TrimSpace(req.Msg.GetEmail()))
	ticket := strings.TrimSpace(req.Msg.GetMergeTicket())

	var otherUserUUID string

	if ticket != "" {
		claims, err := VerifyMergeTicket(ticket)
		if err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid or expired merge ticket: %w", err))
		}
		if claims.SourceUserUUID != userCtx.UserUUID {
			return nil, connect.NewError(connect.CodePermissionDenied, errors.New("merge ticket was issued to a different user session"))
		}
		if email != "" && strings.ToLower(email) != strings.ToLower(claims.Email) {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("merge ticket email mismatch"))
		}
		email = claims.Email
		otherUserUUID = claims.TargetUserUUID
	} else {
		// Fallback verification using magic link token or OTP
		token := strings.TrimSpace(req.Msg.GetToken())
		code := strings.TrimSpace(req.Msg.GetCode())
		if token != "" {
			claims, err := VerifyMagicLinkToken(token)
			if err != nil {
				return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid or expired magic link: %w", err))
			}
			if claims.Purpose != "add_email" || claims.RequesterUserUUID != userCtx.UserUUID {
				return nil, connect.NewError(connect.CodePermissionDenied, errors.New("invalid or unauthorized magic link token"))
			}
			if email == "" {
				email = claims.Email
			}
			_, _ = s.db.Exec(ctx, `UPDATE email_otps SET used_at = NOW() WHERE uuid = $1`, claims.OtpUUID)
		} else if email != "" && code != "" {
			challengeID := extractChallengeID(req.Msg.GetChallengeId(), req.Header())
			if challengeID == "" {
				return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("challenge session expired or missing"))
			}
			var otpUUID string
			var codeHash string
			err := s.db.QueryRow(ctx, `
				UPDATE email_otps
				SET failed_attempts = failed_attempts + 1, used_at = NOW()
				WHERE email = $1 AND challenge_id_hash = $2 AND purpose = 'add_email' AND requested_by_user_uuid = $3 AND used_at IS NULL AND failed_attempts < 3
				RETURNING uuid, code_hash`,
				email, HashToken(challengeID), userCtx.UserUUID).Scan(&otpUUID, &codeHash)
			if err != nil || !VerifyOTP(code, codeHash) {
				return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("invalid or expired verification code"))
			}
		} else {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("merge ticket (or verification code) is required to merge accounts"))
		}

		err := s.db.QueryRow(ctx, `
			SELECT u.uuid
			FROM users u
			WHERE LOWER(u.email) = LOWER($1) AND u.uuid != $2 AND u.deleted_at IS NULL
			UNION
			SELECT u.uuid
			FROM user_emails ue
			JOIN users u ON u.uuid = ue.user_uuid
			WHERE LOWER(ue.email) = LOWER($1) AND u.uuid != $2 AND u.deleted_at IS NULL
			LIMIT 1`, email, userCtx.UserUUID).Scan(&otherUserUUID)
		if err != nil {
			return nil, connect.NewError(connect.CodeNotFound, errors.New("no other account found for this email"))
		}
	}

	var otherUsername, otherEmail string
	err := s.db.QueryRow(ctx, `SELECT id, email FROM users WHERE uuid = $1`, otherUserUUID).Scan(&otherUsername, &otherEmail)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("account to merge was not found"))
	}

	// Begin atomic merge transaction
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to begin transaction: %w", err))
	}
	defer tx.Rollback(ctx)

	// 1. Count builds to transfer
	var buildCount int
	_ = tx.QueryRow(ctx, `SELECT COUNT(*) FROM builds WHERE user_uuid = $1`, otherUserUUID).Scan(&buildCount)

	// 2. Reassign builds to current user
	_, err = tx.Exec(ctx, `
		UPDATE builds
		SET user_uuid = $1, author_username = $2
		WHERE user_uuid = $3`,
		userCtx.UserUUID, userCtx.UserID, otherUserUUID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to transfer builds: %w", err))
	}

	// 3. Collect all email addresses belonging to the other account (both user_emails and users.email)
	var emailsToTransfer []string
	mailRows, err := tx.Query(ctx, `
		SELECT DISTINCT email FROM (
			SELECT email FROM user_emails WHERE user_uuid = $1
			UNION
			SELECT email FROM users WHERE uuid = $1
		) sub`, otherUserUUID)
	if err == nil {
		defer mailRows.Close()
		for mailRows.Next() {
			var em string
			if err := mailRows.Scan(&em); err == nil && em != "" {
				emailsToTransfer = append(emailsToTransfer, em)
			}
		}
	}
	if otherEmail != "" && !slices.Contains(emailsToTransfer, otherEmail) {
		emailsToTransfer = append(emailsToTransfer, otherEmail)
	}
	if email != "" && !slices.Contains(emailsToTransfer, email) {
		emailsToTransfer = append(emailsToTransfer, email)
	}

	// 4. Reassign and transfer all emails to the current user
	_, err = tx.Exec(ctx, `DELETE FROM user_emails WHERE user_uuid = $1`, otherUserUUID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to clear old user emails: %w", err))
	}

	var transferredEmails []string
	for _, em := range emailsToTransfer {
		emLower := strings.ToLower(strings.TrimSpace(em))
		if emLower == "" {
			continue
		}
		newEmailUUID := NewUUIDv7()
		_, err = tx.Exec(ctx, `
			INSERT INTO user_emails (uuid, user_uuid, email, is_primary, verified_via_otp, created_at)
			VALUES ($1, $2, $3, false, true, NOW())
			ON CONFLICT (email) DO UPDATE SET user_uuid = $2, is_primary = false`,
			newEmailUUID, userCtx.UserUUID, emLower)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to associate email %s: %w", emLower, err))
		}
		transferredEmails = append(transferredEmails, emLower)
	}

	// 5. Transfer OAuth accounts
	_, _ = tx.Exec(ctx, `
		UPDATE oauth_accounts
		SET user_uuid = $1
		WHERE user_uuid = $2`,
		userCtx.UserUUID, otherUserUUID)

	// 6. Transfer and record username history
	_, _ = tx.Exec(ctx, `
		UPDATE username_histories
		SET user_uuid = $1
		WHERE user_uuid = $2`,
		userCtx.UserUUID, otherUserUUID)

	histUUID := NewUUIDv7()
	_, _ = tx.Exec(ctx, `
		INSERT INTO username_histories (uuid, user_uuid, old_username, renamed_at, first_archived_at)
		VALUES ($1, $2, $3, NOW(), NOW())`,
		histUUID, userCtx.UserUUID, otherUsername)

	// 7. Revoke other user's active sessions
	_, _ = tx.Exec(ctx, `DELETE FROM refresh_tokens WHERE user_uuid = $1`, otherUserUUID)

	// 8. Delete email_otps requested by the other user (prevents fk_email_otps_requested_by_user_uuid violation)
	_, _ = tx.Exec(ctx, `DELETE FROM email_otps WHERE requested_by_user_uuid = $1`, otherUserUUID)

	// 9. Delete other user's profile
	_, _ = tx.Exec(ctx, `DELETE FROM user_profiles WHERE user_uuid = $1`, otherUserUUID)

	// 10. Delete other user record from users table
	_, err = tx.Exec(ctx, `DELETE FROM users WHERE uuid = $1`, otherUserUUID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to finalize merge: %w", err))
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to commit merge: %w", err))
	}

	return connect.NewResponse(&pb.MergeAccountResponse{
		Success:               true,
		Message:               fmt.Sprintf("Successfully merged @%s into your account. %d builds transferred.", otherUsername, buildCount),
		TransferredBuildCount: int32(buildCount),
		MergedUsername:        otherUsername,
		MergedEmails:          transferredEmails,
	}), nil
}

func (s *AuthService) SetPrimaryEmail(ctx context.Context, req *connect.Request[pb.SetPrimaryEmailRequest]) (*connect.Response[pb.SetPrimaryEmailResponse], error) {
	userCtx := UserContextFromContext(ctx)
	if userCtx == nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("authentication required"))
	}

	targetEmail := strings.ToLower(strings.TrimSpace(req.Msg.GetEmail()))
	if targetEmail == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("email required"))
	}

	var exists bool
	err := s.db.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM user_emails
			WHERE user_uuid = $1 AND LOWER(email) = LOWER($2)
		)`, userCtx.UserUUID, targetEmail).Scan(&exists)
	if err != nil || !exists {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("email not found on your account"))
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, `
		UPDATE user_emails
		SET is_primary = (LOWER(email) = LOWER($1))
		WHERE user_uuid = $2`,
		targetEmail, userCtx.UserUUID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	_, err = tx.Exec(ctx, `
		UPDATE users
		SET email = $1, updated_at = NOW()
		WHERE uuid = $2`,
		targetEmail, userCtx.UserUUID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&pb.SetPrimaryEmailResponse{
		Success: true,
		Message: "Primary email updated",
	}), nil
}

func (s *AuthService) RemoveUserEmail(ctx context.Context, req *connect.Request[pb.RemoveUserEmailRequest]) (*connect.Response[pb.RemoveUserEmailResponse], error) {
	userCtx := UserContextFromContext(ctx)
	if userCtx == nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("authentication required"))
	}

	targetEmail := strings.ToLower(strings.TrimSpace(req.Msg.GetEmail()))
	if targetEmail == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("email required"))
	}

	var isPrimary bool
	err := s.db.QueryRow(ctx, `
		SELECT is_primary
		FROM user_emails
		WHERE user_uuid = $1 AND LOWER(email) = LOWER($2)`,
		userCtx.UserUUID, targetEmail).Scan(&isPrimary)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, connect.NewError(connect.CodeNotFound, errors.New("email not found on your account"))
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	if isPrimary {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("cannot remove your primary email. Please set another email as primary first"))
	}

	var count int
	_ = s.db.QueryRow(ctx, `SELECT COUNT(*) FROM user_emails WHERE user_uuid = $1`, userCtx.UserUUID).Scan(&count)
	if count <= 1 {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("cannot remove the only email on your account"))
	}

	_, err = s.db.Exec(ctx, `
		DELETE FROM user_emails
		WHERE user_uuid = $1 AND LOWER(email) = LOWER($2)`,
		userCtx.UserUUID, targetEmail)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&pb.RemoveUserEmailResponse{
		Success: true,
		Message: "Email removed from your account",
	}), nil
}

// LinkOAuthAccount associates an external OAuth provider account with a user and links its email in user_email_oauth_accounts.
func (s *AuthService) LinkOAuthAccount(ctx context.Context, userUUID, provider, providerAccountID, email string) error {
	provider = strings.ToLower(strings.TrimSpace(provider))
	email = strings.ToLower(strings.TrimSpace(email))
	if userUUID == "" || provider == "" || providerAccountID == "" || email == "" {
		return errors.New("userUUID, provider, providerAccountID, and email are required")
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// 1. Upsert oauth_accounts
	var oauthUUID string
	err = tx.QueryRow(ctx, `
		SELECT uuid FROM oauth_accounts WHERE user_uuid = $1 AND LOWER(provider) = $2`,
		userUUID, provider).Scan(&oauthUUID)
	if errors.Is(err, pgx.ErrNoRows) {
		oauthUUID = NewUUIDv7()
		_, err = tx.Exec(ctx, `
			INSERT INTO oauth_accounts (uuid, user_uuid, provider, provider_account_id, created_at)
			VALUES ($1, $2, $3, $4, NOW())`,
			oauthUUID, userUUID, provider, providerAccountID)
		if err != nil {
			return fmt.Errorf("failed to insert oauth account: %w", err)
		}
	} else if err != nil {
		return fmt.Errorf("failed to query oauth account: %w", err)
	} else {
		_, err = tx.Exec(ctx, `
			UPDATE oauth_accounts
			SET provider_account_id = $1
			WHERE uuid = $2`,
			providerAccountID, oauthUUID)
		if err != nil {
			return fmt.Errorf("failed to update oauth account: %w", err)
		}
	}

	// 2. Ensure user_emails has this email
	var emailUUID string
	err = tx.QueryRow(ctx, `
		SELECT uuid FROM user_emails WHERE LOWER(email) = $1`, email).Scan(&emailUUID)
	if errors.Is(err, pgx.ErrNoRows) {
		emailUUID = NewUUIDv7()
		_, err = tx.Exec(ctx, `
			INSERT INTO user_emails (uuid, user_uuid, email, is_primary, verified_via_otp, created_at)
			VALUES ($1, $2, $3, false, false, NOW())`,
			emailUUID, userUUID, email)
		if err != nil {
			return fmt.Errorf("failed to insert user email: %w", err)
		}
	} else if err != nil {
		return fmt.Errorf("failed to query user email: %w", err)
	}

	// 3. Link in user_email_oauth_accounts
	linkUUID := NewUUIDv7()
	_, err = tx.Exec(ctx, `
		INSERT INTO user_email_oauth_accounts (uuid, user_email_uuid, oauth_account_uuid, created_at)
		VALUES ($1, $2, $3, NOW())
		ON CONFLICT (user_email_uuid, oauth_account_uuid) DO NOTHING`,
		linkUUID, emailUUID, oauthUUID)
	if err != nil {
		return fmt.Errorf("failed to link oauth account to email: %w", err)
	}

	return tx.Commit(ctx)
}

// UnlinkOAuthAccount unlinks an OAuth provider from a user, enforcing primary email safeguards
// and pruning orphaned secondary emails.
func (s *AuthService) UnlinkOAuthAccount(ctx context.Context, userUUID, provider string) error {
	provider = strings.ToLower(strings.TrimSpace(provider))
	if userUUID == "" || provider == "" {
		return errors.New("userUUID and provider are required")
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var oauthUUID string
	err = tx.QueryRow(ctx, `
		SELECT uuid FROM oauth_accounts WHERE user_uuid = $1 AND LOWER(provider) = $2`,
		userUUID, provider).Scan(&oauthUUID)
	if errors.Is(err, pgx.ErrNoRows) {
		return connect.NewError(connect.CodeNotFound, errors.New("oauth provider not linked"))
	} else if err != nil {
		return connect.NewError(connect.CodeInternal, err)
	}

	// Total emails on account
	var totalEmails int
	err = tx.QueryRow(ctx, `SELECT COUNT(*) FROM user_emails WHERE user_uuid = $1`, userUUID).Scan(&totalEmails)
	if err != nil {
		return connect.NewError(connect.CodeInternal, err)
	}

	// Inspect all emails associated with this oauth account
	rows, err := tx.Query(ctx, `
		SELECT ue.uuid, ue.email, ue.is_primary, ue.verified_via_otp,
		       COUNT(uea2.uuid) AS other_links
		FROM user_email_oauth_accounts uea
		JOIN user_emails ue ON uea.user_email_uuid = ue.uuid
		LEFT JOIN user_email_oauth_accounts uea2 ON uea2.user_email_uuid = ue.uuid AND uea2.oauth_account_uuid != uea.oauth_account_uuid
		WHERE uea.oauth_account_uuid = $1
		GROUP BY ue.uuid, ue.email, ue.is_primary, ue.verified_via_otp`, oauthUUID)
	if err != nil {
		return connect.NewError(connect.CodeInternal, err)
	}
	defer rows.Close()

	type linkedEmailInfo struct {
		uuid           string
		email          string
		isPrimary      bool
		verifiedViaOTP bool
		otherLinks     int
	}
	var emails []linkedEmailInfo
	for rows.Next() {
		var em linkedEmailInfo
		if err := rows.Scan(&em.uuid, &em.email, &em.isPrimary, &em.verifiedViaOTP, &em.otherLinks); err != nil {
			return connect.NewError(connect.CodeInternal, err)
		}
		emails = append(emails, em)
	}
	if err := rows.Err(); err != nil {
		return connect.NewError(connect.CodeInternal, err)
	}

	// Primary and sole email safeguards
	for _, em := range emails {
		solelyVerifiedByThis := !em.verifiedViaOTP && em.otherLinks == 0
		if solelyVerifiedByThis {
			if em.isPrimary || totalEmails <= 1 {
				return connect.NewError(connect.CodeFailedPrecondition, errors.New("Cannot unlink this provider because it supplies your primary email. Please designate another verified email as primary before unlinking."))
			}
		}
	}

	// Delete oauth_account (cascades to delete user_email_oauth_accounts)
	_, err = tx.Exec(ctx, `DELETE FROM oauth_accounts WHERE uuid = $1`, oauthUUID)
	if err != nil {
		return connect.NewError(connect.CodeInternal, err)
	}

	// Prune secondary emails solely verified by this provider
	for _, em := range emails {
		solelyVerifiedByThis := !em.verifiedViaOTP && em.otherLinks == 0
		if solelyVerifiedByThis && !em.isPrimary {
			_, err = tx.Exec(ctx, `DELETE FROM user_emails WHERE uuid = $1`, em.uuid)
			if err != nil {
				return connect.NewError(connect.CodeInternal, err)
			}
		}
	}

	return tx.Commit(ctx)
}

func (s *AuthService) ListArchivedUsernames(ctx context.Context, _ *connect.Request[pb.ListArchivedUsernamesRequest]) (*connect.Response[pb.ListArchivedUsernamesResponse], error) {
	userCtx := UserContextFromContext(ctx)
	if userCtx == nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("authentication required"))
	}

	var lastRenamedAt *time.Time
	var currentUsername string
	err := s.db.QueryRow(ctx, `SELECT id, last_renamed_at FROM users WHERE uuid = $1 AND deleted_at IS NULL`, userCtx.UserUUID).Scan(&currentUsername, &lastRenamedAt)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("user not found"))
	}

	now := time.Now()

	// Calculate user rename cooldowns
	var nextAllowedNewRenameAt *timestamppb.Timestamp
	var nextAllowedArchiveRenameAt *timestamppb.Timestamp

	if lastRenamedAt != nil {
		timeSinceRename := now.Sub(*lastRenamedAt)
		// Brand-new username cooldown: 30 days
		if timeSinceRename < 30*24*time.Hour {
			nextAllowedNewRenameAt = timestamppb.New(lastRenamedAt.Add(30 * 24 * time.Hour))
		}
		// Archived username reclaim rules:
		// Within 24 hours: instant reclaim allowed (grace period).
		// Between 24 hours and 7 days: cooldown active.
		// After 7 days: reclaim allowed.
		if timeSinceRename >= 24*time.Hour && timeSinceRename < 7*24*time.Hour {
			nextAllowedArchiveRenameAt = timestamppb.New(lastRenamedAt.Add(7 * 24 * time.Hour))
		}
	}

	// Fetch active holds in username_histories for this user (excluding their current active username)
	rows, err := s.db.Query(ctx, `
		SELECT DISTINCT ON (LOWER(hh.old_username))
			hh.old_username,
			hh.renamed_at,
			COALESCE(hh.first_archived_at, hh.renamed_at)
		FROM username_histories hh
		WHERE hh.user_uuid = $1
		  AND LOWER(hh.old_username) != LOWER($2)
		  AND COALESCE(hh.first_archived_at, hh.renamed_at) + INTERVAL '30 days' > NOW()
		ORDER BY LOWER(hh.old_username), hh.renamed_at DESC`,
		userCtx.UserUUID, currentUsername)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	defer rows.Close()

	var archivedList []*pb.ArchivedUsername
	for rows.Next() {
		var username string
		var renamedAt, firstArchivedAt time.Time
		if err := rows.Scan(&username, &renamedAt, &firstArchivedAt); err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}

		releaseAt := firstArchivedAt.Add(30 * 24 * time.Hour)
		daysRemaining := int32(math.Ceil(releaseAt.Sub(now).Hours() / 24))
		if daysRemaining < 1 {
			daysRemaining = 1
		}

		canReclaimNow := true
		var nextReclaimAllowedAt *timestamppb.Timestamp

		if lastRenamedAt != nil {
			timeSinceRename := now.Sub(*lastRenamedAt)
			if timeSinceRename >= 24*time.Hour && timeSinceRename < 7*24*time.Hour {
				canReclaimNow = false
				nextReclaimAllowedAt = timestamppb.New(lastRenamedAt.Add(7 * 24 * time.Hour))
			}
		}

		archivedList = append(archivedList, &pb.ArchivedUsername{
			Username:             username,
			FirstArchivedAt:      timestamppb.New(firstArchivedAt),
			RenamedAt:            timestamppb.New(renamedAt),
			ReleaseAt:            timestamppb.New(releaseAt),
			DaysRemaining:        daysRemaining,
			CanReclaimNow:        canReclaimNow,
			NextReclaimAllowedAt: nextReclaimAllowedAt,
		})
	}

	return connect.NewResponse(&pb.ListArchivedUsernamesResponse{
		ArchivedUsernames:          archivedList,
		NextAllowedNewRenameAt:     nextAllowedNewRenameAt,
		NextAllowedArchiveRenameAt: nextAllowedArchiveRenameAt,
	}), nil
}

func (s *AuthService) ChangeUsername(ctx context.Context, req *connect.Request[pb.ChangeUsernameRequest]) (*connect.Response[pb.ChangeUsernameResponse], error) {
	userCtx := UserContextFromContext(ctx)
	if userCtx == nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("authentication required"))
	}

	newUsername := strings.ToLower(strings.TrimSpace(req.Msg.GetNewUsername()))
	if err := ValidateUsername(newUsername); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	var currentUsername string
	var lastRenamedAt *time.Time
	err := s.db.QueryRow(ctx, `SELECT id, last_renamed_at FROM users WHERE uuid = $1 AND deleted_at IS NULL`, userCtx.UserUUID).Scan(&currentUsername, &lastRenamedAt)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("user not found"))
	}

	if strings.ToLower(currentUsername) == newUsername {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("new username must be different from your current username"))
	}

	now := time.Now()

	// Check if this new username is owned by caller as an active archived username
	var isCallerArchivedUsername bool
	err = s.db.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM username_histories
			WHERE user_uuid = $1
			  AND LOWER(old_username) = $2
			  AND COALESCE(first_archived_at, renamed_at) + INTERVAL '30 days' > NOW()
		)`, userCtx.UserUUID, newUsername).Scan(&isCallerArchivedUsername)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	if isCallerArchivedUsername {
		// Relaxed reclaim restrictions:
		// 1. Instant revert within 24 hours of rename.
		// 2. Cooldown of 7 days between renames if >24 hours has passed.
		if lastRenamedAt != nil {
			timeSinceRename := now.Sub(*lastRenamedAt)
			if timeSinceRename >= 24*time.Hour && timeSinceRename < 7*24*time.Hour {
				daysLeft := int(math.Ceil((7*24*time.Hour - timeSinceRename).Hours() / 24))
				return nil, connect.NewError(connect.CodeFailedPrecondition,
					fmt.Errorf("you can only reclaim this username within 24 hours of renaming, or after a 7-day cooldown (available in %d day(s))", daysLeft))
			}
		}
	} else {
		// Brand-new username rules:
		// 1. Standard 30-day cooldown from last_renamed_at.
		if lastRenamedAt != nil {
			timeSinceRename := now.Sub(*lastRenamedAt)
			if timeSinceRename < 30*24*time.Hour {
				daysLeft := int(math.Ceil((30*24*time.Hour - timeSinceRename).Hours() / 24))
				return nil, connect.NewError(connect.CodeFailedPrecondition,
					fmt.Errorf("you can only change to a new username once every 30 days (available in %d day(s))", daysLeft))
			}
		}

		// 2. Check if taken by another active user or held in 30-day deletion grace period
		var activeExists bool
		err = s.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM users WHERE LOWER(id) = $1 AND (deleted_at IS NULL OR deleted_at + INTERVAL '30 days' > NOW()))`, newUsername).Scan(&activeExists)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		if activeExists {
			return nil, connect.NewError(connect.CodeAlreadyExists, errors.New("this username is already taken"))
		}

		// 3. Check if held by another user in username_histories within 30 days
		var heldByOther bool
		err = s.db.QueryRow(ctx, `
			SELECT EXISTS(
				SELECT 1 FROM username_histories
				WHERE LOWER(old_username) = $1
				  AND user_uuid != $2
				  AND COALESCE(first_archived_at, renamed_at) + INTERVAL '30 days' > NOW()
			)`, newUsername, userCtx.UserUUID).Scan(&heldByOther)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		if heldByOther {
			return nil, connect.NewError(connect.CodePermissionDenied, errors.New("this username is reserved by another user"))
		}
	}

	// Begin rename transaction
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	defer tx.Rollback(ctx)

	// 1. Update user username and last_renamed_at
	_, err = tx.Exec(ctx, `
		UPDATE users
		SET id = $1, last_renamed_at = NOW(), updated_at = NOW()
		WHERE uuid = $2`,
		newUsername, userCtx.UserUUID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to update username: %w", err))
	}

	// 2. Anti-hoarding hold tracking:
	// If currentUsername was previously archived by this user and its 30-day window hasn't expired,
	// preserve its original first_archived_at. Otherwise, start a fresh 30-day hold from NOW().
	var existingFirstArchivedAt *time.Time
	_ = tx.QueryRow(ctx, `
		SELECT COALESCE(first_archived_at, renamed_at)
		FROM username_histories
		WHERE user_uuid = $1
		  AND LOWER(old_username) = LOWER($2)
		  AND COALESCE(first_archived_at, renamed_at) + INTERVAL '30 days' > NOW()
		ORDER BY COALESCE(first_archived_at, renamed_at) ASC
		LIMIT 1`, userCtx.UserUUID, currentUsername).Scan(&existingFirstArchivedAt)

	firstArchivedAt := now
	if existingFirstArchivedAt != nil {
		firstArchivedAt = *existingFirstArchivedAt
	}

	histUUID := NewUUIDv7()
	_, err = tx.Exec(ctx, `
		INSERT INTO username_histories (uuid, user_uuid, old_username, renamed_at, first_archived_at)
		VALUES ($1, $2, $3, NOW(), $4)`,
		histUUID, userCtx.UserUUID, currentUsername, firstArchivedAt)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to record username history: %w", err))
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to commit username change: %w", err))
	}

	// Fetch updated user to return
	var user pb.User
	var newLastRenamedAt, deletedAt, bannedAt, createdAt, updatedAt *time.Time
	err = s.db.QueryRow(ctx, `
		SELECT uuid, id, email, role, last_renamed_at, deleted_at, banned_at, created_at, updated_at
		FROM users WHERE uuid = $1`, userCtx.UserUUID).Scan(
		&user.Uuid, &user.Id, &user.Email, &user.Role,
		&newLastRenamedAt, &deletedAt, &bannedAt, &createdAt, &updatedAt)
	if err == nil {
		if newLastRenamedAt != nil {
			user.LastRenamedAt = timestamppb.New(*newLastRenamedAt)
		}
		if createdAt != nil {
			user.CreatedAt = timestamppb.New(*createdAt)
		}
		if updatedAt != nil {
			user.UpdatedAt = timestamppb.New(*updatedAt)
		}
	}

	// Security alert email dispatch: Send to all user's emails
	var emails []string
	rows, err := s.db.Query(ctx, `SELECT email FROM user_emails WHERE user_uuid = $1`, userCtx.UserUUID)
	if err == nil {
		for rows.Next() {
			var e string
			if err := rows.Scan(&e); err == nil && e != "" {
				emails = append(emails, e)
			}
		}
		rows.Close()
	}
	if len(emails) == 0 && user.Email != "" {
		emails = append(emails, user.Email)
	}
	for _, em := range emails {
		log.Printf("[SECURITY ALERT] To: %s | Username changed from @%s to @%s. Previous username @%s is reserved for 30 days.",
			em, currentUsername, newUsername, currentUsername)
	}

	return connect.NewResponse(&pb.ChangeUsernameResponse{
		Success:                    true,
		Message:                    fmt.Sprintf("Your username has been changed to @%s", newUsername),
		User:                       &user,
		NextAllowedNewRenameAt:     timestamppb.New(now.Add(30 * 24 * time.Hour)),
		NextAllowedArchiveRenameAt: timestamppb.New(now), // Immediately eligible for 24h grace revert
	}), nil
}

// ---------------------------------------------------------
// Helper Methods
// ---------------------------------------------------------

func (s *AuthService) returnOtpError(msg string, remaining int32, isExhausted bool, cooldownSeconds int32) (*connect.Response[pb.AuthResponse], error) {
	detail, _ := connect.NewErrorDetail(&pb.OtpVerificationError{
		Message:           msg,
		RemainingAttempts: remaining,
		MaxAttempts:       MaxOtpAttempts,
		IsExhausted:       isExhausted,
		CooldownSeconds:   cooldownSeconds,
	})
	connErr := connect.NewError(connect.CodeInvalidArgument, errors.New(msg))
	if detail != nil {
		connErr.AddDetail(detail)
	}
	return nil, connErr
}

func pluralAttempts(n int32) string {
	if n == 1 {
		return "attempt"
	}
	return "attempts"
}

const ChallengeCookieName = "qs_challenge"

func extractClientIP(header http.Header, peerAddr string, trustedHops ...int) string {
	hops := 0
	if len(trustedHops) > 0 {
		hops = trustedHops[0]
	} else {
		hopsStr := os.Getenv("TRUSTED_PROXY_HOPS")
		hops, _ = strconv.Atoi(hopsStr)
	}
	if hops > 0 || os.Getenv("ENV") != "production" {
		xff := header.Get("X-Forwarded-For")
		if xff != "" {
			parts := strings.Split(xff, ",")
			if hops > 0 && len(parts) >= hops {
				return strings.TrimSpace(parts[len(parts)-hops])
			}
			return strings.TrimSpace(parts[0])
		}
	}
	if peerAddr != "" {
		host, _, err := net.SplitHostPort(peerAddr)
		if err == nil && host != "" {
			return host
		}
		return peerAddr
	}
	return ""
}

func (s *AuthService) getClientIP[T any](req *connect.Request[T]) string {
	peerAddr := ""
	if req.Peer().Addr != "" {
		peerAddr = req.Peer().Addr
	}
	hops := 0
	if s != nil && s.cfg != nil {
		hops = s.cfg.TrustedProxyHops
	}
	return extractClientIP(req.Header(), peerAddr, hops)
}

func getClientIP[T any](req *connect.Request[T]) string {
	peerAddr := ""
	if req.Peer().Addr != "" {
		peerAddr = req.Peer().Addr
	}
	return extractClientIP(req.Header(), peerAddr)
}

func isPrivateOrLocalIP(ipStr string) bool {
	if ipStr == "" || ipStr == "127.0.0.1" || ipStr == "::1" || ipStr == "localhost" {
		return true
	}
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return false
	}
	return ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast()
}

func extractLocationHint(header http.Header, ipStr string) string {
	if isPrivateOrLocalIP(ipStr) {
		return "Local Network"
	}
	if header != nil {
		city := strings.TrimSpace(header.Get("CF-IPCity"))
		country := strings.TrimSpace(header.Get("CF-IPCountry"))
		if city == "" {
			city = strings.TrimSpace(header.Get("X-Geo-City"))
		}
		if country == "" {
			country = strings.TrimSpace(header.Get("X-Geo-Country"))
		}
		if city != "" && country != "" {
			return city + ", " + country
		} else if country != "" {
			return country
		} else if city != "" {
			return city
		}
	}
	return "Unknown Location"
}

func parseDeviceHint(ua string) string {
	if ua == "" {
		return "Web Browser"
	}
	osName := ""
	switch {
	case strings.Contains(ua, "iPhone") || strings.Contains(ua, "iPad") || strings.Contains(ua, "iOS"):
		osName = "iOS"
	case strings.Contains(ua, "Macintosh") || strings.Contains(ua, "Mac OS X"):
		osName = "macOS"
	case strings.Contains(ua, "Windows"):
		osName = "Windows"
	case strings.Contains(ua, "Android"):
		osName = "Android"
	case strings.Contains(ua, "Linux"):
		osName = "Linux"
	}

	browser := ""
	switch {
	case strings.Contains(ua, "Edg/"):
		browser = "Edge"
	case strings.Contains(ua, "Chrome/") && !strings.Contains(ua, "Edg/"):
		browser = "Chrome"
	case strings.Contains(ua, "Firefox/"):
		browser = "Firefox"
	case strings.Contains(ua, "Safari/") && !strings.Contains(ua, "Chrome/"):
		browser = "Safari"
	}

	if osName != "" && browser != "" {
		return fmt.Sprintf("%s on %s", browser, osName)
	} else if osName != "" {
		return osName
	} else if browser != "" {
		return browser
	}
	return "Web Browser"
}

func (s *AuthService) getBaseURL[T any](req *connect.Request[T]) string {
	secure := false
	if s != nil && s.cfg != nil {
		secure = s.cfg.SecureCookies
	} else if os.Getenv("ENV") == "production" || os.Getenv("SECURE_COOKIES") == "true" {
		secure = true
	}
	if req != nil {
		return extractBaseURL(req.Header(), secure)
	}
	return extractBaseURL(nil, secure)
}

func getBaseURL[T any](req *connect.Request[T]) string {
	if req != nil {
		return extractBaseURL(req.Header())
	}
	return extractBaseURL(nil)
}

func extractBaseURL(header http.Header, isSecure ...bool) string {
	secure := false
	if len(isSecure) > 0 {
		secure = isSecure[0]
	} else if os.Getenv("ENV") == "production" || os.Getenv("SECURE_COOKIES") == "true" {
		secure = true
	}

	// 1. Explicit environment variable configuration
	for _, envKey := range []string{"APP_URL", "WEB_URL", "BASE_URL"} {
		if val := strings.TrimRight(strings.TrimSpace(os.Getenv(envKey)), "/"); val != "" {
			return val
		}
	}

	// 2. Extract from request headers if available
	if header != nil {
		// Check Origin header (e.g. http://localhost:5173 or http://localhost:8080)
		if origin := strings.TrimRight(strings.TrimSpace(header.Get("Origin")), "/"); origin != "" {
			if u, err := url.Parse(origin); err == nil && (u.Scheme == "http" || u.Scheme == "https") && u.Host != "" {
				return origin
			}
		}

		// Check Referer header (e.g. http://localhost:5173/login)
		if referer := strings.TrimSpace(header.Get("Referer")); referer != "" {
			if u, err := url.Parse(referer); err == nil && (u.Scheme == "http" || u.Scheme == "https") && u.Host != "" {
				return fmt.Sprintf("%s://%s", u.Scheme, u.Host)
			}
		}

		// Check X-Forwarded-Host or Host
		host := header.Get("X-Forwarded-Host")
		if host == "" {
			host = header.Get("Host")
		}
		if host != "" {
			if strings.Contains(host, ",") {
				host = strings.TrimSpace(strings.Split(host, ",")[0])
			}
			proto := header.Get("X-Forwarded-Proto")
			if proto == "" {
				if secure {
					proto = "https"
				} else {
					proto = "http"
				}
			}
			return fmt.Sprintf("%s://%s", proto, strings.TrimRight(host, "/"))
		}
	}

	// 3. Fallback to localhost with configured PORT
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	if port == "80" {
		return "http://localhost"
	}
	if port == "443" {
		return "https://localhost"
	}
	return fmt.Sprintf("http://localhost:%s", port)
}

func (s *AuthService) setChallengeCookie(header http.Header, challengeToken string) {
	isSecure := false
	maxAge := int(MagicLinkTTL.Seconds())
	if s != nil && s.cfg != nil {
		isSecure = s.cfg.SecureCookies
		maxAge = int(s.cfg.MagicLinkTTL.Seconds())
	} else if os.Getenv("ENV") == "production" || os.Getenv("SECURE_COOKIES") == "true" {
		isSecure = true
	}
	cookie := http.Cookie{
		Name:     ChallengeCookieName,
		Value:    challengeToken,
		Path:     "/",
		MaxAge:   maxAge,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   isSecure,
	}
	header.Add("Set-Cookie", cookie.String())
}

func setChallengeCookie(header http.Header, challengeToken string) {
	isSecure := os.Getenv("ENV") == "production" || os.Getenv("SECURE_COOKIES") == "true"
	cookie := http.Cookie{
		Name:     ChallengeCookieName,
		Value:    challengeToken,
		Path:     "/",
		MaxAge:   int(MagicLinkTTL.Seconds()),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   isSecure,
	}
	header.Add("Set-Cookie", cookie.String())
}

func extractChallengeID(reqChallenge string, header http.Header) string {
	if strings.TrimSpace(reqChallenge) != "" {
		return strings.TrimSpace(reqChallenge)
	}
	rawCookies := header.Get("Cookie")
	if rawCookies != "" {
		cookies := strings.Split(rawCookies, ";")
		for _, c := range cookies {
			parts := strings.SplitN(strings.TrimSpace(c), "=", 2)
			if len(parts) == 2 && parts[0] == ChallengeCookieName {
				return parts[1]
			}
		}
	}
	return ""
}

func (s *AuthService) issueAuthSession(ctx context.Context, email, deviceHint string) (*connect.Response[pb.AuthResponse], error) {
	var user pb.User
	var userUUID, username, role string
	var lastRenamedAt, deletedAt, bannedAt, createdAt, updatedAt *time.Time

	// 1. Try finding user via user_emails
	err := s.db.QueryRow(ctx, `
		SELECT u.uuid, u.id, u.email, u.role, u.last_renamed_at, u.deleted_at, u.banned_at, u.created_at, u.updated_at
		FROM user_emails ue
		JOIN users u ON u.uuid = ue.user_uuid
		WHERE LOWER(ue.email) = LOWER($1) AND (u.deleted_at IS NULL OR u.deleted_at + INTERVAL '30 days' > NOW())
		LIMIT 1`, email).Scan(
		&userUUID, &username, &user.Email, &role,
		&lastRenamedAt, &deletedAt, &bannedAt, &createdAt, &updatedAt)

	if errors.Is(err, pgx.ErrNoRows) {
		// Fallback: check users table directly
		err = s.db.QueryRow(ctx, `
			SELECT uuid, id, email, role, last_renamed_at, deleted_at, banned_at, created_at, updated_at
			FROM users
			WHERE LOWER(email) = LOWER($1) AND (deleted_at IS NULL OR deleted_at + INTERVAL '30 days' > NOW())`, email).Scan(
			&userUUID, &username, &user.Email, &role,
			&lastRenamedAt, &deletedAt, &bannedAt, &createdAt, &updatedAt)
	}

	if errors.Is(err, pgx.ErrNoRows) {
		userUUID = NewUUIDv7()
		username = s.generateUniqueUsername(ctx, email)
		role = "USER"
		now := time.Now()

		_, err = s.db.Exec(ctx, `
			INSERT INTO users (uuid, id, email, role, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $5)`,
			userUUID, username, email, role, now)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to create user: %w", err))
		}

		profileUUID := NewUUIDv7()
		_, err = s.db.Exec(ctx, `
			INSERT INTO user_profiles (uuid, user_uuid, avatar_url, bio, updated_at)
			VALUES ($1, $2, '', '', $3)`,
			profileUUID, userUUID, now)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to create profile: %w", err))
		}

		emailUUID := NewUUIDv7()
		_, _ = s.db.Exec(ctx, `
			INSERT INTO user_emails (uuid, user_uuid, email, is_primary, verified_via_otp, created_at)
			VALUES ($1, $2, $3, true, true, $4)
			ON CONFLICT (email) DO NOTHING`,
			emailUUID, userUUID, email, now)

		user.Uuid = userUUID
		user.Id = username
		user.Email = email
		user.Role = role
		user.CreatedAt = timestamppb.New(now)
		user.UpdatedAt = timestamppb.New(now)
	} else if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	} else {
		// Ensure user_emails has this email
		emailUUID := NewUUIDv7()
		_, _ = s.db.Exec(ctx, `
			INSERT INTO user_emails (uuid, user_uuid, email, is_primary, verified_via_otp, created_at)
			VALUES ($1, $2, $3, true, true, NOW())
			ON CONFLICT (email) DO NOTHING`,
			emailUUID, userUUID, email)

		user.Uuid = userUUID
		user.Id = username
		user.Role = role
		if lastRenamedAt != nil {
			user.LastRenamedAt = timestamppb.New(*lastRenamedAt)
		}
		if deletedAt != nil {
			user.DeletedAt = timestamppb.New(*deletedAt)
		}
		if createdAt != nil {
			user.CreatedAt = timestamppb.New(*createdAt)
		}
		if updatedAt != nil {
			user.UpdatedAt = timestamppb.New(*updatedAt)
		}
	}

	var profile pb.UserProfile
	var profileUpdatedAt *time.Time
	_ = s.db.QueryRow(ctx, `
		SELECT uuid, user_uuid, avatar_url, bio, updated_at
		FROM user_profiles WHERE user_uuid = $1`, userUUID).Scan(
		&profile.Uuid, &profile.UserUuid, &profile.AvatarUrl, &profile.Bio, &profileUpdatedAt)
	if profileUpdatedAt != nil {
		profile.UpdatedAt = timestamppb.New(*profileUpdatedAt)
	}

	sessionUUID := NewUUIDv7()
	rawRefreshToken, tokenHash, err := GenerateRefreshToken()
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	now := time.Now()
	expiresAt := now.Add(s.cfg.RefreshTokenTTL)
	_, err = s.db.Exec(ctx, `
		INSERT INTO refresh_tokens (uuid, user_uuid, token_hash, expires_at, last_used_at, device_hint, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $5)`,
		sessionUUID, userUUID, tokenHash, expiresAt, now, deviceHint)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to save session: %w", err))
	}

	accessToken, err := SignAccessToken(userUUID, username, email, role, sessionUUID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to sign access token: %w", err))
	}

	var accountStatus *string
	var scheduledPurgeAt *timestamppb.Timestamp
	if deletedAt != nil {
		status := "PENDING_DELETION"
		accountStatus = &status
		scheduledPurgeAt = timestamppb.New(deletedAt.Add(30 * 24 * time.Hour))
	}

	res := connect.NewResponse(&pb.AuthResponse{
		AccessToken:      accessToken,
		RefreshToken:     rawRefreshToken,
		User:             &user,
		Profile:          &profile,
		AccountStatus:    accountStatus,
		ScheduledPurgeAt: scheduledPurgeAt,
	})

	cookie := &http.Cookie{
		Name:     "qs_session",
		Value:    rawRefreshToken,
		Path:     "/",
		MaxAge:   int(s.cfg.RefreshTokenTTL.Seconds()),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   s.cfg.SecureCookies,
	}
	res.Header().Set("Set-Cookie", cookie.String())

	return res, nil
}

var usernameStructureRegex = regexp.MustCompile(`^[a-z][a-z0-9_]{1,18}[a-z0-9]$`)

// IsReservedUsername returns true if the username is reserved by the platform.
func IsReservedUsername(username string) bool {
	switch username {
	case "admin", "root", "system", "bot", "robot":
		return true
	default:
		return false
	}
}

// ValidateUsername validates a candidate username against the rules declared in user.proto.
//
// Rules enforced:
// 1. Length between 3 and 20 characters.
// 2. Starts with a lowercase letter, ends with an alphanumeric character.
// 3. Contains only lowercase letters, digits, and underscores.
// 4. Cannot contain consecutive underscores ("__").
// 5. Cannot contain "quadsmith" or "quad_smith".
// 6. Cannot equal reserved usernames "admin", "root", "system", "bot", or "robot".
func ValidateUsername(username string) error {
	if !usernameStructureRegex.MatchString(username) {
		return errors.New("username must be 3-20 characters, start with a letter, end with alphanumeric, and contain only lowercase letters, numbers, and underscores")
	}
	if strings.Contains(username, "__") {
		return errors.New("username cannot contain consecutive underscores")
	}
	if strings.Contains(username, "quadsmith") || strings.Contains(username, "quad_smith") {
		return errors.New("username cannot contain 'quadsmith' or 'quad_smith'")
	}
	if IsReservedUsername(username) {
		return errors.New("reserved username")
	}
	return nil
}

func (s *AuthService) generateUniqueUsername(ctx context.Context, email string) string {
	parts := strings.Split(email, "@")
	base := strings.ToLower(parts[0])
	if idx := strings.Index(base, "+"); idx != -1 {
		base = base[:idx]
	}
	reg := regexp.MustCompile(`[^a-z0-9_]`)
	cleaned := reg.ReplaceAllString(base, "_")
	for strings.Contains(cleaned, "__") {
		cleaned = strings.ReplaceAll(cleaned, "__", "_")
	}
	cleaned = strings.Trim(cleaned, "_")
	cleaned = strings.ReplaceAll(cleaned, "quadsmith", "user")
	cleaned = strings.ReplaceAll(cleaned, "quad_smith", "user")
	if len(cleaned) == 0 || cleaned[0] < 'a' || cleaned[0] > 'z' {
		cleaned = "user_" + cleaned
	}
	cleaned = strings.Trim(cleaned, "_")
	if len(cleaned) < 3 {
		cleaned = "user_" + cleaned
	}
	if len(cleaned) > 20 {
		cleaned = cleaned[:20]
		cleaned = strings.TrimRight(cleaned, "_")
	}
	if IsReservedUsername(cleaned) {
		cleaned = "user_" + cleaned
	}

	candidate := cleaned
	counter := 1
	for {
		var exists bool
		err := s.db.QueryRow(ctx, `
			SELECT EXISTS(SELECT 1 FROM users WHERE id = $1 AND (deleted_at IS NULL OR deleted_at + INTERVAL '30 days' > NOW()))
			OR EXISTS(SELECT 1 FROM username_histories WHERE old_username = $1 AND COALESCE(first_archived_at, renamed_at) >= NOW() - INTERVAL '30 days')`, candidate).Scan(&exists)
		if err != nil || !exists {
			return candidate
		}
		counter++
		suffix := fmt.Sprintf("_%d", counter)
		baseLen := 20 - len(suffix)
		if len(cleaned) > baseLen {
			candidate = cleaned[:baseLen] + suffix
		} else {
			candidate = cleaned + suffix
		}
	}
}

// SendDeleteAccountVerification initiates account deletion by dispatching an out-of-band email verification challenge.
func (s *AuthService) SendDeleteAccountVerification(ctx context.Context, req *connect.Request[pb.SendDeleteAccountVerificationRequest]) (*connect.Response[pb.SendDeleteAccountVerificationResponse], error) {
	userCtx := UserContextFromContext(ctx)
	if userCtx == nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("authentication required"))
	}

	username := strings.ToLower(strings.TrimSpace(req.Msg.GetUsername()))
	if username == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("username is required"))
	}

	// Verify username matches authenticated caller's identity
	var currentUsername, primaryEmail string
	err := s.db.QueryRow(ctx, `
		SELECT u.id, COALESCE(ue.email, u.email)
		FROM users u
		LEFT JOIN user_emails ue ON ue.user_uuid = u.uuid AND ue.is_primary = true
		WHERE u.uuid = $1 AND u.deleted_at IS NULL`, userCtx.UserUUID).Scan(&currentUsername, &primaryEmail)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, connect.NewError(connect.CodeNotFound, errors.New("active user not found"))
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	if !strings.EqualFold(username, currentUsername) {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("username does not match your account"))
	}

	// Rate limit per IP
	clientIP := getClientIP(req)
	if err := CheckIPRateLimit(clientIP); err != nil {
		return nil, connect.NewError(connect.CodeResourceExhausted, err)
	}

	cooldown, err := CheckEmailCooldown(ctx, s.db, primaryEmail)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if cooldown > 0 {
		connErr := connect.NewError(connect.CodeResourceExhausted, fmt.Errorf("please wait %d seconds before requesting another code", cooldown))
		if detail, detailErr := connect.NewErrorDetail(&pb.OtpVerificationError{
			Message:           fmt.Sprintf("Please wait %d seconds before requesting another code", cooldown),
			RemainingAttempts: 0,
			MaxAttempts:       MaxOtpAttempts,
			IsExhausted:       false,
			CooldownSeconds:   cooldown,
		}); detailErr == nil {
			connErr.AddDetail(detail)
		}
		return nil, connErr
	}

	// Invalidate previous unconsumed delete_account OTP challenges for this user/email
	_, _ = s.db.Exec(ctx, `
		UPDATE email_otps
		SET used_at = NOW()
		WHERE (email = $1 OR requested_by_user_uuid = $2) AND purpose = 'delete_account' AND used_at IS NULL`,
		primaryEmail, userCtx.UserUUID)

	otpCode, codeHash, err := GenerateOTP()
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to generate otp: %w", err))
	}

	challengeToken, challengeHash, err := GenerateChallengeToken()
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to generate challenge token: %w", err))
	}

	otpUUID := NewUUIDv7()
	now := time.Now()
	expiresAt := now.Add(s.cfg.MagicLinkTTL)

	_, err = s.db.Exec(ctx, `
		INSERT INTO email_otps (uuid, email, challenge_id_hash, code_hash, purpose, requested_by_user_uuid, expires_at, created_at, failed_attempts)
		VALUES ($1, $2, $3, $4, 'delete_account', $5, $6, $7, 0)`,
		otpUUID, primaryEmail, challengeHash, codeHash, userCtx.UserUUID, expiresAt, now)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to save delete account challenge: %w", err))
	}

	magicToken, err := SignMagicLinkToken(otpUUID, primaryEmail, "delete_account", userCtx.UserUUID, expiresAt)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to generate magic link token: %w", err))
	}

	baseURL := s.getBaseURL(req)
	magicLinkURL := fmt.Sprintf("%s/delete-account?token=%s", baseURL, magicToken)

	// Send verification email synchronously with strict 5s timeout
	emailCtx, emailCancel := context.WithTimeout(ctx, 5*time.Second)
	defer emailCancel()

	if err := s.emailSender.SendVerification(emailCtx, primaryEmail, otpCode, magicLinkURL, "Account Deletion Confirmation", s.cfg.OtpTTL); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to deliver verification email: %w", err))
	}

	return connect.NewResponse(&pb.SendDeleteAccountVerificationResponse{
		Success:     true,
		Message:     "Account deletion verification code and link sent to your primary email",
		ChallengeId: challengeToken,
		ExpiresAt:   timestamppb.New(expiresAt),
	}), nil
}

// DeleteAccount verifies the deletion challenge (code or magic token) and marks the account as soft-deleted.
func (s *AuthService) DeleteAccount(ctx context.Context, req *connect.Request[pb.DeleteAccountRequest]) (*connect.Response[pb.DeleteAccountResponse], error) {
	userCtx := UserContextFromContext(ctx)
	if userCtx == nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("authentication required"))
	}

	username := strings.ToLower(strings.TrimSpace(req.Msg.GetUsername()))
	if username == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("username is required"))
	}

	// Verify username matches authenticated user
	var currentUsername string
	err := s.db.QueryRow(ctx, `SELECT id FROM users WHERE uuid = $1 AND deleted_at IS NULL`, userCtx.UserUUID).Scan(&currentUsername)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, connect.NewError(connect.CodeNotFound, errors.New("active user not found"))
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if !strings.EqualFold(username, currentUsername) {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("username does not match your account"))
	}

	// Check verification method: either code (with challenge_id) or magic_token
	code := strings.TrimSpace(req.Msg.GetCode())
	magicToken := strings.TrimSpace(req.Msg.GetMagicToken())
	challengeID := strings.TrimSpace(req.Msg.GetChallengeId())

	if code == "" && magicToken == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("verification code or magic token required"))
	}

	if code != "" {
		if challengeID == "" {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("challenge_id required when verifying with code"))
		}
		challengeHash := HashToken(challengeID)

		var otpUUID, codeHash string
		var failedAttempts int
		var expiresAt time.Time

		err := s.db.QueryRow(ctx, `
			SELECT uuid, code_hash, failed_attempts, expires_at
			FROM email_otps
			WHERE challenge_id_hash = $1
			  AND purpose = 'delete_account'
			  AND requested_by_user_uuid = $2
			  AND used_at IS NULL
			ORDER BY created_at DESC
			LIMIT 1`, challengeHash, userCtx.UserUUID).Scan(&otpUUID, &codeHash, &failedAttempts, &expiresAt)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("invalid or expired deletion challenge"))
			}
			return nil, connect.NewError(connect.CodeInternal, err)
		}

		if time.Now().After(expiresAt) {
			return nil, connect.NewError(connect.CodeDeadlineExceeded, errors.New("verification code has expired"))
		}

		if failedAttempts >= MaxOtpAttempts {
			return nil, connect.NewError(connect.CodeResourceExhausted, errors.New("too many failed attempts; please request a new verification code"))
		}

		if !VerifyOTP(code, codeHash) {
			failedAttempts++
			_, _ = s.db.Exec(ctx, `UPDATE email_otps SET failed_attempts = $1 WHERE uuid = $2`, failedAttempts, otpUUID)
			remaining := MaxOtpAttempts - failedAttempts
			if remaining <= 0 {
				return nil, connect.NewError(connect.CodeResourceExhausted, errors.New("too many failed attempts; please request a new verification code"))
			}
			return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid verification code; %d attempt(s) remaining", remaining))
		}

		// Mark challenge used
		_, _ = s.db.Exec(ctx, `UPDATE email_otps SET used_at = NOW() WHERE uuid = $1`, otpUUID)
	} else if magicToken != "" {
		claims, err := VerifyMagicLinkToken(magicToken)
		if err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid or expired magic link: %w", err))
		}
		if claims.Purpose != "delete_account" {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("invalid token purpose"))
		}
		if claims.RequesterUserUUID != userCtx.UserUUID {
			return nil, connect.NewError(connect.CodePermissionDenied, errors.New("magic link was issued to a different user"))
		}

		// Check and mark used
		var usedAt *time.Time
		err = s.db.QueryRow(ctx, `SELECT used_at FROM email_otps WHERE uuid = $1`, claims.OtpUUID).Scan(&usedAt)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("challenge not found"))
			}
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		if usedAt != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("magic link has already been used"))
		}
		_, _ = s.db.Exec(ctx, `UPDATE email_otps SET used_at = NOW() WHERE uuid = $1`, claims.OtpUUID)
	}

	// 1-line soft deletion: set deleted_at = NOW()
	now := time.Now()
	_, err = s.db.Exec(ctx, `UPDATE users SET deleted_at = $1, updated_at = $1 WHERE uuid = $2`, now, userCtx.UserUUID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to schedule account deletion: %w", err))
	}

	// Revoke active refresh tokens
	_, _ = s.db.Exec(ctx, `DELETE FROM refresh_tokens WHERE user_uuid = $1`, userCtx.UserUUID)

	scheduledPurgeAt := now.Add(30 * 24 * time.Hour)
	return connect.NewResponse(&pb.DeleteAccountResponse{
		Success:          true,
		Message:          "Your account has been scheduled for deletion. You have a 30-day grace period to restore it by logging in.",
		ScheduledPurgeAt: timestamppb.New(scheduledPurgeAt),
	}), nil
}

// CancelAccountDeletion restores a soft-deleted account within the 30-day grace period.
func (s *AuthService) CancelAccountDeletion(ctx context.Context, _ *connect.Request[pb.CancelAccountDeletionRequest]) (*connect.Response[pb.CancelAccountDeletionResponse], error) {
	userCtx := UserContextFromContext(ctx)
	if userCtx == nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("authentication required"))
	}

	// Verify that the user is currently soft-deleted and within 30-day grace period
	var deletedAt *time.Time
	err := s.db.QueryRow(ctx, `SELECT deleted_at FROM users WHERE uuid = $1`, userCtx.UserUUID).Scan(&deletedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, connect.NewError(connect.CodeNotFound, errors.New("user not found"))
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	if deletedAt == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("account is not scheduled for deletion"))
	}

	if time.Now().After(deletedAt.Add(30 * 24 * time.Hour)) {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("the 30-day grace period for this account has expired"))
	}

	// 1-Line Resurrection: clear deleted_at
	_, err = s.db.Exec(ctx, `UPDATE users SET deleted_at = NULL, updated_at = NOW() WHERE uuid = $1`, userCtx.UserUUID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to restore account: %w", err))
	}

	// Fetch updated user
	var user pb.User
	var lastRenamedAt, bannedAt, createdAt, updatedAt *time.Time
	err = s.db.QueryRow(ctx, `
		SELECT uuid, id, email, role, last_renamed_at, banned_at, created_at, updated_at
		FROM users WHERE uuid = $1`, userCtx.UserUUID).Scan(
		&user.Uuid, &user.Id, &user.Email, &user.Role,
		&lastRenamedAt, &bannedAt, &createdAt, &updatedAt)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if lastRenamedAt != nil {
		user.LastRenamedAt = timestamppb.New(*lastRenamedAt)
	}
	if createdAt != nil {
		user.CreatedAt = timestamppb.New(*createdAt)
	}
	if updatedAt != nil {
		user.UpdatedAt = timestamppb.New(*updatedAt)
	}

	return connect.NewResponse(&pb.CancelAccountDeletionResponse{
		Success: true,
		Message: "Your account has been successfully restored",
		User:    &user,
	}), nil
}

// ---------------------------------------------------------
// UserService
// ---------------------------------------------------------

type UserService struct {
	apiconnect.UnimplementedUserServiceHandler
	db *pgxpool.Pool
}

func NewUserService(db *pgxpool.Pool) *UserService {
	return &UserService{db: db}
}

func (s *UserService) GetUser(ctx context.Context, req *connect.Request[pb.GetUserRequest]) (*connect.Response[pb.User], error) {
	id := req.Msg.GetId()
	if id == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("user id or username required"))
	}

	var user pb.User
	var lastRenamedAt, deletedAt, bannedAt, createdAt, updatedAt *time.Time
	err := s.db.QueryRow(ctx, `
		SELECT uuid, id, email, role, last_renamed_at, deleted_at, banned_at, created_at, updated_at
		FROM users
		WHERE (id = $1 OR uuid::text = $1) AND deleted_at IS NULL`, id).Scan(
		&user.Uuid, &user.Id, &user.Email, &user.Role,
		&lastRenamedAt, &deletedAt, &bannedAt, &createdAt, &updatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// Check if this was a recent username within the 30-day hold window
			err = s.db.QueryRow(ctx, `
				SELECT u.uuid, u.id, u.email, u.role, u.last_renamed_at, u.deleted_at, u.banned_at, u.created_at, u.updated_at
				FROM username_histories hh
				JOIN users u ON u.uuid = hh.user_uuid
				WHERE LOWER(hh.old_username) = LOWER($1) AND u.deleted_at IS NULL AND COALESCE(hh.first_archived_at, hh.renamed_at) >= NOW() - INTERVAL '30 days'
				ORDER BY hh.renamed_at DESC
				LIMIT 1`, id).Scan(
				&user.Uuid, &user.Id, &user.Email, &user.Role,
				&lastRenamedAt, &deletedAt, &bannedAt, &createdAt, &updatedAt)
			if err != nil {
				if errors.Is(err, pgx.ErrNoRows) {
					return nil, connect.NewError(connect.CodeNotFound, errors.New("user not found"))
				}
				return nil, connect.NewError(connect.CodeInternal, err)
			}
		} else {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
	}

	// Privacy & Authorization: Users may only inspect their own account; admins may inspect any account
	userCtx := UserContextFromContext(ctx)
	if userCtx == nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("authentication required"))
	}
	if userCtx.Role != "ADMIN" && userCtx.UserUUID != user.Uuid {
		return nil, connect.NewError(connect.CodePermissionDenied, errors.New("permission denied"))
	}

	if lastRenamedAt != nil {
		user.LastRenamedAt = timestamppb.New(*lastRenamedAt)
	}
	if createdAt != nil {
		user.CreatedAt = timestamppb.New(*createdAt)
	}
	if updatedAt != nil {
		user.UpdatedAt = timestamppb.New(*updatedAt)
	}
	return connect.NewResponse(&user), nil
}

func (s *UserService) GetPublicProfile(ctx context.Context, req *connect.Request[pb.GetPublicProfileRequest]) (*connect.Response[pb.PublicUserProfile], error) {
	id := req.Msg.GetId()
	if id == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("user id or username required"))
	}

	var userUUID, username string
	var createdAt *time.Time
	var avatarUrl, bio *string

	err := s.db.QueryRow(ctx, `
		SELECT u.uuid, u.id, u.created_at, p.avatar_url, p.bio
		FROM users u
		LEFT JOIN user_profiles p ON p.user_uuid = u.uuid
		WHERE (u.id = $1 OR u.uuid::text = $1) AND u.deleted_at IS NULL`, id).Scan(
		&userUUID, &username, &createdAt, &avatarUrl, &bio)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// Check if this was a recent username within the 30-day hold window
			err = s.db.QueryRow(ctx, `
				SELECT u.uuid, u.id, u.created_at, p.avatar_url, p.bio
				FROM username_histories hh
				JOIN users u ON u.uuid = hh.user_uuid
				LEFT JOIN user_profiles p ON p.user_uuid = u.uuid
				WHERE LOWER(hh.old_username) = LOWER($1) AND u.deleted_at IS NULL AND COALESCE(hh.first_archived_at, hh.renamed_at) >= NOW() - INTERVAL '30 days'
				ORDER BY hh.renamed_at DESC
				LIMIT 1`, id).Scan(&userUUID, &username, &createdAt, &avatarUrl, &bio)
			if err != nil {
				if errors.Is(err, pgx.ErrNoRows) {
					return nil, connect.NewError(connect.CodeNotFound, errors.New("user profile not found"))
				}
				return nil, connect.NewError(connect.CodeInternal, err)
			}
		} else {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
	}

	var buildCount int32
	_ = s.db.QueryRow(ctx, `SELECT COUNT(*) FROM builds WHERE user_uuid = $1`, userUUID).Scan(&buildCount)

	var reputationScore int32
	_ = s.db.QueryRow(ctx, `SELECT score FROM user_reputations WHERE user_uuid = $1`, userUUID).Scan(&reputationScore)

	pub := &pb.PublicUserProfile{
		Id:              username,
		BuildCount:      buildCount,
		ReputationScore: reputationScore,
	}
	if avatarUrl != nil {
		pub.AvatarUrl = *avatarUrl
	}
	if bio != nil {
		pub.Bio = *bio
	}
	if createdAt != nil {
		pub.CreatedAt = timestamppb.New(*createdAt)
	}

	return connect.NewResponse(pub), nil
}

func (s *UserService) CheckUsernameAvailable(ctx context.Context, req *connect.Request[pb.CheckUsernameAvailableRequest]) (*connect.Response[pb.CheckUsernameAvailableResponse], error) {
	username := strings.ToLower(strings.TrimSpace(req.Msg.GetUsername()))
	if username == "" {
		return connect.NewResponse(&pb.CheckUsernameAvailableResponse{
			Status:    pb.UsernameAvailabilityStatus_USERNAME_AVAILABILITY_STATUS_INVALID,
			Available: false,
			Message:   "Username is required",
		}), nil
	}
	if IsReservedUsername(username) {
		return connect.NewResponse(&pb.CheckUsernameAvailableResponse{
			Status:    pb.UsernameAvailabilityStatus_USERNAME_AVAILABILITY_STATUS_UNAVAILABLE,
			Available: false,
			Message:   "This username is already taken",
		}), nil
	}
	if err := ValidateUsername(username); err != nil {
		return connect.NewResponse(&pb.CheckUsernameAvailableResponse{
			Status:    pb.UsernameAvailabilityStatus_USERNAME_AVAILABILITY_STATUS_INVALID,
			Available: false,
			Message:   err.Error(),
		}), nil
	}

	userCtx := UserContextFromContext(ctx)

	// Check if active user already has this username (or is in 30-day deletion grace period)
	var activeOwnerUUID string
	err := s.db.QueryRow(ctx, `SELECT uuid FROM users WHERE LOWER(id) = $1 AND (deleted_at IS NULL OR deleted_at + INTERVAL '30 days' > NOW())`, username).Scan(&activeOwnerUUID)
	if err == nil {
		if userCtx != nil && activeOwnerUUID == userCtx.UserUUID {
			return connect.NewResponse(&pb.CheckUsernameAvailableResponse{
				Status:    pb.UsernameAvailabilityStatus_USERNAME_AVAILABILITY_STATUS_UNAVAILABLE,
				Available: false,
				Message:   "This is your current username",
			}), nil
		}
		return connect.NewResponse(&pb.CheckUsernameAvailableResponse{
			Status:    pb.UsernameAvailabilityStatus_USERNAME_AVAILABILITY_STATUS_UNAVAILABLE,
			Available: false,
			Message:   "This username is already taken",
		}), nil
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	// Check if held in username_histories within 30-day window
	var historyOwnerUUID string
	err = s.db.QueryRow(ctx, `
		SELECT user_uuid
		FROM username_histories
		WHERE LOWER(old_username) = $1 AND COALESCE(first_archived_at, renamed_at) + INTERVAL '30 days' > NOW()
		ORDER BY COALESCE(first_archived_at, renamed_at) DESC
		LIMIT 1`, username).Scan(&historyOwnerUUID)
	if err == nil {
		if userCtx != nil && historyOwnerUUID == userCtx.UserUUID {
			return connect.NewResponse(&pb.CheckUsernameAvailableResponse{
				Status:    pb.UsernameAvailabilityStatus_USERNAME_AVAILABILITY_STATUS_AVAILABLE,
				Available: true,
				Message:   "Your archived username (available to reclaim)",
			}), nil
		}
		return connect.NewResponse(&pb.CheckUsernameAvailableResponse{
			Status:    pb.UsernameAvailabilityStatus_USERNAME_AVAILABILITY_STATUS_UNAVAILABLE,
			Available: false,
			Message:   "This username is already taken",
		}), nil
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&pb.CheckUsernameAvailableResponse{
		Status:    pb.UsernameAvailabilityStatus_USERNAME_AVAILABILITY_STATUS_AVAILABLE,
		Available: true,
		Message:   "Username is available",
	}), nil
}
