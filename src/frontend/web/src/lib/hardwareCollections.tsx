import React from "react";
import { formatStatorSize } from "./format";
import type { FieldDef } from "../components/SmartFilterInput";

// Protobuf Schemas
import { AntennaSchema } from "../gen/quadsmith/antenna_pb";
import { BatterySchema } from "../gen/quadsmith/battery_pb";
import { CameraSchema } from "../gen/quadsmith/camera_pb";
import { ElectronicSpeedControllerSchema } from "../gen/quadsmith/electronic_speed_controller_pb";
import { FlightControllerSchema } from "../gen/quadsmith/flight_controller_pb";
import { FrameSchema } from "../gen/quadsmith/frame_pb";
import { GpsReceiverSchema } from "../gen/quadsmith/gps_receiver_pb";
import { MotorSchema } from "../gen/quadsmith/motor_pb";
import { PropellerSchema } from "../gen/quadsmith/propeller_pb";
import { ReceiverSchema } from "../gen/quadsmith/receiver_pb";
import { VideoTransmitterSchema } from "../gen/quadsmith/video_transmitter_pb";

// Connect Query Services
import { listAntennas, getAntenna } from "../gen/quadsmith/antenna-AntennaService_connectquery";
import { listBatteries, getBattery } from "../gen/quadsmith/battery-BatteryService_connectquery";
import { listCameras, getCamera } from "../gen/quadsmith/camera-CameraService_connectquery";
import {
  listElectronicSpeedControllers,
  getElectronicSpeedController,
} from "../gen/quadsmith/electronic_speed_controller-ElectronicSpeedControllerService_connectquery";
import {
  listFlightControllers,
  getFlightController,
} from "../gen/quadsmith/flight_controller-FlightControllerService_connectquery";
import { listFrames, getFrame } from "../gen/quadsmith/frame-FrameService_connectquery";
import {
  listGpsReceivers,
  getGpsReceiver,
} from "../gen/quadsmith/gps_receiver-GpsReceiverService_connectquery";
import { listMotors, getMotor } from "../gen/quadsmith/motor-MotorService_connectquery";
import {
  listPropellers,
  getPropeller,
} from "../gen/quadsmith/propeller-PropellerService_connectquery";
import { listReceivers, getReceiver } from "../gen/quadsmith/receiver-ReceiverService_connectquery";
import {
  listVideoTransmitters,
  getVideoTransmitter,
} from "../gen/quadsmith/video_transmitter-VideoTransmitterService_connectquery";

export interface ColumnConfig {
  id: string;
  title: string;
  renderCell: (item: any) => React.ReactNode;
}

export interface SpecHighlight {
  label: string;
  value: (item: any) => React.ReactNode;
}

export interface TechnicalSpec {
  label: string;
  value: (item: any) => React.ReactNode;
}

export interface HardwareCollectionDef {
  id: string;
  aliases?: string[];
  name: string;
  singular: string;
  schema: any;
  listQuery: any;
  getQuery: any;
  getDataList: (response: any, includeInternal?: boolean) => any[];
  fields: FieldDef[];
  presets: { label: string; query: string }[];
  columns: Record<string, ColumnConfig>;
  defaultColumnIds: string[];
  highlights: SpecHighlight[];
  technicalSpecs: TechnicalSpec[];
}

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
    is_internal_only: {
      id: "is_internal_only",
      title: "Internal Only",
      renderCell: (m) => (m.isInternalOnly ? "Yes" : "No"),
    },
    weight_g: {
      id: "weight_g",
      title: "Weight (g)",
      renderCell: (m) =>
        m.weightG != null && (m.weightG > 0 || m.isInternalOnly) ? `${m.weightG}` : "-",
    },
    description: {
      id: "description",
      title: "Description",
      renderCell: (m) => m.description || "-",
    },
  };
}

export const HARDWARE_COLLECTIONS: HardwareCollectionDef[] = [
  // 1. Antennas
  {
    id: "antennas",
    name: "Antennas",
    singular: "Antenna",
    schema: AntennaSchema,
    listQuery: listAntennas,
    getQuery: getAntenna,
    getDataList: (res, includeInternal = false) =>
      (res?.antennas ?? []).filter((item: any) => includeInternal || !item.isInternalOnly),
    defaultColumnIds: [
      "manufacturer",
      "name",
      "connector",
      "polarization",
      "frequency_band_mhz",
      "gain_dbi",
    ],
    fields: [
      {
        name: "is_internal_only",
        type: "boolean",
        description: "Whether this antenna is integrated/internal only",
        examples: ["is_internal_only == false", "is_internal_only == true"],
      },
      {
        name: "id",
        type: "string",
        description: "Unique identifier",
        examples: ['id.contains("axii")', 'id.startsWith("lumenier")'],
      },
      {
        name: "uuid",
        type: "string",
        description: "Unique UUID identifier",
        examples: ['uuid.startsWith("ad7b")'],
      },
      {
        name: "manufacturer",
        type: "string",
        description: "Manufacturer / Brand name",
        examples: ['manufacturer == "Lumenier"', 'manufacturer.contains("TrueRC")'],
      },
      {
        name: "name",
        type: "string",
        description: "Antenna model name",
        examples: ['name.contains("AXII")'],
      },
      {
        name: "connector",
        type: "string",
        description: "Connector type (SMA, U.FL, MMCX, RP-SMA)",
        examples: ['connector == "SMA"', 'connector == "U.FL"'],
      },
      {
        name: "polarization",
        type: "string",
        description: "Signal polarization (RHCP, LHCP, Linear)",
        examples: ['polarization == "RHCP"', 'polarization == "LHCP"'],
      },
      {
        name: "frequency_band_mhz",
        type: "number",
        description: "Frequency band in MHz (e.g. 5800, 2400)",
        examples: ["frequency_band_mhz == 5800", "frequency_band_mhz == 2400"],
      },
      {
        name: "gain_dbi",
        type: "number",
        description: "Antenna gain in dBi",
        examples: ["gain_dbi >= 2.0", "gain_dbi > 2.5"],
      },
      {
        name: "length_mm",
        type: "number",
        description: "Antenna length in millimeters",
        examples: ["length_mm <= 75.0", "length_mm < 100.0"],
      },
      {
        name: "weight_g",
        type: "number",
        description: "Antenna weight in grams",
        examples: ["weight_g < 10.0", "weight_g <= 2.5"],
      },
      {
        name: "description",
        type: "string",
        description: "Product description",
        examples: ['description.contains("RHCP")'],
      },
    ],
    presets: [
      { label: "5.8GHz Band", query: "frequency_band_mhz == 5800" },
      { label: "2.4GHz Band", query: "frequency_band_mhz == 2400" },
      { label: "RHCP Polarization", query: 'polarization == "RHCP"' },
      { label: "LHCP Polarization", query: 'polarization == "LHCP"' },
      { label: "SMA Connector", query: 'connector == "SMA"' },
      { label: "U.FL Connector", query: 'connector == "U.FL"' },
      { label: "Under 5g", query: "weight_g < 5.0" },
    ],
    columns: {
      ...createStandardColumns(),
      connector: {
        id: "connector",
        title: "Connector",
        renderCell: (a) => a.connector || "-",
      },
      polarization: {
        id: "polarization",
        title: "Polarization",
        renderCell: (a) => a.polarization || "-",
      },
      frequency_band_mhz: {
        id: "frequency_band_mhz",
        title: "Frequency (MHz)",
        renderCell: (a) => (a.frequencyBandMhz ? `${a.frequencyBandMhz}` : "-"),
      },
      gain_dbi: {
        id: "gain_dbi",
        title: "Gain (dBi)",
        renderCell: (a) => (a.gainDbi != null ? `${a.gainDbi}` : "-"),
      },
      length_mm: {
        id: "length_mm",
        title: "Length (mm)",
        renderCell: (a) => (a.lengthMm != null && a.lengthMm > 0 ? `${a.lengthMm}` : "-"),
      },
    },
    highlights: [
      { label: "Connector", value: (a) => a.connector || "N/A" },
      { label: "Polarization", value: (a) => a.polarization || "N/A" },
      {
        label: "Frequency",
        value: (a) => (a.frequencyBandMhz ? `${a.frequencyBandMhz} MHz` : "N/A"),
      },
      {
        label: "Gain",
        value: (a) => (a.gainDbi != null ? `${a.gainDbi} dBi` : "N/A"),
      },
      {
        label: "Weight (g)",
        value: (a) => (a.weightG != null && a.weightG > 0 ? `${a.weightG}` : "N/A"),
      },
      {
        label: "Length (mm)",
        value: (a) => (a.lengthMm != null && a.lengthMm > 0 ? `${a.lengthMm}` : "N/A"),
      },
    ],
    technicalSpecs: [
      { label: "Connector", value: (a) => a.connector || "-" },
      { label: "Polarization", value: (a) => a.polarization || "-" },
      {
        label: "Frequency Band (MHz)",
        value: (a) => (a.frequencyBandMhz ? `${a.frequencyBandMhz}` : "-"),
      },
      {
        label: "Gain (dBi)",
        value: (a) => (a.gainDbi != null ? `${a.gainDbi}` : "-"),
      },
      {
        label: "Length (mm)",
        value: (a) => (a.lengthMm != null && a.lengthMm > 0 ? `${a.lengthMm}` : "-"),
      },
      {
        label: "Weight (g)",
        value: (a) => (a.weightG != null && a.weightG > 0 ? `${a.weightG}` : "-"),
      },
    ],
  },

  // 2. Batteries
  {
    id: "batteries",
    name: "Batteries",
    singular: "Battery",
    schema: BatterySchema,
    listQuery: listBatteries,
    getQuery: getBattery,
    getDataList: (res) => res?.batteries ?? [],
    defaultColumnIds: ["manufacturer", "name", "cell_count_s", "capacity_mah"],
    fields: [
      {
        name: "id",
        type: "string",
        description: "Unique identifier",
        examples: ['id.contains("r-line")', 'id.contains("6s")'],
      },
      {
        name: "uuid",
        type: "string",
        description: "Unique UUID identifier",
        examples: ['uuid.startsWith("1f58")'],
      },
      {
        name: "manufacturer",
        type: "string",
        description: "Manufacturer / Brand name",
        examples: ['manufacturer == "Tattu"', 'manufacturer == "BETAFPV"'],
      },
      {
        name: "name",
        type: "string",
        description: "Battery model name",
        examples: ['name.contains("R-Line")'],
      },
      {
        name: "cell_count_s",
        type: "number",
        description: "Cell count (e.g. 1, 4, 6)",
        examples: ["cell_count_s == 6", "cell_count_s == 4", "cell_count_s == 1"],
      },
      {
        name: "capacity_mah",
        type: "number",
        description: "Capacity in milliampere-hours",
        examples: ["capacity_mah >= 1300", "capacity_mah == 1200"],
      },
      {
        name: "chemistry",
        type: "string",
        description: "Battery chemistry (LiPo, LiHV, Li-ion)",
        examples: ['chemistry == "LiPo"', 'chemistry == "LiHV"'],
      },
      {
        name: "connector",
        type: "string",
        description: "Connector type (XT60, XT30, BT2.0)",
        examples: ['connector == "XT60"', 'connector == "XT30"'],
      },
      {
        name: "weight_g",
        type: "number",
        description: "Battery weight in grams",
        examples: ["weight_g < 200.0", "weight_g <= 30.0"],
      },
      {
        name: "description",
        type: "string",
        description: "Product description",
        examples: ['description.contains("LiPo")'],
      },
    ],
    presets: [
      { label: "6S (22.2V)", query: "cell_count_s == 6" },
      { label: "4S (14.8V)", query: "cell_count_s == 4" },
      { label: "1S (3.7V)", query: "cell_count_s == 1" },
      {
        label: "1000 - 1500 mAh",
        query: "capacity_mah >= 1000 && capacity_mah <= 1500",
      },
      { label: "XT60 Connector", query: 'connector == "XT60"' },
      { label: "XT30 Connector", query: 'connector == "XT30"' },
      { label: "Under 150g", query: "weight_g < 150.0" },
    ],
    columns: {
      ...createStandardColumns(),
      cell_count_s: {
        id: "cell_count_s",
        title: "Cell Count (S)",
        renderCell: (b) => (b.cellCountS ? `${b.cellCountS}` : "-"),
      },
      capacity_mah: {
        id: "capacity_mah",
        title: "Capacity (mAh)",
        renderCell: (b) => (b.capacityMah ? `${b.capacityMah}` : "-"),
      },
      chemistry: {
        id: "chemistry",
        title: "Chemistry",
        renderCell: (b) => b.chemistry || "-",
      },
      connector: {
        id: "connector",
        title: "Connector",
        renderCell: (b) => b.connector || "-",
      },
    },
    highlights: [
      {
        label: "Cell Count",
        value: (b) => (b.cellCountS ? `${b.cellCountS}S` : "N/A"),
      },
      {
        label: "Capacity",
        value: (b) => (b.capacityMah ? `${b.capacityMah} mAh` : "N/A"),
      },
      { label: "Chemistry", value: (b) => b.chemistry || "N/A" },
      { label: "Connector", value: (b) => b.connector || "N/A" },
      {
        label: "Weight (g)",
        value: (b) => (b.weightG != null && b.weightG > 0 ? `${b.weightG}` : "N/A"),
      },
    ],
    technicalSpecs: [
      {
        label: "Cell Count (S)",
        value: (b) => (b.cellCountS ? `${b.cellCountS}` : "-"),
      },
      {
        label: "Capacity (mAh)",
        value: (b) => (b.capacityMah ? `${b.capacityMah}` : "-"),
      },
      { label: "Chemistry", value: (b) => b.chemistry || "-" },
      { label: "Connector", value: (b) => b.connector || "-" },
      {
        label: "Weight (g)",
        value: (b) => (b.weightG != null && b.weightG > 0 ? `${b.weightG}` : "-"),
      },
    ],
  },

  // 3. Cameras
  {
    id: "cameras",
    name: "Cameras",
    singular: "Camera",
    schema: CameraSchema,
    listQuery: listCameras,
    getQuery: getCamera,
    getDataList: (res, includeInternal = false) =>
      (res?.cameras ?? []).filter((item: any) => includeInternal || !item.isInternalOnly),
    defaultColumnIds: ["manufacturer", "name", "protocol", "sensor_size", "width_mm"],
    fields: [
      {
        name: "is_internal_only",
        type: "boolean",
        description: "Whether this camera is integrated/internal only",
        examples: ["is_internal_only == false", "is_internal_only == true"],
      },
      {
        name: "id",
        type: "string",
        description: "Unique identifier",
        examples: ['id.contains("avatar")', 'id.contains("camera")'],
      },
      {
        name: "uuid",
        type: "string",
        description: "Unique UUID identifier",
        examples: ['uuid.startsWith("a4c2")'],
      },
      {
        name: "manufacturer",
        type: "string",
        description: "Manufacturer / Brand name",
        examples: [
          'manufacturer == "Walksnail"',
          'manufacturer == "Foxeer"',
          'manufacturer == "RunCam"',
        ],
      },
      {
        name: "name",
        type: "string",
        description: "Camera model name",
        examples: ['name.contains("Pro")'],
      },
      {
        name: "protocol",
        type: "string",
        description: "Video protocol (Analog, DJI O3, DJI O4, Walksnail Avatar, HDZero, MIPI)",
        examples: ['protocol == "DJI O3"', 'protocol == "Analog"'],
      },
      {
        name: "sensor_size",
        type: "string",
        description: "Image sensor format",
        examples: ['sensor_size.contains("1/1.8")', 'sensor_size.contains("1/3")'],
      },
      {
        name: "width_mm",
        type: "number",
        description: "Mounting width in mm (14 for nano, 19 for micro, 22 for standard)",
        examples: ["width_mm == 19", "width_mm == 14"],
      },
      {
        name: "lens_size_mm",
        type: "number",
        description: "Lens focal length descriptor in mm",
        examples: ["lens_size_mm == 2.1", "lens_size_mm >= 1.8"],
      },
      {
        name: "weight_g",
        type: "number",
        description: "Camera weight in grams",
        examples: ["weight_g < 10.0", "weight_g <= 5.0"],
      },
      {
        name: "description",
        type: "string",
        description: "Product description",
        examples: ['description.contains("CMOS")'],
      },
    ],
    presets: [
      { label: "Digital Protocol", query: 'protocol != "Analog"' },
      { label: "Analog Protocol", query: 'protocol == "Analog"' },
      { label: "19mm Width (Micro)", query: "width_mm == 19" },
      { label: "14mm Width (Nano)", query: "width_mm == 14" },
      { label: '1/1.8" CMOS Sensor', query: 'sensor_size.contains("1/1.8")' },
      { label: "Under 10g", query: "weight_g < 10.0" },
    ],
    columns: {
      ...createStandardColumns(),
      protocol: {
        id: "protocol",
        title: "Protocol",
        renderCell: (c) => c.protocol || "-",
      },
      sensor_size: {
        id: "sensor_size",
        title: "Sensor Size",
        renderCell: (c) => c.sensorSize || "-",
      },
      width_mm: {
        id: "width_mm",
        title: "Width (mm)",
        renderCell: (c) => (c.widthMm ? `${c.widthMm}` : "-"),
      },
      lens_size_mm: {
        id: "lens_size_mm",
        title: "Lens Size (mm)",
        renderCell: (c) => (c.lensSizeMm != null && c.lensSizeMm > 0 ? `${c.lensSizeMm}` : "-"),
      },
    },
    highlights: [
      { label: "Protocol", value: (c) => c.protocol || "N/A" },
      { label: "Sensor Size", value: (c) => c.sensorSize || "N/A" },
      {
        label: "Width (mm)",
        value: (c) => (c.widthMm ? `${c.widthMm}` : "N/A"),
      },
      {
        label: "Lens Size (mm)",
        value: (c) => (c.lensSizeMm != null && c.lensSizeMm > 0 ? `${c.lensSizeMm}` : "N/A"),
      },
      {
        label: "Weight (g)",
        value: (c) => (c.weightG != null && c.weightG > 0 ? `${c.weightG}` : "N/A"),
      },
    ],
    technicalSpecs: [
      { label: "Protocol", value: (c) => c.protocol || "-" },
      { label: "Sensor Size", value: (c) => c.sensorSize || "-" },
      { label: "Width (mm)", value: (c) => (c.widthMm ? `${c.widthMm}` : "-") },
      {
        label: "Lens Size (mm)",
        value: (c) => (c.lensSizeMm != null && c.lensSizeMm > 0 ? `${c.lensSizeMm}` : "-"),
      },
      {
        label: "Weight (g)",
        value: (c) => (c.weightG != null && c.weightG > 0 ? `${c.weightG}` : "-"),
      },
    ],
  },

  // 4. Electronic Speed Controllers
  {
    id: "electronic-speed-controllers",
    aliases: ["electronic_speed_controllers", "escs"],
    name: "Electronic Speed Controllers",
    singular: "Electronic Speed Controller",
    schema: ElectronicSpeedControllerSchema,
    listQuery: listElectronicSpeedControllers,
    getQuery: getElectronicSpeedController,
    getDataList: (res, includeInternal = false) =>
      (res?.electronicSpeedControllers ?? []).filter(
        (item: any) => includeInternal || !item.isInternalOnly,
      ),
    defaultColumnIds: ["manufacturer", "name", "motor_current_max_a", "max_motors", "firmware"],
    fields: [
      {
        name: "is_internal_only",
        type: "boolean",
        description: "Whether this ESC is integrated/internal only",
        examples: ["is_internal_only == false", "is_internal_only == true"],
      },
      {
        name: "id",
        type: "string",
        description: "Unique identifier",
        examples: ['id.contains("aio")', 'id.contains("45a")'],
      },
      {
        name: "uuid",
        type: "string",
        description: "Unique UUID identifier",
        examples: ['uuid.startsWith("59b0")'],
      },
      {
        name: "manufacturer",
        type: "string",
        description: "Manufacturer / Brand name",
        examples: ['manufacturer == "HGLRC"', 'manufacturer == "Foxeer"'],
      },
      {
        name: "name",
        type: "string",
        description: "ESC model name",
        examples: ['name.contains("AIO")'],
      },
      {
        name: "motor_current_max_a",
        type: "number",
        description: "Continuous current per motor in Amperes",
        examples: ["motor_current_max_a >= 45.0", "motor_current_max_a >= 60.0"],
      },
      {
        name: "motor_current_burst_a",
        type: "number",
        description: "Burst current per motor in Amperes",
        examples: ["motor_current_burst_a >= 55.0"],
      },
      {
        name: "max_motors",
        type: "number",
        description: "Motor outputs (4 for 4-in-1, 1 for single)",
        examples: ["max_motors == 4", "max_motors == 1"],
      },
      {
        name: "firmware",
        type: "string",
        description: "ESC firmware (BLHeli_32, BLHeli_S, AM32, Bluejay)",
        examples: ['firmware.contains("BLHeli_32")', 'firmware.contains("AM32")'],
      },
      {
        name: "weight_g",
        type: "number",
        description: "ESC weight in grams",
        examples: ["weight_g < 15.0", "weight_g <= 10.0"],
      },
      {
        name: "description",
        type: "string",
        description: "Product description",
        examples: ['description.contains("BLHeli")'],
      },
    ],
    presets: [
      { label: "4-in-1 (4 Motors)", query: "max_motors == 4" },
      { label: "Single ESC (1 Motor)", query: "max_motors == 1" },
      { label: "45A+ Continuous", query: "motor_current_max_a >= 45.0" },
      { label: "60A+ Continuous", query: "motor_current_max_a >= 60.0" },
      { label: "BLHeli_32 Firmware", query: 'firmware.contains("BLHeli_32")' },
      { label: "AM32 Firmware", query: 'firmware.contains("AM32")' },
      { label: "Under 10g", query: "weight_g < 10.0" },
    ],
    columns: {
      ...createStandardColumns(),
      motor_current_max_a: {
        id: "motor_current_max_a",
        title: "Max Current (A)",
        renderCell: (e) => (e.motorCurrentMaxA ? `${e.motorCurrentMaxA}` : "-"),
      },
      motor_current_burst_a: {
        id: "motor_current_burst_a",
        title: "Burst Current (A)",
        renderCell: (e) => (e.motorCurrentBurstA ? `${e.motorCurrentBurstA}` : "-"),
      },
      max_motors: {
        id: "max_motors",
        title: "Motor Outputs",
        renderCell: (e) =>
          e.maxMotors === 4
            ? "4-in-1"
            : e.maxMotors === 1
              ? "Single"
              : e.maxMotors
                ? `${e.maxMotors}`
                : "-",
      },
      firmware: {
        id: "firmware",
        title: "Firmware",
        renderCell: (e) => e.firmware || "-",
      },
    },
    highlights: [
      {
        label: "Max Current",
        value: (e) => (e.motorCurrentMaxA ? `${e.motorCurrentMaxA} A` : "N/A"),
      },
      {
        label: "Burst Current",
        value: (e) => (e.motorCurrentBurstA ? `${e.motorCurrentBurstA} A` : "N/A"),
      },
      {
        label: "Motor Outputs",
        value: (e) =>
          e.maxMotors === 4
            ? "4-in-1"
            : e.maxMotors === 1
              ? "Single (1)"
              : e.maxMotors
                ? `${e.maxMotors}`
                : "N/A",
      },
      { label: "Firmware", value: (e) => e.firmware || "N/A" },
      {
        label: "Weight (g)",
        value: (e) => (e.weightG != null && e.weightG > 0 ? `${e.weightG}` : "N/A"),
      },
    ],
    technicalSpecs: [
      {
        label: "Max Continuous Current (A)",
        value: (e) => (e.motorCurrentMaxA ? `${e.motorCurrentMaxA}` : "-"),
      },
      {
        label: "Burst Current (A)",
        value: (e) => (e.motorCurrentBurstA ? `${e.motorCurrentBurstA}` : "-"),
      },
      {
        label: "Motor Outputs",
        value: (e) =>
          e.maxMotors === 4
            ? "4-in-1 (4 Motors)"
            : e.maxMotors === 1
              ? "Single (1 Motor)"
              : e.maxMotors
                ? `${e.maxMotors}`
                : "-",
      },
      { label: "Firmware", value: (e) => e.firmware || "-" },
      {
        label: "Weight (g)",
        value: (e) => (e.weightG != null && e.weightG > 0 ? `${e.weightG}` : "-"),
      },
    ],
  },

  // 5. Flight Controllers
  {
    id: "flight-controllers",
    aliases: ["flight_controllers", "fcs"],
    name: "Flight Controllers",
    singular: "Flight Controller",
    schema: FlightControllerSchema,
    listQuery: listFlightControllers,
    getQuery: getFlightController,
    getDataList: (res, includeInternal = false) =>
      (res?.flightControllers ?? []).filter((item: any) => includeInternal || !item.isInternalOnly),
    defaultColumnIds: ["manufacturer", "name", "processor", "gyro", "weight_g"],
    fields: [
      {
        name: "is_internal_only",
        type: "boolean",
        description: "Whether this flight controller is integrated/internal only",
        examples: ["is_internal_only == false", "is_internal_only == true"],
      },
      {
        name: "id",
        type: "string",
        description: "Unique identifier",
        examples: ['id.contains("f722")', 'id.contains("f405")'],
      },
      {
        name: "uuid",
        type: "string",
        description: "Unique UUID identifier",
        examples: ['uuid.startsWith("b233")'],
      },
      {
        name: "manufacturer",
        type: "string",
        description: "Manufacturer / Brand name",
        examples: ['manufacturer == "HGLRC"', 'manufacturer == "SpeedyBee"'],
      },
      {
        name: "name",
        type: "string",
        description: "Flight Controller model name",
        examples: ['name.contains("Zeus")'],
      },
      {
        name: "processor",
        type: "string",
        description: "Processor / MCU (STM32F405, STM32F722, STM32H743)",
        examples: ['processor.contains("F7")', 'processor.contains("F4")'],
      },
      {
        name: "gyro",
        type: "string",
        description: "IMU / Gyro sensor (MPU6000, BMI270, ICM42688)",
        examples: ['gyro.contains("MPU6000")', 'gyro.contains("BMI270")'],
      },
      {
        name: "weight_g",
        type: "number",
        description: "Flight Controller weight in grams",
        examples: ["weight_g < 10.0", "weight_g <= 6.0"],
      },
      {
        name: "description",
        type: "string",
        description: "Product description",
        examples: ['description.contains("MCU")'],
      },
    ],
    presets: [
      { label: "STM32F7 / F722 MCU", query: 'processor.contains("F7")' },
      { label: "STM32F4 / F405 MCU", query: 'processor.contains("F4")' },
      { label: "STM32H7 / H743 MCU", query: 'processor.contains("H7")' },
      { label: "MPU6000 Gyro", query: 'gyro.contains("MPU6000")' },
      { label: "BMI270 Gyro", query: 'gyro.contains("BMI270")' },
      { label: "ICM42688 Gyro", query: 'gyro.contains("ICM42688")' },
      { label: "Under 10g", query: "weight_g < 10.0" },
    ],
    columns: {
      ...createStandardColumns(),
      processor: {
        id: "processor",
        title: "Processor",
        renderCell: (f) => f.processor || "-",
      },
      gyro: {
        id: "gyro",
        title: "Gyro",
        renderCell: (f) => f.gyro || "-",
      },
      aio: {
        id: "aio",
        title: "Integrated AIO",
        renderCell: (f) =>
          f.internalElectronicSpeedControllerUuid ||
          f.internalReceiverUuid ||
          f.internalVideoTransmitterUuid
            ? "Yes"
            : "No",
      },
    },
    highlights: [
      { label: "Processor", value: (f) => f.processor || "N/A" },
      { label: "Gyro", value: (f) => f.gyro || "N/A" },
      {
        label: "Weight (g)",
        value: (f) => (f.weightG != null && f.weightG > 0 ? `${f.weightG}` : "N/A"),
      },
      {
        label: "Integrated ESC",
        value: (f) => (f.internalElectronicSpeedControllerUuid ? "Yes (AIO)" : "No"),
      },
      {
        label: "Integrated Receiver",
        value: (f) => (f.internalReceiverUuid ? "Yes" : "No"),
      },
      {
        label: "Integrated VTX",
        value: (f) => (f.internalVideoTransmitterUuid ? "Yes" : "No"),
      },
    ],
    technicalSpecs: [
      { label: "Processor (MCU)", value: (f) => f.processor || "-" },
      { label: "Gyro / IMU", value: (f) => f.gyro || "-" },
      {
        label: "Integrated ESC",
        value: (f) => (f.internalElectronicSpeedControllerUuid ? "Yes (AIO)" : "No"),
      },
      {
        label: "Integrated Receiver",
        value: (f) => (f.internalReceiverUuid ? "Yes" : "No"),
      },
      {
        label: "Integrated Video Transmitter",
        value: (f) => (f.internalVideoTransmitterUuid ? "Yes" : "No"),
      },
      {
        label: "Weight (g)",
        value: (f) => (f.weightG != null && f.weightG > 0 ? `${f.weightG}` : "-"),
      },
    ],
  },

  // 6. Frames
  {
    id: "frames",
    name: "Frames",
    singular: "Frame",
    schema: FrameSchema,
    listQuery: listFrames,
    getQuery: getFrame,
    getDataList: (res) => res?.frames ?? [],
    defaultColumnIds: ["manufacturer", "name", "wheelbase_mm", "geometry"],
    fields: [
      {
        name: "id",
        type: "string",
        description: "Unique identifier",
        examples: ['id.contains("apex")', 'id.contains("volador")'],
      },
      {
        name: "uuid",
        type: "string",
        description: "Unique UUID identifier",
        examples: ['uuid.startsWith("7338")'],
      },
      {
        name: "manufacturer",
        type: "string",
        description: "Manufacturer / Brand name",
        examples: ['manufacturer == "ImpulseRC"', 'manufacturer == "FlyFishRC"'],
      },
      {
        name: "name",
        type: "string",
        description: "Frame model name",
        examples: ['name.contains("Apex")'],
      },
      {
        name: "wheelbase_mm",
        type: "number",
        description: "Diagonal motor-to-motor wheelbase in mm",
        examples: ["wheelbase_mm >= 210.0 && wheelbase_mm <= 235.0", "wheelbase_mm > 220.0"],
      },
      {
        name: "max_prop_size_mm",
        type: "number",
        description: "Maximum propeller diameter supported in mm",
        examples: ["max_prop_size_mm >= 127.0", "max_prop_size_mm >= 130.0"],
      },
      {
        name: "geometry",
        type: "string",
        description: "Frame layout geometry (Deadcat, True-X, Squashed X)",
        examples: ['geometry == "Deadcat"', 'geometry == "True-X"'],
      },
      {
        name: "weight_g",
        type: "number",
        description: "Frame weight in grams",
        examples: ["weight_g < 140.0", "weight_g <= 120.0"],
      },
      {
        name: "description",
        type: "string",
        description: "Product description",
        examples: ['description.contains("freestyle")'],
      },
    ],
    presets: [
      { label: '5" Props (~127mm+)', query: "max_prop_size_mm >= 127.0" },
      { label: "Deadcat Geometry", query: 'geometry == "Deadcat"' },
      {
        label: "True-X Geometry",
        query: 'geometry == "True-X" || geometry == "True X"',
      },
      {
        label: "Wheelbase 210-235mm",
        query: "wheelbase_mm >= 210.0 && wheelbase_mm <= 235.0",
      },
      { label: "Under 130g", query: "weight_g < 130.0" },
    ],
    columns: {
      ...createStandardColumns(),
      wheelbase_mm: {
        id: "wheelbase_mm",
        title: "Wheelbase (mm)",
        renderCell: (f) => (f.wheelbaseMm ? `${f.wheelbaseMm}` : "-"),
      },
      max_prop_size_mm: {
        id: "max_prop_size_mm",
        title: "Max Prop (mm)",
        renderCell: (f) => (f.maxPropSizeMm ? `${f.maxPropSizeMm}` : "-"),
      },
      geometry: {
        id: "geometry",
        title: "Geometry",
        renderCell: (f) => f.geometry || "-",
      },
    },
    highlights: [
      {
        label: "Wheelbase",
        value: (f) => (f.wheelbaseMm ? `${f.wheelbaseMm} mm` : "N/A"),
      },
      {
        label: "Max Prop Size",
        value: (f) =>
          f.maxPropSizeMm
            ? `${f.maxPropSizeMm} mm (~${(f.maxPropSizeMm / 25.4).toFixed(1)}")`
            : "N/A",
      },
      { label: "Geometry", value: (f) => f.geometry || "N/A" },
      {
        label: "Weight (g)",
        value: (f) => (f.weightG != null && f.weightG > 0 ? `${f.weightG}` : "N/A"),
      },
    ],
    technicalSpecs: [
      {
        label: "Wheelbase (mm)",
        value: (f) => (f.wheelbaseMm ? `${f.wheelbaseMm}` : "-"),
      },
      {
        label: "Max Propeller Size (mm)",
        value: (f) =>
          f.maxPropSizeMm
            ? `${f.maxPropSizeMm} mm (~${(f.maxPropSizeMm / 25.4).toFixed(1)}")`
            : "-",
      },
      { label: "Geometry", value: (f) => f.geometry || "-" },
      {
        label: "Weight (g)",
        value: (f) => (f.weightG != null && f.weightG > 0 ? `${f.weightG}` : "-"),
      },
    ],
  },

  // 7. GPS Receivers
  {
    id: "gps-receivers",
    aliases: ["gps_receivers", "gps"],
    name: "GPS Receivers",
    singular: "GPS Receiver",
    schema: GpsReceiverSchema,
    listQuery: listGpsReceivers,
    getQuery: getGpsReceiver,
    getDataList: (res, includeInternal = false) =>
      (res?.gpsReceivers ?? []).filter((item: any) => includeInternal || !item.isInternalOnly),
    defaultColumnIds: ["manufacturer", "name", "chipset", "has_compass"],
    fields: [
      {
        name: "is_internal_only",
        type: "boolean",
        description: "Whether this GPS receiver is integrated/internal only",
        examples: ["is_internal_only == false", "is_internal_only == true"],
      },
      {
        name: "id",
        type: "string",
        description: "Unique identifier",
        examples: ['id.contains("m10")', 'id.contains("gps")'],
      },
      {
        name: "uuid",
        type: "string",
        description: "Unique UUID identifier",
        examples: ['uuid.startsWith("0192")'],
      },
      {
        name: "manufacturer",
        type: "string",
        description: "Manufacturer / Brand name",
        examples: ['manufacturer == "Matek Systems"', 'manufacturer == "Flywoo"'],
      },
      {
        name: "name",
        type: "string",
        description: "GPS receiver model name",
        examples: ['name.contains("M10")'],
      },
      {
        name: "chipset",
        type: "string",
        description: "GNSS chipset model",
        examples: ['chipset.contains("M10")', 'chipset.contains("MAX-M10S")'],
      },
      {
        name: "protocol",
        type: "string",
        description: "Serial communication protocol (UBLOX, NMEA)",
        examples: ['protocol == "UBLOX"', 'protocol == "NMEA"'],
      },
      {
        name: "has_compass",
        type: "boolean",
        description: "Integrated magnetic compass",
        examples: ["has_compass == true", "has_compass == false"],
      },
      {
        name: "compass_chip",
        type: "string",
        description: "Magnetometer / Compass chip model",
        examples: ['compass_chip == "QMC5883L"'],
      },
      {
        name: "input_voltage_min_v",
        type: "number",
        description: "Minimum operating voltage",
        examples: ["input_voltage_min_v <= 3.3"],
      },
      {
        name: "input_voltage_max_v",
        type: "number",
        description: "Maximum operating voltage",
        examples: ["input_voltage_max_v >= 5.0"],
      },
      {
        name: "weight_g",
        type: "number",
        description: "GPS module weight in grams",
        examples: ["weight_g < 5.0", "weight_g <= 3.0"],
      },
      {
        name: "description",
        type: "string",
        description: "Product description",
        examples: ['description.contains("GNSS")'],
      },
    ],
    presets: [
      { label: "u-blox M10 Chipset", query: 'chipset.contains("M10")' },
      { label: "Has Integrated Compass", query: "has_compass == true" },
      { label: "UBLOX Protocol", query: 'protocol == "UBLOX"' },
      { label: "Under 5g", query: "weight_g < 5.0" },
      { label: "Matek Systems", query: 'manufacturer.contains("Matek")' },
    ],
    columns: {
      ...createStandardColumns(),
      chipset: {
        id: "chipset",
        title: "Chipset",
        renderCell: (g) => g.chipset || "-",
      },
      protocol: {
        id: "protocol",
        title: "Protocol",
        renderCell: (g) => g.protocol || "-",
      },
      has_compass: {
        id: "has_compass",
        title: "Compass",
        renderCell: (g) => (g.hasCompass === true ? "Yes" : g.hasCompass === false ? "No" : "-"),
      },
      compass_chip: {
        id: "compass_chip",
        title: "Compass Chip",
        renderCell: (g) => g.compassChip || "-",
      },
      voltage_range: {
        id: "voltage_range",
        title: "Voltage (V)",
        renderCell: (g) =>
          g.inputVoltageMinV && g.inputVoltageMaxV
            ? `${g.inputVoltageMinV}-${g.inputVoltageMaxV}`
            : "-",
      },
    },
    highlights: [
      { label: "Chipset", value: (g) => g.chipset || "N/A" },
      {
        label: "Compass",
        value: (g) => (g.hasCompass ? (g.compassChip ? `Yes (${g.compassChip})` : "Yes") : "No"),
      },
      { label: "Protocol", value: (g) => g.protocol || "N/A" },
      {
        label: "Voltage Range",
        value: (g) =>
          g.inputVoltageMinV && g.inputVoltageMaxV
            ? `${g.inputVoltageMinV}V - ${g.inputVoltageMaxV}V`
            : "N/A",
      },
      {
        label: "Weight (g)",
        value: (g) => (g.weightG != null && g.weightG > 0 ? `${g.weightG}` : "N/A"),
      },
    ],
    technicalSpecs: [
      { label: "Chipset", value: (g) => g.chipset || "-" },
      { label: "Protocol", value: (g) => g.protocol || "-" },
      {
        label: "Integrated Compass",
        value: (g) => (g.hasCompass ? "Yes" : "No"),
      },
      { label: "Compass Chip", value: (g) => g.compassChip || "-" },
      {
        label: "Min Input Voltage (V)",
        value: (g) => (g.inputVoltageMinV ? `${g.inputVoltageMinV}` : "-"),
      },
      {
        label: "Max Input Voltage (V)",
        value: (g) => (g.inputVoltageMaxV ? `${g.inputVoltageMaxV}` : "-"),
      },
      {
        label: "Weight (g)",
        value: (g) => (g.weightG != null && g.weightG > 0 ? `${g.weightG}` : "-"),
      },
    ],
  },

  // 8. Motors
  {
    id: "motors",
    name: "Motors",
    singular: "Motor",
    schema: MotorSchema,
    listQuery: listMotors,
    getQuery: getMotor,
    getDataList: (res) => res?.motors ?? [],
    defaultColumnIds: ["manufacturer", "name", "kv", "stator_diameter_mm"],
    fields: [
      {
        name: "id",
        type: "string",
        description: "Unique identifier",
        examples: ['id == "flash-2207-fpv-motor"', 'id.contains("2207")'],
      },
      {
        name: "uuid",
        type: "string",
        description: "Unique UUID identifier",
        examples: ['uuid.startsWith("f07a")'],
      },
      {
        name: "manufacturer",
        type: "string",
        description: "Manufacturer / Brand name",
        examples: ['manufacturer.contains("T-Motor")', 'manufacturer == "Emax"'],
      },
      {
        name: "name",
        type: "string",
        description: "Motor model name",
        examples: ['name.contains("Velox")'],
      },
      {
        name: "kv",
        type: "number",
        description: "RPM per volt (velocity constant)",
        examples: ["kv >= 1900", "kv == 1750"],
      },
      {
        name: "weight_g",
        type: "number",
        description: "Motor weight in grams",
        examples: ["weight_g < 35.0"],
      },
      {
        name: "stator_diameter_mm",
        type: "number",
        description: "Stator diameter in millimeters",
        examples: ["stator_diameter_mm == 22.0"],
      },
      {
        name: "stator_height_mm",
        type: "number",
        description: "Stator height in millimeters",
        examples: ["stator_height_mm == 7.0"],
      },
      {
        name: "description",
        type: "string",
        description: "Product description",
        examples: ['description.contains("brushless")'],
      },
    ],
    presets: [
      {
        label: "2207 Stator",
        query: "stator_diameter_mm == 22.0 && stator_height_mm == 7.0",
      },
      {
        label: "2306 Stator",
        query: "stator_diameter_mm == 23.0 && stator_height_mm == 6.0",
      },
      {
        label: "1404 Stator",
        query: "stator_diameter_mm == 14.0 && stator_height_mm == 4.0",
      },
      { label: "Under 35g", query: "weight_g < 35.0" },
      { label: "T-Motor", query: 'manufacturer.contains("T-Motor")' },
    ],
    columns: {
      ...createStandardColumns(),
      kv: {
        id: "kv",
        title: "KV",
        renderCell: (m) => (m.kv ? `${m.kv}` : "-"),
      },
      stator_diameter_mm: {
        id: "stator_diameter_mm",
        title: "Stator Size",
        renderCell: (m) => (
          <span className="font-mono">
            {formatStatorSize(m.statorDiameterMm, m.statorHeightMm)}
          </span>
        ),
      },
    },
    highlights: [
      { label: "KV", value: (m) => m.kv || "N/A" },
      {
        label: "Weight (g)",
        value: (m) => (m.weightG ? `${m.weightG}` : "N/A"),
      },
      {
        label: "Stator Size",
        value: (m) => (
          <span className="font-mono">
            {formatStatorSize(m.statorDiameterMm, m.statorHeightMm, "N/A")}
          </span>
        ),
      },
    ],
    technicalSpecs: [
      { label: "KV Rating", value: (m) => (m.kv ? `${m.kv}` : "-") },
      {
        label: "Stator Diameter (mm)",
        value: (m) => (m.statorDiameterMm ? `${m.statorDiameterMm}` : "-"),
      },
      {
        label: "Stator Height (mm)",
        value: (m) => (m.statorHeightMm ? `${m.statorHeightMm}` : "-"),
      },
      {
        label: "Stator Size",
        value: (m) => formatStatorSize(m.statorDiameterMm, m.statorHeightMm, "-"),
      },
      { label: "Weight (g)", value: (m) => (m.weightG ? `${m.weightG}` : "-") },
    ],
  },

  // 9. Propellers
  {
    id: "propellers",
    name: "Propellers",
    singular: "Propeller",
    schema: PropellerSchema,
    listQuery: listPropellers,
    getQuery: getPropeller,
    getDataList: (res) => res?.propellers ?? [],
    defaultColumnIds: ["manufacturer", "name", "diameter_mm", "pitch_mm", "blades"],
    fields: [
      {
        name: "id",
        type: "string",
        description: "Unique identifier",
        examples: ['id.contains("5x4.3x3")', 'id.contains("prop")'],
      },
      {
        name: "uuid",
        type: "string",
        description: "Unique UUID identifier",
        examples: ['uuid.startsWith("5bb5")'],
      },
      {
        name: "manufacturer",
        type: "string",
        description: "Manufacturer / Brand name",
        examples: ['manufacturer == "HQProp"', 'manufacturer == "Gemfan"'],
      },
      {
        name: "name",
        type: "string",
        description: "Propeller model name",
        examples: ['name.contains("V1S")'],
      },
      {
        name: "diameter_mm",
        type: "number",
        description: 'Propeller diameter in millimeters (127mm for 5")',
        examples: ["diameter_mm >= 126.0 && diameter_mm <= 132.0", "diameter_mm > 100.0"],
      },
      {
        name: "pitch_mm",
        type: "number",
        description: "Propeller pitch in millimeters",
        examples: ["pitch_mm >= 90.0", "pitch_mm <= 115.0"],
      },
      {
        name: "blades",
        type: "number",
        description: "Number of propeller blades (2, 3, 4)",
        examples: ["blades == 3", "blades == 2"],
      },
      {
        name: "material",
        type: "string",
        description: "Blade material (Polycarbonate, Carbon Fiber)",
        examples: ['material.contains("Polycarbonate")'],
      },
      {
        name: "weight_g",
        type: "number",
        description: "Single propeller weight in grams",
        examples: ["weight_g < 4.0", "weight_g <= 1.0"],
      },
      {
        name: "description",
        type: "string",
        description: "Product description",
        examples: ['description.contains("blade")'],
      },
    ],
    presets: [
      {
        label: '5" Diameter (~127mm)',
        query: "diameter_mm >= 126.0 && diameter_mm <= 132.0",
      },
      { label: "3-Blade (Tri-Blade)", query: "blades == 3" },
      { label: "2-Blade (Bi-Blade)", query: "blades == 2" },
      {
        label: "Polycarbonate Material",
        query: 'material.contains("Polycarbonate")',
      },
      { label: "Under 4g", query: "weight_g < 4.0" },
      { label: "HQProp", query: 'manufacturer.contains("HQProp")' },
    ],
    columns: {
      ...createStandardColumns(),
      diameter_mm: {
        id: "diameter_mm",
        title: "Diameter (mm)",
        renderCell: (p) => (p.diameterMm ? `${p.diameterMm}` : "-"),
      },
      pitch_mm: {
        id: "pitch_mm",
        title: "Pitch (mm)",
        renderCell: (p) => (p.pitchMm ? `${p.pitchMm}` : "-"),
      },
      blades: {
        id: "blades",
        title: "Blades",
        renderCell: (p) => (p.blades ? `${p.blades}` : "-"),
      },
      material: {
        id: "material",
        title: "Material",
        renderCell: (p) => p.material || "-",
      },
    },
    highlights: [
      {
        label: "Diameter",
        value: (p) =>
          p.diameterMm ? `${p.diameterMm} mm (~${(p.diameterMm / 25.4).toFixed(1)}")` : "N/A",
      },
      {
        label: "Pitch",
        value: (p) => (p.pitchMm ? `${p.pitchMm} mm (~${(p.pitchMm / 25.4).toFixed(1)}")` : "N/A"),
      },
      {
        label: "Blades",
        value: (p) => (p.blades ? `${p.blades} Blades` : "N/A"),
      },
      { label: "Material", value: (p) => p.material || "N/A" },
      {
        label: "Weight (g)",
        value: (p) => (p.weightG != null && p.weightG > 0 ? `${p.weightG}` : "N/A"),
      },
    ],
    technicalSpecs: [
      {
        label: "Diameter (mm)",
        value: (p) =>
          p.diameterMm ? `${p.diameterMm} mm (~${(p.diameterMm / 25.4).toFixed(1)}")` : "-",
      },
      {
        label: "Pitch (mm)",
        value: (p) => (p.pitchMm ? `${p.pitchMm} mm (~${(p.pitchMm / 25.4).toFixed(1)}")` : "-"),
      },
      {
        label: "Blades",
        value: (p) => (p.blades ? `${p.blades} Blades` : "-"),
      },
      { label: "Material", value: (p) => p.material || "-" },
      {
        label: "Weight (g)",
        value: (p) => (p.weightG != null && p.weightG > 0 ? `${p.weightG}` : "-"),
      },
    ],
  },

  // 10. Receivers
  {
    id: "receivers",
    name: "Receivers",
    singular: "Receiver",
    schema: ReceiverSchema,
    listQuery: listReceivers,
    getQuery: getReceiver,
    getDataList: (res, includeInternal = false) =>
      (res?.receivers ?? []).filter((item: any) => includeInternal || !item.isInternalOnly),
    defaultColumnIds: ["manufacturer", "name", "protocol", "frequency_band_mhz", "has_telemetry"],
    fields: [
      {
        name: "is_internal_only",
        type: "boolean",
        description: "Whether this receiver is integrated/internal only",
        examples: ["is_internal_only == false", "is_internal_only == true"],
      },
      {
        name: "id",
        type: "string",
        description: "Unique identifier",
        examples: ['id.contains("crossfire")', 'id.contains("elrs")'],
      },
      {
        name: "uuid",
        type: "string",
        description: "Unique UUID identifier",
        examples: ['uuid.startsWith("b0cf")'],
      },
      {
        name: "manufacturer",
        type: "string",
        description: "Manufacturer / Brand name",
        examples: [
          'manufacturer.contains("Team Black Sheep")',
          'manufacturer.contains("Happymodel")',
        ],
      },
      {
        name: "name",
        type: "string",
        description: "Receiver model name",
        examples: ['name.contains("Crossfire")'],
      },
      {
        name: "protocol",
        type: "string",
        description: "Radio protocol (ExpressLRS, TBS Crossfire (CRSF), FrSky)",
        examples: ['protocol.contains("Crossfire")', 'protocol.contains("ExpressLRS")'],
      },
      {
        name: "frequency_band_mhz",
        type: "number",
        description: "Frequency band in MHz (2400, 915, 868)",
        examples: ["frequency_band_mhz == 2400", "frequency_band_mhz == 915"],
      },
      {
        name: "has_telemetry",
        type: "boolean",
        description: "Telemetry transmission support",
        examples: ["has_telemetry == true", "has_telemetry == false"],
      },
      {
        name: "weight_g",
        type: "number",
        description: "Receiver weight in grams",
        examples: ["weight_g < 1.0", "weight_g <= 3.5"],
      },
      {
        name: "description",
        type: "string",
        description: "Product description",
        examples: ['description.contains("CRSF")'],
      },
    ],
    presets: [
      { label: "TBS Crossfire", query: 'protocol.contains("Crossfire")' },
      { label: "ExpressLRS (ELRS)", query: 'protocol.contains("ExpressLRS")' },
      { label: "2.4GHz Band", query: "frequency_band_mhz == 2400" },
      { label: "915MHz Band", query: "frequency_band_mhz == 915" },
      { label: "Telemetry Supported", query: "has_telemetry == true" },
      { label: "Sub-1g Micro RX", query: "weight_g < 1.0" },
    ],
    columns: {
      ...createStandardColumns(),
      protocol: {
        id: "protocol",
        title: "Protocol",
        renderCell: (r) => r.protocol || "-",
      },
      frequency_band_mhz: {
        id: "frequency_band_mhz",
        title: "Frequency (MHz)",
        renderCell: (r) => (r.frequencyBandMhz ? `${r.frequencyBandMhz}` : "-"),
      },
      has_telemetry: {
        id: "has_telemetry",
        title: "Telemetry",
        renderCell: (r) => (r.hasTelemetry ? "Yes" : "No"),
      },
    },
    highlights: [
      { label: "Protocol", value: (r) => r.protocol || "N/A" },
      {
        label: "Frequency",
        value: (r) => (r.frequencyBandMhz ? `${r.frequencyBandMhz} MHz` : "N/A"),
      },
      {
        label: "Telemetry",
        value: (r) => (r.hasTelemetry ? "Supported" : "No"),
      },
      {
        label: "Weight (g)",
        value: (r) => (r.weightG != null && r.weightG > 0 ? `${r.weightG}` : "N/A"),
      },
    ],
    technicalSpecs: [
      { label: "Protocol", value: (r) => r.protocol || "-" },
      {
        label: "Frequency Band (MHz)",
        value: (r) => (r.frequencyBandMhz ? `${r.frequencyBandMhz}` : "-"),
      },
      {
        label: "Telemetry",
        value: (r) => (r.hasTelemetry ? "Supported" : "No"),
      },
      {
        label: "Weight (g)",
        value: (r) => (r.weightG != null && r.weightG > 0 ? `${r.weightG}` : "-"),
      },
    ],
  },

  // 11. Video Transmitters
  {
    id: "video-transmitters",
    aliases: ["video_transmitters", "vtxs", "vtx"],
    name: "Video Transmitters",
    singular: "Video Transmitter",
    schema: VideoTransmitterSchema,
    listQuery: listVideoTransmitters,
    getQuery: getVideoTransmitter,
    getDataList: (res, includeInternal = false) =>
      (res?.videoTransmitters ?? []).filter((item: any) => includeInternal || !item.isInternalOnly),
    defaultColumnIds: ["manufacturer", "name", "protocol", "max_power_mw", "weight_g"],
    fields: [
      {
        name: "is_internal_only",
        type: "boolean",
        description: "Whether this video transmitter is integrated/internal only",
        examples: ["is_internal_only == false", "is_internal_only == true"],
      },
      {
        name: "id",
        type: "string",
        description: "Unique identifier",
        examples: ['id.contains("unify")', 'id.contains("avatar")', 'id.contains("vtx")'],
      },
      {
        name: "uuid",
        type: "string",
        description: "Unique UUID identifier",
        examples: ['uuid.startsWith("ecc4")'],
      },
      {
        name: "manufacturer",
        type: "string",
        description: "Manufacturer / Brand name",
        examples: [
          'manufacturer == "Walksnail"',
          'manufacturer.contains("Team Black Sheep")',
          'manufacturer == "HDZero"',
        ],
      },
      {
        name: "name",
        type: "string",
        description: "Video Transmitter model name",
        examples: ['name.contains("Avatar")'],
      },
      {
        name: "protocol",
        type: "string",
        description:
          "Video transmission protocol (Analog, DJI O3, DJI O4, Walksnail Avatar, HDZero)",
        examples: ['protocol.contains("DJI")', 'protocol == "Analog"'],
      },
      {
        name: "max_power_mw",
        type: "number",
        description: "Maximum RF output power in milliwatts",
        examples: ["max_power_mw >= 800", "max_power_mw >= 1000"],
      },
      {
        name: "input_voltage_min_v",
        type: "number",
        description: "Minimum operating input voltage",
        examples: ["input_voltage_min_v <= 3.3"],
      },
      {
        name: "input_voltage_max_v",
        type: "number",
        description: "Maximum operating input voltage",
        examples: ["input_voltage_max_v >= 5.0"],
      },
      {
        name: "weight_g",
        type: "number",
        description: "Video Transmitter weight in grams",
        examples: ["weight_g < 10.0", "weight_g <= 3.0"],
      },
      {
        name: "description",
        type: "string",
        description: "Product description",
        examples: ['description.contains("output power")'],
      },
    ],
    presets: [
      {
        label: "Digital Protocol",
        query: 'protocol != "Analog"',
      },
      { label: "Analog Protocol", query: 'protocol.contains("Analog")' },
      { label: "1000mW+ (1W+) Output", query: "max_power_mw >= 1000" },
      {
        label: "400 - 800mW Output",
        query: "max_power_mw >= 400 && max_power_mw <= 800",
      },
      { label: "Under 10g", query: "weight_g < 10.0" },
    ],
    columns: {
      ...createStandardColumns(),
      protocol: {
        id: "protocol",
        title: "Protocol",
        renderCell: (v) => v.protocol || "-",
      },
      max_power_mw: {
        id: "max_power_mw",
        title: "Max Power (mW)",
        renderCell: (v) => (v.maxPowerMw ? `${v.maxPowerMw}` : "-"),
      },
      voltage_range: {
        id: "voltage_range",
        title: "Voltage (V)",
        renderCell: (v) =>
          v.inputVoltageMinV && v.inputVoltageMaxV
            ? `${v.inputVoltageMinV}-${v.inputVoltageMaxV}`
            : "-",
      },
    },
    highlights: [
      { label: "Protocol", value: (v) => v.protocol || "N/A" },
      {
        label: "Max Power",
        value: (v) => (v.maxPowerMw ? `${v.maxPowerMw} mW` : "N/A"),
      },
      {
        label: "Voltage Range",
        value: (v) =>
          v.inputVoltageMinV && v.inputVoltageMaxV
            ? `${v.inputVoltageMinV}V - ${v.inputVoltageMaxV}V`
            : "N/A",
      },
      {
        label: "Weight (g)",
        value: (v) => (v.weightG != null && v.weightG > 0 ? `${v.weightG}` : "N/A"),
      },
    ],
    technicalSpecs: [
      { label: "Protocol", value: (v) => v.protocol || "-" },
      {
        label: "Max Output Power (mW)",
        value: (v) => (v.maxPowerMw ? `${v.maxPowerMw}` : "-"),
      },
      {
        label: "Min Input Voltage (V)",
        value: (v) => (v.inputVoltageMinV ? `${v.inputVoltageMinV}` : "-"),
      },
      {
        label: "Max Input Voltage (V)",
        value: (v) => (v.inputVoltageMaxV ? `${v.inputVoltageMaxV}` : "-"),
      },
      {
        label: "Weight (g)",
        value: (v) => (v.weightG != null && v.weightG > 0 ? `${v.weightG}` : "-"),
      },
    ],
  },
];

export function getHardwareCollection(collectionId?: string): HardwareCollectionDef | undefined {
  if (!collectionId) return undefined;
  const normalized = collectionId.toLowerCase().trim();
  const kebab = normalized.replace(/_/g, "-");
  return HARDWARE_COLLECTIONS.find(
    (c) =>
      c.id === normalized ||
      c.id === kebab ||
      c.aliases?.includes(normalized) ||
      c.aliases?.includes(kebab),
  );
}
