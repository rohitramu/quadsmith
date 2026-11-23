import { ScalarType, type DescEnum, type DescField, type DescMessage } from "@bufbuild/protobuf";
import type { GenService } from "@bufbuild/protobuf/codegenv2";

import { AntennaSchema } from "../gen/quadsmith/api/antenna_pb";
import { BatterySchema } from "../gen/quadsmith/api/battery_pb";
import { CameraSchema } from "../gen/quadsmith/api/camera_pb";
import { ChargerSchema } from "../gen/quadsmith/api/charger_pb";
import { ElectronicSpeedControllerSchema } from "../gen/quadsmith/api/electronic_speed_controller_pb";
import { EscFirmwareSchema } from "../gen/quadsmith/api/esc_firmware_pb";
import { FcFirmwareSchema } from "../gen/quadsmith/api/fc_firmware_pb";
import { FieldToolSchema } from "../gen/quadsmith/api/field_tool_pb";
import { FlightControllerSchema } from "../gen/quadsmith/api/flight_controller_pb";
import { FrameSchema } from "../gen/quadsmith/api/frame_pb";
import { GogglesSchema } from "../gen/quadsmith/api/goggles_pb";
import { GpsReceiverSchema } from "../gen/quadsmith/api/gps_receiver_pb";
import { MotorSchema } from "../gen/quadsmith/api/motor_pb";
import { PropellerSchema } from "../gen/quadsmith/api/propeller_pb";
import { RadioFirmwareSchema } from "../gen/quadsmith/api/radio_firmware_pb";
import { ReceiverSchema } from "../gen/quadsmith/api/receiver_pb";
import { SoftwareToolSchema } from "../gen/quadsmith/api/software_tool_pb";
import { TransmitterSchema } from "../gen/quadsmith/api/transmitter_pb";
import { VideoTransmitterSchema } from "../gen/quadsmith/api/video_transmitter_pb";
import { VtxFirmwareSchema } from "../gen/quadsmith/api/vtx_firmware_pb";
import {
  AntennaOpsService,
  BatteryOpsService,
  CameraOpsService,
  ChargerOpsService,
  ElectronicSpeedControllerOpsService,
  EscFirmwareOpsService,
  FcFirmwareOpsService,
  FieldToolOpsService,
  FlightControllerOpsService,
  FrameOpsService,
  GogglesOpsService,
  GpsReceiverOpsService,
  MotorOpsService,
  PropellerOpsService,
  RadioFirmwareOpsService,
  ReceiverOpsService,
  SoftwareToolOpsService,
  TransmitterOpsService,
  VideoTransmitterOpsService,
  VtxFirmwareOpsService,
} from "../gen/quadsmith/ops/catalog_pb";
import type { FieldDef } from "../components/SmartFilterInput";
import { formatStatorSize } from "./format";

export type CatalogCategory = "hardware" | "gear" | "software";

export interface CatalogTableSpecColumn {
  key: string;
  sortKey: string;
  label: string;
  render: (item: any) => string;
}

export interface CatalogSpecFormField {
  snakeName: string;
  localName: string;
  label: string;
  inputType: "text" | "number" | "boolean" | "enum" | "string_list";
  enumOptions?: { value: number; label: string }[];
}

export interface CatalogCollectionMeta {
  collectionId: string;
  category: CatalogCategory;
  aliases: string[];
  pluralLabel: string;
  singularLabel: string;
  opsServiceName: string;
  paramKey: string;
  listResponseKey: string;
  schema: DescMessage;
  opsService: GenService<any>;
  opsMethods: {
    get: string;
    list: string;
    create: string;
    update: string;
    delete: string;
    restore: string;
  };
  presets: { label: string; query: string }[];
  specColumns: CatalogTableSpecColumn[];
}

export const CATALOG_CATEGORY_GROUPS: { id: CatalogCategory; label: string }[] = [
  { id: "hardware", label: "Hardware" },
  { id: "gear", label: "Pilot Gear" },
  { id: "software", label: "Software & Firmware" },
];

export const CATALOG_COLLECTIONS: CatalogCollectionMeta[] = [
  // --- Hardware (11) ---
  {
    collectionId: "motors",
    category: "hardware",
    aliases: ["motor", "components/hardware/motors"],
    pluralLabel: "Motors",
    singularLabel: "Motor",
    opsServiceName: "MotorOpsService",
    paramKey: "motor",
    listResponseKey: "motors",
    schema: MotorSchema,
    opsService: MotorOpsService,
    opsMethods: {
      get: "getMotor",
      list: "listMotors",
      create: "createMotor",
      update: "updateMotor",
      delete: "deleteMotor",
      restore: "restoreMotor",
    },
    presets: [
      { label: "6S Freestyle (<2000KV)", query: "kv >= 1700 && kv <= 2000" },
      { label: "Ultralight (<15g)", query: "weight_g > 0 && weight_g <= 15" },
      { label: "Archived Only", query: "is_archived == true" },
    ],
    specColumns: [
      {
        key: "stator",
        sortKey: "stator_diameter_mm",
        label: "Stator",
        render: (m) => formatStatorSize(m.statorDiameterMm, m.statorHeightMm) || "—",
      },
      {
        key: "kv",
        sortKey: "kv",
        label: "KV",
        render: (m) => (m.kv ? `${m.kv} KV` : "—"),
      },
      {
        key: "weight",
        sortKey: "weight_g",
        label: "Weight",
        render: (m) => (m.weightG ? `${Number(m.weightG).toFixed(1)}g` : "—"),
      },
    ],
  },
  {
    collectionId: "frames",
    category: "hardware",
    aliases: ["frame", "components/hardware/frames"],
    pluralLabel: "Frames",
    singularLabel: "Frame",
    opsServiceName: "FrameOpsService",
    paramKey: "frame",
    listResponseKey: "frames",
    schema: FrameSchema,
    opsService: FrameOpsService,
    opsMethods: {
      get: "getFrame",
      list: "listFrames",
      create: "createFrame",
      update: "updateFrame",
      delete: "deleteFrame",
      restore: "restoreFrame",
    },
    presets: [
      { label: '5" Freestyle', query: "wheelbase_mm >= 210 && wheelbase_mm <= 240" },
      { label: "Sub-250g (<80g)", query: "weight_g > 0 && weight_g <= 80" },
      { label: "Archived Only", query: "is_archived == true" },
    ],
    specColumns: [
      {
        key: "wheelbase",
        sortKey: "wheelbase_mm",
        label: "Wheelbase",
        render: (f) => (f.wheelbaseMm ? `${f.wheelbaseMm} mm` : "—"),
      },
      {
        key: "maxProp",
        sortKey: "max_prop_size_in",
        label: "Max Prop",
        render: (f) => (f.maxPropSizeIn ? `${f.maxPropSizeIn}"` : "—"),
      },
      {
        key: "weight",
        sortKey: "weight_g",
        label: "Weight",
        render: (f) => (f.weightG ? `${Number(f.weightG).toFixed(1)}g` : "—"),
      },
    ],
  },
  {
    collectionId: "flight-controllers",
    category: "hardware",
    aliases: [
      "flight_controllers",
      "flight-controller",
      "flight_controller",
      "fcs",
      "fc",
      "components/hardware/flight-controllers",
    ],
    pluralLabel: "Flight Controllers",
    singularLabel: "Flight Controller",
    opsServiceName: "FlightControllerOpsService",
    paramKey: "flightController",
    listResponseKey: "flightControllers",
    schema: FlightControllerSchema,
    opsService: FlightControllerOpsService,
    opsMethods: {
      get: "getFlightController",
      list: "listFlightControllers",
      create: "createFlightController",
      update: "updateFlightController",
      delete: "deleteFlightController",
      restore: "restoreFlightController",
    },
    presets: [
      { label: "F7 / H7 MCU", query: 'mcu.contains("F7") || mcu.contains("H7")' },
      { label: "Archived Only", query: "is_archived == true" },
    ],
    specColumns: [
      {
        key: "mcu",
        sortKey: "mcu",
        label: "MCU",
        render: (fc) => fc.mcu || "—",
      },
      {
        key: "weight",
        sortKey: "weight_g",
        label: "Weight",
        render: (fc) => (fc.weightG ? `${Number(fc.weightG).toFixed(1)}g` : "—"),
      },
    ],
  },
  {
    collectionId: "electronic-speed-controllers",
    category: "hardware",
    aliases: [
      "electronic_speed_controllers",
      "electronic-speed-controller",
      "electronic_speed_controller",
      "escs",
      "esc",
      "components/hardware/electronic-speed-controllers",
    ],
    pluralLabel: "Electronic Speed Controllers",
    singularLabel: "Electronic Speed Controller",
    opsServiceName: "ElectronicSpeedControllerOpsService",
    paramKey: "electronicSpeedController",
    listResponseKey: "electronicSpeedControllers",
    schema: ElectronicSpeedControllerSchema,
    opsService: ElectronicSpeedControllerOpsService,
    opsMethods: {
      get: "getElectronicSpeedController",
      list: "listElectronicSpeedControllers",
      create: "createElectronicSpeedController",
      update: "updateElectronicSpeedController",
      delete: "deleteElectronicSpeedController",
      restore: "restoreElectronicSpeedController",
    },
    presets: [
      { label: "High Current (>=55A)", query: "continuous_current_a >= 55" },
      { label: "Internal AIO Modules", query: "is_internal_only == true" },
      { label: "Archived Only", query: "is_archived == true" },
    ],
    specColumns: [
      {
        key: "current",
        sortKey: "continuous_current_a",
        label: "Cont. Current",
        render: (e) => (e.continuousCurrentA ? `${e.continuousCurrentA}A` : "—"),
      },
      {
        key: "weight",
        sortKey: "weight_g",
        label: "Weight",
        render: (e) => (e.weightG ? `${Number(e.weightG).toFixed(1)}g` : "—"),
      },
    ],
  },
  {
    collectionId: "propellers",
    category: "hardware",
    aliases: ["propeller", "props", "prop", "components/hardware/propellers"],
    pluralLabel: "Propellers",
    singularLabel: "Propeller",
    opsServiceName: "PropellerOpsService",
    paramKey: "propeller",
    listResponseKey: "propellers",
    schema: PropellerSchema,
    opsService: PropellerOpsService,
    opsMethods: {
      get: "getPropeller",
      list: "listPropellers",
      create: "createPropeller",
      update: "updatePropeller",
      delete: "deletePropeller",
      restore: "restorePropeller",
    },
    presets: [
      { label: '5" Props', query: "diameter_in >= 4.9 && diameter_in <= 5.2" },
      { label: "Archived Only", query: "is_archived == true" },
    ],
    specColumns: [
      {
        key: "diameter",
        sortKey: "diameter_in",
        label: "Size",
        render: (p) =>
          p.diameterIn ? `${p.diameterIn}"${p.pitchIn ? ` × ${p.pitchIn}` : ""}` : "—",
      },
      {
        key: "blades",
        sortKey: "blades",
        label: "Blades",
        render: (p) => (p.blades ? `${p.blades}-blade` : "—"),
      },
      {
        key: "weight",
        sortKey: "weight_g",
        label: "Weight",
        render: (p) => (p.weightG ? `${Number(p.weightG).toFixed(1)}g` : "—"),
      },
    ],
  },
  {
    collectionId: "batteries",
    category: "hardware",
    aliases: ["battery", "components/hardware/batteries"],
    pluralLabel: "Batteries",
    singularLabel: "Battery",
    opsServiceName: "BatteryOpsService",
    paramKey: "battery",
    listResponseKey: "batteries",
    schema: BatterySchema,
    opsService: BatteryOpsService,
    opsMethods: {
      get: "getBattery",
      list: "listBatteries",
      create: "createBattery",
      update: "updateBattery",
      delete: "deleteBattery",
      restore: "restoreBattery",
    },
    presets: [
      { label: "6S LiPo", query: "cell_count_s == 6" },
      { label: "4S LiPo", query: "cell_count_s == 4" },
      { label: "Archived Only", query: "is_archived == true" },
    ],
    specColumns: [
      {
        key: "cells",
        sortKey: "cell_count_s",
        label: "Cells",
        render: (b) => (b.cellCountS ? `${b.cellCountS}S` : "—"),
      },
      {
        key: "capacity",
        sortKey: "capacity_mah",
        label: "Capacity",
        render: (b) => (b.capacityMah ? `${b.capacityMah} mAh` : "—"),
      },
      {
        key: "weight",
        sortKey: "weight_g",
        label: "Weight",
        render: (b) => (b.weightG ? `${Number(b.weightG).toFixed(1)}g` : "—"),
      },
    ],
  },
  {
    collectionId: "cameras",
    category: "hardware",
    aliases: ["camera", "components/hardware/cameras"],
    pluralLabel: "Cameras",
    singularLabel: "Camera",
    opsServiceName: "CameraOpsService",
    paramKey: "camera",
    listResponseKey: "cameras",
    schema: CameraSchema,
    opsService: CameraOpsService,
    opsMethods: {
      get: "getCamera",
      list: "listCameras",
      create: "createCamera",
      update: "updateCamera",
      delete: "deleteCamera",
      restore: "restoreCamera",
    },
    presets: [{ label: "Archived Only", query: "is_archived == true" }],
    specColumns: [
      {
        key: "fov",
        sortKey: "fov_deg",
        label: "FOV",
        render: (c) => (c.fovDeg ? `${c.fovDeg}°` : "—"),
      },
      {
        key: "weight",
        sortKey: "weight_g",
        label: "Weight",
        render: (c) => (c.weightG ? `${Number(c.weightG).toFixed(1)}g` : "—"),
      },
    ],
  },
  {
    collectionId: "video-transmitters",
    category: "hardware",
    aliases: [
      "video_transmitters",
      "video-transmitter",
      "video_transmitter",
      "vtxs",
      "vtx",
      "components/hardware/video-transmitters",
    ],
    pluralLabel: "Video Transmitters",
    singularLabel: "Video Transmitter",
    opsServiceName: "VideoTransmitterOpsService",
    paramKey: "videoTransmitter",
    listResponseKey: "videoTransmitters",
    schema: VideoTransmitterSchema,
    opsService: VideoTransmitterOpsService,
    opsMethods: {
      get: "getVideoTransmitter",
      list: "listVideoTransmitters",
      create: "createVideoTransmitter",
      update: "updateVideoTransmitter",
      delete: "deleteVideoTransmitter",
      restore: "restoreVideoTransmitter",
    },
    presets: [
      { label: "High Power (>=1000mW)", query: "max_power_mw >= 1000" },
      { label: "Internal AIO Modules", query: "is_internal_only == true" },
      { label: "Archived Only", query: "is_archived == true" },
    ],
    specColumns: [
      {
        key: "power",
        sortKey: "max_power_mw",
        label: "Max Power",
        render: (v) => (v.maxPowerMw ? `${v.maxPowerMw} mW` : "—"),
      },
      {
        key: "weight",
        sortKey: "weight_g",
        label: "Weight",
        render: (v) => (v.weightG ? `${Number(v.weightG).toFixed(1)}g` : "—"),
      },
    ],
  },
  {
    collectionId: "receivers",
    category: "hardware",
    aliases: ["receiver", "rx", "components/hardware/receivers"],
    pluralLabel: "Receivers",
    singularLabel: "Receiver",
    opsServiceName: "ReceiverOpsService",
    paramKey: "receiver",
    listResponseKey: "receivers",
    schema: ReceiverSchema,
    opsService: ReceiverOpsService,
    opsMethods: {
      get: "getReceiver",
      list: "listReceivers",
      create: "createReceiver",
      update: "updateReceiver",
      delete: "deleteReceiver",
      restore: "restoreReceiver",
    },
    presets: [
      { label: "Internal AIO Modules", query: "is_internal_only == true" },
      { label: "Archived Only", query: "is_archived == true" },
    ],
    specColumns: [
      {
        key: "protocol",
        sortKey: "protocol",
        label: "Protocol",
        render: (r) => r.protocol || "—",
      },
      {
        key: "weight",
        sortKey: "weight_g",
        label: "Weight",
        render: (r) => (r.weightG ? `${Number(r.weightG).toFixed(1)}g` : "—"),
      },
    ],
  },
  {
    collectionId: "antennas",
    category: "hardware",
    aliases: ["antenna", "components/hardware/antennas"],
    pluralLabel: "Antennas",
    singularLabel: "Antenna",
    opsServiceName: "AntennaOpsService",
    paramKey: "antenna",
    listResponseKey: "antennas",
    schema: AntennaSchema,
    opsService: AntennaOpsService,
    opsMethods: {
      get: "getAntenna",
      list: "listAntennas",
      create: "createAntenna",
      update: "updateAntenna",
      delete: "deleteAntenna",
      restore: "restoreAntenna",
    },
    presets: [{ label: "Archived Only", query: "is_archived == true" }],
    specColumns: [
      {
        key: "gain",
        sortKey: "gain_dbi",
        label: "Gain",
        render: (a) => (a.gainDbi ? `${a.gainDbi} dBi` : "—"),
      },
      {
        key: "weight",
        sortKey: "weight_g",
        label: "Weight",
        render: (a) => (a.weightG ? `${Number(a.weightG).toFixed(1)}g` : "—"),
      },
    ],
  },
  {
    collectionId: "gps-receivers",
    category: "hardware",
    aliases: [
      "gps_receivers",
      "gps-receiver",
      "gps_receiver",
      "gps",
      "components/hardware/gps-receivers",
    ],
    pluralLabel: "GPS Receivers",
    singularLabel: "GPS Receiver",
    opsServiceName: "GpsReceiverOpsService",
    paramKey: "gpsReceiver",
    listResponseKey: "gpsReceivers",
    schema: GpsReceiverSchema,
    opsService: GpsReceiverOpsService,
    opsMethods: {
      get: "getGpsReceiver",
      list: "listGpsReceivers",
      create: "createGpsReceiver",
      update: "updateGpsReceiver",
      delete: "deleteGpsReceiver",
      restore: "restoreGpsReceiver",
    },
    presets: [
      { label: "With Compass", query: "has_compass == true" },
      { label: "Archived Only", query: "is_archived == true" },
    ],
    specColumns: [
      {
        key: "chipset",
        sortKey: "chipset",
        label: "Chipset",
        render: (g) => g.chipset || "—",
      },
      {
        key: "weight",
        sortKey: "weight_g",
        label: "Weight",
        render: (g) => (g.weightG ? `${Number(g.weightG).toFixed(1)}g` : "—"),
      },
    ],
  },

  // --- Pilot Gear (4) ---
  {
    collectionId: "transmitters",
    category: "gear",
    aliases: ["transmitter", "tx", "radios", "radio", "components/gear/transmitters"],
    pluralLabel: "Transmitters",
    singularLabel: "Transmitter",
    opsServiceName: "TransmitterOpsService",
    paramKey: "transmitter",
    listResponseKey: "transmitters",
    schema: TransmitterSchema,
    opsService: TransmitterOpsService,
    opsMethods: {
      get: "getTransmitter",
      list: "listTransmitters",
      create: "createTransmitter",
      update: "updateTransmitter",
      delete: "deleteTransmitter",
      restore: "restoreTransmitter",
    },
    presets: [{ label: "Archived Only", query: "is_archived == true" }],
    specColumns: [
      {
        key: "power",
        sortKey: "max_power_mw",
        label: "Max RF Power",
        render: (t) => (t.maxPowerMw ? `${t.maxPowerMw} mW` : "—"),
      },
      {
        key: "weight",
        sortKey: "weight_g",
        label: "Weight",
        render: (t) => (t.weightG ? `${Number(t.weightG).toFixed(0)}g` : "—"),
      },
    ],
  },
  {
    collectionId: "goggles",
    category: "gear",
    aliases: ["goggle", "components/gear/goggles"],
    pluralLabel: "Goggles",
    singularLabel: "Goggles",
    opsServiceName: "GogglesOpsService",
    paramKey: "goggles",
    listResponseKey: "goggles",
    schema: GogglesSchema,
    opsService: GogglesOpsService,
    opsMethods: {
      get: "getGoggles",
      list: "listGoggles",
      create: "createGoggles",
      update: "updateGoggles",
      delete: "deleteGoggles",
      restore: "restoreGoggles",
    },
    presets: [{ label: "Archived Only", query: "is_archived == true" }],
    specColumns: [
      {
        key: "resolution",
        sortKey: "resolution",
        label: "Resolution",
        render: (g) => g.resolution || "—",
      },
      {
        key: "fov",
        sortKey: "fov_deg",
        label: "FOV",
        render: (g) => (g.fovDeg ? `${g.fovDeg}°` : "—"),
      },
    ],
  },
  {
    collectionId: "chargers",
    category: "gear",
    aliases: ["charger", "components/gear/chargers"],
    pluralLabel: "Chargers",
    singularLabel: "Charger",
    opsServiceName: "ChargerOpsService",
    paramKey: "charger",
    listResponseKey: "chargers",
    schema: ChargerSchema,
    opsService: ChargerOpsService,
    opsMethods: {
      get: "getCharger",
      list: "listChargers",
      create: "createCharger",
      update: "updateCharger",
      delete: "deleteCharger",
      restore: "restoreCharger",
    },
    presets: [{ label: "Archived Only", query: "is_archived == true" }],
    specColumns: [
      {
        key: "power",
        sortKey: "max_power_w",
        label: "Max Power",
        render: (c) => (c.maxPowerW ? `${c.maxPowerW}W` : "—"),
      },
      {
        key: "channels",
        sortKey: "channels",
        label: "Channels",
        render: (c) => (c.channels ? `${c.channels} ch` : "—"),
      },
    ],
  },
  {
    collectionId: "field-tools",
    category: "gear",
    aliases: ["field_tools", "field-tool", "field_tool", "components/gear/field-tools"],
    pluralLabel: "Field Tools",
    singularLabel: "Field Tool",
    opsServiceName: "FieldToolOpsService",
    paramKey: "fieldTool",
    listResponseKey: "fieldTools",
    schema: FieldToolSchema,
    opsService: FieldToolOpsService,
    opsMethods: {
      get: "getFieldTool",
      list: "listFieldTools",
      create: "createFieldTool",
      update: "updateFieldTool",
      delete: "deleteFieldTool",
      restore: "restoreFieldTool",
    },
    presets: [{ label: "Archived Only", query: "is_archived == true" }],
    specColumns: [
      {
        key: "category",
        sortKey: "category",
        label: "Category",
        render: (ft) => ft.category || "—",
      },
      {
        key: "weight",
        sortKey: "weight_g",
        label: "Weight",
        render: (ft) => (ft.weightG ? `${Number(ft.weightG).toFixed(0)}g` : "—"),
      },
    ],
  },

  // --- Software & Firmware (5) ---
  {
    collectionId: "fc-firmwares",
    category: "software",
    aliases: ["fc_firmwares", "fc-firmware", "fc_firmware", "components/software/fc-firmwares"],
    pluralLabel: "FC Firmwares",
    singularLabel: "FC Firmware",
    opsServiceName: "FcFirmwareOpsService",
    paramKey: "fcFirmware",
    listResponseKey: "fcFirmwares",
    schema: FcFirmwareSchema,
    opsService: FcFirmwareOpsService,
    opsMethods: {
      get: "getFcFirmware",
      list: "listFcFirmwares",
      create: "createFcFirmware",
      update: "updateFcFirmware",
      delete: "deleteFcFirmware",
      restore: "restoreFcFirmware",
    },
    presets: [{ label: "Archived Only", query: "is_archived == true" }],
    specColumns: [
      {
        key: "license",
        sortKey: "license",
        label: "License",
        render: (fw) => fw.license || "—",
      },
      {
        key: "version",
        sortKey: "latest_version",
        label: "Latest Version",
        render: (fw) => fw.latestVersion || "—",
      },
    ],
  },
  {
    collectionId: "esc-firmwares",
    category: "software",
    aliases: ["esc_firmwares", "esc-firmware", "esc_firmware", "components/software/esc-firmwares"],
    pluralLabel: "ESC Firmwares",
    singularLabel: "ESC Firmware",
    opsServiceName: "EscFirmwareOpsService",
    paramKey: "escFirmware",
    listResponseKey: "escFirmwares",
    schema: EscFirmwareSchema,
    opsService: EscFirmwareOpsService,
    opsMethods: {
      get: "getEscFirmware",
      list: "listEscFirmwares",
      create: "createEscFirmware",
      update: "updateEscFirmware",
      delete: "deleteEscFirmware",
      restore: "restoreEscFirmware",
    },
    presets: [{ label: "Archived Only", query: "is_archived == true" }],
    specColumns: [
      {
        key: "license",
        sortKey: "license",
        label: "License",
        render: (fw) => fw.license || "—",
      },
      {
        key: "version",
        sortKey: "latest_version",
        label: "Latest Version",
        render: (fw) => fw.latestVersion || "—",
      },
    ],
  },
  {
    collectionId: "vtx-firmwares",
    category: "software",
    aliases: ["vtx_firmwares", "vtx-firmware", "vtx_firmware", "components/software/vtx-firmwares"],
    pluralLabel: "VTX Firmwares",
    singularLabel: "VTX Firmware",
    opsServiceName: "VtxFirmwareOpsService",
    paramKey: "vtxFirmware",
    listResponseKey: "vtxFirmwares",
    schema: VtxFirmwareSchema,
    opsService: VtxFirmwareOpsService,
    opsMethods: {
      get: "getVtxFirmware",
      list: "listVtxFirmwares",
      create: "createVtxFirmware",
      update: "updateVtxFirmware",
      delete: "deleteVtxFirmware",
      restore: "restoreVtxFirmware",
    },
    presets: [{ label: "Archived Only", query: "is_archived == true" }],
    specColumns: [
      {
        key: "license",
        sortKey: "license",
        label: "License",
        render: (fw) => fw.license || "—",
      },
      {
        key: "version",
        sortKey: "latest_version",
        label: "Latest Version",
        render: (fw) => fw.latestVersion || "—",
      },
    ],
  },
  {
    collectionId: "radio-firmwares",
    category: "software",
    aliases: [
      "radio_firmwares",
      "radio-firmware",
      "radio_firmware",
      "components/software/radio-firmwares",
    ],
    pluralLabel: "Radio Firmwares",
    singularLabel: "Radio Firmware",
    opsServiceName: "RadioFirmwareOpsService",
    paramKey: "radioFirmware",
    listResponseKey: "radioFirmwares",
    schema: RadioFirmwareSchema,
    opsService: RadioFirmwareOpsService,
    opsMethods: {
      get: "getRadioFirmware",
      list: "listRadioFirmwares",
      create: "createRadioFirmware",
      update: "updateRadioFirmware",
      delete: "deleteRadioFirmware",
      restore: "restoreRadioFirmware",
    },
    presets: [{ label: "Archived Only", query: "is_archived == true" }],
    specColumns: [
      {
        key: "license",
        sortKey: "license",
        label: "License",
        render: (fw) => fw.license || "—",
      },
      {
        key: "version",
        sortKey: "latest_version",
        label: "Latest Version",
        render: (fw) => fw.latestVersion || "—",
      },
    ],
  },
  {
    collectionId: "software-tools",
    category: "software",
    aliases: [
      "software_tools",
      "software-tool",
      "software_tool",
      "components/software/software-tools",
    ],
    pluralLabel: "Software Tools",
    singularLabel: "Software Tool",
    opsServiceName: "SoftwareToolOpsService",
    paramKey: "softwareTool",
    listResponseKey: "softwareTools",
    schema: SoftwareToolSchema,
    opsService: SoftwareToolOpsService,
    opsMethods: {
      get: "getSoftwareTool",
      list: "listSoftwareTools",
      create: "createSoftwareTool",
      update: "updateSoftwareTool",
      delete: "deleteSoftwareTool",
      restore: "restoreSoftwareTool",
    },
    presets: [{ label: "Archived Only", query: "is_archived == true" }],
    specColumns: [
      {
        key: "category",
        sortKey: "category",
        label: "Category",
        render: (st) => st.category || "—",
      },
      {
        key: "license",
        sortKey: "license",
        label: "License",
        render: (st) => st.license || "—",
      },
    ],
  },
];

export function getCatalogCollectionMeta(collectionId?: string): CatalogCollectionMeta | undefined {
  if (!collectionId) return undefined;
  const normalized = collectionId
    .toLowerCase()
    .trim()
    .replace(/^\/+|\/+$/g, "");
  const kebab = normalized.replace(/_/g, "-");
  return CATALOG_COLLECTIONS.find(
    (c) =>
      c.collectionId === normalized ||
      c.collectionId === kebab ||
      c.paramKey.toLowerCase() === normalized ||
      c.paramKey.toLowerCase() === kebab ||
      c.aliases.includes(normalized) ||
      c.aliases.includes(kebab),
  );
}

export function formatSpecFieldLabel(snakeName: string): string {
  return snakeName
    .split("_")
    .map((p) => {
      if (p === "mm") return "(mm)";
      if (p === "in") return "(in)";
      if (p === "g") return "(g)";
      if (p === "v") return "(V)";
      if (p === "a") return "(A)";
      if (p === "w") return "(W)";
      if (p === "mw") return "(mW)";
      if (p === "mah") return "(mAh)";
      if (p === "dbi") return "(dBi)";
      if (p === "deg") return "(deg)";
      if (p === "ghz") return "(GHz)";
      if (p === "mhz") return "(MHz)";
      if (p === "kv") return "KV";
      if (p === "fc") return "FC";
      if (p === "esc") return "ESC";
      if (p === "vtx") return "VTX";
      if (p === "rx") return "RX";
      if (p === "tx") return "TX";
      if (p === "gps") return "GPS";
      if (p === "mcu") return "MCU";
      if (p === "imu") return "IMU";
      if (p === "fov") return "FOV";
      if (p === "osd") return "OSD";
      if (p === "bec") return "BEC";
      if (p === "usb") return "USB";
      if (p === "uuid") return "ID/UUID";
      if (p === "uuids") return "IDs/UUIDs";
      return p.charAt(0).toUpperCase() + p.slice(1);
    })
    .join(" ");
}

function isNumericScalar(scalar: ScalarType): boolean {
  return (
    scalar === ScalarType.DOUBLE ||
    scalar === ScalarType.FLOAT ||
    scalar === ScalarType.INT32 ||
    scalar === ScalarType.INT64 ||
    scalar === ScalarType.UINT32 ||
    scalar === ScalarType.UINT64 ||
    scalar === ScalarType.SINT32 ||
    scalar === ScalarType.SINT64 ||
    scalar === ScalarType.FIXED32 ||
    scalar === ScalarType.FIXED64 ||
    scalar === ScalarType.SFIXED32 ||
    scalar === ScalarType.SFIXED64
  );
}

function formatEnumOptionLabel(descEnum: DescEnum, rawName: string): string {
  const prefix = descEnum.sharedPrefix ? descEnum.sharedPrefix.toUpperCase() : "";
  const trimmed = prefix && rawName.startsWith(prefix) ? rawName.slice(prefix.length) : rawName;
  return trimmed.replace(/_/g, " ");
}

export function deriveCelFieldsFromSchema(schema: DescMessage): FieldDef[] {
  const result: FieldDef[] = [];
  for (const field of schema.fields) {
    if (field.fieldKind === "message" || field.fieldKind === "map") {
      continue;
    }
    if (field.fieldKind === "scalar") {
      if (field.scalar === ScalarType.BOOL) {
        result.push({
          name: field.name,
          type: "boolean",
          description: formatSpecFieldLabel(field.name),
          examples: ["true", "false"],
        });
      } else if (field.scalar === ScalarType.STRING) {
        result.push({
          name: field.name,
          type: "string",
          description: formatSpecFieldLabel(field.name),
          examples: field.name === "manufacturer" ? ['"T-Motor"', '"SpeedyBee"'] : ['"value"'],
        });
      } else if (isNumericScalar(field.scalar)) {
        result.push({
          name: field.name,
          type: "number",
          description: formatSpecFieldLabel(field.name),
          examples: field.name === "kv" ? ["1950", "2400"] : ["10", "50"],
        });
      }
    }
  }
  return result;
}

const EXCLUDED_SPEC_FORM_FIELDS = new Set([
  "uuid",
  "id",
  "name",
  "manufacturer",
  "description",
  "primary_display_image",
  "reference_links",
  "media",
  "is_archived",
]);

export function deriveSpecFormFieldsFromSchema(schema: DescMessage): CatalogSpecFormField[] {
  const fields: CatalogSpecFormField[] = [];
  for (const f of schema.fields as DescField[]) {
    if (EXCLUDED_SPEC_FORM_FIELDS.has(f.name)) {
      continue;
    }
    const label = formatSpecFieldLabel(f.name);
    if (f.fieldKind === "scalar") {
      if (f.scalar === ScalarType.BOOL) {
        fields.push({
          snakeName: f.name,
          localName: f.localName,
          label,
          inputType: "boolean",
        });
      } else if (f.scalar === ScalarType.STRING) {
        fields.push({
          snakeName: f.name,
          localName: f.localName,
          label,
          inputType: "text",
        });
      } else if (isNumericScalar(f.scalar)) {
        fields.push({
          snakeName: f.name,
          localName: f.localName,
          label,
          inputType: "number",
        });
      }
    } else if (f.fieldKind === "enum") {
      fields.push({
        snakeName: f.name,
        localName: f.localName,
        label,
        inputType: "enum",
        enumOptions: f.enum.values.map((ev) => ({
          value: ev.number,
          label: ev.number === 0 ? "Unspecified" : formatEnumOptionLabel(f.enum, ev.name),
        })),
      });
    } else if (
      f.fieldKind === "list" &&
      f.listKind === "scalar" &&
      f.scalar === ScalarType.STRING
    ) {
      fields.push({
        snakeName: f.name,
        localName: f.localName,
        label: `${label} (comma-separated)`,
        inputType: "string_list",
      });
    }
  }
  return fields;
}
