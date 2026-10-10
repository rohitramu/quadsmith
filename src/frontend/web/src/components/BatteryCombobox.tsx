import { useState, useRef, useEffect, useMemo } from "react";
import { useQuery } from "@connectrpc/connect-query";
import { Search, ChevronDown, X, Check, Loader2 } from "lucide-react";
import { search } from "../gen/quadsmith/search-SearchService_connectquery";
import { ProductHoverCard } from "./ProductHoverCard";

export interface BatteryComboboxProps {
  selectedBatteryId: string;
  onSelectBattery: (batteryId: string, battery?: any) => void;
  batteries?: any[];
  placeholder?: string;
  className?: string;
  disabled?: boolean;
  isLoading?: boolean;
  ariaLabel?: string;
}

export function BatteryCombobox({
  selectedBatteryId,
  onSelectBattery,
  batteries = [],
  placeholder = "Search or select battery...",
  className = "",
  disabled = false,
  isLoading = false,
  ariaLabel = "Select battery",
}: BatteryComboboxProps) {
  const [isOpen, setIsOpen] = useState(false);
  const [searchQuery, setSearchQuery] = useState("");
  const [debouncedQuery, setDebouncedQuery] = useState("");
  const [highlightedIndex, setHighlightedIndex] = useState(-1);

  const containerRef = useRef<HTMLDivElement>(null);
  const inputRef = useRef<HTMLInputElement>(null);
  const listRef = useRef<HTMLDivElement>(null);

  // Debounce search query by 200ms
  useEffect(() => {
    const timer = setTimeout(() => {
      setDebouncedQuery(searchQuery.trim());
    }, 200);
    return () => clearTimeout(timer);
  }, [searchQuery]);

  // Query search API for batteries if query is non-empty
  const { data: searchResponse, isLoading: isSearching } = useQuery(
    search,
    {
      query: debouncedQuery,
      selectors: [{ path: "components/hardware/batteries" }],
      limit: 15,
    },
    { enabled: debouncedQuery.length > 0 },
  );

  // Find currently selected battery object
  const selectedBattery = useMemo(() => {
    return batteries.find((b) => b.id === selectedBatteryId || b.uuid === selectedBatteryId);
  }, [batteries, selectedBatteryId]);

  // Combine and deduplicate options:
  // If debouncedQuery is non-empty, use search results (merging with catalog objects if matched)
  // Otherwise, use all available batteries passed in props
  const displayItems = useMemo(() => {
    if (debouncedQuery.length > 0) {
      const results = searchResponse?.results || [];
      return results.map((res) => {
        const matched = batteries.find((b) => b.id === res.id || b.uuid === res.uuid);
        if (matched) return matched;
        return {
          id: res.id,
          uuid: res.uuid,
          name: res.name,
          description: res.description,
          primaryDisplayImage: res.primaryDisplayImage,
        };
      });
    }
    return batteries;
  }, [debouncedQuery, searchResponse, batteries]);

  // Reset highlighted index when items change
  useEffect(() => {
    setHighlightedIndex(-1);
  }, [displayItems]);

  // Close dropdown on click outside
  useEffect(() => {
    function handleClickOutside(e: MouseEvent) {
      if (containerRef.current && !containerRef.current.contains(e.target as Node)) {
        setIsOpen(false);
        setSearchQuery("");
      }
    }
    document.addEventListener("mousedown", handleClickOutside);
    return () => document.removeEventListener("mousedown", handleClickOutside);
  }, []);

  const handleSelect = (battery: any) => {
    const id = battery.id || battery.uuid;
    onSelectBattery(id, battery);
    setSearchQuery("");
    setIsOpen(false);
  };

  const handleClear = (e: React.MouseEvent) => {
    e.stopPropagation();
    if (searchQuery) {
      setSearchQuery("");
      inputRef.current?.focus();
    } else {
      setIsOpen(false);
    }
  };

  const handleKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {
    if (disabled) return;

    if (!isOpen) {
      if (e.key === "ArrowDown" || e.key === "ArrowUp" || e.key === "Enter") {
        e.preventDefault();
        setIsOpen(true);
      }
      return;
    }

    if (e.key === "ArrowDown") {
      e.preventDefault();
      setHighlightedIndex((prev) => (prev < displayItems.length - 1 ? prev + 1 : 0));
    } else if (e.key === "ArrowUp") {
      e.preventDefault();
      setHighlightedIndex((prev) => (prev > 0 ? prev - 1 : displayItems.length - 1));
    } else if (e.key === "Enter") {
      e.preventDefault();
      if (highlightedIndex >= 0 && highlightedIndex < displayItems.length) {
        handleSelect(displayItems[highlightedIndex]);
      } else if (displayItems.length > 0) {
        handleSelect(displayItems[0]);
      }
    } else if (e.key === "Escape") {
      e.preventDefault();
      setIsOpen(false);
      setSearchQuery("");
    }
  };

  // Format label for display when closed or idle
  const displayLabel = useMemo(() => {
    if (!selectedBattery) return "";
    const specs = [
      selectedBattery.weightG ? `${selectedBattery.weightG}g` : "",
      selectedBattery.cellCountS ? `${selectedBattery.cellCountS}S` : "",
      selectedBattery.capacityMah ? `${selectedBattery.capacityMah}mAh` : "",
    ]
      .filter(Boolean)
      .join(", ");
    return specs ? `${selectedBattery.name} (${specs})` : selectedBattery.name;
  }, [selectedBattery]);

  return (
    <div ref={containerRef} className={`relative ${className}`}>
      {/* Combobox Trigger / Search Input */}
      <div className="relative">
        <div className="absolute inset-y-0 left-0 pl-2.5 flex items-center pointer-events-none text-zinc-400">
          {isSearching || isLoading ? (
            <Loader2 className="w-3.5 h-3.5 animate-spin text-blue-500" />
          ) : (
            <Search className="w-3.5 h-3.5" />
          )}
        </div>

        <input
          ref={inputRef}
          type="text"
          role="combobox"
          aria-expanded={isOpen}
          aria-haspopup="listbox"
          aria-label={ariaLabel}
          aria-autocomplete="list"
          disabled={disabled}
          value={isOpen ? searchQuery : displayLabel}
          onChange={(e) => {
            setSearchQuery(e.target.value);
            if (!isOpen) setIsOpen(true);
          }}
          onFocus={() => {
            if (!disabled) setIsOpen(true);
          }}
          onClick={() => {
            if (!disabled && !isOpen) setIsOpen(true);
          }}
          onKeyDown={handleKeyDown}
          placeholder={isOpen ? "Type to search batteries..." : displayLabel || placeholder}
          className="w-full pl-8 pr-14 py-1.5 rounded-lg bg-zinc-50 dark:bg-zinc-900 border border-zinc-200 dark:border-zinc-800 hover:border-zinc-300 dark:hover:border-zinc-700 text-xs text-zinc-900 dark:text-zinc-100 placeholder-zinc-400 dark:placeholder-zinc-500 focus:outline-none focus:ring-1 focus:ring-blue-500 font-medium transition-all shadow-xs truncate cursor-pointer disabled:opacity-50 disabled:cursor-not-allowed"
          data-testid="battery-combobox-input"
          data-battery-id={selectedBatteryId}
        />

        <div className="absolute inset-y-0 right-0 pr-1.5 flex items-center gap-0.5">
          {isOpen && searchQuery && (
            <button
              type="button"
              onClick={handleClear}
              aria-label="Clear search"
              className="p-1 rounded text-zinc-400 hover:text-zinc-600 dark:hover:text-zinc-200 hover:bg-zinc-200 dark:hover:bg-zinc-800 transition-colors"
            >
              <X size={12} />
            </button>
          )}
          <button
            type="button"
            disabled={disabled}
            onClick={() => {
              if (!disabled) {
                setIsOpen(!isOpen);
                if (!isOpen) inputRef.current?.focus();
              }
            }}
            aria-label="Toggle battery list"
            className="p-1 rounded text-zinc-400 hover:text-zinc-600 dark:hover:text-zinc-200 transition-colors"
          >
            <ChevronDown
              size={13}
              className={`transition-transform duration-200 ${isOpen ? "rotate-180" : ""}`}
            />
          </button>
        </div>
      </div>

      {/* Dropdown Menu */}
      {isOpen && (
        <div
          role="listbox"
          aria-label="Batteries"
          ref={listRef}
          className="absolute left-0 right-0 top-full mt-1 z-50 rounded-xl bg-white dark:bg-zinc-900 border border-zinc-200 dark:border-zinc-800 shadow-xl overflow-hidden animate-in fade-in zoom-in-95 duration-100"
          data-testid="battery-combobox-dropdown"
        >
          {/* Header count info */}
          <div className="px-3 py-1.5 border-b border-zinc-100 dark:border-zinc-800 bg-zinc-50 dark:bg-zinc-950/60 flex items-center justify-between text-[10px] text-zinc-500 dark:text-zinc-400">
            <span>
              {displayItems.length} {displayItems.length === 1 ? "battery" : "batteries"}{" "}
              {debouncedQuery ? "found" : "available"}
            </span>
            <span className="font-mono text-[9px] text-zinc-400">Hover for specs</span>
          </div>

          {/* List items */}
          <ul className="max-h-60 overflow-y-auto divide-y divide-zinc-100 dark:divide-zinc-800/40 text-xs">
            {displayItems.length === 0 ? (
              <li className="p-3 text-center text-xs text-zinc-400 dark:text-zinc-500">
                {isSearching ? "Searching batteries..." : "No matching batteries found"}
              </li>
            ) : (
              displayItems.map((b, idx) => {
                const bId = b.id || b.uuid;
                const isSelected = selectedBatteryId === bId;
                const isHighlighted = highlightedIndex === idx;

                return (
                  <ProductHoverCard
                    key={bId || idx}
                    collectionId="batteries"
                    item={b}
                    productId={bId}
                    as="li"
                    className={`p-2.5 hover:bg-zinc-100 dark:hover:bg-zinc-800/80 cursor-pointer flex items-center justify-between transition-colors ${
                      isSelected
                        ? "bg-blue-50/80 dark:bg-blue-950/30 border-l-2 border-blue-500"
                        : ""
                    } ${isHighlighted ? "bg-zinc-100 dark:bg-zinc-800" : ""}`}
                  >
                    <div
                      role="option"
                      aria-selected={isSelected}
                      onClick={() => handleSelect(b)}
                      className="flex items-center gap-2.5 min-w-0 flex-1"
                    >
                      <span
                        className={`w-2 h-2 rounded-full shrink-0 ${
                          b.cellCountS === 6
                            ? "bg-amber-500 dark:bg-amber-400"
                            : b.cellCountS === 4
                              ? "bg-emerald-500 dark:bg-emerald-400"
                              : "bg-blue-500 dark:bg-blue-400"
                        }`}
                      />
                      <div className="min-w-0 flex-1">
                        <div className="font-medium text-zinc-900 dark:text-zinc-100 truncate">
                          {b.name}
                        </div>
                        <div className="text-[10px] text-zinc-500 dark:text-zinc-400 font-mono flex items-center gap-1.5 flex-wrap">
                          {b.cellCountS ? <span>{b.cellCountS}S</span> : null}
                          {b.capacityMah ? <span>• {b.capacityMah}mAh</span> : null}
                          {b.weightG ? <span>• {b.weightG}g</span> : null}
                          {b.connector ? <span>• {b.connector}</span> : null}
                        </div>
                      </div>
                    </div>
                    {isSelected && (
                      <Check size={14} className="text-blue-600 dark:text-blue-400 shrink-0 ml-2" />
                    )}
                  </ProductHoverCard>
                );
              })
            )}
          </ul>
        </div>
      )}
    </div>
  );
}
