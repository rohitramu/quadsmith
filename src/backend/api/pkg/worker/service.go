package worker

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	connect "connectrpc.com/connect"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/protobuf/types/known/timestamppb"

	pb "quadsmith/api/gen/quadsmith"
	"quadsmith/api/gen/quadsmith/quadsmithconnect"
)

var _ quadsmithconnect.WorkerServiceHandler = (*WorkerService)(nil)

// WorkerService implements the ConnectRPC quadsmith.WorkerService.
type WorkerService struct {
	db       *pgxpool.Pool
	registry *Registry
	pool     *WorkerPool
}

// NewWorkerService creates a new WorkerService instance.
func NewWorkerService(db *pgxpool.Pool, registry *Registry, pool *WorkerPool) *WorkerService {
	return &WorkerService{
		db:       db,
		registry: registry,
		pool:     pool,
	}
}

// ListJobs returns a list of recent background jobs with optional status filter.
func (s *WorkerService) ListJobs(ctx context.Context, req *connect.Request[pb.ListJobsRequest]) (*connect.Response[pb.ListJobsResponse], error) {
	limit := req.Msg.GetLimit()
	if limit <= 0 {
		limit = 50
	} else if limit > 200 {
		limit = 200
	}

	statusFilter := jobStatusToString(req.Msg.GetStatusFilter())

	query := `
		SELECT uuid, job_type, payload, status, unique_key, max_attempts, scheduled_at, created_at, updated_at
		FROM background_jobs
		WHERE ($1 = '' OR status = $1)
		ORDER BY created_at DESC
		LIMIT $2;`

	rows, err := s.db.Query(ctx, query, statusFilter, limit)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to query jobs: %w", err))
	}
	defer rows.Close()

	var jobs []*pb.BackgroundJob
	for rows.Next() {
		var (
			uuidStr     string
			jobType     string
			payloadStr  string
			statusStr   string
			uniqueKey   sql.NullString
			maxAttempts int32
			scheduledAt time.Time
			createdAt   time.Time
			updatedAt   time.Time
		)

		if err := rows.Scan(
			&uuidStr,
			&jobType,
			&payloadStr,
			&statusStr,
			&uniqueKey,
			&maxAttempts,
			&scheduledAt,
			&createdAt,
			&updatedAt,
		); err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to scan job: %w", err))
		}

		job := &pb.BackgroundJob{
			Uuid:        uuidStr,
			JobType:     jobType,
			PayloadJson: payloadStr,
			Status:      stringToJobStatus(statusStr),
			MaxAttempts: maxAttempts,
			ScheduledAt: timestamppb.New(scheduledAt),
			CreatedAt:   timestamppb.New(createdAt),
			UpdatedAt:   timestamppb.New(updatedAt),
		}
		if uniqueKey.Valid {
			job.UniqueKey = uniqueKey.String
		}

		jobs = append(jobs, job)
	}

	return connect.NewResponse(&pb.ListJobsResponse{Jobs: jobs}), nil
}

// GetJob returns detailed job metadata along with all recorded execution attempts.
func (s *WorkerService) GetJob(ctx context.Context, req *connect.Request[pb.GetJobRequest]) (*connect.Response[pb.GetJobResponse], error) {
	jobUUID := req.Msg.GetJobUuid()
	if jobUUID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("job_uuid is required"))
	}

	jobQuery := `
		SELECT uuid, job_type, payload, status, unique_key, max_attempts, scheduled_at, created_at, updated_at
		FROM background_jobs
		WHERE uuid = $1;`

	var (
		uuidStr     string
		jobType     string
		payloadStr  string
		statusStr   string
		uniqueKey   sql.NullString
		maxAttempts int32
		scheduledAt time.Time
		createdAt   time.Time
		updatedAt   time.Time
	)

	err := s.db.QueryRow(ctx, jobQuery, jobUUID).Scan(
		&uuidStr,
		&jobType,
		&payloadStr,
		&statusStr,
		&uniqueKey,
		&maxAttempts,
		&scheduledAt,
		&createdAt,
		&updatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("job %s not found", jobUUID))
		}
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to get job: %w", err))
	}

	job := &pb.BackgroundJob{
		Uuid:        uuidStr,
		JobType:     jobType,
		PayloadJson: payloadStr,
		Status:      stringToJobStatus(statusStr),
		MaxAttempts: maxAttempts,
		ScheduledAt: timestamppb.New(scheduledAt),
		CreatedAt:   timestamppb.New(createdAt),
		UpdatedAt:   timestamppb.New(updatedAt),
	}
	if uniqueKey.Valid {
		job.UniqueKey = uniqueKey.String
	}

	// Fetch attempts
	attemptsQuery := `
		SELECT uuid, job_uuid, attempt_number, worker_id, started_at, completed_at, status, error_message
		FROM background_job_attempts
		WHERE job_uuid = $1
		ORDER BY attempt_number ASC;`

	attRows, err := s.db.Query(ctx, attemptsQuery, jobUUID)
	if err == nil {
		defer attRows.Close()
		for attRows.Next() {
			var (
				attUUID     string
				jUUID       string
				attNum      int32
				workerID    string
				startedAt   time.Time
				completedAt sql.NullTime
				attStatus   string
				errMsg      sql.NullString
			)

			if err := attRows.Scan(
				&attUUID,
				&jUUID,
				&attNum,
				&workerID,
				&startedAt,
				&completedAt,
				&attStatus,
				&errMsg,
			); err == nil {
				attempt := &pb.BackgroundJobAttempt{
					Uuid:          attUUID,
					JobUuid:       jUUID,
					AttemptNumber: attNum,
					WorkerId:      workerID,
					StartedAt:     timestamppb.New(startedAt),
					Status:        stringToAttemptStatus(attStatus),
				}
				if completedAt.Valid {
					attempt.CompletedAt = timestamppb.New(completedAt.Time)
				}
				if errMsg.Valid {
					attempt.ErrorMessage = errMsg.String
				}
				job.Attempts = append(job.Attempts, attempt)
			}
		}
	}

	return connect.NewResponse(&pb.GetJobResponse{Job: job}), nil
}

// TriggerJob manually dispatches or enqueues a job for execution.
func (s *WorkerService) TriggerJob(ctx context.Context, req *connect.Request[pb.TriggerJobRequest]) (*connect.Response[pb.TriggerJobResponse], error) {
	jobType := req.Msg.GetJobType()
	if jobType == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("job_type is required"))
	}

	payloadJSON := req.Msg.GetPayloadJson()
	if strings.TrimSpace(payloadJSON) == "" {
		payloadJSON = "{}"
	}

	var opts []EnqueueOption
	if key := req.Msg.GetUniqueKey(); key != "" {
		opts = append(opts, WithUniqueKey(key))
	}

	jobUUID, err := Enqueue(ctx, s.db, jobType, []byte(payloadJSON), opts...)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to trigger job: %w", err))
	}

	return connect.NewResponse(&pb.TriggerJobResponse{JobUuid: jobUUID}), nil
}

// RetryJob resets a failed job back to pending status for immediate re-execution.
func (s *WorkerService) RetryJob(ctx context.Context, req *connect.Request[pb.RetryJobRequest]) (*connect.Response[pb.RetryJobResponse], error) {
	jobUUID := req.Msg.GetJobUuid()
	if jobUUID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("job_uuid is required"))
	}

	cmd, err := s.db.Exec(ctx, `
		UPDATE background_jobs
		SET status = 'pending', scheduled_at = NOW(), updated_at = NOW()
		WHERE uuid = $1;`, jobUUID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to retry job: %w", err))
	}

	if cmd.RowsAffected() == 0 {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("job %s not found", jobUUID))
	}

	return connect.NewResponse(&pb.RetryJobResponse{Success: true}), nil
}

// GetWorkerStatus returns health metrics and queue depth statistics.
func (s *WorkerService) GetWorkerStatus(ctx context.Context, _ *connect.Request[pb.GetWorkerStatusRequest]) (*connect.Response[pb.GetWorkerStatusResponse], error) {
	resp := &pb.GetWorkerStatusResponse{
		IsHealthy:       true,
		RegisteredTasks: s.registry.TaskTypes(),
	}

	query := `
		SELECT status, count(*) 
		FROM background_jobs 
		GROUP BY status;`

	rows, err := s.db.Query(ctx, query)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var status string
			var count int32
			if err := rows.Scan(&status, &count); err == nil {
				switch status {
				case "pending":
					resp.PendingJobs = count
				case "running":
					resp.RunningJobs = count
				case "failed":
					resp.FailedJobs = count
				}
			}
		}
	}

	return connect.NewResponse(resp), nil
}

func jobStatusToString(s pb.JobStatus) string {
	switch s {
	case pb.JobStatus_JOB_STATUS_PENDING:
		return "pending"
	case pb.JobStatus_JOB_STATUS_RUNNING:
		return "running"
	case pb.JobStatus_JOB_STATUS_COMPLETED:
		return "completed"
	case pb.JobStatus_JOB_STATUS_FAILED:
		return "failed"
	case pb.JobStatus_JOB_STATUS_CANCELLED:
		return "cancelled"
	default:
		return ""
	}
}

func stringToJobStatus(s string) pb.JobStatus {
	switch s {
	case "pending":
		return pb.JobStatus_JOB_STATUS_PENDING
	case "running":
		return pb.JobStatus_JOB_STATUS_RUNNING
	case "completed":
		return pb.JobStatus_JOB_STATUS_COMPLETED
	case "failed":
		return pb.JobStatus_JOB_STATUS_FAILED
	case "cancelled":
		return pb.JobStatus_JOB_STATUS_CANCELLED
	default:
		return pb.JobStatus_JOB_STATUS_UNSPECIFIED
	}
}

func stringToAttemptStatus(s string) pb.AttemptStatus {
	switch s {
	case "running":
		return pb.AttemptStatus_ATTEMPT_STATUS_RUNNING
	case "completed":
		return pb.AttemptStatus_ATTEMPT_STATUS_COMPLETED
	case "failed":
		return pb.AttemptStatus_ATTEMPT_STATUS_FAILED
	default:
		return pb.AttemptStatus_ATTEMPT_STATUS_UNSPECIFIED
	}
}
