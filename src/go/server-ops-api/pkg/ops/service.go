package ops

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"runtime"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"quadsmith/shared/pkg/auth"
	"quadsmith/shared/pkg/blob"
	"quadsmith/shared/pkg/opsauth"
	"quadsmith/shared/pkg/telemetry"

	"google.golang.org/protobuf/types/known/timestamppb"
	configpb "quadsmith/shared/gen/quadsmith/config"
	pb "quadsmith/shared/gen/quadsmith/ops"
	"quadsmith/shared/gen/quadsmith/ops/opsconnect"
)

type OpsService struct {
	opsconnect.UnimplementedOpsServiceHandler
	publicDb  *pgxpool.Pool
	opsDb     *pgxpool.Pool
	cfg       *configpb.OpsApiServerConfig
	telemetry *telemetry.Registry
	startTime time.Time
}

func NewOpsService(publicDb, opsDb *pgxpool.Pool, cfg *configpb.OpsApiServerConfig, reg ...*telemetry.Registry) *OpsService {
	var r *telemetry.Registry
	if len(reg) > 0 && reg[0] != nil {
		r = reg[0]
	} else {
		r = telemetry.NewRegistry("server-ops-api", cfg.GetEnvironment(), true)
	}
	return &OpsService{
		publicDb:  publicDb,
		opsDb:     opsDb,
		cfg:       cfg,
		telemetry: r,
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

func (s *OpsService) GetDatabaseStats(ctx context.Context, req *connect.Request[pb.GetDatabaseStatsRequest]) (*connect.Response[pb.GetDatabaseStatsResponse], error) {
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

	filtered, err := filterAndSortTableStats(tables, req.Msg.GetFilter(), req.Msg.GetSort())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	return connect.NewResponse(&pb.GetDatabaseStatsResponse{
		DatabaseName:   dbName,
		TotalSizeBytes: totalSize,
		Tables:         filtered,
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

func (s *OpsService) PurgeSoftDeletedAccounts(ctx context.Context, _ *connect.Request[pb.PurgeSoftDeletedAccountsRequest]) (*connect.Response[pb.PurgeSoftDeletedAccountsResponse], error) {
	count, err := auth.HandlePurgeSoftDeletedAccounts(ctx, s.publicDb)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&pb.PurgeSoftDeletedAccountsResponse{PurgedCount: count}), nil
}

func (s *OpsService) SendUsernameReleaseWarnings(ctx context.Context, _ *connect.Request[pb.SendUsernameReleaseWarningsRequest]) (*connect.Response[pb.SendUsernameReleaseWarningsResponse], error) {
	count, err := auth.HandleUsernameReleaseWarning(ctx, s.publicDb)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&pb.SendUsernameReleaseWarningsResponse{WarningCount: count}), nil
}

func (s *OpsService) SetUserRole(ctx context.Context, req *connect.Request[pb.SetUserRoleRequest]) (*connect.Response[pb.SetUserRoleResponse], error) {
	opCtx := opsauth.OperatorContextFromContext(ctx)
	if opCtx == nil || opCtx.GroupName == "unauthenticated" || opCtx.OperatorType == "UNKNOWN" {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("authentication required"))
	}
	if opCtx.OperatorType != "HUMAN" {
		return nil, connect.NewError(connect.CodePermissionDenied, errors.New("sensitive procedure requires a HUMAN operator"))
	}

	userID := req.Msg.GetUserId()
	if strings.TrimSpace(userID) == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("user identifier is required"))
	}

	targetRole := strings.ToUpper(strings.TrimSpace(req.Msg.GetRole()))
	if targetRole != "USER" && targetRole != "ADMIN" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid role %q: role must be USER or ADMIN", req.Msg.GetRole()))
	}

	tx, err := s.publicDb.Begin(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to begin transaction: %w", err))
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var userUUID, username, previousRole string
	var deletedAt *time.Time
	err = tx.QueryRow(ctx, `
		SELECT uuid::text, id, role, deleted_at
		FROM users
		WHERE id = $1 OR uuid::text = $1
		FOR UPDATE`, userID).Scan(&userUUID, &username, &previousRole, &deletedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("user %q not found", userID))
		}
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to query user: %w", err))
	}

	if deletedAt != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("cannot change role of a soft-deleted user"))
	}

	if targetRole == "ADMIN" {
		var hasActiveBan bool
		err = tx.QueryRow(ctx, `
			SELECT EXISTS (
				SELECT 1 FROM bans
				WHERE user_uuid = $1::uuid
				  AND revoked_at IS NULL
				  AND (expires_at IS NULL OR expires_at > NOW())
			)`, userUUID).Scan(&hasActiveBan)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to check active ban: %w", err))
		}
		if hasActiveBan {
			return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("cannot promote a banned user to ADMIN; revoke the active ban first"))
		}
	}

	_, err = tx.Exec(ctx, `
		UPDATE users
		SET role = $1, updated_at = NOW()
		WHERE uuid = $2::uuid`, targetRole, userUUID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to update user role: %w", err))
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to commit transaction: %w", err))
	}

	return connect.NewResponse(&pb.SetUserRoleResponse{
		UserUuid:     userUUID,
		Username:     username,
		PreviousRole: previousRole,
		Role:         targetRole,
	}), nil
}

// ---------------------------------------------------------
// Tasks, Jobs & Audit
// ---------------------------------------------------------

var taskCatalog = []*pb.TaskDef{
	{
		Name:            "cleanup_expired_otps",
		Procedure:       "CleanupExpiredOtps",
		Description:     "Purges used and expired OTPs",
		DefaultInterval: "6h",
	},
	{
		Name:            "cleanup_expired_tokens",
		Procedure:       "CleanupExpiredTokens",
		Description:     "Purges expired refresh tokens",
		DefaultInterval: "12h",
	},
	{
		Name:            "cleanup_orphaned_blobs",
		Procedure:       "CleanupOrphanedBlobs",
		Description:     "Deletes unreferenced blob objects older than 10 minutes",
		DefaultInterval: "10m",
	},
	{
		Name:            "purge_soft_deleted_accounts",
		Procedure:       "PurgeSoftDeletedAccounts",
		Description:     "Anonymizes and permanently purges soft-deleted user accounts past their 30-day grace period",
		DefaultInterval: "8h",
	},
	{
		Name:            "send_username_release_warnings",
		Procedure:       "SendUsernameReleaseWarnings",
		Description:     "Sends release warning notifications for archived usernames approaching their 30-day hold expiration",
		DefaultInterval: "24h",
	},
	{
		Name:            "set_user_role",
		Procedure:       "SetUserRole",
		Description:     "Updates a user account role ('USER' or 'ADMIN')",
		DefaultInterval: "manual",
	},
}

func (s *OpsService) ListTasks(ctx context.Context, req *connect.Request[pb.ListTasksRequest]) (*connect.Response[pb.ListTasksResponse], error) {
	tasks := make([]*pb.TaskDef, len(taskCatalog))
	for i, t := range taskCatalog {
		tasks[i] = &pb.TaskDef{
			Name:            t.Name,
			Procedure:       t.Procedure,
			Description:     t.Description,
			DefaultInterval: t.DefaultInterval,
		}
	}
	filtered, err := filterAndSortTasks(tasks, req.Msg.GetFilter(), req.Msg.GetSort())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return connect.NewResponse(&pb.ListTasksResponse{Tasks: filtered}), nil
}

func (s *OpsService) GetTask(ctx context.Context, req *connect.Request[pb.GetTaskRequest]) (*connect.Response[pb.GetTaskResponse], error) {
	name := strings.ToLower(strings.TrimSpace(req.Msg.GetTaskName()))

	for _, t := range taskCatalog {
		if strings.ToLower(t.Name) == name || strings.ToLower(t.Procedure) == name {
			return connect.NewResponse(&pb.GetTaskResponse{
				Task: &pb.TaskDef{
					Name:            t.Name,
					Procedure:       t.Procedure,
					Description:     t.Description,
					DefaultInterval: t.DefaultInterval,
				},
			}), nil
		}
	}
	return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("task %q not found", req.Msg.GetTaskName()))
}

func (s *OpsService) ListJobs(ctx context.Context, req *connect.Request[pb.ListJobsRequest]) (*connect.Response[pb.ListJobsResponse], error) {
	if _, err := filterAndSortJobs(nil, req.Msg.GetFilter(), req.Msg.GetSort()); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	limit := int(req.Msg.GetLimit())
	if limit <= 0 || limit > 500 {
		limit = 50
	}

	filterProcedure := ""
	if task := strings.TrimSpace(req.Msg.GetTaskName()); task != "" {
		filterProcedure = taskToProcedure(task)
		if filterProcedure == "" {
			return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("unknown task %q", task))
		}
	}

	query := `
		SELECT uuid::text, id, method, caller, caller_type, status, duration_ms, error_message, created_at
		FROM internal_api_calls
		WHERE ($1 = '' OR method = $1)
		  AND method IN (
			'CleanupExpiredOtps',
			'CleanupExpiredTokens',
			'CleanupOrphanedBlobs',
			'PurgeSoftDeletedAccounts',
			'SendUsernameReleaseWarnings',
			'SetUserRole'
		  )
		ORDER BY created_at DESC
		LIMIT $2`

	rows, err := s.opsDb.Query(ctx, query, filterProcedure, limit)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to query job history: %w", err))
	}
	defer rows.Close()

	var records []*pb.JobRecord
	for rows.Next() {
		var uuid, id, method, caller, callerType, status string
		var durationMs int64
		var errMsg *string
		var createdAt time.Time

		if err := rows.Scan(&uuid, &id, &method, &caller, &callerType, &status, &durationMs, &errMsg, &createdAt); err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to scan job record: %w", err))
		}

		rec := &pb.JobRecord{
			Uuid:       uuid,
			JobId:      id,
			TaskName:   procedureToTask(method),
			Procedure:  method,
			Caller:     caller,
			CallerType: callerType,
			Status:     status,
			DurationMs: durationMs,
			CreatedAt:  timestamppb.New(createdAt),
			Region:     telemetry.CurrentRegion(),
			TraceId:    uuid,
		}
		if errMsg != nil {
			rec.ErrorMessage = errMsg
		}
		records = append(records, rec)
	}

	if err := rows.Err(); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("error reading job records: %w", err))
	}

	filtered, err := filterAndSortJobs(records, req.Msg.GetFilter(), req.Msg.GetSort())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	return connect.NewResponse(&pb.ListJobsResponse{Jobs: filtered}), nil
}

func (s *OpsService) GetJob(ctx context.Context, req *connect.Request[pb.GetJobRequest]) (*connect.Response[pb.GetJobResponse], error) {
	jobID := strings.TrimSpace(req.Msg.GetJobId())
	if jobID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("job_id cannot be empty"))
	}

	query := `
		SELECT uuid::text, id, method, caller, caller_type, status, duration_ms, error_message, created_at
		FROM internal_api_calls
		WHERE (id = $1 OR uuid::text = $1)
		  AND method IN ('CleanupExpiredOtps', 'CleanupExpiredTokens', 'CleanupOrphanedBlobs', 'PurgeSoftDeletedAccounts', 'SendUsernameReleaseWarnings', 'SetUserRole')
		LIMIT 1`

	var uuid, id, method, caller, callerType, status string
	var durationMs int64
	var errMsg *string
	var createdAt time.Time

	err := s.opsDb.QueryRow(ctx, query, jobID).Scan(&uuid, &id, &method, &caller, &callerType, &status, &durationMs, &errMsg, &createdAt)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("job %q not found", jobID))
	}

	rec := &pb.JobRecord{
		Uuid:       uuid,
		JobId:      id,
		TaskName:   procedureToTask(method),
		Procedure:  method,
		Caller:     caller,
		CallerType: callerType,
		Status:     status,
		DurationMs: durationMs,
		CreatedAt:  timestamppb.New(createdAt),
		Region:     telemetry.CurrentRegion(),
		TraceId:    uuid,
	}
	if errMsg != nil {
		rec.ErrorMessage = errMsg
	}

	return connect.NewResponse(&pb.GetJobResponse{Job: rec}), nil
}

func taskToProcedure(task string) string {
	clean := strings.ToLower(strings.TrimSpace(task))
	for _, t := range taskCatalog {
		if strings.ToLower(t.Name) == clean || strings.ToLower(t.Procedure) == clean {
			return t.Procedure
		}
	}
	return ""
}

func procedureToTask(procedure string) string {
	for _, t := range taskCatalog {
		if t.Procedure == procedure {
			return t.Name
		}
	}
	return strings.ToLower(procedure)
}

func (s *OpsService) ListInternalApiCalls(ctx context.Context, req *connect.Request[pb.ListInternalApiCallsRequest]) (*connect.Response[pb.ListInternalApiCallsResponse], error) {
	if _, err := filterAndSortInternalApiCalls(nil, req.Msg.GetFilter(), req.Msg.GetSort()); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

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
		r.Region = telemetry.CurrentRegion()
		r.TraceId = r.Uuid
		r.CreatedAt = timestamppb.New(createdAt)
		calls = append(calls, &r)
	}

	filtered, err := filterAndSortInternalApiCalls(calls, req.Msg.GetFilter(), req.Msg.GetSort())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	return connect.NewResponse(&pb.ListInternalApiCallsResponse{Calls: filtered}), nil
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
	tokenHash := opsauth.HashToken(tokenSecret)
	tokenUUID := auth.NewUUIDv7()
	tokenID := fmt.Sprintf("tok_%d", time.Now().UnixNano())
	expiresAt := time.Now().Add(24 * time.Hour)

	_, err := s.opsDb.Exec(ctx, `
		INSERT INTO internal_tokens (
			uuid, id, name, type, group_name, token_hash, expires_at, created_at
		) VALUES (
			$1, $2, $3, 'HUMAN', $4, $5, $6, NOW()
		)`, tokenUUID, tokenID, operatorName, groupName, tokenHash, expiresAt)
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
	opCtx := opsauth.OperatorContextFromContext(ctx)
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

func (s *OpsService) ListIamUsers(ctx context.Context, req *connect.Request[pb.ListIamUsersRequest]) (*connect.Response[pb.ListIamUsersResponse], error) {
	if _, err := filterAndSortIamUsers(nil, req.Msg.GetFilter(), req.Msg.GetSort()); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

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

	filtered, err := filterAndSortIamUsers(users, req.Msg.GetFilter(), req.Msg.GetSort())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	return connect.NewResponse(&pb.ListIamUsersResponse{Users: filtered}), nil
}

func (s *OpsService) ListIamGroups(ctx context.Context, req *connect.Request[pb.ListIamGroupsRequest]) (*connect.Response[pb.ListIamGroupsResponse], error) {
	var groups []*pb.IamGroupSummary
	if s.cfg != nil {
		for _, g := range s.cfg.Groups {
			groups = append(groups, &pb.IamGroupSummary{
				Name:           g.GetName(),
				Description:    g.GetDescription(),
				AllowedMethods: g.AllowedMethods,
			})
		}
	}
	filtered, err := filterAndSortIamGroups(groups, req.Msg.GetFilter(), req.Msg.GetSort())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return connect.NewResponse(&pb.ListIamGroupsResponse{Groups: filtered}), nil
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
	tokenHash := opsauth.HashToken(tokenSecret)
	tokenUUID := auth.NewUUIDv7()
	tokenID := fmt.Sprintf("tok_%d", time.Now().UnixNano())
	expiresAt := time.Now().AddDate(0, 0, days)

	var record pb.InternalTokenRecord
	var createdAt time.Time
	err := s.opsDb.QueryRow(ctx, `
		INSERT INTO internal_tokens (
			uuid, id, name, type, group_name, token_hash, expires_at, created_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, NOW()
		)
		RETURNING uuid, id, name, type, group_name, token_hash, expires_at, created_at`,
		tokenUUID, tokenID, name, tokenType, groupName, tokenHash, expiresAt,
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

func (s *OpsService) ListInternalTokens(ctx context.Context, req *connect.Request[pb.ListInternalTokensRequest]) (*connect.Response[pb.ListInternalTokensResponse], error) {
	if _, err := filterAndSortInternalTokens(nil, req.Msg.GetFilter(), req.Msg.GetSort()); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

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

	filtered, err := filterAndSortInternalTokens(tokens, req.Msg.GetFilter(), req.Msg.GetSort())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	return connect.NewResponse(&pb.ListInternalTokensResponse{Tokens: filtered}), nil
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
	opCtx := opsauth.OperatorContextFromContext(ctx)
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

	grantUUID := auth.NewUUIDv7()
	grantID := fmt.Sprintf("jit_%d", time.Now().UnixNano())
	expiresAt := time.Now().Add(time.Duration(hours) * time.Hour)

	var grant pb.JitGrantRecord
	var createdAt time.Time
	err := s.opsDb.QueryRow(ctx, `
		INSERT INTO jit_grants (
			uuid, id, token_hash, operator, operator_type, reason, operation, status, hours, expires_at, created_at
		) VALUES (
			$1, $2, '', $3, $4, $5, $6, 'ACTIVE', $7, $8, NOW()
		)
		RETURNING uuid, id, operator, operator_type, reason, operation, status, hours, expires_at, created_at`,
		grantUUID, grantID, opCtx.Operator, opCtx.OperatorType, reason, operation, hours, expiresAt,
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
	if _, err := filterAndSortJitGrants(nil, req.Msg.GetFilter(), req.Msg.GetSort()); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

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

	filtered, err := filterAndSortJitGrants(grants, req.Msg.GetFilter(), req.Msg.GetSort())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	return connect.NewResponse(&pb.ListJitGrantsResponse{Grants: filtered}), nil
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
