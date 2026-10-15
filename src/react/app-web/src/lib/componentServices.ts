import {
  createAntenna,
  updateAntenna,
  deleteAntenna,
} from "../gen/quadsmith/api/antenna-AntennaService_connectquery";
import { AntennaSchema } from "../gen/quadsmith/api/antenna_pb";
import {
  createBattery,
  updateBattery,
  deleteBattery,
} from "../gen/quadsmith/api/battery-BatteryService_connectquery";
import { BatterySchema } from "../gen/quadsmith/api/battery_pb";
import {
  createCamera,
  updateCamera,
  deleteCamera,
} from "../gen/quadsmith/api/camera-CameraService_connectquery";
import { CameraSchema } from "../gen/quadsmith/api/camera_pb";
import {
  createCharger,
  updateCharger,
  deleteCharger,
} from "../gen/quadsmith/api/charger-ChargerService_connectquery";
import { ChargerSchema } from "../gen/quadsmith/api/charger_pb";
import {
  createElectronicSpeedController,
  updateElectronicSpeedController,
  deleteElectronicSpeedController,
} from "../gen/quadsmith/api/electronic_speed_controller-ElectronicSpeedControllerService_connectquery";
import { ElectronicSpeedControllerSchema } from "../gen/quadsmith/api/electronic_speed_controller_pb";
import {
  createEscFirmware,
  updateEscFirmware,
  deleteEscFirmware,
} from "../gen/quadsmith/api/esc_firmware-EscFirmwareService_connectquery";
import { EscFirmwareSchema } from "../gen/quadsmith/api/esc_firmware_pb";
import {
  createFcFirmware,
  updateFcFirmware,
  deleteFcFirmware,
} from "../gen/quadsmith/api/fc_firmware-FcFirmwareService_connectquery";
import { FcFirmwareSchema } from "../gen/quadsmith/api/fc_firmware_pb";
import {
  createFieldTool,
  updateFieldTool,
  deleteFieldTool,
} from "../gen/quadsmith/api/field_tool-FieldToolService_connectquery";
import { FieldToolSchema } from "../gen/quadsmith/api/field_tool_pb";
import {
  createFlightController,
  updateFlightController,
  deleteFlightController,
} from "../gen/quadsmith/api/flight_controller-FlightControllerService_connectquery";
import { FlightControllerSchema } from "../gen/quadsmith/api/flight_controller_pb";
import {
  createFrame,
  updateFrame,
  deleteFrame,
} from "../gen/quadsmith/api/frame-FrameService_connectquery";
import { FrameSchema } from "../gen/quadsmith/api/frame_pb";
import {
  createGoggles,
  updateGoggles,
  deleteGoggles,
} from "../gen/quadsmith/api/goggles-GogglesService_connectquery";
import { GogglesSchema } from "../gen/quadsmith/api/goggles_pb";
import {
  createGpsReceiver,
  updateGpsReceiver,
  deleteGpsReceiver,
} from "../gen/quadsmith/api/gps_receiver-GpsReceiverService_connectquery";
import { GpsReceiverSchema } from "../gen/quadsmith/api/gps_receiver_pb";
import {
  createMotor,
  updateMotor,
  deleteMotor,
} from "../gen/quadsmith/api/motor-MotorService_connectquery";
import { MotorSchema } from "../gen/quadsmith/api/motor_pb";
import {
  createPropeller,
  updatePropeller,
  deletePropeller,
} from "../gen/quadsmith/api/propeller-PropellerService_connectquery";
import { PropellerSchema } from "../gen/quadsmith/api/propeller_pb";
import {
  createRadioFirmware,
  updateRadioFirmware,
  deleteRadioFirmware,
} from "../gen/quadsmith/api/radio_firmware-RadioFirmwareService_connectquery";
import { RadioFirmwareSchema } from "../gen/quadsmith/api/radio_firmware_pb";
import {
  createReceiver,
  updateReceiver,
  deleteReceiver,
} from "../gen/quadsmith/api/receiver-ReceiverService_connectquery";
import { ReceiverSchema } from "../gen/quadsmith/api/receiver_pb";
import {
  createSoftwareTool,
  updateSoftwareTool,
  deleteSoftwareTool,
} from "../gen/quadsmith/api/software_tool-SoftwareToolService_connectquery";
import { SoftwareToolSchema } from "../gen/quadsmith/api/software_tool_pb";
import {
  createTransmitter,
  updateTransmitter,
  deleteTransmitter,
} from "../gen/quadsmith/api/transmitter-TransmitterService_connectquery";
import { TransmitterSchema } from "../gen/quadsmith/api/transmitter_pb";
import {
  createVideoTransmitter,
  updateVideoTransmitter,
  deleteVideoTransmitter,
} from "../gen/quadsmith/api/video_transmitter-VideoTransmitterService_connectquery";
import { VideoTransmitterSchema } from "../gen/quadsmith/api/video_transmitter_pb";
import {
  createVtxFirmware,
  updateVtxFirmware,
  deleteVtxFirmware,
} from "../gen/quadsmith/api/vtx_firmware-VtxFirmwareService_connectquery";
import { VtxFirmwareSchema } from "../gen/quadsmith/api/vtx_firmware_pb";

export interface ComponentServiceDef {
  collectionId: string;
  aliases: string[];
  paramKey: string;
  singularLabel: string;
  schema: any;
  createMethod: any;
  updateMethod: any;
  deleteMethod: any;
}

export const COMPONENT_SERVICES: ComponentServiceDef[] = [
  {
    collectionId: "antennas",
    aliases: ["antenna", "components/hardware/antennas"],
    paramKey: "antenna",
    singularLabel: "Antenna",
    schema: AntennaSchema,
    createMethod: createAntenna,
    updateMethod: updateAntenna,
    deleteMethod: deleteAntenna,
  },
  {
    collectionId: "batteries",
    aliases: ["battery", "components/hardware/batteries"],
    paramKey: "battery",
    singularLabel: "Battery",
    schema: BatterySchema,
    createMethod: createBattery,
    updateMethod: updateBattery,
    deleteMethod: deleteBattery,
  },
  {
    collectionId: "cameras",
    aliases: ["camera", "components/hardware/cameras"],
    paramKey: "camera",
    singularLabel: "Camera",
    schema: CameraSchema,
    createMethod: createCamera,
    updateMethod: updateCamera,
    deleteMethod: deleteCamera,
  },
  {
    collectionId: "chargers",
    aliases: ["charger", "components/gear/chargers"],
    paramKey: "charger",
    singularLabel: "Charger",
    schema: ChargerSchema,
    createMethod: createCharger,
    updateMethod: updateCharger,
    deleteMethod: deleteCharger,
  },
  {
    collectionId: "electronic-speed-controllers",
    aliases: [
      "electronic_speed_controllers",
      "electronic-speed-controller",
      "electronic_speed_controller",
      "escs",
      "esc",
      "components/hardware/electronic-speed-controllers",
    ],
    paramKey: "electronicSpeedController",
    singularLabel: "Electronic Speed Controller",
    schema: ElectronicSpeedControllerSchema,
    createMethod: createElectronicSpeedController,
    updateMethod: updateElectronicSpeedController,
    deleteMethod: deleteElectronicSpeedController,
  },
  {
    collectionId: "esc-firmwares",
    aliases: ["esc_firmwares", "esc-firmware", "esc_firmware", "components/software/esc-firmwares"],
    paramKey: "escFirmware",
    singularLabel: "ESC Firmware",
    schema: EscFirmwareSchema,
    createMethod: createEscFirmware,
    updateMethod: updateEscFirmware,
    deleteMethod: deleteEscFirmware,
  },
  {
    collectionId: "fc-firmwares",
    aliases: ["fc_firmwares", "fc-firmware", "fc_firmware", "components/software/fc-firmwares"],
    paramKey: "fcFirmware",
    singularLabel: "FC Firmware",
    schema: FcFirmwareSchema,
    createMethod: createFcFirmware,
    updateMethod: updateFcFirmware,
    deleteMethod: deleteFcFirmware,
  },
  {
    collectionId: "field-tools",
    aliases: ["field_tools", "field-tool", "field_tool", "components/gear/field-tools"],
    paramKey: "fieldTool",
    singularLabel: "Field Tool",
    schema: FieldToolSchema,
    createMethod: createFieldTool,
    updateMethod: updateFieldTool,
    deleteMethod: deleteFieldTool,
  },
  {
    collectionId: "flight-controllers",
    aliases: [
      "flight_controllers",
      "flight-controller",
      "flight_controller",
      "fcs",
      "fc",
      "components/hardware/flight-controllers",
    ],
    paramKey: "flightController",
    singularLabel: "Flight Controller",
    schema: FlightControllerSchema,
    createMethod: createFlightController,
    updateMethod: updateFlightController,
    deleteMethod: deleteFlightController,
  },
  {
    collectionId: "frames",
    aliases: ["frame", "components/hardware/frames"],
    paramKey: "frame",
    singularLabel: "Frame",
    schema: FrameSchema,
    createMethod: createFrame,
    updateMethod: updateFrame,
    deleteMethod: deleteFrame,
  },
  {
    collectionId: "goggles",
    aliases: ["goggle", "components/gear/goggles"],
    paramKey: "goggles",
    singularLabel: "Goggles",
    schema: GogglesSchema,
    createMethod: createGoggles,
    updateMethod: updateGoggles,
    deleteMethod: deleteGoggles,
  },
  {
    collectionId: "gps-receivers",
    aliases: [
      "gps_receivers",
      "gps-receiver",
      "gps_receiver",
      "gps",
      "components/hardware/gps-receivers",
    ],
    paramKey: "gpsReceiver",
    singularLabel: "GPS Receiver",
    schema: GpsReceiverSchema,
    createMethod: createGpsReceiver,
    updateMethod: updateGpsReceiver,
    deleteMethod: deleteGpsReceiver,
  },
  {
    collectionId: "motors",
    aliases: ["motor", "components/hardware/motors"],
    paramKey: "motor",
    singularLabel: "Motor",
    schema: MotorSchema,
    createMethod: createMotor,
    updateMethod: updateMotor,
    deleteMethod: deleteMotor,
  },
  {
    collectionId: "propellers",
    aliases: ["propeller", "props", "prop", "components/hardware/propellers"],
    paramKey: "propeller",
    singularLabel: "Propeller",
    schema: PropellerSchema,
    createMethod: createPropeller,
    updateMethod: updatePropeller,
    deleteMethod: deletePropeller,
  },
  {
    collectionId: "radio-firmwares",
    aliases: [
      "radio_firmwares",
      "radio-firmware",
      "radio_firmware",
      "components/software/radio-firmwares",
    ],
    paramKey: "radioFirmware",
    singularLabel: "Radio Firmware",
    schema: RadioFirmwareSchema,
    createMethod: createRadioFirmware,
    updateMethod: updateRadioFirmware,
    deleteMethod: deleteRadioFirmware,
  },
  {
    collectionId: "receivers",
    aliases: ["receiver", "rx", "components/hardware/receivers"],
    paramKey: "receiver",
    singularLabel: "Receiver",
    schema: ReceiverSchema,
    createMethod: createReceiver,
    updateMethod: updateReceiver,
    deleteMethod: deleteReceiver,
  },
  {
    collectionId: "software-tools",
    aliases: [
      "software_tools",
      "software-tool",
      "software_tool",
      "components/software/software-tools",
    ],
    paramKey: "softwareTool",
    singularLabel: "Software Tool",
    schema: SoftwareToolSchema,
    createMethod: createSoftwareTool,
    updateMethod: updateSoftwareTool,
    deleteMethod: deleteSoftwareTool,
  },
  {
    collectionId: "transmitters",
    aliases: ["transmitter", "tx", "radios", "radio", "components/gear/transmitters"],
    paramKey: "transmitter",
    singularLabel: "Transmitter",
    schema: TransmitterSchema,
    createMethod: createTransmitter,
    updateMethod: updateTransmitter,
    deleteMethod: deleteTransmitter,
  },
  {
    collectionId: "video-transmitters",
    aliases: [
      "video_transmitters",
      "video-transmitter",
      "video_transmitter",
      "vtxs",
      "vtx",
      "components/hardware/video-transmitters",
    ],
    paramKey: "videoTransmitter",
    singularLabel: "Video Transmitter",
    schema: VideoTransmitterSchema,
    createMethod: createVideoTransmitter,
    updateMethod: updateVideoTransmitter,
    deleteMethod: deleteVideoTransmitter,
  },
  {
    collectionId: "vtx-firmwares",
    aliases: ["vtx_firmwares", "vtx-firmware", "vtx_firmware", "components/software/vtx-firmwares"],
    paramKey: "vtxFirmware",
    singularLabel: "VTX Firmware",
    schema: VtxFirmwareSchema,
    createMethod: createVtxFirmware,
    updateMethod: updateVtxFirmware,
    deleteMethod: deleteVtxFirmware,
  },
];

export function getComponentServiceDef(collectionId?: string): ComponentServiceDef | undefined {
  if (!collectionId) return undefined;
  const normalized = collectionId
    .toLowerCase()
    .trim()
    .replace(/^\/+|\/+$/g, "");
  const kebab = normalized.replace(/_/g, "-");
  return COMPONENT_SERVICES.find(
    (s) =>
      s.collectionId === normalized ||
      s.collectionId === kebab ||
      s.paramKey.toLowerCase() === normalized ||
      s.paramKey.toLowerCase() === kebab ||
      s.aliases.includes(normalized) ||
      s.aliases.includes(kebab),
  );
}
