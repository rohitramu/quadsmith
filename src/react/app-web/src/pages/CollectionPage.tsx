import { useParams, Link, useSearchParams } from "react-router-dom";
import { useQuery } from "@connectrpc/connect-query";
import { getOption } from "@bufbuild/protobuf";
import { frontend as frontendOpt } from "@quadsmith/shared/gen/quadsmith/common/common_pb";
import {
  ArrowUp,
  ArrowDown,
  Columns3,
  GripVertical,
  ChevronUp,
  ChevronDown,
  Plus,
  ChevronLeft,
  ChevronRight,
  ChevronsLeft,
  ExternalLink,
} from "lucide-react";
import { useState, useMemo, useRef, useEffect } from "react";
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
import { ComponentFormModal } from "../components/ComponentFormModal";
import { StyledSelect } from "../components/StyledSelect";

function ColumnSelector({
  allColumns,
  selectedColumnIds,
  defaultColumnIds,
  onChange,
}: {
  allColumns: ColumnConfig[];
  selectedColumnIds: string[];
  defaultColumnIds: string[];
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

  const columnMap = useMemo(() => {
    return new Map(allColumns.map((col) => [col.id, col]));
  }, [allColumns]);

  // Active/Visible columns in their current order
  const visibleColumns = useMemo(() => {
    return selectedColumnIds
      .map((id) => columnMap.get(id))
      .filter((col): col is ColumnConfig => !!col);
  }, [selectedColumnIds, columnMap]);

  // Hidden columns
  const hiddenColumns = useMemo(() => {
    return allColumns.filter((col) => !selectedColumnIds.includes(col.id));
  }, [allColumns, selectedColumnIds]);

  const moveColumn = (index: number, direction: -1 | 1) => {
    const targetIndex = index + direction;
    if (targetIndex < 0 || targetIndex >= selectedColumnIds.length) return;
    const next = [...selectedColumnIds];
    const [moved] = next.splice(index, 1);
    next.splice(targetIndex, 0, moved);
    onChange(next);
  };

  const hideColumn = (id: string) => {
    if (selectedColumnIds.length <= 1) return; // Keep at least one column visible
    onChange(selectedColumnIds.filter((colId) => colId !== id));
  };

  const showColumn = (id: string) => {
    onChange([...selectedColumnIds, id]);
  };

  const resetToDefault = () => {
    onChange(defaultColumnIds);
  };

  return (
    <div className="relative inline-block" ref={dropdownRef}>
      <button
        type="button"
        onClick={() => setIsOpen(!isOpen)}
        className="flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium bg-white dark:bg-zinc-900 border border-zinc-200 dark:border-zinc-800 rounded-lg text-zinc-700 dark:text-zinc-300 hover:bg-zinc-50 dark:hover:bg-zinc-800 transition-colors shadow-xs cursor-pointer"
        title="Customize and reorder visible columns"
      >
        <Columns3 size={14} />
        <span>Columns ({selectedColumnIds.length})</span>
      </button>

      {isOpen && (
        <div className="absolute right-0 top-full mt-1.5 w-72 bg-white dark:bg-zinc-900 border border-zinc-200 dark:border-zinc-800 rounded-lg shadow-xl z-50 p-2.5 text-xs">
          <div className="flex items-center justify-between px-2 py-1.5 border-b border-zinc-100 dark:border-zinc-800 mb-2">
            <span className="font-semibold text-zinc-800 dark:text-zinc-200">
              Columns{" "}
              <span className="font-normal text-zinc-400">({visibleColumns.length} visible)</span>
            </span>
            <button
              type="button"
              onClick={resetToDefault}
              className="text-[11px] text-blue-600 dark:text-blue-400 hover:underline cursor-pointer"
            >
              Reset to Default
            </button>
          </div>

          {/* Visible Columns (Reorderable) */}
          <div className="mb-2">
            <div className="px-2 py-0.5 text-[10px] font-semibold text-zinc-400 dark:text-zinc-500 uppercase tracking-wider flex justify-between items-center">
              <span>Visible (drag or arrows)</span>
            </div>
            <div className="space-y-1 mt-1 max-h-56 overflow-y-auto">
              {visibleColumns.map((col, idx) => (
                <div
                  key={col.id}
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
                      const next = [...selectedColumnIds];
                      next.splice(idx, 0, hiddenId);
                      onChange(next);
                      setDraggedIndex(null);
                      return;
                    }
                    if (draggedIndex !== null && draggedIndex !== idx) {
                      const next = [...selectedColumnIds];
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
                      ? "opacity-30 bg-zinc-100 dark:bg-zinc-800"
                      : dragOverIndex === idx
                        ? "border-t-2 border-blue-500 bg-blue-50/50 dark:bg-blue-950/30"
                        : "hover:bg-zinc-50 dark:hover:bg-zinc-800/60 bg-zinc-50/50 dark:bg-zinc-900/50 border border-zinc-100 dark:border-zinc-800/70"
                  }`}
                >
                  <span
                    title="Drag to reorder"
                    className="cursor-grab active:cursor-grabbing shrink-0 flex items-center"
                  >
                    <GripVertical
                      size={13}
                      className="text-zinc-400 group-hover:text-zinc-600 dark:group-hover:text-zinc-200"
                    />
                  </span>
                  <input
                    type="checkbox"
                    checked={true}
                    disabled={visibleColumns.length <= 1}
                    onChange={() => hideColumn(col.id)}
                    className="rounded border-zinc-300 dark:border-zinc-700 text-blue-600 focus:ring-blue-500 cursor-pointer disabled:opacity-40"
                    title={
                      visibleColumns.length <= 1
                        ? "At least one column must be visible"
                        : "Hide column"
                    }
                  />
                  <span className="font-medium text-zinc-800 dark:text-zinc-200 flex-1 truncate select-none">
                    {col.title}
                  </span>
                  <div className="flex items-center gap-0.5 shrink-0">
                    <button
                      type="button"
                      disabled={idx === 0}
                      onClick={() => moveColumn(idx, -1)}
                      className="p-0.5 rounded text-zinc-400 hover:text-zinc-700 dark:hover:text-zinc-200 hover:bg-zinc-200 dark:hover:bg-zinc-700 disabled:opacity-20 disabled:hover:bg-transparent cursor-pointer disabled:cursor-not-allowed"
                      title="Move up / left"
                    >
                      <ChevronUp size={13} />
                    </button>
                    <button
                      type="button"
                      disabled={idx === visibleColumns.length - 1}
                      onClick={() => moveColumn(idx, 1)}
                      className="p-0.5 rounded text-zinc-400 hover:text-zinc-700 dark:hover:text-zinc-200 hover:bg-zinc-200 dark:hover:bg-zinc-700 disabled:opacity-20 disabled:hover:bg-transparent cursor-pointer disabled:cursor-not-allowed"
                      title="Move down / right"
                    >
                      <ChevronDown size={13} />
                    </button>
                  </div>
                </div>
              ))}
            </div>
          </div>

          {/* Hidden Columns */}
          {hiddenColumns.length > 0 && (
            <div className="border-t border-zinc-100 dark:border-zinc-800 pt-2">
              <div className="px-2 py-0.5 text-[10px] font-semibold text-zinc-400 dark:text-zinc-500 uppercase tracking-wider">
                <span>Hidden Columns (click to show)</span>
              </div>
              <div className="space-y-0.5 mt-1 max-h-36 overflow-y-auto">
                {hiddenColumns.map((col) => (
                  <div
                    key={col.id}
                    draggable
                    onDragStart={(e) => {
                      e.dataTransfer.effectAllowed = "copyMove";
                      e.dataTransfer.setData("text/quadsmith-hidden-id", col.id);
                    }}
                    onClick={() => showColumn(col.id)}
                    className="flex items-center gap-2 px-2 py-1.5 rounded hover:bg-zinc-50 dark:hover:bg-zinc-800/60 cursor-pointer select-none text-zinc-500 dark:text-zinc-400 group"
                    title="Click to add or drag into visible list"
                  >
                    <input
                      type="checkbox"
                      checked={false}
                      onChange={() => showColumn(col.id)}
                      className="rounded border-zinc-300 dark:border-zinc-700 text-blue-600 focus:ring-blue-500 cursor-pointer"
                    />
                    <span className="flex-1 truncate group-hover:text-zinc-700 dark:group-hover:text-zinc-300">
                      {col.title}
                    </span>
                    <Plus
                      size={12}
                      className="text-zinc-400 group-hover:text-zinc-600 dark:group-hover:text-zinc-200 shrink-0"
                    />
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

  const selectedColumnIds = useMemo(() => {
    const colsParam = searchParams.get("columns");
    if (!colsParam) {
      return protoDefaultCols;
    }
    const parsed = colsParam
      .split(",")
      .map((c) => c.trim())
      .filter((c) => c in collection.columns);
    return parsed.length > 0 ? parsed : protoDefaultCols;
  }, [searchParams, protoDefaultCols, collection.columns]);

  const filterQuery = searchParams.get("filter") || "";

  const showInternal =
    searchParams.get("show_internal") === "true" || searchParams.get("internal") === "true";

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
  const [isCreateModalOpen, setIsCreateModalOpen] = useState<boolean>(false);
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
    if (!hasInternalField || showInternal) {
      return filterQuery;
    }
    if (filterQuery && filterQuery.includes("is_internal_only")) {
      return filterQuery;
    }
    if (!filterQuery) {
      return "is_internal_only == false";
    }
    return `(${filterQuery}) && is_internal_only == false`;
  }, [hasInternalField, showInternal, filterQuery]);

  const activeColumns = useMemo(() => {
    return selectedColumnIds
      .map((colKey) => collection.columns[colKey])
      .filter((c): c is ColumnConfig => !!c);
  }, [selectedColumnIds, collection]);

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

  const handleColumnsChange = (newCols: string[]) => {
    const nextParams = new URLSearchParams(searchParams);
    const isDefault =
      newCols.length === protoDefaultCols.length &&
      newCols.every((c, i) => c === protoDefaultCols[i]);
    if (isDefault) {
      nextParams.delete("columns");
    } else {
      nextParams.set("columns", newCols.join(","));
    }
    setSearchParams(nextParams);
  };

  const handleShowInternalChange = (checked: boolean) => {
    const nextParams = new URLSearchParams(searchParams);
    if (checked) {
      nextParams.set("show_internal", "true");
    } else {
      nextParams.delete("show_internal");
      nextParams.delete("internal");
    }
    setSearchParams(nextParams);
  };

  const handlePageSizeChange = (newSize: number) => {
    setPageSize(newSize);
    setPageIndex(0);
    setTokenHistory([""]);
  };

  const currentToken = tokenHistory[pageIndex] || "";

  const { data, isLoading, isFetching, error, refetch } = useQuery(
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
    const includeInternal =
      showInternal || Boolean(filterQuery && filterQuery.includes("is_internal_only"));
    return collection.getDataList(data, includeInternal);
  }, [collection, data, showInternal, filterQuery]);
  const count = items.length;
  const startItem = count > 0 ? pageIndex * pageSize + 1 : 0;
  const endItem = pageIndex * pageSize + count;
  const hasNextPage = Boolean(
    nextPageToken || (pageIndex + 1 < tokenHistory.length && tokenHistory[pageIndex + 1]),
  );
  const hasPrevPage = pageIndex > 0;

  return (
    <div>
      <nav
        aria-label="Breadcrumb"
        className="mb-4 text-sm text-zinc-500 dark:text-zinc-400 flex items-center gap-1.5 flex-wrap"
      >
        <Link
          to={categoryId ? `/components/${categoryId}` : "/"}
          className="capitalize hover:text-zinc-900 dark:hover:text-zinc-100 hover:underline transition-colors"
        >
          {categoryId || "Home"}
        </Link>
        <ChevronRight
          size={14}
          className="text-zinc-400 dark:text-zinc-500 shrink-0"
          aria-hidden="true"
        />
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

        <button
          type="button"
          onClick={() => setIsCreateModalOpen(true)}
          className="inline-flex items-center gap-1.5 px-3.5 py-1.5 rounded-xl bg-emerald-600 hover:bg-emerald-500 text-white text-xs sm:text-sm font-semibold shadow-xs transition-colors cursor-pointer"
          aria-label="Add Component"
        >
          <Plus size={16} />
          <span>+ Add New</span>
        </button>
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
                    <span className="font-medium text-zinc-700 dark:text-zinc-300">
                      {startItem}–{endItem}
                    </span>{" "}
                    components
                  </>
                ) : (
                  "0 components found"
                )}
              </span>
              {isFetching && !isLoading && (
                <span className="text-zinc-400 dark:text-zinc-500 animate-pulse">
                  (updating...)
                </span>
              )}
            </div>

            <div className="flex items-center flex-wrap gap-2.5 sm:gap-3">
              {/* Toolbar Sort Dropdown */}
              <div className="flex items-center gap-1.5">
                <StyledSelect
                  value={
                    sortField || (collection.columns["name"] ? "name" : activeColumns[0]?.id || "")
                  }
                  onChange={(newField) => {
                    const nextParams = new URLSearchParams(searchParams);
                    nextParams.set("sort", sortDir === "desc" ? `^${newField}` : newField);
                    nextParams.delete("dir");
                    nextParams.delete("sort_dir");
                    nextParams.delete("order");
                    setSearchParams(nextParams);
                  }}
                  options={activeColumns.map((col) => ({
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
                      (collection.columns["name"] ? "name" : activeColumns[0]?.id || "");
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
                  className="p-1.5 rounded-lg border border-zinc-200 dark:border-zinc-800 bg-white dark:bg-zinc-900 hover:bg-zinc-100 dark:hover:bg-zinc-800 text-zinc-700 dark:text-zinc-300 transition-colors shadow-xs cursor-pointer shrink-0"
                >
                  {sortDir === "desc" ? <ArrowDown size={14} /> : <ArrowUp size={14} />}
                </button>
              </div>

              {hasInternalField && (
                <label className="flex items-center gap-1.5 cursor-pointer text-zinc-600 dark:text-zinc-400 hover:text-zinc-900 dark:hover:text-zinc-200">
                  <input
                    type="checkbox"
                    checked={showInternal}
                    onChange={(e) => handleShowInternalChange(e.target.checked)}
                    className="rounded border-zinc-300 dark:border-zinc-700 text-emerald-600 focus:ring-emerald-500 cursor-pointer"
                  />
                  <span>Show internal</span>
                </label>
              )}

              <ColumnSelector
                allColumns={Object.values(collection.columns)}
                selectedColumnIds={selectedColumnIds}
                defaultColumnIds={protoDefaultCols}
                onChange={handleColumnsChange}
              />
            </div>
          </div>

          {items.length === 0 ? (
            <div className="rounded-2xl border border-dashed border-zinc-200 dark:border-zinc-800 p-8 text-center text-sm text-zinc-500">
              No {collection.name.toLowerCase()} found matching the active filters.
            </div>
          ) : (
            <div
              data-testid="catalog-grid"
              className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-2 2xl:grid-cols-3 gap-4"
            >
              {items.map((item: any) => {
                const productUrl = `/${getCollectionPath(collection)}/${item.id}`;
                return (
                  <article
                    key={item.id}
                    data-testid="catalog-card"
                    className="rounded-2xl border border-zinc-200 dark:border-zinc-800 bg-white dark:bg-zinc-900/90 p-4 sm:p-5 shadow-xs hover:border-zinc-300 dark:hover:border-zinc-700 hover:shadow-md transition-all flex flex-col justify-between"
                  >
                    <div>
                      {/* Header: Thumbnail + Badges/Manufacturer + Name + Community/Internal */}
                      <div className="flex items-start gap-3 pb-3 mb-3 border-b border-zinc-100 dark:border-zinc-800/80">
                        {/* Thumbnail / Icon */}
                        <div
                          className={`w-12 h-12 rounded-xl shrink-0 flex items-center justify-center overflow-hidden border border-zinc-200/80 dark:border-zinc-800 bg-zinc-50 dark:bg-zinc-800/80 ${colColor.textClass}`}
                        >
                          {item.primaryDisplayImage ? (
                            <img
                              src={item.primaryDisplayImage}
                              alt=""
                              className="w-full h-full object-cover"
                            />
                          ) : (
                            <CollectionIcon collection={collection.id} size={22} />
                          )}
                        </div>

                        {/* Title & Metadata */}
                        <div className="flex-1 min-w-0">
                          <div className="flex items-center justify-between gap-1.5 mb-1">
                            <div className="flex items-center gap-1.5 min-w-0 flex-wrap">
                              <CollectionBadge collection={collection.id} size="xs" />
                              {item.manufacturer && (
                                <span className="text-[11px] font-semibold text-zinc-400 dark:text-zinc-500 uppercase tracking-wider truncate">
                                  {item.manufacturer}
                                </span>
                              )}
                            </div>
                            <div className="flex items-center gap-1 shrink-0">
                              {(item.isUserCreated || item.is_user_created) && (
                                <span className="px-2 py-0.5 rounded-full text-[10px] font-semibold bg-emerald-500/10 text-emerald-600 dark:text-emerald-400 border border-emerald-500/20">
                                  Community
                                </span>
                              )}
                              {item.is_internal_only && (
                                <span className="px-2 py-0.5 rounded-full text-[10px] font-semibold bg-amber-500/10 text-amber-600 dark:text-amber-400 border border-amber-500/20">
                                  Internal
                                </span>
                              )}
                            </div>
                          </div>

                          <Link
                            to={productUrl}
                            className="font-bold text-sm sm:text-base text-zinc-900 dark:text-zinc-50 hover:text-emerald-600 dark:hover:text-emerald-400 transition-colors line-clamp-1 block"
                            aria-label={`View ${item.name || item.id}`}
                          >
                            {item.name || item.id}
                          </Link>
                        </div>
                      </div>

                      {/* Spec Grid for active columns */}
                      <div className="grid grid-cols-2 gap-2">
                        {activeColumns
                          .filter((col) => col.id !== "name")
                          .map((col) => (
                            <div
                              key={col.id}
                              className="bg-zinc-50 dark:bg-zinc-800/50 rounded-xl px-2.5 py-1.5 border border-zinc-100 dark:border-zinc-800/80 flex flex-col min-w-0"
                            >
                              <span className="text-[10px] font-medium uppercase tracking-wider text-zinc-400 dark:text-zinc-500 truncate">
                                {col.title}
                              </span>
                              <div className="text-xs font-semibold text-zinc-800 dark:text-zinc-200 truncate mt-0.5">
                                {col.renderCell(item)}
                              </div>
                            </div>
                          ))}
                      </div>

                      {/* Description if present */}
                      {item.description && (
                        <p className="text-xs text-zinc-500 dark:text-zinc-400 line-clamp-2 mt-3 leading-relaxed">
                          {item.description}
                        </p>
                      )}
                    </div>

                    {/* Footer Direct Link & ID */}
                    <div className="mt-3.5 pt-2.5 border-t border-zinc-100 dark:border-zinc-800/80 flex items-center justify-between text-xs">
                      <span className="font-mono text-[11px] text-zinc-400 dark:text-zinc-500 truncate">
                        {item.id}
                      </span>
                      <Link
                        to={productUrl}
                        className="inline-flex items-center gap-1 font-semibold text-emerald-600 dark:text-emerald-400 hover:underline shrink-0"
                      >
                        <span>Details</span>
                        <ExternalLink size={12} />
                      </Link>
                    </div>
                  </article>
                );
              })}
            </div>
          )}

          {/* Bottom Pagination Controls */}
          {(count > 0 || pageIndex > 0) && (
            <div className="flex flex-col sm:flex-row items-center justify-between gap-3 mt-6 text-xs text-zinc-500 dark:text-zinc-400">
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
                <span className="text-zinc-300 dark:text-zinc-700">|</span>
                <span>
                  Showing {startItem}–{endItem}
                </span>
              </div>

              {/* Page navigation controls */}
              <div className="flex items-center gap-1.5">
                <span className="mr-2 font-medium text-zinc-700 dark:text-zinc-300">
                  Page {pageIndex + 1}
                </span>
                <button
                  type="button"
                  onClick={handleFirstPage}
                  disabled={!hasPrevPage || isFetching}
                  title="First Page"
                  className="p-1.5 rounded border border-zinc-200 dark:border-zinc-800 bg-white dark:bg-zinc-900 hover:bg-zinc-50 dark:hover:bg-zinc-800 disabled:opacity-30 disabled:cursor-not-allowed transition-colors cursor-pointer text-zinc-700 dark:text-zinc-300 shadow-xs"
                >
                  <ChevronsLeft size={14} />
                </button>
                <button
                  type="button"
                  onClick={handlePrevPage}
                  disabled={!hasPrevPage || isFetching}
                  title="Previous Page"
                  className="flex items-center gap-1 px-2.5 py-1.5 rounded border border-zinc-200 dark:border-zinc-800 bg-white dark:bg-zinc-900 hover:bg-zinc-50 dark:hover:bg-zinc-800 disabled:opacity-30 disabled:cursor-not-allowed transition-colors cursor-pointer text-zinc-700 dark:text-zinc-300 shadow-xs"
                >
                  <ChevronLeft size={14} />
                  <span>Previous</span>
                </button>
                <button
                  type="button"
                  onClick={handleNextPage}
                  disabled={!hasNextPage || isFetching}
                  title="Next Page"
                  className="flex items-center gap-1 px-2.5 py-1.5 rounded border border-zinc-200 dark:border-zinc-800 bg-white dark:bg-zinc-900 hover:bg-zinc-50 dark:hover:bg-zinc-800 disabled:opacity-30 disabled:cursor-not-allowed transition-colors cursor-pointer text-zinc-700 dark:text-zinc-300 shadow-xs"
                >
                  <span>Next</span>
                  <ChevronRight size={14} />
                </button>
              </div>
            </div>
          )}
        </>
      )}

      <ComponentFormModal
        isOpen={isCreateModalOpen}
        onClose={() => setIsCreateModalOpen(false)}
        collectionId={collection.id}
        onSuccess={() => {
          refetch();
        }}
      />
    </div>
  );
}

export function CollectionPage() {
  const { categoryId, collectionId } = useParams();
  const collection = getHardwareCollection(collectionId);

  if (!collection) {
    return (
      <div>
        <nav
          aria-label="Breadcrumb"
          className="mb-4 text-sm text-zinc-500 dark:text-zinc-400 flex items-center gap-1.5 flex-wrap"
        >
          <Link
            to={`/components/${categoryId}`}
            className="capitalize hover:text-zinc-900 dark:hover:text-zinc-100 hover:underline transition-colors"
          >
            {categoryId}
          </Link>
          <ChevronRight
            size={14}
            className="text-zinc-400 dark:text-zinc-500 shrink-0"
            aria-hidden="true"
          />
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
