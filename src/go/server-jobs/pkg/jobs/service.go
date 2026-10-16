package jobs

import (
	"context"
	"fmt"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/jackc/pgx/v5/pgxpool"

	"quadsmith/shared/pkg/config"

	durationpb "google.golang.org/protobuf/types/known/durationpb"
	timestamppb "google.golang.org/protobuf/types/known/timestamppb"
	configpb "quadsmith/shared/gen/quadsmith/config"
	jobspb "quadsmith/shared/gen/quadsmith/jobs"
	"quadsmith/shared/gen/quadsmith/jobs/jobsconnect"
)

type Service struct {
	jobsconnect.UnimplementedJobsServiceHandler
	db  *pgxpool.Pool
	cfg *configpb.JobServerConfig
}

func NewService(db *pgxpool.Pool, cfg *configpb.JobServerConfig) *Service {
	return &Service{
		db:  db,
		cfg: cfg,
	}
}

func (s *Service) ListJobs(ctx context.Context, req *connect.Request[jobspb.ListJobsRequest]) (*connect.Response[jobspb.ListJobsResponse], error) {
	limit := int(req.Msg.GetLimit())
	if limit <= 0 || limit > 500 {
		limit = 50
	}

	filterProcedure := ""
	if task := strings.TrimSpace(req.Msg.GetTaskName()); task != "" {
		filterProcedure = taskToProcedure(task)
	}

	query := `
		SELECT id, method, caller, caller_type, status, duration_ms, error_message, created_at
		FROM internal_api_calls
		WHERE ($1 = '' OR method = $1 OR method ILIKE $2)
		  AND method IN (
			'CleanupExpiredOtps',
			'CleanupExpiredTokens',
			'CleanupOrphanedBlobs',
			'PurgeSoftDeletedAccounts',
			'SendUsernameReleaseWarnings',
			'RunDatabaseMaintenance',
			'SeedDatabase'
		  )
		ORDER BY created_at DESC
		LIMIT $3`

	likePattern := ""
	if filterProcedure != "" {
		likePattern = "%" + filterProcedure + "%"
	}

	rows, err := s.db.Query(ctx, query, filterProcedure, likePattern, limit)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to query job history: %w", err))
	}
	defer rows.Close()

	var records []*jobspb.JobRecord
	for rows.Next() {
		var id, method, caller, callerType, status string
		var durationMs int64
		var errMsg *string
		var createdAt time.Time

		if err := rows.Scan(&id, &method, &caller, &callerType, &status, &durationMs, &errMsg, &createdAt); err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to scan job record: %w", err))
		}

		rec := &jobspb.JobRecord{
			JobId:            id,
			TaskName:         procedureToTask(method),
			Procedure:        method,
			Caller:           caller,
			CallerType:       callerType,
			Status:           status,
			DurationMs:       durationMs,
			CreatedAt:        timestamppb.New(createdAt),
			ScheduleInterval: s.getScheduleForProcedure(method),
		}
		if errMsg != nil {
			rec.ErrorMessage = errMsg
		}
		records = append(records, rec)
	}

	if err := rows.Err(); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("error reading job records: %w", err))
	}

	return connect.NewResponse(&jobspb.ListJobsResponse{Jobs: records}), nil
}

func (s *Service) GetJob(ctx context.Context, req *connect.Request[jobspb.GetJobRequest]) (*connect.Response[jobspb.GetJobResponse], error) {
	jobID := strings.TrimSpace(req.Msg.GetJobId())
	if jobID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("job_id cannot be empty"))
	}

	query := `
		SELECT id, method, caller, caller_type, status, duration_ms, error_message, created_at
		FROM internal_api_calls
		WHERE (id = $1 OR uuid::text = $1)
		  AND method IN ('CleanupExpiredOtps', 'CleanupExpiredTokens', 'CleanupOrphanedBlobs', 'PurgeSoftDeletedAccounts', 'SendUsernameReleaseWarnings', 'RunDatabaseMaintenance', 'SeedDatabase')
		LIMIT 1`

	var id, method, caller, callerType, status string
	var durationMs int64
	var errMsg *string
	var createdAt time.Time

	err := s.db.QueryRow(ctx, query, jobID).Scan(&id, &method, &caller, &callerType, &status, &durationMs, &errMsg, &createdAt)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("job %q not found", jobID))
	}

	rec := &jobspb.JobRecord{
		JobId:            id,
		TaskName:         procedureToTask(method),
		Procedure:        method,
		Caller:           caller,
		CallerType:       callerType,
		Status:           status,
		DurationMs:       durationMs,
		CreatedAt:        timestamppb.New(createdAt),
		ScheduleInterval: s.getScheduleForProcedure(method),
	}
	if errMsg != nil {
		rec.ErrorMessage = errMsg
	}

	return connect.NewResponse(&jobspb.GetJobResponse{Job: rec}), nil
}

func (s *Service) getScheduleForProcedure(procedure string) *string {
	switch procedure {
	case "CleanupExpiredOtps":
		if s := s.cfg.GetCleanupExpiredOtps(); s != nil {
			return formatSchedule(s.GetCron(), s.GetInterval())
		}
	case "CleanupExpiredTokens":
		if s := s.cfg.GetCleanupExpiredTokens(); s != nil {
			return formatSchedule(s.GetCron(), s.GetInterval())
		}
	case "CleanupOrphanedBlobs":
		if s := s.cfg.GetCleanupOrphanedBlobs(); s != nil {
			return formatSchedule(s.GetCron(), s.GetInterval())
		}
	case "PurgeSoftDeletedAccounts":
		if s := s.cfg.GetPurgeSoftDeletedAccounts(); s != nil {
			return formatSchedule(s.GetCron(), s.GetInterval())
		}
	case "SendUsernameReleaseWarnings":
		if s := s.cfg.GetSendUsernameReleaseWarnings(); s != nil {
			return formatSchedule(s.GetCron(), s.GetInterval())
		}
	case "RunDatabaseMaintenance":
		if s := s.cfg.GetDatabaseMaintenance(); s != nil {
			return formatSchedule(s.GetCron(), s.GetInterval())
		}
	}
	return nil
}

func formatSchedule(cron string, interval *durationpb.Duration) *string {
	if cron != "" {
		return &cron
	}
	if interval != nil {
		dur := interval.AsDuration()
		if c, err := config.DurationToCron(dur); err == nil {
			res := fmt.Sprintf("%v (%s)", dur, c)
			return &res
		}
		res := dur.String()
		return &res
	}
	return nil
}

func taskToProcedure(task string) string {
	clean := strings.ToLower(strings.TrimSpace(task))
	clean = strings.TrimPrefix(clean, "auth.")
	clean = strings.TrimPrefix(clean, "blob.")
	switch clean {
	case "cleanup_expired_otps", "cleanupexpiredotps":
		return "CleanupExpiredOtps"
	case "cleanup_expired_tokens", "cleanupexpiredtokens":
		return "CleanupExpiredTokens"
	case "cleanup_orphaned_blobs", "cleanuporphanedblobs":
		return "CleanupOrphanedBlobs"
	case "purge_soft_deleted_accounts", "purgesoftdeletedaccounts", "purge_accounts":
		return "PurgeSoftDeletedAccounts"
	case "send_username_release_warnings", "sendusernamereleasewarnings", "warn_usernames":
		return "SendUsernameReleaseWarnings"
	case "run_database_maintenance", "rundatabasemaintenance", "vacuum":
		return "RunDatabaseMaintenance"
	case "seed_database", "seeddatabase", "seed":
		return "SeedDatabase"
	default:
		return task
	}
}

func procedureToTask(procedure string) string {
	switch procedure {
	case "CleanupExpiredOtps":
		return "cleanup_expired_otps"
	case "CleanupExpiredTokens":
		return "cleanup_expired_tokens"
	case "CleanupOrphanedBlobs":
		return "cleanup_orphaned_blobs"
	case "PurgeSoftDeletedAccounts":
		return "purge_soft_deleted_accounts"
	case "SendUsernameReleaseWarnings":
		return "send_username_release_warnings"
	case "RunDatabaseMaintenance":
		return "run_database_maintenance"
	case "SeedDatabase":
		return "seed_database"
	default:
		return strings.ToLower(procedure)
	}
}
