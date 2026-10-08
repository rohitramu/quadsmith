import { useParams, Link } from "react-router-dom";
import { useQuery } from "@connectrpc/connect-query";
import { getOption } from "@bufbuild/protobuf";
import { default_columns } from "../gen/quadsmith/_common_pb";
import {
  ArrowUp,
  ArrowDown,
  ArrowUpDown,
  Columns3,
  GripVertical,
  ChevronUp,
  ChevronDown,
  Plus,
  ChevronLeft,
  ChevronRight,
  ChevronsLeft,
} from "lucide-react";
import { useState, useMemo, useRef, useEffect } from "react";
import { keepPreviousData } from "@tanstack/react-query";
import { SmartFilterInput } from "../components/SmartFilterInput";
import {
  getHardwareCollection,
  type HardwareCollectionDef,
  type ColumnConfig,
} from "../lib/hardwareCollections";

function ColumnHeader({
  title,
  field,
  sortField,
  sortDir,
  onSortToggle,
}: {
  title: string;
  field: string;
  sortField: string | null;
  sortDir: "asc" | "desc";
  onSortToggle: (field: string) => void;
}) {
  return (
    <div className="flex items-center gap-1.5">
      <span>{title}</span>
      <button
        type="button"
        onClick={() => onSortToggle(field)}
        title={`Sort by ${title}`}
        className={`p-1 rounded transition-colors cursor-pointer ${
          sortField === field
            ? "text-blue-600 dark:text-blue-400 bg-blue-50 dark:bg-blue-950/50"
            : "hover:bg-zinc-200 dark:hover:bg-zinc-800 text-zinc-400"
        }`}
      >
        {sortField === field && sortDir === "asc" ? (
          <ArrowUp size={13} />
        ) : sortField === field && sortDir === "desc" ? (
          <ArrowDown size={13} />
        ) : (
          <ArrowUpDown size={13} />
        )}
      </button>
    </div>
  );
}

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
              Columns <span className="font-normal text-zinc-400">({visibleColumns.length} visible)</span>
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
                  <span title="Drag to reorder" className="cursor-grab active:cursor-grabbing shrink-0 flex items-center">
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
                    title={visibleColumns.length <= 1 ? "At least one column must be visible" : "Hide column"}
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
                    <Plus size={12} className="text-zinc-400 group-hover:text-zinc-600 dark:group-hover:text-zinc-200 shrink-0" />
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

function CollectionTableView({
  categoryId,
  collection,
}: {
  categoryId: string;
  collection: HardwareCollectionDef;
}) {
  const [filterQuery, setFilterQuery] = useState("");
  const [sortField, setSortField] = useState<string | null>(null);
  const [sortDir, setSortDir] = useState<"asc" | "desc">("asc");
  const [pageSize, setPageSize] = useState<number>(20);
  const [pageIndex, setPageIndex] = useState<number>(0);
  const [tokenHistory, setTokenHistory] = useState<string[]>([""]);

  // Read default columns from protobuf option if available, otherwise collection defaults
  const protoDefaultCols = useMemo(() => {
    try {
      const protoCols = getOption(collection.schema, default_columns);
      if (protoCols && protoCols.length > 0) {
        return protoCols;
      }
    } catch {
      // Fallback if option not present
    }
    return collection.defaultColumnIds;
  }, [collection]);

  const [selectedColumnIds, setSelectedColumnIds] = useState<string[]>(protoDefaultCols);

  const activeColumns = useMemo(() => {
    return selectedColumnIds
      .map((colKey) => collection.columns[colKey])
      .filter((c): c is ColumnConfig => !!c);
  }, [selectedColumnIds, collection]);

  const sortArray = useMemo(() => {
    if (!sortField) return [];
    return sortDir === "desc" ? [`^${sortField}`] : [sortField];
  }, [sortField, sortDir]);

  const handleSortToggle = (field: string) => {
    setPageIndex(0);
    setTokenHistory([""]);
    if (sortField === field) {
      if (sortDir === "asc") setSortDir("desc");
      else setSortField(null);
    } else {
      setSortField(field);
      setSortDir("asc");
    }
  };

  const handleFilterApply = (q: string) => {
    setFilterQuery(q);
    setPageIndex(0);
    setTokenHistory([""]);
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
      filter: filterQuery,
      sort: sortArray,
      pageSize,
      pageToken: currentToken,
    },
    {
      placeholderData: keepPreviousData,
    }
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

  const items = useMemo(() => collection.getDataList(data), [collection, data]);
  const count = items.length;
  const startItem = count > 0 ? pageIndex * pageSize + 1 : 0;
  const endItem = pageIndex * pageSize + count;
  const hasNextPage = Boolean(
    nextPageToken || (pageIndex + 1 < tokenHistory.length && tokenHistory[pageIndex + 1])
  );
  const hasPrevPage = pageIndex > 0;

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
        <ChevronRight size={14} className="text-zinc-400 dark:text-zinc-500 shrink-0" aria-hidden="true" />
        <span className="text-zinc-900 dark:text-zinc-100 font-medium capitalize" aria-current="page">
          {collection.name}
        </span>
      </nav>

      <div className="mb-6 flex items-center justify-between">
        <h1 className="text-3xl font-bold">{collection.name}</h1>
      </div>

      {/* Smart Filter Input above the table */}
      <SmartFilterInput
        value={filterQuery}
        onApply={handleFilterApply}
        fields={collection.fields}
        presets={collection.presets}
        error={error ? error.message : null}
      />

      {isLoading && <p className="text-sm text-zinc-500 mb-4">Loading {collection.name.toLowerCase()}...</p>}

      {data && (
        <>
          {/* Table Toolbar with Component Count & Column Selector */}
          <div className="flex items-center justify-between mb-3 text-xs text-zinc-500">
            <div className="flex items-center gap-2">
              <span>
                {count > 0 ? (
                  <>
                    Showing <span className="font-medium text-zinc-700 dark:text-zinc-300">{startItem}–{endItem}</span> components
                  </>
                ) : (
                  "0 components found"
                )}
              </span>
              {isFetching && !isLoading && (
                <span className="text-zinc-400 dark:text-zinc-500 animate-pulse">(updating...)</span>
              )}
            </div>
            <ColumnSelector
              allColumns={Object.values(collection.columns)}
              selectedColumnIds={selectedColumnIds}
              defaultColumnIds={protoDefaultCols}
              onChange={setSelectedColumnIds}
            />
          </div>

          <div className="overflow-x-auto rounded-lg border border-zinc-200 dark:border-zinc-800 bg-white dark:bg-zinc-900 shadow-xs">
            <table className="w-full text-left text-sm text-zinc-600 dark:text-zinc-400">
              <thead className="bg-zinc-50 dark:bg-zinc-950/50 text-xs uppercase font-semibold text-zinc-500 border-b border-zinc-200 dark:border-zinc-800">
                <tr>
                  {activeColumns.map((col) => (
                    <th key={col.id} className="px-4 py-3">
                      <ColumnHeader
                        title={col.title}
                        field={col.id}
                        sortField={sortField}
                        sortDir={sortDir}
                        onSortToggle={handleSortToggle}
                      />
                    </th>
                  ))}
                </tr>
              </thead>
              <tbody className="divide-y divide-zinc-200 dark:divide-zinc-800">
                {items.map((item: any) => (
                  <tr
                    key={item.id}
                    className="hover:bg-zinc-50 dark:hover:bg-zinc-800/50 transition-colors group relative"
                  >
                    {activeColumns.map((col, idx) => (
                      <td key={col.id} className="px-4 py-3">
                        {idx === 0 && (
                          <Link
                            to={`/components/${categoryId}/${collection.id}/${item.id}`}
                            className="absolute inset-0 z-10"
                            aria-label={`View ${item.name || item.id}`}
                          />
                        )}
                        {col.renderCell(item)}
                      </td>
                    ))}
                  </tr>
                ))}
                {items.length === 0 && (
                  <tr>
                    <td colSpan={activeColumns.length} className="px-4 py-8 text-center text-zinc-500">
                      No {collection.name.toLowerCase()} found matching the active filters.
                    </td>
                  </tr>
                )}
              </tbody>
            </table>
          </div>

          {/* Bottom Pagination Controls */}
          {(count > 0 || pageIndex > 0) && (
            <div className="flex flex-col sm:flex-row items-center justify-between gap-3 mt-4 text-xs text-zinc-500 dark:text-zinc-400">
              {/* Rows per page selector */}
              <div className="flex items-center gap-2">
                <span>Rows per page:</span>
                <select
                  value={pageSize}
                  onChange={(e) => handlePageSizeChange(Number(e.target.value))}
                  className="bg-white dark:bg-zinc-900 border border-zinc-200 dark:border-zinc-800 rounded px-2 py-1 text-zinc-700 dark:text-zinc-300 font-medium focus:outline-none focus:ring-1 focus:ring-blue-500 cursor-pointer shadow-xs"
                >
                  <option value={10}>10</option>
                  <option value={20}>20</option>
                  <option value={50}>50</option>
                  <option value={100}>100</option>
                </select>
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
          <ChevronRight size={14} className="text-zinc-400 dark:text-zinc-500 shrink-0" aria-hidden="true" />
          <span className="text-zinc-900 dark:text-zinc-100 font-medium capitalize" aria-current="page">
            {collectionId?.replace(/[-_]/g, " ")}
          </span>
        </nav>
        <p className="text-zinc-500">Implementation for {collectionId?.replace(/[-_]/g, " ")} coming soon.</p>
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
