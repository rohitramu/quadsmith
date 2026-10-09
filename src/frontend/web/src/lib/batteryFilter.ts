export interface ElectricalLimits {
  minVoltage?: number;
  maxVoltage?: number;
  maxCurrentA?: number;
  defaultBatteryId?: string;
}

/**
 * Constructs a CEL filter expression for compatible batteries based on build electrical limits.
 *
 * Example: "min_voltage >= 14.80 && max_voltage <= 25.20 && max_current_a >= 39.40"
 */
export function buildBatteryCelFilter(limits?: ElectricalLimits | null): string {
  if (!limits) return "";

  const conditions: string[] = [];

  if (limits.minVoltage !== undefined && limits.minVoltage > 0) {
    conditions.push(`min_voltage >= ${limits.minVoltage.toFixed(2)}`);
  }
  if (limits.maxVoltage !== undefined && limits.maxVoltage > 0) {
    conditions.push(`max_voltage <= ${limits.maxVoltage.toFixed(2)}`);
  }
  if (limits.maxCurrentA !== undefined && limits.maxCurrentA > 0) {
    conditions.push(`max_current_a >= ${limits.maxCurrentA.toFixed(2)}`);
  }

  return conditions.join(" && ");
}
