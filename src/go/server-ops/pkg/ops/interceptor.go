package ops

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/jackc/pgx/v5/pgxpool"

	"quadsmith/shared/gen/quadsmith/config"
)

type OpsInterceptor struct {
	opsDb *pgxpool.Pool
	cfg   *config.OpsServerConfig
}

func NewOpsInterceptor(opsDb *pgxpool.Pool, cfg *config.OpsServerConfig) *OpsInterceptor {
	return &OpsInterceptor{opsDb: opsDb, cfg: cfg}
}

func (in *OpsInterceptor) WrapUnary(next connect.UnaryFunc) connect.UnaryFunc {
	return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		procedure := req.Spec().Procedure
		methodName := procedure
		if idx := strings.LastIndex(procedure, "/"); idx != -1 {
			methodName = procedure[idx+1:]
		}

		// Public endpoint: ExchangeOAuthCode does not require auth
		if methodName == "ExchangeOAuthCode" {
			return next(ctx, req)
		}

		// 1. Authenticate Operator Token
		authHeader := req.Header().Get("Authorization")
		token := strings.TrimPrefix(authHeader, "Bearer ")
		token = strings.TrimSpace(token)
		if token == "" {
			return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("unauthenticated: missing bearer token in Authorization header"))
		}

		opCtx, err := in.authenticateToken(ctx, token)
		if err != nil {
			return nil, err
		}

		// 2. Group Authorization
		if err := in.authorizeGroup(opCtx.GroupName, methodName); err != nil {
			return nil, err
		}

		// 3. JIT Elevation Check
		if in.isSensitiveMethod(methodName) {
			grantUUID, grantID, err := in.checkJitGrant(ctx, opCtx.Operator, methodName)
			if err != nil {
				return nil, err
			}
			opCtx.JitGrantUUID = grantUUID
			opCtx.JitGrantID = grantID
		}

		// Inject operator context
		ctx = WithOperatorContext(ctx, opCtx)

		// 4. Execution & Audit Logging
		t0 := time.Now()
		res, err := next(ctx, req)
		durationMs := time.Since(t0).Milliseconds()

		status := "OK"
		var errMsg *string
		if err != nil {
			status = "ERROR"
			msg := err.Error()
			errMsg = &msg
		}

		// Asynchronously log audit entry
		go in.logAudit(methodName, opCtx, status, errMsg, durationMs)

		return res, err
	}
}

func (in *OpsInterceptor) WrapStreamingClient(next connect.StreamingClientFunc) connect.StreamingClientFunc {
	return next
}

func (in *OpsInterceptor) WrapStreamingHandler(next connect.StreamingHandlerFunc) connect.StreamingHandlerFunc {
	return next
}

func (in *OpsInterceptor) authenticateToken(ctx context.Context, token string) (*OperatorContext, error) {
	for _, bt := range in.cfg.BootstrapTokens {
		if bt.GetToken() != "" && bt.GetToken() == token {
			return &OperatorContext{
				Operator:     bt.GetName(),
				OperatorType: bt.GetType(),
				GroupName:    bt.GetGroupName(),
			}, nil
		}
	}

	// Dynamic token check in quadsmith_ops.internal_tokens
	tokenHash := HashToken(token)
	var name, tokenType, groupName string
	var expiresAt *time.Time
	var revokedAt *time.Time

	err := in.opsDb.QueryRow(ctx, `
		SELECT name, type, group_name, expires_at, revoked_at
		FROM internal_tokens
		WHERE token_hash = $1
		ORDER BY created_at DESC
		LIMIT 1`, tokenHash).Scan(&name, &tokenType, &groupName, &expiresAt, &revokedAt)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("invalid or unapproved operations token"))
	}

	if revokedAt != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("operations token has been revoked"))
	}
	if expiresAt != nil && expiresAt.Before(time.Now()) {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("operations token has expired"))
	}

	return &OperatorContext{
		Operator:     name,
		OperatorType: tokenType,
		GroupName:    groupName,
	}, nil
}

func (in *OpsInterceptor) authorizeGroup(groupName, methodName string) error {
	var group *config.InternalGroupConfig
	for _, g := range in.cfg.Groups {
		if g.GetName() == groupName {
			group = g
			break
		}
	}
	if group == nil {
		return connect.NewError(connect.CodePermissionDenied, fmt.Errorf("group %q is not configured", groupName))
	}

	for _, rule := range group.AllowedMethods {
		if rule == "*" || rule == methodName {
			return nil
		}
		if strings.HasSuffix(rule, "*") {
			prefix := strings.TrimSuffix(rule, "*")
			if strings.HasPrefix(methodName, prefix) {
				return nil
			}
		}
	}

	return connect.NewError(connect.CodePermissionDenied, fmt.Errorf("group %q is not authorized to execute %s", groupName, methodName))
}

func (in *OpsInterceptor) isSensitiveMethod(methodName string) bool {
	if in.cfg.Jit == nil || !in.cfg.Jit.GetEnabled() {
		return false
	}
	for _, m := range in.cfg.Jit.SensitiveMethods {
		if m == "*" || m == methodName {
			return true
		}
	}
	return false
}

func (in *OpsInterceptor) checkJitGrant(ctx context.Context, operator, methodName string) (string, string, error) {
	var grantUUID, grantID string
	var expiresAt time.Time
	err := in.opsDb.QueryRow(ctx, `
		SELECT uuid::text, id, expires_at
		FROM jit_grants
		WHERE operator = $1 AND status = 'ACTIVE' AND expires_at > NOW() AND (operation = '*' OR operation = $2)
		ORDER BY expires_at DESC
		LIMIT 1`, operator, methodName).Scan(&grantUUID, &grantID, &expiresAt)
	if err != nil {
		return "", "", connect.NewError(connect.CodePermissionDenied, fmt.Errorf("procedure %s requires active JIT elevation: acquire a lease via 'qsops jit request'", methodName))
	}
	return grantUUID, grantID, nil
}

func (in *OpsInterceptor) logAudit(method string, opCtx *OperatorContext, status string, errMsg *string, durationMs int64) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	callID := fmt.Sprintf("call_%d", time.Now().UnixNano())
	var jitUUID *string
	if opCtx.JitGrantUUID != "" {
		jitUUID = &opCtx.JitGrantUUID
	}

	_, err := in.opsDb.Exec(ctx, `
		INSERT INTO internal_api_calls (
			uuid, id, method, caller, caller_type, group_name, jit_grant_uuid, status, error_message, duration_ms, created_at
		) VALUES (
			gen_random_uuid(), $1, $2, $3, $4, $5, $6, $7, $8, $9, NOW()
		)`,
		callID, method, opCtx.Operator, opCtx.OperatorType, opCtx.GroupName, jitUUID, status, errMsg, durationMs,
	)
	if err != nil {
		log.Printf("[AuditLogger] Failed to write audit record for %s: %v", method, err)
	}
}

func HashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}
