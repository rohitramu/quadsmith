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
 * Returns chromatic Tailwind color classes for a given thrust-to-weight ratio.
 */
export function getTwrColor(twr: number): {
  text: string;
  badge: string;
} {
  if (twr >= 8.0) {
    return {
      text: "text-purple-600 dark:text-purple-400",
      badge:
        "bg-purple-100 text-purple-700 dark:bg-purple-950/60 dark:text-purple-400 border-purple-200 dark:border-purple-800/50",
    };
  }
  if (twr >= 5.5) {
    return {
      text: "text-emerald-600 dark:text-emerald-400",
      badge:
        "bg-emerald-100 text-emerald-700 dark:bg-emerald-950/60 dark:text-emerald-400 border-emerald-200 dark:border-emerald-800/50",
    };
  }
  if (twr >= 4.0) {
    return {
      text: "text-cyan-600 dark:text-cyan-400",
      badge:
        "bg-cyan-100 text-cyan-700 dark:bg-cyan-950/60 dark:text-cyan-400 border-cyan-200 dark:border-cyan-800/50",
    };
  }
  if (twr >= 2.8) {
    return {
      text: "text-sky-600 dark:text-sky-400",
      badge:
        "bg-sky-100 text-sky-700 dark:bg-sky-950/60 dark:text-sky-400 border-sky-200 dark:border-sky-800/50",
    };
  }
  if (twr >= 1.8) {
    return {
      text: "text-amber-600 dark:text-amber-400",
      badge:
        "bg-amber-100 text-amber-700 dark:bg-amber-950/60 dark:text-amber-400 border-amber-200 dark:border-amber-800/50",
    };
  }
  if (twr >= 1.0) {
    return {
      text: "text-orange-600 dark:text-orange-400",
      badge:
        "bg-orange-100 text-orange-700 dark:bg-orange-950/60 dark:text-orange-400 border-orange-200 dark:border-orange-800/50",
    };
  }
  return {
    text: "text-rose-600 dark:text-rose-400",
    badge:
      "bg-rose-100 text-rose-700 dark:bg-rose-950/60 dark:text-rose-400 border-rose-200 dark:border-rose-800/50",
  };
}

/**
 * Returns chromatic Tailwind color classes and status text for a given hover throttle percentage.
 * Category 1 (< 35%): Emerald Green
 * Category 2 (35%–50%): Subtle Sky Blue
 * Category 3 (50%–65%): Amber
 * Category 4 (>= 65%): Rose Red
 */
export function getHoverThrottleColor(hover: number): {
  text: string;
  badge: string;
  bar: string;
  category: string;
  description: string;
} {
  if (hover < 35) {
    return {
      text: "text-emerald-600 dark:text-emerald-400",
      badge:
        "bg-emerald-100 text-emerald-700 dark:bg-emerald-950/60 dark:text-emerald-400 border-emerald-200 dark:border-emerald-800/50",
      bar: "bg-emerald-500",
      category: "Effortless (<35%)",
      description: "Ample thrust headroom for aggressive maneuvers & punchouts",
    };
  }
  if (hover < 50) {
    return {
      text: "text-sky-600 dark:text-sky-400",
      badge:
        "bg-sky-100 text-sky-700 dark:bg-sky-950/60 dark:text-sky-400 border-sky-200 dark:border-sky-800/50",
      bar: "bg-sky-500",
      category: "Cruising (35–50%)",
      description: "Balanced efficiency for cruising and cinematic flight",
    };
  }
  if (hover < 65) {
    return {
      text: "text-amber-600 dark:text-amber-400",
      badge:
        "bg-amber-100 text-amber-700 dark:bg-amber-950/60 dark:text-amber-400 border-amber-200 dark:border-amber-800/50",
      bar: "bg-amber-500",
      category: "Heavy (50–65%)",
      description: "Heavy payload; limited headroom for recovery from steep dives",
    };
  }
  return {
    text: "text-rose-600 dark:text-rose-400",
    badge:
      "bg-rose-100 text-rose-700 dark:bg-rose-950/60 dark:text-rose-400 border-rose-200 dark:border-rose-800/50",
    bar: "bg-rose-500",
    category: "Overloaded (≥65%)",
    description: "Critical load; motor strain and thermal saturation risk",
  };
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
