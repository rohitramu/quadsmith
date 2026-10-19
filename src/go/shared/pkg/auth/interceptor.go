package auth

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"connectrpc.com/connect"
	"github.com/jackc/pgx/v5/pgxpool"

	"quadsmith/shared/gen/quadsmith/api/apiconnect"
)

var publicProcedures = map[string]bool{
	// AuthService public endpoints
	apiconnect.AuthServiceSendEmailOtpProcedure:             true,
	apiconnect.AuthServiceVerifyEmailOtpProcedure:           true,
	apiconnect.AuthServiceVerifyMagicLinkProcedure:          true,
	apiconnect.AuthServiceGetLoginChallengeDetailsProcedure: true,
	apiconnect.AuthServiceApproveLoginChallengeProcedure:    true,
	apiconnect.AuthServiceDenyLoginChallengeProcedure:       true,
	apiconnect.AuthServiceGetLoginChallengeStatusProcedure:  true,
	apiconnect.AuthServiceCompleteAccountSetupProcedure:     true,
	apiconnect.AuthServiceRefreshTokenProcedure:             true,
	apiconnect.AuthServiceStartOAuthProcedure:               true,
	apiconnect.AuthServiceCompleteOAuthProcedure:            true,

	// UserService public endpoints
	apiconnect.UserServiceCheckUsernameAvailableProcedure: true,
	apiconnect.UserServiceGetPublicProfileProcedure:       true,

	// Public social queries
	apiconnect.SocialServiceListReviewsProcedure:            true,
	apiconnect.SocialServiceListReviewCommentsProcedure:     true,
	apiconnect.SocialServiceListQuestionsProcedure:          true,
	apiconnect.SocialServiceListAnswersProcedure:            true,
	apiconnect.SocialServiceGetEntitySocialSummaryProcedure: true,
	apiconnect.SocialServiceGetCollectionProcedure:          true,
	apiconnect.SocialServiceListUserCollectionsProcedure:    true,
	apiconnect.SocialServiceListCollectionItemsProcedure:    true,

	// Search, Evaluator, Compatibility, Link Preview
	apiconnect.SearchServiceSearchProcedure:                           true,
	apiconnect.EvaluatorServiceEvaluateBuildProcedure:                 true,
	apiconnect.EvaluatorServiceGetBuildElectricalLimitsProcedure:      true,
	apiconnect.EvaluatorServiceEvaluateComponentsProcedure:            true,
	apiconnect.EvaluatorServiceGetComponentsElectricalLimitsProcedure: true,
	apiconnect.CompatibilityServiceCheckCompatibilityProcedure:        true,
	apiconnect.LinkPreviewServiceGetLinkPreviewProcedure:              true,

	// Build service
	apiconnect.BuildServiceGetBuildProcedure:   true,
	apiconnect.BuildServiceListBuildsProcedure: true,

	// Component & Hardware Catalog (Get and List)
	apiconnect.MotorServiceGetMotorProcedure:                                           true,
	apiconnect.MotorServiceListMotorsProcedure:                                         true,
	apiconnect.FrameServiceGetFrameProcedure:                                           true,
	apiconnect.FrameServiceListFramesProcedure:                                         true,
	apiconnect.BatteryServiceGetBatteryProcedure:                                       true,
	apiconnect.BatteryServiceListBatteriesProcedure:                                    true,
	apiconnect.ElectronicSpeedControllerServiceGetElectronicSpeedControllerProcedure:   true,
	apiconnect.ElectronicSpeedControllerServiceListElectronicSpeedControllersProcedure: true,
	apiconnect.FlightControllerServiceGetFlightControllerProcedure:                     true,
	apiconnect.FlightControllerServiceListFlightControllersProcedure:                   true,
	apiconnect.ReceiverServiceGetReceiverProcedure:                                     true,
	apiconnect.ReceiverServiceListReceiversProcedure:                                   true,
	apiconnect.VideoTransmitterServiceGetVideoTransmitterProcedure:                     true,
	apiconnect.VideoTransmitterServiceListVideoTransmittersProcedure:                   true,
	apiconnect.AntennaServiceGetAntennaProcedure:                                       true,
	apiconnect.AntennaServiceListAntennasProcedure:                                     true,
	apiconnect.CameraServiceGetCameraProcedure:                                         true,
	apiconnect.CameraServiceListCamerasProcedure:                                       true,
	apiconnect.PropellerServiceGetPropellerProcedure:                                   true,
	apiconnect.PropellerServiceListPropellersProcedure:                                 true,
	apiconnect.GpsReceiverServiceGetGpsReceiverProcedure:                               true,
	apiconnect.GpsReceiverServiceListGpsReceiversProcedure:                             true,
	apiconnect.FcFirmwareServiceGetFcFirmwareProcedure:                                 true,
	apiconnect.FcFirmwareServiceListFcFirmwaresProcedure:                               true,
	apiconnect.EscFirmwareServiceGetEscFirmwareProcedure:                               true,
	apiconnect.EscFirmwareServiceListEscFirmwaresProcedure:                             true,
	apiconnect.VtxFirmwareServiceGetVtxFirmwareProcedure:                               true,
	apiconnect.VtxFirmwareServiceListVtxFirmwaresProcedure:                             true,
	apiconnect.RadioFirmwareServiceGetRadioFirmwareProcedure:                           true,
	apiconnect.RadioFirmwareServiceListRadioFirmwaresProcedure:                         true,
	apiconnect.SoftwareToolServiceGetSoftwareToolProcedure:                             true,
	apiconnect.SoftwareToolServiceListSoftwareToolsProcedure:                           true,
	apiconnect.TransmitterServiceGetTransmitterProcedure:                               true,
	apiconnect.TransmitterServiceListTransmittersProcedure:                             true,
	apiconnect.GogglesServiceGetGogglesProcedure:                                       true,
	apiconnect.GogglesServiceListGogglesProcedure:                                      true,
	apiconnect.ChargerServiceGetChargerProcedure:                                       true,
	apiconnect.ChargerServiceListChargersProcedure:                                     true,
	apiconnect.FieldToolServiceGetFieldToolProcedure:                                   true,
	apiconnect.FieldToolServiceListFieldToolsProcedure:                                 true,
}

// IsPublicProcedure checks whether a ConnectRPC procedure may be accessed anonymously.
func IsPublicProcedure(procedure string) bool {
	if publicProcedures[procedure] {
		return true
	}
	// Fallback heuristic: Any Get* or List* method on domain services (excluding AuthService and UserService)
	if strings.HasPrefix(procedure, "/quadsmith.api.") {
		parts := strings.Split(procedure, "/")
		if len(parts) == 3 {
			service := parts[1]
			method := parts[2]
			if service != "quadsmith.api.AuthService" && service != "quadsmith.api.UserService" {
				if strings.HasPrefix(method, "Get") || strings.HasPrefix(method, "List") {
					return true
				}
			}
		}
	}
	return false
}

var allowedBannedProcedures = map[string]bool{
	// AuthService
	apiconnect.AuthServiceSendEmailOtpProcedure:          true,
	apiconnect.AuthServiceVerifyEmailOtpProcedure:        true,
	apiconnect.AuthServiceRefreshTokenProcedure:          true,
	apiconnect.AuthServiceLogoutProcedure:                true,
	apiconnect.AuthServiceUpdateProfileProcedure:         true,
	apiconnect.AuthServiceListActiveSessionsProcedure:    true,
	apiconnect.AuthServiceRevokeSessionProcedure:         true,
	apiconnect.AuthServiceRevokeOtherSessionsProcedure:   true,
	apiconnect.AuthServiceGetCurrentUserProcedure:        true,
	apiconnect.AuthServiceListUserEmailsProcedure:        true,
	apiconnect.AuthServiceDeleteAccountProcedure:         true,
	apiconnect.AuthServiceCancelAccountDeletionProcedure: true,
	apiconnect.AuthServiceStartOAuthProcedure:            true,
	apiconnect.AuthServiceCompleteOAuthProcedure:         true,

	// UserService
	apiconnect.UserServiceGetUserBanStatusProcedure: true,
}

// IsAllowedBannedProcedure checks whether a ConnectRPC procedure may be accessed by a banned user.
func IsAllowedBannedProcedure(procedure string) bool {
	return allowedBannedProcedures[procedure]
}

// NewAuthInterceptor creates a ConnectRPC unary interceptor for authentication
func NewAuthInterceptor(pool *pgxpool.Pool) connect.UnaryInterceptorFunc {
	return func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			procedure := req.Spec().Procedure
			header := req.Header()
			httpReq := &http.Request{Header: header}
			userCtx := AuthenticateHTTPRequest(ctx, pool, httpReq)

			// If authenticated, inject user context and check active ban
			if userCtx != nil {
				var isBanned bool
				_ = pool.QueryRow(ctx, `
					SELECT EXISTS (
						SELECT 1 FROM bans
						WHERE user_uuid = $1::uuid
						  AND revoked_at IS NULL
						  AND (expires_at IS NULL OR expires_at > NOW())
					)`, userCtx.UserUUID).Scan(&isBanned)
				userCtx.IsBanned = isBanned
				ctx = WithUserContext(ctx, userCtx)

				// If user is banned, restrict access strictly to allowed procedures
				if isBanned && !allowedBannedProcedures[procedure] {
					return nil, connect.NewError(connect.CodePermissionDenied, errors.New("account suspended: access restricted to account settings"))
				}
			}

			// 3. Enforce authentication on protected endpoints
			if !IsPublicProcedure(procedure) && userCtx == nil {
				return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("authentication required"))
			}

			return next(ctx, req)
		}
	}
}

// AuthenticateHTTPRequest validates an incoming HTTP request via Bearer token or session cookie.
func AuthenticateHTTPRequest(ctx context.Context, pool *pgxpool.Pool, r *http.Request) *UserContext {
	// 1. Try Bearer token in Authorization header
	authHeader := r.Header.Get("Authorization")
	if strings.HasPrefix(strings.ToLower(authHeader), "bearer ") {
		tokenStr := strings.TrimSpace(authHeader[7:])
		if claims, err := VerifyAccessToken(tokenStr); err == nil && claims != nil {
			if pool != nil {
				if claims.UserUUID == SystemUserUUID {
					if claims.SessionUUID == "" {
						return nil
					}
					var role string
					err := pool.QueryRow(ctx, `
						SELECT u.role
						FROM users u
						JOIN refresh_tokens r ON r.user_uuid = u.uuid
						WHERE u.uuid = $1::uuid AND r.uuid = $2::uuid AND r.expires_at > NOW()`,
						claims.UserUUID, claims.SessionUUID,
					).Scan(&role)
					if err != nil || role == "" {
						return nil
					}
					claims.Role = role
				} else {
					var role string
					err := pool.QueryRow(ctx, `SELECT role FROM users WHERE uuid::text = $1`, claims.UserUUID).Scan(&role)
					if err == nil && role != "" {
						claims.Role = role
					}
				}
			}
			return claims
		}
	}

	// 2. If no valid Bearer token, try session cookie (qs_session)
	if c, err := r.Cookie("qs_session"); err == nil && c.Value != "" {
		tokenHash := HashToken(c.Value)
		var sessionUUID, userUUID, username, email, role string
		err := pool.QueryRow(ctx, `
			SELECT r.uuid, u.uuid, u.id, COALESCE(ue.email, ''), u.role
			FROM refresh_tokens r
			JOIN users u ON u.uuid = r.user_uuid
			LEFT JOIN user_emails ue ON ue.user_uuid = u.uuid AND ue.is_primary = true
			WHERE r.token_hash = $1 AND r.expires_at > NOW() AND (u.deleted_at IS NULL OR u.deleted_at + INTERVAL '30 days' > NOW())
			LIMIT 1`, tokenHash).Scan(&sessionUUID, &userUUID, &username, &email, &role)
		if err == nil {
			return &UserContext{
				UserUUID:    userUUID,
				UserID:      username,
				Email:       email,
				Role:        role,
				SessionUUID: sessionUUID,
			}
		}
	}

	return nil
}
