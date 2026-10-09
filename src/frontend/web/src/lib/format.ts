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

/**
 * Returns human-readable flight discipline/handling description for a given thrust-to-weight ratio.
 */
export function getTwrDescription(twr: number): string {
  if (twr >= 8.0) return "Competition Racing";
  if (twr >= 5.5) return "Freestyle Acro";
  if (twr >= 4.0) return "Sport & Toothpick";
  if (twr >= 2.8) return "Long Range Cruiser";
  if (twr >= 1.8) return "Cinelifter & Heavy Payload";
  if (twr >= 1.0) return "Sluggish / Underpowered";
  return "Cannot Take Off";
}

/**
 * Formats a product's display title including manufacturer if available, like "<manufacturer> - <product_name>".
 * Strips redundant manufacturer prefix if the product name already begins with it.
 *
 * Examples:
 *   ("EMAX", "ECO II 2207") -> "EMAX - ECO II 2207"
 *   ("Sub250", "Sub250 1404 4500KV") -> "Sub250 - 1404 4500KV"
 *   ("Foxeer", "Foxeer") -> "Foxeer"
 *   (null, "ECO II 2207") -> "ECO II 2207"
 */
export function formatProductTitle(
  manufacturer?: string | null,
  name?: string | null,
  fallback: string = "",
): string {
  const rawName = (name || fallback || "").trim();
  const mfg = (manufacturer || "").trim();

  if (!mfg) {
    return rawName;
  }

  if (!rawName) {
    return mfg;
  }

  let cleanName = rawName;
  if (rawName.toLowerCase().startsWith(mfg.toLowerCase())) {
    const rest = rawName
      .slice(mfg.length)
      .replace(/^[\s\-:]+/, "")
      .trim();
    if (rest) {
      cleanName = rest;
    }
  }

  return cleanName.toLowerCase() === mfg.toLowerCase() ? mfg : `${mfg} - ${cleanName}`;
}
