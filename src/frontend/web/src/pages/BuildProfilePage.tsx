import { useParams, Link } from "react-router-dom";
import { useState, useMemo, useEffect } from "react";
import { useQuery } from "@connectrpc/connect-query";
import { keepPreviousData } from "@tanstack/react-query";
import { getBuild } from "../gen/quadsmith/build-BuildService_connectquery";
import {
  evaluateBuild,
  getBuildElectricalLimits,
} from "../gen/quadsmith/evaluator-EvaluatorService_connectquery";
import { getFrame } from "../gen/quadsmith/frame-FrameService_connectquery";
import { getMotor } from "../gen/quadsmith/motor-MotorService_connectquery";
import { listBatteries } from "../gen/quadsmith/battery-BatteryService_connectquery";
import { getFlightController } from "../gen/quadsmith/flight_controller-FlightControllerService_connectquery";
import { getElectronicSpeedController } from "../gen/quadsmith/electronic_speed_controller-ElectronicSpeedControllerService_connectquery";
import { getPropeller } from "../gen/quadsmith/propeller-PropellerService_connectquery";
import { getCamera } from "../gen/quadsmith/camera-CameraService_connectquery";
import { getVideoTransmitter } from "../gen/quadsmith/video_transmitter-VideoTransmitterService_connectquery";
import { getReceiver } from "../gen/quadsmith/receiver-ReceiverService_connectquery";
import { getAntenna } from "../gen/quadsmith/antenna-AntennaService_connectquery";
import { getGpsReceiver } from "../gen/quadsmith/gps_receiver-GpsReceiverService_connectquery";
import { ReferenceLinkType } from "../gen/quadsmith/reference_link_pb";
import { SystemMessageSeverity } from "../gen/quadsmith/evaluator_pb";
import { MediaGallery } from "../components/MediaGallery";
import { BatteryPickerModal } from "../components/BatteryPickerModal";
import { buildBatteryCelFilter } from "../lib/batteryFilter";
import { getTwrDescription } from "../lib/format";
import {
  ChevronRight,
  Gauge,
  Clock,
  Weight,
  ExternalLink,
  AlertTriangle,
  AlertOctagon,
  CheckCircle2,
  Sliders,
  Layers,
  Cpu,
  Radio,
  Camera as CameraIcon,
  Compass,
  Zap,
  Rocket,
  Wind,
  Plus,
  Minus,
  RefreshCw,
  Search,
} from "lucide-react";

const LINK_TYPE_LABELS: Record<number, string> = {
  [ReferenceLinkType.PURCHASE]: "Purchase",
  [ReferenceLinkType.PRODUCT_PAGE]: "Official Product Page",
  [ReferenceLinkType.DOCUMENTATION]: "Documentation",
  [ReferenceLinkType.FORUM_POST]: "Forum Discussion",
  [ReferenceLinkType.REVIEW]: "Review",
  [ReferenceLinkType.OTHER]: "Other",
  [ReferenceLinkType.UNSPECIFIED]: "Reference Link",
};

interface BomComponentProps {
  label: string;
  collectionId: string;
  uuid: string;
  queryMethod: any;
  icon: React.ReactNode;
  subtitle?: (item: any) => string;
}

function BomComponentCard({
  label,
  collectionId,
  uuid,
  queryMethod,
  icon,
  subtitle,
}: BomComponentProps) {
  const {
    data: rawItem,
    isLoading,
    error,
  } = useQuery(queryMethod, { id: uuid }, { enabled: !!uuid });
  const item = rawItem as any;

  if (!uuid) return null;

  return (
    <div className="flex items-center justify-between p-3.5 rounded-xl border border-zinc-200 dark:border-zinc-800 bg-white dark:bg-zinc-900/60 hover:border-zinc-300 dark:hover:border-zinc-700 transition-colors">
      <div className="flex items-center gap-3 min-w-0">
        <div className="w-9 h-9 rounded-lg bg-zinc-100 dark:bg-zinc-800 flex items-center justify-center text-zinc-500 shrink-0">
          {icon}
        </div>
        <div className="min-w-0">
          <div className="text-xs uppercase font-semibold text-zinc-400 dark:text-zinc-500">
            {label}
          </div>
          {isLoading ? (
            <div className="text-sm text-zinc-400 animate-pulse">Loading {label}...</div>
          ) : error || !item ? (
            <div className="text-sm font-mono text-zinc-600 dark:text-zinc-400 truncate max-w-xs">
              {uuid}
            </div>
          ) : (
            <div>
              <Link
                to={`/components/hardware/${collectionId}/${item.id || item.uuid}`}
                className="text-sm font-bold text-zinc-900 dark:text-zinc-100 hover:text-blue-600 dark:hover:text-blue-400 transition-colors truncate block"
              >
                {item.name || item.id}
              </Link>
              {subtitle && <div className="text-xs text-zinc-500 truncate">{subtitle(item)}</div>}
            </div>
          )}
        </div>
      </div>

      {item && (
        <Link
          to={`/components/hardware/${collectionId}/${item.id || item.uuid}`}
          className="text-xs text-zinc-400 hover:text-blue-600 dark:hover:text-blue-400 p-1.5 rounded-md hover:bg-zinc-100 dark:hover:bg-zinc-800 transition-colors shrink-0"
          title={`View ${item.name || item.id} in catalog`}
        >
          <ExternalLink size={14} />
        </Link>
      )}
    </div>
  );
}

export function BuildProfilePage() {
  const { buildId } = useParams<{ buildId: string }>();
  const [payloadInput, setPayloadInput] = useState<string>("0");
  const payloadWeightG = Math.max(0, parseFloat(payloadInput) || 0);

  const [selectedBatteryId, setSelectedBatteryId] = useState<string>("");
  const [isBatteryModalOpen, setIsBatteryModalOpen] = useState(false);

  const {
    data: build,
    isLoading: isLoadingBuild,
    error: buildError,
  } = useQuery(getBuild, { id: buildId || "" }, { enabled: !!buildId });

  const { data: electricalLimits } = useQuery(
    getBuildElectricalLimits,
    { build, buildId: build?.id || buildId },
    { enabled: !!build },
  );

  const batteryFilter = useMemo(() => {
    return buildBatteryCelFilter(electricalLimits);
  }, [electricalLimits]);

  const { data: batteryResponse, isLoading: isLoadingBatteries } = useQuery(
    listBatteries,
    {
      filter: batteryFilter,
      sort: ["weight_g"],
      pageSize: 100,
    },
    { enabled: !!electricalLimits },
  );
  const compatibleBatteries = batteryResponse?.batteries || [];

  // Lightest compatible battery selected by default
  useEffect(() => {
    if (!selectedBatteryId) {
      if (electricalLimits?.defaultBatteryId) {
        setSelectedBatteryId(electricalLimits.defaultBatteryId);
      } else if (compatibleBatteries.length > 0) {
        setSelectedBatteryId(compatibleBatteries[0].id || compatibleBatteries[0].uuid);
      }
    }
  }, [electricalLimits, compatibleBatteries, selectedBatteryId]);

  const activeBattery = useMemo(() => {
    return compatibleBatteries.find(
      (b) => b.id === selectedBatteryId || b.uuid === selectedBatteryId,
    );
  }, [compatibleBatteries, selectedBatteryId]);

  const {
    data: evaluation,
    isLoading: isLoadingEvaluation,
    isFetching: isFetchingEvaluation,
    error: evalError,
  } = useQuery(
    evaluateBuild,
    {
      build,
      buildId: build?.id || buildId,
      payloadWeightG,
      batteryId: selectedBatteryId,
    },
    {
      enabled: !!build && !!selectedBatteryId,
      placeholderData: keepPreviousData,
    },
  );

  if (isLoadingBuild) {
    return (
      <div className="max-w-4xl mx-auto py-8">
        <p className="text-zinc-500 animate-pulse">Loading build profile...</p>
      </div>
    );
  }

  if (buildError || !build) {
    return (
      <div className="max-w-4xl mx-auto py-8">
        <h1 className="text-2xl font-bold text-red-500 mb-2">Build Not Found</h1>
        <p className="text-zinc-500 mb-4">
          Could not find a build profile for &quot;{buildId}&quot;.
        </p>
        <Link
          to="/"
          className="inline-flex items-center gap-1.5 text-sm font-medium text-blue-600 hover:underline"
        >
          ← Return to Builds Feed
        </Link>
      </div>
    );
  }

  return (
    <div className="max-w-4xl mx-auto">
      {/* Breadcrumb Navigation */}
      <nav
        aria-label="Breadcrumb"
        className="mb-4 text-sm text-zinc-500 dark:text-zinc-400 flex items-center gap-1.5 flex-wrap"
      >
        <Link
          to="/"
          className="hover:text-zinc-900 dark:hover:text-zinc-100 hover:underline transition-colors"
        >
          Home
        </Link>
        <ChevronRight size={14} className="text-zinc-400 dark:text-zinc-500 shrink-0" />
        <Link
          to="/"
          className="hover:text-zinc-900 dark:hover:text-zinc-100 hover:underline transition-colors"
        >
          Builds
        </Link>
        <ChevronRight size={14} className="text-zinc-400 dark:text-zinc-500 shrink-0" />
        <span className="text-zinc-900 dark:text-zinc-100 font-medium truncate max-w-md">
          {build.name || build.id}
        </span>
      </nav>

      {/* Build Profile Header */}
      <div className="mb-8">
        <div className="flex items-center gap-2 mb-1">
          <span className="text-xs font-mono px-2 py-0.5 rounded bg-blue-50 dark:bg-blue-950/60 text-blue-600 dark:text-blue-400 border border-blue-200/50 dark:border-blue-900/50">
            @{build.id}
          </span>
        </div>
        <h1 className="text-3xl sm:text-4xl font-extrabold tracking-tight text-zinc-900 dark:text-zinc-50">
          {build.name}
        </h1>
        {build.description && (
          <p className="mt-3 text-base sm:text-lg text-zinc-600 dark:text-zinc-400 leading-relaxed max-w-3xl">
            {build.description}
          </p>
        )}
      </div>

      {/* ============================================================ */}
      {/* BUILD EVALUATION SECTION                                      */}
      {/* ============================================================ */}
      <section className="mb-10 p-6 rounded-2xl border border-zinc-200 dark:border-zinc-800 bg-zinc-50/60 dark:bg-zinc-900/40 shadow-xs">
        <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4 mb-6 pb-4 border-b border-zinc-200 dark:border-zinc-800">
          <div>
            <h2 className="text-xl font-bold text-zinc-900 dark:text-zinc-100 flex items-center gap-2">
              <Gauge className="text-blue-600 dark:text-blue-400" size={22} />
              <span>Build Evaluation & Performance</span>
              {isFetchingEvaluation && (
                <span
                  role="status"
                  aria-label="Updating evaluation"
                  className="inline-flex items-center gap-1 text-xs font-normal text-blue-600 dark:text-blue-400 bg-blue-50 dark:bg-blue-950/60 border border-blue-200/60 dark:border-blue-800/60 px-2 py-0.5 rounded-full"
                >
                  <RefreshCw size={11} className="animate-spin" />
                  <span>Updating...</span>
                </span>
              )}
            </h2>
            <p className="text-xs sm:text-sm text-zinc-500 dark:text-zinc-400 mt-0.5">
              Automated aerodynamic and electrical physics estimation calculated by Quadsmith
              Evaluator.
            </p>
          </div>

          {/* Runtime Flight Parameters: Battery & Payload */}
          <div className="flex flex-col lg:flex-row items-stretch lg:items-center gap-3">
            {/* Battery Selector */}
            <div className="bg-white dark:bg-zinc-950 p-3 rounded-xl border border-zinc-200 dark:border-zinc-800/80 shadow-xs min-w-[270px] max-w-sm">
              <div className="flex items-center justify-between text-xs mb-1.5">
                <span className="font-semibold text-zinc-700 dark:text-zinc-300 flex items-center gap-1.5">
                  <Zap size={13} className="text-amber-500" />
                  Battery
                </span>
                {electricalLimits ? (
                  <span className="text-[10px] text-zinc-400 font-mono">
                    {electricalLimits.minVoltage && electricalLimits.maxVoltage
                      ? `${electricalLimits.minVoltage.toFixed(1)}–${electricalLimits.maxVoltage.toFixed(1)}V`
                      : ""}
                    {electricalLimits.maxCurrentA
                      ? `, ≥${electricalLimits.maxCurrentA.toFixed(0)}A`
                      : ""}
                  </span>
                ) : null}
              </div>

              <div className="flex items-center gap-1.5 my-1.5">
                <select
                  value={selectedBatteryId}
                  onChange={(e) => setSelectedBatteryId(e.target.value)}
                  aria-label="Select battery"
                  className="flex-1 min-w-0 bg-zinc-50 dark:bg-zinc-900 border border-zinc-200 dark:border-zinc-800 rounded-lg px-2.5 py-1.5 text-xs font-medium text-zinc-900 dark:text-zinc-100 focus:outline-none focus:ring-2 focus:ring-blue-500 truncate cursor-pointer shadow-xs"
                >
                  {compatibleBatteries.length === 0 ? (
                    <option value="" disabled>
                      {isLoadingBatteries
                        ? "Loading compatible batteries..."
                        : "No compatible batteries"}
                    </option>
                  ) : (
                    compatibleBatteries.map((b) => (
                      <option key={b.id || b.uuid} value={b.id || b.uuid}>
                        {b.name} ({b.weightG ? `${b.weightG}g` : ""}
                        {b.cellCountS ? `, ${b.cellCountS}S` : ""}
                        {b.capacityMah ? `, ${b.capacityMah}mAh` : ""})
                      </option>
                    ))
                  )}
                </select>

                <button
                  type="button"
                  onClick={() => setIsBatteryModalOpen(true)}
                  title="Browse all compatible batteries"
                  aria-label="Browse all compatible batteries"
                  className="px-2.5 py-1.5 rounded-lg border border-zinc-200 dark:border-zinc-800 bg-zinc-50 dark:bg-zinc-900 hover:bg-zinc-100 dark:hover:bg-zinc-800 text-zinc-700 dark:text-zinc-300 transition-colors text-xs font-medium flex items-center gap-1 shrink-0 cursor-pointer shadow-xs"
                >
                  <Search size={12} className="text-zinc-400" />
                  <span>Browse</span>
                </button>
              </div>

              {activeBattery && (
                <div className="flex items-center gap-2 text-[10px] text-zinc-500 dark:text-zinc-400 mt-1">
                  <span className="font-semibold text-zinc-700 dark:text-zinc-300 font-mono">
                    {activeBattery.weightG}g
                  </span>
                  <span>•</span>
                  <span>
                    {activeBattery.cellCountS}S {activeBattery.chemistry}
                  </span>
                  {activeBattery.maxCurrentA ? (
                    <>
                      <span>•</span>
                      <span>Max {activeBattery.maxCurrentA.toFixed(0)}A</span>
                    </>
                  ) : null}
                </div>
              )}
            </div>

            {/* Interactive Payload Weight Text Box */}
            <div className="bg-white dark:bg-zinc-950 p-3 rounded-xl border border-zinc-200 dark:border-zinc-800/80 shadow-xs min-w-[260px]">
              <div className="flex items-center justify-between text-xs mb-1.5">
                <span className="font-semibold text-zinc-700 dark:text-zinc-300 flex items-center gap-1.5">
                  <Sliders size={13} className="text-blue-500" />
                  Payload Simulator
                </span>
                <span className="font-mono font-bold text-blue-600 dark:text-blue-400">
                  +{payloadWeightG}g
                </span>
              </div>
              <div className="flex items-center gap-1.5 my-1.5">
                <button
                  type="button"
                  onClick={() => {
                    const current = Math.max(0, parseFloat(payloadInput) || 0);
                    const next = Math.max(0, Math.round(current - 10));
                    setPayloadInput(String(next));
                  }}
                  disabled={payloadWeightG <= 0}
                  title="Remove 10g"
                  aria-label="Remove 10 grams"
                  className="w-8 h-8 flex items-center justify-center rounded-lg border border-zinc-200 dark:border-zinc-800 bg-zinc-50 dark:bg-zinc-900 hover:bg-zinc-100 dark:hover:bg-zinc-800 disabled:opacity-30 disabled:cursor-not-allowed text-zinc-700 dark:text-zinc-300 transition-colors cursor-pointer shrink-0 shadow-xs"
                >
                  <Minus size={14} />
                </button>

                <div className="relative flex-1 flex items-center">
                  <input
                    type="text"
                    inputMode="decimal"
                    value={payloadInput}
                    onChange={(e) => {
                      const val = e.target.value;
                      if (val === "" || /^\d*\.?\d*$/.test(val)) {
                        setPayloadInput(val);
                      }
                    }}
                    placeholder="0"
                    className="w-full bg-zinc-50 dark:bg-zinc-900 border border-zinc-200 dark:border-zinc-800 rounded-lg px-3 py-1.5 pr-7 text-sm font-mono text-zinc-900 dark:text-zinc-100 focus:outline-none focus:ring-2 focus:ring-blue-500 shadow-xs text-center"
                    aria-label="Payload weight in grams"
                  />
                  <span className="absolute right-2.5 text-xs text-zinc-400 font-mono pointer-events-none select-none">
                    g
                  </span>
                </div>

                <button
                  type="button"
                  onClick={() => {
                    const current = Math.max(0, parseFloat(payloadInput) || 0);
                    const next = Math.round(current + 10);
                    setPayloadInput(String(next));
                  }}
                  title="Add 10g"
                  aria-label="Add 10 grams"
                  className="w-8 h-8 flex items-center justify-center rounded-lg border border-zinc-200 dark:border-zinc-800 bg-zinc-50 dark:bg-zinc-900 hover:bg-zinc-100 dark:hover:bg-zinc-800 text-zinc-700 dark:text-zinc-300 transition-colors cursor-pointer shrink-0 shadow-xs"
                >
                  <Plus size={14} />
                </button>
              </div>
              <div className="flex gap-1 mt-2 text-[10px]">
                <button
                  type="button"
                  onClick={() => setPayloadInput("0")}
                  className={`px-2 py-0.5 rounded cursor-pointer transition-colors font-medium ${
                    payloadWeightG === 0
                      ? "bg-blue-600 text-white"
                      : "bg-zinc-100 dark:bg-zinc-800 text-zinc-500 hover:text-zinc-700 dark:hover:text-zinc-300"
                  }`}
                >
                  Bare (0g)
                </button>
                <button
                  type="button"
                  onClick={() => setPayloadInput("16")}
                  className={`px-2 py-0.5 rounded cursor-pointer transition-colors font-medium ${
                    payloadWeightG === 16
                      ? "bg-blue-600 text-white"
                      : "bg-zinc-100 dark:bg-zinc-800 text-zinc-500 hover:text-zinc-700 dark:hover:text-zinc-300"
                  }`}
                >
                  Thumb (+16g)
                </button>
                <button
                  type="button"
                  onClick={() => setPayloadInput("133")}
                  className={`px-2 py-0.5 rounded cursor-pointer transition-colors font-medium ${
                    payloadWeightG === 133
                      ? "bg-blue-600 text-white"
                      : "bg-zinc-100 dark:bg-zinc-800 text-zinc-500 hover:text-zinc-700 dark:hover:text-zinc-300"
                  }`}
                >
                  GoPro (+133g)
                </button>
              </div>
            </div>
          </div>
        </div>

        {/* Evaluation Metrics Cards */}
        {isLoadingEvaluation && !evaluation ? (
          <div>
            <div className="grid grid-cols-2 md:grid-cols-3 gap-4 mb-6">
              {[1, 2, 3, 4, 5, 6].map((i) => (
                <div
                  key={i}
                  className="p-4 rounded-xl bg-white dark:bg-zinc-950 border border-zinc-200 dark:border-zinc-800 shadow-xs min-h-[116px] flex flex-col justify-between animate-pulse"
                >
                  <div className="h-3.5 bg-zinc-200 dark:bg-zinc-800 rounded w-24" />
                  <div>
                    <div className="h-7 bg-zinc-200 dark:bg-zinc-800 rounded w-16 mb-2" />
                    <div className="h-3 bg-zinc-100 dark:bg-zinc-800/60 rounded w-28" />
                  </div>
                </div>
              ))}
            </div>
            <div className="h-11 rounded-xl bg-zinc-100 dark:bg-zinc-900 border border-zinc-200 dark:border-zinc-800 animate-pulse" />
          </div>
        ) : evalError ? (
          <div className="p-4 rounded-xl bg-red-50 dark:bg-red-950/40 text-red-600 dark:text-red-400 text-sm">
            Evaluation error: {evalError.message}
          </div>
        ) : evaluation ? (
          <div
            className={`transition-opacity duration-150 ${
              isFetchingEvaluation ? "opacity-75" : "opacity-100"
            }`}
          >
            <div className="grid grid-cols-2 md:grid-cols-3 gap-4 mb-6">
              {/* Metric 1: AUW Total Weight */}
              <div className="p-4 rounded-xl bg-white dark:bg-zinc-950 border border-zinc-200 dark:border-zinc-800 shadow-xs min-h-[116px] flex flex-col justify-between">
                <div className="flex items-center gap-1.5 text-xs font-semibold uppercase text-zinc-400">
                  <Weight size={14} />
                  <span>All-Up Weight</span>
                </div>
                <div>
                  <div className="text-2xl sm:text-3xl font-bold mt-1 text-zinc-900 dark:text-zinc-100">
                    {Math.round(evaluation.buildWeightG || evaluation.totalWeightG)}
                    <span className="text-sm font-normal text-zinc-500 ml-1">g</span>
                  </div>
                  <div className="text-[11px] text-zinc-500 mt-1">
                    {payloadWeightG > 0
                      ? `Includes +${payloadWeightG}g payload`
                      : "Quadcopter + LiPo"}
                  </div>
                </div>
              </div>

              {/* Metric 2: Thrust-to-Weight Ratio */}
              <div className="p-4 rounded-xl bg-white dark:bg-zinc-950 border border-zinc-200 dark:border-zinc-800 shadow-xs min-h-[116px] flex flex-col justify-between">
                <div className="flex items-center gap-1.5 text-xs font-semibold uppercase text-zinc-400">
                  <Gauge size={14} />
                  <span>Thrust / Weight</span>
                </div>
                <div>
                  <div className="text-2xl sm:text-3xl font-bold mt-1 text-emerald-600 dark:text-emerald-400">
                    {evaluation.thrustToWeightRatio.toFixed(1)}
                    <span className="text-sm font-normal text-zinc-500 ml-1">: 1</span>
                  </div>
                  <div className="text-[11px] text-zinc-500 mt-1">
                    {getTwrDescription(evaluation.thrustToWeightRatio)}
                  </div>
                </div>
              </div>

              {/* Metric 3: Max Acceleration */}
              <div className="p-4 rounded-xl bg-white dark:bg-zinc-950 border border-zinc-200 dark:border-zinc-800 shadow-xs min-h-[116px] flex flex-col justify-between">
                <div className="flex items-center gap-1.5 text-xs font-semibold uppercase text-zinc-400">
                  <Rocket size={14} />
                  <span>Max Acceleration</span>
                </div>
                <div>
                  <div className="text-2xl sm:text-3xl font-bold mt-1 text-zinc-900 dark:text-zinc-100">
                    {evaluation.maxAccelerationMps2.toFixed(1)}
                    <span className="text-sm font-normal text-zinc-500 ml-1">m/s²</span>
                  </div>
                  <div className="text-[11px] text-zinc-500 mt-1">
                    {evaluation.maxAccelerationMps2 > 0
                      ? `~${(evaluation.maxAccelerationMps2 / 9.80665).toFixed(1)} G vertical punchout`
                      : "No positive climb"}
                  </div>
                </div>
              </div>

              {/* Metric 4: Top Speed */}
              <div className="p-4 rounded-xl bg-white dark:bg-zinc-950 border border-zinc-200 dark:border-zinc-800 shadow-xs min-h-[116px] flex flex-col justify-between">
                <div className="flex items-center gap-1.5 text-xs font-semibold uppercase text-zinc-400">
                  <Wind size={14} />
                  <span>Top Speed</span>
                </div>
                <div>
                  <div className="text-2xl sm:text-3xl font-bold mt-1 text-zinc-900 dark:text-zinc-100">
                    {Math.round(evaluation.topSpeedKmh)}
                    <span className="text-sm font-normal text-zinc-500 ml-1">km/h</span>
                  </div>
                  <div className="text-[11px] text-zinc-500 mt-1">
                    {evaluation.topSpeedKmh > 0
                      ? `~${Math.round(evaluation.topSpeedKmh * 0.621371)} mph terminal`
                      : "Insufficient forward thrust"}
                  </div>
                </div>
              </div>

              {/* Metric 5: Hover Throttle */}
              <div className="p-4 rounded-xl bg-white dark:bg-zinc-950 border border-zinc-200 dark:border-zinc-800 shadow-xs min-h-[116px] flex flex-col justify-between">
                <div className="flex items-center gap-1.5 text-xs font-semibold uppercase text-zinc-400">
                  <Zap size={14} />
                  <span>Hover Throttle</span>
                </div>
                <div>
                  <div className="text-2xl sm:text-3xl font-bold mt-1 text-zinc-900 dark:text-zinc-100">
                    {evaluation.hoverThrottlePercent.toFixed(1)}
                    <span className="text-sm font-normal text-zinc-500 ml-1">%</span>
                  </div>
                  {/* Progress bar */}
                  <div className="w-full bg-zinc-100 dark:bg-zinc-800 h-1.5 rounded-full mt-2 overflow-hidden">
                    <div
                      className={`h-full rounded-full transition-all duration-300 ${
                        evaluation.hoverThrottlePercent > 50
                          ? "bg-red-500"
                          : evaluation.hoverThrottlePercent > 35
                            ? "bg-amber-500"
                            : "bg-emerald-500"
                      }`}
                      style={{
                        width: `${Math.min(100, evaluation.hoverThrottlePercent)}%`,
                      }}
                    />
                  </div>
                  <div
                    className="text-[11px] text-zinc-500 mt-1"
                    title="Average propeller RPM at hover assuming sea-level air pressure (1.225 kg/m³), no wind, and horizontal stability"
                  >
                    {evaluation.hoverThrottlePercent <= 100 && evaluation.hoverRpm > 0
                      ? `~${evaluation.hoverRpm.toLocaleString()} prop RPM`
                      : "Cannot achieve hover"}
                  </div>
                </div>
              </div>

              {/* Metric 6: Estimated Flight Time */}
              <div className="p-4 rounded-xl bg-white dark:bg-zinc-950 border border-zinc-200 dark:border-zinc-800 shadow-xs min-h-[116px] flex flex-col justify-between">
                <div className="flex items-center gap-1.5 text-xs font-semibold uppercase text-zinc-400">
                  <Clock size={14} />
                  <span>Est. Flight Time</span>
                </div>
                <div>
                  <div className="text-xl sm:text-2xl lg:text-3xl font-bold mt-1 text-blue-600 dark:text-blue-400">
                    {evaluation.minFlightTimeMin > 0 && evaluation.maxFlightTimeMin > 0
                      ? `${evaluation.minFlightTimeMin.toFixed(1)} – ${evaluation.maxFlightTimeMin.toFixed(1)}`
                      : evaluation.maxFlightTimeMin > 0
                        ? `~${evaluation.maxFlightTimeMin.toFixed(1)}`
                        : "—"}
                    <span className="text-sm font-normal text-zinc-500 ml-1">min</span>
                  </div>
                  <div className="text-[11px] text-zinc-500 mt-1">
                    Varies with throttle management
                  </div>
                </div>
              </div>
            </div>

            {/* Diagnostic Alerts / Compatibility Checks */}
            {(() => {
              const errors = evaluation.systemMessages.filter(
                (m) => m.severity === SystemMessageSeverity.ERROR,
              );
              const warnings = evaluation.systemMessages.filter(
                (m) => m.severity === SystemMessageSeverity.WARNING,
              );

              return (
                <>
                  {errors.length > 0 && (
                    <div className="mb-3 p-4 rounded-xl bg-red-50 dark:bg-red-950/30 border border-red-200 dark:border-red-900/60 text-red-700 dark:text-red-400 flex items-start gap-3">
                      <AlertOctagon size={20} className="shrink-0 mt-0.5 text-red-600" />
                      <div>
                        <div className="font-bold text-sm">Compatibility Issues Detected</div>
                        <ul className="list-disc list-inside text-xs mt-1 space-y-0.5">
                          {errors.map((err, i) => (
                            <li key={i}>{err.message}</li>
                          ))}
                        </ul>
                      </div>
                    </div>
                  )}

                  {warnings.length > 0 && (
                    <div className="mb-3 p-4 rounded-xl bg-amber-50 dark:bg-amber-950/30 border border-amber-200 dark:border-amber-900/60 text-amber-800 dark:text-amber-400 flex items-start gap-3">
                      <AlertTriangle size={20} className="shrink-0 mt-0.5 text-amber-600" />
                      <div>
                        <div className="font-bold text-sm">Evaluation Warnings</div>
                        <ul className="list-disc list-inside text-xs mt-1 space-y-0.5">
                          {warnings.map((warn, i) => (
                            <li key={i}>{warn.message}</li>
                          ))}
                        </ul>
                      </div>
                    </div>
                  )}

                  {errors.length === 0 && warnings.length === 0 && (
                    <div className="p-3.5 rounded-xl bg-emerald-50 dark:bg-emerald-950/30 border border-emerald-200 dark:border-emerald-900/60 text-emerald-800 dark:text-emerald-300 flex items-center gap-2.5 text-xs font-medium">
                      <CheckCircle2 size={18} className="text-emerald-600 shrink-0" />
                      <span>All evaluated components are fully compatible and flight-ready.</span>
                    </div>
                  )}
                </>
              );
            })()}
          </div>
        ) : null}
      </section>

      {/* ============================================================ */}
      {/* BILL OF MATERIALS (BOM) SECTION                              */}
      {/* ============================================================ */}
      <section className="mb-10">
        <h2 className="text-xl font-bold text-zinc-900 dark:text-zinc-100 mb-4 flex items-center gap-2">
          <Layers className="text-blue-600 dark:text-blue-400" size={22} />
          <span>Bill of Materials (Hardware Components)</span>
        </h2>

        <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
          {/* Frame */}
          <BomComponentCard
            label="Frame"
            collectionId="frames"
            uuid={build.frameUuid}
            queryMethod={getFrame}
            icon={<Layers size={18} />}
            subtitle={(item) =>
              `${item.manufacturer} • ${item.wheelbaseMm ? `${item.wheelbaseMm}mm wheelbase` : ""}`
            }
          />

          {/* Motors */}
          <BomComponentCard
            label="Motors (4x)"
            collectionId="motors"
            uuid={build.motorUuid}
            queryMethod={getMotor}
            icon={<Cpu size={18} />}
            subtitle={(item) =>
              `${item.manufacturer} • ${item.kv ? `${item.kv}KV` : ""} • ${item.weightG ? `${item.weightG}g each` : ""}`
            }
          />

          {/* Flight Controller */}
          <BomComponentCard
            label="Flight Controller"
            collectionId="flight_controllers"
            uuid={build.flightControllerUuid}
            queryMethod={getFlightController}
            icon={<Cpu size={18} />}
            subtitle={(item) => `${item.manufacturer} • ${item.processor || "FC Board"}`}
          />

          {/* Electronic Speed Controller */}
          {build.electronicSpeedControllerUuids.map((escUuid, idx) => (
            <BomComponentCard
              key={escUuid + idx}
              label={
                build.electronicSpeedControllerUuids.length > 1
                  ? `ESC #${idx + 1}`
                  : "Electronic Speed Controller"
              }
              collectionId="electronic_speed_controllers"
              uuid={escUuid}
              queryMethod={getElectronicSpeedController}
              icon={<Zap size={18} />}
              subtitle={(item) =>
                `${item.manufacturer} • ${item.motorCurrentMaxA}A • ${item.maxMotors}x motors`
              }
            />
          ))}

          {/* Propellers */}
          <BomComponentCard
            label="Propellers"
            collectionId="propellers"
            uuid={build.propellerUuid}
            queryMethod={getPropeller}
            icon={<Compass size={18} />}
            subtitle={(item) =>
              `${item.manufacturer} • ${item.blades ? `${item.blades}-blade` : "Props"}`
            }
          />

          {/* Video Transmitter */}
          <BomComponentCard
            label="Video Transmitter (VTX)"
            collectionId="video_transmitters"
            uuid={build.videoTransmitterUuid}
            queryMethod={getVideoTransmitter}
            icon={<Radio size={18} />}
            subtitle={(item) =>
              `${item.manufacturer} • ${item.maxPowerMw ? `${item.maxPowerMw}mW` : ""} • ${item.protocol || ""}`
            }
          />

          {/* Cameras */}
          {build.cameraUuids.map((camUuid, idx) => (
            <BomComponentCard
              key={camUuid + idx}
              label={build.cameraUuids.length > 1 ? `Camera #${idx + 1}` : "Camera"}
              collectionId="cameras"
              uuid={camUuid}
              queryMethod={getCamera}
              icon={<CameraIcon size={18} />}
              subtitle={(item) => `${item.manufacturer} • ${item.sensor || "FPV Camera"}`}
            />
          ))}

          {/* Receivers */}
          {build.receiverUuids.map((rxUuid, idx) => (
            <BomComponentCard
              key={rxUuid + idx}
              label={build.receiverUuids.length > 1 ? `Receiver #${idx + 1}` : "Receiver"}
              collectionId="receivers"
              uuid={rxUuid}
              queryMethod={getReceiver}
              icon={<Radio size={18} />}
              subtitle={(item) => `${item.manufacturer} • ${item.protocol || "RC Link"}`}
            />
          ))}

          {/* Antennas */}
          {build.antennaUuids.map((antUuid, idx) => (
            <BomComponentCard
              key={antUuid + idx}
              label={build.antennaUuids.length > 1 ? `Antenna #${idx + 1}` : "Antenna"}
              collectionId="antennas"
              uuid={antUuid}
              queryMethod={getAntenna}
              icon={<Radio size={18} />}
              subtitle={(item) =>
                `${item.manufacturer} • ${item.polarization || ""} ${item.frequencyBandMhz ? `${item.frequencyBandMhz}MHz` : ""}`
              }
            />
          ))}

          {/* Optional GPS Receiver */}
          {build.gpsReceiverUuid && (
            <BomComponentCard
              label="GPS Receiver"
              collectionId="gps_receivers"
              uuid={build.gpsReceiverUuid}
              queryMethod={getGpsReceiver}
              icon={<Compass size={18} />}
              subtitle={(item) => `${item.manufacturer} • ${item.protocol || "GNSS"}`}
            />
          )}
        </div>
      </section>

      {/* ============================================================ */}
      {/* BUILD MEDIA & SHOWCASE                                       */}
      {/* ============================================================ */}
      <MediaGallery
        primaryDisplayImage={build.primaryDisplayImage}
        media={build.media}
        title="Build Media & Video Showcase"
      />

      {/* ============================================================ */}
      {/* REFERENCE LINKS                                              */}
      {/* ============================================================ */}
      {build.referenceLinks && build.referenceLinks.length > 0 && (
        <section className="mt-8 mb-12">
          <h2 className="text-xl font-bold text-zinc-900 dark:text-zinc-100 mb-3">
            Reference Links & Documentation
          </h2>
          <div className="flex flex-col gap-2">
            {build.referenceLinks.map((link, idx) => (
              <a
                key={idx}
                href={link.url}
                target="_blank"
                rel="noopener noreferrer"
                className="flex items-center justify-between p-3.5 bg-white dark:bg-zinc-900 hover:bg-zinc-50 dark:hover:bg-zinc-800/80 border border-zinc-200 dark:border-zinc-800 rounded-xl transition-colors group shadow-xs"
              >
                <div className="flex items-center gap-3 min-w-0">
                  <span className="px-2.5 py-0.5 text-xs font-semibold rounded-md bg-zinc-100 dark:bg-zinc-800 text-zinc-700 dark:text-zinc-300 shrink-0">
                    {LINK_TYPE_LABELS[link.type] || "Link"}
                  </span>
                  <span className="text-sm text-zinc-700 dark:text-zinc-300 font-mono truncate max-w-lg">
                    {link.url}
                  </span>
                </div>
                <ExternalLink className="w-4 h-4 text-zinc-400 group-hover:text-blue-600 dark:group-hover:text-blue-400 shrink-0 ml-2" />
              </a>
            ))}
          </div>
        </section>
      )}

      {/* Battery Picker Explorer Modal */}
      <BatteryPickerModal
        isOpen={isBatteryModalOpen}
        onClose={() => setIsBatteryModalOpen(false)}
        selectedBatteryId={selectedBatteryId}
        onSelectBattery={(id) => setSelectedBatteryId(id)}
        limits={electricalLimits}
        compatibleBatteries={compatibleBatteries}
        isLoading={isLoadingBatteries}
      />
    </div>
  );
}
