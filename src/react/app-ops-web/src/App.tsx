import React, { useCallback, useEffect, useMemo, useState } from "react";
import { createClient, type Interceptor } from "@connectrpc/connect";
import { createConnectTransport } from "@connectrpc/connect-web";
import { ArrowDown, ArrowUp } from "lucide-react";
import {
  SmartFilterInput,
  filterAndSortRecords,
  type FieldDef,
  type SyntaxExamples,
} from "@quadsmith/shared";
import {
  OpsService,
  type GetCurrentOperatorResponse,
  type GetDatabaseStatsResponse,
  type GetOAuthConfigResponse,
  type GetSystemHealthResponse,
  type GetSystemMetricsResponse,
  type GetTraceResponse,
  type IamGroupSummary,
  type IamUserSummary,
  type InternalApiCallRecord,
  type InternalTokenRecord,
  type JobRecord,
  type RouteMetricStat,
  type RouteRequestLog,
  type TableStat,
  type TaskDef,
  type TraceLogRecord,
  type TraceSpan,
} from "@quadsmith/shared/gen/quadsmith/ops/ops_pb";

const OPS_TOKEN_STORAGE_KEY = "qs_ops_token";
const SANDBOX_DEFAULT_OPERATOR_TOKEN = "qs_op_sandbox_token_12345";

export type SortDirection = "asc" | "desc";
export type FilterPreset = { label: string; query: string };

// ---------------------------------------------------------------------------
// Collection CEL Field Definitions, Quick Filter Presets & Syntax Guides
// ---------------------------------------------------------------------------

export const ROUTE_METRIC_FIELDS: FieldDef[] = [
  {
    name: "service_name",
    type: "string",
    description: "Go service binary name",
    examples: ['"server-api"', '"server-ops-api"', '"server-web"'],
  },
  {
    name: "region",
    type: "string",
    description: "Deployment region identifier",
    examples: ['"local"', '"us-central1"'],
  },
  {
    name: "method",
    type: "string",
    description: "HTTP method or RPC verb",
    examples: ['"POST"', '"GET"'],
  },
  {
    name: "route",
    type: "string",
    description: "ConnectRPC procedure or HTTP route path",
    examples: ['"/quadsmith.api.AntennaService/ListAntennas"', '"/healthz"'],
  },
  {
    name: "total_requests",
    type: "number",
    description: "Total requests recorded in window",
    examples: ["10", "100"],
  },
  {
    name: "success_count",
    type: "number",
    description: "Successful (2xx/OK) request count",
    examples: ["10", "50"],
  },
  {
    name: "error_count",
    type: "number",
    description: "Failed (5xx/Error) request count",
    examples: ["0", "1"],
  },
  {
    name: "availability_percent",
    type: "number",
    description: "Route availability percentage (0–100)",
    examples: ["99.9", "100.0"],
  },
  {
    name: "p50_rtt_ms",
    type: "number",
    description: "Median (P50) round-trip latency in milliseconds",
    examples: ["1.5", "5.0"],
  },
  {
    name: "p95_rtt_ms",
    type: "number",
    description: "95th percentile (P95) round-trip latency in milliseconds",
    examples: ["10.0", "25.0"],
  },
  {
    name: "p99_rtt_ms",
    type: "number",
    description: "99th percentile (P99) round-trip latency in milliseconds",
    examples: ["20.0", "50.0"],
  },
];

export const DEFAULT_SERVICE_CEL_FILTER =
  'service_name == "server-api" && route != "/healthz" && route != "/metrics"';

export const ROUTE_METRIC_PRESETS: FilterPreset[] = [
  { label: "Public API", query: DEFAULT_SERVICE_CEL_FILTER },
  {
    label: "Ops API",
    query: 'service_name == "server-ops-api" && route != "/healthz" && route != "/metrics"',
  },
  { label: "Errors Only", query: "error_count > 0" },
  { label: "High P95 (>10ms)", query: "p95_rtt_ms > 10.0" },
];

export const ROUTE_METRIC_SYNTAX: SyntaxExamples = {
  stringMatching: [
    'route.contains("Antenna")',
    'service_name == "server-api"',
    'route.startsWith("/quadsmith.ops.")',
  ],
  numericComparison: ["p95_rtt_ms >= 5.0", "error_count > 0", "availability_percent < 99.9"],
  compoundConditions: [
    'service_name == "server-api" && route != "/healthz" && route != "/metrics"',
    "error_count > 0 || availability_percent < 99.5",
  ],
};

export const TRACE_REQUEST_FIELDS: FieldDef[] = [
  {
    name: "trace_id",
    type: "string",
    description: "Ingress-minted UUIDv7 X-Trace-Id",
    examples: ['"0195f0..."'],
  },
  {
    name: "region",
    type: "string",
    description: "Deployment region identifier",
    examples: ['"local"', '"us-central1"'],
  },
  {
    name: "service_name",
    type: "string",
    description: "Target Go service name",
    examples: ['"server-api"', '"server-ops-api"'],
  },
  {
    name: "method",
    type: "string",
    description: "HTTP request method",
    examples: ['"POST"', '"GET"'],
  },
  {
    name: "route",
    type: "string",
    description: "Request path or ConnectRPC procedure",
    examples: ['"/quadsmith.ops.OpsService/GetSystemHealth"'],
  },
  {
    name: "status_code",
    type: "number",
    description: "HTTP response status code",
    examples: ["200", "403", "500"],
  },
  {
    name: "rtt_ms",
    type: "number",
    description: "Request duration in milliseconds",
    examples: ["2.5", "15.0"],
  },
  {
    name: "success",
    type: "boolean",
    description: "Whether the request completed without error",
    examples: ["true", "false"],
  },
  {
    name: "occurred_at",
    type: "string",
    description: "ISO-8601 UTC timestamp of request",
    examples: ['"2026-10-18"'],
  },
];

export const TRACE_REQUEST_PRESETS: FilterPreset[] = [
  { label: "Public API", query: DEFAULT_SERVICE_CEL_FILTER },
  {
    label: "Ops API",
    query: 'service_name == "server-ops-api" && route != "/healthz" && route != "/metrics"',
  },
  { label: "Failed Requests", query: "success == false" },
  { label: "5xx Status", query: "status_code >= 500" },
  { label: "Slow (>15ms)", query: "rtt_ms > 15.0" },
];

export const TRACE_REQUEST_SYNTAX: SyntaxExamples = {
  stringMatching: [
    'trace_id.startsWith("019")',
    'route.contains("OpsService")',
    'service_name == "server-ops-api"',
  ],
  numericComparison: ["status_code >= 400", "rtt_ms > 15.0", "status_code == 200"],
  compoundConditions: [
    'success == false && service_name == "server-api"',
    "status_code >= 500 || rtt_ms > 25.0",
  ],
};

export const TASK_FIELDS: FieldDef[] = [
  {
    name: "name",
    type: "string",
    description: "Unique snake_case task identifier",
    examples: ['"cleanup_expired_otps"', '"set_user_role"'],
  },
  {
    name: "description",
    type: "string",
    description: "Human-readable task summary",
    examples: ['"Purge"', '"role"'],
  },
  {
    name: "procedure",
    type: "string",
    description: "OpsService ConnectRPC method name",
    examples: ['"CleanupExpiredOtps"', '"SetUserRole"'],
  },
  {
    name: "default_interval",
    type: "string",
    description: "Default cron execution cadence",
    examples: ['"10m"', '"6h"', '"24h"', '"manual"'],
  },
];

export const TASK_PRESETS: FilterPreset[] = [
  { label: "Cleanup Tasks", query: 'name.startsWith("cleanup_")' },
  { label: "Scheduled Cron", query: 'default_interval != "manual"' },
  { label: "Manual Only", query: 'default_interval == "manual"' },
];

export const TASK_SYNTAX: SyntaxExamples = {
  stringMatching: [
    'name.startsWith("cleanup_")',
    'description.contains("Purges")',
    'default_interval == "6h"',
  ],
  numericComparison: ['default_interval != "manual"'],
  compoundConditions: [
    'name.contains("cleanup") && default_interval != "manual"',
    'default_interval == "24h" || default_interval == "manual"',
  ],
};

export const JOB_FIELDS: FieldDef[] = [
  {
    name: "job_id",
    type: "string",
    description: "Human-readable job execution ID",
    examples: ['"job_019..."'],
  },
  {
    name: "trace_id",
    type: "string",
    description: "Correlated UUIDv7 X-Trace-Id",
    examples: ['"0195f0..."'],
  },
  {
    name: "region",
    type: "string",
    description: "Deployment region where the job executed",
    examples: ['"local"', '"us-central1"'],
  },
  {
    name: "task_name",
    type: "string",
    description: "Triggered operational task name",
    examples: ['"cleanup_expired_otps"', '"set_user_role"'],
  },
  {
    name: "procedure",
    type: "string",
    description: "Invoked OpsService RPC procedure",
    examples: ['"/quadsmith.ops.OpsService/CleanupExpiredOtps"'],
  },
  {
    name: "caller",
    type: "string",
    description: "Operator or robot identity that triggered the job",
    examples: ['"rohitramu"', '"system:cron"'],
  },
  {
    name: "caller_type",
    type: "string",
    description: "Identity classification (HUMAN or ROBOT)",
    examples: ['"HUMAN"', '"ROBOT"'],
  },
  {
    name: "status",
    type: "string",
    description: "Execution outcome (OK or ERROR)",
    examples: ['"OK"', '"ERROR"'],
  },
  {
    name: "duration_ms",
    type: "number",
    description: "Execution duration in milliseconds",
    examples: ["5", "50"],
  },
  {
    name: "error_message",
    type: "string",
    description: "Error diagnostic message if failed",
    examples: ['"timeout"'],
  },
  {
    name: "created_at",
    type: "string",
    description: "ISO-8601 UTC execution timestamp",
    examples: ['"2026-10-18"'],
  },
];

export const JOB_PRESETS: FilterPreset[] = [
  { label: "Failed Jobs", query: 'status != "OK"' },
  { label: "Robot / Cron Triggered", query: 'caller_type == "ROBOT"' },
  { label: "Human Triggered", query: 'caller_type == "HUMAN"' },
  { label: "Slow (>50ms)", query: "duration_ms >= 50" },
];

export const JOB_SYNTAX: SyntaxExamples = {
  stringMatching: ['task_name.contains("cleanup")', 'caller_type == "ROBOT"', 'status == "OK"'],
  numericComparison: ["duration_ms >= 50", "duration_ms < 10"],
  compoundConditions: [
    'caller_type == "HUMAN" && status == "OK"',
    'status != "OK" || duration_ms >= 100',
  ],
};

export const DB_TABLE_FIELDS: FieldDef[] = [
  {
    name: "table_name",
    type: "string",
    description: "PostgreSQL table name in quadsmith_api",
    examples: ['"motors"', '"frames"', '"user_accounts"'],
  },
  {
    name: "row_count",
    type: "number",
    description: "Estimated live tuple count",
    examples: ["0", "25", "100"],
  },
  {
    name: "total_size_bytes",
    type: "number",
    description: "Total relation disk footprint in bytes",
    examples: ["16384", "65536"],
  },
];

export const DB_TABLE_PRESETS: FilterPreset[] = [
  { label: "Non-Empty Tables", query: "row_count > 0" },
  { label: "Large Tables (>32KB)", query: "total_size_bytes >= 32768" },
  {
    label: "User & Auth Tables",
    query: 'table_name.contains("user") || table_name.contains("auth")',
  },
];

export const DB_TABLE_SYNTAX: SyntaxExamples = {
  stringMatching: [
    'table_name.contains("motor")',
    'table_name.startsWith("user")',
    'table_name == "builds"',
  ],
  numericComparison: ["row_count > 0", "total_size_bytes >= 32768"],
  compoundConditions: [
    "row_count >= 10 && total_size_bytes > 16384",
    'table_name.contains("user") || row_count > 50',
  ],
};

export const TOKEN_FIELDS: FieldDef[] = [
  {
    name: "id",
    type: "string",
    description: "Token resource identifier",
    examples: ['"tok_019..."'],
  },
  {
    name: "name",
    type: "string",
    description: "Descriptive token label",
    examples: ['"oncall-cli"', '"cron-worker"'],
  },
  {
    name: "type",
    type: "string",
    description: "Token principal type (HUMAN or ROBOT)",
    examples: ['"HUMAN"', '"ROBOT"'],
  },
  {
    name: "group_name",
    type: "string",
    description: "Assigned IAM authorization group",
    examples: ['"operators"', '"admins"', '"cron-robots"'],
  },
  {
    name: "status",
    type: "string",
    description: "Lifecycle status (ACTIVE or REVOKED)",
    examples: ['"ACTIVE"', '"REVOKED"'],
  },
  {
    name: "created_at",
    type: "string",
    description: "ISO-8601 creation timestamp",
    examples: ['"2026-10-18"'],
  },
  {
    name: "expires_at",
    type: "string",
    description: "ISO-8601 expiration timestamp",
    examples: ['"2026-11-18"'],
  },
];

export const TOKEN_PRESETS: FilterPreset[] = [
  { label: "Active Only", query: 'status == "ACTIVE"' },
  { label: "Robot Tokens", query: 'type == "ROBOT"' },
  { label: "Human Tokens", query: 'type == "HUMAN"' },
  { label: "Revoked", query: 'status == "REVOKED"' },
];

export const TOKEN_SYNTAX: SyntaxExamples = {
  stringMatching: ['type == "ROBOT"', 'group_name == "operators"', 'name.contains("cli")'],
  numericComparison: ['status == "ACTIVE"', 'status != "REVOKED"'],
  compoundConditions: [
    'type == "ROBOT" && status == "ACTIVE"',
    'group_name == "admins" || group_name == "operators"',
  ],
};

export const AUDIT_CALL_FIELDS: FieldDef[] = [
  {
    name: "trace_id",
    type: "string",
    description: "Correlated UUIDv7 X-Trace-Id",
    examples: ['"0195f0..."'],
  },
  {
    name: "region",
    type: "string",
    description: "Deployment region of server-ops-api",
    examples: ['"local"', '"us-central1"'],
  },
  {
    name: "caller",
    type: "string",
    description: "Operator username or robot name",
    examples: ['"rohitramu"', '"system:cron"'],
  },
  {
    name: "caller_type",
    type: "string",
    description: "Caller principal type (HUMAN or ROBOT)",
    examples: ['"HUMAN"', '"ROBOT"'],
  },
  {
    name: "group_name",
    type: "string",
    description: "Caller IAM group",
    examples: ['"admins"', '"operators"'],
  },
  {
    name: "method",
    type: "string",
    description: "Invoked ConnectRPC procedure path",
    examples: ['"/quadsmith.ops.OpsService/AcquireJitGrant"'],
  },
  {
    name: "status",
    type: "string",
    description: "RPC outcome status (OK or ERROR)",
    examples: ['"OK"', '"ERROR"'],
  },
  {
    name: "duration_ms",
    type: "number",
    description: "RPC execution duration in milliseconds",
    examples: ["2", "25"],
  },
  {
    name: "jit_grant_id",
    type: "string",
    description: "Active JIT grant ID if elevated",
    examples: ['"jit_019..."'],
  },
  {
    name: "created_at",
    type: "string",
    description: "ISO-8601 UTC audit timestamp",
    examples: ['"2026-10-18"'],
  },
];

export const AUDIT_CALL_PRESETS: FilterPreset[] = [
  { label: "JIT-Elevated Calls", query: 'jit_grant_id != ""' },
  { label: "Errors / Denied", query: 'status != "OK"' },
  { label: "Human Operators", query: 'caller_type == "HUMAN"' },
  { label: "Robot Callers", query: 'caller_type == "ROBOT"' },
];

export const AUDIT_CALL_SYNTAX: SyntaxExamples = {
  stringMatching: [
    'method.contains("AcquireJitGrant")',
    'caller == "rohitramu"',
    'jit_grant_id != ""',
  ],
  numericComparison: ["duration_ms >= 10", 'status == "OK"'],
  compoundConditions: [
    'jit_grant_id != "" && caller_type == "HUMAN"',
    'status != "OK" || duration_ms > 25',
  ],
};

export function getOpsApiBaseUrl(): string {
  if (typeof window !== "undefined") {
    const win = window as unknown as { __QS_OPS_API_URL__?: string };
    if (win.__QS_OPS_API_URL__) {
      return win.__QS_OPS_API_URL__;
    }
    const host = window.location.hostname;
    if (host === "ops.quadsmith.com") {
      return "https://ops-api.quadsmith.com";
    }
    if (host.endsWith(".ops.quadsmith.com")) {
      return `https://${host.replace(".ops.quadsmith.com", ".ops-api.quadsmith.com")}`;
    }
  }
  return "http://127.0.0.1:8082";
}

export function setOpsCookieAndStorage(token: string | null): void {
  if (typeof window === "undefined") return;
  if (token && token.trim() !== "") {
    const clean = token.trim();
    localStorage.setItem(OPS_TOKEN_STORAGE_KEY, clean);
    document.cookie = `qs_ops_token=${encodeURIComponent(clean)}; Path=/; SameSite=Lax; Max-Age=86400`;
  } else {
    localStorage.removeItem(OPS_TOKEN_STORAGE_KEY);
    document.cookie = "qs_ops_token=; Path=/; SameSite=Lax; Max-Age=0";
  }
}

export function getStoredOpsToken(): string {
  if (typeof window === "undefined") return "";
  return localStorage.getItem(OPS_TOKEN_STORAGE_KEY) || "";
}

type ActiveTab = "health" | "tasks" | "jobs" | "db" | "iam";
export type TelemetryExplorerTab = "routes" | "traces";

export function applyRouteToTraceCelFilter(
  currentFilter: string,
  route: string,
  serviceName?: string,
): string {
  let cleaned = currentFilter
    .replace(/\s*&&\s*route\s*==\s*("[^"]*"|'[^']*')/g, "")
    .replace(/route\s*==\s*("[^"]*"|'[^']*')\s*&&\s*/g, "")
    .replace(/^\s*route\s*==\s*("[^"]*"|'[^']*')\s*$/g, "")
    .trim();

  if (serviceName && /service_name\s*==\s*("[^"]*"|'[^']*')/.test(cleaned)) {
    cleaned = cleaned.replace(
      /service_name\s*==\s*("[^"]*"|'[^']*')/g,
      `service_name == "${serviceName}"`,
    );
  }

  const routeClause = `route == "${route}"`;
  if (!cleaned) {
    return routeClause;
  }
  return `${cleaned} && ${routeClause}`;
}

function formatTimestamp(ts?: { seconds: bigint }): string {
  if (!ts) return "—";
  const ms = Number(ts.seconds) * 1000;
  if (!Number.isFinite(ms) || ms <= 0) return "—";
  return new Date(ms).toISOString().replace("T", " ").slice(0, 19) + " UTC";
}

function formatIsoTimestamp(ts?: { seconds: bigint }): string {
  if (!ts) return "";
  const ms = Number(ts.seconds) * 1000;
  if (!Number.isFinite(ms) || ms <= 0) return "";
  return new Date(ms).toISOString();
}

function formatBytes(bytes: bigint | number): string {
  const b = Number(bytes);
  if (!Number.isFinite(b) || b <= 0) return "0 B";
  if (b < 1024) return `${b} B`;
  if (b < 1024 * 1024) return `${(b / 1024).toFixed(1)} KB`;
  return `${(b / (1024 * 1024)).toFixed(2)} MB`;
}

function buildSortSpec(field: string, dir: SortDirection): string {
  if (!field) return "";
  return dir === "desc" ? `^${field}` : field;
}

interface SortOption {
  field: string;
  label: string;
}

interface CollectionSortBarProps {
  totalCount: number;
  filteredCount: number;
  noun: string;
  sortField: string;
  sortDir: SortDirection;
  options: SortOption[];
  onSortFieldChange: (field: string) => void;
  onSortDirToggle: () => void;
  testIdPrefix: string;
}

function CollectionSortBar({
  totalCount,
  filteredCount,
  noun,
  sortField,
  sortDir,
  options,
  onSortFieldChange,
  onSortDirToggle,
  testIdPrefix,
}: CollectionSortBarProps) {
  return (
    <div className="flex flex-wrap items-center justify-between gap-2 text-xs font-mono text-slate-400">
      <div>
        Showing <span className="text-slate-200 font-semibold">{filteredCount}</span> of{" "}
        <span className="text-slate-200 font-semibold">{totalCount}</span> {noun}
      </div>
      <div className="flex items-center gap-1.5">
        <select
          aria-label={`Sort ${noun} by field`}
          data-testid={`${testIdPrefix}-sort-select`}
          value={sortField}
          onChange={(e) => onSortFieldChange(e.target.value)}
          className="bg-[#090d16] border border-slate-700 rounded-lg px-2.5 py-1 text-xs font-mono text-slate-200"
        >
          {options.map((opt) => (
            <option key={opt.field} value={opt.field}>
              Sort: {opt.label}
            </option>
          ))}
        </select>
        <button
          type="button"
          data-testid={`${testIdPrefix}-sort-dir-btn`}
          onClick={onSortDirToggle}
          title={
            sortDir === "asc"
              ? "Sort ascending (click for descending)"
              : "Sort descending (click for ascending)"
          }
          aria-label={sortDir === "asc" ? "Sort ascending" : "Sort descending"}
          className="p-1.5 rounded-lg border border-slate-700 bg-[#090d16] hover:bg-slate-800 text-slate-200 transition-colors cursor-pointer"
        >
          {sortDir === "desc" ? <ArrowDown size={14} /> : <ArrowUp size={14} />}
        </button>
      </div>
    </div>
  );
}

interface SortableThProps {
  label: string;
  field: string;
  activeField: string;
  direction: SortDirection;
  onToggle: (field: string) => void;
}

function SortableTh({ label, field, activeField, direction, onToggle }: SortableThProps) {
  const isActive = activeField === field;
  return (
    <th className="py-2 px-3 select-none">
      <button
        type="button"
        onClick={() => onToggle(field)}
        className={`inline-flex items-center gap-1 font-mono text-xs uppercase tracking-wider cursor-pointer transition-colors ${
          isActive ? "text-sky-300 font-bold" : "text-slate-400 hover:text-slate-200"
        }`}
      >
        <span>{label}</span>
        {isActive && (direction === "desc" ? <ArrowDown size={12} /> : <ArrowUp size={12} />)}
      </button>
    </th>
  );
}

function toggleColumnSort(
  clickedField: string,
  currentField: string,
  currentDir: SortDirection,
  setField: (f: string) => void,
  setDir: (d: SortDirection) => void,
) {
  if (clickedField === currentField) {
    setDir(currentDir === "asc" ? "desc" : "asc");
  } else {
    setField(clickedField);
    setDir("asc");
  }
}

export default function App() {
  const [token, setToken] = useState<string>(() => getStoredOpsToken());
  const [manualTokenInput, setManualTokenInput] = useState<string>("");
  const [oauthConfig, setOauthConfig] = useState<GetOAuthConfigResponse | null>(null);
  const [operatorInfo, setOperatorInfo] = useState<GetCurrentOperatorResponse | null>(null);
  const [authError, setAuthError] = useState<string>("");
  const [loading, setLoading] = useState<boolean>(false);

  const [activeTab, setActiveTab] = useState<ActiveTab>("health");
  const [telemetrySubTab, setTelemetrySubTab] = useState<TelemetryExplorerTab>("routes");
  const [regionFilter, setRegionFilter] = useState<string>("all");
  const [serviceFilter, setServiceFilter] = useState<string>("all");
  const [metricsWindow, setMetricsWindow] = useState<string>("5m");

  // CEL Filter & Sort state per collection
  const [routeCelFilter, setRouteCelFilter] = useState<string>(DEFAULT_SERVICE_CEL_FILTER);
  const [routeSortField, setRouteSortField] = useState<string>("total_requests");
  const [routeSortDir, setRouteSortDir] = useState<SortDirection>("desc");

  const [traceCelFilter, setTraceCelFilter] = useState<string>(DEFAULT_SERVICE_CEL_FILTER);
  const [traceSortField, setTraceSortField] = useState<string>("occurred_at");
  const [traceSortDir, setTraceSortDir] = useState<SortDirection>("desc");

  const [taskCelFilter, setTaskCelFilter] = useState<string>("");
  const [taskSortField, setTaskSortField] = useState<string>("name");
  const [taskSortDir, setTaskSortDir] = useState<SortDirection>("asc");

  const [jobCelFilter, setJobCelFilter] = useState<string>("");
  const [jobSortField, setJobSortField] = useState<string>("created_at");
  const [jobSortDir, setJobSortDir] = useState<SortDirection>("desc");

  const [dbCelFilter, setDbCelFilter] = useState<string>("");
  const [dbSortField, setDbSortField] = useState<string>("row_count");
  const [dbSortDir, setDbSortDir] = useState<SortDirection>("desc");

  const [tokenCelFilter, setTokenCelFilter] = useState<string>("");
  const [tokenSortField, setTokenSortField] = useState<string>("created_at");
  const [tokenSortDir, setTokenSortDir] = useState<SortDirection>("desc");

  const [auditCelFilter, setAuditCelFilter] = useState<string>("");
  const [auditSortField, setAuditSortField] = useState<string>("created_at");
  const [auditSortDir, setAuditSortDir] = useState<SortDirection>("desc");

  // Telemetry & Ops state
  const [systemHealth, setSystemHealth] = useState<GetSystemHealthResponse | null>(null);
  const [systemMetrics, setSystemMetrics] = useState<GetSystemMetricsResponse | null>(null);
  const [routeRequests, setRouteRequests] = useState<RouteRequestLog[]>([]);
  const [tasks, setTasks] = useState<TaskDef[]>([]);
  const [jobs, setJobs] = useState<JobRecord[]>([]);
  const [jobTaskFilter, setJobTaskFilter] = useState<string>("");
  const [selectedJob, setSelectedJob] = useState<JobRecord | null>(null);
  const [selectedTraceId, setSelectedTraceId] = useState<string>("");
  const [selectedTrace, setSelectedTrace] = useState<GetTraceResponse | null>(null);
  const [traceLoading, setTraceLoading] = useState<boolean>(false);
  const [traceError, setTraceError] = useState<string>("");
  const [dbStats, setDbStats] = useState<GetDatabaseStatsResponse | null>(null);
  const [iamUsers, setIamUsers] = useState<IamUserSummary[]>([]);
  const [iamGroups, setIamGroups] = useState<IamGroupSummary[]>([]);
  const [tokens, setTokens] = useState<InternalTokenRecord[]>([]);
  const [auditCalls, setAuditCalls] = useState<InternalApiCallRecord[]>([]);

  // Action feedback & forms
  const [bannerMessage, setBannerMessage] = useState<string>("");
  const [bannerError, setBannerError] = useState<string>("");
  const [jitReason, setJitReason] = useState<string>("Routine ops diagnostics & maintenance");
  const [jitHours, setJitHours] = useState<number>(2);
  const [roleTargetUser, setRoleTargetUser] = useState<string>("");
  const [roleTargetValue, setRoleTargetValue] = useState<string>("ADMIN");
  const [newTokenName, setNewTokenName] = useState<string>("");
  const [newTokenGroup, setNewTokenGroup] = useState<string>("operators");
  const [newTokenType, setNewTokenType] = useState<string>("HUMAN");
  const [createdTokenSecret, setCreatedTokenSecret] = useState<string>("");

  const client = useMemo(() => {
    const authInterceptor: Interceptor = (next) => async (req) => {
      if (token) {
        req.header.set("Authorization", `Bearer ${token}`);
      }
      return next(req);
    };
    const transport = createConnectTransport({
      baseUrl: getOpsApiBaseUrl(),
      interceptors: [authInterceptor],
    });
    return createClient(OpsService, transport);
  }, [token]);

  useEffect(() => {
    setOpsCookieAndStorage(token || null);
  }, [token]);

  // Load OAuth config & handle OAuth callback code on mount
  useEffect(() => {
    let cancelled = false;
    async function initOAuth() {
      try {
        const cfg = await client.getOAuthConfig({});
        if (!cancelled) {
          setOauthConfig(cfg);
        }
      } catch {
        // Ignore if server-ops-api is not reachable yet in unit tests
      }

      if (typeof window !== "undefined") {
        const params = new URLSearchParams(window.location.search);
        const code = params.get("code");
        if (code) {
          try {
            const exchanged = await client.exchangeOAuthCode({
              code,
              redirectUri: window.location.origin,
            });
            if (!cancelled && exchanged.token) {
              setOpsCookieAndStorage(exchanged.token);
              setToken(exchanged.token);
              window.history.replaceState({}, document.title, window.location.pathname);
            }
          } catch (err) {
            if (!cancelled) {
              setAuthError(err instanceof Error ? err.message : "OAuth code exchange failed");
            }
          }
        }
      }
    }
    void initOAuth();
    return () => {
      cancelled = true;
    };
  }, [client]);

  const refreshDashboard = useCallback(async () => {
    if (!token) {
      setOperatorInfo(null);
      return;
    }
    setLoading(true);
    setAuthError("");
    try {
      const op = await client.getCurrentOperator({});
      setOperatorInfo(op);

      const reqRegion = regionFilter === "all" ? undefined : regionFilter;
      const reqService = serviceFilter === "all" ? undefined : serviceFilter;

      const [
        healthResp,
        metricsResp,
        requestsResp,
        tasksResp,
        jobsResp,
        dbResp,
        usersResp,
        groupsResp,
        tokensResp,
        auditResp,
      ] = await Promise.all([
        client.getSystemHealth({ regionFilter: reqRegion }),
        client.getSystemMetrics({
          regionFilter: reqRegion,
          serviceFilter: reqService,
          window: metricsWindow,
        }),
        client.listRouteRequests({
          regionFilter: reqRegion,
          serviceName: reqService,
          window: metricsWindow,
          limit: 50,
        }),
        client.listTasks({}),
        client.listJobs({
          limit: 50,
          taskName: jobTaskFilter || undefined,
        }),
        client.getDatabaseStats({}),
        client.listIamUsers({}),
        client.listIamGroups({}),
        client.listInternalTokens({}),
        client.listInternalApiCalls({ limit: 50 }),
      ]);

      setSystemHealth(healthResp);
      setSystemMetrics(metricsResp);
      setRouteRequests(requestsResp.requests ?? []);
      setTasks(tasksResp.tasks ?? []);
      setJobs(jobsResp.jobs ?? []);
      setDbStats(dbResp);
      setIamUsers(usersResp.users ?? []);
      setIamGroups(groupsResp.groups ?? []);
      setTokens(tokensResp.tokens ?? []);
      setAuditCalls(auditResp.calls ?? []);
    } catch (err) {
      setAuthError(err instanceof Error ? err.message : "Failed to authenticate with Ops API");
      setOperatorInfo(null);
    } finally {
      setLoading(false);
    }
  }, [client, token, regionFilter, serviceFilter, metricsWindow, jobTaskFilter]);

  useEffect(() => {
    void refreshDashboard();
  }, [refreshDashboard]);

  const activeJitGrant = operatorInfo?.activeJitGrant;
  const hasActiveJit = Boolean(activeJitGrant && activeJitGrant.status === "ACTIVE");

  const handleSandboxLogin = () => {
    setOpsCookieAndStorage(SANDBOX_DEFAULT_OPERATOR_TOKEN);
    setToken(SANDBOX_DEFAULT_OPERATOR_TOKEN);
  };

  const handleManualTokenSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (!manualTokenInput.trim()) return;
    setOpsCookieAndStorage(manualTokenInput.trim());
    setToken(manualTokenInput.trim());
    setManualTokenInput("");
  };

  const handleSignOut = () => {
    setOpsCookieAndStorage(null);
    setToken("");
    setOperatorInfo(null);
    setBannerMessage("");
    setBannerError("");
  };

  const handleAcquireJit = async () => {
    setBannerError("");
    setBannerMessage("");
    try {
      const resp = await client.acquireJitGrant({
        hours: jitHours,
        reason: jitReason.trim() || "Operational maintenance & diagnostics",
        operation: "*",
      });
      setBannerMessage(
        `JIT Elevation Lease acquired (${resp.grant?.id ?? "active"} for ${jitHours}h).`,
      );
      await refreshDashboard();
    } catch (err) {
      setBannerError(err instanceof Error ? err.message : "Failed to acquire JIT lease");
    }
  };

  const handleExtendJit = async () => {
    if (!activeJitGrant) return;
    setBannerError("");
    setBannerMessage("");
    try {
      await client.extendJitGrant({
        grantId: activeJitGrant.id,
        hours: 2,
      });
      setBannerMessage(`Extended JIT lease ${activeJitGrant.id} by +2 hours.`);
      await refreshDashboard();
    } catch (err) {
      setBannerError(err instanceof Error ? err.message : "Failed to extend JIT lease");
    }
  };

  const handleRevokeJit = async () => {
    if (!activeJitGrant) return;
    setBannerError("");
    setBannerMessage("");
    try {
      await client.revokeJitGrant({
        grantId: activeJitGrant.id,
      });
      setBannerMessage(`Revoked JIT lease ${activeJitGrant.id}.`);
      await refreshDashboard();
    } catch (err) {
      setBannerError(err instanceof Error ? err.message : "Failed to revoke JIT lease");
    }
  };

  const handleLoginAsQuadsmith = async () => {
    setBannerError("");
    setBannerMessage("");
    try {
      const resp = await client.loginAsQuadsmith({});
      let baseUrl = "http://localhost:8080";
      if (typeof window !== "undefined") {
        const { protocol, hostname, port } = window.location;
        if (port === "8083") {
          baseUrl = `${protocol}//${hostname}:8080`;
        } else if (hostname.startsWith("ops.")) {
          const mainHost = hostname.slice(4);
          baseUrl = `${protocol}//${mainHost}${port ? `:${port}` : ""}`;
        } else {
          baseUrl = `${protocol}//${hostname}:8080`;
        }
      }
      const targetUrl = `${baseUrl}/?ops_refresh_token=${encodeURIComponent(resp.refreshToken)}`;
      if (typeof window !== "undefined" && typeof window.open === "function") {
        window.open(targetUrl, "_blank", "noopener,noreferrer");
      }
      setBannerMessage(
        `Authenticated as @${resp.username} (${resp.role}). Opened Quadsmith Web SPA in a new tab.`,
      );
      await refreshDashboard();
    } catch (err) {
      setBannerError(err instanceof Error ? err.message : "Failed to log in as @quadsmith");
    }
  };

  const handleTriggerTask = async (taskName: string) => {
    setBannerError("");
    setBannerMessage("");
    try {
      switch (taskName) {
        case "cleanup_expired_otps": {
          const r = await client.cleanupExpiredOtps({});
          setBannerMessage(
            `Triggered cleanup_expired_otps: cleaned ${r.cleanedCount.toString()} expired OTPs.`,
          );
          break;
        }
        case "cleanup_expired_tokens": {
          const r = await client.cleanupExpiredTokens({});
          setBannerMessage(
            `Triggered cleanup_expired_tokens: cleaned ${r.cleanedCount.toString()} expired tokens.`,
          );
          break;
        }
        case "cleanup_orphaned_blobs": {
          const r = await client.cleanupOrphanedBlobs({});
          setBannerMessage(
            `Triggered cleanup_orphaned_blobs: cleaned ${r.cleanedCount.toString()} orphaned blobs.`,
          );
          break;
        }
        case "purge_soft_deleted_accounts": {
          const r = await client.purgeSoftDeletedAccounts({});
          setBannerMessage(
            `Triggered purge_soft_deleted_accounts: purged ${r.purgedCount.toString()} accounts.`,
          );
          break;
        }
        case "send_username_release_warnings": {
          const r = await client.sendUsernameReleaseWarnings({});
          setBannerMessage(
            `Triggered send_username_release_warnings: sent ${r.warningCount.toString()} warnings.`,
          );
          break;
        }
        case "set_user_role": {
          if (!roleTargetUser.trim()) {
            setBannerError("Target username or UUID is required for set_user_role.");
            return;
          }
          const r = await client.setUserRole({
            userId: roleTargetUser.trim(),
            role: roleTargetValue,
          });
          setBannerMessage(`Updated @${r.username} role from ${r.previousRole} to ${r.role}.`);
          break;
        }
        default:
          setBannerError(`Unknown task: ${taskName}`);
          return;
      }
      await refreshDashboard();
    } catch (err) {
      setBannerError(err instanceof Error ? err.message : `Failed to trigger ${taskName}`);
    }
  };

  const handleCreateToken = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!newTokenName.trim()) return;
    setBannerError("");
    setBannerMessage("");
    try {
      const resp = await client.createInternalToken({
        name: newTokenName.trim(),
        groupName: newTokenGroup,
        type: newTokenType,
        durationDays: 30,
      });
      setCreatedTokenSecret(resp.tokenSecret);
      setNewTokenName("");
      setBannerMessage(`Issued new ${newTokenType} token ${resp.tokenId}.`);
      await refreshDashboard();
    } catch (err) {
      setBannerError(err instanceof Error ? err.message : "Failed to create internal token");
    }
  };

  const handleRevokeToken = async (id: string) => {
    setBannerError("");
    setBannerMessage("");
    try {
      await client.revokeInternalToken({ id });
      setBannerMessage(`Revoked token ${id}.`);
      await refreshDashboard();
    } catch (err) {
      setBannerError(err instanceof Error ? err.message : "Failed to revoke token");
    }
  };

  const buildFallbackTrace = useCallback(
    (traceId: string): GetTraceResponse => {
      const spans: TraceSpan[] = [];
      const logs: TraceLogRecord[] = [];
      let totalRttMs = 0;
      let success = true;
      let rootService = "";
      let rootRoute = "";
      let rootMethod = "POST";
      let statusCode = 200;
      let region = "local";

      for (const req of routeRequests) {
        if (req.traceId === traceId) {
          region = req.region || region;
          if (!rootService) {
            rootService = req.serviceName;
            rootRoute = req.route;
            rootMethod = req.method;
            statusCode = req.statusCode;
          }
          if (req.rttMs > totalRttMs) {
            totalRttMs = req.rttMs;
          }
          if (!req.success) {
            success = false;
            statusCode = req.statusCode;
          }
          const hopType = spans.length === 0 ? "ingress" : "internal_hop";
          spans.push({
            $typeName: "quadsmith.ops.TraceSpan",
            spanId: `span_${req.serviceName}_${spans.length + 1}`,
            serviceName: req.serviceName,
            instanceId: req.instanceId || `${req.serviceName}-1`,
            region: req.region || "local",
            operation: req.route,
            method: req.method,
            statusCode: req.statusCode,
            success: req.success,
            durationMs: req.rttMs,
            occurredAt: req.occurredAt,
            attributes: {
              hop_type: hopType,
              route: req.route,
              method: req.method,
              status_code: String(req.statusCode),
              rtt_ms: req.rttMs.toFixed(2),
            },
          });
          logs.push({
            $typeName: "quadsmith.ops.TraceLogRecord",
            timestamp: req.occurredAt,
            level: "DEBUG",
            serviceName: req.serviceName,
            instanceId: req.instanceId || `${req.serviceName}-1`,
            region: req.region || "local",
            message: "trace_context_initialized",
            attributes: {
              route: req.route,
              method: req.method,
              hop_type: hopType,
            },
          });
          logs.push({
            $typeName: "quadsmith.ops.TraceLogRecord",
            timestamp: req.occurredAt,
            level: req.success ? "INFO" : "ERROR",
            serviceName: req.serviceName,
            instanceId: req.instanceId || `${req.serviceName}-1`,
            region: req.region || "local",
            message: "http_request",
            attributes: {
              route: req.route,
              method: req.method,
              status_code: String(req.statusCode),
              rtt_ms: req.rttMs.toFixed(2),
            },
          });
          logs.push({
            $typeName: "quadsmith.ops.TraceLogRecord",
            timestamp: req.occurredAt,
            level: "DEBUG",
            serviceName: req.serviceName,
            instanceId: req.instanceId || `${req.serviceName}-1`,
            region: req.region || "local",
            message: "response_flushed",
            attributes: {
              route: req.route,
              method: req.method,
              hop_type: hopType,
              status_code: String(req.statusCode),
              rtt_ms: req.rttMs.toFixed(2),
            },
          });
        }
      }

      for (const call of auditCalls) {
        if (call.traceId === traceId) {
          region = call.region || region;
          if (!rootService) {
            rootService = "server-ops-api";
            rootRoute = call.method;
          }
          const rtt = Number(call.durationMs);
          if (rtt > totalRttMs) {
            totalRttMs = rtt;
          }
          const ok = call.status === "OK";
          if (!ok) {
            success = false;
            statusCode = 500;
          }
          const attrs: { [key: string]: string } = {
            hop_type: "audit",
            route: call.method,
            caller: call.caller,
            caller_type: call.callerType,
            group_name: call.groupName,
            status: call.status,
            duration_ms: String(rtt),
          };
          if (call.jitGrantId) {
            attrs.jit_grant_id = call.jitGrantId;
          }
          spans.push({
            $typeName: "quadsmith.ops.TraceSpan",
            spanId: `span_audit_${spans.length + 1}`,
            serviceName: "server-ops-api",
            instanceId: "server-ops-api-1",
            region: call.region || "local",
            operation: call.method,
            method: "POST",
            statusCode: ok ? 200 : 500,
            success: ok,
            durationMs: rtt,
            occurredAt: call.createdAt,
            attributes: attrs,
          });
          logs.push({
            $typeName: "quadsmith.ops.TraceLogRecord",
            timestamp: call.createdAt,
            level: "DEBUG",
            serviceName: "server-ops-api",
            instanceId: "server-ops-api-1",
            region: call.region || "local",
            message: "ops_auth_verified",
            attributes: {
              route: call.method,
              operator: call.caller,
              operator_type: call.callerType,
              group_name: call.groupName,
            },
          });
          logs.push({
            $typeName: "quadsmith.ops.TraceLogRecord",
            timestamp: call.createdAt,
            level: ok ? "INFO" : "ERROR",
            serviceName: "server-ops-api",
            instanceId: "server-ops-api-1",
            region: call.region || "local",
            message: "ops_rpc_audit",
            attributes: attrs,
          });
        }
      }

      for (const job of jobs) {
        if (job.traceId === traceId) {
          region = job.region || region;
          const rtt = Number(job.durationMs);
          if (rtt > totalRttMs) {
            totalRttMs = rtt;
          }
          const ok = job.status === "OK";
          if (!ok) {
            success = false;
            statusCode = 500;
          }
          if (!rootService) {
            rootService = "server-ops-api";
            rootRoute = job.procedure;
          }
          logs.push({
            $typeName: "quadsmith.ops.TraceLogRecord",
            timestamp: job.createdAt,
            level: ok ? "INFO" : "ERROR",
            serviceName: "server-ops-api",
            instanceId: "server-ops-api-1",
            region: job.region || "local",
            message: "job_execution",
            attributes: {
              job_id: job.jobId,
              task_name: job.taskName,
              route: job.procedure,
              caller: job.caller,
              caller_type: job.callerType,
              status: job.status,
              rtt_ms: String(rtt),
            },
          });
        }
      }

      return {
        $typeName: "quadsmith.ops.GetTraceResponse",
        traceId,
        region,
        rootService: rootService || "server-ops-api",
        rootRoute: rootRoute || "/",
        rootMethod,
        statusCode,
        totalRttMs,
        success,
        startedAt: spans[0]?.occurredAt ?? logs[0]?.timestamp,
        spans,
        logs,
      };
    },
    [routeRequests, auditCalls, jobs],
  );

  const handleSelectRouteFromMetrics = useCallback((r: RouteMetricStat) => {
    setTraceCelFilter((prev) => applyRouteToTraceCelFilter(prev, r.route, r.serviceName));
    setTelemetrySubTab("traces");
  }, []);

  const handleInspectTrace = useCallback(
    async (rawTraceId: string) => {
      const cleanId = rawTraceId.trim();
      if (!cleanId || cleanId === "—") return;
      setSelectedTraceId(cleanId);
      setTraceLoading(true);
      setTraceError("");
      try {
        const resp = await client.getTrace({ traceId: cleanId });
        if ((resp.spans?.length ?? 0) > 0 || (resp.logs?.length ?? 0) > 0) {
          setSelectedTrace(resp);
        } else {
          setSelectedTrace(buildFallbackTrace(cleanId));
        }
      } catch (err) {
        const fallback = buildFallbackTrace(cleanId);
        setSelectedTrace(fallback);
        if (fallback.spans.length === 0 && fallback.logs.length === 0) {
          setTraceError(err instanceof Error ? err.message : "Failed to load trace details");
        }
      } finally {
        setTraceLoading(false);
      }
    },
    [client, buildFallbackTrace],
  );

  const allRoutes = useMemo(() => {
    const list: RouteMetricStat[] = [];
    for (const svc of systemMetrics?.services ?? []) {
      for (const r of svc.routes ?? []) {
        list.push(r);
      }
    }
    return list;
  }, [systemMetrics]);

  const routeEvalResult = useMemo(
    () =>
      filterAndSortRecords(
        allRoutes,
        (r: RouteMetricStat) => ({
          service_name: r.serviceName,
          region: r.region,
          method: r.method,
          route: r.route,
          total_requests: Number(r.totalRequests),
          success_count: Number(r.successCount),
          error_count: Number(r.errorCount),
          availability_percent: r.availabilityPercent,
          p50_rtt_ms: r.p50RttMs,
          p95_rtt_ms: r.p95RttMs,
          p99_rtt_ms: r.p99RttMs,
        }),
        routeCelFilter,
        buildSortSpec(routeSortField, routeSortDir),
        ROUTE_METRIC_FIELDS,
      ),
    [allRoutes, routeCelFilter, routeSortField, routeSortDir],
  );

  const traceEvalResult = useMemo(
    () =>
      filterAndSortRecords(
        routeRequests,
        (req: RouteRequestLog) => ({
          trace_id: req.traceId,
          region: req.region,
          service_name: req.serviceName,
          method: req.method,
          route: req.route,
          status_code: req.statusCode,
          rtt_ms: req.rttMs,
          success: req.success,
          occurred_at: formatIsoTimestamp(req.occurredAt),
        }),
        traceCelFilter,
        buildSortSpec(traceSortField, traceSortDir),
        TRACE_REQUEST_FIELDS,
      ),
    [routeRequests, traceCelFilter, traceSortField, traceSortDir],
  );

  const taskEvalResult = useMemo(
    () =>
      filterAndSortRecords(
        tasks,
        (t: TaskDef) => ({
          name: t.name,
          description: t.description,
          procedure: t.procedure,
          default_interval: t.defaultInterval,
        }),
        taskCelFilter,
        buildSortSpec(taskSortField, taskSortDir),
        TASK_FIELDS,
      ),
    [tasks, taskCelFilter, taskSortField, taskSortDir],
  );

  const jobEvalResult = useMemo(
    () =>
      filterAndSortRecords(
        jobs,
        (j: JobRecord) => ({
          job_id: j.jobId,
          trace_id: j.traceId,
          region: j.region,
          task_name: j.taskName,
          procedure: j.procedure,
          caller: j.caller,
          caller_type: j.callerType,
          status: j.status,
          duration_ms: Number(j.durationMs),
          error_message: j.errorMessage,
          created_at: formatIsoTimestamp(j.createdAt),
        }),
        jobCelFilter,
        buildSortSpec(jobSortField, jobSortDir),
        JOB_FIELDS,
      ),
    [jobs, jobCelFilter, jobSortField, jobSortDir],
  );

  const dbTableList: TableStat[] = useMemo(() => dbStats?.tables ?? [], [dbStats]);
  const dbEvalResult = useMemo(
    () =>
      filterAndSortRecords(
        dbTableList,
        (t: TableStat) => ({
          table_name: t.tableName,
          row_count: Number(t.rowCount),
          total_size_bytes: Number(t.totalSizeBytes),
        }),
        dbCelFilter,
        buildSortSpec(dbSortField, dbSortDir),
        DB_TABLE_FIELDS,
      ),
    [dbTableList, dbCelFilter, dbSortField, dbSortDir],
  );

  const tokenEvalResult = useMemo(
    () =>
      filterAndSortRecords(
        tokens,
        (tok: InternalTokenRecord) => ({
          id: tok.id,
          name: tok.name,
          type: tok.type,
          group_name: tok.groupName,
          status: tok.revokedAt ? "REVOKED" : "ACTIVE",
          created_at: formatIsoTimestamp(tok.createdAt),
          expires_at: formatIsoTimestamp(tok.expiresAt),
        }),
        tokenCelFilter,
        buildSortSpec(tokenSortField, tokenSortDir),
        TOKEN_FIELDS,
      ),
    [tokens, tokenCelFilter, tokenSortField, tokenSortDir],
  );

  const auditEvalResult = useMemo(
    () =>
      filterAndSortRecords(
        auditCalls,
        (c: InternalApiCallRecord) => ({
          trace_id: c.traceId,
          region: c.region,
          caller: c.caller,
          caller_type: c.callerType,
          group_name: c.groupName,
          method: c.method,
          status: c.status,
          duration_ms: Number(c.durationMs),
          jit_grant_id: c.jitGrantId,
          created_at: formatIsoTimestamp(c.createdAt),
        }),
        auditCelFilter,
        buildSortSpec(auditSortField, auditSortDir),
        AUDIT_CALL_FIELDS,
      ),
    [auditCalls, auditCelFilter, auditSortField, auditSortDir],
  );

  const fleetTotals = useMemo(() => {
    let totalReqs = 0;
    let totalOk = 0;
    let totalErr = 0;
    let maxP95 = 0;
    for (const svc of systemMetrics?.services ?? []) {
      totalReqs += Number(svc.totalRequests);
      totalOk += Number(svc.successCount);
      totalErr += Number(svc.errorCount);
      if (svc.p95RttMs > maxP95) {
        maxP95 = svc.p95RttMs;
      }
    }
    const avail = totalReqs > 0 ? ((totalOk / totalReqs) * 100).toFixed(2) : "100.00";
    return { totalReqs, totalErr, avail, maxP95 };
  }, [systemMetrics]);

  // Unauthenticated view
  if (!token || !operatorInfo) {
    const envName = oauthConfig?.environment || "sandbox";
    const regionName = oauthConfig?.region || "local";

    return (
      <div className="min-h-screen bg-[#090d16] text-slate-100 flex flex-col justify-center items-center p-6">
        <div className="w-full max-w-md bg-[#111827] border border-slate-800 rounded-xl p-6 shadow-2xl space-y-6">
          <div className="flex items-center justify-between border-b border-slate-800 pb-4">
            <div>
              <div className="text-xs font-mono uppercase tracking-widest text-sky-400">
                QUADSMITH // INTERNAL OPS
              </div>
              <h1 className="text-xl font-bold text-slate-100 mt-1">
                Ops &amp; Health Monitoring Console
              </h1>
            </div>
            <span className="px-2.5 py-1 text-xs font-mono rounded bg-slate-800 text-slate-300 border border-slate-700">
              {envName} · {regionName}
            </span>
          </div>

          {authError && (
            <div
              role="alert"
              className="p-3 rounded-lg bg-rose-950/60 border border-rose-800 text-rose-200 text-xs font-mono"
            >
              {authError}
            </div>
          )}

          <div className="space-y-3">
            <a
              href={oauthConfig?.authorizeUrl || "https://github.com/login/oauth/authorize"}
              className="w-full flex items-center justify-center gap-2 py-2.5 px-4 rounded-lg bg-sky-600 hover:bg-sky-500 text-white font-semibold text-sm transition"
            >
              Sign in with GitHub OAuth
            </a>

            {envName === "sandbox" && (
              <button
                type="button"
                data-testid="sandbox-login-btn"
                onClick={handleSandboxLogin}
                className="w-full py-2.5 px-4 rounded-lg bg-emerald-600/20 hover:bg-emerald-600/30 text-emerald-300 border border-emerald-500/40 font-mono text-xs transition cursor-pointer"
              >
                ⚡ Authenticate as Sandbox Operator (rohitramu)
              </button>
            )}
          </div>

          <form
            onSubmit={handleManualTokenSubmit}
            className="space-y-3 pt-3 border-t border-slate-800"
          >
            <label className="block text-xs font-mono text-slate-400">
              OR AUTHENTICATE WITH OPERATOR / ROBOT TOKEN
            </label>
            <div className="flex gap-2">
              <input
                type="password"
                data-testid="token-input"
                value={manualTokenInput}
                onChange={(e) => setManualTokenInput(e.target.value)}
                placeholder="qs_op_..."
                className="flex-1 bg-[#090d16] border border-slate-700 rounded-lg px-3 py-2 text-xs font-mono text-slate-100 focus:outline-none focus:border-sky-500"
              />
              <button
                type="submit"
                data-testid="token-submit-btn"
                className="px-4 py-2 rounded-lg bg-slate-800 hover:bg-slate-700 text-slate-200 text-xs font-mono border border-slate-700 cursor-pointer"
              >
                Connect
              </button>
            </div>
          </form>
        </div>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-[#090d16] text-slate-100 flex flex-col">
      {/* Top Header */}
      <header className="bg-[#111827] border-b border-slate-800 px-6 py-3.5 flex flex-wrap items-center justify-between gap-4">
        <div className="flex items-center gap-4">
          <div className="flex items-center gap-2.5">
            <span className="h-2.5 w-2.5 rounded-full bg-emerald-400 animate-pulse" />
            <span className="font-mono font-bold tracking-wider text-sm text-slate-100">
              QUADSMITH // OPS CONSOLE
            </span>
          </div>
          <span className="px-2 py-0.5 text-xs font-mono rounded bg-slate-800 text-sky-300 border border-slate-700">
            env:{systemHealth?.environment || oauthConfig?.environment || "sandbox"}
          </span>
          <div className="flex items-center gap-1.5 text-xs font-mono text-slate-400">
            <span>Region:</span>
            <select
              aria-label="Deployment Region Filter"
              data-testid="region-filter-select"
              value={regionFilter}
              onChange={(e) => setRegionFilter(e.target.value)}
              className="bg-[#090d16] border border-slate-700 rounded px-2 py-1 text-xs font-mono text-slate-200"
            >
              <option value="all">All Regions ({systemHealth?.region || "local"})</option>
              <option value="local">local</option>
              <option value="us-central1">us-central1</option>
            </select>
          </div>
        </div>

        <div className="flex items-center gap-3">
          <button
            type="button"
            onClick={() => void refreshDashboard()}
            disabled={loading}
            className="px-3 py-1.5 rounded bg-slate-800 hover:bg-slate-700 text-slate-200 border border-slate-700 text-xs font-mono cursor-pointer"
          >
            {loading ? "Refreshing..." : "↻ Refresh"}
          </button>
          <span
            data-testid="operator-badge"
            className="px-2.5 py-1 rounded bg-slate-900 border border-slate-700 text-xs font-mono text-slate-300"
          >
            {operatorInfo.operator} ({operatorInfo.groupName})
          </span>
          <button
            type="button"
            onClick={handleSignOut}
            className="px-2.5 py-1 rounded bg-rose-950/60 hover:bg-rose-900/60 text-rose-300 border border-rose-800/60 text-xs font-mono cursor-pointer"
          >
            Sign Out
          </button>
        </div>
      </header>

      {/* JIT Elevation & Grafana Gate Bar */}
      <section className="bg-[#0d1424] border-b border-slate-800 px-6 py-3 flex flex-wrap items-center justify-between gap-4">
        <div className="flex flex-wrap items-center gap-3">
          <span
            data-testid="jit-status-badge"
            className={`px-2.5 py-1 rounded text-xs font-mono font-semibold border ${
              hasActiveJit
                ? "bg-amber-500/15 text-amber-300 border-amber-500/40"
                : "bg-slate-800 text-slate-400 border-slate-700"
            }`}
          >
            {hasActiveJit
              ? `⚡ JIT ELEVATED · ${activeJitGrant?.id} (expires ${formatTimestamp(activeJitGrant?.expiresAt)})`
              : "🔒 JIT LOCKED · READ-ONLY DIAGNOSTICS"}
          </span>

          {!hasActiveJit ? (
            <div className="flex flex-wrap items-center gap-2">
              <input
                type="text"
                data-testid="jit-reason-input"
                value={jitReason}
                onChange={(e) => setJitReason(e.target.value)}
                placeholder="Reason for JIT elevation..."
                className="bg-[#090d16] border border-slate-700 rounded px-2.5 py-1 text-xs font-mono text-slate-200 w-64"
              />
              <select
                aria-label="JIT Lease Hours"
                data-testid="jit-hours-select"
                value={jitHours}
                onChange={(e) => setJitHours(Number(e.target.value))}
                className="bg-[#090d16] border border-slate-700 rounded px-2 py-1 text-xs font-mono text-slate-200"
              >
                <option value={1}>1h</option>
                <option value={2}>2h</option>
                <option value={4}>4h</option>
                <option value={8}>8h</option>
                <option value={12}>12h</option>
              </select>
              <button
                type="button"
                data-testid="acquire-jit-btn"
                onClick={() => void handleAcquireJit()}
                className="px-3 py-1 rounded bg-amber-500/20 hover:bg-amber-500/30 text-amber-200 border border-amber-500/40 text-xs font-mono font-semibold cursor-pointer"
              >
                Acquire JIT Lease
              </button>
            </div>
          ) : (
            <div className="flex items-center gap-2">
              <button
                type="button"
                data-testid="extend-jit-btn"
                onClick={() => void handleExtendJit()}
                className="px-2.5 py-1 rounded bg-slate-800 hover:bg-slate-700 text-slate-200 border border-slate-700 text-xs font-mono cursor-pointer"
              >
                +2h Extend
              </button>
              <button
                type="button"
                data-testid="revoke-jit-btn"
                onClick={() => void handleRevokeJit()}
                className="px-2.5 py-1 rounded bg-rose-950/60 hover:bg-rose-900/60 text-rose-300 border border-rose-800/60 text-xs font-mono cursor-pointer"
              >
                Revoke Lease
              </button>
            </div>
          )}
        </div>

        {/* JIT-gated @quadsmith Impersonation & Grafana Deep-Dive Links */}
        <div className="flex items-center gap-2">
          {hasActiveJit ? (
            <>
              <button
                type="button"
                data-testid="login-quadsmith-btn"
                onClick={() => void handleLoginAsQuadsmith()}
                className="px-3 py-1.5 rounded bg-amber-500/20 hover:bg-amber-500/30 text-amber-200 border border-amber-500/40 text-xs font-mono font-semibold transition cursor-pointer"
              >
                🚀 Log in as @quadsmith
              </button>
              <a
                href="/grafana/d/quadsmith-health/quadsmith-service-health"
                target="_blank"
                rel="noreferrer"
                data-testid="open-grafana-btn"
                className="px-3 py-1.5 rounded bg-sky-600/25 hover:bg-sky-600/35 text-sky-200 border border-sky-500/40 text-xs font-mono font-semibold transition"
              >
                📊 Open Grafana Dashboard (/grafana/)
              </a>
            </>
          ) : (
            <>
              <span
                data-testid="login-quadsmith-btn"
                title="Acquire an active JIT lease to log in as @quadsmith"
                className="px-3 py-1.5 rounded bg-slate-900 text-slate-500 border border-slate-800 text-xs font-mono cursor-not-allowed"
              >
                🔒 Log in as @quadsmith (Requires JIT)
              </span>
              <span
                data-testid="open-grafana-btn"
                title="Acquire an active JIT lease to access /grafana/*"
                className="px-3 py-1.5 rounded bg-slate-900 text-slate-500 border border-slate-800 text-xs font-mono cursor-not-allowed"
              >
                🔒 Grafana Dashboard (Requires JIT)
              </span>
            </>
          )}
        </div>
      </section>

      {/* Feedback Banners */}
      {(bannerMessage || bannerError) && (
        <div className="px-6 pt-4">
          {bannerMessage && (
            <div
              data-testid="ops-banner-success"
              className="p-3 rounded-lg bg-emerald-950/60 border border-emerald-700/60 text-emerald-200 text-xs font-mono"
            >
              {bannerMessage}
            </div>
          )}
          {bannerError && (
            <div
              data-testid="ops-banner-error"
              className="p-3 rounded-lg bg-rose-950/60 border border-rose-700/60 text-rose-200 text-xs font-mono"
            >
              {bannerError}
            </div>
          )}
        </div>
      )}

      {/* Navigation Tabs */}
      <nav className="px-6 pt-4 border-b border-slate-800 flex flex-wrap gap-2">
        {(
          [
            ["health", "1. Service Health & RED Metrics"],
            ["tasks", "2. Background Tasks & Triggers"],
            ["jobs", "3. Job Execution History"],
            ["db", "4. Public DB Statistics"],
            ["iam", "5. IAM, Tokens & Audit Log"],
          ] as const
        ).map(([id, label]) => (
          <button
            key={id}
            type="button"
            data-testid={`tab-${id}`}
            onClick={() => setActiveTab(id)}
            className={`px-4 py-2 text-xs font-mono rounded-t-lg border-t border-x transition cursor-pointer ${
              activeTab === id
                ? "bg-[#111827] text-sky-300 border-slate-700 font-semibold"
                : "bg-transparent text-slate-400 border-transparent hover:text-slate-200"
            }`}
          >
            {label}
          </button>
        ))}
      </nav>

      {/* Main Content Area */}
      <main className="flex-1 p-6 space-y-6 max-w-7xl w-full mx-auto">
        {activeTab === "health" && (
          <div className="space-y-6">
            {/* Fleet KPI Summary */}
            <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
              <div className="bg-[#111827] border border-slate-800 rounded-xl p-4">
                <div className="text-xs font-mono text-slate-400">OVERALL SYSTEM HEALTH</div>
                <div
                  data-testid="overall-health-status"
                  className={`text-xl font-bold font-mono mt-1 ${
                    systemHealth?.status === "HEALTHY"
                      ? "text-emerald-400"
                      : systemHealth?.status === "DEGRADED"
                        ? "text-amber-400"
                        : "text-rose-400"
                  }`}
                >
                  {systemHealth?.status || "HEALTHY"}
                </div>
                <div className="text-xs font-mono text-slate-500 mt-1">
                  {(systemHealth?.services ?? []).length} services monitored in{" "}
                  {systemHealth?.region || "local"}
                </div>
              </div>

              <div className="bg-[#111827] border border-slate-800 rounded-xl p-4">
                <div className="text-xs font-mono text-slate-400">
                  FLEET AVAILABILITY ({metricsWindow})
                </div>
                <div className="text-xl font-bold font-mono text-sky-300 mt-1">
                  {fleetTotals.avail}%
                </div>
                <div className="text-xs font-mono text-slate-500 mt-1">
                  {fleetTotals.totalReqs} reqs · {fleetTotals.totalErr} errors
                </div>
              </div>

              <div className="bg-[#111827] border border-slate-800 rounded-xl p-4">
                <div className="text-xs font-mono text-slate-400">
                  PEAK P95 RTT ({metricsWindow})
                </div>
                <div className="text-xl font-bold font-mono text-slate-100 mt-1">
                  {fleetTotals.maxP95.toFixed(2)} ms
                </div>
                <div className="text-xs font-mono text-slate-500 mt-1">
                  Across all instrumented HTTP/RPC routes
                </div>
              </div>

              <div className="bg-[#111827] border border-slate-800 rounded-xl p-4">
                <div className="text-xs font-mono text-slate-400">TRACE CORRELATION</div>
                <div className="text-xl font-bold font-mono text-emerald-300 mt-1">
                  X-Trace-Id (UUIDv7)
                </div>
                <div className="text-xs font-mono text-slate-500 mt-1">
                  Ingress-minted · Client headers stripped
                </div>
              </div>
            </div>

            {/* 5-Service Deep Health Matrix */}
            <div className="bg-[#111827] border border-slate-800 rounded-xl p-5 space-y-4">
              <div className="flex items-center justify-between">
                <h2 className="text-sm font-mono font-bold uppercase tracking-wider text-slate-200">
                  5-Service Deep Health Matrix (/healthz &amp; /metrics)
                </h2>
                <span className="text-xs font-mono text-slate-400">
                  Checked: {formatTimestamp(systemHealth?.checkedAt)}
                </span>
              </div>

              <div data-testid="service-health-matrix" className="flex flex-col gap-3">
                {(systemHealth?.services ?? []).map((svc) => (
                  <div
                    key={`${svc.serviceName}-${svc.region}`}
                    data-testid={`health-card-${svc.serviceName}`}
                    className="bg-[#090d16] border border-slate-800 rounded-lg p-4 space-y-3"
                  >
                    <div className="flex flex-wrap items-center justify-between gap-3">
                      <div className="flex flex-wrap items-center gap-3">
                        <span className="font-mono font-bold text-sm text-slate-100">
                          {svc.serviceName}
                        </span>
                        <span
                          className={`px-2 py-0.5 rounded text-[10px] font-mono font-bold ${
                            svc.status === "HEALTHY"
                              ? "bg-emerald-500/15 text-emerald-300 border border-emerald-500/30"
                              : svc.status === "DEGRADED"
                                ? "bg-amber-500/15 text-amber-300 border border-amber-500/30"
                                : "bg-rose-500/15 text-rose-300 border border-rose-500/30"
                          }`}
                        >
                          {svc.status}
                        </span>
                        <div className="flex flex-wrap items-center gap-x-4 gap-y-1 text-[11px] font-mono text-slate-400">
                          <span>Region: {svc.region}</span>
                          <span>Instance: {svc.instanceId}</span>
                          <span>
                            Uptime: {svc.uptimeSeconds.toString()}s · Goroutines:{" "}
                            {svc.numGoroutines}
                          </span>
                          {svc.dbMaxConns > 0 && (
                            <span>
                              DB Pool: {svc.dbAcquiredConns}/{svc.dbMaxConns} (idle{" "}
                              {svc.dbIdleConns})
                            </span>
                          )}
                        </div>
                      </div>

                      <div className="flex items-center gap-2">
                        <a
                          href={`/services/${svc.serviceName}/healthz`}
                          target="_blank"
                          rel="noreferrer"
                          data-testid={`healthz-link-${svc.serviceName}`}
                          className="inline-flex items-center gap-1 px-2.5 py-1 rounded bg-slate-800/90 hover:bg-slate-700 text-sky-300 border border-slate-700 text-[11px] font-mono transition"
                        >
                          /healthz ↗
                        </a>
                        <a
                          href={`/services/${svc.serviceName}/metrics`}
                          target="_blank"
                          rel="noreferrer"
                          data-testid={`metrics-link-${svc.serviceName}`}
                          className="inline-flex items-center gap-1 px-2.5 py-1 rounded bg-slate-800/90 hover:bg-slate-700 text-sky-300 border border-slate-700 text-[11px] font-mono transition"
                        >
                          /metrics ↗
                        </a>
                      </div>
                    </div>

                    <div className="pt-2.5 border-t border-slate-800/80 grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-x-6 gap-y-1">
                      {(svc.checks ?? []).map((chk) => (
                        <div
                          key={chk.name}
                          className="flex items-center justify-between text-[11px] font-mono"
                        >
                          <span className="text-slate-300 truncate">{chk.name}</span>
                          <span
                            className={
                              chk.status === "HEALTHY" ? "text-emerald-400" : "text-rose-400"
                            }
                          >
                            {chk.status} ({chk.latencyMs.toFixed(1)}ms)
                          </span>
                        </div>
                      ))}
                    </div>
                  </div>
                ))}
              </div>
            </div>

            {/* Route RED Metrics & Trace Explorer Filters */}
            <div className="bg-[#111827] border border-slate-800 rounded-xl p-5 space-y-4">
              <div className="flex flex-wrap items-center justify-between gap-3">
                <h2 className="text-sm font-mono font-bold uppercase tracking-wider text-slate-200">
                  Per-Route RED Metrics &amp; Request Trace Explorer
                </h2>

                <div className="flex flex-wrap items-center gap-2">
                  <select
                    aria-label="Service Filter"
                    data-testid="service-filter-select"
                    value={serviceFilter}
                    onChange={(e) => setServiceFilter(e.target.value)}
                    className="bg-[#090d16] border border-slate-700 rounded px-2.5 py-1 text-xs font-mono text-slate-200"
                  >
                    <option value="all">All Services</option>
                    <option value="server-api">server-api</option>
                    <option value="server-web">server-web</option>
                    <option value="server-ops-api">server-ops-api</option>
                    <option value="server-ops-web">server-ops-web</option>
                    <option value="server-cron">server-cron</option>
                  </select>

                  <select
                    aria-label="Metrics Window"
                    data-testid="window-filter-select"
                    value={metricsWindow}
                    onChange={(e) => setMetricsWindow(e.target.value)}
                    className="bg-[#090d16] border border-slate-700 rounded px-2.5 py-1 text-xs font-mono text-slate-200"
                  >
                    <option value="5m">Window: 5m</option>
                    <option value="15m">Window: 15m</option>
                    <option value="1h">Window: 1h</option>
                    <option value="24h">Window: 24h</option>
                  </select>
                </div>
              </div>

              {/* Sub-tab bar: 1. Per-Route RED Metrics | 2. Recent Traced Requests */}
              <div
                role="tablist"
                aria-label="Route Metrics and Traced Requests Explorer Tabs"
                className="flex flex-wrap items-center gap-2 border-b border-slate-800 pb-3"
              >
                <button
                  type="button"
                  role="tab"
                  aria-selected={telemetrySubTab === "routes"}
                  data-testid="telemetry-tab-routes"
                  onClick={() => setTelemetrySubTab("routes")}
                  className={`px-3.5 py-1.5 rounded-lg text-xs font-mono font-bold transition-colors cursor-pointer flex items-center gap-2 ${
                    telemetrySubTab === "routes"
                      ? "bg-sky-600 text-white shadow-sm"
                      : "bg-[#090d16] text-slate-400 border border-slate-800 hover:text-slate-200 hover:border-slate-700"
                  }`}
                >
                  <span>1. Per-Route RED Metrics</span>
                  <span
                    className={`px-1.5 py-0.5 rounded text-[10px] ${
                      telemetrySubTab === "routes"
                        ? "bg-sky-800/80 text-sky-100"
                        : "bg-slate-800 text-slate-300"
                    }`}
                  >
                    {routeEvalResult.items.length}
                  </span>
                </button>

                <button
                  type="button"
                  role="tab"
                  aria-selected={telemetrySubTab === "traces"}
                  data-testid="telemetry-tab-traces"
                  onClick={() => setTelemetrySubTab("traces")}
                  className={`px-3.5 py-1.5 rounded-lg text-xs font-mono font-bold transition-colors cursor-pointer flex items-center gap-2 ${
                    telemetrySubTab === "traces"
                      ? "bg-sky-600 text-white shadow-sm"
                      : "bg-[#090d16] text-slate-400 border border-slate-800 hover:text-slate-200 hover:border-slate-700"
                  }`}
                >
                  <span>2. Recent Traced Requests</span>
                  <span
                    className={`px-1.5 py-0.5 rounded text-[10px] ${
                      telemetrySubTab === "traces"
                        ? "bg-sky-800/80 text-sky-100"
                        : "bg-slate-800 text-slate-300"
                    }`}
                  >
                    {traceEvalResult.items.length}
                  </span>
                </button>
              </div>

              {telemetrySubTab === "routes" && (
                <div data-testid="telemetry-panel-routes" className="space-y-4">
                  {/* CEL SmartFilterInput for Per-Route RED Metrics */}
                  <div data-testid="routes-cel-filter-section" className="space-y-3">
                    <SmartFilterInput
                      value={routeCelFilter}
                      onApply={setRouteCelFilter}
                      fields={ROUTE_METRIC_FIELDS}
                      presets={ROUTE_METRIC_PRESETS}
                      syntaxExamples={ROUTE_METRIC_SYNTAX}
                      error={routeEvalResult.error}
                      placeholder="Filter routes with CEL (e.g., error_count > 0 || p95_rtt_ms > 10.0)..."
                      inputTestId="route-search-input"
                      helpTestId="routes-cel-help-btn"
                      applyTestId="routes-cel-apply-btn"
                      className="w-full"
                    />

                    <CollectionSortBar
                      totalCount={allRoutes.length}
                      filteredCount={routeEvalResult.items.length}
                      noun="routes"
                      sortField={routeSortField}
                      sortDir={routeSortDir}
                      options={[
                        { field: "total_requests", label: "Requests" },
                        { field: "error_count", label: "Errors" },
                        { field: "availability_percent", label: "Availability" },
                        { field: "p50_rtt_ms", label: "P50 RTT" },
                        { field: "p95_rtt_ms", label: "P95 RTT" },
                        { field: "p99_rtt_ms", label: "P99 RTT" },
                        { field: "service_name", label: "Service" },
                        { field: "route", label: "Route" },
                      ]}
                      onSortFieldChange={setRouteSortField}
                      onSortDirToggle={() =>
                        setRouteSortDir((d: SortDirection) => (d === "asc" ? "desc" : "asc"))
                      }
                      testIdPrefix="routes"
                    />
                  </div>

                  {/* Per-Route RED Table */}
                  <div className="overflow-x-auto">
                    <table
                      data-testid="routes-red-table"
                      className="w-full text-left border-collapse text-xs font-mono"
                    >
                      <thead>
                        <tr className="border-b border-slate-800 text-slate-400">
                          <SortableTh
                            label="SERVICE"
                            field="service_name"
                            activeField={routeSortField}
                            direction={routeSortDir}
                            onToggle={(f) =>
                              toggleColumnSort(
                                f,
                                routeSortField,
                                routeSortDir,
                                setRouteSortField,
                                setRouteSortDir,
                              )
                            }
                          />
                          <SortableTh
                            label="REGION"
                            field="region"
                            activeField={routeSortField}
                            direction={routeSortDir}
                            onToggle={(f) =>
                              toggleColumnSort(
                                f,
                                routeSortField,
                                routeSortDir,
                                setRouteSortField,
                                setRouteSortDir,
                              )
                            }
                          />
                          <SortableTh
                            label="METHOD"
                            field="method"
                            activeField={routeSortField}
                            direction={routeSortDir}
                            onToggle={(f) =>
                              toggleColumnSort(
                                f,
                                routeSortField,
                                routeSortDir,
                                setRouteSortField,
                                setRouteSortDir,
                              )
                            }
                          />
                          <SortableTh
                            label="ROUTE"
                            field="route"
                            activeField={routeSortField}
                            direction={routeSortDir}
                            onToggle={(f) =>
                              toggleColumnSort(
                                f,
                                routeSortField,
                                routeSortDir,
                                setRouteSortField,
                                setRouteSortDir,
                              )
                            }
                          />
                          <SortableTh
                            label="REQS"
                            field="total_requests"
                            activeField={routeSortField}
                            direction={routeSortDir}
                            onToggle={(f) =>
                              toggleColumnSort(
                                f,
                                routeSortField,
                                routeSortDir,
                                setRouteSortField,
                                setRouteSortDir,
                              )
                            }
                          />
                          <SortableTh
                            label="ERRORS"
                            field="error_count"
                            activeField={routeSortField}
                            direction={routeSortDir}
                            onToggle={(f) =>
                              toggleColumnSort(
                                f,
                                routeSortField,
                                routeSortDir,
                                setRouteSortField,
                                setRouteSortDir,
                              )
                            }
                          />
                          <SortableTh
                            label="AVAILABILITY"
                            field="availability_percent"
                            activeField={routeSortField}
                            direction={routeSortDir}
                            onToggle={(f) =>
                              toggleColumnSort(
                                f,
                                routeSortField,
                                routeSortDir,
                                setRouteSortField,
                                setRouteSortDir,
                              )
                            }
                          />
                          <SortableTh
                            label="P50 RTT"
                            field="p50_rtt_ms"
                            activeField={routeSortField}
                            direction={routeSortDir}
                            onToggle={(f) =>
                              toggleColumnSort(
                                f,
                                routeSortField,
                                routeSortDir,
                                setRouteSortField,
                                setRouteSortDir,
                              )
                            }
                          />
                          <SortableTh
                            label="P95 RTT"
                            field="p95_rtt_ms"
                            activeField={routeSortField}
                            direction={routeSortDir}
                            onToggle={(f) =>
                              toggleColumnSort(
                                f,
                                routeSortField,
                                routeSortDir,
                                setRouteSortField,
                                setRouteSortDir,
                              )
                            }
                          />
                          <SortableTh
                            label="P99 RTT"
                            field="p99_rtt_ms"
                            activeField={routeSortField}
                            direction={routeSortDir}
                            onToggle={(f) =>
                              toggleColumnSort(
                                f,
                                routeSortField,
                                routeSortDir,
                                setRouteSortField,
                                setRouteSortDir,
                              )
                            }
                          />
                        </tr>
                      </thead>
                      <tbody className="divide-y divide-slate-800/60">
                        {routeEvalResult.items.map((r) => (
                          <tr
                            key={`${r.serviceName}-${r.method}-${r.route}`}
                            onClick={() => handleSelectRouteFromMetrics(r)}
                            className="hover:bg-slate-900/60 cursor-pointer transition-colors"
                          >
                            <td className="py-2 px-3 text-sky-300">{r.serviceName}</td>
                            <td className="py-2 px-3 text-slate-400">{r.region}</td>
                            <td className="py-2 px-3 text-slate-300">{r.method}</td>
                            <td className="py-2 px-3">
                              <button
                                type="button"
                                data-testid={`select-route-${r.route}`}
                                onClick={(e) => {
                                  e.stopPropagation();
                                  handleSelectRouteFromMetrics(r);
                                }}
                                title="Click to view traced requests for this route"
                                className="text-sky-300 hover:text-sky-200 underline decoration-sky-500/50 hover:decoration-sky-300 font-mono cursor-pointer text-left"
                              >
                                {r.route}
                              </button>
                            </td>
                            <td className="py-2 px-3">{r.totalRequests.toString()}</td>
                            <td className="py-2 px-3 text-rose-400">{r.errorCount.toString()}</td>
                            <td className="py-2 px-3 text-emerald-400">
                              {r.availabilityPercent.toFixed(2)}%
                            </td>
                            <td className="py-2 px-3">{r.p50RttMs.toFixed(2)} ms</td>
                            <td className="py-2 px-3">{r.p95RttMs.toFixed(2)} ms</td>
                            <td className="py-2 px-3">{r.p99RttMs.toFixed(2)} ms</td>
                          </tr>
                        ))}
                        {routeEvalResult.items.length === 0 && (
                          <tr>
                            <td colSpan={10} className="py-4 text-center text-slate-500">
                              No route metrics recorded for current filter.
                            </td>
                          </tr>
                        )}
                      </tbody>
                    </table>
                  </div>
                </div>
              )}

              {/* Traced Requests Table (Tab 2) */}
              {telemetrySubTab === "traces" && (
                <div data-testid="telemetry-panel-traces" className="space-y-3">
                  <div className="flex flex-wrap items-center justify-between gap-2">
                    <h3 className="text-xs font-mono font-bold uppercase tracking-wider text-slate-400">
                      Recent Traced Requests (VictoriaLogs / Ring Buffer)
                    </h3>
                  </div>

                  <SmartFilterInput
                    value={traceCelFilter}
                    onApply={setTraceCelFilter}
                    fields={TRACE_REQUEST_FIELDS}
                    presets={TRACE_REQUEST_PRESETS}
                    syntaxExamples={TRACE_REQUEST_SYNTAX}
                    error={traceEvalResult.error}
                    placeholder="Filter traced requests with CEL (e.g., success == false || rtt_ms > 15.0)..."
                    inputTestId="traces-filter-input"
                    helpTestId="traces-cel-help-btn"
                    applyTestId="traces-cel-apply-btn"
                    className="w-full"
                  />

                  <CollectionSortBar
                    totalCount={routeRequests.length}
                    filteredCount={traceEvalResult.items.length}
                    noun="traced requests"
                    sortField={traceSortField}
                    sortDir={traceSortDir}
                    options={[
                      { field: "occurred_at", label: "Occurred At" },
                      { field: "rtt_ms", label: "RTT" },
                      { field: "status_code", label: "Status Code" },
                      { field: "service_name", label: "Service" },
                      { field: "route", label: "Route" },
                      { field: "trace_id", label: "Trace ID" },
                    ]}
                    onSortFieldChange={setTraceSortField}
                    onSortDirToggle={() =>
                      setTraceSortDir((d: SortDirection) => (d === "asc" ? "desc" : "asc"))
                    }
                    testIdPrefix="traces"
                  />

                  <div className="overflow-x-auto">
                    <table
                      data-testid="route-requests-table"
                      className="w-full text-left border-collapse text-xs font-mono"
                    >
                      <thead>
                        <tr className="border-b border-slate-800 text-slate-400">
                          <SortableTh
                            label="OCCURRED AT"
                            field="occurred_at"
                            activeField={traceSortField}
                            direction={traceSortDir}
                            onToggle={(f) =>
                              toggleColumnSort(
                                f,
                                traceSortField,
                                traceSortDir,
                                setTraceSortField,
                                setTraceSortDir,
                              )
                            }
                          />
                          <SortableTh
                            label="TRACE ID (UUIDv7)"
                            field="trace_id"
                            activeField={traceSortField}
                            direction={traceSortDir}
                            onToggle={(f) =>
                              toggleColumnSort(
                                f,
                                traceSortField,
                                traceSortDir,
                                setTraceSortField,
                                setTraceSortDir,
                              )
                            }
                          />
                          <SortableTh
                            label="REGION"
                            field="region"
                            activeField={traceSortField}
                            direction={traceSortDir}
                            onToggle={(f) =>
                              toggleColumnSort(
                                f,
                                traceSortField,
                                traceSortDir,
                                setTraceSortField,
                                setTraceSortDir,
                              )
                            }
                          />
                          <SortableTh
                            label="SERVICE"
                            field="service_name"
                            activeField={traceSortField}
                            direction={traceSortDir}
                            onToggle={(f) =>
                              toggleColumnSort(
                                f,
                                traceSortField,
                                traceSortDir,
                                setTraceSortField,
                                setTraceSortDir,
                              )
                            }
                          />
                          <SortableTh
                            label="METHOD"
                            field="method"
                            activeField={traceSortField}
                            direction={traceSortDir}
                            onToggle={(f) =>
                              toggleColumnSort(
                                f,
                                traceSortField,
                                traceSortDir,
                                setTraceSortField,
                                setTraceSortDir,
                              )
                            }
                          />
                          <SortableTh
                            label="ROUTE"
                            field="route"
                            activeField={traceSortField}
                            direction={traceSortDir}
                            onToggle={(f) =>
                              toggleColumnSort(
                                f,
                                traceSortField,
                                traceSortDir,
                                setTraceSortField,
                                setTraceSortDir,
                              )
                            }
                          />
                          <SortableTh
                            label="STATUS"
                            field="status_code"
                            activeField={traceSortField}
                            direction={traceSortDir}
                            onToggle={(f) =>
                              toggleColumnSort(
                                f,
                                traceSortField,
                                traceSortDir,
                                setTraceSortField,
                                setTraceSortDir,
                              )
                            }
                          />
                          <SortableTh
                            label="RTT"
                            field="rtt_ms"
                            activeField={traceSortField}
                            direction={traceSortDir}
                            onToggle={(f) =>
                              toggleColumnSort(
                                f,
                                traceSortField,
                                traceSortDir,
                                setTraceSortField,
                                setTraceSortDir,
                              )
                            }
                          />
                        </tr>
                      </thead>
                      <tbody className="divide-y divide-slate-800/60">
                        {traceEvalResult.items.map((req, idx) => (
                          <tr
                            key={`${req.traceId}-${req.serviceName}-${idx}`}
                            className="hover:bg-slate-900/50"
                          >
                            <td className="py-2 px-3 text-slate-400">
                              {formatTimestamp(req.occurredAt)}
                            </td>
                            <td className="py-2 px-3">
                              <button
                                type="button"
                                data-testid={`inspect-trace-${req.traceId}`}
                                onClick={() => void handleInspectTrace(req.traceId)}
                                title="Click to inspect full cross-service trace"
                                className="text-amber-300 hover:text-amber-200 underline decoration-amber-500/50 hover:decoration-amber-300 font-mono cursor-pointer text-left"
                              >
                                {req.traceId}
                              </button>
                            </td>
                            <td className="py-2 px-3 text-slate-400">{req.region}</td>
                            <td className="py-2 px-3 text-sky-300">{req.serviceName}</td>
                            <td className="py-2 px-3 text-slate-300">{req.method}</td>
                            <td className="py-2 px-3 text-slate-100">{req.route}</td>
                            <td className="py-2 px-3">
                              <span className={req.success ? "text-emerald-400" : "text-rose-400"}>
                                {req.statusCode} ({req.success ? "OK" : "ERR"})
                              </span>
                            </td>
                            <td className="py-2 px-3">{req.rttMs.toFixed(2)} ms</td>
                          </tr>
                        ))}
                        {traceEvalResult.items.length === 0 && (
                          <tr>
                            <td colSpan={8} className="py-4 text-center text-slate-500">
                              No traced requests match current filters.
                            </td>
                          </tr>
                        )}
                      </tbody>
                    </table>
                  </div>
                </div>
              )}
            </div>
          </div>
        )}

        {activeTab === "tasks" && (
          <div className="bg-[#111827] border border-slate-800 rounded-xl p-5 space-y-4">
            <div className="flex items-center justify-between">
              <h2 className="text-sm font-mono font-bold uppercase tracking-wider text-slate-200">
                Registered Operational Tasks &amp; Manual Triggers
              </h2>
              <span className="text-xs font-mono text-slate-400">
                Requires active JIT elevation lease
              </span>
            </div>

            <SmartFilterInput
              value={taskCelFilter}
              onApply={setTaskCelFilter}
              fields={TASK_FIELDS}
              presets={TASK_PRESETS}
              syntaxExamples={TASK_SYNTAX}
              error={taskEvalResult.error}
              placeholder='Filter operational tasks with CEL (e.g., name.startsWith("cleanup_") || default_interval == "1h")...'
              inputTestId="tasks-filter-input"
              helpTestId="tasks-cel-help-btn"
              applyTestId="tasks-cel-apply-btn"
              className="w-full"
            />

            <CollectionSortBar
              totalCount={tasks.length}
              filteredCount={taskEvalResult.items.length}
              noun="tasks"
              sortField={taskSortField}
              sortDir={taskSortDir}
              options={[
                { field: "name", label: "Task Name" },
                { field: "procedure", label: "RPC Procedure" },
                { field: "default_interval", label: "Schedule Interval" },
              ]}
              onSortFieldChange={setTaskSortField}
              onSortDirToggle={() =>
                setTaskSortDir((d: SortDirection) => (d === "asc" ? "desc" : "asc"))
              }
              testIdPrefix="tasks"
            />

            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              {taskEvalResult.items.map((t) => (
                <div
                  key={t.name}
                  data-testid={`task-card-${t.name}`}
                  className="bg-[#090d16] border border-slate-800 rounded-lg p-4 flex flex-col justify-between gap-3"
                >
                  <div className="space-y-1">
                    <div className="flex items-center justify-between">
                      <span className="font-mono font-bold text-sm text-sky-300">{t.name}</span>
                      <span className="text-xs font-mono text-slate-400">
                        Schedule: {t.defaultInterval}
                      </span>
                    </div>
                    <p className="text-xs text-slate-300">{t.description}</p>
                    <div className="text-[11px] font-mono text-slate-500">
                      RPC: /quadsmith.ops.OpsService/{t.procedure}
                    </div>
                  </div>

                  {t.name === "set_user_role" && (
                    <div className="flex gap-2 pt-2">
                      <input
                        type="text"
                        value={roleTargetUser}
                        onChange={(e) => setRoleTargetUser(e.target.value)}
                        placeholder="Username or UUID"
                        className="flex-1 bg-[#111827] border border-slate-700 rounded px-2.5 py-1 text-xs font-mono text-slate-100"
                      />
                      <select
                        aria-label="Target Role"
                        value={roleTargetValue}
                        onChange={(e) => setRoleTargetValue(e.target.value)}
                        className="bg-[#111827] border border-slate-700 rounded px-2 py-1 text-xs font-mono text-slate-100"
                      >
                        <option value="USER">USER</option>
                        <option value="ADMIN">ADMIN</option>
                      </select>
                    </div>
                  )}

                  <div className="pt-2 flex justify-end">
                    <button
                      type="button"
                      data-testid={`trigger-task-${t.name}`}
                      disabled={!hasActiveJit}
                      onClick={() => void handleTriggerTask(t.name)}
                      className={`px-3 py-1.5 rounded text-xs font-mono font-semibold cursor-pointer ${
                        hasActiveJit
                          ? "bg-sky-600 hover:bg-sky-500 text-white"
                          : "bg-slate-800 text-slate-500 border border-slate-700 cursor-not-allowed"
                      }`}
                    >
                      {hasActiveJit ? `▶ Trigger ${t.name}` : "🔒 Requires JIT Lease"}
                    </button>
                  </div>
                </div>
              ))}
            </div>
          </div>
        )}

        {activeTab === "jobs" && (
          <div className="bg-[#111827] border border-slate-800 rounded-xl p-5 space-y-4">
            <div className="flex flex-wrap items-center justify-between gap-3">
              <h2 className="text-sm font-mono font-bold uppercase tracking-wider text-slate-200">
                Background Job Execution History
              </h2>
              <select
                aria-label="Filter Jobs by Task"
                value={jobTaskFilter}
                onChange={(e) => setJobTaskFilter(e.target.value)}
                className="bg-[#090d16] border border-slate-700 rounded px-2.5 py-1 text-xs font-mono text-slate-200"
              >
                <option value="">All Tasks</option>
                {tasks.map((t) => (
                  <option key={t.name} value={t.name}>
                    {t.name}
                  </option>
                ))}
              </select>
            </div>

            <SmartFilterInput
              value={jobCelFilter}
              onApply={setJobCelFilter}
              fields={JOB_FIELDS}
              presets={JOB_PRESETS}
              syntaxExamples={JOB_SYNTAX}
              error={jobEvalResult.error}
              placeholder='Filter job executions with CEL (e.g., status != "OK" || duration_ms >= 50)...'
              inputTestId="jobs-filter-input"
              helpTestId="jobs-cel-help-btn"
              applyTestId="jobs-cel-apply-btn"
              className="w-full"
            />

            <CollectionSortBar
              totalCount={jobs.length}
              filteredCount={jobEvalResult.items.length}
              noun="jobs"
              sortField={jobSortField}
              sortDir={jobSortDir}
              options={[
                { field: "created_at", label: "Created At" },
                { field: "duration_ms", label: "Duration" },
                { field: "task_name", label: "Task Name" },
                { field: "caller", label: "Caller" },
                { field: "status", label: "Status" },
                { field: "job_id", label: "Job ID" },
              ]}
              onSortFieldChange={setJobSortField}
              onSortDirToggle={() =>
                setJobSortDir((d: SortDirection) => (d === "asc" ? "desc" : "asc"))
              }
              testIdPrefix="jobs"
            />

            <div className="overflow-x-auto">
              <table
                data-testid="jobs-table"
                className="w-full text-left border-collapse text-xs font-mono"
              >
                <thead>
                  <tr className="border-b border-slate-800 text-slate-400">
                    <SortableTh
                      label="JOB ID"
                      field="job_id"
                      activeField={jobSortField}
                      direction={jobSortDir}
                      onToggle={(f) =>
                        toggleColumnSort(
                          f,
                          jobSortField,
                          jobSortDir,
                          setJobSortField,
                          setJobSortDir,
                        )
                      }
                    />
                    <SortableTh
                      label="TRACE ID"
                      field="trace_id"
                      activeField={jobSortField}
                      direction={jobSortDir}
                      onToggle={(f) =>
                        toggleColumnSort(
                          f,
                          jobSortField,
                          jobSortDir,
                          setJobSortField,
                          setJobSortDir,
                        )
                      }
                    />
                    <SortableTh
                      label="REGION"
                      field="region"
                      activeField={jobSortField}
                      direction={jobSortDir}
                      onToggle={(f) =>
                        toggleColumnSort(
                          f,
                          jobSortField,
                          jobSortDir,
                          setJobSortField,
                          setJobSortDir,
                        )
                      }
                    />
                    <SortableTh
                      label="TASK"
                      field="task_name"
                      activeField={jobSortField}
                      direction={jobSortDir}
                      onToggle={(f) =>
                        toggleColumnSort(
                          f,
                          jobSortField,
                          jobSortDir,
                          setJobSortField,
                          setJobSortDir,
                        )
                      }
                    />
                    <SortableTh
                      label="CALLER"
                      field="caller"
                      activeField={jobSortField}
                      direction={jobSortDir}
                      onToggle={(f) =>
                        toggleColumnSort(
                          f,
                          jobSortField,
                          jobSortDir,
                          setJobSortField,
                          setJobSortDir,
                        )
                      }
                    />
                    <SortableTh
                      label="STATUS"
                      field="status"
                      activeField={jobSortField}
                      direction={jobSortDir}
                      onToggle={(f) =>
                        toggleColumnSort(
                          f,
                          jobSortField,
                          jobSortDir,
                          setJobSortField,
                          setJobSortDir,
                        )
                      }
                    />
                    <SortableTh
                      label="DURATION"
                      field="duration_ms"
                      activeField={jobSortField}
                      direction={jobSortDir}
                      onToggle={(f) =>
                        toggleColumnSort(
                          f,
                          jobSortField,
                          jobSortDir,
                          setJobSortField,
                          setJobSortDir,
                        )
                      }
                    />
                    <SortableTh
                      label="CREATED AT"
                      field="created_at"
                      activeField={jobSortField}
                      direction={jobSortDir}
                      onToggle={(f) =>
                        toggleColumnSort(
                          f,
                          jobSortField,
                          jobSortDir,
                          setJobSortField,
                          setJobSortDir,
                        )
                      }
                    />
                  </tr>
                </thead>
                <tbody className="divide-y divide-slate-800/60">
                  {jobEvalResult.items.map((j) => (
                    <tr
                      key={j.uuid}
                      onClick={() => setSelectedJob(j)}
                      className="hover:bg-slate-900/50 cursor-pointer"
                    >
                      <td className="py-2 px-3 text-sky-300 font-semibold">{j.jobId}</td>
                      <td className="py-2 px-3">
                        <button
                          type="button"
                          data-testid={`inspect-trace-${j.traceId}`}
                          onClick={(e) => {
                            e.stopPropagation();
                            void handleInspectTrace(j.traceId);
                          }}
                          title="Click to inspect full cross-service trace"
                          className="text-amber-300 hover:text-amber-200 underline decoration-amber-500/50 hover:decoration-amber-300 font-mono cursor-pointer text-left"
                        >
                          {j.traceId}
                        </button>
                      </td>
                      <td className="py-2 px-3 text-slate-400">{j.region}</td>
                      <td className="py-2 px-3 text-slate-100">{j.taskName}</td>
                      <td className="py-2 px-3 text-slate-300">
                        {j.caller} ({j.callerType})
                      </td>
                      <td className="py-2 px-3">
                        <span className={j.status === "OK" ? "text-emerald-400" : "text-rose-400"}>
                          {j.status}
                        </span>
                      </td>
                      <td className="py-2 px-3">{j.durationMs.toString()} ms</td>
                      <td className="py-2 px-3 text-slate-400">{formatTimestamp(j.createdAt)}</td>
                    </tr>
                  ))}
                  {jobEvalResult.items.length === 0 && (
                    <tr>
                      <td colSpan={8} className="py-4 text-center text-slate-500">
                        No job executions recorded yet.
                      </td>
                    </tr>
                  )}
                </tbody>
              </table>
            </div>

            {selectedJob && (
              <div className="p-4 rounded-lg bg-[#090d16] border border-slate-700 space-y-2 text-xs font-mono">
                <div className="flex items-center justify-between">
                  <span className="font-bold text-sky-300">
                    Job Inspector: {selectedJob.jobId} (Trace:{" "}
                    <button
                      type="button"
                      onClick={() => void handleInspectTrace(selectedJob.traceId)}
                      className="text-amber-300 hover:text-amber-200 underline cursor-pointer"
                    >
                      {selectedJob.traceId}
                    </button>
                    )
                  </span>
                  <button
                    type="button"
                    onClick={() => setSelectedJob(null)}
                    className="text-slate-400 hover:text-slate-200 cursor-pointer"
                  >
                    ✕ Close
                  </button>
                </div>
                <div>
                  Task: {selectedJob.taskName} ({selectedJob.procedure}) · Region:{" "}
                  {selectedJob.region}
                </div>
                <div>
                  Caller: {selectedJob.caller} ({selectedJob.callerType}) · Duration:{" "}
                  {selectedJob.durationMs.toString()} ms
                </div>
                {selectedJob.errorMessage && (
                  <div className="text-rose-400">Error: {selectedJob.errorMessage}</div>
                )}
              </div>
            )}
          </div>
        )}

        {activeTab === "db" && (
          <div className="bg-[#111827] border border-slate-800 rounded-xl p-5 space-y-4">
            <div className="flex items-center justify-between">
              <h2 className="text-sm font-mono font-bold uppercase tracking-wider text-slate-200">
                Public Database Statistics ({dbStats?.databaseName || "quadsmith_api"})
              </h2>
              <span className="text-xs font-mono text-sky-300">
                Total Size: {formatBytes(dbStats?.totalSizeBytes ?? 0n)}
              </span>
            </div>

            <SmartFilterInput
              value={dbCelFilter}
              onApply={setDbCelFilter}
              fields={DB_TABLE_FIELDS}
              presets={DB_TABLE_PRESETS}
              syntaxExamples={DB_TABLE_SYNTAX}
              error={dbEvalResult.error}
              placeholder='Filter database tables with CEL (e.g., row_count > 0 && table_name.contains("motor"))...'
              inputTestId="db-filter-input"
              helpTestId="db-cel-help-btn"
              applyTestId="db-cel-apply-btn"
              className="w-full"
            />

            <CollectionSortBar
              totalCount={dbTableList.length}
              filteredCount={dbEvalResult.items.length}
              noun="tables"
              sortField={dbSortField}
              sortDir={dbSortDir}
              options={[
                { field: "row_count", label: "Live Rows" },
                { field: "total_size_bytes", label: "Disk Size" },
                { field: "table_name", label: "Table Name" },
              ]}
              onSortFieldChange={setDbSortField}
              onSortDirToggle={() =>
                setDbSortDir((d: SortDirection) => (d === "asc" ? "desc" : "asc"))
              }
              testIdPrefix="db"
            />

            <div className="overflow-x-auto">
              <table
                data-testid="db-stats-table"
                className="w-full text-left border-collapse text-xs font-mono"
              >
                <thead>
                  <tr className="border-b border-slate-800 text-slate-400">
                    <SortableTh
                      label="TABLE NAME"
                      field="table_name"
                      activeField={dbSortField}
                      direction={dbSortDir}
                      onToggle={(f) =>
                        toggleColumnSort(f, dbSortField, dbSortDir, setDbSortField, setDbSortDir)
                      }
                    />
                    <SortableTh
                      label="LIVE ROWS"
                      field="row_count"
                      activeField={dbSortField}
                      direction={dbSortDir}
                      onToggle={(f) =>
                        toggleColumnSort(f, dbSortField, dbSortDir, setDbSortField, setDbSortDir)
                      }
                    />
                    <SortableTh
                      label="DISK SIZE"
                      field="total_size_bytes"
                      activeField={dbSortField}
                      direction={dbSortDir}
                      onToggle={(f) =>
                        toggleColumnSort(f, dbSortField, dbSortDir, setDbSortField, setDbSortDir)
                      }
                    />
                  </tr>
                </thead>
                <tbody className="divide-y divide-slate-800/60">
                  {dbEvalResult.items.map((tbl) => (
                    <tr key={tbl.tableName} className="hover:bg-slate-900/50">
                      <td className="py-2 px-3 text-slate-100">{tbl.tableName}</td>
                      <td className="py-2 px-3 text-sky-300">{tbl.rowCount.toString()}</td>
                      <td className="py-2 px-3 text-slate-300">
                        {formatBytes(tbl.totalSizeBytes)}
                      </td>
                    </tr>
                  ))}
                  {dbEvalResult.items.length === 0 && (
                    <tr>
                      <td colSpan={3} className="py-4 text-center text-slate-500">
                        No database tables match current filter.
                      </td>
                    </tr>
                  )}
                </tbody>
              </table>
            </div>
          </div>
        )}

        {activeTab === "iam" && (
          <div className="space-y-6">
            {/* IAM Users & Groups */}
            <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
              <div className="bg-[#111827] border border-slate-800 rounded-xl p-5 space-y-3">
                <h2 className="text-sm font-mono font-bold uppercase tracking-wider text-slate-200">
                  Active IAM Operators &amp; Robots
                </h2>
                <div className="space-y-2 text-xs font-mono">
                  {iamUsers.map((u) => (
                    <div
                      key={`${u.operator}-${u.groupName}`}
                      className="flex items-center justify-between p-2.5 rounded bg-[#090d16] border border-slate-800"
                    >
                      <div>
                        <span className="text-slate-100 font-bold">{u.operator}</span>{" "}
                        <span className="text-slate-400">
                          ({u.type} · {u.groupName})
                        </span>
                      </div>
                      <span className={u.hasActiveJit ? "text-amber-300" : "text-slate-500"}>
                        {u.hasActiveJit ? "⚡ JIT Active" : "Standard"}
                      </span>
                    </div>
                  ))}
                </div>
              </div>

              <div className="bg-[#111827] border border-slate-800 rounded-xl p-5 space-y-3">
                <h2 className="text-sm font-mono font-bold uppercase tracking-wider text-slate-200">
                  IAM Authorization Groups
                </h2>
                <div className="space-y-2 text-xs font-mono">
                  {iamGroups.map((g) => (
                    <div
                      key={g.name}
                      className="p-2.5 rounded bg-[#090d16] border border-slate-800 space-y-1"
                    >
                      <div className="text-sky-300 font-bold">{g.name}</div>
                      <div className="text-slate-400">{g.description}</div>
                      <div className="text-slate-500">
                        Allowed RPCs: {(g.allowedMethods ?? []).join(", ")}
                      </div>
                    </div>
                  ))}
                </div>
              </div>
            </div>

            {/* Dynamic Token Management */}
            <div className="bg-[#111827] border border-slate-800 rounded-xl p-5 space-y-4">
              <h2 className="text-sm font-mono font-bold uppercase tracking-wider text-slate-200">
                Internal Bearer Tokens
              </h2>

              <form onSubmit={handleCreateToken} className="flex flex-wrap items-center gap-2">
                <input
                  type="text"
                  value={newTokenName}
                  onChange={(e) => setNewTokenName(e.target.value)}
                  placeholder="New token name (e.g. oncall-cli)"
                  className="bg-[#090d16] border border-slate-700 rounded px-3 py-1.5 text-xs font-mono text-slate-100"
                />
                <select
                  aria-label="New Token Type"
                  value={newTokenType}
                  onChange={(e) => setNewTokenType(e.target.value)}
                  className="bg-[#090d16] border border-slate-700 rounded px-2.5 py-1.5 text-xs font-mono text-slate-200"
                >
                  <option value="HUMAN">HUMAN</option>
                  <option value="ROBOT">ROBOT</option>
                </select>
                <input
                  type="text"
                  value={newTokenGroup}
                  onChange={(e) => setNewTokenGroup(e.target.value)}
                  placeholder="Group (operators)"
                  className="bg-[#090d16] border border-slate-700 rounded px-3 py-1.5 text-xs font-mono text-slate-100 w-36"
                />
                <button
                  type="submit"
                  disabled={!hasActiveJit}
                  className="px-3 py-1.5 rounded bg-sky-600 hover:bg-sky-500 disabled:bg-slate-800 disabled:text-slate-500 text-white text-xs font-mono cursor-pointer"
                >
                  + Issue Token
                </button>
              </form>

              {createdTokenSecret && (
                <div className="p-3 rounded bg-amber-950/50 border border-amber-700 text-amber-200 text-xs font-mono">
                  New Token Secret (copy now):{" "}
                  <span className="font-bold">{createdTokenSecret}</span>
                </div>
              )}

              <SmartFilterInput
                value={tokenCelFilter}
                onApply={setTokenCelFilter}
                fields={TOKEN_FIELDS}
                presets={TOKEN_PRESETS}
                syntaxExamples={TOKEN_SYNTAX}
                error={tokenEvalResult.error}
                placeholder='Filter internal tokens with CEL (e.g., type == "ROBOT" && status == "ACTIVE")...'
                inputTestId="tokens-filter-input"
                helpTestId="tokens-cel-help-btn"
                applyTestId="tokens-cel-apply-btn"
                className="w-full"
              />

              <CollectionSortBar
                totalCount={tokens.length}
                filteredCount={tokenEvalResult.items.length}
                noun="tokens"
                sortField={tokenSortField}
                sortDir={tokenSortDir}
                options={[
                  { field: "created_at", label: "Created At" },
                  { field: "name", label: "Name" },
                  { field: "type", label: "Type" },
                  { field: "group_name", label: "Group" },
                  { field: "status", label: "Status" },
                  { field: "id", label: "Token ID" },
                ]}
                onSortFieldChange={setTokenSortField}
                onSortDirToggle={() =>
                  setTokenSortDir((d: SortDirection) => (d === "asc" ? "desc" : "asc"))
                }
                testIdPrefix="tokens"
              />

              <div className="overflow-x-auto">
                <table
                  data-testid="tokens-table"
                  className="w-full text-left border-collapse text-xs font-mono"
                >
                  <thead>
                    <tr className="border-b border-slate-800 text-slate-400">
                      <SortableTh
                        label="ID"
                        field="id"
                        activeField={tokenSortField}
                        direction={tokenSortDir}
                        onToggle={(f) =>
                          toggleColumnSort(
                            f,
                            tokenSortField,
                            tokenSortDir,
                            setTokenSortField,
                            setTokenSortDir,
                          )
                        }
                      />
                      <SortableTh
                        label="NAME"
                        field="name"
                        activeField={tokenSortField}
                        direction={tokenSortDir}
                        onToggle={(f) =>
                          toggleColumnSort(
                            f,
                            tokenSortField,
                            tokenSortDir,
                            setTokenSortField,
                            setTokenSortDir,
                          )
                        }
                      />
                      <SortableTh
                        label="TYPE"
                        field="type"
                        activeField={tokenSortField}
                        direction={tokenSortDir}
                        onToggle={(f) =>
                          toggleColumnSort(
                            f,
                            tokenSortField,
                            tokenSortDir,
                            setTokenSortField,
                            setTokenSortDir,
                          )
                        }
                      />
                      <SortableTh
                        label="GROUP"
                        field="group_name"
                        activeField={tokenSortField}
                        direction={tokenSortDir}
                        onToggle={(f) =>
                          toggleColumnSort(
                            f,
                            tokenSortField,
                            tokenSortDir,
                            setTokenSortField,
                            setTokenSortDir,
                          )
                        }
                      />
                      <SortableTh
                        label="EXPIRES AT"
                        field="expires_at"
                        activeField={tokenSortField}
                        direction={tokenSortDir}
                        onToggle={(f) =>
                          toggleColumnSort(
                            f,
                            tokenSortField,
                            tokenSortDir,
                            setTokenSortField,
                            setTokenSortDir,
                          )
                        }
                      />
                      <SortableTh
                        label="STATUS"
                        field="status"
                        activeField={tokenSortField}
                        direction={tokenSortDir}
                        onToggle={(f) =>
                          toggleColumnSort(
                            f,
                            tokenSortField,
                            tokenSortDir,
                            setTokenSortField,
                            setTokenSortDir,
                          )
                        }
                      />
                      <th className="py-2 px-3">ACTION</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-slate-800/60">
                    {tokenEvalResult.items.map((tok) => (
                      <tr key={tok.uuid}>
                        <td className="py-2 px-3 text-sky-300">{tok.id}</td>
                        <td className="py-2 px-3 text-slate-100">{tok.name}</td>
                        <td className="py-2 px-3 text-slate-300">{tok.type}</td>
                        <td className="py-2 px-3 text-slate-300">{tok.groupName}</td>
                        <td className="py-2 px-3 text-slate-400">
                          {formatTimestamp(tok.expiresAt)}
                        </td>
                        <td className="py-2 px-3">
                          {tok.revokedAt ? (
                            <span className="text-rose-400">REVOKED</span>
                          ) : (
                            <span className="text-emerald-400">ACTIVE</span>
                          )}
                        </td>
                        <td className="py-2 px-3">
                          {!tok.revokedAt && (
                            <button
                              type="button"
                              disabled={!hasActiveJit}
                              onClick={() => void handleRevokeToken(tok.id)}
                              className="text-rose-400 hover:underline disabled:text-slate-600 cursor-pointer"
                            >
                              Revoke
                            </button>
                          )}
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            </div>

            {/* Immutable Audit Log */}
            <div className="bg-[#111827] border border-slate-800 rounded-xl p-5 space-y-4">
              <h2 className="text-sm font-mono font-bold uppercase tracking-wider text-slate-200">
                Immutable Internal API Audit Log (internal_api_calls)
              </h2>

              <SmartFilterInput
                value={auditCelFilter}
                onApply={setAuditCelFilter}
                fields={AUDIT_CALL_FIELDS}
                presets={AUDIT_CALL_PRESETS}
                syntaxExamples={AUDIT_CALL_SYNTAX}
                error={auditEvalResult.error}
                placeholder='Filter internal API audit calls with CEL (e.g., jit_grant_id != "" || status != "OK")...'
                inputTestId="audit-filter-input"
                helpTestId="audit-cel-help-btn"
                applyTestId="audit-cel-apply-btn"
                className="w-full"
              />

              <CollectionSortBar
                totalCount={auditCalls.length}
                filteredCount={auditEvalResult.items.length}
                noun="audit records"
                sortField={auditSortField}
                sortDir={auditSortDir}
                options={[
                  { field: "created_at", label: "Timestamp" },
                  { field: "duration_ms", label: "Duration" },
                  { field: "caller", label: "Caller" },
                  { field: "method", label: "Method" },
                  { field: "status", label: "Status" },
                  { field: "group_name", label: "Group" },
                ]}
                onSortFieldChange={setAuditSortField}
                onSortDirToggle={() =>
                  setAuditSortDir((d: SortDirection) => (d === "asc" ? "desc" : "asc"))
                }
                testIdPrefix="audit"
              />

              <div className="overflow-x-auto">
                <table
                  data-testid="audit-table"
                  className="w-full text-left border-collapse text-xs font-mono"
                >
                  <thead>
                    <tr className="border-b border-slate-800 text-slate-400">
                      <SortableTh
                        label="TIMESTAMP"
                        field="created_at"
                        activeField={auditSortField}
                        direction={auditSortDir}
                        onToggle={(f) =>
                          toggleColumnSort(
                            f,
                            auditSortField,
                            auditSortDir,
                            setAuditSortField,
                            setAuditSortDir,
                          )
                        }
                      />
                      <SortableTh
                        label="TRACE ID"
                        field="trace_id"
                        activeField={auditSortField}
                        direction={auditSortDir}
                        onToggle={(f) =>
                          toggleColumnSort(
                            f,
                            auditSortField,
                            auditSortDir,
                            setAuditSortField,
                            setAuditSortDir,
                          )
                        }
                      />
                      <SortableTh
                        label="REGION"
                        field="region"
                        activeField={auditSortField}
                        direction={auditSortDir}
                        onToggle={(f) =>
                          toggleColumnSort(
                            f,
                            auditSortField,
                            auditSortDir,
                            setAuditSortField,
                            setAuditSortDir,
                          )
                        }
                      />
                      <SortableTh
                        label="CALLER"
                        field="caller"
                        activeField={auditSortField}
                        direction={auditSortDir}
                        onToggle={(f) =>
                          toggleColumnSort(
                            f,
                            auditSortField,
                            auditSortDir,
                            setAuditSortField,
                            setAuditSortDir,
                          )
                        }
                      />
                      <SortableTh
                        label="GROUP"
                        field="group_name"
                        activeField={auditSortField}
                        direction={auditSortDir}
                        onToggle={(f) =>
                          toggleColumnSort(
                            f,
                            auditSortField,
                            auditSortDir,
                            setAuditSortField,
                            setAuditSortDir,
                          )
                        }
                      />
                      <SortableTh
                        label="METHOD"
                        field="method"
                        activeField={auditSortField}
                        direction={auditSortDir}
                        onToggle={(f) =>
                          toggleColumnSort(
                            f,
                            auditSortField,
                            auditSortDir,
                            setAuditSortField,
                            setAuditSortDir,
                          )
                        }
                      />
                      <SortableTh
                        label="JIT GRANT"
                        field="jit_grant_id"
                        activeField={auditSortField}
                        direction={auditSortDir}
                        onToggle={(f) =>
                          toggleColumnSort(
                            f,
                            auditSortField,
                            auditSortDir,
                            setAuditSortField,
                            setAuditSortDir,
                          )
                        }
                      />
                      <SortableTh
                        label="STATUS"
                        field="status"
                        activeField={auditSortField}
                        direction={auditSortDir}
                        onToggle={(f) =>
                          toggleColumnSort(
                            f,
                            auditSortField,
                            auditSortDir,
                            setAuditSortField,
                            setAuditSortDir,
                          )
                        }
                      />
                      <SortableTh
                        label="DURATION"
                        field="duration_ms"
                        activeField={auditSortField}
                        direction={auditSortDir}
                        onToggle={(f) =>
                          toggleColumnSort(
                            f,
                            auditSortField,
                            auditSortDir,
                            setAuditSortField,
                            setAuditSortDir,
                          )
                        }
                      />
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-slate-800/60">
                    {auditEvalResult.items.map((c) => (
                      <tr key={c.uuid} className="hover:bg-slate-900/50">
                        <td className="py-2 px-3 text-slate-400">{formatTimestamp(c.createdAt)}</td>
                        <td className="py-2 px-3">
                          <button
                            type="button"
                            data-testid={`inspect-trace-${c.traceId}`}
                            onClick={() => void handleInspectTrace(c.traceId)}
                            title="Click to inspect full cross-service trace"
                            className="text-amber-300 hover:text-amber-200 underline decoration-amber-500/50 hover:decoration-amber-300 font-mono cursor-pointer text-left"
                          >
                            {c.traceId}
                          </button>
                        </td>
                        <td className="py-2 px-3 text-slate-400">{c.region}</td>
                        <td className="py-2 px-3 text-slate-100">{c.caller}</td>
                        <td className="py-2 px-3 text-slate-300">{c.groupName}</td>
                        <td className="py-2 px-3 text-sky-300">{c.method}</td>
                        <td className="py-2 px-3 text-amber-300">{c.jitGrantId || "—"}</td>
                        <td className="py-2 px-3">
                          <span
                            className={c.status === "OK" ? "text-emerald-400" : "text-rose-400"}
                          >
                            {c.status}
                          </span>
                        </td>
                        <td className="py-2 px-3">{c.durationMs.toString()} ms</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            </div>
          </div>
        )}
      </main>

      {/* Full Request Trace Inspector Modal */}
      {selectedTraceId && (
        <div
          data-testid="trace-inspector-modal"
          className="fixed inset-0 z-50 bg-slate-950/80 backdrop-blur-sm flex items-center justify-center p-4 overflow-y-auto"
          onClick={() => {
            setSelectedTraceId("");
            setSelectedTrace(null);
            setTraceError("");
          }}
        >
          <div
            className="bg-[#0b1120] border border-slate-700 rounded-xl max-w-5xl w-full max-h-[90vh] flex flex-col shadow-2xl overflow-hidden"
            onClick={(e) => e.stopPropagation()}
          >
            {/* Modal Header */}
            <div className="px-5 py-4 border-b border-slate-800 bg-[#111827] flex flex-wrap items-center justify-between gap-3">
              <div className="space-y-1">
                <div className="flex flex-wrap items-center gap-2">
                  <span className="px-2 py-0.5 rounded bg-amber-950/70 border border-amber-700 text-amber-300 text-[11px] font-mono font-bold uppercase tracking-wider">
                    Full Request Trace
                  </span>
                  <span
                    data-testid="trace-inspector-id"
                    className="font-mono font-bold text-sm text-amber-300"
                  >
                    {selectedTraceId}
                  </span>
                  {selectedTrace && (
                    <span
                      className={`px-2 py-0.5 rounded text-[11px] font-mono font-bold ${
                        selectedTrace.success
                          ? "bg-emerald-950/70 border border-emerald-700 text-emerald-300"
                          : "bg-rose-950/70 border border-rose-700 text-rose-300"
                      }`}
                    >
                      {selectedTrace.success ? "OK" : "ERROR"}
                    </span>
                  )}
                </div>
                {selectedTrace && (
                  <div className="text-xs font-mono text-slate-400 flex flex-wrap items-center gap-3">
                    <span>
                      Region:{" "}
                      <strong className="text-slate-200">{selectedTrace.region || "local"}</strong>
                    </span>
                    <span>
                      Root:{" "}
                      <strong className="text-sky-300">
                        {selectedTrace.rootService || "—"} {selectedTrace.rootRoute || ""}
                      </strong>
                    </span>
                    <span>
                      Total Duration:{" "}
                      <strong className="text-slate-200">
                        {Number(selectedTrace.totalRttMs || 0).toFixed(2)} ms
                      </strong>
                    </span>
                    <span>
                      Spans:{" "}
                      <strong className="text-slate-200">
                        {(selectedTrace.spans ?? []).length}
                      </strong>
                    </span>
                    <span>
                      Logs:{" "}
                      <strong className="text-slate-200">
                        {(selectedTrace.logs ?? []).length}
                      </strong>
                    </span>
                    <span>
                      Debug Logs:{" "}
                      <strong data-testid="trace-debug-logs-count" className="text-sky-300">
                        {(selectedTrace.logs ?? []).filter((l) => l.level === "DEBUG").length}
                      </strong>
                    </span>
                    {selectedTrace.startedAt && (
                      <span>Started: {formatTimestamp(selectedTrace.startedAt)}</span>
                    )}
                  </div>
                )}
              </div>
              <button
                type="button"
                data-testid="close-trace-inspector"
                onClick={() => {
                  setSelectedTraceId("");
                  setSelectedTrace(null);
                  setTraceError("");
                }}
                className="px-3 py-1.5 rounded-lg border border-slate-700 bg-slate-900 hover:bg-slate-800 text-slate-300 text-xs font-mono cursor-pointer"
              >
                ✕ Close
              </button>
            </div>

            {/* Modal Body */}
            <div className="p-5 overflow-y-auto space-y-6 text-xs font-mono">
              {traceLoading && (
                <div className="py-6 text-center text-slate-400">
                  Loading cross-service spans and structured logs for trace {selectedTraceId}...
                </div>
              )}

              {traceError && (
                <div className="p-3 rounded-lg bg-rose-950/60 border border-rose-700 text-rose-200">
                  {traceError}
                </div>
              )}

              {selectedTrace && !traceLoading && (
                <>
                  {/* Cross-Service Span Waterfall */}
                  <div className="space-y-3">
                    <h3 className="text-xs font-mono font-bold uppercase tracking-wider text-slate-300">
                      Cross-Service Hop Waterfall ({(selectedTrace.spans ?? []).length} Spans)
                    </h3>
                    <div
                      data-testid="trace-spans-list"
                      className="space-y-2 bg-[#090d16] border border-slate-800 rounded-lg p-3"
                    >
                      {(selectedTrace.spans ?? []).map((span, idx) => {
                        const spanRtt = Number(span.durationMs || 0);
                        const maxRtt = Math.max(
                          Number(selectedTrace.totalRttMs || 1),
                          spanRtt || 1,
                          1,
                        );
                        const widthPct = Math.min(100, Math.max(8, (spanRtt / maxRtt) * 100));
                        const attrs = span.attributes ?? {};
                        const hopType = attrs.hop_type || (idx === 0 ? "ingress" : "hop");
                        const caller = attrs.caller || "";
                        const jitGrantId = attrs.jit_grant_id || "";
                        return (
                          <div
                            key={`${span.serviceName}-${span.operation}-${idx}`}
                            data-testid={`trace-span-row-${idx}`}
                            className="p-2.5 rounded bg-[#111827] border border-slate-800/80 space-y-1.5"
                          >
                            <div className="flex flex-wrap items-center justify-between gap-2">
                              <div className="flex flex-wrap items-center gap-2">
                                <span className="px-1.5 py-0.5 rounded bg-slate-800 text-slate-300 text-[10px] uppercase">
                                  {hopType}
                                </span>
                                <span className="text-sky-300 font-bold">{span.serviceName}</span>
                                <span className="text-slate-500">({span.instanceId})</span>
                                <span className="text-slate-200 font-semibold">
                                  {span.method} {span.operation}
                                </span>
                              </div>
                              <div className="flex items-center gap-3">
                                {caller && (
                                  <span className="text-slate-400">
                                    Caller: <strong className="text-slate-200">{caller}</strong>
                                    {jitGrantId ? ` (JIT: ${jitGrantId})` : ""}
                                  </span>
                                )}
                                <span
                                  className={span.success ? "text-emerald-400" : "text-rose-400"}
                                >
                                  HTTP {span.statusCode}
                                </span>
                                <span className="text-amber-300 font-bold">
                                  {spanRtt.toFixed(2)} ms
                                </span>
                              </div>
                            </div>
                            <div className="w-full h-2 rounded bg-slate-900 overflow-hidden">
                              <div
                                className={`h-full rounded ${
                                  span.success ? "bg-sky-500" : "bg-rose-500"
                                }`}
                                style={{ width: `${widthPct}%` }}
                              />
                            </div>
                          </div>
                        );
                      })}
                      {(selectedTrace.spans ?? []).length === 0 && (
                        <div className="py-3 text-center text-slate-500">
                          No span hops recorded for this trace ID.
                        </div>
                      )}
                    </div>
                  </div>

                  {/* Chronological Cross-Service Logs */}
                  <div className="space-y-3">
                    <h3 className="text-xs font-mono font-bold uppercase tracking-wider text-slate-300">
                      Chronological Cross-Service Structured &amp; Debug Logs (
                      {(selectedTrace.logs ?? []).length} Events)
                    </h3>
                    <div className="overflow-x-auto bg-[#090d16] border border-slate-800 rounded-lg">
                      <table
                        data-testid="trace-logs-table"
                        className="w-full text-left border-collapse text-xs font-mono"
                      >
                        <thead>
                          <tr className="border-b border-slate-800 text-slate-400">
                            <th className="py-2 px-3">TIMESTAMP</th>
                            <th className="py-2 px-3">SERVICE</th>
                            <th className="py-2 px-3">LEVEL</th>
                            <th className="py-2 px-3">EVENT</th>
                            <th className="py-2 px-3">ROUTE &amp; ATTRIBUTES</th>
                            <th className="py-2 px-3">STATUS / RTT</th>
                          </tr>
                        </thead>
                        <tbody className="divide-y divide-slate-800/60">
                          {(selectedTrace.logs ?? []).map((log, idx) => {
                            const attrs = log.attributes ?? {};
                            const routeSummary =
                              attrs.route || attrs.operation
                                ? `${attrs.method ? `${attrs.method} ` : ""}${attrs.route || attrs.operation}`
                                : log.message;
                            const statusCode = attrs.status_code || attrs.status || "";
                            const rttStr = attrs.rtt_ms || attrs.duration_ms || "";
                            const attrPairs = Object.entries(attrs).filter(
                              ([k]) =>
                                k !== "route" &&
                                k !== "method" &&
                                k !== "status_code" &&
                                k !== "rtt_ms" &&
                                k !== "duration_ms",
                            );
                            return (
                              <tr
                                key={`${log.serviceName}-${log.message}-${idx}`}
                                className="hover:bg-slate-900/50"
                              >
                                <td className="py-2 px-3 text-slate-400 whitespace-nowrap">
                                  {formatTimestamp(log.timestamp)}
                                </td>
                                <td className="py-2 px-3 text-sky-300 whitespace-nowrap">
                                  {log.serviceName}{" "}
                                  <span className="text-slate-500">({log.instanceId})</span>
                                </td>
                                <td className="py-2 px-3">
                                  <span
                                    className={
                                      log.level === "ERROR"
                                        ? "text-rose-400 font-bold"
                                        : log.level === "WARN"
                                          ? "text-amber-300 font-bold"
                                          : log.level === "DEBUG"
                                            ? "text-sky-300 font-bold"
                                            : "text-emerald-400"
                                    }
                                  >
                                    {log.level}
                                  </span>
                                </td>
                                <td className="py-2 px-3 text-amber-300">{log.message}</td>
                                <td className="py-2 px-3 text-slate-200">
                                  <div>{routeSummary}</div>
                                  {attrPairs.length > 0 && (
                                    <div className="text-[11px] text-slate-500 break-all">
                                      {attrPairs.map(([k, v]) => `${k}=${v}`).join(" · ")}
                                    </div>
                                  )}
                                </td>
                                <td className="py-2 px-3 whitespace-nowrap text-slate-300">
                                  {statusCode ? `Status ${statusCode}` : "—"}
                                  {rttStr ? ` · ${rttStr} ms` : ""}
                                </td>
                              </tr>
                            );
                          })}
                          {(selectedTrace.logs ?? []).length === 0 && (
                            <tr>
                              <td colSpan={6} className="py-4 text-center text-slate-500">
                                No structured log entries recorded for this trace ID.
                              </td>
                            </tr>
                          )}
                        </tbody>
                      </table>
                    </div>
                  </div>
                </>
              )}
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
