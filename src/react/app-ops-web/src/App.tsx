import React, { useCallback, useEffect, useMemo, useState } from "react";
import { createClient, type Interceptor } from "@connectrpc/connect";
import { createConnectTransport } from "@connectrpc/connect-web";
import {
  OpsService,
  type GetCurrentOperatorResponse,
  type GetDatabaseStatsResponse,
  type GetOAuthConfigResponse,
  type GetSystemHealthResponse,
  type GetSystemMetricsResponse,
  type IamGroupSummary,
  type IamUserSummary,
  type InternalApiCallRecord,
  type InternalTokenRecord,
  type JobRecord,
  type RouteMetricStat,
  type RouteRequestLog,
  type TaskDef,
} from "@quadsmith/shared/gen/quadsmith/ops/ops_pb";

const OPS_TOKEN_STORAGE_KEY = "qs_ops_token";
const SANDBOX_DEFAULT_OPERATOR_TOKEN = "qs_op_sandbox_token_12345";

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

function formatTimestamp(ts?: { seconds: bigint }): string {
  if (!ts) return "—";
  const ms = Number(ts.seconds) * 1000;
  if (!Number.isFinite(ms) || ms <= 0) return "—";
  return new Date(ms).toISOString().replace("T", " ").slice(0, 19) + " UTC";
}

function formatBytes(bytes: bigint | number): string {
  const b = Number(bytes);
  if (!Number.isFinite(b) || b <= 0) return "0 B";
  if (b < 1024) return `${b} B`;
  if (b < 1024 * 1024) return `${(b / 1024).toFixed(1)} KB`;
  return `${(b / (1024 * 1024)).toFixed(2)} MB`;
}

export default function App() {
  const [token, setToken] = useState<string>(() => getStoredOpsToken());
  const [manualTokenInput, setManualTokenInput] = useState<string>("");
  const [oauthConfig, setOauthConfig] = useState<GetOAuthConfigResponse | null>(null);
  const [operatorInfo, setOperatorInfo] = useState<GetCurrentOperatorResponse | null>(null);
  const [authError, setAuthError] = useState<string>("");
  const [loading, setLoading] = useState<boolean>(false);

  const [activeTab, setActiveTab] = useState<ActiveTab>("health");
  const [regionFilter, setRegionFilter] = useState<string>("all");
  const [serviceFilter, setServiceFilter] = useState<string>("all");
  const [metricsWindow, setMetricsWindow] = useState<string>("5m");
  const [routeSearch, setRouteSearch] = useState<string>("");
  const [onlyErrors, setOnlyErrors] = useState<boolean>(false);

  // Telemetry & Ops state
  const [systemHealth, setSystemHealth] = useState<GetSystemHealthResponse | null>(null);
  const [systemMetrics, setSystemMetrics] = useState<GetSystemMetricsResponse | null>(null);
  const [routeRequests, setRouteRequests] = useState<RouteRequestLog[]>([]);
  const [tasks, setTasks] = useState<TaskDef[]>([]);
  const [jobs, setJobs] = useState<JobRecord[]>([]);
  const [jobTaskFilter, setJobTaskFilter] = useState<string>("");
  const [selectedJob, setSelectedJob] = useState<JobRecord | null>(null);
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
          route: routeSearch.trim() || undefined,
          window: metricsWindow,
          onlyErrors,
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
  }, [
    client,
    token,
    regionFilter,
    serviceFilter,
    metricsWindow,
    routeSearch,
    onlyErrors,
    jobTaskFilter,
  ]);

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

  const flattenedRoutes = useMemo(() => {
    const list: RouteMetricStat[] = [];
    for (const svc of systemMetrics?.services ?? []) {
      for (const r of svc.routes ?? []) {
        if (
          routeSearch.trim() &&
          !r.route.toLowerCase().includes(routeSearch.trim().toLowerCase())
        ) {
          continue;
        }
        if (onlyErrors && Number(r.errorCount) === 0) {
          continue;
        }
        list.push(r);
      }
    }
    return list;
  }, [systemMetrics, routeSearch, onlyErrors]);

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

        {/* JIT-gated Grafana Deep-Dive Link */}
        <div className="flex items-center gap-2">
          {hasActiveJit ? (
            <a
              href="/grafana/d/quadsmith-health/quadsmith-service-health"
              target="_blank"
              rel="noreferrer"
              data-testid="open-grafana-btn"
              className="px-3 py-1.5 rounded bg-sky-600/25 hover:bg-sky-600/35 text-sky-200 border border-sky-500/40 text-xs font-mono font-semibold transition"
            >
              📊 Open Grafana Dashboard (/grafana/)
            </a>
          ) : (
            <span
              data-testid="open-grafana-btn"
              title="Acquire an active JIT lease to access /grafana/*"
              className="px-3 py-1.5 rounded bg-slate-900 text-slate-500 border border-slate-800 text-xs font-mono cursor-not-allowed"
            >
              🔒 Grafana Dashboard (Requires JIT)
            </span>
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
                  5-Service Deep Health Matrix (/healthz)
                </h2>
                <span className="text-xs font-mono text-slate-400">
                  Checked: {formatTimestamp(systemHealth?.checkedAt)}
                </span>
              </div>

              <div
                data-testid="service-health-matrix"
                className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-5 gap-3"
              >
                {(systemHealth?.services ?? []).map((svc) => (
                  <div
                    key={`${svc.serviceName}-${svc.region}`}
                    data-testid={`health-card-${svc.serviceName}`}
                    className="bg-[#090d16] border border-slate-800 rounded-lg p-3.5 space-y-2.5"
                  >
                    <div className="flex items-center justify-between">
                      <span className="font-mono font-bold text-xs text-slate-100">
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
                    </div>

                    <div className="text-[11px] font-mono text-slate-400 space-y-0.5">
                      <div>Region: {svc.region}</div>
                      <div>Instance: {svc.instanceId}</div>
                      <div>
                        Uptime: {svc.uptimeSeconds.toString()}s · Goroutines: {svc.numGoroutines}
                      </div>
                      {svc.dbMaxConns > 0 && (
                        <div>
                          DB Pool: {svc.dbAcquiredConns}/{svc.dbMaxConns} (idle {svc.dbIdleConns})
                        </div>
                      )}
                    </div>

                    <div className="pt-2 border-t border-slate-800/80 space-y-1">
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

                  <input
                    type="text"
                    data-testid="route-search-input"
                    value={routeSearch}
                    onChange={(e) => setRouteSearch(e.target.value)}
                    placeholder="Filter route or trace_id..."
                    className="bg-[#090d16] border border-slate-700 rounded px-2.5 py-1 text-xs font-mono text-slate-200 w-56"
                  />

                  <label className="flex items-center gap-1.5 text-xs font-mono text-slate-300 cursor-pointer">
                    <input
                      type="checkbox"
                      checked={onlyErrors}
                      onChange={(e) => setOnlyErrors(e.target.checked)}
                    />
                    Errors Only
                  </label>
                </div>
              </div>

              {/* Per-Route RED Table */}
              <div className="overflow-x-auto">
                <table className="w-full text-left border-collapse text-xs font-mono">
                  <thead>
                    <tr className="border-b border-slate-800 text-slate-400">
                      <th className="py-2 px-3">SERVICE</th>
                      <th className="py-2 px-3">REGION</th>
                      <th className="py-2 px-3">METHOD</th>
                      <th className="py-2 px-3">ROUTE</th>
                      <th className="py-2 px-3">REQS</th>
                      <th className="py-2 px-3">ERRORS</th>
                      <th className="py-2 px-3">AVAILABILITY</th>
                      <th className="py-2 px-3">P50 RTT</th>
                      <th className="py-2 px-3">P95 RTT</th>
                      <th className="py-2 px-3">P99 RTT</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-slate-800/60">
                    {flattenedRoutes.map((r) => (
                      <tr
                        key={`${r.serviceName}-${r.method}-${r.route}`}
                        className="hover:bg-slate-900/50"
                      >
                        <td className="py-2 px-3 text-sky-300">{r.serviceName}</td>
                        <td className="py-2 px-3 text-slate-400">{r.region}</td>
                        <td className="py-2 px-3 text-slate-300">{r.method}</td>
                        <td className="py-2 px-3 text-slate-100">{r.route}</td>
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
                    {flattenedRoutes.length === 0 && (
                      <tr>
                        <td colSpan={10} className="py-4 text-center text-slate-500">
                          No route metrics recorded for current filter.
                        </td>
                      </tr>
                    )}
                  </tbody>
                </table>
              </div>

              {/* Traced Requests Table */}
              <div className="pt-4 border-t border-slate-800">
                <h3 className="text-xs font-mono font-bold uppercase tracking-wider text-slate-400 mb-2">
                  Recent Traced Requests (VictoriaLogs / Ring Buffer)
                </h3>
                <div className="overflow-x-auto">
                  <table
                    data-testid="route-requests-table"
                    className="w-full text-left border-collapse text-xs font-mono"
                  >
                    <thead>
                      <tr className="border-b border-slate-800 text-slate-400">
                        <th className="py-2 px-3">OCCURRED AT</th>
                        <th className="py-2 px-3">TRACE ID (UUIDv7)</th>
                        <th className="py-2 px-3">REGION</th>
                        <th className="py-2 px-3">SERVICE</th>
                        <th className="py-2 px-3">METHOD</th>
                        <th className="py-2 px-3">ROUTE</th>
                        <th className="py-2 px-3">STATUS</th>
                        <th className="py-2 px-3">RTT</th>
                      </tr>
                    </thead>
                    <tbody className="divide-y divide-slate-800/60">
                      {routeRequests.map((req, idx) => (
                        <tr
                          key={`${req.traceId}-${req.serviceName}-${idx}`}
                          className="hover:bg-slate-900/50"
                        >
                          <td className="py-2 px-3 text-slate-400">
                            {formatTimestamp(req.occurredAt)}
                          </td>
                          <td className="py-2 px-3 text-amber-300">{req.traceId}</td>
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
                      {routeRequests.length === 0 && (
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

            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              {tasks.map((t) => (
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

            <div className="overflow-x-auto">
              <table className="w-full text-left border-collapse text-xs font-mono">
                <thead>
                  <tr className="border-b border-slate-800 text-slate-400">
                    <th className="py-2 px-3">JOB ID</th>
                    <th className="py-2 px-3">TRACE ID</th>
                    <th className="py-2 px-3">REGION</th>
                    <th className="py-2 px-3">TASK</th>
                    <th className="py-2 px-3">CALLER</th>
                    <th className="py-2 px-3">STATUS</th>
                    <th className="py-2 px-3">DURATION</th>
                    <th className="py-2 px-3">CREATED AT</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-slate-800/60">
                  {jobs.map((j) => (
                    <tr
                      key={j.uuid}
                      onClick={() => setSelectedJob(j)}
                      className="hover:bg-slate-900/50 cursor-pointer"
                    >
                      <td className="py-2 px-3 text-sky-300 font-semibold">{j.jobId}</td>
                      <td className="py-2 px-3 text-amber-300">{j.traceId}</td>
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
                  {jobs.length === 0 && (
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
                    Job Inspector: {selectedJob.jobId} (Trace: {selectedJob.traceId})
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

            <div className="overflow-x-auto">
              <table className="w-full text-left border-collapse text-xs font-mono">
                <thead>
                  <tr className="border-b border-slate-800 text-slate-400">
                    <th className="py-2 px-3">TABLE NAME</th>
                    <th className="py-2 px-3">LIVE ROWS</th>
                    <th className="py-2 px-3">DISK SIZE</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-slate-800/60">
                  {(dbStats?.tables ?? []).map((tbl) => (
                    <tr key={tbl.tableName} className="hover:bg-slate-900/50">
                      <td className="py-2 px-3 text-slate-100">{tbl.tableName}</td>
                      <td className="py-2 px-3 text-sky-300">{tbl.rowCount.toString()}</td>
                      <td className="py-2 px-3 text-slate-300">
                        {formatBytes(tbl.totalSizeBytes)}
                      </td>
                    </tr>
                  ))}
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

              <div className="overflow-x-auto">
                <table className="w-full text-left border-collapse text-xs font-mono">
                  <thead>
                    <tr className="border-b border-slate-800 text-slate-400">
                      <th className="py-2 px-3">ID</th>
                      <th className="py-2 px-3">NAME</th>
                      <th className="py-2 px-3">TYPE</th>
                      <th className="py-2 px-3">GROUP</th>
                      <th className="py-2 px-3">EXPIRES AT</th>
                      <th className="py-2 px-3">STATUS</th>
                      <th className="py-2 px-3">ACTION</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-slate-800/60">
                    {tokens.map((tok) => (
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
              <div className="overflow-x-auto">
                <table className="w-full text-left border-collapse text-xs font-mono">
                  <thead>
                    <tr className="border-b border-slate-800 text-slate-400">
                      <th className="py-2 px-3">TIMESTAMP</th>
                      <th className="py-2 px-3">TRACE ID</th>
                      <th className="py-2 px-3">REGION</th>
                      <th className="py-2 px-3">CALLER</th>
                      <th className="py-2 px-3">GROUP</th>
                      <th className="py-2 px-3">METHOD</th>
                      <th className="py-2 px-3">JIT GRANT</th>
                      <th className="py-2 px-3">STATUS</th>
                      <th className="py-2 px-3">DURATION</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-slate-800/60">
                    {auditCalls.map((c) => (
                      <tr key={c.uuid} className="hover:bg-slate-900/50">
                        <td className="py-2 px-3 text-slate-400">{formatTimestamp(c.createdAt)}</td>
                        <td className="py-2 px-3 text-amber-300">{c.traceId}</td>
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
    </div>
  );
}
