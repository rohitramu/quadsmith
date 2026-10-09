export interface CollectionColorDef {
  id: string;
  name: string;
  colorName: string;
  hex: string;
  badgeClass: string;
  trimClass: string;
  dotClass: string;
  borderTopClass: string;
  borderLeftClass: string;
  textClass: string;
  iconBgClass: string;
  hoverBorderClass: string;
}

export const COLLECTION_COLORS: Record<string, CollectionColorDef> = {
  antennas: {
    id: "antennas",
    name: "Antennas",
    colorName: "amber",
    hex: "#f59e0b",
    badgeClass:
      "bg-amber-100 text-amber-800 border-amber-300 dark:bg-amber-950/60 dark:text-amber-300 dark:border-amber-800/80",
    trimClass: "bg-amber-500",
    dotClass: "bg-amber-500",
    borderTopClass: "border-t-amber-500",
    borderLeftClass: "border-l-amber-500",
    textClass: "text-amber-600 dark:text-amber-400",
    iconBgClass: "bg-amber-100 text-amber-700 dark:bg-amber-950/60 dark:text-amber-300",
    hoverBorderClass: "hover:border-amber-500",
  },
  batteries: {
    id: "batteries",
    name: "Batteries",
    colorName: "emerald",
    hex: "#10b981",
    badgeClass:
      "bg-emerald-100 text-emerald-800 border-emerald-300 dark:bg-emerald-950/60 dark:text-emerald-300 dark:border-emerald-800/80",
    trimClass: "bg-emerald-500",
    dotClass: "bg-emerald-500",
    borderTopClass: "border-t-emerald-500",
    borderLeftClass: "border-l-emerald-500",
    textClass: "text-emerald-600 dark:text-emerald-400",
    iconBgClass: "bg-emerald-100 text-emerald-700 dark:bg-emerald-950/60 dark:text-emerald-300",
    hoverBorderClass: "hover:border-emerald-500",
  },
  builds: {
    id: "builds",
    name: "Builds",
    colorName: "indigo",
    hex: "#6366f1",
    badgeClass:
      "bg-indigo-100 text-indigo-800 border-indigo-300 dark:bg-indigo-950/60 dark:text-indigo-300 dark:border-indigo-800/80",
    trimClass: "bg-indigo-500",
    dotClass: "bg-indigo-500",
    borderTopClass: "border-t-indigo-500",
    borderLeftClass: "border-l-indigo-500",
    textClass: "text-indigo-600 dark:text-indigo-400",
    iconBgClass: "bg-indigo-100 text-indigo-700 dark:bg-indigo-950/60 dark:text-indigo-300",
    hoverBorderClass: "hover:border-indigo-500",
  },
  cameras: {
    id: "cameras",
    name: "Cameras",
    colorName: "rose",
    hex: "#f43f5e",
    badgeClass:
      "bg-rose-100 text-rose-800 border-rose-300 dark:bg-rose-950/60 dark:text-rose-300 dark:border-rose-800/80",
    trimClass: "bg-rose-500",
    dotClass: "bg-rose-500",
    borderTopClass: "border-t-rose-500",
    borderLeftClass: "border-l-rose-500",
    textClass: "text-rose-600 dark:text-rose-400",
    iconBgClass: "bg-rose-100 text-rose-700 dark:bg-rose-950/60 dark:text-rose-300",
    hoverBorderClass: "hover:border-rose-500",
  },
  "electronic-speed-controllers": {
    id: "electronic-speed-controllers",
    name: "Electronic Speed Controllers",
    colorName: "orange",
    hex: "#f97316",
    badgeClass:
      "bg-orange-100 text-orange-800 border-orange-300 dark:bg-orange-950/60 dark:text-orange-300 dark:border-orange-800/80",
    trimClass: "bg-orange-500",
    dotClass: "bg-orange-500",
    borderTopClass: "border-t-orange-500",
    borderLeftClass: "border-l-orange-500",
    textClass: "text-orange-600 dark:text-orange-400",
    iconBgClass: "bg-orange-100 text-orange-700 dark:bg-orange-950/60 dark:text-orange-300",
    hoverBorderClass: "hover:border-orange-500",
  },
  "flight-controllers": {
    id: "flight-controllers",
    name: "Flight Controllers",
    colorName: "blue",
    hex: "#3b82f6",
    badgeClass:
      "bg-blue-100 text-blue-800 border-blue-300 dark:bg-blue-950/60 dark:text-blue-300 dark:border-blue-800/80",
    trimClass: "bg-blue-500",
    dotClass: "bg-blue-500",
    borderTopClass: "border-t-blue-500",
    borderLeftClass: "border-l-blue-500",
    textClass: "text-blue-600 dark:text-blue-400",
    iconBgClass: "bg-blue-100 text-blue-700 dark:bg-blue-950/60 dark:text-blue-300",
    hoverBorderClass: "hover:border-blue-500",
  },
  frames: {
    id: "frames",
    name: "Frames",
    colorName: "cyan",
    hex: "#06b6d4",
    badgeClass:
      "bg-cyan-100 text-cyan-800 border-cyan-300 dark:bg-cyan-950/60 dark:text-cyan-300 dark:border-cyan-800/80",
    trimClass: "bg-cyan-500",
    dotClass: "bg-cyan-500",
    borderTopClass: "border-t-cyan-500",
    borderLeftClass: "border-l-cyan-500",
    textClass: "text-cyan-600 dark:text-cyan-400",
    iconBgClass: "bg-cyan-100 text-cyan-700 dark:bg-cyan-950/60 dark:text-cyan-300",
    hoverBorderClass: "hover:border-cyan-500",
  },
  "gps-receivers": {
    id: "gps-receivers",
    name: "GPS Receivers",
    colorName: "teal",
    hex: "#14b8a6",
    badgeClass:
      "bg-teal-100 text-teal-800 border-teal-300 dark:bg-teal-950/60 dark:text-teal-300 dark:border-teal-800/80",
    trimClass: "bg-teal-500",
    dotClass: "bg-teal-500",
    borderTopClass: "border-t-teal-500",
    borderLeftClass: "border-l-teal-500",
    textClass: "text-teal-600 dark:text-teal-400",
    iconBgClass: "bg-teal-100 text-teal-700 dark:bg-teal-950/60 dark:text-teal-300",
    hoverBorderClass: "hover:border-teal-500",
  },
  motors: {
    id: "motors",
    name: "Motors",
    colorName: "red",
    hex: "#ef4444",
    badgeClass:
      "bg-red-100 text-red-800 border-red-300 dark:bg-red-950/60 dark:text-red-300 dark:border-red-800/80",
    trimClass: "bg-red-500",
    dotClass: "bg-red-500",
    borderTopClass: "border-t-red-500",
    borderLeftClass: "border-l-red-500",
    textClass: "text-red-600 dark:text-red-400",
    iconBgClass: "bg-red-100 text-red-700 dark:bg-red-950/60 dark:text-red-300",
    hoverBorderClass: "hover:border-red-500",
  },
  propellers: {
    id: "propellers",
    name: "Propellers",
    colorName: "sky",
    hex: "#0ea5e9",
    badgeClass:
      "bg-sky-100 text-sky-800 border-sky-300 dark:bg-sky-950/60 dark:text-sky-300 dark:border-sky-800/80",
    trimClass: "bg-sky-500",
    dotClass: "bg-sky-500",
    borderTopClass: "border-t-sky-500",
    borderLeftClass: "border-l-sky-500",
    textClass: "text-sky-600 dark:text-sky-400",
    iconBgClass: "bg-sky-100 text-sky-700 dark:bg-sky-950/60 dark:text-sky-300",
    hoverBorderClass: "hover:border-sky-500",
  },
  receivers: {
    id: "receivers",
    name: "Receivers",
    colorName: "purple",
    hex: "#a855f7",
    badgeClass:
      "bg-purple-100 text-purple-800 border-purple-300 dark:bg-purple-950/60 dark:text-purple-300 dark:border-purple-800/80",
    trimClass: "bg-purple-500",
    dotClass: "bg-purple-500",
    borderTopClass: "border-t-purple-500",
    borderLeftClass: "border-l-purple-500",
    textClass: "text-purple-600 dark:text-purple-400",
    iconBgClass: "bg-purple-100 text-purple-700 dark:bg-purple-950/60 dark:text-purple-300",
    hoverBorderClass: "hover:border-purple-500",
  },
  "video-transmitters": {
    id: "video-transmitters",
    name: "Video Transmitters",
    colorName: "fuchsia",
    hex: "#d946ef",
    badgeClass:
      "bg-fuchsia-100 text-fuchsia-800 border-fuchsia-300 dark:bg-fuchsia-950/60 dark:text-fuchsia-300 dark:border-fuchsia-800/80",
    trimClass: "bg-fuchsia-500",
    dotClass: "bg-fuchsia-500",
    borderTopClass: "border-t-fuchsia-500",
    borderLeftClass: "border-l-fuchsia-500",
    textClass: "text-fuchsia-600 dark:text-fuchsia-400",
    iconBgClass: "bg-fuchsia-100 text-fuchsia-700 dark:bg-fuchsia-950/60 dark:text-fuchsia-300",
    hoverBorderClass: "hover:border-fuchsia-500",
  },
};

export const DEFAULT_COLLECTION_COLOR: CollectionColorDef = {
  id: "default",
  name: "Collection",
  colorName: "slate",
  hex: "#64748b",
  badgeClass:
    "bg-slate-100 text-slate-800 border-slate-300 dark:bg-slate-800 dark:text-slate-300 dark:border-slate-700",
  trimClass: "bg-slate-500",
  dotClass: "bg-slate-500",
  borderTopClass: "border-t-slate-500",
  borderLeftClass: "border-l-slate-500",
  textClass: "text-slate-600 dark:text-slate-400",
  iconBgClass: "bg-slate-100 text-slate-700 dark:bg-slate-800 dark:text-slate-300",
  hoverBorderClass: "hover:border-slate-500",
};

/**
 * Normalizes any collection path, alias, plural, or singular identifier into a canonical key.
 */
export function normalizeCollectionKey(raw?: string | null): string {
  if (!raw) return "";
  let clean = raw.trim().toLowerCase();

  // Strip leading and trailing slashes
  clean = clean.replace(/^\/+|\/+$/g, "");

  // If it's a full path, take the last segment (e.g. components/hardware/batteries -> batteries)
  if (clean.includes("/")) {
    const parts = clean.split("/");
    clean = parts[parts.length - 1];
  }

  // Convert underscores to dashes
  clean = clean.replace(/_/g, "-");

  // Alias lookup
  switch (clean) {
    case "antenna":
    case "antennas":
      return "antennas";

    case "battery":
    case "batteries":
      return "batteries";

    case "build":
    case "builds":
      return "builds";

    case "camera":
    case "cameras":
      return "cameras";

    case "esc":
    case "escs":
    case "electronic-speed-controller":
    case "electronic-speed-controllers":
    case "electronicspeedcontroller":
    case "electronicspeedcontrollers":
      return "electronic-speed-controllers";

    case "fc":
    case "fcs":
    case "flight-controller":
    case "flight-controllers":
    case "flightcontroller":
    case "flightcontrollers":
      return "flight-controllers";

    case "frame":
    case "frames":
      return "frames";

    case "gps":
    case "gps-receiver":
    case "gps-receivers":
    case "gpsreceiver":
    case "gpsreceivers":
      return "gps-receivers";

    case "motor":
    case "motors":
      return "motors";

    case "propeller":
    case "propellers":
    case "prop":
    case "props":
      return "propellers";

    case "rx":
    case "rxs":
    case "receiver":
    case "receivers":
      return "receivers";

    case "vtx":
    case "vtxs":
    case "video-transmitter":
    case "video-transmitters":
    case "videotransmitter":
    case "videotransmitters":
      return "video-transmitters";

    default:
      return clean;
  }
}

/**
 * Returns the color definition associated with a collection, or the default fallback.
 */
export function getCollectionColor(rawKey?: string | null): CollectionColorDef {
  const normalized = normalizeCollectionKey(rawKey);
  if (normalized && COLLECTION_COLORS[normalized]) {
    return COLLECTION_COLORS[normalized];
  }
  return DEFAULT_COLLECTION_COLOR;
}
