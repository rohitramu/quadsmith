import type { HardwareCollectionDef, ColumnConfig } from "./hardwareCollections";
import { getCollectionColorFromSchema } from "./collectionColors";
import { getCollectionPath } from "./hardwareCollections";

// Protobuf Schemas
import { FcFirmwareSchema } from "../gen/quadsmith/api/fc_firmware_pb";
import { EscFirmwareSchema } from "../gen/quadsmith/api/esc_firmware_pb";
import { VtxFirmwareSchema } from "../gen/quadsmith/api/vtx_firmware_pb";
import { RadioFirmwareSchema } from "../gen/quadsmith/api/radio_firmware_pb";
import { SoftwareToolSchema } from "../gen/quadsmith/api/software_tool_pb";

// Connect Query Services
import {
  listFcFirmwares,
  getFcFirmware,
} from "../gen/quadsmith/api/fc_firmware-FcFirmwareService_connectquery";
import {
  listEscFirmwares,
  getEscFirmware,
} from "../gen/quadsmith/api/esc_firmware-EscFirmwareService_connectquery";
import {
  listVtxFirmwares,
  getVtxFirmware,
} from "../gen/quadsmith/api/vtx_firmware-VtxFirmwareService_connectquery";
import {
  listRadioFirmwares,
  getRadioFirmware,
} from "../gen/quadsmith/api/radio_firmware-RadioFirmwareService_connectquery";
import {
  listSoftwareTools,
  getSoftwareTool,
} from "../gen/quadsmith/api/software_tool-SoftwareToolService_connectquery";

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
      title: "Organization / Developer",
      renderCell: (m) => m.manufacturer || "Unknown",
    },
    name: {
      id: "name",
      title: "Name",
      renderCell: (m) => (
        <span className="font-medium text-zinc-900 dark:text-zinc-100">{m.name || m.id}</span>
      ),
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

export const SOFTWARE_COLLECTIONS: HardwareCollectionDef[] = [
  // 1. FC Firmware
  {
    id: "fc-firmware",
    name: "Flight Controller Firmware",
    description:
      "Core embedded flight control systems for multi-rotor stabilization, filtering, and autonomous navigation.",
    schema: FcFirmwareSchema,
    listQuery: listFcFirmwares,
    getQuery: getFcFirmware,
    getDataList: (res) => res?.fcFirmwares ?? [],
    defaultColumnIds: ["manufacturer", "name", "version", "target_architectures", "is_open_source"],
    fields: [
      {
        name: "id",
        type: "string",
        description: "Unique identifier",
        examples: ['id == "betaflight-4.5"', 'id.contains("inav")'],
      },
      {
        name: "manufacturer",
        type: "string",
        description: "Organization / Project name",
        examples: ['manufacturer == "Betaflight"', 'manufacturer == "INAV"'],
      },
      {
        name: "version",
        type: "string",
        description: "Software release version",
        examples: ['version.startsWith("4.5")', 'version == "7.1.2"'],
      },
      {
        name: "is_open_source",
        type: "boolean",
        description: "Whether software is open source",
        examples: ["is_open_source == true"],
      },
    ],
    presets: [
      { label: "Open Source Only", query: "is_open_source == true" },
      { label: "Betaflight Ecosystem", query: 'manufacturer == "Betaflight"' },
      { label: "INAV Autonomous", query: 'manufacturer == "INAV"' },
    ],
    columns: {
      ...createStandardColumns(),
      version: {
        id: "version",
        title: "Version",
        renderCell: (m) => (
          <span className="font-mono text-xs font-semibold px-2 py-0.5 rounded bg-zinc-100 dark:bg-zinc-800 text-zinc-700 dark:text-zinc-300">
            {m.version || "-"}
          </span>
        ),
      },
      target_architectures: {
        id: "target_architectures",
        title: "Supported Architectures",
        renderCell: (m) => m.targetArchitectures || "-",
      },
      supported_protocols: {
        id: "supported_protocols",
        title: "Protocols",
        renderCell: (m) => m.supportedProtocols || "-",
      },
      is_open_source: {
        id: "is_open_source",
        title: "Open Source",
        renderCell: (m) =>
          m.isOpenSource ? (
            <span className="text-emerald-600 dark:text-emerald-400 font-medium">Yes</span>
          ) : (
            <span className="text-zinc-400">No</span>
          ),
      },
      repository_url: {
        id: "repository_url",
        title: "Repository",
        renderCell: (m) =>
          m.repositoryUrl ? (
            <a
              href={m.repositoryUrl}
              target="_blank"
              rel="noopener noreferrer"
              className="text-blue-600 dark:text-blue-400 hover:underline text-xs"
            >
              {m.repositoryUrl.replace(/^https?:\/\/(www\.)?github\.com\//, "")}
            </a>
          ) : (
            "-"
          ),
      },
    },
    highlights: [
      { label: "Version", value: (m) => m.version || "-" },
      { label: "Architectures", value: (m) => m.targetArchitectures || "-" },
      { label: "Protocols", value: (m) => m.supportedProtocols || "-" },
      { label: "License", value: (m) => (m.isOpenSource ? "Open Source" : "Proprietary") },
    ],
    technicalSpecs: [
      { label: "Developer / Project", value: (m) => m.manufacturer || "-" },
      { label: "Latest Version", value: (m) => m.version || "-" },
      { label: "Supported Hardware Architectures", value: (m) => m.targetArchitectures || "-" },
      { label: "Control Protocols", value: (m) => m.supportedProtocols || "-" },
      { label: "Open Source", value: (m) => (m.isOpenSource ? "Yes" : "No") },
      {
        label: "Source Code Repository",
        value: (m) =>
          m.repositoryUrl ? (
            <a
              href={m.repositoryUrl}
              target="_blank"
              rel="noopener noreferrer"
              className="text-blue-600 dark:text-blue-400 hover:underline"
            >
              {m.repositoryUrl}
            </a>
          ) : (
            "-"
          ),
      },
      { label: "Description", value: (m) => m.description || "-" },
    ],
  },

  // 2. ESC Firmware
  {
    id: "esc-firmware",
    name: "ESC Firmware",
    description:
      "Speed controller firmware managing motor timing, commutation algorithms, and bidirectional DShot RPM telemetry.",
    schema: EscFirmwareSchema,
    listQuery: listEscFirmwares,
    getQuery: getEscFirmware,
    getDataList: (res) => res?.escFirmwares ?? [],
    defaultColumnIds: ["manufacturer", "name", "version", "protocols", "is_open_source"],
    fields: [
      {
        name: "id",
        type: "string",
        description: "Unique identifier",
        examples: ['id == "bluejay-0.19"', 'id.contains("am32")'],
      },
      {
        name: "manufacturer",
        type: "string",
        description: "Firmware Project / Brand",
        examples: ['manufacturer == "Bluejay"', 'manufacturer == "BLHeli"'],
      },
      {
        name: "version",
        type: "string",
        description: "Firmware release version",
        examples: ['version.startsWith("0.19")'],
      },
      {
        name: "is_open_source",
        type: "boolean",
        description: "Whether firmware is open source",
        examples: ["is_open_source == true"],
      },
    ],
    presets: [
      { label: "Open Source ESCs", query: "is_open_source == true" },
      { label: "Bluejay", query: 'manufacturer == "Bluejay"' },
      { label: "AM32", query: 'id.contains("am32")' },
    ],
    columns: {
      ...createStandardColumns(),
      version: {
        id: "version",
        title: "Version",
        renderCell: (m) => (
          <span className="font-mono text-xs font-semibold px-2 py-0.5 rounded bg-zinc-100 dark:bg-zinc-800 text-zinc-700 dark:text-zinc-300">
            {m.version || "-"}
          </span>
        ),
      },
      protocols: {
        id: "protocols",
        title: "Protocols",
        renderCell: (m) => m.protocols || "-",
      },
      pwm_frequency_range: {
        id: "pwm_frequency_range",
        title: "PWM Frequency Range",
        renderCell: (m) => m.pwmFrequencyRange || "-",
      },
      is_open_source: {
        id: "is_open_source",
        title: "Open Source",
        renderCell: (m) =>
          m.isOpenSource ? (
            <span className="text-emerald-600 dark:text-emerald-400 font-medium">Yes</span>
          ) : (
            <span className="text-zinc-400">No</span>
          ),
      },
    },
    highlights: [
      { label: "Version", value: (m) => m.version || "-" },
      { label: "Protocols", value: (m) => m.protocols || "-" },
      { label: "PWM Frequencies", value: (m) => m.pwmFrequencyRange || "-" },
      { label: "License", value: (m) => (m.isOpenSource ? "Open Source" : "Proprietary") },
    ],
    technicalSpecs: [
      { label: "Developer / Project", value: (m) => m.manufacturer || "-" },
      { label: "Version", value: (m) => m.version || "-" },
      { label: "Supported Protocols", value: (m) => m.protocols || "-" },
      { label: "PWM Frequency Range", value: (m) => m.pwmFrequencyRange || "-" },
      { label: "Open Source", value: (m) => (m.isOpenSource ? "Yes" : "No") },
      { label: "Description", value: (m) => m.description || "-" },
    ],
  },

  // 3. VTX Firmware
  {
    id: "vtx-firmware",
    name: "Video Transmitter Firmware",
    description: "Video transmitter operating systems and codecs for HD and analog video feeds.",
    schema: VtxFirmwareSchema,
    listQuery: listVtxFirmwares,
    getQuery: getVtxFirmware,
    getDataList: (res) => res?.vtxFirmwares ?? [],
    defaultColumnIds: ["manufacturer", "name", "version", "protocol", "supported_resolutions"],
    fields: [
      {
        name: "id",
        type: "string",
        description: "Unique identifier",
        examples: ['id.contains("hdzero")', 'id.contains("avatar")'],
      },
      {
        name: "protocol",
        type: "string",
        description: "Video transmission protocol",
        examples: ['protocol == "HDZero"', 'protocol.contains("Walksnail")'],
      },
    ],
    presets: [
      { label: "HDZero", query: 'protocol == "HDZero"' },
      { label: "Walksnail Avatar", query: 'protocol.contains("Walksnail")' },
      { label: "DJI Systems", query: 'protocol.contains("DJI")' },
    ],
    columns: {
      ...createStandardColumns(),
      version: {
        id: "version",
        title: "Version",
        renderCell: (m) => (
          <span className="font-mono text-xs font-semibold px-2 py-0.5 rounded bg-zinc-100 dark:bg-zinc-800 text-zinc-700 dark:text-zinc-300">
            {m.version || "-"}
          </span>
        ),
      },
      protocol: {
        id: "protocol",
        title: "Video Protocol",
        renderCell: (m) => m.protocol || "-",
      },
      supported_resolutions: {
        id: "supported_resolutions",
        title: "Resolutions & Framerates",
        renderCell: (m) => m.supportedResolutions || "-",
      },
    },
    highlights: [
      { label: "Version", value: (m) => m.version || "-" },
      { label: "Video Protocol", value: (m) => m.protocol || "-" },
      { label: "Resolutions", value: (m) => m.supportedResolutions || "-" },
    ],
    technicalSpecs: [
      { label: "Manufacturer", value: (m) => m.manufacturer || "-" },
      { label: "Firmware Version", value: (m) => m.version || "-" },
      { label: "Supported Protocol", value: (m) => m.protocol || "-" },
      { label: "Supported Resolutions", value: (m) => m.supportedResolutions || "-" },
      { label: "Description", value: (m) => m.description || "-" },
    ],
  },

  // 4. Radio Firmware
  {
    id: "radio-firmware",
    name: "Radio Firmware",
    description:
      "Operating systems and RF communication stacks powering handheld radio transmitters and TX modules.",
    schema: RadioFirmwareSchema,
    listQuery: listRadioFirmwares,
    getQuery: getRadioFirmware,
    getDataList: (res) => res?.radioFirmwares ?? [],
    defaultColumnIds: ["manufacturer", "name", "version", "supported_protocols", "is_open_source"],
    fields: [
      {
        name: "id",
        type: "string",
        description: "Unique identifier",
        examples: ['id.contains("edgetx")', 'id.contains("expresslrs")'],
      },
      {
        name: "manufacturer",
        type: "string",
        description: "Developer / Project",
        examples: ['manufacturer == "EdgeTX"', 'manufacturer == "ExpressLRS"'],
      },
    ],
    presets: [
      { label: "EdgeTX Releases", query: 'manufacturer == "EdgeTX"' },
      { label: "ExpressLRS Links", query: 'manufacturer == "ExpressLRS"' },
    ],
    columns: {
      ...createStandardColumns(),
      version: {
        id: "version",
        title: "Version",
        renderCell: (m) => (
          <span className="font-mono text-xs font-semibold px-2 py-0.5 rounded bg-zinc-100 dark:bg-zinc-800 text-zinc-700 dark:text-zinc-300">
            {m.version || "-"}
          </span>
        ),
      },
      supported_protocols: {
        id: "supported_protocols",
        title: "Supported Protocols",
        renderCell: (m) => m.supportedProtocols || "-",
      },
      is_open_source: {
        id: "is_open_source",
        title: "Open Source",
        renderCell: (m) =>
          m.isOpenSource ? (
            <span className="text-emerald-600 dark:text-emerald-400 font-medium">Yes</span>
          ) : (
            <span className="text-zinc-400">No</span>
          ),
      },
    },
    highlights: [
      { label: "Version", value: (m) => m.version || "-" },
      { label: "Protocols", value: (m) => m.supportedProtocols || "-" },
      { label: "License", value: (m) => (m.isOpenSource ? "Open Source" : "Proprietary") },
    ],
    technicalSpecs: [
      { label: "Project / Organization", value: (m) => m.manufacturer || "-" },
      { label: "Version", value: (m) => m.version || "-" },
      { label: "Supported Protocols", value: (m) => m.supportedProtocols || "-" },
      { label: "Open Source", value: (m) => (m.isOpenSource ? "Yes" : "No") },
      { label: "Description", value: (m) => m.description || "-" },
    ],
  },

  // 5. Software Tools
  {
    id: "software-tools",
    name: "Software Tools",
    description:
      "Desktop configurators, log analyzers, and firmware flashers for managing flight hardware.",
    schema: SoftwareToolSchema,
    listQuery: listSoftwareTools,
    getQuery: getSoftwareTool,
    getDataList: (res) => res?.softwareTools ?? [],
    defaultColumnIds: ["manufacturer", "name", "version", "category", "platform"],
    fields: [
      {
        name: "id",
        type: "string",
        description: "Unique identifier",
        examples: ['id.contains("configurator")', 'id.contains("blackbox")'],
      },
      {
        name: "category",
        type: "string",
        description: "Tool category",
        examples: ['category == "Configurator"', 'category.contains("Analysis")'],
      },
    ],
    presets: [
      { label: "Configurators", query: 'category == "Configurator"' },
      { label: "Analysis Tools", query: 'category.contains("Analysis")' },
    ],
    columns: {
      ...createStandardColumns(),
      version: {
        id: "version",
        title: "Version",
        renderCell: (m) => (
          <span className="font-mono text-xs font-semibold px-2 py-0.5 rounded bg-zinc-100 dark:bg-zinc-800 text-zinc-700 dark:text-zinc-300">
            {m.version || "-"}
          </span>
        ),
      },
      category: {
        id: "category",
        title: "Category",
        renderCell: (m) => (
          <span className="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-blue-50 dark:bg-blue-950/60 text-blue-700 dark:text-blue-300 border border-blue-200 dark:border-blue-800/60">
            {m.category || "-"}
          </span>
        ),
      },
      platform: {
        id: "platform",
        title: "Supported Platforms",
        renderCell: (m) => m.platform || "-",
      },
    },
    highlights: [
      { label: "Version", value: (m) => m.version || "-" },
      { label: "Category", value: (m) => m.category || "-" },
      { label: "Platforms", value: (m) => m.platform || "-" },
    ],
    technicalSpecs: [
      { label: "Developer", value: (m) => m.manufacturer || "-" },
      { label: "Version", value: (m) => m.version || "-" },
      { label: "Category", value: (m) => m.category || "-" },
      { label: "Platform Availability", value: (m) => m.platform || "-" },
      { label: "Description", value: (m) => m.description || "-" },
    ],
  },
];

for (const col of SOFTWARE_COLLECTIONS) {
  col.path = getCollectionPath(col);
  col.color = getCollectionColorFromSchema(col.schema, col.id, col.name);
}
