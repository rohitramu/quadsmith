import {
  createContext,
  useContext,
  useState,
  useMemo,
  useEffect,
  useCallback,
  useRef,
} from "react";
import { useQuery, useMutation, useTransport } from "@connectrpc/connect-query";
import { createClient } from "@connectrpc/connect";
import { create } from "@bufbuild/protobuf";
import { useNavigate } from "react-router-dom";

// ConnectQuery hooks for component listing
import { listFrames } from "../../gen/quadsmith/frame-FrameService_connectquery";
import { listMotors } from "../../gen/quadsmith/motor-MotorService_connectquery";
import { listPropellers } from "../../gen/quadsmith/propeller-PropellerService_connectquery";
import { listFlightControllers } from "../../gen/quadsmith/flight_controller-FlightControllerService_connectquery";
import { listElectronicSpeedControllers } from "../../gen/quadsmith/electronic_speed_controller-ElectronicSpeedControllerService_connectquery";
import { listReceivers } from "../../gen/quadsmith/receiver-ReceiverService_connectquery";
import { listVideoTransmitters } from "../../gen/quadsmith/video_transmitter-VideoTransmitterService_connectquery";
import { listCameras } from "../../gen/quadsmith/camera-CameraService_connectquery";
import { listAntennas } from "../../gen/quadsmith/antenna-AntennaService_connectquery";
import { listGpsReceivers } from "../../gen/quadsmith/gps_receiver-GpsReceiverService_connectquery";
import { listBatteries } from "../../gen/quadsmith/battery-BatteryService_connectquery";
import { createBuild, listBuilds } from "../../gen/quadsmith/build-BuildService_connectquery";

// Protobuf types & services
import { BuildSchema, type Build } from "../../gen/quadsmith/build_pb";
import { FrameService, type Frame } from "../../gen/quadsmith/frame_pb";
import { MotorService, type Motor } from "../../gen/quadsmith/motor_pb";
import { PropellerService, type Propeller } from "../../gen/quadsmith/propeller_pb";
import {
  FlightControllerService,
  type FlightController,
} from "../../gen/quadsmith/flight_controller_pb";
import {
  ElectronicSpeedControllerSchema,
  type ElectronicSpeedController,
} from "../../gen/quadsmith/electronic_speed_controller_pb";
import { ReceiverSchema, type Receiver } from "../../gen/quadsmith/receiver_pb";
import {
  VideoTransmitterSchema,
  type VideoTransmitter,
} from "../../gen/quadsmith/video_transmitter_pb";
import { type Camera } from "../../gen/quadsmith/camera_pb";
import { type Antenna } from "../../gen/quadsmith/antenna_pb";
import { type GpsReceiver } from "../../gen/quadsmith/gps_receiver_pb";
import { type Battery } from "../../gen/quadsmith/battery_pb";
import {
  AssembledComponentsSchema,
  type AssembledComponents,
} from "../../gen/quadsmith/components_pb";
import {
  CheckComponentsCompatibilityRequestSchema,
  type CompatibilityMessage,
} from "../../gen/quadsmith/compatibility_pb";
import {
  EvaluateComponentsRequestSchema,
  GetComponentsElectricalLimitsRequestSchema,
} from "../../gen/quadsmith/evaluator_pb";
import { wasmEngine, useWasmEngine } from "../../lib/wasmEngine";

export function getRxAntennaCount(rx: Receiver | null | undefined): number {
  if (!rx) return 1;
  const text = `${rx.name} ${rx.description} ${rx.protocol}`.toLowerCase();
  if (
    text.includes("true diversity") ||
    text.includes("antenna diversity") ||
    text.includes("diversity") ||
    text.includes("gemini") ||
    text.includes("dual") ||
    (rx.antennaUuids && rx.antennaUuids.length >= 2)
  ) {
    return 2;
  }
  return 1;
}

export function getVtxAntennaCount(vtx: VideoTransmitter | null | undefined): number {
  if (!vtx) return 1;
  const name = (vtx.name || "").toLowerCase();
  const proto = (vtx.protocol || "").toLowerCase();

  if (proto.includes("o4 lite") || name.includes("o4 lite")) {
    return 1;
  }

  if (
    proto.includes("dji o3") ||
    proto.includes("dji o4 pro") ||
    proto.includes("o4 pro") ||
    name.includes("dji o3") ||
    name.includes("o3 air unit") ||
    name.includes("dji o4 pro") ||
    name.includes("o4 pro") ||
    name.includes("o4 air unit") ||
    name.includes("moonlight") ||
    name.includes("dual")
  ) {
    return 2;
  }

  if (vtx.antennaUuids && vtx.antennaUuids.length > 0) {
    return vtx.antennaUuids.length;
  }
  return 1;
}

export function getVtxFrequency(vtx?: VideoTransmitter | null): number {
  if (!vtx) return 5800;
  if (vtx.frequencyBandMhz && vtx.frequencyBandMhz > 0) {
    return vtx.frequencyBandMhz;
  }
  const text = `${vtx.name} ${vtx.description || ""} ${vtx.protocol || ""}`.toLowerCase();
  if (
    text.includes("1.2g") ||
    text.includes("1.3g") ||
    text.includes("1.2 ghz") ||
    text.includes("1.3 ghz")
  ) {
    return 1200;
  }
  if (text.includes("2.4g") || text.includes("2.4 ghz")) {
    return 2400;
  }
  if (text.includes("3.3g") || text.includes("3.3 ghz")) {
    return 3300;
  }
  return 5800;
}

export function getVtxConnectors(
  vtx?: VideoTransmitter | null,
  allAntennas: Antenna[] = [],
): string[] {
  if (!vtx) return [];
  if (vtx.antennaConnectors && vtx.antennaConnectors.length > 0) {
    return vtx.antennaConnectors;
  }
  if (vtx.antennaUuids && vtx.antennaUuids.length > 0) {
    const connectors = new Set<string>();
    for (const uuid of vtx.antennaUuids) {
      const ant = allAntennas.find((a) => a.uuid === uuid);
      if (ant && ant.connector) {
        connectors.add(ant.connector);
        if (ant.connector.toLowerCase() === "dual u.fl") {
          connectors.add("U.FL");
        }
      }
    }
    if (connectors.size > 0) {
      return Array.from(connectors);
    }
  }
  const text = `${vtx.name} ${vtx.description || ""}`.toLowerCase();
  const proto = (vtx.protocol || "").toLowerCase();
  if (
    text.includes("dji o3") ||
    text.includes("o3 air unit") ||
    text.includes("dji o4") ||
    text.includes("o4 air unit") ||
    proto.includes("o3") ||
    proto.includes("o4") ||
    text.includes("dual u.fl") ||
    text.includes("dual ufl")
  ) {
    return ["Dual U.FL", "U.FL"];
  }
  if (text.includes("u.fl") || text.includes("ufl") || text.includes("ipex")) {
    return ["U.FL"];
  }
  if (text.includes("mmcx")) {
    return ["MMCX", "SMA"];
  }
  if (text.includes("rp-sma") || text.includes("rpsma")) {
    return ["RP-SMA"];
  }
  if (text.includes("sma")) {
    return ["SMA"];
  }
  return ["SMA", "U.FL"];
}

export function checkVtxAntennaCompatibility(
  vtx: VideoTransmitter | null | undefined,
  ant: Antenna,
  allAntennas: Antenna[] = [],
): { compatible: boolean; reason?: string } {
  if (!vtx) return { compatible: true };

  // 1. Frequency check
  const vtxFreq = getVtxFrequency(vtx);
  if (ant.frequencyBandMhz > 0 && Math.abs(ant.frequencyBandMhz - vtxFreq) > 500) {
    return {
      compatible: false,
      reason: `Frequency mismatch: VTX operates at ${formatFrequencyBand(vtxFreq)}, antenna operates at ${formatFrequencyBand(ant.frequencyBandMhz)}.`,
    };
  }

  // 2. Connector check
  const vtxConnectors = getVtxConnectors(vtx, allAntennas);
  if (vtxConnectors.length > 0 && ant.connector) {
    const antConnLower = ant.connector.trim().toLowerCase();
    const matched = vtxConnectors.some((vc) => {
      const vcLower = vc.trim().toLowerCase();
      if (antConnLower === vcLower) return true;
      if (
        (antConnLower.includes("u.fl") || antConnLower.includes("ipex")) &&
        (vcLower.includes("u.fl") || vcLower.includes("ipex"))
      ) {
        return true;
      }
      return false;
    });
    if (!matched) {
      return {
        compatible: false,
        reason: `Connector mismatch: VTX requires ${vtxConnectors.join(" or ")}, but selected antenna has a ${ant.connector} connector.`,
      };
    }
  }

  return { compatible: true };
}

export function formatFrequencyBand(mhz?: number): string {
  if (!mhz) return "2.4 GHz";
  if (mhz >= 1000) {
    const ghz = mhz / 1000;
    return `${ghz % 1 === 0 ? ghz.toFixed(0) : ghz.toFixed(1)} GHz`;
  }
  return `${mhz} MHz`;
}

export function isRxAntenna(a: Antenna): boolean {
  if (a.isInternalOnly) return false;
  if (a.frequencyBandMhz > 0 && a.frequencyBandMhz <= 3000) return true;
  const nameLower = a.name.toLowerCase();
  return (
    nameLower.includes("t-antenna") ||
    nameLower.includes("dipole") ||
    nameLower.includes("moxon") ||
    nameLower.includes("rx") ||
    nameLower.includes("2.4") ||
    nameLower.includes("915") ||
    nameLower.includes("868")
  );
}

export function rxHasIntegratedAntenna(
  rx?: Receiver | null,
  ants: Antenna[] = [],
  fc?: FlightController | null,
  useIntegratedRx?: boolean,
): boolean {
  if (!rx) return false;

  if (rx.antennaUuids && rx.antennaUuids.length > 0) {
    const hasIntegratedUuid = rx.antennaUuids.some((uuid) => {
      if (uuid === "01923019-3009-7001-8001-000000000012") return true;
      const ant = ants.find((a) => a.uuid === uuid);
      if (ant && (ant.isInternalOnly || ant.connector?.toLowerCase() === "integrated")) {
        return true;
      }
      return false;
    });
    if (hasIntegratedUuid) return true;
  }

  const rxText = `${rx.name} ${rx.description}`.toLowerCase();
  if (
    rxText.includes("ceramic") ||
    rxText.includes("smd antenna") ||
    rxText.includes("integrated antenna") ||
    rxText.includes("built-in antenna") ||
    rxText.includes("internal antenna") ||
    rxText.includes("onboard ceramic") ||
    rxText.includes("on-board ceramic")
  ) {
    return true;
  }

  if (useIntegratedRx && fc) {
    const fcText = `${fc.name} ${fc.description}`.toLowerCase();
    if (
      fcText.includes("ceramic antenna") ||
      fcText.includes("smd antenna") ||
      fcText.includes("integrated antenna") ||
      fcText.includes("built-in antenna") ||
      fcText.includes("internal antenna") ||
      fcText.includes("integrated ceramic")
    ) {
      return true;
    }
  }

  return false;
}

export function vtxHasIntegratedAntenna(
  vtx?: VideoTransmitter | null,
  ants: Antenna[] = [],
): boolean {
  if (!vtx) return false;
  if (vtx.antennaUuids && vtx.antennaUuids.length > 0) {
    return vtx.antennaUuids.some((uuid) => {
      const ant = ants.find((a) => a.uuid === uuid);
      return ant && (ant.isInternalOnly || ant.connector?.toLowerCase() === "integrated");
    });
  }
  const text = `${vtx.name} ${vtx.description}`.toLowerCase();
  return text.includes("integrated antenna") || text.includes("built-in antenna");
}

export interface NoneSelections {
  camera?: boolean;
  vtx?: boolean;
  vtxAntenna?: boolean;
  gps?: boolean;
  esc?: boolean;
}

interface ForgeContextType {
  // Navigation
  activeStep: number;
  setActiveStep: (step: number) => void;
  unlockedSteps: number[];
  stepCompletion: Record<number, boolean>;
  stepConflicts: Record<number, string>;
  stepBatteryConflicts: Record<number, boolean>;
  handleNextStep: () => void;
  handlePrevStep: () => void;

  // Catalog Data
  frames: Frame[];
  fcs: FlightController[];
  rxs: Receiver[];
  escs: ElectronicSpeedController[];
  motors: Motor[];
  props: Propeller[];
  cams: Camera[];
  vtxs: VideoTransmitter[];
  antennas: Antenna[];
  gpsList: GpsReceiver[];
  batteries: Battery[];
  templateBuilds: Build[];
  isWasmReady: boolean;

  // Baseline / Template (Step 0)
  selectedTemplateId: string | null;
  isTemplateConfirmed: boolean;
  applyTemplate: (template: Build) => Promise<void>;
  selectNoTemplate: () => void;

  // Selections
  selectedFrame: Frame | null;
  setSelectedFrame: (f: Frame | null) => void;
  selectedFc: FlightController | null;
  setSelectedFc: (fc: FlightController | null) => void;
  selectedRx: Receiver | null;
  setSelectedRx: (rx: Receiver | null) => void;
  selectedRxAnt: Antenna | null;
  setSelectedRxAnt: (ant: Antenna | null) => void;
  selectedEsc: ElectronicSpeedController | null;
  setSelectedEsc: (esc: ElectronicSpeedController | null) => void;
  selectedMotor: Motor | null;
  setSelectedMotor: (m: Motor | null) => void;
  selectedProp: Propeller | null;
  setSelectedProp: (p: Propeller | null) => void;
  selectedCam: Camera | null;
  setSelectedCam: (c: Camera | null) => void;
  selectedVtx: VideoTransmitter | null;
  setSelectedVtx: (v: VideoTransmitter | null) => void;
  selectedVtxAnt: Antenna | null;
  setSelectedVtxAnt: (ant: Antenna | null) => void;
  selectedGps: GpsReceiver | null;
  setSelectedGps: (g: GpsReceiver | null) => void;
  selectedBatteryId: string;
  setSelectedBatteryId: (id: string) => void;
  activeBattery: Battery | null;

  // Integrated & Optional state
  noneSelections: NoneSelections;
  setNoneSelections: React.Dispatch<React.SetStateAction<NoneSelections>>;
  useIntegratedEsc: boolean;
  setUseIntegratedEsc: (val: boolean) => void;
  integratedEsc: ElectronicSpeedController | null;
  fcHasIntegratedEsc: boolean;
  fcHasAdequateIntegratedEsc: boolean;

  useIntegratedRx: boolean;
  setUseIntegratedRx: (val: boolean) => void;
  integratedRx: Receiver | null;
  fcHasIntegratedRx: boolean;
  isRxAntennaIntegrated: boolean;

  useIntegratedVtx: boolean;
  setUseIntegratedVtx: (val: boolean) => void;
  integratedVtx: VideoTransmitter | null;
  fcHasIntegratedVtx: boolean;
  isVtxAntennaIntegrated: boolean;

  // Derived counts
  motorCount: number;
  rxAntCount: number;
  vtxAntCount: number;

  // CEL Filters & Compatible Only
  compatibleOnlyMap: Record<string, boolean>;
  toggleCompatibleOnly: (collection: string) => void;
  isCompatibleOnly: (collection: string) => boolean;
  getCelFilterForStep: (stepNumber: number) => string;
  batteryCelFilter: string;

  // Search queries
  searchQueries: Record<string, string>;
  setSearchQuery: (collection: string, query: string) => void;

  // Evaluator & Limits
  electricalLimits: {
    minVoltage: number;
    maxVoltage: number;
    maxCurrentA: number;
    connector: string;
  };
  evaluatedMetrics: any;
  wasmCompatibility: any;
  dryWeightG: number;
  allUpWeightG: number;
  payloadWeightG: number;
  setPayloadWeightG: (val: number) => void;
  compatibleBatteries: Battery[];

  // Build Metadata & Saving
  buildName: string;
  setBuildName: (name: string) => void;
  buildDesc: string;
  setBuildDesc: (desc: string) => void;
  isPrivate: boolean;
  setIsPrivate: (val: boolean) => void;
  isSaving: boolean;
  saveError: string | null;
  handleSaveBuild: () => Promise<void>;
  resetBuild: () => void;
}

const ForgeContext = createContext<ForgeContextType | null>(null);

export function ForgeProvider({ children }: { children: React.ReactNode }) {
  const navigate = useNavigate();
  const transport = useTransport();
  const { isReady: isWasmReady } = useWasmEngine();

  // Active step
  const [activeStep, setActiveStep] = useState<number>(0);

  // Queries
  const { data: framesData } = useQuery(listFrames, { pageSize: 100 });
  const { data: fcsData } = useQuery(listFlightControllers, { pageSize: 100 });
  const { data: rxsData } = useQuery(listReceivers, { pageSize: 100 });
  const { data: escsData } = useQuery(listElectronicSpeedControllers, { pageSize: 100 });
  const { data: motorsData } = useQuery(listMotors, { pageSize: 100 });
  const { data: propsData } = useQuery(listPropellers, { pageSize: 100 });
  const { data: camsData } = useQuery(listCameras, { pageSize: 100 });
  const { data: vtxsData } = useQuery(listVideoTransmitters, { pageSize: 100 });
  const { data: antennasData } = useQuery(listAntennas, { pageSize: 100 });
  const { data: gpsData } = useQuery(listGpsReceivers, { pageSize: 100 });
  const { data: batteriesData } = useQuery(listBatteries, { pageSize: 100 });
  const { data: buildsData } = useQuery(listBuilds, { pageSize: 50 });

  const frames = useMemo(() => framesData?.frames || [], [framesData]);
  const fcs = useMemo(() => fcsData?.flightControllers || [], [fcsData]);
  const rxs = useMemo(() => rxsData?.receivers || [], [rxsData]);
  const escs = useMemo(() => escsData?.electronicSpeedControllers || [], [escsData]);
  const motors = useMemo(() => motorsData?.motors || [], [motorsData]);
  const props = useMemo(() => propsData?.propellers || [], [propsData]);
  const cams = useMemo(() => camsData?.cameras || [], [camsData]);
  const vtxs = useMemo(() => vtxsData?.videoTransmitters || [], [vtxsData]);
  const antennas = useMemo(() => antennasData?.antennas || [], [antennasData]);
  const gpsList = useMemo(() => gpsData?.gpsReceivers || [], [gpsData]);
  const batteries = useMemo(() => batteriesData?.batteries || [], [batteriesData]);
  const templateBuilds = useMemo(() => buildsData?.builds || [], [buildsData]);

  // Step 0 Baseline
  const [selectedTemplateId, setSelectedTemplateId] = useState<string | null>(null);
  const [isTemplateConfirmed, setIsTemplateConfirmed] = useState<boolean>(false);

  // Component Selections
  const [selectedFrame, setSelectedFrame] = useState<Frame | null>(null);
  const [selectedFc, setSelectedFc] = useState<FlightController | null>(null);
  const [selectedRx, setSelectedRx] = useState<Receiver | null>(null);
  const [selectedRxAnt, setSelectedRxAnt] = useState<Antenna | null>(null);
  const [selectedEsc, setSelectedEsc] = useState<ElectronicSpeedController | null>(null);
  const [selectedMotor, setSelectedMotor] = useState<Motor | null>(null);
  const [selectedProp, setSelectedProp] = useState<Propeller | null>(null);
  const [selectedCam, setSelectedCam] = useState<Camera | null>(null);
  const [selectedVtx, setSelectedVtx] = useState<VideoTransmitter | null>(null);
  const [selectedVtxAnt, setSelectedVtxAnt] = useState<Antenna | null>(null);
  const [selectedGps, setSelectedGps] = useState<GpsReceiver | null>(null);
  const [selectedBatteryId, setSelectedBatteryId] = useState<string>("");
  const [payloadWeightG, setPayloadWeightG] = useState<number>(0);

  // Optional / None Selections
  const [noneSelections, setNoneSelections] = useState<NoneSelections>({
    camera: false,
    vtx: false,
    vtxAntenna: false,
    gps: false,
    esc: false,
  });

  // Integrated Parts flags
  const [useIntegratedEsc, setUseIntegratedEsc] = useState<boolean>(false);
  const [useIntegratedRx, setUseIntegratedRx] = useState<boolean>(false);
  const [useIntegratedVtx, setUseIntegratedVtx] = useState<boolean>(false);

  // Search queries per collection
  const [searchQueries, setSearchQueries] = useState<Record<string, string>>({});
  const setSearchQuery = useCallback((collection: string, query: string) => {
    setSearchQueries((prev) => ({ ...prev, [collection]: query }));
  }, []);

  // Compatible-Only toggles (default true)
  const [compatibleOnlyMap, setCompatibleOnlyMap] = useState<Record<string, boolean>>({
    frames: true,
    flight_controllers: true,
    receivers: true,
    rx_antennas: true,
    electronic_speed_controllers: true,
    motors: true,
    propellers: true,
    cameras: true,
    video_transmitters: true,
    vtx_antennas: true,
    gps_receivers: true,
    batteries: true,
  });

  const toggleCompatibleOnly = useCallback((collection: string) => {
    setCompatibleOnlyMap((prev) => ({ ...prev, [collection]: !prev[collection] }));
  }, []);

  const isCompatibleOnly = useCallback(
    (collection: string) => compatibleOnlyMap[collection] ?? true,
    [compatibleOnlyMap],
  );

  // Build Metadata
  const [buildName, setBuildName] = useState<string>("Custom Build");
  const [buildDesc, setBuildDesc] = useState<string>(
    "Custom build configured via Quadsmith: The Forge.",
  );
  const [isPrivate, setIsPrivate] = useState<boolean>(false);
  const [saveError, setSaveError] = useState<string | null>(null);

  const saveMutation = useMutation(createBuild);
  const isSaving = saveMutation.isPending;

  // Active battery
  const activeBattery = useMemo(() => {
    return (
      batteries.find((b) => b.id === selectedBatteryId || b.uuid === selectedBatteryId) || null
    );
  }, [batteries, selectedBatteryId]);

  // Motor Count
  const motorCount = selectedFrame?.motorCount || 4;

  // Integrated ESC on selected FC
  const integratedEsc = useMemo<ElectronicSpeedController | null>(() => {
    if (!selectedFc) return null;
    if (selectedFc.internalElectronicSpeedControllerUuid) {
      const match = escs.find((e) => e.uuid === selectedFc.internalElectronicSpeedControllerUuid);
      if (match) return match;
    }
    const fcDesc = (selectedFc.description || "").toLowerCase();
    const fcName = (selectedFc.name || "").toLowerCase();
    if (fcDesc.includes("aio") || fcDesc.includes("all-in-one") || fcName.includes("aio")) {
      let currentRating = 12.0;
      const currentMatch = (selectedFc.name + " " + selectedFc.description).match(
        /(\d+)A(?:\s+ESC|\s+AIO)/i,
      );
      if (currentMatch) {
        currentRating = parseFloat(currentMatch[1]);
      }
      return create(ElectronicSpeedControllerSchema, {
        uuid: `integrated-esc-${selectedFc.uuid}`,
        id: `integrated-esc-${selectedFc.id || selectedFc.uuid}`,
        name: `${selectedFc.name} (Integrated ESC)`,
        manufacturer: selectedFc.manufacturer,
        isInternalOnly: true,
        maxMotors: 4,
        motorCurrentMaxA: currentRating,
        minVoltage: selectedFc.minVoltage || 3.0,
        maxVoltage: selectedFc.maxVoltage || 13.0,
        firmware: "BLHeli_S",
      });
    }
    return null;
  }, [selectedFc, escs]);

  const fcHasIntegratedEsc = !!integratedEsc;
  const fcHasAdequateIntegratedEsc = useMemo(() => {
    if (!integratedEsc) return false;
    if (selectedFrame && (integratedEsc.maxMotors || 4) < motorCount) {
      return false;
    }
    return true;
  }, [integratedEsc, selectedFrame, motorCount]);

  // Integrated RX on selected FC
  const integratedRx = useMemo<Receiver | null>(() => {
    if (!selectedFc) return null;
    if (selectedFc.internalReceiverUuid) {
      const match = rxs.find((r) => r.uuid === selectedFc.internalReceiverUuid);
      if (match) return match;
    }
    const fcDesc = (selectedFc.description || "").toLowerCase();
    const fcName = (selectedFc.name || "").toLowerCase();
    if (fcDesc.includes("elrs") || fcDesc.includes("expresslrs") || fcName.includes("elrs")) {
      let proto = "ExpressLRS 2.4GHz";
      let band = 2400;
      if (fcDesc.includes("915") || fcName.includes("915")) {
        proto = "ExpressLRS 915MHz";
        band = 915;
      }
      return create(ReceiverSchema, {
        uuid: `integrated-rx-${selectedFc.uuid}`,
        id: `integrated-rx-${selectedFc.id || selectedFc.uuid}`,
        name: `${selectedFc.name} (Integrated RX)`,
        manufacturer: selectedFc.manufacturer,
        isInternalOnly: true,
        protocol: proto,
        frequencyBandMhz: band,
        hasTelemetry: true,
        antennaUuids: ["01923019-3009-7001-8001-000000000012"],
      });
    }
    return null;
  }, [selectedFc, rxs]);

  const fcHasIntegratedRx = !!integratedRx;

  // Integrated VTX on selected FC
  const integratedVtx = useMemo<VideoTransmitter | null>(() => {
    if (!selectedFc) return null;
    if (selectedFc.internalVideoTransmitterUuid) {
      const match = vtxs.find((v) => v.uuid === selectedFc.internalVideoTransmitterUuid);
      if (match) return match;
    }
    const fcDesc = (selectedFc.description || "").toLowerCase();
    const fcName = (selectedFc.name || "").toLowerCase();
    if (fcDesc.includes("vtx") || fcName.includes("vtx")) {
      return create(VideoTransmitterSchema, {
        uuid: `integrated-vtx-${selectedFc.uuid}`,
        id: `integrated-vtx-${selectedFc.id || selectedFc.uuid}`,
        name: `${selectedFc.name} (Integrated VTX)`,
        manufacturer: selectedFc.manufacturer,
        isInternalOnly: true,
        protocol: "Analog",
        maxPowerMw: 400,
      });
    }
    return null;
  }, [selectedFc, vtxs]);

  const fcHasIntegratedVtx = !!integratedVtx;

  const autoSelectedFcRef = useRef<string | null>(null);

  // Auto-selection of integrated components when FC is chosen
  useEffect(() => {
    const fcUuid = selectedFc?.uuid || selectedFc?.id || null;
    if (fcUuid && fcUuid !== autoSelectedFcRef.current) {
      autoSelectedFcRef.current = fcUuid;

      // Integrated ESC: auto-select if adequate and no external ESC selected
      if (fcHasAdequateIntegratedEsc && !selectedEsc) {
        setUseIntegratedEsc(true);
        setSelectedEsc(integratedEsc);
        setNoneSelections((prev) => ({ ...prev, esc: false }));
      }

      // Integrated RX: auto-select if no external RX selected
      if (fcHasIntegratedRx && !selectedRx) {
        setUseIntegratedRx(true);
        setSelectedRx(integratedRx);
      }

      // Integrated VTX:
      // USER RULE: "FC-integrated VTXs typically use the analog protocol. Don't auto-select the integrated VTX if it is not compatible with the camera's protocol."
      const isCameraDigital =
        selectedCam &&
        selectedCam.protocol &&
        !selectedCam.protocol.toLowerCase().includes("analog");
      if (fcHasIntegratedVtx && !selectedVtx && !isCameraDigital) {
        setUseIntegratedVtx(true);
        setSelectedVtx(integratedVtx);
        setNoneSelections((prev) => ({ ...prev, vtx: false }));
      }
    } else if (!fcUuid) {
      autoSelectedFcRef.current = null;
    }
  }, [
    selectedFc,
    fcHasAdequateIntegratedEsc,
    integratedEsc,
    selectedEsc,
    fcHasIntegratedRx,
    integratedRx,
    selectedRx,
    fcHasIntegratedVtx,
    integratedVtx,
    selectedVtx,
    selectedCam,
  ]);

  // Integrated Antenna checks
  const isRxAntennaIntegrated = useMemo(() => {
    const activeR = useIntegratedRx ? integratedRx : selectedRx;
    return rxHasIntegratedAntenna(activeR, antennas, selectedFc, useIntegratedRx);
  }, [useIntegratedRx, integratedRx, selectedRx, antennas, selectedFc]);

  const isVtxAntennaIntegrated = useMemo(() => {
    const activeV = useIntegratedVtx ? integratedVtx : selectedVtx;
    return vtxHasIntegratedAntenna(activeV, antennas);
  }, [useIntegratedVtx, integratedVtx, selectedVtx, antennas]);

  // Antenna counts
  const rxAntCount = useMemo(() => {
    const activeR = useIntegratedRx ? integratedRx : selectedRx;
    return getRxAntennaCount(activeR);
  }, [useIntegratedRx, integratedRx, selectedRx]);

  const vtxAntCount = useMemo(() => {
    const activeV = useIntegratedVtx ? integratedVtx : selectedVtx;
    return getVtxAntennaCount(activeV);
  }, [useIntegratedVtx, integratedVtx, selectedVtx]);

  // Rule: Camera set to None forces VTX and VTX Antennas to None
  useEffect(() => {
    if (noneSelections.camera) {
      if (!noneSelections.vtx) {
        setNoneSelections((prev) => ({ ...prev, vtx: true, vtxAntenna: true }));
        setSelectedVtx(null);
        setUseIntegratedVtx(false);
        setSelectedVtxAnt(null);
      }
    }
  }, [noneSelections.camera, noneSelections.vtx]);

  // Rule: VTX set to None forces VTX Antennas to None
  useEffect(() => {
    if (noneSelections.vtx) {
      if (!noneSelections.vtxAntenna) {
        setNoneSelections((prev) => ({ ...prev, vtxAntenna: true }));
        setSelectedVtxAnt(null);
      }
    }
  }, [noneSelections.vtx, noneSelections.vtxAntenna]);

  // Helper to build AssembledComponents up to step N
  const buildAssembledUpTo = useCallback(
    (maxStep: number): AssembledComponents => {
      return create(AssembledComponentsSchema, {
        frame: maxStep >= 1 && selectedFrame ? selectedFrame : undefined,
        flightController: maxStep >= 2 && selectedFc ? selectedFc : undefined,
        receivers:
          maxStep >= 3
            ? useIntegratedRx && integratedRx
              ? [integratedRx]
              : selectedRx
                ? [selectedRx]
                : []
            : [],
        antennas: [
          ...(maxStep >= 4 && selectedRxAnt && !isRxAntennaIntegrated ? [selectedRxAnt] : []),
          ...(maxStep >= 10 && selectedVtxAnt && !noneSelections.vtxAntenna
            ? [selectedVtxAnt]
            : []),
        ],
        electronicSpeedControllers:
          maxStep >= 5
            ? useIntegratedEsc && integratedEsc
              ? [integratedEsc]
              : selectedEsc && !noneSelections.esc
                ? [selectedEsc]
                : []
            : [],
        motor: maxStep >= 6 && selectedMotor ? selectedMotor : undefined,
        propeller: maxStep >= 7 && selectedProp ? selectedProp : undefined,
        cameras: maxStep >= 8 && selectedCam && !noneSelections.camera ? [selectedCam] : [],
        videoTransmitter:
          maxStep >= 9
            ? useIntegratedVtx && integratedVtx
              ? integratedVtx
              : selectedVtx && !noneSelections.vtx
                ? selectedVtx
                : undefined
            : undefined,
        gpsReceiver: maxStep >= 11 && selectedGps && !noneSelections.gps ? selectedGps : undefined,
        battery: activeBattery || undefined,
      });
    },
    [
      selectedFrame,
      selectedFc,
      selectedRx,
      useIntegratedRx,
      integratedRx,
      selectedRxAnt,
      isRxAntennaIntegrated,
      selectedEsc,
      useIntegratedEsc,
      integratedEsc,
      noneSelections,
      selectedMotor,
      selectedProp,
      selectedCam,
      selectedVtx,
      useIntegratedVtx,
      integratedVtx,
      selectedVtxAnt,
      selectedGps,
      activeBattery,
    ],
  );

  // Full assembled build for Evaluator
  const fullAssembled = useMemo(() => buildAssembledUpTo(11), [buildAssembledUpTo]);

  // Live Evaluator Metrics
  const evaluatedMetrics = useMemo(() => {
    if (!isWasmReady || !selectedFrame || !selectedMotor || !selectedProp || !activeBattery) {
      return null;
    }
    try {
      return wasmEngine.evaluateComponents(
        create(EvaluateComponentsRequestSchema, {
          components: fullAssembled,
          payloadWeightG,
        }),
      );
    } catch {
      return null;
    }
  }, [
    isWasmReady,
    selectedFrame,
    selectedMotor,
    selectedProp,
    activeBattery,
    fullAssembled,
    payloadWeightG,
  ]);

  // Electrical Limits
  const electricalLimits = useMemo(() => {
    if (!isWasmReady) {
      return { minVoltage: 0, maxVoltage: 0, maxCurrentA: 0, connector: "" };
    }
    try {
      return wasmEngine.computeElectricalLimits(
        create(GetComponentsElectricalLimitsRequestSchema, {
          components: fullAssembled,
        }),
      );
    } catch {
      return { minVoltage: 0, maxVoltage: 0, maxCurrentA: 0, connector: "" };
    }
  }, [isWasmReady, fullAssembled]);

  // Compatible Batteries dynamically filtered by electrical limits
  const compatibleBatteries = useMemo(() => {
    if (!batteries.length) return [];
    if (
      !electricalLimits ||
      (electricalLimits.minVoltage === 0 && electricalLimits.maxVoltage === 0)
    ) {
      return batteries;
    }
    return batteries.filter((b) => {
      if (electricalLimits.minVoltage > 0 && b.maxVoltage < electricalLimits.minVoltage) {
        return false;
      }
      if (electricalLimits.maxVoltage > 0 && b.minVoltage > electricalLimits.maxVoltage) {
        return false;
      }
      if (electricalLimits.connector && b.connector && electricalLimits.connector !== b.connector) {
        return false;
      }
      return true;
    });
  }, [batteries, electricalLimits]);

  // Battery dynamic CEL filter based on active build electrical envelope
  const batteryCelFilter = useMemo(() => {
    if (!isWasmReady) return "";
    try {
      return wasmEngine.generateCelFilter("batteries", fullAssembled);
    } catch {
      return "";
    }
  }, [isWasmReady, fullAssembled]);

  // Step-by-Step Unidirectional Incompatibility Engine
  // Evaluates Step K against Steps 1..K-1.
  const { stepConflicts, stepBatteryConflicts } = useMemo(() => {
    const conflicts: Record<number, string> = {};
    const batConflicts: Record<number, boolean> = {};

    if (!isWasmReady) {
      return { stepConflicts: conflicts, stepBatteryConflicts: batConflicts };
    }

    // Step 2: FC vs Frame
    if (selectedFc && selectedFrame) {
      const msgs = wasmEngine.checkCompatibility(
        create(CheckComponentsCompatibilityRequestSchema, {
          components: buildAssembledUpTo(2),
        }),
      );
      const fcConflict = msgs.messages.find(
        (m: CompatibilityMessage) => m.checkerName === "FlightControllerFrameMountChecker",
      );
      if (fcConflict) conflicts[2] = fcConflict.message;
    }

    // Step 4: RX Antenna vs RX
    const activeR = useIntegratedRx ? integratedRx : selectedRx;
    if (selectedRxAnt && activeR && !isRxAntennaIntegrated) {
      const msgs = wasmEngine.checkCompatibility(
        create(CheckComponentsCompatibilityRequestSchema, {
          components: buildAssembledUpTo(4),
        }),
      );
      const antConflict = msgs.messages.find((m: CompatibilityMessage) =>
        m.checkerName.startsWith("ReceiverAntenna"),
      );
      if (antConflict) conflicts[4] = antConflict.message;
    }

    // Step 5: ESC vs Frame / FC
    const activeE = useIntegratedEsc ? integratedEsc : selectedEsc;
    if (activeE && !noneSelections.esc) {
      const msgs = wasmEngine.checkCompatibility(
        create(CheckComponentsCompatibilityRequestSchema, {
          components: buildAssembledUpTo(5),
        }),
      );
      const escConflict = msgs.messages.find(
        (m: CompatibilityMessage) =>
          m.checkerName === "EscChannelCountChecker" ||
          m.checkerName === "FlightControllerElectronicSpeedControllerVoltageChecker",
      );
      if (escConflict) conflicts[5] = escConflict.message;
    }

    // Step 6: Motor vs Frame / ESC / FC
    if (selectedMotor) {
      const msgs = wasmEngine.checkCompatibility(
        create(CheckComponentsCompatibilityRequestSchema, {
          components: buildAssembledUpTo(6),
        }),
      );
      const motorConflict = msgs.messages.find(
        (m: CompatibilityMessage) =>
          m.checkerName === "MotorFrameMountChecker" ||
          m.checkerName === "MotorElectronicSpeedControllerChecker" ||
          m.checkerName === "MotorElectronicSpeedControllerVoltageChecker" ||
          m.checkerName === "MotorFlightControllerVoltageChecker",
      );
      if (motorConflict) conflicts[6] = motorConflict.message;
    }

    // Step 7: Propeller vs Frame / Motor
    if (selectedProp) {
      const msgs = wasmEngine.checkCompatibility(
        create(CheckComponentsCompatibilityRequestSchema, {
          components: buildAssembledUpTo(7),
        }),
      );
      const propConflict = msgs.messages.find(
        (m: CompatibilityMessage) =>
          m.checkerName === "PropellerFrameChecker" ||
          m.checkerName === "MotorPropellerMountChecker",
      );
      if (propConflict) conflicts[7] = propConflict.message;
    }

    // Step 8: Camera vs Frame
    if (selectedCam && !noneSelections.camera) {
      const msgs = wasmEngine.checkCompatibility(
        create(CheckComponentsCompatibilityRequestSchema, {
          components: buildAssembledUpTo(8),
        }),
      );
      const camConflict = msgs.messages.find(
        (m: CompatibilityMessage) => m.checkerName === "CameraFrameMountChecker",
      );
      if (camConflict) conflicts[8] = camConflict.message;
    }

    // Step 9: VTX vs Frame / FC / Camera
    const activeV = useIntegratedVtx ? integratedVtx : selectedVtx;
    if (activeV && !noneSelections.vtx) {
      const msgs = wasmEngine.checkCompatibility(
        create(CheckComponentsCompatibilityRequestSchema, {
          components: buildAssembledUpTo(9),
        }),
      );
      const vtxConflict = msgs.messages.find(
        (m: CompatibilityMessage) =>
          m.checkerName === "VideoTransmitterFrameMountChecker" ||
          m.checkerName === "CameraVideoTransmitterProtocolChecker" ||
          m.checkerName === "FlightControllerVtxVoltageChecker" ||
          m.checkerName === "FlightControllerVtxPortChecker",
      );
      if (vtxConflict) conflicts[9] = vtxConflict.message;
    }

    // Step 10: VTX Antenna vs VTX
    if (selectedVtxAnt && activeV && !noneSelections.vtxAntenna && !isVtxAntennaIntegrated) {
      const msgs = wasmEngine.checkCompatibility(
        create(CheckComponentsCompatibilityRequestSchema, {
          components: buildAssembledUpTo(10),
        }),
      );
      const antConflict = msgs.messages.find((m: CompatibilityMessage) =>
        m.checkerName.startsWith("VideoTransmitterAntenna"),
      );
      if (antConflict) {
        conflicts[10] = antConflict.message;
      } else {
        const check = checkVtxAntennaCompatibility(activeV, selectedVtxAnt, antennas);
        if (!check.compatible) {
          conflicts[10] =
            check.reason || "VTX antenna is incompatible with the selected video transmitter.";
        }
      }
    }

    // Battery Conflict Check against selected components
    if (activeBattery) {
      const bMin = activeBattery.minVoltage || 0;
      const bMax = activeBattery.maxVoltage || 0;

      // FC vs Battery
      if (selectedFc) {
        if (selectedFc.maxVoltage > 0 && bMin > selectedFc.maxVoltage) {
          batConflicts[2] = true;
        } else if (selectedFc.minVoltage > 0 && bMax < selectedFc.minVoltage) {
          batConflicts[2] = true;
        }
      }

      // ESC vs Battery
      if (activeE && !noneSelections.esc) {
        if (activeE.maxVoltage > 0 && bMin > activeE.maxVoltage) {
          batConflicts[5] = true;
        } else if (activeE.minVoltage > 0 && bMax < activeE.minVoltage) {
          batConflicts[5] = true;
        }
      }

      // Motor vs Battery
      if (selectedMotor) {
        if (selectedMotor.maxVoltage > 0 && bMin > selectedMotor.maxVoltage) {
          batConflicts[6] = true;
        } else if (selectedMotor.minVoltage > 0 && bMax < selectedMotor.minVoltage) {
          batConflicts[6] = true;
        }
      }
    }

    return {
      stepConflicts: conflicts,
      stepBatteryConflicts: batConflicts,
    };
  }, [
    isWasmReady,
    selectedFrame,
    selectedFc,
    selectedRx,
    useIntegratedRx,
    integratedRx,
    selectedRxAnt,
    isRxAntennaIntegrated,
    selectedEsc,
    useIntegratedEsc,
    integratedEsc,
    noneSelections,
    selectedMotor,
    selectedProp,
    selectedCam,
    selectedVtx,
    useIntegratedVtx,
    integratedVtx,
    selectedVtxAnt,
    isVtxAntennaIntegrated,
    activeBattery,
    buildAssembledUpTo,
  ]);

  // Overall WASM Compatibility
  const wasmCompatibility = useMemo(() => {
    if (!isWasmReady || !selectedFrame) return null;
    try {
      return wasmEngine.checkCompatibility(
        create(CheckComponentsCompatibilityRequestSchema, {
          components: fullAssembled,
        }),
      );
    } catch {
      return null;
    }
  }, [isWasmReady, selectedFrame, fullAssembled]);

  // Step Completion Status
  const stepCompletion = useMemo<Record<number, boolean>>(() => {
    return {
      0: isTemplateConfirmed,
      1: !!selectedFrame,
      2: !!selectedFc,
      3: !!(useIntegratedRx ? integratedRx : selectedRx),
      4: !!(isRxAntennaIntegrated || selectedRxAnt),
      5: !!(useIntegratedEsc ? integratedEsc : selectedEsc || noneSelections.esc),
      6: !!selectedMotor,
      7: !!selectedProp,
      8: !!(selectedCam || noneSelections.camera),
      9: !!(noneSelections.vtx || (useIntegratedVtx ? integratedVtx : selectedVtx)),
      10: !!(noneSelections.vtxAntenna || isVtxAntennaIntegrated || selectedVtxAnt),
      11: !!(selectedGps || noneSelections.gps),
      12: false,
    };
  }, [
    isTemplateConfirmed,
    selectedFrame,
    selectedFc,
    useIntegratedRx,
    integratedRx,
    selectedRx,
    isRxAntennaIntegrated,
    selectedRxAnt,
    useIntegratedEsc,
    integratedEsc,
    selectedEsc,
    noneSelections,
    selectedMotor,
    selectedProp,
    selectedCam,
    useIntegratedVtx,
    integratedVtx,
    selectedVtx,
    isVtxAntennaIntegrated,
    selectedVtxAnt,
    selectedGps,
  ]);

  // Unlocked Steps Logic:
  // Step 0 is always unlocked.
  // Step 1 unlocks once Step 0 is confirmed.
  // Step K+1 unlocks if all steps <= K are completed AND no step <= K has an unresolved conflict.
  // If a step has a conflict, that step itself is unlocked (so the user can navigate to it and fix it), but all subsequent steps are LOCKED.
  const unlockedSteps = useMemo(() => {
    const unlocked = [0];
    if (!stepCompletion[0]) return unlocked;
    unlocked.push(1);

    for (let step = 1; step <= 11; step++) {
      // If this step is not complete or has a conflict, subsequent steps are locked
      if (!stepCompletion[step]) break;
      if (stepConflicts[step]) break;
      unlocked.push(step + 1);
    }

    return unlocked;
  }, [stepCompletion, stepConflicts]);

  // Dynamic CEL filter generator for catalog queries
  const getCelFilterForStep = useCallback(
    (stepNumber: number): string => {
      if (!isWasmReady) return "";
      try {
        switch (stepNumber) {
          case 1:
            return "";
          case 2:
            return wasmEngine.generateCelFilter("flight_controllers", buildAssembledUpTo(1));
          case 3:
            return wasmEngine.generateCelFilter("receivers", buildAssembledUpTo(2));
          case 4:
            return wasmEngine.generateCelFilter("rx_antennas", buildAssembledUpTo(3));
          case 5:
            return wasmEngine.generateCelFilter(
              "electronic_speed_controllers",
              buildAssembledUpTo(4),
            );
          case 6:
            return wasmEngine.generateCelFilter("motors", buildAssembledUpTo(5));
          case 7:
            return wasmEngine.generateCelFilter("propellers", buildAssembledUpTo(6));
          case 8:
            return wasmEngine.generateCelFilter("cameras", buildAssembledUpTo(7));
          case 9:
            return wasmEngine.generateCelFilter("video_transmitters", buildAssembledUpTo(8));
          case 10:
            return wasmEngine.generateCelFilter("vtx_antennas", buildAssembledUpTo(9));
          case 11:
            return wasmEngine.generateCelFilter("gps_receivers", buildAssembledUpTo(10));
          default:
            return "";
        }
      } catch {
        return "";
      }
    },
    [isWasmReady, buildAssembledUpTo],
  );

  // Navigation handlers with Skip Rules
  const handleNextStep = useCallback(() => {
    // Cannot proceed if current step has an unresolved conflict
    if (stepConflicts[activeStep]) return;

    let target = activeStep + 1;

    // Skip rules:
    // If Step 3 (RX) has integrated antenna, skip Step 4 (RX Antennas)
    if (activeStep === 3 && isRxAntennaIntegrated) {
      target = 5;
    }
    // If Step 8 (Camera) is None, skip Step 9 (VTX) and Step 10 (VTX Antennas) -> straight to Step 11 (GPS)
    else if (activeStep === 8 && noneSelections.camera) {
      target = 11;
    }
    // If Step 9 (VTX) is None or has integrated antenna, skip Step 10 (VTX Antennas)
    else if (activeStep === 9 && (noneSelections.vtx || isVtxAntennaIntegrated)) {
      target = 11;
    }

    if (target <= 12 && unlockedSteps.includes(target)) {
      setActiveStep(target);
    }
  }, [
    activeStep,
    stepConflicts,
    isRxAntennaIntegrated,
    noneSelections.camera,
    noneSelections.vtx,
    isVtxAntennaIntegrated,
    unlockedSteps,
  ]);

  const handlePrevStep = useCallback(() => {
    if (activeStep <= 0) return;
    let target = activeStep - 1;

    // Backward skip rules:
    if (activeStep === 5 && isRxAntennaIntegrated) {
      target = 3;
    } else if (activeStep === 11 && noneSelections.camera) {
      target = 8;
    } else if (activeStep === 11 && (noneSelections.vtx || isVtxAntennaIntegrated)) {
      target = 9;
    }

    setActiveStep(Math.max(0, target));
  }, [
    activeStep,
    isRxAntennaIntegrated,
    noneSelections.camera,
    noneSelections.vtx,
    isVtxAntennaIntegrated,
  ]);

  // Baseline Template handlers
  const applyTemplate = useCallback(
    async (build: Build) => {
      setSelectedTemplateId(build.id || build.uuid);
      setIsTemplateConfirmed(true);
      setBuildName(build.name || "Custom Build");
      if (build.description) setBuildDesc(build.description);

      // Frame
      let f = build.frameUuid ? frames.find((x) => x.uuid === build.frameUuid) : null;
      if (!f && build.frameUuid) {
        try {
          const client = createClient(FrameService, transport);
          f = await client.getFrame({ id: build.frameUuid });
        } catch (e) {
          console.error("Failed to fetch template frame:", e);
        }
      }
      if (f) setSelectedFrame(f);

      // Motor
      let m = build.motorUuid ? motors.find((x) => x.uuid === build.motorUuid) : null;
      if (!m && build.motorUuid) {
        try {
          const client = createClient(MotorService, transport);
          m = await client.getMotor({ id: build.motorUuid });
        } catch (e) {
          console.error("Failed to fetch template motor:", e);
        }
      }
      if (m) setSelectedMotor(m);

      // Propeller
      let p = build.propellerUuid ? props.find((x) => x.uuid === build.propellerUuid) : null;
      if (!p && build.propellerUuid) {
        try {
          const client = createClient(PropellerService, transport);
          p = await client.getPropeller({ id: build.propellerUuid });
        } catch (e) {
          console.error("Failed to fetch template propeller:", e);
        }
      }
      if (p) setSelectedProp(p);

      // FC
      let fc = build.flightControllerUuid
        ? fcs.find((x) => x.uuid === build.flightControllerUuid)
        : null;
      if (!fc && build.flightControllerUuid) {
        try {
          const client = createClient(FlightControllerService, transport);
          fc = await client.getFlightController({ id: build.flightControllerUuid });
        } catch (e) {
          console.error("Failed to fetch template FC:", e);
        }
      }
      if (fc) setSelectedFc(fc);

      // ESC
      let esc = build.electronicSpeedControllerUuids?.[0]
        ? escs.find((x) => x.uuid === build.electronicSpeedControllerUuids[0])
        : null;
      if (esc) {
        setSelectedEsc(esc);
        setUseIntegratedEsc(false);
        setNoneSelections((prev) => ({ ...prev, esc: false }));
      }

      // Receiver
      let rx = build.receiverUuids?.[0] ? rxs.find((x) => x.uuid === build.receiverUuids[0]) : null;
      if (rx) {
        setSelectedRx(rx);
        setUseIntegratedRx(false);
      }

      // VTX
      let v = build.videoTransmitterUuid
        ? vtxs.find((x) => x.uuid === build.videoTransmitterUuid)
        : null;
      if (v) {
        setSelectedVtx(v);
        setUseIntegratedVtx(false);
        setNoneSelections((prev) => ({ ...prev, vtx: false }));
      }

      // Camera
      let cam = build.cameraUuids?.[0] ? cams.find((x) => x.uuid === build.cameraUuids[0]) : null;
      if (cam) {
        setSelectedCam(cam);
        setNoneSelections((prev) => ({ ...prev, camera: false }));
      }

      // GPS
      let g = build.gpsReceiverUuid ? gpsList.find((x) => x.uuid === build.gpsReceiverUuid) : null;
      if (g) {
        setSelectedGps(g);
        setNoneSelections((prev) => ({ ...prev, gps: false }));
      }

      // Antennas
      if (build.antennaUuids && build.antennaUuids.length > 0) {
        for (const antUuid of build.antennaUuids) {
          const ant = antennas.find((x) => x.uuid === antUuid);
          if (ant) {
            if (isRxAntenna(ant)) {
              setSelectedRxAnt(ant);
            } else {
              setSelectedVtxAnt(ant);
              setNoneSelections((prev) => ({ ...prev, vtxAntenna: false }));
            }
          }
        }
      }
    },
    [frames, motors, props, fcs, escs, rxs, vtxs, cams, gpsList, antennas, transport],
  );

  const selectNoTemplate = useCallback(() => {
    setSelectedTemplateId(null);
    setIsTemplateConfirmed(true);
  }, []);

  // Reset build to clean slate
  const resetBuild = useCallback(() => {
    setSelectedTemplateId(null);
    setIsTemplateConfirmed(false);
    setSelectedFrame(null);
    setSelectedFc(null);
    setSelectedRx(null);
    setSelectedRxAnt(null);
    setSelectedEsc(null);
    setSelectedMotor(null);
    setSelectedProp(null);
    setSelectedCam(null);
    setSelectedVtx(null);
    setSelectedVtxAnt(null);
    setSelectedGps(null);
    setSelectedBatteryId("");
    setUseIntegratedEsc(false);
    setUseIntegratedRx(false);
    setUseIntegratedVtx(false);
    setNoneSelections({
      camera: false,
      vtx: false,
      vtxAntenna: false,
      gps: false,
      esc: false,
    });
    setBuildName("Custom Build");
    setBuildDesc("Custom build configured via Quadsmith: The Forge.");
    setActiveStep(0);
    setSaveError(null);
  }, []);

  // Dry & All-Up Weights
  const dryWeightG = useMemo(() => {
    let w = 0;
    if (selectedFrame) w += selectedFrame.weightG || 0;
    if (selectedMotor) w += (selectedMotor.weightG || 0) * motorCount;
    if (selectedProp) w += (selectedProp.weightG || 0) * motorCount;
    if (selectedFc) w += selectedFc.weightG || 0;
    if (selectedEsc && !useIntegratedEsc && !noneSelections.esc) w += selectedEsc.weightG || 0;
    if (selectedRx && !useIntegratedRx) w += selectedRx.weightG || 0;
    if (selectedRxAnt && !isRxAntennaIntegrated) w += (selectedRxAnt.weightG || 0) * rxAntCount;
    if (selectedCam && !noneSelections.camera) w += selectedCam.weightG || 0;
    if (selectedVtx && !useIntegratedVtx && !noneSelections.vtx) w += selectedVtx.weightG || 0;
    if (selectedVtxAnt && !noneSelections.vtxAntenna)
      w += (selectedVtxAnt.weightG || 0) * vtxAntCount;
    if (selectedGps && !noneSelections.gps) w += selectedGps.weightG || 0;
    return w;
  }, [
    selectedFrame,
    selectedMotor,
    motorCount,
    selectedProp,
    selectedFc,
    selectedEsc,
    useIntegratedEsc,
    noneSelections,
    selectedRx,
    useIntegratedRx,
    selectedRxAnt,
    isRxAntennaIntegrated,
    rxAntCount,
    selectedCam,
    selectedVtx,
    useIntegratedVtx,
    selectedVtxAnt,
    vtxAntCount,
    selectedGps,
  ]);

  const allUpWeightG = useMemo(() => {
    return dryWeightG + (activeBattery?.weightG || 0) + payloadWeightG;
  }, [dryWeightG, activeBattery, payloadWeightG]);

  // Save Build Handler
  const handleSaveBuild = useCallback(async () => {
    if (!selectedFrame || !selectedMotor || !selectedProp || !selectedFc) {
      setSaveError("Please complete all required components before saving.");
      return;
    }

    try {
      setSaveError(null);
      const res = await saveMutation.mutateAsync({
        build: create(BuildSchema, {
          name: buildName,
          description: buildDesc,
          frameUuid: selectedFrame.uuid,
          motorUuid: selectedMotor.uuid,
          propellerUuid: selectedProp.uuid,
          flightControllerUuid: selectedFc.uuid,
          electronicSpeedControllerUuids:
            selectedEsc && !useIntegratedEsc && !noneSelections.esc ? [selectedEsc.uuid] : [],
          receiverUuids: selectedRx && !useIntegratedRx ? [selectedRx.uuid] : [],
          antennaUuids: [
            selectedRxAnt && !isRxAntennaIntegrated ? selectedRxAnt.uuid : null,
            selectedVtxAnt && !noneSelections.vtxAntenna ? selectedVtxAnt.uuid : null,
          ].filter(Boolean) as string[],
          cameraUuids: selectedCam && !noneSelections.camera ? [selectedCam.uuid] : [],
          videoTransmitterUuid:
            selectedVtx && !useIntegratedVtx && !noneSelections.vtx ? selectedVtx.uuid : "",
          gpsReceiverUuid: selectedGps && !noneSelections.gps ? selectedGps.uuid : "",
        }),
      });

      if (res?.id || res?.uuid) {
        navigate(`/builds/${res.id || res.uuid}`);
      }
    } catch (err: any) {
      setSaveError(err.message || "Failed to save build. Please check all fields.");
    }
  }, [
    selectedFrame,
    selectedMotor,
    selectedProp,
    selectedFc,
    selectedEsc,
    useIntegratedEsc,
    noneSelections,
    selectedRx,
    useIntegratedRx,
    selectedRxAnt,
    isRxAntennaIntegrated,
    selectedVtxAnt,
    selectedCam,
    selectedVtx,
    useIntegratedVtx,
    selectedGps,
    activeBattery,
    buildName,
    buildDesc,
    isPrivate,
    saveMutation,
    navigate,
  ]);

  const value: ForgeContextType = {
    activeStep,
    setActiveStep,
    unlockedSteps,
    stepCompletion,
    stepConflicts,
    stepBatteryConflicts,
    handleNextStep,
    handlePrevStep,

    frames,
    fcs,
    rxs,
    escs,
    motors,
    props,
    cams,
    vtxs,
    antennas,
    gpsList,
    batteries,
    templateBuilds,
    isWasmReady,

    selectedTemplateId,
    isTemplateConfirmed,
    applyTemplate,
    selectNoTemplate,

    selectedFrame,
    setSelectedFrame,
    selectedFc,
    setSelectedFc,
    selectedRx,
    setSelectedRx,
    selectedRxAnt,
    setSelectedRxAnt,
    selectedEsc,
    setSelectedEsc,
    selectedMotor,
    setSelectedMotor,
    selectedProp,
    setSelectedProp,
    selectedCam,
    setSelectedCam,
    selectedVtx,
    setSelectedVtx,
    selectedVtxAnt,
    setSelectedVtxAnt,
    selectedGps,
    setSelectedGps,
    selectedBatteryId,
    setSelectedBatteryId,
    activeBattery,

    noneSelections,
    setNoneSelections,
    useIntegratedEsc,
    setUseIntegratedEsc,
    integratedEsc,
    fcHasIntegratedEsc,
    fcHasAdequateIntegratedEsc,

    useIntegratedRx,
    setUseIntegratedRx,
    integratedRx,
    fcHasIntegratedRx,
    isRxAntennaIntegrated,

    useIntegratedVtx,
    setUseIntegratedVtx,
    integratedVtx,
    fcHasIntegratedVtx,
    isVtxAntennaIntegrated,

    motorCount,
    rxAntCount,
    vtxAntCount,

    compatibleOnlyMap,
    toggleCompatibleOnly,
    isCompatibleOnly,
    getCelFilterForStep,
    batteryCelFilter,

    searchQueries,
    setSearchQuery,

    electricalLimits,
    evaluatedMetrics,
    wasmCompatibility,
    dryWeightG,
    allUpWeightG,
    payloadWeightG,
    setPayloadWeightG,
    compatibleBatteries,

    buildName,
    setBuildName,
    buildDesc,
    setBuildDesc,
    isPrivate,
    setIsPrivate,
    isSaving,
    saveError,
    handleSaveBuild,
    resetBuild,
  };

  return <ForgeContext.Provider value={value}>{children}</ForgeContext.Provider>;
}

export function useForge() {
  const ctx = useContext(ForgeContext);
  if (!ctx) {
    throw new Error("useForge must be used within a ForgeProvider");
  }
  return ctx;
}
