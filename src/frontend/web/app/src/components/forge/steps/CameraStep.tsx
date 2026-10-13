import { useMemo, useEffect } from "react";
import { Search, X, ArrowRight, ArrowLeft, AlertCircle, EyeOff, Check } from "lucide-react";
import { useForge, isCameraMountCompatible } from "../ForgeContext";
import { ProductHoverCard } from "../../ProductHoverCard";
import { CollectionIcon } from "../../CollectionIcon";
import { CompatibilityHoverCard } from "../../CompatibilityHoverCard";
import { getCollectionColor } from "../../../lib/collectionColors";
import type { Camera } from "../../../gen/quadsmith/api/camera_pb";

export function CameraStep() {
  const {
    cams,
    selectedCam,
    setSelectedCam,
    noneSelections,
    setNoneSelections,
    selectedFrame,
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
        .getElementById("selected-cam-item")
        ?.scrollIntoView({ behavior: "smooth", block: "center" });
    }, 50);
    return () => clearTimeout(timer);
  }, []);

  const search = searchQueries.cameras || "";
  const conflictMsg = stepConflicts[8];
  const isCompatibleOnly = compatibleOnlyMap.cameras ?? true;
  const isOmitted = noneSelections.camera;

  const checkCameraCompatibility = (c: Camera) => {
    // Camera mount interface vs frame
    if (selectedFrame?.cameraMountInterfaces?.length && c.cameraMountInterface) {
      if (!isCameraMountCompatible(c.cameraMountInterface, selectedFrame.cameraMountInterfaces)) {
        return {
          compatible: false,
          reason: `Camera mount interface (${c.cameraMountInterface}) is not supported by frame camera mounts (${selectedFrame.cameraMountInterfaces.join(", ")}).`,
        };
      }
    }

    return { compatible: true };
  };

  const filteredCams = useMemo(() => {
    let list = cams;

    if (isCompatibleOnly) {
      list = list.filter(
        (c) =>
          c.uuid === selectedCam?.uuid ||
          c.id === selectedCam?.id ||
          checkCameraCompatibility(c).compatible,
      );
    }

    if (!search.trim()) return list;
    const q = search.toLowerCase();
    return list.filter(
      (c) =>
        c.name.toLowerCase().includes(q) ||
        c.manufacturer.toLowerCase().includes(q) ||
        c.protocol?.toLowerCase().includes(q) ||
        c.cameraMountInterface?.toLowerCase().includes(q),
    );
  }, [cams, search, isCompatibleOnly, selectedFrame, selectedCam]);

  return (
    <div className="space-y-4">
      {/* Compatibility Conflict Banner */}
      {conflictMsg && (
        <div className="p-3.5 rounded-xl border border-amber-500/30 bg-amber-500/10 text-amber-800 dark:text-amber-300 flex items-start gap-3 text-xs">
          <AlertCircle size={16} className="text-amber-500 mt-0.5 shrink-0" />
          <div>
            <span className="font-bold">Incompatibility Detected with Frame Mount</span>
            <p className="opacity-90 mt-0.5">{conflictMsg}</p>
          </div>
        </div>
      )}

      {/* Main Container */}
      <div
        id="section-cam"
        className="p-4 rounded-2xl border border-zinc-200 dark:border-zinc-800 bg-white dark:bg-zinc-900 space-y-3"
      >
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-2">
            <CollectionIcon
              collectionId="cameras"
              size={16}
              color={getCollectionColor("cameras")}
            />
            <h3 className="font-bold text-sm text-zinc-900 dark:text-zinc-100">8. FPV Camera</h3>
            <span className="text-[10px] uppercase font-bold tracking-wider px-2 py-0.5 rounded bg-zinc-100 dark:bg-zinc-800 text-zinc-600 dark:text-zinc-400">
              Optional
            </span>
          </div>
          {selectedCam ? (
            <button
              type="button"
              onClick={() => {
                document
                  .getElementById("selected-cam-item")
                  ?.scrollIntoView({ behavior: "smooth", block: "center" });
              }}
              title="Scroll to selected camera"
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
          The FPV camera captures the pilot's forward video feed. The mounting interface (Nano 14mm,
          Micro 19mm, DJI O3 20mm) must be supported by the frame.
        </p>

        {/* Omit Camera Card */}
        <div
          role="button"
          tabIndex={0}
          id={isOmitted ? "selected-cam-item" : undefined}
          onClick={() => {
            setNoneSelections((prev) => ({ ...prev, camera: true, vtx: true, vtxAntenna: true }));
            setSelectedCam(null);
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
                No Camera (Omitted)
              </span>
              <p className="text-[11px] text-zinc-500 dark:text-zinc-400 mt-0.5">
                Build without an FPV camera. This also omits video transmitter and video antennas.
              </p>
            </div>
          </div>

          <div className="w-4 h-4 rounded-full border border-blue-500 flex items-center justify-center shrink-0">
            {isOmitted && <div className="w-2 h-2 rounded-full bg-blue-500" />}
          </div>
        </div>

        {/* Search & Compatibility Toggle */}
        <div className="flex items-center gap-2">
          <div className="relative flex-1">
            <Search size={14} className="absolute left-3 top-1/2 -translate-y-1/2 text-zinc-400" />
            <input
              type="text"
              value={search}
              onChange={(e) => setSearchQuery("cameras", e.target.value)}
              placeholder="Search cameras by name, brand, mount interface, protocol..."
              className="w-full bg-zinc-50 dark:bg-zinc-950 border border-zinc-200 dark:border-zinc-800 rounded-xl pl-9 pr-8 py-1.5 text-xs text-zinc-900 dark:text-zinc-100 placeholder-zinc-400 focus:outline-none focus:ring-1 focus:ring-blue-500"
            />
            {search && (
              <button
                type="button"
                onClick={() => setSearchQuery("cameras", "")}
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
              onChange={() => toggleCompatibleOnly("cameras")}
              className="rounded border-zinc-300 dark:border-zinc-700 text-blue-600 focus:ring-blue-500 h-3.5 w-3.5 cursor-pointer"
            />
            <CompatibilityHoverCard filter={getCelFilterForStep(8)}>
              <span>Compatible only</span>
            </CompatibilityHoverCard>
          </label>
        </div>

        {/* Camera Catalog List */}
        <div className="overflow-y-auto max-h-80 space-y-2 pr-1.5 focus:outline-none">
          {filteredCams.map((c: Camera) => {
            const isSelected =
              !isOmitted && (selectedCam?.uuid === c.uuid || selectedCam?.id === c.id);
            const compat = checkCameraCompatibility(c);
            const isIncompatible = !compat.compatible;

            return (
              <ProductHoverCard
                key={c.uuid || c.id}
                collectionId="cameras"
                item={c}
                productId={c.id || c.uuid}
                as="div"
                placement="side"
                id={isSelected ? "selected-cam-item" : undefined}
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
                    setNoneSelections((prev) => ({ ...prev, camera: false }));
                    setSelectedCam(c);
                  }}
                  className="flex-1 min-w-0"
                >
                  <div className="flex items-center gap-2 flex-wrap">
                    <span className="text-xs font-bold text-zinc-900 dark:text-zinc-100 truncate">
                      {c.name}
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
                    {c.cameraMountInterface && (
                      <span className="text-[10px] px-2 py-0.5 rounded-full font-semibold bg-emerald-500/10 text-emerald-600 dark:text-emerald-400 border border-emerald-500/20 shrink-0">
                        {c.cameraMountInterface}
                      </span>
                    )}
                    {c.protocol && (
                      <span className="text-[10px] px-2 py-0.5 rounded-full font-semibold bg-purple-500/10 text-purple-600 dark:text-purple-400 border border-purple-500/20 shrink-0">
                        {c.protocol}
                      </span>
                    )}
                    {c.sensorSize && (
                      <span className="text-[10px] px-2 py-0.5 rounded-full font-semibold bg-blue-500/10 text-blue-600 dark:text-blue-400 border border-blue-500/20 shrink-0">
                        {c.sensorSize}
                      </span>
                    )}
                  </div>
                  <div className="text-[11px] text-zinc-500 dark:text-zinc-400 mt-0.5 flex items-center gap-2 flex-wrap">
                    <span>by {c.manufacturer}</span>
                    {c.weightG ? <span>• {c.weightG.toFixed(1)}g</span> : null}
                    {c.lensSizeMm ? <span>• Lens: {c.lensSizeMm}mm</span> : null}
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

          {filteredCams.length === 0 && (
            <p className="text-xs text-zinc-400 italic py-2 text-center">
              No cameras found matching your criteria.
            </p>
          )}
        </div>
      </div>

      {/* Navigation Footer */}
      <div className="flex items-center justify-between">
        <button
          type="button"
          onClick={handlePrevStep}
          className="flex items-center gap-1.5 px-4 py-2 rounded-xl text-xs font-semibold text-zinc-600 dark:text-zinc-400 hover:bg-zinc-100 dark:hover:bg-zinc-800 transition-colors"
        >
          <ArrowLeft size={14} />
          <span>Back: Propellers</span>
        </button>

        <button
          type="button"
          disabled={(!selectedCam && !isOmitted) || !!conflictMsg}
          onClick={handleNextStep}
          className="flex items-center gap-2 px-5 py-2.5 rounded-xl font-bold text-xs bg-blue-600 hover:bg-blue-700 disabled:opacity-40 disabled:cursor-not-allowed text-white transition-all shadow-sm"
        >
          <span>{isOmitted ? "Next: GPS (Video Skipped)" : "Next: Video Transmitter (VTX)"}</span>
          <ArrowRight size={14} />
        </button>
      </div>
    </div>
  );
}
