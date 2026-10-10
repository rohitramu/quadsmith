import { create } from "@bufbuild/protobuf";
import { MotorSchema, type Motor } from "../../gen/quadsmith/motor_pb";
import { FrameSchema, type Frame } from "../../gen/quadsmith/frame_pb";
import {
  FlightControllerSchema,
  type FlightController,
} from "../../gen/quadsmith/flight_controller_pb";
import { ReferenceLinkType } from "../../gen/quadsmith/reference_link_pb";
import { MediaType } from "../../gen/quadsmith/media_pb";

export const mockMotor1: Motor = create(MotorSchema, {
  uuid: "018f0000-0000-7000-0000-000000000001",
  id: "emax-eco-ii-2207",
  manufacturer: "EMAX",
  name: "ECO II 2207",
  weightG: 33.6,
  statorDiameterMm: 22,
  statorHeightMm: 7,
  kv: 1900,
  description: "Durable and affordable 2207 brushless motor for 5-inch freestyle quadcopters.",
  referenceLinks: [
    {
      types: [ReferenceLinkType.PURCHASE],
      url: "https://store.example.com/emax-eco-ii",
    },
    {
      types: [ReferenceLinkType.PRODUCT_PAGE],
      url: "https://emax-usa.com/products/eco-ii-2207",
    },
  ],
  primaryDisplayImage: "https://emax-usa.com/images/eco-ii-2207.jpg",
  media: [
    {
      type: MediaType.IMAGE,
      url: "https://emax-usa.com/images/eco-ii-2207-diagram.jpg",
      title: "Motor Dimensions & Specs",
    },
    {
      type: MediaType.YOUTUBE,
      url: "https://www.youtube.com/watch?v=dQw4w9WgXcQ",
      title: "EMAX ECO II Thrust Bench Test",
    },
  ],
});

export const mockMotor2: Motor = create(MotorSchema, {
  uuid: "018f0000-0000-7000-0000-000000000002",
  id: "tmotor-f60-pro-v",
  manufacturer: "T-Motor",
  name: "F60 PRO V",
  weightG: 34.1,
  statorDiameterMm: 22,
  statorHeightMm: 7.5,
  kv: 2020,
  description: "High-performance racing and freestyle motor engineered for maximum output.",
  referenceLinks: [],
});

export const mockMotors: Motor[] = [mockMotor1, mockMotor2];

export const mockFrame1: Frame = create(FrameSchema, {
  uuid: "018f0000-0000-7000-0000-000000000010",
  id: "speedybee-master-5-v2",
  manufacturer: "SpeedyBee",
  name: "Master 5 V2",
  wheelbaseMm: 225,
  weightG: 145,
  maxPropSizeMm: 130,
  geometry: "True X",
  description:
    "Freestyle 5-inch frame featuring an innovative anti-vibration stack and CNC camera mount.",
  referenceLinks: [
    {
      types: [ReferenceLinkType.PRODUCT_PAGE],
      url: "https://speedybee.com/master-5-v2",
    },
  ],
});

export const mockFrames: Frame[] = [mockFrame1];

export const mockFC1: FlightController = create(FlightControllerSchema, {
  uuid: "018f0000-0000-7000-0000-000000000020",
  id: "speedybee-f405-v4",
  manufacturer: "SpeedyBee",
  name: "F405 V4 FC",
  processor: "STM32F405",
  weightG: 8.8,
  description:
    "Affordable F4 flight controller with wireless Bluetooth configuration via SpeedyBee App.",
  referenceLinks: [],
});

export const mockFC2: FlightController = create(FlightControllerSchema, {
  uuid: "018f0000-0000-7000-0000-000000000021",
  id: "integrated-fc-aio",
  manufacturer: "BetaFPV",
  name: "Integrated FC AIO",
  processor: "STM32F411",
  weightG: 0,
  isInternalOnly: true,
  internalElectronicSpeedControllerUuid: "018f0000-0000-7000-0000-000000000045",
  internalReceiverUuid: "018f0000-0000-7000-0000-000000000085",
  description: "Internal integrated flight controller board.",
  referenceLinks: [],
});

export const mockFlightControllers: FlightController[] = [mockFC1, mockFC2];

import { BatterySchema, type Battery } from "../../gen/quadsmith/battery_pb";
import {
  ElectronicSpeedControllerSchema,
  type ElectronicSpeedController,
} from "../../gen/quadsmith/electronic_speed_controller_pb";
import { PropellerSchema, type Propeller } from "../../gen/quadsmith/propeller_pb";
import { CameraSchema, type Camera } from "../../gen/quadsmith/camera_pb";
import {
  VideoTransmitterSchema,
  type VideoTransmitter,
} from "../../gen/quadsmith/video_transmitter_pb";
import { ReceiverSchema, type Receiver } from "../../gen/quadsmith/receiver_pb";
import { AntennaSchema, type Antenna } from "../../gen/quadsmith/antenna_pb";
import { GpsReceiverSchema, type GpsReceiver } from "../../gen/quadsmith/gps_receiver_pb";
import { BuildSchema, type Build } from "../../gen/quadsmith/build_pb";
import {
  EvaluateBuildResponseSchema,
  type EvaluateBuildResponse,
} from "../../gen/quadsmith/evaluator_pb";

export const mockBattery1: Battery = create(BatterySchema, {
  uuid: "018f0000-0000-7000-0000-000000000030",
  id: "cnhl-black-series-1500-6s",
  manufacturer: "CNHL",
  name: "Black Series 1500mAh 6S 100C",
  capacityMah: 1500,
  cellCountS: 6,
  weightG: 220,
  chemistry: "LiPo",
  connector: "XT60",
  minVoltage: 19.8,
  maxVoltage: 25.2,
  maxCurrentA: 150.0,
  description: "High discharge 6S battery pack for freestyle drones.",
});

export const mockBatteries: Battery[] = [mockBattery1];

export const mockESC1: ElectronicSpeedController = create(ElectronicSpeedControllerSchema, {
  uuid: "018f0000-0000-7000-0000-000000000040",
  id: "speedybee-50a-4in1",
  manufacturer: "SpeedyBee",
  name: "SpeedyBee 50A 4-in-1 ESC",
  weightG: 14,
  maxMotors: 4,
  motorCurrentMaxA: 50,
  motorCurrentBurstA: 60,
  firmware: "BLHeli_S",
  description: "Durable 50A 4-in-1 BLHeli_S electronic speed controller.",
});

export const mockInternalESC: ElectronicSpeedController = create(ElectronicSpeedControllerSchema, {
  uuid: "018f0000-0000-7000-0000-000000000045",
  id: "betafpv-integrated-20a-esc",
  manufacturer: "BetaFPV",
  name: "BetaFPV Integrated 20A ESC",
  weightG: 0,
  maxMotors: 4,
  motorCurrentMaxA: 20,
  motorCurrentBurstA: 25,
  firmware: "BLHeli_S",
  isInternalOnly: true,
  description: "Internal 20A 4-in-1 BLHeli_S electronic speed controller.",
});

export const mockESCs: ElectronicSpeedController[] = [mockESC1, mockInternalESC];

export const mockPropeller1: Propeller = create(PropellerSchema, {
  uuid: "018f0000-0000-7000-0000-000000000050",
  id: "gemfan-51433-3-blade",
  manufacturer: "Gemfan",
  name: "Hurricane 51433",
  weightG: 3.8,
  diameterMm: 129.5,
  pitchMm: 109.2,
  blades: 3,
  description: "Crisp throttle response and durable polycarbonate construction.",
});

export const mockCamera1: Camera = create(CameraSchema, {
  uuid: "018f0000-0000-7000-0000-000000000060",
  id: "caddx-ratel-2",
  manufacturer: "Caddx",
  name: "Ratel 2",
  weightG: 5.9,
  protocol: "Analog",
  sensorSize: "1/1.8 Inch Starlight",
  description: "Great low light micro FPV analog camera.",
});

export const mockVTX1: VideoTransmitter = create(VideoTransmitterSchema, {
  uuid: "018f0000-0000-7000-0000-000000000070",
  id: "tbs-unify-pro32-nano",
  manufacturer: "TBS",
  name: "Unify Pro32 Nano",
  weightG: 8.7,
  maxPowerMw: 1000,
  protocol: "SmartAudio",
  inputVoltageMinV: 3,
  inputVoltageMaxV: 13,
  description: "Ultra-compact 5.8GHz video transmitter with up to 1W output power.",
});

export const mockReceiver1: Receiver = create(ReceiverSchema, {
  uuid: "018f0000-0000-7000-0000-000000000080",
  id: "tbs-crossfire-nano-rx",
  manufacturer: "TBS",
  name: "Crossfire Nano RX",
  weightG: 0.5,
  protocol: "CRSF",
  frequencyBandMhz: 915,
  hasTelemetry: true,
  description: "Long-range CRSF telemetry receiver.",
});

export const mockInternalRx: Receiver = create(ReceiverSchema, {
  uuid: "018f0000-0000-7000-0000-000000000085",
  id: "betafpv-integrated-elrs-rx",
  manufacturer: "BetaFPV",
  name: "BetaFPV Integrated ELRS 2.4GHz RX",
  protocol: "ExpressLRS",
  frequencyBandMhz: 2400,
  hasTelemetry: true,
  weightG: 0,
  isInternalOnly: true,
  description: "Internal SPI ExpressLRS 2.4GHz receiver.",
});

export const mockReceiverCeramic: Receiver = create(ReceiverSchema, {
  uuid: "018f0000-0000-7000-0000-000000000086",
  id: "radiomaster-rp2-elrs",
  manufacturer: "RadioMaster",
  name: "RadioMaster RP2 2.4GHz Receiver",
  protocol: "ExpressLRS",
  frequencyBandMhz: 2400,
  hasTelemetry: true,
  weightG: 0.6,
  antennaUuids: ["01923019-3009-7001-8001-000000000012"],
  description: "ExpressLRS receiver with onboard SMD ceramic antenna.",
});

export const mockReceivers: Receiver[] = [mockReceiver1, mockInternalRx, mockReceiverCeramic];

export const mockAntenna1: Antenna = create(AntennaSchema, {
  uuid: "018f0000-0000-7000-0000-000000000090",
  id: "foxeer-lollipop-4",
  manufacturer: "Foxeer",
  name: "Lollipop 4 RHCP",
  weightG: 7.3,
  frequencyBandMhz: 5800,
  gainDbi: 2.6,
  polarization: "RHCP",
  connector: "SMA",
  description: "Omnidirectional high-gain circular polarized antenna.",
});

export const mockRxAntenna1: Antenna = create(AntennaSchema, {
  uuid: "018f0000-0000-7000-0000-000000000091",
  id: "radiomaster-t-antenna-2.4ghz",
  manufacturer: "RadioMaster",
  name: "RadioMaster T-Antenna 2.4GHz",
  weightG: 1.5,
  frequencyBandMhz: 2400,
  gainDbi: 2.0,
  polarization: "Linear",
  connector: "U.FL",
  description: "Flexible T-style dipole receiver antenna for 2.4GHz receivers.",
});

export const mockAntennas: Antenna[] = [mockAntenna1, mockRxAntenna1];

export const mockGps1: GpsReceiver = create(GpsReceiverSchema, {
  uuid: "018f0000-0000-7000-0000-000000000095",
  id: "matek-m8q-5883",
  manufacturer: "Matek",
  name: "M8Q-5883 GPS & Compass",
  weightG: 11.3,
  protocol: "UBLOX",
  compass: "QMC5883L",
  description: "Compact GNSS module with QMC5883L digital compass.",
});

export const mockBuild1: Build = create(BuildSchema, {
  uuid: "01923019-2131-4192-3192-391294812399",
  id: "bando-basher-5-inch",
  name: "Bando Basher 5 inch",
  description:
    "A durable 5-inch freestyle quadcopter built to withstand concrete hits in abandoned buildings, featuring the TBS Source One V5 frame, T-Motor F60 PRO V 1950KV motors, SpeedyBee F405 V4 stack, Caddx Ratel 2 camera, and CNHL 1500mAh 6S LiPo.",
  frameUuid: mockFrame1.uuid,
  motorUuid: mockMotor1.uuid,
  flightControllerUuid: mockFC1.uuid,
  electronicSpeedControllerUuids: [mockESC1.uuid],
  propellerUuid: mockPropeller1.uuid,
  cameraUuids: [mockCamera1.uuid],
  videoTransmitterUuid: mockVTX1.uuid,
  receiverUuids: [mockReceiver1.uuid],
  antennaUuids: [mockAntenna1.uuid],
  gpsReceiverUuid: mockGps1.uuid,
  referenceLinks: [
    {
      types: [ReferenceLinkType.DOCUMENTATION],
      url: "https://github.com/tbs-trappy/source_one",
    },
    {
      types: [ReferenceLinkType.PURCHASE],
      url: "https://www.getfpv.com/tbs-source-one-v5-5-frame-kit.html",
    },
  ],
  primaryDisplayImage:
    "https://images.unsplash.com/photo-1527977966376-1c8408f9f108?auto=format&fit=crop&w=1200&q=80",
  media: [
    {
      type: MediaType.IMAGE,
      url: "https://images.unsplash.com/photo-1527977966376-1c8408f9f108?auto=format&fit=crop&w=1200&q=80",
      title: "Bando Basher 5 inch Freestyle Frame",
    },
    {
      type: MediaType.YOUTUBE,
      url: "https://www.youtube.com/watch?v=dQw4w9WgXcQ",
      title: "Bando Basher Flight & Durability Test",
    },
  ],
});

export const mockBuild2: Build = create(BuildSchema, {
  uuid: "01923019-2131-4192-3192-391294812398",
  id: "long-range-explorer",
  name: "Long Range Explorer 7 inch",
  description:
    "A dedicated 7-inch mountain surfer and long-range cruiser engineered for 30+ minute endurance flights with Li-ion pack.",
  frameUuid: mockFrame1.uuid,
  motorUuid: mockMotor2.uuid,
  flightControllerUuid: mockFC1.uuid,
  electronicSpeedControllerUuids: [mockESC1.uuid],
  propellerUuid: mockPropeller1.uuid,
  cameraUuids: [mockCamera1.uuid],
  videoTransmitterUuid: mockVTX1.uuid,
  receiverUuids: [mockReceiver1.uuid],
  antennaUuids: [mockAntenna1.uuid],
  referenceLinks: [],
});

export const mockBuilds: Build[] = [mockBuild1, mockBuild2];

export const mockEvaluation1: EvaluateBuildResponse = create(EvaluateBuildResponseSchema, {
  allUpWeightG: 565.5,
  thrustToWeightRatio: 6.4,
  hoverThrottlePercent: 39.5,
  minFlightTimeMin: 3.8,
  maxFlightTimeMin: 7.2,
  maxAccelerationMps2: 53.4,
  topSpeedKmh: 172.5,
  hoverRpm: 11463,
  systemMessages: [],
  buildId: "bando-basher-5-inch",
  payloadWeightG: 0,
  batteryId: "tattu-rline-v5-1400mah-6s",
  minVoltage: 14.8,
  maxVoltage: 25.2,
  maxCurrentA: 39.4,
});
