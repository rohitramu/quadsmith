import { useMemo, useEffect } from "react";
import { Search, X, ArrowRight, ArrowLeft, AlertCircle, AlertTriangle, Check } from "lucide-react";
import { useForge, isMountPatternCompatible } from "../ForgeContext";
import { ProductHoverCard } from "../../ProductHoverCard";
import { CollectionIcon } from "../../CollectionIcon";
import { CompatibilityHoverCard } from "../../CompatibilityHoverCard";
import { getCollectionColor } from "../../../lib/collectionColors";
import { formatStatorSize } from "../../../lib/format";
import type { Motor } from "../../../gen/quadsmith/api/motor_pb";

export function MotorStep() {
  const {
    motors,
    selectedMotor,
    setSelectedMotor,
    selectedFrame,
    selectedFc,
    selectedEsc,
    useIntegratedEsc,
    integratedEsc,
    activeBattery,
    handleNextStep,
    handlePrevStep,
    searchQueries,
    setSearchQuery,
    compatibleOnlyMap,
    toggleCompatibleOnly,
    getCelFilterForStep,
    stepConflicts,
    stepBatteryConflicts,
  } = useForge();

  useEffect(() => {
    const timer = setTimeout(() => {
      document
        .getElementById("selected-motor-item")
        ?.scrollIntoView({ behavior: "smooth", block: "center" });
    }, 50);
    return () => clearTimeout(timer);
  }, []);

  const search = searchQueries.motors || "";
  const conflictMsg = stepConflicts[6];
  const batteryConflict = stepBatteryConflicts[6];
  const isCompatibleOnly = compatibleOnlyMap.motors ?? true;
  const activeEsc = useIntegratedEsc ? integratedEsc : selectedEsc;

  const checkMotorCompatibility = (m: Motor) => {
    // Motor mount pattern match against frame
    if (selectedFrame?.motorMountPatterns?.length && m.mountPattern) {
      if (!isMountPatternCompatible(m.mountPattern, selectedFrame.motorMountPatterns)) {
        return {
          compatible: false,
          reason: `Mounting pattern (${m.mountPattern}) is not supported by frame motor mounts (${selectedFrame.motorMountPatterns.join(", ")}).`,
        };
      }
    }

    // ESC Current rating
    if (activeEsc?.motorCurrentMaxA) {
      const escLimit = (activeEsc.motorCurrentBurstA || activeEsc.motorCurrentMaxA) * 1.15;
      if (m.maxCurrentA && m.maxCurrentA > escLimit) {
        return {
          compatible: false,
          reason: `Motor max current (${m.maxCurrentA}A) exceeds ESC rated current limit (${escLimit.toFixed(0)}A).`,
        };
      }
    }

    // ESC Voltage limits
    if (activeEsc) {
      if (activeEsc.maxVoltage > 0 && m.minVoltage > 0 && m.minVoltage > activeEsc.maxVoltage) {
        return {
          compatible: false,
          reason: `Motor min voltage (${m.minVoltage.toFixed(1)}V) exceeds ESC max voltage (${activeEsc.maxVoltage.toFixed(1)}V).`,
        };
      }
      if (activeEsc.minVoltage > 0 && m.maxVoltage > 0 && m.maxVoltage < activeEsc.minVoltage) {
        return {
          compatible: false,
          reason: `Motor max voltage (${m.maxVoltage.toFixed(1)}V) is below ESC min voltage (${activeEsc.minVoltage.toFixed(1)}V).`,
        };
      }
    }

    // FC Voltage limits
    if (selectedFc) {
      if (selectedFc.minVoltage > 0 && m.maxVoltage > 0 && m.maxVoltage < selectedFc.minVoltage) {
        return {
          compatible: false,
          reason: `Motor max voltage (${m.maxVoltage.toFixed(1)}V) is below FC min voltage (${selectedFc.minVoltage.toFixed(1)}V).`,
        };
      }
      if (selectedFc.maxVoltage > 0 && m.minVoltage > 0 && m.minVoltage > selectedFc.maxVoltage) {
        return {
          compatible: false,
          reason: `Motor min voltage (${m.minVoltage.toFixed(1)}V) exceeds FC max voltage (${selectedFc.maxVoltage.toFixed(1)}V).`,
        };
      }
    }

    // Battery Voltage limits
    if (activeBattery) {
      if (
        activeBattery.minVoltage > 0 &&
        m.maxVoltage > 0 &&
        m.maxVoltage < activeBattery.minVoltage
      ) {
        return {
          compatible: false,
          reason: `Motor max voltage (${m.maxVoltage.toFixed(1)}V) is below battery min voltage (${activeBattery.minVoltage.toFixed(1)}V).`,
        };
      }
      if (
        activeBattery.maxVoltage > 0 &&
        m.minVoltage > 0 &&
        m.minVoltage > activeBattery.maxVoltage
      ) {
        return {
          compatible: false,
          reason: `Motor min voltage (${m.minVoltage.toFixed(1)}V) exceeds battery max voltage (${activeBattery.maxVoltage.toFixed(1)}V).`,
        };
      }
    }

    return { compatible: true };
  };

  const filteredMotors = useMemo(() => {
    let list = motors;

    if (isCompatibleOnly) {
      list = list.filter(
        (m: Motor) =>
          m.uuid === selectedMotor?.uuid ||
          m.id === selectedMotor?.id ||
          checkMotorCompatibility(m).compatible,
      );
    }

    if (!search.trim()) return list;
    const q = search.toLowerCase();
    return list.filter(
      (m: Motor) =>
        m.name.toLowerCase().includes(q) ||
        m.manufacturer.toLowerCase().includes(q) ||
        `${m.statorDiameterMm || ""}${m.statorHeightMm || ""}`.includes(q) ||
        `${m.kv}`.includes(q),
    );
  }, [
    motors,
    search,
    isCompatibleOnly,
    selectedFrame,
    activeEsc,
    selectedFc,
    activeBattery,
    selectedMotor,
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

      {/* Battery Voltage Conflict Banner */}
      {batteryConflict && (
        <div className="p-3.5 rounded-xl border border-amber-400/40 bg-amber-400/10 text-amber-800 dark:text-amber-300 flex items-start gap-3 text-xs">
          <AlertTriangle size={16} className="text-amber-500 mt-0.5 shrink-0" />
          <div>
            <span className="font-bold">Battery Voltage Mismatch</span>
            <p className="opacity-90 mt-0.5">
              The selected battery voltage is outside the operating voltage range supported by this
              motor.
            </p>
          </div>
        </div>
      )}

      {/* Main Container */}
      <div
        id="section-motors"
        className="p-4 rounded-2xl border border-zinc-200 dark:border-zinc-800 bg-white dark:bg-zinc-900 space-y-3"
      >
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-2">
            <CollectionIcon collectionId="motors" size={16} color={getCollectionColor("motors")} />
            <h3 className="font-bold text-sm text-zinc-900 dark:text-zinc-100">6. Motors</h3>
            <span className="text-[10px] uppercase font-bold tracking-wider px-2 py-0.5 rounded bg-zinc-100 dark:bg-zinc-800 text-zinc-600 dark:text-zinc-400">
              Required
            </span>
          </div>
          {selectedMotor ? (
            <button
              type="button"
              onClick={() => {
                document
                  .getElementById("selected-motor-item")
                  ?.scrollIntoView({ behavior: "smooth", block: "center" });
              }}
              title="Scroll to selected motors"
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
          Motors provide rotational torque for the propellers. Stator size and KV rating dictate
          optimal prop size and battery cell count.
        </p>

        {/* Search & Compatibility Toggle */}
        <div className="flex items-center gap-2">
          <div className="relative flex-1">
            <Search size={14} className="absolute left-3 top-1/2 -translate-y-1/2 text-zinc-400" />
            <input
              type="text"
              value={search}
              onChange={(e) => setSearchQuery("motors", e.target.value)}
              placeholder="Search motors by name, brand, stator, KV..."
              className="w-full bg-zinc-50 dark:bg-zinc-950 border border-zinc-200 dark:border-zinc-800 rounded-xl pl-9 pr-8 py-1.5 text-xs text-zinc-900 dark:text-zinc-100 placeholder-zinc-400 focus:outline-none focus:ring-1 focus:ring-blue-500"
            />
            {search && (
              <button
                type="button"
                onClick={() => setSearchQuery("motors", "")}
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
              onChange={() => toggleCompatibleOnly("motors")}
              className="rounded border-zinc-300 dark:border-zinc-700 text-blue-600 focus:ring-blue-500 h-3.5 w-3.5 cursor-pointer"
            />
            <CompatibilityHoverCard filter={getCelFilterForStep(6)}>
              <span>Compatible only</span>
            </CompatibilityHoverCard>
          </label>
        </div>

        {/* Motor Catalog List */}
        <div className="overflow-y-auto max-h-80 space-y-2 pr-1.5 focus:outline-none">
          {filteredMotors.map((m: Motor) => {
            const isSelected = selectedMotor?.uuid === m.uuid || selectedMotor?.id === m.id;
            const stator = formatStatorSize(m.statorDiameterMm, m.statorHeightMm);
            const compat = checkMotorCompatibility(m);
            const isIncompatible = !compat.compatible;

            return (
              <ProductHoverCard
                key={m.uuid || m.id}
                collectionId="motors"
                item={m}
                productId={m.id || m.uuid}
                as="div"
                placement="side"
                id={isSelected ? "selected-motor-item" : undefined}
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
                  onClick={() => setSelectedMotor(m)}
                  className="flex-1 min-w-0"
                >
                  <div className="flex items-center gap-2 flex-wrap">
                    <span className="text-xs font-bold text-zinc-900 dark:text-zinc-100 truncate">
                      {m.name}
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
                    {stator && (
                      <span className="text-[10px] px-2 py-0.5 rounded-full font-semibold bg-zinc-500/10 text-zinc-600 dark:text-zinc-400 border border-zinc-500/20 shrink-0 font-mono">
                        {stator}
                      </span>
                    )}
                    {m.kv && (
                      <span className="text-[10px] px-2 py-0.5 rounded-full font-semibold bg-blue-500/10 text-blue-600 dark:text-blue-400 border border-blue-500/20 shrink-0">
                        {m.kv} KV
                      </span>
                    )}
                    {m.mountPattern ? (
                      <span className="text-[10px] px-2 py-0.5 rounded-full font-semibold bg-purple-500/10 text-purple-600 dark:text-purple-400 border border-purple-500/20 shrink-0">
                        {m.mountPattern}
                      </span>
                    ) : null}
                  </div>
                  <div className="text-[11px] text-zinc-500 dark:text-zinc-400 mt-0.5 flex items-center gap-2 flex-wrap">
                    <span>by {m.manufacturer}</span>
                    {m.weightG ? <span>• {m.weightG.toFixed(1)}g ea</span> : null}
                    {m.maxCurrentA ? <span>• Max Current: {m.maxCurrentA}A</span> : null}
                    {m.propMountInterfaces?.length ? (
                      <span>• Prop Mount: {m.propMountInterfaces.join(", ")}</span>
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

          {filteredMotors.length === 0 && (
            <p className="text-xs text-zinc-400 italic py-2 text-center">
              No motors found matching your criteria.
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
          <span>Back: Speed Controller</span>
        </button>

        <button
          type="button"
          disabled={!selectedMotor || !!conflictMsg}
          onClick={handleNextStep}
          className="flex items-center gap-2 px-5 py-2.5 rounded-xl font-bold text-xs bg-blue-600 hover:bg-blue-700 disabled:opacity-40 disabled:cursor-not-allowed text-white transition-all shadow-sm"
        >
          <span>Next: Propellers</span>
          <ArrowRight size={14} />
        </button>
      </div>
    </div>
  );
}
