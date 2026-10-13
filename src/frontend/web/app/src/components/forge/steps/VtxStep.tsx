import { useMemo, useEffect } from "react";
import {
  Search,
  X,
  ArrowRight,
  ArrowLeft,
  AlertCircle,
  EyeOff,
  Sparkles,
  Check,
  Lock,
} from "lucide-react";
import { useForge, checkVtxAntennaCompatibility, isMountPatternCompatible } from "../ForgeContext";
import { ProductHoverCard } from "../../ProductHoverCard";
import { CollectionIcon } from "../../CollectionIcon";
import { CompatibilityHoverCard } from "../../CompatibilityHoverCard";
import { getCollectionColor } from "../../../lib/collectionColors";
import type { VideoTransmitter } from "../../../gen/quadsmith/api/video_transmitter_pb";

export function VtxStep() {
  const {
    vtxs,
    selectedVtx,
    setSelectedVtx,
    useIntegratedVtx,
    setUseIntegratedVtx,
    integratedVtx,
    fcHasIntegratedVtx,
    selectedFrame,
    selectedVtxAnt,
    antennas,
    isVtxAntennaIntegrated,
    noneSelections,
    setNoneSelections,
    selectedCam,
    handleNextStep,
    handlePrevStep,
    searchQueries,
    setSearchQuery,
    compatibleOnlyMap,
    toggleCompatibleOnly,
    getCelFilterForStep,
    stepConflicts,
  } = useForge();

  useEffect(() => {
    const timer = setTimeout(() => {
      document
        .getElementById("selected-vtx-item")
        ?.scrollIntoView({ behavior: "smooth", block: "center" });
    }, 50);
    return () => clearTimeout(timer);
  }, []);

  const search = searchQueries.video_transmitters || "";
  const conflictMsg = stepConflicts[9];
  const isCompatibleOnly = compatibleOnlyMap.video_transmitters ?? true;
  const isOmitted = noneSelections.vtx;
  const isUsingIntegrated = useIntegratedVtx && !!integratedVtx;

  // Check if camera is digital
  const isCameraDigital = useMemo(() => {
    if (!selectedCam || !selectedCam.protocol) return false;
    const proto = selectedCam.protocol.toLowerCase();
    return !proto.includes("analog");
  }, [selectedCam]);

  const checkVtxCompatibility = (v: VideoTransmitter) => {
    // Protocol matching with selected camera
    if (selectedCam?.protocol && !noneSelections.camera && v.protocol) {
      const camProto = selectedCam.protocol.toLowerCase();
      const vProto = v.protocol.toLowerCase();
      let match = false;
      if (camProto.includes("analog")) match = vProto.includes("analog");
      else if (camProto.includes("dji")) match = vProto.includes("dji");
      else if (camProto.includes("walksnail") || camProto.includes("avatar")) {
        match = vProto.includes("walksnail") || vProto.includes("avatar");
      } else if (camProto.includes("hdzero")) {
        match = vProto.includes("hdzero");
      } else {
        match = vProto === camProto;
      }
      if (!match) {
        return {
          compatible: false,
          reason: `VTX protocol (${v.protocol}) is incompatible with selected camera (${selectedCam.protocol}).`,
        };
      }
    }

    // Frame board mount pattern
    if (selectedFrame?.boardMountPatterns?.length && v.mountPattern) {
      if (!isMountPatternCompatible(v.mountPattern, selectedFrame.boardMountPatterns)) {
        return {
          compatible: false,
          reason: `VTX mounting pattern (${v.mountPattern}) is not supported by frame board mounts (${selectedFrame.boardMountPatterns.join(", ")}).`,
        };
      }
    }

    // VTX Antenna compatibility
    if (selectedVtxAnt && !noneSelections.vtxAntenna && !isVtxAntennaIntegrated) {
      const check = checkVtxAntennaCompatibility(v, selectedVtxAnt, antennas);
      if (!check.compatible) {
        return check;
      }
    }

    return { compatible: true };
  };

  const filteredVtxs = useMemo(() => {
    let list = vtxs;

    if (isCompatibleOnly) {
      list = list.filter(
        (v) =>
          v.uuid === selectedVtx?.uuid ||
          v.id === selectedVtx?.id ||
          checkVtxCompatibility(v).compatible,
      );
    }

    if (!search.trim()) return list;
    const q = search.toLowerCase();
    return list.filter(
      (v) =>
        v.name.toLowerCase().includes(q) ||
        v.manufacturer.toLowerCase().includes(q) ||
        v.protocol?.toLowerCase().includes(q),
    );
  }, [
    vtxs,
    search,
    isCompatibleOnly,
    selectedCam,
    noneSelections.camera,
    selectedFrame,
    selectedVtxAnt,
    noneSelections.vtxAntenna,
    isVtxAntennaIntegrated,
    antennas,
    selectedVtx,
  ]);

  return (
    <div className="space-y-4">
      {/* Compatibility Conflict Banner */}
      {conflictMsg && (
        <div className="p-3.5 rounded-xl border border-amber-500/30 bg-amber-500/10 text-amber-800 dark:text-amber-300 flex items-start gap-3 text-xs">
          <AlertCircle size={16} className="text-amber-500 mt-0.5 shrink-0" />
          <div>
            <span className="font-bold">Incompatibility Detected</span>
            <p className="opacity-90 mt-0.5">{conflictMsg}</p>
          </div>
        </div>
      )}

      {/* Main Container */}
      <div
        id="section-vtx"
        className="p-4 rounded-2xl border border-zinc-200 dark:border-zinc-800 bg-white dark:bg-zinc-900 space-y-3"
      >
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-2">
            <CollectionIcon
              collectionId="video_transmitters"
              size={16}
              color={getCollectionColor("video_transmitters")}
            />
            <h3 className="font-bold text-sm text-zinc-900 dark:text-zinc-100">
              9. Video Transmitter (VTX)
            </h3>
            <span className="text-[10px] uppercase font-bold tracking-wider px-2 py-0.5 rounded bg-zinc-100 dark:bg-zinc-800 text-zinc-600 dark:text-zinc-400">
              Optional
            </span>
          </div>
          {selectedVtx || isUsingIntegrated ? (
            <button
              type="button"
              onClick={() => {
                document
                  .getElementById(isUsingIntegrated ? "selected-fc-vtx-item" : "selected-vtx-item")
                  ?.scrollIntoView({ behavior: "smooth", block: "center" });
              }}
              title="Scroll to selected VTX"
              className="text-xs font-semibold text-emerald-600 dark:text-emerald-400 hover:text-emerald-700 dark:hover:text-emerald-300 hover:underline flex items-center gap-1 px-1.5 py-0.5 rounded bg-emerald-50 dark:bg-emerald-950/40 border border-emerald-200 dark:border-emerald-800/60 transition-all cursor-pointer shadow-2xs"
            >
              <Check size={12} strokeWidth={2.5} /> Selected
            </button>
          ) : (
            <span className="text-xs font-semibold text-blue-600 dark:text-blue-400 truncate max-w-[200px]">
              {isOmitted ? "None (Omitted)" : "None selected"}
            </span>
          )}
        </div>

        <p className="text-xs text-zinc-500 dark:text-zinc-400">
          The VTX broadcasts the camera's video signal to your FPV goggles or monitor. Protocol must
          match the selected camera (Analog, DJI, Walksnail, or HDZero).
        </p>

        {noneSelections.camera ? (
          <div className="p-4 rounded-xl border border-zinc-200 dark:border-zinc-800 bg-zinc-50 dark:bg-zinc-900/40 text-xs space-y-2">
            <div className="flex items-center gap-2 text-zinc-700 dark:text-zinc-300 font-bold">
              <Lock size={16} className="text-zinc-500 shrink-0" />
              <span>Camera Omitted — Video Transmitter Locked to None</span>
            </div>
            <p className="text-zinc-500 dark:text-zinc-400 text-[11px] leading-relaxed">
              Because no FPV camera was selected in Step 8, the video transmitter has been
              automatically set to None and locked. External VTX antennas are also omitted.
            </p>
          </div>
        ) : (
          <>
            {/* Option Cards: FC Integrated VTX (only shown if Camera is NOT digital) & Omit VTX */}
            <div className="space-y-2">
              {fcHasIntegratedVtx && integratedVtx && !isCameraDigital && (
                <div
                  role="button"
                  tabIndex={0}
                  id={isUsingIntegrated ? "selected-vtx-item" : undefined}
                  onClick={() => {
                    setUseIntegratedVtx(true);
                    setSelectedVtx(integratedVtx);
                    setNoneSelections((prev) => ({ ...prev, vtx: false }));
                  }}
                  className={`p-3.5 rounded-xl border transition-all cursor-pointer flex items-center justify-between gap-3 ${
                    isUsingIntegrated
                      ? "border-blue-500 bg-blue-50/50 dark:bg-blue-950/30 ring-1 ring-blue-500 shadow-xs"
                      : "border-purple-500/30 bg-purple-500/5 hover:border-purple-500/60"
                  }`}
                >
                  <div className="flex items-center gap-3 min-w-0">
                    <div className="w-8 h-8 rounded-lg bg-orange-500/10 text-orange-600 dark:text-orange-400 flex items-center justify-center shrink-0">
                      <Sparkles size={16} />
                    </div>
                    <div className="min-w-0">
                      <div className="flex items-center gap-2">
                        <span className="text-xs font-bold text-zinc-900 dark:text-zinc-100">
                          Use FC Integrated VTX
                        </span>
                        <span className="text-[10px] font-semibold px-2 py-0.5 rounded-full bg-orange-500/10 text-orange-600 dark:text-orange-400 border border-orange-500/20">
                          Analog Built-in
                        </span>
                      </div>
                      <p className="text-[11px] text-zinc-500 dark:text-zinc-400 mt-0.5 truncate">
                        {integratedVtx.name} • {integratedVtx.maxPowerMw}mW Max Power
                      </p>
                    </div>
                  </div>

                  <div className="w-4 h-4 rounded-full border border-blue-500 flex items-center justify-center shrink-0">
                    {isUsingIntegrated && <div className="w-2 h-2 rounded-full bg-blue-500" />}
                  </div>
                </div>
              )}

              {/* Omit VTX Card */}
              <div
                role="button"
                tabIndex={0}
                id={isOmitted ? "selected-vtx-item" : undefined}
                onClick={() => {
                  setNoneSelections((prev) => ({ ...prev, vtx: true, vtxAntenna: true }));
                  setSelectedVtx(null);
                  setUseIntegratedVtx(false);
                }}
                className={`p-3.5 rounded-xl border transition-all cursor-pointer flex items-center justify-between gap-3 ${
                  isOmitted
                    ? "border-blue-500 bg-blue-50/50 dark:bg-blue-950/30 ring-1 ring-blue-500 shadow-xs"
                    : "border-zinc-200 dark:border-zinc-800 bg-zinc-50/50 dark:bg-zinc-900/40 hover:border-zinc-400 dark:hover:border-zinc-700"
                }`}
              >
                <div className="flex items-center gap-3 min-w-0">
                  <div className="w-8 h-8 rounded-lg bg-zinc-200 dark:bg-zinc-800 text-zinc-600 dark:text-zinc-400 flex items-center justify-center shrink-0">
                    <EyeOff size={16} />
                  </div>
                  <div className="min-w-0">
                    <span className="text-xs font-bold text-zinc-900 dark:text-zinc-100">
                      Omit Video Transmitter
                    </span>
                    <p className="text-[11px] text-zinc-500 dark:text-zinc-400 mt-0.5">
                      Build without a video transmitter or use standalone onboard recording.
                    </p>
                  </div>
                </div>

                <div className="w-4 h-4 rounded-full border border-blue-500 flex items-center justify-center shrink-0">
                  {isOmitted && <div className="w-2 h-2 rounded-full bg-blue-500" />}
                </div>
              </div>
            </div>

            {/* Search & Compatibility Toggle */}
            <div className="flex items-center gap-2">
              <div className="relative flex-1">
                <Search
                  size={14}
                  className="absolute left-3 top-1/2 -translate-y-1/2 text-zinc-400"
                />
                <input
                  type="text"
                  value={search}
                  onChange={(e) => setSearchQuery("video_transmitters", e.target.value)}
                  placeholder="Search VTXs by name, brand, protocol..."
                  className="w-full bg-zinc-50 dark:bg-zinc-950 border border-zinc-200 dark:border-zinc-800 rounded-xl pl-9 pr-8 py-1.5 text-xs text-zinc-900 dark:text-zinc-100 placeholder-zinc-400 focus:outline-none focus:ring-1 focus:ring-blue-500"
                />
                {search && (
                  <button
                    type="button"
                    onClick={() => setSearchQuery("video_transmitters", "")}
                    className="absolute right-2.5 top-1/2 -translate-y-1/2 text-zinc-400 hover:text-zinc-600"
                  >
                    <X size={13} />
                  </button>
                )}
              </div>

              <label className="flex items-center gap-1.5 cursor-pointer text-xs text-zinc-600 dark:text-zinc-400 select-none shrink-0">
                <input
                  type="checkbox"
                  checked={isCompatibleOnly}
                  onChange={() => toggleCompatibleOnly("video_transmitters")}
                  className="rounded border-zinc-300 dark:border-zinc-700 text-blue-600 focus:ring-blue-500 h-3.5 w-3.5 cursor-pointer"
                />
                <CompatibilityHoverCard filter={getCelFilterForStep(9)}>
                  <span>Compatible only</span>
                </CompatibilityHoverCard>
              </label>
            </div>

            {/* VTX Catalog List */}
            <div className="overflow-y-auto max-h-80 space-y-2 pr-1.5 focus:outline-none">
              {filteredVtxs.map((v: VideoTransmitter) => {
                const isSelected =
                  !isOmitted &&
                  !isUsingIntegrated &&
                  (selectedVtx?.uuid === v.uuid || selectedVtx?.id === v.id);
                const compat = checkVtxCompatibility(v);
                const isIncompatible = !compat.compatible;

                return (
                  <ProductHoverCard
                    key={v.uuid || v.id}
                    collectionId="video_transmitters"
                    item={v}
                    productId={v.id || v.uuid}
                    as="div"
                    placement="side"
                    id={isSelected ? "selected-vtx-item" : undefined}
                    className={`p-3 rounded-xl border transition-all cursor-pointer flex flex-col sm:flex-row sm:items-center justify-between gap-2 ${
                      isSelected
                        ? isIncompatible
                          ? "border-amber-500 bg-amber-50/50 dark:bg-amber-950/30 ring-1 ring-amber-500 shadow-xs"
                          : "border-blue-500 bg-blue-50/50 dark:bg-blue-950/30 ring-1 ring-blue-500 shadow-xs"
                        : isIncompatible
                          ? "border-zinc-200 dark:border-zinc-800 bg-zinc-50/30 dark:bg-zinc-900/30 opacity-60 hover:opacity-100 hover:border-zinc-400 dark:hover:border-zinc-700"
                          : "border-zinc-200 dark:border-zinc-800 bg-zinc-50/50 dark:bg-zinc-900/60 hover:border-zinc-400 dark:hover:border-zinc-700"
                    }`}
                  >
                    <div
                      role="button"
                      tabIndex={0}
                      onClick={() => {
                        setNoneSelections((prev) => ({ ...prev, vtx: false }));
                        setUseIntegratedVtx(false);
                        setSelectedVtx(v);
                      }}
                      className="flex-1 min-w-0"
                    >
                      <div className="flex items-center gap-2 flex-wrap">
                        <span className="text-xs font-bold text-zinc-900 dark:text-zinc-100 truncate">
                          {v.name}
                        </span>
                        {isIncompatible && (
                          <span
                            title={compat.reason}
                            className="text-[10px] px-2 py-0.5 rounded-full font-semibold bg-amber-500/10 text-amber-600 dark:text-amber-400 border border-amber-500/20 shrink-0 flex items-center gap-1"
                          >
                            <AlertCircle size={10} />
                            <span>Incompatible</span>
                          </span>
                        )}
                        {v.protocol && (
                          <span className="text-[10px] px-2 py-0.5 rounded-full font-semibold bg-purple-500/10 text-purple-600 dark:text-purple-400 border border-purple-500/20 shrink-0">
                            {v.protocol}
                          </span>
                        )}
                        {v.maxPowerMw ? (
                          <span className="text-[10px] px-2 py-0.5 rounded-full font-semibold bg-orange-500/10 text-orange-600 dark:text-orange-400 border border-orange-500/20 shrink-0">
                            {v.maxPowerMw}mW
                          </span>
                        ) : null}
                      </div>
                      <div className="text-[11px] text-zinc-500 dark:text-zinc-400 mt-0.5 flex items-center gap-2 flex-wrap">
                        <span>by {v.manufacturer}</span>
                        {v.weightG ? <span>• {v.weightG.toFixed(1)}g</span> : null}
                        {v.inputVoltageMinV && v.inputVoltageMaxV ? (
                          <span>
                            • {v.inputVoltageMinV}V - {v.inputVoltageMaxV}V
                          </span>
                        ) : null}
                      </div>
                    </div>

                    <div className="flex items-center gap-2 self-end sm:self-center shrink-0">
                      <div className="w-4 h-4 rounded-full border border-blue-500 flex items-center justify-center">
                        {isSelected && <div className="w-2 h-2 rounded-full bg-blue-500" />}
                      </div>
                    </div>
                  </ProductHoverCard>
                );
              })}

              {filteredVtxs.length === 0 && (
                <p className="text-xs text-zinc-400 italic py-2 text-center">
                  No video transmitters found matching your criteria.
                </p>
              )}
            </div>
          </>
        )}
      </div>

      {/* Navigation Footer */}
      <div className="flex items-center justify-between">
        <button
          type="button"
          onClick={handlePrevStep}
          className="flex items-center gap-1.5 px-4 py-2 rounded-xl text-xs font-semibold text-zinc-600 dark:text-zinc-400 hover:bg-zinc-100 dark:hover:bg-zinc-800 transition-colors"
        >
          <ArrowLeft size={14} />
          <span>Back: Camera</span>
        </button>

        <button
          type="button"
          disabled={
            !noneSelections.camera &&
            ((!selectedVtx && !isUsingIntegrated && !isOmitted) || !!conflictMsg)
          }
          onClick={handleNextStep}
          className="flex items-center gap-2 px-5 py-2.5 rounded-xl font-bold text-xs bg-blue-600 hover:bg-blue-700 disabled:opacity-40 disabled:cursor-not-allowed text-white transition-all shadow-sm"
        >
          <span>
            {noneSelections.camera
              ? "Next: GPS (Camera Omitted)"
              : isOmitted || isVtxAntennaIntegrated
                ? "Next: GPS (Antenna Skipped)"
                : "Next: VTX Antennas"}
          </span>
          <ArrowRight size={14} />
        </button>
      </div>
    </div>
  );
}
