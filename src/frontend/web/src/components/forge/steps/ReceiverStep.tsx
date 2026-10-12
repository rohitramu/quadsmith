import { useMemo, useEffect } from "react";
import { Search, X, ArrowRight, ArrowLeft, AlertCircle, Sparkles, Check } from "lucide-react";
import { useForge } from "../ForgeContext";
import { ProductHoverCard } from "../../ProductHoverCard";
import { CollectionIcon } from "../../CollectionIcon";
import { CompatibilityHoverCard } from "../../CompatibilityHoverCard";
import { getCollectionColor } from "../../../lib/collectionColors";
import type { Receiver } from "../../../gen/quadsmith/receiver_pb";

export function ReceiverStep() {
  const {
    rxs,
    selectedRx,
    setSelectedRx,
    selectedFc,
    useIntegratedRx,
    setUseIntegratedRx,
    integratedRx,
    fcHasIntegratedRx,
    isRxAntennaIntegrated,
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
        .getElementById("selected-rx-item")
        ?.scrollIntoView({ behavior: "smooth", block: "center" });
    }, 50);
    return () => clearTimeout(timer);
  }, []);

  const search = searchQueries.receivers || "";
  const conflictMsg = stepConflicts[3];
  const isCompatibleOnly = compatibleOnlyMap.receivers ?? true;

  const filteredRxs = useMemo(() => {
    let list = rxs;
    if (!search.trim()) return list;
    const q = search.toLowerCase();
    return list.filter(
      (r) =>
        r.name.toLowerCase().includes(q) ||
        r.manufacturer.toLowerCase().includes(q) ||
        r.protocol?.toLowerCase().includes(q),
    );
  }, [rxs, search]);

  const isUsingIntegrated = useIntegratedRx && !!integratedRx;

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
        id="section-rx"
        className="p-4 rounded-2xl border border-zinc-200 dark:border-zinc-800 bg-white dark:bg-zinc-900 space-y-3"
      >
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-2">
            <CollectionIcon
              collectionId="receivers"
              size={16}
              color={getCollectionColor("receivers")}
            />
            <h3 className="font-bold text-sm text-zinc-900 dark:text-zinc-100">
              3. Radio Control Receiver
            </h3>
            <span className="text-[10px] uppercase font-bold tracking-wider px-2 py-0.5 rounded bg-zinc-100 dark:bg-zinc-800 text-zinc-600 dark:text-zinc-400">
              Required
            </span>
          </div>
          {selectedRx || useIntegratedRx ? (
            <button
              type="button"
              onClick={() => {
                document
                  .getElementById("selected-rx-item")
                  ?.scrollIntoView({ behavior: "smooth", block: "center" });
              }}
              title="Scroll to selected receiver"
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
          The receiver picks up control stick inputs from your radio transmitter (ExpressLRS,
          Crossfire, etc.).
        </p>

        {/* FC Integrated Option Card */}
        <div
          role="button"
          tabIndex={0}
          id={isUsingIntegrated ? "selected-rx-item" : undefined}
          onClick={() => {
            if (!fcHasIntegratedRx) return;
            setUseIntegratedRx(true);
            setSelectedRx(integratedRx);
          }}
          className={`p-3.5 rounded-xl border transition-all flex items-center justify-between gap-3 ${
            !fcHasIntegratedRx
              ? "opacity-40 cursor-not-allowed border-zinc-200 dark:border-zinc-800 bg-zinc-100/50 dark:bg-zinc-900/40"
              : isUsingIntegrated
                ? "border-blue-500 bg-blue-50/50 dark:bg-blue-950/30 ring-1 ring-blue-500 shadow-xs cursor-pointer"
                : "border-purple-500/30 bg-purple-500/5 hover:border-purple-500/60 cursor-pointer"
          }`}
        >
          <div className="flex items-center gap-3 min-w-0">
            <div className="w-8 h-8 rounded-lg bg-blue-500/10 text-blue-600 dark:text-blue-400 flex items-center justify-center shrink-0">
              <Sparkles size={16} />
            </div>
            <div className="min-w-0">
              <div className="flex items-center gap-2">
                <span className="text-xs font-bold text-zinc-900 dark:text-zinc-100">
                  Use Integrated FC Receiver
                </span>
                <span className="text-[10px] font-semibold px-2 py-0.5 rounded-full bg-blue-500/10 text-blue-600 dark:text-blue-400 border border-blue-500/20">
                  Built-in
                </span>
              </div>
              <p className="text-[11px] text-zinc-500 dark:text-zinc-400 mt-0.5 truncate">
                {integratedRx
                  ? `${integratedRx.name} (${integratedRx.protocol || "ExpressLRS"})`
                  : selectedFc
                    ? `Integrated into ${selectedFc.name}`
                    : "Requires an FC with built-in receiver"}
              </p>
            </div>
          </div>

          {fcHasIntegratedRx && (
            <div className="w-4 h-4 rounded-full border border-blue-500 flex items-center justify-center shrink-0">
              {isUsingIntegrated && <div className="w-2 h-2 rounded-full bg-blue-500" />}
            </div>
          )}
        </div>

        {/* Search & Compatible Only Toggle */}
        <div className="flex items-center gap-2">
          <div className="relative flex-1">
            <Search size={14} className="absolute left-3 top-1/2 -translate-y-1/2 text-zinc-400" />
            <input
              type="text"
              value={search}
              onChange={(e) => setSearchQuery("receivers", e.target.value)}
              placeholder="Search external receivers by name, brand, protocol..."
              className="w-full bg-zinc-50 dark:bg-zinc-950 border border-zinc-200 dark:border-zinc-800 rounded-xl pl-9 pr-8 py-1.5 text-xs text-zinc-900 dark:text-zinc-100 placeholder-zinc-400 focus:outline-none focus:ring-1 focus:ring-blue-500"
            />
            {search && (
              <button
                type="button"
                onClick={() => setSearchQuery("receivers", "")}
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
              onChange={() => toggleCompatibleOnly("receivers")}
              className="rounded border-zinc-300 dark:border-zinc-700 text-blue-600 focus:ring-blue-500 h-3.5 w-3.5 cursor-pointer"
            />
            <CompatibilityHoverCard filter={getCelFilterForStep(3)}>
              <span>Compatible only</span>
            </CompatibilityHoverCard>
          </label>
        </div>
        {/* Receiver List */}
        <div className="overflow-y-auto max-h-80 space-y-2 pr-1.5 focus:outline-none">
          {filteredRxs.map((r: Receiver) => {
            const isSelected =
              !isUsingIntegrated && (selectedRx?.uuid === r.uuid || selectedRx?.id === r.id);

            return (
              <ProductHoverCard
                key={r.uuid || r.id}
                collectionId="receivers"
                item={r}
                productId={r.id || r.uuid}
                as="div"
                placement="side"
                id={isSelected ? "selected-rx-item" : undefined}
                className={`p-3 rounded-xl border transition-all cursor-pointer flex flex-col sm:flex-row sm:items-center justify-between gap-2 ${
                  isSelected
                    ? "border-blue-500 bg-blue-50/50 dark:bg-blue-950/30 ring-1 ring-blue-500 shadow-xs"
                    : "border-zinc-200 dark:border-zinc-800 bg-zinc-50/50 dark:bg-zinc-900/60 hover:border-zinc-400 dark:hover:border-zinc-700"
                }`}
              >
                <div
                  role="button"
                  tabIndex={0}
                  onClick={() => {
                    setUseIntegratedRx(false);
                    setSelectedRx(r);
                  }}
                  className="flex-1 min-w-0"
                >
                  <div className="flex items-center gap-2 flex-wrap">
                    <span className="text-xs font-bold text-zinc-900 dark:text-zinc-100 truncate">
                      {r.name}
                    </span>
                    {r.protocol && (
                      <span className="text-[10px] px-2 py-0.5 rounded-full font-semibold bg-purple-500/10 text-purple-600 dark:text-purple-400 border border-purple-500/20 shrink-0">
                        {r.protocol}
                      </span>
                    )}
                    {r.frequencyBandMhz && (
                      <span className="text-[10px] px-2 py-0.5 rounded-full font-semibold bg-blue-500/10 text-blue-600 dark:text-blue-400 border border-blue-500/20 shrink-0">
                        {r.frequencyBandMhz >= 1000
                          ? `${(r.frequencyBandMhz / 1000).toFixed(1)} GHz`
                          : `${r.frequencyBandMhz} MHz`}
                      </span>
                    )}
                  </div>
                  <div className="text-[11px] text-zinc-500 dark:text-zinc-400 mt-0.5 flex items-center gap-2 flex-wrap">
                    <span>by {r.manufacturer}</span>
                    {r.weightG ? <span>• {r.weightG.toFixed(1)}g</span> : null}
                    {r.hasTelemetry ? <span>• Telemetry</span> : null}
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

          {filteredRxs.length === 0 && (
            <p className="text-xs text-zinc-400 italic py-2 text-center">
              No receivers found matching your criteria.
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
          <span>Back: Flight Controller</span>
        </button>

        <button
          type="button"
          disabled={(!selectedRx && !isUsingIntegrated) || !!conflictMsg}
          onClick={handleNextStep}
          className="flex items-center gap-2 px-5 py-2.5 rounded-xl font-bold text-xs bg-blue-600 hover:bg-blue-700 disabled:opacity-40 disabled:cursor-not-allowed text-white transition-all shadow-sm"
        >
          <span>
            {isRxAntennaIntegrated ? "Next: ESC (Antenna Integrated)" : "Next: RX Antennas"}
          </span>
          <ArrowRight size={14} />
        </button>
      </div>
    </div>
  );
}
