import { createRouterTransport } from "@connectrpc/connect";
import { ConnectError, Code } from "@connectrpc/connect";
import { MotorService } from "../../gen/quadsmith/motor_pb";
import { FrameService } from "../../gen/quadsmith/frame_pb";
import { FlightControllerService } from "../../gen/quadsmith/flight_controller_pb";
import { BatteryService } from "../../gen/quadsmith/battery_pb";
import { CameraService } from "../../gen/quadsmith/camera_pb";
import { ElectronicSpeedControllerService } from "../../gen/quadsmith/electronic_speed_controller_pb";
import { PropellerService } from "../../gen/quadsmith/propeller_pb";
import { ReceiverService } from "../../gen/quadsmith/receiver_pb";
import { VideoTransmitterService } from "../../gen/quadsmith/video_transmitter_pb";
import { AntennaService } from "../../gen/quadsmith/antenna_pb";
import { GpsReceiverService } from "../../gen/quadsmith/gps_receiver_pb";
import { BuildService } from "../../gen/quadsmith/build_pb";
import { EvaluatorService } from "../../gen/quadsmith/evaluator_pb";

import {
  mockMotors,
  mockFrames,
  mockFlightControllers,
  mockBattery1,
  mockESC1,
  mockPropeller1,
  mockCamera1,
  mockVTX1,
  mockReceiver1,
  mockAntenna1,
  mockGps1,
  mockBuilds,
  mockEvaluation1,
} from "./fixtures";

export interface MockTransportOptions {
  motors?: typeof mockMotors;
  frames?: typeof mockFrames;
  flightControllers?: typeof mockFlightControllers;
  builds?: typeof mockBuilds;
  nextPageToken?: string;
  simulateError?: boolean;
}

export function createMockTransport(options: MockTransportOptions = {}) {
  const {
    motors = mockMotors,
    frames = mockFrames,
    flightControllers = mockFlightControllers,
    builds = mockBuilds,
    nextPageToken = "",
    simulateError = false,
  } = options;

  return createRouterTransport(({ service }) => {
    service(MotorService, {
      listMotors(req) {
        if (simulateError) {
          throw new ConnectError("Failed to fetch motors from server", Code.Internal);
        }
        let list = [...motors];
        if (req.filter) {
          const lower = req.filter.toLowerCase();
          list = list.filter(
            (m) =>
              m.name.toLowerCase().includes(lower) ||
              m.manufacturer.toLowerCase().includes(lower) ||
              m.id.toLowerCase().includes(lower),
          );
        }
        if (req.sort && req.sort.length > 0) {
          const sortRule = req.sort[0];
          const isDesc = sortRule.startsWith("^");
          const field = isDesc ? sortRule.slice(1) : sortRule;
          list.sort((a: any, b: any) => {
            const valA = a[field] ?? "";
            const valB = b[field] ?? "";
            if (valA < valB) return isDesc ? 1 : -1;
            if (valA > valB) return isDesc ? -1 : 1;
            return 0;
          });
        }
        return {
          motors: list,
          nextPageToken: req.pageToken ? "" : nextPageToken,
        };
      },
      getMotor(req) {
        if (simulateError) {
          throw new ConnectError("Failed to fetch motor details", Code.Internal);
        }
        const item = motors.find((m) => m.id === req.id || m.uuid === req.id);
        if (!item) {
          throw new ConnectError(`Motor with ID '${req.id}' not found`, Code.NotFound);
        }
        return item;
      },
    });

    service(FrameService, {
      listFrames() {
        if (simulateError) {
          throw new ConnectError("Failed to fetch frames", Code.Internal);
        }
        return { frames, nextPageToken: "" };
      },
      getFrame(req) {
        const item = frames.find((f) => f.id === req.id || f.uuid === req.id);
        if (!item) {
          throw new ConnectError(`Frame with ID '${req.id}' not found`, Code.NotFound);
        }
        return item;
      },
    });

    service(FlightControllerService, {
      listFlightControllers() {
        return { flightControllers, nextPageToken: "" };
      },
      getFlightController(req) {
        const item = flightControllers.find((fc) => fc.id === req.id || fc.uuid === req.id);
        if (!item) {
          throw new ConnectError(`Flight Controller with ID '${req.id}' not found`, Code.NotFound);
        }
        return item;
      },
    });

    service(BuildService, {
      listBuilds(req) {
        if (simulateError) {
          throw new ConnectError("Failed to fetch builds", Code.Internal);
        }
        let list = [...builds];
        if (req.filter) {
          const lower = req.filter.toLowerCase();
          list = list.filter(
            (b) =>
              b.name.toLowerCase().includes(lower) ||
              b.id.toLowerCase().includes(lower) ||
              b.description.toLowerCase().includes(lower),
          );
        }
        const pageSize = req.pageSize || 20;
        let pToken = "";
        let returnList = list;
        if (req.pageToken) {
          // simple offset simulation
          const offset = parseInt(req.pageToken, 10) || 0;
          returnList = list.slice(offset, offset + pageSize);
          if (offset + pageSize < list.length) {
            pToken = String(offset + pageSize);
          }
        } else {
          returnList = list.slice(0, pageSize);
          if (list.length > pageSize) {
            pToken = String(pageSize);
          } else if (nextPageToken) {
            pToken = nextPageToken;
          }
        }
        return {
          builds: returnList,
          nextPageToken: pToken,
        };
      },
      getBuild(req) {
        if (simulateError) {
          throw new ConnectError("Failed to fetch build", Code.Internal);
        }
        const item = builds.find((b) => b.id === req.id || b.uuid === req.id);
        if (!item) {
          throw new ConnectError(`Build with ID '${req.id}' not found`, Code.NotFound);
        }
        return item;
      },
    });

    service(EvaluatorService, {
      evaluateBuild(req) {
        if (simulateError) {
          throw new ConnectError("Evaluation failed", Code.Internal);
        }
        const payload = req.payloadWeightG || 0;
        const baseWeight = mockEvaluation1.totalWeightG;
        const safeBase = Math.max(1, baseWeight);
        const totalWeight = baseWeight + payload;
        const payloadRatio = Math.min(1.2, payload / safeBase);
        const baseThrust = 3645; // Realistic 5" 6S installed static thrust (4x ~911g)
        const payloadObs = 1.0 / (1.0 + 0.08 * payloadRatio);
        const totalThrust = baseThrust * payloadObs;
        const twr = totalWeight > 0 ? totalThrust / totalWeight : 0;
        const twrFactor = Math.min(1.0, Math.max(0.0, (twr - 1.0) / 5.0));
        const gamma = 0.68 + 0.1 * twrFactor;
        const hover = twr > 0 ? Math.pow(1.0 / twr, gamma) * 100 : 100;
        const weightRatio = totalWeight / safeBase;
        const warnings: string[] = [];
        const errors: string[] = [];
        if (hover > 50) {
          warnings.push("Drone will be very sluggish (Hover throttle > 50%)");
        }
        if (hover > 100) {
          errors.push("Drone is too heavy to take off (Hover throttle > 100%)");
        }
        const maxFlightTime = parseFloat(Math.max(1, 7.2 / Math.pow(weightRatio, 1.35)).toFixed(1));
        const minFlightTime = parseFloat((maxFlightTime / 1.9).toFixed(1));
        const flightTime = parseFloat(((minFlightTime + maxFlightTime) / 2).toFixed(1));

        const maxAccelerationMps2 = parseFloat(Math.max(0, (twr - 1.0) * 9.80665).toFixed(1));
        const fwdThrustN = (totalThrust * 0.95) / 101.97162;
        const cdA = 0.0095 + 0.006 * payloadRatio;
        const pitchSpeed = 61.5;
        const denom = 0.5 * 1.225 * cdA + fwdThrustN / (pitchSpeed * pitchSpeed);
        const topSpeedKmh = parseFloat(
          (Math.sqrt(Math.max(0, fwdThrustN / denom)) * 3.6).toFixed(1),
        );

        return {
          totalWeightG: totalWeight,
          thrustToWeightRatio: parseFloat(twr.toFixed(2)),
          hoverThrottlePercent: parseFloat(hover.toFixed(1)),
          estimatedFlightTimeMin: flightTime,
          minFlightTimeMin: minFlightTime,
          maxFlightTimeMin: maxFlightTime,
          maxAccelerationMps2,
          topSpeedKmh,
          warnings,
          errors,
        };
      },
    });

    service(BatteryService, {
      listBatteries: () => ({ batteries: [mockBattery1], nextPageToken: "" }),
      getBattery: (req) => {
        if (req.id === mockBattery1.id || req.id === mockBattery1.uuid) {
          return mockBattery1;
        }
        throw new ConnectError("Battery not found", Code.NotFound);
      },
    });

    service(CameraService, {
      listCameras: () => ({ cameras: [mockCamera1], nextPageToken: "" }),
      getCamera: (req) => {
        if (req.id === mockCamera1.id || req.id === mockCamera1.uuid) {
          return mockCamera1;
        }
        throw new ConnectError("Camera not found", Code.NotFound);
      },
    });

    service(ElectronicSpeedControllerService, {
      listElectronicSpeedControllers: () => ({
        electronicSpeedControllers: [mockESC1],
        nextPageToken: "",
      }),
      getElectronicSpeedController: (req) => {
        if (req.id === mockESC1.id || req.id === mockESC1.uuid) {
          return mockESC1;
        }
        throw new ConnectError("ESC not found", Code.NotFound);
      },
    });

    service(PropellerService, {
      listPropellers: () => ({ propellers: [mockPropeller1], nextPageToken: "" }),
      getPropeller: (req) => {
        if (req.id === mockPropeller1.id || req.id === mockPropeller1.uuid) {
          return mockPropeller1;
        }
        throw new ConnectError("Propeller not found", Code.NotFound);
      },
    });

    service(ReceiverService, {
      listReceivers: () => ({ receivers: [mockReceiver1], nextPageToken: "" }),
      getReceiver: (req) => {
        if (req.id === mockReceiver1.id || req.id === mockReceiver1.uuid) {
          return mockReceiver1;
        }
        throw new ConnectError("Receiver not found", Code.NotFound);
      },
    });

    service(VideoTransmitterService, {
      listVideoTransmitters: () => ({
        videoTransmitters: [mockVTX1],
        nextPageToken: "",
      }),
      getVideoTransmitter: (req) => {
        if (req.id === mockVTX1.id || req.id === mockVTX1.uuid) {
          return mockVTX1;
        }
        throw new ConnectError("VTX not found", Code.NotFound);
      },
    });

    service(AntennaService, {
      listAntennas: () => ({ antennas: [mockAntenna1], nextPageToken: "" }),
      getAntenna: (req) => {
        if (req.id === mockAntenna1.id || req.id === mockAntenna1.uuid) {
          return mockAntenna1;
        }
        throw new ConnectError("Antenna not found", Code.NotFound);
      },
    });

    service(GpsReceiverService, {
      listGpsReceivers: () => ({ gpsReceivers: [mockGps1], nextPageToken: "" }),
      getGpsReceiver: (req) => {
        if (req.id === mockGps1.id || req.id === mockGps1.uuid) {
          return mockGps1;
        }
        throw new ConnectError("GPS receiver not found", Code.NotFound);
      },
    });
  });
}
