import { useMemo, useEffect } from "react";
import {
  Search,
  X,
  ArrowRight,
  ArrowLeft,
  AlertCircle,
  Radio,
  Sparkles,
  Check,
} from "lucide-react";
import { useForge, isRxAntenna, checkRxAntennaCompatibility } from "../ForgeContext";
import { ProductHoverCard } from "../../ProductHoverCard";
import { CollectionIcon } from "@quadsmith/shared";
import { CompatibilityHoverCard } from "../../CompatibilityHoverCard";
import { getCollectionColor } from "@quadsmith/shared";
import type { Antenna } from "@quadsmith/shared/gen/quadsmith/api/antenna_pb";

export function ReceiverAntennaStep() {
  const {
    antennas,
    selectedRx,
    useIntegratedRx,
    integratedRx,
    selectedRxAnt,
    setSelectedRxAnt,
    isRxAntennaIntegrated,
    rxAntCount,
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
        .getElementById("selected-rx-ant-item")
        ?.scrollIntoView({ behavior: "smooth", block: "center" });
    }, 50);
    return () => clearTimeout(timer);
  }, []);

  const search = searchQueries.rx_antennas || "";
  const conflictMsg = stepConflicts[4];
  const activeRx = useIntegratedRx ? integratedRx : selectedRx;
  const isCompatibleOnly = compatibleOnlyMap.rx_antennas ?? true;

  const filteredAntennas = useMemo(() => {
    let list = antennas.filter((a) => !a.isInternalOnly);

    if (isCompatibleOnly) {
      list = list.filter(
        (a) =>
          a.uuid === selectedRxAnt?.uuid ||
          a.id === selectedRxAnt?.id ||
          (isRxAntenna(a) && checkRxAntennaCompatibility(activeRx, a).compatible),
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
  }, [antennas, activeRx, search, isCompatibleOnly, selectedRxAnt]);

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
        id="section-rx-ant"
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
              4. Radio Receiver Antenna(s)
            </h3>
            <span className="text-[10px] uppercase font-bold tracking-wider px-2 py-0.5 rounded bg-zinc-100 dark:bg-zinc-800 text-zinc-600 dark:text-zinc-400">
              {isRxAntennaIntegrated ? "Integrated" : "Required"}
            </span>
          </div>
          {selectedRxAnt ? (
            <button
              type="button"
              onClick={() => {
                document
                  .getElementById("selected-rx-ant-item")
                  ?.scrollIntoView({ behavior: "smooth", block: "center" });
              }}
              title="Scroll to selected antenna"
              className="text-xs font-semibold text-emerald-600 dark:text-emerald-400 hover:text-emerald-700 dark:hover:text-emerald-300 hover:underline flex items-center gap-1 px-1.5 py-0.5 rounded bg-emerald-50 dark:bg-emerald-950/40 border border-emerald-200 dark:border-emerald-800/60 transition-all cursor-pointer shadow-2xs"
            >
              <Check size={12} strokeWidth={2.5} /> Selected
            </button>
          ) : (
            <span className="text-xs font-semibold text-blue-600 dark:text-blue-400 truncate max-w-[200px]">
              {isRxAntennaIntegrated ? "Integrated SMD Antenna" : "None selected"}
            </span>
          )}
        </div>

        <p className="text-xs text-zinc-500 dark:text-zinc-400">
          The receiver antenna must match the radio frequency band (e.g. 2.4GHz or 915MHz) and
          connector type (U.FL / IPEX).
        </p>

        {/* Integrated Ceramic Antenna Banner */}
        {isRxAntennaIntegrated ? (
          <div className="p-4 rounded-xl border border-emerald-500/30 bg-emerald-50/50 dark:bg-emerald-950/20 text-xs space-y-2">
            <div className="flex items-center gap-2 text-emerald-700 dark:text-emerald-300 font-bold">
              <Sparkles size={16} />
              <span>Integrated SMD / Ceramic Antenna Active</span>
            </div>
            <p className="text-zinc-600 dark:text-zinc-400 text-[11px] leading-relaxed">
              Your selected receiver ({activeRx?.name || "Integrated RX"}) features an onboard
              ceramic/SMD antenna. No external antenna or connector cables are required.
            </p>
          </div>
        ) : (
          <>
            {/* Multi-antenna notice */}
            {rxAntCount > 1 && (
              <div className="p-2.5 rounded-xl bg-blue-500/10 border border-blue-500/20 text-blue-700 dark:text-blue-300 text-xs flex items-center gap-2 font-medium">
                <Radio size={14} />
                <span>Diversity / Gemini Receiver: {rxAntCount} antennas will be equipped.</span>
              </div>
            )}

            {/* Search & Compatible Only Toggle */}
            <div className="flex items-center gap-2">
              <div className="relative flex-1">
                <Search
                  size={14}
                  className="absolute left-3 top-1/2 -translate-y-1/2 text-zinc-400"
                />
                <input
                  type="text"
                  value={search}
                  onChange={(e) => setSearchQuery("rx_antennas", e.target.value)}
                  placeholder="Search antennas by name, connector..."
                  className="w-full bg-zinc-50 dark:bg-zinc-950 border border-zinc-200 dark:border-zinc-800 rounded-xl pl-9 pr-8 py-1.5 text-xs text-zinc-900 dark:text-zinc-100 placeholder-zinc-400 focus:outline-none focus:ring-1 focus:ring-blue-500"
                />
                {search && (
                  <button
                    type="button"
                    onClick={() => setSearchQuery("rx_antennas", "")}
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
                  onChange={() => toggleCompatibleOnly("rx_antennas")}
                  className="rounded border-zinc-300 dark:border-zinc-700 text-blue-600 focus:ring-blue-500 h-3.5 w-3.5 cursor-pointer"
                />
                <CompatibilityHoverCard filter={getCelFilterForStep(4)}>
                  <span>Compatible only</span>
                </CompatibilityHoverCard>
              </label>
            </div>

            {/* Antennas List */}
            <div className="overflow-y-auto max-h-80 space-y-2 pr-1.5 focus:outline-none">
              {filteredAntennas.map((ant: Antenna) => {
                const isSelected = selectedRxAnt?.uuid === ant.uuid || selectedRxAnt?.id === ant.id;
                const compat = checkRxAntennaCompatibility(activeRx, ant);
                const isIncompatible = !compat.compatible;

                return (
                  <ProductHoverCard
                    key={ant.uuid || ant.id}
                    collectionId="antennas"
                    item={ant}
                    productId={ant.id || ant.uuid}
                    as="div"
                    placement="side"
                    id={isSelected ? "selected-rx-ant-item" : undefined}
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
                      onClick={() => setSelectedRxAnt(ant)}
                      className="flex-1 min-w-0"
                    >
                      <div className="flex items-center gap-2 flex-wrap">
                        <span className="text-xs font-bold text-zinc-900 dark:text-zinc-100 truncate">
                          {ant.name}
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
                        {ant.connector && (
                          <span className="text-[10px] px-2 py-0.5 rounded-full font-semibold bg-zinc-500/10 text-zinc-600 dark:text-zinc-400 border border-zinc-500/20 shrink-0">
                            {ant.connector}
                          </span>
                        )}
                        {ant.frequencyBandMhz ? (
                          <span className="text-[10px] px-2 py-0.5 rounded-full font-semibold bg-blue-500/10 text-blue-600 dark:text-blue-400 border border-blue-500/20 shrink-0">
                            {ant.frequencyBandMhz >= 1000
                              ? `${(ant.frequencyBandMhz / 1000).toFixed(1)} GHz`
                              : `${ant.frequencyBandMhz} MHz`}
                          </span>
                        ) : null}
                      </div>
                      <div className="text-[11px] text-zinc-500 dark:text-zinc-400 mt-0.5 flex items-center gap-2 flex-wrap">
                        <span>by {ant.manufacturer}</span>
                        {ant.gainDbi ? <span>• Gain: {ant.gainDbi} dBi</span> : null}
                        {ant.weightG ? <span>• {ant.weightG.toFixed(1)}g</span> : null}
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

              {filteredAntennas.length === 0 && (
                <p className="text-xs text-zinc-400 italic py-2 text-center">
                  No receiver antennas found matching your criteria.
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
          <span>Back: Radio Receiver</span>
        </button>

        <button
          type="button"
          disabled={(!isRxAntennaIntegrated && !selectedRxAnt) || !!conflictMsg}
          onClick={handleNextStep}
          className="flex items-center gap-2 px-5 py-2.5 rounded-xl font-bold text-xs bg-blue-600 hover:bg-blue-700 disabled:opacity-40 disabled:cursor-not-allowed text-white transition-all shadow-sm"
        >
          <span>Next: Speed Controller (ESC)</span>
          <ArrowRight size={14} />
        </button>
      </div>
    </div>
  );
}
