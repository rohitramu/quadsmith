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

  it("opens the Full Request Trace Inspector modal when clicking a Trace ID", async () => {
    const traceId = "0195f0a1-1234-7000-8000-abcdef012345";
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
        payload = { window: "5m", services: [] };
      } else if (url.endsWith("/ListRouteRequests")) {
        payload = {
          requests: [
            {
              traceId,
              region: "local",
              serviceName: "server-ops-api",
              method: "POST",
              route: "/quadsmith.ops.OpsService/CleanupExpiredOtps",
              statusCode: 200,
              rttMs: 4.75,
              success: true,
              occurredAt: "2026-10-18T12:00:00Z",
            },
          ],
        };
      } else if (url.endsWith("/GetTrace")) {
        payload = {
          traceId,
          region: "local",
          rootService: "server-ops-api",
          rootRoute: "/quadsmith.ops.OpsService/CleanupExpiredOtps",
          rootMethod: "POST",
          statusCode: 200,
          totalRttMs: 4.75,
          success: true,
          spans: [
            {
              spanId: "span_server-ops-api_1",
              serviceName: "server-ops-api",
              instanceId: "server-ops-api-1",
              region: "local",
              operation: "/quadsmith.ops.OpsService/CleanupExpiredOtps",
              method: "POST",
              statusCode: 200,
              durationMs: 4.75,
              success: true,
              occurredAt: "2026-10-18T12:00:00Z",
              attributes: {
                hop_type: "ingress",
                caller: "rohitramu",
                caller_type: "HUMAN",
              },
            },
          ],
          logs: [
            {
              timestamp: "2026-10-18T12:00:00Z",
              level: "INFO",
              serviceName: "server-ops-api",
              instanceId: "server-ops-api-1",
              region: "local",
              message: "http_request",
              attributes: {
                method: "POST",
                route: "/quadsmith.ops.OpsService/CleanupExpiredOtps",
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
      fireEvent.click(traceBtn);

      const modal = await screen.findByTestId("trace-inspector-modal");
      expect(modal).toBeInTheDocument();
      expect(screen.getByTestId("trace-inspector-id")).toHaveTextContent(traceId);
      expect(screen.getByTestId("trace-spans-list")).toHaveTextContent(
        "/quadsmith.ops.OpsService/CleanupExpiredOtps",
      );
      expect(screen.getByTestId("trace-logs-table")).toHaveTextContent("http_request");

      fireEvent.click(screen.getByTestId("close-trace-inspector"));
      expect(screen.queryByTestId("trace-inspector-modal")).not.toBeInTheDocument();
    } finally {
      fetchSpy.mockRestore();
      setOpsCookieAndStorage(null);
    }
  });
});
