import { Link } from "react-router-dom";
import { useInfiniteQuery } from "@connectrpc/connect-query";
import { listBuilds } from "../gen/quadsmith/build-BuildService_connectquery";
import { BuildCard } from "../components/BuildCard";
import { useMemo, useRef, useEffect } from "react";
import { Sparkles, Layers, RefreshCw, CheckCircle, ArrowRight } from "lucide-react";
import { useDocumentMeta } from "../hooks/useDocumentMeta";

export function HomePage() {
  useDocumentMeta({
    title: "Quadsmith — FPV Drone Component Platform",
    description:
      "Open-source FPV drone engineering platform. Calculate thrust-to-weight, simulate flight times, check hardware compatibility, and explore quadcopter builds.",
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

  return (
    <div className="max-w-3xl mx-auto">
      {/* Top Welcome & Navigation Banner */}
      <section className="mb-8 p-6 rounded-2xl border border-zinc-200 dark:border-zinc-800 bg-gradient-to-br from-zinc-50 to-white dark:from-zinc-900/60 dark:to-zinc-950 shadow-xs">
        <h1 className="text-3xl sm:text-4xl font-extrabold tracking-tight mb-3 text-zinc-900 dark:text-zinc-50">
          Welcome to Quadsmith
        </h1>
        <p className="text-sm sm:text-base text-zinc-600 dark:text-zinc-400 mb-6 leading-relaxed">
          Quadsmith is the ultimate hardware data sourcing and component browser for FPV drone
          builders. Explore verified builds and inspect performance evaluations below, or design
          your dream drone and let Quadsmith automatically check for component compatibility and
          estimated performance.
        </p>

        <div className="flex flex-wrap items-center gap-3">
          <Link
            to="/components/hardware"
            className="inline-flex items-center gap-2 px-4 py-2 rounded-xl bg-blue-600 hover:bg-blue-700 text-white font-medium text-xs sm:text-sm shadow-xs transition-colors"
          >
            <Layers size={16} />
            <span>Browse Hardware</span>
            <ArrowRight size={14} />
          </Link>
          <Link
            to="/builds/new"
            className="inline-flex items-center gap-2 px-4 py-2 rounded-xl border border-zinc-200 dark:border-zinc-800 bg-white dark:bg-zinc-900 hover:bg-zinc-50 dark:hover:bg-zinc-800/80 text-zinc-800 dark:text-zinc-200 font-medium text-xs sm:text-sm shadow-xs transition-colors"
          >
            <Sparkles size={16} className="text-blue-600 dark:text-blue-400" />
            <span>Build Wizard</span>
            <ArrowRight size={14} />
          </Link>
        </div>
      </section>

      {/* Social-Style Auto-Loading Builds Feed */}
      <section aria-label="Builds Feed">
        <div className="flex items-center justify-between mb-4 px-1">
          <div className="flex items-center gap-2">
            <Sparkles className="text-blue-600 dark:text-blue-400" size={18} />
            <h2 className="text-lg font-bold text-zinc-900 dark:text-zinc-100">
              Community & Curated Builds
            </h2>
          </div>
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
