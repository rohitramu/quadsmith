import { useMemo, useEffect } from "react";
import {
  Search,
  X,
  ArrowRight,
  ArrowLeft,
  AlertCircle,
  AlertTriangle,
  Sparkles,
  Check,
} from "lucide-react";
import { useForge } from "../ForgeContext";
import { ProductHoverCard } from "../../ProductHoverCard";
import { CollectionIcon } from "../../CollectionIcon";
import { CompatibilityHoverCard } from "../../CompatibilityHoverCard";
import { getCollectionColor } from "../../../lib/collectionColors";
import type { ElectronicSpeedController } from "../../../gen/quadsmith/electronic_speed_controller_pb";

export function EscStep() {
  const {
    escs,
    selectedEsc,
    setSelectedEsc,
    useIntegratedEsc,
    setUseIntegratedEsc,
    integratedEsc,
    fcHasAdequateIntegratedEsc,
    selectedFrame,
    selectedFc,
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
        .getElementById("selected-esc-item")
        ?.scrollIntoView({ behavior: "smooth", block: "center" });
    }, 50);
    return () => clearTimeout(timer);
  }, []);

  const search = searchQueries.electronic_speed_controllers || "";
  const conflictMsg = stepConflicts[5];
  const batteryConflict = stepBatteryConflicts[5];
  const isCompatibleOnly = compatibleOnlyMap.electronic_speed_controllers ?? true;

  const checkEscCompatibility = (esc: ElectronicSpeedController) => {
    // Channel count vs frame motor count
    if (selectedFrame?.motorCount && esc.maxMotors && esc.maxMotors < selectedFrame.motorCount) {
      return {
        compatible: false,
        reason: `ESC supports only ${esc.maxMotors} channels, but frame requires ${selectedFrame.motorCount} channels.`,
      };
    }

    // Voltage limits with FC
    if (selectedFc) {
      if (
        selectedFc.minVoltage > 0 &&
        esc.maxVoltage > 0 &&
        esc.maxVoltage < selectedFc.minVoltage
      ) {
        return {
          compatible: false,
          reason: `ESC max voltage (${esc.maxVoltage.toFixed(1)}V) is below FC min voltage (${selectedFc.minVoltage.toFixed(1)}V).`,
        };
      }
      if (
        selectedFc.maxVoltage > 0 &&
        esc.minVoltage > 0 &&
        esc.minVoltage > selectedFc.maxVoltage
      ) {
        return {
          compatible: false,
          reason: `ESC min voltage (${esc.minVoltage.toFixed(1)}V) exceeds FC max voltage (${selectedFc.maxVoltage.toFixed(1)}V).`,
        };
      }
    }

    // Battery voltage limits
    if (activeBattery) {
      if (
        activeBattery.minVoltage > 0 &&
        esc.maxVoltage > 0 &&
        esc.maxVoltage < activeBattery.minVoltage
      ) {
        return {
          compatible: false,
          reason: `ESC max voltage (${esc.maxVoltage.toFixed(1)}V) is below battery min voltage (${activeBattery.minVoltage.toFixed(1)}V).`,
        };
      }
      if (
        activeBattery.maxVoltage > 0 &&
        esc.minVoltage > 0 &&
        esc.minVoltage > activeBattery.maxVoltage
      ) {
        return {
          compatible: false,
          reason: `ESC min voltage (${esc.minVoltage.toFixed(1)}V) exceeds battery max voltage (${activeBattery.maxVoltage.toFixed(1)}V).`,
        };
      }
    }

    return { compatible: true };
  };

  const filteredEscs = useMemo(() => {
    let list = escs;

    if (isCompatibleOnly) {
      list = list.filter(
        (e) =>
          e.uuid === selectedEsc?.uuid ||
          e.id === selectedEsc?.id ||
          checkEscCompatibility(e).compatible,
      );
    }

    if (!search.trim()) return list;
    const q = search.toLowerCase();
    return list.filter(
      (e) =>
        e.name.toLowerCase().includes(q) ||
        e.manufacturer.toLowerCase().includes(q) ||
        e.firmware?.toLowerCase().includes(q),
    );
  }, [escs, search, isCompatibleOnly, selectedFrame, selectedFc, activeBattery, selectedEsc]);

  const isUsingIntegrated = useIntegratedEsc && !!integratedEsc;

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
              ESC.
            </p>
          </div>
        </div>
      )}

      {/* Main Container */}
      <div
        id="section-esc"
        className="p-4 rounded-2xl border border-zinc-200 dark:border-zinc-800 bg-white dark:bg-zinc-900 space-y-3"
      >
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-2">
            <CollectionIcon
              collectionId="electronic_speed_controllers"
              size={16}
              color={getCollectionColor("electronic_speed_controllers")}
            />
            <h3 className="font-bold text-sm text-zinc-900 dark:text-zinc-100">
              5. Electronic Speed Controller (ESC)
            </h3>
            <span className="text-[10px] uppercase font-bold tracking-wider px-2 py-0.5 rounded bg-zinc-100 dark:bg-zinc-800 text-zinc-600 dark:text-zinc-400">
              Required
            </span>
          </div>
          {selectedEsc || useIntegratedEsc ? (
            <button
              type="button"
              onClick={() => {
                document
                  .getElementById("selected-esc-item")
                  ?.scrollIntoView({ behavior: "smooth", block: "center" });
              }}
              title="Scroll to selected speed controller"
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
          The ESC regulates power delivered to brushless motors from the flight battery and handles
          commutation timing.
        </p>

        {/* FC Integrated ESC Option Card */}
        <div
          role="button"
          tabIndex={0}
          id={isUsingIntegrated ? "selected-esc-item" : undefined}
          onClick={() => {
            if (!fcHasAdequateIntegratedEsc) return;
            setUseIntegratedEsc(true);
            setSelectedEsc(integratedEsc);
          }}
          className={`p-3.5 rounded-xl border transition-all flex items-center justify-between gap-3 ${
            !fcHasAdequateIntegratedEsc
              ? "opacity-40 cursor-not-allowed border-zinc-200 dark:border-zinc-800 bg-zinc-100/50 dark:bg-zinc-900/40"
              : isUsingIntegrated
                ? "border-blue-500 bg-blue-50/50 dark:bg-blue-950/30 ring-1 ring-blue-500 shadow-xs cursor-pointer"
                : "border-purple-500/30 bg-purple-500/5 hover:border-purple-500/60 cursor-pointer"
          }`}
        >
          <div className="flex items-center gap-3 min-w-0">
            <div className="w-8 h-8 rounded-lg bg-emerald-500/10 text-emerald-600 dark:text-emerald-400 flex items-center justify-center shrink-0">
              <Sparkles size={16} />
            </div>
            <div className="min-w-0">
              <div className="flex items-center gap-2">
                <span className="text-xs font-bold text-zinc-900 dark:text-zinc-100">
                  Use Integrated FC ESC
                </span>
                <span className="text-[10px] font-semibold px-2 py-0.5 rounded-full bg-emerald-500/10 text-emerald-600 dark:text-emerald-400 border border-emerald-500/20">
                  AIO Built-in
                </span>
              </div>
              <p className="text-[11px] text-zinc-500 dark:text-zinc-400 mt-0.5 truncate">
                {integratedEsc
                  ? `${integratedEsc.name} • ${integratedEsc.motorCurrentMaxA}A Continuous`
                  : selectedFc
                    ? `Integrated into ${selectedFc.name}`
                    : "Requires an FC with built-in ESC"}
              </p>
            </div>
          </div>

          {fcHasAdequateIntegratedEsc && (
            <div className="w-4 h-4 rounded-full border border-blue-500 flex items-center justify-center shrink-0">
              {isUsingIntegrated && <div className="w-2 h-2 rounded-full bg-blue-500" />}
            </div>
          )}
        </div>

        {/* Search & Compatibility Toggle */}
        <div className="flex items-center gap-2">
          <div className="relative flex-1">
            <Search size={14} className="absolute left-3 top-1/2 -translate-y-1/2 text-zinc-400" />
            <input
              type="text"
              value={search}
              onChange={(e) => setSearchQuery("electronic_speed_controllers", e.target.value)}
              placeholder="Search external ESCs by name, brand, current rating..."
              className="w-full bg-zinc-50 dark:bg-zinc-950 border border-zinc-200 dark:border-zinc-800 rounded-xl pl-9 pr-8 py-1.5 text-xs text-zinc-900 dark:text-zinc-100 placeholder-zinc-400 focus:outline-none focus:ring-1 focus:ring-blue-500"
            />
            {search && (
              <button
                type="button"
                onClick={() => setSearchQuery("electronic_speed_controllers", "")}
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
              onChange={() => toggleCompatibleOnly("electronic_speed_controllers")}
              className="rounded border-zinc-300 dark:border-zinc-700 text-blue-600 focus:ring-blue-500 h-3.5 w-3.5 cursor-pointer"
            />
            <CompatibilityHoverCard filter={getCelFilterForStep(5)}>
              <span>Compatible only</span>
            </CompatibilityHoverCard>
          </label>
        </div>

        {/* ESC Catalog List */}
        <div className="overflow-y-auto max-h-80 space-y-2 pr-1.5 focus:outline-none">
          {filteredEscs.map((esc: ElectronicSpeedController) => {
            const isSelected =
              !isUsingIntegrated && (selectedEsc?.uuid === esc.uuid || selectedEsc?.id === esc.id);
            const compat = checkEscCompatibility(esc);
            const isIncompatible = !compat.compatible;

            return (
              <ProductHoverCard
                key={esc.uuid || esc.id}
                collectionId="electronic_speed_controllers"
                item={esc}
                productId={esc.id || esc.uuid}
                as="div"
                placement="side"
                id={isSelected ? "selected-esc-item" : undefined}
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
                    setUseIntegratedEsc(false);
                    setSelectedEsc(esc);
                  }}
                  className="flex-1 min-w-0"
                >
                  <div className="flex items-center gap-2 flex-wrap">
                    <span className="text-xs font-bold text-zinc-900 dark:text-zinc-100 truncate">
                      {esc.name}
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
                    {esc.motorCurrentMaxA ? (
                      <span className="text-[10px] px-2 py-0.5 rounded-full font-semibold bg-emerald-500/10 text-emerald-600 dark:text-emerald-400 border border-emerald-500/20 shrink-0">
                        {esc.motorCurrentMaxA}A Cont.
                      </span>
                    ) : null}
                    {esc.maxMotors ? (
                      <span className="text-[10px] px-2 py-0.5 rounded-full font-semibold bg-blue-500/10 text-blue-600 dark:text-blue-400 border border-blue-500/20 shrink-0">
                        {esc.maxMotors} Channel
                      </span>
                    ) : null}
                  </div>
                  <div className="text-[11px] text-zinc-500 dark:text-zinc-400 mt-0.5 flex items-center gap-2 flex-wrap">
                    <span>by {esc.manufacturer}</span>
                    {esc.minVoltage && esc.maxVoltage ? (
                      <span>
                        • {esc.minVoltage}V - {esc.maxVoltage}V
                      </span>
                    ) : null}
                    {esc.firmware ? <span>• {esc.firmware}</span> : null}
                    {esc.weightG ? <span>• {esc.weightG.toFixed(1)}g</span> : null}
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

          {filteredEscs.length === 0 && (
            <p className="text-xs text-zinc-400 italic py-2 text-center">
              No ESCs found matching your criteria.
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
          <span>Back</span>
        </button>

        <button
          type="button"
          disabled={(!selectedEsc && !isUsingIntegrated) || !!conflictMsg}
          onClick={handleNextStep}
          className="flex items-center gap-2 px-5 py-2.5 rounded-xl font-bold text-xs bg-blue-600 hover:bg-blue-700 disabled:opacity-40 disabled:cursor-not-allowed text-white transition-all shadow-sm"
        >
          <span>Next: Motors</span>
          <ArrowRight size={14} />
        </button>
      </div>
    </div>
  );
}
