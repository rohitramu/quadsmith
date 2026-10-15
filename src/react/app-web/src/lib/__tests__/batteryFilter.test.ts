// @vitest-environment node
import { describe, it, expect } from "vitest";
import { buildBatteryCelFilter, formatBatteryFilterForDisplay } from "@quadsmith/shared";

describe("buildBatteryCelFilter", () => {
  it("returns empty string when limits is null or undefined", () => {
    expect(buildBatteryCelFilter(null)).toBe("");
    expect(buildBatteryCelFilter(undefined)).toBe("");
  });

  it("builds CEL filter with voltage and current constraints", () => {
    const filter = buildBatteryCelFilter({
      minVoltage: 14.8,
      maxVoltage: 25.2,
      maxCurrentA: 45.0,
    });
    expect(filter).toBe("min_voltage >= 14.80 && max_voltage <= 25.20 && max_current_a >= 45.00");
  });

  it("includes single connector when specified", () => {
    const filter = buildBatteryCelFilter({
      minVoltage: 14.8,
      maxVoltage: 25.2,
      maxCurrentA: 45.0,
      connector: "XT60",
      compatibleConnectors: ["XT60"],
    });
    expect(filter).toBe(
      'min_voltage >= 14.80 && max_voltage <= 25.20 && max_current_a >= 45.00 && connector == "XT60"',
    );
  });

  it("includes multiple compatible connectors with OR group", () => {
    const filter = buildBatteryCelFilter({
      minVoltage: 3.0,
      maxVoltage: 4.35,
      maxCurrentA: 15.0,
      connector: "BT2.0",
      compatibleConnectors: ["BT2.0", "A30", "PH2.0"],
    });
    expect(filter).toBe(
      'min_voltage >= 3.00 && max_voltage <= 4.35 && max_current_a >= 15.00 && (connector == "BT2.0" || connector == "A30" || connector == "PH2.0")',
    );
  });

  it("adds a newline after each joining && when multiline option is true", () => {
    const filter = buildBatteryCelFilter(
      {
        minVoltage: 14.8,
        maxVoltage: 25.2,
        maxCurrentA: 45.0,
        connector: "XT60",
      },
      { multiline: true },
    );
    expect(filter).toBe(
      'min_voltage >= 14.80 &&\nmax_voltage <= 25.20 &&\nmax_current_a >= 45.00 &&\nconnector == "XT60"',
    );
  });
});

describe("formatBatteryFilterForDisplay", () => {
  it("returns empty string for null or empty input", () => {
    expect(formatBatteryFilterForDisplay(null)).toBe("");
    expect(formatBatteryFilterForDisplay("")).toBe("");
  });

  it("adds a newline after each joining &&", () => {
    const input =
      'min_voltage >= 14.80 && max_voltage <= 25.20 && max_current_a >= 45.00 && connector == "XT60"';
    const formatted = formatBatteryFilterForDisplay(input);
    expect(formatted).toBe(
      'min_voltage >= 14.80 &&\nmax_voltage <= 25.20 &&\nmax_current_a >= 45.00 &&\nconnector == "XT60"',
    );
  });
});
