import { Link } from "react-router-dom";
import { useInfiniteQuery } from "@connectrpc/connect-query";
import { listBuilds } from "../gen/quadsmith/build-BuildService_connectquery";
import { BuildCard } from "../components/BuildCard";
import { useMemo, useRef, useEffect } from "react";
import { Hammer, RefreshCw, CheckCircle, ArrowRight, GalleryVertical } from "lucide-react";
import { useDocumentMeta } from "../hooks/useDocumentMeta";
import { getCollectionColor } from "../lib/collectionColors";

export function BuildFeedPage() {
  useDocumentMeta({
    title: "Build Feed — Community & Curated Quadcopters | Quadsmith",
    description:
      "Explore community-shared and curated FPV drone builds. Inspect bills of materials, motor and battery pairings, and simulated flight performance.",
    image: "/og-default.png",
  });

  const sentinelRef = useRef<HTMLDivElement>(null);

  const { data, isLoading, isFetchingNextPage, hasNextPage, fetchNextPage, error, refetch } =
    useInfiniteQuery(
      listBuilds,
      { pageSize: 6, pageToken: "" },
      {
        pageParamKey: "pageToken",
        getNextPageParam: (lastPage) => lastPage.nextPageToken || undefined,
      },
    );

  const builds = useMemo(() => {
    return data?.pages.flatMap((page) => page.builds) ?? [];
  }, [data]);

  // Infinite scroll auto-loading intersection observer
  useEffect(() => {
    const sentinel = sentinelRef.current;
    if (!sentinel || typeof IntersectionObserver === "undefined") return;

    const observer = new IntersectionObserver(
      (entries) => {
        if (entries[0].isIntersecting && hasNextPage && !isFetchingNextPage) {
          fetchNextPage();
        }
      },
      { rootMargin: "250px" },
    );

    observer.observe(sentinel);
    return () => observer.disconnect();
  }, [hasNextPage, isFetchingNextPage, fetchNextPage]);

  const buildColor = getCollectionColor("builds");

  return (
    <div className="max-w-3xl mx-auto">
      {/* Page Header */}
      <div className="mb-6 flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4 pb-6 border-b border-zinc-200 dark:border-zinc-800">
        <div>
          <div className="flex items-center gap-2.5 mb-1.5">
            <div
              className={`w-8 h-8 rounded-lg flex items-center justify-center bg-blue-500/10 ${buildColor.textClass} border border-blue-500/20`}
            >
              <GalleryVertical size={18} />
            </div>
            <h1 className="text-2xl sm:text-3xl font-extrabold tracking-tight text-zinc-900 dark:text-zinc-50">
              Build Feed
            </h1>
          </div>
          <p className="text-xs sm:text-sm text-zinc-600 dark:text-zinc-400 max-w-xl leading-relaxed">
            Explore verified community and curated quadcopter builds. Inspect detailed Bills of
            Materials, component compatibility, and simulated flight performance.
          </p>
        </div>

        <Link
          to="/builds/new"
          className="inline-flex items-center justify-center gap-2 px-4 py-2.5 rounded-xl bg-orange-600 hover:bg-orange-500 text-white font-semibold text-xs sm:text-sm shadow-xs transition-colors shrink-0 self-start sm:self-center"
        >
          <Hammer size={16} className="shrink-0" />
          <span>New Build in The Forge</span>
          <ArrowRight size={14} />
        </Link>
      </div>

      {/* Social-Style Auto-Loading Builds Stream */}
      <section aria-label="Builds Feed">
        <div className="flex items-center justify-between mb-4 px-1">
          <span className="text-xs font-semibold uppercase tracking-wider text-zinc-500 dark:text-zinc-400">
            All Builds
          </span>
          {builds.length > 0 && (
            <span className="text-xs text-zinc-500 font-medium">
              {builds.length} {builds.length === 1 ? "build" : "builds"} loaded
            </span>
          )}
        </div>

        {/* Loading Initial Skeleton */}
        {isLoading && (
          <div className="space-y-6">
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
                <div className="h-32 bg-zinc-100 dark:bg-zinc-800/40 rounded-xl" />
                <div className="h-4 bg-zinc-100 dark:bg-zinc-800/60 rounded w-3/4" />
              </div>
            ))}
          </div>
        )}

        {/* Feed Error State */}
        {error && (
          <div className="p-6 rounded-2xl border border-red-200 dark:border-red-900/60 bg-red-50 dark:bg-red-950/30 text-center">
            <p className="text-sm text-red-600 dark:text-red-400 mb-3">
              Failed to load builds: {error.message}
            </p>
            <button
              type="button"
              onClick={() => refetch()}
              className="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-red-600 text-white text-xs font-medium hover:bg-red-700 cursor-pointer"
            >
              <RefreshCw size={13} />
              <span>Retry</span>
            </button>
          </div>
        )}

        {/* Empty State */}
        {!isLoading && !error && builds.length === 0 && (
          <div className="p-8 rounded-2xl border border-zinc-200 dark:border-zinc-800 text-center bg-white dark:bg-zinc-900">
            <p className="text-zinc-500 text-sm">No builds found in the catalog.</p>
          </div>
        )}

        {/* Build Feed Stream */}
        <div className="space-y-6">
          {builds.map((b) => (
            <BuildCard key={b.uuid || b.id} build={b} />
          ))}
        </div>

        {/* Auto-loading Sentinel & Indicator */}
        <div ref={sentinelRef} className="py-8 text-center" aria-live="polite">
          {isFetchingNextPage && (
            <div className="inline-flex items-center gap-2 text-xs text-zinc-500 font-medium">
              <RefreshCw size={14} className="animate-spin text-blue-600" />
              <span>Loading more builds...</span>
            </div>
          )}

          {!isLoading && !hasNextPage && builds.length > 0 && (
            <div className="inline-flex items-center gap-2 text-xs text-zinc-400 dark:text-zinc-500 font-medium">
              <CheckCircle size={14} className="text-emerald-500" />
              <span>You&apos;ve caught up with all builds!</span>
            </div>
          )}
        </div>
      </section>
    </div>
  );
}
