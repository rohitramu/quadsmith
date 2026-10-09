import { getOption } from "@bufbuild/protobuf";
import { frontend } from "../gen/quadsmith/_common_pb";

// Protobuf Schemas for Collections
import { AntennaSchema } from "../gen/quadsmith/antenna_pb";
import { BatterySchema } from "../gen/quadsmith/battery_pb";
import { BuildSchema } from "../gen/quadsmith/build_pb";
import { CameraSchema } from "../gen/quadsmith/camera_pb";
import { ElectronicSpeedControllerSchema } from "../gen/quadsmith/electronic_speed_controller_pb";
import { FlightControllerSchema } from "../gen/quadsmith/flight_controller_pb";
import { FrameSchema } from "../gen/quadsmith/frame_pb";
import { GpsReceiverSchema } from "../gen/quadsmith/gps_receiver_pb";
import { MotorSchema } from "../gen/quadsmith/motor_pb";
import { PropellerSchema } from "../gen/quadsmith/propeller_pb";
import { ReceiverSchema } from "../gen/quadsmith/receiver_pb";
import { VideoTransmitterSchema } from "../gen/quadsmith/video_transmitter_pb";

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

export type ColorStylePreset = Omit<CollectionColorDef, "id" | "name">;

export const COLOR_PRESETS_BY_HEX: Record<string, ColorStylePreset> = {
  "#eab308": {
    colorName: "yellow",
    hex: "#eab308",
    badgeClass:
      "bg-yellow-100 text-yellow-800 border-yellow-300 dark:bg-yellow-950/60 dark:text-yellow-300 dark:border-yellow-800/80",
    trimClass: "bg-yellow-500",
    dotClass: "bg-yellow-500",
    borderTopClass: "border-t-yellow-500",
    borderLeftClass: "border-l-yellow-500",
    textClass: "text-yellow-600 dark:text-yellow-400",
    iconBgClass: "bg-yellow-100 text-yellow-700 dark:bg-yellow-950/60 dark:text-yellow-300",
    hoverBorderClass: "hover:border-yellow-500",
  },
  "#84cc16": {
    colorName: "lime",
    hex: "#84cc16",
    badgeClass:
      "bg-lime-100 text-lime-800 border-lime-300 dark:bg-lime-950/60 dark:text-lime-300 dark:border-lime-800/80",
    trimClass: "bg-lime-500",
    dotClass: "bg-lime-500",
    borderTopClass: "border-t-lime-500",
    borderLeftClass: "border-l-lime-500",
    textClass: "text-lime-600 dark:text-lime-400",
    iconBgClass: "bg-lime-100 text-lime-700 dark:bg-lime-950/60 dark:text-lime-300",
    hoverBorderClass: "hover:border-lime-500",
  },
  "#ec4899": {
    colorName: "pink",
    hex: "#ec4899",
    badgeClass:
      "bg-pink-100 text-pink-800 border-pink-300 dark:bg-pink-950/60 dark:text-pink-300 dark:border-pink-800/80",
    trimClass: "bg-pink-500",
    dotClass: "bg-pink-500",
    borderTopClass: "border-t-pink-500",
    borderLeftClass: "border-l-pink-500",
    textClass: "text-pink-600 dark:text-pink-400",
    iconBgClass: "bg-pink-100 text-pink-700 dark:bg-pink-950/60 dark:text-pink-300",
    hoverBorderClass: "hover:border-pink-500",
  },
  "#f59e0b": {
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
  "#10b981": {
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
  "#6366f1": {
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
  "#f43f5e": {
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
  "#f97316": {
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
  "#3b82f6": {
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
  "#06b6d4": {
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
  "#14b8a6": {
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
  "#ef4444": {
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
  "#0ea5e9": {
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
  "#a855f7": {
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
  "#d946ef": {
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
  "#64748b": {
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
  },
};

export const DEFAULT_COLLECTION_COLOR: CollectionColorDef = {
  id: "default",
  name: "Collection",
  ...COLOR_PRESETS_BY_HEX["#64748b"],
};

/**
 * Resolves a CollectionColorDef by hex code or color code string.
 * Falls back to the default slate color if the code is empty, null, or unrecognized.
 */
export function getCollectionColorByCode(
  hexOrColorCode?: string | null,
  id: string = "default",
  name: string = "Collection",
): CollectionColorDef {
  let normalizedHex = (hexOrColorCode ?? "").trim().toLowerCase();
  if (normalizedHex && !normalizedHex.startsWith("#")) {
    normalizedHex = "#" + normalizedHex;
  }
  const preset = COLOR_PRESETS_BY_HEX[normalizedHex] || COLOR_PRESETS_BY_HEX["#64748b"];
  return {
    id,
    name,
    colorName: preset.colorName,
    hex: preset.hex,
    badgeClass: preset.badgeClass,
    trimClass: preset.trimClass,
    dotClass: preset.dotClass,
    borderTopClass: preset.borderTopClass,
    borderLeftClass: preset.borderLeftClass,
    textClass: preset.textClass,
    iconBgClass: preset.iconBgClass,
    hoverBorderClass: preset.hoverBorderClass,
  };
}

/**
 * Extracts the frontend color_code option directly from a protobuf schema
 * and returns the corresponding CollectionColorDef.
 * If unset or missing, returns the default slate color.
 */
export function getCollectionColorFromSchema(
  schema: any,
  id: string = "default",
  name: string = "Collection",
): CollectionColorDef {
  if (!schema) {
    return { ...DEFAULT_COLLECTION_COLOR, id, name };
  }
  try {
    const fe = getOption(schema, frontend);
    if (fe?.colorCode) {
      return getCollectionColorByCode(fe.colorCode, id, name);
    }
  } catch {
    // fallback
  }
  return { ...DEFAULT_COLLECTION_COLOR, id, name };
}

const SCHEMA_COLLECTIONS: { id: string; name: string; schema: any }[] = [
  { id: "antennas", name: "Antennas", schema: AntennaSchema },
  { id: "batteries", name: "Batteries", schema: BatterySchema },
  { id: "builds", name: "Builds", schema: BuildSchema },
  { id: "cameras", name: "Cameras", schema: CameraSchema },
  {
    id: "electronic-speed-controllers",
    name: "Electronic Speed Controllers",
    schema: ElectronicSpeedControllerSchema,
  },
  {
    id: "flight-controllers",
    name: "Flight Controllers",
    schema: FlightControllerSchema,
  },
  { id: "frames", name: "Frames", schema: FrameSchema },
  {
    id: "gps-receivers",
    name: "GPS Receivers",
    schema: GpsReceiverSchema,
  },
  { id: "motors", name: "Motors", schema: MotorSchema },
  { id: "propellers", name: "Propellers", schema: PropellerSchema },
  { id: "receivers", name: "Receivers", schema: ReceiverSchema },
  {
    id: "video-transmitters",
    name: "Video Transmitters",
    schema: VideoTransmitterSchema,
  },
];

/**
 * Map of canonical collection IDs to color definitions,
 * initialized directly from protobuf schema options as the source of truth.
 */
export const COLLECTION_COLORS: Record<string, CollectionColorDef> = Object.fromEntries(
  SCHEMA_COLLECTIONS.map(({ id, name, schema }) => [
    id,
    getCollectionColorFromSchema(schema, id, name),
  ]),
);

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
