import { useParams, Link } from "react-router-dom";
import { useState, useMemo, useEffect, useCallback } from "react";
import { useQuery } from "@connectrpc/connect-query";
import { keepPreviousData } from "@tanstack/react-query";
import { getBuild } from "@quadsmith/shared/gen/quadsmith/api/build-BuildService_connectquery";
import {
  evaluateBuild,
  getBuildElectricalLimits,
} from "@quadsmith/shared/gen/quadsmith/api/evaluator-EvaluatorService_connectquery";
import { checkCompatibility } from "@quadsmith/shared/gen/quadsmith/api/compatibility-CompatibilityService_connectquery";
import { getFrame } from "@quadsmith/shared/gen/quadsmith/api/frame-FrameService_connectquery";
import { getMotor } from "@quadsmith/shared/gen/quadsmith/api/motor-MotorService_connectquery";
import { listBatteries } from "@quadsmith/shared/gen/quadsmith/api/battery-BatteryService_connectquery";
import { getFlightController } from "@quadsmith/shared/gen/quadsmith/api/flight_controller-FlightControllerService_connectquery";
import { getElectronicSpeedController } from "@quadsmith/shared/gen/quadsmith/api/electronic_speed_controller-ElectronicSpeedControllerService_connectquery";
import { getPropeller } from "@quadsmith/shared/gen/quadsmith/api/propeller-PropellerService_connectquery";
import { getCamera } from "@quadsmith/shared/gen/quadsmith/api/camera-CameraService_connectquery";
import { getVideoTransmitter } from "@quadsmith/shared/gen/quadsmith/api/video_transmitter-VideoTransmitterService_connectquery";
import { getReceiver } from "@quadsmith/shared/gen/quadsmith/api/receiver-ReceiverService_connectquery";
import { getAntenna } from "@quadsmith/shared/gen/quadsmith/api/antenna-AntennaService_connectquery";
import { getGpsReceiver } from "@quadsmith/shared/gen/quadsmith/api/gps_receiver-GpsReceiverService_connectquery";
import { SystemMessageSeverity } from "@quadsmith/shared/gen/quadsmith/api/evaluator_pb";
import type { Battery } from "@quadsmith/shared/gen/quadsmith/api/battery_pb";
import { MediaGallery } from "../components/MediaGallery";
import { BatteryPickerModal } from "../components/BatteryPickerModal";
import { buildBatteryCelFilter } from "@quadsmith/shared";
import { CollectionIcon } from "@quadsmith/shared";
import { getTwrDescription, getTwrColor, getHoverThrottleColor } from "@quadsmith/shared";
import { getHardwareCollection, getCollectionPath } from "../lib/hardwareCollections";
import { getCollectionColor } from "@quadsmith/shared";
import { CollectionBadge } from "@quadsmith/shared";
import { ReferenceLinksList } from "../components/SocialLinkPreviewCard";
import { ProductHoverCard } from "../components/ProductHoverCard";
import { UserBadge } from "../components/UserBadge";
import { BatteryCombobox } from "../components/BatteryCombobox";
import { SocialToolbar } from "../components/SocialToolbar";
import { SocialTabs, type SocialTabType } from "../components/SocialTabs";
import { useDocumentMeta } from "@quadsmith/shared";
import { useAuth } from "../context/AuthContext";
import {
  ChevronRight,
  Gauge,
  Clock,
  Weight,
  ExternalLink,
  AlertTriangle,
  AlertOctagon,
  AlertCircle,
  Sliders,
  Layers,
  Zap,
  Rocket,
  Wind,
  Plus,
  Minus,
  RefreshCw,
  Search,
  Hammer,
  Edit,
} from "lucide-react";

interface BomComponentProps {
  label: string;
  collectionId: string;
  uuid: string;
  queryMethod: any;
  icon?: React.ReactNode;
  subtitle?: (item: any) => string;
  placement?: "left" | "right";
}

function BomComponentCard({
  label,
  collectionId,
  uuid,
  queryMethod,
  icon,
  subtitle,
  placement,
}: BomComponentProps) {
  const {
    data: rawItem,
    isLoading,
    error,
  } = useQuery(queryMethod, { id: uuid }, { enabled: !!uuid });
  const item = rawItem as any;

  if (!uuid) return null;

  const col = getHardwareCollection(collectionId);
  const targetPath = col ? getCollectionPath(col) : `components/hardware/${collectionId}`;
  const colColor = col?.color || getCollectionColor(collectionId);

  const cardContent = (
    <>
      <div className="flex items-center gap-3 min-w-0">
        <div
          className={`w-9 h-9 rounded-lg flex items-center justify-center shrink-0 bg-zinc-100 dark:bg-zinc-800 border border-zinc-200/80 dark:border-zinc-700/60 ${colColor.textClass}`}
        >
          {icon ?? <CollectionIcon collection={collectionId} size={18} />}
        </div>
        <div className="min-w-0">
          <div className="flex items-center gap-1.5 mb-0.5">
            <span className="text-xs uppercase font-semibold text-zinc-400 dark:text-zinc-500">
              {label}
            </span>
            <CollectionBadge collection={collectionId} size="xs" />
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
                to={`/${targetPath}/${item.id || item.uuid}`}
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
          to={`/${targetPath}/${item.id || item.uuid}`}
          className="text-xs text-zinc-400 hover:text-blue-600 dark:hover:text-blue-400 p-1.5 rounded-md hover:bg-zinc-100 dark:hover:bg-zinc-800 transition-colors shrink-0"
          title={`View ${item.name || item.id} in catalog`}
        >
          <ExternalLink size={14} />
        </Link>
      )}
    </>
  );

  const cardClasses =
    "flex items-center justify-between p-3.5 rounded-xl border border-zinc-200 dark:border-zinc-800 bg-white dark:bg-zinc-900/60 hover:border-zinc-300 dark:hover:border-zinc-700 transition-colors";

  if (item) {
    return (
      <ProductHoverCard
        collectionId={collectionId}
        item={item}
        productId={item.id || item.uuid}
        as="div"
        placement={placement}
        className={cardClasses}
      >
        {cardContent}
      </ProductHoverCard>
    );
  }

  return (
    <div data-component-card className={cardClasses}>
      {cardContent}
    </div>
  );
}

export function BuildProfilePage() {
  const { buildId } = useParams<{ buildId: string }>();
  const [payloadInput, setPayloadInput] = useState<string>("0");
  const payloadWeightG = Math.max(0, parseFloat(payloadInput) || 0);

  const [selectedBatteryId, setSelectedBatteryId] = useState<string>("");
  const [isBatteryModalOpen, setIsBatteryModalOpen] = useState(false);
  const [activeSocialTab, setActiveSocialTab] = useState<SocialTabType>("reviews");

  const { user, isAuthenticated } = useAuth();
  const {
    data: build,
    isLoading: isLoadingBuild,
    error: buildError,
  } = useQuery(getBuild, { id: buildId || "" }, { enabled: !!buildId });

  const isCreator =
    isAuthenticated &&
    user &&
    (user.uuid === build?.userUuid || user.id === build?.authorUsername || user.role === "ADMIN");

  useDocumentMeta({
    title: build ? `${build.name} — FPV Drone Build — Quadsmith` : "Build Profile — Quadsmith",
    description:
      build?.description ||
      "Explore custom FPV drone build specifications, flight times, and component compatibility on Quadsmith.",
    image: build?.primaryDisplayImage || "/og-default.png",
    type: "article",
  });

  const {
    data: electricalLimits,
    isLoading: isLoadingLimits,
    error: limitsError,
  } = useQuery(
    getBuildElectricalLimits,
    { buildSource: { case: "buildId", value: build?.id || buildId || "" } },
    { enabled: !!build },
  );

  const batteryFilter = useMemo(() => {
    return buildBatteryCelFilter(electricalLimits);
  }, [electricalLimits]);

  const {
    data: batteryResponse,
    isLoading: isLoadingBatteries,
    error: batteryError,
  } = useQuery(
    listBatteries,
    {
      filter: batteryFilter,
      sort: ["weight_g"],
      pageSize: 100,
    },
    { enabled: !!electricalLimits },
  );
  const compatibleBatteries = batteryResponse?.batteries || [];

  // Selected battery object state (for custom or incompatible selection)
  const [selectedBatteryObj, setSelectedBatteryObj] = useState<Battery | null>(null);

  // Reset selected battery when changing builds
  useEffect(() => {
    setSelectedBatteryId("");
    setSelectedBatteryObj(null);
    setActiveSocialTab("reviews");
  }, [buildId]);

  // Maintain active compatible battery: select recommended or first compatible battery
  useEffect(() => {
    if (compatibleBatteries.length > 0) {
      const isSelectedCompatible = compatibleBatteries.some(
        (b) => (b.id || b.uuid) === selectedBatteryId,
      );
      if (!selectedBatteryId || !isSelectedCompatible) {
        const defaultBattery =
          electricalLimits?.defaultBatteryId &&
          compatibleBatteries.find(
            (b) =>
              b.id === electricalLimits.defaultBatteryId ||
              b.uuid === electricalLimits.defaultBatteryId,
          );
        const nextBattery = defaultBattery || compatibleBatteries[0];
        const nextId = nextBattery.id || nextBattery.uuid;
        setSelectedBatteryId(nextId);
        setSelectedBatteryObj(nextBattery);
      }
    } else if (selectedBatteryId) {
      setSelectedBatteryId("");
      setSelectedBatteryObj(null);
    }
  }, [buildId, electricalLimits, compatibleBatteries, selectedBatteryId]);

  const handleSelectBattery = useCallback((id: string, battery?: Battery) => {
    setSelectedBatteryId(id);
    setSelectedBatteryObj(battery || null);
  }, []);

  const {
    data: compatibilityResponse,
    isLoading: isLoadingCompatibility,
    error: compatibilityError,
  } = useQuery(checkCompatibility, { build: build! }, { enabled: !!build });

  const componentIncompatibilities = useMemo(() => {
    const list: string[] = [];
    if (
      electricalLimits?.minVoltage &&
      electricalLimits?.maxVoltage &&
      electricalLimits.minVoltage > electricalLimits.maxVoltage
    ) {
      list.push(
        `Component voltage conflict: Minimum required component operating voltage (${electricalLimits.minVoltage.toFixed(
          1,
        )}V) exceeds maximum component voltage rating (${electricalLimits.maxVoltage.toFixed(1)}V).`,
      );
    }
    if (compatibilityResponse?.messages) {
      for (const m of compatibilityResponse.messages) {
        if (m.severityLevel === 0) {
          if (m.checkerName === "ComponentVoltageChecker" && list.length > 0) {
            continue;
          }
          list.push(m.message);
        }
      }
    }
    return list;
  }, [electricalLimits, compatibilityResponse]);

  const hasComponentConflict = componentIncompatibilities.length > 0;

  const isBatteryLoading =
    isLoadingLimits || isLoadingCompatibility || (!!electricalLimits && isLoadingBatteries);
  const isBatteryLoaded =
    !isLoadingLimits && !isLoadingCompatibility && !isLoadingBatteries && !!electricalLimits;
  const hasNoCompatibleBatteries =
    !hasComponentConflict &&
    isBatteryLoaded &&
    (compatibleBatteries.length === 0 || !electricalLimits.defaultBatteryId);

  const activeBattery = useMemo(() => {
    return (
      compatibleBatteries.find((b) => b.id === selectedBatteryId || b.uuid === selectedBatteryId) ||
      (selectedBatteryObj &&
      (selectedBatteryObj.id === selectedBatteryId || selectedBatteryObj.uuid === selectedBatteryId)
        ? selectedBatteryObj
        : undefined)
    );
  }, [compatibleBatteries, selectedBatteryId, selectedBatteryObj]);

  const comboboxBatteries = useMemo(() => {
    if (
      activeBattery &&
      !compatibleBatteries.some(
        (b) => (b.id || b.uuid) === (activeBattery.id || activeBattery.uuid),
      )
    ) {
      return [activeBattery, ...compatibleBatteries];
    }
    return compatibleBatteries;
  }, [activeBattery, compatibleBatteries]);

  const bomComponents = useMemo(() => {
    if (!build) return [];
    const list: Array<{
      key: string;
      label: string;
      collectionId: string;
      uuid: string;
      queryMethod: any;
      icon?: React.ReactNode;
      subtitle?: (item: any) => string;
    }> = [];

    if (build.frameUuid) {
      list.push({
        key: "frame",
        label: "Frame",
        collectionId: "frames",
        uuid: build.frameUuid,
        queryMethod: getFrame,
        subtitle: (item) =>
          `${item.manufacturer} • ${item.wheelbaseMm ? `${item.wheelbaseMm}mm wheelbase` : ""}`,
      });
    }

    if (build.motorUuid) {
      list.push({
        key: "motor",
        label: "Motors (4x)",
        collectionId: "motors",
        uuid: build.motorUuid,
        queryMethod: getMotor,
        subtitle: (item) =>
          `${item.manufacturer} • ${item.kv ? `${item.kv}KV` : ""} • ${item.weightG ? `${item.weightG.toFixed(2)}g each` : ""}`,
      });
    }

    if (build.flightControllerUuid) {
      list.push({
        key: "fc",
        label: "Flight Controller",
        collectionId: "flight-controllers",
        uuid: build.flightControllerUuid,
        queryMethod: getFlightController,
        subtitle: (item) => `${item.manufacturer} • ${item.processor || "FC Board"}`,
      });
    }

    build.electronicSpeedControllerUuids.forEach((escUuid, idx) => {
      if (escUuid) {
        list.push({
          key: `esc-${escUuid}-${idx}`,
          label:
            build.electronicSpeedControllerUuids.length > 1
              ? `ESC #${idx + 1}`
              : "Electronic Speed Controller",
          collectionId: "electronic-speed-controllers",
          uuid: escUuid,
          queryMethod: getElectronicSpeedController,
          subtitle: (item) =>
            `${item.manufacturer} • ${item.motorCurrentMaxA}A • ${item.maxMotors}x motors`,
        });
      }
    });

    if (build.propellerUuid) {
      list.push({
        key: "propeller",
        label: "Propellers",
        collectionId: "propellers",
        uuid: build.propellerUuid,
        queryMethod: getPropeller,
        subtitle: (item) =>
          `${item.manufacturer} • ${item.blades ? `${item.blades}-blade` : "Props"}`,
      });
    }

    if (build.videoTransmitterUuid) {
      list.push({
        key: "vtx",
        label: "Video Transmitter (VTX)",
        collectionId: "video-transmitters",
        uuid: build.videoTransmitterUuid,
        queryMethod: getVideoTransmitter,
        subtitle: (item) =>
          `${item.manufacturer} • ${item.maxPowerMw ? `${item.maxPowerMw}mW` : ""} • ${item.protocol || ""}`,
      });
    }

    build.cameraUuids.forEach((camUuid, idx) => {
      if (camUuid) {
        list.push({
          key: `cam-${camUuid}-${idx}`,
          label: build.cameraUuids.length > 1 ? `Camera #${idx + 1}` : "Camera",
          collectionId: "cameras",
          uuid: camUuid,
          queryMethod: getCamera,
          subtitle: (item) => `${item.manufacturer} • ${item.sensor || "FPV Camera"}`,
        });
      }
    });

    build.receiverUuids.forEach((rxUuid, idx) => {
      if (rxUuid) {
        list.push({
          key: `rx-${rxUuid}-${idx}`,
          label: build.receiverUuids.length > 1 ? `Receiver #${idx + 1}` : "Receiver",
          collectionId: "receivers",
          uuid: rxUuid,
          queryMethod: getReceiver,
          subtitle: (item) => `${item.manufacturer} • ${item.protocol || "RC Link"}`,
        });
      }
    });

    build.antennaUuids.forEach((antUuid, idx) => {
      if (antUuid) {
        list.push({
          key: `ant-${antUuid}-${idx}`,
          label: build.antennaUuids.length > 1 ? `Antenna #${idx + 1}` : "Antenna",
          collectionId: "antennas",
          uuid: antUuid,
          queryMethod: getAntenna,
          subtitle: (item) =>
            `${item.manufacturer} • ${item.polarization || ""} ${item.frequencyBandMhz ? `${item.frequencyBandMhz}MHz` : ""}`,
        });
      }
    });

    if (build.gpsReceiverUuid) {
      list.push({
        key: "gps",
        label: "GPS Receiver",
        collectionId: "gps-receivers",
        uuid: build.gpsReceiverUuid,
        queryMethod: getGpsReceiver,
        subtitle: (item) => `${item.manufacturer} • ${item.protocol || "GNSS"}`,
      });
    }

    return list;
  }, [build]);

  const {
    data: evaluation,
    isLoading: isLoadingEvaluation,
    isFetching: isFetchingEvaluation,
    error: evalError,
  } = useQuery(
    evaluateBuild,
    {
      buildSource: { case: "buildId", value: build?.id || buildId || "" },
      payloadWeightG,
      batteryId: selectedBatteryId,
    },
    {
      enabled: !!build && !!selectedBatteryId,
      placeholderData: keepPreviousData,
    },
  );

  const activeEvaluation = selectedBatteryId ? evaluation : undefined;

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
          ← Return to Builds
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
        <CollectionBadge collection="builds" label="Builds" size="md" to="/" />
        <ChevronRight size={14} className="text-zinc-400 dark:text-zinc-500 shrink-0" />
        <span className="text-zinc-900 dark:text-zinc-100 font-medium truncate max-w-md">
          {build.name || build.id}
        </span>
      </nav>

      {/* Build Profile Header */}
      <div className="mb-8">
        <div className="flex flex-col sm:flex-row sm:items-start justify-between gap-4">
          <div>
            <h1 className="text-3xl sm:text-4xl font-extrabold tracking-tight text-zinc-900 dark:text-zinc-50">
              {build.name}
            </h1>
            {build.authorUsername && (
              <div className="mt-2 text-sm text-zinc-500 dark:text-zinc-400 flex items-center gap-1.5">
                <span>Created by</span>
                <UserBadge username={build.authorUsername} size="sm" />
              </div>
            )}
            {build.description && (
              <p className="mt-3 text-base sm:text-lg text-zinc-600 dark:text-zinc-400 leading-relaxed max-w-3xl">
                {build.description}
              </p>
            )}
          </div>

          <div className="flex items-center gap-2 sm:gap-2.5 shrink-0 pt-1">
            <Link
              to={`/builds/new?from=${build.id || buildId}`}
              className="inline-flex items-center gap-2 px-4 py-2 sm:px-5 sm:py-2.5 rounded-xl bg-gradient-to-r from-orange-600 via-orange-500 to-amber-500 hover:from-orange-500 hover:via-orange-400 hover:to-amber-400 active:from-orange-700 active:to-amber-600 text-white font-semibold text-xs sm:text-sm shadow-md shadow-orange-500/20 hover:shadow-lg hover:shadow-orange-500/30 active:scale-95 transition-all cursor-pointer group"
              title="Open this build configuration in The Forge as a baseline"
            >
              <Hammer
                size={16}
                className="text-white group-hover:-rotate-12 transition-transform shrink-0"
              />
              <span>Build Similar</span>
            </Link>
            {isCreator && (
              <Link
                to={`/builds/${build.id || buildId}/edit`}
                className="inline-flex items-center gap-1.5 px-3.5 py-2 sm:px-4 sm:py-2.5 rounded-xl border border-zinc-200 dark:border-zinc-800 bg-white dark:bg-zinc-900 hover:bg-zinc-50 dark:hover:bg-zinc-800 text-xs sm:text-sm font-medium text-zinc-700 dark:text-zinc-300 shadow-2xs transition-colors cursor-pointer"
              >
                <Edit size={15} className="text-zinc-400" />
                <span>Edit Build</span>
              </Link>
            )}
          </div>
        </div>

        <div className="mt-4">
          <SocialToolbar
            target={{ type: "build", id: build.id || buildId || "", collection: "builds" }}
            targetName={build.name}
            onJumpToReviews={() => {
              setActiveSocialTab("reviews");
              const el = document.getElementById("social-hub");
              el?.scrollIntoView({ behavior: "smooth" });
            }}
            onJumpToQa={() => {
              setActiveSocialTab("qa");
              const el = document.getElementById("social-hub");
              el?.scrollIntoView({ behavior: "smooth" });
            }}
          />
        </div>
      </div>

      {/* ============================================================ */}
      {/* BUILD EVALUATION SECTION                                      */}
      {/* ============================================================ */}
      <section className="mb-10 p-6 rounded-2xl border border-zinc-200 dark:border-zinc-800 bg-zinc-50/60 dark:bg-zinc-900/40 shadow-xs">
        <div className="mb-6 pb-4 border-b border-zinc-200 dark:border-zinc-800">
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
          <p className="text-xs sm:text-sm text-zinc-500 dark:text-zinc-400 mt-1">
            Calculated flight metrics are theoretical estimates based on analytical physics models.
          </p>
        </div>

        {limitsError || batteryError || compatibilityError ? (
          <div
            role="alert"
            className="p-4 rounded-xl bg-red-50 dark:bg-red-950/40 text-red-600 dark:text-red-400 text-sm"
          >
            Failed to load build parameters:{" "}
            {limitsError?.message || batteryError?.message || compatibilityError?.message}
          </div>
        ) : hasComponentConflict ? (
          <div
            role="alert"
            className="p-5 rounded-xl bg-red-50 dark:bg-red-950/40 border border-red-200 dark:border-red-900/60 text-red-700 dark:text-red-300 flex items-start gap-3.5"
          >
            <AlertOctagon className="text-red-600 dark:text-red-400 shrink-0 mt-0.5" size={20} />
            <div className="flex-1">
              <h3 className="text-sm font-semibold text-red-900 dark:text-red-200">
                Component Compatibility Conflict
              </h3>
              <p className="text-xs sm:text-sm mt-1 text-red-700 dark:text-red-300/90 leading-relaxed">
                The components in this build are physically or electrically incompatible with each
                other. Build evaluation cannot be calculated until these conflicts are resolved.
              </p>
              <ul className="list-disc list-inside text-xs sm:text-sm mt-2.5 space-y-1">
                {componentIncompatibilities.map((msg, i) => (
                  <li key={i} className="font-medium text-red-800 dark:text-red-200">
                    {msg}
                  </li>
                ))}
              </ul>
            </div>
          </div>
        ) : hasNoCompatibleBatteries ? (
          <div
            role="alert"
            className="p-5 rounded-xl bg-red-50 dark:bg-red-950/40 border border-red-200 dark:border-red-900/60 text-red-700 dark:text-red-300 flex items-start gap-3.5"
          >
            <AlertTriangle className="text-red-600 dark:text-red-400 shrink-0 mt-0.5" size={20} />
            <div className="flex-1">
              <h3 className="text-sm font-semibold text-red-900 dark:text-red-200">
                No Compatible Batteries Found
              </h3>
              <p className="text-xs sm:text-sm mt-1 text-red-700 dark:text-red-300/90 leading-relaxed">
                No batteries in the database match the electrical requirements for this build
                {electricalLimits?.minVoltage &&
                electricalLimits?.maxVoltage &&
                electricalLimits.minVoltage <= electricalLimits.maxVoltage
                  ? ` (${electricalLimits.minVoltage.toFixed(1)}–${electricalLimits.maxVoltage.toFixed(1)}V${
                      electricalLimits.maxCurrentA
                        ? `, ≥${electricalLimits.maxCurrentA.toFixed(0)}A`
                        : ""
                    })`
                  : ""}
                . Build evaluation cannot be calculated without a compatible battery.
              </p>
            </div>
          </div>
        ) : isBatteryLoading ? (
          <div>
            <div className="grid grid-cols-1 sm:grid-cols-2 gap-4 mb-6 animate-pulse">
              <div className="h-24 bg-zinc-200/70 dark:bg-zinc-800/70 rounded-xl" />
              <div className="h-24 bg-zinc-200/70 dark:bg-zinc-800/70 rounded-xl" />
            </div>
            <div className="grid grid-cols-2 md:grid-cols-3 gap-4 mb-6 animate-pulse">
              {[1, 2, 3, 4, 5, 6].map((i) => (
                <div key={i} className="h-28 bg-zinc-200/70 dark:bg-zinc-800/70 rounded-xl" />
              ))}
            </div>
          </div>
        ) : (
          <>
            {/* Runtime Flight Parameters: Battery & Payload */}
            <div className="grid grid-cols-1 sm:grid-cols-2 gap-4 mb-6">
              {/* Battery Selector */}
              <div className="bg-white dark:bg-zinc-950 p-3.5 rounded-xl border border-zinc-200 dark:border-zinc-800/80 shadow-xs flex flex-col justify-between">
                <div className="flex items-center justify-between text-xs mb-1.5">
                  <span className="font-semibold text-zinc-700 dark:text-zinc-300 flex items-center gap-1.5">
                    <Zap size={13} className="text-amber-500" />
                    Battery
                  </span>
                  {electricalLimits ? (
                    <span className="text-[10px] text-zinc-400 font-mono">
                      {electricalLimits.minVoltage &&
                      electricalLimits.maxVoltage &&
                      electricalLimits.minVoltage <= electricalLimits.maxVoltage
                        ? `${electricalLimits.minVoltage.toFixed(1)}–${electricalLimits.maxVoltage.toFixed(1)}V`
                        : ""}
                      {electricalLimits.maxCurrentA
                        ? `, ≥${electricalLimits.maxCurrentA.toFixed(0)}A`
                        : ""}
                      {electricalLimits.connector ? `, ${electricalLimits.connector}` : ""}
                    </span>
                  ) : null}
                </div>

                <div className="flex items-center gap-1.5 my-1.5">
                  <BatteryCombobox
                    selectedBatteryId={selectedBatteryId}
                    onSelectBattery={handleSelectBattery}
                    batteries={comboboxBatteries}
                    isLoading={isLoadingBatteries}
                    ariaLabel="Select battery"
                    className="flex-1 min-w-0"
                  />

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
                      {activeBattery.weightG.toFixed(2)}g
                    </span>
                    <span>•</span>
                    <span>
                      {activeBattery.cellCountS}S {activeBattery.chemistry}
                    </span>
                    {activeBattery.connector ? (
                      <>
                        <span>•</span>
                        <span className="font-semibold text-blue-600 dark:text-blue-400">
                          {activeBattery.connector}
                        </span>
                      </>
                    ) : null}
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
              <div className="bg-white dark:bg-zinc-950 p-3.5 rounded-xl border border-zinc-200 dark:border-zinc-800/80 shadow-xs flex flex-col justify-between">
                <div className="flex items-center justify-between text-xs mb-1.5">
                  <span className="font-semibold text-zinc-700 dark:text-zinc-300 flex items-center gap-1.5">
                    <Sliders size={13} className="text-blue-500" />
                    Payload
                  </span>
                  <span className="font-mono font-bold text-blue-600 dark:text-blue-400">
                    +{payloadWeightG.toFixed(2)}g
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

            {/* Evaluation Metrics Cards */}
            {!selectedBatteryId ? (
              <div className="p-4 rounded-xl bg-amber-500/10 border border-amber-500/20 text-amber-800 dark:text-amber-300 flex items-start gap-3 mb-6">
                <AlertCircle size={20} className="shrink-0 mt-0.5 text-amber-500" />
                <div>
                  <div className="font-bold text-sm">No Compatible Batteries</div>
                  <p className="text-xs mt-1 text-amber-700 dark:text-amber-400 leading-relaxed">
                    No batteries in the database match this build's electrical voltage and current
                    limits.
                  </p>
                </div>
              </div>
            ) : isLoadingEvaluation && !activeEvaluation ? (
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
            ) : activeEvaluation ? (
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
                        {activeEvaluation.allUpWeightG.toFixed(2)}
                        <span className="text-sm font-normal text-zinc-500 ml-1">g</span>
                      </div>
                      <div className="text-[11px] text-zinc-500 mt-1">
                        {payloadWeightG > 0
                          ? `Includes +${payloadWeightG.toFixed(2)}g payload`
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
                      <div
                        className={`text-2xl sm:text-3xl font-bold mt-1 ${getTwrColor(activeEvaluation.thrustToWeightRatio).text}`}
                      >
                        {activeEvaluation.thrustToWeightRatio.toFixed(1)}
                        <span className="text-sm font-normal text-zinc-500 ml-1">: 1</span>
                      </div>
                      <div className="text-[11px] text-zinc-500 mt-1">
                        {getTwrDescription(activeEvaluation.thrustToWeightRatio)}
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
                        {activeEvaluation.maxAccelerationMps2.toFixed(1)}
                        <span className="text-sm font-normal text-zinc-500 ml-1">m/s²</span>
                      </div>
                      <div className="text-[11px] text-zinc-500 mt-1">
                        {activeEvaluation.maxAccelerationMps2 > 0
                          ? `~${(activeEvaluation.maxAccelerationMps2 / 9.80665).toFixed(1)} G vertical punchout`
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
                        {Math.round(activeEvaluation.topSpeedKmh)}
                        <span className="text-sm font-normal text-zinc-500 ml-1">km/h</span>
                      </div>
                      <div className="text-[11px] text-zinc-500 mt-1">
                        {activeEvaluation.topSpeedKmh > 0
                          ? `~${Math.round(activeEvaluation.topSpeedKmh * 0.621371)} mph terminal`
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
                      <div
                        className={`text-2xl sm:text-3xl font-bold mt-1 ${getHoverThrottleColor(activeEvaluation.hoverThrottlePercent).text}`}
                      >
                        {activeEvaluation.hoverThrottlePercent.toFixed(1)}
                        <span className="text-sm font-normal text-zinc-500 ml-1">%</span>
                      </div>
                      {/* Progress bar */}
                      <div className="w-full bg-zinc-100 dark:bg-zinc-800 h-1.5 rounded-full mt-2 overflow-hidden">
                        <div
                          className={`h-full rounded-full transition-all duration-300 ${getHoverThrottleColor(activeEvaluation.hoverThrottlePercent).bar}`}
                          style={{
                            width: `${Math.min(100, activeEvaluation.hoverThrottlePercent)}%`,
                          }}
                        />
                      </div>
                      <div
                        className="text-[11px] text-zinc-500 mt-1"
                        title="Average propeller RPM at hover assuming sea-level air pressure (1.225 kg/m³), no wind, and horizontal stability"
                      >
                        {activeEvaluation.hoverThrottlePercent <= 100 &&
                        activeEvaluation.hoverRpm > 0
                          ? `~${activeEvaluation.hoverRpm.toLocaleString()} prop RPM`
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
                        {activeEvaluation.minFlightTimeMin > 0 &&
                        activeEvaluation.maxFlightTimeMin > 0
                          ? `${activeEvaluation.minFlightTimeMin.toFixed(1)} – ${activeEvaluation.maxFlightTimeMin.toFixed(1)}`
                          : activeEvaluation.maxFlightTimeMin > 0
                            ? `~${activeEvaluation.maxFlightTimeMin.toFixed(1)}`
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
                  const errors = activeEvaluation.systemMessages.filter(
                    (m) => m.severity === SystemMessageSeverity.ERROR,
                  );
                  const warnings = activeEvaluation.systemMessages.filter(
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
                    </>
                  );
                })()}
              </div>
            ) : null}
          </>
        )}
      </section>

      {/* ============================================================ */}
      {/* HARDWARE (BOM) SECTION                                       */}
      {/* ============================================================ */}
      <section className="mb-10">
        <h2 className="text-xl font-bold text-zinc-900 dark:text-zinc-100 mb-4 flex items-center gap-2">
          <Layers className="text-blue-600 dark:text-blue-400" size={22} />
          <span>Hardware</span>
        </h2>

        <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
          {bomComponents.map((item, idx) => (
            <BomComponentCard
              key={item.key}
              label={item.label}
              collectionId={item.collectionId}
              uuid={item.uuid}
              queryMethod={item.queryMethod}
              subtitle={item.subtitle}
              icon={item.icon}
              placement={idx % 2 === 0 ? "left" : "right"}
            />
          ))}
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
          <ReferenceLinksList links={build.referenceLinks} />
        </section>
      )}

      {/* ============================================================ */}
      {/* SOCIAL INTERACTIONS: REVIEWS, Q&A, COLLECTIONS               */}
      {/* ============================================================ */}
      <SocialTabs
        target={{ type: "build", id: build.id || buildId || "", collection: "builds" }}
        targetAuthorUuid={build.userUuid}
        entityName={build.name}
        activeTab={activeSocialTab}
        onTabChange={setActiveSocialTab}
      />

      {/* Battery Picker Explorer Modal */}
      <BatteryPickerModal
        isOpen={isBatteryModalOpen}
        onClose={() => setIsBatteryModalOpen(false)}
        selectedBatteryId={selectedBatteryId}
        onSelectBattery={handleSelectBattery}
        limits={electricalLimits}
        compatibleBatteries={compatibleBatteries}
        isLoading={isLoadingBatteries}
      />
    </div>
  );
}
