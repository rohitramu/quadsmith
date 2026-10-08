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

import {
  mockMotors,
  mockFrames,
  mockFlightControllers,
} from "./fixtures";

export interface MockTransportOptions {
  motors?: typeof mockMotors;
  frames?: typeof mockFrames;
  flightControllers?: typeof mockFlightControllers;
  nextPageToken?: string;
  simulateError?: boolean;
}

export function createMockTransport(options: MockTransportOptions = {}) {
  const {
    motors = mockMotors,
    frames = mockFrames,
    flightControllers = mockFlightControllers,
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
              m.id.toLowerCase().includes(lower)
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

    service(BatteryService, {
      listBatteries: () => ({ batteries: [], nextPageToken: "" }),
      getBattery: () => {
        throw new ConnectError("Battery not found", Code.NotFound);
      },
    });

    service(CameraService, {
      listCameras: () => ({ cameras: [], nextPageToken: "" }),
      getCamera: () => {
        throw new ConnectError("Camera not found", Code.NotFound);
      },
    });

    service(ElectronicSpeedControllerService, {
      listElectronicSpeedControllers: () => ({ electronicSpeedControllers: [], nextPageToken: "" }),
      getElectronicSpeedController: () => {
        throw new ConnectError("ESC not found", Code.NotFound);
      },
    });

    service(PropellerService, {
      listPropellers: () => ({ propellers: [], nextPageToken: "" }),
      getPropeller: () => {
        throw new ConnectError("Propeller not found", Code.NotFound);
      },
    });

    service(ReceiverService, {
      listReceivers: () => ({ receivers: [], nextPageToken: "" }),
      getReceiver: () => {
        throw new ConnectError("Receiver not found", Code.NotFound);
      },
    });

    service(VideoTransmitterService, {
      listVideoTransmitters: () => ({ videoTransmitters: [], nextPageToken: "" }),
      getVideoTransmitter: () => {
        throw new ConnectError("VTX not found", Code.NotFound);
      },
    });

    service(AntennaService, {
      listAntennas: () => ({ antennas: [], nextPageToken: "" }),
      getAntenna: () => {
        throw new ConnectError("Antenna not found", Code.NotFound);
      },
    });

    service(GpsReceiverService, {
      listGpsReceivers: () => ({ gpsReceivers: [], nextPageToken: "" }),
      getGpsReceiver: () => {
        throw new ConnectError("GPS receiver not found", Code.NotFound);
      },
    });
  });
}
