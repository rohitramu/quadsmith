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

	"connectrpc.com/connect"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"quadsmith/shared/internal/cel2sql"
	"quadsmith/shared/pkg/blob"
	"quadsmith/shared/pkg/email"
	"quadsmith/shared/pkg/pagination"

	"google.golang.org/protobuf/types/known/timestamppb"
	pb "quadsmith/shared/gen/quadsmith/api"
	"quadsmith/shared/gen/quadsmith/api/apiconnect"
	"quadsmith/shared/gen/quadsmith/config"
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
	OAuth            *config.AuthOAuthConfig
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
		OAuth:            p.GetOauth(),
	}
}

type AuthService struct {
	apiconnect.UnimplementedAuthServiceHandler
	db          *pgxpool.Pool
	cfg         *AuthConfig
	emailSender email.EmailSender
	oauthClient OAuthClient
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

	if db != nil {
		_, _ = EnsureSystemUser(context.Background(), db)
	}

	return &AuthService{
		db:          db,
		cfg:         authCfg,
		emailSender: sender,
		oauthClient: NewOAuthClient(authCfg.OAuth),
	}
}

// SetEmailSender overrides the active EmailSender implementation (useful for tests).
func (s *AuthService) SetEmailSender(sender email.EmailSender) {
	s.emailSender = sender
}

// SetOAuthClient overrides the active OAuthClient implementation (useful for tests).
func (s *AuthService) SetOAuthClient(client OAuthClient) {
	s.oauthClient = client
}

func purposeToDBString(p pb.EmailOtpPurpose) string {
	switch p {
	case pb.EmailOtpPurpose_EMAIL_OTP_PURPOSE_ADD_EMAIL:
		return "add_email"
	case pb.EmailOtpPurpose_EMAIL_OTP_PURPOSE_EXPORT_DATA:
		return "export_data"
	case pb.EmailOtpPurpose_EMAIL_OTP_PURPOSE_DELETE_ACCOUNT:
		return "delete_account"
	default:
		return "login"
	}
}

func purposeFromDBString(s string) pb.EmailOtpPurpose {
	switch s {
	case "add_email":
		return pb.EmailOtpPurpose_EMAIL_OTP_PURPOSE_ADD_EMAIL
	case "export_data":
		return pb.EmailOtpPurpose_EMAIL_OTP_PURPOSE_EXPORT_DATA
	case "delete_account":
		return pb.EmailOtpPurpose_EMAIL_OTP_PURPOSE_DELETE_ACCOUNT
	default:
		return pb.EmailOtpPurpose_EMAIL_OTP_PURPOSE_LOGIN
	}
}

func (s *AuthService) SendEmailOtp(ctx context.Context, req *connect.Request[pb.SendEmailOtpRequest]) (*connect.Response[pb.SendEmailOtpResponse], error) {
	purpose := req.Msg.GetPurpose()
	if purpose == pb.EmailOtpPurpose_EMAIL_OTP_PURPOSE_UNSPECIFIED {
		purpose = pb.EmailOtpPurpose_EMAIL_OTP_PURPOSE_LOGIN
	}

	purposeStr := purposeToDBString(purpose)
	email := strings.ToLower(strings.TrimSpace(req.Msg.GetEmail()))

	userCtx := UserContextFromContext(ctx)

	// If purpose is export_data or delete_account and email is empty, resolve primary email
	if purpose == pb.EmailOtpPurpose_EMAIL_OTP_PURPOSE_EXPORT_DATA || purpose == pb.EmailOtpPurpose_EMAIL_OTP_PURPOSE_DELETE_ACCOUNT {
		if userCtx == nil {
			return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("authentication required"))
		}
		if purpose == pb.EmailOtpPurpose_EMAIL_OTP_PURPOSE_DELETE_ACCOUNT && (userCtx.IsSystem() || strings.EqualFold(userCtx.UserID, SystemUserHandle)) {
			return nil, connect.NewError(connect.CodePermissionDenied, errors.New("the @quadsmith system account cannot be deleted"))
		}
		if email == "" {
			_ = s.db.QueryRow(ctx, `
				SELECT COALESCE(ue.email, '')
				FROM users u
				LEFT JOIN user_emails ue ON ue.user_uuid = u.uuid AND ue.is_primary = true
				WHERE u.uuid = $1 AND u.deleted_at IS NULL`, userCtx.UserUUID).Scan(&email)
			if email == "" && userCtx.Email != "" {
				email = userCtx.Email
			}
			if email == "" {
				return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("no email found on account"))
			}
		}
	}

	if purpose == pb.EmailOtpPurpose_EMAIL_OTP_PURPOSE_ADD_EMAIL {
		if userCtx == nil {
			return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("authentication required"))
		}
		if email == "" || !strings.Contains(email, "@") {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("valid email address required"))
		}
		var alreadyAssociated bool
		err := s.db.QueryRow(ctx, `
			SELECT EXISTS(
				SELECT 1 FROM user_emails
				WHERE user_uuid = $1 AND LOWER(email) = LOWER($2)
			)`, userCtx.UserUUID, email).Scan(&alreadyAssociated)
		if err == nil && alreadyAssociated {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("this email address is already associated with your account"))
		}
	}

	if purpose == pb.EmailOtpPurpose_EMAIL_OTP_PURPOSE_LOGIN {
		emailRegex := regexp.MustCompile(`^[a-z0-9._%+\-]+@[a-z0-9.\-]+\.[a-z]{2,}$`)
		if !emailRegex.MatchString(email) {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("please enter a valid email address"))
		}

		// Check if email address is suspended and has no active account
		var isEmailBanned bool
		_ = s.db.QueryRow(ctx, `
			SELECT EXISTS (
				SELECT 1 FROM banned_emails be
				JOIN bans b ON b.uuid = be.ban_uuid
				WHERE LOWER(be.email) = LOWER($1)
				  AND b.revoked_at IS NULL
				  AND (b.expires_at IS NULL OR b.expires_at > NOW())
			)`, email).Scan(&isEmailBanned)

		if isEmailBanned {
			var hasActiveUser bool
			_ = s.db.QueryRow(ctx, `
				SELECT EXISTS (
					SELECT 1 FROM user_emails ue
					JOIN users u ON u.uuid = ue.user_uuid
					WHERE LOWER(ue.email) = LOWER($1)
					  AND (u.deleted_at IS NULL OR u.deleted_at + INTERVAL '30 days' > NOW())
				)`, email).Scan(&hasActiveUser)
			if !hasActiveUser {
				return nil, connect.NewError(connect.CodePermissionDenied, errors.New("email address is suspended"))
			}
		}
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

	// Invalidate any existing active challenges for this email and purpose
	if userCtx != nil && (purpose == pb.EmailOtpPurpose_EMAIL_OTP_PURPOSE_EXPORT_DATA || purpose == pb.EmailOtpPurpose_EMAIL_OTP_PURPOSE_DELETE_ACCOUNT) {
		_, _ = s.db.Exec(ctx, `
			UPDATE email_otps
			SET used_at = NOW()
			WHERE (email = $1 OR requested_by_user_uuid = $2) AND purpose = $3 AND used_at IS NULL`,
			email, userCtx.UserUUID, purposeStr)
	} else {
		_, _ = s.db.Exec(ctx, `UPDATE email_otps SET used_at = NOW() WHERE email = $1 AND purpose = $2 AND used_at IS NULL`, email, purposeStr)
	}

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

	var reqUserUUID *string
	if userCtx != nil {
		u := userCtx.UserUUID
		reqUserUUID = &u
	}

	_, err = s.db.Exec(ctx, `
		INSERT INTO email_otps (uuid, email, challenge_id_hash, code_hash, purpose, requested_by_user_uuid, expires_at, created_at, failed_attempts, ip_address, user_agent, location_hint, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, 0, $9, $10, $11, 'pending')`,
		otpUUID, email, challengeHash, codeHash, purposeStr, reqUserUUID, expiresAt, now, clientIP, userAgent, locationHint)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to save otp challenge: %w", err))
	}

	requesterUUIDStr := ""
	if userCtx != nil {
		requesterUUIDStr = userCtx.UserUUID
	}
	magicToken, err := SignMagicLinkToken(otpUUID, email, purposeStr, requesterUUIDStr, expiresAt)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to generate magic link token: %w", err))
	}

	baseURL := s.getBaseURL(req)
	magicLinkURL := fmt.Sprintf("%s/login?token=%s", baseURL, magicToken)
	var emailAction string
	var successMessage string

	switch purpose {
	case pb.EmailOtpPurpose_EMAIL_OTP_PURPOSE_ADD_EMAIL:
		emailAction = "add_email"
		successMessage = "Verification code sent to email"
	case pb.EmailOtpPurpose_EMAIL_OTP_PURPOSE_EXPORT_DATA:
		emailAction = "GDPR Data Archive Export"
		successMessage = "Data export verification code and magic link sent to your primary email"
	case pb.EmailOtpPurpose_EMAIL_OTP_PURPOSE_DELETE_ACCOUNT:
		emailAction = "Account Deletion Confirmation"
		successMessage = "Account deletion confirmation code sent to your primary email"
	default:
		emailAction = "login"
		successMessage = "Verification code sent to your email"
	}

	ctxTimeout, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if s.emailSender != nil {
		if err := s.emailSender.SendVerification(ctxTimeout, email, otpCode, magicLinkURL, emailAction, s.cfg.OtpTTL); err != nil {
			return nil, connect.NewError(connect.CodeUnavailable, fmt.Errorf("unable to deliver verification email: %w", err))
		}
	} else if s.cfg.DevPrintOTP {
		fmt.Printf("\n=======================================================\n")
		fmt.Printf("[AUTH] Email OTP for %s: %s (valid for %v)\n", email, otpCode, s.cfg.OtpTTL)
		fmt.Printf("[AUTH] Purpose: %s\n", purposeStr)
		fmt.Printf("[AUTH] Challenge Token: %s\n", challengeToken)
		fmt.Printf("[AUTH] Magic Link: %s (valid for %v)\n", magicLinkURL, s.cfg.MagicLinkTTL)
		fmt.Printf("=======================================================\n\n")
	}

	exists := false
	if purpose == pb.EmailOtpPurpose_EMAIL_OTP_PURPOSE_LOGIN {
		exists, err = s.userExistsForEmail(ctx, email)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
	}

	res := connect.NewResponse(&pb.SendEmailOtpResponse{
		Success:           true,
		Message:           successMessage,
		CooldownSeconds:   int32(s.cfg.EmailCooldown.Seconds()),
		ChallengeId:       &challengeToken,
		IsExistingAccount: exists,
		Email:             email,
		ExpiresAt:         timestamppb.New(expiresAt),
	})
	s.setChallengeCookie(res.Header(), challengeToken)
	return res, nil
}

func (s *AuthService) VerifyEmailOtp(ctx context.Context, req *connect.Request[pb.VerifyEmailOtpRequest]) (*connect.Response[pb.VerifyEmailOtpResponse], error) {
	token := strings.TrimSpace(req.Msg.GetToken())
	email := strings.ToLower(strings.TrimSpace(req.Msg.GetEmail()))
	code := strings.TrimSpace(req.Msg.GetCode())
	reqPurpose := req.Msg.GetPurpose()

	userCtx := UserContextFromContext(ctx)
	clientIP := getClientIP(req)

	if token == "" && (email == "" || code == "") {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("email and 6-digit code (or magic link token) are required"))
	}

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

	var otpUUID string
	var purposeStr string
	var effectivePurpose pb.EmailOtpPurpose

	if token != "" {
		claims, err := VerifyMagicLinkToken(token)
		if err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid or expired magic link: %w", err))
		}

		purposeStr = claims.Purpose
		effectivePurpose = purposeFromDBString(purposeStr)
		if reqPurpose != pb.EmailOtpPurpose_EMAIL_OTP_PURPOSE_UNSPECIFIED && effectivePurpose != reqPurpose {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("invalid magic link purpose"))
		}

		if (purposeStr == "add_email" || purposeStr == "export_data" || purposeStr == "delete_account") && userCtx != nil {
			if claims.RequesterUserUUID != "" && claims.RequesterUserUUID != userCtx.UserUUID {
				return nil, connect.NewError(connect.CodePermissionDenied, errors.New("this magic link was issued to a different account session"))
			}
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
			return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("this magic link has already been used. Please request a new code"))
		}
		if time.Since(createdAt) > s.cfg.MagicLinkTTL {
			_, _ = s.db.Exec(ctx, `UPDATE email_otps SET used_at = NOW() WHERE uuid = $1`, otpUUID)
			return nil, connect.NewError(connect.CodeDeadlineExceeded, errors.New("magic link has expired. Please request a new code"))
		}
	} else {
		// Code verification
		effectivePurpose = reqPurpose
		if effectivePurpose == pb.EmailOtpPurpose_EMAIL_OTP_PURPOSE_UNSPECIFIED {
			effectivePurpose = pb.EmailOtpPurpose_EMAIL_OTP_PURPOSE_LOGIN
		}
		purposeStr = purposeToDBString(effectivePurpose)

		challengeID := extractChallengeID(req.Msg.GetChallengeId(), req.Header())
		if challengeID == "" {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("challenge session expired or missing. Please request a new verification code"))
		}
		challengeHash := HashToken(challengeID)

		var codeHash string
		var failedAttempts int32
		var createdAt time.Time

		var err error
		if userCtx != nil && (purposeStr == "export_data" || purposeStr == "delete_account") {
			err = s.db.QueryRow(ctx, `
				UPDATE email_otps
				SET failed_attempts = failed_attempts + 1
				WHERE challenge_id_hash = $1 AND purpose = $2 AND requested_by_user_uuid = $3 AND used_at IS NULL AND failed_attempts < $4
				RETURNING uuid, code_hash, failed_attempts, created_at`,
				challengeHash, purposeStr, userCtx.UserUUID, MaxOtpAttempts).Scan(&otpUUID, &codeHash, &failedAttempts, &createdAt)
		} else if userCtx != nil && purposeStr == "add_email" {
			err = s.db.QueryRow(ctx, `
				UPDATE email_otps
				SET failed_attempts = failed_attempts + 1
				WHERE email = $1 AND challenge_id_hash = $2 AND purpose = $3 AND requested_by_user_uuid = $4 AND used_at IS NULL AND failed_attempts < $5
				RETURNING uuid, code_hash, failed_attempts, created_at`,
				email, challengeHash, purposeStr, userCtx.UserUUID, MaxOtpAttempts).Scan(&otpUUID, &codeHash, &failedAttempts, &createdAt)
		} else {
			err = s.db.QueryRow(ctx, `
				UPDATE email_otps
				SET failed_attempts = failed_attempts + 1
				WHERE email = $1 AND challenge_id_hash = $2 AND purpose = $3 AND used_at IS NULL AND failed_attempts < $4
				RETURNING uuid, code_hash, failed_attempts, created_at`,
				email, challengeHash, purposeStr, MaxOtpAttempts).Scan(&otpUUID, &codeHash, &failedAttempts, &createdAt)
		}

		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				var existingUUID string
				var existingFailures int32
				var existingUsedAt *time.Time
				checkErr := s.db.QueryRow(ctx, `
					SELECT uuid, failed_attempts, used_at
					FROM email_otps
					WHERE challenge_id_hash = $1 AND purpose = $2
					ORDER BY created_at DESC
					LIMIT 1`, challengeHash, purposeStr).Scan(&existingUUID, &existingFailures, &existingUsedAt)
				if checkErr == nil {
					if existingUsedAt != nil {
						return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("this verification challenge has already been used or expired. Please request a new code"))
					}
					if existingFailures >= MaxOtpAttempts {
						_, _ = s.db.Exec(ctx, `UPDATE email_otps SET used_at = NOW() WHERE uuid = $1`, existingUUID)
						return s.returnOtpError("Verification challenge exhausted. Please request a new code after the cooldown period.", 0, true, int32(ExhaustedCooldown.Seconds()))
					}
				}
				return nil, connect.NewError(connect.CodeNotFound, errors.New("no active verification challenge found. Please request a new code"))
			}
			return nil, connect.NewError(connect.CodeInternal, err)
		}

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
	}

	// Verification succeeded: mark challenge consumed and approved
	RecordIPSuccess(clientIP)
	_, _ = s.db.Exec(ctx, `UPDATE email_otps SET used_at = NOW(), status = 'approved', approved_at = NOW() WHERE uuid = $1`, otpUUID)

	switch effectivePurpose {
	case pb.EmailOtpPurpose_EMAIL_OTP_PURPOSE_ADD_EMAIL:
		if userCtx == nil {
			return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("authentication required"))
		}
		verifyResp, err := s.completeAddEmailVerification(ctx, userCtx.UserUUID, email)
		if err != nil {
			return nil, err
		}
		return connect.NewResponse(verifyResp), nil

	case pb.EmailOtpPurpose_EMAIL_OTP_PURPOSE_EXPORT_DATA:
		if userCtx == nil {
			return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("authentication required"))
		}
		var username string
		err := s.db.QueryRow(ctx, `SELECT id FROM users WHERE uuid = $1 AND deleted_at IS NULL`, userCtx.UserUUID).Scan(&username)
		if err != nil {
			return nil, connect.NewError(connect.CodeNotFound, errors.New("active user not found"))
		}

		exportToken, err := SignExportToken(userCtx.UserUUID, username)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to sign export token: %w", err))
		}

		dateStr := time.Now().UTC().Format("2006-01-02")
		filename := fmt.Sprintf("quadsmith-export-%s-%s.zip", username, dateStr)
		downloadURL := fmt.Sprintf("/api/account/export?token=%s", exportToken)
		exportExpiresAt := time.Now().Add(ExportTokenTTL)

		return connect.NewResponse(&pb.VerifyEmailOtpResponse{
			Success:         true,
			Purpose:         pb.EmailOtpPurpose_EMAIL_OTP_PURPOSE_EXPORT_DATA,
			Email:           &email,
			Message:         "Data export authorized",
			ExportToken:     &exportToken,
			DownloadUrl:     &downloadURL,
			Filename:        &filename,
			ExportExpiresAt: timestamppb.New(exportExpiresAt),
		}), nil

	case pb.EmailOtpPurpose_EMAIL_OTP_PURPOSE_DELETE_ACCOUNT:
		if userCtx == nil {
			return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("authentication required"))
		}
		if userCtx.IsSystem() || strings.EqualFold(userCtx.UserID, SystemUserHandle) || userCtx.UserUUID == SystemUserUUID {
			return nil, connect.NewError(connect.CodePermissionDenied, errors.New("the @quadsmith system account cannot be deleted"))
		}
		now := time.Now()
		purgeAt := now.Add(30 * 24 * time.Hour)
		_, err := s.db.Exec(ctx, `UPDATE users SET deleted_at = $1 WHERE uuid = $2`, now, userCtx.UserUUID)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to schedule account deletion: %w", err))
		}
		_, _ = s.db.Exec(ctx, `DELETE FROM refresh_tokens WHERE user_uuid = $1`, userCtx.UserUUID)

		return connect.NewResponse(&pb.VerifyEmailOtpResponse{
			Success:          true,
			Purpose:          pb.EmailOtpPurpose_EMAIL_OTP_PURPOSE_DELETE_ACCOUNT,
			Email:            &email,
			Message:          "Account deletion scheduled. Your account will be permanently purged in 30 days.",
			ScheduledPurgeAt: timestamppb.New(purgeAt),
		}), nil

	default: // LOGIN
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
			return connect.NewResponse(&pb.VerifyEmailOtpResponse{
				Success:              true,
				Purpose:              pb.EmailOtpPurpose_EMAIL_OTP_PURPOSE_LOGIN,
				Email:                &email,
				AccountSetupRequired: true,
				SetupToken:           setupToken,
			}), nil
		}

		_, _ = s.db.Exec(ctx, `UPDATE user_emails SET verified_via_otp = true WHERE LOWER(email) = LOWER($1)`, email)

		locationHint := extractLocationHint(req.Header(), clientIP)
		authRes, err := s.issueAuthSession(ctx, email, req.Header().Get("User-Agent"), clientIP, locationHint)
		if err != nil {
			return nil, err
		}

		res := connect.NewResponse(&pb.VerifyEmailOtpResponse{
			Success:              true,
			Purpose:              pb.EmailOtpPurpose_EMAIL_OTP_PURPOSE_LOGIN,
			Email:                &email,
			AccessToken:          authRes.Msg.AccessToken,
			RefreshToken:         authRes.Msg.RefreshToken,
			User:                 authRes.Msg.User,
			Profile:              authRes.Msg.Profile,
			AccountSetupRequired: authRes.Msg.AccountSetupRequired,
			SetupToken:           authRes.Msg.SetupToken,
			AccountStatus:        authRes.Msg.AccountStatus,
			ActiveBan:            authRes.Msg.ActiveBan,
			ScheduledPurgeAt:     authRes.Msg.ScheduledPurgeAt,
		})
		for k, v := range authRes.Header() {
			res.Header()[k] = v
		}
		return res, nil
	}
}

func (s *AuthService) completeAddEmailVerification(ctx context.Context, userUUID, email string) (*pb.VerifyEmailOtpResponse, error) {
	if userUUID == "" {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("authentication required"))
	}
	var otherUserUUID, otherUsername string
	err := s.db.QueryRow(ctx, `
		SELECT u.uuid, u.id
		FROM user_emails ue
		JOIN users u ON u.uuid = ue.user_uuid
		WHERE LOWER(ue.email) = LOWER($1) AND u.uuid != $2 AND (u.deleted_at IS NULL OR u.deleted_at + INTERVAL '30 days' > NOW())
		LIMIT 1`, email, userUUID).Scan(&otherUserUUID, &otherUsername)

	if err == nil && otherUserUUID != "" {
		mergeTicket, err := SignMergeTicket(userUUID, otherUserUUID, email, time.Now().Add(15*time.Minute))
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to sign merge ticket: %w", err))
		}

		var buildCount int
		_ = s.db.QueryRow(ctx, `SELECT COUNT(*) FROM entities WHERE user_uuid = $1 AND collection = 'builds' AND is_archived = false`, otherUserUUID).Scan(&buildCount)

		var buildNames []string
		rows, err := s.db.Query(ctx, `SELECT name FROM entities WHERE user_uuid = $1 AND collection = 'builds' AND is_archived = false ORDER BY name ASC LIMIT 10`, otherUserUUID)
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
		mailRows, err := s.db.Query(ctx, `SELECT email FROM user_emails WHERE user_uuid = $1 ORDER BY email ASC`, otherUserUUID)
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

		return &pb.VerifyEmailOtpResponse{
			Success:             true,
			Purpose:             pb.EmailOtpPurpose_EMAIL_OTP_PURPOSE_ADD_EMAIL,
			Email:               &email,
			MergeRequired:       true,
			OtherUserUsername:   otherUsername,
			OtherBuildCount:     int32(buildCount),
			OtherBuildNames:     buildNames,
			OtherEmails:         otherEmails,
			OtherOauthProviders: oauthProviders,
			MergeTicket:         &mergeTicket,
			Message:             fmt.Sprintf("This email belongs to account @%s (%d builds). Merging required to transfer data.", otherUsername, buildCount),
		}, nil
	}

	newUUID := NewUUIDv7()
	_, err = s.db.Exec(ctx, `
		INSERT INTO user_emails (uuid, user_uuid, email, is_primary, verified_via_otp, created_at)
		VALUES ($1, $2, $3, false, true, NOW())
		ON CONFLICT (email) DO UPDATE SET user_uuid = $2, is_primary = false, verified_via_otp = true`,
		newUUID, userUUID, email)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to add email: %w", err))
	}

	return &pb.VerifyEmailOtpResponse{
		Success:       true,
		Purpose:       pb.EmailOtpPurpose_EMAIL_OTP_PURPOSE_ADD_EMAIL,
		Email:         &email,
		MergeRequired: false,
		Message:       "Email successfully added to your account",
	}, nil
}

func (s *AuthService) VerifyMagicLink(ctx context.Context, req *connect.Request[pb.VerifyMagicLinkRequest]) (*connect.Response[pb.VerifyEmailOtpResponse], error) {
	token := strings.TrimSpace(req.Msg.GetToken())
	if token == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("magic link token required"))
	}
	verifyReq := connect.NewRequest(&pb.VerifyEmailOtpRequest{
		Token: &token,
	})
	for k, v := range req.Header() {
		verifyReq.Header()[k] = v
	}
	return s.VerifyEmailOtp(ctx, verifyReq)
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

	var email, ipAddress, userAgent, locationHint, statusStr, purposeStr string
	var createdAt, expiresAt time.Time
	var usedAt *time.Time

	err = s.db.QueryRow(ctx, `
		SELECT email, ip_address, user_agent, location_hint, status, purpose, created_at, expires_at, used_at
		FROM email_otps
		WHERE uuid = $1`, claims.OtpUUID).Scan(&email, &ipAddress, &userAgent, &locationHint, &statusStr, &purposeStr, &createdAt, &expiresAt, &usedAt)
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

	purpose := pb.EmailOtpPurpose_EMAIL_OTP_PURPOSE_LOGIN
	switch purposeStr {
	case "add_email":
		purpose = pb.EmailOtpPurpose_EMAIL_OTP_PURPOSE_ADD_EMAIL
	case "export_data":
		purpose = pb.EmailOtpPurpose_EMAIL_OTP_PURPOSE_EXPORT_DATA
	case "delete_account":
		purpose = pb.EmailOtpPurpose_EMAIL_OTP_PURPOSE_DELETE_ACCOUNT
	default:
		purpose = pb.EmailOtpPurpose_EMAIL_OTP_PURPOSE_LOGIN
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
		Purpose:      purpose,
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

	var email, purposeStr string
	var reqUserUUID *string
	err = s.db.QueryRow(ctx, `
		SELECT email, purpose, requested_by_user_uuid
		FROM email_otps
		WHERE uuid = $1 AND status = 'pending' AND used_at IS NULL AND expires_at > NOW()`, claims.OtpUUID).Scan(&email, &purposeStr, &reqUserUUID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("login challenge is no longer valid, expired, or already answered"))
		}
		return nil, connect.NewError(connect.CodeInternal, err)
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

	var verifyResp *pb.VerifyEmailOtpResponse
	if purposeStr == "export_data" && reqUserUUID != nil && *reqUserUUID != "" {
		var username string
		_ = s.db.QueryRow(ctx, `SELECT id FROM users WHERE uuid = $1 AND deleted_at IS NULL`, *reqUserUUID).Scan(&username)
		if username != "" {
			if exportToken, err := SignExportToken(*reqUserUUID, username); err == nil {
				dateStr := time.Now().UTC().Format("2006-01-02")
				filename := fmt.Sprintf("quadsmith-export-%s-%s.zip", username, dateStr)
				downloadURL := fmt.Sprintf("/api/account/export?token=%s", exportToken)
				expAt := time.Now().Add(ExportTokenTTL)
				verifyResp = &pb.VerifyEmailOtpResponse{
					Success:         true,
					Purpose:         pb.EmailOtpPurpose_EMAIL_OTP_PURPOSE_EXPORT_DATA,
					Email:           &email,
					Message:         "Data export authorized",
					ExportToken:     &exportToken,
					DownloadUrl:     &downloadURL,
					Filename:        &filename,
					ExportExpiresAt: timestamppb.New(expAt),
				}
			}
		}
	}

	return connect.NewResponse(&pb.ApproveLoginChallengeResponse{
		Success:        true,
		Message:        "Challenge approved",
		VerifyResponse: verifyResp,
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

	var otpUUID, email, statusStr, purposeStr string
	var reqUserUUID *string
	var expiresAt time.Time
	var usedAt *time.Time

	err := s.db.QueryRow(ctx, `
		SELECT uuid, email, status, purpose, requested_by_user_uuid, expires_at, used_at
		FROM email_otps
		WHERE challenge_id_hash = $1
		ORDER BY created_at DESC
		LIMIT 1`, challengeHash).Scan(&otpUUID, &email, &statusStr, &purposeStr, &reqUserUUID, &expiresAt, &usedAt)
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

		userCtx := UserContextFromContext(ctx)
		userUUID := ""
		if reqUserUUID != nil && *reqUserUUID != "" {
			userUUID = *reqUserUUID
		} else if userCtx != nil {
			userUUID = userCtx.UserUUID
		}

		switch purposeStr {
		case "export_data":
			var username string
			err := s.db.QueryRow(ctx, `SELECT id FROM users WHERE uuid = $1 AND deleted_at IS NULL`, userUUID).Scan(&username)
			if err != nil {
				return nil, connect.NewError(connect.CodeNotFound, errors.New("active user not found"))
			}

			exportToken, err := SignExportToken(userUUID, username)
			if err != nil {
				return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to sign export token: %w", err))
			}

			dateStr := time.Now().UTC().Format("2006-01-02")
			filename := fmt.Sprintf("quadsmith-export-%s-%s.zip", username, dateStr)
			downloadURL := fmt.Sprintf("/api/account/export?token=%s", exportToken)
			exportExpiresAt := time.Now().Add(ExportTokenTTL)

			verifyResp := &pb.VerifyEmailOtpResponse{
				Success:         true,
				Purpose:         pb.EmailOtpPurpose_EMAIL_OTP_PURPOSE_EXPORT_DATA,
				Email:           &email,
				Message:         "Data export authorized",
				ExportToken:     &exportToken,
				DownloadUrl:     &downloadURL,
				Filename:        &filename,
				ExportExpiresAt: timestamppb.New(exportExpiresAt),
			}
			return connect.NewResponse(&pb.GetLoginChallengeStatusResponse{
				Status:         pb.LoginChallengeStatus_LOGIN_CHALLENGE_STATUS_APPROVED,
				VerifyResponse: verifyResp,
			}), nil

		case "add_email":
			verifyResp, err := s.completeAddEmailVerification(ctx, userUUID, email)
			if err != nil {
				return nil, err
			}
			return connect.NewResponse(&pb.GetLoginChallengeStatusResponse{
				Status:         pb.LoginChallengeStatus_LOGIN_CHALLENGE_STATUS_APPROVED,
				VerifyResponse: verifyResp,
			}), nil

		case "delete_account":
			if userUUID == SystemUserUUID || (userCtx != nil && (userCtx.IsSystem() || strings.EqualFold(userCtx.UserID, SystemUserHandle))) {
				return nil, connect.NewError(connect.CodePermissionDenied, errors.New("the @quadsmith system account cannot be deleted"))
			}
			now := time.Now()
			purgeAt := now.Add(30 * 24 * time.Hour)
			_, err = s.db.Exec(ctx, `UPDATE users SET deleted_at = $1 WHERE uuid = $2`, now, userUUID)
			if err != nil {
				return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to schedule account deletion: %w", err))
			}
			_, _ = s.db.Exec(ctx, `DELETE FROM refresh_tokens WHERE user_uuid = $1`, userUUID)

			verifyResp := &pb.VerifyEmailOtpResponse{
				Success:          true,
				Purpose:          pb.EmailOtpPurpose_EMAIL_OTP_PURPOSE_DELETE_ACCOUNT,
				Email:            &email,
				Message:          "Account deletion scheduled. Your account will be permanently purged in 30 days.",
				ScheduledPurgeAt: timestamppb.New(purgeAt),
			}
			return connect.NewResponse(&pb.GetLoginChallengeStatusResponse{
				Status:         pb.LoginChallengeStatus_LOGIN_CHALLENGE_STATUS_APPROVED,
				VerifyResponse: verifyResp,
			}), nil

		default: // LOGIN
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
				verifyResp := &pb.VerifyEmailOtpResponse{
					Success:              true,
					Purpose:              pb.EmailOtpPurpose_EMAIL_OTP_PURPOSE_LOGIN,
					Email:                &email,
					AccountSetupRequired: true,
					SetupToken:           setupToken,
				}
				return connect.NewResponse(&pb.GetLoginChallengeStatusResponse{
					Status: pb.LoginChallengeStatus_LOGIN_CHALLENGE_STATUS_APPROVED,
					AuthResponse: &pb.AuthResponse{
						AccountSetupRequired: true,
						SetupToken:           setupToken,
					},
					VerifyResponse: verifyResp,
				}), nil
			}

			if purposeStr != "cli_oauth" {
				_, _ = s.db.Exec(ctx, `UPDATE user_emails SET verified_via_otp = true WHERE LOWER(email) = LOWER($1)`, email)
			}

			locationHint := extractLocationHint(req.Header(), clientIP)
			authRes, err := s.issueAuthSession(ctx, email, req.Header().Get("User-Agent"), clientIP, locationHint)
			if err != nil {
				return nil, err
			}

			verifyResp := &pb.VerifyEmailOtpResponse{
				Success:              true,
				Purpose:              pb.EmailOtpPurpose_EMAIL_OTP_PURPOSE_LOGIN,
				Email:                &email,
				AccessToken:          authRes.Msg.AccessToken,
				RefreshToken:         authRes.Msg.RefreshToken,
				User:                 authRes.Msg.User,
				Profile:              authRes.Msg.Profile,
				AccountSetupRequired: authRes.Msg.AccountSetupRequired,
				SetupToken:           authRes.Msg.SetupToken,
				AccountStatus:        authRes.Msg.AccountStatus,
				ActiveBan:            authRes.Msg.ActiveBan,
				ScheduledPurgeAt:     authRes.Msg.ScheduledPurgeAt,
			}
			res := connect.NewResponse(&pb.GetLoginChallengeStatusResponse{
				Status:         pb.LoginChallengeStatus_LOGIN_CHALLENGE_STATUS_APPROVED,
				AuthResponse:   authRes.Msg,
				VerifyResponse: verifyResp,
			})
			for _, cookie := range authRes.Header().Values("Set-Cookie") {
				res.Header().Add("Set-Cookie", cookie)
			}
			return res, nil
		}

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
	claims, err := VerifySetupTokenClaims(setupToken)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("invalid or expired setup token: %w", err))
	}
	email := claims.Email

	checkEmails := []string{strings.ToLower(email)}
	for _, ve := range claims.VerifiedEmails {
		veNorm := strings.ToLower(strings.TrimSpace(ve))
		if veNorm != "" && !slices.Contains(checkEmails, veNorm) {
			checkEmails = append(checkEmails, veNorm)
		}
	}

	var isEmailSuspended bool
	_ = s.db.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM banned_emails be
			JOIN bans b ON b.uuid = be.ban_uuid
			WHERE LOWER(be.email) = ANY($1)
			  AND b.revoked_at IS NULL
			  AND (b.expires_at IS NULL OR b.expires_at > NOW())
		)`, checkEmails).Scan(&isEmailSuspended)
	if isEmailSuspended {
		return nil, connect.NewError(connect.CodePermissionDenied, errors.New("this email address is suspended from creating a new account"))
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

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, `
		INSERT INTO users (uuid, id, created_at, updated_at)
		VALUES ($1, $2, $3, $3)`,
		userUUID, username, now)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to create user: %w", err))
	}

	profileUUID := NewUUIDv7()
	avatarURL := strings.TrimSpace(req.Msg.GetAvatarUrl())
	if avatarURL == "" && claims.AvatarURL != "" {
		avatarURL = claims.AvatarURL
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO user_profiles (uuid, user_uuid, avatar_url, bio, updated_at)
		VALUES ($1, $2, $3, '', $4)`,
		profileUUID, userUUID, avatarURL, now)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to create profile: %w", err))
	}

	verifiedViaOTP := claims.Provider == ""
	emailUUID := NewUUIDv7()
	_, err = tx.Exec(ctx, `
		INSERT INTO user_emails (uuid, user_uuid, email, is_primary, verified_via_otp, created_at)
		VALUES ($1, $2, $3, true, $4, $5)`,
		emailUUID, userUUID, email, verifiedViaOTP, now)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to create user email: %w", err))
	}

	// If signed up via OAuth, link the provider and secondary emails
	if claims.Provider != "" && claims.ProviderAccountID != "" {
		oauthUUID := NewUUIDv7()
		_, err = tx.Exec(ctx, `
			INSERT INTO oauth_accounts (uuid, user_uuid, provider, provider_account_id, created_at)
			VALUES ($1, $2, $3, $4, $5)`,
			oauthUUID, userUUID, claims.Provider, claims.ProviderAccountID, now)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to insert oauth account: %w", err))
		}

		linkUUID := NewUUIDv7()
		_, err = tx.Exec(ctx, `
			INSERT INTO user_email_oauth_accounts (uuid, user_email_uuid, oauth_account_uuid, created_at)
			VALUES ($1, $2, $3, $4)`,
			linkUUID, emailUUID, oauthUUID, now)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to link email to oauth account: %w", err))
		}

		for _, secEmail := range claims.VerifiedEmails {
			secNorm := strings.ToLower(strings.TrimSpace(secEmail))
			if secNorm == "" || strings.EqualFold(secNorm, email) {
				continue
			}
			secEmailUUID := NewUUIDv7()
			_, err = tx.Exec(ctx, `
				INSERT INTO user_emails (uuid, user_uuid, email, is_primary, verified_via_otp, created_at)
				VALUES ($1, $2, $3, false, false, $4)
				ON CONFLICT (email) DO NOTHING`,
				secEmailUUID, userUUID, secNorm, now)
			if err != nil {
				continue
			}
			secLinkUUID := NewUUIDv7()
			_, _ = tx.Exec(ctx, `
				INSERT INTO user_email_oauth_accounts (uuid, user_email_uuid, oauth_account_uuid, created_at)
				SELECT $1, uuid, $2, $3 FROM user_emails WHERE LOWER(email) = LOWER($4) AND user_uuid = $5
				ON CONFLICT (user_email_uuid, oauth_account_uuid) DO NOTHING`,
				secLinkUUID, oauthUUID, now, secNorm, userUUID)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to commit account creation: %w", err))
	}

	// If this was initiated from a CLI login challenge, mark the challenge approved
	if claims.ChallengeIDHash != "" {
		_, _ = s.db.Exec(ctx, `
			UPDATE email_otps
			SET status = 'approved', email = $1, used_at = NULL
			WHERE challenge_id_hash = $2 AND status = 'pending'`,
			email, claims.ChallengeIDHash)
	}

	clientIP := getClientIP(req)
	if err := CheckAndRecordIPSignIn(clientIP, email); err != nil {
		return nil, connect.NewError(connect.CodeResourceExhausted, err)
	}

	locationHint := extractLocationHint(req.Header(), clientIP)
	return s.issueAuthSession(ctx, email, req.Header().Get("User-Agent"), clientIP, locationHint)
}

func (s *AuthService) userExistsForEmail(ctx context.Context, email string) (bool, error) {
	var count int
	err := s.db.QueryRow(ctx, `
		SELECT COUNT(*) FROM user_emails ue
		JOIN users u ON u.uuid = ue.user_uuid
		WHERE LOWER(ue.email) = LOWER($1) AND (u.deleted_at IS NULL OR u.deleted_at + INTERVAL '30 days' > NOW())`, email).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func getActiveBanForUser(ctx context.Context, db *pgxpool.Pool, userUUID string) (*pb.Ban, error) {
	var ban pb.Ban
	var expiresAt, createdAt, revokedAt *time.Time
	var revokedByUserUUID, revokeReason *string
	err := db.QueryRow(ctx, `
		SELECT uuid, user_uuid, banned_by_user_uuid, reason, internal_notes, expires_at, created_at, revoked_at, revoked_by_user_uuid, revoke_reason
		FROM bans
		WHERE user_uuid = $1::uuid
		  AND revoked_at IS NULL
		  AND (expires_at IS NULL OR expires_at > NOW())
		ORDER BY created_at DESC
		LIMIT 1`, userUUID).Scan(
		&ban.Uuid, &ban.UserUuid, &ban.BannedByUserUuid, &ban.Reason, &ban.InternalNotes,
		&expiresAt, &createdAt, &revokedAt, &revokedByUserUUID, &revokeReason)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	if expiresAt != nil {
		ban.ExpiresAt = timestamppb.New(*expiresAt)
	}
	if createdAt != nil {
		ban.CreatedAt = timestamppb.New(*createdAt)
	}
	if revokedAt != nil {
		ban.RevokedAt = timestamppb.New(*revokedAt)
	}
	if revokedByUserUUID != nil {
		ban.RevokedByUserUuid = revokedByUserUUID
	}
	if revokeReason != nil {
		ban.RevokeReason = *revokeReason
	}
	return &ban, nil
}

func getActiveBanForEmail(ctx context.Context, db *pgxpool.Pool, email string) (*pb.Ban, error) {
	var ban pb.Ban
	var expiresAt, createdAt, revokedAt *time.Time
	var revokedByUserUUID, revokeReason *string
	err := db.QueryRow(ctx, `
		SELECT b.uuid, b.user_uuid, b.banned_by_user_uuid, b.reason, b.internal_notes, b.expires_at, b.created_at, b.revoked_at, b.revoked_by_user_uuid, b.revoke_reason
		FROM banned_emails be
		JOIN bans b ON b.uuid = be.ban_uuid
		WHERE LOWER(be.email) = LOWER($1)
		  AND b.revoked_at IS NULL
		  AND (b.expires_at IS NULL OR b.expires_at > NOW())
		ORDER BY b.created_at DESC
		LIMIT 1`, email).Scan(
		&ban.Uuid, &ban.UserUuid, &ban.BannedByUserUuid, &ban.Reason, &ban.InternalNotes,
		&expiresAt, &createdAt, &revokedAt, &revokedByUserUUID, &revokeReason)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	if expiresAt != nil {
		ban.ExpiresAt = timestamppb.New(*expiresAt)
	}
	if createdAt != nil {
		ban.CreatedAt = timestamppb.New(*createdAt)
	}
	if revokedAt != nil {
		ban.RevokedAt = timestamppb.New(*revokedAt)
	}
	if revokedByUserUUID != nil {
		ban.RevokedByUserUuid = revokedByUserUUID
	}
	if revokeReason != nil {
		ban.RevokeReason = *revokeReason
	}
	return &ban, nil
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
	clientIP := getClientIP(req)
	locationHint := extractLocationHint(req.Header(), clientIP)
	if clientIP != "" && locationHint != "" && locationHint != "Unknown Location" {
		_, err = s.db.Exec(ctx, `
			UPDATE refresh_tokens
			SET expires_at = $1, last_used_at = NOW(), ip_address = $3, location_hint = $4
			WHERE uuid = $2`, newExpiresAt, sessionUUID, clientIP, locationHint)
	} else if clientIP != "" {
		_, err = s.db.Exec(ctx, `
			UPDATE refresh_tokens
			SET expires_at = $1, last_used_at = NOW(), ip_address = $3
			WHERE uuid = $2`, newExpiresAt, sessionUUID, clientIP)
	} else {
		_, err = s.db.Exec(ctx, `
			UPDATE refresh_tokens
			SET expires_at = $1, last_used_at = NOW()
			WHERE uuid = $2`, newExpiresAt, sessionUUID)
	}
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	// Fetch User
	var user pb.User
	var username, email, role string
	var lastRenamedAt, deletedAt, createdAt, updatedAt *time.Time
	err = s.db.QueryRow(ctx, `
		SELECT u.uuid, u.id, COALESCE(ue.email, ''), u.role, u.last_renamed_at, u.deleted_at, u.created_at, u.updated_at
		FROM users u
		LEFT JOIN user_emails ue ON ue.user_uuid = u.uuid AND ue.is_primary = true
		WHERE u.uuid = $1`, userUUID).Scan(
		&user.Uuid, &username, &email, &role,
		&lastRenamedAt, &deletedAt, &createdAt, &updatedAt)
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

	activeBan, err := getActiveBanForUser(ctx, s.db, userUUID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if activeBan != nil {
		user.ActiveBan = activeBan
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

	accessToken, err := SignAccessToken(userUUID, username, email, sessionUUID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	var accountStatus *string
	var scheduledPurgeAt *timestamppb.Timestamp
	if activeBan != nil {
		status := "BANNED"
		accountStatus = &status
	} else if deletedAt != nil {
		status := "PENDING_DELETION"
		accountStatus = &status
		scheduledPurgeAt = timestamppb.New(deletedAt.Add(30 * 24 * time.Hour))
	}

	res := connect.NewResponse(&pb.AuthResponse{
		AccessToken:      accessToken,
		RefreshToken:     token,
		User:             &user,
		Profile:          &profile,
		AccountStatus:    accountStatus,
		ScheduledPurgeAt: scheduledPurgeAt,
		ActiveBan:        activeBan,
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
	var lastRenamedAt, deletedAt, createdAt, updatedAt *time.Time
	err := s.db.QueryRow(ctx, `
		SELECT u.uuid, u.id, COALESCE(ue.email, ''), u.role, u.last_renamed_at, u.deleted_at, u.created_at, u.updated_at
		FROM users u
		LEFT JOIN user_emails ue ON ue.user_uuid = u.uuid AND ue.is_primary = true
		WHERE u.uuid = $1`, userCtx.UserUUID).Scan(
		&user.Uuid, &user.Id, &user.Email, &user.Role,
		&lastRenamedAt, &deletedAt, &createdAt, &updatedAt)
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

	activeBan, err := getActiveBanForUser(ctx, s.db, userCtx.UserUUID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if activeBan != nil {
		user.ActiveBan = activeBan
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
		tx, txErr := s.db.Begin(ctx)
		if txErr == nil {
			_, _ = tx.Exec(ctx, `UPDATE user_profiles SET avatar_url = $1, updated_at = NOW() WHERE user_uuid = $2`, *req.Msg.AvatarUrl, userCtx.UserUUID)
			_ = blob.UnlinkEntityBlobs(ctx, tx, "user_avatar", userCtx.UserUUID)
			if *req.Msg.AvatarUrl != "" {
				_ = blob.LinkEntityBlobs(ctx, tx, "user_avatar", userCtx.UserUUID, []string{*req.Msg.AvatarUrl})
			}
			_ = tx.Commit(ctx)
		}
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

func (s *AuthService) ListActiveSessions(ctx context.Context, _ *connect.Request[pb.ListActiveSessionsRequest]) (*connect.Response[pb.ListActiveSessionsResponse], error) {
	userCtx := UserContextFromContext(ctx)
	if userCtx == nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("authentication required"))
	}

	rows, err := s.db.Query(ctx, `
		SELECT uuid, device_hint, ip_address, location_hint, created_at, last_used_at
		FROM refresh_tokens
		WHERE user_uuid = $1 AND (expires_at IS NULL OR expires_at > NOW())
		ORDER BY (uuid = $2) DESC, COALESCE(last_used_at, created_at) DESC`,
		userCtx.UserUUID, userCtx.SessionUUID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to query sessions: %w", err))
	}
	defer rows.Close()

	var sessions []*pb.Session
	for rows.Next() {
		var sessionUUID, deviceHint, ipAddress, locationHint string
		var createdAt, lastUsedAt *time.Time
		if err := rows.Scan(&sessionUUID, &deviceHint, &ipAddress, &locationHint, &createdAt, &lastUsedAt); err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to scan session: %w", err))
		}

		sess := &pb.Session{
			Uuid:         sessionUUID,
			DeviceHint:   deviceHint,
			IpAddress:    ipAddress,
			LocationHint: locationHint,
			IsCurrent:    userCtx.SessionUUID != "" && sessionUUID == userCtx.SessionUUID,
		}
		if createdAt != nil {
			sess.CreatedAt = timestamppb.New(*createdAt)
		}
		if lastUsedAt != nil {
			sess.LastActiveAt = timestamppb.New(*lastUsedAt)
		} else if createdAt != nil {
			sess.LastActiveAt = timestamppb.New(*createdAt)
		}
		sessions = append(sessions, sess)
	}
	if err := rows.Err(); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("error reading sessions: %w", err))
	}

	return connect.NewResponse(&pb.ListActiveSessionsResponse{
		Sessions: sessions,
	}), nil
}

func (s *AuthService) RevokeSession(ctx context.Context, req *connect.Request[pb.RevokeSessionRequest]) (*connect.Response[pb.RevokeSessionResponse], error) {
	userCtx := UserContextFromContext(ctx)
	if userCtx == nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("authentication required"))
	}

	sessionUUID := strings.TrimSpace(req.Msg.GetSessionUuid())
	if sessionUUID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("session_uuid is required"))
	}

	tag, err := s.db.Exec(ctx, `
		DELETE FROM refresh_tokens
		WHERE uuid = $1 AND user_uuid = $2`,
		sessionUUID, userCtx.UserUUID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to revoke session: %w", err))
	}
	if tag.RowsAffected() == 0 {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("session not found"))
	}

	res := connect.NewResponse(&pb.RevokeSessionResponse{
		Success: true,
		Message: "Session revoked successfully",
	})
	if userCtx.SessionUUID != "" && sessionUUID == userCtx.SessionUUID {
		clearCookie := &http.Cookie{
			Name:     "qs_session",
			Value:    "",
			Path:     "/",
			MaxAge:   -1,
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
		}
		res.Header().Set("Set-Cookie", clearCookie.String())
	}
	return res, nil
}

func (s *AuthService) RevokeOtherSessions(ctx context.Context, _ *connect.Request[pb.RevokeOtherSessionsRequest]) (*connect.Response[pb.RevokeOtherSessionsResponse], error) {
	userCtx := UserContextFromContext(ctx)
	if userCtx == nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("authentication required"))
	}

	var tag any
	var count int32
	if userCtx.SessionUUID != "" {
		cmdTag, err := s.db.Exec(ctx, `
			DELETE FROM refresh_tokens
			WHERE user_uuid = $1 AND uuid != $2`,
			userCtx.UserUUID, userCtx.SessionUUID)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to revoke other sessions: %w", err))
		}
		count = int32(cmdTag.RowsAffected())
	} else {
		cmdTag, err := s.db.Exec(ctx, `
			DELETE FROM refresh_tokens
			WHERE user_uuid = $1`,
			userCtx.UserUUID)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to revoke other sessions: %w", err))
		}
		count = int32(cmdTag.RowsAffected())
	}
	_ = tag

	return connect.NewResponse(&pb.RevokeOtherSessionsResponse{
		Success:      true,
		RevokedCount: count,
		Message:      fmt.Sprintf("%d other session(s) revoked successfully", count),
	}), nil
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

func (s *AuthService) MergeAccount(ctx context.Context, req *connect.Request[pb.MergeAccountRequest]) (*connect.Response[pb.MergeAccountResponse], error) {
	userCtx := UserContextFromContext(ctx)
	if userCtx == nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("authentication required"))
	}
	if userCtx.IsSystem() || strings.EqualFold(userCtx.UserID, SystemUserHandle) || userCtx.UserUUID == SystemUserUUID {
		return nil, connect.NewError(connect.CodePermissionDenied, errors.New("the @quadsmith system account cannot be merged"))
	}

	email := strings.ToLower(strings.TrimSpace(req.Msg.GetEmail()))
	ticket := strings.TrimSpace(req.Msg.GetMergeTicket())

	var otherUserUUID string
	var mergeClaims *MergeTicketClaims

	if ticket != "" {
		claims, err := VerifyMergeTicket(ticket)
		if err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid or expired merge ticket: %w", err))
		}
		if claims.SourceUserUUID != userCtx.UserUUID {
			return nil, connect.NewError(connect.CodePermissionDenied, errors.New("merge ticket was issued to a different user session"))
		}
		if claims.Provider == "" && email != "" && !strings.EqualFold(email, claims.Email) {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("merge ticket email mismatch"))
		}
		email = claims.Email
		otherUserUUID = claims.TargetUserUUID
		mergeClaims = claims
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
			FROM user_emails ue
			JOIN users u ON u.uuid = ue.user_uuid
			WHERE LOWER(ue.email) = LOWER($1) AND u.uuid != $2 AND u.deleted_at IS NULL
			LIMIT 1`, email, userCtx.UserUUID).Scan(&otherUserUUID)
		if err != nil {
			return nil, connect.NewError(connect.CodeNotFound, errors.New("no other account found for this email"))
		}
	}

	if activeBan, err := getActiveBanForUser(ctx, s.db, otherUserUUID); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	} else if activeBan != nil {
		return nil, connect.NewError(connect.CodePermissionDenied, errors.New("cannot merge a suspended account"))
	}

	var currentUsername string
	_ = s.db.QueryRow(ctx, `SELECT id FROM users WHERE uuid = $1`, userCtx.UserUUID).Scan(&currentUsername)

	var otherUsername, otherEmail, otherRole string
	err := s.db.QueryRow(ctx, `
		SELECT u.id, COALESCE(ue.email, ''), u.role
		FROM users u
		LEFT JOIN user_emails ue ON ue.user_uuid = u.uuid AND ue.is_primary = true
		WHERE u.uuid = $1`, otherUserUUID).Scan(&otherUsername, &otherEmail, &otherRole)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("account to merge was not found"))
	}
	if otherRole == RoleSystem || strings.EqualFold(otherUsername, SystemUserHandle) || otherUserUUID == SystemUserUUID || strings.EqualFold(currentUsername, SystemUserHandle) {
		return nil, connect.NewError(connect.CodePermissionDenied, errors.New("the @quadsmith system account cannot be merged"))
	}

	// Begin atomic merge transaction
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to begin transaction: %w", err))
	}
	defer tx.Rollback(ctx)

	// 1. Count builds to transfer
	var buildCount int
	_ = tx.QueryRow(ctx, `SELECT COUNT(*) FROM entities WHERE user_uuid = $1 AND collection = 'builds' AND is_archived = false`, otherUserUUID).Scan(&buildCount)

	// 2. Reassign builds and components to current user (updating author_or_manufacturer on builds)
	_, err = tx.Exec(ctx, `
		UPDATE entities
		SET user_uuid = $1,
		    author_or_manufacturer = CASE WHEN collection = 'builds' AND $3 != '' THEN $3 ELSE author_or_manufacturer END
		WHERE user_uuid = $2`,
		userCtx.UserUUID, otherUserUUID, currentUsername)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to transfer builds: %w", err))
	}

	// 3. Collect all email addresses belonging to the other account
	var transferredEmails []string
	mailRows, err := tx.Query(ctx, `
		SELECT DISTINCT LOWER(email) FROM user_emails WHERE user_uuid = $1`, otherUserUUID)
	if err == nil {
		defer mailRows.Close()
		for mailRows.Next() {
			var em string
			if err := mailRows.Scan(&em); err == nil && em != "" {
				transferredEmails = append(transferredEmails, em)
			}
		}
	}

	// 4. Reassign all emails in-place to the current user (preserving user_emails.uuid, verified_via_otp, and user_email_oauth_accounts)
	_, err = tx.Exec(ctx, `
		UPDATE user_emails
		SET user_uuid = $1, is_primary = false
		WHERE user_uuid = $2`,
		userCtx.UserUUID, otherUserUUID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to reassign user emails: %w", err))
	}

	if (mergeClaims == nil || mergeClaims.Provider == "") && email != "" {
		_, _ = tx.Exec(ctx, `
			UPDATE user_emails
			SET verified_via_otp = true
			WHERE user_uuid = $1 AND LOWER(email) = LOWER($2)`,
			userCtx.UserUUID, email)
	}

	// 5. Transfer OAuth accounts (re-pointing email links if both users linked the same provider)
	_, _ = tx.Exec(ctx, `
		UPDATE user_email_oauth_accounts uea
		SET oauth_account_uuid = oa_dest.uuid
		FROM oauth_accounts oa_src, oauth_accounts oa_dest
		WHERE uea.oauth_account_uuid = oa_src.uuid
		  AND oa_src.user_uuid = $2
		  AND oa_dest.user_uuid = $1
		  AND LOWER(oa_dest.provider) = LOWER(oa_src.provider)
		  AND NOT EXISTS (
		      SELECT 1 FROM user_email_oauth_accounts uea2
		      WHERE uea2.user_email_uuid = uea.user_email_uuid
		        AND uea2.oauth_account_uuid = oa_dest.uuid
		  )`,
		userCtx.UserUUID, otherUserUUID)

	_, _ = tx.Exec(ctx, `
		UPDATE oauth_accounts oa
		SET user_uuid = $1
		WHERE user_uuid = $2
		  AND NOT EXISTS (
		      SELECT 1 FROM oauth_accounts oa2
		      WHERE oa2.user_uuid = $1 AND LOWER(oa2.provider) = LOWER(oa.provider)
		  )`,
		userCtx.UserUUID, otherUserUUID)
	_, _ = tx.Exec(ctx, `DELETE FROM oauth_accounts WHERE user_uuid = $1`, otherUserUUID)

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

	// 7. Reassign blobs and unlink old user avatar blob links
	_ = blob.UnlinkEntityBlobs(ctx, tx, "user_avatar", otherUserUUID)
	_, _ = tx.Exec(ctx, `UPDATE blobs SET user_uuid = $1 WHERE user_uuid = $2`, userCtx.UserUUID, otherUserUUID)

	// 8. Reassign collections (merge duplicate system collections first)
	_, _ = tx.Exec(ctx, `
		INSERT INTO collection_items (uuid, collection_uuid, entity_uuid, created_at)
		SELECT gen_random_uuid(), c_dest.uuid, ci.entity_uuid, ci.created_at
		FROM collection_items ci
		JOIN collections c_src ON ci.collection_uuid = c_src.uuid
		JOIN collections c_dest ON c_dest.user_uuid = $1 AND c_dest.is_system_collection = true AND c_dest.name = c_src.name
		WHERE c_src.user_uuid = $2 AND c_src.is_system_collection = true
		ON CONFLICT (collection_uuid, entity_uuid) DO NOTHING`,
		userCtx.UserUUID, otherUserUUID)
	_, _ = tx.Exec(ctx, `
		DELETE FROM collections
		WHERE user_uuid = $2 AND is_system_collection = true
		  AND EXISTS (
		      SELECT 1 FROM collections c2
		      WHERE c2.user_uuid = $1 AND c2.is_system_collection = true AND c2.name = collections.name
		  )`,
		userCtx.UserUUID, otherUserUUID)
	_, _ = tx.Exec(ctx, `UPDATE collections SET user_uuid = $1 WHERE user_uuid = $2`, userCtx.UserUUID, otherUserUUID)

	// 9. Reassign Q&A, comments, reviews, and votes
	_, _ = tx.Exec(ctx, `UPDATE questions SET user_uuid = $1 WHERE user_uuid = $2`, userCtx.UserUUID, otherUserUUID)
	_, _ = tx.Exec(ctx, `UPDATE question_comments SET user_uuid = $1 WHERE user_uuid = $2`, userCtx.UserUUID, otherUserUUID)
	_, _ = tx.Exec(ctx, `
		UPDATE question_votes qv
		SET user_uuid = $1
		WHERE user_uuid = $2
		  AND NOT EXISTS (
		      SELECT 1 FROM question_votes qv2
		      WHERE qv2.user_uuid = $1 AND qv2.question_uuid = qv.question_uuid
		  )`, userCtx.UserUUID, otherUserUUID)
	_, _ = tx.Exec(ctx, `DELETE FROM question_votes WHERE user_uuid = $1`, otherUserUUID)

	_, _ = tx.Exec(ctx, `UPDATE answers SET user_uuid = $1 WHERE user_uuid = $2`, userCtx.UserUUID, otherUserUUID)
	_, _ = tx.Exec(ctx, `UPDATE answer_comments SET user_uuid = $1 WHERE user_uuid = $2`, userCtx.UserUUID, otherUserUUID)
	_, _ = tx.Exec(ctx, `
		UPDATE answer_votes av
		SET user_uuid = $1
		WHERE user_uuid = $2
		  AND NOT EXISTS (
		      SELECT 1 FROM answer_votes av2
		      WHERE av2.user_uuid = $1 AND av2.answer_uuid = av.answer_uuid
		  )`, userCtx.UserUUID, otherUserUUID)
	_, _ = tx.Exec(ctx, `DELETE FROM answer_votes WHERE user_uuid = $1`, otherUserUUID)

	_, _ = tx.Exec(ctx, `
		UPDATE reviews r
		SET user_uuid = $1
		WHERE user_uuid = $2
		  AND NOT EXISTS (
		      SELECT 1 FROM reviews r2
		      WHERE r2.user_uuid = $1 AND r2.entity_uuid = r.entity_uuid
		  )`, userCtx.UserUUID, otherUserUUID)
	_, _ = tx.Exec(ctx, `DELETE FROM reviews WHERE user_uuid = $1`, otherUserUUID)
	_, _ = tx.Exec(ctx, `UPDATE review_comments SET user_uuid = $1 WHERE user_uuid = $2`, userCtx.UserUUID, otherUserUUID)
	_, _ = tx.Exec(ctx, `
		UPDATE review_votes rv
		SET user_uuid = $1
		WHERE user_uuid = $2
		  AND NOT EXISTS (
		      SELECT 1 FROM review_votes rv2
		      WHERE rv2.user_uuid = $1 AND rv2.review_uuid = rv.review_uuid
		  )`, userCtx.UserUUID, otherUserUUID)
	_, _ = tx.Exec(ctx, `DELETE FROM review_votes WHERE user_uuid = $1`, otherUserUUID)

	// 10. Clean up reputation and bans references
	_, _ = tx.Exec(ctx, `DELETE FROM user_reputations WHERE user_uuid = $1`, otherUserUUID)
	_, _ = tx.Exec(ctx, `UPDATE bans SET user_uuid = $1 WHERE user_uuid = $2`, userCtx.UserUUID, otherUserUUID)
	_, _ = tx.Exec(ctx, `UPDATE bans SET banned_by_user_uuid = $1 WHERE banned_by_user_uuid = $2`, userCtx.UserUUID, otherUserUUID)
	_, _ = tx.Exec(ctx, `UPDATE bans SET revoked_by_user_uuid = $1 WHERE revoked_by_user_uuid = $2`, userCtx.UserUUID, otherUserUUID)

	// 11. Revoke other user's active sessions
	_, _ = tx.Exec(ctx, `DELETE FROM refresh_tokens WHERE user_uuid = $1`, otherUserUUID)

	// 12. Delete email_otps requested by the other user (prevents fk_email_otps_requested_by_user_uuid violation)
	_, _ = tx.Exec(ctx, `DELETE FROM email_otps WHERE requested_by_user_uuid = $1`, otherUserUUID)

	// 13. Backfill current user's profile from other user's profile if empty, then delete other user's profile
	_, _ = tx.Exec(ctx, `
		UPDATE user_profiles up_dest
		SET avatar_url = CASE WHEN up_dest.avatar_url = '' THEN up_src.avatar_url ELSE up_dest.avatar_url END,
		    bio = CASE WHEN up_dest.bio = '' THEN up_src.bio ELSE up_dest.bio END
		FROM user_profiles up_src
		WHERE up_dest.user_uuid = $1 AND up_src.user_uuid = $2`,
		userCtx.UserUUID, otherUserUUID)
	_, _ = tx.Exec(ctx, `DELETE FROM user_profiles WHERE user_uuid = $1`, otherUserUUID)

	// 14. Delete other user record from users table
	_, err = tx.Exec(ctx, `DELETE FROM users WHERE uuid = $1`, otherUserUUID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to finalize merge: %w", err))
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to commit merge: %w", err))
	}

	// If this merge was triggered by an OAuth connection, link the OAuth identity and its verified emails now
	if mergeClaims != nil && mergeClaims.Provider != "" && mergeClaims.ProviderAccountID != "" {
		if mergeClaims.Email != "" {
			_ = s.LinkOAuthAccount(ctx, userCtx.UserUUID, mergeClaims.Provider, mergeClaims.ProviderAccountID, mergeClaims.Email)
			emNorm := strings.ToLower(strings.TrimSpace(mergeClaims.Email))
			if !slices.Contains(transferredEmails, emNorm) {
				transferredEmails = append(transferredEmails, emNorm)
			}
		}
		for _, sec := range mergeClaims.VerifiedEmails {
			secNorm := strings.ToLower(strings.TrimSpace(sec))
			if secNorm != "" {
				_ = s.LinkOAuthAccount(ctx, userCtx.UserUUID, mergeClaims.Provider, mergeClaims.ProviderAccountID, secNorm)
				if !slices.Contains(transferredEmails, secNorm) {
					transferredEmails = append(transferredEmails, secNorm)
				}
			}
		}
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
		SET updated_at = NOW()
		WHERE uuid = $1`,
		userCtx.UserUUID)
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

	// Prune any orphaned oauth_accounts for this user that have no remaining email links
	_, _ = s.db.Exec(ctx, `
		DELETE FROM oauth_accounts oa
		WHERE oa.user_uuid = $1
		  AND NOT EXISTS (
		      SELECT 1 FROM user_email_oauth_accounts uea
		      WHERE uea.oauth_account_uuid = oa.uuid
		  )`, userCtx.UserUUID)

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

	// 2. Ensure user_emails has this email owned by userUUID
	var emailUUID, ownerUUID string
	err = tx.QueryRow(ctx, `
		SELECT uuid, user_uuid FROM user_emails WHERE LOWER(email) = $1`, email).Scan(&emailUUID, &ownerUUID)
	if errors.Is(err, pgx.ErrNoRows) {
		var hasPrimary bool
		_ = tx.QueryRow(ctx, `
			SELECT EXISTS(SELECT 1 FROM user_emails WHERE user_uuid = $1 AND is_primary = true)`,
			userUUID).Scan(&hasPrimary)

		emailUUID = NewUUIDv7()
		_, err = tx.Exec(ctx, `
			INSERT INTO user_emails (uuid, user_uuid, email, is_primary, verified_via_otp, created_at)
			VALUES ($1, $2, $3, $4, false, NOW())`,
			emailUUID, userUUID, email, !hasPrimary)
		if err != nil {
			return fmt.Errorf("failed to insert user email: %w", err)
		}
	} else if err != nil {
		return fmt.Errorf("failed to query user email: %w", err)
	} else if ownerUUID != userUUID {
		return fmt.Errorf("email %s is already associated with another user", email)
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

func (s *AuthService) StartOAuth(ctx context.Context, req *connect.Request[pb.StartOAuthRequest]) (*connect.Response[pb.StartOAuthResponse], error) {
	provider := req.Msg.GetProvider()
	if provider == pb.OAuthProvider_OAUTH_PROVIDER_UNSPECIFIED {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("valid oauth provider is required"))
	}

	purpose := req.Msg.GetPurpose()
	if purpose == pb.OAuthPurpose_OAUTH_PURPOSE_UNSPECIFIED {
		purpose = pb.OAuthPurpose_OAUTH_PURPOSE_LOGIN
	}

	var requesterUserUUID string
	if purpose == pb.OAuthPurpose_OAUTH_PURPOSE_ADD_EMAIL {
		userCtx := UserContextFromContext(ctx)
		if userCtx == nil {
			return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("authentication required to link oauth provider"))
		}
		if userCtx.IsBanned {
			return nil, connect.NewError(connect.CodePermissionDenied, errors.New("account is suspended"))
		}
		requesterUserUUID = userCtx.UserUUID
	}

	var challengeID, challengeIDHash string
	if purpose == pb.OAuthPurpose_OAUTH_PURPOSE_CLI_LOGIN {
		challengeID = NewUUIDv7()
		challengeIDHash = HashToken(challengeID)
		otpUUID := NewUUIDv7()
		expiresAt := time.Now().Add(10 * time.Minute)
		_, err := s.db.Exec(ctx, `
			INSERT INTO email_otps (uuid, email, code_hash, purpose, status, challenge_id_hash, expires_at, created_at)
			VALUES ($1, 'cli_oauth@quadsmith.internal', '', 'cli_oauth', 'pending', $2, $3, NOW())`,
			otpUUID, challengeIDHash, expiresAt)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to create cli oauth challenge: %w", err))
		}
	}

	baseURL := ""
	isMockOAuth := s.cfg == nil || s.cfg.OAuth == nil || s.cfg.OAuth.GetMock() != nil || s.cfg.OAuth.GetProviders() == nil
	if isMockOAuth {
		if origin := strings.TrimRight(strings.TrimSpace(req.Header().Get("Origin")), "/"); origin != "" {
			if u, err := url.Parse(origin); err == nil && (u.Scheme == "http" || u.Scheme == "https") && u.Host != "" {
				baseURL = origin
			}
		} else if referer := strings.TrimSpace(req.Header().Get("Referer")); referer != "" {
			if u, err := url.Parse(referer); err == nil && (u.Scheme == "http" || u.Scheme == "https") && u.Host != "" {
				baseURL = fmt.Sprintf("%s://%s", u.Scheme, u.Host)
			}
		}
	}
	if baseURL == "" && s.cfg != nil && s.cfg.OAuth != nil && s.cfg.OAuth.GetWebBaseUrl() != "" {
		baseURL = s.cfg.OAuth.GetWebBaseUrl()
	}
	if baseURL == "" {
		if origin := strings.TrimRight(strings.TrimSpace(req.Header().Get("Origin")), "/"); origin != "" {
			baseURL = origin
		} else {
			baseURL = "http://localhost:8080"
		}
	}

	redirectPath := req.Msg.GetRedirectPath()
	if redirectPath == "" {
		redirectPath = "/login"
	}
	redirectURI := strings.TrimRight(baseURL, "/") + "/" + strings.TrimLeft(redirectPath, "/")

	codeVerifier, codeChallenge, err := GeneratePKCE()
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to generate pkce: %w", err))
	}

	providerStr := OAuthProviderToString(provider)
	var purposeStr string
	switch purpose {
	case pb.OAuthPurpose_OAUTH_PURPOSE_ADD_EMAIL:
		purposeStr = "add_email"
	case pb.OAuthPurpose_OAUTH_PURPOSE_CLI_LOGIN:
		purposeStr = "cli_login"
	default:
		purposeStr = "login"
	}

	state, err := SignOAuthState(providerStr, purposeStr, codeVerifier, redirectURI, requesterUserUUID, challengeIDHash)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to sign oauth state: %w", err))
	}

	authURL, err := s.oauthClient.GetAuthorizationURL(provider, state, codeChallenge, redirectURI)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	res := &pb.StartOAuthResponse{
		AuthorizationUrl: authURL,
		State:            state,
	}
	if challengeID != "" {
		res.ChallengeId = &challengeID
	}
	return connect.NewResponse(res), nil
}

func (s *AuthService) CompleteOAuth(ctx context.Context, req *connect.Request[pb.CompleteOAuthRequest]) (*connect.Response[pb.CompleteOAuthResponse], error) {
	code := strings.TrimSpace(req.Msg.GetCode())
	state := strings.TrimSpace(req.Msg.GetState())
	if code == "" || state == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("code and state are required"))
	}

	stateClaims, err := VerifyOAuthState(state)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("invalid or expired oauth state: %w", err))
	}

	provider := StringToOAuthProvider(stateClaims.Provider)
	identity, err := s.oauthClient.ExchangeCode(ctx, provider, code, stateClaims.CodeVerifier, stateClaims.RedirectURI)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("oauth token exchange failed: %w", err))
	}

	purpose := pb.OAuthPurpose_OAUTH_PURPOSE_LOGIN
	switch stateClaims.Purpose {
	case "add_email":
		purpose = pb.OAuthPurpose_OAUTH_PURPOSE_ADD_EMAIL
	case "cli_login":
		purpose = pb.OAuthPurpose_OAUTH_PURPOSE_CLI_LOGIN
	}

	if purpose == pb.OAuthPurpose_OAUTH_PURPOSE_ADD_EMAIL {
		return s.completeOAuthAddEmail(ctx, req, provider, purpose, identity, stateClaims)
	}
	return s.completeOAuthLogin(ctx, req, provider, purpose, identity, stateClaims)
}

func (s *AuthService) completeOAuthAddEmail(
	ctx context.Context,
	req *connect.Request[pb.CompleteOAuthRequest],
	provider pb.OAuthProvider,
	purpose pb.OAuthPurpose,
	identity *OAuthIdentity,
	stateClaims *OAuthStateClaims,
) (*connect.Response[pb.CompleteOAuthResponse], error) {
	requesterUserUUID := stateClaims.RequesterUserUUID
	userCtx := UserContextFromContext(ctx)
	if userCtx != nil {
		if userCtx.IsBanned {
			return nil, connect.NewError(connect.CodePermissionDenied, errors.New("account is suspended"))
		}
		if requesterUserUUID != "" && userCtx.UserUUID != requesterUserUUID {
			return nil, connect.NewError(connect.CodePermissionDenied, errors.New("oauth state was issued to a different user session"))
		}
		if requesterUserUUID == "" {
			requesterUserUUID = userCtx.UserUUID
		}
	}
	if requesterUserUUID == "" {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("authentication required to link oauth provider"))
	}
	if userCtx == nil {
		if activeBan, err := getActiveBanForUser(ctx, s.db, requesterUserUUID); err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		} else if activeBan != nil {
			return nil, connect.NewError(connect.CodePermissionDenied, errors.New("account is suspended"))
		}
	}

	// Check if any of the OAuth emails are banned
	var isEmailSuspended bool
	_ = s.db.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM banned_emails be
			JOIN bans b ON b.uuid = be.ban_uuid
			WHERE LOWER(be.email) = ANY($1)
			  AND b.revoked_at IS NULL
			  AND (b.expires_at IS NULL OR b.expires_at > NOW())
		)`, identity.VerifiedEmails).Scan(&isEmailSuspended)
	if isEmailSuspended {
		return nil, connect.NewError(connect.CodePermissionDenied, errors.New("this email address is suspended and cannot be linked"))
	}

	// 1. Check if oauth provider account is already linked to another active user
	var otherOAuthUserUUID string
	err := s.db.QueryRow(ctx, `
		SELECT oa.user_uuid
		FROM oauth_accounts oa
		JOIN users u ON u.uuid = oa.user_uuid
		WHERE LOWER(oa.provider) = $1 AND oa.provider_account_id = $2
		  AND (u.deleted_at IS NULL OR u.deleted_at + INTERVAL '30 days' > NOW())`,
		strings.ToLower(identity.Provider), identity.ProviderAccountID).Scan(&otherOAuthUserUUID)
	if err == nil && otherOAuthUserUUID != requesterUserUUID {
		return s.buildOAuthMergeResponse(ctx, provider, purpose, identity, requesterUserUUID, otherOAuthUserUUID)
	}

	// 2. Check if any verified email belongs to another active user
	var otherEmailUserUUID string
	err = s.db.QueryRow(ctx, `
		SELECT DISTINCT ue.user_uuid
		FROM user_emails ue
		JOIN users u ON u.uuid = ue.user_uuid
		WHERE LOWER(ue.email) = ANY($1) AND ue.user_uuid != $2 AND (u.deleted_at IS NULL OR u.deleted_at + INTERVAL '30 days' > NOW())
		LIMIT 1`,
		identity.VerifiedEmails, requesterUserUUID).Scan(&otherEmailUserUUID)
	if err == nil && otherEmailUserUUID != "" && otherEmailUserUUID != requesterUserUUID {
		return s.buildOAuthMergeResponse(ctx, provider, purpose, identity, requesterUserUUID, otherEmailUserUUID)
	}

	// 3. No conflict: Link oauth account and all verified emails
	err = s.LinkOAuthAccount(ctx, requesterUserUUID, identity.Provider, identity.ProviderAccountID, identity.PrimaryEmail)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to link oauth account: %w", err))
	}
	for _, sec := range identity.VerifiedEmails {
		if !strings.EqualFold(sec, identity.PrimaryEmail) {
			_ = s.LinkOAuthAccount(ctx, requesterUserUUID, identity.Provider, identity.ProviderAccountID, sec)
		}
	}

	// Backfill avatar URL if empty
	if identity.AvatarURL != "" {
		_, _ = s.db.Exec(ctx, `
			UPDATE user_profiles
			SET avatar_url = $1, updated_at = NOW()
			WHERE user_uuid = $2 AND (avatar_url IS NULL OR avatar_url = '')`,
			identity.AvatarURL, requesterUserUUID)
	}

	return connect.NewResponse(&pb.CompleteOAuthResponse{
		Success:            true,
		Message:            "OAuth provider connected successfully",
		Provider:           provider,
		Purpose:            purpose,
		PrimaryEmail:       identity.PrimaryEmail,
		VerifiedEmails:     identity.VerifiedEmails,
		SuggestedAvatarUrl: &identity.AvatarURL,
	}), nil
}

func (s *AuthService) completeOAuthLogin(
	ctx context.Context,
	req *connect.Request[pb.CompleteOAuthRequest],
	provider pb.OAuthProvider,
	purpose pb.OAuthPurpose,
	identity *OAuthIdentity,
	stateClaims *OAuthStateClaims,
) (*connect.Response[pb.CompleteOAuthResponse], error) {
	// 1. Check for matching users via oauth_accounts or verified emails
	matchedUUIDs := make([]string, 0, 2)

	var directOAuthUserUUID string
	err := s.db.QueryRow(ctx, `
		SELECT oa.user_uuid
		FROM oauth_accounts oa
		JOIN users u ON u.uuid = oa.user_uuid
		WHERE LOWER(oa.provider) = $1 AND oa.provider_account_id = $2
		  AND (u.deleted_at IS NULL OR u.deleted_at + INTERVAL '30 days' > NOW())`,
		strings.ToLower(identity.Provider), identity.ProviderAccountID).Scan(&directOAuthUserUUID)
	if err == nil && directOAuthUserUUID != "" {
		matchedUUIDs = append(matchedUUIDs, directOAuthUserUUID)
	}

	rows, err := s.db.Query(ctx, `
		SELECT DISTINCT ue.user_uuid
		FROM user_emails ue
		JOIN users u ON u.uuid = ue.user_uuid
		WHERE LOWER(ue.email) = ANY($1) AND (u.deleted_at IS NULL OR u.deleted_at + INTERVAL '30 days' > NOW())`,
		identity.VerifiedEmails)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var uUUID string
			if err := rows.Scan(&uUUID); err == nil && uUUID != "" && !slices.Contains(matchedUUIDs, uUUID) {
				matchedUUIDs = append(matchedUUIDs, uUUID)
			}
		}
	}

	// Case A: 0 matched users -> Brand new user setup required
	if len(matchedUUIDs) == 0 {
		var isEmailSuspended bool
		_ = s.db.QueryRow(ctx, `
			SELECT EXISTS (
				SELECT 1 FROM banned_emails be
				JOIN bans b ON b.uuid = be.ban_uuid
				WHERE LOWER(be.email) = ANY($1)
				  AND b.revoked_at IS NULL
				  AND (b.expires_at IS NULL OR b.expires_at > NOW())
			)`, identity.VerifiedEmails).Scan(&isEmailSuspended)
		if isEmailSuspended {
			return nil, connect.NewError(connect.CodePermissionDenied, errors.New("this email address is suspended from creating a new account"))
		}

		setupToken, err := SignOAuthSetupToken(identity.PrimaryEmail, identity.VerifiedEmails, identity.Provider, identity.ProviderAccountID, identity.AvatarURL, stateClaims.ChallengeIDHash)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to sign setup token: %w", err))
		}

		return connect.NewResponse(&pb.CompleteOAuthResponse{
			Success:            true,
			Message:            "Account setup required",
			Provider:           provider,
			Purpose:            purpose,
			PrimaryEmail:       identity.PrimaryEmail,
			VerifiedEmails:     identity.VerifiedEmails,
			SuggestedAvatarUrl: &identity.AvatarURL,
			AuthResponse: &pb.AuthResponse{
				AccountSetupRequired: true,
				SetupToken:           setupToken,
			},
		}), nil
	}

	// Case B: Exactly 1 (or > 1) matched user -> Log in as primaryUserUUID
	primaryUserUUID := matchedUUIDs[0]

	// Link oauth account and all verified emails (LinkOAuthAccount skips emails owned by another user)
	_ = s.LinkOAuthAccount(ctx, primaryUserUUID, identity.Provider, identity.ProviderAccountID, identity.PrimaryEmail)
	for _, sec := range identity.VerifiedEmails {
		if !strings.EqualFold(sec, identity.PrimaryEmail) {
			_ = s.LinkOAuthAccount(ctx, primaryUserUUID, identity.Provider, identity.ProviderAccountID, sec)
		}
	}

	// Resolve primaryUserUUID's actual email in user_emails for session issuance
	var primaryUserEmail string
	_ = s.db.QueryRow(ctx, `
		SELECT email FROM user_emails
		WHERE user_uuid = $1
		ORDER BY is_primary DESC, created_at ASC
		LIMIT 1`, primaryUserUUID).Scan(&primaryUserEmail)
	if primaryUserEmail == "" {
		primaryUserEmail = identity.PrimaryEmail
	}

	// Backfill avatar URL if empty
	if identity.AvatarURL != "" {
		_, _ = s.db.Exec(ctx, `
			UPDATE user_profiles
			SET avatar_url = $1, updated_at = NOW()
			WHERE user_uuid = $2 AND (avatar_url IS NULL OR avatar_url = '')`,
			identity.AvatarURL, primaryUserUUID)
	}

	// If this was initiated from a CLI login challenge, mark the challenge approved
	if purpose == pb.OAuthPurpose_OAUTH_PURPOSE_CLI_LOGIN && stateClaims.ChallengeIDHash != "" {
		_, _ = s.db.Exec(ctx, `
			UPDATE email_otps
			SET status = 'approved', email = $1, used_at = NULL
			WHERE challenge_id_hash = $2 AND status = 'pending'`,
			primaryUserEmail, stateClaims.ChallengeIDHash)
	}

	clientIP := getClientIP(req)
	RecordIPSuccess(clientIP)
	if err := CheckAndRecordIPSignIn(clientIP, primaryUserEmail); err != nil {
		return nil, connect.NewError(connect.CodeResourceExhausted, err)
	}

	locationHint := extractLocationHint(req.Header(), clientIP)
	authRes, err := s.issueAuthSession(ctx, primaryUserEmail, req.Header().Get("User-Agent"), clientIP, locationHint)
	if err != nil {
		return nil, err
	}

	// Case C: If > 1 matched users, also attach merge required payload
	if len(matchedUUIDs) > 1 {
		otherUserUUID := matchedUUIDs[1]
		mergeTicket, err := SignOAuthMergeTicket(primaryUserUUID, otherUserUUID, identity.PrimaryEmail, identity.VerifiedEmails, identity.Provider, identity.ProviderAccountID, time.Now().Add(15*time.Minute))
		if err == nil {
			var otherUsername string
			_ = s.db.QueryRow(ctx, `SELECT id FROM users WHERE uuid = $1`, otherUserUUID).Scan(&otherUsername)

			var otherBuildCount int
			_ = s.db.QueryRow(ctx, `SELECT COUNT(*) FROM entities WHERE user_uuid = $1 AND collection = 'builds' AND is_archived = false`, otherUserUUID).Scan(&otherBuildCount)

			var otherBuildNames []string
			bRows, err := s.db.Query(ctx, `SELECT name FROM entities WHERE user_uuid = $1 AND collection = 'builds' AND is_archived = false ORDER BY updated_at DESC LIMIT 5`, otherUserUUID)
			if err == nil {
				defer bRows.Close()
				for bRows.Next() {
					var bn string
					if err := bRows.Scan(&bn); err == nil && bn != "" {
						otherBuildNames = append(otherBuildNames, bn)
					}
				}
			}

			var otherEmails []string
			eRows, err := s.db.Query(ctx, `SELECT email FROM user_emails WHERE user_uuid = $1`, otherUserUUID)
			if err == nil {
				defer eRows.Close()
				for eRows.Next() {
					var em string
					if err := eRows.Scan(&em); err == nil && em != "" {
						otherEmails = append(otherEmails, em)
					}
				}
			}

			var otherProviders []string
			pRows, err := s.db.Query(ctx, `SELECT DISTINCT provider FROM oauth_accounts WHERE user_uuid = $1`, otherUserUUID)
			if err == nil {
				defer pRows.Close()
				for pRows.Next() {
					var p string
					if err := pRows.Scan(&p); err == nil && p != "" {
						otherProviders = append(otherProviders, p)
					}
				}
			}

			res := connect.NewResponse(&pb.CompleteOAuthResponse{
				Success:             true,
				Message:             "Authenticated with primary account. Additional accounts match your verified emails.",
				Provider:            provider,
				Purpose:             purpose,
				PrimaryEmail:        identity.PrimaryEmail,
				VerifiedEmails:      identity.VerifiedEmails,
				SuggestedAvatarUrl:  &identity.AvatarURL,
				AuthResponse:        authRes.Msg,
				MergeRequired:       true,
				MergeTicket:         &mergeTicket,
				OtherUserUsername:   otherUsername,
				OtherBuildCount:     int32(otherBuildCount),
				OtherBuildNames:     otherBuildNames,
				OtherEmails:         otherEmails,
				OtherOauthProviders: otherProviders,
			})
			for _, cookie := range authRes.Header().Values("Set-Cookie") {
				res.Header().Add("Set-Cookie", cookie)
			}
			return res, nil
		}
	}

	res := connect.NewResponse(&pb.CompleteOAuthResponse{
		Success:            true,
		Message:            "Successfully authenticated",
		Provider:           provider,
		Purpose:            purpose,
		PrimaryEmail:       identity.PrimaryEmail,
		VerifiedEmails:     identity.VerifiedEmails,
		SuggestedAvatarUrl: &identity.AvatarURL,
		AuthResponse:       authRes.Msg,
	})
	for _, cookie := range authRes.Header().Values("Set-Cookie") {
		res.Header().Add("Set-Cookie", cookie)
	}
	return res, nil
}

func (s *AuthService) buildOAuthMergeResponse(
	ctx context.Context,
	provider pb.OAuthProvider,
	purpose pb.OAuthPurpose,
	identity *OAuthIdentity,
	sourceUserUUID, targetUserUUID string,
) (*connect.Response[pb.CompleteOAuthResponse], error) {
	mergeTicket, err := SignOAuthMergeTicket(sourceUserUUID, targetUserUUID, identity.PrimaryEmail, identity.VerifiedEmails, identity.Provider, identity.ProviderAccountID, time.Now().Add(15*time.Minute))
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to sign merge ticket: %w", err))
	}

	var otherUsername string
	_ = s.db.QueryRow(ctx, `SELECT id FROM users WHERE uuid = $1`, targetUserUUID).Scan(&otherUsername)

	var otherBuildCount int
	_ = s.db.QueryRow(ctx, `SELECT COUNT(*) FROM entities WHERE user_uuid = $1 AND collection = 'builds' AND is_archived = false`, targetUserUUID).Scan(&otherBuildCount)

	var otherBuildNames []string
	bRows, err := s.db.Query(ctx, `SELECT name FROM entities WHERE user_uuid = $1 AND collection = 'builds' AND is_archived = false ORDER BY updated_at DESC LIMIT 5`, targetUserUUID)
	if err == nil {
		defer bRows.Close()
		for bRows.Next() {
			var bn string
			if err := bRows.Scan(&bn); err == nil && bn != "" {
				otherBuildNames = append(otherBuildNames, bn)
			}
		}
	}

	var otherEmails []string
	eRows, err := s.db.Query(ctx, `SELECT email FROM user_emails WHERE user_uuid = $1`, targetUserUUID)
	if err == nil {
		defer eRows.Close()
		for eRows.Next() {
			var em string
			if err := eRows.Scan(&em); err == nil && em != "" {
				otherEmails = append(otherEmails, em)
			}
		}
	}

	var otherProviders []string
	pRows, err := s.db.Query(ctx, `SELECT DISTINCT provider FROM oauth_accounts WHERE user_uuid = $1`, targetUserUUID)
	if err == nil {
		defer pRows.Close()
		for pRows.Next() {
			var p string
			if err := pRows.Scan(&p); err == nil && p != "" {
				otherProviders = append(otherProviders, p)
			}
		}
	}

	return connect.NewResponse(&pb.CompleteOAuthResponse{
		Success:             false,
		Message:             "An account already exists with one or more of these verified emails. Please confirm whether you wish to merge accounts.",
		Provider:            provider,
		Purpose:             purpose,
		PrimaryEmail:        identity.PrimaryEmail,
		VerifiedEmails:      identity.VerifiedEmails,
		SuggestedAvatarUrl:  &identity.AvatarURL,
		MergeRequired:       true,
		MergeTicket:         &mergeTicket,
		OtherUserUsername:   otherUsername,
		OtherBuildCount:     int32(otherBuildCount),
		OtherBuildNames:     otherBuildNames,
		OtherEmails:         otherEmails,
		OtherOauthProviders: otherProviders,
	}), nil
}

func (s *AuthService) UnlinkOAuth(ctx context.Context, req *connect.Request[pb.UnlinkOAuthRequest]) (*connect.Response[pb.UnlinkOAuthResponse], error) {
	userCtx := UserContextFromContext(ctx)
	if userCtx == nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("authentication required"))
	}
	if userCtx.IsBanned {
		return nil, connect.NewError(connect.CodePermissionDenied, errors.New("account is suspended"))
	}

	provider := req.Msg.GetProvider()
	if provider == pb.OAuthProvider_OAUTH_PROVIDER_UNSPECIFIED {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("valid oauth provider required"))
	}

	providerStr := OAuthProviderToString(provider)
	if err := s.UnlinkOAuthAccount(ctx, userCtx.UserUUID, providerStr); err != nil {
		return nil, err
	}

	return connect.NewResponse(&pb.UnlinkOAuthResponse{
		Success: true,
		Message: "OAuth provider unlinked successfully",
	}), nil
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
	if userCtx.IsSystem() || strings.EqualFold(userCtx.UserID, SystemUserHandle) || userCtx.UserUUID == SystemUserUUID {
		return nil, connect.NewError(connect.CodePermissionDenied, errors.New("the @quadsmith system account cannot be renamed"))
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
	if strings.EqualFold(currentUsername, SystemUserHandle) {
		return nil, connect.NewError(connect.CodePermissionDenied, errors.New("the @quadsmith system account cannot be renamed"))
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
	var newLastRenamedAt, deletedAt, createdAt, updatedAt *time.Time
	err = s.db.QueryRow(ctx, `
		SELECT u.uuid, u.id, COALESCE(ue.email, ''), u.role, u.last_renamed_at, u.deleted_at, u.created_at, u.updated_at
		FROM users u
		LEFT JOIN user_emails ue ON ue.user_uuid = u.uuid AND ue.is_primary = true
		WHERE u.uuid = $1`, userCtx.UserUUID).Scan(
		&user.Uuid, &user.Id, &user.Email, &user.Role,
		&newLastRenamedAt, &deletedAt, &createdAt, &updatedAt)
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

func (s *AuthService) returnOtpError(msg string, remaining int32, isExhausted bool, cooldownSeconds int32) (*connect.Response[pb.VerifyEmailOtpResponse], error) {
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
	if strings.Contains(ua, "qs") || strings.Contains(ua, "Quadsmith") {
		return "Quadsmith CLI"
	}
	if strings.Contains(ua, " on ") || (!strings.Contains(ua, "Mozilla/") && !strings.Contains(ua, "Go-http-client") && len(ua) < 40) {
		return ua
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

func (s *AuthService) issueAuthSession(ctx context.Context, email, userAgent, clientIP, locationHint string) (*connect.Response[pb.AuthResponse], error) {
	var user pb.User
	var userUUID, username, role string
	var lastRenamedAt, deletedAt, createdAt, updatedAt *time.Time

	// 1. Try finding user via user_emails (resolving their primary email for session claims)
	err := s.db.QueryRow(ctx, `
		SELECT u.uuid, u.id,
		       COALESCE((SELECT ue2.email FROM user_emails ue2 WHERE ue2.user_uuid = u.uuid AND ue2.is_primary = true LIMIT 1), ue.email),
		       u.role, u.last_renamed_at, u.deleted_at, u.created_at, u.updated_at
		FROM user_emails ue
		JOIN users u ON u.uuid = ue.user_uuid
		WHERE LOWER(ue.email) = LOWER($1) AND (u.deleted_at IS NULL OR u.deleted_at + INTERVAL '30 days' > NOW())
		LIMIT 1`, email).Scan(
		&userUUID, &username, &user.Email, &role,
		&lastRenamedAt, &deletedAt, &createdAt, &updatedAt)

	if errors.Is(err, pgx.ErrNoRows) {
		// Defense-in-depth: do not allow creating a new user if email is suspended
		var isEmailBanned bool
		_ = s.db.QueryRow(ctx, `
			SELECT EXISTS (
				SELECT 1 FROM banned_emails be
				JOIN bans b ON b.uuid = be.ban_uuid
				WHERE LOWER(be.email) = LOWER($1)
				  AND b.revoked_at IS NULL
				  AND (b.expires_at IS NULL OR b.expires_at > NOW())
			)`, email).Scan(&isEmailBanned)
		if isEmailBanned {
			return nil, connect.NewError(connect.CodePermissionDenied, errors.New("email address is suspended"))
		}

		userUUID = NewUUIDv7()
		username = s.generateUniqueUsername(ctx, email)
		role = "USER"
		now := time.Now()

		_, err = s.db.Exec(ctx, `
			INSERT INTO users (uuid, id, created_at, updated_at)
			VALUES ($1, $2, $3, $3)`,
			userUUID, username, now)
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

	activeBan, err := getActiveBanForUser(ctx, s.db, userUUID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if activeBan != nil {
		user.ActiveBan = activeBan
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

	deviceHint := parseDeviceHint(userAgent)
	now := time.Now()
	expiresAt := now.Add(s.cfg.RefreshTokenTTL)
	_, err = s.db.Exec(ctx, `
		INSERT INTO refresh_tokens (uuid, user_uuid, token_hash, expires_at, last_used_at, device_hint, ip_address, location_hint, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $5)`,
		sessionUUID, userUUID, tokenHash, expiresAt, now, deviceHint, clientIP, locationHint)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to save session: %w", err))
	}

	accessToken, err := SignAccessToken(userUUID, username, user.Email, sessionUUID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to sign access token: %w", err))
	}

	var accountStatus *string
	var scheduledPurgeAt *timestamppb.Timestamp
	if activeBan != nil {
		status := "BANNED"
		accountStatus = &status
	} else if deletedAt != nil {
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
		ActiveBan:        activeBan,
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

// DeleteAccount verifies the deletion challenge (code or magic token) and marks the account as soft-deleted.
func (s *AuthService) DeleteAccount(ctx context.Context, req *connect.Request[pb.DeleteAccountRequest]) (*connect.Response[pb.DeleteAccountResponse], error) {
	userCtx := UserContextFromContext(ctx)
	if userCtx == nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("authentication required"))
	}
	if userCtx.IsSystem() || strings.EqualFold(userCtx.UserID, SystemUserHandle) || userCtx.UserUUID == SystemUserUUID {
		return nil, connect.NewError(connect.CodePermissionDenied, errors.New("the @quadsmith system account cannot be deleted"))
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
	if strings.EqualFold(currentUsername, SystemUserHandle) {
		return nil, connect.NewError(connect.CodePermissionDenied, errors.New("the @quadsmith system account cannot be deleted"))
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
	var lastRenamedAt, createdAt, updatedAt *time.Time
	err = s.db.QueryRow(ctx, `
		SELECT u.uuid, u.id, COALESCE(ue.email, ''), u.role, u.last_renamed_at, u.created_at, u.updated_at
		FROM users u
		LEFT JOIN user_emails ue ON ue.user_uuid = u.uuid AND ue.is_primary = true
		WHERE u.uuid = $1`, userCtx.UserUUID).Scan(
		&user.Uuid, &user.Id, &user.Email, &user.Role,
		&lastRenamedAt, &createdAt, &updatedAt)
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
	var lastRenamedAt, deletedAt, createdAt, updatedAt *time.Time
	err := s.db.QueryRow(ctx, `
		SELECT u.uuid, u.id, COALESCE(ue.email, ''), u.role, u.last_renamed_at, u.deleted_at, u.created_at, u.updated_at
		FROM users u
		LEFT JOIN user_emails ue ON ue.user_uuid = u.uuid AND ue.is_primary = true
		WHERE (u.id = $1 OR u.uuid::text = $1) AND u.deleted_at IS NULL`, id).Scan(
		&user.Uuid, &user.Id, &user.Email, &user.Role,
		&lastRenamedAt, &deletedAt, &createdAt, &updatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// Check if this was a recent username within the 30-day hold window
			err = s.db.QueryRow(ctx, `
				SELECT u.uuid, u.id, COALESCE(ue.email, ''), u.role, u.last_renamed_at, u.deleted_at, u.created_at, u.updated_at
				FROM username_histories hh
				JOIN users u ON u.uuid = hh.user_uuid
				LEFT JOIN user_emails ue ON ue.user_uuid = u.uuid AND ue.is_primary = true
				WHERE LOWER(hh.old_username) = LOWER($1) AND u.deleted_at IS NULL AND COALESCE(hh.first_archived_at, hh.renamed_at) >= NOW() - INTERVAL '30 days'
				ORDER BY hh.renamed_at DESC
				LIMIT 1`, id).Scan(
				&user.Uuid, &user.Id, &user.Email, &user.Role,
				&lastRenamedAt, &deletedAt, &createdAt, &updatedAt)
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
	if !userCtx.IsAdmin() && userCtx.UserUUID != user.Uuid {
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

	activeBan, err := getActiveBanForUser(ctx, s.db, user.Uuid)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if activeBan != nil {
		user.ActiveBan = activeBan
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
	_ = s.db.QueryRow(ctx, `SELECT COUNT(*) FROM entities WHERE user_uuid = $1 AND collection = 'builds' AND is_archived = false`, userUUID).Scan(&buildCount)

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

func (s *UserService) BanUser(ctx context.Context, req *connect.Request[pb.BanUserRequest]) (*connect.Response[pb.BanUserResponse], error) {
	userCtx := UserContextFromContext(ctx)
	if userCtx == nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("authentication required"))
	}
	if !userCtx.IsAdmin() {
		return nil, connect.NewError(connect.CodePermissionDenied, errors.New("admin privileges required"))
	}

	targetID := strings.TrimSpace(req.Msg.GetUserId())
	if targetID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("user id or username required"))
	}

	reason := strings.TrimSpace(req.Msg.GetReason())
	if reason == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("ban reason required"))
	}
	internalNotes := strings.TrimSpace(req.Msg.GetInternalNotes())

	var targetUUID, targetHandle, targetRole string
	err := s.db.QueryRow(ctx, `
		SELECT uuid, id, role FROM users
		WHERE LOWER(id) = LOWER($1) OR uuid::text = $1
		LIMIT 1`, targetID).Scan(&targetUUID, &targetHandle, &targetRole)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, connect.NewError(connect.CodeNotFound, errors.New("target user not found"))
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	if targetRole == RoleSystem || strings.EqualFold(targetHandle, SystemUserHandle) || targetUUID == SystemUserUUID {
		return nil, connect.NewError(connect.CodePermissionDenied, errors.New("the @quadsmith system account cannot be banned"))
	}

	if targetUUID == userCtx.UserUUID {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("cannot ban yourself"))
	}

	existingBan, err := getActiveBanForUser(ctx, s.db, targetUUID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if existingBan != nil {
		return nil, connect.NewError(connect.CodeAlreadyExists, errors.New("user already has an active ban"))
	}

	banUUID := NewUUIDv7()
	var expiresAt *time.Time
	if req.Msg.ExpiresAt != nil {
		t := req.Msg.ExpiresAt.AsTime()
		expiresAt = &t
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to begin transaction: %w", err))
	}
	defer func() { _ = tx.Rollback(ctx) }()

	now := time.Now()
	_, err = tx.Exec(ctx, `
		INSERT INTO bans (uuid, user_uuid, banned_by_user_uuid, reason, internal_notes, expires_at, created_at, revoke_reason)
		VALUES ($1, $2, $3, $4, $5, $6, $7, '')`,
		banUUID, targetUUID, userCtx.UserUUID, reason, internalNotes, expiresAt, now)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to insert ban: %w", err))
	}

	_, err = tx.Exec(ctx, `
		UPDATE users
		SET role = 'USER', updated_at = $2
		WHERE uuid = $1::uuid`, targetUUID, now)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to demote banned user: %w", err))
	}

	rows, err := tx.Query(ctx, `SELECT email FROM user_emails WHERE user_uuid = $1`, targetUUID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to query user emails: %w", err))
	}
	defer rows.Close()

	var emails []string
	for rows.Next() {
		var email string
		if err := rows.Scan(&email); err == nil {
			emails = append(emails, email)
		}
	}
	rows.Close()

	for _, email := range emails {
		bannedEmailUUID := NewUUIDv7()
		_, err = tx.Exec(ctx, `
			INSERT INTO banned_emails (uuid, ban_uuid, email, created_at)
			VALUES ($1, $2, $3, $4)`,
			bannedEmailUUID, banUUID, email, now)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to insert banned email: %w", err))
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to commit ban transaction: %w", err))
	}

	ban := &pb.Ban{
		Uuid:             banUUID,
		UserUuid:         targetUUID,
		BannedByUserUuid: userCtx.UserUUID,
		Reason:           reason,
		InternalNotes:    internalNotes,
		CreatedAt:        timestamppb.New(now),
	}
	if expiresAt != nil {
		ban.ExpiresAt = timestamppb.New(*expiresAt)
	}

	return connect.NewResponse(&pb.BanUserResponse{
		Success: true,
		Message: fmt.Sprintf("User @%s successfully banned", targetHandle),
		Ban:     ban,
	}), nil
}

func (s *UserService) UpdateBan(ctx context.Context, req *connect.Request[pb.UpdateBanRequest]) (*connect.Response[pb.UpdateBanResponse], error) {
	userCtx := UserContextFromContext(ctx)
	if userCtx == nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("authentication required"))
	}
	if !userCtx.IsAdmin() {
		return nil, connect.NewError(connect.CodePermissionDenied, errors.New("admin privileges required"))
	}

	targetID := strings.TrimSpace(req.Msg.GetUserId())
	if targetID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("user id or username required"))
	}

	// 1. Resolve user
	var targetUUID, targetHandle string
	err := s.db.QueryRow(ctx, `
		SELECT uuid::text, id
		FROM users
		WHERE uuid::text = $1 OR LOWER(id) = LOWER($1)
	`, targetID).Scan(&targetUUID, &targetHandle)
	if errors.Is(err, pgx.ErrNoRows) {
		var banUserUUID string
		err2 := s.db.QueryRow(ctx, `
			SELECT b.user_uuid::text, u.id
			FROM bans b
			JOIN users u ON u.uuid = b.user_uuid
			WHERE b.uuid::text = $1
		`, targetID).Scan(&banUserUUID, &targetHandle)
		if errors.Is(err2, pgx.ErrNoRows) {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("user or ban %q not found", targetID))
		}
		targetUUID = banUserUUID
	} else if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("database query error: %w", err))
	}

	// 2. Fetch current active ban
	var banUUID string
	var currReason, currNotes string
	var currExpiresAt *time.Time
	var createdAt time.Time
	var bannedByUUID string

	err = s.db.QueryRow(ctx, `
		SELECT uuid::text, banned_by_user_uuid::text, reason, internal_notes, expires_at, created_at
		FROM bans
		WHERE user_uuid = $1::uuid
		  AND revoked_at IS NULL
		  AND (expires_at IS NULL OR expires_at > NOW())
		ORDER BY created_at DESC
		LIMIT 1
	`, targetUUID).Scan(&banUUID, &bannedByUUID, &currReason, &currNotes, &currExpiresAt, &createdAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("no active ban found for user @%s", targetHandle))
	} else if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to lookup active ban: %w", err))
	}

	// Apply updates
	newReason := currReason
	if req.Msg.Reason != nil {
		trimmed := strings.TrimSpace(req.Msg.GetReason())
		if trimmed != "" {
			newReason = trimmed
		}
	}

	newNotes := currNotes
	if req.Msg.InternalNotes != nil {
		newNotes = strings.TrimSpace(req.Msg.GetInternalNotes())
	}

	newExpiresAt := currExpiresAt
	if req.Msg.GetClearExpiresAt() {
		newExpiresAt = nil
	} else if req.Msg.ExpiresAt != nil {
		t := req.Msg.ExpiresAt.AsTime()
		newExpiresAt = &t
	}

	var revokedAt *time.Time
	var revokedByUserUUID *string
	var revokeReason string
	if req.Msg.GetRevoke() {
		now := time.Now()
		revokedAt = &now
		revokedByUserUUID = &userCtx.UserUUID
		revokeReason = "Ban revoked by administrator"
		if req.Msg.RevokeReason != nil && strings.TrimSpace(req.Msg.GetRevokeReason()) != "" {
			revokeReason = strings.TrimSpace(req.Msg.GetRevokeReason())
		}
	}

	_, err = s.db.Exec(ctx, `
		UPDATE bans
		SET reason = $2,
		    internal_notes = $3,
		    expires_at = $4,
		    revoked_at = $5,
		    revoked_by_user_uuid = $6,
		    revoke_reason = $7
		WHERE uuid = $1::uuid
	`, banUUID, newReason, newNotes, newExpiresAt, revokedAt, revokedByUserUUID, revokeReason)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to update ban: %w", err))
	}

	resBan := &pb.Ban{
		Uuid:             banUUID,
		UserUuid:         targetUUID,
		BannedByUserUuid: bannedByUUID,
		Reason:           newReason,
		InternalNotes:    newNotes,
		CreatedAt:        timestamppb.New(createdAt),
	}
	if newExpiresAt != nil {
		resBan.ExpiresAt = timestamppb.New(*newExpiresAt)
	}
	if revokedAt != nil {
		resBan.RevokedAt = timestamppb.New(*revokedAt)
		resBan.RevokedByUserUuid = revokedByUserUUID
		resBan.RevokeReason = revokeReason
	}

	msg := fmt.Sprintf("Ban on @%s successfully updated", targetHandle)
	if revokedAt != nil {
		msg = fmt.Sprintf("Ban on @%s successfully revoked", targetHandle)
	}

	return connect.NewResponse(&pb.UpdateBanResponse{
		Success: true,
		Message: msg,
		Ban:     resBan,
	}), nil
}

func (s *UserService) ListBans(ctx context.Context, req *connect.Request[pb.ListBansRequest]) (*connect.Response[pb.ListBansResponse], error) {
	userCtx := UserContextFromContext(ctx)
	if userCtx == nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("authentication required"))
	}
	if !userCtx.IsAdmin() {
		return nil, connect.NewError(connect.CodePermissionDenied, errors.New("admin privileges required"))
	}

	pageSize, err := pagination.ParsePageSize(req.Msg.GetPageSize())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	effectiveFilter, effectiveSort, offset, err := pagination.ResolveParams(req.Msg.GetFilter(), req.Msg.GetSort(), req.Msg.GetPageToken())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	var targetUserUUID *string
	if req.Msg.UserId != nil && strings.TrimSpace(*req.Msg.UserId) != "" {
		targetID := strings.TrimSpace(*req.Msg.UserId)
		var tuuid string
		err := s.db.QueryRow(ctx, `
			SELECT uuid::text FROM users
			WHERE LOWER(id) = LOWER($1) OR uuid::text = $1
			LIMIT 1`, targetID).Scan(&tuuid)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil, connect.NewError(connect.CodeNotFound, errors.New("user not found"))
			}
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		targetUserUUID = &tuuid
	}

	query := `
		SELECT uuid, user_uuid, banned_by_user_uuid, reason, internal_notes, expires_at, created_at, revoked_at, revoked_by_user_uuid, revoke_reason
		FROM bans
		WHERE 1=1`
	args := []any{}

	if targetUserUUID != nil {
		args = append(args, *targetUserUUID)
		query += fmt.Sprintf(" AND user_uuid = $%d::uuid", len(args))
	}

	if effectiveFilter != "" {
		whereClause, filterArgs, err := cel2sql.Compile(effectiveFilter)
		if err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		}
		if whereClause != "" {
			startOffset := len(args)
			re := regexp.MustCompile(`\$(\d+)`)
			adjustedWhere := re.ReplaceAllStringFunc(whereClause, func(m string) string {
				n, _ := strconv.Atoi(m[1:])
				return fmt.Sprintf("$%d", n+startOffset)
			})
			query += " AND (" + adjustedWhere + ")"
			args = append(args, filterArgs...)
		}
	}

	validSortCols := map[string]bool{
		"uuid":                 true,
		"user_uuid":            true,
		"banned_by_user_uuid":  true,
		"reason":               true,
		"internal_notes":       true,
		"expires_at":           true,
		"created_at":           true,
		"revoked_at":           true,
		"revoked_by_user_uuid": true,
		"revoke_reason":        true,
	}

	var orderClauses []string
	if len(effectiveSort) > 0 {
		for _, s := range effectiveSort {
			col := s
			dir := "ASC NULLS LAST"
			if strings.HasPrefix(s, "^") {
				col = s[1:]
				dir = "DESC NULLS LAST"
			}
			if !validSortCols[col] {
				return nil, connect.NewError(connect.CodeInvalidArgument, pagination.ErrInvalidSortColumn(col))
			}
			orderClauses = append(orderClauses, pgx.Identifier{col}.Sanitize()+" "+dir)
		}
	} else {
		orderClauses = append(orderClauses, "created_at DESC")
	}
	query += " ORDER BY " + strings.Join(orderClauses, ", ")

	limit := pageSize + 1
	args = append(args, limit)
	query += fmt.Sprintf(" LIMIT $%d", len(args))

	if offset > 0 {
		args = append(args, offset)
		query += fmt.Sprintf(" OFFSET $%d", len(args))
	}

	rows, err := s.db.Query(ctx, query, args...)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	defer rows.Close()

	bans := make([]*pb.Ban, 0)
	for rows.Next() {
		var ban pb.Ban
		var expiresAt, createdAt, revokedAt *time.Time
		var revokedByUserUUID, revokeReason *string
		if err := rows.Scan(
			&ban.Uuid, &ban.UserUuid, &ban.BannedByUserUuid, &ban.Reason, &ban.InternalNotes,
			&expiresAt, &createdAt, &revokedAt, &revokedByUserUUID, &revokeReason); err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		if expiresAt != nil {
			ban.ExpiresAt = timestamppb.New(*expiresAt)
		}
		if createdAt != nil {
			ban.CreatedAt = timestamppb.New(*createdAt)
		}
		if revokedAt != nil {
			ban.RevokedAt = timestamppb.New(*revokedAt)
		}
		if revokedByUserUUID != nil {
			ban.RevokedByUserUuid = revokedByUserUUID
		}
		if revokeReason != nil {
			ban.RevokeReason = *revokeReason
		}
		bans = append(bans, &ban)
	}

	var nextPageToken string
	if int32(len(bans)) > pageSize {
		bans = bans[:pageSize]
		nextPageToken, err = pagination.EncodePageToken(offset+pageSize, effectiveFilter, effectiveSort)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to encode next page token: %w", err))
		}
	}

	return connect.NewResponse(&pb.ListBansResponse{
		Bans:          bans,
		NextPageToken: nextPageToken,
	}), nil
}

func (s *UserService) GetUserBanStatus(ctx context.Context, req *connect.Request[pb.GetUserBanStatusRequest]) (*connect.Response[pb.GetUserBanStatusResponse], error) {
	userCtx := UserContextFromContext(ctx)
	if userCtx == nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("authentication required"))
	}

	targetID := strings.TrimSpace(req.Msg.GetUserId())
	var targetUUID string

	if targetID == "" {
		targetUUID = userCtx.UserUUID
	} else {
		err := s.db.QueryRow(ctx, `
			SELECT uuid::text FROM users
			WHERE LOWER(id) = LOWER($1) OR uuid::text = $1
			LIMIT 1`, targetID).Scan(&targetUUID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil, connect.NewError(connect.CodeNotFound, errors.New("user not found"))
			}
			return nil, connect.NewError(connect.CodeInternal, err)
		}
	}

	// Privacy: Users may only inspect their own ban status; admins may inspect anyone
	if !userCtx.IsAdmin() && userCtx.UserUUID != targetUUID {
		return nil, connect.NewError(connect.CodePermissionDenied, errors.New("permission denied"))
	}

	ban, err := getActiveBanForUser(ctx, s.db, targetUUID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	if ban != nil {
		return connect.NewResponse(&pb.GetUserBanStatusResponse{
			IsBanned:  true,
			ActiveBan: ban,
		}), nil
	}

	return connect.NewResponse(&pb.GetUserBanStatusResponse{
		IsBanned:  false,
		ActiveBan: nil,
	}), nil
}
