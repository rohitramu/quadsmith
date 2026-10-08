import { create } from "@bufbuild/protobuf";
import { MotorSchema, type Motor } from "../../gen/quadsmith/motor_pb";
import { FrameSchema, type Frame } from "../../gen/quadsmith/frame_pb";
import { FlightControllerSchema, type FlightController } from "../../gen/quadsmith/flight_controller_pb";
import { ReferenceLinkType } from "../../gen/quadsmith/reference_link_pb";

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
      type: ReferenceLinkType.PURCHASE,
      url: "https://store.example.com/emax-eco-ii",
    },
    {
      type: ReferenceLinkType.PRODUCT_PAGE,
      url: "https://emax-usa.com/products/eco-ii-2207",
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
  description: "Freestyle 5-inch frame featuring an innovative anti-vibration stack and CNC camera mount.",
  referenceLinks: [
    {
      type: ReferenceLinkType.PRODUCT_PAGE,
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
  description: "Affordable F4 flight controller with wireless Bluetooth configuration via SpeedyBee App.",
  referenceLinks: [],
});

export const mockFlightControllers: FlightController[] = [mockFC1];
