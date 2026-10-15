import { useMemo, useEffect } from "react";
import { Search, X, ArrowRight, ArrowLeft, AlertCircle, Check } from "lucide-react";
import { useForge } from "../ForgeContext";
import { ProductHoverCard } from "../../ProductHoverCard";
import { CollectionIcon } from "@quadsmith/shared";
import { CompatibilityHoverCard } from "../../CompatibilityHoverCard";
import { getCollectionColor } from "@quadsmith/shared";
import type { Propeller } from "@quadsmith/shared/gen/quadsmith/api/propeller_pb";

export function PropellerStep() {
  const {
    props,
    selectedProp,
    setSelectedProp,
    selectedFrame,
    selectedMotor,
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
        .getElementById("selected-prop-item")
        ?.scrollIntoView({ behavior: "smooth", block: "center" });
    }, 50);
    return () => clearTimeout(timer);
  }, []);

  const search = searchQueries.propellers || "";
  const conflictMsg = stepConflicts[7];
  const isCompatibleOnly = compatibleOnlyMap.propellers ?? true;

  const checkPropCompatibility = (p: Propeller) => {
    if (
      selectedFrame &&
      selectedFrame.maxPropSizeMm > 0 &&
      p.diameterMm &&
      p.diameterMm > selectedFrame.maxPropSizeMm
    ) {
      return {
        compatible: false,
        reason: `Exceeds max frame prop size (${selectedFrame.maxPropSizeMm}mm)`,
      };
    }
    if (selectedMotor?.propMountInterfaces?.length && p.propMountInterface) {
      const pIface = p.propMountInterface.trim().toLowerCase();
      const matched = selectedMotor.propMountInterfaces.some(
        (mIface) => mIface.trim().toLowerCase() === pIface,
      );
      if (!matched) {
        return {
          compatible: false,
          reason: `Prop mount (${p.propMountInterface}) incompatible with motor (${selectedMotor.propMountInterfaces.join(", ")})`,
        };
      }
    }
    return { compatible: true };
  };

  const filteredProps = useMemo(() => {
    let list = props;

    if (isCompatibleOnly) {
      list = list.filter(
        (p: Propeller) =>
          p.uuid === selectedProp?.uuid ||
          p.id === selectedProp?.id ||
          checkPropCompatibility(p).compatible,
      );
    }

    if (!search.trim()) return list;
    const q = search.toLowerCase();
    return list.filter(
      (p: Propeller) =>
        p.name.toLowerCase().includes(q) ||
        p.manufacturer.toLowerCase().includes(q) ||
        p.propMountInterface?.toLowerCase().includes(q) ||
        `${p.diameterMm}`.includes(q),
    );
  }, [props, search, isCompatibleOnly, selectedFrame, selectedMotor, selectedProp]);

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
        id="section-props"
        className="p-4 rounded-2xl border border-zinc-200 dark:border-zinc-800 bg-white dark:bg-zinc-900 space-y-3"
      >
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-2">
            <CollectionIcon
              collectionId="propellers"
              size={16}
              color={getCollectionColor("propellers")}
            />
            <h3 className="font-bold text-sm text-zinc-900 dark:text-zinc-100">7. Propellers</h3>
            <span className="text-[10px] uppercase font-bold tracking-wider px-2 py-0.5 rounded bg-zinc-100 dark:bg-zinc-800 text-zinc-600 dark:text-zinc-400">
              Required
            </span>
          </div>
          {selectedProp ? (
            <button
              type="button"
              onClick={() => {
                document
                  .getElementById("selected-prop-item")
                  ?.scrollIntoView({ behavior: "smooth", block: "center" });
              }}
              title="Scroll to selected propellers"
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
          Propellers generate aerodynamic thrust. Diameter is bounded by frame clearance, and
          mounting interface must match the motor shaft.
        </p>

        {/* Search & Compatibility Toggle */}
        <div className="flex items-center gap-2">
          <div className="relative flex-1">
            <Search size={14} className="absolute left-3 top-1/2 -translate-y-1/2 text-zinc-400" />
            <input
              type="text"
              value={search}
              onChange={(e) => setSearchQuery("propellers", e.target.value)}
              placeholder="Search propellers by name, diameter, mount..."
              className="w-full bg-zinc-50 dark:bg-zinc-950 border border-zinc-200 dark:border-zinc-800 rounded-xl pl-9 pr-8 py-1.5 text-xs text-zinc-900 dark:text-zinc-100 placeholder-zinc-400 focus:outline-none focus:ring-1 focus:ring-blue-500"
            />
            {search && (
              <button
                type="button"
                onClick={() => setSearchQuery("propellers", "")}
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
              onChange={() => toggleCompatibleOnly("propellers")}
              className="rounded border-zinc-300 dark:border-zinc-700 text-blue-600 focus:ring-blue-500 h-3.5 w-3.5 cursor-pointer"
            />
            <CompatibilityHoverCard filter={getCelFilterForStep(7)}>
              <span>Compatible only</span>
            </CompatibilityHoverCard>
          </label>
        </div>

        {/* Propellers Catalog List */}
        <div className="overflow-y-auto max-h-80 space-y-2 pr-1.5 focus:outline-none">
          {filteredProps.map((p: Propeller) => {
            const isSelected = selectedProp?.uuid === p.uuid || selectedProp?.id === p.id;
            const compat = checkPropCompatibility(p);
            const isIncompatible = !compat.compatible;

            return (
              <ProductHoverCard
                key={p.uuid || p.id}
                collectionId="propellers"
                item={p}
                productId={p.id || p.uuid}
                as="div"
                placement="side"
                id={isSelected ? "selected-prop-item" : undefined}
                className={`p-3 rounded-xl border transition-all cursor-pointer flex flex-col sm:flex-row sm:items-center justify-between gap-2 ${
                  isSelected
                    ? isIncompatible
                      ? "border-amber-500 bg-amber-50/50 dark:bg-amber-950/30 ring-1 ring-amber-500 shadow-xs"
                      : "border-blue-500 bg-blue-50/50 dark:bg-blue-950/30 ring-1 ring-blue-500 shadow-xs"
                    : isIncompatible
                      ? "opacity-60 saturate-50 hover:opacity-85 border-zinc-200 dark:border-zinc-800 bg-zinc-100/50 dark:bg-zinc-900/40"
                      : "border-zinc-200 dark:border-zinc-800 bg-zinc-50/50 dark:bg-zinc-900/60 hover:border-zinc-400 dark:hover:border-zinc-700"
                }`}
              >
                <div
                  role="button"
                  tabIndex={0}
                  onClick={() => setSelectedProp(p)}
                  className="flex-1 min-w-0"
                >
                  <div className="flex items-center gap-2 flex-wrap">
                    <span className="text-xs font-bold text-zinc-900 dark:text-zinc-100 truncate">
                      {p.name}
                    </span>
                    {isIncompatible && (
                      <span
                        title={compat.reason}
                        className="text-[10px] px-2 py-0.5 rounded font-medium bg-amber-500/10 text-amber-700 dark:text-amber-400 border border-amber-500/20 shrink-0 flex items-center gap-1"
                      >
                        <AlertCircle size={10} />
                        <span>{compat.reason}</span>
                      </span>
                    )}
                    {p.diameterMm && (
                      <span className="text-[10px] px-2 py-0.5 rounded-full font-semibold bg-blue-500/10 text-blue-600 dark:text-blue-400 border border-blue-500/20 shrink-0">
                        {p.diameterMm}mm ({(p.diameterMm / 25.4).toFixed(1)}")
                      </span>
                    )}
                    {p.pitchMm && (
                      <span className="text-[10px] px-2 py-0.5 rounded-full font-semibold bg-zinc-500/10 text-zinc-600 dark:text-zinc-400 border border-zinc-500/20 shrink-0">
                        pitch {(p.pitchMm / 25.4).toFixed(1)}"
                      </span>
                    )}
                    {p.blades && (
                      <span className="text-[10px] px-2 py-0.5 rounded-full font-semibold bg-purple-500/10 text-purple-600 dark:text-purple-400 border border-purple-500/20 shrink-0">
                        {p.blades}-blade
                      </span>
                    )}
                    {p.propMountInterface && (
                      <span className="text-[10px] px-2 py-0.5 rounded-full font-semibold bg-emerald-500/10 text-emerald-600 dark:text-emerald-400 border border-emerald-500/20 shrink-0">
                        {p.propMountInterface}
                      </span>
                    )}
                  </div>
                  <div className="text-[11px] text-zinc-500 dark:text-zinc-400 mt-0.5 flex items-center gap-2 flex-wrap">
                    <span>by {p.manufacturer}</span>
                    {p.weightG ? <span>• {p.weightG.toFixed(1)}g ea</span> : null}
                    {p.material ? <span>• {p.material}</span> : null}
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

          {filteredProps.length === 0 && (
            <p className="text-xs text-zinc-400 italic py-2 text-center">
              No propellers found matching your criteria.
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
          <span>Back: Motors</span>
        </button>

        <button
          type="button"
          disabled={!selectedProp || !!conflictMsg}
          onClick={handleNextStep}
          className="flex items-center gap-2 px-5 py-2.5 rounded-xl font-bold text-xs bg-blue-600 hover:bg-blue-700 disabled:opacity-40 disabled:cursor-not-allowed text-white transition-all shadow-sm"
        >
          <span>Next: FPV Camera</span>
          <ArrowRight size={14} />
        </button>
      </div>
    </div>
  );
}
