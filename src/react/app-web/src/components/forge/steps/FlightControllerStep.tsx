import { useMemo, useEffect } from "react";
import {
  Search,
  X,
  ArrowRight,
  ArrowLeft,
  AlertCircle,
  AlertTriangle,
  Cpu,
  Radio,
  Zap,
  Video,
  Check,
} from "lucide-react";
import { useForge, isMountPatternCompatible } from "../ForgeContext";
import { ProductHoverCard } from "../../ProductHoverCard";
import { CollectionIcon } from "@quadsmith/shared";
import { CompatibilityHoverCard } from "../../CompatibilityHoverCard";
import { getCollectionColor } from "@quadsmith/shared";
import type { FlightController } from "@quadsmith/shared/gen/quadsmith/api/flight_controller_pb";

export function FlightControllerStep() {
  const {
    fcs,
    selectedFc,
    setSelectedFc,
    selectedFrame,
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
    fcHasIntegratedEsc,
    fcHasIntegratedRx,
    fcHasIntegratedVtx,
  } = useForge();

  useEffect(() => {
    const timer = setTimeout(() => {
      document
        .getElementById("selected-fc-item")
        ?.scrollIntoView({ behavior: "smooth", block: "center" });
    }, 50);
    return () => clearTimeout(timer);
  }, []);

  const search = searchQueries.flight_controllers || "";
  const conflictMsg = stepConflicts[2];
  const batteryConflict = stepBatteryConflicts[2];
  const isCompatibleOnly = compatibleOnlyMap.flight_controllers ?? true;

  const checkFcCompatibility = (fc: FlightController) => {
    if (selectedFrame?.boardMountPatterns?.length && fc.mountPattern) {
      if (!isMountPatternCompatible(fc.mountPattern, selectedFrame.boardMountPatterns)) {
        return {
          compatible: false,
          reason: `Mounting pattern (${fc.mountPattern}) is not supported by frame board mounts (${selectedFrame.boardMountPatterns.join(", ")}).`,
        };
      }
    }
    if (activeBattery) {
      if (
        activeBattery.minVoltage > 0 &&
        fc.maxVoltage > 0 &&
        activeBattery.minVoltage > fc.maxVoltage
      ) {
        return {
          compatible: false,
          reason: `Battery minimum voltage (${activeBattery.minVoltage.toFixed(1)}V) exceeds FC maximum voltage (${fc.maxVoltage.toFixed(1)}V).`,
        };
      }
      if (
        activeBattery.maxVoltage > 0 &&
        fc.minVoltage > 0 &&
        activeBattery.maxVoltage < fc.minVoltage
      ) {
        return {
          compatible: false,
          reason: `Battery maximum voltage (${activeBattery.maxVoltage.toFixed(1)}V) is below FC minimum voltage (${fc.minVoltage.toFixed(1)}V).`,
        };
      }
    }
    return { compatible: true };
  };

  const filteredFcs = useMemo(() => {
    let list = fcs;

    if (isCompatibleOnly) {
      list = list.filter(
        (fc: FlightController) =>
          fc.uuid === selectedFc?.uuid ||
          fc.id === selectedFc?.id ||
          checkFcCompatibility(fc).compatible,
      );
    }

    if (!search.trim()) return list;
    const q = search.toLowerCase();
    return list.filter(
      (fc: FlightController) =>
        fc.name.toLowerCase().includes(q) ||
        fc.manufacturer.toLowerCase().includes(q) ||
        fc.processor?.toLowerCase().includes(q),
    );
  }, [fcs, search, isCompatibleOnly, selectedFrame, activeBattery, selectedFc]);

  return (
    <div className="space-y-4">
      {/* Compatibility Conflict Banner */}
      {conflictMsg && (
        <div className="p-3.5 rounded-xl border border-amber-500/30 bg-amber-500/10 text-amber-800 dark:text-amber-300 flex items-start gap-3 text-xs">
          <AlertCircle size={16} className="text-amber-500 mt-0.5 shrink-0" />
          <div>
            <span className="font-bold">Incompatibility Detected with Frame</span>
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
              flight controller.
            </p>
          </div>
        </div>
      )}

      {/* Main FC Selection Container */}
      <div
        id="section-fc"
        className="p-4 rounded-2xl border border-zinc-200 dark:border-zinc-800 bg-white dark:bg-zinc-900 space-y-3"
      >
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-2">
            <CollectionIcon
              collectionId="flight_controllers"
              size={16}
              color={getCollectionColor("flight_controllers")}
            />
            <h3 className="font-bold text-sm text-zinc-900 dark:text-zinc-100">
              2. Flight Controller
            </h3>
            <span className="text-[10px] uppercase font-bold tracking-wider px-2 py-0.5 rounded bg-zinc-100 dark:bg-zinc-800 text-zinc-600 dark:text-zinc-400">
              Required
            </span>
          </div>
          {selectedFc ? (
            <button
              type="button"
              onClick={() => {
                document
                  .getElementById("selected-fc-item")
                  ?.scrollIntoView({ behavior: "smooth", block: "center" });
              }}
              title="Scroll to selected flight controller"
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
          The flight controller coordinates sensor orientation, motor telemetry, and radio
          communication. AIO boards may integrate ESCs, receivers, or video transmitters.
        </p>

        {/* Selected FC Integrated Capabilities Spotlight */}
        {selectedFc && (fcHasIntegratedEsc || fcHasIntegratedRx || fcHasIntegratedVtx) && (
          <div className="p-3 rounded-xl border border-purple-500/20 bg-purple-50/50 dark:bg-purple-950/20 text-xs space-y-1.5">
            <div className="flex items-center gap-1.5 text-purple-700 dark:text-purple-300 font-bold">
              <Cpu size={14} />
              <span>All-In-One (AIO) Capabilities Detected</span>
            </div>
            <div className="flex flex-wrap gap-2 text-[11px] text-zinc-600 dark:text-zinc-400">
              {fcHasIntegratedEsc && (
                <span className="flex items-center gap-1 px-2 py-0.5 rounded-full bg-emerald-500/10 text-emerald-600 dark:text-emerald-400 border border-emerald-500/20">
                  <Zap size={11} /> Integrated ESC
                </span>
              )}
              {fcHasIntegratedRx && (
                <span className="flex items-center gap-1 px-2 py-0.5 rounded-full bg-blue-500/10 text-blue-600 dark:text-blue-400 border border-blue-500/20">
                  <Radio size={11} /> Integrated RX
                </span>
              )}
              {fcHasIntegratedVtx && (
                <span className="flex items-center gap-1 px-2 py-0.5 rounded-full bg-orange-500/10 text-orange-600 dark:text-orange-400 border border-orange-500/20">
                  <Video size={11} /> Integrated VTX
                </span>
              )}
            </div>
          </div>
        )}

        {/* Search & Compatibility Toggle */}
        <div className="flex items-center gap-2">
          <div className="relative flex-1">
            <Search size={14} className="absolute left-3 top-1/2 -translate-y-1/2 text-zinc-400" />
            <input
              type="text"
              value={search}
              onChange={(e) => setSearchQuery("flight_controllers", e.target.value)}
              placeholder="Search flight controllers by name, MCU, brand..."
              className="w-full bg-zinc-50 dark:bg-zinc-950 border border-zinc-200 dark:border-zinc-800 rounded-xl pl-9 pr-8 py-1.5 text-xs text-zinc-900 dark:text-zinc-100 placeholder-zinc-400 focus:outline-none focus:ring-1 focus:ring-blue-500"
            />
            {search && (
              <button
                type="button"
                onClick={() => setSearchQuery("flight_controllers", "")}
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
              onChange={() => toggleCompatibleOnly("flight_controllers")}
              className="rounded border-zinc-300 dark:border-zinc-700 text-blue-600 focus:ring-blue-500 h-3.5 w-3.5 cursor-pointer"
            />
            <CompatibilityHoverCard filter={getCelFilterForStep(2)}>
              <span>Compatible only</span>
            </CompatibilityHoverCard>
          </label>
        </div>

        {/* FC Catalog List */}
        <div className="overflow-y-auto max-h-80 space-y-2 pr-1.5 focus:outline-none">
          {filteredFcs.map((fc: FlightController) => {
            const isSelected = selectedFc?.uuid === fc.uuid || selectedFc?.id === fc.id;
            const compat = checkFcCompatibility(fc);
            const isIncompatible = !compat.compatible;

            return (
              <ProductHoverCard
                key={fc.uuid || fc.id}
                collectionId="flight_controllers"
                item={fc}
                productId={fc.id || fc.uuid}
                as="div"
                placement="side"
                id={isSelected ? "selected-fc-item" : undefined}
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
                  onClick={() => setSelectedFc(fc)}
                  className="flex-1 min-w-0"
                >
                  <div className="flex items-center gap-2 flex-wrap">
                    <span className="text-xs font-bold text-zinc-900 dark:text-zinc-100 truncate">
                      {fc.name}
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
                    {fc.processor && (
                      <span className="text-[10px] px-2 py-0.5 rounded-full font-semibold bg-zinc-500/10 text-zinc-600 dark:text-zinc-400 border border-zinc-500/20 shrink-0">
                        {fc.processor}
                      </span>
                    )}
                    {fc.mountPattern ? (
                      <span className="text-[10px] px-2 py-0.5 rounded-full font-semibold bg-blue-500/10 text-blue-600 dark:text-blue-400 border border-blue-500/20 shrink-0">
                        {fc.mountPattern}
                      </span>
                    ) : null}
                  </div>
                  <div className="text-[11px] text-zinc-500 dark:text-zinc-400 mt-0.5 flex items-center gap-2 flex-wrap">
                    <span>by {fc.manufacturer}</span>
                    {fc.minVoltage && fc.maxVoltage ? (
                      <span>
                        • {fc.minVoltage}V - {fc.maxVoltage}V
                      </span>
                    ) : null}
                    {fc.weightG ? <span>• {fc.weightG.toFixed(1)}g</span> : null}
                    {fc.internalElectronicSpeedControllerUuid && (
                      <span className="text-emerald-500">• Built-in ESC</span>
                    )}
                    {fc.internalReceiverUuid && (
                      <span className="text-blue-500">• Built-in RX</span>
                    )}
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

          {filteredFcs.length === 0 && (
            <p className="text-xs text-zinc-400 italic py-2 text-center">
              No flight controllers found matching your criteria.
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
          <span>Back: Frame</span>
        </button>

        <button
          type="button"
          disabled={!selectedFc || !!conflictMsg}
          onClick={handleNextStep}
          className="flex items-center gap-2 px-5 py-2.5 rounded-xl font-bold text-xs bg-blue-600 hover:bg-blue-700 disabled:opacity-40 disabled:cursor-not-allowed text-white transition-all shadow-sm"
        >
          <span>Next: Radio Receiver</span>
          <ArrowRight size={14} />
        </button>
      </div>
    </div>
  );
}
