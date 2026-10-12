import React, { useState, useMemo, useEffect, useRef, useCallback } from "react";
import { useQuery } from "@connectrpc/connect-query";
import type { Battery } from "../gen/quadsmith/battery_pb";
import { listBatteries } from "../gen/quadsmith/battery-BatteryService_connectquery";
import { X, Search, Weight, Gauge, Check, ArrowUpDown, Filter } from "lucide-react";
import { getCollectionColor } from "../lib/collectionColors";
import { CollectionBadge } from "./CollectionBadge";
import { CollectionIcon } from "./CollectionIcon";
import { buildBatteryCelFilter } from "../lib/batteryFilter";
import { CopyFilterButton } from "./CopyFilterButton";

export interface BatteryPickerModalProps {
  isOpen: boolean;
  onClose: () => void;
  selectedBatteryId: string;
  onSelectBattery: (batteryId: string, battery?: Battery) => void;
  limits?: {
    minVoltage?: number;
    maxVoltage?: number;
    maxCurrentA?: number;
    connector?: string;
    compatibleConnectors?: string[];
  } | null;
  compatibleBatteries: Battery[];
  allBatteries?: Battery[];
  isLoading?: boolean;
}

type SortOption = "weight_asc" | "capacity_desc" | "current_desc";

export function BatteryPickerModal({
  isOpen,
  onClose,
  selectedBatteryId,
  onSelectBattery,
  limits,
  compatibleBatteries,
  allBatteries: propAllBatteries,
  isLoading,
}: BatteryPickerModalProps) {
  const [searchQuery, setSearchQuery] = useState("");
  const [sortBy, setSortBy] = useState<SortOption>("weight_asc");
  const [compatibleOnly, setCompatibleOnly] = useState<boolean>(true);
  const dialogRef = useRef<HTMLDialogElement>(null);

  // Sync native dialog state and handle escape key
  useEffect(() => {
    const dialog = dialogRef.current;
    if (!dialog) return;

    if (isOpen) {
      if (!dialog.open) {
        dialog.showModal();
      }
    } else {
      if (dialog.open) {
        dialog.close();
      }
    }
  }, [isOpen]);

  // Handle native cancel (escape key)
  const handleCancel = (e: React.SyntheticEvent) => {
    e.preventDefault();
    onClose();
  };

  // Fetch all batteries when modal is open if not provided in props
  const { data: allBatteriesResponse, isLoading: isLoadingAllBatteries } = useQuery(
    listBatteries,
    { sort: ["weight_g"], pageSize: 100 },
    { enabled: isOpen && !propAllBatteries },
  );

  const allBatteries = propAllBatteries || allBatteriesResponse?.batteries || [];

  const compatibleIdSet = useMemo(() => {
    return new Set(compatibleBatteries.map((b) => b.id || b.uuid).filter(Boolean));
  }, [compatibleBatteries]);

  const batteryFilter = useMemo(() => {
    return buildBatteryCelFilter(limits);
  }, [limits]);

  const getIncompatibilityReason = useCallback(
    (b: Battery): string | null => {
      // If compatibleIdSet already contains this battery, it is confirmed compatible
      if (compatibleIdSet.has(b.id) || compatibleIdSet.has(b.uuid)) {
        return null;
      }

      if (limits) {
        if (
          limits.minVoltage !== undefined &&
          limits.minVoltage > 0 &&
          b.maxVoltage &&
          b.maxVoltage < limits.minVoltage - 0.05
        ) {
          return `${b.maxVoltage.toFixed(1)}V < ${limits.minVoltage.toFixed(1)}V min`;
        }
        if (
          limits.minVoltage !== undefined &&
          limits.minVoltage > 0 &&
          b.minVoltage &&
          b.minVoltage < limits.minVoltage - 0.05
        ) {
          return `${b.minVoltage.toFixed(1)}V < ${limits.minVoltage.toFixed(1)}V min`;
        }
        if (
          limits.maxVoltage !== undefined &&
          limits.maxVoltage > 0 &&
          b.maxVoltage &&
          b.maxVoltage > limits.maxVoltage + 0.05
        ) {
          return `${b.maxVoltage.toFixed(1)}V > ${limits.maxVoltage.toFixed(1)}V max`;
        }
        if (
          limits.maxCurrentA !== undefined &&
          limits.maxCurrentA > 0 &&
          b.maxCurrentA &&
          b.maxCurrentA < limits.maxCurrentA - 0.1
        ) {
          return `${b.maxCurrentA.toFixed(0)}A < ${limits.maxCurrentA.toFixed(0)}A req`;
        }
        const conns =
          limits.compatibleConnectors && limits.compatibleConnectors.length > 0
            ? limits.compatibleConnectors
            : limits.connector
              ? [limits.connector]
              : [];
        if (conns.length > 0 && b.connector && !conns.includes(b.connector)) {
          return `${b.connector} ≠ ${conns.join("/")}`;
        }
      }

      if (compatibleIdSet.size > 0) {
        return "Limits exceeded";
      }

      return null;
    },
    [limits, compatibleIdSet],
  );

  const candidateBatteries = useMemo(() => {
    if (compatibleOnly) {
      return compatibleBatteries;
    }
    const seen = new Set<string>();
    const list: Battery[] = [];
    for (const b of [...allBatteries, ...compatibleBatteries]) {
      const id = b.id || b.uuid;
      if (id && !seen.has(id)) {
        seen.add(id);
        list.push(b);
      }
    }
    return list;
  }, [compatibleOnly, compatibleBatteries, allBatteries]);

  // Filter and sort batteries
  const filteredAndSortedBatteries = useMemo(() => {
    let result = [...candidateBatteries];

    if (searchQuery.trim()) {
      const q = searchQuery.toLowerCase().trim();
      result = result.filter(
        (b) =>
          b.name.toLowerCase().includes(q) ||
          b.manufacturer.toLowerCase().includes(q) ||
          b.connector.toLowerCase().includes(q) ||
          b.chemistry.toLowerCase().includes(q) ||
          b.id.toLowerCase().includes(q),
      );
    }

    result.sort((a, b) => {
      switch (sortBy) {
        case "weight_asc":
          return (a.weightG || 9999) - (b.weightG || 9999);
        case "capacity_desc":
          return (b.capacityMah || 0) - (a.capacityMah || 0);
        case "current_desc":
          return (b.maxCurrentA || 0) - (a.maxCurrentA || 0);
        default:
          return 0;
      }
    });

    return result;
  }, [candidateBatteries, searchQuery, sortBy]);

  if (!isOpen) return null;

  return (
    <dialog
      ref={dialogRef}
      onCancel={handleCancel}
      onClick={(e) => {
        // Light dismiss if clicking the dialog backdrop itself
        if (e.target === dialogRef.current) {
          onClose();
        }
      }}
      className="backdrop:bg-black/60 backdrop:backdrop-blur-xs fixed inset-0 m-auto max-w-2xl w-[92vw] max-h-[85vh] p-0 rounded-2xl bg-white dark:bg-zinc-900 border border-zinc-200 dark:border-zinc-800 shadow-2xl overflow-hidden flex flex-col z-50 text-zinc-900 dark:text-zinc-100"
    >
      {/* Header */}
      <div className="p-4 sm:p-5 border-b border-zinc-200 dark:border-zinc-800 flex items-center justify-between gap-3">
        <div className="min-w-0">
          <div className="flex items-center gap-2">
            <div
              className={`p-1.5 rounded-lg bg-zinc-100 dark:bg-zinc-800 border border-zinc-200/80 dark:border-zinc-700/60 ${getCollectionColor("batteries").textClass}`}
            >
              <CollectionIcon collection="batteries" size={18} />
            </div>
            <h2 className="text-lg font-bold truncate">Choose Compatible Battery</h2>
            <CollectionBadge collection="batteries" size="xs" />
          </div>
          {limits && (
            <div className="flex items-center gap-2 mt-1 text-xs text-zinc-500 dark:text-zinc-400 flex-wrap">
              <span className="font-medium text-zinc-600 dark:text-zinc-300">
                Electrical Limits:
              </span>
              {limits.minVoltage && limits.maxVoltage ? (
                <span className="px-1.5 py-0.5 rounded bg-zinc-100 dark:bg-zinc-800 text-zinc-700 dark:text-zinc-300 font-mono">
                  {limits.minVoltage.toFixed(1)}V – {limits.maxVoltage.toFixed(1)}V
                </span>
              ) : null}
              {limits.maxCurrentA ? (
                <span className="px-1.5 py-0.5 rounded bg-zinc-100 dark:bg-zinc-800 text-zinc-700 dark:text-zinc-300 font-mono">
                  ≥ {limits.maxCurrentA.toFixed(1)}A
                </span>
              ) : null}
              {limits.connector ? (
                <span className="px-1.5 py-0.5 rounded bg-blue-50 dark:bg-blue-900/30 text-blue-700 dark:text-blue-300 font-semibold font-mono">
                  {limits.connector} Connector
                </span>
              ) : null}
            </div>
          )}
          {compatibleOnly && batteryFilter && (
            <div className="mt-2 text-xs px-2.5 py-1.5 rounded-lg border bg-emerald-500/10 border-emerald-500/20 text-emerald-700 dark:text-emerald-300 flex items-center justify-between gap-2">
              <div className="flex items-center gap-1.5">
                <span className="font-semibold text-[10px] uppercase tracking-wider text-emerald-600 dark:text-emerald-400 shrink-0">
                  Compatibility Filter:
                </span>
                <CopyFilterButton filter={batteryFilter} />
              </div>
            </div>
          )}
        </div>

        <button
          type="button"
          onClick={onClose}
          aria-label="Close battery picker"
          className="p-1.5 rounded-lg text-zinc-400 hover:text-zinc-600 dark:hover:text-zinc-200 hover:bg-zinc-100 dark:hover:bg-zinc-800 transition-colors"
        >
          <X size={20} />
        </button>
      </div>

      {/* Filter and Sort Toolbar */}
      <div className="p-3 sm:px-5 border-b border-zinc-200 dark:border-zinc-800 bg-zinc-50/70 dark:bg-zinc-900/60 flex flex-col sm:flex-row gap-2.5 items-stretch sm:items-center justify-between">
        <div className="relative flex-1">
          <Search
            size={15}
            className="absolute left-3 top-1/2 -translate-y-1/2 text-zinc-400 pointer-events-none"
          />
          <input
            type="text"
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            placeholder="Search by name, brand, connector..."
            className="w-full pl-9 pr-3 py-1.5 text-xs sm:text-sm rounded-lg border border-zinc-300 dark:border-zinc-700 bg-white dark:bg-zinc-800 focus:outline-none focus:ring-2 focus:ring-blue-500 dark:focus:ring-blue-400"
          />
        </div>

        <div className="flex items-center gap-3 sm:gap-4 shrink-0 text-xs">
          <label className="flex items-center gap-1.5 cursor-pointer text-xs font-medium text-zinc-700 dark:text-zinc-300 select-none">
            <input
              type="checkbox"
              checked={compatibleOnly}
              onChange={(e) => setCompatibleOnly(e.target.checked)}
              className="rounded border-zinc-400 text-emerald-600 focus:ring-emerald-500 cursor-pointer"
            />
            <span>Compatible only</span>
          </label>

          <div className="flex items-center gap-1.5">
            <ArrowUpDown size={14} className="text-zinc-400" />
            <span className="text-zinc-500 dark:text-zinc-400 font-medium">Sort:</span>
            <select
              value={sortBy}
              onChange={(e) => setSortBy(e.target.value as SortOption)}
              className="px-2 py-1 rounded-lg border border-zinc-300 dark:border-zinc-700 bg-white dark:bg-zinc-800 text-zinc-700 dark:text-zinc-200 focus:outline-none focus:ring-2 focus:ring-blue-500"
            >
              <option value="weight_asc">Lightest (Recommended)</option>
              <option value="capacity_desc">Capacity (High to Low)</option>
              <option value="current_desc">Max Current (High to Low)</option>
            </select>
          </div>
        </div>
      </div>

      {/* Battery List Content */}
      <div className="flex-1 overflow-y-auto p-4 sm:p-5 space-y-2.5 min-h-[280px]">
        {isLoading ||
        (isLoadingAllBatteries && !compatibleOnly && candidateBatteries.length === 0) ? (
          <div className="py-12 text-center text-zinc-400 text-sm animate-pulse">
            Loading batteries...
          </div>
        ) : filteredAndSortedBatteries.length === 0 ? (
          <div className="py-12 text-center">
            <Filter size={32} className="mx-auto text-zinc-300 dark:text-zinc-600 mb-2" />
            <p className="text-sm font-semibold text-zinc-600 dark:text-zinc-300">
              No {compatibleOnly ? "compatible " : ""}batteries found
            </p>
            <p className="text-xs text-zinc-400 mt-1 max-w-sm mx-auto">
              {searchQuery
                ? `No batteries match "${searchQuery}". Try a different search term.`
                : compatibleOnly
                  ? "No batteries in the database match this build's electrical voltage and current limits. Uncheck 'Compatible only' to browse all batteries."
                  : "No batteries found in the database."}
            </p>
          </div>
        ) : (
          filteredAndSortedBatteries.map((b) => {
            const isSelected = b.id === selectedBatteryId || b.uuid === selectedBatteryId;
            const incompatibilityReason = getIncompatibilityReason(b);
            const isIncompatible = !!incompatibilityReason;

            return (
              <div
                key={b.uuid || b.id}
                onClick={() => {
                  onSelectBattery(b.id || b.uuid, b);
                  onClose();
                }}
                className={`group cursor-pointer p-3.5 rounded-xl border transition-all flex flex-col sm:flex-row sm:items-center justify-between gap-3 ${
                  isSelected
                    ? "border-blue-500 bg-blue-50/50 dark:bg-blue-950/30 ring-1 ring-blue-500"
                    : isIncompatible
                      ? "opacity-60 saturate-50 hover:opacity-85 border-zinc-200 dark:border-zinc-800 bg-zinc-100/50 dark:bg-zinc-900/40"
                      : "border-zinc-200 dark:border-zinc-800 bg-white dark:bg-zinc-900/40 hover:border-zinc-300 dark:hover:border-zinc-700 hover:bg-zinc-50/80 dark:hover:bg-zinc-850"
                }`}
              >
                <div className="min-w-0 flex-1">
                  <div className="flex items-center gap-2 flex-wrap mb-1">
                    <span className="font-semibold text-sm text-zinc-900 dark:text-zinc-100">
                      {b.name}
                    </span>
                    <span className="text-xs px-1.5 py-0.5 rounded bg-zinc-100 dark:bg-zinc-800 text-zinc-600 dark:text-zinc-400 font-mono">
                      {b.manufacturer}
                    </span>
                    {b.cellCountS ? (
                      <span className="text-xs px-1.5 py-0.5 rounded font-bold bg-amber-100 dark:bg-amber-950/60 text-amber-800 dark:text-amber-300 border border-amber-200/60 dark:border-amber-900/60">
                        {b.cellCountS}S
                      </span>
                    ) : null}
                    {b.connector ? (
                      <span className="text-xs px-1.5 py-0.5 rounded bg-blue-50 dark:bg-blue-950/50 text-blue-700 dark:text-blue-300">
                        {b.connector}
                      </span>
                    ) : null}
                    {isIncompatible && (
                      <span className="text-[10px] px-1.5 py-0.5 rounded font-medium bg-amber-500/10 text-amber-700 dark:text-amber-400 border border-amber-500/20 shrink-0">
                        Incompatible ({incompatibilityReason})
                      </span>
                    )}
                  </div>

                  <div className="flex items-center gap-4 text-xs text-zinc-500 dark:text-zinc-400 flex-wrap">
                    {b.weightG ? (
                      <span className="flex items-center gap-1">
                        <Weight size={12} className="text-zinc-400" />
                        <span className="font-semibold text-zinc-800 dark:text-zinc-200">
                          {b.weightG.toFixed(2)}g
                        </span>
                      </span>
                    ) : null}
                    {b.capacityMah ? (
                      <span>
                        Capacity:{" "}
                        <span className="font-medium text-zinc-700 dark:text-zinc-300">
                          {b.capacityMah}mAh
                        </span>
                      </span>
                    ) : null}
                    {b.minVoltage && b.maxVoltage ? (
                      <span>
                        Voltage:{" "}
                        <span className="font-medium font-mono text-zinc-700 dark:text-zinc-300">
                          {b.minVoltage.toFixed(1)}–{b.maxVoltage.toFixed(1)}V
                        </span>
                      </span>
                    ) : null}
                    {b.maxCurrentA ? (
                      <span className="flex items-center gap-1">
                        <Gauge size={12} className="text-zinc-400" />
                        <span>
                          Current:{" "}
                          <span className="font-medium text-zinc-700 dark:text-zinc-300">
                            {b.maxCurrentA.toFixed(1)}A
                          </span>
                        </span>
                      </span>
                    ) : null}
                  </div>
                </div>

                <div className="shrink-0 flex items-center gap-2">
                  {isSelected ? (
                    <span className="inline-flex items-center gap-1 px-2.5 py-1 rounded-lg text-xs font-semibold bg-blue-600 text-white shadow-xs">
                      <Check size={13} />
                      <span>Active</span>
                    </span>
                  ) : (
                    <span className="inline-flex items-center px-2.5 py-1 rounded-lg text-xs font-medium border border-zinc-300 dark:border-zinc-700 text-zinc-700 dark:text-zinc-300 group-hover:bg-zinc-100 dark:group-hover:bg-zinc-800 transition-colors">
                      Select
                    </span>
                  )}
                </div>
              </div>
            );
          })
        )}
      </div>

      {/* Footer */}
      <div className="p-3 sm:px-5 border-t border-zinc-200 dark:border-zinc-800 bg-zinc-50/50 dark:bg-zinc-900/40 flex items-center justify-between text-xs text-zinc-500">
        <span>
          Showing {filteredAndSortedBatteries.length} {compatibleOnly ? "compatible " : ""}
          {filteredAndSortedBatteries.length === 1 ? "battery" : "batteries"}
        </span>
        <button
          type="button"
          onClick={onClose}
          className="px-3 py-1.5 rounded-lg border border-zinc-300 dark:border-zinc-700 bg-white dark:bg-zinc-800 hover:bg-zinc-50 dark:hover:bg-zinc-700 text-zinc-700 dark:text-zinc-200 transition-colors font-medium"
        >
          Close
        </button>
      </div>
    </dialog>
  );
}
