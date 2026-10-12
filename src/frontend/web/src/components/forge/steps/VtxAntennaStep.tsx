import { useMemo } from "react";
import {
  Search,
  X,
  ArrowRight,
  ArrowLeft,
  AlertCircle,
  Radio,
  Sparkles,
  EyeOff,
  Check,
} from "lucide-react";
import { useForge, checkVtxAntennaCompatibility, formatFrequencyBand } from "../ForgeContext";
import { ProductHoverCard } from "../../ProductHoverCard";
import { CollectionIcon } from "../../CollectionIcon";
import { CompatibilityHoverCard } from "../../CompatibilityHoverCard";
import { getCollectionColor } from "../../../lib/collectionColors";
import type { Antenna } from "../../../gen/quadsmith/antenna_pb";

export function VtxAntennaStep() {
  const {
    antennas,
    selectedVtx,
    useIntegratedVtx,
    integratedVtx,
    selectedVtxAnt,
    setSelectedVtxAnt,
    isVtxAntennaIntegrated,
    vtxAntCount,
    noneSelections,
    setNoneSelections,
    handleNextStep,
    handlePrevStep,
    searchQueries,
    setSearchQuery,
    compatibleOnlyMap,
    toggleCompatibleOnly,
    getCelFilterForStep,
    stepConflicts,
  } = useForge();

  const isCompatibleOnly = compatibleOnlyMap.vtx_antennas ?? true;

  const search = searchQueries.vtx_antennas || "";
  const conflictMsg = stepConflicts[10];
  const activeVtx = useIntegratedVtx ? integratedVtx : selectedVtx;
  const isVtxNone = noneSelections.vtx || !activeVtx;
  const isAntennaNone = noneSelections.vtxAntenna || isVtxNone;

  // Video antennas matching active VTX frequency & connector
  const filteredAntennas = useMemo(() => {
    let list = antennas.filter((a) => !a.isInternalOnly);

    if (isCompatibleOnly && activeVtx) {
      list = list.filter(
        (a) =>
          a.uuid === selectedVtxAnt?.uuid ||
          a.id === selectedVtxAnt?.id ||
          checkVtxAntennaCompatibility(activeVtx, a, antennas).compatible,
      );
    }

    if (!search.trim()) return list;
    const q = search.toLowerCase();
    return list.filter(
      (a) =>
        a.name.toLowerCase().includes(q) ||
        a.manufacturer.toLowerCase().includes(q) ||
        a.connector?.toLowerCase().includes(q),
    );
  }, [antennas, search, isCompatibleOnly, activeVtx, selectedVtxAnt]);

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
        id="section-vtx-ant"
        className="p-4 rounded-2xl border border-zinc-200 dark:border-zinc-800 bg-white dark:bg-zinc-900 space-y-3"
      >
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-2">
            <CollectionIcon
              collectionId="antennas"
              size={16}
              color={getCollectionColor("antennas")}
            />
            <h3 className="font-bold text-sm text-zinc-900 dark:text-zinc-100">
              10. VTX Antenna(s)
            </h3>
            <span className="text-[10px] uppercase font-bold tracking-wider px-2 py-0.5 rounded bg-zinc-100 dark:bg-zinc-800 text-zinc-600 dark:text-zinc-400">
              {isVtxNone ? "Omitted" : isVtxAntennaIntegrated ? "Integrated" : "Required"}
            </span>
          </div>
          {selectedVtxAnt ? (
            <button
              type="button"
              onClick={() => {
                document
                  .getElementById("selected-vtx-ant-item")
                  ?.scrollIntoView({ behavior: "smooth", block: "center" });
              }}
              title="Scroll to selected VTX antenna"
              className="text-xs font-semibold text-emerald-600 dark:text-emerald-400 hover:text-emerald-700 dark:hover:text-emerald-300 hover:underline flex items-center gap-1 px-1.5 py-0.5 rounded bg-emerald-50 dark:bg-emerald-950/40 border border-emerald-200 dark:border-emerald-800/60 transition-all cursor-pointer shadow-2xs"
            >
              <Check size={12} strokeWidth={2.5} /> Selected
            </button>
          ) : (
            <span className="text-xs font-semibold text-blue-600 dark:text-blue-400 truncate max-w-[200px]">
              {isVtxNone
                ? "None (VTX Omitted)"
                : isVtxAntennaIntegrated
                  ? "Integrated Antenna"
                  : "None selected"}
            </span>
          )}
        </div>

        <p className="text-xs text-zinc-500 dark:text-zinc-400">
          The video transmitter antenna radiates the video signal. The antenna must match the VTX
          frequency band and connector type (e.g. SMA, U.FL, Dual U.FL).
        </p>

        {isVtxNone ? (
          <div className="p-4 rounded-xl border border-zinc-200 dark:border-zinc-800 bg-zinc-50 dark:bg-zinc-900/40 text-xs flex items-center gap-3">
            <EyeOff size={16} className="text-zinc-400 shrink-0" />
            <div>
              <span className="font-bold text-zinc-800 dark:text-zinc-200">
                Video Transmitter Omitted
              </span>
              <p className="text-zinc-500 dark:text-zinc-400 text-[11px] mt-0.5">
                No video transmitter was selected in Step 9, so external VTX antennas are omitted.
              </p>
            </div>
          </div>
        ) : isVtxAntennaIntegrated ? (
          <div className="p-4 rounded-xl border border-emerald-500/30 bg-emerald-50/50 dark:bg-emerald-950/20 text-xs space-y-2">
            <div className="flex items-center gap-2 text-emerald-700 dark:text-emerald-300 font-bold">
              <Sparkles size={16} />
              <span>Integrated VTX Antenna Active</span>
            </div>
            <p className="text-zinc-600 dark:text-zinc-400 text-[11px] leading-relaxed">
              Your selected video transmitter ({activeVtx?.name || "Integrated VTX"}) features an
              onboard built-in antenna. No external antenna or connector cables are required.
            </p>
          </div>
        ) : (
          <>
            {/* Multi-antenna notice */}
            {vtxAntCount > 1 && (
              <div className="p-2.5 rounded-xl bg-orange-500/10 border border-orange-500/20 text-orange-700 dark:text-orange-300 text-xs flex items-center gap-2 font-medium">
                <Radio size={14} />
                <span>Dual-Antenna Digital Air Unit: {vtxAntCount} antennas will be equipped.</span>
              </div>
            )}

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
                  onChange={(e) => setSearchQuery("vtx_antennas", e.target.value)}
                  placeholder="Search antennas by name, connector..."
                  className="w-full bg-zinc-50 dark:bg-zinc-950 border border-zinc-200 dark:border-zinc-800 rounded-xl pl-9 pr-8 py-1.5 text-xs text-zinc-900 dark:text-zinc-100 placeholder-zinc-400 focus:outline-none focus:ring-1 focus:ring-blue-500"
                />
                {search && (
                  <button
                    type="button"
                    onClick={() => setSearchQuery("vtx_antennas", "")}
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
                  onChange={() => toggleCompatibleOnly("vtx_antennas")}
                  className="rounded border-zinc-300 dark:border-zinc-700 text-blue-600 focus:ring-blue-500 h-3.5 w-3.5 cursor-pointer"
                />
                <CompatibilityHoverCard filter={getCelFilterForStep(10)}>
                  <span>Compatible only</span>
                </CompatibilityHoverCard>
              </label>
            </div>

            {/* Antennas List */}
            <div className="overflow-y-auto max-h-80 space-y-2 pr-1.5 focus:outline-none">
              {filteredAntennas.map((ant: Antenna) => {
                const isSelected =
                  selectedVtxAnt?.uuid === ant.uuid || selectedVtxAnt?.id === ant.id;
                const compat = checkVtxAntennaCompatibility(activeVtx, ant, antennas);
                const isIncompatible = !compat.compatible;

                return (
                  <ProductHoverCard
                    key={ant.uuid || ant.id}
                    collectionId="antennas"
                    item={ant}
                    productId={ant.id || ant.uuid}
                    as="div"
                    placement="side"
                    id={isSelected ? "selected-vtx-ant-item" : undefined}
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
                        setNoneSelections((prev) => ({ ...prev, vtxAntenna: false }));
                        setSelectedVtxAnt(isSelected ? null : ant);
                      }}
                      className="flex-1 min-w-0"
                    >
                      <div className="flex items-center gap-2 flex-wrap">
                        <span className="text-xs font-bold text-zinc-900 dark:text-zinc-100 truncate">
                          {ant.name}
                        </span>
                        {ant.connector && (
                          <span className="text-[10px] px-2 py-0.5 rounded-full font-semibold bg-zinc-500/10 text-zinc-600 dark:text-zinc-400 border border-zinc-500/20 shrink-0">
                            {ant.connector}
                          </span>
                        )}
                        {ant.polarization && (
                          <span className="text-[10px] px-2 py-0.5 rounded-full font-semibold bg-blue-500/10 text-blue-600 dark:text-blue-400 border border-blue-500/20 shrink-0">
                            {ant.polarization}
                          </span>
                        )}
                        {ant.frequencyBandMhz ? (
                          <span className="text-[10px] px-2 py-0.5 rounded-full font-semibold bg-orange-500/10 text-orange-600 dark:text-orange-400 border border-orange-500/20 shrink-0">
                            {formatFrequencyBand(ant.frequencyBandMhz)}
                          </span>
                        ) : null}
                        {isIncompatible && (
                          <span
                            className="text-[10px] px-2 py-0.5 rounded-full font-semibold bg-rose-500/10 text-rose-600 dark:text-rose-400 border border-rose-500/20 shrink-0 flex items-center gap-1"
                            title={compat.reason}
                          >
                            <AlertCircle size={10} />
                            {compat.reason}
                          </span>
                        )}
                      </div>
                      <div className="text-[11px] text-zinc-500 dark:text-zinc-400 mt-0.5 flex items-center gap-2 flex-wrap">
                        <span>by {ant.manufacturer}</span>
                        {ant.gainDbi ? <span>• Gain: {ant.gainDbi} dBi</span> : null}
                        {ant.weightG ? <span>• {ant.weightG.toFixed(1)}g ea</span> : null}
                      </div>
                    </div>

                    <div className="flex items-center gap-2 self-end sm:self-center shrink-0">
                      <div
                        className={`w-4 h-4 rounded-full border flex items-center justify-center ${
                          isIncompatible && isSelected ? "border-amber-500" : "border-blue-500"
                        }`}
                      >
                        {isSelected && (
                          <div
                            className={`w-2 h-2 rounded-full ${
                              isIncompatible ? "bg-amber-500" : "bg-blue-500"
                            }`}
                          />
                        )}
                      </div>
                    </div>
                  </ProductHoverCard>
                );
              })}

              {filteredAntennas.length === 0 && (
                <p className="text-xs text-zinc-400 italic py-2 text-center">
                  No video antennas found matching your criteria.
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
          <span>Back: Video Transmitter</span>
        </button>

        <button
          type="button"
          disabled={(!isAntennaNone && !isVtxAntennaIntegrated && !selectedVtxAnt) || !!conflictMsg}
          onClick={handleNextStep}
          className="flex items-center gap-2 px-5 py-2.5 rounded-xl font-bold text-xs bg-blue-600 hover:bg-blue-700 disabled:opacity-40 disabled:cursor-not-allowed text-white transition-all shadow-sm"
        >
          <span>Next: GPS Navigation</span>
          <ArrowRight size={14} />
        </button>
      </div>
    </div>
  );
}
