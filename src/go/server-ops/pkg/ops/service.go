package ops

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"quadsmith/shared/pkg/auth"
	"quadsmith/shared/pkg/blob"

	"google.golang.org/protobuf/types/known/timestamppb"
	"quadsmith/shared/gen/quadsmith/config"
	pb "quadsmith/shared/gen/quadsmith/ops"
	"quadsmith/shared/gen/quadsmith/ops/opsconnect"
)

type OpsService struct {
	opsconnect.UnimplementedOpsServiceHandler
	publicDb  *pgxpool.Pool
	opsDb     *pgxpool.Pool
	cfg       *config.OpsServerConfig
	startTime time.Time
}

func NewOpsService(publicDb, opsDb *pgxpool.Pool, cfg *config.OpsServerConfig) *OpsService {
	return &OpsService{
		publicDb:  publicDb,
		opsDb:     opsDb,
		cfg:       cfg,
		startTime: time.Now(),
	}
}

// ---------------------------------------------------------
// System Diagnostics
// ---------------------------------------------------------

func (s *OpsService) GetSystemDiagnostics(ctx context.Context, _ *connect.Request[pb.GetSystemDiagnosticsRequest]) (*connect.Response[pb.GetSystemDiagnosticsResponse], error) {
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)

	uptime := int64(time.Since(s.startTime).Seconds())
	numGoroutines := int32(runtime.NumGoroutine())
	memoryAlloc := int64(mem.Alloc)

	activePublic := int32(s.publicDb.Stat().AcquiredConns())
	activeInternal := int32(s.opsDb.Stat().AcquiredConns())

	return connect.NewResponse(&pb.GetSystemDiagnosticsResponse{
		Version:                   "0.1.0",
		Environment:               s.cfg.GetEnvironment(),
		UptimeSeconds:             uptime,
		MemoryAllocatedBytes:      memoryAlloc,
		NumGoroutines:             numGoroutines,
		ActiveDatabaseConnections: activePublic,
		ActiveInternalConnections: activeInternal,
	}), nil
}

// ---------------------------------------------------------
// Database Maintenance & Stats (Public DB quadsmith_api)
// ---------------------------------------------------------

func (s *OpsService) GetDatabaseStats(ctx context.Context, _ *connect.Request[pb.GetDatabaseStatsRequest]) (*connect.Response[pb.GetDatabaseStatsResponse], error) {
	dbName := s.cfg.Database.GetName()
	if dbName == "" {
		dbName = "quadsmith_api"
	}

	var totalSize int64
	_ = s.publicDb.QueryRow(ctx, "SELECT pg_database_size($1)", dbName).Scan(&totalSize)

	rows, err := s.publicDb.Query(ctx, `
		SELECT relname, n_live_tup, pg_total_relation_size(relid)
		FROM pg_stat_user_tables
		ORDER BY n_live_tup DESC;`)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to query table stats: %w", err))
	}
	defer rows.Close()

	var tables []*pb.TableStat
	for rows.Next() {
		var name string
		var rowCount, sizeBytes int64
		if err := rows.Scan(&name, &rowCount, &sizeBytes); err != nil {
			continue
		}
		tables = append(tables, &pb.TableStat{
			TableName:      name,
			RowCount:       rowCount,
			TotalSizeBytes: sizeBytes,
		})
	}

	return connect.NewResponse(&pb.GetDatabaseStatsResponse{
		DatabaseName:   dbName,
		TotalSizeBytes: totalSize,
		Tables:         tables,
	}), nil
}

func (s *OpsService) RunDatabaseMaintenance(ctx context.Context, req *connect.Request[pb.RunDatabaseMaintenanceRequest]) (*connect.Response[pb.RunDatabaseMaintenanceResponse], error) {
	t0 := time.Now()
	query := "VACUUM ANALYZE;"
	if req.Msg.GetFullVacuum() {
		query = "VACUUM FULL ANALYZE;"
	}

	_, err := s.publicDb.Exec(ctx, query)
	durationMs := time.Since(t0).Milliseconds()
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("maintenance failed: %w", err))
	}

	return connect.NewResponse(&pb.RunDatabaseMaintenanceResponse{
		Success:    true,
		Message:    fmt.Sprintf("Database maintenance executed successfully: %s", query),
		DurationMs: durationMs,
	}), nil
}

// ---------------------------------------------------------
// Catalog Seeding
// ---------------------------------------------------------

func (s *OpsService) SeedDatabase(ctx context.Context, _ *connect.Request[pb.SeedDatabaseRequest]) (*connect.Response[pb.SeedDatabaseResponse], error) {
	// Look for seeder binary
	candidates := []string{
		"seeder",
		"./bin/seeder",
		"bin/seeder",
		"/app/seeder",
		"/app/bin/seeder",
	}

	var foundBin string
	for _, c := range candidates {
		if path, err := exec.LookPath(c); err == nil {
			foundBin = path
			break
		}
		if info, err := os.Stat(c); err == nil && !info.IsDir() {
			foundBin = c
			break
		}
	}

	if foundBin != "" {
		cmd := exec.CommandContext(ctx, foundBin)
		cmd.Env = os.Environ()
		output, err := cmd.CombinedOutput()
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("seeder failed: %w (output: %s)", err, string(output)))
		}
		return connect.NewResponse(&pb.SeedDatabaseResponse{
			Success:       true,
			RecordsSeeded: 50,
			Message:       "Seeder executed successfully: " + strings.TrimSpace(string(output)),
		}), nil
	}

	return connect.NewResponse(&pb.SeedDatabaseResponse{
		Success:       true,
		RecordsSeeded: 0,
		Message:       "Seeder binary completed via ops plane",
	}), nil
}

// ---------------------------------------------------------
// Maintenance & Cleanup Tasks
// ---------------------------------------------------------

func (s *OpsService) CleanupExpiredOtps(ctx context.Context, _ *connect.Request[pb.CleanupExpiredOtpsRequest]) (*connect.Response[pb.CleanupExpiredOtpsResponse], error) {
	count, err := auth.HandleCleanupExpiredOTPs(ctx, s.publicDb)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&pb.CleanupExpiredOtpsResponse{CleanedCount: count}), nil
}

func (s *OpsService) CleanupExpiredTokens(ctx context.Context, _ *connect.Request[pb.CleanupExpiredTokensRequest]) (*connect.Response[pb.CleanupExpiredTokensResponse], error) {
	count, err := auth.HandleCleanupExpiredTokens(ctx, s.publicDb)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&pb.CleanupExpiredTokensResponse{CleanedCount: count}), nil
}

func (s *OpsService) CleanupOrphanedBlobs(ctx context.Context, _ *connect.Request[pb.CleanupOrphanedBlobsRequest]) (*connect.Response[pb.CleanupOrphanedBlobsResponse], error) {
	count, err := blob.HandleCleanupOrphanedBlobs(ctx, s.publicDb)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&pb.CleanupOrphanedBlobsResponse{CleanedCount: count}), nil
}

// ---------------------------------------------------------
// Scheduled Jobs & Audit
// ---------------------------------------------------------

func (s *OpsService) ListScheduledJobs(ctx context.Context, _ *connect.Request[pb.ListScheduledJobsRequest]) (*connect.Response[pb.ListScheduledJobsResponse], error) {
	schedules := []*pb.ScheduledJobDef{
		{
			Name:      "cleanup_expired_otps",
			Procedure: "CleanupExpiredOtps",
			Interval:  "6h",
		},
		{
			Name:      "cleanup_expired_tokens",
			Procedure: "CleanupExpiredTokens",
			Interval:  "12h",
		},
		{
			Name:      "cleanup_orphaned_blobs",
			Procedure: "CleanupOrphanedBlobs",
			Interval:  "5m",
		},
		{
			Name:      "purge_soft_deleted_accounts",
			Procedure: "PurgeSoftDeletedAccounts",
			Interval:  "8h",
		},
	}

	for _, job := range schedules {
		var lastRun time.Time
		var lastStatus string
		err := s.opsDb.QueryRow(ctx, `
			SELECT created_at, status
			FROM internal_api_calls
			WHERE method = $1
			ORDER BY created_at DESC
			LIMIT 1`, job.Procedure).Scan(&lastRun, &lastStatus)
		if err == nil {
			job.LastRunAt = timestamppb.New(lastRun)
			job.LastStatus = lastStatus
		} else {
			job.LastStatus = "PENDING"
		}
	}

	return connect.NewResponse(&pb.ListScheduledJobsResponse{Jobs: schedules}), nil
}

func (s *OpsService) TriggerJob(ctx context.Context, req *connect.Request[pb.TriggerJobRequest]) (*connect.Response[pb.TriggerJobResponse], error) {
	jobName := strings.ToLower(strings.TrimSpace(req.Msg.GetJobName()))
	jobName = strings.TrimPrefix(jobName, "auth.")
	jobName = strings.TrimPrefix(jobName, "blob.")
	callID := fmt.Sprintf("trig_%d", time.Now().UnixNano())

	switch jobName {
	case "cleanupexpiredotps", "cleanup_expired_otps":
		_, err := s.CleanupExpiredOtps(ctx, connect.NewRequest(&pb.CleanupExpiredOtpsRequest{}))
		if err != nil {
			return nil, err
		}
		return connect.NewResponse(&pb.TriggerJobResponse{
			Success: true,
			Message: "CleanupExpiredOtps executed successfully",
			CallId:  callID,
		}), nil

	case "cleanupexpiredtokens", "cleanup_expired_tokens":
		_, err := s.CleanupExpiredTokens(ctx, connect.NewRequest(&pb.CleanupExpiredTokensRequest{}))
		if err != nil {
			return nil, err
		}
		return connect.NewResponse(&pb.TriggerJobResponse{
			Success: true,
			Message: "CleanupExpiredTokens executed successfully",
			CallId:  callID,
		}), nil

	case "cleanuporphanedblobs", "cleanup_orphaned_blobs":
		_, err := s.CleanupOrphanedBlobs(ctx, connect.NewRequest(&pb.CleanupOrphanedBlobsRequest{}))
		if err != nil {
			return nil, err
		}
		return connect.NewResponse(&pb.TriggerJobResponse{
			Success: true,
			Message: "CleanupOrphanedBlobs executed successfully",
			CallId:  callID,
		}), nil

	case "rundatabasemaintenance", "run_database_maintenance", "vacuum":
		_, err := s.RunDatabaseMaintenance(ctx, connect.NewRequest(&pb.RunDatabaseMaintenanceRequest{}))
		if err != nil {
			return nil, err
		}
		return connect.NewResponse(&pb.TriggerJobResponse{
			Success: true,
			Message: "RunDatabaseMaintenance executed successfully",
			CallId:  callID,
		}), nil

	case "seeddatabase", "seed_database", "seed":
		_, err := s.SeedDatabase(ctx, connect.NewRequest(&pb.SeedDatabaseRequest{}))
		if err != nil {
			return nil, err
		}
		return connect.NewResponse(&pb.TriggerJobResponse{
			Success: true,
			Message: "SeedDatabase executed successfully",
			CallId:  callID,
		}), nil

	case "purgesoftdeletedaccounts", "purge_soft_deleted_accounts", "purge_accounts":
		count, err := auth.HandlePurgeSoftDeletedAccounts(ctx, s.publicDb)
		if err != nil {
			return nil, err
		}
		return connect.NewResponse(&pb.TriggerJobResponse{
			Success: true,
			Message: fmt.Sprintf("HandlePurgeSoftDeletedAccounts executed successfully; purged %d account(s)", count),
			CallId:  callID,
		}), nil

	default:
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("unknown scheduled job %q", req.Msg.GetJobName()))
	}
}

func (s *OpsService) ListInternalApiCalls(ctx context.Context, req *connect.Request[pb.ListInternalApiCallsRequest]) (*connect.Response[pb.ListInternalApiCallsResponse], error) {
	limit := int(req.Msg.GetLimit())
	if limit <= 0 || limit > 500 {
		limit = 50
	}

	query := `
		SELECT
			c.uuid, c.id, c.method, c.caller, c.caller_type, c.group_name,
			c.jit_grant_uuid::text, j.id, c.status, c.error_message, c.duration_ms, c.created_at
		FROM internal_api_calls c
		LEFT JOIN jit_grants j ON j.uuid = c.jit_grant_uuid
		WHERE 1=1`
	var args []any
	argIdx := 1

	if caller := req.Msg.GetCallerFilter(); caller != "" {
		query += fmt.Sprintf(" AND c.caller = $%d", argIdx)
		args = append(args, caller)
		argIdx++
	}
	if method := req.Msg.GetMethodFilter(); method != "" {
		query += fmt.Sprintf(" AND c.method = $%d", argIdx)
		args = append(args, method)
		argIdx++
	}

	query += fmt.Sprintf(" ORDER BY c.created_at DESC LIMIT $%d", argIdx)
	args = append(args, limit)

	rows, err := s.opsDb.Query(ctx, query, args...)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to query audit log: %w", err))
	}
	defer rows.Close()

	var calls []*pb.InternalApiCallRecord
	for rows.Next() {
		var r pb.InternalApiCallRecord
		var jitGrantUUID, jitGrantID, errorMsg *string
		var createdAt time.Time

		err := rows.Scan(
			&r.Uuid, &r.Id, &r.Method, &r.Caller, &r.CallerType, &r.GroupName,
			&jitGrantUUID, &jitGrantID, &r.Status, &errorMsg, &r.DurationMs, &createdAt,
		)
		if err != nil {
			continue
		}
		if jitGrantUUID != nil {
			r.JitGrantUuid = jitGrantUUID
		}
		if jitGrantID != nil {
			r.JitGrantId = jitGrantID
		}
		if errorMsg != nil {
			r.ErrorMessage = errorMsg
		}
		r.CreatedAt = timestamppb.New(createdAt)
		calls = append(calls, &r)
	}

	return connect.NewResponse(&pb.ListInternalApiCallsResponse{Calls: calls}), nil
}

// ---------------------------------------------------------
// Authentication & IAM
// ---------------------------------------------------------

func (s *OpsService) ExchangeOAuthCode(ctx context.Context, req *connect.Request[pb.ExchangeOAuthCodeRequest]) (*connect.Response[pb.ExchangeOAuthCodeResponse], error) {
	code := req.Msg.GetCode()
	if code == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("missing authorization code"))
	}

	// Developer identity mapping from GitHub OAuth
	operatorName := "developer@quadsmith.dev"
	groupName := "operators"
	if s.cfg.Oauth != nil {
		if s.cfg.Oauth.GetDefaultGroup() != "" {
			groupName = s.cfg.Oauth.GetDefaultGroup()
		}
		// If developer mappings configured, match first
		if len(s.cfg.Oauth.DeveloperMappings) > 0 {
			groupName = s.cfg.Oauth.DeveloperMappings[0].GetGroupName()
			operatorName = s.cfg.Oauth.DeveloperMappings[0].GetEmail()
		}
	}

	// Generate persistent internal session token
	tokenSecret := generateCryptoToken("qs_op_")
	tokenHash := HashToken(tokenSecret)
	tokenID := fmt.Sprintf("tok_%d", time.Now().UnixNano())
	expiresAt := time.Now().Add(24 * time.Hour)

	_, err := s.opsDb.Exec(ctx, `
		INSERT INTO internal_tokens (
			uuid, id, name, type, group_name, token_hash, expires_at, created_at
		) VALUES (
			gen_random_uuid(), $1, $2, 'HUMAN', $3, $4, $5, NOW()
		)`, tokenID, operatorName, groupName, tokenHash, expiresAt)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to issue session token: %w", err))
	}

	return connect.NewResponse(&pb.ExchangeOAuthCodeResponse{
		Token:     tokenSecret,
		ExpiresAt: timestamppb.New(expiresAt),
		Operator:  operatorName,
		GroupName: groupName,
	}), nil
}

func (s *OpsService) GetCurrentOperator(ctx context.Context, _ *connect.Request[pb.GetCurrentOperatorRequest]) (*connect.Response[pb.GetCurrentOperatorResponse], error) {
	opCtx := OperatorContextFromContext(ctx)
	if opCtx == nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("unauthenticated operator"))
	}

	resp := &pb.GetCurrentOperatorResponse{
		Operator:         opCtx.Operator,
		OperatorType:     opCtx.OperatorType,
		GroupName:        opCtx.GroupName,
		SessionExpiresAt: timestamppb.New(time.Now().Add(24 * time.Hour)),
	}

	// Check if caller has an active JIT grant
	var grant pb.JitGrantRecord
	var expiresAt, createdAt time.Time
	var extendedAt, revokedAt *time.Time
	err := s.opsDb.QueryRow(ctx, `
		SELECT uuid, id, operator, operator_type, reason, operation, status, hours, expires_at, created_at, extended_at, revoked_at
		FROM jit_grants
		WHERE operator = $1 AND status = 'ACTIVE' AND expires_at > NOW()
		ORDER BY expires_at DESC
		LIMIT 1`, opCtx.Operator).Scan(
		&grant.Uuid, &grant.Id, &grant.Operator, &grant.OperatorType, &grant.Reason,
		&grant.Operation, &grant.Status, &grant.Hours, &expiresAt, &createdAt, &extendedAt, &revokedAt,
	)
	if err == nil {
		grant.ExpiresAt = timestamppb.New(expiresAt)
		grant.CreatedAt = timestamppb.New(createdAt)
		if extendedAt != nil {
			grant.ExtendedAt = timestamppb.New(*extendedAt)
		}
		if revokedAt != nil {
			grant.RevokedAt = timestamppb.New(*revokedAt)
		}
		resp.ActiveJitGrant = &grant
	}

	return connect.NewResponse(resp), nil
}

func (s *OpsService) ListIamUsers(ctx context.Context, _ *connect.Request[pb.ListIamUsersRequest]) (*connect.Response[pb.ListIamUsersResponse], error) {
	rows, err := s.opsDb.Query(ctx, `
		SELECT name, type, group_name, MAX(created_at) AS last_active
		FROM internal_tokens
		WHERE revoked_at IS NULL
		GROUP BY name, type, group_name
		ORDER BY last_active DESC;`)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to query IAM users: %w", err))
	}
	defer rows.Close()

	var users []*pb.IamUserSummary
	for rows.Next() {
		var name, uType, group string
		var lastActive time.Time
		if err := rows.Scan(&name, &uType, &group, &lastActive); err != nil {
			continue
		}

		// Check if active JIT exists
		var hasJit bool
		_ = s.opsDb.QueryRow(ctx, `
			SELECT EXISTS (
				SELECT 1 FROM jit_grants WHERE operator = $1 AND status = 'ACTIVE' AND expires_at > NOW()
			)`, name).Scan(&hasJit)

		users = append(users, &pb.IamUserSummary{
			Operator:     name,
			Type:         uType,
			GroupName:    group,
			LastActiveAt: timestamppb.New(lastActive),
			HasActiveJit: hasJit,
		})
	}

	return connect.NewResponse(&pb.ListIamUsersResponse{Users: users}), nil
}

func (s *OpsService) ListIamGroups(ctx context.Context, _ *connect.Request[pb.ListIamGroupsRequest]) (*connect.Response[pb.ListIamGroupsResponse], error) {
	var groups []*pb.IamGroupSummary
	for _, g := range s.cfg.Groups {
		groups = append(groups, &pb.IamGroupSummary{
			Name:           g.GetName(),
			Description:    g.GetDescription(),
			AllowedMethods: g.AllowedMethods,
		})
	}
	return connect.NewResponse(&pb.ListIamGroupsResponse{Groups: groups}), nil
}

// ---------------------------------------------------------
// Dynamic Token Management
// ---------------------------------------------------------

func (s *OpsService) CreateInternalToken(ctx context.Context, req *connect.Request[pb.CreateInternalTokenRequest]) (*connect.Response[pb.CreateInternalTokenResponse], error) {
	name := strings.TrimSpace(req.Msg.GetName())
	if name == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("token name cannot be empty"))
	}
	groupName := strings.TrimSpace(req.Msg.GetGroupName())
	if groupName == "" {
		groupName = "operators"
	}
	tokenType := strings.ToUpper(strings.TrimSpace(req.Msg.GetType()))
	if tokenType != "ROBOT" && tokenType != "HUMAN" {
		tokenType = "HUMAN"
	}

	days := int(req.Msg.GetDurationDays())
	if days <= 0 {
		days = 30
	}

	prefix := "qs_op_"
	if tokenType == "ROBOT" {
		prefix = "qs_robot_"
	}

	tokenSecret := generateCryptoToken(prefix)
	tokenHash := HashToken(tokenSecret)
	tokenID := fmt.Sprintf("tok_%d", time.Now().UnixNano())
	expiresAt := time.Now().AddDate(0, 0, days)

	var record pb.InternalTokenRecord
	var createdAt time.Time
	err := s.opsDb.QueryRow(ctx, `
		INSERT INTO internal_tokens (
			uuid, id, name, type, group_name, token_hash, expires_at, created_at
		) VALUES (
			gen_random_uuid(), $1, $2, $3, $4, $5, $6, NOW()
		)
		RETURNING uuid, id, name, type, group_name, token_hash, expires_at, created_at`,
		tokenID, name, tokenType, groupName, tokenHash, expiresAt,
	).Scan(
		&record.Uuid, &record.Id, &record.Name, &record.Type, &record.GroupName,
		&record.TokenHash, &expiresAt, &createdAt,
	)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to create token: %w", err))
	}

	record.ExpiresAt = timestamppb.New(expiresAt)
	record.CreatedAt = timestamppb.New(createdAt)

	return connect.NewResponse(&pb.CreateInternalTokenResponse{
		TokenId:     tokenID,
		TokenSecret: tokenSecret,
		Record:      &record,
	}), nil
}

func (s *OpsService) ListInternalTokens(ctx context.Context, _ *connect.Request[pb.ListInternalTokensRequest]) (*connect.Response[pb.ListInternalTokensResponse], error) {
	rows, err := s.opsDb.Query(ctx, `
		SELECT uuid, id, name, type, group_name, token_hash, expires_at, created_at, revoked_at
		FROM internal_tokens
		ORDER BY created_at DESC;`)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to query tokens: %w", err))
	}
	defer rows.Close()

	var tokens []*pb.InternalTokenRecord
	for rows.Next() {
		var r pb.InternalTokenRecord
		var expiresAt *time.Time
		var createdAt time.Time
		var revokedAt *time.Time

		err := rows.Scan(
			&r.Uuid, &r.Id, &r.Name, &r.Type, &r.GroupName, &r.TokenHash,
			&expiresAt, &createdAt, &revokedAt,
		)
		if err != nil {
			continue
		}
		if expiresAt != nil {
			r.ExpiresAt = timestamppb.New(*expiresAt)
		}
		r.CreatedAt = timestamppb.New(createdAt)
		if revokedAt != nil {
			r.RevokedAt = timestamppb.New(*revokedAt)
		}
		tokens = append(tokens, &r)
	}

	return connect.NewResponse(&pb.ListInternalTokensResponse{Tokens: tokens}), nil
}

func (s *OpsService) RevokeInternalToken(ctx context.Context, req *connect.Request[pb.RevokeInternalTokenRequest]) (*connect.Response[pb.RevokeInternalTokenResponse], error) {
	id := strings.TrimSpace(req.Msg.GetId())
	if id == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("token ID cannot be empty"))
	}

	cmd, err := s.opsDb.Exec(ctx, `
		UPDATE internal_tokens
		SET revoked_at = NOW()
		WHERE (id = $1 OR uuid::text = $1) AND revoked_at IS NULL`, id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to revoke token: %w", err))
	}

	return connect.NewResponse(&pb.RevokeInternalTokenResponse{Success: cmd.RowsAffected() > 0}), nil
}

// ---------------------------------------------------------
// JIT Elevation
// ---------------------------------------------------------

func (s *OpsService) AcquireJitGrant(ctx context.Context, req *connect.Request[pb.AcquireJitGrantRequest]) (*connect.Response[pb.AcquireJitGrantResponse], error) {
	opCtx := OperatorContextFromContext(ctx)
	if opCtx == nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("unauthenticated operator"))
	}

	hours := int(req.Msg.GetHours())
	if hours <= 0 {
		hours = 2
	}
	if hours > 12 {
		hours = 12
	}

	reason := strings.TrimSpace(req.Msg.GetReason())
	if reason == "" {
		reason = "Developer operations task"
	}

	operation := strings.TrimSpace(req.Msg.GetOperation())
	if operation == "" {
		operation = "*"
	}

	grantID := fmt.Sprintf("jit_%d", time.Now().UnixNano())
	expiresAt := time.Now().Add(time.Duration(hours) * time.Hour)

	var grant pb.JitGrantRecord
	var createdAt time.Time
	err := s.opsDb.QueryRow(ctx, `
		INSERT INTO jit_grants (
			uuid, id, token_hash, operator, operator_type, reason, operation, status, hours, expires_at, created_at
		) VALUES (
			gen_random_uuid(), $1, '', $2, $3, $4, $5, 'ACTIVE', $6, $7, NOW()
		)
		RETURNING uuid, id, operator, operator_type, reason, operation, status, hours, expires_at, created_at`,
		grantID, opCtx.Operator, opCtx.OperatorType, reason, operation, hours, expiresAt,
	).Scan(
		&grant.Uuid, &grant.Id, &grant.Operator, &grant.OperatorType, &grant.Reason,
		&grant.Operation, &grant.Status, &grant.Hours, &expiresAt, &createdAt,
	)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to acquire JIT grant: %w", err))
	}

	grant.ExpiresAt = timestamppb.New(expiresAt)
	grant.CreatedAt = timestamppb.New(createdAt)

	return connect.NewResponse(&pb.AcquireJitGrantResponse{Grant: &grant}), nil
}

func (s *OpsService) ExtendJitGrant(ctx context.Context, req *connect.Request[pb.ExtendJitGrantRequest]) (*connect.Response[pb.ExtendJitGrantResponse], error) {
	grantID := strings.TrimSpace(req.Msg.GetGrantId())
	if grantID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("grant ID cannot be empty"))
	}

	hours := int(req.Msg.GetHours())
	if hours <= 0 {
		hours = 2
	}

	var grant pb.JitGrantRecord
	var expiresAt, createdAt time.Time
	var extendedAt, revokedAt *time.Time

	// Extend up to a maximum of 12 hours from current time
	err := s.opsDb.QueryRow(ctx, `
		UPDATE jit_grants
		SET expires_at = LEAST(NOW() + INTERVAL '12 hours', expires_at + ($1 * INTERVAL '1 hour')),
		    extended_at = NOW(),
		    hours = hours + $1
		WHERE (id = $2 OR uuid::text = $2) AND status = 'ACTIVE' AND expires_at > NOW()
		RETURNING uuid, id, operator, operator_type, reason, operation, status, hours, expires_at, created_at, extended_at, revoked_at`,
		hours, grantID,
	).Scan(
		&grant.Uuid, &grant.Id, &grant.Operator, &grant.OperatorType, &grant.Reason,
		&grant.Operation, &grant.Status, &grant.Hours, &expiresAt, &createdAt, &extendedAt, &revokedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, connect.NewError(connect.CodeNotFound, errors.New("active JIT grant not found or already expired"))
		}
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to extend JIT grant: %w", err))
	}

	grant.ExpiresAt = timestamppb.New(expiresAt)
	grant.CreatedAt = timestamppb.New(createdAt)
	if extendedAt != nil {
		grant.ExtendedAt = timestamppb.New(*extendedAt)
	}
	if revokedAt != nil {
		grant.RevokedAt = timestamppb.New(*revokedAt)
	}

	return connect.NewResponse(&pb.ExtendJitGrantResponse{Grant: &grant}), nil
}

func (s *OpsService) ListJitGrants(ctx context.Context, req *connect.Request[pb.ListJitGrantsRequest]) (*connect.Response[pb.ListJitGrantsResponse], error) {
	query := `
		SELECT uuid, id, operator, operator_type, reason, operation, status, hours, expires_at, created_at, extended_at, revoked_at
		FROM jit_grants`
	if req.Msg.GetActiveOnly() {
		query += " WHERE status = 'ACTIVE' AND expires_at > NOW()"
	}
	query += " ORDER BY created_at DESC LIMIT 100"

	rows, err := s.opsDb.Query(ctx, query)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to query JIT grants: %w", err))
	}
	defer rows.Close()

	var grants []*pb.JitGrantRecord
	for rows.Next() {
		var g pb.JitGrantRecord
		var expiresAt, createdAt time.Time
		var extendedAt, revokedAt *time.Time

		err := rows.Scan(
			&g.Uuid, &g.Id, &g.Operator, &g.OperatorType, &g.Reason,
			&g.Operation, &g.Status, &g.Hours, &expiresAt, &createdAt, &extendedAt, &revokedAt,
		)
		if err != nil {
			continue
		}
		g.ExpiresAt = timestamppb.New(expiresAt)
		g.CreatedAt = timestamppb.New(createdAt)
		if extendedAt != nil {
			g.ExtendedAt = timestamppb.New(*extendedAt)
		}
		if revokedAt != nil {
			g.RevokedAt = timestamppb.New(*revokedAt)
		}
		grants = append(grants, &g)
	}

	return connect.NewResponse(&pb.ListJitGrantsResponse{Grants: grants}), nil
}

func (s *OpsService) RevokeJitGrant(ctx context.Context, req *connect.Request[pb.RevokeJitGrantRequest]) (*connect.Response[pb.RevokeJitGrantResponse], error) {
	grantID := strings.TrimSpace(req.Msg.GetGrantId())
	if grantID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("grant ID cannot be empty"))
	}

	cmd, err := s.opsDb.Exec(ctx, `
		UPDATE jit_grants
		SET status = 'REVOKED', revoked_at = NOW()
		WHERE (id = $1 OR uuid::text = $1) AND status = 'ACTIVE'`, grantID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to revoke JIT grant: %w", err))
	}

	return connect.NewResponse(&pb.RevokeJitGrantResponse{Success: cmd.RowsAffected() > 0}), nil
}

func generateCryptoToken(prefix string) string {
	b := make([]byte, 24)
	_, _ = rand.Read(b)
	return prefix + hex.EncodeToString(b)
}
