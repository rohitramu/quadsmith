import { Link } from "react-router-dom";
import { useQuery } from "@connectrpc/connect-query";
import {
  evaluateBuild,
  getBuildElectricalLimits,
} from "@quadsmith/shared/gen/quadsmith/api/evaluator-EvaluatorService_connectquery";
import {
  Gauge,
  Clock,
  Weight,
  Layers,
  ChevronRight,
  Share2,
  Check,
  Cpu,
  Radio,
  Camera,
  Compass,
  Copy,
} from "lucide-react";
import { useState } from "react";
import { getCollectionColor } from "@quadsmith/shared";
import { CollectionBadge } from "@quadsmith/shared";
import { UserBadge } from "./UserBadge";

import type { Build, ComponentMatchDetail } from "@quadsmith/shared/gen/quadsmith/api/build_pb";

export interface BuildCardProps {
  build: Build;
  similarityScore?: number;
  matchDetails?: ComponentMatchDetail[];
  variant?: "vertical" | "horizontal";
}

export function BuildCard({
  build,
  similarityScore,
  matchDetails,
  variant = "vertical",
}: BuildCardProps) {
  const [copied, setCopied] = useState(false);

  // Fetch electrical limits to determine default lightest compatible battery
  const { data: electricalLimits, isLoading: isLoadingLimits } = useQuery(
    getBuildElectricalLimits,
    { buildSource: { case: "buildId", value: build.id || build.uuid } },
    { staleTime: 60_000 },
  );

  // Quick evaluation stats preview
  const { data: evaluation } = useQuery(
    evaluateBuild,
    {
      buildSource: { case: "buildId", value: build.id || build.uuid },
      payloadWeightG: 0,
      batteryId: electricalLimits?.defaultBatteryId || "",
    },
    {
      enabled: !!electricalLimits?.defaultBatteryId,
      staleTime: 60_000,
    },
  );

  const handleShare = async (e: React.MouseEvent) => {
    e.preventDefault();
    e.stopPropagation();
    const url = `${window.location.origin}/builds/${build.id}`;
    if (navigator.clipboard) {
      await navigator.clipboard.writeText(url);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    }
  };

  // Derive a visual category badge
  const categoryTag = (() => {
    const text = `${build.name} ${build.description}`.toLowerCase();
    if (text.includes("7 inch") || text.includes('7"') || text.includes("long range"))
      return "7″ Long Range";
    if (text.includes("whoop") || text.includes("pavo") || text.includes("cinewhoop"))
      return "CineWhoop";
    if (text.includes("toothpick") || text.includes("3 inch") || text.includes('3"'))
      return "3″ Toothpick";
    if (text.includes("freestyle") || text.includes("5 inch") || text.includes('5"'))
      return "5″ Freestyle";
    return "Quadcopter";
  })();

  const componentCount = [
    build.frameUuid,
    build.motorUuid,
    build.flightControllerUuid,
    ...build.electronicSpeedControllerUuids,
    ...build.receiverUuids,
    ...build.antennaUuids,
    build.propellerUuid,
    ...build.cameraUuids,
    build.videoTransmitterUuid,
    build.gpsReceiverUuid,
  ].filter(Boolean).length;

  const buildColor = getCollectionColor("builds");

  if (variant === "horizontal") {
    return (
      <Link
        to={`/builds/${build.id}`}
        className="group relative flex flex-col sm:flex-row rounded-2xl border border-zinc-200 dark:border-zinc-800 bg-white dark:bg-zinc-900 overflow-hidden shadow-xs hover:shadow-md hover:border-blue-500/50 dark:hover:border-blue-500/50 transition-all duration-200 cursor-pointer block no-underline text-inherit"
      >
        {/* Card trim line */}
        <div className={`h-1 sm:h-auto w-full sm:w-1.5 ${buildColor.trimClass} shrink-0`} />

        {/* Thumbnail area */}
        <div className="relative sm:w-56 h-48 sm:h-auto shrink-0 bg-zinc-900 overflow-hidden">
          {build.primaryDisplayImage ? (
            <img
              src={build.primaryDisplayImage}
              alt=""
              className="w-full h-full object-cover group-hover:scale-105 transition-transform duration-300"
              loading="lazy"
            />
          ) : (
            <div className="w-full h-full min-h-[140px] flex flex-col items-center justify-center p-4 bg-gradient-to-br from-zinc-100 to-zinc-200 dark:from-zinc-900 dark:to-zinc-800 text-zinc-400">
              <Compass size={36} className="transform rotate-45 text-blue-500/60 mb-1" />
              <span className="text-[11px] font-semibold text-zinc-500 uppercase tracking-wider">
                Complete Build
              </span>
            </div>
          )}
          <div className="absolute inset-0 bg-gradient-to-t from-black/60 via-transparent to-transparent sm:hidden" />
          <div className="absolute bottom-2.5 left-3 sm:bottom-2 sm:left-2 flex items-center gap-1.5 text-white text-xs">
            <span className="font-medium bg-black/60 backdrop-blur-xs px-2 py-0.5 rounded text-[11px]">
              {componentCount} components
            </span>
          </div>
        </div>

        {/* Card body */}
        <div className="p-5 flex-1 flex flex-col justify-between min-w-0">
          <div>
            {/* Top row: Title, Category, Match Badge */}
            <div className="flex items-start justify-between gap-3 mb-2">
              <div className="min-w-0">
                <div className="flex items-center gap-2 flex-wrap mb-1">
                  <h4 className="text-base sm:text-lg font-bold text-zinc-900 dark:text-zinc-100 group-hover:text-blue-600 dark:group-hover:text-blue-400 transition-colors">
                    {build.name}
                  </h4>
                  <span className="px-2 py-0.5 rounded-full text-xs font-semibold bg-zinc-100 dark:bg-zinc-800 text-zinc-600 dark:text-zinc-300">
                    {categoryTag}
                  </span>
                </div>
                <div className="flex items-center gap-2 text-xs text-zinc-500 font-mono">
                  <UserBadge username={build.authorUsername} size="xs" />
                  <span>•</span>
                  <CollectionBadge collection="builds" label="Build" size="xs" />
                </div>
              </div>

              {similarityScore !== undefined && (
                <span
                  className={`px-3 py-1 rounded-full text-xs font-extrabold border shrink-0 ${
                    similarityScore >= 80
                      ? "bg-emerald-500/15 text-emerald-600 dark:text-emerald-400 border-emerald-500/30"
                      : similarityScore >= 50
                        ? "bg-blue-500/15 text-blue-600 dark:text-blue-400 border-blue-500/30"
                        : "bg-amber-500/15 text-amber-600 dark:text-amber-400 border-amber-500/30"
                  }`}
                >
                  {Math.round(similarityScore)}% Match
                </span>
              )}
            </div>

            {/* Description */}
            {build.description && (
              <p className="text-sm text-zinc-600 dark:text-zinc-400 line-clamp-2 leading-relaxed mb-3">
                {build.description}
              </p>
            )}

            {/* Matching Specifications Breakdown Chips */}
            {matchDetails && matchDetails.length > 0 && (
              <div className="space-y-1.5 mb-3">
                <div className="text-[11px] font-semibold text-zinc-400 dark:text-zinc-500 uppercase tracking-wider">
                  Matching Specifications
                </div>
                <div className="flex flex-wrap gap-1.5">
                  {matchDetails.map((detail, idx) => (
                    <span
                      key={idx}
                      className={`inline-flex items-center gap-1.5 text-xs px-2.5 py-1 rounded-lg font-medium ${
                        detail.exactMatch
                          ? "bg-blue-500/10 text-blue-600 dark:text-blue-400 border border-blue-500/20"
                          : "bg-zinc-100 dark:bg-zinc-800 text-zinc-700 dark:text-zinc-300 border border-zinc-200 dark:border-zinc-700"
                      }`}
                    >
                      <span className="text-emerald-500 font-bold">✓</span>
                      <span>{detail.description}</span>
                    </span>
                  ))}
                </div>
              </div>
            )}
          </div>

          {/* Bottom Bar: Evaluation Metrics Preview */}
          <div className="pt-3 border-t border-zinc-100 dark:border-zinc-800/80 flex flex-wrap items-center justify-between gap-3 text-xs">
            {evaluation ? (
              <div className="flex items-center gap-4 text-zinc-600 dark:text-zinc-400 flex-wrap">
                <span className="flex items-center gap-1.5 font-medium" title="All Up Weight (AUW)">
                  <Weight size={13} className="text-zinc-400" />
                  <strong className="text-zinc-800 dark:text-zinc-200 font-mono">
                    {evaluation.allUpWeightG.toFixed(2)}g
                  </strong>{" "}
                  AUW
                </span>
                <span
                  className="flex items-center gap-1.5 font-medium text-emerald-600 dark:text-emerald-400"
                  title="Thrust to Weight Ratio"
                >
                  <Gauge size={13} />
                  <strong className="font-mono">
                    {evaluation.thrustToWeightRatio.toFixed(1)}:1
                  </strong>{" "}
                  TWR
                </span>
                {evaluation.maxFlightTimeMin > 0 && (
                  <span
                    className="flex items-center gap-1.5 font-medium text-blue-600 dark:text-blue-400"
                    title="Flight Time"
                  >
                    <Clock size={13} />
                    <strong className="font-mono">
                      {evaluation.minFlightTimeMin > 0
                        ? `${evaluation.minFlightTimeMin.toFixed(1)}–${evaluation.maxFlightTimeMin.toFixed(1)} min`
                        : `~${evaluation.maxFlightTimeMin.toFixed(1)} min`}
                    </strong>
                  </span>
                )}
                {evaluation.topSpeedKmh > 0 && (
                  <span
                    className="flex items-center gap-1.5 font-medium text-indigo-600 dark:text-indigo-400"
                    title="Top Speed"
                  >
                    <Compass size={13} />
                    <strong className="font-mono">{Math.round(evaluation.topSpeedKmh)}</strong> km/h
                  </span>
                )}
              </div>
            ) : !isLoadingLimits && !electricalLimits?.defaultBatteryId ? (
              <div className="text-red-500/80 dark:text-red-400/80 text-xs italic">
                No compatible battery
              </div>
            ) : (
              <div className="text-zinc-400 text-xs italic">Evaluating build specs...</div>
            )}

            <div className="flex items-center gap-1 text-zinc-400 group-hover:text-blue-600 dark:group-hover:text-blue-400 transition-colors ml-auto">
              <ChevronRight size={16} className="group-hover:translate-x-1 transition-transform" />
            </div>
          </div>
        </div>
      </Link>
    );
  }

  return (
    <article className="group relative flex flex-col rounded-2xl border border-zinc-200 dark:border-zinc-800 bg-white dark:bg-zinc-900 overflow-hidden shadow-xs hover:shadow-md transition-all duration-200">
      {/* Card top trim line */}
      <div className={`h-1 w-full ${buildColor.trimClass}`} />

      {/* Header bar: Avatar, Title, Handle */}
      <div className="p-4 sm:p-5 pb-3 flex items-start justify-between gap-3">
        <div className="flex items-center gap-3 min-w-0">
          {build.primaryDisplayImage ? (
            <img
              src={build.primaryDisplayImage}
              alt=""
              className="w-10 h-10 rounded-xl object-cover shadow-xs shrink-0 border border-zinc-200 dark:border-zinc-800"
            />
          ) : (
            <div className="w-10 h-10 rounded-xl bg-gradient-to-br from-blue-500 to-indigo-600 dark:from-blue-600 dark:to-indigo-700 flex items-center justify-center text-white font-bold text-sm shadow-xs shrink-0 select-none">
              {build.name.charAt(0) || "Q"}
            </div>
          )}
          <div className="min-w-0">
            <Link
              to={`/builds/${build.id}`}
              className="text-base sm:text-lg font-bold text-zinc-900 dark:text-zinc-100 hover:text-blue-600 dark:hover:text-blue-400 transition-colors line-clamp-1"
            >
              {build.name}
            </Link>
            <div className="flex items-center gap-2 text-xs text-zinc-500 font-mono">
              <UserBadge username={build.authorUsername} size="xs" />
              <span>•</span>
              <CollectionBadge collection="builds" label="Build" size="xs" />
              <span>•</span>
              <span className="text-blue-600 dark:text-blue-400 font-sans font-medium">
                {categoryTag}
              </span>
            </div>
          </div>
        </div>

        <div className="flex items-center gap-2 shrink-0">
          {similarityScore !== undefined && (
            <span
              className={`px-2.5 py-1 rounded-full text-xs font-extrabold border ${
                similarityScore >= 80
                  ? "bg-emerald-500/15 text-emerald-600 dark:text-emerald-400 border-emerald-500/30"
                  : similarityScore >= 50
                    ? "bg-blue-500/15 text-blue-600 dark:text-blue-400 border-blue-500/30"
                    : "bg-amber-500/15 text-amber-600 dark:text-amber-400 border-amber-500/30"
              }`}
            >
              {Math.round(similarityScore)}% Match
            </span>
          )}
          <button
            type="button"
            onClick={handleShare}
            title="Share build link"
            className="p-2 rounded-lg text-zinc-400 hover:text-zinc-700 dark:hover:text-zinc-200 hover:bg-zinc-100 dark:hover:bg-zinc-800 transition-colors shrink-0 cursor-pointer"
          >
            {copied ? <Check size={16} className="text-emerald-500" /> : <Share2 size={16} />}
          </button>
        </div>
      </div>

      {/* Visual / Banner area with schematic styling or build photo */}
      <Link
        to={`/builds/${build.id}`}
        tabIndex={-1}
        aria-hidden="true"
        className="block relative bg-gradient-to-b from-zinc-100/80 to-zinc-50 dark:from-zinc-950/60 dark:to-zinc-900/40 border-y border-zinc-100 dark:border-zinc-800/60 overflow-hidden"
      >
        {build.primaryDisplayImage ? (
          <div className="relative h-44 w-full overflow-hidden bg-zinc-900">
            <img
              src={build.primaryDisplayImage}
              alt=""
              className="w-full h-full object-cover group-hover:scale-105 transition-transform duration-300"
              loading="lazy"
            />
            <div className="absolute inset-0 bg-gradient-to-t from-black/60 via-transparent to-transparent" />
            <div className="absolute bottom-2.5 left-3 right-3 flex items-center justify-between text-white text-xs">
              <span className="font-medium bg-black/40 backdrop-blur-xs px-2 py-0.5 rounded">
                {componentCount} components
              </span>
            </div>
          </div>
        ) : (
          <div className="px-5 py-6 text-center">
            <div className="relative z-10 flex flex-col items-center justify-center py-2">
              {/* Stylized Quad Silhouette */}
              <div className="w-20 h-20 rounded-full bg-blue-50 dark:bg-blue-950/40 border border-blue-100 dark:border-blue-900/60 flex items-center justify-center text-blue-600 dark:text-blue-400 mb-3 shadow-inner group-hover:scale-105 transition-transform duration-200">
                <Compass size={38} className="transform rotate-45" />
              </div>
              <span className="text-xs uppercase tracking-wider font-semibold text-zinc-400 dark:text-zinc-500">
                Complete Build Profile
              </span>
              <span className="text-sm font-medium text-zinc-700 dark:text-zinc-300 mt-0.5">
                {componentCount} hardware components configured
              </span>
            </div>
          </div>
        )}
      </Link>

      {/* Card Body: Description */}
      <div className="p-4 sm:p-5 pt-3 flex-1 flex flex-col justify-between">
        <div>
          {build.description && (
            <p className="text-sm text-zinc-600 dark:text-zinc-400 line-clamp-2 mb-4 leading-relaxed">
              {build.description}
            </p>
          )}

          {/* Quick Hardware Component Badges */}
          <div className="flex flex-wrap gap-1.5 mb-4">
            {build.frameUuid && (
              <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded-md text-xs bg-zinc-100 dark:bg-zinc-800/80 text-zinc-600 dark:text-zinc-400 border border-zinc-200/50 dark:border-zinc-700/50">
                <Layers size={11} className="text-zinc-400" />
                <span>Frame</span>
              </span>
            )}
            {build.motorUuid && (
              <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded-md text-xs bg-zinc-100 dark:bg-zinc-800/80 text-zinc-600 dark:text-zinc-400 border border-zinc-200/50 dark:border-zinc-700/50">
                <Cpu size={11} className="text-zinc-400" />
                <span>4x Motors</span>
              </span>
            )}
            {build.flightControllerUuid && (
              <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded-md text-xs bg-zinc-100 dark:bg-zinc-800/80 text-zinc-600 dark:text-zinc-400 border border-zinc-200/50 dark:border-zinc-700/50">
                <Cpu size={11} className="text-zinc-400" />
                <span>FC Stack</span>
              </span>
            )}
            {build.videoTransmitterUuid && (
              <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded-md text-xs bg-zinc-100 dark:bg-zinc-800/80 text-zinc-600 dark:text-zinc-400 border border-zinc-200/50 dark:border-zinc-700/50">
                <Radio size={11} className="text-zinc-400" />
                <span>VTX</span>
              </span>
            )}
            {build.cameraUuids.length > 0 && (
              <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded-md text-xs bg-zinc-100 dark:bg-zinc-800/80 text-zinc-600 dark:text-zinc-400 border border-zinc-200/50 dark:border-zinc-700/50">
                <Camera size={11} className="text-zinc-400" />
                <span>Cam</span>
              </span>
            )}
          </div>

          {/* Similarity Matching Breakdown Chips */}
          {matchDetails && matchDetails.length > 0 && (
            <div className="mb-4 pt-3 border-t border-zinc-100 dark:border-zinc-800/80 space-y-1.5">
              <div className="text-[11px] font-semibold text-zinc-400 dark:text-zinc-500 uppercase tracking-wider">
                Matching Specifications
              </div>
              <div className="flex flex-wrap gap-1.5">
                {matchDetails.map((detail, idx) => (
                  <span
                    key={idx}
                    className={`inline-flex items-center gap-1 text-[11px] px-2 py-0.5 rounded-md font-medium ${
                      detail.exactMatch
                        ? "bg-blue-500/10 text-blue-600 dark:text-blue-400 border border-blue-500/20"
                        : "bg-zinc-100 dark:bg-zinc-800 text-zinc-700 dark:text-zinc-300 border border-zinc-200 dark:border-zinc-700"
                    }`}
                  >
                    <span>✓</span>
                    <span>{detail.description}</span>
                  </span>
                ))}
              </div>
            </div>
          )}
        </div>

        {/* Evaluation Metrics Preview */}
        <div className="pt-3 border-t border-zinc-100 dark:border-zinc-800/60 flex items-center justify-between text-xs">
          {evaluation ? (
            <div className="flex items-center gap-3 text-zinc-600 dark:text-zinc-400">
              <span className="flex items-center gap-1 font-medium" title="All Up Weight (AUW)">
                <Weight size={13} className="text-zinc-400" />
                {evaluation.allUpWeightG.toFixed(2)}g
              </span>
              <span
                className="flex items-center gap-1 font-medium text-emerald-600 dark:text-emerald-400"
                title="Thrust to Weight Ratio"
              >
                <Gauge size={13} />
                {evaluation.thrustToWeightRatio.toFixed(1)}:1 TWR
              </span>
              {evaluation.maxFlightTimeMin > 0 && (
                <span
                  className="flex items-center gap-1 font-medium text-blue-600 dark:text-blue-400"
                  title="Estimated Flight Time (Freestyle to Cruise)"
                >
                  <Clock size={13} />
                  {evaluation.minFlightTimeMin > 0
                    ? `${evaluation.minFlightTimeMin.toFixed(1)}–${evaluation.maxFlightTimeMin.toFixed(1)} min`
                    : `~${evaluation.maxFlightTimeMin.toFixed(1)} min`}
                </span>
              )}
            </div>
          ) : !isLoadingLimits && !electricalLimits?.defaultBatteryId ? (
            <div className="text-red-500/80 dark:text-red-400/80 text-xs italic">
              No compatible battery
            </div>
          ) : (
            <div className="text-zinc-400 text-xs italic">Evaluating build specs...</div>
          )}

          <div className="flex items-center gap-3">
            <Link
              to={`/builds/new?from=${build.id || build.uuid}`}
              className="flex items-center gap-1 text-xs font-semibold text-zinc-500 hover:text-zinc-800 dark:hover:text-zinc-200 transition-colors"
              title="Build similar aircraft starting with these components"
            >
              <Copy size={12} />
              <span>Build Similar</span>
            </Link>
            <Link
              to={`/builds/${build.id}`}
              className="flex items-center gap-1 text-xs font-semibold text-blue-600 dark:text-blue-400 hover:text-blue-700 dark:hover:text-blue-300 group-hover:translate-x-0.5 transition-all"
            >
              <span>View Profile</span>
              <ChevronRight size={14} />
            </Link>
          </div>
        </div>
      </div>
    </article>
  );
}
