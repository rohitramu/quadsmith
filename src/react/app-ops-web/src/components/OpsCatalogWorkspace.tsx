import React, { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { createClient, type Transport } from "@connectrpc/connect";
import {
  AlertCircle,
  AlertTriangle,
  ArrowDown,
  ArrowUp,
  ChevronLeft,
  ChevronRight,
  Copy,
  Edit,
  Loader2,
  Plus,
  RotateCcw,
  Trash2,
  Upload,
  X,
} from "lucide-react";
import {
  CATALOG_CATEGORY_GROUPS,
  CATALOG_COLLECTIONS,
  CollectionIcon,
  SmartFilterInput,
  deriveCelFieldsFromSchema,
  deriveSpecFormFieldsFromSchema,
  getCollectionColor,
  type CatalogCollectionMeta,
  type CatalogSpecFormField,
} from "@quadsmith/shared";

export interface OpsCatalogWorkspaceProps {
  transport: Transport;
  opsBaseUrl: string;
  token: string;
  hasActiveJit: boolean;
  operatorType?: string;
  onFeedback?: (message: string, isError?: boolean) => void;
}

interface ModalState {
  mode: "create" | "clone" | "edit";
  item?: any;
}

export function OpsCatalogWorkspace({
  transport,
  opsBaseUrl,
  token,
  hasActiveJit,
  operatorType,
  onFeedback,
}: OpsCatalogWorkspaceProps) {
  const canMutate = hasActiveJit && operatorType !== "ROBOT";

  const [selectedCollectionId, setSelectedCollectionId] = useState<string>("motors");
  const [includeArchived, setIncludeArchived] = useState<boolean>(false);
  const [celFilter, setCelFilter] = useState<string>("");
  const [sortField, setSortField] = useState<string>("name");
  const [sortDir, setSortDir] = useState<"asc" | "desc">("asc");
  const [pageSize, setPageSize] = useState<number>(25);
  const [pageIndex, setPageIndex] = useState<number>(0);
  const [tokenHistory, setTokenHistory] = useState<string[]>([""]);

  const [items, setItems] = useState<any[]>([]);
  const [nextPageToken, setNextPageToken] = useState<string>("");
  const [isLoading, setIsLoading] = useState<boolean>(false);
  const [fetchError, setFetchError] = useState<string | null>(null);

  const [modalState, setModalState] = useState<ModalState | null>(null);
  const [deleteTarget, setDeleteTarget] = useState<any | null>(null);
  const [isDeleting, setIsDeleting] = useState<boolean>(false);
  const [restoringId, setRestoringId] = useState<string | null>(null);

  const activeCollection: CatalogCollectionMeta = useMemo(() => {
    return (
      CATALOG_COLLECTIONS.find((c) => c.collectionId === selectedCollectionId) ||
      CATALOG_COLLECTIONS[0]
    );
  }, [selectedCollectionId]);

  const celFields = useMemo(
    () => deriveCelFieldsFromSchema(activeCollection.schema),
    [activeCollection.schema],
  );

  const specFormFields = useMemo(
    () => deriveSpecFormFieldsFromSchema(activeCollection.schema),
    [activeCollection.schema],
  );

  const opsClient = useMemo(
    () => createClient(activeCollection.opsService, transport) as Record<string, any>,
    [activeCollection.opsService, transport],
  );

  const currentToken = tokenHistory[pageIndex] || "";

  const loadItems = useCallback(async () => {
    setIsLoading(true);
    setFetchError(null);
    try {
      const sortArr = sortField ? [sortDir === "desc" ? `^${sortField}` : sortField] : [];
      const listFn = opsClient[activeCollection.opsMethods.list];
      const resp = await listFn({
        filter: celFilter,
        sort: sortArr,
        pageSize,
        pageToken: currentToken,
        includeArchived,
      });
      const list = resp?.[activeCollection.listResponseKey] ?? [];
      setItems(list);
      setNextPageToken(resp?.nextPageToken || "");
    } catch (err: unknown) {
      setFetchError(err instanceof Error ? err.message : "Failed to load catalog collection");
      setItems([]);
      setNextPageToken("");
    } finally {
      setIsLoading(false);
    }
  }, [
    opsClient,
    activeCollection,
    celFilter,
    sortField,
    sortDir,
    pageSize,
    currentToken,
    includeArchived,
  ]);

  useEffect(() => {
    void loadItems();
  }, [loadItems]);

  const handleSelectCollection = (colId: string) => {
    setSelectedCollectionId(colId);
    setCelFilter("");
    setSortField("name");
    setSortDir("asc");
    setPageIndex(0);
    setTokenHistory([""]);
  };

  const handleFilterApply = (nextFilter: string) => {
    const trimmed = nextFilter.trim();
    if (trimmed.includes("is_archived == true") && !includeArchived) {
      setIncludeArchived(true);
    }
    setCelFilter(trimmed);
    setPageIndex(0);
    setTokenHistory([""]);
  };

  const handleToggleSort = (key: string) => {
    if (sortField === key) {
      setSortDir((prev) => (prev === "asc" ? "desc" : "asc"));
    } else {
      setSortField(key);
      setSortDir("asc");
    }
    setPageIndex(0);
    setTokenHistory([""]);
  };

  const handleNextPage = () => {
    if (pageIndex + 1 < tokenHistory.length && tokenHistory[pageIndex + 1]) {
      setPageIndex((p) => p + 1);
      return;
    }
    if (nextPageToken) {
      setTokenHistory((prev) => {
        const next = [...prev];
        next[pageIndex + 1] = nextPageToken;
        return next;
      });
      setPageIndex((p) => p + 1);
    }
  };

  const handlePrevPage = () => {
    if (pageIndex > 0) {
      setPageIndex((p) => p - 1);
    }
  };

  const handleConfirmDelete = async () => {
    if (!deleteTarget || !canMutate) return;
    setIsDeleting(true);
    try {
      const targetId = deleteTarget.id || deleteTarget.uuid;
      const deleteFn = opsClient[activeCollection.opsMethods.delete];
      const res = await deleteFn({ id: targetId });
      setDeleteTarget(null);
      if (res?.isArchived) {
        onFeedback?.(
          `Component '${targetId}' has active references and was archived/delisted.`,
          false,
        );
      } else {
        onFeedback?.(`Component '${targetId}' was permanently deleted.`, false);
      }
      await loadItems();
    } catch (err: unknown) {
      onFeedback?.(err instanceof Error ? err.message : "Failed to delete component", true);
    } finally {
      setIsDeleting(false);
    }
  };

  const handleRestore = async (item: any) => {
    if (!canMutate) return;
    const targetId = item.id || item.uuid;
    setRestoringId(targetId);
    try {
      const restoreFn = opsClient[activeCollection.opsMethods.restore];
      await restoreFn({ id: targetId });
      onFeedback?.(`Restored '${targetId}' to the active catalog.`, false);
      await loadItems();
    } catch (err: unknown) {
      onFeedback?.(err instanceof Error ? err.message : "Failed to restore component", true);
    } finally {
      setRestoringId(null);
    }
  };

  const colColor = getCollectionColor(activeCollection.collectionId);
  const hasNextPage = Boolean(
    nextPageToken || (pageIndex + 1 < tokenHistory.length && tokenHistory[pageIndex + 1]),
  );

  return (
    <div data-testid="ops-catalog-workspace" className="grid grid-cols-1 lg:grid-cols-12 gap-5">
      {/* Left Rail: 20 Collections Grouped by Category */}
      <aside className="lg:col-span-3 bg-[#111827] border border-slate-800 rounded-xl p-3.5 space-y-4 self-start">
        <div className="px-2 pt-0.5 pb-2 border-b border-slate-800 flex items-center justify-between">
          <span className="text-xs font-mono font-bold uppercase tracking-wider text-slate-300">
            Catalog Collections
          </span>
          <span className="px-1.5 py-0.5 rounded bg-[#090d16] border border-slate-800 text-[10px] font-mono text-slate-400">
            20 Total
          </span>
        </div>

        {CATALOG_CATEGORY_GROUPS.map((group) => {
          const groupCollections = CATALOG_COLLECTIONS.filter((c) => c.category === group.id);
          return (
            <div key={group.id} className="space-y-1">
              <div className="px-2 text-[10px] font-mono font-bold uppercase tracking-wider text-slate-500 flex items-center justify-between">
                <span>{group.label}</span>
                <span>({groupCollections.length})</span>
              </div>
              <div className="space-y-0.5">
                {groupCollections.map((col) => {
                  const isActive = col.collectionId === activeCollection.collectionId;
                  const iconColor = getCollectionColor(col.collectionId);
                  return (
                    <button
                      key={col.collectionId}
                      type="button"
                      data-testid={`catalog-nav-${col.collectionId}`}
                      onClick={() => handleSelectCollection(col.collectionId)}
                      className={`w-full flex items-center justify-between gap-2 px-2.5 py-1.5 rounded-lg text-left text-xs font-mono transition cursor-pointer ${
                        isActive
                          ? "bg-sky-600/20 text-sky-200 border border-sky-500/40 font-semibold"
                          : "text-slate-400 hover:text-slate-200 hover:bg-slate-800/60 border border-transparent"
                      }`}
                    >
                      <span className="flex items-center gap-2 min-w-0">
                        <CollectionIcon
                          collection={col.collectionId}
                          size={14}
                          className={`shrink-0 ${isActive ? "text-sky-300" : iconColor.textClass}`}
                        />
                        <span className="truncate">{col.pluralLabel}</span>
                      </span>
                      {isActive && !isLoading && (
                        <span className="px-1.5 py-0.5 rounded bg-sky-950/80 border border-sky-500/30 text-[10px] text-sky-300 shrink-0">
                          {items.length}
                        </span>
                      )}
                    </button>
                  );
                })}
              </div>
            </div>
          );
        })}
      </aside>

      {/* Right Panel: Active Collection Workspace */}
      <section className="lg:col-span-9 bg-[#111827] border border-slate-800 rounded-xl p-5 space-y-4">
        {/* Collection Header & JIT-Gated Action Bar */}
        <div className="flex flex-wrap items-start justify-between gap-4 pb-3 border-b border-slate-800">
          <div className="space-y-1">
            <div className="flex flex-wrap items-center gap-2.5">
              <CollectionIcon
                collection={activeCollection.collectionId}
                size={20}
                className={`${colColor.textClass} shrink-0`}
              />
              <h2 className="text-base font-mono font-bold text-slate-100">
                {activeCollection.pluralLabel} Catalog
              </h2>
              <span className="px-2 py-0.5 rounded bg-[#090d16] border border-slate-700 text-[11px] font-mono text-sky-300">
                quadsmith.ops.{activeCollection.opsServiceName}
              </span>
            </div>
            <p className="text-xs font-mono text-slate-400">
              Read-only inspection is open to all operators. Mutations require an active HUMAN JIT
              grant and attribute components internally to @quadsmith.
            </p>
          </div>

          <div className="flex flex-wrap items-center gap-2.5">
            <label className="inline-flex items-center gap-2 px-2.5 py-1.5 rounded-lg bg-[#090d16] border border-slate-700 text-xs font-mono text-slate-300 cursor-pointer select-none">
              <input
                type="checkbox"
                data-testid="catalog-include-archived"
                checked={includeArchived}
                onChange={(e) => {
                  setIncludeArchived(e.target.checked);
                  setPageIndex(0);
                  setTokenHistory([""]);
                }}
                className="rounded border-slate-600 text-sky-500 focus:ring-sky-500 cursor-pointer"
              />
              <span>Include Archived</span>
            </label>

            <button
              type="button"
              data-testid="catalog-refresh-btn"
              onClick={() => void loadItems()}
              disabled={isLoading}
              className="px-2.5 py-1.5 rounded-lg bg-slate-800 hover:bg-slate-700 text-slate-200 border border-slate-700 text-xs font-mono cursor-pointer disabled:opacity-50 whitespace-nowrap"
            >
              {isLoading ? "Loading..." : "↻ Refresh"}
            </button>

            {canMutate ? (
              <button
                type="button"
                data-testid="catalog-add-btn"
                onClick={() => setModalState({ mode: "create" })}
                className="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-emerald-600 hover:bg-emerald-500 text-white text-xs font-mono font-semibold shadow-sm transition cursor-pointer whitespace-nowrap"
              >
                <Plus size={14} />
                <span>+ Add {activeCollection.singularLabel}</span>
              </button>
            ) : (
              <button
                type="button"
                data-testid="catalog-add-btn"
                disabled
                title="Acquire an active HUMAN JIT grant to create catalog components"
                className="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-slate-900 text-slate-500 border border-slate-800 text-xs font-mono cursor-not-allowed whitespace-nowrap"
              >
                <span>🔒 + Add {activeCollection.singularLabel} (JIT Required)</span>
              </button>
            )}
          </div>
        </div>

        {/* CEL Filter Input */}
        <SmartFilterInput
          value={celFilter}
          onApply={handleFilterApply}
          fields={celFields}
          presets={activeCollection.presets}
          error={fetchError}
          inputTestId="catalog-cel-input"
          helpTestId="catalog-cel-help"
        />

        {/* Component Data Table */}
        <div className="overflow-x-auto border border-slate-800 rounded-lg bg-[#090d16]">
          <table className="w-full text-left border-collapse text-xs font-mono">
            <thead>
              <tr className="border-b border-slate-800 bg-[#0d1424] text-slate-400 uppercase text-[11px]">
                <th className="py-2.5 px-3">
                  <button
                    type="button"
                    onClick={() => handleToggleSort("id")}
                    className="inline-flex items-center gap-1 hover:text-slate-200 cursor-pointer"
                  >
                    <span>ID Slug</span>
                    {sortField === "id" &&
                      (sortDir === "asc" ? <ArrowUp size={12} /> : <ArrowDown size={12} />)}
                  </button>
                </th>
                <th className="py-2.5 px-3">
                  <button
                    type="button"
                    onClick={() => handleToggleSort("name")}
                    className="inline-flex items-center gap-1 hover:text-slate-200 cursor-pointer"
                  >
                    <span>Manufacturer &amp; Name</span>
                    {sortField === "name" &&
                      (sortDir === "asc" ? <ArrowUp size={12} /> : <ArrowDown size={12} />)}
                  </button>
                </th>
                {activeCollection.specColumns.map((col) => (
                  <th key={col.key} className="py-2.5 px-3 whitespace-nowrap">
                    <button
                      type="button"
                      onClick={() => handleToggleSort(col.sortKey)}
                      className="inline-flex items-center gap-1 hover:text-slate-200 cursor-pointer"
                    >
                      <span>{col.label}</span>
                      {sortField === col.sortKey &&
                        (sortDir === "asc" ? <ArrowUp size={12} /> : <ArrowDown size={12} />)}
                    </button>
                  </th>
                ))}
                <th className="py-2.5 px-3">Status</th>
                <th className="py-2.5 px-3 text-right">Actions</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-slate-800/80">
              {isLoading && items.length === 0 ? (
                <tr>
                  <td
                    colSpan={4 + activeCollection.specColumns.length}
                    className="py-10 text-center text-slate-400"
                  >
                    <Loader2 size={18} className="animate-spin inline-block mr-2 text-sky-400" />
                    Loading {activeCollection.pluralLabel.toLowerCase()}...
                  </td>
                </tr>
              ) : items.length === 0 ? (
                <tr>
                  <td
                    colSpan={4 + activeCollection.specColumns.length}
                    className="py-10 text-center text-slate-500"
                  >
                    No {activeCollection.pluralLabel.toLowerCase()} found matching current filter.
                  </td>
                </tr>
              ) : (
                items.map((item) => {
                  const isArchived = Boolean(item.isArchived);
                  const isInternal = Boolean(item.isInternalOnly);
                  const rowId = item.id || item.uuid;
                  return (
                    <tr
                      key={rowId}
                      data-testid={`catalog-row-${rowId}`}
                      className={`hover:bg-slate-900/60 transition ${
                        isArchived ? "opacity-75 bg-amber-950/10" : ""
                      }`}
                    >
                      <td className="py-2.5 px-3 text-slate-300 select-all max-w-[200px] truncate">
                        {item.id}
                      </td>
                      <td className="py-2.5 px-3">
                        <div className="flex items-center gap-2.5">
                          <div
                            className={`w-8 h-8 rounded-lg shrink-0 flex items-center justify-center overflow-hidden border border-slate-800 bg-slate-900 ${colColor.textClass}`}
                          >
                            {item.primaryDisplayImage ? (
                              <img
                                src={item.primaryDisplayImage}
                                alt=""
                                className="w-full h-full object-cover"
                              />
                            ) : (
                              <CollectionIcon
                                collection={activeCollection.collectionId}
                                size={15}
                              />
                            )}
                          </div>
                          <div className="min-w-0">
                            <div className="font-semibold text-slate-100 truncate max-w-[260px]">
                              {item.name || item.id}
                            </div>
                            {item.manufacturer && (
                              <div className="text-[10px] uppercase tracking-wider text-slate-400 truncate">
                                {item.manufacturer}
                              </div>
                            )}
                          </div>
                        </div>
                      </td>
                      {activeCollection.specColumns.map((col) => (
                        <td key={col.key} className="py-2.5 px-3 text-slate-300 whitespace-nowrap">
                          {col.render(item)}
                        </td>
                      ))}
                      <td className="py-2.5 px-3 whitespace-nowrap">
                        <div className="flex items-center gap-1.5">
                          {isArchived ? (
                            <span className="px-2 py-0.5 rounded bg-amber-500/15 text-amber-300 border border-amber-500/30 text-[10px] font-semibold">
                              ARCHIVED
                            </span>
                          ) : (
                            <span className="px-2 py-0.5 rounded bg-emerald-500/15 text-emerald-300 border border-emerald-500/30 text-[10px] font-semibold">
                              ACTIVE
                            </span>
                          )}
                          {isInternal && (
                            <span className="px-2 py-0.5 rounded bg-sky-500/15 text-sky-300 border border-sky-500/30 text-[10px] font-semibold">
                              INTERNAL
                            </span>
                          )}
                        </div>
                      </td>
                      <td className="py-2.5 px-3 text-right whitespace-nowrap">
                        <div className="inline-flex items-center justify-end gap-1.5">
                          <button
                            type="button"
                            data-testid={`catalog-clone-${rowId}`}
                            disabled={!canMutate}
                            onClick={() => setModalState({ mode: "clone", item })}
                            title={
                              canMutate
                                ? "Create similar component with pre-filled specs"
                                : "Requires active HUMAN JIT grant"
                            }
                            className="inline-flex items-center gap-1 px-2 py-1 rounded border border-slate-700 bg-slate-900 hover:bg-slate-800 text-slate-200 disabled:text-slate-600 disabled:border-slate-800 disabled:cursor-not-allowed cursor-pointer whitespace-nowrap"
                          >
                            <Copy size={12} />
                            <span>Create Similar</span>
                          </button>

                          <button
                            type="button"
                            data-testid={`catalog-edit-${rowId}`}
                            disabled={!canMutate}
                            onClick={() => setModalState({ mode: "edit", item })}
                            title={
                              canMutate ? "Edit component specs" : "Requires active HUMAN JIT grant"
                            }
                            className="inline-flex items-center gap-1 px-2 py-1 rounded border border-slate-700 bg-slate-900 hover:bg-slate-800 text-sky-300 disabled:text-slate-600 disabled:border-slate-800 disabled:cursor-not-allowed cursor-pointer whitespace-nowrap"
                          >
                            <Edit size={12} />
                            <span>Edit</span>
                          </button>

                          {isArchived && (
                            <button
                              type="button"
                              data-testid={`catalog-restore-${rowId}`}
                              disabled={!canMutate || restoringId === rowId}
                              onClick={() => void handleRestore(item)}
                              title={
                                canMutate
                                  ? "Restore archived component to active catalog"
                                  : "Requires active HUMAN JIT grant"
                              }
                              className="inline-flex items-center gap-1 px-2 py-1 rounded border border-emerald-700/60 bg-emerald-950/50 hover:bg-emerald-900/60 text-emerald-300 disabled:text-slate-600 disabled:border-slate-800 disabled:bg-slate-900 disabled:cursor-not-allowed cursor-pointer whitespace-nowrap"
                            >
                              {restoringId === rowId ? (
                                <Loader2 size={12} className="animate-spin" />
                              ) : (
                                <RotateCcw size={12} />
                              )}
                              <span>Restore</span>
                            </button>
                          )}

                          <button
                            type="button"
                            data-testid={`catalog-delete-${rowId}`}
                            disabled={!canMutate}
                            onClick={() => setDeleteTarget(item)}
                            title={
                              canMutate
                                ? "Delete or archive component"
                                : "Requires active HUMAN JIT grant"
                            }
                            className="inline-flex items-center gap-1 px-2 py-1 rounded border border-rose-800/60 bg-rose-950/40 hover:bg-rose-900/50 text-rose-300 disabled:text-slate-600 disabled:border-slate-800 disabled:bg-slate-900 disabled:cursor-not-allowed cursor-pointer whitespace-nowrap"
                          >
                            <Trash2 size={12} />
                            <span>Delete</span>
                          </button>
                        </div>
                      </td>
                    </tr>
                  );
                })
              )}
            </tbody>
          </table>
        </div>

        {/* Pagination Controls */}
        <div className="flex flex-wrap items-center justify-between gap-3 pt-1 text-xs font-mono text-slate-400">
          <div className="flex items-center gap-2">
            <span>Rows per page:</span>
            <select
              aria-label="Catalog rows per page"
              value={pageSize}
              onChange={(e) => {
                setPageSize(Number(e.target.value));
                setPageIndex(0);
                setTokenHistory([""]);
              }}
              className="bg-[#090d16] border border-slate-700 rounded px-2 py-1 text-xs text-slate-200"
            >
              <option value={10}>10</option>
              <option value={25}>25</option>
              <option value={50}>50</option>
              <option value={100}>100</option>
            </select>
            <span className="text-slate-600">·</span>
            <span>Showing {items.length} row(s)</span>
          </div>

          <div className="flex items-center gap-2">
            <span className="text-slate-300">Page {pageIndex + 1}</span>
            <button
              type="button"
              disabled={pageIndex === 0 || isLoading}
              onClick={handlePrevPage}
              className="inline-flex items-center gap-1 px-2.5 py-1 rounded border border-slate-700 bg-[#090d16] hover:bg-slate-800 text-slate-200 disabled:opacity-40 disabled:cursor-not-allowed cursor-pointer"
            >
              <ChevronLeft size={13} />
              <span>Prev</span>
            </button>
            <button
              type="button"
              disabled={!hasNextPage || isLoading}
              onClick={handleNextPage}
              className="inline-flex items-center gap-1 px-2.5 py-1 rounded border border-slate-700 bg-[#090d16] hover:bg-slate-800 text-slate-200 disabled:opacity-40 disabled:cursor-not-allowed cursor-pointer"
            >
              <span>Next</span>
              <ChevronRight size={13} />
            </button>
          </div>
        </div>
      </section>

      {/* Schema-Driven Create / Clone / Edit Modal */}
      {modalState && (
        <OpsComponentFormModal
          collection={activeCollection}
          specFields={specFormFields}
          mode={modalState.mode}
          initialData={modalState.item}
          opsClient={opsClient}
          opsBaseUrl={opsBaseUrl}
          token={token}
          onClose={() => setModalState(null)}
          onSaved={async (savedItem, actionLabel) => {
            setModalState(null);
            onFeedback?.(
              `${actionLabel} ${activeCollection.singularLabel} '${savedItem?.id || savedItem?.name}'.`,
              false,
            );
            await loadItems();
          }}
        />
      )}

      {/* Hybrid Delete / Soft-Archive Confirmation Modal */}
      {deleteTarget && (
        <div
          role="dialog"
          aria-modal="true"
          aria-labelledby="ops-delete-component-title"
          className="fixed inset-0 z-50 bg-black/70 backdrop-blur-xs flex items-center justify-center p-4"
        >
          <div className="bg-[#111827] border border-slate-700 rounded-2xl w-full max-w-md p-6 shadow-2xl space-y-4 font-mono">
            <div className="w-10 h-10 rounded-full bg-amber-950/80 border border-amber-700/50 text-amber-400 flex items-center justify-center">
              <AlertTriangle size={20} />
            </div>

            <div className="space-y-1.5">
              <h3 id="ops-delete-component-title" className="text-base font-bold text-slate-100">
                Delete or Archive {activeCollection.singularLabel}
              </h3>
              <p className="text-xs text-slate-300">
                Target: <span className="text-amber-300 font-semibold">{deleteTarget.id}</span>
              </p>
              <p className="text-xs text-slate-400 leading-relaxed font-sans">
                If this component is referenced by builds, parent AIO hardware, reviews, Q&amp;A, or
                collections, it will be <strong>soft-archived</strong> (delisted from public browse
                while preserving historical build integrity). Otherwise, it will be permanently
                deleted.
              </p>
            </div>

            <div className="flex items-center justify-end gap-2 pt-2">
              <button
                type="button"
                disabled={isDeleting}
                onClick={() => setDeleteTarget(null)}
                className="px-3.5 py-1.5 rounded-lg border border-slate-700 text-xs text-slate-300 hover:bg-slate-800 cursor-pointer"
              >
                Cancel
              </button>
              <button
                type="button"
                data-testid="ops-confirm-delete-btn"
                disabled={isDeleting}
                onClick={() => void handleConfirmDelete()}
                className="inline-flex items-center gap-1.5 px-4 py-1.5 rounded-lg bg-rose-600 hover:bg-rose-500 text-white text-xs font-semibold cursor-pointer disabled:opacity-50"
              >
                {isDeleting && <Loader2 size={13} className="animate-spin" />}
                <span>Confirm Delete</span>
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

function OpsComponentFormModal({
  collection,
  specFields,
  mode,
  initialData,
  opsClient,
  opsBaseUrl,
  token,
  onClose,
  onSaved,
}: {
  collection: CatalogCollectionMeta;
  specFields: CatalogSpecFormField[];
  mode: "create" | "clone" | "edit";
  initialData?: any;
  opsClient: Record<string, any>;
  opsBaseUrl: string;
  token: string;
  onClose: () => void;
  onSaved: (saved: any, actionLabel: string) => Promise<void>;
}) {
  const isEditing = mode === "edit";
  const isClone = mode === "clone";
  const fileInputRef = useRef<HTMLInputElement>(null);

  const [idSlug, setIdSlug] = useState<string>(() => (isEditing ? initialData?.id || "" : ""));
  const [manufacturer, setManufacturer] = useState<string>(() => initialData?.manufacturer || "");
  const [name, setName] = useState<string>(() => initialData?.name || "");
  const [description, setDescription] = useState<string>(() => initialData?.description || "");
  const [primaryDisplayImage, setPrimaryDisplayImage] = useState<string>(
    () => initialData?.primaryDisplayImage || "",
  );
  const [referenceLinks, setReferenceLinks] = useState<{ label: string; url: string }[]>(() =>
    Array.isArray(initialData?.referenceLinks)
      ? initialData.referenceLinks.map((l: any) => ({
          label: l.label || "",
          url: l.url || "",
        }))
      : [],
  );

  const [specValues, setSpecValues] = useState<Record<string, any>>(() => {
    const init: Record<string, any> = {};
    if (!initialData) return init;
    for (const f of specFields) {
      const raw =
        initialData[f.localName] !== undefined
          ? initialData[f.localName]
          : initialData[f.snakeName];
      if (raw === undefined || raw === null) continue;
      if (f.inputType === "string_list" && Array.isArray(raw)) {
        init[f.localName] = raw.join(", ");
      } else {
        init[f.localName] = raw;
      }
    }
    return init;
  });

  const [isUploading, setIsUploading] = useState<boolean>(false);
  const [isSubmitting, setIsSubmitting] = useState<boolean>(false);
  const [formError, setFormError] = useState<string | null>(null);

  useEffect(() => {
    const onKeyDown = (e: KeyboardEvent) => {
      if (e.key === "Escape") onClose();
    };
    window.addEventListener("keydown", onKeyDown);
    return () => window.removeEventListener("keydown", onKeyDown);
  }, [onClose]);

  const handleFileUpload = async (file: File) => {
    if (!file) return;
    setIsUploading(true);
    setFormError(null);
    try {
      const headers: Record<string, string> = {
        "Content-Type": file.type || "application/octet-stream",
        "X-Filename": file.name,
      };
      if (token) {
        headers["Authorization"] = `Bearer ${token}`;
      }
      const base = opsBaseUrl.replace(/\/+$/, "");
      const resp = await fetch(`${base}/api/ops/blobs`, {
        method: "POST",
        headers,
        body: file,
      });
      if (!resp.ok) {
        const errText = await resp.text();
        throw new Error(errText || `Blob upload failed (${resp.status})`);
      }
      const data = await resp.json();
      const blobUrl = data.url || (data.id ? `/blobs/${data.id}` : "");
      if (blobUrl) {
        setPrimaryDisplayImage(blobUrl);
      }
    } catch (err: unknown) {
      setFormError(err instanceof Error ? err.message : "Failed to upload catalog image");
    } finally {
      setIsUploading(false);
    }
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (collection.category !== "software" && !manufacturer.trim()) {
      setFormError("Manufacturer is required.");
      return;
    }
    if (!name.trim()) {
      setFormError("Model name is required.");
      return;
    }

    setIsSubmitting(true);
    setFormError(null);
    try {
      const payload: Record<string, any> = {
        name: name.trim(),
        description: description.trim(),
        primaryDisplayImage: primaryDisplayImage.trim() || undefined,
        referenceLinks: referenceLinks
          .filter((l) => l.url.trim().length > 0)
          .map((l) => ({
            label: l.label.trim(),
            url: l.url.trim(),
          })),
      };

      if (manufacturer.trim()) {
        payload.manufacturer = manufacturer.trim();
      }
      if (idSlug.trim()) {
        payload.id = idSlug.trim();
      }
      if (isEditing && initialData?.uuid) {
        payload.uuid = initialData.uuid;
      }

      for (const field of specFields) {
        const val = specValues[field.localName];
        if (val === undefined || val === "") continue;
        if (field.inputType === "number") {
          const n = Number(val);
          if (!Number.isNaN(n)) {
            payload[field.localName] = n;
          }
        } else if (field.inputType === "enum") {
          payload[field.localName] = Number(val);
        } else if (field.inputType === "boolean") {
          payload[field.localName] = Boolean(val);
        } else if (field.inputType === "string_list") {
          payload[field.localName] = String(val)
            .split(",")
            .map((s) => s.trim())
            .filter(Boolean);
        } else {
          payload[field.localName] = String(val).trim();
        }
      }

      if (isEditing) {
        const updateFn = opsClient[collection.opsMethods.update];
        const saved = await updateFn({ [collection.paramKey]: payload });
        await onSaved(saved, "Updated");
      } else {
        const createFn = opsClient[collection.opsMethods.create];
        const saved = await createFn({ [collection.paramKey]: payload });
        await onSaved(saved, isClone ? "Cloned" : "Created");
      }
    } catch (err: unknown) {
      setFormError(err instanceof Error ? err.message : "Catalog mutation failed");
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <div
      role="dialog"
      aria-modal="true"
      aria-labelledby="ops-component-modal-title"
      className="fixed inset-0 z-50 bg-black/70 backdrop-blur-xs flex items-center justify-center p-4"
    >
      <div className="bg-[#111827] border border-slate-700 rounded-2xl w-full max-w-2xl max-h-[90vh] flex flex-col shadow-2xl overflow-hidden">
        {/* Header */}
        <div className="px-6 py-4 border-b border-slate-800 flex items-center justify-between bg-[#0d1424]">
          <div>
            <h2
              id="ops-component-modal-title"
              className="text-base font-mono font-bold text-slate-100"
            >
              {isClone
                ? `Create Similar ${collection.singularLabel}`
                : isEditing
                  ? `Edit ${collection.singularLabel}`
                  : `Add New ${collection.singularLabel}`}
            </h2>
            <p className="text-xs font-mono text-slate-400 mt-0.5">
              {isClone
                ? `Pre-filled from '${initialData?.id}'. Adjust specs to publish a new catalog entry.`
                : `Managed via quadsmith.ops.${collection.opsServiceName} (attributed to @quadsmith).`}
            </p>
          </div>
          <button
            type="button"
            aria-label="Close modal"
            onClick={onClose}
            className="p-1.5 rounded-lg text-slate-400 hover:text-slate-200 hover:bg-slate-800 cursor-pointer"
          >
            <X size={18} />
          </button>
        </div>

        {/* Form Body */}
        <form onSubmit={handleSubmit} className="flex-1 flex flex-col overflow-hidden">
          <div className="p-6 overflow-y-auto space-y-5 flex-1 text-xs">
            {formError && (
              <div
                data-testid="ops-component-form-error"
                className="p-3 rounded-lg bg-rose-950/60 border border-rose-700/60 text-rose-200 font-mono flex items-start gap-2"
              >
                <AlertCircle size={15} className="shrink-0 mt-0.5" />
                <span>{formError}</span>
              </div>
            )}

            {/* Core Identity */}
            <div className="space-y-3">
              <h3 className="font-mono font-bold uppercase tracking-wider text-slate-400 text-[11px]">
                Catalog Identity
              </h3>
              <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
                <div>
                  <label
                    htmlFor="ops-input-manufacturer"
                    className="block font-mono text-slate-300 mb-1"
                  >
                    Manufacturer {collection.category !== "software" ? "*" : ""}
                  </label>
                  <input
                    id="ops-input-manufacturer"
                    type="text"
                    value={manufacturer}
                    onChange={(e) => setManufacturer(e.target.value)}
                    placeholder="e.g. T-Motor, SpeedyBee, DJI"
                    className="w-full px-3 py-2 rounded-lg border border-slate-700 bg-[#090d16] text-slate-100 font-mono"
                  />
                </div>
                <div>
                  <label
                    htmlFor="ops-input-model-name"
                    className="block font-mono text-slate-300 mb-1"
                  >
                    Model Name *
                  </label>
                  <input
                    id="ops-input-model-name"
                    type="text"
                    required
                    value={name}
                    onChange={(e) => setName(e.target.value)}
                    placeholder="e.g. F60 Pro V, F405 V4"
                    className="w-full px-3 py-2 rounded-lg border border-slate-700 bg-[#090d16] text-slate-100 font-mono"
                  />
                </div>
              </div>

              <div>
                <label htmlFor="ops-input-id-slug" className="block font-mono text-slate-300 mb-1">
                  ID Slug {isEditing ? "(immutable)" : "(optional — auto-generated if blank)"}
                </label>
                <input
                  id="ops-input-id-slug"
                  type="text"
                  value={idSlug}
                  disabled={isEditing}
                  onChange={(e) => setIdSlug(e.target.value)}
                  placeholder="e.g. tmotor-f60-pro-v-1950kv"
                  className="w-full px-3 py-2 rounded-lg border border-slate-700 bg-[#090d16] text-slate-200 font-mono disabled:opacity-60"
                />
              </div>

              <div>
                <label
                  htmlFor="ops-input-description"
                  className="block font-mono text-slate-300 mb-1"
                >
                  Description
                </label>
                <textarea
                  id="ops-input-description"
                  rows={2}
                  value={description}
                  onChange={(e) => setDescription(e.target.value)}
                  placeholder="Official manufacturer technical summary..."
                  className="w-full px-3 py-2 rounded-lg border border-slate-700 bg-[#090d16] text-slate-100"
                />
              </div>
            </div>

            {/* Schema-Driven Technical Specifications */}
            {specFields.length > 0 && (
              <div className="space-y-3 pt-4 border-t border-slate-800">
                <h3 className="font-mono font-bold uppercase tracking-wider text-slate-400 text-[11px]">
                  {collection.singularLabel} Specifications
                </h3>
                <div className="grid grid-cols-1 sm:grid-cols-3 gap-3">
                  {specFields.map((field) => {
                    const val = specValues[field.localName] ?? "";
                    if (field.inputType === "boolean") {
                      return (
                        <label
                          key={field.snakeName}
                          htmlFor={`ops-spec-${field.snakeName}`}
                          className="flex items-center gap-2 pt-5 font-mono text-slate-300 cursor-pointer"
                        >
                          <input
                            id={`ops-spec-${field.snakeName}`}
                            type="checkbox"
                            checked={Boolean(val)}
                            onChange={(e) =>
                              setSpecValues((prev) => ({
                                ...prev,
                                [field.localName]: e.target.checked,
                              }))
                            }
                            className="rounded border-slate-600 text-sky-500"
                          />
                          <span>{field.label}</span>
                        </label>
                      );
                    }

                    if (field.inputType === "enum" && field.enumOptions) {
                      return (
                        <div key={field.snakeName}>
                          <label
                            htmlFor={`ops-spec-${field.snakeName}`}
                            className="block font-mono text-slate-300 mb-1 truncate"
                          >
                            {field.label}
                          </label>
                          <select
                            id={`ops-spec-${field.snakeName}`}
                            value={val === "" ? 0 : Number(val)}
                            onChange={(e) =>
                              setSpecValues((prev) => ({
                                ...prev,
                                [field.localName]: Number(e.target.value),
                              }))
                            }
                            className="w-full px-2.5 py-2 rounded-lg border border-slate-700 bg-[#090d16] text-slate-100 font-mono"
                          >
                            {field.enumOptions.map((opt) => (
                              <option key={opt.value} value={opt.value}>
                                {opt.label}
                              </option>
                            ))}
                          </select>
                        </div>
                      );
                    }

                    return (
                      <div key={field.snakeName}>
                        <label
                          htmlFor={`ops-spec-${field.snakeName}`}
                          className="block font-mono text-slate-300 mb-1 truncate"
                        >
                          {field.label}
                        </label>
                        <input
                          id={`ops-spec-${field.snakeName}`}
                          type={field.inputType === "number" ? "number" : "text"}
                          step={field.inputType === "number" ? "any" : undefined}
                          value={val}
                          onChange={(e) =>
                            setSpecValues((prev) => ({
                              ...prev,
                              [field.localName]: e.target.value,
                            }))
                          }
                          className="w-full px-2.5 py-2 rounded-lg border border-slate-700 bg-[#090d16] text-slate-100 font-mono"
                        />
                      </div>
                    );
                  })}
                </div>
              </div>
            )}

            {/* Catalog Image Upload (/api/ops/blobs) */}
            <div className="space-y-3 pt-4 border-t border-slate-800">
              <h3 className="font-mono font-bold uppercase tracking-wider text-slate-400 text-[11px]">
                Imagery (/api/ops/blobs)
              </h3>
              <div>
                <label
                  htmlFor="ops-input-primary-image"
                  className="block font-mono text-slate-300 mb-1"
                >
                  Primary Display Image URL
                </label>
                <input
                  id="ops-input-primary-image"
                  type="text"
                  value={primaryDisplayImage}
                  onChange={(e) => setPrimaryDisplayImage(e.target.value)}
                  placeholder="/blobs/<id> or https://..."
                  className="w-full px-3 py-2 rounded-lg border border-slate-700 bg-[#090d16] text-slate-100 font-mono"
                />
              </div>

              <input
                ref={fileInputRef}
                type="file"
                accept="image/png,image/jpeg,image/webp"
                className="hidden"
                onChange={(e) => {
                  const file = e.target.files?.[0];
                  if (file) void handleFileUpload(file);
                }}
              />
              <button
                type="button"
                onClick={() => fileInputRef.current?.click()}
                disabled={isUploading}
                className="w-full border border-dashed border-slate-700 hover:border-sky-500 rounded-xl p-3.5 text-center bg-[#090d16]/70 text-slate-300 font-mono cursor-pointer transition"
              >
                {isUploading ? (
                  <span className="inline-flex items-center gap-2 text-sky-300">
                    <Loader2 size={15} className="animate-spin" />
                    Uploading via /api/ops/blobs...
                  </span>
                ) : (
                  <span className="inline-flex items-center gap-2">
                    <Upload size={15} className="text-sky-400" />
                    Upload catalog image via /api/ops/blobs (PNG, JPG, WebP)
                  </span>
                )}
              </button>
            </div>

            {/* Official Manufacturer Reference Links */}
            <div className="space-y-3 pt-4 border-t border-slate-800">
              <div className="flex items-center justify-between">
                <h3 className="font-mono font-bold uppercase tracking-wider text-slate-400 text-[11px]">
                  Official Reference Links
                </h3>
                <button
                  type="button"
                  onClick={() => setReferenceLinks((prev) => [...prev, { label: "", url: "" }])}
                  className="inline-flex items-center gap-1 font-mono text-sky-400 hover:underline cursor-pointer"
                >
                  <Plus size={13} />
                  <span>Add Link</span>
                </button>
              </div>
              {referenceLinks.length === 0 ? (
                <p className="font-mono text-slate-500">No manufacturer spec sheet links added.</p>
              ) : (
                <div className="space-y-2">
                  {referenceLinks.map((link, idx) => (
                    <div key={idx} className="flex items-center gap-2">
                      <input
                        type="text"
                        value={link.label}
                        onChange={(e) =>
                          setReferenceLinks((prev) =>
                            prev.map((item, i) =>
                              i === idx ? { ...item, label: e.target.value } : item,
                            ),
                          )
                        }
                        placeholder="Official Spec Sheet"
                        className="w-1/3 px-2.5 py-1.5 rounded-lg border border-slate-700 bg-[#090d16] text-slate-100 font-mono"
                      />
                      <input
                        type="url"
                        value={link.url}
                        onChange={(e) =>
                          setReferenceLinks((prev) =>
                            prev.map((item, i) =>
                              i === idx ? { ...item, url: e.target.value } : item,
                            ),
                          )
                        }
                        placeholder="https://manufacturer.example/specs"
                        className="flex-1 px-2.5 py-1.5 rounded-lg border border-slate-700 bg-[#090d16] text-slate-100 font-mono"
                      />
                      <button
                        type="button"
                        onClick={() =>
                          setReferenceLinks((prev) => prev.filter((_, i) => i !== idx))
                        }
                        className="p-1.5 text-slate-400 hover:text-rose-400 cursor-pointer"
                      >
                        <Trash2 size={14} />
                      </button>
                    </div>
                  ))}
                </div>
              )}
            </div>
          </div>

          {/* Footer */}
          <div className="px-6 py-3.5 border-t border-slate-800 flex items-center justify-between bg-[#0d1424] font-mono text-xs">
            <span className="text-slate-500">* Required fields</span>
            <div className="flex items-center gap-2">
              <button
                type="button"
                disabled={isSubmitting}
                onClick={onClose}
                className="px-4 py-1.5 rounded-lg border border-slate-700 text-slate-300 hover:bg-slate-800 cursor-pointer"
              >
                Cancel
              </button>
              <button
                type="submit"
                data-testid="ops-component-save-btn"
                disabled={isSubmitting}
                className="inline-flex items-center gap-1.5 px-4 py-1.5 rounded-lg bg-emerald-600 hover:bg-emerald-500 text-white font-semibold cursor-pointer disabled:opacity-50"
              >
                {isSubmitting && <Loader2 size={13} className="animate-spin" />}
                <span>
                  {isClone
                    ? `Create ${collection.singularLabel}`
                    : isEditing
                      ? `Update ${collection.singularLabel}`
                      : `Save ${collection.singularLabel}`}
                </span>
              </button>
            </div>
          </div>
        </form>
      </div>
    </div>
  );
}
