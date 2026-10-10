package auth

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	connect "connectrpc.com/connect"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/protobuf/types/known/timestamppb"

	pb "quadsmith/api/gen/quadsmith"
	"quadsmith/api/gen/quadsmith/quadsmithconnect"
)

// ---------------------------------------------------------
// AuthService
// ---------------------------------------------------------

type AuthService struct {
	quadsmithconnect.UnimplementedAuthServiceHandler
	db *pgxpool.Pool
}

func NewAuthService(db *pgxpool.Pool) *AuthService {
	return &AuthService{db: db}
}

func (s *AuthService) SendEmailOtp(ctx context.Context, req *connect.Request[pb.SendEmailOtpRequest]) (*connect.Response[pb.SendEmailOtpResponse], error) {
	email := strings.ToLower(strings.TrimSpace(req.Msg.GetEmail()))
	emailRegex := regexp.MustCompile(`^[a-z0-9._%+\-]+@[a-z0-9.\-]+\.[a-z]{2,}$`)
	if !emailRegex.MatchString(email) {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("please enter a valid email address"))
	}

	clientIP := extractClientIP(req.Header())
	if err := CheckIPRateLimit(clientIP); err != nil {
		return nil, connect.NewError(connect.CodeResourceExhausted, err)
	}

	cooldown, err := CheckEmailCooldown(ctx, s.db, email)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if cooldown > 0 {
		return nil, connect.NewError(connect.CodeResourceExhausted, fmt.Errorf("please wait %d seconds before requesting another code", cooldown))
	}

	otpCode, codeHash, err := GenerateOTP()
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to generate otp: %w", err))
	}

	otpUUID := NewUUIDv7()
	now := time.Now()
	expiresAt := now.Add(MagicLinkTTL) // 15 min max lifespan for the challenge

	_, err = s.db.Exec(ctx, `
		INSERT INTO email_otps (uuid, email, code_hash, expires_at, created_at, failed_attempts)
		VALUES ($1, $2, $3, $4, $5, 0)`,
		otpUUID, email, codeHash, expiresAt, now)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to save otp challenge: %w", err))
	}

	magicToken, err := SignMagicLinkToken(otpUUID, email, expiresAt)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to generate magic link token: %w", err))
	}

	// Development & Sandbox notification logging
	fmt.Printf("\n=======================================================\n")
	fmt.Printf("[AUTH] Email OTP for %s: %s (valid for 3 minutes)\n", email, otpCode)
	fmt.Printf("[AUTH] Magic Link: /login/verify?token=%s (valid for 15 minutes)\n", magicToken)
	fmt.Printf("=======================================================\n\n")

	return connect.NewResponse(&pb.SendEmailOtpResponse{
		Success:         true,
		Message:         "Verification code sent to your email",
		CooldownSeconds: 60,
	}), nil
}

func (s *AuthService) VerifyEmailOtp(ctx context.Context, req *connect.Request[pb.VerifyEmailOtpRequest]) (*connect.Response[pb.AuthResponse], error) {
	email := strings.ToLower(strings.TrimSpace(req.Msg.GetEmail()))
	code := strings.TrimSpace(req.Msg.GetCode())
	if email == "" || code == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("email and 6-digit code are required"))
	}

	clientIP := extractClientIP(req.Header())
	if err := CheckIPRateLimit(clientIP); err != nil {
		return nil, connect.NewError(connect.CodeResourceExhausted, err)
	}

	var otpUUID string
	var codeHash string
	var failedAttempts int32
	var createdAt time.Time
	var usedAt *time.Time

	err := s.db.QueryRow(ctx, `
		SELECT uuid, code_hash, failed_attempts, created_at, used_at
		FROM email_otps
		WHERE email = $1 AND used_at IS NULL
		ORDER BY created_at DESC
		LIMIT 1`, email).Scan(&otpUUID, &codeHash, &failedAttempts, &createdAt, &usedAt)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			RecordIPFailure(clientIP)
			return nil, connect.NewError(connect.CodeNotFound, errors.New("no active verification challenge found for this email. Please request a new code"))
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	// Verify 3-minute OTP lifespan
	if time.Since(createdAt) > OtpTTL {
		_, _ = s.db.Exec(ctx, `UPDATE email_otps SET used_at = NOW() WHERE uuid = $1`, otpUUID)
		return nil, connect.NewError(connect.CodeDeadlineExceeded, errors.New("verification code has expired (3-minute limit). Please request a new code"))
	}

	if failedAttempts >= MaxOtpAttempts {
		_, _ = s.db.Exec(ctx, `UPDATE email_otps SET used_at = NOW() WHERE uuid = $1`, otpUUID)
		return s.returnOtpError("Verification challenge exhausted. Please request a new code", 0, true)
	}

	if !VerifyOTP(code, codeHash) {
		RecordIPFailure(clientIP)
		newFailures := failedAttempts + 1
		remaining := int32(MaxOtpAttempts - newFailures)
		if remaining <= 0 {
			_, _ = s.db.Exec(ctx, `UPDATE email_otps SET failed_attempts = $1, used_at = NOW() WHERE uuid = $2`, newFailures, otpUUID)
			return s.returnOtpError("Incorrect code. You have 0 attempts left. Please request a new code", 0, true)
		}
		_, _ = s.db.Exec(ctx, `UPDATE email_otps SET failed_attempts = $1 WHERE uuid = $2`, newFailures, otpUUID)
		msg := fmt.Sprintf("Incorrect code. You have %d %s left", remaining, pluralAttempts(remaining))
		return s.returnOtpError(msg, remaining, false)
	}

	// Verification succeeded
	RecordIPSuccess(clientIP)
	_, _ = s.db.Exec(ctx, `UPDATE email_otps SET used_at = NOW() WHERE uuid = $1`, otpUUID)

	return s.issueAuthSession(ctx, email, req.Header().Get("User-Agent"))
}

func (s *AuthService) VerifyMagicLink(ctx context.Context, req *connect.Request[pb.VerifyMagicLinkRequest]) (*connect.Response[pb.AuthResponse], error) {
	token := strings.TrimSpace(req.Msg.GetToken())
	if token == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("magic link token required"))
	}

	otpUUID, email, err := VerifyMagicLinkToken(token)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid or expired magic link: %w", err))
	}

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

	if time.Since(createdAt) > MagicLinkTTL {
		_, _ = s.db.Exec(ctx, `UPDATE email_otps SET used_at = NOW() WHERE uuid = $1`, otpUUID)
		return nil, connect.NewError(connect.CodeDeadlineExceeded, errors.New("magic link has expired. Please request a new code"))
	}

	_, _ = s.db.Exec(ctx, `UPDATE email_otps SET used_at = NOW() WHERE uuid = $1`, otpUUID)

	clientIP := extractClientIP(req.Header())
	RecordIPSuccess(clientIP)

	return s.issueAuthSession(ctx, email, req.Header().Get("User-Agent"))
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

	// 5-day sliding window: roll expiration forward to NOW() + 5 days
	newExpiresAt := time.Now().Add(RefreshTokenTTL)
	_, err = s.db.Exec(ctx, `
		UPDATE refresh_tokens
		SET expires_at = $1, last_used_at = NOW()
		WHERE uuid = $2`, newExpiresAt, sessionUUID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	// Fetch User
	var user pb.User
	var handle, email, role string
	var emailVerifiedAt, lastRenamedAt, deletedAt, bannedAt, createdAt, updatedAt *time.Time
	err = s.db.QueryRow(ctx, `
		SELECT uuid, id, email, role, email_verified_at, last_renamed_at, deleted_at, banned_at, created_at, updated_at
		FROM users WHERE uuid = $1`, userUUID).Scan(
		&user.Uuid, &handle, &email, &role,
		&emailVerifiedAt, &lastRenamedAt, &deletedAt, &bannedAt, &createdAt, &updatedAt)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	user.Id = handle
	user.Email = email
	user.Role = role
	if emailVerifiedAt != nil {
		user.EmailVerifiedAt = timestamppb.New(*emailVerifiedAt)
	}
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
		SELECT uuid, user_uuid, display_name, avatar_url, bio, updated_at
		FROM user_profiles WHERE user_uuid = $1`, userUUID).Scan(
		&profile.Uuid, &profile.UserUuid, &profile.DisplayName, &profile.AvatarUrl, &profile.Bio, &profileUpdatedAt)
	if profileUpdatedAt != nil {
		profile.UpdatedAt = timestamppb.New(*profileUpdatedAt)
	}

	accessToken, err := SignAccessToken(userUUID, handle, email, role, sessionUUID)
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
		MaxAge:   int(RefreshTokenTTL.Seconds()),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
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
	var emailVerifiedAt, lastRenamedAt, deletedAt, bannedAt, createdAt, updatedAt *time.Time
	err := s.db.QueryRow(ctx, `
		SELECT uuid, id, email, role, email_verified_at, last_renamed_at, deleted_at, banned_at, created_at, updated_at
		FROM users WHERE uuid = $1`, userCtx.UserUUID).Scan(
		&user.Uuid, &user.Id, &user.Email, &user.Role,
		&emailVerifiedAt, &lastRenamedAt, &deletedAt, &bannedAt, &createdAt, &updatedAt)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("user not found"))
	}
	if emailVerifiedAt != nil {
		user.EmailVerifiedAt = timestamppb.New(*emailVerifiedAt)
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
			INSERT INTO user_profiles (uuid, user_uuid, display_name, avatar_url, bio, updated_at)
			VALUES ($1, $2, $3, '', '', NOW())`,
			profileUUID, userCtx.UserUUID, userCtx.UserID)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
	}

	if req.Msg.DisplayName != nil {
		_, _ = s.db.Exec(ctx, `UPDATE user_profiles SET display_name = $1, updated_at = NOW() WHERE user_uuid = $2`, *req.Msg.DisplayName, userCtx.UserUUID)
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
		SELECT uuid, user_uuid, display_name, avatar_url, bio, updated_at
		FROM user_profiles WHERE user_uuid = $1`, userCtx.UserUUID).Scan(
		&profile.Uuid, &profile.UserUuid, &profile.DisplayName, &profile.AvatarUrl, &profile.Bio, &updatedAt)
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
			INSERT INTO user_emails (uuid, user_uuid, email, is_primary, verified_at, created_at)
			VALUES ($1, $2, $3, true, NOW(), NOW())
			ON CONFLICT (email) DO NOTHING`,
			emailUUID, userCtx.UserUUID, userCtx.Email)
	}

	rows, err := s.db.Query(ctx, `
		SELECT uuid, user_uuid, email, is_primary, verified_at, created_at
		FROM user_emails
		WHERE user_uuid = $1
		ORDER BY is_primary DESC, created_at ASC`, userCtx.UserUUID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to list user emails: %w", err))
	}
	defer rows.Close()

	var emails []*pb.UserEmail
	for rows.Next() {
		var em pb.UserEmail
		var verifiedAt, createdAt *time.Time
		if err := rows.Scan(&em.Uuid, &em.UserUuid, &em.Email, &em.IsPrimary, &verifiedAt, &createdAt); err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to scan email: %w", err))
		}
		if verifiedAt != nil {
			em.VerifiedAt = timestamppb.New(*verifiedAt)
		}
		if createdAt != nil {
			em.CreatedAt = timestamppb.New(*createdAt)
		}
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

	// Check if this email is registered to ANOTHER account
	var otherUserUUID, otherHandle string
	err = s.db.QueryRow(ctx, `
		SELECT u.uuid, u.id
		FROM users u
		WHERE LOWER(u.email) = LOWER($1) AND u.uuid != $2 AND u.deleted_at IS NULL
		UNION
		SELECT u.uuid, u.id
		FROM user_emails ue
		JOIN users u ON u.uuid = ue.user_uuid
		WHERE LOWER(ue.email) = LOWER($1) AND u.uuid != $2 AND u.deleted_at IS NULL
		LIMIT 1`, email, userCtx.UserUUID).Scan(&otherUserUUID, &otherHandle)

	var isExistingAccount bool
	var existingBuildCount int
	if err == nil && otherUserUUID != "" {
		isExistingAccount = true
		_ = s.db.QueryRow(ctx, `SELECT COUNT(*) FROM builds WHERE user_uuid = $1`, otherUserUUID).Scan(&existingBuildCount)
	}

	// Rate limit per IP
	clientIP := extractClientIP(req.Header())
	if err := CheckIPRateLimit(clientIP); err != nil {
		return nil, connect.NewError(connect.CodeResourceExhausted, err)
	}

	cooldown, err := CheckEmailCooldown(ctx, s.db, email)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if cooldown > 0 {
		return nil, connect.NewError(connect.CodeResourceExhausted, fmt.Errorf("please wait %d seconds before requesting another code", cooldown))
	}

	otpCode, codeHash, err := GenerateOTP()
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to generate otp: %w", err))
	}

	otpUUID := NewUUIDv7()
	now := time.Now()
	expiresAt := now.Add(MagicLinkTTL)

	_, err = s.db.Exec(ctx, `
		INSERT INTO email_otps (uuid, email, code_hash, expires_at, created_at, failed_attempts)
		VALUES ($1, $2, $3, $4, $5, 0)`,
		otpUUID, email, codeHash, expiresAt, now)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to save otp challenge: %w", err))
	}

	magicToken, err := SignMagicLinkToken(otpUUID, email, expiresAt)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to generate magic link token: %w", err))
	}

	// Development & Sandbox notification logging
	fmt.Printf("\n=======================================================\n")
	fmt.Printf("[AUTH] Add/Merge Email OTP for %s: %s (valid for 3 minutes)\n", email, otpCode)
	if isExistingAccount {
		fmt.Printf("[AUTH] Merge Magic Link: /account?merge_token=%s (valid for 15 minutes)\n", magicToken)
	} else {
		fmt.Printf("[AUTH] Add Email Magic Link: /account?add_email_token=%s (valid for 15 minutes)\n", magicToken)
	}
	fmt.Printf("=======================================================\n\n")

	return connect.NewResponse(&pb.SendEmailOtpResponse{
		Success:            true,
		Message:            "Verification code sent to email",
		CooldownSeconds:    60,
		IsExistingAccount:  isExistingAccount,
		ExistingUserHandle: otherHandle,
		ExistingBuildCount: int32(existingBuildCount),
	}), nil
}

func (s *AuthService) VerifyAddEmailOtp(ctx context.Context, req *connect.Request[pb.VerifyAddEmailOtpRequest]) (*connect.Response[pb.VerifyAddEmailOtpResponse], error) {
	userCtx := UserContextFromContext(ctx)
	if userCtx == nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("authentication required"))
	}

	email := strings.ToLower(strings.TrimSpace(req.Msg.GetEmail()))
	code := strings.TrimSpace(req.Msg.GetCode())
	token := strings.TrimSpace(req.Msg.GetToken())

	clientIP := extractClientIP(req.Header())
	if err := CheckIPRateLimit(clientIP); err != nil {
		return nil, connect.NewError(connect.CodeResourceExhausted, err)
	}

	var otpUUID string
	var err error
	if token != "" {
		var tokenEmail string
		otpUUID, tokenEmail, err = VerifyMagicLinkToken(token)
		if err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid or expired magic link: %w", err))
		}
		if email == "" {
			email = tokenEmail
		} else if strings.ToLower(email) != strings.ToLower(tokenEmail) {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("magic link token email mismatch"))
		}

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
		if time.Since(createdAt) > MagicLinkTTL {
			_, _ = s.db.Exec(ctx, `UPDATE email_otps SET used_at = NOW() WHERE uuid = $1`, otpUUID)
			return nil, connect.NewError(connect.CodeDeadlineExceeded, errors.New("magic link has expired"))
		}
	} else {
		if email == "" || code == "" {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("email and 6-digit code (or magic link token) are required"))
		}

		// Verify OTP
		var codeHash string
		var failedAttempts int32
		var createdAt time.Time
		err = s.db.QueryRow(ctx, `
			SELECT uuid, code_hash, failed_attempts, created_at
			FROM email_otps
			WHERE email = $1 AND used_at IS NULL
			ORDER BY created_at DESC
			LIMIT 1`, email).Scan(&otpUUID, &codeHash, &failedAttempts, &createdAt)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				RecordIPFailure(clientIP)
				return nil, connect.NewError(connect.CodeNotFound, errors.New("no active verification challenge found. Please request a new code"))
			}
			return nil, connect.NewError(connect.CodeInternal, err)
		}

		if time.Since(createdAt) > OtpTTL {
			_, _ = s.db.Exec(ctx, `UPDATE email_otps SET used_at = NOW() WHERE uuid = $1`, otpUUID)
			return nil, connect.NewError(connect.CodeDeadlineExceeded, errors.New("verification code has expired"))
		}

		if failedAttempts >= MaxOtpAttempts {
			_, _ = s.db.Exec(ctx, `UPDATE email_otps SET used_at = NOW() WHERE uuid = $1`, otpUUID)
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("verification challenge exhausted"))
		}

		if !VerifyOTP(code, codeHash) {
			RecordIPFailure(clientIP)
			newFailures := failedAttempts + 1
			_, _ = s.db.Exec(ctx, `UPDATE email_otps SET failed_attempts = $1 WHERE uuid = $2`, newFailures, otpUUID)
			remaining := int32(MaxOtpAttempts - newFailures)
			return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("incorrect code. %d attempts left", remaining))
		}
	}

	RecordIPSuccess(clientIP)

	// Check if this email is registered to ANOTHER account
	var otherUserUUID, otherHandle string
	err = s.db.QueryRow(ctx, `
		SELECT u.uuid, u.id
		FROM users u
		WHERE LOWER(u.email) = LOWER($1) AND u.uuid != $2 AND u.deleted_at IS NULL
		UNION
		SELECT u.uuid, u.id
		FROM user_emails ue
		JOIN users u ON u.uuid = ue.user_uuid
		WHERE LOWER(ue.email) = LOWER($1) AND u.uuid != $2 AND u.deleted_at IS NULL
		LIMIT 1`, email, userCtx.UserUUID).Scan(&otherUserUUID, &otherHandle)

	if err == nil && otherUserUUID != "" {
		// Another account was found! Merge required - do not mark OTP used yet so MergeAccount can complete it.
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
		mailRows, err := s.db.Query(ctx, `SELECT email FROM user_emails WHERE user_uuid = $1 ORDER BY email ASC`, otherUserUUID)
		if err == nil {
			defer mailRows.Close()
			for mailRows.Next() {
				var em string
				if err := mailRows.Scan(&em); err == nil {
					otherEmails = append(otherEmails, em)
				}
			}
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
			OtherUserHandle:     otherHandle,
			OtherBuildCount:     int32(buildCount),
			OtherBuildNames:     buildNames,
			OtherEmails:         otherEmails,
			OtherOauthProviders: oauthProviders,
			Message:             fmt.Sprintf("This email belongs to account @%s (%d builds). Merging required to transfer data.", otherHandle, buildCount),
		}), nil
	}

	// OTP is valid and no other account -> Mark as used and associate directly
	_, _ = s.db.Exec(ctx, `UPDATE email_otps SET used_at = NOW() WHERE uuid = $1`, otpUUID)

	// No other account found -> Add directly to current user's emails
	newUUID := NewUUIDv7()
	_, err = s.db.Exec(ctx, `
		INSERT INTO user_emails (uuid, user_uuid, email, is_primary, verified_at, created_at)
		VALUES ($1, $2, $3, false, NOW(), NOW())
		ON CONFLICT (email) DO UPDATE SET user_uuid = $2, is_primary = false, verified_at = NOW()`,
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
	code := strings.TrimSpace(req.Msg.GetCode())
	token := strings.TrimSpace(req.Msg.GetToken())

	var otpUUID string
	var err error
	if token != "" {
		var tokenEmail string
		otpUUID, tokenEmail, err = VerifyMagicLinkToken(token)
		if err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid or expired magic link: %w", err))
		}
		if email == "" {
			email = tokenEmail
		} else if strings.ToLower(email) != strings.ToLower(tokenEmail) {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("magic link token email mismatch"))
		}

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
		if time.Since(createdAt) > MagicLinkTTL {
			_, _ = s.db.Exec(ctx, `UPDATE email_otps SET used_at = NOW() WHERE uuid = $1`, otpUUID)
			return nil, connect.NewError(connect.CodeDeadlineExceeded, errors.New("magic link has expired"))
		}
	} else {
		if email == "" || code == "" {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("email and 6-digit verification code (or magic link token) required"))
		}

		// Verify OTP
		var codeHash string
		var failedAttempts int32
		var createdAt time.Time
		err = s.db.QueryRow(ctx, `
			SELECT uuid, code_hash, failed_attempts, created_at
			FROM email_otps
			WHERE email = $1 AND used_at IS NULL
			ORDER BY created_at DESC
			LIMIT 1`, email).Scan(&otpUUID, &codeHash, &failedAttempts, &createdAt)
		if err != nil {
			return nil, connect.NewError(connect.CodeNotFound, errors.New("no active verification challenge found for this email"))
		}
		if time.Since(createdAt) > OtpTTL || failedAttempts >= MaxOtpAttempts || !VerifyOTP(code, codeHash) {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("invalid or expired verification code"))
		}
	}
	_, _ = s.db.Exec(ctx, `UPDATE email_otps SET used_at = NOW() WHERE uuid = $1`, otpUUID)

	// Identify the other account
	var otherUserUUID, otherHandle, otherEmail string
	err = s.db.QueryRow(ctx, `
		SELECT u.uuid, u.id, u.email
		FROM users u
		WHERE LOWER(u.email) = LOWER($1) AND u.uuid != $2 AND u.deleted_at IS NULL
		UNION
		SELECT u.uuid, u.id, u.email
		FROM user_emails ue
		JOIN users u ON u.uuid = ue.user_uuid
		WHERE LOWER(ue.email) = LOWER($1) AND u.uuid != $2 AND u.deleted_at IS NULL
		LIMIT 1`, email, userCtx.UserUUID).Scan(&otherUserUUID, &otherHandle, &otherEmail)

	if err != nil || otherUserUUID == "" {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("no other active account found for this email address to merge"))
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
		SET user_uuid = $1, author_handle = $2
		WHERE user_uuid = $3`,
		userCtx.UserUUID, userCtx.UserID, otherUserUUID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to transfer builds: %w", err))
	}

	// 3. Transfer user_emails
	_, err = tx.Exec(ctx, `
		UPDATE user_emails
		SET user_uuid = $1, is_primary = false
		WHERE user_uuid = $2`,
		userCtx.UserUUID, otherUserUUID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to transfer emails: %w", err))
	}

	// 4. Ensure other account's email is present in user_emails for current user
	newEmailUUID := NewUUIDv7()
	_, err = tx.Exec(ctx, `
		INSERT INTO user_emails (uuid, user_uuid, email, is_primary, verified_at, created_at)
		VALUES ($1, $2, $3, false, NOW(), NOW())
		ON CONFLICT (email) DO UPDATE SET user_uuid = $2, is_primary = false, verified_at = NOW()`,
		newEmailUUID, userCtx.UserUUID, otherEmail)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to associate primary email: %w", err))
	}

	// 5. Transfer OAuth accounts
	_, _ = tx.Exec(ctx, `
		UPDATE oauth_accounts
		SET user_uuid = $1
		WHERE user_uuid = $2`,
		userCtx.UserUUID, otherUserUUID)

	// 6. Transfer and record handle history
	_, _ = tx.Exec(ctx, `
		UPDATE handle_histories
		SET user_uuid = $1
		WHERE user_uuid = $2`,
		userCtx.UserUUID, otherUserUUID)

	histUUID := NewUUIDv7()
	_, _ = tx.Exec(ctx, `
		INSERT INTO handle_histories (uuid, user_uuid, old_handle, renamed_at)
		VALUES ($1, $2, $3, NOW())`,
		histUUID, userCtx.UserUUID, otherHandle)

	// 7. Revoke other user's active sessions
	_, _ = tx.Exec(ctx, `DELETE FROM refresh_tokens WHERE user_uuid = $1`, otherUserUUID)

	// 8. Delete other user's profile
	_, _ = tx.Exec(ctx, `DELETE FROM user_profiles WHERE user_uuid = $1`, otherUserUUID)

	// 9. Delete other user record from users table
	_, err = tx.Exec(ctx, `DELETE FROM users WHERE uuid = $1`, otherUserUUID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to finalize merge: %w", err))
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to commit merge: %w", err))
	}

	return connect.NewResponse(&pb.MergeAccountResponse{
		Success:               true,
		Message:               fmt.Sprintf("Successfully merged @%s into your account. %d builds transferred.", otherHandle, buildCount),
		TransferredBuildCount: int32(buildCount),
		MergedHandle:          otherHandle,
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

// ---------------------------------------------------------
// Helper Methods
// ---------------------------------------------------------

func (s *AuthService) returnOtpError(msg string, remaining int32, isExhausted bool) (*connect.Response[pb.AuthResponse], error) {
	detail, _ := connect.NewErrorDetail(&pb.OtpVerificationError{
		Message:           msg,
		RemainingAttempts: remaining,
		MaxAttempts:       MaxOtpAttempts,
		IsExhausted:       isExhausted,
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

func extractClientIP(header http.Header) string {
	clientIP := header.Get("X-Forwarded-For")
	if clientIP == "" {
		clientIP = header.Get("X-Real-IP")
	} else {
		parts := strings.Split(clientIP, ",")
		clientIP = strings.TrimSpace(parts[0])
	}
	return clientIP
}

func (s *AuthService) issueAuthSession(ctx context.Context, email, deviceHint string) (*connect.Response[pb.AuthResponse], error) {
	var user pb.User
	var userUUID, handle, role string
	var emailVerifiedAt, lastRenamedAt, deletedAt, bannedAt, createdAt, updatedAt *time.Time

	// 1. Try finding user via user_emails
	err := s.db.QueryRow(ctx, `
		SELECT u.uuid, u.id, u.email, u.role, u.email_verified_at, u.last_renamed_at, u.deleted_at, u.banned_at, u.created_at, u.updated_at
		FROM user_emails ue
		JOIN users u ON u.uuid = ue.user_uuid
		WHERE LOWER(ue.email) = LOWER($1) AND u.deleted_at IS NULL
		LIMIT 1`, email).Scan(
		&userUUID, &handle, &user.Email, &role,
		&emailVerifiedAt, &lastRenamedAt, &deletedAt, &bannedAt, &createdAt, &updatedAt)

	if errors.Is(err, pgx.ErrNoRows) {
		// Fallback: check users table directly
		err = s.db.QueryRow(ctx, `
			SELECT uuid, id, email, role, email_verified_at, last_renamed_at, deleted_at, banned_at, created_at, updated_at
			FROM users
			WHERE LOWER(email) = LOWER($1) AND deleted_at IS NULL`, email).Scan(
			&userUUID, &handle, &user.Email, &role,
			&emailVerifiedAt, &lastRenamedAt, &deletedAt, &bannedAt, &createdAt, &updatedAt)
	}

	if errors.Is(err, pgx.ErrNoRows) {
		userUUID = NewUUIDv7()
		handle = s.generateUniqueHandle(ctx, email)
		role = "USER"
		now := time.Now()

		_, err = s.db.Exec(ctx, `
			INSERT INTO users (uuid, id, email, role, email_verified_at, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $5, $5)`,
			userUUID, handle, email, role, now)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to create user: %w", err))
		}

		profileUUID := NewUUIDv7()
		_, err = s.db.Exec(ctx, `
			INSERT INTO user_profiles (uuid, user_uuid, display_name, avatar_url, bio, updated_at)
			VALUES ($1, $2, $3, '', '', $4)`,
			profileUUID, userUUID, handle, now)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to create profile: %w", err))
		}

		emailUUID := NewUUIDv7()
		_, _ = s.db.Exec(ctx, `
			INSERT INTO user_emails (uuid, user_uuid, email, is_primary, verified_at, created_at)
			VALUES ($1, $2, $3, true, $4, $4)
			ON CONFLICT (email) DO NOTHING`,
			emailUUID, userUUID, email, now)

		user.Uuid = userUUID
		user.Id = handle
		user.Email = email
		user.Role = role
		user.EmailVerifiedAt = timestamppb.New(now)
		user.CreatedAt = timestamppb.New(now)
		user.UpdatedAt = timestamppb.New(now)
	} else if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	} else {
		// Ensure user_emails has this email
		emailUUID := NewUUIDv7()
		_, _ = s.db.Exec(ctx, `
			INSERT INTO user_emails (uuid, user_uuid, email, is_primary, verified_at, created_at)
			VALUES ($1, $2, $3, true, NOW(), NOW())
			ON CONFLICT (email) DO NOTHING`,
			emailUUID, userUUID, email)

		user.Uuid = userUUID
		user.Id = handle
		user.Role = role
		if emailVerifiedAt == nil {
			_, _ = s.db.Exec(ctx, `UPDATE users SET email_verified_at = NOW() WHERE uuid = $1`, userUUID)
			user.EmailVerifiedAt = timestamppb.Now()
		} else {
			user.EmailVerifiedAt = timestamppb.New(*emailVerifiedAt)
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
	}

	var profile pb.UserProfile
	var profileUpdatedAt *time.Time
	_ = s.db.QueryRow(ctx, `
		SELECT uuid, user_uuid, display_name, avatar_url, bio, updated_at
		FROM user_profiles WHERE user_uuid = $1`, userUUID).Scan(
		&profile.Uuid, &profile.UserUuid, &profile.DisplayName, &profile.AvatarUrl, &profile.Bio, &profileUpdatedAt)
	if profileUpdatedAt != nil {
		profile.UpdatedAt = timestamppb.New(*profileUpdatedAt)
	}

	sessionUUID := NewUUIDv7()
	rawRefreshToken, tokenHash, err := GenerateRefreshToken()
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	now := time.Now()
	expiresAt := now.Add(RefreshTokenTTL)
	_, err = s.db.Exec(ctx, `
		INSERT INTO refresh_tokens (uuid, user_uuid, token_hash, expires_at, last_used_at, device_hint, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $5)`,
		sessionUUID, userUUID, tokenHash, expiresAt, now, deviceHint)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to save session: %w", err))
	}

	accessToken, err := SignAccessToken(userUUID, handle, email, role, sessionUUID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to sign access token: %w", err))
	}

	res := connect.NewResponse(&pb.AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: rawRefreshToken,
		User:         &user,
		Profile:      &profile,
	})

	cookie := &http.Cookie{
		Name:     "qs_session",
		Value:    rawRefreshToken,
		Path:     "/",
		MaxAge:   int(RefreshTokenTTL.Seconds()),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	}
	res.Header().Set("Set-Cookie", cookie.String())

	return res, nil
}

var handleStructureRegex = regexp.MustCompile(`^[a-z][a-z0-9_]{1,18}[a-z0-9]$`)

// ValidatePilotHandle validates a candidate pilot handle against the rules declared in user.proto.
//
// Rules enforced:
// 1. Length between 3 and 20 characters.
// 2. Starts with a lowercase letter, ends with an alphanumeric character.
// 3. Contains only lowercase letters, digits, and underscores.
// 4. Cannot contain consecutive underscores ("__").
// 5. Cannot contain "quadsmith" or "quad_smith".
// 6. Cannot equal reserved handles "admin" or "root".
func ValidatePilotHandle(handle string) error {
	if !handleStructureRegex.MatchString(handle) {
		return errors.New("handle must be 3-20 characters, start with a letter, end with alphanumeric, and contain only lowercase letters, numbers, and underscores")
	}
	if strings.Contains(handle, "__") {
		return errors.New("handle cannot contain consecutive underscores")
	}
	if strings.Contains(handle, "quadsmith") || strings.Contains(handle, "quad_smith") {
		return errors.New("handle cannot contain 'quadsmith' or 'quad_smith'")
	}
	if handle == "admin" || handle == "root" {
		return errors.New("reserved handle")
	}
	return nil
}

func (s *AuthService) generateUniqueHandle(ctx context.Context, email string) string {
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
	cleaned = strings.ReplaceAll(cleaned, "quadsmith", "pilot")
	cleaned = strings.ReplaceAll(cleaned, "quad_smith", "pilot")
	if len(cleaned) == 0 || cleaned[0] < 'a' || cleaned[0] > 'z' {
		cleaned = "pilot_" + cleaned
	}
	cleaned = strings.Trim(cleaned, "_")
	if len(cleaned) < 3 {
		cleaned = "pilot_" + cleaned
	}
	if len(cleaned) > 20 {
		cleaned = cleaned[:20]
		cleaned = strings.TrimRight(cleaned, "_")
	}
	if cleaned == "admin" || cleaned == "root" {
		cleaned = "pilot_" + cleaned
	}

	candidate := cleaned
	counter := 1
	for {
		var exists bool
		err := s.db.QueryRow(ctx, `
			SELECT EXISTS(SELECT 1 FROM users WHERE id = $1)
			OR EXISTS(SELECT 1 FROM handle_histories WHERE old_handle = $1)`, candidate).Scan(&exists)
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

// ---------------------------------------------------------
// UserService
// ---------------------------------------------------------

type UserService struct {
	quadsmithconnect.UnimplementedUserServiceHandler
	db *pgxpool.Pool
}

func NewUserService(db *pgxpool.Pool) *UserService {
	return &UserService{db: db}
}

func (s *UserService) GetUser(ctx context.Context, req *connect.Request[pb.GetUserRequest]) (*connect.Response[pb.User], error) {
	id := req.Msg.GetId()
	if id == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("user id or handle required"))
	}

	var user pb.User
	var emailVerifiedAt, lastRenamedAt, deletedAt, bannedAt, createdAt, updatedAt *time.Time
	err := s.db.QueryRow(ctx, `
		SELECT uuid, id, email, role, email_verified_at, last_renamed_at, deleted_at, banned_at, created_at, updated_at
		FROM users
		WHERE (id = $1 OR uuid::text = $1) AND deleted_at IS NULL`, id).Scan(
		&user.Uuid, &user.Id, &user.Email, &user.Role,
		&emailVerifiedAt, &lastRenamedAt, &deletedAt, &bannedAt, &createdAt, &updatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, connect.NewError(connect.CodeNotFound, errors.New("user not found"))
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if emailVerifiedAt != nil {
		user.EmailVerifiedAt = timestamppb.New(*emailVerifiedAt)
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
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("user id or handle required"))
	}

	var userUUID, handle string
	var createdAt *time.Time
	var displayName, avatarUrl, bio *string

	err := s.db.QueryRow(ctx, `
		SELECT u.uuid, u.id, u.created_at, p.display_name, p.avatar_url, p.bio
		FROM users u
		LEFT JOIN user_profiles p ON p.user_uuid = u.uuid
		WHERE (u.id = $1 OR u.uuid::text = $1) AND u.deleted_at IS NULL`, id).Scan(
		&userUUID, &handle, &createdAt, &displayName, &avatarUrl, &bio)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, connect.NewError(connect.CodeNotFound, errors.New("pilot profile not found"))
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	var buildCount int32
	_ = s.db.QueryRow(ctx, `SELECT COUNT(*) FROM builds WHERE user_uuid = $1`, userUUID).Scan(&buildCount)

	pub := &pb.PublicUserProfile{
		Id:         handle,
		BuildCount: buildCount,
	}
	if displayName != nil && *displayName != "" {
		pub.DisplayName = *displayName
	} else {
		pub.DisplayName = handle
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

func (s *UserService) CheckHandleAvailable(ctx context.Context, req *connect.Request[pb.CheckHandleAvailableRequest]) (*connect.Response[pb.CheckHandleAvailableResponse], error) {
	handle := strings.ToLower(strings.TrimSpace(req.Msg.GetHandle()))
	if err := ValidatePilotHandle(handle); err != nil {
		return connect.NewResponse(&pb.CheckHandleAvailableResponse{
			Available: false,
			Message:   err.Error(),
		}), nil
	}

	var exists bool
	err := s.db.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM users WHERE LOWER(id) = $1)
		OR EXISTS(SELECT 1 FROM handle_histories WHERE LOWER(old_handle) = $1)`, handle).Scan(&exists)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	if exists {
		return connect.NewResponse(&pb.CheckHandleAvailableResponse{
			Available: false,
			Message:   "This handle is already taken",
		}), nil
	}

	return connect.NewResponse(&pb.CheckHandleAvailableResponse{
		Available: true,
		Message:   "Handle is available",
	}), nil
}
