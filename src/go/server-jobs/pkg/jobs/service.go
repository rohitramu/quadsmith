package jobs

import (
	"context"
	"fmt"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/jackc/pgx/v5/pgxpool"

	"quadsmith/shared/pkg/config"

	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"
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
			'SendUsernameReleaseWarnings'
		  )
		ORDER BY created_at DESC
		LIMIT $2`

	rows, err := s.db.Query(ctx, query, filterProcedure, limit)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to query job history: %w", err))
	}
	defer rows.Close()

	var records []*jobspb.JobRecord
	for rows.Next() {
		var uuid, id, method, caller, callerType, status string
		var durationMs int64
		var errMsg *string
		var createdAt time.Time

		if err := rows.Scan(&uuid, &id, &method, &caller, &callerType, &status, &durationMs, &errMsg, &createdAt); err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to scan job record: %w", err))
		}

		rec := &jobspb.JobRecord{
			Uuid:             uuid,
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
		SELECT uuid::text, id, method, caller, caller_type, status, duration_ms, error_message, created_at
		FROM internal_api_calls
		WHERE (id = $1 OR uuid::text = $1)
		  AND method IN ('CleanupExpiredOtps', 'CleanupExpiredTokens', 'CleanupOrphanedBlobs', 'PurgeSoftDeletedAccounts', 'SendUsernameReleaseWarnings')
		LIMIT 1`

	var uuid, id, method, caller, callerType, status string
	var durationMs int64
	var errMsg *string
	var createdAt time.Time

	err := s.db.QueryRow(ctx, query, jobID).Scan(&uuid, &id, &method, &caller, &callerType, &status, &durationMs, &errMsg, &createdAt)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("job %q not found", jobID))
	}

	rec := &jobspb.JobRecord{
		Uuid:             uuid,
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
		if sched := s.cfg.GetCleanupExpiredOtps(); sched != nil {
			return formatSchedule(sched.GetEnabled(), sched.GetCron(), sched.GetInterval())
		}
	case "CleanupExpiredTokens":
		if sched := s.cfg.GetCleanupExpiredTokens(); sched != nil {
			return formatSchedule(sched.GetEnabled(), sched.GetCron(), sched.GetInterval())
		}
	case "CleanupOrphanedBlobs":
		if sched := s.cfg.GetCleanupOrphanedBlobs(); sched != nil {
			return formatSchedule(sched.GetEnabled(), sched.GetCron(), sched.GetInterval())
		}
	case "PurgeSoftDeletedAccounts":
		if sched := s.cfg.GetPurgeSoftDeletedAccounts(); sched != nil {
			return formatSchedule(sched.GetEnabled(), sched.GetCron(), sched.GetInterval())
		}
	case "SendUsernameReleaseWarnings":
		if sched := s.cfg.GetSendUsernameReleaseWarnings(); sched != nil {
			return formatSchedule(sched.GetEnabled(), sched.GetCron(), sched.GetInterval())
		}
	}
	return nil
}

func formatSchedule(enabled bool, cron string, interval *durationpb.Duration) *string {
	if !enabled {
		disabled := "disabled"
		return &disabled
	}
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
	switch clean {
	case "cleanup_expired_otps":
		return "CleanupExpiredOtps"
	case "cleanup_expired_tokens":
		return "CleanupExpiredTokens"
	case "cleanup_orphaned_blobs":
		return "CleanupOrphanedBlobs"
	case "purge_soft_deleted_accounts":
		return "PurgeSoftDeletedAccounts"
	case "send_username_release_warnings":
		return "SendUsernameReleaseWarnings"
	default:
		return ""
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
	default:
		return strings.ToLower(procedure)
	}
}
