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

  it("filters routes and traced requests to server-api (excluding /healthz and /metrics) by default and opens the Full Request Trace Inspector modal when clicking a Trace ID", async () => {
    const traceId = "0195f0a1-1234-7000-8000-abcdef012345";
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

      const traceBtn = await screen.findByTestId(`inspect-trace-${traceId}`);
      expect(screen.queryByTestId(`inspect-trace-${healthTraceId}`)).not.toBeInTheDocument();
      expect(screen.queryByTestId(`inspect-trace-${opsTraceId}`)).not.toBeInTheDocument();

      const routesTable = screen.getByTestId("routes-red-table");
      expect(routesTable).toHaveTextContent("/quadsmith.api.MotorService/ListMotors");
      expect(routesTable).not.toHaveTextContent("/quadsmith.ops.OpsService/GetSystemHealth");

      fireEvent.click(traceBtn);

      const modal = await screen.findByTestId("trace-inspector-modal");
      expect(modal).toBeInTheDocument();
      expect(screen.getByTestId("trace-inspector-id")).toHaveTextContent(traceId);
      expect(screen.getByTestId("trace-spans-list")).toHaveTextContent(
        "/quadsmith.api.MotorService/ListMotors",
      );
      expect(screen.getByTestId("trace-logs-table")).toHaveTextContent("http_request");

      fireEvent.click(screen.getByTestId("close-trace-inspector"));
      expect(screen.queryByTestId("trace-inspector-modal")).not.toBeInTheDocument();
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
