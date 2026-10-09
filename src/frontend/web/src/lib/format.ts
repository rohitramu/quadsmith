/**
 * Formats motor stator diameter and height into standard stator size format.
 * Format:
 *   If diameter is a decimal number: "<diameter>x<height>"
 *   Otherwise: <minimum-2-digit-diameter><minimum-2-digit-height>
 * Examples:
 *   (22, 7) -> "2207"
 *   (14, 4) -> "1404"
 *   (7, 2)  -> "0702"
 *   (23, 6.5) -> "2306.5"
 *   (22.6, 6.5) -> "22.6x6.5"
 *   (8.5, 20) -> "8.5x20"
 */
export function formatStatorSize(
  diameter?: number | null,
  height?: number | null,
  fallback: string = "-",
): string {
  if (diameter == null || height == null) return fallback;

  if (!Number.isInteger(diameter)) {
    return `${diameter}x${height}`;
  }

  const pad = (val: number): string => {
    if (Number.isInteger(val)) {
      return val.toString().padStart(2, "0");
    }
    const [intPart, decPart] = val.toString().split(".");
    return `${intPart.padStart(2, "0")}.${decPart}`;
  };

  return `${pad(diameter)}${pad(height)}`;
}

/**
 * Extracts YouTube video ID and returns the standard embed URL.
 * Supports youtu.be, youtube.com/watch?v=, youtube.com/shorts/, and youtube.com/embed/.
 */
export function getYouTubeEmbedUrl(url?: string | null): string | null {
  if (!url) return null;
  try {
    const parsed = new URL(url);
    if (parsed.hostname.includes("youtu.be")) {
      const id = parsed.pathname.replace(/^\//, "").split("/")[0];
      return id ? `https://www.youtube.com/embed/${id}` : null;
    }
    if (parsed.pathname.startsWith("/shorts/")) {
      const id = parsed.pathname.split("/")[2];
      return id ? `https://www.youtube.com/embed/${id}` : null;
    }
    if (parsed.searchParams.has("v")) {
      const id = parsed.searchParams.get("v");
      return id ? `https://www.youtube.com/embed/${id}` : null;
    }
    if (parsed.pathname.startsWith("/embed/")) {
      return url;
    }
  } catch {
    // Return null if not a valid URL
  }
  return null;
}
