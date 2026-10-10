import { useState, useMemo, useEffect } from "react";
import { useNavigate } from "react-router-dom";
import { useQuery, useMutation, useTransport } from "@connectrpc/connect-query";
import { createClient } from "@connectrpc/connect";
import { create } from "@bufbuild/protobuf";
import { Search, X, ArrowLeft, Sparkles, AlertCircle, RotateCcw, Save, Check } from "lucide-react";
import { useDocumentMeta } from "../hooks/useDocumentMeta";
import { WizardStageBar } from "../components/WizardStageBar";

// ConnectQuery hooks for component listing
import { listFrames } from "../gen/quadsmith/frame-FrameService_connectquery";
import { listMotors } from "../gen/quadsmith/motor-MotorService_connectquery";
import { listPropellers } from "../gen/quadsmith/propeller-PropellerService_connectquery";
import { listFlightControllers } from "../gen/quadsmith/flight_controller-FlightControllerService_connectquery";
import { listElectronicSpeedControllers } from "../gen/quadsmith/electronic_speed_controller-ElectronicSpeedControllerService_connectquery";
import { listReceivers } from "../gen/quadsmith/receiver-ReceiverService_connectquery";
import { listVideoTransmitters } from "../gen/quadsmith/video_transmitter-VideoTransmitterService_connectquery";
import { listCameras } from "../gen/quadsmith/camera-CameraService_connectquery";
import { listAntennas } from "../gen/quadsmith/antenna-AntennaService_connectquery";
import { listGpsReceivers } from "../gen/quadsmith/gps_receiver-GpsReceiverService_connectquery";
import { listBatteries } from "../gen/quadsmith/battery-BatteryService_connectquery";

import {
  evaluateBuild,
  getBuildElectricalLimits,
} from "../gen/quadsmith/evaluator-EvaluatorService_connectquery";
import { checkCompatibility } from "../gen/quadsmith/compatibility-CompatibilityService_connectquery";
import { createBuild, listBuilds } from "../gen/quadsmith/build-BuildService_connectquery";
import { wasmEngine, useWasmEngine } from "../lib/wasmEngine";
import { WasmLoadingIndicator } from "../components/WasmLoadingIndicator";
import { AssembledComponentsSchema } from "../gen/quadsmith/components_pb";
import {
  EvaluateComponentsRequestSchema,
  GetComponentsElectricalLimitsRequestSchema,
} from "../gen/quadsmith/evaluator_pb";
import { CheckComponentsCompatibilityRequestSchema } from "../gen/quadsmith/compatibility_pb";

// Protobuf types & services
import { BuildSchema, type Build } from "../gen/quadsmith/build_pb";
import { FrameService, type Frame } from "../gen/quadsmith/frame_pb";
import { MotorService, type Motor } from "../gen/quadsmith/motor_pb";
import { PropellerService, type Propeller } from "../gen/quadsmith/propeller_pb";
import {
  FlightControllerService,
  type FlightController,
} from "../gen/quadsmith/flight_controller_pb";
import {
  ElectronicSpeedControllerService,
  ElectronicSpeedControllerSchema,
  type ElectronicSpeedController,
} from "../gen/quadsmith/electronic_speed_controller_pb";
import { ReceiverService, ReceiverSchema, type Receiver } from "../gen/quadsmith/receiver_pb";
import {
  VideoTransmitterService,
  VideoTransmitterSchema,
  type VideoTransmitter,
} from "../gen/quadsmith/video_transmitter_pb";
import { CameraService, type Camera } from "../gen/quadsmith/camera_pb";
import { AntennaService, type Antenna } from "../gen/quadsmith/antenna_pb";
import { GpsReceiverService, type GpsReceiver } from "../gen/quadsmith/gps_receiver_pb";

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

export function isVtxAntenna(a: Antenna): boolean {
  if (a.isInternalOnly) return false;
  if (a.frequencyBandMhz >= 4000) return true;
  const nameLower = a.name.toLowerCase();
  return (
    nameLower.includes("rhcp") ||
    nameLower.includes("lhcp") ||
    nameLower.includes("5.8") ||
    nameLower.includes("vtx") ||
    nameLower.includes("lollipop") ||
    nameLower.includes("axii") ||
    nameLower.includes("singularity")
  );
}

export function BuildWizardPage() {
  const navigate = useNavigate();
  const transport = useTransport();

  useDocumentMeta({
    title: "Build Wizard — Design Custom Quadcopter | Quadsmith",
    description:
      "Interactive Quadsmith build configurator. Design your custom FPV drone, check hardware compatibility, and simulate real-time physics telemetry.",
  });

  // Current active stage (0 to 4)
  const [activeStage, setActiveStage] = useState<number>(0);

  // Stage 0: Template Selection state
  const [selectedTemplateId, setSelectedTemplateId] = useState<string | null>(null);
  const [searchTemplate, setSearchTemplate] = useState<string>("");

  // Selected Parts (Initial State: 0 parts selected)
  const [selectedFrame, setSelectedFrame] = useState<Frame | null>(null);
  const [selectedMotor, setSelectedMotor] = useState<Motor | null>(null);
  const [selectedProp, setSelectedProp] = useState<Propeller | null>(null);
  const [selectedFc, setSelectedFc] = useState<FlightController | null>(null);
  const [selectedEsc, setSelectedEsc] = useState<ElectronicSpeedController | null>(null);
  const [useIntegratedEsc, setUseIntegratedEsc] = useState<boolean>(false);
  const [selectedRx, setSelectedRx] = useState<Receiver | null>(null);
  const [useIntegratedRx, setUseIntegratedRx] = useState<boolean>(false);
  const [selectedRxAnt, setSelectedRxAnt] = useState<Antenna | null>(null);
  const [rxAntCount, setRxAntCount] = useState<number>(1);
  const [selectedGps, setSelectedGps] = useState<GpsReceiver | null>(null);

  const [selectedVtx, setSelectedVtx] = useState<VideoTransmitter | null>(null);
  const [useIntegratedVtx, setUseIntegratedVtx] = useState<boolean>(false);
  const [selectedCam, setSelectedCam] = useState<Camera | null>(null);
  const [selectedVtxAnt, setSelectedVtxAnt] = useState<Antenna | null>(null);
  const [vtxAntCount, setVtxAntCount] = useState<number>(1);

  // Explicit "None" flags for optional parts
  const [noneSelections, setNoneSelections] = useState<{
    esc?: boolean;
    rxAntenna?: boolean;
    gps?: boolean;
    vtx?: boolean;
    camera?: boolean;
    vtxAntenna?: boolean;
  }>({});

  // Search filter strings for component selectors
  const [searchFrame, setSearchFrame] = useState("");
  const [searchMotor, setSearchMotor] = useState("");
  const [searchProp, setSearchProp] = useState("");
  const [searchFc, setSearchFc] = useState("");
  const [searchEsc, setSearchEsc] = useState("");
  const [searchRx, setSearchRx] = useState("");
  const [searchRxAnt, setSearchRxAnt] = useState("");
  const [searchGps, setSearchGps] = useState("");
  const [searchVtx, setSearchVtx] = useState("");
  const [searchCam, setSearchCam] = useState("");
  const [searchVtxAnt, setSearchVtxAnt] = useState("");

  // Evaluator Sidebar runtime parameters
  const [selectedBatteryId, setSelectedBatteryId] = useState<string>("");
  const [payloadWeightG, setPayloadWeightG] = useState<number>(0);
  const [payloadInput, setPayloadInput] = useState<string>("0");

  // Review & Save fields
  const [buildName, setBuildName] = useState("My Custom Quadcopter");
  const [buildDesc, setBuildDesc] = useState(
    "Custom build configured via the Quadsmith Build Wizard.",
  );
  const [saveError, setSaveError] = useState<string | null>(null);

  // WASM Engine Integration
  const {
    isReady: isWasmReady,
    isDelayed: isWasmDelayed,
    progress: wasmProgress,
  } = useWasmEngine();

  // Intelligent compatibility filter toggle (default true)
  const [filterCompatibleOnly, setFilterCompatibleOnly] = useState<boolean>(true);

  // Dynamic Tier 1 CEL filters generated client-side by WASM Engine
  const propCelFilter = useMemo(() => {
    if (!isWasmReady || !filterCompatibleOnly || !selectedFrame) return "";
    const comps = create(AssembledComponentsSchema, { frame: selectedFrame });
    return wasmEngine.generateCelFilter("propellers", comps);
  }, [isWasmReady, filterCompatibleOnly, selectedFrame]);

  const escCelFilter = useMemo(() => {
    if (!isWasmReady || !filterCompatibleOnly || !selectedMotor) return "";
    const comps = create(AssembledComponentsSchema, { motor: selectedMotor });
    return wasmEngine.generateCelFilter("electronic_speed_controllers", comps);
  }, [isWasmReady, filterCompatibleOnly, selectedMotor]);

  const camCelFilter = useMemo(() => {
    if (!isWasmReady || !filterCompatibleOnly || !selectedVtx) return "";
    const comps = create(AssembledComponentsSchema, { videoTransmitter: selectedVtx });
    return wasmEngine.generateCelFilter("cameras", comps);
  }, [isWasmReady, filterCompatibleOnly, selectedVtx]);

  const vtxCelFilter = useMemo(() => {
    if (!isWasmReady || !filterCompatibleOnly || !selectedCam) return "";
    const comps = create(AssembledComponentsSchema, { cameras: [selectedCam] });
    return wasmEngine.generateCelFilter("video_transmitters", comps);
  }, [isWasmReady, filterCompatibleOnly, selectedCam]);

  const batteryCelFilter = useMemo(() => {
    if (!isWasmReady || !filterCompatibleOnly) return "";
    const comps = create(AssembledComponentsSchema, {
      frame: selectedFrame || undefined,
      flightController: selectedFc || undefined,
      motor: selectedMotor || undefined,
      electronicSpeedControllers: selectedEsc ? [selectedEsc] : [],
    });
    return wasmEngine.generateCelFilter("batteries", comps);
  }, [isWasmReady, filterCompatibleOnly, selectedFrame, selectedFc, selectedMotor, selectedEsc]);

  // Fetch component catalogs
  const { data: framesData } = useQuery(listFrames, { pageSize: 100 });
  const { data: motorsData } = useQuery(listMotors, { pageSize: 100 });
  const { data: propsData } = useQuery(listPropellers, {
    filter: propCelFilter || undefined,
    pageSize: 100,
  });
  const { data: fcsData } = useQuery(listFlightControllers, { pageSize: 100 });
  const { data: escsData } = useQuery(listElectronicSpeedControllers, {
    filter: escCelFilter || undefined,
    pageSize: 100,
  });
  const { data: rxsData } = useQuery(listReceivers, { pageSize: 100 });
  const { data: vtxsData } = useQuery(listVideoTransmitters, {
    filter: vtxCelFilter || undefined,
    pageSize: 100,
  });
  const { data: camsData } = useQuery(listCameras, {
    filter: camCelFilter || undefined,
    pageSize: 100,
  });
  const { data: antsData } = useQuery(listAntennas, { pageSize: 100 });
  const { data: gpsData } = useQuery(listGpsReceivers, { pageSize: 100 });
  const { data: batteriesData } = useQuery(listBatteries, {
    filter: batteryCelFilter || undefined,
    pageSize: 100,
  });
  const { data: buildsData } = useQuery(listBuilds, { pageSize: 50 });

  const frames = useMemo(() => framesData?.frames ?? [], [framesData]);
  const motors = useMemo(() => motorsData?.motors ?? [], [motorsData]);
  const props = useMemo(() => propsData?.propellers ?? [], [propsData]);
  const fcs = useMemo(() => fcsData?.flightControllers ?? [], [fcsData]);
  const escs = useMemo(() => escsData?.electronicSpeedControllers ?? [], [escsData]);
  const rxs = useMemo(() => rxsData?.receivers ?? [], [rxsData]);
  const vtxs = useMemo(() => vtxsData?.videoTransmitters ?? [], [vtxsData]);
  const cams = useMemo(() => camsData?.cameras ?? [], [camsData]);
  const ants = useMemo(() => antsData?.antennas ?? [], [antsData]);
  const gpsList = useMemo(() => gpsData?.gpsReceivers ?? [], [gpsData]);
  const batteries = useMemo(() => batteriesData?.batteries ?? [], [batteriesData]);
  const templateBuilds = useMemo(() => buildsData?.builds ?? [], [buildsData]);

  // Dynamic motor and propeller quantity determined by selected Frame (Option 2)
  const motorCount = selectedFrame?.motorCount || 4;

  // Integrated ESC on selected Flight Controller
  const integratedEsc = useMemo<ElectronicSpeedController | null>(() => {
    if (!selectedFc) return null;
    if (selectedFc.internalElectronicSpeedControllerUuid) {
      const found = escs.find((e) => e.uuid === selectedFc.internalElectronicSpeedControllerUuid);
      if (found) return found;
    }
    const nameLower = selectedFc.name.toLowerCase();
    const isAIO =
      nameLower.includes("aio") ||
      nameLower.includes("whoop") ||
      nameLower.includes("12a") ||
      nameLower.includes("20a") ||
      nameLower.includes("45a");
    if (isAIO) {
      let currentA = 20;
      if (nameLower.includes("12a")) currentA = 12;
      else if (nameLower.includes("20a")) currentA = 20;
      else if (nameLower.includes("45a")) currentA = 45;
      return create(ElectronicSpeedControllerSchema, {
        uuid: selectedFc.internalElectronicSpeedControllerUuid || `int-esc-${selectedFc.uuid}`,
        id: `integrated-esc-${selectedFc.id}`,
        manufacturer: selectedFc.manufacturer,
        name: `${selectedFc.name} Integrated ESC`,
        maxMotors: 4,
        motorCurrentMaxA: currentA,
        motorCurrentBurstA: Math.round(currentA * 1.2),
        firmware: "BLHeli_S",
        weightG: 0,
        isInternalOnly: true,
      });
    }
    return null;
  }, [selectedFc, escs]);

  const fcHasAdequateIntegratedEsc = useMemo(() => {
    if (!integratedEsc) return false;
    return !integratedEsc.maxMotors || integratedEsc.maxMotors >= motorCount;
  }, [integratedEsc, motorCount]);

  // Integrated Receiver on selected Flight Controller
  const integratedRx = useMemo<Receiver | null>(() => {
    if (!selectedFc) return null;
    if (selectedFc.internalReceiverUuid) {
      const found = rxs.find((r) => r.uuid === selectedFc.internalReceiverUuid);
      if (found) return found;
    }
    const nameLower = selectedFc.name.toLowerCase();
    const hasElrs = nameLower.includes("elrs");
    const hasFrsky = nameLower.includes("frsky");
    if (hasElrs || hasFrsky) {
      return create(ReceiverSchema, {
        uuid: selectedFc.internalReceiverUuid || `int-rx-${selectedFc.uuid}`,
        id: `integrated-rx-${selectedFc.id}`,
        manufacturer: selectedFc.manufacturer,
        name: `${selectedFc.name} Integrated ${hasElrs ? "ELRS" : "FrSky"} RX`,
        protocol: hasElrs ? "ExpressLRS" : "FrSky D16",
        frequencyBandMhz: 2400,
        hasTelemetry: true,
        weightG: 0,
        isInternalOnly: true,
      });
    }
    return null;
  }, [selectedFc, rxs]);

  const fcHasIntegratedRx = !!integratedRx;

  // Integrated VTX on selected Flight Controller
  const integratedVtx = useMemo<VideoTransmitter | null>(() => {
    if (!selectedFc) return null;
    if (selectedFc.internalVideoTransmitterUuid) {
      const found = vtxs.find((v) => v.uuid === selectedFc.internalVideoTransmitterUuid);
      if (found) return found;
    }
    const nameLower = selectedFc.name.toLowerCase();
    const hasVtx = nameLower.includes("vtx") || nameLower.includes("whoop");
    if (hasVtx) {
      return create(VideoTransmitterSchema, {
        uuid: selectedFc.internalVideoTransmitterUuid || `int-vtx-${selectedFc.uuid}`,
        id: `integrated-vtx-${selectedFc.id}`,
        manufacturer: selectedFc.manufacturer,
        name: `${selectedFc.name} Integrated VTX`,
        protocol: "Analog",
        maxPowerMw: 400,
        weightG: 0,
        isInternalOnly: true,
      });
    }
    return null;
  }, [selectedFc, vtxs]);

  const fcHasIntegratedVtx = !!integratedVtx;

  // If FC changes to one without that integrated component capability, reset
  useEffect(() => {
    if (useIntegratedEsc && !fcHasAdequateIntegratedEsc) {
      setUseIntegratedEsc(false);
      setSelectedEsc(null);
    }
  }, [fcHasAdequateIntegratedEsc, useIntegratedEsc]);

  useEffect(() => {
    if (useIntegratedRx && !fcHasIntegratedRx) {
      setUseIntegratedRx(false);
      setSelectedRx(null);
    }
  }, [fcHasIntegratedRx, useIntegratedRx]);

  useEffect(() => {
    if (useIntegratedVtx && !fcHasIntegratedVtx) {
      setUseIntegratedVtx(false);
      setSelectedVtx(null);
    }
  }, [fcHasIntegratedVtx, useIntegratedVtx]);

  // Stage completion checks (when template is selected, all stages are complete)
  const stage0Complete = true;
  const stage1Complete = !!selectedTemplateId || !!(selectedFrame && selectedMotor && selectedProp);
  const stage2Complete =
    !!selectedTemplateId ||
    !!(
      selectedFc &&
      (selectedRx || useIntegratedRx) &&
      (useIntegratedEsc || noneSelections.esc || selectedEsc)
    );
  const stage3Complete =
    !!selectedTemplateId ||
    ((!!selectedVtx || !!useIntegratedVtx || !!noneSelections.vtx) &&
      (!!selectedCam || !!noneSelections.camera) &&
      (!!selectedVtxAnt || !!noneSelections.vtxAntenna));
  const stage4Complete =
    !!selectedTemplateId || (stage1Complete && stage2Complete && buildName.trim().length > 0);

  // Unlocked stages based on gating logic
  const unlockedStages = useMemo(() => {
    if (selectedTemplateId) return [0, 1, 2, 3, 4];
    const list = [0, 1];
    if (stage1Complete) list.push(2);
    if (stage1Complete && stage2Complete) {
      list.push(3);
      list.push(4);
    }
    return list;
  }, [selectedTemplateId, stage1Complete, stage2Complete]);

  // Dry weight calculation (with dynamic motor, prop, and antenna quantities)
  const dryWeightG = useMemo(() => {
    let weight = 0;
    if (selectedFrame) weight += selectedFrame.weightG || 0;
    if (selectedMotor) weight += (selectedMotor.weightG || 0) * motorCount;
    if (selectedProp) weight += (selectedProp.weightG || 0) * motorCount;
    if (selectedFc) weight += selectedFc.weightG || 0;
    if (selectedEsc && !useIntegratedEsc && !noneSelections.esc) {
      weight += selectedEsc.weightG || 0;
    }
    if (selectedRx && !useIntegratedRx) {
      weight += selectedRx.weightG || 0;
    }
    if (selectedRxAnt && !noneSelections.rxAntenna) {
      weight += (selectedRxAnt.weightG || 0) * rxAntCount;
    }
    if (selectedGps && !noneSelections.gps) {
      weight += selectedGps.weightG || 0;
    }
    if (selectedVtx && !useIntegratedVtx && !noneSelections.vtx) {
      weight += selectedVtx.weightG || 0;
    }
    if (selectedCam && !noneSelections.camera) weight += selectedCam.weightG || 0;
    if (selectedVtxAnt && !noneSelections.vtxAntenna) {
      weight += (selectedVtxAnt.weightG || 0) * vtxAntCount;
    }
    return parseFloat(weight.toFixed(1));
  }, [
    selectedFrame,
    selectedMotor,
    selectedProp,
    selectedFc,
    selectedEsc,
    useIntegratedEsc,
    selectedRx,
    useIntegratedRx,
    selectedRxAnt,
    rxAntCount,
    selectedGps,
    selectedVtx,
    useIntegratedVtx,
    selectedCam,
    selectedVtxAnt,
    vtxAntCount,
    noneSelections,
    motorCount,
  ]);

  // Construct draft Build object for live evaluation
  const draftBuild = useMemo<Build | null>(() => {
    if (!selectedFrame || !selectedMotor) return null;

    const antennaUuids: string[] = [];
    if (selectedRxAnt && !noneSelections.rxAntenna) {
      for (let i = 0; i < rxAntCount; i++) {
        antennaUuids.push(selectedRxAnt.uuid);
      }
    }
    if (selectedVtxAnt && !noneSelections.vtxAntenna) {
      for (let i = 0; i < vtxAntCount; i++) {
        antennaUuids.push(selectedVtxAnt.uuid);
      }
    }

    return create(BuildSchema, {
      id: "draft-wizard-build",
      name: buildName || "Draft Build",
      description: buildDesc || "",
      frameUuid: selectedFrame.uuid,
      motorUuid: selectedMotor.uuid,
      propellerUuid: selectedProp ? selectedProp.uuid : "",
      flightControllerUuid: selectedFc ? selectedFc.uuid : "",
      electronicSpeedControllerUuids:
        useIntegratedEsc || noneSelections.esc || !selectedEsc ? [] : [selectedEsc.uuid],
      receiverUuids: useIntegratedRx
        ? integratedRx?.uuid
          ? [integratedRx.uuid]
          : []
        : selectedRx
          ? [selectedRx.uuid]
          : [],
      antennaUuids,
      cameraUuids: selectedCam && !noneSelections.camera ? [selectedCam.uuid] : [],
      videoTransmitterUuid: useIntegratedVtx
        ? integratedVtx?.uuid || ""
        : selectedVtx && !noneSelections.vtx
          ? selectedVtx.uuid
          : "",
      gpsReceiverUuid: selectedGps && !noneSelections.gps ? selectedGps.uuid : undefined,
      referenceLinks: [],
      media: [],
    });
  }, [
    selectedFrame,
    selectedMotor,
    selectedProp,
    selectedFc,
    selectedEsc,
    useIntegratedEsc,
    selectedRx,
    useIntegratedRx,
    integratedRx,
    selectedRxAnt,
    rxAntCount,
    selectedGps,
    selectedVtx,
    useIntegratedVtx,
    integratedVtx,
    selectedCam,
    selectedVtxAnt,
    vtxAntCount,
    noneSelections,
    buildName,
    buildDesc,
  ]);

  // Active Battery
  const activeBattery = useMemo(() => {
    return batteries.find((b) => b.id === selectedBatteryId || b.uuid === selectedBatteryId);
  }, [batteries, selectedBatteryId]);

  // Construct AssembledComponents for WASM
  const assembledComponents = useMemo(() => {
    return create(AssembledComponentsSchema, {
      frame: selectedFrame || undefined,
      motor: selectedMotor || undefined,
      propeller: selectedProp || undefined,
      battery: activeBattery || undefined,
      flightController: selectedFc || undefined,
      electronicSpeedControllers:
        selectedEsc && !useIntegratedEsc && !noneSelections.esc
          ? [selectedEsc]
          : integratedEsc
            ? [integratedEsc]
            : [],
      videoTransmitter: useIntegratedVtx
        ? integratedVtx || undefined
        : selectedVtx && !noneSelections.vtx
          ? selectedVtx
          : undefined,
      cameras: selectedCam && !noneSelections.camera ? [selectedCam] : [],
      receivers: useIntegratedRx
        ? integratedRx
          ? [integratedRx]
          : []
        : selectedRx
          ? [selectedRx]
          : [],
      antennas: [
        ...(selectedRxAnt && !noneSelections.rxAntenna
          ? Array(rxAntCount).fill(selectedRxAnt)
          : []),
        ...(selectedVtxAnt && !noneSelections.vtxAntenna
          ? Array(vtxAntCount).fill(selectedVtxAnt)
          : []),
      ],
      gpsReceiver: selectedGps && !noneSelections.gps ? selectedGps : undefined,
    });
  }, [
    selectedFrame,
    selectedMotor,
    selectedProp,
    activeBattery,
    selectedFc,
    selectedEsc,
    useIntegratedEsc,
    integratedEsc,
    selectedVtx,
    useIntegratedVtx,
    integratedVtx,
    selectedCam,
    selectedRx,
    useIntegratedRx,
    integratedRx,
    selectedRxAnt,
    rxAntCount,
    selectedVtxAnt,
    vtxAntCount,
    selectedGps,
    noneSelections,
  ]);

  // WASM Real-Time Evaluations
  const wasmEvaluation = useMemo(() => {
    if (!isWasmReady || !selectedFrame || !selectedMotor || !selectedProp || !activeBattery) {
      return null;
    }
    try {
      return wasmEngine.evaluateComponents(
        create(EvaluateComponentsRequestSchema, {
          components: assembledComponents,
          payloadWeightG,
        }),
      );
    } catch {
      return null;
    }
  }, [
    isWasmReady,
    assembledComponents,
    payloadWeightG,
    selectedFrame,
    selectedMotor,
    selectedProp,
    activeBattery,
  ]);

  const wasmElectricalLimits = useMemo(() => {
    if (!isWasmReady || (!selectedFc && !selectedMotor && !selectedEsc)) {
      return null;
    }
    try {
      return wasmEngine.computeElectricalLimits(
        create(GetComponentsElectricalLimitsRequestSchema, {
          components: assembledComponents,
          candidateBatteries: batteries,
        }),
      );
    } catch {
      return null;
    }
  }, [isWasmReady, assembledComponents, batteries, selectedFc, selectedMotor, selectedEsc]);

  const wasmCompatibility = useMemo(() => {
    if (!isWasmReady || !selectedFrame) {
      return null;
    }
    try {
      return wasmEngine.checkCompatibility(
        create(CheckComponentsCompatibilityRequestSchema, {
          components: assembledComponents,
        }),
      );
    } catch {
      return null;
    }
  }, [isWasmReady, assembledComponents, selectedFrame]);

  // Evaluator queries (fallback if WASM is not yet ready)
  const { data: queryElectricalLimits } = useQuery(
    getBuildElectricalLimits,
    { buildSource: { case: "build", value: draftBuild! } },
    { enabled: !!draftBuild && !wasmElectricalLimits },
  );

  const electricalLimits = wasmElectricalLimits || queryElectricalLimits;

  // Auto-select lightest compatible battery when limits arrive or batteries change
  useEffect(() => {
    if (!selectedBatteryId && batteries.length > 0) {
      if (electricalLimits?.defaultBatteryId) {
        setSelectedBatteryId(electricalLimits.defaultBatteryId);
      } else {
        setSelectedBatteryId(batteries[0].id || batteries[0].uuid);
      }
    }
  }, [electricalLimits, batteries, selectedBatteryId]);

  const { data: queryEvaluation } = useQuery(
    evaluateBuild,
    {
      buildSource: { case: "build", value: draftBuild! },
      payloadWeightG,
      batteryId: selectedBatteryId,
    },
    {
      enabled: !!draftBuild && !!selectedBatteryId && !wasmEvaluation,
    },
  );

  const evaluation = wasmEvaluation || queryEvaluation;

  const { data: queryCompatibilityData } = useQuery(
    checkCompatibility,
    { build: draftBuild! },
    { enabled: !!draftBuild && !wasmCompatibility },
  );

  const compatibilityData = wasmCompatibility || queryCompatibilityData;

  // Mutation for creating the build
  const { mutateAsync: saveBuildMutation, isPending: isSaving } = useMutation(createBuild);

  // Filtered component catalogs based on search inputs
  const filteredTemplates = useMemo(() => {
    if (!searchTemplate.trim()) return templateBuilds;
    const q = searchTemplate.toLowerCase();
    return templateBuilds.filter(
      (b) => b.name.toLowerCase().includes(q) || b.description.toLowerCase().includes(q),
    );
  }, [templateBuilds, searchTemplate]);

  const filteredFrames = useMemo(() => {
    if (!searchFrame.trim()) return frames;
    const q = searchFrame.toLowerCase();
    return frames.filter(
      (f) => f.name.toLowerCase().includes(q) || f.manufacturer.toLowerCase().includes(q),
    );
  }, [frames, searchFrame]);

  const filteredMotors = useMemo(() => {
    if (!searchMotor.trim()) return motors;
    const q = searchMotor.toLowerCase();
    return motors.filter(
      (m) =>
        m.name.toLowerCase().includes(q) ||
        m.manufacturer.toLowerCase().includes(q) ||
        String(m.kv).includes(q),
    );
  }, [motors, searchMotor]);

  const filteredProps = useMemo(() => {
    let list = props;
    if (filterCompatibleOnly && selectedFrame && selectedFrame.maxPropSizeMm > 0) {
      list = list.filter((p) => p.diameterMm <= selectedFrame.maxPropSizeMm);
    }
    if (!searchProp.trim()) return list;
    const q = searchProp.toLowerCase();
    return list.filter(
      (p) => p.name.toLowerCase().includes(q) || p.manufacturer.toLowerCase().includes(q),
    );
  }, [props, filterCompatibleOnly, selectedFrame, searchProp]);

  const filteredFcs = useMemo(() => {
    if (!searchFc.trim()) return fcs;
    const q = searchFc.toLowerCase();
    return fcs.filter(
      (fc) => fc.name.toLowerCase().includes(q) || fc.manufacturer.toLowerCase().includes(q),
    );
  }, [fcs, searchFc]);

  const filteredEscs = useMemo(() => {
    let external = escs.filter((esc) => !esc.isInternalOnly);
    if (filterCompatibleOnly && selectedMotor && selectedMotor.statorDiameterMm > 20) {
      external = external.filter((esc) => esc.motorCurrentMaxA >= 20);
    }
    if (!searchEsc.trim()) return external;
    const q = searchEsc.toLowerCase();
    return external.filter(
      (esc) => esc.name.toLowerCase().includes(q) || esc.manufacturer.toLowerCase().includes(q),
    );
  }, [escs, filterCompatibleOnly, selectedMotor, searchEsc]);

  const filteredRxs = useMemo(() => {
    const external = rxs.filter((rx) => !rx.isInternalOnly);
    if (!searchRx.trim()) return external;
    const q = searchRx.toLowerCase();
    return external.filter(
      (rx) =>
        rx.name.toLowerCase().includes(q) ||
        rx.manufacturer.toLowerCase().includes(q) ||
        rx.protocol.toLowerCase().includes(q),
    );
  }, [rxs, searchRx]);

  const filteredVtxs = useMemo(() => {
    let external = vtxs.filter((vtx) => !vtx.isInternalOnly);
    if (filterCompatibleOnly && selectedCam && selectedCam.protocol) {
      external = external.filter(
        (v) => v.protocol.toLowerCase() === selectedCam.protocol.toLowerCase(),
      );
    }
    if (!searchVtx.trim()) return external;
    const q = searchVtx.toLowerCase();
    return external.filter(
      (v) => v.name.toLowerCase().includes(q) || v.manufacturer.toLowerCase().includes(q),
    );
  }, [vtxs, filterCompatibleOnly, selectedCam, searchVtx]);

  const filteredCams = useMemo(() => {
    let list = cams;
    if (filterCompatibleOnly && selectedVtx && selectedVtx.protocol) {
      list = list.filter((c) => c.protocol.toLowerCase() === selectedVtx.protocol.toLowerCase());
    }
    if (!searchCam.trim()) return list;
    const q = searchCam.toLowerCase();
    return list.filter(
      (c) => c.name.toLowerCase().includes(q) || c.manufacturer.toLowerCase().includes(q),
    );
  }, [cams, filterCompatibleOnly, selectedVtx, searchCam]);

  const filteredRxAnts = useMemo(() => {
    const rxAnts = ants.filter(isRxAntenna);
    if (!searchRxAnt.trim()) return rxAnts;
    const q = searchRxAnt.toLowerCase();
    return rxAnts.filter(
      (a) =>
        a.name.toLowerCase().includes(q) ||
        a.manufacturer.toLowerCase().includes(q) ||
        a.connector.toLowerCase().includes(q) ||
        a.polarization.toLowerCase().includes(q),
    );
  }, [ants, searchRxAnt]);

  const filteredVtxAnts = useMemo(() => {
    const vtxAnts = ants.filter(isVtxAntenna);
    if (!searchVtxAnt.trim()) return vtxAnts;
    const q = searchVtxAnt.toLowerCase();
    return vtxAnts.filter(
      (a) =>
        a.name.toLowerCase().includes(q) ||
        a.manufacturer.toLowerCase().includes(q) ||
        a.connector.toLowerCase().includes(q) ||
        a.polarization.toLowerCase().includes(q),
    );
  }, [ants, searchVtxAnt]);

  const filteredGps = useMemo(() => {
    if (!searchGps.trim()) return gpsList;
    const q = searchGps.toLowerCase();
    return gpsList.filter(
      (g) => g.name.toLowerCase().includes(q) || g.manufacturer.toLowerCase().includes(q),
    );
  }, [gpsList, searchGps]);

  // Payload Handlers
  const handlePayloadInput = (val: string) => {
    setPayloadInput(val);
    const parsed = parseFloat(val);
    if (!isNaN(parsed) && parsed >= 0) {
      setPayloadWeightG(parsed);
    }
  };

  const adjustPayload = (delta: number) => {
    const next = Math.max(0, payloadWeightG + delta);
    setPayloadWeightG(next);
    setPayloadInput(String(next));
  };

  const setPresetPayload = (weight: number) => {
    setPayloadWeightG(weight);
    setPayloadInput(String(weight));
  };

  // Skip All Stage 3 (Video)
  const skipAllStage3 = () => {
    setNoneSelections((prev) => ({
      ...prev,
      vtx: true,
      camera: true,
      vtxAntenna: true,
    }));
    setSelectedVtx(null);
    setSelectedCam(null);
    setSelectedVtxAnt(null);
  };

  // Apply build template
  const applyTemplate = async (build: Build) => {
    setSelectedTemplateId(build.id || build.uuid);
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

    // Flight Controller
    let matchedFc: FlightController | null = null;
    if (build.flightControllerUuid) {
      let fc = fcs.find((x) => x.uuid === build.flightControllerUuid);
      if (!fc) {
        try {
          const client = createClient(FlightControllerService, transport);
          fc = await client.getFlightController({ id: build.flightControllerUuid });
        } catch (e) {
          console.error("Failed to fetch template FC:", e);
        }
      }
      if (fc) {
        matchedFc = fc;
        setSelectedFc(fc);
      }
    }

    // ESC
    if (build.electronicSpeedControllerUuids && build.electronicSpeedControllerUuids.length > 0) {
      const targetEscUuid = build.electronicSpeedControllerUuids[0];
      let esc = escs.find((x) => x.uuid === targetEscUuid);
      if (!esc) {
        try {
          const client = createClient(ElectronicSpeedControllerService, transport);
          esc = await client.getElectronicSpeedController({ id: targetEscUuid });
        } catch (e) {
          console.error("Failed to fetch template ESC:", e);
        }
      }
      if (esc) {
        setSelectedEsc(esc);
        setUseIntegratedEsc(false);
        setNoneSelections((prev) => ({ ...prev, esc: false }));
      }
    } else if (matchedFc) {
      setUseIntegratedEsc(true);
      setSelectedEsc(null);
      setNoneSelections((prev) => ({ ...prev, esc: false }));
    }

    // Receiver
    if (build.receiverUuids && build.receiverUuids.length > 0) {
      const targetRxUuid = build.receiverUuids[0];
      let rx = rxs.find((x) => x.uuid === targetRxUuid);
      if (!rx) {
        try {
          const client = createClient(ReceiverService, transport);
          rx = await client.getReceiver({ id: targetRxUuid });
        } catch (e) {
          console.error("Failed to fetch template RX:", e);
        }
      }
      if (rx) {
        setSelectedRx(rx);
        setUseIntegratedRx(false);
      }
    } else if (matchedFc) {
      setUseIntegratedRx(true);
      setSelectedRx(null);
    }

    // Antennas
    if (build.antennaUuids && build.antennaUuids.length > 0) {
      const fetchedAnts: Antenna[] = [];
      for (const u of build.antennaUuids) {
        let a = ants.find((x) => x.uuid === u);
        if (!a) {
          try {
            const client = createClient(AntennaService, transport);
            a = await client.getAntenna({ id: u });
          } catch (e) {
            console.error("Failed to fetch template antenna:", e);
          }
        }
        if (a) fetchedAnts.push(a);
      }

      const rxAnt = fetchedAnts.find(isRxAntenna);
      if (rxAnt) {
        setSelectedRxAnt(rxAnt);
        const count = build.antennaUuids.filter((u) => u === rxAnt.uuid).length;
        setRxAntCount(count > 1 ? 2 : 1);
        setNoneSelections((prev) => ({ ...prev, rxAntenna: false }));
      } else {
        setNoneSelections((prev) => ({ ...prev, rxAntenna: true }));
      }

      const vtxAnt = fetchedAnts.find(isVtxAntenna);
      if (vtxAnt) {
        setSelectedVtxAnt(vtxAnt);
        const count = build.antennaUuids.filter((u) => u === vtxAnt.uuid).length;
        setVtxAntCount(count > 1 ? 2 : 1);
        setNoneSelections((prev) => ({ ...prev, vtxAntenna: false }));
      } else {
        setNoneSelections((prev) => ({ ...prev, vtxAntenna: true }));
      }
    } else {
      setNoneSelections((prev) => ({ ...prev, rxAntenna: true, vtxAntenna: true }));
    }

    // GPS
    if (build.gpsReceiverUuid) {
      let gps = gpsList.find((x) => x.uuid === build.gpsReceiverUuid);
      if (!gps) {
        try {
          const client = createClient(GpsReceiverService, transport);
          gps = await client.getGpsReceiver({ id: build.gpsReceiverUuid });
        } catch (e) {
          console.error("Failed to fetch template GPS:", e);
        }
      }
      if (gps) {
        setSelectedGps(gps);
        setNoneSelections((prev) => ({ ...prev, gps: false }));
      }
    } else {
      setNoneSelections((prev) => ({ ...prev, gps: true }));
    }

    // VTX
    if (build.videoTransmitterUuid) {
      let vtx = vtxs.find((x) => x.uuid === build.videoTransmitterUuid);
      if (!vtx) {
        try {
          const client = createClient(VideoTransmitterService, transport);
          vtx = await client.getVideoTransmitter({ id: build.videoTransmitterUuid });
        } catch (e) {
          console.error("Failed to fetch template VTX:", e);
        }
      }
      if (vtx) {
        setSelectedVtx(vtx);
        setUseIntegratedVtx(false);
        setNoneSelections((prev) => ({ ...prev, vtx: false }));
      }
    } else {
      setNoneSelections((prev) => ({ ...prev, vtx: true }));
    }

    // Camera
    if (build.cameraUuids && build.cameraUuids.length > 0) {
      const targetCamUuid = build.cameraUuids[0];
      let cam = cams.find((x) => x.uuid === targetCamUuid);
      if (!cam) {
        try {
          const client = createClient(CameraService, transport);
          cam = await client.getCamera({ id: targetCamUuid });
        } catch (e) {
          console.error("Failed to fetch template camera:", e);
        }
      }
      if (cam) {
        setSelectedCam(cam);
        setNoneSelections((prev) => ({ ...prev, camera: false }));
      }
    } else {
      setNoneSelections((prev) => ({ ...prev, camera: true }));
    }
  };

  // Select Scratch (blank canvas)
  const selectScratch = () => {
    setSelectedTemplateId(null);
    setSelectedFrame(null);
    setSelectedMotor(null);
    setSelectedProp(null);
    setSelectedFc(null);
    setSelectedEsc(null);
    setUseIntegratedEsc(false);
    setSelectedRx(null);
    setUseIntegratedRx(false);
    setSelectedRxAnt(null);
    setRxAntCount(1);
    setSelectedGps(null);
    setSelectedVtx(null);
    setUseIntegratedVtx(false);
    setSelectedCam(null);
    setSelectedVtxAnt(null);
    setVtxAntCount(1);
    setNoneSelections({});
  };

  // Reset all selections to zero and return to Stage 0
  const resetWizard = () => {
    selectScratch();
    setSearchTemplate("");
    setSelectedBatteryId("");
    setPayloadWeightG(0);
    setPayloadInput("0");
    setActiveStage(0);
    setBuildName("My Custom Quadcopter");
    setBuildDesc("Custom build configured via the Quadsmith Build Wizard.");
  };

  // Save Build to PostgreSQL via CreateBuild RPC
  const handleSaveBuild = async () => {
    if (!stage1Complete || !stage2Complete) {
      setSaveError("Please complete required components in Stages 1 and 2 before saving.");
      return;
    }
    if (!buildName.trim()) {
      setSaveError("Build name is required.");
      return;
    }

    try {
      setSaveError(null);
      const antennaUuids: string[] = [];
      if (selectedRxAnt && !noneSelections.rxAntenna) {
        for (let i = 0; i < rxAntCount; i++) {
          antennaUuids.push(selectedRxAnt.uuid);
        }
      }
      if (selectedVtxAnt && !noneSelections.vtxAntenna) {
        for (let i = 0; i < vtxAntCount; i++) {
          antennaUuids.push(selectedVtxAnt.uuid);
        }
      }

      const newBuild = create(BuildSchema, {
        name: buildName.trim(),
        description: buildDesc.trim(),
        frameUuid: selectedFrame!.uuid,
        motorUuid: selectedMotor!.uuid,
        propellerUuid: selectedProp!.uuid,
        flightControllerUuid: selectedFc!.uuid,
        electronicSpeedControllerUuids:
          useIntegratedEsc || noneSelections.esc || !selectedEsc ? [] : [selectedEsc.uuid],
        receiverUuids: useIntegratedRx
          ? integratedRx?.uuid
            ? [integratedRx.uuid]
            : []
          : selectedRx
            ? [selectedRx.uuid]
            : [],
        cameraUuids: selectedCam && !noneSelections.camera ? [selectedCam.uuid] : [],
        antennaUuids,
        videoTransmitterUuid: useIntegratedVtx
          ? integratedVtx?.uuid || ""
          : selectedVtx && !noneSelections.vtx
            ? selectedVtx.uuid
            : "",
        gpsReceiverUuid: selectedGps && !noneSelections.gps ? selectedGps.uuid : undefined,
      });

      const res = await saveBuildMutation({ build: newBuild });
      if (res && (res.id || res.uuid)) {
        navigate(`/builds/${res.id || res.uuid}`);
      }
    } catch (err: unknown) {
      setSaveError(err instanceof Error ? err.message : "Failed to save build to database");
    }
  };

  // Navigation between stages
  const handleNextStage = () => {
    if (activeStage === 0) setActiveStage(1);
    else if (activeStage === 1 && stage1Complete) setActiveStage(2);
    else if (activeStage === 2 && stage2Complete) setActiveStage(3);
    else if (activeStage === 3) setActiveStage(4);
  };

  const handlePrevStage = () => {
    if (activeStage > 0) setActiveStage((prev) => prev - 1);
  };

  return (
    <div className="max-w-7xl mx-auto space-y-6">
      {/* Top Banner */}
      <div className="flex flex-col md:flex-row md:items-center justify-between gap-4 pb-4 border-b border-zinc-200 dark:border-zinc-800">
        <div>
          <div className="flex items-center gap-2 mb-1">
            <span className="inline-flex items-center gap-1.5 px-2.5 py-0.5 rounded-full text-xs font-semibold bg-blue-500/10 text-blue-600 dark:text-blue-400 border border-blue-500/20">
              <Sparkles size={14} />
              Interactive Build Wizard
            </span>
            <span className="text-xs text-zinc-500 dark:text-zinc-400">
              • Stage {activeStage} of 4
            </span>
          </div>
          <h1 className="text-2xl sm:text-3xl font-extrabold tracking-tight text-zinc-900 dark:text-zinc-50">
            Design Custom Drone
          </h1>
          <p className="text-xs sm:text-sm text-zinc-500 dark:text-zinc-400 mt-1">
            Select parts sequentially. Flight electronics include FC, ESC, RX, Antenna &amp; GPS.
            Video includes VTX, Camera &amp; Antenna. Test flight physics and battery options in the
            Live Evaluator on the right.
          </p>
        </div>
      </div>

      {/* 5-Stage Stepper Navigation */}
      <WizardStageBar
        currentStage={activeStage}
        unlockedStages={unlockedStages}
        stageCompletion={{
          0: stage0Complete,
          1: stage1Complete,
          2: stage2Complete,
          3: stage3Complete,
          4: stage4Complete,
        }}
        stageProgressText={{
          0: selectedTemplateId ? "Template ✓" : "Scratch ✓",
          1: `${(selectedFrame ? 1 : 0) + (selectedMotor ? 1 : 0) + (selectedProp ? 1 : 0)}/3`,
          2: `${(selectedFc ? 1 : 0) + (selectedRx || useIntegratedRx ? 1 : 0) + (selectedEsc || useIntegratedEsc || noneSelections.esc ? 1 : 0)}/3`,
          3: "Optional",
          4: "Finalize",
        }}
        onSelectStage={(stage) => setActiveStage(stage)}
      />

      {/* Main Workspace: Left 8 Cols (Stage Workarea) + Right 4 Cols (Live Evaluator) */}
      <div className="grid grid-cols-1 lg:grid-cols-12 gap-6 items-start">
        {/* Left Column (Stage Contents) */}
        <div className="lg:col-span-8 space-y-5">
          {/* Stage 0 Banner */}
          {activeStage === 0 && (
            <div className="p-3.5 rounded-xl border border-blue-500/20 bg-blue-50/50 dark:bg-blue-950/20 text-blue-900 dark:text-blue-300 flex items-start gap-3 text-xs">
              <Sparkles size={16} className="text-blue-500 mt-0.5 shrink-0" />
              <div>
                <span className="font-bold">
                  {selectedTemplateId
                    ? `Stage 0: Template Selected — ${templateBuilds.find((b) => (b.id || b.uuid) === selectedTemplateId)?.name || "Template"}`
                    : "Stage 0: Starting from Scratch (Blank Canvas)"}
                </span>
                <p className="opacity-90 mt-0.5">
                  {selectedTemplateId
                    ? "All compatible components pre-populated across all stages. You can customize them in Stages 1-3 or review now."
                    : "You are designing from a clean slate. Click Next to establish your airframe & propulsion in Stage 1."}
                </p>
              </div>
            </div>
          )}

          {/* STAGE 0: Template Selection */}
          {activeStage === 0 && (
            <div className="space-y-4">
              <div className="p-4 rounded-2xl border border-zinc-200 dark:border-zinc-800 bg-white dark:bg-zinc-900 space-y-3">
                <div className="flex items-center justify-between">
                  <div className="flex items-center gap-2">
                    <span className="w-2 h-2 rounded-full bg-blue-500"></span>
                    <h3 className="font-bold text-sm text-zinc-900 dark:text-zinc-100">
                      Choose Starting Baseline
                    </h3>
                    <span className="text-[10px] font-bold uppercase tracking-wider px-2 py-0.5 rounded bg-blue-500/10 text-blue-600 dark:text-blue-400 border border-blue-500/20">
                      Stage 0
                    </span>
                  </div>
                  <span className="text-xs font-semibold text-blue-600 dark:text-blue-400">
                    {selectedTemplateId
                      ? `Template: ${templateBuilds.find((b) => (b.id || b.uuid) === selectedTemplateId)?.name || "Selected"}`
                      : "Start from Scratch"}
                  </span>
                </div>

                <p className="text-xs text-zinc-500 dark:text-zinc-400">
                  Select whether you want to build from a blank slate or begin from a proven
                  existing quadcopter build. You can freely customize or swap any part in the
                  following stages.
                </p>

                {/* Search Input */}
                <div className="relative">
                  <Search
                    size={14}
                    className="absolute left-3 top-1/2 -translate-y-1/2 text-zinc-400"
                  />
                  <input
                    type="text"
                    value={searchTemplate}
                    onChange={(e) => setSearchTemplate(e.target.value)}
                    placeholder="Filter templates by build name, frame, style, or specs..."
                    className="w-full bg-zinc-50 dark:bg-zinc-950 border border-zinc-200 dark:border-zinc-800 rounded-xl pl-9 pr-8 py-1.5 text-xs text-zinc-900 dark:text-zinc-100 placeholder-zinc-400 focus:outline-none focus:ring-1 focus:ring-blue-500"
                  />
                  {searchTemplate && (
                    <button
                      type="button"
                      onClick={() => setSearchTemplate("")}
                      className="absolute right-2.5 top-1/2 -translate-y-1/2 text-zinc-400 hover:text-zinc-600"
                    >
                      <X size={13} />
                    </button>
                  )}
                </div>

                {/* Scrollable Templates List with Limited Height */}
                <div className="overflow-y-auto max-h-72 sm:max-h-80 space-y-2.5 pr-1.5 focus:outline-none">
                  {/* Option 1: Start from Scratch (Always First) */}
                  {(!searchTemplate ||
                    "start from scratch blank canvas full custom".includes(
                      searchTemplate.toLowerCase(),
                    )) && (
                    <div
                      onClick={selectScratch}
                      className={`p-3.5 rounded-xl border cursor-pointer transition-all flex flex-col sm:flex-row sm:items-center justify-between gap-3 ${
                        !selectedTemplateId
                          ? "border-blue-500 bg-blue-50/50 dark:bg-blue-950/30 ring-1 ring-blue-500 shadow-xs"
                          : "border-zinc-200 dark:border-zinc-800 bg-zinc-50/50 dark:bg-zinc-900/60 hover:border-zinc-400 dark:hover:border-zinc-700"
                      }`}
                    >
                      <div className="flex-1 min-w-0">
                        <div className="flex items-center gap-2 flex-wrap">
                          <span className="text-xs sm:text-sm font-bold text-zinc-900 dark:text-zinc-100 truncate">
                            Start from Scratch
                          </span>
                          <span className="text-[10px] px-2 py-0.5 rounded-full font-semibold bg-blue-500/10 text-blue-600 dark:text-blue-400 border border-blue-500/20 shrink-0">
                            Blank Canvas
                          </span>
                          <span className="text-[10px] px-2 py-0.5 rounded-full font-semibold bg-zinc-200 dark:bg-zinc-800 text-zinc-700 dark:text-zinc-300 shrink-0">
                            Full Custom
                          </span>
                        </div>
                        <div className="text-[11px] text-zinc-500 dark:text-zinc-400 mt-1">
                          Design your build from the ground up. Hand-pick each frame, motor, prop,
                          electronic, and video component.
                        </div>
                      </div>
                      <div className="flex items-center gap-2 self-end sm:self-center shrink-0">
                        <span
                          className={`text-[10px] uppercase font-bold tracking-wide px-2.5 py-1 rounded ${
                            !selectedTemplateId
                              ? "bg-blue-600 text-white"
                              : "bg-zinc-200 dark:bg-zinc-800 text-zinc-600 dark:text-zinc-400"
                          }`}
                        >
                          {!selectedTemplateId ? "Selected (Default)" : "Select Scratch"}
                        </span>
                        <div className="w-4 h-4 rounded-full border border-blue-500 flex items-center justify-center">
                          {!selectedTemplateId && (
                            <div className="w-2 h-2 rounded-full bg-blue-500" />
                          )}
                        </div>
                      </div>
                    </div>
                  )}

                  {/* Existing Builds as Templates */}
                  {filteredTemplates.map((template) => {
                    const isSelected = selectedTemplateId === (template.id || template.uuid);
                    const frameObj = frames.find((f) => f.uuid === template.frameUuid);
                    const motorObj = motors.find((m) => m.uuid === template.motorUuid);
                    const fcObj = fcs.find((f) => f.uuid === template.flightControllerUuid);

                    const summaryParts = [frameObj?.name, motorObj?.name, fcObj?.name].filter(
                      Boolean,
                    );

                    return (
                      <div
                        key={template.id || template.uuid}
                        onClick={() => applyTemplate(template)}
                        className={`p-3.5 rounded-xl border cursor-pointer transition-all flex flex-col sm:flex-row sm:items-center justify-between gap-3 ${
                          isSelected
                            ? "border-blue-500 bg-blue-50/50 dark:bg-blue-950/30 ring-1 ring-blue-500 shadow-xs"
                            : "border-zinc-200 dark:border-zinc-800 bg-zinc-50/50 dark:bg-zinc-900/60 hover:border-zinc-400 dark:hover:border-zinc-700"
                        }`}
                      >
                        <div className="flex-1 min-w-0">
                          <div className="flex items-center gap-2 flex-wrap">
                            <span className="text-xs sm:text-sm font-bold text-zinc-900 dark:text-zinc-100 truncate">
                              {template.name}
                            </span>
                            {frameObj?.geometry && (
                              <span className="text-[10px] px-2 py-0.5 rounded-full font-semibold bg-purple-500/10 text-purple-600 dark:text-purple-400 border border-purple-500/20 shrink-0">
                                {frameObj.geometry}
                              </span>
                            )}
                          </div>
                          {summaryParts.length > 0 && (
                            <div className="text-[11px] text-zinc-500 dark:text-zinc-400 mt-1 truncate">
                              {summaryParts.join(" • ")}
                            </div>
                          )}
                          {template.description && (
                            <div className="text-[10px] text-zinc-400 dark:text-zinc-500 mt-0.5 line-clamp-1">
                              {template.description}
                            </div>
                          )}
                        </div>
                        <div className="flex items-center gap-2 self-end sm:self-center shrink-0">
                          <span
                            className={`text-[10px] uppercase font-bold tracking-wide px-2.5 py-1 rounded ${
                              isSelected
                                ? "bg-blue-600 text-white"
                                : "bg-zinc-200 dark:bg-zinc-800 text-zinc-600 dark:text-zinc-400"
                            }`}
                          >
                            {isSelected ? "Template Active" : "Use Template"}
                          </span>
                          <div className="w-4 h-4 rounded-full border border-blue-500 flex items-center justify-center">
                            {isSelected && <div className="w-2 h-2 rounded-full bg-blue-500" />}
                          </div>
                        </div>
                      </div>
                    );
                  })}

                  {filteredTemplates.length === 0 && searchTemplate && (
                    <p className="text-xs text-zinc-400 italic py-2 text-center">
                      No templates matching "{searchTemplate}".
                    </p>
                  )}
                </div>
              </div>
            </div>
          )}
          {/* Stage Gating Banner */}
          {activeStage === 1 && !stage1Complete && (
            <div className="p-3.5 rounded-xl border border-amber-500/30 bg-amber-500/10 text-amber-800 dark:text-amber-300 flex items-start gap-3 text-xs">
              <AlertCircle size={16} className="text-amber-500 mt-0.5 shrink-0" />
              <div>
                <span className="font-bold">
                  Stage 1 Incomplete: Select Frame, Motors, and Propellers to establish propulsion
                  base
                </span>
                <p className="opacity-90 mt-0.5">
                  The Frame sets prop clearance and stack mounting. Motors & Props set thrust and
                  ESC current requirements for Stage 2.
                </p>
              </div>
            </div>
          )}

          {activeStage === 2 && !stage2Complete && (
            <div className="p-3.5 rounded-xl border border-amber-500/30 bg-amber-500/10 text-amber-800 dark:text-amber-300 flex items-start gap-3 text-xs">
              <AlertCircle size={16} className="text-amber-500 mt-0.5 shrink-0" />
              <div>
                <span className="font-bold">
                  Stage 2 Incomplete: Flight Controller, ESC, and Radio Receiver are required
                </span>
                <p className="opacity-90 mt-0.5">
                  External ESC is optional only if your flight controller has an integrated ESC with
                  enough motor drivers.
                </p>
              </div>
            </div>
          )}

          {/* STAGE 1: Airframe & Propulsion */}
          {activeStage === 1 && (
            <div className="space-y-6">
              {/* 1A. Frame Chassis */}
              <div className="p-4 rounded-2xl border border-zinc-200 dark:border-zinc-800 bg-white dark:bg-zinc-900 space-y-3">
                <div className="flex items-center justify-between">
                  <div className="flex items-center gap-2">
                    <span className="w-2 h-2 rounded-full bg-blue-500"></span>
                    <h3 className="font-bold text-sm text-zinc-900 dark:text-zinc-100">
                      1A. Frame Chassis
                    </h3>
                  </div>
                  {selectedFrame ? (
                    <span className="text-xs font-semibold text-emerald-600 dark:text-emerald-400 flex items-center gap-1">
                      <Check size={12} strokeWidth={2.5} /> Selected
                    </span>
                  ) : (
                    <span className="text-xs font-semibold text-red-500 dark:text-red-400">
                      Required
                    </span>
                  )}
                </div>

                {/* In-line Search Box */}
                <div className="relative">
                  <Search
                    size={14}
                    className="absolute left-3 top-1/2 -translate-y-1/2 text-zinc-400"
                  />
                  <input
                    type="text"
                    value={searchFrame}
                    onChange={(e) => setSearchFrame(e.target.value)}
                    placeholder="Search frames by name, brand, geometry..."
                    className="w-full bg-zinc-50 dark:bg-zinc-950 border border-zinc-200 dark:border-zinc-800 rounded-xl pl-9 pr-8 py-1.5 text-xs text-zinc-900 dark:text-zinc-100 placeholder-zinc-400 focus:outline-none focus:ring-1 focus:ring-blue-500"
                  />
                  {searchFrame && (
                    <button
                      type="button"
                      onClick={() => setSearchFrame("")}
                      className="absolute right-2.5 top-1/2 -translate-y-1/2 text-zinc-400 hover:text-zinc-600"
                    >
                      <X size={13} />
                    </button>
                  )}
                </div>

                {/* Products List */}
                <div className="overflow-y-auto max-h-64 sm:max-h-72 space-y-2 pr-1.5 focus:outline-none">
                  {filteredFrames.map((f) => {
                    const isSelected = selectedFrame?.uuid === f.uuid || selectedFrame?.id === f.id;
                    const propSize = f.maxPropSizeMm
                      ? `${(f.maxPropSizeMm / 25.4).toFixed(1)}"`
                      : '5.1"';
                    return (
                      <div
                        key={f.uuid || f.id}
                        onClick={() => setSelectedFrame(f)}
                        className={`p-3 rounded-xl border cursor-pointer transition-all flex flex-col sm:flex-row sm:items-center justify-between gap-2 ${
                          isSelected
                            ? "border-blue-500 bg-blue-50/50 dark:bg-blue-950/30 ring-1 ring-blue-500"
                            : "border-zinc-200 dark:border-zinc-800 bg-zinc-50/50 dark:bg-zinc-900/60 hover:border-zinc-400 dark:hover:border-zinc-700"
                        }`}
                      >
                        <div className="flex-1 min-w-0">
                          <div className="flex items-center gap-2 flex-wrap">
                            <span className="text-xs font-bold text-zinc-900 dark:text-zinc-100 truncate">
                              {f.name}
                            </span>
                            <span className="text-[10px] px-2 py-0.5 rounded-full font-semibold bg-purple-500/10 text-purple-600 dark:text-purple-400 border border-purple-500/20 shrink-0">
                              {f.motorCount || 4}x Motors
                            </span>
                          </div>
                          <div className="text-[11px] text-zinc-500 dark:text-zinc-400 mt-0.5">
                            {f.manufacturer} • Weight: {f.weightG}g • {f.geometry || "Standard"}
                          </div>
                        </div>
                        <div className="text-[11px] text-blue-600 dark:text-blue-400 font-mono shrink-0">
                          Max Prop: {propSize}
                        </div>
                      </div>
                    );
                  })}
                </div>
                {filteredFrames.length === 0 && (
                  <p className="text-xs text-zinc-400 italic py-2 text-center">
                    No compatible frames matching your search.
                  </p>
                )}
              </div>

              {/* 1B. Motors */}
              <div className="p-4 rounded-2xl border border-zinc-200 dark:border-zinc-800 bg-white dark:bg-zinc-900 space-y-3">
                <div className="flex items-center justify-between">
                  <div className="flex items-center gap-2">
                    <span className="w-2 h-2 rounded-full bg-blue-500"></span>
                    <h3 className="font-bold text-sm text-zinc-900 dark:text-zinc-100">
                      1B. Motors{" "}
                      <span className="text-xs text-blue-600 dark:text-blue-400 font-mono font-semibold">
                        ({motorCount}x)
                      </span>
                    </h3>
                  </div>
                  {selectedMotor ? (
                    <span className="text-xs font-semibold text-emerald-600 dark:text-emerald-400 flex items-center gap-1">
                      <Check size={12} strokeWidth={2.5} /> Selected
                    </span>
                  ) : (
                    <span className="text-xs font-semibold text-red-500 dark:text-red-400">
                      Required
                    </span>
                  )}
                </div>

                {/* In-line Search Box */}
                <div className="relative">
                  <Search
                    size={14}
                    className="absolute left-3 top-1/2 -translate-y-1/2 text-zinc-400"
                  />
                  <input
                    type="text"
                    value={searchMotor}
                    onChange={(e) => setSearchMotor(e.target.value)}
                    placeholder="Search motors by KV, stator size, manufacturer..."
                    className="w-full bg-zinc-50 dark:bg-zinc-950 border border-zinc-200 dark:border-zinc-800 rounded-xl pl-9 pr-8 py-1.5 text-xs text-zinc-900 dark:text-zinc-100 placeholder-zinc-400 focus:outline-none focus:ring-1 focus:ring-blue-500"
                  />
                  {searchMotor && (
                    <button
                      type="button"
                      onClick={() => setSearchMotor("")}
                      className="absolute right-2.5 top-1/2 -translate-y-1/2 text-zinc-400 hover:text-zinc-600"
                    >
                      <X size={13} />
                    </button>
                  )}
                </div>

                <div className="overflow-y-auto max-h-64 sm:max-h-72 space-y-2 pr-1.5 focus:outline-none">
                  {filteredMotors.map((m) => {
                    const isSelected = selectedMotor?.uuid === m.uuid || selectedMotor?.id === m.id;
                    return (
                      <div
                        key={m.uuid || m.id}
                        onClick={() => setSelectedMotor(m)}
                        className={`p-3 rounded-xl border cursor-pointer transition-all flex flex-col sm:flex-row sm:items-center justify-between gap-2 ${
                          isSelected
                            ? "border-blue-500 bg-blue-50/50 dark:bg-blue-950/30 ring-1 ring-blue-500"
                            : "border-zinc-200 dark:border-zinc-800 bg-zinc-50/50 dark:bg-zinc-900/60 hover:border-zinc-400 dark:hover:border-zinc-700"
                        }`}
                      >
                        <div className="flex-1 min-w-0">
                          <div className="flex items-center gap-2 flex-wrap">
                            <span className="text-xs font-bold text-zinc-900 dark:text-zinc-100 truncate">
                              {m.name}
                            </span>
                            <span className="text-[10px] px-2 py-0.5 rounded-full font-semibold bg-blue-500/10 text-blue-600 dark:text-blue-400 border border-blue-500/20 shrink-0">
                              {m.kv} KV
                            </span>
                          </div>
                          <div className="text-[11px] text-zinc-500 dark:text-zinc-400 mt-0.5">
                            {m.manufacturer} • Weight: {m.weightG}g ea (
                            {((m.weightG || 0) * motorCount).toFixed(1)}g total)
                          </div>
                        </div>
                        <div className="text-[11px] text-emerald-600 dark:text-emerald-400 font-mono shrink-0">
                          {m.statorDiameterMm
                            ? `${m.statorDiameterMm}${m.statorHeightMm || ""}`
                            : "Brushless"}
                        </div>
                      </div>
                    );
                  })}
                </div>
                {filteredMotors.length === 0 && (
                  <p className="text-xs text-zinc-400 italic py-2 text-center">
                    No compatible motors matching your search.
                  </p>
                )}
              </div>

              {/* 1C. Propellers */}
              <div className="p-4 rounded-2xl border border-zinc-200 dark:border-zinc-800 bg-white dark:bg-zinc-900 space-y-3">
                <div className="flex items-center justify-between">
                  <div className="flex items-center gap-2">
                    <span className="w-2 h-2 rounded-full bg-blue-500"></span>
                    <h3 className="font-bold text-sm text-zinc-900 dark:text-zinc-100">
                      1C. Propellers{" "}
                      <span className="text-xs text-blue-600 dark:text-blue-400 font-mono font-semibold">
                        ({motorCount}x)
                      </span>
                    </h3>
                  </div>
                  {selectedProp ? (
                    <span className="text-xs font-semibold text-emerald-600 dark:text-emerald-400 flex items-center gap-1">
                      <Check size={12} strokeWidth={2.5} /> Selected
                    </span>
                  ) : (
                    <span className="text-xs font-semibold text-red-500 dark:text-red-400">
                      Required
                    </span>
                  )}
                </div>

                {/* In-line Search Box */}
                <div className="relative">
                  <Search
                    size={14}
                    className="absolute left-3 top-1/2 -translate-y-1/2 text-zinc-400"
                  />
                  <input
                    type="text"
                    value={searchProp}
                    onChange={(e) => setSearchProp(e.target.value)}
                    placeholder="Search propellers by size, pitch, manufacturer..."
                    className="w-full bg-zinc-50 dark:bg-zinc-950 border border-zinc-200 dark:border-zinc-800 rounded-xl pl-9 pr-8 py-1.5 text-xs text-zinc-900 dark:text-zinc-100 placeholder-zinc-400 focus:outline-none focus:ring-1 focus:ring-blue-500"
                  />
                  {searchProp && (
                    <button
                      type="button"
                      onClick={() => setSearchProp("")}
                      className="absolute right-2.5 top-1/2 -translate-y-1/2 text-zinc-400 hover:text-zinc-600"
                    >
                      <X size={13} />
                    </button>
                  )}
                </div>

                {/* Dynamic WASM CEL Compatibility Filter Banner */}
                {propCelFilter && (
                  <div className="flex items-center justify-between text-xs px-2.5 py-1.5 rounded-lg bg-emerald-500/10 border border-emerald-500/20 text-emerald-700 dark:text-emerald-300">
                    <div className="flex items-center gap-1.5 flex-wrap">
                      <span className="font-semibold text-[10px] uppercase tracking-wider text-emerald-600 dark:text-emerald-400">
                        Compatibility Filter:
                      </span>
                      <code className="font-mono text-[11px] bg-emerald-500/10 px-1.5 py-0.5 rounded text-emerald-800 dark:text-emerald-200">
                        {propCelFilter}
                      </code>
                    </div>
                    <label className="flex items-center gap-1.5 cursor-pointer text-[11px] font-medium shrink-0">
                      <input
                        type="checkbox"
                        checked={filterCompatibleOnly}
                        onChange={(e) => setFilterCompatibleOnly(e.target.checked)}
                        className="rounded border-emerald-400 text-emerald-600 focus:ring-emerald-500"
                      />
                      <span>Compatible Only</span>
                    </label>
                  </div>
                )}

                <div className="overflow-y-auto max-h-64 sm:max-h-72 space-y-2 pr-1.5 focus:outline-none">
                  {filteredProps.map((p) => {
                    const isSelected = selectedProp?.uuid === p.uuid || selectedProp?.id === p.id;
                    const diam = p.diameterMm ? (p.diameterMm / 25.4).toFixed(1) : "5.0";
                    return (
                      <div
                        key={p.uuid || p.id}
                        onClick={() => setSelectedProp(p)}
                        className={`p-3 rounded-xl border cursor-pointer transition-all flex flex-col sm:flex-row sm:items-center justify-between gap-2 ${
                          isSelected
                            ? "border-blue-500 bg-blue-50/50 dark:bg-blue-950/30 ring-1 ring-blue-500"
                            : "border-zinc-200 dark:border-zinc-800 bg-zinc-50/50 dark:bg-zinc-900/60 hover:border-zinc-400 dark:hover:border-zinc-700"
                        }`}
                      >
                        <div className="flex-1 min-w-0">
                          <div className="flex items-center gap-2 flex-wrap">
                            <span className="text-xs font-bold text-zinc-900 dark:text-zinc-100 truncate">
                              {p.name}
                            </span>
                            <span className="text-[10px] px-2 py-0.5 rounded-full font-semibold bg-amber-500/10 text-amber-600 dark:text-amber-400 border border-amber-500/20 shrink-0">
                              {diam}"
                            </span>
                          </div>
                          <div className="text-[11px] text-zinc-500 dark:text-zinc-400 mt-0.5">
                            {p.manufacturer} • Weight: {p.weightG}g ea (
                            {((p.weightG || 0) * motorCount).toFixed(1)}g total)
                          </div>
                        </div>
                        <div className="text-[11px] text-emerald-600 dark:text-emerald-400 font-mono shrink-0">
                          {p.blades || 3}-Blade
                        </div>
                      </div>
                    );
                  })}
                </div>
                {filteredProps.length === 0 && (
                  <p className="text-xs text-zinc-400 italic py-2 text-center">
                    No compatible propellers matching your search.
                  </p>
                )}
              </div>
            </div>
          )}

          {/* STAGE 2: Flight Electronics & Power (FC, ESC, Receiver) */}
          {activeStage === 2 && (
            <div className="space-y-6">
              {/* 2A. Flight Controller */}
              <div className="p-4 rounded-2xl border border-zinc-200 dark:border-zinc-800 bg-white dark:bg-zinc-900 space-y-3">
                <div className="flex items-center justify-between">
                  <div className="flex items-center gap-2">
                    <span className="w-2 h-2 rounded-full bg-indigo-500"></span>
                    <h3 className="font-bold text-sm text-zinc-900 dark:text-zinc-100">
                      2A. Flight Controller
                    </h3>
                  </div>
                  {selectedFc ? (
                    <span className="text-xs font-semibold text-emerald-600 dark:text-emerald-400 flex items-center gap-1">
                      <Check size={12} strokeWidth={2.5} /> Selected
                    </span>
                  ) : (
                    <span className="text-xs font-semibold text-red-500 dark:text-red-400">
                      Required
                    </span>
                  )}
                </div>

                <div className="relative">
                  <Search
                    size={14}
                    className="absolute left-3 top-1/2 -translate-y-1/2 text-zinc-400"
                  />
                  <input
                    type="text"
                    value={searchFc}
                    onChange={(e) => setSearchFc(e.target.value)}
                    placeholder="Search flight controllers by MCU, gyro, mounting pattern..."
                    className="w-full bg-zinc-50 dark:bg-zinc-950 border border-zinc-200 dark:border-zinc-800 rounded-xl pl-9 pr-8 py-1.5 text-xs text-zinc-900 dark:text-zinc-100 placeholder-zinc-400 focus:outline-none focus:ring-1 focus:ring-blue-500"
                  />
                  {searchFc && (
                    <button
                      type="button"
                      onClick={() => setSearchFc("")}
                      className="absolute right-2.5 top-1/2 -translate-y-1/2 text-zinc-400 hover:text-zinc-600"
                    >
                      <X size={13} />
                    </button>
                  )}
                </div>

                <div className="overflow-y-auto max-h-64 sm:max-h-72 space-y-2 pr-1.5 focus:outline-none">
                  {filteredFcs.map((fc) => {
                    const isSelected = selectedFc?.uuid === fc.uuid || selectedFc?.id === fc.id;
                    const isAio =
                      !!fc.internalElectronicSpeedControllerUuid ||
                      fc.name.toLowerCase().includes("aio") ||
                      fc.name.toLowerCase().includes("whoop") ||
                      fc.name.toLowerCase().includes("12a") ||
                      fc.name.toLowerCase().includes("20a") ||
                      fc.name.toLowerCase().includes("45a");
                    const hasRx =
                      !!fc.internalReceiverUuid ||
                      fc.name.toLowerCase().includes("elrs") ||
                      fc.name.toLowerCase().includes("frsky");
                    const hasVtx =
                      !!fc.internalVideoTransmitterUuid || fc.name.toLowerCase().includes("vtx");
                    return (
                      <div
                        key={fc.uuid || fc.id}
                        onClick={() => setSelectedFc(fc)}
                        className={`p-3 rounded-xl border cursor-pointer transition-all flex flex-col sm:flex-row sm:items-center justify-between gap-2 ${
                          isSelected
                            ? "border-blue-500 bg-blue-50/50 dark:bg-blue-950/30 ring-1 ring-blue-500"
                            : "border-zinc-200 dark:border-zinc-800 bg-zinc-50/50 dark:bg-zinc-900/60 hover:border-zinc-400 dark:hover:border-zinc-700"
                        }`}
                      >
                        <div className="flex-1 min-w-0">
                          <div className="flex items-center gap-2 flex-wrap">
                            <span className="text-xs font-bold text-zinc-900 dark:text-zinc-100 truncate">
                              {fc.name}
                            </span>
                            <span
                              className={`text-[10px] px-2 py-0.5 rounded-full font-semibold border shrink-0 ${
                                isAio
                                  ? "bg-emerald-500/10 text-emerald-600 dark:text-emerald-400 border-emerald-500/20"
                                  : "bg-indigo-500/10 text-indigo-600 dark:text-indigo-400 border-indigo-500/20"
                              }`}
                            >
                              {isAio ? "AIO (Integrated ESC)" : "Standalone FC"}
                            </span>
                          </div>
                          <div className="text-[11px] text-zinc-500 dark:text-zinc-400 mt-0.5">
                            {fc.manufacturer} • Weight: {fc.weightG}g • {fc.processor || "MCU"}
                          </div>
                        </div>
                        <div className="text-[11px] text-zinc-400 font-mono shrink-0">
                          {isAio
                            ? `✓ Integrated ESC${hasRx ? " + RX" : ""}${hasVtx ? " + VTX" : ""} (external optional)`
                            : "Requires separate ESC"}
                        </div>
                      </div>
                    );
                  })}
                </div>
                {filteredFcs.length === 0 && (
                  <p className="text-xs text-zinc-400 italic py-2 text-center">
                    No compatible flight controllers matching your search.
                  </p>
                )}
              </div>

              {/* 2B. Electronic Speed Controller (ESC) - Conditional Logic */}
              <div className="p-4 rounded-2xl border border-zinc-200 dark:border-zinc-800 bg-white dark:bg-zinc-900 space-y-3">
                <div className="flex items-center justify-between">
                  <div className="flex items-center gap-2">
                    <span className="w-2 h-2 rounded-full bg-indigo-500"></span>
                    <h3 className="font-bold text-sm text-zinc-900 dark:text-zinc-100">
                      2B. Electronic Speed Controller (ESC)
                    </h3>
                  </div>
                  {selectedEsc || useIntegratedEsc ? (
                    <span className="text-xs font-semibold text-emerald-600 dark:text-emerald-400 flex items-center gap-1">
                      <Check size={12} strokeWidth={2.5} /> Selected
                    </span>
                  ) : fcHasAdequateIntegratedEsc ? (
                    <span className="text-xs text-zinc-500 dark:text-zinc-400 font-medium">
                      Optional
                    </span>
                  ) : (
                    <span className="text-xs font-semibold text-red-500 dark:text-red-400">
                      Required
                    </span>
                  )}
                </div>

                <p className="text-xs text-zinc-500 dark:text-zinc-400">
                  {fcHasAdequateIntegratedEsc
                    ? "External ESC is optional since your flight controller includes an integrated ESC with sufficient channels."
                    : `Regulates power to your ${motorCount} motors. Standalone FC requires a dedicated ESC.`}
                </p>

                <div className="relative">
                  <Search
                    size={14}
                    className="absolute left-3 top-1/2 -translate-y-1/2 text-zinc-400"
                  />
                  <input
                    type="text"
                    value={searchEsc}
                    onChange={(e) => setSearchEsc(e.target.value)}
                    placeholder="Search ESCs by current rating, protocol, mounting pattern..."
                    className="w-full bg-zinc-50 dark:bg-zinc-950 border border-zinc-200 dark:border-zinc-800 rounded-xl pl-9 pr-8 py-1.5 text-xs text-zinc-900 dark:text-zinc-100 placeholder-zinc-400 focus:outline-none focus:ring-1 focus:ring-blue-500"
                  />
                  {searchEsc && (
                    <button
                      type="button"
                      onClick={() => setSearchEsc("")}
                      className="absolute right-2.5 top-1/2 -translate-y-1/2 text-zinc-400 hover:text-zinc-600"
                    >
                      <X size={13} />
                    </button>
                  )}
                </div>

                {/* Dynamic WASM CEL Compatibility Filter Banner */}
                {escCelFilter && (
                  <div className="flex items-center justify-between text-xs px-2.5 py-1.5 rounded-lg bg-emerald-500/10 border border-emerald-500/20 text-emerald-700 dark:text-emerald-300">
                    <div className="flex items-center gap-1.5 flex-wrap">
                      <span className="font-semibold text-[10px] uppercase tracking-wider text-emerald-600 dark:text-emerald-400">
                        Compatibility Filter:
                      </span>
                      <code className="font-mono text-[11px] bg-emerald-500/10 px-1.5 py-0.5 rounded text-emerald-800 dark:text-emerald-200">
                        {escCelFilter}
                      </code>
                    </div>
                    <label className="flex items-center gap-1.5 cursor-pointer text-[11px] font-medium shrink-0">
                      <input
                        type="checkbox"
                        checked={filterCompatibleOnly}
                        onChange={(e) => setFilterCompatibleOnly(e.target.checked)}
                        className="rounded border-emerald-400 text-emerald-600 focus:ring-emerald-500"
                      />
                      <span>Compatible Only</span>
                    </label>
                  </div>
                )}

                <div className="overflow-y-auto max-h-64 sm:max-h-72 space-y-2 pr-1.5 focus:outline-none">
                  {/* Option Card: Integrated FC ESC (Enabled only if FC has adequate ESC) */}
                  <div
                    onClick={() => {
                      if (fcHasAdequateIntegratedEsc) {
                        setUseIntegratedEsc(true);
                        setSelectedEsc(integratedEsc);
                        setNoneSelections((prev) => ({ ...prev, esc: true }));
                      }
                    }}
                    className={`p-3 rounded-xl border transition-all flex flex-col sm:flex-row sm:items-center justify-between gap-2 ${
                      !fcHasAdequateIntegratedEsc
                        ? "opacity-40 cursor-not-allowed border-zinc-200 dark:border-zinc-800 bg-zinc-100/50 dark:bg-zinc-900/40"
                        : useIntegratedEsc
                          ? "border-emerald-500 bg-emerald-50/50 dark:bg-emerald-950/30 ring-1 ring-emerald-500 cursor-pointer"
                          : "border-zinc-200 dark:border-zinc-800 bg-zinc-50/50 dark:bg-zinc-900/60 hover:border-zinc-400 cursor-pointer"
                    }`}
                  >
                    <div className="flex-1 min-w-0">
                      <div className="flex items-center gap-2 flex-wrap">
                        <span className="font-bold text-xs text-zinc-900 dark:text-zinc-100 truncate">
                          Use Integrated FC ESC
                        </span>
                        {integratedEsc && (
                          <span className="text-[10px] px-2 py-0.5 rounded-full font-semibold bg-indigo-500/10 text-indigo-600 dark:text-indigo-400 border border-indigo-500/20 shrink-0">
                            {integratedEsc.motorCurrentMaxA || 20}A
                          </span>
                        )}
                        <span className="text-[10px] px-2 py-0.5 rounded-full font-semibold bg-zinc-200 dark:bg-zinc-800 text-zinc-700 dark:text-zinc-300 shrink-0">
                          Integrated
                        </span>
                      </div>
                      <div className="text-[10px] text-zinc-500 dark:text-zinc-400 mt-0.5">
                        {integratedEsc
                          ? integratedEsc.name
                          : selectedFc
                            ? `Integrated into ${selectedFc.name}`
                            : "Requires FC selection"}
                      </div>
                      <div className="text-[11px] text-zinc-500 dark:text-zinc-400 mt-0.5">
                        {integratedEsc
                          ? `Weight: 0g (FC integrated) • ${integratedEsc.maxMotors || 4}x Motors • ${integratedEsc.firmware || "BLHeli_S"}`
                          : "Uses FC drivers (0g added)"}
                      </div>
                    </div>
                    <div className="text-[11px] text-emerald-600 dark:text-emerald-400 font-mono shrink-0">
                      {fcHasAdequateIntegratedEsc
                        ? "✓ Available on selected FC"
                        : "Requires AIO FC with integrated ESC"}
                    </div>
                  </div>

                  {filteredEscs.map((esc) => {
                    const isSelected =
                      !useIntegratedEsc &&
                      (selectedEsc?.uuid === esc.uuid || selectedEsc?.id === esc.id);
                    return (
                      <div
                        key={esc.uuid || esc.id}
                        onClick={() => {
                          setSelectedEsc(esc);
                          setUseIntegratedEsc(false);
                          setNoneSelections((prev) => ({ ...prev, esc: false }));
                        }}
                        className={`p-3 rounded-xl border cursor-pointer transition-all flex flex-col sm:flex-row sm:items-center justify-between gap-2 ${
                          isSelected
                            ? "border-blue-500 bg-blue-50/50 dark:bg-blue-950/30 ring-1 ring-blue-500"
                            : "border-zinc-200 dark:border-zinc-800 bg-zinc-50/50 dark:bg-zinc-900/60 hover:border-zinc-400 dark:hover:border-zinc-700"
                        }`}
                      >
                        <div className="flex-1 min-w-0">
                          <div className="flex items-center gap-2 flex-wrap">
                            <span className="text-xs font-bold text-zinc-900 dark:text-zinc-100 truncate">
                              {esc.name}
                            </span>
                            <span className="text-[10px] px-2 py-0.5 rounded-full font-semibold bg-indigo-500/10 text-indigo-600 dark:text-indigo-400 border border-indigo-500/20 shrink-0">
                              {esc.motorCurrentMaxA || 50}A
                            </span>
                          </div>
                          <div className="text-[11px] text-zinc-500 dark:text-zinc-400 mt-0.5">
                            {esc.manufacturer} • Weight: {esc.weightG}g • {esc.maxMotors || 4}x
                            Motors
                          </div>
                        </div>
                        <div className="text-[11px] text-emerald-600 dark:text-emerald-400 font-mono shrink-0">
                          4-in-1 ESC
                        </div>
                      </div>
                    );
                  })}
                </div>
                {filteredEscs.length === 0 && (
                  <p className="text-xs text-zinc-400 italic py-2 text-center">
                    No compatible ESCs matching your search.
                  </p>
                )}
              </div>

              {/* 2C. Radio Receiver */}
              <div className="p-4 rounded-2xl border border-zinc-200 dark:border-zinc-800 bg-white dark:bg-zinc-900 space-y-3">
                <div className="flex items-center justify-between">
                  <div className="flex items-center gap-2">
                    <span className="w-2 h-2 rounded-full bg-indigo-500"></span>
                    <h3 className="font-bold text-sm text-zinc-900 dark:text-zinc-100">
                      2C. Radio Control Receiver
                    </h3>
                  </div>
                  {selectedRx || useIntegratedRx ? (
                    <span className="text-xs font-semibold text-emerald-600 dark:text-emerald-400 flex items-center gap-1">
                      <Check size={12} strokeWidth={2.5} /> Selected
                    </span>
                  ) : fcHasIntegratedRx ? (
                    <span className="text-xs text-zinc-500 dark:text-zinc-400 font-medium">
                      Optional
                    </span>
                  ) : (
                    <span className="text-xs font-semibold text-red-500 dark:text-red-400">
                      Required
                    </span>
                  )}
                </div>
                <p className="text-xs text-zinc-500 dark:text-zinc-400">
                  {fcHasIntegratedRx
                    ? "External receiver is optional since your flight controller includes an integrated receiver."
                    : "Provides pilot command link directly to the Flight Controller via serial UART."}
                </p>

                <div className="relative">
                  <Search
                    size={14}
                    className="absolute left-3 top-1/2 -translate-y-1/2 text-zinc-400"
                  />
                  <input
                    type="text"
                    value={searchRx}
                    onChange={(e) => setSearchRx(e.target.value)}
                    placeholder="Search receivers by protocol (ELRS, Crossfire), brand..."
                    className="w-full bg-zinc-50 dark:bg-zinc-950 border border-zinc-200 dark:border-zinc-800 rounded-xl pl-9 pr-8 py-1.5 text-xs text-zinc-900 dark:text-zinc-100 placeholder-zinc-400 focus:outline-none focus:ring-1 focus:ring-blue-500"
                  />
                  {searchRx && (
                    <button
                      type="button"
                      onClick={() => setSearchRx("")}
                      className="absolute right-2.5 top-1/2 -translate-y-1/2 text-zinc-400 hover:text-zinc-600"
                    >
                      <X size={13} />
                    </button>
                  )}
                </div>

                <div className="overflow-y-auto max-h-64 sm:max-h-72 space-y-2 pr-1.5 focus:outline-none">
                  {/* Option Card: Integrated FC Receiver */}
                  <div
                    onClick={() => {
                      if (fcHasIntegratedRx) {
                        setUseIntegratedRx(true);
                        setSelectedRx(integratedRx);
                      }
                    }}
                    className={`p-3 rounded-xl border transition-all flex flex-col sm:flex-row sm:items-center justify-between gap-2 ${
                      !fcHasIntegratedRx
                        ? "opacity-40 cursor-not-allowed border-zinc-200 dark:border-zinc-800 bg-zinc-100/50 dark:bg-zinc-900/40"
                        : useIntegratedRx
                          ? "border-emerald-500 bg-emerald-50/50 dark:bg-emerald-950/30 ring-1 ring-emerald-500 cursor-pointer"
                          : "border-zinc-200 dark:border-zinc-800 bg-zinc-50/50 dark:bg-zinc-900/60 hover:border-zinc-400 cursor-pointer"
                    }`}
                  >
                    <div className="flex-1 min-w-0">
                      <div className="flex items-center gap-2 flex-wrap">
                        <span className="font-bold text-xs text-zinc-900 dark:text-zinc-100 truncate">
                          Use Integrated FC Receiver
                        </span>
                        {integratedRx && (
                          <span className="text-[10px] px-2 py-0.5 rounded-full font-semibold bg-emerald-500/10 text-emerald-600 dark:text-emerald-400 border border-emerald-500/20 shrink-0">
                            {integratedRx.protocol || "ExpressLRS"}
                          </span>
                        )}
                        <span className="text-[10px] px-2 py-0.5 rounded-full font-semibold bg-zinc-200 dark:bg-zinc-800 text-zinc-700 dark:text-zinc-300 shrink-0">
                          Integrated
                        </span>
                      </div>
                      <div className="text-[10px] text-zinc-500 dark:text-zinc-400 mt-0.5">
                        {integratedRx
                          ? integratedRx.name
                          : selectedFc
                            ? `Integrated into ${selectedFc.name}`
                            : "Requires FC selection"}
                      </div>
                      <div className="text-[11px] text-zinc-500 dark:text-zinc-400 mt-0.5">
                        {integratedRx
                          ? `Weight: 0g (FC integrated) • ${formatFrequencyBand(integratedRx.frequencyBandMhz)} • ${integratedRx.hasTelemetry ? "Telemetry" : "Non-telemetry"}`
                          : "Built-in receiver link"}
                      </div>
                    </div>
                    <div className="text-[11px] text-emerald-600 dark:text-emerald-400 font-mono shrink-0">
                      {fcHasIntegratedRx
                        ? "✓ Internal SPI/UART (0x UART used)"
                        : "Requires FC with integrated receiver"}
                    </div>
                  </div>

                  {filteredRxs.map((rx) => {
                    const isSelected =
                      !useIntegratedRx &&
                      (selectedRx?.uuid === rx.uuid || selectedRx?.id === rx.id);
                    return (
                      <div
                        key={rx.uuid || rx.id}
                        onClick={() => {
                          setSelectedRx(rx);
                          setUseIntegratedRx(false);
                        }}
                        className={`p-3 rounded-xl border cursor-pointer transition-all flex flex-col sm:flex-row sm:items-center justify-between gap-2 ${
                          isSelected
                            ? "border-blue-500 bg-blue-50/50 dark:bg-blue-950/30 ring-1 ring-blue-500"
                            : "border-zinc-200 dark:border-zinc-800 bg-zinc-50/50 dark:bg-zinc-900/60 hover:border-zinc-400 dark:hover:border-zinc-700"
                        }`}
                      >
                        <div className="flex-1 min-w-0">
                          <div className="flex items-center gap-2 flex-wrap">
                            <span className="text-xs font-bold text-zinc-900 dark:text-zinc-100 truncate">
                              {rx.name}
                            </span>
                            <span className="text-[10px] px-2 py-0.5 rounded-full font-semibold bg-emerald-500/10 text-emerald-600 dark:text-emerald-400 border border-emerald-500/20 shrink-0">
                              {rx.protocol || "Serial"}
                            </span>
                          </div>
                          <div className="text-[11px] text-zinc-500 dark:text-zinc-400 mt-0.5">
                            {rx.manufacturer} • Weight: {rx.weightG}g •{" "}
                            {formatFrequencyBand(rx.frequencyBandMhz)}
                          </div>
                        </div>
                        <div className="text-[11px] text-emerald-600 dark:text-emerald-400 font-mono shrink-0">
                          Requires 1x FC UART
                        </div>
                      </div>
                    );
                  })}
                </div>
                {filteredRxs.length === 0 && (
                  <p className="text-xs text-zinc-400 italic py-2 text-center">
                    No compatible receivers matching your search.
                  </p>
                )}
              </div>

              {/* 2D. Radio Receiver Antenna(s) */}
              <div className="p-4 rounded-2xl border border-zinc-200 dark:border-zinc-800 bg-white dark:bg-zinc-900 space-y-3">
                <div className="flex items-center justify-between">
                  <div className="flex items-center gap-2">
                    <span className="w-2 h-2 rounded-full bg-indigo-500"></span>
                    <h3 className="font-bold text-sm text-zinc-900 dark:text-zinc-100">
                      2D. Radio Receiver Antenna(s)
                    </h3>
                  </div>
                  {selectedRxAnt && !noneSelections.rxAntenna ? (
                    <span className="text-xs font-semibold text-emerald-600 dark:text-emerald-400 flex items-center gap-1">
                      <Check size={12} strokeWidth={2.5} />{" "}
                      {rxAntCount > 1 ? "2x Selected (Diversity)" : "Selected"}
                    </span>
                  ) : noneSelections.rxAntenna ? (
                    <span className="text-xs font-semibold text-zinc-500 flex items-center gap-1">
                      <Check size={12} strokeWidth={2.5} /> None
                    </span>
                  ) : (
                    <span className="text-xs text-zinc-500 dark:text-zinc-400 font-medium">
                      Optional
                    </span>
                  )}
                </div>
                <p className="text-xs text-zinc-500 dark:text-zinc-400">
                  Receives pilot control link signals. Supports a single antenna or dual diversity
                  antennas.
                </p>

                <div className="relative">
                  <Search
                    size={14}
                    className="absolute left-3 top-1/2 -translate-y-1/2 text-zinc-400"
                  />
                  <input
                    type="text"
                    value={searchRxAnt}
                    onChange={(e) => setSearchRxAnt(e.target.value)}
                    placeholder="Search RX antennas by connector, polarization, brand..."
                    className="w-full bg-zinc-50 dark:bg-zinc-950 border border-zinc-200 dark:border-zinc-800 rounded-xl pl-9 pr-8 py-1.5 text-xs text-zinc-900 dark:text-zinc-100 placeholder-zinc-400 focus:outline-none focus:ring-1 focus:ring-blue-500"
                  />
                  {searchRxAnt && (
                    <button
                      type="button"
                      onClick={() => setSearchRxAnt("")}
                      className="absolute right-2.5 top-1/2 -translate-y-1/2 text-zinc-400 hover:text-zinc-600"
                    >
                      <X size={13} />
                    </button>
                  )}
                </div>

                {selectedRxAnt && !noneSelections.rxAntenna && (
                  <div className="flex items-center justify-between p-2.5 rounded-xl bg-blue-50/60 dark:bg-blue-950/30 border border-blue-200 dark:border-blue-900/50 text-xs">
                    <div className="flex items-center gap-2">
                      <span className="font-semibold text-zinc-800 dark:text-zinc-200">
                        Antenna Quantity:
                      </span>
                      <span className="text-zinc-500 dark:text-zinc-400">
                        {rxAntCount === 1
                          ? "Single Antenna (Standard RX)"
                          : "Dual Antennas (True Diversity / 90° mounting)"}
                      </span>
                    </div>
                    <div className="flex items-center gap-1 bg-white dark:bg-zinc-900 p-0.5 rounded-lg border border-zinc-200 dark:border-zinc-800 shadow-2xs">
                      <button
                        type="button"
                        onClick={() => setRxAntCount(1)}
                        className={`px-2.5 py-1 rounded-md text-[11px] font-bold transition-all cursor-pointer ${
                          rxAntCount === 1
                            ? "bg-blue-600 text-white shadow-xs"
                            : "text-zinc-600 dark:text-zinc-400 hover:text-zinc-900 dark:hover:text-zinc-100"
                        }`}
                      >
                        1x Single
                      </button>
                      <button
                        type="button"
                        onClick={() => setRxAntCount(2)}
                        className={`px-2.5 py-1 rounded-md text-[11px] font-bold transition-all cursor-pointer ${
                          rxAntCount === 2
                            ? "bg-blue-600 text-white shadow-xs"
                            : "text-zinc-600 dark:text-zinc-400 hover:text-zinc-900 dark:hover:text-zinc-100"
                        }`}
                      >
                        2x Diversity
                      </button>
                    </div>
                  </div>
                )}

                <div className="overflow-y-auto max-h-64 sm:max-h-72 space-y-2 pr-1.5 focus:outline-none">
                  {/* Explicit None Card */}
                  <div
                    onClick={() => {
                      setSelectedRxAnt(null);
                      setNoneSelections((prev) => ({ ...prev, rxAntenna: true }));
                    }}
                    className={`p-3 rounded-xl border cursor-pointer text-xs transition-all flex flex-col sm:flex-row sm:items-center justify-between gap-2 ${
                      noneSelections.rxAntenna
                        ? "border-blue-500 bg-blue-50/50 dark:bg-blue-950/30 ring-1 ring-blue-500"
                        : "border-zinc-200 dark:border-zinc-800 bg-zinc-50/50 dark:bg-zinc-900/60 hover:border-zinc-400"
                    }`}
                  >
                    <div className="flex-1 min-w-0">
                      <div className="font-bold text-zinc-900 dark:text-zinc-200">
                        No External Antenna (None)
                      </div>
                      <div className="text-[11px] text-zinc-500 mt-0.5">
                        Built-in ceramic antenna or saves weight (0g)
                      </div>
                    </div>
                    <div className="text-[11px] text-zinc-500 font-mono shrink-0">0g</div>
                  </div>

                  {filteredRxAnts.map((a) => {
                    const isSelected =
                      !noneSelections.rxAntenna &&
                      (selectedRxAnt?.uuid === a.uuid || selectedRxAnt?.id === a.id);
                    return (
                      <div
                        key={a.uuid || a.id}
                        onClick={() => {
                          setSelectedRxAnt(a);
                          setNoneSelections((prev) => ({ ...prev, rxAntenna: false }));
                        }}
                        className={`p-3 rounded-xl border cursor-pointer transition-all flex flex-col sm:flex-row sm:items-center justify-between gap-2 ${
                          isSelected
                            ? "border-blue-500 bg-blue-50/50 dark:bg-blue-950/30 ring-1 ring-blue-500"
                            : "border-zinc-200 dark:border-zinc-800 bg-zinc-50/50 dark:bg-zinc-900/60 hover:border-zinc-400 dark:hover:border-zinc-700"
                        }`}
                      >
                        <div className="flex-1 min-w-0">
                          <div className="flex items-center gap-2 flex-wrap">
                            <span className="text-xs font-bold text-zinc-900 dark:text-zinc-100 truncate">
                              {a.name}
                            </span>
                            <span className="text-[10px] px-2 py-0.5 rounded-full font-semibold bg-emerald-500/10 text-emerald-600 dark:text-emerald-400 border border-emerald-500/20 shrink-0">
                              {formatFrequencyBand(a.frequencyBandMhz)}
                            </span>
                          </div>
                          <div className="text-[11px] text-zinc-500 dark:text-zinc-400 mt-0.5">
                            {a.manufacturer} • Weight:{" "}
                            {((a.weightG || 0) * (isSelected ? rxAntCount : 1)).toFixed(1)}g{" "}
                            {isSelected && rxAntCount > 1 ? `(${rxAntCount}x ${a.weightG}g)` : ""} •{" "}
                            {a.polarization || "Linear"}
                          </div>
                        </div>
                        <div className="text-[11px] text-emerald-600 dark:text-emerald-400 font-mono shrink-0">
                          Connector: {a.connector || "U.FL"}
                        </div>
                      </div>
                    );
                  })}
                </div>
                {filteredRxAnts.length === 0 && (
                  <p className="text-xs text-zinc-400 italic py-2 text-center">
                    No compatible RX antennas matching your search.
                  </p>
                )}
              </div>

              {/* 2E. GPS Receiver & Compass */}
              <div className="p-4 rounded-2xl border border-zinc-200 dark:border-zinc-800 bg-white dark:bg-zinc-900 space-y-3">
                <div className="flex items-center justify-between">
                  <div className="flex items-center gap-2">
                    <span className="w-2 h-2 rounded-full bg-indigo-500"></span>
                    <h3 className="font-bold text-sm text-zinc-900 dark:text-zinc-100">
                      2E. GPS Receiver & Compass
                    </h3>
                  </div>
                  {selectedGps || noneSelections.gps ? (
                    <span className="text-xs font-semibold text-emerald-600 dark:text-emerald-400 flex items-center gap-1">
                      <Check size={12} strokeWidth={2.5} /> Selected
                    </span>
                  ) : (
                    <span className="text-xs text-zinc-500 dark:text-zinc-400 font-medium">
                      Optional
                    </span>
                  )}
                </div>
                <p className="text-xs text-zinc-500 dark:text-zinc-400">
                  Enables satellite positioning, return-to-home (RTH), speed, and rescue
                  capabilities.
                </p>

                <div className="relative">
                  <Search
                    size={14}
                    className="absolute left-3 top-1/2 -translate-y-1/2 text-zinc-400"
                  />
                  <input
                    type="text"
                    value={searchGps}
                    onChange={(e) => setSearchGps(e.target.value)}
                    placeholder="Search GPS by chipset, compass, brand..."
                    className="w-full bg-zinc-50 dark:bg-zinc-950 border border-zinc-200 dark:border-zinc-800 rounded-xl pl-9 pr-8 py-1.5 text-xs text-zinc-900 dark:text-zinc-100 placeholder-zinc-400 focus:outline-none focus:ring-1 focus:ring-blue-500"
                  />
                  {searchGps && (
                    <button
                      type="button"
                      onClick={() => setSearchGps("")}
                      className="absolute right-2.5 top-1/2 -translate-y-1/2 text-zinc-400 hover:text-zinc-600"
                    >
                      <X size={13} />
                    </button>
                  )}
                </div>

                <div className="overflow-y-auto max-h-64 sm:max-h-72 space-y-2 pr-1.5 focus:outline-none">
                  <div
                    onClick={() => {
                      setSelectedGps(null);
                      setNoneSelections((prev) => ({ ...prev, gps: true }));
                    }}
                    className={`p-3 rounded-xl border cursor-pointer text-xs transition-all flex flex-col sm:flex-row sm:items-center justify-between gap-2 ${
                      noneSelections.gps
                        ? "border-blue-500 bg-blue-50/50 dark:bg-blue-950/30 ring-1 ring-blue-500"
                        : "border-zinc-200 dark:border-zinc-800 bg-zinc-50/50 dark:bg-zinc-900/60 hover:border-zinc-400"
                    }`}
                  >
                    <div className="flex-1 min-w-0">
                      <div className="font-bold text-zinc-900 dark:text-zinc-200">
                        No GPS (None)
                      </div>
                      <div className="text-[11px] text-zinc-500 mt-0.5">
                        Saves weight & frees 1 UART
                      </div>
                    </div>
                    <div className="text-[11px] text-zinc-500 font-mono shrink-0">0g</div>
                  </div>

                  {filteredGps.map((g) => {
                    const isSelected =
                      !noneSelections.gps &&
                      (selectedGps?.uuid === g.uuid || selectedGps?.id === g.id);
                    return (
                      <div
                        key={g.uuid || g.id}
                        onClick={() => {
                          setSelectedGps(g);
                          setNoneSelections((prev) => ({ ...prev, gps: false }));
                        }}
                        className={`p-3 rounded-xl border cursor-pointer text-xs transition-all flex flex-col sm:flex-row sm:items-center justify-between gap-2 ${
                          isSelected
                            ? "border-blue-500 bg-blue-50/50 dark:bg-blue-950/30 ring-1 ring-blue-500"
                            : "border-zinc-200 dark:border-zinc-800 bg-zinc-50/50 dark:bg-zinc-900/60 hover:border-zinc-400"
                        }`}
                      >
                        <div className="flex-1 min-w-0">
                          <div className="flex items-center gap-2 flex-wrap">
                            <span className="font-bold text-zinc-900 dark:text-zinc-200">
                              {g.name}
                            </span>
                            <span className="text-[10px] px-2 py-0.5 rounded-full font-semibold bg-indigo-500/10 text-indigo-600 dark:text-indigo-400 border border-indigo-500/20 shrink-0">
                              {g.protocol || "UBLOX"}
                            </span>
                          </div>
                          <div className="text-[11px] text-zinc-500 dark:text-zinc-400 mt-0.5">
                            {g.manufacturer} • Weight: {g.weightG}g •{" "}
                            {g.compass ? "Compass included" : "No compass"}
                          </div>
                        </div>
                        <div className="text-[11px] text-emerald-600 dark:text-emerald-400 font-mono shrink-0">
                          1x FC UART
                        </div>
                      </div>
                    );
                  })}
                </div>
              </div>
            </div>
          )}

          {/* STAGE 3: Video (All Optional) */}
          {activeStage === 3 && (
            <div className="space-y-6">
              {/* Optional Notice & Skip All Button */}
              <div className="p-3.5 rounded-xl border border-blue-500/20 bg-blue-50/50 dark:bg-blue-950/20 text-blue-900 dark:text-blue-200 text-xs flex flex-col sm:flex-row sm:items-center justify-between gap-2">
                <span>
                  ℹ️ All video components in Stage 3 are optional. Configure an FPV video system or
                  choose "None" for line-of-sight flying.
                </span>
                <button
                  type="button"
                  onClick={skipAllStage3}
                  className="px-3 py-1 rounded-lg bg-blue-600 hover:bg-blue-500 text-white font-bold text-[11px] transition-colors shrink-0 cursor-pointer"
                >
                  Set All to None (Line-of-Sight)
                </button>
              </div>

              {/* 3A. Video Transmitter (VTX) */}
              <div className="p-4 rounded-2xl border border-zinc-200 dark:border-zinc-800 bg-white dark:bg-zinc-900 space-y-3">
                <div className="flex items-center justify-between">
                  <h3 className="font-bold text-sm text-zinc-900 dark:text-zinc-100">
                    3A. Video Transmitter (VTX)
                  </h3>
                  {selectedVtx || useIntegratedVtx || noneSelections.vtx ? (
                    <span className="text-xs font-semibold text-emerald-600 dark:text-emerald-400 flex items-center gap-1">
                      <Check size={12} strokeWidth={2.5} /> Selected
                    </span>
                  ) : (
                    <span className="text-xs text-zinc-500 dark:text-zinc-400 font-medium">
                      Optional
                    </span>
                  )}
                </div>

                <div className="relative">
                  <Search
                    size={14}
                    className="absolute left-3 top-1/2 -translate-y-1/2 text-zinc-400"
                  />
                  <input
                    type="text"
                    value={searchVtx}
                    onChange={(e) => setSearchVtx(e.target.value)}
                    placeholder="Search VTX by power, brand, analog/digital..."
                    className="w-full bg-zinc-50 dark:bg-zinc-950 border border-zinc-200 dark:border-zinc-800 rounded-xl pl-9 pr-8 py-1.5 text-xs text-zinc-900 dark:text-zinc-100 placeholder-zinc-400 focus:outline-none focus:ring-1 focus:ring-blue-500"
                  />
                  {searchVtx && (
                    <button
                      type="button"
                      onClick={() => setSearchVtx("")}
                      className="absolute right-2.5 top-1/2 -translate-y-1/2 text-zinc-400 hover:text-zinc-600"
                    >
                      <X size={13} />
                    </button>
                  )}
                </div>

                {/* Dynamic WASM CEL Compatibility Filter Banner */}
                {vtxCelFilter && (
                  <div className="flex items-center justify-between text-xs px-2.5 py-1.5 rounded-lg bg-emerald-500/10 border border-emerald-500/20 text-emerald-700 dark:text-emerald-300">
                    <div className="flex items-center gap-1.5 flex-wrap">
                      <span className="font-semibold text-[10px] uppercase tracking-wider text-emerald-600 dark:text-emerald-400">
                        Compatibility Filter:
                      </span>
                      <code className="font-mono text-[11px] bg-emerald-500/10 px-1.5 py-0.5 rounded text-emerald-800 dark:text-emerald-200">
                        {vtxCelFilter}
                      </code>
                    </div>
                    <label className="flex items-center gap-1.5 cursor-pointer text-[11px] font-medium shrink-0">
                      <input
                        type="checkbox"
                        checked={filterCompatibleOnly}
                        onChange={(e) => setFilterCompatibleOnly(e.target.checked)}
                        className="rounded border-emerald-400 text-emerald-600 focus:ring-emerald-500"
                      />
                      <span>Compatible Only</span>
                    </label>
                  </div>
                )}

                <div className="overflow-y-auto max-h-64 sm:max-h-72 space-y-2 pr-1.5 focus:outline-none">
                  {/* Explicit None Card */}
                  <div
                    onClick={() => {
                      setSelectedVtx(null);
                      setUseIntegratedVtx(false);
                      setNoneSelections((prev) => ({ ...prev, vtx: true }));
                    }}
                    className={`p-3 rounded-xl border cursor-pointer text-xs transition-all flex flex-col sm:flex-row sm:items-center justify-between gap-2 ${
                      noneSelections.vtx
                        ? "border-blue-500 bg-blue-50/50 dark:bg-blue-950/30 ring-1 ring-blue-500"
                        : "border-zinc-200 dark:border-zinc-800 bg-zinc-50/50 dark:bg-zinc-900/60 hover:border-zinc-400"
                    }`}
                  >
                    <div className="flex-1 min-w-0">
                      <div className="font-bold text-zinc-900 dark:text-zinc-200">
                        No VTX (None)
                      </div>
                      <div className="text-[11px] text-zinc-500 mt-0.5">Saves weight (0g)</div>
                    </div>
                    <div className="text-[11px] text-zinc-500 font-mono shrink-0">0g</div>
                  </div>

                  {/* Option Card: Integrated FC VTX (if FC integrates VTX) */}
                  {fcHasIntegratedVtx && (
                    <div
                      onClick={() => {
                        setUseIntegratedVtx(true);
                        setSelectedVtx(integratedVtx);
                        setNoneSelections((prev) => ({ ...prev, vtx: false }));
                      }}
                      className={`p-3 rounded-xl border cursor-pointer transition-all flex flex-col sm:flex-row sm:items-center justify-between gap-2 ${
                        useIntegratedVtx
                          ? "border-emerald-500 bg-emerald-50/50 dark:bg-emerald-950/30 ring-1 ring-emerald-500"
                          : "border-zinc-200 dark:border-zinc-800 bg-zinc-50/50 dark:bg-zinc-900/60 hover:border-zinc-400"
                      }`}
                    >
                      <div className="flex-1 min-w-0">
                        <div className="flex items-center gap-2 flex-wrap">
                          <span className="text-xs font-bold text-zinc-900 dark:text-zinc-100 truncate">
                            {integratedVtx?.name || "Use Integrated FC VTX"}
                          </span>
                          <span className="text-[10px] px-1.5 py-0.5 rounded-full font-semibold bg-zinc-200 dark:bg-zinc-800 text-zinc-700 dark:text-zinc-300 shrink-0">
                            Integrated
                          </span>
                          <span className="text-[10px] px-2 py-0.5 rounded-full font-semibold bg-pink-500/10 text-pink-600 dark:text-pink-400 border border-pink-500/20 shrink-0">
                            {integratedVtx?.maxPowerMw || 400}mW
                          </span>
                        </div>
                        <div className="text-[11px] text-zinc-500 dark:text-zinc-400 mt-0.5">
                          {integratedVtx?.manufacturer || selectedFc?.manufacturer} • Integrated
                          into {selectedFc?.name} • Weight: 0g (FC integrated) •{" "}
                          {integratedVtx?.protocol || "Analog"}
                        </div>
                      </div>
                      <div className="text-[11px] text-pink-600 dark:text-pink-400 font-mono shrink-0">
                        On-board VTX
                      </div>
                    </div>
                  )}

                  {filteredVtxs.map((v) => {
                    const isSelected =
                      !useIntegratedVtx &&
                      !noneSelections.vtx &&
                      (selectedVtx?.uuid === v.uuid || selectedVtx?.id === v.id);
                    return (
                      <div
                        key={v.uuid || v.id}
                        onClick={() => {
                          setSelectedVtx(v);
                          setUseIntegratedVtx(false);
                          setNoneSelections((prev) => ({ ...prev, vtx: false }));
                        }}
                        className={`p-3 rounded-xl border cursor-pointer transition-all flex flex-col sm:flex-row sm:items-center justify-between gap-2 ${
                          isSelected
                            ? "border-blue-500 bg-blue-50/50 dark:bg-blue-950/30 ring-1 ring-blue-500"
                            : "border-zinc-200 dark:border-zinc-800 bg-zinc-50/50 dark:bg-zinc-900/60 hover:border-zinc-400 dark:hover:border-zinc-700"
                        }`}
                      >
                        <div className="flex-1 min-w-0">
                          <div className="flex items-center gap-2 flex-wrap">
                            <span className="text-xs font-bold text-zinc-900 dark:text-zinc-100 truncate">
                              {v.name}
                            </span>
                            {v.maxPowerMw && (
                              <span className="text-[10px] px-2 py-0.5 rounded-full font-semibold bg-pink-500/10 text-pink-600 dark:text-pink-400 border border-pink-500/20 shrink-0">
                                {v.maxPowerMw}mW
                              </span>
                            )}
                          </div>
                          <div className="text-[11px] text-zinc-500 dark:text-zinc-400 mt-0.5">
                            {v.manufacturer} • Protocol: {v.protocol || "5.8GHz"} • Weight:{" "}
                            {v.weightG}g
                          </div>
                        </div>
                        <div className="text-[11px] text-zinc-500 font-mono shrink-0">
                          {v.weightG}g
                        </div>
                      </div>
                    );
                  })}
                </div>
                {filteredVtxs.length === 0 && !fcHasIntegratedVtx && (
                  <p className="text-xs text-zinc-400 italic py-2 text-center">
                    No compatible VTX products matching your search.
                  </p>
                )}
              </div>

              {/* 3B. Camera */}
              <div className="p-4 rounded-2xl border border-zinc-200 dark:border-zinc-800 bg-white dark:bg-zinc-900 space-y-3">
                <div className="flex items-center justify-between">
                  <h3 className="font-bold text-sm text-zinc-900 dark:text-zinc-100">
                    3B. FPV Camera
                  </h3>
                  {selectedCam || noneSelections.camera ? (
                    <span className="text-xs font-semibold text-emerald-600 dark:text-emerald-400 flex items-center gap-1">
                      <Check size={12} strokeWidth={2.5} /> Selected
                    </span>
                  ) : (
                    <span className="text-xs text-zinc-500 dark:text-zinc-400 font-medium">
                      Optional
                    </span>
                  )}
                </div>

                <div className="relative">
                  <Search
                    size={14}
                    className="absolute left-3 top-1/2 -translate-y-1/2 text-zinc-400"
                  />
                  <input
                    type="text"
                    value={searchCam}
                    onChange={(e) => setSearchCam(e.target.value)}
                    placeholder="Search camera by sensor, size, brand..."
                    className="w-full bg-zinc-50 dark:bg-zinc-950 border border-zinc-200 dark:border-zinc-800 rounded-xl pl-9 pr-8 py-1.5 text-xs text-zinc-900 dark:text-zinc-100 placeholder-zinc-400 focus:outline-none focus:ring-1 focus:ring-blue-500"
                  />
                  {searchCam && (
                    <button
                      type="button"
                      onClick={() => setSearchCam("")}
                      className="absolute right-2.5 top-1/2 -translate-y-1/2 text-zinc-400 hover:text-zinc-600"
                    >
                      <X size={13} />
                    </button>
                  )}
                </div>

                {/* Dynamic WASM CEL Compatibility Filter Banner */}
                {camCelFilter && (
                  <div className="flex items-center justify-between text-xs px-2.5 py-1.5 rounded-lg bg-emerald-500/10 border border-emerald-500/20 text-emerald-700 dark:text-emerald-300">
                    <div className="flex items-center gap-1.5 flex-wrap">
                      <span className="font-semibold text-[10px] uppercase tracking-wider text-emerald-600 dark:text-emerald-400">
                        Compatibility Filter:
                      </span>
                      <code className="font-mono text-[11px] bg-emerald-500/10 px-1.5 py-0.5 rounded text-emerald-800 dark:text-emerald-200">
                        {camCelFilter}
                      </code>
                    </div>
                    <label className="flex items-center gap-1.5 cursor-pointer text-[11px] font-medium shrink-0">
                      <input
                        type="checkbox"
                        checked={filterCompatibleOnly}
                        onChange={(e) => setFilterCompatibleOnly(e.target.checked)}
                        className="rounded border-emerald-400 text-emerald-600 focus:ring-emerald-500"
                      />
                      <span>Compatible Only</span>
                    </label>
                  </div>
                )}

                <div className="overflow-y-auto max-h-64 sm:max-h-72 space-y-2 pr-1.5 focus:outline-none">
                  <div
                    onClick={() => {
                      setSelectedCam(null);
                      setNoneSelections((prev) => ({ ...prev, camera: true }));
                    }}
                    className={`p-3 rounded-xl border cursor-pointer text-xs transition-all flex flex-col sm:flex-row sm:items-center justify-between gap-2 ${
                      noneSelections.camera
                        ? "border-blue-500 bg-blue-50/50 dark:bg-blue-950/30 ring-1 ring-blue-500"
                        : "border-zinc-200 dark:border-zinc-800 bg-zinc-50/50 dark:bg-zinc-900/60 hover:border-zinc-400"
                    }`}
                  >
                    <div className="flex-1 min-w-0">
                      <div className="font-bold text-zinc-900 dark:text-zinc-200">
                        No Camera (None)
                      </div>
                      <div className="text-[11px] text-zinc-500 mt-0.5">Saves weight (0g)</div>
                    </div>
                    <div className="text-[11px] text-zinc-500 font-mono shrink-0">0g</div>
                  </div>

                  {filteredCams.map((c) => {
                    const isSelected =
                      !noneSelections.camera &&
                      (selectedCam?.uuid === c.uuid || selectedCam?.id === c.id);
                    return (
                      <div
                        key={c.uuid || c.id}
                        onClick={() => {
                          setSelectedCam(c);
                          setNoneSelections((prev) => ({ ...prev, camera: false }));
                        }}
                        className={`p-3 rounded-xl border cursor-pointer transition-all flex flex-col sm:flex-row sm:items-center justify-between gap-2 ${
                          isSelected
                            ? "border-blue-500 bg-blue-50/50 dark:bg-blue-950/30 ring-1 ring-blue-500"
                            : "border-zinc-200 dark:border-zinc-800 bg-zinc-50/50 dark:bg-zinc-900/60 hover:border-zinc-400 dark:hover:border-zinc-700"
                        }`}
                      >
                        <div className="flex-1 min-w-0">
                          <div className="flex items-center gap-2 flex-wrap">
                            <span className="text-xs font-bold text-zinc-900 dark:text-zinc-100 truncate">
                              {c.name}
                            </span>
                            <span className="text-[10px] px-2 py-0.5 rounded-full font-semibold bg-violet-500/10 text-violet-600 dark:text-violet-400 border border-violet-500/20 shrink-0">
                              {c.protocol || "Analog"}
                            </span>
                          </div>
                          <div className="text-[11px] text-zinc-500 dark:text-zinc-400 mt-0.5">
                            {c.manufacturer} • Weight: {c.weightG}g
                          </div>
                        </div>
                        <div className="text-[11px] text-zinc-500 font-mono shrink-0">
                          {c.weightG}g
                        </div>
                      </div>
                    );
                  })}
                </div>
                {filteredCams.length === 0 && (
                  <p className="text-xs text-zinc-400 italic py-2 text-center">
                    No compatible cameras matching your search.
                  </p>
                )}
              </div>

              {/* 3C. Video Transmitter Antenna(s) */}
              <div className="p-4 rounded-2xl border border-zinc-200 dark:border-zinc-800 bg-white dark:bg-zinc-900 space-y-3">
                <div className="flex items-center justify-between">
                  <h3 className="font-bold text-sm text-zinc-900 dark:text-zinc-100">
                    3C. Video Transmitter Antenna(s)
                  </h3>
                  {selectedVtxAnt && !noneSelections.vtxAntenna ? (
                    <span className="text-xs font-semibold text-emerald-600 dark:text-emerald-400 flex items-center gap-1">
                      <Check size={12} strokeWidth={2.5} />{" "}
                      {vtxAntCount > 1 ? "2x Selected (Dual)" : "Selected"}
                    </span>
                  ) : noneSelections.vtxAntenna ? (
                    <span className="text-xs font-semibold text-zinc-500 flex items-center gap-1">
                      <Check size={12} strokeWidth={2.5} /> None
                    </span>
                  ) : (
                    <span className="text-xs text-zinc-500 dark:text-zinc-400 font-medium">
                      Optional
                    </span>
                  )}
                </div>
                <p className="text-xs text-zinc-500 dark:text-zinc-400">
                  Transmits video signal to FPV goggles. Supports single antenna or dual antenna
                  setups (e.g. DJI O3 / diversity).
                </p>

                <div className="relative">
                  <Search
                    size={14}
                    className="absolute left-3 top-1/2 -translate-y-1/2 text-zinc-400"
                  />
                  <input
                    type="text"
                    value={searchVtxAnt}
                    onChange={(e) => setSearchVtxAnt(e.target.value)}
                    placeholder="Search VTX antennas by connector, polarization, brand..."
                    className="w-full bg-zinc-50 dark:bg-zinc-950 border border-zinc-200 dark:border-zinc-800 rounded-xl pl-9 pr-8 py-1.5 text-xs text-zinc-900 dark:text-zinc-100 placeholder-zinc-400 focus:outline-none focus:ring-1 focus:ring-blue-500"
                  />
                  {searchVtxAnt && (
                    <button
                      type="button"
                      onClick={() => setSearchVtxAnt("")}
                      className="absolute right-2.5 top-1/2 -translate-y-1/2 text-zinc-400 hover:text-zinc-600"
                    >
                      <X size={13} />
                    </button>
                  )}
                </div>

                {selectedVtxAnt && !noneSelections.vtxAntenna && (
                  <div className="flex items-center justify-between p-2.5 rounded-xl bg-blue-50/60 dark:bg-blue-950/30 border border-blue-200 dark:border-blue-900/50 text-xs">
                    <div className="flex items-center gap-2">
                      <span className="font-semibold text-zinc-800 dark:text-zinc-200">
                        Antenna Quantity:
                      </span>
                      <span className="text-zinc-500 dark:text-zinc-400">
                        {vtxAntCount === 1
                          ? "Single Antenna (Standard VTX)"
                          : "Dual Antennas (e.g. DJI O3 / Diversity)"}
                      </span>
                    </div>
                    <div className="flex items-center gap-1 bg-white dark:bg-zinc-900 p-0.5 rounded-lg border border-zinc-200 dark:border-zinc-800 shadow-2xs">
                      <button
                        type="button"
                        onClick={() => setVtxAntCount(1)}
                        className={`px-2.5 py-1 rounded-md text-[11px] font-bold transition-all cursor-pointer ${
                          vtxAntCount === 1
                            ? "bg-blue-600 text-white shadow-xs"
                            : "text-zinc-600 dark:text-zinc-400 hover:text-zinc-900 dark:hover:text-zinc-100"
                        }`}
                      >
                        1x Single
                      </button>
                      <button
                        type="button"
                        onClick={() => setVtxAntCount(2)}
                        className={`px-2.5 py-1 rounded-md text-[11px] font-bold transition-all cursor-pointer ${
                          vtxAntCount === 2
                            ? "bg-blue-600 text-white shadow-xs"
                            : "text-zinc-600 dark:text-zinc-400 hover:text-zinc-900 dark:hover:text-zinc-100"
                        }`}
                      >
                        2x Dual
                      </button>
                    </div>
                  </div>
                )}

                <div className="overflow-y-auto max-h-64 sm:max-h-72 space-y-2 pr-1.5 focus:outline-none">
                  <div
                    onClick={() => {
                      setSelectedVtxAnt(null);
                      setNoneSelections((prev) => ({ ...prev, vtxAntenna: true }));
                    }}
                    className={`p-3 rounded-xl border cursor-pointer text-xs transition-all flex flex-col sm:flex-row sm:items-center justify-between gap-2 ${
                      noneSelections.vtxAntenna
                        ? "border-blue-500 bg-blue-50/50 dark:bg-blue-950/30 ring-1 ring-blue-500"
                        : "border-zinc-200 dark:border-zinc-800 bg-zinc-50/50 dark:bg-zinc-900/60 hover:border-zinc-400"
                    }`}
                  >
                    <div className="flex-1 min-w-0">
                      <div className="font-bold text-zinc-900 dark:text-zinc-200">
                        No Antenna (None)
                      </div>
                      <div className="text-[11px] text-zinc-500 mt-0.5">Saves weight (0g)</div>
                    </div>
                    <div className="text-[11px] text-zinc-500 font-mono shrink-0">0g</div>
                  </div>

                  {filteredVtxAnts.map((a) => {
                    const isSelected =
                      !noneSelections.vtxAntenna &&
                      (selectedVtxAnt?.uuid === a.uuid || selectedVtxAnt?.id === a.id);
                    return (
                      <div
                        key={a.uuid || a.id}
                        onClick={() => {
                          setSelectedVtxAnt(a);
                          setNoneSelections((prev) => ({ ...prev, vtxAntenna: false }));
                        }}
                        className={`p-3 rounded-xl border cursor-pointer transition-all flex flex-col sm:flex-row sm:items-center justify-between gap-2 ${
                          isSelected
                            ? "border-blue-500 bg-blue-50/50 dark:bg-blue-950/30 ring-1 ring-blue-500"
                            : "border-zinc-200 dark:border-zinc-800 bg-zinc-50/50 dark:bg-zinc-900/60 hover:border-zinc-400 dark:hover:border-zinc-700"
                        }`}
                      >
                        <div className="flex-1 min-w-0">
                          <div className="flex items-center gap-2 flex-wrap">
                            <span className="text-xs font-bold text-zinc-900 dark:text-zinc-100 truncate">
                              {a.name}
                            </span>
                            <span className="text-[10px] px-2 py-0.5 rounded-full font-semibold bg-pink-500/10 text-pink-600 dark:text-pink-400 border border-pink-500/20 shrink-0">
                              {a.polarization || "RHCP"}
                            </span>
                          </div>
                          <div className="text-[11px] text-zinc-500 dark:text-zinc-400 mt-0.5">
                            {a.manufacturer} • Weight:{" "}
                            {((a.weightG || 0) * (isSelected ? vtxAntCount : 1)).toFixed(1)}g{" "}
                            {isSelected && vtxAntCount > 1 ? `(${vtxAntCount}x ${a.weightG}g)` : ""}
                          </div>
                        </div>
                        <div className="text-[11px] text-zinc-500 font-mono shrink-0">
                          {((a.weightG || 0) * (isSelected ? vtxAntCount : 1)).toFixed(1)}g
                        </div>
                      </div>
                    );
                  })}
                </div>
                {filteredVtxAnts.length === 0 && (
                  <p className="text-xs text-zinc-400 italic py-2 text-center">
                    No compatible VTX antennas matching your search.
                  </p>
                )}
              </div>
            </div>
          )}

          {/* STAGE 4: Review & Finalize (BOM & Save) */}
          {activeStage === 4 && (
            <div className="p-5 rounded-2xl border border-zinc-200 dark:border-zinc-800 bg-white dark:bg-zinc-900 space-y-5">
              <div>
                <h3 className="font-bold text-base text-zinc-900 dark:text-zinc-100">
                  Review & Save Build
                </h3>
                <p className="text-xs text-zinc-500 dark:text-zinc-400 mt-0.5">
                  Verify your Bill of Materials, provide a title and description, and save to the
                  database.
                </p>
              </div>

              {saveError && (
                <div className="p-3 rounded-xl border border-red-500/30 bg-red-500/10 text-red-600 dark:text-red-400 text-xs">
                  {saveError}
                </div>
              )}

              <div className="space-y-3">
                <div>
                  <label className="block text-xs font-semibold text-zinc-700 dark:text-zinc-300 mb-1">
                    Build Name <span className="text-red-500">*</span>
                  </label>
                  <input
                    type="text"
                    value={buildName}
                    onChange={(e) => setBuildName(e.target.value)}
                    placeholder="e.g. My Freestyle 5-Inch"
                    className="w-full px-3.5 py-2 rounded-xl bg-zinc-50 dark:bg-zinc-950 border border-zinc-200 dark:border-zinc-800 text-sm font-semibold text-zinc-900 dark:text-zinc-100 focus:outline-none focus:ring-2 focus:ring-blue-500"
                  />
                </div>

                <div>
                  <label className="block text-xs font-semibold text-zinc-700 dark:text-zinc-300 mb-1">
                    Description
                  </label>
                  <textarea
                    rows={2}
                    value={buildDesc}
                    onChange={(e) => setBuildDesc(e.target.value)}
                    placeholder="Short description of this build setup..."
                    className="w-full px-3.5 py-2 rounded-xl bg-zinc-50 dark:bg-zinc-950 border border-zinc-200 dark:border-zinc-800 text-xs text-zinc-900 dark:text-zinc-100 focus:outline-none focus:ring-2 focus:ring-blue-500"
                  />
                </div>
              </div>

              {/* Bill of Materials (BOM) Table */}
              <div className="rounded-xl border border-zinc-200 dark:border-zinc-800 overflow-hidden text-xs">
                <div className="px-4 py-2.5 bg-zinc-100 dark:bg-zinc-800/80 border-b border-zinc-200 dark:border-zinc-800 flex justify-between font-bold text-zinc-700 dark:text-zinc-300">
                  <span>Bill of Materials (BOM)</span>
                  <span>
                    Dry Weight:{" "}
                    <strong className="text-blue-600 dark:text-blue-400">
                      {dryWeightG.toFixed(1)}g
                    </strong>
                  </span>
                </div>
                <div className="divide-y divide-zinc-200 dark:divide-zinc-800/60 p-2">
                  {selectedFrame && (
                    <div className="py-1.5 px-2 flex justify-between">
                      <span className="text-zinc-700 dark:text-zinc-300">
                        Frame: {selectedFrame.name}
                      </span>
                      <span className="text-zinc-500 font-mono">{selectedFrame.weightG}g</span>
                    </div>
                  )}
                  {selectedMotor && (
                    <div className="py-1.5 px-2 flex justify-between">
                      <span className="text-zinc-700 dark:text-zinc-300">
                        Motors: {selectedMotor.name} ({motorCount}x)
                      </span>
                      <span className="text-zinc-500 font-mono">
                        {((selectedMotor.weightG || 0) * motorCount).toFixed(1)}g
                      </span>
                    </div>
                  )}
                  {selectedProp && (
                    <div className="py-1.5 px-2 flex justify-between">
                      <span className="text-zinc-700 dark:text-zinc-300">
                        Propellers: {selectedProp.name} ({motorCount}x)
                      </span>
                      <span className="text-zinc-500 font-mono">
                        {((selectedProp.weightG || 0) * motorCount).toFixed(1)}g
                      </span>
                    </div>
                  )}
                  {selectedFc && (
                    <div className="py-1.5 px-2 flex justify-between">
                      <span className="text-zinc-700 dark:text-zinc-300">
                        Flight Controller: {selectedFc.name}
                      </span>
                      <span className="text-zinc-500 font-mono">{selectedFc.weightG}g</span>
                    </div>
                  )}
                  {useIntegratedEsc ? (
                    <div className="py-1.5 px-2 flex justify-between text-zinc-500">
                      <span>
                        ESC:{" "}
                        {integratedEsc
                          ? `${integratedEsc.name} (Integrated ${integratedEsc.motorCurrentMaxA || 20}A)`
                          : "Using Integrated FC ESC"}
                      </span>
                      <span className="font-mono">0g</span>
                    </div>
                  ) : selectedEsc ? (
                    <div className="py-1.5 px-2 flex justify-between">
                      <span className="text-zinc-700 dark:text-zinc-300">
                        ESC: {selectedEsc.name}
                      </span>
                      <span className="text-zinc-500 font-mono">{selectedEsc.weightG}g</span>
                    </div>
                  ) : null}
                  {useIntegratedRx ? (
                    <div className="py-1.5 px-2 flex justify-between text-zinc-500">
                      <span>
                        Receiver:{" "}
                        {integratedRx
                          ? `${integratedRx.name} (Integrated ${integratedRx.protocol || "ExpressLRS"} @ ${formatFrequencyBand(integratedRx.frequencyBandMhz)})`
                          : "Using Integrated FC Receiver"}
                      </span>
                      <span className="font-mono">0g</span>
                    </div>
                  ) : selectedRx ? (
                    <div className="py-1.5 px-2 flex justify-between">
                      <span className="text-zinc-700 dark:text-zinc-300">
                        Receiver: {selectedRx.name} ({selectedRx.protocol} @{" "}
                        {formatFrequencyBand(selectedRx.frequencyBandMhz)})
                      </span>
                      <span className="text-zinc-500 font-mono">{selectedRx.weightG}g</span>
                    </div>
                  ) : null}
                  {selectedRxAnt && !noneSelections.rxAntenna && (
                    <div className="py-1.5 px-2 flex justify-between">
                      <span className="text-zinc-700 dark:text-zinc-300">
                        RX Antenna: {selectedRxAnt.name} ({rxAntCount}x)
                      </span>
                      <span className="text-zinc-500 font-mono">
                        {((selectedRxAnt.weightG || 0) * rxAntCount).toFixed(1)}g
                      </span>
                    </div>
                  )}
                  {selectedGps && !noneSelections.gps && (
                    <div className="py-1.5 px-2 flex justify-between">
                      <span className="text-zinc-700 dark:text-zinc-300">
                        GPS: {selectedGps.name}
                      </span>
                      <span className="text-zinc-500 font-mono">{selectedGps.weightG}g</span>
                    </div>
                  )}
                  {useIntegratedVtx ? (
                    <div className="py-1.5 px-2 flex justify-between text-zinc-500">
                      <span>
                        VTX:{" "}
                        {integratedVtx
                          ? `${integratedVtx.name} (Integrated ${integratedVtx.maxPowerMw || 400}mW)`
                          : "Using Integrated FC VTX"}
                      </span>
                      <span className="font-mono">0g</span>
                    </div>
                  ) : selectedVtx && !noneSelections.vtx ? (
                    <div className="py-1.5 px-2 flex justify-between">
                      <span className="text-zinc-700 dark:text-zinc-300">
                        VTX: {selectedVtx.name}
                      </span>
                      <span className="text-zinc-500 font-mono">{selectedVtx.weightG}g</span>
                    </div>
                  ) : null}
                  {selectedCam && !noneSelections.camera && (
                    <div className="py-1.5 px-2 flex justify-between">
                      <span className="text-zinc-700 dark:text-zinc-300">
                        Camera: {selectedCam.name}
                      </span>
                      <span className="text-zinc-500 font-mono">{selectedCam.weightG}g</span>
                    </div>
                  )}
                  {selectedVtxAnt && !noneSelections.vtxAntenna && (
                    <div className="py-1.5 px-2 flex justify-between">
                      <span className="text-zinc-700 dark:text-zinc-300">
                        VTX Antenna: {selectedVtxAnt.name} ({vtxAntCount}x)
                      </span>
                      <span className="text-zinc-500 font-mono">
                        {((selectedVtxAnt.weightG || 0) * vtxAntCount).toFixed(1)}g
                      </span>
                    </div>
                  )}
                </div>
              </div>

              {/* Save Button */}
              <button
                type="button"
                onClick={handleSaveBuild}
                disabled={isSaving}
                className="w-full py-3 rounded-xl bg-blue-600 hover:bg-blue-500 text-white font-bold text-sm shadow-md transition-all flex items-center justify-center gap-2 cursor-pointer disabled:opacity-50"
              >
                <Save size={16} />
                <span>
                  {isSaving ? "Saving to Database..." : "Create & Save Build to Database"}
                </span>
              </button>
            </div>
          )}

          {/* Bottom Stage Navigation Controls */}
          <div className="p-4 rounded-xl border border-zinc-200 dark:border-zinc-800 bg-white dark:bg-zinc-900 flex flex-col sm:flex-row items-center justify-between gap-3">
            <button
              type="button"
              disabled={activeStage === 0}
              onClick={handlePrevStage}
              className="w-full sm:w-auto px-4 py-2 rounded-xl border border-zinc-200 dark:border-zinc-800 hover:bg-zinc-100 dark:hover:bg-zinc-800 text-xs font-semibold text-zinc-700 dark:text-zinc-300 disabled:opacity-40 disabled:cursor-not-allowed transition-colors"
            >
              <ArrowLeft size={14} className="inline mr-1" />
              Previous Stage
            </button>

            {activeStage < 4 && (
              <button
                type="button"
                onClick={handleNextStage}
                disabled={
                  (activeStage === 1 && !stage1Complete) || (activeStage === 2 && !stage2Complete)
                }
                className="w-full sm:w-auto px-5 py-2.5 rounded-xl bg-blue-600 hover:bg-blue-500 text-white text-xs font-bold shadow-xs transition-all disabled:opacity-40 disabled:cursor-not-allowed"
              >
                <span>
                  {activeStage === 0
                    ? "Next: Airframe & Propulsion →"
                    : activeStage === 1
                      ? "Next: Flight Electronics →"
                      : activeStage === 2
                        ? "Next: Video →"
                        : "Next: Review & Save →"}
                </span>
              </button>
            )}
          </div>
        </div>

        {/* Right Column: Live Build Evaluator */}
        <div className="lg:col-span-4 space-y-4 lg:sticky lg:top-4">
          {/* Prominent Reset Wizard Action Bar (Distinguished from Live Evaluation) */}
          <div className="p-3 rounded-2xl border border-red-500/30 bg-gradient-to-r from-red-50 dark:from-red-950/40 via-white dark:via-zinc-900/80 to-zinc-50 dark:to-zinc-900/90 shadow-xs flex items-center justify-between gap-3">
            <div className="flex items-center gap-2.5 min-w-0">
              <div className="w-8 h-8 rounded-xl bg-red-100 dark:bg-red-500/20 border border-red-200 dark:border-red-500/30 flex items-center justify-center shrink-0 text-red-600 dark:text-red-400">
                <RotateCcw size={16} />
              </div>
              <div className="min-w-0">
                <div className="text-xs font-bold text-zinc-900 dark:text-zinc-100">
                  Reset Wizard
                </div>
                <div className="text-[11px] text-zinc-500 dark:text-zinc-400 truncate">
                  Clear all selections &amp; return to Stage 0
                </div>
              </div>
            </div>
            <button
              type="button"
              onClick={resetWizard}
              className="px-3 py-1.5 rounded-xl bg-red-600 hover:bg-red-500 active:scale-95 text-white text-xs font-bold transition-all shadow-xs shrink-0 flex items-center gap-1.5 cursor-pointer"
              title="Clear all fields and return to Stage 0"
            >
              <RotateCcw size={13} />
              <span>Reset</span>
            </button>
          </div>

          {/* Distinct Live Evaluation Section Header */}
          <div className="flex items-center justify-between px-1 pt-1">
            <span className="text-xs font-bold uppercase tracking-wider text-zinc-500 dark:text-zinc-400">
              Live Evaluation
            </span>
            <span className="text-[11px] font-mono text-zinc-400 dark:text-zinc-500">
              Real-time Telemetry
            </span>
          </div>

          {isWasmDelayed && !isWasmReady && (
            <WasmLoadingIndicator progress={wasmProgress} className="mb-2" />
          )}

          <div className="rounded-2xl border border-zinc-200 dark:border-zinc-800 bg-gradient-to-b from-white dark:from-zinc-900 to-zinc-50 dark:to-zinc-950 p-4 space-y-4 shadow-lg">
            {/* Header */}
            <div className="flex items-center justify-between border-b border-zinc-200 dark:border-zinc-800 pb-2.5">
              <div className="flex items-center gap-2">
                <span className="w-2.5 h-2.5 rounded-full bg-blue-500"></span>
                <h3 className="text-xs font-bold uppercase tracking-wider text-zinc-800 dark:text-zinc-200">
                  Live Build Evaluator
                </h3>
              </div>
              {isWasmReady ? (
                <span className="text-[10px] font-mono font-semibold text-emerald-600 dark:text-emerald-400 bg-emerald-500/10 px-2 py-0.5 rounded-full border border-emerald-500/20">
                  Physics Engine (0ms)
                </span>
              ) : (
                <span className="text-[11px] font-mono text-zinc-400">Physics Engine</span>
              )}
            </div>

            {/* Test Battery Selection */}
            <div className="space-y-1.5 text-xs">
              <label className="block font-semibold text-zinc-700 dark:text-zinc-300">
                Test Battery (Runtime Parameter)
              </label>
              <select
                value={selectedBatteryId}
                onChange={(e) => setSelectedBatteryId(e.target.value)}
                className="w-full px-3 py-2 rounded-xl bg-zinc-50 dark:bg-zinc-950 border border-zinc-200 dark:border-zinc-800 text-xs text-zinc-900 dark:text-zinc-100 focus:outline-none focus:ring-1 focus:ring-blue-500 font-mono"
              >
                {batteries.map((b) => (
                  <option key={b.id || b.uuid} value={b.id || b.uuid}>
                    {b.name} ({b.weightG}g)
                  </option>
                ))}
              </select>
            </div>

            {/* Runtime Payload Textbox with +/- and Quick Presets */}
            <div className="bg-zinc-50 dark:bg-zinc-900/60 p-3 rounded-xl border border-zinc-200 dark:border-zinc-800 space-y-2 text-xs">
              <div className="flex items-center justify-between">
                <span className="font-semibold text-zinc-700 dark:text-zinc-300">
                  Payload Simulation
                </span>
                <span className="font-mono font-bold text-blue-600 dark:text-blue-400">
                  +{payloadWeightG}g
                </span>
              </div>
              <div className="flex items-center gap-1.5">
                <button
                  type="button"
                  onClick={() => adjustPayload(-10)}
                  disabled={payloadWeightG <= 0}
                  aria-label="Remove 10g"
                  title="Remove 10g"
                  className="w-7 h-7 flex items-center justify-center rounded-lg border border-zinc-200 dark:border-zinc-700 bg-white dark:bg-zinc-800 hover:bg-zinc-100 dark:hover:bg-zinc-700 disabled:opacity-30 disabled:cursor-not-allowed text-zinc-700 dark:text-zinc-200 transition-colors cursor-pointer shrink-0 font-bold"
                >
                  -
                </button>
                <div className="relative flex-1 flex items-center">
                  <input
                    type="text"
                    inputMode="decimal"
                    value={payloadInput}
                    onChange={(e) => handlePayloadInput(e.target.value)}
                    placeholder="0"
                    className="w-full bg-white dark:bg-zinc-950 border border-zinc-200 dark:border-zinc-700 rounded-lg px-2.5 py-1 pr-6 text-xs font-mono text-zinc-900 dark:text-zinc-100 focus:outline-none focus:ring-1 focus:ring-blue-500 text-center"
                  />
                  <span className="absolute right-2 text-xs text-zinc-400 font-mono pointer-events-none select-none">
                    g
                  </span>
                </div>
                <button
                  type="button"
                  onClick={() => adjustPayload(10)}
                  aria-label="Add 10g"
                  title="Add 10g"
                  className="w-7 h-7 flex items-center justify-center rounded-lg border border-zinc-200 dark:border-zinc-700 bg-white dark:bg-zinc-800 hover:bg-zinc-100 dark:hover:bg-zinc-700 text-zinc-700 dark:text-zinc-200 transition-colors cursor-pointer shrink-0 font-bold"
                >
                  +
                </button>
              </div>
              <div className="flex gap-1 text-[10px]">
                <button
                  type="button"
                  onClick={() => setPresetPayload(0)}
                  className={`px-2 py-0.5 rounded cursor-pointer transition-colors font-medium ${
                    payloadWeightG === 0
                      ? "bg-blue-600 text-white"
                      : "bg-zinc-200 dark:bg-zinc-800 text-zinc-600 dark:text-zinc-400 hover:text-zinc-900 dark:hover:text-zinc-200"
                  }`}
                >
                  Bare (0g)
                </button>
                <button
                  type="button"
                  onClick={() => setPresetPayload(16)}
                  className={`px-2 py-0.5 rounded cursor-pointer transition-colors font-medium ${
                    payloadWeightG === 16
                      ? "bg-blue-600 text-white"
                      : "bg-zinc-200 dark:bg-zinc-800 text-zinc-600 dark:text-zinc-400 hover:text-zinc-900 dark:hover:text-zinc-200"
                  }`}
                >
                  Thumb (+16g)
                </button>
                <button
                  type="button"
                  onClick={() => setPresetPayload(133)}
                  className={`px-2 py-0.5 rounded cursor-pointer transition-colors font-medium ${
                    payloadWeightG === 133
                      ? "bg-blue-600 text-white"
                      : "bg-zinc-200 dark:bg-zinc-800 text-zinc-600 dark:text-zinc-400 hover:text-zinc-900 dark:hover:text-zinc-200"
                  }`}
                >
                  GoPro (+133g)
                </button>
              </div>
            </div>

            {/* TWR Hero Metric */}
            <div className="text-center p-3 rounded-xl bg-zinc-50 dark:bg-zinc-900/60 border border-zinc-200 dark:border-zinc-800">
              <div className="text-[11px] uppercase font-semibold text-zinc-500">
                Estimated Thrust-to-Weight
              </div>
              <div className="text-3xl font-extrabold text-blue-600 dark:text-blue-400 mt-1">
                {evaluation ? `${evaluation.thrustToWeightRatio.toFixed(2)} : 1` : "-- : 1"}
              </div>
              <span className="inline-block mt-1 px-2.5 py-0.5 rounded-full text-[11px] font-bold bg-zinc-200 dark:bg-zinc-800 text-zinc-600 dark:text-zinc-400">
                {evaluation
                  ? evaluation.thrustToWeightRatio >= 5
                    ? "🚀 Extreme Punchout"
                    : evaluation.thrustToWeightRatio >= 3
                      ? "⚡ Agile Freestyle"
                      : "Cruiser / Sluggish"
                  : "Select Propulsion Parts"}
              </span>
            </div>

            {/* Telemetry Metrics Grid */}
            <div className="grid grid-cols-2 gap-2 text-xs">
              <div className="p-2.5 rounded-xl bg-zinc-50 dark:bg-zinc-900/40 border border-zinc-200 dark:border-zinc-800">
                <span className="text-zinc-500">Dry Weight</span>
                <div className="text-sm font-bold text-zinc-900 dark:text-zinc-100 font-mono mt-0.5">
                  {dryWeightG.toFixed(1)}g
                </div>
              </div>
              <div className="p-2.5 rounded-xl bg-zinc-50 dark:bg-zinc-900/40 border border-zinc-200 dark:border-zinc-800">
                <span className="text-zinc-500">All-Up Weight</span>
                <div className="text-sm font-bold text-zinc-900 dark:text-zinc-100 font-mono mt-0.5">
                  {evaluation
                    ? `${evaluation.allUpWeightG.toFixed(1)}g`
                    : activeBattery
                      ? `${(dryWeightG + (activeBattery.weightG || 0) + payloadWeightG).toFixed(1)}g`
                      : `${dryWeightG}g`}
                </div>
              </div>
              <div className="p-2.5 rounded-xl bg-zinc-50 dark:bg-zinc-900/40 border border-zinc-200 dark:border-zinc-800">
                <span className="text-zinc-500">Hover Throttle</span>
                <div className="text-sm font-bold text-emerald-600 dark:text-emerald-400 font-mono mt-0.5">
                  {evaluation ? `${evaluation.hoverThrottlePercent.toFixed(1)}%` : "--%"}
                </div>
              </div>
              <div className="p-2.5 rounded-xl bg-zinc-50 dark:bg-zinc-900/40 border border-zinc-200 dark:border-zinc-800">
                <span className="text-zinc-500">Flight Time</span>
                <div className="text-sm font-bold text-amber-600 dark:text-amber-400 font-mono mt-0.5">
                  {evaluation
                    ? `${evaluation.minFlightTimeMin.toFixed(1)} - ${evaluation.maxFlightTimeMin.toFixed(1)}m`
                    : "-- min"}
                </div>
              </div>
            </div>

            {/* Compatibility Rule Checklist */}
            <div className="p-3 rounded-xl bg-zinc-50 dark:bg-zinc-900/70 border border-zinc-200 dark:border-zinc-800 space-y-1.5 text-[11px]">
              <div className="font-bold text-zinc-700 dark:text-zinc-300 flex justify-between">
                <span>Rule Engine Checks</span>
                <span className="text-zinc-500">
                  {draftBuild ? "Evaluated" : "Pending Stage 1"}
                </span>
              </div>
              <div className="flex justify-between text-zinc-600 dark:text-zinc-400">
                <span>Prop vs Frame Size</span>
                <span
                  className={
                    selectedFrame && selectedProp
                      ? "text-emerald-600 dark:text-emerald-400 font-bold"
                      : "text-zinc-400"
                  }
                >
                  {selectedFrame && selectedProp ? "✓ Compatible" : "--"}
                </span>
              </div>
              <div className="flex justify-between text-zinc-600 dark:text-zinc-400">
                <span>ESC Amperage Rating</span>
                <span
                  className={
                    selectedEsc || useIntegratedEsc
                      ? "text-emerald-600 dark:text-emerald-400 font-bold"
                      : "text-zinc-400"
                  }
                >
                  {selectedEsc || useIntegratedEsc ? "✓ Sufficient" : "--"}
                </span>
              </div>
              <div className="flex justify-between text-zinc-600 dark:text-zinc-400">
                <span>Stack Mounting Fit</span>
                <span
                  className={
                    selectedFc
                      ? "text-emerald-600 dark:text-emerald-400 font-bold"
                      : "text-zinc-400"
                  }
                >
                  {selectedFc ? "✓ Fits frame" : "--"}
                </span>
              </div>
              {compatibilityData?.messages && compatibilityData.messages.length > 0 && (
                <div className="pt-1 border-t border-zinc-200 dark:border-zinc-800 space-y-1">
                  {compatibilityData.messages.map((m, idx) => (
                    <div key={idx} className="text-amber-600 dark:text-amber-400 text-[10px]">
                      ⚠️ {m.message}
                    </div>
                  ))}
                </div>
              )}
            </div>

            {/* Live Dynamic Compatibility Filter Display */}
            {isWasmReady && (
              <div className="p-3 rounded-xl bg-zinc-50 dark:bg-zinc-900/70 border border-zinc-200 dark:border-zinc-800 space-y-1.5 text-[11px]">
                <div className="flex items-center justify-between">
                  <span className="font-bold text-zinc-700 dark:text-zinc-300 uppercase tracking-wider text-[10px]">
                    Dynamic Compatibility Filter
                  </span>
                  <span className="text-[10px] font-mono px-2 py-0.5 rounded-full bg-emerald-500/10 text-emerald-600 dark:text-emerald-400 border border-emerald-500/20 font-semibold">
                    {activeStage === 1
                      ? "Propeller Target"
                      : activeStage === 2
                        ? "ESC Target"
                        : activeStage === 3
                          ? "Camera/VTX Target"
                          : "Battery Target"}
                  </span>
                </div>
                <div className="bg-zinc-100 dark:bg-zinc-950 p-2 rounded-lg border border-zinc-200 dark:border-zinc-800 font-mono text-[10px] text-indigo-600 dark:text-indigo-400 break-all select-all">
                  <code>
                    {(activeStage === 1 && propCelFilter) ||
                      (activeStage === 2 && escCelFilter) ||
                      (activeStage === 3 && (camCelFilter || vtxCelFilter)) ||
                      batteryCelFilter ||
                      "CEL engine active: all hardware compatible"}
                  </code>
                </div>
              </div>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}
