import { useParams, Link, useSearchParams } from "react-router-dom";
import { useQuery } from "@connectrpc/connect-query";
import { getOption } from "@bufbuild/protobuf";
import { frontend as frontendOpt } from "@quadsmith/shared/gen/quadsmith/common/common_pb";
import {
  ArrowUp,
  ArrowDown,
  SlidersHorizontal,
  GripVertical,
  ChevronUp,
  ChevronDown,
  Plus,
  ChevronLeft,
  ChevronRight,
  ChevronsLeft,
} from "lucide-react";
import { useState, useMemo, useRef, useEffect, useLayoutEffect } from "react";
import { keepPreviousData } from "@tanstack/react-query";
import { SmartFilterInput } from "../components/SmartFilterInput";
import {
  getHardwareCollection,
  getCollectionPath,
  type HardwareCollectionDef,
  type ColumnConfig,
} from "../lib/hardwareCollections";
import { CollectionBadge } from "@quadsmith/shared";
import { CollectionIcon } from "@quadsmith/shared";
import { getCollectionColor } from "@quadsmith/shared";
import { useDocumentMeta } from "@quadsmith/shared";
import { StyledSelect } from "../components/StyledSelect";

function PropertySelector({
  allProperties,
  selectedPropertyIds,
  defaultPropertyIds,
  onChange,
}: {
  allProperties: ColumnConfig[];
  selectedPropertyIds: string[];
  defaultPropertyIds: string[];
  onChange: (selectedIds: string[]) => void;
}) {
  const [isOpen, setIsOpen] = useState(false);
  const [draggedIndex, setDraggedIndex] = useState<number | null>(null);
  const [dragOverIndex, setDragOverIndex] = useState<number | null>(null);
  const dropdownRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    function handleClickOutside(event: MouseEvent) {
      if (dropdownRef.current && !dropdownRef.current.contains(event.target as Node)) {
        setIsOpen(false);
      }
    }
    function handleKeyDown(event: KeyboardEvent) {
      if (event.key === "Escape") {
        setIsOpen(false);
      }
    }
    if (isOpen) {
      document.addEventListener("mousedown", handleClickOutside);
      document.addEventListener("keydown", handleKeyDown);
    }
    return () => {
      document.removeEventListener("mousedown", handleClickOutside);
      document.removeEventListener("keydown", handleKeyDown);
    };
  }, [isOpen]);

  const propertyMap = useMemo(() => {
    return new Map(allProperties.map((prop) => [prop.id, prop]));
  }, [allProperties]);

  // Active/Visible properties in their current order
  const visibleProperties = useMemo(() => {
    return selectedPropertyIds
      .map((id) => propertyMap.get(id))
      .filter((prop): prop is ColumnConfig => !!prop);
  }, [selectedPropertyIds, propertyMap]);

  // Hidden properties
  const hiddenProperties = useMemo(() => {
    return allProperties.filter((prop) => !selectedPropertyIds.includes(prop.id));
  }, [allProperties, selectedPropertyIds]);

  const moveProperty = (index: number, direction: -1 | 1) => {
    const targetIndex = index + direction;
    if (targetIndex < 0 || targetIndex >= selectedPropertyIds.length) return;
    const next = [...selectedPropertyIds];
    const [moved] = next.splice(index, 1);
    next.splice(targetIndex, 0, moved);
    onChange(next);
  };

  const hideProperty = (id: string) => {
    if (selectedPropertyIds.length <= 1) return; // Keep at least one property visible
    onChange(selectedPropertyIds.filter((propId) => propId !== id));
  };

  const showProperty = (id: string) => {
    onChange([...selectedPropertyIds, id]);
  };

  const resetToDefault = () => {
    onChange(defaultPropertyIds);
  };

  return (
    <div className="relative inline-block" ref={dropdownRef}>
      <button
        type="button"
        onClick={() => setIsOpen(!isOpen)}
        className="flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium bg-zinc-900 border border-zinc-800 rounded-lg text-zinc-300 hover:bg-zinc-800 transition-colors shadow-xs cursor-pointer"
        title="Customize and reorder visible properties"
      >
        <SlidersHorizontal size={14} />
        <span>Properties ({selectedPropertyIds.length})</span>
      </button>

      {isOpen && (
        <div className="absolute right-0 top-full mt-1.5 w-72 bg-zinc-900 border border-zinc-800 rounded-lg shadow-xl z-50 p-2.5 text-xs">
          <div className="flex items-center justify-between px-2 py-1.5 border-b border-zinc-800 mb-2">
            <span className="font-semibold text-zinc-200">
              Properties{" "}
              <span className="font-normal text-zinc-400">
                ({visibleProperties.length} visible)
              </span>
            </span>
            <button
              type="button"
              onClick={resetToDefault}
              className="text-[11px] text-blue-400 hover:underline cursor-pointer"
            >
              Reset to Default
            </button>
          </div>

          {/* Visible Properties (Reorderable) */}
          <div className="mb-2">
            <div className="px-2 py-0.5 text-[10px] font-semibold text-zinc-500 uppercase tracking-wider flex justify-between items-center">
              <span>Visible (drag or arrows)</span>
            </div>
            <div className="space-y-1 mt-1 max-h-56 overflow-y-auto">
              {visibleProperties.map((prop, idx) => (
                <div
                  key={prop.id}
                  draggable
                  onDragStart={(e) => {
                    e.dataTransfer.effectAllowed = "move";
                    e.dataTransfer.setData("text/quadsmith-visible-idx", idx.toString());
                    setDraggedIndex(idx);
                  }}
                  onDragOver={(e) => {
                    e.preventDefault();
                    e.dataTransfer.dropEffect = "move";
                    if (dragOverIndex !== idx) setDragOverIndex(idx);
                  }}
                  onDragLeave={() => {
                    if (dragOverIndex === idx) setDragOverIndex(null);
                  }}
                  onDrop={(e) => {
                    e.preventDefault();
                    setDragOverIndex(null);
                    const hiddenId = e.dataTransfer.getData("text/quadsmith-hidden-id");
                    if (hiddenId) {
                      const next = [...selectedPropertyIds];
                      next.splice(idx, 0, hiddenId);
                      onChange(next);
                      setDraggedIndex(null);
                      return;
                    }
                    if (draggedIndex !== null && draggedIndex !== idx) {
                      const next = [...selectedPropertyIds];
                      const [moved] = next.splice(draggedIndex, 1);
                      next.splice(idx, 0, moved);
                      onChange(next);
                    }
                    setDraggedIndex(null);
                  }}
                  onDragEnd={() => {
                    setDraggedIndex(null);
                    setDragOverIndex(null);
                  }}
                  className={`flex items-center gap-1.5 px-2 py-1 rounded transition-colors group ${
                    draggedIndex === idx
                      ? "opacity-30 bg-zinc-800"
                      : dragOverIndex === idx
                        ? "border-t-2 border-blue-500 bg-blue-950/30"
                        : "hover:bg-zinc-800/60 bg-zinc-900/50 border border-zinc-800/70"
                  }`}
                >
                  <span
                    title="Drag to reorder"
                    className="cursor-grab active:cursor-grabbing shrink-0 flex items-center"
                  >
                    <GripVertical size={13} className="text-zinc-400 group-hover:text-zinc-200" />
                  </span>
                  <input
                    type="checkbox"
                    checked={true}
                    disabled={visibleProperties.length <= 1}
                    onChange={() => hideProperty(prop.id)}
                    className="rounded border-zinc-700 text-blue-600 focus:ring-blue-500 cursor-pointer disabled:opacity-40"
                    title={
                      visibleProperties.length <= 1
                        ? "At least one property must be visible"
                        : "Hide property"
                    }
                  />
                  <span className="font-medium text-zinc-200 flex-1 truncate select-none">
                    {prop.title}
                  </span>
                  <div className="flex items-center gap-0.5 shrink-0">
                    <button
                      type="button"
                      disabled={idx === 0}
                      onClick={() => moveProperty(idx, -1)}
                      className="p-0.5 rounded text-zinc-400 hover:text-zinc-200 hover:bg-zinc-700 disabled:opacity-20 disabled:hover:bg-transparent cursor-pointer disabled:cursor-not-allowed"
                      title="Move up / left"
                    >
                      <ChevronUp size={13} />
                    </button>
                    <button
                      type="button"
                      disabled={idx === visibleProperties.length - 1}
                      onClick={() => moveProperty(idx, 1)}
                      className="p-0.5 rounded text-zinc-400 hover:text-zinc-200 hover:bg-zinc-700 disabled:opacity-20 disabled:hover:bg-transparent cursor-pointer disabled:cursor-not-allowed"
                      title="Move down / right"
                    >
                      <ChevronDown size={13} />
                    </button>
                  </div>
                </div>
              ))}
            </div>
          </div>

          {/* Hidden Properties */}
          {hiddenProperties.length > 0 && (
            <div className="border-t border-zinc-800 pt-2">
              <div className="px-2 py-0.5 text-[10px] font-semibold text-zinc-500 uppercase tracking-wider">
                <span>Hidden Properties (click to show)</span>
              </div>
              <div className="space-y-0.5 mt-1 max-h-36 overflow-y-auto">
                {hiddenProperties.map((prop) => (
                  <div
                    key={prop.id}
                    draggable
                    onDragStart={(e) => {
                      e.dataTransfer.effectAllowed = "copyMove";
                      e.dataTransfer.setData("text/quadsmith-hidden-id", prop.id);
                    }}
                    onClick={() => showProperty(prop.id)}
                    className="flex items-center gap-2 px-2 py-1.5 rounded hover:bg-zinc-800/60 cursor-pointer select-none text-zinc-400 group"
                    title="Click to add or drag into visible list"
                  >
                    <input
                      type="checkbox"
                      checked={false}
                      onChange={() => showProperty(prop.id)}
                      className="rounded border-zinc-700 text-blue-600 focus:ring-blue-500 cursor-pointer"
                    />
                    <span className="flex-1 truncate group-hover:text-zinc-300">{prop.title}</span>
                    <Plus size={12} className="text-zinc-400 group-hover:text-zinc-200 shrink-0" />
                  </div>
                ))}
              </div>
            </div>
          )}
        </div>
      )}
    </div>
  );
}

const HEADER_COLUMN_IDS = new Set(["name", "manufacturer"]);
const NON_AUTO_SPEC_COLUMN_IDS = new Set([
  "id",
  "uuid",
  "name",
  "manufacturer",
  "is_internal_only",
  "weight_g",
  "description",
  "primary_display_image",
]);
const TARGET_DEFAULT_SPEC_COUNT = 5;

const XL_SPEC_GRID_COLS: Record<number, string> = {
  1: "xl:grid-cols-1",
  2: "xl:grid-cols-2",
  3: "xl:grid-cols-3",
  4: "xl:grid-cols-4",
  5: "xl:grid-cols-5",
};

function CatalogRowCard({
  item,
  productUrl,
  collectionId,
  colTextClass,
  activeColumns,
  xlSpecColsClass,
}: {
  item: any;
  productUrl: string;
  collectionId: string;
  colTextClass: string;
  activeColumns: ColumnConfig[];
  xlSpecColsClass: string;
}) {
  const cardRef = useRef<HTMLElement>(null);
  const bodyRef = useRef<HTMLDivElement>(null);

  useLayoutEffect(() => {
    const cardEl = cardRef.current;
    const bodyEl = bodyRef.current;
    if (!cardEl || !bodyEl) return;

    let lastApplied = 0;
    const syncPhotoSize = () => {
      const bodyHeight = Math.round(bodyEl.getBoundingClientRect().height);
      if (bodyHeight <= 0) return;
      const cardWidth = Math.round(cardEl.clientWidth);
      const nextSize =
        cardWidth > 0 ? Math.min(bodyHeight, Math.floor(cardWidth * 0.48)) : bodyHeight;
      if (nextSize !== lastApplied) {
        lastApplied = nextSize;
        cardEl.style.setProperty("--card-photo-size", `${nextSize}px`);
      }
    };

    syncPhotoSize();
    if (typeof ResizeObserver === "undefined") return;

    const observer = new ResizeObserver(() => {
      syncPhotoSize();
    });
    observer.observe(bodyEl);
    observer.observe(cardEl);
    return () => observer.disconnect();
  }, [activeColumns.length, item.description]);

  return (
    <Link
      to={productUrl}
      data-testid="catalog-card"
      aria-label={`View ${item.name || item.id}`}
      className="group block rounded-2xl focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-emerald-500/70"
    >
      <article
        ref={cardRef}
        className="flex items-stretch rounded-2xl border border-zinc-800 bg-zinc-900/90 overflow-clip shadow-xs transition-all group-hover:border-zinc-700 group-hover:bg-zinc-900 group-hover:shadow-md"
      >
        {/* Left Identity + Middle Spec Pills */}
        <div
          ref={bodyRef}
          className="flex-1 min-w-0 p-2.5 sm:p-4 lg:px-5 flex flex-col lg:flex-row lg:items-center gap-2 sm:gap-3 lg:gap-6"
        >
          <div className="lg:w-64 xl:w-[21rem] shrink-0 min-w-0">
            <div className="flex items-center gap-2 mb-0.5 min-w-0">
              {item.manufacturer && (
                <span className="text-[10px] sm:text-[11px] font-bold text-zinc-400 uppercase tracking-wider truncate">
                  {item.manufacturer}
                </span>
              )}
              {(item.isInternalOnly || item.is_internal_only) && (
                <span className="px-2 py-0.5 rounded-full text-[10px] font-semibold bg-amber-500/10 text-amber-400 border border-amber-500/20 shrink-0">
                  Internal
                </span>
              )}
            </div>
            <h2 className="font-bold text-[13.5px] sm:text-base text-zinc-50 group-hover:text-emerald-400 transition-colors truncate">
              {item.name || item.id}
            </h2>
            {item.description && (
              <p className="hidden sm:block text-xs text-zinc-400 truncate mt-0.5 leading-relaxed">
                {item.description}
              </p>
            )}
          </div>

          <div className={`flex-1 min-w-0 grid grid-cols-3 ${xlSpecColsClass} gap-1.5 sm:gap-2`}>
            {activeColumns.map((col) => (
              <div
                key={col.id}
                className="bg-zinc-800/50 rounded-lg sm:rounded-xl px-1.5 py-1 sm:px-2.5 sm:py-1.5 border border-zinc-800/80 flex flex-col min-w-0"
              >
                <span className="text-[9px] sm:text-[10px] font-semibold uppercase tracking-wider text-zinc-500 truncate">
                  {col.title}
                </span>
                <div className="text-[11px] sm:text-[13px] font-semibold text-zinc-200 truncate mt-0.5">
                  {col.renderCell(item)}
                </div>
              </div>
            ))}
          </div>
        </div>

        {/* Right Full-Height 1:1 Square Product Photo */}
        <div
          data-testid="catalog-card-photo"
          className={`w-[var(--card-photo-size,6rem)] aspect-square self-stretch shrink-0 relative overflow-clip border-l border-zinc-800/80 bg-zinc-950/65 flex items-center justify-center ${colTextClass}`}
        >
          {item.primaryDisplayImage ? (
            <img
              src={item.primaryDisplayImage}
              alt=""
              loading="lazy"
              className="absolute inset-0 block w-full h-full object-cover transition-transform duration-200 group-hover:scale-105"
            />
          ) : (
            <CollectionIcon collection={collectionId} size={28} className="opacity-70" />
          )}
        </div>
      </article>
    </Link>
  );
}

export function CollectionTableView({
  categoryId,
  collection,
}: {
  categoryId: string;
  collection: HardwareCollectionDef;
}) {
  const [searchParams, setSearchParams] = useSearchParams();

  // Read default columns from protobuf option if available, otherwise collection defaults
  const protoDefaultCols = useMemo(() => {
    try {
      const opts = getOption(collection.schema, frontendOpt);
      if (opts?.defaultColumns && opts.defaultColumns.length > 0) {
        return opts.defaultColumns;
      }
    } catch {
      // Fallback if option not present
    }
    return collection.defaultColumnIds;
  }, [collection]);

  const defaultSpecColumnIds = useMemo(() => {
    const result = protoDefaultCols.filter(
      (c) => c in collection.columns && !HEADER_COLUMN_IDS.has(c) && c !== "primary_display_image",
    );
    const hasWeight = "weight_g" in collection.columns;
    const reserveWeightSlot =
      hasWeight && !result.includes("weight_g") && result.length < TARGET_DEFAULT_SPEC_COUNT;
    const domainLimit = reserveWeightSlot
      ? TARGET_DEFAULT_SPEC_COUNT - 1
      : TARGET_DEFAULT_SPEC_COUNT;

    for (const key of Object.keys(collection.columns)) {
      if (result.length >= domainLimit) break;
      if (NON_AUTO_SPEC_COLUMN_IDS.has(key)) continue;
      if (!result.includes(key)) {
        result.push(key);
      }
    }

    if (reserveWeightSlot && result.length < TARGET_DEFAULT_SPEC_COUNT) {
      result.push("weight_g");
    }

    for (const key of Object.keys(collection.columns)) {
      if (result.length >= TARGET_DEFAULT_SPEC_COUNT) break;
      if (NON_AUTO_SPEC_COLUMN_IDS.has(key)) continue;
      if (!result.includes(key)) {
        result.push(key);
      }
    }

    return result.length > 0 ? result : Object.keys(collection.columns).slice(0, 1);
  }, [protoDefaultCols, collection.columns]);

  const selectableProperties = useMemo(() => {
    return Object.values(collection.columns).filter(
      (col) => !HEADER_COLUMN_IDS.has(col.id) && col.id !== "primary_display_image",
    );
  }, [collection.columns]);

  const selectedColumnIds = useMemo(() => {
    const propsParam = searchParams.get("properties") || searchParams.get("columns");
    if (!propsParam) {
      return defaultSpecColumnIds;
    }
    const parsed = propsParam
      .split(",")
      .map((c) => c.trim())
      .filter(
        (c) =>
          c in collection.columns && !HEADER_COLUMN_IDS.has(c) && c !== "primary_display_image",
      );
    return parsed.length > 0 ? parsed : defaultSpecColumnIds;
  }, [searchParams, defaultSpecColumnIds, collection.columns]);

  const filterQuery = searchParams.get("filter") || "";

  const { sortField, sortDir } = useMemo(() => {
    const sortParam = searchParams.get("sort");
    const dirParam =
      searchParams.get("dir") || searchParams.get("sort_dir") || searchParams.get("order");
    if (!sortParam) {
      return { sortField: null, sortDir: "asc" as const };
    }
    if (sortParam.startsWith("^") || sortParam.startsWith("-")) {
      const field = sortParam.slice(1);
      return {
        sortField: field in collection.columns ? field : null,
        sortDir: "desc" as const,
      };
    }
    return {
      sortField: sortParam in collection.columns ? sortParam : null,
      sortDir: dirParam === "desc" ? ("desc" as const) : ("asc" as const),
    };
  }, [searchParams, collection.columns]);

  useDocumentMeta({
    title: `${collection.name} Catalog — Quadsmith`,
    description: `Browse, filter, and compare ${collection.name} specifications, weights, and compatibility on Quadsmith.`,
  });

  const [pageSize, setPageSize] = useState<number>(20);
  const [pageIndex, setPageIndex] = useState<number>(0);
  const [tokenHistory, setTokenHistory] = useState<string[]>([""]);
  const colColor = collection.color || getCollectionColor(collection.id);

  // Reset pagination when search params change
  const prevSearchParamsRef = useRef(searchParams.toString());
  useEffect(() => {
    const currentParams = searchParams.toString();
    if (prevSearchParamsRef.current !== currentParams) {
      prevSearchParamsRef.current = currentParams;
      setPageIndex(0);
      setTokenHistory([""]);
    }
  }, [searchParams]);

  const hasInternalField = useMemo(
    () => collection.fields.some((f) => f.name === "is_internal_only"),
    [collection],
  );

  const effectiveFilter = useMemo(() => {
    if (!hasInternalField) {
      return filterQuery;
    }
    if (filterQuery && filterQuery.includes("is_internal_only")) {
      return filterQuery;
    }
    if (!filterQuery) {
      return "is_internal_only == false";
    }
    return `(${filterQuery}) && is_internal_only == false`;
  }, [hasInternalField, filterQuery]);

  const activeColumns = useMemo(() => {
    return selectedColumnIds
      .map((colKey) => collection.columns[colKey])
      .filter((c): c is ColumnConfig => !!c);
  }, [selectedColumnIds, collection]);

  const sortableColumns = useMemo(() => {
    const ordered: ColumnConfig[] = [];
    const seen = new Set<string>();
    const pushCol = (id: string) => {
      if (id === "primary_display_image" || seen.has(id)) return;
      const col = collection.columns[id];
      if (col) {
        seen.add(id);
        ordered.push(col);
      }
    };
    pushCol("name");
    pushCol("manufacturer");
    for (const id of selectedColumnIds) {
      pushCol(id);
    }
    for (const id of Object.keys(collection.columns)) {
      pushCol(id);
    }
    return ordered;
  }, [collection.columns, selectedColumnIds]);

  const sortArray = useMemo(() => {
    if (!sortField) return [];
    return sortDir === "desc" ? [`^${sortField}`] : [sortField];
  }, [sortField, sortDir]);

  const handleFilterApply = (q: string) => {
    const nextParams = new URLSearchParams(searchParams);
    const trimmed = q.trim();
    if (trimmed) {
      nextParams.set("filter", trimmed);
    } else {
      nextParams.delete("filter");
    }
    setSearchParams(nextParams);
  };

  const handlePropertiesChange = (newProps: string[]) => {
    const nextParams = new URLSearchParams(searchParams);
    const isDefault =
      newProps.length === defaultSpecColumnIds.length &&
      newProps.every((c, i) => c === defaultSpecColumnIds[i]);
    if (isDefault) {
      nextParams.delete("properties");
      nextParams.delete("columns");
    } else {
      nextParams.set("properties", newProps.join(","));
      nextParams.delete("columns");
    }
    setSearchParams(nextParams);
  };

  const handlePageSizeChange = (newSize: number) => {
    setPageSize(newSize);
    setPageIndex(0);
    setTokenHistory([""]);
  };

  const currentToken = tokenHistory[pageIndex] || "";

  const { data, isLoading, isFetching, error } = useQuery(
    collection.listQuery,
    {
      filter: effectiveFilter,
      sort: sortArray,
      pageSize,
      pageToken: currentToken,
    },
    {
      placeholderData: keepPreviousData,
    },
  );

  const nextPageToken = (data as any)?.nextPageToken;

  const handleNextPage = () => {
    if (pageIndex + 1 < tokenHistory.length && tokenHistory[pageIndex + 1]) {
      setPageIndex((prev) => prev + 1);
      return;
    }
    if (nextPageToken) {
      setTokenHistory((prev) => {
        const next = [...prev];
        next[pageIndex + 1] = nextPageToken;
        return next;
      });
      setPageIndex((prev) => prev + 1);
    }
  };

  const handlePrevPage = () => {
    if (pageIndex > 0) {
      setPageIndex((prev) => prev - 1);
    }
  };

  const handleFirstPage = () => {
    setPageIndex(0);
  };

  const items = useMemo(() => {
    const includeInternal = Boolean(filterQuery && filterQuery.includes("is_internal_only"));
    return collection.getDataList(data, includeInternal);
  }, [collection, data, filterQuery]);
  const count = items.length;
  const startItem = count > 0 ? pageIndex * pageSize + 1 : 0;
  const endItem = pageIndex * pageSize + count;
  const hasNextPage = Boolean(
    nextPageToken || (pageIndex + 1 < tokenHistory.length && tokenHistory[pageIndex + 1]),
  );
  const hasPrevPage = pageIndex > 0;

  const xlSpecColsClass =
    XL_SPEC_GRID_COLS[Math.min(Math.max(activeColumns.length, 1), 5)] || "xl:grid-cols-5";

  return (
    <div data-testid="collection-page-container" className="max-w-[110rem] mx-auto">
      <nav
        aria-label="Breadcrumb"
        className="mb-4 text-sm text-zinc-400 flex items-center gap-1.5 flex-wrap"
      >
        <Link
          to={categoryId ? `/components/${categoryId}` : "/"}
          className="capitalize hover:text-zinc-100 hover:underline transition-colors"
        >
          {categoryId || "Home"}
        </Link>
        <ChevronRight size={14} className="text-zinc-500 shrink-0" aria-hidden="true" />
        <CollectionBadge collection={collection.id} label={collection.name} size="md" />
      </nav>

      <div className="mb-6 flex items-center justify-between">
        <div className="flex items-center gap-3">
          <CollectionIcon
            collection={collection.id}
            size={32}
            className={`${collection.color?.textClass || getCollectionColor(collection.id).textClass} shrink-0`}
          />
          <h1 className="text-3xl font-bold">{collection.name}</h1>
        </div>
      </div>

      {/* Smart Filter Input above the table */}
      <SmartFilterInput
        value={filterQuery}
        onApply={handleFilterApply}
        fields={collection.fields}
        presets={collection.presets}
        error={error ? error.message : null}
      />

      {isLoading && (
        <p className="text-sm text-zinc-500 mb-4">Loading {collection.name.toLowerCase()}...</p>
      )}

      {data && (
        <>
          {/* Catalog Toolbar with Count, Sort, Columns & Internal Toggle */}
          <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-3 mb-4 text-xs text-zinc-500">
            <div className="flex items-center gap-2">
              <span>
                {count > 0 ? (
                  <>
                    Showing{" "}
                    <span className="font-medium text-zinc-300">
                      {startItem}–{endItem}
                    </span>{" "}
                    components
                  </>
                ) : (
                  "0 components found"
                )}
              </span>
              {isFetching && !isLoading && (
                <span className="text-zinc-500 animate-pulse">(updating...)</span>
              )}
            </div>

            <div className="flex items-center flex-wrap gap-2.5 sm:gap-3">
              {/* Toolbar Sort Dropdown */}
              <div className="flex items-center gap-1.5">
                <StyledSelect
                  value={
                    sortField ||
                    (collection.columns["name"] ? "name" : sortableColumns[0]?.id || "")
                  }
                  onChange={(newField) => {
                    const nextParams = new URLSearchParams(searchParams);
                    nextParams.set("sort", sortDir === "desc" ? `^${newField}` : newField);
                    nextParams.delete("dir");
                    nextParams.delete("sort_dir");
                    nextParams.delete("order");
                    setSearchParams(nextParams);
                  }}
                  options={sortableColumns.map((col) => ({
                    value: col.id,
                    label: `Sort: ${col.title}`,
                  }))}
                  ariaLabel="Sort by field"
                  size="sm"
                />
                <button
                  type="button"
                  onClick={() => {
                    const currentField =
                      sortField ||
                      (collection.columns["name"] ? "name" : sortableColumns[0]?.id || "");
                    const nextDir = sortDir === "asc" ? "desc" : "asc";
                    const nextParams = new URLSearchParams(searchParams);
                    nextParams.set("sort", nextDir === "desc" ? `^${currentField}` : currentField);
                    nextParams.delete("dir");
                    nextParams.delete("sort_dir");
                    nextParams.delete("order");
                    setSearchParams(nextParams);
                  }}
                  title={
                    sortDir === "asc"
                      ? "Sort ascending (click for descending)"
                      : "Sort descending (click for ascending)"
                  }
                  aria-label={sortDir === "asc" ? "Sort ascending" : "Sort descending"}
                  className="p-1.5 rounded-lg border border-zinc-800 bg-zinc-900 hover:bg-zinc-800 text-zinc-300 transition-colors shadow-xs cursor-pointer shrink-0"
                >
                  {sortDir === "desc" ? <ArrowDown size={14} /> : <ArrowUp size={14} />}
                </button>
              </div>

              <PropertySelector
                allProperties={selectableProperties}
                selectedPropertyIds={selectedColumnIds}
                defaultPropertyIds={defaultSpecColumnIds}
                onChange={handlePropertiesChange}
              />
            </div>
          </div>

          {items.length === 0 ? (
            <div className="rounded-2xl border border-dashed border-zinc-800 p-8 text-center text-sm text-zinc-500">
              No {collection.name.toLowerCase()} found matching the active filters.
            </div>
          ) : (
            <div data-testid="catalog-grid" className="grid grid-cols-1 gap-3">
              {items.map((item: any) => (
                <CatalogRowCard
                  key={item.id}
                  item={item}
                  productUrl={`/${getCollectionPath(collection)}/${item.id}`}
                  collectionId={collection.id}
                  colTextClass={colColor.textClass}
                  activeColumns={activeColumns}
                  xlSpecColsClass={xlSpecColsClass}
                />
              ))}
            </div>
          )}

          {/* Bottom Pagination Controls */}
          {(count > 0 || pageIndex > 0) && (
            <div className="flex flex-col sm:flex-row items-center justify-between gap-3 mt-6 text-xs text-zinc-400">
              {/* Rows per page selector */}
              <div className="flex items-center gap-2">
                <span>Rows per page:</span>
                <StyledSelect<number>
                  value={pageSize}
                  onChange={(newSize) => handlePageSizeChange(newSize)}
                  options={[
                    { value: 10, label: "10" },
                    { value: 20, label: "20" },
                    { value: 50, label: "50" },
                    { value: 100, label: "100" },
                  ]}
                  ariaLabel="Rows per page"
                  size="sm"
                />
                <span className="text-zinc-700">|</span>
                <span>
                  Showing {startItem}–{endItem}
                </span>
              </div>

              {/* Page navigation controls */}
              <div className="flex items-center gap-1.5">
                <span className="mr-2 font-medium text-zinc-300">Page {pageIndex + 1}</span>
                <button
                  type="button"
                  onClick={handleFirstPage}
                  disabled={!hasPrevPage || isFetching}
                  title="First Page"
                  className="p-1.5 rounded border border-zinc-800 bg-zinc-900 hover:bg-zinc-800 disabled:opacity-30 disabled:cursor-not-allowed transition-colors cursor-pointer text-zinc-300 shadow-xs"
                >
                  <ChevronsLeft size={14} />
                </button>
                <button
                  type="button"
                  onClick={handlePrevPage}
                  disabled={!hasPrevPage || isFetching}
                  title="Previous Page"
                  className="flex items-center gap-1 px-2.5 py-1.5 rounded border border-zinc-800 bg-zinc-900 hover:bg-zinc-800 disabled:opacity-30 disabled:cursor-not-allowed transition-colors cursor-pointer text-zinc-300 shadow-xs"
                >
                  <ChevronLeft size={14} />
                  <span>Previous</span>
                </button>
                <button
                  type="button"
                  onClick={handleNextPage}
                  disabled={!hasNextPage || isFetching}
                  title="Next Page"
                  className="flex items-center gap-1 px-2.5 py-1.5 rounded border border-zinc-800 bg-zinc-900 hover:bg-zinc-800 disabled:opacity-30 disabled:cursor-not-allowed transition-colors cursor-pointer text-zinc-300 shadow-xs"
                >
                  <span>Next</span>
                  <ChevronRight size={14} />
                </button>
              </div>
            </div>
          )}
        </>
      )}
    </div>
  );
}

export function CollectionPage() {
  const { categoryId, collectionId } = useParams();
  const collection = getHardwareCollection(collectionId);

  if (!collection) {
    return (
      <div data-testid="collection-page-container" className="max-w-[110rem] mx-auto">
        <nav
          aria-label="Breadcrumb"
          className="mb-4 text-sm text-zinc-400 flex items-center gap-1.5 flex-wrap"
        >
          <Link
            to={`/components/${categoryId}`}
            className="capitalize hover:text-zinc-100 hover:underline transition-colors"
          >
            {categoryId}
          </Link>
          <ChevronRight size={14} className="text-zinc-500 shrink-0" aria-hidden="true" />
          <CollectionBadge
            collection={collectionId}
            label={collectionId?.replace(/[-_]/g, " ")}
            size="md"
          />
        </nav>
        <div className="flex items-center gap-3 mb-6">
          <CollectionIcon
            collection={collectionId}
            size={32}
            className={`${getCollectionColor(collectionId).textClass} shrink-0`}
          />
          <h1 className="text-3xl font-bold capitalize">{collectionId?.replace(/[-_]/g, " ")}</h1>
        </div>
        <p className="text-zinc-500">
          Implementation for {collectionId?.replace(/[-_]/g, " ")} coming soon.
        </p>
      </div>
    );
  }

  return (
    <CollectionTableView
      key={collection.id}
      categoryId={categoryId || "hardware"}
      collection={collection}
    />
  );
}
