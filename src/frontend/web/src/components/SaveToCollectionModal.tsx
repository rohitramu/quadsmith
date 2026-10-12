import { useState, useEffect } from "react";
import { useQuery, useMutation } from "@connectrpc/connect-query";
import { useQueryClient } from "@tanstack/react-query";
import {
  listUserCollections,
  createCollection,
  addCollectionItem,
  removeCollectionItem,
} from "../gen/quadsmith/social-SocialService_connectquery";
import { useAuth } from "../context/AuthContext";
import { X, Plus, Check, Bookmark, Lock, Globe, Loader2 } from "lucide-react";

export interface SaveToCollectionModalProps {
  isOpen: boolean;
  onClose: () => void;
  target: { type?: "build" | "component"; id: string; collection?: string };
  targetName?: string;
}

export function SaveToCollectionModal({
  isOpen,
  onClose,
  target,
  targetName,
}: SaveToCollectionModalProps) {
  const { user, isAuthenticated } = useAuth();
  const queryClient = useQueryClient();

  const [newColName, setNewColName] = useState("");
  const [newColDesc, setNewColDesc] = useState("");
  const [isCreating, setIsCreating] = useState(false);
  const [actionError, setActionError] = useState<string | null>(null);

  // Track toggled collections loading state
  const [pendingCollections, setPendingCollections] = useState<Record<string, boolean>>({});
  // Track optimistic overrides for membership: Record<collectionId, boolean>
  const [membershipOverrides, setMembershipOverrides] = useState<Record<string, boolean>>({});

  // Reset overrides when modal opens or closes or target changes
  useEffect(() => {
    setMembershipOverrides({});
    setActionError(null);
  }, [isOpen, target.id]);

  // Close on Escape key
  useEffect(() => {
    function handleKeyDown(e: KeyboardEvent) {
      if (e.key === "Escape" && isOpen) {
        onClose();
      }
    }
    window.addEventListener("keydown", handleKeyDown);
    return () => window.removeEventListener("keydown", handleKeyDown);
  }, [isOpen, onClose]);

  const queryPayload = {
    userId: user?.id || "",
    containsEntityId: target.id,
    containsCollection: target.collection || (target.type === "build" ? "builds" : undefined),
  };

  const {
    data: collectionsData,
    isLoading: isLoadingCollections,
    refetch: refetchCollections,
  } = useQuery(listUserCollections, queryPayload, {
    enabled: isOpen && isAuthenticated && !!user?.id,
  });

  const { mutateAsync: mutateCreateCollection } = useMutation(createCollection);
  const { mutateAsync: mutateAddItem } = useMutation(addCollectionItem);
  const { mutateAsync: mutateRemoveItem } = useMutation(removeCollectionItem);

  if (!isOpen) return null;

  const collections = (collectionsData?.collections || []).filter((c) => !c.isSystemCollection);

  async function handleToggleCollection(colId: string, currentlyInCollection: boolean) {
    setActionError(null);
    setPendingCollections((prev) => ({ ...prev, [colId]: true }));
    setMembershipOverrides((prev) => ({ ...prev, [colId]: !currentlyInCollection }));
    try {
      if (currentlyInCollection) {
        await mutateRemoveItem({
          collectionId: colId,
          entityId: target.id,
          collection: target.collection || (target.type === "build" ? "builds" : undefined),
        });
      } else {
        await mutateAddItem({
          collectionId: colId,
          entityId: target.id,
          collection: target.collection || (target.type === "build" ? "builds" : undefined),
        });
      }
      await refetchCollections();
      await queryClient.invalidateQueries();
    } catch (err: unknown) {
      setMembershipOverrides((prev) => ({ ...prev, [colId]: currentlyInCollection }));
      setActionError(err instanceof Error ? err.message : "Failed to update collection");
    } finally {
      setPendingCollections((prev) => ({ ...prev, [colId]: false }));
    }
  }

  async function handleCreateNewCollection(e: React.FormEvent) {
    e.preventDefault();
    if (!newColName.trim()) return;

    setActionError(null);
    setIsCreating(true);
    try {
      const created = await mutateCreateCollection({
        name: newColName.trim(),
        description: newColDesc.trim(),
        isPublic: true,
      });

      // Optimistically record that the item is in the newly created collection
      setMembershipOverrides((prev) => ({ ...prev, [created.id]: true }));

      // Automatically add current item to newly created collection
      await mutateAddItem({
        collectionId: created.id,
        entityId: target.id,
        collection: target.collection || (target.type === "build" ? "builds" : undefined),
      });

      setNewColName("");
      setNewColDesc("");
      await refetchCollections();
      await queryClient.invalidateQueries();
    } catch (err: unknown) {
      setActionError(err instanceof Error ? err.message : "Failed to create collection");
    } finally {
      setIsCreating(false);
    }
  }

  return (
    <div
      role="dialog"
      aria-modal="true"
      aria-labelledby="collection-modal-title"
      className="fixed inset-0 z-50 bg-black/70 backdrop-blur-xs flex items-center justify-center p-4 animate-in fade-in duration-150"
    >
      <div className="w-full max-w-md rounded-2xl border border-zinc-200 dark:border-zinc-800 bg-white dark:bg-zinc-900 p-6 space-y-5 shadow-2xl">
        {/* Header */}
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-2">
            <Bookmark className="w-5 h-5 text-blue-500" />
            <h3
              id="collection-modal-title"
              className="font-bold text-base text-zinc-900 dark:text-zinc-100"
            >
              Save to Collection
            </h3>
          </div>
          <button
            onClick={onClose}
            className="p-1 rounded-lg text-zinc-400 hover:text-zinc-600 dark:hover:text-zinc-200 hover:bg-zinc-100 dark:hover:bg-zinc-800 transition-colors"
            aria-label="Close"
          >
            <X size={18} />
          </button>
        </div>

        {targetName && (
          <p className="text-xs text-zinc-500 dark:text-zinc-400 truncate">
            Saving:{" "}
            <span className="font-semibold text-zinc-800 dark:text-zinc-200">{targetName}</span>
          </p>
        )}

        {actionError && (
          <div className="p-3 rounded-xl bg-red-500/10 border border-red-500/20 text-red-500 text-xs">
            {actionError}
          </div>
        )}

        {!isAuthenticated ? (
          <div className="p-6 text-center text-xs text-zinc-500 space-y-2">
            <p>Please log in to save items to your personal collections.</p>
          </div>
        ) : isLoadingCollections ? (
          <div className="py-8 flex justify-center text-zinc-400">
            <Loader2 className="animate-spin text-blue-500" size={24} />
          </div>
        ) : (
          <div className="space-y-3 max-h-60 overflow-y-auto pr-1">
            {collections.length === 0 ? (
              <p className="text-xs text-zinc-500 text-center py-4">
                No collections yet. Create your first collection below!
              </p>
            ) : (
              collections.map((col) => {
                const isPending = pendingCollections[col.id] || false;
                const isAdded =
                  membershipOverrides[col.id] !== undefined
                    ? membershipOverrides[col.id]
                    : Boolean(col.containsItem);
                const initiallyHad = Boolean(col.containsItem);
                const displayItemCount =
                  col.itemCount +
                  (isAdded && !initiallyHad ? 1 : !isAdded && initiallyHad ? -1 : 0);

                return (
                  <button
                    key={col.id}
                    type="button"
                    disabled={isPending}
                    onClick={() => handleToggleCollection(col.id, isAdded)}
                    aria-pressed={isAdded}
                    aria-label={
                      isAdded
                        ? `Remove ${targetName || "item"} from ${col.name}`
                        : `Add ${targetName || "item"} to ${col.name}`
                    }
                    className={`w-full flex items-center justify-between p-3 rounded-xl border text-left transition-all cursor-pointer group ${
                      isAdded
                        ? "border-emerald-500/40 bg-emerald-50/50 dark:bg-emerald-950/20 hover:bg-emerald-100/50 dark:hover:bg-emerald-900/30"
                        : "border-zinc-200 dark:border-zinc-800 bg-zinc-50 dark:bg-zinc-950/60 hover:bg-zinc-100 dark:hover:bg-zinc-800/40"
                    }`}
                  >
                    <div className="flex items-center gap-3 min-w-0">
                      <div
                        className={`w-8 h-8 rounded-lg flex items-center justify-center shrink-0 transition-colors ${
                          isAdded
                            ? "bg-emerald-500/10 text-emerald-600 dark:text-emerald-400"
                            : "bg-blue-500/10 text-blue-500"
                        }`}
                      >
                        {col.isPublic ? <Globe size={14} /> : <Lock size={14} />}
                      </div>
                      <div className="min-w-0">
                        <div className="text-sm font-semibold text-zinc-900 dark:text-zinc-100 truncate flex items-center gap-2">
                          <span>{col.name}</span>
                          {isAdded && (
                            <span className="text-[10px] uppercase font-bold tracking-wider px-1.5 py-0.5 rounded-sm bg-emerald-500/15 text-emerald-600 dark:text-emerald-400">
                              Added
                            </span>
                          )}
                        </div>
                        <div className="text-[11px] text-zinc-500">
                          {col.isPublic ? "Public" : "Private"} • {Math.max(0, displayItemCount)}{" "}
                          items
                        </div>
                      </div>
                    </div>
                    <div className="shrink-0 pl-2">
                      {isPending ? (
                        <div className="w-6 h-6 flex items-center justify-center">
                          <Loader2 size={16} className="animate-spin text-zinc-400" />
                        </div>
                      ) : isAdded ? (
                        <div
                          data-testid="tick-icon"
                          className="w-6 h-6 rounded-full bg-emerald-500 text-white flex items-center justify-center shadow-xs transition-transform group-hover:scale-105"
                          title="Added to collection (click to remove)"
                        >
                          <Check size={14} strokeWidth={2.5} />
                        </div>
                      ) : (
                        <div
                          data-testid="plus-icon"
                          className="w-6 h-6 rounded-full border border-zinc-300 dark:border-zinc-700 flex items-center justify-center text-zinc-400 group-hover:border-blue-500 group-hover:text-blue-500 transition-colors"
                          title="Add to collection"
                        >
                          <Plus size={14} />
                        </div>
                      )}
                    </div>
                  </button>
                );
              })
            )}
          </div>
        )}

        {/* Quick create collection form */}
        {isAuthenticated && (
          <form
            onSubmit={handleCreateNewCollection}
            className="pt-3 border-t border-zinc-200 dark:border-zinc-800 space-y-2.5"
          >
            <label className="block text-xs font-semibold text-zinc-600 dark:text-zinc-400">
              Create New Collection
            </label>
            <div className="flex gap-2">
              <input
                type="text"
                value={newColName}
                onChange={(e) => setNewColName(e.target.value)}
                placeholder="Collection name (e.g. 5-inch Inspo)"
                className="flex-1 rounded-xl border border-zinc-200 dark:border-zinc-700 bg-zinc-50 dark:bg-zinc-950 px-3 py-2 text-xs text-zinc-900 dark:text-zinc-100 placeholder-zinc-400 focus:outline-hidden focus:border-blue-500"
              />
              <button
                type="submit"
                disabled={isCreating || !newColName.trim()}
                className="px-3.5 py-2 rounded-xl bg-zinc-900 hover:bg-zinc-800 dark:bg-zinc-800 dark:hover:bg-zinc-700 text-white text-xs font-semibold disabled:opacity-50 transition-colors flex items-center gap-1.5"
              >
                {isCreating && <Loader2 size={12} className="animate-spin" />}
                <span>Create</span>
              </button>
            </div>
          </form>
        )}

        <div className="flex justify-end pt-2">
          <button
            type="button"
            onClick={onClose}
            className="w-full py-2.5 rounded-xl bg-blue-600 hover:bg-blue-500 text-white font-semibold text-xs shadow-md transition-colors"
          >
            Done
          </button>
        </div>
      </div>
    </div>
  );
}
