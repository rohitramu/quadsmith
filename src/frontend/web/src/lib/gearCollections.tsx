import type { HardwareCollectionDef, ColumnConfig } from "./hardwareCollections";
import { getCollectionColorFromSchema } from "./collectionColors";
import { getCollectionPath } from "./hardwareCollections";

// Protobuf Schemas
import { TransmitterSchema } from "../gen/quadsmith/transmitter_pb";
import { GogglesSchema } from "../gen/quadsmith/goggles_pb";
import { ChargerSchema } from "../gen/quadsmith/charger_pb";
import { FieldToolSchema } from "../gen/quadsmith/field_tool_pb";

// Connect Query Services
import {
  listTransmitters,
  getTransmitter,
} from "../gen/quadsmith/transmitter-TransmitterService_connectquery";
import { listGoggles, getGoggles } from "../gen/quadsmith/goggles-GogglesService_connectquery";
import { listChargers, getCharger } from "../gen/quadsmith/charger-ChargerService_connectquery";
import {
  listFieldTools,
  getFieldTool,
} from "../gen/quadsmith/field_tool-FieldToolService_connectquery";

function createStandardColumns(): Record<string, ColumnConfig> {
  return {
    id: {
      id: "id",
      title: "ID",
      renderCell: (m) => (
        <span className="relative z-20 font-mono text-xs text-zinc-700 dark:text-zinc-300 select-all">
          {m.id}
        </span>
      ),
    },
    uuid: {
      id: "uuid",
      title: "UUID",
      renderCell: (m) => (
        <span
          className="relative z-20 font-mono text-[11px] text-zinc-500 dark:text-zinc-400 select-all block truncate max-w-[140px]"
          title={m.uuid}
        >
          {m.uuid}
        </span>
      ),
    },
    manufacturer: {
      id: "manufacturer",
      title: "Manufacturer",
      renderCell: (m) => m.manufacturer || "Unknown",
    },
    name: {
      id: "name",
      title: "Name",
      renderCell: (m) => (
        <span className="font-medium text-zinc-900 dark:text-zinc-100">{m.name || m.id}</span>
      ),
    },
    weight_g: {
      id: "weight_g",
      title: "Weight (g)",
      renderCell: (m) => (m.weightG != null && m.weightG > 0 ? `${m.weightG.toFixed(2)}g` : "-"),
    },
    description: {
      id: "description",
      title: "Description",
      renderCell: (m) => m.description || "-",
    },
    primary_display_image: {
      id: "primary_display_image",
      title: "Image",
      renderCell: (m) =>
        m.primaryDisplayImage ? (
          <img
            src={m.primaryDisplayImage}
            alt={m.name || m.id}
            className="w-9 h-9 object-cover rounded-md border border-zinc-200 dark:border-zinc-800 shadow-2xs"
            loading="lazy"
          />
        ) : (
          <span className="text-zinc-400">-</span>
        ),
    },
  };
}

export const GEAR_COLLECTIONS: HardwareCollectionDef[] = [
  // 1. Transmitters / Radios
  {
    id: "transmitters",
    name: "Transmitters",
    description:
      "Handheld RC radio controllers and transmitters powering user command inputs with high-precision gimbals.",
    schema: TransmitterSchema,
    listQuery: listTransmitters,
    getQuery: getTransmitter,
    getDataList: (res) => res?.transmitters ?? [],
    defaultColumnIds: [
      "manufacturer",
      "name",
      "rf_protocol",
      "operating_system",
      "gimbal_type",
      "max_power_mw",
    ],
    fields: [
      {
        name: "id",
        type: "string",
        description: "Unique identifier",
        examples: ['id.contains("boxer")', 'id.contains("pocket")'],
      },
      {
        name: "manufacturer",
        type: "string",
        description: "Manufacturer name",
        examples: ['manufacturer == "RadioMaster"', 'manufacturer.contains("TBS")'],
      },
      {
        name: "rf_protocol",
        type: "string",
        description: "Built-in RF Protocol",
        examples: ['rf_protocol.contains("ExpressLRS")', 'rf_protocol.contains("Crossfire")'],
      },
      {
        name: "max_power_mw",
        type: "number",
        description: "Maximum RF transmit power in mW",
        examples: ["max_power_mw >= 1000", "max_power_mw == 250"],
      },
    ],
    presets: [
      { label: "ExpressLRS Radios", query: 'rf_protocol.contains("ExpressLRS")' },
      { label: "1W High Power", query: "max_power_mw >= 1000" },
      { label: "RadioMaster", query: 'manufacturer == "RadioMaster"' },
    ],
    columns: {
      ...createStandardColumns(),
      rf_protocol: {
        id: "rf_protocol",
        title: "RF Protocol",
        renderCell: (m) => m.rfProtocol || "-",
      },
      operating_system: {
        id: "operating_system",
        title: "OS",
        renderCell: (m) => m.operatingSystem || "-",
      },
      gimbal_type: {
        id: "gimbal_type",
        title: "Gimbals",
        renderCell: (m) => m.gimbalType || "-",
      },
      max_power_mw: {
        id: "max_power_mw",
        title: "Max Power (mW)",
        renderCell: (m) => (m.maxPowerMw ? `${m.maxPowerMw} mW` : "-"),
      },
      channels: {
        id: "channels",
        title: "Channels",
        renderCell: (m) => m.channels || "-",
      },
      battery_type: {
        id: "battery_type",
        title: "Battery Type",
        renderCell: (m) => m.batteryType || "-",
      },
    },
    highlights: [
      { label: "RF Protocol", value: (m) => m.rfProtocol || "-" },
      { label: "Operating System", value: (m) => m.operatingSystem || "-" },
      { label: "Gimbals", value: (m) => m.gimbalType || "-" },
      { label: "Max Power", value: (m) => (m.maxPowerMw ? `${m.maxPowerMw} mW` : "-") },
    ],
    technicalSpecs: [
      { label: "Manufacturer", value: (m) => m.manufacturer || "-" },
      { label: "Model", value: (m) => m.name || m.id },
      { label: "Weight", value: (m) => (m.weightG ? `${m.weightG.toFixed(2)}g` : "-") },
      { label: "RF Protocol", value: (m) => m.rfProtocol || "-" },
      { label: "Operating System", value: (m) => m.operatingSystem || "-" },
      { label: "Gimbal Mechanism", value: (m) => m.gimbalType || "-" },
      { label: "Max Transmit Power", value: (m) => (m.maxPowerMw ? `${m.maxPowerMw} mW` : "-") },
      { label: "Channels", value: (m) => m.channels || "-" },
      { label: "Power Supply / Battery", value: (m) => m.batteryType || "-" },
      { label: "Description", value: (m) => m.description || "-" },
    ],
  },

  // 2. FPV Goggles / Video Headsets
  {
    id: "goggles",
    name: "FPV Goggles",
    description:
      "First-person-view video headsets with high-framerate OLED/micro-OLED displays and video receivers.",
    schema: GogglesSchema,
    listQuery: listGoggles,
    getQuery: getGoggles,
    getDataList: (res) => res?.goggles ?? [],
    defaultColumnIds: [
      "manufacturer",
      "name",
      "video_protocol",
      "display_type",
      "resolution",
      "fov_deg",
    ],
    fields: [
      {
        name: "id",
        type: "string",
        description: "Unique identifier",
        examples: ['id.contains("walksnail")', 'id.contains("hdzero")'],
      },
      {
        name: "video_protocol",
        type: "string",
        description: "Supported video system",
        examples: ['video_protocol.contains("Walksnail")', 'video_protocol.contains("Analog")'],
      },
      {
        name: "display_type",
        type: "string",
        description: "Display panel technology",
        examples: ['display_type == "OLED"', 'display_type.contains("Micro-OLED")'],
      },
    ],
    presets: [
      { label: "OLED Displays", query: 'display_type.contains("OLED")' },
      { label: "Digital HD", query: '!video_protocol.contains("Analog")' },
      { label: "Wide FOV (>= 50°)", query: "fov_deg >= 50.0" },
    ],
    columns: {
      ...createStandardColumns(),
      video_protocol: {
        id: "video_protocol",
        title: "Video Protocol",
        renderCell: (m) => m.videoProtocol || "-",
      },
      display_type: {
        id: "display_type",
        title: "Display",
        renderCell: (m) => m.displayType || "-",
      },
      resolution: {
        id: "resolution",
        title: "Resolution",
        renderCell: (m) => m.resolution || "-",
      },
      fov_deg: {
        id: "fov_deg",
        title: "FOV (°)",
        renderCell: (m) => (m.fovDeg != null ? `${m.fovDeg}°` : "-"),
      },
      ipd_range_mm: {
        id: "ipd_range_mm",
        title: "IPD Range",
        renderCell: (m) => m.ipdRangeMm || "-",
      },
      dvr_supported: {
        id: "dvr_supported",
        title: "DVR",
        renderCell: (m) => (m.dvrSupported ? "Yes" : "No"),
      },
    },
    highlights: [
      { label: "Protocol", value: (m) => m.videoProtocol || "-" },
      { label: "Display", value: (m) => m.displayType || "-" },
      { label: "Resolution", value: (m) => m.resolution || "-" },
      { label: "FOV", value: (m) => (m.fovDeg != null ? `${m.fovDeg}°` : "-") },
    ],
    technicalSpecs: [
      { label: "Manufacturer", value: (m) => m.manufacturer || "-" },
      { label: "Model", value: (m) => m.name || m.id },
      { label: "Weight", value: (m) => (m.weightG ? `${m.weightG.toFixed(2)}g` : "-") },
      { label: "Video Protocol", value: (m) => m.videoProtocol || "-" },
      { label: "Display Panel", value: (m) => m.displayType || "-" },
      { label: "Display Resolution", value: (m) => m.resolution || "-" },
      { label: "Field of View (FOV)", value: (m) => (m.fovDeg != null ? `${m.fovDeg}°` : "-") },
      { label: "IPD Adjustment Range", value: (m) => m.ipdRangeMm || "-" },
      { label: "Operating Voltage", value: (m) => m.voltageRange || "-" },
      { label: "Built-in DVR", value: (m) => (m.dvrSupported ? "Yes" : "No") },
      { label: "Description", value: (m) => m.description || "-" },
    ],
  },

  // 3. Battery Chargers
  {
    id: "chargers",
    name: "Battery Chargers",
    description:
      "Smart micro-processor balance chargers and dischargers for LiPo, LiHV, and Li-ion flight packs.",
    schema: ChargerSchema,
    listQuery: listChargers,
    getQuery: getCharger,
    getDataList: (res) => res?.chargers ?? [],
    defaultColumnIds: [
      "manufacturer",
      "name",
      "channels",
      "max_charge_power_w",
      "max_charge_current_a",
      "max_cells_s",
    ],
    fields: [
      {
        name: "id",
        type: "string",
        description: "Unique identifier",
        examples: ['id.contains("m6d")', 'id.contains("q6")'],
      },
      {
        name: "channels",
        type: "number",
        description: "Number of simultaneous charging channels",
        examples: ["channels == 2", "channels == 1"],
      },
      {
        name: "max_charge_power_w",
        type: "number",
        description: "Maximum output wattage",
        examples: ["max_charge_power_w >= 500", "max_charge_power_w >= 200"],
      },
    ],
    presets: [
      { label: "Dual Channel (2 Ports)", query: "channels == 2" },
      { label: "High Power (>= 500W)", query: "max_charge_power_w >= 500.0" },
      { label: "ToolkitRC", query: 'manufacturer == "ToolkitRC"' },
    ],
    columns: {
      ...createStandardColumns(),
      channels: {
        id: "channels",
        title: "Channels",
        renderCell: (m) => (m.channels ? `${m.channels} Ch` : "-"),
      },
      max_charge_power_w: {
        id: "max_charge_power_w",
        title: "Max Power (W)",
        renderCell: (m) => (m.maxChargePowerW ? `${m.maxChargePowerW} W` : "-"),
      },
      max_charge_current_a: {
        id: "max_charge_current_a",
        title: "Max Current (A)",
        renderCell: (m) => (m.maxChargeCurrentA ? `${m.maxChargeCurrentA} A` : "-"),
      },
      max_cells_s: {
        id: "max_cells_s",
        title: "Max Cells (S)",
        renderCell: (m) => (m.maxCellsS ? `${m.maxCellsS}S` : "-"),
      },
      input_voltage_range: {
        id: "input_voltage_range",
        title: "Input Voltage",
        renderCell: (m) => m.inputVoltageRange || "-",
      },
      supported_chemistries: {
        id: "supported_chemistries",
        title: "Chemistries",
        renderCell: (m) => m.supportedChemistries || "-",
      },
    },
    highlights: [
      { label: "Channels", value: (m) => (m.channels ? `${m.channels} Ch` : "-") },
      { label: "Max Power", value: (m) => (m.maxChargePowerW ? `${m.maxChargePowerW} W` : "-") },
      {
        label: "Max Current",
        value: (m) => (m.maxChargeCurrentA ? `${m.maxChargeCurrentA} A` : "-"),
      },
      { label: "Max Cells", value: (m) => (m.maxCellsS ? `${m.maxCellsS}S` : "-") },
    ],
    technicalSpecs: [
      { label: "Manufacturer", value: (m) => m.manufacturer || "-" },
      { label: "Model", value: (m) => m.name || m.id },
      { label: "Weight", value: (m) => (m.weightG ? `${m.weightG.toFixed(2)}g` : "-") },
      { label: "Output Channels", value: (m) => (m.channels ? `${m.channels}` : "-") },
      {
        label: "Max Charging Power",
        value: (m) => (m.maxChargePowerW ? `${m.maxChargePowerW} W` : "-"),
      },
      {
        label: "Max Charging Current",
        value: (m) => (m.maxChargeCurrentA ? `${m.maxChargeCurrentA} A` : "-"),
      },
      {
        label: "Supported Cell Count",
        value: (m) => (m.maxCellsS ? `Up to ${m.maxCellsS}S` : "-"),
      },
      { label: "Input Voltage Range", value: (m) => m.inputVoltageRange || "-" },
      { label: "Supported Battery Chemistries", value: (m) => m.supportedChemistries || "-" },
      { label: "Description", value: (m) => m.description || "-" },
    ],
  },

  // 4. Field Tools & Bench Gear
  {
    id: "field-tools",
    name: "Field Tools",
    description:
      "Portable smart soldering irons, smoke stoppers, servo testers, and bench equipment for drone repair.",
    schema: FieldToolSchema,
    listQuery: listFieldTools,
    getQuery: getFieldTool,
    getDataList: (res) => res?.fieldTools ?? [],
    defaultColumnIds: ["manufacturer", "name", "tool_type", "power_source", "weight_g"],
    fields: [
      {
        name: "id",
        type: "string",
        description: "Unique identifier",
        examples: ['id.contains("soldering")', 'id.contains("shortsaver")'],
      },
      {
        name: "tool_type",
        type: "string",
        description: "Tool function / type",
        examples: ['tool_type.contains("Soldering")', 'tool_type.contains("Smoke Stopper")'],
      },
    ],
    presets: [
      { label: "Soldering Irons", query: 'tool_type.contains("Soldering")' },
      { label: "Smoke Stoppers", query: 'tool_type.contains("Smoke Stopper")' },
    ],
    columns: {
      ...createStandardColumns(),
      tool_type: {
        id: "tool_type",
        title: "Tool Type",
        renderCell: (m) => (
          <span className="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-zinc-100 dark:bg-zinc-800 text-zinc-800 dark:text-zinc-200">
            {m.toolType || "-"}
          </span>
        ),
      },
      power_source: {
        id: "power_source",
        title: "Power Source",
        renderCell: (m) => m.powerSource || "-",
      },
    },
    highlights: [
      { label: "Tool Type", value: (m) => m.toolType || "-" },
      { label: "Power Source", value: (m) => m.powerSource || "-" },
      { label: "Weight", value: (m) => (m.weightG ? `${m.weightG.toFixed(2)}g` : "-") },
    ],
    technicalSpecs: [
      { label: "Manufacturer", value: (m) => m.manufacturer || "-" },
      { label: "Tool Name", value: (m) => m.name || m.id },
      { label: "Tool Type", value: (m) => m.toolType || "-" },
      { label: "Power Source", value: (m) => m.powerSource || "-" },
      { label: "Weight", value: (m) => (m.weightG ? `${m.weightG.toFixed(2)}g` : "-") },
      { label: "Description", value: (m) => m.description || "-" },
    ],
  },
];

for (const col of GEAR_COLLECTIONS) {
  col.path = getCollectionPath(col);
  col.color = getCollectionColorFromSchema(col.schema, col.id, col.name);
}
