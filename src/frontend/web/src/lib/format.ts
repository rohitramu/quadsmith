/**
 * Formats motor stator diameter and height into standard stator size format.
 * Format: <minimum-2-digit-diameter><minimum-2-digit-height>
 * Examples:
 *   (22, 7) -> "2207"
 *   (14, 4) -> "1404"
 *   (7, 2)  -> "0702"
 *   (23, 6.5) -> "2306.5"
 */
export function formatStatorSize(
  diameter?: number | null,
  height?: number | null,
  fallback: string = "-"
): string {
  if (diameter == null || height == null) return fallback;

  const pad = (val: number): string => {
    if (Number.isInteger(val)) {
      return val.toString().padStart(2, "0");
    }
    const [intPart, decPart] = val.toString().split(".");
    return `${intPart.padStart(2, "0")}.${decPart}`;
  };

  return `${pad(diameter)}${pad(height)}`;
}
