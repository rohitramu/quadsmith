import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { SmartFilterInput, filterAndSortRecords } from "@quadsmith/shared";
import App, {
  ROUTE_METRIC_FIELDS,
  ROUTE_METRIC_PRESETS,
  ROUTE_METRIC_SYNTAX,
  getStoredOpsToken,
  setOpsCookieAndStorage,
} from "../App.tsx";

describe("Ops Console App", () => {
  it("renders authentication screen when no operator token is stored", () => {
    setOpsCookieAndStorage(null);
    render(<App />);
    expect(screen.getByText(/Ops & Health Monitoring Console/i)).toBeInTheDocument();
    expect(screen.getByTestId("sandbox-login-btn")).toBeInTheDocument();
  });

  it("stores and clears qs_ops_token in localStorage and cookie", () => {
    setOpsCookieAndStorage("qs_op_test_token_999");
    expect(getStoredOpsToken()).toBe("qs_op_test_token_999");
    expect(document.cookie).toContain("qs_ops_token=qs_op_test_token_999");

    setOpsCookieAndStorage(null);
    expect(getStoredOpsToken()).toBe("");
  });

  it("evaluates CEL filter expressions and multi-type sorting for Ops collections", () => {
    const sampleRoutes = [
      {
        service_name: "server-api",
        route: "/quadsmith.api.AntennaService/ListAntennas",
        total_requests: 120,
        error_count: 0,
        p95_rtt_ms: 4.2,
      },
      {
        service_name: "server-ops-api",
        route: "/quadsmith.ops.OpsService/GetSystemMetrics",
        total_requests: 45,
        error_count: 2,
        p95_rtt_ms: 18.5,
      },
      {
        service_name: "server-api",
        route: "/quadsmith.api.MotorService/ListMotors",
        total_requests: 300,
        error_count: 1,
        p95_rtt_ms: 12.1,
      },
    ];

    const res = filterAndSortRecords(
      sampleRoutes,
      (r) => r,
      'service_name == "server-api" && total_requests >= 100',
      "^total_requests",
      ROUTE_METRIC_FIELDS,
    );
    expect(res.error).toBeNull();
    expect(res.items).toHaveLength(2);
    expect(res.items[0]?.route).toContain("ListMotors");
    expect(res.items[1]?.route).toContain("ListAntennas");

    // String method + OR logic
    const errOrSlow = filterAndSortRecords(
      sampleRoutes,
      (r) => r,
      'route.contains("OpsService") || error_count > 0',
      "p95_rtt_ms",
      ROUTE_METRIC_FIELDS,
    );
    expect(errOrSlow.error).toBeNull();
    expect(errOrSlow.items).toHaveLength(2);
    expect(errOrSlow.items[0]?.p95_rtt_ms).toBe(12.1);
    expect(errOrSlow.items[1]?.p95_rtt_ms).toBe(18.5);

    // Unknown field validation error
    const badField = filterAndSortRecords(
      sampleRoutes,
      (r) => r,
      "non_existent_metric > 10",
      "",
      ROUTE_METRIC_FIELDS,
    );
    expect(badField.error).toContain("Unknown field");
  });

  it("renders SmartFilterInput with CEL syntax helper and clickable field pills", () => {
    const onApply = vi.fn();
    render(
      <SmartFilterInput
        value=""
        onApply={onApply}
        fields={ROUTE_METRIC_FIELDS}
        presets={ROUTE_METRIC_PRESETS}
        syntaxExamples={ROUTE_METRIC_SYNTAX}
        inputTestId="test-cel-input"
        helpTestId="test-cel-help"
      />,
    );

    // Quick preset chip triggers onApply
    fireEvent.click(screen.getByText("Errors Only"));
    expect(onApply).toHaveBeenCalledWith("error_count > 0");

    // Toggle CEL Syntax Helper guide
    fireEvent.click(screen.getByTestId("test-cel-help"));
    expect(
      screen.getByText(/Common Expression Language \(CEL\) Filter Syntax/i),
    ).toBeInTheDocument();
    expect(screen.getByText(/Available Fields for this Collection/i)).toBeInTheDocument();
    expect(screen.getByText("p95_rtt_ms")).toBeInTheDocument();
  });

  it("renders sticky top bar with unnumbered navbar, full-width layout, Service Health as the first tab, Per-Route Metrics and Traced Requests tabs, and separate Route and Attributes columns in Trace & Debug Logs with selectable columns", async () => {
    const traceId = "0195f0a1-1234-7000-8000-abcdef012345";
    const otherApiTraceId = "0195f0a1-8888-7000-8000-000000000009";
    const healthTraceId = "0195f0a1-9999-7000-8000-000000000001";
    const opsTraceId = "0195f0a1-9999-7000-8000-000000000002";
    const fetchSpy = vi.spyOn(globalThis, "fetch").mockImplementation(async (input) => {
      const url = typeof input === "string" ? input : input instanceof URL ? input.href : input.url;
      let payload: Record<string, unknown> = {};
      if (url.endsWith("/GetCurrentOperator")) {
        payload = {
          operator: "rohitramu",
          operatorType: "HUMAN",
          groupName: "admins",
        };
      } else if (url.endsWith("/GetSystemHealth")) {
        payload = { services: [] };
      } else if (url.endsWith("/GetSystemMetrics")) {
        payload = {
          window: "5m",
          services: [
            {
              serviceName: "server-api",
              region: "local",
              routes: [
                {
                  serviceName: "server-api",
                  region: "local",
                  method: "POST",
                  route: "/quadsmith.api.MotorService/ListMotors",
                  totalRequests: "42",
                  successCount: "42",
                  errorCount: "0",
                  availabilityPercent: 100,
                  p50RttMs: 1.5,
                  p95RttMs: 3.2,
                  p99RttMs: 4.1,
                },
                {
                  serviceName: "server-api",
                  region: "local",
                  method: "POST",
                  route: "/quadsmith.api.FrameService/ListFrames",
                  totalRequests: "18",
                  successCount: "18",
                  errorCount: "0",
                  availabilityPercent: 100,
                  p50RttMs: 1.2,
                  p95RttMs: 2.4,
                  p99RttMs: 3.1,
                },
                {
                  serviceName: "server-api",
                  region: "local",
                  method: "GET",
                  route: "/healthz",
                  totalRequests: "500",
                  successCount: "500",
                  errorCount: "0",
                  availabilityPercent: 100,
                  p50RttMs: 0.2,
                  p95RttMs: 0.5,
                  p99RttMs: 0.8,
                },
              ],
            },
            {
              serviceName: "server-ops-api",
              region: "local",
              routes: [
                {
                  serviceName: "server-ops-api",
                  region: "local",
                  method: "POST",
                  route: "/quadsmith.ops.OpsService/GetSystemHealth",
                  totalRequests: "10",
                  successCount: "10",
                  errorCount: "0",
                  availabilityPercent: 100,
                  p50RttMs: 2.0,
                  p95RttMs: 4.0,
                  p99RttMs: 5.0,
                },
              ],
            },
          ],
        };
      } else if (url.endsWith("/ListRouteRequests")) {
        payload = {
          requests: [
            {
              traceId,
              region: "local",
              serviceName: "server-api",
              method: "POST",
              route: "/quadsmith.api.MotorService/ListMotors",
              statusCode: 200,
              rttMs: 4.75,
              success: true,
              occurredAt: "2026-10-18T12:00:00Z",
            },
            {
              traceId: otherApiTraceId,
              region: "local",
              serviceName: "server-api",
              method: "POST",
              route: "/quadsmith.api.FrameService/ListFrames",
              statusCode: 200,
              rttMs: 2.1,
              success: true,
              occurredAt: "2026-10-18T12:00:00.500Z",
            },
            {
              traceId: healthTraceId,
              region: "local",
              serviceName: "server-api",
              method: "GET",
              route: "/healthz",
              statusCode: 200,
              rttMs: 0.3,
              success: true,
              occurredAt: "2026-10-18T12:00:01Z",
            },
            {
              traceId: opsTraceId,
              region: "local",
              serviceName: "server-ops-api",
              method: "POST",
              route: "/quadsmith.ops.OpsService/CleanupExpiredOtps",
              statusCode: 200,
              rttMs: 3.1,
              success: true,
              occurredAt: "2026-10-18T12:00:02Z",
            },
          ],
        };
      } else if (url.endsWith("/GetTrace")) {
        payload = {
          traceId,
          region: "local",
          rootService: "server-api",
          rootRoute: "/quadsmith.api.MotorService/ListMotors",
          rootMethod: "POST",
          statusCode: 200,
          totalRttMs: 4.75,
          success: true,
          spans: [
            {
              spanId: "span_server-api_1",
              serviceName: "server-api",
              instanceId: "server-api-1",
              region: "local",
              operation: "/quadsmith.api.MotorService/ListMotors",
              method: "POST",
              statusCode: 200,
              durationMs: 4.75,
              success: true,
              occurredAt: "2026-10-18T12:00:00Z",
              attributes: {
                hop_type: "ingress",
              },
            },
          ],
          logs: [
            {
              timestamp: "2026-10-18T12:00:00Z",
              level: "DEBUG",
              serviceName: "server-api",
              instanceId: "server-api-1",
              region: "local",
              message: "trace_context_initialized",
              attributes: {
                method: "POST",
                route: "/quadsmith.api.MotorService/ListMotors",
              },
            },
            {
              timestamp: "2026-10-18T12:00:00.001Z",
              level: "DEBUG",
              serviceName: "server-api",
              instanceId: "server-api-1",
              region: "local",
              message: "db_query",
              attributes: {
                database: "quadsmith_api",
                sql: "SELECT uuid, id, brand, model FROM motors ORDER BY id ASC",
                rows_affected: "12",
                duration_ms: "1.15",
              },
            },
            {
              timestamp: "2026-10-18T12:00:00.004Z",
              level: "INFO",
              serviceName: "server-api",
              instanceId: "server-api-1",
              region: "local",
              message: "http_request",
              attributes: {
                method: "POST",
                route: "/quadsmith.api.MotorService/ListMotors",
                status_code: "200",
                rtt_ms: "4.75",
              },
            },
          ],
        };
      }
      return new Response(JSON.stringify(payload), {
        status: 200,
        headers: { "Content-Type": "application/json" },
      });
    });

    try {
      setOpsCookieAndStorage("qs_op_sandbox_token_12345");
      render(<App />);

      // Verify sticky top bar, unnumbered navigation bar, and full-width layout
      const stickyBar = await screen.findByTestId("ops-sticky-top-bar");
      expect(stickyBar.className).toContain("sticky");
      expect(stickyBar.className).toContain("top-0");

      const navbar = screen.getByTestId("ops-main-navbar");
      expect(navbar).toBeInTheDocument();
      expect(screen.getByTestId("tab-health")).toHaveTextContent("Service Health & Route Metrics");
      expect(screen.getByTestId("tab-tasks")).toHaveTextContent("Background Tasks & Triggers");
      expect(screen.getByTestId("tab-jobs")).toHaveTextContent("Job Execution History");
      expect(screen.getByTestId("tab-db")).toHaveTextContent("Public DB Statistics");
      expect(screen.getByTestId("tab-iam")).toHaveTextContent("IAM, Tokens & Audit Log");

      const mainContent = screen.getByTestId("ops-main-content");
      expect(mainContent.className).toContain("w-full");
      expect(mainContent.className).not.toContain("max-w-7xl");

      // Tab 1 (Service Health) is active by default and unnumbered
      expect(screen.getByTestId("telemetry-tab-health")).toHaveAttribute("aria-selected", "true");
      expect(screen.getByTestId("telemetry-tab-health")).toHaveTextContent("Service Health");
      expect(screen.getByTestId("telemetry-tab-routes")).toHaveAttribute("aria-selected", "false");
      expect(screen.getByTestId("telemetry-tab-routes")).toHaveTextContent("Per-Route Metrics");
      expect(screen.getByTestId("telemetry-tab-traces")).toHaveAttribute("aria-selected", "false");
      expect(screen.getByTestId("telemetry-tab-traces")).toHaveTextContent(
        "Recent Traced Requests",
      );
      expect(screen.getByTestId("telemetry-tab-trace-detail")).toHaveAttribute(
        "aria-selected",
        "false",
      );
      expect(screen.getByTestId("telemetry-tab-trace-detail")).toHaveTextContent(
        "Trace & Debug Logs",
      );
      expect(screen.getByTestId("telemetry-panel-health")).toBeInTheDocument();
      expect(screen.getByTestId("health-column-selector")).toBeInTheDocument();

      // Switch to Tab 2 (Per-Route Metrics)
      fireEvent.click(screen.getByTestId("telemetry-tab-routes"));
      expect(screen.getByTestId("telemetry-tab-routes")).toHaveAttribute("aria-selected", "true");
      const routeBtn = await screen.findByTestId(
        "select-route-/quadsmith.api.MotorService/ListMotors",
      );

      const routesTable = screen.getByTestId("routes-red-table");
      expect(routesTable).toHaveTextContent("/quadsmith.api.MotorService/ListMotors");
      expect(routesTable).toHaveTextContent("/quadsmith.api.FrameService/ListFrames");
      expect(routesTable).not.toHaveTextContent("/healthz");
      expect(routesTable).not.toHaveTextContent("/quadsmith.ops.OpsService/GetSystemHealth");

      // Toggle a column in Per-Route Metrics using the ColumnSelector
      expect(routesTable).toHaveTextContent("REGION");
      fireEvent.click(screen.getByTestId("routes-col-toggle-region"));
      expect(routesTable).not.toHaveTextContent("REGION");

      // Clicking a route switches to Tab 3 (Recent Traced Requests) and adds the route filter
      fireEvent.click(routeBtn);

      expect(screen.getByTestId("telemetry-tab-traces")).toHaveAttribute("aria-selected", "true");
      const tracesFilterInput = screen.getByTestId("traces-filter-input") as HTMLInputElement;
      expect(tracesFilterInput.value).toContain(
        'route == "/quadsmith.api.MotorService/ListMotors"',
      );

      const traceBtn = await screen.findByTestId(`inspect-trace-${traceId}`);
      expect(screen.queryByTestId(`inspect-trace-${otherApiTraceId}`)).not.toBeInTheDocument();
      expect(screen.queryByTestId(`inspect-trace-${healthTraceId}`)).not.toBeInTheDocument();
      expect(screen.queryByTestId(`inspect-trace-${opsTraceId}`)).not.toBeInTheDocument();

      // Clicking a trace ID switches to Tab 4 (Trace & Debug Logs)
      fireEvent.click(traceBtn);

      expect(screen.getByTestId("telemetry-tab-trace-detail")).toHaveAttribute(
        "aria-selected",
        "true",
      );
      const tracePanel = await screen.findByTestId("trace-inspector-panel");
      expect(tracePanel).toBeInTheDocument();
      expect(screen.getByTestId("trace-inspector-id")).toHaveTextContent(traceId);
      expect(screen.getByTestId("trace-debug-logs-count")).toHaveTextContent("2");
      expect(screen.getByTestId("trace-spans-list")).toHaveTextContent(
        "/quadsmith.api.MotorService/ListMotors",
      );
      const logsTable = screen.getByTestId("trace-logs-table");
      expect(logsTable).toHaveTextContent("trace_context_initialized");
      expect(logsTable).toHaveTextContent("db_query");
      expect(logsTable).toHaveTextContent("http_request");

      // Verify ROUTE and ATTRIBUTES are separate columns, ATTRIBUTES is the last column,
      // and ATTRIBUTES wraps at natural word breaks (whitespace-normal break-words) without ellipsis
      const logHeaders = Array.from(logsTable.querySelectorAll("thead th")).map((th) =>
        th.textContent?.trim(),
      );
      expect(logHeaders).toEqual([
        "TIMESTAMP",
        "SERVICE",
        "LEVEL",
        "EVENT",
        "ROUTE",
        "STATUS / RTT",
        "ATTRIBUTES",
      ]);
      const dbQueryAttrsCell = screen.getByTestId("trace-log-attributes-1");
      expect(dbQueryAttrsCell.className).toContain("whitespace-normal");
      expect(dbQueryAttrsCell.className).toContain("break-words");
      expect(dbQueryAttrsCell.className).not.toContain("break-all");
      expect(dbQueryAttrsCell.className).not.toContain("truncate");
      expect(dbQueryAttrsCell).toHaveTextContent(
        "SELECT uuid, id, brand, model FROM motors ORDER BY id ASC",
      );

      // Toggle ATTRIBUTES column via trace-logs-column-selector
      fireEvent.click(screen.getByTestId("trace-logs-col-toggle-attributes"));
      const updatedHeaders = Array.from(logsTable.querySelectorAll("thead th")).map((th) =>
        th.textContent?.trim(),
      );
      expect(updatedHeaders).toEqual([
        "TIMESTAMP",
        "SERVICE",
        "LEVEL",
        "EVENT",
        "ROUTE",
        "STATUS / RTT",
      ]);

      // Clicking Back to Traced Requests switches back to Recent Traced Requests
      fireEvent.click(screen.getByTestId("close-trace-inspector"));
      expect(screen.getByTestId("telemetry-tab-traces")).toHaveAttribute("aria-selected", "true");
      expect(screen.queryByTestId("trace-inspector-panel")).not.toBeInTheDocument();

      // Verify pagination bar and historical lookback controls are rendered on Recent Traced Requests
      expect(screen.getByTestId("traces-pagination-bar")).toBeInTheDocument();
      expect(screen.getByTestId("traces-page-indicator")).toHaveTextContent("Page 1 of 1");
      expect(screen.getByTestId("traces-history-limit-select")).toHaveValue("500");
      expect(screen.getByTestId("traces-page-size-select")).toHaveValue("25");
      expect(screen.getByTestId("traces-prev-page-btn")).toBeDisabled();
      expect(screen.getByTestId("traces-next-page-btn")).toBeDisabled();

      // Clear the route filter so both API requests show up (2 requests total)
      const remountedTracesFilterInput = screen.getByTestId(
        "traces-filter-input",
      ) as HTMLInputElement;
      fireEvent.change(remountedTracesFilterInput, {
        target: { value: 'service_name == "server-api" && !route.startsWith("/healthz")' },
      });
      fireEvent.click(screen.getByTestId("traces-cel-apply-btn"));
      await screen.findByTestId(`inspect-trace-${otherApiTraceId}`);
    } finally {
      fetchSpy.mockRestore();
      setOpsCookieAndStorage(null);
    }
  });

  it("renders JIT-gated Log in as @quadsmith button and triggers LoginAsQuadsmith when elevated", async () => {
    const openSpy = vi.spyOn(window, "open").mockImplementation(() => null);
    const fetchSpy = vi.spyOn(globalThis, "fetch").mockImplementation(async (input) => {
      const url = typeof input === "string" ? input : input instanceof URL ? input.href : input.url;
      let payload: Record<string, unknown> = {};
      if (url.endsWith("/GetCurrentOperator")) {
        payload = {
          operator: "rohitramu",
          operatorType: "HUMAN",
          groupName: "admins",
          activeJitGrant: {
            id: "jit_test_1",
            operator: "rohitramu",
            hours: 2,
            reason: "Testing @quadsmith login",
            status: "ACTIVE",
            operation: "*",
          },
        };
      } else if (url.endsWith("/LoginAsQuadsmith")) {
        payload = {
          accessToken: "sys_access_token_123",
          refreshToken: "sys_refresh_token_456",
          username: "quadsmith",
          userUuid: "018f0000-0000-7000-8000-000000000000",
          role: "SYSTEM",
          sessionUuid: "sys_session_789",
        };
      }
      return new Response(JSON.stringify(payload), {
        status: 200,
        headers: { "Content-Type": "application/json" },
      });
    });

    try {
      setOpsCookieAndStorage("qs_op_sandbox_token_12345");
      render(<App />);

      const loginBtn = await screen.findByTestId("login-quadsmith-btn");
      expect(loginBtn).toHaveTextContent(/Log in as @quadsmith/i);
      fireEvent.click(loginBtn);

      await vi.waitFor(() => {
        expect(openSpy).toHaveBeenCalledWith(
          expect.stringContaining("ops_refresh_token=sys_refresh_token_456"),
          "_blank",
          "noopener,noreferrer",
        );
      });
    } finally {
      openSpy.mockRestore();
      fetchSpy.mockRestore();
      setOpsCookieAndStorage(null);
    }
  });

  it("renders vertical service health cards with direct /healthz and /metrics links", async () => {
    const fetchSpy = vi.spyOn(globalThis, "fetch").mockImplementation(async (input) => {
      const url = typeof input === "string" ? input : input instanceof URL ? input.href : input.url;
      let payload: Record<string, unknown> = {};
      if (url.endsWith("/GetCurrentOperator")) {
        payload = {
          operator: "rohitramu",
          operatorType: "HUMAN",
          groupName: "admins",
        };
      } else if (url.endsWith("/GetSystemHealth")) {
        payload = {
          status: "HEALTHY",
          region: "local",
          environment: "sandbox",
          services: [
            {
              serviceName: "server-api",
              region: "local",
              instanceId: "server-api-local",
              status: "HEALTHY",
              uptimeSeconds: "120",
              numGoroutines: 14,
              dbAcquiredConns: 2,
              dbMaxConns: 25,
              dbIdleConns: 4,
              checks: [{ name: "postgres:quadsmith_api", status: "HEALTHY", latencyMs: 1.2 }],
            },
            {
              serviceName: "server-cron",
              region: "local",
              instanceId: "server-cron-local",
              status: "HEALTHY",
              uptimeSeconds: "120",
              numGoroutines: 8,
              checks: [{ name: "scheduler_loop", status: "HEALTHY", latencyMs: 0.1 }],
            },
          ],
        };
      } else if (url.endsWith("/GetSystemMetrics")) {
        payload = { window: "5m", services: [] };
      }
      return new Response(JSON.stringify(payload), {
        status: 200,
        headers: { "Content-Type": "application/json" },
      });
    });

    try {
      setOpsCookieAndStorage("qs_op_sandbox_token_12345");
      render(<App />);

      const matrix = await screen.findByTestId("service-health-matrix");
      expect(matrix.className).toContain("flex-col");

      const apiHealthz = await screen.findByTestId("healthz-link-server-api");
      expect(apiHealthz).toHaveAttribute("href", "/services/server-api/healthz");

      const apiMetrics = screen.getByTestId("metrics-link-server-api");
      expect(apiMetrics).toHaveAttribute("href", "/services/server-api/metrics");

      const cronHealthz = screen.getByTestId("healthz-link-server-cron");
      expect(cronHealthz).toHaveAttribute("href", "/services/server-cron/healthz");

      const cronMetrics = screen.getByTestId("metrics-link-server-cron");
      expect(cronMetrics).toHaveAttribute("href", "/services/server-cron/metrics");
    } finally {
      fetchSpy.mockRestore();
      setOpsCookieAndStorage(null);
    }
  });
});
