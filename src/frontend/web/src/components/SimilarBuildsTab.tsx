import { useQuery } from "@connectrpc/connect-query";
import { listSimilarBuilds } from "../gen/quadsmith/build-BuildService_connectquery";
import { BuildCard } from "./BuildCard";
import { Link } from "react-router-dom";
import { Layers, Plus, Compass, Sparkles } from "lucide-react";

export interface SimilarBuildsTabProps {
  buildId: string;
}

export function SimilarBuildsTab({ buildId }: SimilarBuildsTabProps) {
  const { data, isLoading, error } = useQuery(
    listSimilarBuilds,
    { buildId, limit: 6 },
    { enabled: !!buildId, staleTime: 60_000 },
  );

  const similarBuilds = data?.similarBuilds || [];

  return (
    <div className="space-y-6 animate-in fade-in duration-150">
      {/* Weight Breakdown & Information Banner */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4 p-5 rounded-2xl border border-zinc-200 dark:border-zinc-800 bg-white dark:bg-zinc-900/60 shadow-2xs">
        <div className="space-y-1">
          <h3 className="font-bold text-sm text-zinc-900 dark:text-zinc-100 flex items-center gap-2">
            <Sparkles size={16} className="text-blue-500" />
            <span>Hardware Similarity Rating</span>
          </h3>
          <p className="text-xs text-zinc-500 dark:text-zinc-400">
            Similar builds are rated by shared components and specifications with this quadcopter.
          </p>
        </div>

        {/* Priority Weight Badges */}
        <div className="flex items-center gap-1.5 flex-wrap">
          <span className="px-2.5 py-1 rounded-lg text-xs font-semibold bg-blue-500/10 text-blue-600 dark:text-blue-400 border border-blue-500/20">
            Frame: 50%
          </span>
          <span className="px-2.5 py-1 rounded-lg text-xs font-semibold bg-indigo-500/10 text-indigo-600 dark:text-indigo-400 border border-indigo-500/20">
            VTX: 25%
          </span>
          <span className="px-2.5 py-1 rounded-lg text-xs font-semibold bg-emerald-500/10 text-emerald-600 dark:text-emerald-400 border border-emerald-500/20">
            Camera: 15%
          </span>
          <span className="px-2.5 py-1 rounded-lg text-xs font-semibold bg-purple-500/10 text-purple-600 dark:text-purple-400 border border-purple-500/20">
            Motors: 10%
          </span>
        </div>
      </div>

      {/* Loading Skeleton */}
      {isLoading && (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
          {[1, 2, 3].map((n) => (
            <div
              key={n}
              className="rounded-2xl border border-zinc-200 dark:border-zinc-800 p-6 bg-white dark:bg-zinc-900 animate-pulse space-y-4"
            >
              <div className="flex items-center justify-between">
                <div className="w-10 h-10 rounded-xl bg-zinc-200 dark:bg-zinc-800" />
                <div className="h-5 bg-zinc-200 dark:bg-zinc-800 rounded w-20" />
              </div>
              <div className="h-4 bg-zinc-200 dark:bg-zinc-800 rounded w-2/3" />
              <div className="h-20 bg-zinc-100 dark:bg-zinc-800/60 rounded" />
            </div>
          ))}
        </div>
      )}

      {/* Error State */}
      {error && !isLoading && (
        <div
          role="alert"
          className="p-5 rounded-2xl border border-red-200 dark:border-red-900/60 bg-red-50 dark:bg-red-950/40 text-red-700 dark:text-red-300 text-xs"
        >
          Failed to load similar builds: {error.message}
        </div>
      )}

      {/* Empty State */}
      {!isLoading && !error && similarBuilds.length === 0 && (
        <div className="p-12 text-center rounded-2xl border border-dashed border-zinc-200 dark:border-zinc-800 bg-white dark:bg-zinc-900/40 space-y-4">
          <div className="w-12 h-12 rounded-2xl bg-zinc-100 dark:bg-zinc-800 flex items-center justify-center mx-auto text-zinc-400">
            <Layers size={24} />
          </div>
          <div className="space-y-1">
            <h4 className="font-bold text-sm text-zinc-900 dark:text-zinc-100">
              No Similar Builds Found
            </h4>
            <p className="text-xs text-zinc-500 dark:text-zinc-400 max-w-sm mx-auto">
              No other builds in the database share 25% or more of this quadcopter&apos;s
              components. Be the first to build a variant!
            </p>
          </div>
          <div className="flex items-center justify-center gap-3 pt-2">
            <Link
              to="/wizard"
              className="px-4 py-2 rounded-xl bg-blue-600 hover:bg-blue-500 text-white font-semibold text-xs transition-colors flex items-center gap-1.5 shadow-md shadow-blue-500/20"
            >
              <Plus size={14} />
              <span>Create Build Variant</span>
            </Link>
            <Link
              to="/"
              className="px-4 py-2 rounded-xl border border-zinc-200 dark:border-zinc-800 hover:bg-zinc-100 dark:hover:bg-zinc-800 text-zinc-700 dark:text-zinc-300 font-semibold text-xs transition-colors flex items-center gap-1.5"
            >
              <Compass size={14} />
              <span>Explore Build Feed</span>
            </Link>
          </div>
        </div>
      )}

      {/* Results Grid */}
      {!isLoading && !error && similarBuilds.length > 0 && (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
          {similarBuilds.map(
            (sb, idx) =>
              sb.build && (
                <BuildCard
                  key={sb.build.uuid || sb.build.id || idx}
                  build={sb.build}
                  similarityScore={sb.similarityScore}
                  matchDetails={sb.matchDetails}
                />
              ),
          )}
        </div>
      )}
    </div>
  );
}
