package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	connect "connectrpc.com/connect"
	"github.com/spf13/cobra"
	"google.golang.org/protobuf/encoding/protojson"

	pb "quadsmith/api/gen/quadsmith"
	"quadsmith/api/gen/quadsmith/quadsmithconnect"
	fconfig "quadsmith/frontend/config"
)

var (
	envName   string
	workerURL string
	jsonOut   bool
)

func main() {
	if err := newRootCmd().Execute(); err != nil {
		os.Exit(1)
	}
}

func newRootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "qsops",
		Short: "Quadsmith Operations & Developer CLI",
		Long:  "Operational and developer tooling for background workers, queue management, and maintenance tasks.",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			fcfg, err := fconfig.Load(envName)
			if err != nil {
				return err
			}
			if cmd.Flags().Lookup("worker-url").Changed {
				fcfg.Ops.WorkerURL = workerURL
			} else {
				workerURL = fcfg.Ops.WorkerURL
			}
			if err := fcfg.ValidateOps(); err != nil {
				return err
			}
			return nil
		},
	}

	rootCmd.PersistentFlags().StringVar(&envName, "env", "", "Target environment (sandbox, test, staging, prod)")
	rootCmd.PersistentFlags().StringVar(&workerURL, "worker-url", "", "Worker service ConnectRPC base URL (env: QS_OPS_WORKER_URL)")
	rootCmd.PersistentFlags().BoolVar(&jsonOut, "json", false, "Output results in JSON format")

	rootCmd.AddCommand(newWorkersCmd())

	return rootCmd
}

func getWorkerClient() quadsmithconnect.WorkerServiceClient {
	return quadsmithconnect.NewWorkerServiceClient(http.DefaultClient, workerURL)
}

func newWorkersCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "workers",
		Aliases: []string{"worker"},
		Short:   "Inspect, trigger, and manage background workers and job queues",
	}

	cmd.AddCommand(newWorkersStatusCmd())
	cmd.AddCommand(newWorkersListCmd())
	cmd.AddCommand(newWorkersViewCmd())
	cmd.AddCommand(newWorkersTriggerCmd())
	cmd.AddCommand(newWorkersRetryCmd())

	return cmd
}

func newWorkersStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Display background worker daemon health and queue statistics",
		RunE: func(cmd *cobra.Command, args []string) error {
			client := getWorkerClient()
			resp, err := client.GetWorkerStatus(cmd.Context(), connect.NewRequest(&pb.GetWorkerStatusRequest{}))
			if err != nil {
				return fmt.Errorf("failed to get worker status: %w", err)
			}

			if jsonOut {
				out, _ := protojson.MarshalOptions{Multiline: true, Indent: "  "}.Marshal(resp.Msg)
				fmt.Println(string(out))
				return nil
			}

			msg := resp.Msg
			fmt.Printf("Worker Daemon Health: %s\n", formatHealth(msg.GetIsHealthy()))
			fmt.Println("Queue Depths:")
			fmt.Printf("  Pending: %d\n", msg.GetPendingJobs())
			fmt.Printf("  Running: %d\n", msg.GetRunningJobs())
			fmt.Printf("  Failed:  %d\n", msg.GetFailedJobs())
			fmt.Println("\nRegistered Task Types:")
			for _, t := range msg.GetRegisteredTasks() {
				fmt.Printf("  - %s\n", t)
			}
			return nil
		},
	}
}

func newWorkersListCmd() *cobra.Command {
	var statusFilter string
	var limit int32

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List recent background jobs",
		RunE: func(cmd *cobra.Command, args []string) error {
			client := getWorkerClient()

			var statusEnum pb.JobStatus
			switch strings.ToLower(statusFilter) {
			case "pending":
				statusEnum = pb.JobStatus_JOB_STATUS_PENDING
			case "running":
				statusEnum = pb.JobStatus_JOB_STATUS_RUNNING
			case "completed":
				statusEnum = pb.JobStatus_JOB_STATUS_COMPLETED
			case "failed":
				statusEnum = pb.JobStatus_JOB_STATUS_FAILED
			case "cancelled":
				statusEnum = pb.JobStatus_JOB_STATUS_CANCELLED
			default:
				statusEnum = pb.JobStatus_JOB_STATUS_UNSPECIFIED
			}

			req := &pb.ListJobsRequest{
				StatusFilter: statusEnum,
				Limit:        limit,
			}

			resp, err := client.ListJobs(cmd.Context(), connect.NewRequest(req))
			if err != nil {
				return fmt.Errorf("failed to list jobs: %w", err)
			}

			if jsonOut {
				out, _ := protojson.MarshalOptions{Multiline: true, Indent: "  "}.Marshal(resp.Msg)
				fmt.Println(string(out))
				return nil
			}

			jobs := resp.Msg.GetJobs()
			if len(jobs) == 0 {
				fmt.Println("No jobs found matching the criteria.")
				return nil
			}

			w := tabwriter.NewWriter(os.Stdout, 0, 4, 3, ' ', 0)
			fmt.Fprintln(w, "UUID\tJOB TYPE\tSTATUS\tSCHEDULED\tCREATED")
			for _, j := range jobs {
				sched := "-"
				if j.GetScheduledAt() != nil {
					sched = j.GetScheduledAt().AsTime().Local().Format("2006-01-02 15:04:05")
				}
				created := "-"
				if j.GetCreatedAt() != nil {
					created = j.GetCreatedAt().AsTime().Local().Format("2006-01-02 15:04:05")
				}

				fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
					j.GetUuid(),
					j.GetJobType(),
					formatJobStatus(j.GetStatus()),
					sched,
					created,
				)
			}
			return w.Flush()
		},
	}

	cmd.Flags().StringVar(&statusFilter, "status", "", "Filter jobs by status (pending, running, completed, failed)")
	cmd.Flags().Int32Var(&limit, "limit", 20, "Maximum number of jobs to list")

	return cmd
}

func newWorkersViewCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "view <job_uuid>",
		Short: "Display full details and execution attempts for a background job",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client := getWorkerClient()
			resp, err := client.GetJob(cmd.Context(), connect.NewRequest(&pb.GetJobRequest{JobUuid: args[0]}))
			if err != nil {
				return fmt.Errorf("failed to get job %s: %w", args[0], err)
			}

			if jsonOut {
				out, _ := protojson.MarshalOptions{Multiline: true, Indent: "  "}.Marshal(resp.Msg)
				fmt.Println(string(out))
				return nil
			}

			j := resp.Msg.GetJob()
			fmt.Printf("Job UUID:     %s\n", j.GetUuid())
			fmt.Printf("Job Type:     %s\n", j.GetJobType())
			fmt.Printf("Status:       %s\n", formatJobStatus(j.GetStatus()))
			if j.GetUniqueKey() != "" {
				fmt.Printf("Unique Key:   %s\n", j.GetUniqueKey())
			}
			fmt.Printf("Max Attempts: %d\n", j.GetMaxAttempts())
			if j.GetScheduledAt() != nil {
				fmt.Printf("Scheduled At: %s\n", j.GetScheduledAt().AsTime().Local().Format(time.RFC3339))
			}
			if j.GetCreatedAt() != nil {
				fmt.Printf("Created At:   %s\n", j.GetCreatedAt().AsTime().Local().Format(time.RFC3339))
			}
			if j.GetUpdatedAt() != nil {
				fmt.Printf("Updated At:   %s\n", j.GetUpdatedAt().AsTime().Local().Format(time.RFC3339))
			}

			fmt.Println("\nPayload:")
			var rawJSON map[string]any
			if err := json.Unmarshal([]byte(j.GetPayloadJson()), &rawJSON); err == nil {
				pretty, _ := json.MarshalIndent(rawJSON, "  ", "  ")
				fmt.Printf("  %s\n", string(pretty))
			} else {
				fmt.Printf("  %s\n", j.GetPayloadJson())
			}

			fmt.Println("\nExecution Attempts:")
			attempts := j.GetAttempts()
			if len(attempts) == 0 {
				fmt.Println("  (No attempts recorded yet)")
			} else {
				w := tabwriter.NewWriter(os.Stdout, 0, 4, 3, ' ', 0)
				fmt.Fprintln(w, "  #\tWORKER ID\tSTATUS\tSTARTED\tDURATION\tERROR")
				for _, att := range attempts {
					duration := "-"
					if att.GetCompletedAt() != nil && att.GetStartedAt() != nil {
						d := att.GetCompletedAt().AsTime().Sub(att.GetStartedAt().AsTime())
						duration = d.Round(time.Millisecond).String()
					}
					started := att.GetStartedAt().AsTime().Local().Format("15:04:05")
					errMsg := att.GetErrorMessage()
					if len(errMsg) > 50 {
						errMsg = errMsg[:47] + "..."
					}
					if errMsg == "" {
						errMsg = "-"
					}
					fmt.Fprintf(w, "  %d\t%s\t%s\t%s\t%s\t%s\n",
						att.GetAttemptNumber(),
						att.GetWorkerId(),
						formatAttemptStatus(att.GetStatus()),
						started,
						duration,
						errMsg,
					)
				}
				_ = w.Flush()
			}

			return nil
		},
	}
}

func newWorkersTriggerCmd() *cobra.Command {
	var payloadStr string
	var uniqueKey string

	cmd := &cobra.Command{
		Use:   "trigger <job_type>",
		Short: "Dispatch or enqueue a background job for immediate execution",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client := getWorkerClient()
			resp, err := client.TriggerJob(cmd.Context(), connect.NewRequest(&pb.TriggerJobRequest{
				JobType:     args[0],
				PayloadJson: payloadStr,
				UniqueKey:   uniqueKey,
			}))
			if err != nil {
				return fmt.Errorf("failed to trigger job: %w", err)
			}

			fmt.Printf("Job triggered successfully! Job UUID: %s\n", resp.Msg.GetJobUuid())
			return nil
		},
	}

	cmd.Flags().StringVar(&payloadStr, "payload", "{}", "JSON payload string for the job")
	cmd.Flags().StringVar(&uniqueKey, "unique-key", "", "Optional idempotency key")

	return cmd
}

func newWorkersRetryCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "retry <job_uuid>",
		Short: "Re-queue a failed job for execution",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client := getWorkerClient()
			resp, err := client.RetryJob(cmd.Context(), connect.NewRequest(&pb.RetryJobRequest{
				JobUuid: args[0],
			}))
			if err != nil {
				return fmt.Errorf("failed to retry job: %w", err)
			}

			if resp.Msg.GetSuccess() {
				fmt.Printf("Job %s successfully re-queued for execution.\n", args[0])
			} else {
				fmt.Printf("Failed to re-queue job %s.\n", args[0])
			}
			return nil
		},
	}
}

func formatHealth(healthy bool) string {
	if healthy {
		return "HEALTHY"
	}
	return "UNHEALTHY"
}

func formatJobStatus(s pb.JobStatus) string {
	switch s {
	case pb.JobStatus_JOB_STATUS_PENDING:
		return "PENDING"
	case pb.JobStatus_JOB_STATUS_RUNNING:
		return "RUNNING"
	case pb.JobStatus_JOB_STATUS_COMPLETED:
		return "COMPLETED"
	case pb.JobStatus_JOB_STATUS_FAILED:
		return "FAILED"
	case pb.JobStatus_JOB_STATUS_CANCELLED:
		return "CANCELLED"
	default:
		return "UNKNOWN"
	}
}

func formatAttemptStatus(s pb.AttemptStatus) string {
	switch s {
	case pb.AttemptStatus_ATTEMPT_STATUS_RUNNING:
		return "RUNNING"
	case pb.AttemptStatus_ATTEMPT_STATUS_COMPLETED:
		return "COMPLETED"
	case pb.AttemptStatus_ATTEMPT_STATUS_FAILED:
		return "FAILED"
	default:
		return "UNKNOWN"
	}
}
