import { useState } from "react";
import { useParams, Link, useNavigate } from "react-router-dom";
import { useQuery, useMutation } from "@connectrpc/connect-query";
import { useQueryClient } from "@tanstack/react-query";
import {
  listUserCollections,
  getCollection,
  listCollectionItems,
  deleteCollection,
  removeCollectionItem,
  createCollection,
} from "../gen/quadsmith/social-SocialService_connectquery";
import type { Collection, CollectionItem } from "../gen/quadsmith/social_pb";
import { useAuth } from "../context/AuthContext";
import {
  Bookmark,
  Heart,
  Plus,
  Trash2,
  ExternalLink,
  ChevronRight,
  Layers,
  Hammer,
  Cpu,
  Loader2,
} from "lucide-react";

export function UserCollectionsPage() {
  const { id: userId, collectionId } = useParams<{ id: string; collectionId?: string }>();
  const { user } = useAuth();
  const queryClient = useQueryClient();
  const navigate = useNavigate();

  const [isCreateOpen, setIsCreateOpen] = useState(false);
  const [newColName, setNewColName] = useState("");
  const [newColDesc, setNewColDesc] = useState("");
  const [newColIsPublic, setNewColIsPublic] = useState(true);
  const [isCreating, setIsCreating] = useState(false);

  const isOwner = user?.id === userId;

  // List of collections for user
  const {
    data: colsRes,
    isLoading: isLoadingCols,
    refetch: refetchCols,
  } = useQuery(
    listUserCollections,
    { userId: userId || "" },
    { enabled: !!userId && !collectionId },
  );

  // Single collection details
  const { data: singleCol, isLoading: isLoadingSingleCol } = useQuery(
    getCollection,
    { id: collectionId || "" },
    { enabled: !!collectionId },
  );

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
  const { mutateAsync: mutateCreateCollection } = useMutation(createCollection);

  async function handleDeleteCollection(colId: string) {
    if (!window.confirm("Are you sure you want to delete this collection?")) return;
    try {
      await mutateDeleteCollection({ collectionId: colId });
      await refetchCols();
      await queryClient.invalidateQueries();
      navigate(`/users/${userId}/collections`);
    } catch (err: unknown) {
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

  async function handleCreateCollection(e: React.FormEvent) {
    e.preventDefault();
    if (!newColName.trim()) return;
    setIsCreating(true);
    try {
      await mutateCreateCollection({
        name: newColName.trim(),
        description: newColDesc.trim(),
        isPublic: newColIsPublic,
      });
      setNewColName("");
      setNewColDesc("");
      setIsCreateOpen(false);
      await refetchCols();
      await queryClient.invalidateQueries();
    } catch (err: unknown) {
      alert(err instanceof Error ? err.message : "Failed to create collection");
    } finally {
      setIsCreating(false);
    }
  }

  // 1. Single Collection View
  if (collectionId) {
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
            to={`/users/${userId}/collections`}
            className="text-blue-500 hover:underline text-xs"
          >
            ← Back to collections
          </Link>
        </div>
      );
    }

    const items = itemsRes?.items || [];

    return (
      <div className="max-w-5xl mx-auto px-4 py-8 space-y-8">
        {/* Breadcrumb */}
        <nav
          aria-label="Breadcrumb"
          className="text-sm text-zinc-500 dark:text-zinc-400 flex items-center gap-1.5 flex-wrap"
        >
          <Link to={`/users/${userId}`} className="hover:underline">
            @{userId}
          </Link>
          <ChevronRight size={14} className="text-zinc-400" />
          <Link to={`/users/${userId}/collections`} className="hover:underline">
            Collections
          </Link>
          <ChevronRight size={14} className="text-zinc-400" />
          <span className="font-semibold text-zinc-900 dark:text-zinc-100">{singleCol.name}</span>
        </nav>

        {/* Collection Header Card */}
        <div className="p-8 rounded-2xl border border-zinc-200 dark:border-zinc-800 bg-white dark:bg-zinc-900 shadow-xs space-y-4">
          <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
            <div className="flex items-center gap-4">
              <div className="w-14 h-14 rounded-2xl bg-blue-500/10 text-blue-500 flex items-center justify-center font-bold text-2xl">
                {singleCol.isDefaultLiked ? (
                  <Heart size={28} className="fill-rose-500 text-rose-500" />
                ) : (
                  <Bookmark size={28} />
                )}
              </div>
              <div>
                <div className="flex items-center gap-2 flex-wrap">
                  <h1 className="text-2xl font-black text-zinc-900 dark:text-zinc-50">
                    {singleCol.name}
                  </h1>
                  <span
                    className={`px-2 py-0.5 rounded text-[10px] font-semibold border ${
                      singleCol.isPublic
                        ? "bg-emerald-500/10 text-emerald-600 dark:text-emerald-400 border-emerald-500/20"
                        : "bg-zinc-100 dark:bg-zinc-800 text-zinc-500 border-zinc-200 dark:border-zinc-700"
                    }`}
                  >
                    {singleCol.isPublic ? "Public" : "Private"}
                  </span>
                </div>
                <p className="text-xs text-zinc-500 dark:text-zinc-400 mt-0.5">
                  Curated by @{singleCol.authorUsername} • {items.length} items
                </p>
              </div>
            </div>

            {isOwner && !singleCol.isDefaultLiked && (
              <button
                type="button"
                onClick={() => handleDeleteCollection(singleCol.id)}
                className="px-3 py-2 rounded-xl text-xs text-rose-500 hover:bg-rose-500/10 border border-rose-500/20 flex items-center gap-1.5 transition-colors self-start sm:self-auto cursor-pointer"
              >
                <Trash2 size={14} />
                <span>Delete Collection</span>
              </button>
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
      </div>
    );
  }

  // 2. All User Collections View
  return (
    <div className="max-w-5xl mx-auto px-4 py-8 space-y-8">
      {/* Breadcrumb */}
      <nav
        aria-label="Breadcrumb"
        className="text-sm text-zinc-500 dark:text-zinc-400 flex items-center gap-1.5 flex-wrap"
      >
        <Link to={`/users/${userId}`} className="hover:underline">
          @{userId}
        </Link>
        <ChevronRight size={14} className="text-zinc-400" />
        <span className="font-semibold text-zinc-900 dark:text-zinc-100">Collections</span>
      </nav>

      {/* Header */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4 p-8 rounded-2xl border border-zinc-200 dark:border-zinc-800 bg-white dark:bg-zinc-900 shadow-xs">
        <div>
          <h1 className="text-2xl font-black text-zinc-900 dark:text-zinc-50">
            Collections by @{userId}
          </h1>
          <p className="text-xs text-zinc-500 dark:text-zinc-400 mt-1">
            Curated quadcopter builds, hardware wishlists, and component pairings.
          </p>
        </div>

        {isOwner && (
          <button
            type="button"
            onClick={() => setIsCreateOpen((prev) => !prev)}
            className="px-5 py-2.5 rounded-xl bg-blue-600 hover:bg-blue-500 text-white font-semibold text-xs shadow-md shadow-blue-500/20 flex items-center gap-1.5 transition-colors self-start sm:self-auto cursor-pointer"
          >
            <Plus size={14} />
            <span>Create Collection</span>
          </button>
        )}
      </div>

      {/* Create Collection Inline Form */}
      {isCreateOpen && (
        <form
          onSubmit={handleCreateCollection}
          className="p-6 rounded-2xl border border-blue-500/30 bg-white dark:bg-zinc-900 space-y-4 shadow-xl"
        >
          <div className="flex items-center justify-between">
            <h3 className="font-bold text-sm text-zinc-900 dark:text-zinc-100">
              New Custom Collection
            </h3>
            <button
              type="button"
              onClick={() => setIsCreateOpen(false)}
              className="text-zinc-400 hover:text-zinc-600 dark:hover:text-zinc-200 text-xs"
            >
              Cancel
            </button>
          </div>

          <div className="space-y-1.5">
            <label className="block text-xs font-semibold text-zinc-600 dark:text-zinc-400">
              Name
            </label>
            <input
              type="text"
              value={newColName}
              onChange={(e) => setNewColName(e.target.value)}
              placeholder="e.g. Cinematic Freestyle 6S Rigs"
              className="w-full rounded-xl border border-zinc-200 dark:border-zinc-700 bg-zinc-50 dark:bg-zinc-950 p-3 text-sm text-zinc-900 dark:text-zinc-100 placeholder-zinc-400 focus:outline-hidden focus:border-blue-500"
            />
          </div>

          <div className="space-y-1.5">
            <label className="block text-xs font-semibold text-zinc-600 dark:text-zinc-400">
              Description (optional)
            </label>
            <textarea
              rows={2}
              value={newColDesc}
              onChange={(e) => setNewColDesc(e.target.value)}
              placeholder="Notes on what is collected here..."
              className="w-full rounded-xl border border-zinc-200 dark:border-zinc-700 bg-zinc-50 dark:bg-zinc-950 p-3 text-sm text-zinc-900 dark:text-zinc-100 placeholder-zinc-400 focus:outline-hidden focus:border-blue-500"
            />
          </div>

          <div className="flex items-center justify-between pt-2">
            <label className="flex items-center gap-2 cursor-pointer text-xs text-zinc-700 dark:text-zinc-300">
              <input
                type="checkbox"
                checked={newColIsPublic}
                onChange={(e) => setNewColIsPublic(e.target.checked)}
                className="rounded border-zinc-300 text-blue-600 focus:ring-blue-500"
              />
              <span>Public collection (visible on your public profile)</span>
            </label>

            <button
              type="submit"
              disabled={isCreating || !newColName.trim()}
              className="px-5 py-2 rounded-xl bg-blue-600 hover:bg-blue-500 text-white font-semibold text-xs transition-colors flex items-center gap-1.5 disabled:opacity-50"
            >
              {isCreating && <Loader2 size={12} className="animate-spin" />}
              <span>Create</span>
            </button>
          </div>
        </form>
      )}

      {/* Grid of Collections */}
      {isLoadingCols ? (
        <div className="py-16 flex justify-center text-zinc-400">
          <Loader2 size={24} className="animate-spin text-blue-500" />
        </div>
      ) : (colsRes?.collections || []).length === 0 ? (
        <div className="p-12 text-center rounded-2xl border border-dashed border-zinc-200 dark:border-zinc-800 text-zinc-400 text-xs">
          No collections found for @{userId}.
        </div>
      ) : (
        <div className="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 gap-4">
          {((colsRes?.collections as Collection[]) || []).map((col: Collection) => (
            <div
              key={col.id}
              className="rounded-2xl border border-zinc-200 dark:border-zinc-800 bg-white dark:bg-zinc-900/60 p-5 space-y-3 hover:border-zinc-300 dark:hover:border-zinc-700 transition-colors shadow-2xs flex flex-col justify-between"
            >
              <div className="space-y-2">
                <div className="flex items-center justify-between">
                  <div className="flex items-center gap-2">
                    <div className="w-8 h-8 rounded-lg bg-blue-500/10 text-blue-500 flex items-center justify-center font-bold text-xs">
                      {col.isDefaultLiked ? "❤️" : <Bookmark size={15} />}
                    </div>
                    <div>
                      <h4 className="font-bold text-sm text-zinc-900 dark:text-zinc-100">
                        {col.name}
                      </h4>
                      <p className="text-[10px] text-zinc-400">
                        {col.isDefaultLiked ? "System Collection" : `@${col.authorUsername}`}
                      </p>
                    </div>
                  </div>
                  <span
                    className={`px-2 py-0.5 rounded text-[10px] font-semibold border ${
                      col.isPublic
                        ? "bg-emerald-500/10 text-emerald-600 dark:text-emerald-400 border-emerald-500/20"
                        : "bg-zinc-100 dark:bg-zinc-800 text-zinc-500 border-zinc-200 dark:border-zinc-700"
                    }`}
                  >
                    {col.isPublic ? "Public" : "Private"}
                  </span>
                </div>

                {col.description && (
                  <p className="text-xs text-zinc-600 dark:text-zinc-400 line-clamp-2">
                    {col.description}
                  </p>
                )}
              </div>

              <div className="flex items-center justify-between text-xs text-zinc-500 pt-3 border-t border-zinc-100 dark:border-zinc-800">
                <span className="font-mono">{col.itemCount} items</span>
                <Link
                  to={`/users/${userId}/collections/${col.id}`}
                  className="text-blue-500 hover:underline flex items-center gap-1 font-medium"
                >
                  <span>View Details</span>
                  <ChevronRight size={13} />
                </Link>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
