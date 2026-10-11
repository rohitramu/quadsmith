import { createRouterTransport } from "@connectrpc/connect";
import { ConnectError, Code } from "@connectrpc/connect";
import { create } from "@bufbuild/protobuf";
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
import { BuildService, type Build } from "../../gen/quadsmith/build_pb";
import { EvaluatorService } from "../../gen/quadsmith/evaluator_pb";
import { CompatibilityService } from "../../gen/quadsmith/compatibility_pb";
import { LinkPreviewService } from "../../gen/quadsmith/link_preview_pb";
import { SearchService } from "../../gen/quadsmith/search_pb";
import {
  AuthService,
  SendEmailOtpResponseSchema,
  ArchivedUsernameSchema,
  OtpVerificationErrorSchema,
} from "../../gen/quadsmith/auth_pb";
import { UserService, UserEmailSchema } from "../../gen/quadsmith/user_pb";

import {
  mockMotors,
  mockFrames,
  mockFlightControllers,
  mockBattery1,
  mockESCs,
  mockUser,
  mockProfile,
  mockPropeller1,
  mockCamera1,
  mockVTX1,
  mockReceivers,
  mockAntennas,
  mockGps1,
  mockBuilds,
  mockEvaluation1,
  mockBatteries,
  mockUserEmails,
} from "./fixtures";

export interface MockTransportOptions {
  motors?: typeof mockMotors;
  frames?: typeof mockFrames;
  flightControllers?: typeof mockFlightControllers;
  builds?: typeof mockBuilds;
  batteries?: typeof mockBatteries;
  userEmails?: typeof mockUserEmails;
  defaultBatteryId?: string;
  nextPageToken?: string;
  simulateError?: boolean;
  isAuthenticated?: boolean;
}

export function createMockTransport(options: MockTransportOptions = {}) {
  const {
    motors = mockMotors,
    frames = mockFrames,
    flightControllers = mockFlightControllers,
    builds = [...(options.builds ?? mockBuilds)],
    batteries = mockBatteries,
    defaultBatteryId = batteries.length > 0 ? batteries[0].id : "",
    nextPageToken = "",
    simulateError = false,
    isAuthenticated = true,
  } = options;
  const userEmails = (options.userEmails ?? mockUserEmails).map((e) => create(UserEmailSchema, e));

  return createRouterTransport(({ service }) => {
    service(MotorService, {
      listMotors(req) {
        if (simulateError) {
          throw new ConnectError("Failed to fetch motors from server", Code.Internal);
        }
        let list = [...motors];
        if (
          req.filter &&
          !req.filter.includes("<=") &&
          !req.filter.includes(">=") &&
          !req.filter.includes("==")
        ) {
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
      createBuild(req) {
        if (simulateError) {
          throw new ConnectError("Failed to create build", Code.Internal);
        }
        if (!req.build || !req.build.name) {
          throw new ConnectError("Build name is required", Code.InvalidArgument);
        }
        const b = req.build;
        let id = b.id || b.name.toLowerCase().replace(/[^a-z0-9]+/g, "-");
        id = id.replace(/^-+|-+$/g, "");
        if (!id) {
          id = `build-${(b.uuid || "").substring(0, 8) || "custom"}`;
        }
        if (builds.some((existing) => existing.id === id)) {
          const copyRegex = /^(.*?)-copy(\d*)$/i;
          const match = id.match(copyRegex);
          let base = id;
          let nextNum = 1;
          if (match) {
            if (match[1]) {
              base = match[1];
            } else {
              base = "build";
            }
            if (match[2]) {
              nextNum = parseInt(match[2], 10) + 1;
            }
          }
          const existingIds = new Set(builds.map((existing) => existing.id));
          while (existingIds.has(`${base}-copy${nextNum}`)) {
            nextNum++;
          }
          id = `${base}-copy${nextNum}`;
        }
        const newBuild = {
          ...b,
          id,
          uuid:
            b.uuid ||
            `01912345-${Math.random().toString(16).substring(2, 6)}-7000-8000-${Math.random().toString(16).substring(2, 14)}`,
        } as unknown as Build;
        builds.push(newBuild);
        return newBuild;
      },
    });

    service(EvaluatorService, {
      evaluateBuild(req) {
        if (simulateError) {
          throw new ConnectError("Evaluation failed", Code.Internal);
        }
        if (!req.batteryId) {
          throw new ConnectError("battery_id is required", Code.InvalidArgument);
        }
        const payload = req.payloadWeightG || 0;
        const baseWeight = mockEvaluation1.allUpWeightG;
        const safeBase = Math.max(1, baseWeight);
        const totalWeight = baseWeight + payload;
        const payloadRatio = Math.min(1.2, payload / safeBase);
        const baseThrust = 3645; // Realistic 5" 6S installed static thrust (4x ~911g)
        const payloadObs = 1.0 / (1.0 + 0.08 * payloadRatio);
        const totalThrust = baseThrust * payloadObs;
        const twr = totalWeight > 0 ? totalThrust / totalWeight : 0;
        const invTwr = twr > 0 ? 1.0 / twr : 1.0;
        const sqrtInvTwr = Math.sqrt(invTwr);
        const rpmLoadFactor = 0.72;
        const nominalCellV = 3.85;
        const burstCellVoltage = 3.55 - 0.12 * payloadRatio;
        const hoverCellVoltage =
          nominalCellV - (nominalCellV - burstCellVoltage) * Math.min(1.0, invTwr);
        const voltageRatio = burstCellVoltage / hoverCellVoltage;
        const dutyCycle =
          voltageRatio * (rpmLoadFactor * sqrtInvTwr + (1.0 - rpmLoadFactor) * invTwr);
        const hover = twr > 0 ? dutyCycle * 100 : 100;
        const weightRatio = totalWeight / safeBase;
        const systemMessages: any[] = [];
        if (hover > 50) {
          systemMessages.push({
            severity: 2, // WARNING
            message: "Drone will be very sluggish (Hover throttle > 50%)",
          });
        }
        if (hover > 100) {
          systemMessages.push({
            severity: 3, // ERROR
            message: "Drone is too heavy to take off (Hover throttle > 100%)",
          });
        }
        const maxFlightTime = parseFloat(Math.max(1, 7.2 / Math.pow(weightRatio, 1.35)).toFixed(1));
        const minFlightTime = parseFloat((maxFlightTime / 1.9).toFixed(1));

        const maxAccelerationMps2 = parseFloat(Math.max(0, (twr - 1.0) * 9.80665).toFixed(1));
        const fwdThrustN = (totalThrust * 0.95) / 101.97162;
        const cdA = 0.0095 + 0.006 * payloadRatio;
        const pitchSpeed = 61.5;
        const denom = 0.5 * 1.225 * cdA + fwdThrustN / (pitchSpeed * pitchSpeed);
        const topSpeedKmh = parseFloat(
          (Math.sqrt(Math.max(0, fwdThrustN / denom)) * 3.6).toFixed(1),
        );
        const hoverRpm = twr >= 1.0 ? Math.round(29000 / Math.sqrt(twr)) : 0;
        const bId =
          req.buildSource?.case === "buildId"
            ? req.buildSource.value
            : req.buildSource?.case === "build"
              ? req.buildSource.value.id
              : "";

        return {
          allUpWeightG: totalWeight,
          thrustToWeightRatio: parseFloat(twr.toFixed(2)),
          hoverThrottlePercent: parseFloat(hover.toFixed(1)),
          hoverRpm,
          minFlightTimeMin: minFlightTime,
          maxFlightTimeMin: maxFlightTime,
          maxAccelerationMps2,
          topSpeedKmh,
          systemMessages,
          buildId: bId,
          payloadWeightG: payload,
          batteryId: req.batteryId || mockBattery1.id,
          minVoltage: 14.8,
          maxVoltage: 25.2,
          maxCurrentA: 39.4,
        };
      },
      getBuildElectricalLimits(req) {
        if (simulateError) {
          throw new ConnectError("Failed to fetch electrical limits", Code.Internal);
        }
        const bId =
          req.buildSource?.case === "buildId"
            ? req.buildSource.value
            : req.buildSource?.case === "build"
              ? req.buildSource.value.id
              : "";
        return {
          minVoltage: 14.8,
          maxVoltage: 25.2,
          maxCurrentA: 39.4,
          defaultBatteryId:
            options.defaultBatteryId !== undefined ? options.defaultBatteryId : defaultBatteryId,
          buildId: bId,
        };
      },
    });

    service(CompatibilityService, {
      checkCompatibility(_req) {
        return {
          messages: [],
        };
      },
    });

    service(BatteryService, {
      listBatteries: () => ({ batteries, nextPageToken: "" }),
      getBattery: (req) => {
        const found = batteries.find((b) => b.id === req.id || b.uuid === req.id);
        if (found) {
          return found;
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
        electronicSpeedControllers: mockESCs,
        nextPageToken: "",
      }),
      getElectronicSpeedController: (req) => {
        const esc = mockESCs.find((e) => e.id === req.id || e.uuid === req.id);
        if (esc) {
          return esc;
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
      listReceivers: () => ({ receivers: mockReceivers, nextPageToken: "" }),
      getReceiver: (req) => {
        const rx = mockReceivers.find((r) => r.id === req.id || r.uuid === req.id);
        if (rx) {
          return rx;
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
      listAntennas: () => ({ antennas: mockAntennas, nextPageToken: "" }),
      getAntenna: (req) => {
        const item = mockAntennas.find((a) => a.id === req.id || a.uuid === req.id);
        if (item) {
          return item;
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

    service(LinkPreviewService, {
      getLinkPreview: (req) => {
        return {
          url: req.url,
          title: `Preview for ${req.url}`,
          description: "A simulated mock link preview description.",
          image: "https://quadsmith.net/mock-preview.jpg",
          siteName: "example.com",
          favicon: "https://example.com/favicon.ico",
        };
      },
    });

    service(SearchService, {
      search: (req) => {
        const q = (req.query || "").toLowerCase();
        const results = batteries
          .filter(
            (b) =>
              !q ||
              b.name.toLowerCase().includes(q) ||
              b.id.toLowerCase().includes(q) ||
              (b.manufacturer && b.manufacturer.toLowerCase().includes(q)),
          )
          .map((b) => ({
            id: b.id,
            uuid: b.uuid,
            name: b.name,
            path: "components/hardware/batteries",
            collectionName: "Batteries",
            description: b.description || "",
            matchScore: 1.0,
            metadata: {},
          }));
        return { results };
      },
    });

    service(AuthService, {
      sendEmailOtp: () => ({ success: true, message: "Code sent", cooldownSeconds: 20 }),
      verifyEmailOtp: (req) => {
        if (req.code === "888888") {
          const err = new ConnectError(
            "Incorrect code. You have 2 attempts left",
            Code.InvalidArgument,
            undefined,
            [
              {
                desc: OtpVerificationErrorSchema,
                value: {
                  remainingAttempts: 2,
                  isExhausted: false,
                  cooldownSeconds: 0,
                  message: "Incorrect code. You have 2 attempts left",
                },
              },
            ],
          );
          throw err;
        }
        if (req.code === "999999") {
          const err = new ConnectError(
            "Incorrect code. You have 0 attempts left. Please request a new code",
            Code.InvalidArgument,
            undefined,
            [
              {
                desc: OtpVerificationErrorSchema,
                value: {
                  remainingAttempts: 0,
                  isExhausted: true,
                  cooldownSeconds: 60,
                  message: "Incorrect code. You have 0 attempts left. Please request a new code",
                },
              },
            ],
          );
          throw err;
        }
        if (req.code === "000000") {
          throw new ConnectError(
            "Too many failed verification attempts from this IP. Please wait 60 minutes before trying again",
            Code.ResourceExhausted,
          );
        }
        return {
          accessToken: "mock-jwt-token",
          refreshToken: "mock-refresh-token",
          user: mockUser,
          profile: mockProfile,
        };
      },
      verifyMagicLink: () => ({
        accessToken: "mock-jwt-token",
        refreshToken: "mock-refresh-token",
        user: mockUser,
        profile: mockProfile,
      }),
      refreshToken: () => {
        if (!isAuthenticated) {
          throw new ConnectError("Unauthenticated", Code.Unauthenticated);
        }
        return {
          accessToken: "mock-jwt-token",
          refreshToken: "mock-refresh-token",
          user: mockUser,
          profile: mockProfile,
        };
      },
      getCurrentUser: () => {
        if (!isAuthenticated) {
          throw new ConnectError("Unauthenticated", Code.Unauthenticated);
        }
        return mockUser;
      },
      updateProfile: (req) => ({
        ...mockProfile,
        displayName: req.displayName ?? mockProfile.displayName,
        avatarUrl: req.avatarUrl ?? mockProfile.avatarUrl,
        bio: req.bio ?? mockProfile.bio,
      }),
      logout: () => ({ success: true }),
      listUserEmails: () => ({
        emails: userEmails,
      }),
      sendAddEmailOtp: (req) => {
        const isExisting =
          req.email.toLowerCase().includes("other") ||
          req.email.toLowerCase().includes("old.racer") ||
          req.email.toLowerCase().includes("existing");
        return create(SendEmailOtpResponseSchema, {
          success: true,
          message: `Verification code sent to ${req.email}`,
          cooldownSeconds: 20,
          isExistingAccount: isExisting,
        });
      },
      verifyAddEmailOtp: (req) => {
        if (req.code !== "123456" && !req.token) {
          throw new ConnectError("Incorrect verification code", Code.InvalidArgument);
        }
        const emailLower = (req.email || "").toLowerCase();
        const isExisting =
          emailLower.includes("other") ||
          emailLower.includes("old.racer") ||
          emailLower.includes("existing") ||
          Boolean(req.token && !req.email);

        if (isExisting) {
          return {
            success: true,
            message: "Account merge required",
            mergeRequired: true,
            otherUserUsername: "old_racer",
            otherBuildCount: 4,
            otherBuildNames: [
              "Bando Basher 5-inch",
              "CineWhoop 3.5 Pro",
              "Long Range Cruiser 7-inch",
              "Ultralight Toothpick 2S",
            ],
            otherEmails: ["old.racer@fpv.org", "old.racer.backup@example.com"],
            otherOauthProviders: ["github"],
          };
        }

        const newEmail = create(UserEmailSchema, {
          uuid: `mock-email-uuid-${userEmails.length + 1}`,
          userUuid: mockUser.uuid,
          email: req.email || "pilot@example.com",
          isPrimary: false,
        });
        userEmails.push(newEmail);
        return {
          success: true,
          message: "Email verified and added",
          mergeRequired: false,
          otherUserUsername: "",
          otherBuildCount: 0,
          otherBuildNames: [],
          otherEmails: [],
          otherOauthProviders: [],
        };
      },
      mergeAccount: (req) => {
        if (req.code !== "123456" && !req.token) {
          throw new ConnectError("Incorrect verification code", Code.InvalidArgument);
        }
        const mergedEmail = create(UserEmailSchema, {
          uuid: `mock-email-uuid-${userEmails.length + 1}`,
          userUuid: mockUser.uuid,
          email: req.email || "old.racer@fpv.org",
          isPrimary: false,
        });
        userEmails.push(mergedEmail);
        return {
          success: true,
          message: "Accounts merged successfully",
          transferredBuildCount: 4,
          mergedUsername: "old_racer",
        };
      },
      setPrimaryEmail: (req) => {
        userEmails.forEach((e) => {
          e.isPrimary = e.email.toLowerCase() === req.email.toLowerCase();
        });
        return {
          success: true,
          message: "Primary email updated",
        };
      },
      removeUserEmail: (req) => {
        const idx = userEmails.findIndex((e) => e.email.toLowerCase() === req.email.toLowerCase());
        if (idx !== -1) {
          userEmails.splice(idx, 1);
        }
        return {
          success: true,
          message: "Email removed",
        };
      },
      listArchivedUsernames: () => {
        return {
          archivedUsernames: [
            create(ArchivedUsernameSchema, {
              username: "old_racer",
              daysRemaining: 23,
              canReclaimNow: true,
              firstArchivedAt: {
                seconds: BigInt(Math.floor((Date.now() - 7 * 24 * 3600 * 1000) / 1000)),
                nanos: 0,
              },
              renamedAt: {
                seconds: BigInt(Math.floor((Date.now() - 7 * 24 * 3600 * 1000) / 1000)),
                nanos: 0,
              },
              releaseAt: {
                seconds: BigInt(Math.floor((Date.now() + 23 * 24 * 3600 * 1000) / 1000)),
                nanos: 0,
              },
            }),
          ],
          nextAllowedNewRenameAt: undefined,
          nextAllowedArchiveRenameAt: undefined,
        };
      },
      changeUsername: (req) => {
        if (!req.newUsername || req.newUsername === "taken") {
          throw new ConnectError("Username is already taken", Code.AlreadyExists);
        }
        mockUser.id = req.newUsername;
        return {
          success: true,
          message: `Your username has been changed to @${req.newUsername}`,
          user: mockUser,
          nextAllowedNewRenameAt: undefined,
          nextAllowedArchiveRenameAt: undefined,
        };
      },
    });

    service(UserService, {
      getUser: () => mockUser,
      getPublicProfile: (req) => ({
        id: req.id || mockUser.id,
        displayName: mockProfile.displayName,
        avatarUrl: mockProfile.avatarUrl,
        bio: mockProfile.bio,
        createdAt: mockUser.createdAt,
        buildCount: 3,
      }),
      checkUsernameAvailable: (req) => ({
        available: req.username !== "taken",
        message: req.username === "taken" ? "Username taken" : "Username available",
      }),
    });
  });
}
