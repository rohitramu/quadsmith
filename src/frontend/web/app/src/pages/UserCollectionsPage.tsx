import { useState } from "react";
import { useParams, Link, useNavigate, Navigate } from "react-router-dom";
import { useQuery, useMutation } from "@connectrpc/connect-query";
import { useQueryClient } from "@tanstack/react-query";
import {
  getCollection,
  listCollectionItems,
  deleteCollection,
  removeCollectionItem,
  updateCollection,
} from "../gen/quadsmith/api/social-SocialService_connectquery";
import type { CollectionItem } from "../gen/quadsmith/api/social_pb";
import { useAuth } from "../context/AuthContext";
import {
  Bookmark,
  Heart,
  Trash2,
  ExternalLink,
  ChevronRight,
  Layers,
  Hammer,
  Cpu,
  Loader2,
  Edit3,
  X,
} from "lucide-react";

export function UserCollectionsPage() {
  const { id: userId, collectionId } = useParams<{ id: string; collectionId?: string }>();
  const { user } = useAuth();
  const queryClient = useQueryClient();
  const navigate = useNavigate();

  // Rename modal state
  const [isRenameOpen, setIsRenameOpen] = useState(false);
  const [renameName, setRenameName] = useState("");
  const [isSavingRename, setIsSavingRename] = useState(false);
  const [isDeleting, setIsDeleting] = useState(false);

  const isOwner = user?.id === userId;

  // Single collection details
  const {
    data: singleCol,
    isLoading: isLoadingSingleCol,
    refetch: refetchSingleCol,
  } = useQuery(getCollection, { id: collectionId || "" }, { enabled: !!collectionId });

  // Items in single collection
  const {
    data: itemsRes,
    isLoading: isLoadingItems,
    refetch: refetchItems,
  } = useQuery(
    listCollectionItems,
    { collectionId: collectionId || "" },
    { enabled: !!collectionId },
  );

  const { mutateAsync: mutateDeleteCollection } = useMutation(deleteCollection);
  const { mutateAsync: mutateRemoveItem } = useMutation(removeCollectionItem);
  const { mutateAsync: mutateUpdateCollection } = useMutation(updateCollection);

  // If accessed without a specific collection ID, redirect to user profile's collections tab
  if (!collectionId) {
    return <Navigate to={`/users/${userId}?tab=collections`} replace />;
  }

  function openRenameModal() {
    if (!singleCol) return;
    setRenameName(singleCol.name);
    setIsRenameOpen(true);
  }

  async function handleSaveRename(e: React.FormEvent) {
    e.preventDefault();
    if (!singleCol || !renameName.trim()) return;
    setIsSavingRename(true);
    try {
      await mutateUpdateCollection({
        collectionId: singleCol.id,
        name: renameName.trim(),
        description: singleCol.description,
      });
      setIsRenameOpen(false);
      await refetchSingleCol();
      await queryClient.invalidateQueries();
    } catch (err: unknown) {
      alert(err instanceof Error ? err.message : "Failed to rename collection");
    } finally {
      setIsSavingRename(false);
    }
  }

  async function handleDeleteCollection(colId?: string) {
    if (!window.confirm("Are you sure you want to delete this collection?")) return;
    const targetColId = colId || collectionId || singleCol?.id || singleCol?.uuid;
    if (!targetColId) return;
    setIsDeleting(true);
    const targetUser = singleCol?.authorUsername || userId || user?.id || "";
    try {
      await mutateDeleteCollection({ collectionId: targetColId });
      navigate(`/users/${targetUser}?tab=collections`, { replace: true });
      await queryClient.invalidateQueries();
    } catch (err: unknown) {
      setIsDeleting(false);
      alert(err instanceof Error ? err.message : "Failed to delete collection");
    }
  }

  async function handleRemoveItem(target: { entityId: string; collection?: string }) {
    if (!collectionId) return;
    try {
      await mutateRemoveItem({
        collectionId,
        entityId: target.entityId,
        collection: target.collection,
      });
      await refetchItems();
      await queryClient.invalidateQueries();
    } catch (err: unknown) {
      alert(err instanceof Error ? err.message : "Failed to remove item");
    }
  }

  if (isLoadingSingleCol || isLoadingItems) {
    return (
      <div className="max-w-5xl mx-auto py-16 flex justify-center text-zinc-400">
        <Loader2 size={24} className="animate-spin text-blue-500" />
      </div>
    );
  }

  if (!singleCol) {
    return (
      <div className="max-w-2xl mx-auto py-16 text-center space-y-4">
        <h2 className="text-xl font-bold">Collection Not Found</h2>
        <Link
          to={`/users/${userId}?tab=collections`}
          className="text-blue-500 hover:underline text-xs"
        >
          ← Back to collections
        </Link>
      </div>
    );
  }

  const items = itemsRes?.items || [];

  return (
    <div className="max-w-5xl mx-auto px-4 py-8 space-y-8 animate-in fade-in duration-200">
      {/* Breadcrumb */}
      <nav
        aria-label="Breadcrumb"
        className="text-sm text-zinc-500 dark:text-zinc-400 flex items-center gap-1.5 flex-wrap"
      >
        <Link to={`/users/${userId}`} className="hover:underline">
          @{userId}
        </Link>
        <ChevronRight size={14} className="text-zinc-400" />
        <Link to={`/users/${userId}?tab=collections`} className="hover:underline">
          Collections
        </Link>
        <ChevronRight size={14} className="text-zinc-400" />
        <span className="font-semibold text-zinc-900 dark:text-zinc-100">{singleCol.name}</span>
      </nav>

      {/* Collection Header Card */}
      <div className="p-8 rounded-2xl border border-zinc-200 dark:border-zinc-800 bg-white dark:bg-zinc-900 shadow-xs space-y-4">
        <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
          <div className="flex items-center gap-4">
            <div className="w-14 h-14 rounded-2xl bg-blue-500/10 text-blue-500 flex items-center justify-center font-bold text-2xl shrink-0">
              {singleCol.isSystemCollection ? (
                <Heart size={28} className="fill-rose-500 text-rose-500" />
              ) : (
                <Bookmark size={28} />
              )}
            </div>
            <div>
              <div className="flex items-center gap-2.5 flex-wrap">
                <h1 className="text-2xl font-black text-zinc-900 dark:text-zinc-50">
                  {singleCol.name}
                </h1>

                {/* Button next to the name to rename the collection */}
                {isOwner && !singleCol.isSystemCollection && (
                  <button
                    type="button"
                    onClick={openRenameModal}
                    className="p-1.5 rounded-lg text-zinc-400 hover:text-blue-500 hover:bg-zinc-100 dark:hover:bg-zinc-800 transition-colors cursor-pointer"
                    title="Rename collection"
                    aria-label="Rename collection"
                  >
                    <Edit3 size={18} />
                  </button>
                )}
              </div>
              <p className="text-xs text-zinc-500 dark:text-zinc-400 mt-0.5">
                Curated by @{singleCol.authorUsername} • {items.length} items
              </p>
            </div>
          </div>

          {/* Delete collection action */}
          {isOwner && !singleCol.isSystemCollection && (
            <div className="flex items-center gap-2 self-start sm:self-auto">
              <button
                type="button"
                onClick={() => handleDeleteCollection(singleCol.id)}
                disabled={isDeleting}
                className="px-3 py-2 rounded-xl text-xs text-rose-500 hover:bg-rose-500/10 border border-rose-500/20 flex items-center gap-1.5 transition-colors cursor-pointer disabled:opacity-50"
                title="Delete collection"
                aria-label="Delete collection"
              >
                {isDeleting ? (
                  <Loader2 size={14} className="animate-spin text-rose-500" />
                ) : (
                  <Trash2 size={14} />
                )}
                <span>{isDeleting ? "Deleting..." : "Delete Collection"}</span>
              </button>
            </div>
          )}
        </div>

        {singleCol.description && (
          <p className="text-sm text-zinc-600 dark:text-zinc-400 pt-2 leading-relaxed">
            {singleCol.description}
          </p>
        )}
      </div>

      {/* Collection Items List */}
      <div className="space-y-4">
        <h2 className="text-base font-bold text-zinc-900 dark:text-zinc-100 flex items-center gap-2">
          <Layers size={18} className="text-blue-500" />
          <span>Items in Collection ({items.length})</span>
        </h2>

        {items.length === 0 ? (
          <div className="p-12 text-center rounded-2xl border border-dashed border-zinc-200 dark:border-zinc-800 text-zinc-400 text-xs">
            This collection has no items yet. Browse builds or components to add items!
          </div>
        ) : (
          <div className="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 gap-4">
            {(items as CollectionItem[]).map((item: CollectionItem) => {
              const isBuild = item.collection === "build" || item.collection === "builds";
              const linkHref = isBuild
                ? `/builds/${item.entityId}`
                : `/components/hardware/${item.entityId}`;

              return (
                <div
                  key={item.uuid}
                  className="rounded-2xl border border-zinc-200 dark:border-zinc-800 bg-white dark:bg-zinc-900/60 p-4 space-y-3 hover:border-zinc-300 dark:hover:border-zinc-700 transition-colors shadow-2xs flex flex-col justify-between"
                >
                  <div className="space-y-3">
                    {item.primaryDisplayImage ? (
                      <img
                        src={item.primaryDisplayImage}
                        alt={item.name}
                        className="w-full h-32 object-cover rounded-xl border border-zinc-100 dark:border-zinc-800"
                      />
                    ) : (
                      <div className="w-full h-32 rounded-xl bg-zinc-100 dark:bg-zinc-800 flex items-center justify-center text-zinc-400">
                        {isBuild ? <Hammer size={24} /> : <Cpu size={24} />}
                      </div>
                    )}

                    <div>
                      <div className="flex items-center gap-1.5 mb-1">
                        <span className="px-1.5 py-0.5 rounded text-[10px] uppercase font-bold bg-zinc-100 dark:bg-zinc-800 text-zinc-500">
                          {isBuild ? "Build" : "Component"}
                        </span>
                        {item.authorOrManufacturer && (
                          <span className="text-[11px] text-zinc-400 truncate">
                            • {item.authorOrManufacturer}
                          </span>
                        )}
                      </div>

                      <Link
                        to={linkHref}
                        className="font-bold text-sm text-zinc-900 dark:text-zinc-100 hover:text-blue-500 transition-colors line-clamp-1 block"
                      >
                        {item.name || item.entityId}
                      </Link>
                    </div>
                  </div>

                  <div className="flex items-center justify-between pt-2 border-t border-zinc-100 dark:border-zinc-800 text-xs">
                    <Link
                      to={linkHref}
                      className="text-blue-500 hover:underline flex items-center gap-1"
                    >
                      <span>View</span>
                      <ExternalLink size={12} />
                    </Link>

                    {isOwner && (
                      <button
                        type="button"
                        onClick={() =>
                          handleRemoveItem({
                            entityId: item.entityId,
                            collection: item.collection,
                          })
                        }
                        className="text-zinc-400 hover:text-rose-500 transition-colors p-1"
                        title="Remove from collection"
                      >
                        <Trash2 size={13} />
                      </button>
                    )}
                  </div>
                </div>
              );
            })}
          </div>
        )}
      </div>

      {/* Rename Modal */}
      {isRenameOpen && (
        <div
          role="dialog"
          aria-modal="true"
          aria-labelledby="rename-col-title"
          className="fixed inset-0 z-50 bg-black/50 backdrop-blur-xs flex items-center justify-center p-4 animate-in fade-in"
        >
          <div className="bg-white dark:bg-zinc-900 border border-zinc-200 dark:border-zinc-800 rounded-2xl w-full max-w-sm shadow-2xl overflow-hidden p-6 space-y-4">
            <div className="flex items-center justify-between pb-3 border-b border-zinc-100 dark:border-zinc-800">
              <h3
                id="rename-col-title"
                className="font-bold text-base text-zinc-900 dark:text-zinc-100 flex items-center gap-2"
              >
                <Edit3 size={16} className="text-blue-500" />
                <span>Rename Collection</span>
              </h3>
              <button
                type="button"
                onClick={() => setIsRenameOpen(false)}
                className="text-zinc-400 hover:text-zinc-600 dark:hover:text-zinc-200 p-1 rounded-lg cursor-pointer"
                aria-label="Close rename modal"
              >
                <X size={18} />
              </button>
            </div>

            <form onSubmit={handleSaveRename} className="space-y-4">
              <div className="space-y-1.5">
                <label
                  htmlFor="rename-col-input"
                  className="block text-xs font-semibold text-zinc-700 dark:text-zinc-300"
                >
                  Collection Name
                </label>
                <input
                  id="rename-col-input"
                  type="text"
                  value={renameName}
                  onChange={(e) => setRenameName(e.target.value)}
                  className="w-full rounded-xl border border-zinc-200 dark:border-zinc-700 bg-zinc-50 dark:bg-zinc-950 p-3 text-sm text-zinc-900 dark:text-zinc-100 placeholder-zinc-400 focus:outline-hidden focus:border-blue-500"
                  required
                  autoFocus
                />
              </div>

              <div className="flex items-center justify-end gap-2 pt-3 border-t border-zinc-100 dark:border-zinc-800">
                <button
                  type="button"
                  onClick={() => setIsRenameOpen(false)}
                  className="px-4 py-2 rounded-xl text-xs font-semibold text-zinc-600 dark:text-zinc-300 hover:bg-zinc-100 dark:hover:bg-zinc-800 transition-colors cursor-pointer"
                >
                  Cancel
                </button>
                <button
                  type="submit"
                  disabled={isSavingRename || !renameName.trim()}
                  className="px-5 py-2 rounded-xl bg-blue-600 hover:bg-blue-500 text-white font-semibold text-xs transition-colors flex items-center gap-1.5 disabled:opacity-50 cursor-pointer"
                >
                  {isSavingRename && <Loader2 size={12} className="animate-spin" />}
                  <span>Save Name</span>
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  );
}
