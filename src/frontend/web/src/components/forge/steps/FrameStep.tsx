import { useMemo } from "react";
import { Search, X, ArrowRight, ArrowLeft, AlertCircle, Check } from "lucide-react";
import { useForge } from "../ForgeContext";
import { ProductHoverCard } from "../../ProductHoverCard";
import { CollectionIcon } from "../../CollectionIcon";
import { CompatibilityHoverCard } from "../../CompatibilityHoverCard";
import { getCollectionColor } from "../../../lib/collectionColors";
import type { Frame } from "../../../gen/quadsmith/frame_pb";

export function FrameStep() {
  const {
    frames,
    selectedFrame,
    setSelectedFrame,
    handleNextStep,
    handlePrevStep,
    searchQueries,
    setSearchQuery,
    stepConflicts,
    compatibleOnlyMap,
    toggleCompatibleOnly,
    getCelFilterForStep,
  } = useForge();

  const search = searchQueries.frames || "";
  const conflictMsg = stepConflicts[1];
  const isCompatibleOnly = compatibleOnlyMap.frames ?? true;

  const filteredFrames = useMemo(() => {
    let list = frames;
    if (!search.trim()) return list;
    const q = search.toLowerCase();
    return list.filter(
      (f) =>
        f.name.toLowerCase().includes(q) ||
        f.manufacturer.toLowerCase().includes(q) ||
        f.geometry.toLowerCase().includes(q),
    );
  }, [frames, search]);

  return (
    <div className="space-y-4">
      {/* Conflict Banner */}
      {conflictMsg && (
        <div className="p-3.5 rounded-xl border border-amber-500/30 bg-amber-500/10 text-amber-800 dark:text-amber-300 flex items-start gap-3 text-xs">
          <AlertCircle size={16} className="text-amber-500 mt-0.5 shrink-0" />
          <div>
            <span className="font-bold">Incompatibility Detected</span>
            <p className="opacity-90 mt-0.5">{conflictMsg}</p>
          </div>
        </div>
      )}

      {/* Frame Selection Container */}
      <div
        id="section-frame"
        className="p-4 rounded-2xl border border-zinc-200 dark:border-zinc-800 bg-white dark:bg-zinc-900 space-y-3"
      >
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-2">
            <CollectionIcon collectionId="frames" size={16} color={getCollectionColor("frames")} />
            <h3 className="font-bold text-sm text-zinc-900 dark:text-zinc-100">1. Frame Chassis</h3>
            <span className="text-[10px] uppercase font-bold tracking-wider px-2 py-0.5 rounded bg-zinc-100 dark:bg-zinc-800 text-zinc-600 dark:text-zinc-400">
              Required
            </span>
          </div>
          {selectedFrame ? (
            <button
              type="button"
              onClick={() => {
                document
                  .getElementById("selected-frame-item")
                  ?.scrollIntoView({ behavior: "smooth", block: "center" });
              }}
              title="Scroll to selected frame"
              className="text-xs font-semibold text-emerald-600 dark:text-emerald-400 hover:text-emerald-700 dark:hover:text-emerald-300 hover:underline flex items-center gap-1 px-1.5 py-0.5 rounded bg-emerald-50 dark:bg-emerald-950/40 border border-emerald-200 dark:border-emerald-800/60 transition-all cursor-pointer shadow-2xs"
            >
              <Check size={12} strokeWidth={2.5} /> Selected
            </button>
          ) : (
            <span className="text-xs font-semibold text-blue-600 dark:text-blue-400 truncate max-w-[200px]">
              None selected
            </span>
          )}
        </div>

        <p className="text-xs text-zinc-500 dark:text-zinc-400">
          The frame establishes the quadcopter's physical geometry, wheelbase, max propeller size,
          camera width, and mounting bolt patterns.
        </p>

        {/* Search & Compatible Only Toggle */}
        <div className="flex items-center gap-2">
          <div className="relative flex-1">
            <Search size={14} className="absolute left-3 top-1/2 -translate-y-1/2 text-zinc-400" />
            <input
              type="text"
              value={search}
              onChange={(e) => setSearchQuery("frames", e.target.value)}
              placeholder="Search frames by name, brand, geometry..."
              className="w-full bg-zinc-50 dark:bg-zinc-950 border border-zinc-200 dark:border-zinc-800 rounded-xl pl-9 pr-8 py-1.5 text-xs text-zinc-900 dark:text-zinc-100 placeholder-zinc-400 focus:outline-none focus:ring-1 focus:ring-blue-500"
            />
            {search && (
              <button
                type="button"
                onClick={() => setSearchQuery("frames", "")}
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
              onChange={() => toggleCompatibleOnly("frames")}
              className="rounded border-zinc-300 dark:border-zinc-700 text-blue-600 focus:ring-blue-500 h-3.5 w-3.5 cursor-pointer"
            />
            <CompatibilityHoverCard filter={getCelFilterForStep(1)}>
              <span>Compatible only</span>
            </CompatibilityHoverCard>
          </label>
        </div>

        {/* List of Frames */}
        <div className="overflow-y-auto max-h-80 space-y-2 pr-1.5 focus:outline-none">
          {filteredFrames.map((f: Frame) => {
            const isSelected = selectedFrame?.uuid === f.uuid || selectedFrame?.id === f.id;

            return (
              <ProductHoverCard
                key={f.uuid || f.id}
                collectionId="frames"
                item={f}
                productId={f.id || f.uuid}
                as="div"
                placement="side"
                id={isSelected ? "selected-frame-item" : undefined}
                className={`p-3 rounded-xl border transition-all cursor-pointer flex flex-col sm:flex-row sm:items-center justify-between gap-2 ${
                  isSelected
                    ? "border-blue-500 bg-blue-50/50 dark:bg-blue-950/30 ring-1 ring-blue-500 shadow-xs"
                    : "border-zinc-200 dark:border-zinc-800 bg-zinc-50/50 dark:bg-zinc-900/60 hover:border-zinc-400 dark:hover:border-zinc-700"
                }`}
              >
                <div
                  role="button"
                  tabIndex={0}
                  onClick={() => setSelectedFrame(isSelected ? null : f)}
                  className="flex-1 min-w-0"
                >
                  <div className="flex items-center gap-2 flex-wrap">
                    <span className="text-xs font-bold text-zinc-900 dark:text-zinc-100 truncate">
                      {f.name}
                    </span>
                    {f.geometry && (
                      <span className="text-[10px] px-2 py-0.5 rounded-full font-semibold bg-purple-500/10 text-purple-600 dark:text-purple-400 border border-purple-500/20 shrink-0">
                        {f.geometry}
                      </span>
                    )}
                    {f.wheelbaseMm && (
                      <span className="text-[10px] px-2 py-0.5 rounded-full font-semibold bg-blue-500/10 text-blue-600 dark:text-blue-400 border border-blue-500/20 shrink-0">
                        {f.wheelbaseMm}mm
                      </span>
                    )}
                  </div>
                  <div className="text-[11px] text-zinc-500 dark:text-zinc-400 mt-0.5 flex items-center gap-2 flex-wrap">
                    <span>by {f.manufacturer}</span>
                    {f.weightG ? <span>• {f.weightG.toFixed(1)}g</span> : null}
                    {f.maxPropSizeMm ? <span>• Max Prop: {f.maxPropSizeMm}mm</span> : null}
                    {f.cameraMountInterfaces?.length ? (
                      <span>• Cam: {f.cameraMountInterfaces.join(", ")}</span>
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

          {filteredFrames.length === 0 && (
            <p className="text-xs text-zinc-400 italic py-2 text-center">
              No compatible frames matching your search.
            </p>
          )}
        </div>
      </div>

      {/* Footer Navigation */}
      <div className="flex items-center justify-between">
        <button
          type="button"
          onClick={handlePrevStep}
          className="flex items-center gap-1.5 px-4 py-2 rounded-xl text-xs font-semibold text-zinc-600 dark:text-zinc-400 hover:bg-zinc-100 dark:hover:bg-zinc-800 transition-colors"
        >
          <ArrowLeft size={14} />
          <span>Back</span>
        </button>

        <button
          type="button"
          disabled={!selectedFrame || !!conflictMsg}
          onClick={handleNextStep}
          className="flex items-center gap-2 px-5 py-2.5 rounded-xl font-bold text-xs bg-blue-600 hover:bg-blue-700 disabled:opacity-40 disabled:cursor-not-allowed text-white transition-all shadow-sm"
        >
          <span>Next: Flight Controller</span>
          <ArrowRight size={14} />
        </button>
      </div>
    </div>
  );
}
