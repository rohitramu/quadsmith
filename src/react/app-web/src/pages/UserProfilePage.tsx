import { useState, useEffect, useMemo, useRef } from "react";
import { useParams, Link, useSearchParams, useNavigate } from "react-router-dom";
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
  Heart,
  RefreshCw,
  CheckCircle,
  Trash2,
  X,
  Cpu,
} from "lucide-react";
import { getPublicProfile } from "@quadsmith/shared/gen/quadsmith/api/user-UserService_connectquery";
import { listBuilds } from "@quadsmith/shared/gen/quadsmith/api/build-BuildService_connectquery";
import { search } from "@quadsmith/shared/gen/quadsmith/api/search-SearchService_connectquery";
import {
  listUserCollections,
  createCollection,
  deleteCollection,
} from "@quadsmith/shared/gen/quadsmith/api/social-SocialService_connectquery";
import { ALL_COLLECTIONS, getCollectionPath } from "../lib/hardwareCollections";
import { ComponentFormModal } from "../components/ComponentFormModal";
import {
  getAvatarPalette,
  getInitial,
  CollectionBadge,
  CollectionIcon,
  getCollectionColor,
} from "@quadsmith/shared";
import { BuildCard } from "../components/BuildCard";
import { UserBadge } from "../components/UserBadge";
import { useAuth } from "../context/AuthContext";
import { useDocumentMeta } from "@quadsmith/shared";

export function UserProfilePage() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const { user } = useAuth();
  const queryClient = useQueryClient();
  const [searchParams, setSearchParams] = useSearchParams();
  const sentinelRef = useRef<HTMLDivElement>(null);

  // Modal & action state for collections
  const [isCreateOpen, setIsCreateOpen] = useState(false);
  const [newColName, setNewColName] = useState("");
  const [newColDesc, setNewColDesc] = useState("");
  const [isCreating, setIsCreating] = useState(false);
  const [deletingColId, setDeletingColId] = useState<string | null>(null);

  // Component state
  const [isComponentModalOpen, setIsComponentModalOpen] = useState(false);
  const [createModalCollectionId, setCreateModalCollectionId] = useState("motors");
  const [selectedEditComponent, setSelectedEditComponent] = useState<any>(null);

  const activeTab =
    searchParams.get("tab") === "collections"
      ? "collections"
      : searchParams.get("tab") === "components"
        ? "components"
        : "builds";

  const handleTabChange = (tab: "builds" | "collections" | "components") => {
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

  const componentSelectors = useMemo(() => {
    return ALL_COLLECTIONS.map((c) => ({
      path: getCollectionPath(c),
      filter: `author_username == "${id}"`,
    }));
  }, [id]);

  const {
    data: componentsSearchData,
    isLoading: isLoadingComponents,
    refetch: refetchComponents,
  } = useQuery(
    search,
    {
      query: "",
      selectors: componentSelectors,
      limit: 100,
    },
    { enabled: !!id },
  );

  const contributedComponents = componentsSearchData?.results || [];

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
      });
      setNewColName("");
      setNewColDesc("");
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
          className="inline-block px-4 py-2 rounded-xl text-xs font-semibold bg-zinc-100 text-zinc-900"
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
      <div className="p-8 rounded-2xl border border-zinc-800 bg-zinc-900 shadow-sm space-y-6">
        <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-6">
          <div className="flex items-center gap-5">
            {profile.avatarUrl ? (
              <img
                src={profile.avatarUrl}
                alt={profile.id}
                className="w-20 h-20 rounded-full object-cover border-2 border-zinc-800 shadow-md"
              />
            ) : (
              <div
                className={`w-20 h-20 rounded-full border-2 flex items-center justify-center font-bold text-3xl shadow-md ${palette.badgeClass}`}
              >
                {initial}
              </div>
            )}
            <div className="space-y-1">
              <h1 className="flex items-center">
                <UserBadge username={profile.id} size="lg" withHoverCard={false} to={null} />
              </h1>
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
          <div className="flex items-center gap-4 bg-zinc-950 p-4 rounded-xl border border-zinc-800 flex-wrap">
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

            <div className="h-8 w-px bg-zinc-800" />

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

            <div className="h-8 w-px bg-zinc-800" />

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
          <div className="p-4 rounded-xl bg-zinc-950 border border-zinc-800 text-xs text-zinc-300 leading-relaxed">
            {profile.bio}
          </div>
        )}
      </div>

      {/* Larger Tabbed Section */}
      <div className="p-4 sm:p-6 lg:p-8 rounded-2xl border border-zinc-800 bg-zinc-900 shadow-xs space-y-6">
        {/* Tab Navigation Bar */}
        <div
          role="tablist"
          aria-label="User profile content"
          className="flex items-center gap-1 sm:gap-2 border-b border-zinc-800 pb-px overflow-x-auto whitespace-nowrap scrollbar-thin"
        >
          <button
            role="tab"
            id="tab-builds"
            aria-controls="panel-builds"
            aria-selected={activeTab === "builds"}
            type="button"
            onClick={() => handleTabChange("builds")}
            className={`flex items-center gap-1.5 sm:gap-2 px-3.5 sm:px-5 py-2.5 sm:py-3 border-b-2 font-bold text-xs sm:text-sm transition-colors cursor-pointer shrink-0 ${
              activeTab === "builds"
                ? "border-emerald-500 text-emerald-400"
                : "border-transparent text-zinc-400 hover:text-zinc-200 hover:border-zinc-700"
            }`}
          >
            <Hammer size={15} />
            <span>Builds</span>
            <span
              className={`px-1.5 sm:px-2 py-0.5 rounded-full text-[10px] sm:text-[11px] font-mono transition-colors ${
                activeTab === "builds"
                  ? "bg-emerald-500/10 text-emerald-400"
                  : "bg-zinc-800 text-zinc-400"
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
            className={`flex items-center gap-1.5 sm:gap-2 px-3.5 sm:px-5 py-2.5 sm:py-3 border-b-2 font-bold text-xs sm:text-sm transition-colors cursor-pointer shrink-0 ${
              activeTab === "collections"
                ? "border-emerald-500 text-emerald-400"
                : "border-transparent text-zinc-400 hover:text-zinc-200 hover:border-zinc-700"
            }`}
          >
            <Bookmark size={15} />
            <span>Collections</span>
            <span
              className={`px-1.5 sm:px-2 py-0.5 rounded-full text-[10px] sm:text-[11px] font-mono transition-colors ${
                activeTab === "collections"
                  ? "bg-emerald-500/10 text-emerald-400"
                  : "bg-zinc-800 text-zinc-400"
              }`}
            >
              {displayColCount}
            </span>
          </button>

          <button
            role="tab"
            id="tab-components"
            aria-controls="panel-components"
            aria-selected={activeTab === "components"}
            type="button"
            onClick={() => handleTabChange("components")}
            className={`flex items-center gap-1.5 sm:gap-2 px-3.5 sm:px-5 py-2.5 sm:py-3 border-b-2 font-bold text-xs sm:text-sm transition-colors cursor-pointer shrink-0 ${
              activeTab === "components"
                ? "border-emerald-500 text-emerald-400"
                : "border-transparent text-zinc-400 hover:text-zinc-200 hover:border-zinc-700"
            }`}
          >
            <Cpu size={15} />
            <span>Components</span>
            <span
              className={`px-1.5 sm:px-2 py-0.5 rounded-full text-[10px] sm:text-[11px] font-mono transition-colors ${
                activeTab === "components"
                  ? "bg-emerald-500/10 text-emerald-400"
                  : "bg-zinc-800 text-zinc-400"
              }`}
            >
              {contributedComponents.length}
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
                <h3 className="text-base font-bold text-zinc-100 flex items-center gap-2">
                  <Hammer size={18} className="text-emerald-500" />
                  <span>Authored Builds</span>
                </h3>
                <p className="text-xs text-zinc-400 mt-0.5">
                  Verified quadcopter builds and component bills of materials engineered by @
                  {profile.id}.
                </p>
              </div>

              {isOwner && builds.length > 0 && (
                <Link
                  to="/build/new"
                  aria-label="Add Build"
                  className="inline-flex items-center gap-1.5 px-3.5 py-1.5 rounded-xl bg-emerald-600 hover:bg-emerald-500 text-white font-semibold text-xs shadow-md shadow-emerald-500/20 transition-colors self-start sm:self-auto cursor-pointer"
                >
                  <Plus size={14} />
                  <span>+ Add New</span>
                </Link>
              )}
            </div>

            {/* Loading Skeleton */}
            {isLoadingBuilds && builds.length === 0 && (
              <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
                {[1, 2].map((n) => (
                  <div
                    key={n}
                    className="rounded-2xl border border-zinc-800 p-6 bg-zinc-900 animate-pulse space-y-4"
                  >
                    <div className="flex items-center gap-3">
                      <div className="w-10 h-10 rounded-xl bg-zinc-800" />
                      <div className="space-y-2 flex-1">
                        <div className="h-4 bg-zinc-800 rounded w-1/3" />
                        <div className="h-3 bg-zinc-800/60 rounded w-1/4" />
                      </div>
                    </div>
                    <div className="h-28 bg-zinc-800/40 rounded-xl" />
                    <div className="h-4 bg-zinc-800/60 rounded w-3/4" />
                  </div>
                ))}
              </div>
            )}

            {/* Error Alert */}
            {buildsError && (
              <div className="p-6 rounded-2xl border border-red-900/60 bg-red-950/30 text-center space-y-3">
                <p className="text-xs text-red-400">Failed to load builds: {buildsError.message}</p>
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
              <div className="p-12 text-center rounded-2xl border border-dashed border-zinc-800 bg-zinc-950 text-zinc-400 text-xs space-y-3">
                <div className="w-12 h-12 rounded-2xl bg-zinc-800 flex items-center justify-center mx-auto text-zinc-400">
                  <Hammer size={20} />
                </div>
                <p className="font-bold text-sm text-zinc-200">No published builds yet</p>
                <p className="text-zinc-500 max-w-sm mx-auto">
                  When @{profile.id} creates and publishes quadcopter builds, they will appear here.
                </p>
                {isOwner && (
                  <div className="pt-2">
                    <Link
                      to="/build/new"
                      aria-label="Add Build"
                      className="inline-flex items-center gap-1.5 px-3.5 py-1.5 rounded-xl bg-emerald-600 hover:bg-emerald-500 text-white font-semibold text-xs shadow-md shadow-emerald-500/20 transition-colors cursor-pointer"
                    >
                      <Plus size={14} />
                      <span>+ Add New</span>
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
                <div className="inline-flex items-center gap-2 text-xs text-zinc-500 font-medium">
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
            <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4 pb-2">
              <div>
                <h3 className="text-base font-bold text-zinc-100 flex items-center gap-2">
                  <Bookmark size={18} className="text-blue-500" />
                  <span>Curated Collections</span>
                </h3>
                <p className="text-xs text-zinc-400 mt-0.5">
                  Curated quadcopter builds, hardware wishlists, and component pairings.
                </p>
              </div>

              {isOwner && collections.length > 0 && (
                <button
                  type="button"
                  onClick={() => setIsCreateOpen(true)}
                  className="inline-flex items-center gap-1.5 px-3.5 py-1.5 rounded-xl bg-emerald-600 hover:bg-emerald-500 text-white font-semibold text-xs shadow-md shadow-emerald-500/20 transition-colors self-start sm:self-auto cursor-pointer"
                  title="Add Collection"
                  aria-label="Add Collection"
                >
                  <Plus size={14} />
                  <span>+ Add New</span>
                </button>
              )}
            </div>

            {/* Loading Skeleton */}
            {isLoadingCollections && collections.length === 0 && (
              <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-5">
                {[1, 2, 3].map((n) => (
                  <div
                    key={n}
                    className="rounded-2xl border border-zinc-800 p-5 bg-zinc-900 animate-pulse space-y-3"
                  >
                    <div className="h-4 bg-zinc-800 rounded w-1/2" />
                    <div className="h-10 bg-zinc-800/60 rounded" />
                    <div className="h-3 bg-zinc-800 rounded w-1/4 pt-2" />
                  </div>
                ))}
              </div>
            )}

            {/* Error Alert */}
            {collectionsError && (
              <div className="p-6 rounded-2xl border border-red-900/60 bg-red-950/30 text-center space-y-3">
                <p className="text-xs text-red-400">
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
              <div className="p-12 text-center rounded-2xl border border-dashed border-zinc-800 bg-zinc-950 text-zinc-400 text-xs space-y-3">
                <div className="w-12 h-12 rounded-2xl bg-zinc-800 flex items-center justify-center mx-auto text-zinc-400">
                  <Bookmark size={20} />
                </div>
                <p className="font-bold text-sm text-zinc-200">No collections found</p>
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
                      className="inline-flex items-center gap-1.5 px-3.5 py-1.5 rounded-xl bg-emerald-600 hover:bg-emerald-500 text-white font-semibold text-xs shadow-md shadow-emerald-500/20 transition-colors cursor-pointer"
                      aria-label="Add Collection"
                    >
                      <Plus size={14} />
                      <span>+ Add New</span>
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
                    onClick={(e) => {
                      if ((e.target as HTMLElement).closest("button")) {
                        return;
                      }
                      navigate(`/users/${profile.id}/collections/${col.id}`);
                    }}
                    className="relative group rounded-2xl border border-zinc-800 bg-zinc-900/60 p-5 gap-3 hover:border-blue-500/50 hover:shadow-md transition-all shadow-2xs flex flex-col justify-between cursor-pointer"
                  >
                    {/* Stretched clickable card link opening the collection page across entire height and width */}
                    <Link
                      to={`/users/${profile.id}/collections/${col.id}`}
                      className="absolute inset-0 rounded-2xl focus:outline-hidden focus:ring-2 focus:ring-blue-500 z-10"
                      aria-label={`Open ${col.name} collection`}
                    />

                    <div className="space-y-2 pointer-events-none">
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
                            <h4 className="font-bold text-sm text-zinc-100 group-hover:text-blue-500 transition-colors truncate">
                              {col.name}
                            </h4>
                            <div className="text-[10px] text-zinc-400 truncate mt-0.5">
                              {col.isSystemCollection ? (
                                <span>System Collection</span>
                              ) : (
                                <UserBadge username={col.authorUsername} size="xs" />
                              )}
                            </div>
                          </div>
                        </div>
                      </div>

                      {col.description && (
                        <p className="text-xs text-zinc-400 line-clamp-2 leading-relaxed">
                          {col.description}
                        </p>
                      )}
                    </div>

                    {/* Bottom footer with item count and delete button */}
                    <div className="flex items-center justify-between text-xs text-zinc-500 pt-3 border-t border-zinc-800 pointer-events-none">
                      <span className="font-mono">{col.itemCount} items</span>
                      {isOwner && !col.isSystemCollection && (
                        <button
                          type="button"
                          onClick={(e) => {
                            e.preventDefault();
                            e.stopPropagation();
                            handleDeleteCollection(col.id);
                          }}
                          disabled={deletingColId === col.id}
                          className="relative z-20 pointer-events-auto text-zinc-400 hover:text-rose-500 transition-colors p-1.5 rounded-lg hover:bg-rose-500/10 cursor-pointer"
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

        {/* Tab 3: Components Panel */}
        {activeTab === "components" && (
          <section
            role="tabpanel"
            id="panel-components"
            aria-labelledby="tab-components"
            className="space-y-6 animate-in fade-in duration-150"
          >
            <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4 pb-2">
              <div>
                <h3 className="text-base font-bold text-zinc-100 flex items-center gap-2">
                  <Cpu size={18} className="text-blue-500" />
                  <span>Contributed Components</span>
                </h3>
                <p className="text-xs text-zinc-400 mt-0.5">
                  Hardware, gear, and software specifications cataloged by @{id}.
                </p>
              </div>

              {isOwner && contributedComponents.length > 0 && (
                <button
                  type="button"
                  onClick={() => {
                    setSelectedEditComponent(null);
                    setCreateModalCollectionId("motors");
                    setIsComponentModalOpen(true);
                  }}
                  className="inline-flex items-center gap-1.5 px-3.5 py-1.5 rounded-xl bg-emerald-600 hover:bg-emerald-500 text-white font-semibold text-xs shadow-md shadow-emerald-500/20 transition-colors self-start sm:self-auto cursor-pointer"
                  aria-label="Add Component"
                >
                  <Plus size={14} />
                  <span>+ Add New</span>
                </button>
              )}
            </div>

            {isLoadingComponents ? (
              <div className="flex h-48 w-full items-center justify-center text-zinc-400">
                <Loader2 size={24} className="animate-spin text-blue-500" />
              </div>
            ) : contributedComponents.length === 0 ? (
              <div className="text-center py-16 px-4 rounded-2xl border border-dashed border-zinc-800 space-y-3">
                <div className="w-12 h-12 rounded-full bg-zinc-800 flex items-center justify-center mx-auto text-zinc-400">
                  <Cpu size={24} />
                </div>
                <h4 className="text-sm font-bold text-zinc-100">No Components Contributed Yet</h4>
                <p className="text-xs text-zinc-500 max-w-sm mx-auto">
                  {isOwner
                    ? "Contribute custom motors, frames, electronics, or firmware to Quadsmith's catalog."
                    : `@${id} has not contributed any catalog components yet.`}
                </p>
                {isOwner && (
                  <div className="pt-2">
                    <button
                      type="button"
                      onClick={() => {
                        setSelectedEditComponent(null);
                        setCreateModalCollectionId("motors");
                        setIsComponentModalOpen(true);
                      }}
                      className="inline-flex items-center gap-1.5 px-3.5 py-1.5 rounded-xl bg-emerald-600 hover:bg-emerald-500 text-white font-semibold text-xs shadow-md shadow-emerald-500/20 transition-colors cursor-pointer"
                      aria-label="Add Component"
                    >
                      <Plus size={14} />
                      <span>+ Add New</span>
                    </button>
                  </div>
                )}
              </div>
            ) : (
              <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
                {contributedComponents.map((item) => {
                  const collectionKey = item.path
                    ? item.path.split("/").pop() || "motors"
                    : "motors";
                  const colColor = getCollectionColor(collectionKey);
                  return (
                    <div
                      key={item.uuid || item.id}
                      className="p-4 rounded-xl border border-zinc-800 bg-zinc-900/50 hover:bg-zinc-800/50 transition-all flex flex-col justify-between space-y-3 group"
                    >
                      <div className="space-y-2">
                        <div className="flex items-center justify-between">
                          <CollectionBadge
                            collection={collectionKey}
                            label={item.collectionName || collectionKey}
                            variant="pill"
                            size="xs"
                          />
                          {isOwner && (
                            <button
                              type="button"
                              onClick={() => {
                                setSelectedEditComponent(item);
                                setCreateModalCollectionId(item.path.split("/").pop() || "motors");
                                setIsComponentModalOpen(true);
                              }}
                              className="text-xs text-blue-400 hover:underline cursor-pointer"
                            >
                              Edit
                            </button>
                          )}
                        </div>

                        <div className="flex items-center gap-3">
                          {item.primaryDisplayImage ? (
                            <img
                              src={item.primaryDisplayImage}
                              alt={item.name}
                              className="w-10 h-10 rounded-lg object-cover bg-zinc-800 border border-zinc-700 shrink-0"
                            />
                          ) : (
                            <div
                              className={`w-10 h-10 rounded-lg flex items-center justify-center shrink-0 border border-zinc-700/60 ${colColor.iconBgClass} ${colColor.textClass}`}
                            >
                              <CollectionIcon collection={collectionKey} color={true} size={18} />
                            </div>
                          )}
                          <div className="min-w-0 flex-1">
                            <Link
                              to={`/${item.path}/${item.id}`}
                              className="font-bold text-sm text-zinc-100 hover:underline truncate block"
                            >
                              {item.name}
                            </Link>
                            {item.description && (
                              <p className="text-xs text-zinc-500 line-clamp-1">
                                {item.description}
                              </p>
                            )}
                          </div>
                        </div>
                      </div>

                      <div className="pt-2 border-t border-zinc-800/80 flex items-center justify-between text-[11px] text-zinc-400 font-mono">
                        <span>{item.id}</span>
                        <Link
                          to={`/${item.path}/${item.id}`}
                          className="text-blue-400 font-sans font-medium hover:underline"
                        >
                          View &rarr;
                        </Link>
                      </div>
                    </div>
                  );
                })}
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
          <div className="bg-zinc-900 border border-zinc-800 rounded-2xl w-full max-w-md shadow-2xl overflow-hidden p-6 space-y-4">
            <div className="flex items-center justify-between pb-3 border-b border-zinc-800">
              <h3
                id="create-col-modal-title"
                className="font-bold text-base text-zinc-100 flex items-center gap-2"
              >
                <Bookmark size={18} className="text-blue-500" />
                <span>Create Collection</span>
              </h3>
              <button
                type="button"
                onClick={() => setIsCreateOpen(false)}
                className="text-zinc-400 hover:text-zinc-200 p-1 rounded-lg cursor-pointer"
                aria-label="Close modal"
              >
                <X size={18} />
              </button>
            </div>

            <form onSubmit={handleCreateCollection} className="space-y-4">
              <div className="space-y-1.5">
                <label
                  htmlFor="new-col-name-input"
                  className="block text-xs font-semibold text-zinc-300"
                >
                  Name
                </label>
                <input
                  id="new-col-name-input"
                  type="text"
                  value={newColName}
                  onChange={(e) => setNewColName(e.target.value)}
                  placeholder="e.g. Cinematic Freestyle 6S Rigs"
                  className="w-full rounded-xl border border-zinc-700 bg-zinc-950 p-3 text-sm text-zinc-100 placeholder-zinc-400 focus:outline-hidden focus:border-blue-500"
                  required
                  autoFocus
                />
              </div>

              <div className="space-y-1.5">
                <label
                  htmlFor="new-col-desc-input"
                  className="block text-xs font-semibold text-zinc-300"
                >
                  Description (optional)
                </label>
                <textarea
                  id="new-col-desc-input"
                  rows={3}
                  value={newColDesc}
                  onChange={(e) => setNewColDesc(e.target.value)}
                  placeholder="Notes on what is collected here..."
                  className="w-full rounded-xl border border-zinc-700 bg-zinc-950 p-3 text-sm text-zinc-100 placeholder-zinc-400 focus:outline-hidden focus:border-blue-500"
                />
              </div>

              <div className="flex items-center justify-end gap-2 pt-3 border-t border-zinc-800">
                <button
                  type="button"
                  onClick={() => setIsCreateOpen(false)}
                  className="px-4 py-2 rounded-xl text-xs font-semibold text-zinc-300 hover:bg-zinc-800 transition-colors cursor-pointer"
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

      {/* Component Form Modal */}
      {isComponentModalOpen && (
        <ComponentFormModal
          isOpen={isComponentModalOpen}
          onClose={() => {
            setIsComponentModalOpen(false);
            setSelectedEditComponent(null);
          }}
          collectionId={createModalCollectionId}
          initialData={selectedEditComponent}
          onSuccess={() => {
            refetchComponents();
          }}
        />
      )}
    </div>
  );
}
