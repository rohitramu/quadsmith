# <a id="tasks-and-jobs-architecture" >Tasks and Jobs Architecture</a>

## <a id="table-of-contents" >Table of Contents</a>

- [1. Background](#1-background)
- [2. Goal](#2-goal)
- [3. Proposal](#3-proposal)
  - [Terminology Alignment: Tasks vs. Jobs](#terminology-alignment-tasks-vs-jobs)
  - [Task RPCs in OpsService (server-ops)](#task-rpcs-in-opsservice-server-ops)
  - [Job RPCs in JobsService (server-jobs)](#job-rpcs-in-jobsservice-server-jobs)
  - [Strongly-Typed Per-Task Schedules in server_jobs.proto](#strongly-typed-per-task-schedules-in-server_jobsproto)
  - [CLI Command Structure: qsops tasks and qsops jobs](#cli-command-structure-qsops-tasks-and-qsops-jobs)
  - [Server and Scheduler Architecture in server-jobs Daemon](#server-and-scheduler-architecture-in-server-jobs-daemon)
  - [Username Release Warning Counter in Shared Auth](#username-release-warning-counter-in-shared-auth)
- [4. Future work](#4-future-work)
  - [Cloud Functions Deployment Tooling](#cloud-functions-deployment-tooling)

---

## <a id="1-background" >1. Background</a>

In Quadsmith, background worker operations currently span two subsystems: the
operations server [`server-ops`](../../src/go/server-ops) and a background
scheduling daemon [`server-jobs`](../../src/go/server-jobs).

Previously, [`server-jobs`](../../src/go/server-jobs/main.go) loaded a generic
repeated list of string schedule definitions
(`repeated JobScheduleConfig schedules`) and triggered operations through a
generic, string-based [`TriggerJob`](../../src/proto/quadsmith/ops/ops.proto)
RPC on [`OpsService`](../../src/proto/quadsmith/ops/ops.proto). Within
[`server-ops`](../../src/go/server-ops/pkg/ops/service.go), `TriggerJob`
contained a string switch statement dispatching to either specific methods like
[`CleanupExpiredOtps`](../../src/proto/quadsmith/ops/ops.proto) or inline
business logic like
[`auth.HandlePurgeSoftDeletedAccounts`](../../src/go/shared/pkg/auth/job.go).

Furthermore, the naming between "tasks" and "jobs" was conflated, and the
boundary between services was asymmetric: `server-jobs` was only a consumer
client without its own RPC interface, while `server-ops` handled both core
operations and scheduler audit reporting.

---

## <a id="2-goal" >2. Goal</a>

Establish a clean separation of concerns and clear taxonomy across subsystems:

1. **Tasks** (definitions and execution methods) live in
   [`server-ops`](../../src/go/server-ops) under
   [`OpsService`](../../src/proto/quadsmith/ops/ops.proto).
2. **Jobs** (execution history and schedule introspection) live in
   [`server-jobs`](../../src/go/server-jobs) under
   [`JobsService`](../../src/proto/quadsmith/jobs/jobs.proto).
3. [`server_jobs.proto`](../../src/proto/quadsmith/config/server_jobs.proto)
   features strongly-typed schedule configurations with one message per task
   type.
4. [`qsops`](../../src/go/cli-ops) organizes commands cleanly under
   `qsops tasks (list|get|trigger)` and `qsops jobs (list|get)`.

---

## <a id="3-proposal" >3. Proposal</a>

> **TL;DR:**
>
> - Clarify taxonomy: a task is triggerable; a job is an instance of a task.
> - Host task definition and trigger RPCs in `OpsService` (`server-ops`).
> - Host job history and schedule query RPCs in `JobsService` (`server-jobs`).
> - Structure `server_jobs.proto` with 1 strongly-typed schedule message per
>   task type.
> - Provide `qsops tasks (list|get|trigger)` and `qsops jobs (list|get)`.

The refactored design cleanly distributes responsibilities between the ops plane
and the job plane.

The operations server [`server-ops`](../../src/go/server-ops) serves
[`OpsService`](../../src/proto/quadsmith/ops/ops.proto), acting as the
authoritative owner of task definitions and execution logic. Every task is
exposed as a strongly-typed Connect RPC method (e.g., `CleanupExpiredOtps`,
`PurgeSoftDeletedAccounts`), and task catalog inspection is served via
`ListTasks` and `GetTask`.

The job server [`server-jobs`](../../src/go/server-jobs) is upgraded into an
active Connect RPC service serving
[`JobsService`](../../src/proto/quadsmith/jobs/jobs.proto) on port 8080 (sandbox
port 8083). It runs the scheduled background ticker loops and serves `ListJobs`
(chronological execution history) and `GetJob` (retrieves execution details and
associated task schedule information).

### <a id="terminology-alignment-tasks-vs-jobs" >Terminology Alignment: Tasks vs. Jobs</a>

- **Task**: A defined, repeatable maintenance action that can be executed or
  scheduled (e.g., `cleanup_expired_otps`, `purge_soft_deleted_accounts`). A
  task has a canonical name, a target RPC procedure, a description, and a
  default schedule. Tasks are owned and executed by `server-ops`.
- **Job**: A concrete execution instance of a task. Every time a task is
  triggered—either automatically by a scheduler ticker or manually by an
  operator via `qsops tasks trigger`—a new job execution record is created with
  a unique identifier, execution duration, caller context, status (`OK` or
  `ERROR`), and associated task schedule information. Job queries are served by
  `server-jobs`.

<details>
<summary><strong>Alternatives considered</strong></summary>

##### <a id="alt-keep-jobs-in-ops" >Host All Job RPCs in server-ops</a>

Keep `ListJobs` and `GetJob` on `OpsService` in `server-ops`.

**Pros:**

- Keeps `server-jobs` as a headless background client without an HTTP listener.

**Cons:**

- Conflates job orchestration and scheduling state into the general ops server.
- Violates the user's explicit directive to place job RPCs in `server-jobs`.

</details>

### <a id="task-rpcs-in-opsservice-server-ops" >Task RPCs in OpsService (server-ops)</a>

In
[`src/proto/quadsmith/ops/ops.proto`](../../src/proto/quadsmith/ops/ops.proto),
all five maintenance and cleanup tasks are defined as dedicated RPCs under
`// Maintenance & Cleanup Tasks`, and task inspection is exposed via `ListTasks`
and `GetTask`:

```protobuf
// Maintenance & Cleanup Tasks
rpc CleanupExpiredOtps(CleanupExpiredOtpsRequest) returns (CleanupExpiredOtpsResponse);
rpc CleanupExpiredTokens(CleanupExpiredTokensRequest) returns (CleanupExpiredTokensResponse);
rpc CleanupOrphanedBlobs(CleanupOrphanedBlobsRequest) returns (CleanupOrphanedBlobsResponse);
rpc PurgeSoftDeletedAccounts(PurgeSoftDeletedAccountsRequest) returns (PurgeSoftDeletedAccountsResponse);
rpc SendUsernameReleaseWarnings(SendUsernameReleaseWarningsRequest) returns (SendUsernameReleaseWarningsResponse);

// Tasks Inspection
rpc ListTasks(ListTasksRequest) returns (ListTasksResponse);
rpc GetTask(GetTaskRequest) returns (GetTaskResponse);
```

When any of the five task RPCs is executed on `server-ops`, the shared
[`OpsInterceptor`](../../src/go/shared/pkg/opsauth/interceptor.go) synchronously
writes the job execution audit record to `quadsmith_ops.internal_api_calls` and
returns the generated job identifier in the `X-Job-Id` response header.

#### <a id="task-messages" >Task Messages</a>

```protobuf
message TaskDef {
  string name = 1;
  string procedure = 2;
  string description = 3;
  string default_interval = 4;
}

message ListTasksRequest {}

message ListTasksResponse {
  repeated TaskDef tasks = 1;
}

message GetTaskRequest {
  string task_name = 1;
}

message GetTaskResponse {
  TaskDef task = 1;
}

message PurgeSoftDeletedAccountsRequest {}

message PurgeSoftDeletedAccountsResponse {
  int64 purged_count = 1;
}

message SendUsernameReleaseWarningsRequest {}

message SendUsernameReleaseWarningsResponse {
  int64 warning_count = 1;
}
```

The untyped `TriggerJob` and `ListScheduledJobs` RPCs and messages are removed
from `ops.proto`.

<details>
<summary><strong>Alternatives considered</strong></summary>

##### <a id="alt-generic-task-payload" >Generic Payload in Task RPCs</a>

Keep a generic `TriggerTask` with a payload string instead of strongly-typed
RPCs.

**Pros:**

- Single endpoint for all task triggering.

**Cons:**

- Loses type-safety and compile-time verification.
- Prevents individual method-level ACLs and granular JIT policies.

</details>

### <a id="job-rpcs-in-jobsservice-server-jobs" >Job RPCs in JobsService (server-jobs)</a>

In
[`src/proto/quadsmith/jobs/jobs.proto`](../../src/proto/quadsmith/jobs/jobs.proto),
define [`JobsService`](../../src/proto/quadsmith/jobs/jobs.proto) to serve job
execution history and schedule metadata:

```protobuf
syntax = "proto3";

package quadsmith.jobs;

import "google/protobuf/timestamp.proto";

option go_package = "quadsmith/shared/gen/quadsmith/jobs;jobs";

// ---------------------------------------------------------
// Services
// ---------------------------------------------------------

service JobsService {
  rpc GetJob(GetJobRequest) returns (GetJobResponse);
  rpc ListJobs(ListJobsRequest) returns (ListJobsResponse);
}

// ---------------------------------------------------------
// Domain model
// ---------------------------------------------------------

message JobRecord {
  string uuid = 1;
  string job_id = 2;
  string task_name = 3;
  string procedure = 4;
  string caller = 5;
  string caller_type = 6;
  string status = 7;
  int64 duration_ms = 8;
  optional string error_message = 9;
  google.protobuf.Timestamp created_at = 10;
  optional string schedule_interval = 11;
}

// ---------------------------------------------------------
// Request/response messages
// ---------------------------------------------------------

message GetJobRequest {
  string job_id = 1;
}

message GetJobResponse {
  JobRecord job = 1;
}

message ListJobsRequest {
  optional int32 limit = 1;
  optional string task_name = 2;
}

message ListJobsResponse {
  repeated JobRecord jobs = 1;
}
```

In `server-jobs`:

- `ListJobs`: Queries job execution records from
  `quadsmith_ops.internal_api_calls` for all task procedures, ordered by
  `created_at DESC` (latest on top).
- `GetJob`: Retrieves a specific job execution instance by its ID or UUID
  (`WHERE (id = $1 OR uuid::text = $1)`), hydrating both its runtime audit
  attributes and the configured schedule interval from `JobServerConfig`.

<details>
<summary><strong>Alternatives considered</strong></summary>

##### <a id="alt-in-memory-history" >Pure In-Memory History Ring Buffer</a>

Store job history only in an in-memory buffer inside `server-jobs`.

**Pros:**

- Eliminates the need for a database connection to `quadsmith_ops`.

**Cons:**

- All historical job data is lost whenever `server-jobs` restarts.
- Cannot see jobs triggered manually by operators via `qsops tasks trigger`.

</details>

### <a id="strongly-typed-per-task-schedules-in-server_jobsproto" >Strongly-Typed Per-Task Schedules in server_jobs.proto</a>

In
[`src/proto/quadsmith/config/server_jobs.proto`](../../src/proto/quadsmith/config/server_jobs.proto),
`JobServerConfig` is updated to include `ServerConfig server = 2;`,
`DatabaseConfig internal_database = 5;`, strongly-typed schedule sub-messages,
and RBAC `groups` and `bootstrap_tokens` (defined in
[`src/proto/quadsmith/config/common.proto`](../../src/proto/quadsmith/config/common.proto)).
Each schedule supports both a cron expression and a typed duration via a
`oneof schedule`:

```protobuf
message JobServerConfig {
  optional string environment = 1;
  ServerConfig server = 2;
  optional string ops_server_url = 3;
  optional string auth_token = 4;
  DatabaseConfig internal_database = 5;

  optional CleanupExpiredOtpsSchedule cleanup_expired_otps = 6;
  optional CleanupExpiredTokensSchedule cleanup_expired_tokens = 7;
  optional CleanupOrphanedBlobsSchedule cleanup_orphaned_blobs = 8;
  optional PurgeSoftDeletedAccountsSchedule purge_soft_deleted_accounts = 9;
  optional SendUsernameReleaseWarningsSchedule send_username_release_warnings = 10;
  repeated InternalGroupConfig groups = 11;
  repeated BootstrapTokenConfig bootstrap_tokens = 12;
}

message CleanupExpiredOtpsSchedule {
  optional bool enabled = 1;
  oneof schedule {
    string cron = 2;
    google.protobuf.Duration interval = 3;
  }
}

message CleanupExpiredTokensSchedule {
  optional bool enabled = 1;
  oneof schedule {
    string cron = 2;
    google.protobuf.Duration interval = 3;
  }
}

message CleanupOrphanedBlobsSchedule {
  optional bool enabled = 1;
  oneof schedule {
    string cron = 2;
    google.protobuf.Duration interval = 3;
  }
}

message PurgeSoftDeletedAccountsSchedule {
  optional bool enabled = 1;
  oneof schedule {
    string cron = 2;
    google.protobuf.Duration interval = 3;
  }
}

message SendUsernameReleaseWarningsSchedule {
  optional bool enabled = 1;
  oneof schedule {
    string cron = 2;
    google.protobuf.Duration interval = 3;
  }
}
```

The textproto configuration files across all environments
([`sandbox.textproto`](../../src/go/server-jobs/config/sandbox.textproto),
[`staging.textproto`](../../src/go/server-jobs/config/staging.textproto),
[`prod.textproto`](../../src/go/server-jobs/config/prod.textproto), and
[`test.textproto`](../../src/go/server-jobs/config/test.textproto)) can specify
either a `cron` string or a typed `interval`:

```protobuf
environment: "sandbox"

server {
  port: 8080
  read_timeout { seconds: 30 }
  write_timeout { seconds: 60 }
  idle_timeout { seconds: 120 }
  max_request_bytes: 2097152
}

ops_server_url: "http://127.0.0.1:8082"
auth_token: "${QS_JOB_SERVER_AUTH_TOKEN:-qs_robot_sandbox_token_67890}"

internal_database {
  url: "${QS_INTERNAL_DATABASE_URL:-postgres://postgres:postgres@127.0.0.1:5432/quadsmith_ops?sslmode=disable}"
  name: "quadsmith_ops"
  max_connections: 10
  min_connections: 2
}

cleanup_expired_otps {
  enabled: true
  interval { seconds: 21600 } # 6 hours
}

cleanup_expired_tokens {
  enabled: true
  cron: "0 */12 * * *"
}

cleanup_orphaned_blobs {
  enabled: true
  interval { seconds: 600 } # 10 minutes
}

purge_soft_deleted_accounts {
  enabled: true
  interval { seconds: 28800 } # 8 hours
}

send_username_release_warnings {
  enabled: true
  cron: "0 0 * * *"
}
```

**Duration-to-Cron Conversion (January 1, 1970 12:00 AM UTC Anchor):**

- When a schedule configures an `interval` duration rather than an explicit
  `cron` expression, `server-jobs` calculates the recurring trigger schedule
  anchored at the Unix epoch: **January 1st, 1970 at 12:00 AM UTC**
  (`time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC)`):
  1. Trigger timestamps are calculated as integer multiples of the interval from
     epoch ($T_k = k \times \text{duration}$).
  2. Because the Unix epoch begins at 00:00:00 UTC, recurring intervals align
     cleanly with standard UTC wall-clock boundaries:
     - Intervals in minutes dividing 60 (e.g. 10m): `*/10 * * * *`
     - Intervals in hours dividing 24 (e.g. 6h, 8h, 12h): `0 */6 * * *`,
       `0 0,8,16 * * *`
     - 24-hour intervals: `0 0 * * *`
     - Multi-day intervals: Day-of-week patterns aligned from the Thursday epoch
       (e.g., 7d corresponds to `0 0 * * 4`).
- The converted string is fed directly into the cron engine, ensuring that both
  `interval` and `cron` schedules are handled through the exact same validated
  cron pipeline!
- Startup validation parses all resulting cron strings using `robfig/cron/v3`
  (`cron.NewParser(...)`). If any expression is invalid, `server-jobs`
  immediately aborts startup (`log.Fatalf`).

<details>
<summary><strong>Alternatives considered</strong></summary>

##### <a id="alt-generic-schedule-repeated" >Retain Repeated Generic Schedule List</a>

Keep generic `repeated JobScheduleConfig schedules` with string fields.

**Pros:**

- Minimal changes to protobuf configuration.

**Cons:**

- Lacks compile-time safety.
- Cannot configure task-specific schedule parameters.

</details>

### <a id="cli-command-structure-qsops-tasks-and-qsops-jobs" >CLI Command Structure: qsops tasks and qsops jobs</a>

In
[`src/proto/quadsmith/config/cli_ops.proto`](../../src/proto/quadsmith/config/cli_ops.proto),
add:

```protobuf
optional string jobs_server_url = 4;
```

Defaulting in sandbox CLI configs to `http://127.0.0.1:8083`.

In [`src/go/cli-ops/main.go`](../../src/go/cli-ops/main.go):

#### <a id="tasks-commands" >1. Tasks Commands (`qsops tasks` -> targets `server-ops`)</a>

- `qsops tasks list`: Calls `OpsService.ListTasks`, displaying all defined
  tasks, descriptions, and target procedures.
- `qsops tasks get <task-name>`: Calls `OpsService.GetTask`, displaying the full
  description, target procedure, and default schedule.
- `qsops tasks trigger <task-name>`: Calls the dedicated typed RPC on
  `OpsService` (e.g., `CleanupExpiredOtps`, `PurgeSoftDeletedAccounts`),
  outputting the execution result and generated job ID (`X-Job-Id`).

#### <a id="jobs-commands" >2. Jobs Commands (`qsops jobs` -> targets `server-jobs`)</a>

- `qsops jobs list [--limit=N] [--task=name]`: Calls `JobsService.ListJobs`,
  displaying the chronological execution history (latest on top).
- `qsops jobs get <job-id>`: Calls `JobsService.GetJob`, displaying full
  execution details, runtime metrics, caller, and associated task schedule
  information.

<details>
<summary><strong>Alternatives considered</strong></summary>

##### <a id="alt-all-cli-under-jobs" >Keep Tasks and Jobs Combined Under qsops jobs</a>

Keep all commands under `qsops jobs` and multiplex RPC calls to both servers.

**Pros:**

- Keeps single top-level command.

**Cons:**

- Obscures the clean service separation and perpetuates terminology confusion.

</details>

### <a id="server-and-scheduler-architecture-in-server-jobs-daemon" >Server and Scheduler Architecture in server-jobs Daemon</a>

[`src/go/server-jobs/main.go`](../../src/go/server-jobs/main.go) will be
refactored into a full dual-mode daemon:

1. **Connect RPC Server & Shared Auth Interceptor**: Initializes an
   `http.Server` on `cfg.GetServer().GetPort()`, serving
   `jobsconnect.NewJobsServiceHandler(svc)` protected by the shared
   [`opsauth.NewOpsInterceptor`](../../src/go/shared/pkg/opsauth/interceptor.go)
   and a standard `/healthz` endpoint.
2. **Background Scheduler Goroutines**: For each strongly-typed task schedule
   that is enabled, it registers a `robfig/cron/v3` job invoking the typed
   `OpsService` RPC over the authenticated Connect client (`ROBOT` tokens bypass
   interactive human JIT checks while still enforcing group RBAC).
3. **Database Pool**: Connects to `quadsmith_ops` via `pgxpool.NewWithConfig` to
   query job execution records from `internal_api_calls`.
4. **Docker & Sandbox Port Mapping**: In
   [`test/sandbox/docker-compose.yaml`](../../test/sandbox/docker-compose.yaml),
   the `jobs` service maps `"${JOBS_PORT:-8083}:8080"` and includes a health
   check: `wget -q -O - http://127.0.0.1:8080/healthz || exit 1`.

<details>
<summary><strong>Alternatives considered</strong></summary>

##### <a id="alt-separate-jobs-api-server" >Split Scheduler Daemon and Jobs API Server</a>

Build two separate binaries: one daemon for the tickers, and one HTTP server for
`JobsService`.

**Pros:**

- Complete separation of the scheduler process and the query server process.

**Cons:**

- Doubles container overhead and binary maintenance for a lightweight jobs
  service.

</details>

### <a id="username-release-warning-counter-in-shared-auth" >Username Release Warning Counter in Shared Auth</a>

In [`src/go/shared/pkg/auth/job.go`](../../src/go/shared/pkg/auth/job.go),
update `HandleUsernameReleaseWarning` signature and implementation from:

`func HandleUsernameReleaseWarning(ctx context.Context, pool *pgxpool.Pool) error`

to:

`func HandleUsernameReleaseWarning(ctx context.Context, pool *pgxpool.Pool) (int64, error)`

**Idempotency & Concurrency Safety:**

- **Idempotency filter**: Explicitly filters by
  `WHERE hh.release_warning_sent_at IS NULL`. Once a warning is dispatched,
  `release_warning_sent_at` is set to `NOW()`. Any subsequent runs (or repeated
  manual triggers) ignore the username hold.
- **Concurrency protection**: Uses a transaction with row locking
  (`FOR UPDATE SKIP LOCKED`) so concurrent executions cannot process the same
  expiring username twice.
- **Return counter**: Returns total count of warnings dispatched.

---

## <a id="4-future-work" >4. Future work</a>

### <a id="cloud-functions-deployment-tooling" >Cloud Functions Deployment Tooling</a>

- Implement Terraform or IaC manifests that read the strongly-typed cron
  configurations to provision Google Cloud Scheduler jobs targeting individual
  Cloud Function endpoints.
- Package serverless function entrypoints for zero-server deployments.
