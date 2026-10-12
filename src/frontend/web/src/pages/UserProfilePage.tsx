import { useState, useEffect, useMemo, useRef } from "react";
import { useParams, Link, useSearchParams } from "react-router-dom";
import { useQuery, useInfiniteQuery, useMutation } from "@connectrpc/connect-query";
import { useQueryClient } from "@tanstack/react-query";
import {
  Loader2,
  Hammer,
  Calendar,
  AlertCircle,
  Star,
  Bookmark,
  Plus,
  Globe,
  Lock,
  Heart,
  RefreshCw,
  CheckCircle,
  Trash2,
  X,
} from "lucide-react";
import { getPublicProfile } from "../gen/quadsmith/user-UserService_connectquery";
import { listBuilds } from "../gen/quadsmith/build-BuildService_connectquery";
import {
  listUserCollections,
  createCollection,
  deleteCollection,
} from "../gen/quadsmith/social-SocialService_connectquery";
import { getAvatarPalette, getInitial } from "../lib/avatarColor";
import { BuildCard } from "../components/BuildCard";
import { useAuth } from "../context/AuthContext";
import { useDocumentMeta } from "../hooks/useDocumentMeta";

export function UserProfilePage() {
  const { id } = useParams<{ id: string }>();
  const { user } = useAuth();
  const queryClient = useQueryClient();
  const [searchParams, setSearchParams] = useSearchParams();
  const sentinelRef = useRef<HTMLDivElement>(null);

  // Modal & action state for collections
  const [isCreateOpen, setIsCreateOpen] = useState(false);
  const [newColName, setNewColName] = useState("");
  const [newColDesc, setNewColDesc] = useState("");
  const [newColIsPublic, setNewColIsPublic] = useState(true);
  const [isCreating, setIsCreating] = useState(false);
  const [deletingColId, setDeletingColId] = useState<string | null>(null);

  const activeTab = searchParams.get("tab") === "collections" ? "collections" : "builds";

  const handleTabChange = (tab: "builds" | "collections") => {
    setSearchParams((prev) => {
      const next = new URLSearchParams(prev);
      if (tab === "builds") {
        next.delete("tab");
      } else {
        next.set("tab", tab);
      }
      return next;
    });
  };

  // Fetch Public User Profile
  const {
    data: profile,
    isLoading: isLoadingProfile,
    error: profileError,
  } = useQuery(getPublicProfile, { id: id || "" }, { enabled: !!id });

  // Infinite Query for User Builds
  const {
    data: buildsData,
    isLoading: isLoadingBuilds,
    isFetchingNextPage: isFetchingNextBuilds,
    hasNextPage: hasNextBuilds,
    fetchNextPage: fetchNextBuilds,
    error: buildsError,
    refetch: refetchBuilds,
  } = useInfiniteQuery(
    listBuilds,
    { pageSize: 8, filter: id ? `author_username == "${id}"` : "", pageToken: "" },
    {
      pageParamKey: "pageToken",
      getNextPageParam: (lastPage) => lastPage.nextPageToken || undefined,
      enabled: !!id,
    },
  );

  const builds = useMemo(() => {
    return buildsData?.pages.flatMap((page) => page.builds) ?? [];
  }, [buildsData]);

  // Infinite scroll observer for builds feed
  useEffect(() => {
    const sentinel = sentinelRef.current;
    if (!sentinel || typeof IntersectionObserver === "undefined") return;

    const observer = new IntersectionObserver(
      (entries) => {
        if (entries[0].isIntersecting && hasNextBuilds && !isFetchingNextBuilds) {
          fetchNextBuilds();
        }
      },
      { rootMargin: "250px" },
    );

    observer.observe(sentinel);
    return () => observer.disconnect();
  }, [hasNextBuilds, isFetchingNextBuilds, fetchNextBuilds]);

  // Fetch User Collections
  const {
    data: collectionsData,
    isLoading: isLoadingCollections,
    error: collectionsError,
    refetch: refetchCollections,
  } = useQuery(listUserCollections, { userId: id || "" }, { enabled: !!id });

  const isOwner = user?.id === id;
  const collections = collectionsData?.collections || [];

  const { mutateAsync: mutateCreateCollection } = useMutation(createCollection);
  const { mutateAsync: mutateDeleteCollection } = useMutation(deleteCollection);

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
      setNewColIsPublic(true);
      setIsCreateOpen(false);
      await refetchCollections();
      await queryClient.invalidateQueries();
    } catch (err: unknown) {
      alert(err instanceof Error ? err.message : "Failed to create collection");
    } finally {
      setIsCreating(false);
    }
  }

  async function handleDeleteCollection(colId: string) {
    if (!window.confirm("Are you sure you want to delete this collection?")) return;
    setDeletingColId(colId);
    try {
      await mutateDeleteCollection({ collectionId: colId });
      await refetchCollections();
      await queryClient.invalidateQueries();
    } catch (err: unknown) {
      alert(err instanceof Error ? err.message : "Failed to delete collection");
    } finally {
      setDeletingColId(null);
    }
  }

  useDocumentMeta({
    title: profile ? `@${profile.id} — Quadsmith Pilot Profile` : "User Profile | Quadsmith",
    description:
      profile?.bio || `Inspect multirotor builds and curated collections by @${id || "pilot"}.`,
  });

  if (isLoadingProfile) {
    return (
      <div className="flex h-96 w-full items-center justify-center text-zinc-400">
        <Loader2 size={24} className="animate-spin text-blue-500" />
      </div>
    );
  }

  if (profileError || !profile) {
    return (
      <div className="max-w-2xl mx-auto py-16 px-4 text-center space-y-4">
        <div className="w-12 h-12 rounded-full bg-red-500/10 text-red-500 flex items-center justify-center mx-auto">
          <AlertCircle size={24} />
        </div>
        <h2 className="text-xl font-bold">User Not Found</h2>
        <p className="text-xs text-zinc-500">
          {profileError?.message || "The requested username does not exist."}
        </p>
        <Link
          to="/"
          className="inline-block px-4 py-2 rounded-xl text-xs font-semibold bg-zinc-900 text-white dark:bg-zinc-100 dark:text-zinc-900"
        >
          Return Home
        </Link>
      </div>
    );
  }

  const palette = getAvatarPalette(profile.id);
  const initial = getInitial(profile.id);
  const displayBuildCount = profile.buildCount ?? builds.length;
  const displayColCount = collections.length;

  return (
    <div className="max-w-5xl mx-auto px-4 py-8 space-y-8">
      {/* User Profile Header Card */}
      <div className="p-8 rounded-2xl border border-zinc-200 dark:border-zinc-800 bg-white dark:bg-zinc-900 shadow-sm space-y-6">
        <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-6">
          <div className="flex items-center gap-5">
            {profile.avatarUrl ? (
              <img
                src={profile.avatarUrl}
                alt={profile.id}
                className="w-20 h-20 rounded-full object-cover border-2 border-zinc-200 dark:border-zinc-800 shadow-md"
              />
            ) : (
              <div
                className={`w-20 h-20 rounded-full border-2 flex items-center justify-center font-bold text-3xl shadow-md ${palette.badgeClass}`}
              >
                {initial}
              </div>
            )}
            <div className="space-y-1">
              <h1 className="text-2xl font-extrabold tracking-tight">@{profile.id}</h1>
              {profile.createdAt && (
                <div className="flex items-center gap-1.5 text-[11px] text-zinc-400 pt-1">
                  <Calendar size={12} />
                  <span>
                    Member since{" "}
                    {new Date(Number(profile.createdAt.seconds) * 1000).toLocaleDateString(
                      undefined,
                      {
                        year: "numeric",
                        month: "short",
                      },
                    )}
                  </span>
                </div>
              )}
            </div>
          </div>

          {/* Interactive Stats Badges */}
          <div className="flex items-center gap-4 bg-zinc-50 dark:bg-zinc-950 p-4 rounded-xl border border-zinc-200 dark:border-zinc-800 flex-wrap">
            <button
              type="button"
              onClick={() => handleTabChange("builds")}
              aria-label="View authored builds"
              className="flex items-center gap-3 text-left hover:opacity-80 transition-opacity cursor-pointer group"
              title="View builds"
            >
              <div className="p-2.5 rounded-lg bg-blue-500/10 text-blue-500 group-hover:scale-105 transition-transform">
                <Hammer size={20} />
              </div>
              <div>
                <div className="text-xl font-bold font-mono">{displayBuildCount}</div>
                <div className="text-[11px] text-zinc-400 font-medium">Authored Builds</div>
              </div>
            </button>

            <div className="h-8 w-px bg-zinc-200 dark:bg-zinc-800" />

            <button
              type="button"
              onClick={() => handleTabChange("collections")}
              aria-label="View collections"
              className="flex items-center gap-3 text-left hover:opacity-80 transition-opacity cursor-pointer group"
              title="View collections"
            >
              <div className="p-2.5 rounded-lg bg-indigo-500/10 text-indigo-500 group-hover:scale-105 transition-transform">
                <Bookmark size={20} />
              </div>
              <div>
                <div className="text-xl font-bold font-mono">{displayColCount}</div>
                <div className="text-[11px] text-zinc-400 font-medium">Collections</div>
              </div>
            </button>

            <div className="h-8 w-px bg-zinc-200 dark:bg-zinc-800" />

            <div className="flex items-center gap-3">
              <div className="p-2.5 rounded-lg bg-amber-500/10 text-amber-500">
                <Star size={20} className="fill-current" />
              </div>
              <div>
                <div className="text-xl font-bold font-mono">{profile.reputationScore ?? 0}</div>
                <div className="text-[11px] text-zinc-400 font-medium">Reputation</div>
              </div>
            </div>
          </div>
        </div>

        {/* Bio */}
        {profile.bio && (
          <div className="p-4 rounded-xl bg-zinc-50 dark:bg-zinc-950 border border-zinc-200 dark:border-zinc-800 text-xs text-zinc-600 dark:text-zinc-300 leading-relaxed">
            {profile.bio}
          </div>
        )}
      </div>

      {/* Larger Tabbed Section */}
      <div className="p-6 sm:p-8 rounded-2xl border border-zinc-200 dark:border-zinc-800 bg-white dark:bg-zinc-900 shadow-sm space-y-6">
        {/* Tab Navigation Bar */}
        <div
          role="tablist"
          aria-label="User profile content"
          className="flex items-center gap-2 border-b border-zinc-200 dark:border-zinc-800 pb-px"
        >
          <button
            role="tab"
            id="tab-builds"
            aria-controls="panel-builds"
            aria-selected={activeTab === "builds"}
            type="button"
            onClick={() => handleTabChange("builds")}
            className={`flex items-center gap-2 px-5 py-3 border-b-2 font-bold text-sm transition-colors cursor-pointer ${
              activeTab === "builds"
                ? "border-blue-500 text-blue-600 dark:text-blue-400"
                : "border-transparent text-zinc-500 dark:text-zinc-400 hover:text-zinc-800 dark:hover:text-zinc-200 hover:border-zinc-300 dark:hover:border-zinc-700"
            }`}
          >
            <Hammer size={16} />
            <span>Builds</span>
            <span
              className={`px-2 py-0.5 rounded-full text-[11px] font-mono transition-colors ${
                activeTab === "builds"
                  ? "bg-blue-500/10 text-blue-600 dark:text-blue-400"
                  : "bg-zinc-100 dark:bg-zinc-800 text-zinc-500 dark:text-zinc-400"
              }`}
            >
              {displayBuildCount}
            </span>
          </button>

          <button
            role="tab"
            id="tab-collections"
            aria-controls="panel-collections"
            aria-selected={activeTab === "collections"}
            type="button"
            onClick={() => handleTabChange("collections")}
            className={`flex items-center gap-2 px-5 py-3 border-b-2 font-bold text-sm transition-colors cursor-pointer ${
              activeTab === "collections"
                ? "border-blue-500 text-blue-600 dark:text-blue-400"
                : "border-transparent text-zinc-500 dark:text-zinc-400 hover:text-zinc-800 dark:hover:text-zinc-200 hover:border-zinc-300 dark:hover:border-zinc-700"
            }`}
          >
            <Bookmark size={16} />
            <span>Collections</span>
            <span
              className={`px-2 py-0.5 rounded-full text-[11px] font-mono transition-colors ${
                activeTab === "collections"
                  ? "bg-blue-500/10 text-blue-600 dark:text-blue-400"
                  : "bg-zinc-100 dark:bg-zinc-800 text-zinc-500 dark:text-zinc-400"
              }`}
            >
              {displayColCount}
            </span>
          </button>
        </div>

        {/* Tab 1: Builds Panel */}
        {activeTab === "builds" && (
          <section
            role="tabpanel"
            id="panel-builds"
            aria-labelledby="tab-builds"
            className="space-y-6 animate-in fade-in duration-150"
          >
            {/* Builds Section Subheader */}
            <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4 pb-2">
              <div>
                <h3 className="text-base font-bold text-zinc-900 dark:text-zinc-100 flex items-center gap-2">
                  <Hammer size={18} className="text-blue-500" />
                  <span>Authored Builds</span>
                </h3>
                <p className="text-xs text-zinc-500 dark:text-zinc-400 mt-0.5">
                  Verified quadcopter builds and component bills of materials engineered by @
                  {profile.id}.
                </p>
              </div>

              {isOwner && (
                <Link
                  to="/build/new"
                  className="inline-flex items-center gap-1.5 px-3.5 py-1.5 rounded-xl bg-blue-600 hover:bg-blue-500 text-white font-semibold text-xs shadow-md shadow-blue-500/20 transition-colors self-start sm:self-auto cursor-pointer"
                >
                  <Plus size={14} />
                  <span>New Build in Forge</span>
                </Link>
              )}
            </div>

            {/* Loading Skeleton */}
            {isLoadingBuilds && builds.length === 0 && (
              <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
                {[1, 2].map((n) => (
                  <div
                    key={n}
                    className="rounded-2xl border border-zinc-200 dark:border-zinc-800 p-6 bg-white dark:bg-zinc-900 animate-pulse space-y-4"
                  >
                    <div className="flex items-center gap-3">
                      <div className="w-10 h-10 rounded-xl bg-zinc-200 dark:bg-zinc-800" />
                      <div className="space-y-2 flex-1">
                        <div className="h-4 bg-zinc-200 dark:bg-zinc-800 rounded w-1/3" />
                        <div className="h-3 bg-zinc-100 dark:bg-zinc-800/60 rounded w-1/4" />
                      </div>
                    </div>
                    <div className="h-28 bg-zinc-100 dark:bg-zinc-800/40 rounded-xl" />
                    <div className="h-4 bg-zinc-100 dark:bg-zinc-800/60 rounded w-3/4" />
                  </div>
                ))}
              </div>
            )}

            {/* Error Alert */}
            {buildsError && (
              <div className="p-6 rounded-2xl border border-red-200 dark:border-red-900/60 bg-red-50 dark:bg-red-950/30 text-center space-y-3">
                <p className="text-xs text-red-600 dark:text-red-400">
                  Failed to load builds: {buildsError.message}
                </p>
                <button
                  type="button"
                  onClick={() => refetchBuilds()}
                  className="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-red-600 text-white text-xs font-medium hover:bg-red-700 cursor-pointer"
                >
                  <RefreshCw size={13} />
                  <span>Retry</span>
                </button>
              </div>
            )}

            {/* Empty State */}
            {!isLoadingBuilds && !buildsError && builds.length === 0 && (
              <div className="p-12 text-center rounded-2xl border border-dashed border-zinc-200 dark:border-zinc-800 bg-zinc-50 dark:bg-zinc-950 text-zinc-400 text-xs space-y-3">
                <div className="w-12 h-12 rounded-2xl bg-zinc-100 dark:bg-zinc-800 flex items-center justify-center mx-auto text-zinc-400">
                  <Hammer size={20} />
                </div>
                <p className="font-bold text-sm text-zinc-700 dark:text-zinc-200">
                  No published builds yet
                </p>
                <p className="text-zinc-500 max-w-sm mx-auto">
                  When @{profile.id} creates and publishes quadcopter builds, they will appear here.
                </p>
                {isOwner && (
                  <div className="pt-2">
                    <Link
                      to="/build/new"
                      className="inline-flex items-center gap-1.5 px-4 py-2 rounded-xl bg-blue-600 hover:bg-blue-500 text-white font-semibold text-xs shadow-md shadow-blue-500/20 transition-colors"
                    >
                      <Plus size={14} />
                      <span>Start a New Build in Forge</span>
                    </Link>
                  </div>
                )}
              </div>
            )}

            {/* Scrollable Builds Grid */}
            {builds.length > 0 && (
              <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
                {builds.map((b) => (
                  <BuildCard key={b.uuid || b.id} build={b} />
                ))}
              </div>
            )}

            {/* Auto-loading Sentinel & Pagination State */}
            <div ref={sentinelRef} className="py-4 text-center" aria-live="polite">
              {isFetchingNextBuilds && (
                <div className="inline-flex items-center gap-2 text-xs text-zinc-500 font-medium">
                  <RefreshCw size={14} className="animate-spin text-blue-600" />
                  <span>Loading more builds...</span>
                </div>
              )}

              {!isLoadingBuilds && !hasNextBuilds && builds.length > 0 && (
                <div className="inline-flex items-center gap-2 text-xs text-zinc-400 dark:text-zinc-500 font-medium">
                  <CheckCircle size={14} className="text-emerald-500" />
                  <span>You&apos;ve caught up with all builds by @{profile.id}</span>
                </div>
              )}
            </div>
          </section>
        )}

        {/* Tab 2: Collections Panel */}
        {activeTab === "collections" && (
          <section
            role="tabpanel"
            id="panel-collections"
            aria-labelledby="tab-collections"
            className="space-y-6 animate-in fade-in duration-150"
          >
            {/* Collections Section Subheader with subtle plus button */}
            <div className="flex items-center justify-between gap-4 pb-2">
              <div>
                <h3 className="text-base font-bold text-zinc-900 dark:text-zinc-100 flex items-center gap-2">
                  <Bookmark size={18} className="text-blue-500" />
                  <span>Curated Collections</span>
                </h3>
                <p className="text-xs text-zinc-500 dark:text-zinc-400 mt-0.5">
                  Curated quadcopter builds, hardware wishlists, and component pairings.
                </p>
              </div>

              {isOwner && collections.length > 0 && (
                <button
                  type="button"
                  onClick={() => setIsCreateOpen(true)}
                  className="inline-flex items-center justify-center p-2 rounded-xl border border-zinc-200 dark:border-zinc-800 hover:bg-zinc-100 dark:hover:bg-zinc-800 text-zinc-600 dark:text-zinc-300 hover:text-zinc-900 dark:hover:text-zinc-100 transition-colors cursor-pointer self-start sm:self-auto"
                  title="Add Collection"
                  aria-label="Add Collection"
                >
                  <Plus size={16} />
                </button>
              )}
            </div>

            {/* Loading Skeleton */}
            {isLoadingCollections && collections.length === 0 && (
              <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-5">
                {[1, 2, 3].map((n) => (
                  <div
                    key={n}
                    className="rounded-2xl border border-zinc-200 dark:border-zinc-800 p-5 bg-white dark:bg-zinc-900 animate-pulse space-y-3"
                  >
                    <div className="h-4 bg-zinc-200 dark:bg-zinc-800 rounded w-1/2" />
                    <div className="h-10 bg-zinc-100 dark:bg-zinc-800/60 rounded" />
                    <div className="h-3 bg-zinc-200 dark:bg-zinc-800 rounded w-1/4 pt-2" />
                  </div>
                ))}
              </div>
            )}

            {/* Error Alert */}
            {collectionsError && (
              <div className="p-6 rounded-2xl border border-red-200 dark:border-red-900/60 bg-red-50 dark:bg-red-950/30 text-center space-y-3">
                <p className="text-xs text-red-600 dark:text-red-400">
                  Failed to load collections: {collectionsError.message}
                </p>
                <button
                  type="button"
                  onClick={() => refetchCollections()}
                  className="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-red-600 text-white text-xs font-medium hover:bg-red-700 cursor-pointer"
                >
                  <RefreshCw size={13} />
                  <span>Retry</span>
                </button>
              </div>
            )}

            {/* Empty State with "Add Collection" button */}
            {!isLoadingCollections && !collectionsError && collections.length === 0 && (
              <div className="p-12 text-center rounded-2xl border border-dashed border-zinc-200 dark:border-zinc-800 bg-zinc-50 dark:bg-zinc-950 text-zinc-400 text-xs space-y-3">
                <div className="w-12 h-12 rounded-2xl bg-zinc-100 dark:bg-zinc-800 flex items-center justify-center mx-auto text-zinc-400">
                  <Bookmark size={20} />
                </div>
                <p className="font-bold text-sm text-zinc-700 dark:text-zinc-200">
                  No collections found
                </p>
                <p className="text-zinc-500 max-w-sm mx-auto">
                  {isOwner
                    ? "You haven't created any collections yet. Collections help you group parts, plan builds, and save favorites."
                    : `@${profile.id} hasn't published any public collections yet.`}
                </p>
                {isOwner && (
                  <div className="pt-2">
                    <button
                      type="button"
                      onClick={() => setIsCreateOpen(true)}
                      className="inline-flex items-center gap-1.5 px-4 py-2 rounded-xl bg-blue-600 hover:bg-blue-500 text-white font-semibold text-xs shadow-md shadow-blue-500/20 transition-colors cursor-pointer"
                    >
                      <Plus size={14} />
                      <span>Add Collection</span>
                    </button>
                  </div>
                )}
              </div>
            )}

            {/* Scrollable Collections Grid - Cards are clickable buttons opening the collection page */}
            {collections.length > 0 && (
              <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-5">
                {collections.map((col) => (
                  <div
                    key={col.id}
                    className="relative group rounded-2xl border border-zinc-200 dark:border-zinc-800 bg-white dark:bg-zinc-900/60 p-5 space-y-3 hover:border-blue-500/50 dark:hover:border-blue-500/50 hover:shadow-md transition-all shadow-2xs flex flex-col justify-between cursor-pointer"
                  >
                    {/* Stretched clickable card link opening the collection page */}
                    <Link
                      to={`/users/${profile.id}/collections/${col.id}`}
                      className="absolute inset-0 rounded-2xl focus:outline-hidden focus:ring-2 focus:ring-blue-500 z-0"
                      aria-label={`Open ${col.name} collection`}
                    />

                    <div className="space-y-2 relative z-10 pointer-events-none">
                      <div className="flex items-center justify-between">
                        <div className="flex items-center gap-2.5">
                          <div className="w-9 h-9 rounded-xl bg-blue-500/10 text-blue-500 flex items-center justify-center font-bold text-sm shrink-0 group-hover:scale-105 transition-transform">
                            {col.isSystemCollection ? (
                              <Heart size={16} className="fill-rose-500 text-rose-500" />
                            ) : (
                              <Bookmark size={16} />
                            )}
                          </div>
                          <div className="min-w-0">
                            <h4 className="font-bold text-sm text-zinc-900 dark:text-zinc-100 group-hover:text-blue-500 transition-colors truncate">
                              {col.name}
                            </h4>
                            <p className="text-[10px] text-zinc-400 truncate">
                              {col.isSystemCollection
                                ? "System Collection"
                                : `@${col.authorUsername}`}
                            </p>
                          </div>
                        </div>

                        <span
                          className={`inline-flex items-center gap-1 px-2 py-0.5 rounded text-[10px] font-semibold border shrink-0 ${
                            col.isPublic
                              ? "bg-emerald-500/10 text-emerald-600 dark:text-emerald-400 border-emerald-500/20"
                              : "bg-zinc-100 dark:bg-zinc-800 text-zinc-500 border-zinc-200 dark:border-zinc-700"
                          }`}
                        >
                          {col.isPublic ? <Globe size={10} /> : <Lock size={10} />}
                          <span>{col.isPublic ? "Public" : "Private"}</span>
                        </span>
                      </div>

                      {col.description && (
                        <p className="text-xs text-zinc-600 dark:text-zinc-400 line-clamp-2 leading-relaxed">
                          {col.description}
                        </p>
                      )}
                    </div>

                    <div className="flex items-center justify-between text-xs text-zinc-500 pt-3 border-t border-zinc-100 dark:border-zinc-800 relative z-10">
                      <span className="font-mono pointer-events-none">{col.itemCount} items</span>
                      {isOwner && !col.isSystemCollection && (
                        <button
                          type="button"
                          onClick={(e) => {
                            e.preventDefault();
                            e.stopPropagation();
                            handleDeleteCollection(col.id);
                          }}
                          disabled={deletingColId === col.id}
                          className="text-zinc-400 hover:text-rose-500 transition-colors p-1.5 rounded-lg hover:bg-rose-500/10 cursor-pointer pointer-events-auto"
                          title="Delete collection"
                          aria-label={`Delete ${col.name}`}
                        >
                          {deletingColId === col.id ? (
                            <Loader2 size={13} className="animate-spin text-rose-500" />
                          ) : (
                            <Trash2 size={13} />
                          )}
                        </button>
                      )}
                    </div>
                  </div>
                ))}
              </div>
            )}
          </section>
        )}
      </div>

      {/* Create Collection Modal */}
      {isCreateOpen && (
        <div
          role="dialog"
          aria-modal="true"
          aria-labelledby="create-col-modal-title"
          className="fixed inset-0 z-50 bg-black/50 backdrop-blur-xs flex items-center justify-center p-4 animate-in fade-in"
        >
          <div className="bg-white dark:bg-zinc-900 border border-zinc-200 dark:border-zinc-800 rounded-2xl w-full max-w-md shadow-2xl overflow-hidden p-6 space-y-4">
            <div className="flex items-center justify-between pb-3 border-b border-zinc-100 dark:border-zinc-800">
              <h3
                id="create-col-modal-title"
                className="font-bold text-base text-zinc-900 dark:text-zinc-100 flex items-center gap-2"
              >
                <Bookmark size={18} className="text-blue-500" />
                <span>Create Collection</span>
              </h3>
              <button
                type="button"
                onClick={() => setIsCreateOpen(false)}
                className="text-zinc-400 hover:text-zinc-600 dark:hover:text-zinc-200 p-1 rounded-lg cursor-pointer"
                aria-label="Close modal"
              >
                <X size={18} />
              </button>
            </div>

            <form onSubmit={handleCreateCollection} className="space-y-4">
              <div className="space-y-1.5">
                <label
                  htmlFor="new-col-name-input"
                  className="block text-xs font-semibold text-zinc-700 dark:text-zinc-300"
                >
                  Name
                </label>
                <input
                  id="new-col-name-input"
                  type="text"
                  value={newColName}
                  onChange={(e) => setNewColName(e.target.value)}
                  placeholder="e.g. Cinematic Freestyle 6S Rigs"
                  className="w-full rounded-xl border border-zinc-200 dark:border-zinc-700 bg-zinc-50 dark:bg-zinc-950 p-3 text-sm text-zinc-900 dark:text-zinc-100 placeholder-zinc-400 focus:outline-hidden focus:border-blue-500"
                  required
                  autoFocus
                />
              </div>

              <div className="space-y-1.5">
                <label
                  htmlFor="new-col-desc-input"
                  className="block text-xs font-semibold text-zinc-700 dark:text-zinc-300"
                >
                  Description (optional)
                </label>
                <textarea
                  id="new-col-desc-input"
                  rows={3}
                  value={newColDesc}
                  onChange={(e) => setNewColDesc(e.target.value)}
                  placeholder="Notes on what is collected here..."
                  className="w-full rounded-xl border border-zinc-200 dark:border-zinc-700 bg-zinc-50 dark:bg-zinc-950 p-3 text-sm text-zinc-900 dark:text-zinc-100 placeholder-zinc-400 focus:outline-hidden focus:border-blue-500"
                />
              </div>

              <div className="space-y-2 pt-1">
                <label className="block text-xs font-semibold text-zinc-700 dark:text-zinc-300">
                  Visibility
                </label>
                <div className="grid grid-cols-2 gap-2">
                  <button
                    type="button"
                    onClick={() => setNewColIsPublic(true)}
                    className={`flex items-center gap-2.5 p-3 rounded-xl border text-left cursor-pointer transition-all ${
                      newColIsPublic
                        ? "border-emerald-500/50 bg-emerald-500/10 text-emerald-700 dark:text-emerald-300 font-semibold"
                        : "border-zinc-200 dark:border-zinc-800 text-zinc-600 dark:text-zinc-400 hover:bg-zinc-50 dark:hover:bg-zinc-800/50"
                    }`}
                  >
                    <Globe
                      size={16}
                      className={newColIsPublic ? "text-emerald-500" : "text-zinc-400"}
                    />
                    <div>
                      <div className="text-xs">Public</div>
                      <div className="text-[10px] text-zinc-500 font-normal">
                        Visible on profile
                      </div>
                    </div>
                  </button>

                  <button
                    type="button"
                    onClick={() => setNewColIsPublic(false)}
                    className={`flex items-center gap-2.5 p-3 rounded-xl border text-left cursor-pointer transition-all ${
                      !newColIsPublic
                        ? "border-blue-500/50 bg-blue-500/10 text-blue-700 dark:text-blue-300 font-semibold"
                        : "border-zinc-200 dark:border-zinc-800 text-zinc-600 dark:text-zinc-400 hover:bg-zinc-50 dark:hover:bg-zinc-800/50"
                    }`}
                  >
                    <Lock
                      size={16}
                      className={!newColIsPublic ? "text-blue-500" : "text-zinc-400"}
                    />
                    <div>
                      <div className="text-xs">Private</div>
                      <div className="text-[10px] text-zinc-500 font-normal">
                        Only you can see it
                      </div>
                    </div>
                  </button>
                </div>
              </div>

              <div className="flex items-center justify-end gap-2 pt-3 border-t border-zinc-100 dark:border-zinc-800">
                <button
                  type="button"
                  onClick={() => setIsCreateOpen(false)}
                  className="px-4 py-2 rounded-xl text-xs font-semibold text-zinc-600 dark:text-zinc-300 hover:bg-zinc-100 dark:hover:bg-zinc-800 transition-colors cursor-pointer"
                >
                  Cancel
                </button>
                <button
                  type="submit"
                  disabled={isCreating || !newColName.trim()}
                  className="px-5 py-2 rounded-xl bg-blue-600 hover:bg-blue-500 text-white font-semibold text-xs transition-colors flex items-center gap-1.5 disabled:opacity-50 cursor-pointer"
                >
                  {isCreating && <Loader2 size={12} className="animate-spin" />}
                  <span>Create Collection</span>
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  );
}
