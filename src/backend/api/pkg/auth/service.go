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

	err := s.db.QueryRow(ctx, `
		SELECT uuid, id, email, role, email_verified_at, last_renamed_at, deleted_at, banned_at, created_at, updated_at
		FROM users
		WHERE email = $1`, email).Scan(
		&userUUID, &handle, &user.Email, &role,
		&emailVerifiedAt, &lastRenamedAt, &deletedAt, &bannedAt, &createdAt, &updatedAt)

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

func (s *AuthService) generateUniqueHandle(ctx context.Context, email string) string {
	parts := strings.Split(email, "@")
	base := strings.ToLower(parts[0])
	if idx := strings.Index(base, "+"); idx != -1 {
		base = base[:idx]
	}
	reg := regexp.MustCompile(`[^a-z0-9_]`)
	cleaned := reg.ReplaceAllString(base, "_")
	cleaned = strings.Trim(cleaned, "_")
	if len(cleaned) < 3 {
		cleaned = "pilot_" + cleaned
	}
	if len(cleaned) > 20 {
		cleaned = cleaned[:20]
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
		candidate = fmt.Sprintf("%s_%d", cleaned, counter)
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
	matched, _ := regexp.MatchString(`^[a-z0-9_]{3,30}$`, handle)
	if !matched {
		return connect.NewResponse(&pb.CheckHandleAvailableResponse{
			Available: false,
			Message:   "Handle must be 3-30 characters and contain only lowercase letters, numbers, and underscores",
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
