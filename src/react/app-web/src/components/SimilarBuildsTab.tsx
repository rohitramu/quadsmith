import { useQuery } from "@connectrpc/connect-query";
import { listSimilarBuilds } from "@quadsmith/shared/gen/quadsmith/api/build-BuildService_connectquery";
import { BuildCard } from "./BuildCard";
import { Link } from "react-router-dom";
import { Layers, Plus, Compass } from "lucide-react";

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
      {/* Loading Skeleton */}
      {isLoading && (
        <div className="space-y-4">
          {[1, 2, 3].map((n) => (
            <div
              key={n}
              className="rounded-2xl border border-zinc-800 p-5 bg-zinc-900 animate-pulse flex flex-col sm:flex-row gap-5"
            >
              <div className="sm:w-56 h-40 rounded-xl bg-zinc-800 shrink-0" />
              <div className="flex-1 space-y-3">
                <div className="h-5 bg-zinc-800 rounded w-1/3" />
                <div className="h-4 bg-zinc-800 rounded w-2/3" />
                <div className="h-8 bg-zinc-800/60 rounded w-1/2" />
                <div className="h-4 bg-zinc-800 rounded w-1/4 mt-4" />
              </div>
            </div>
          ))}
        </div>
      )}

      {/* Error State */}
      {error && !isLoading && (
        <div
          role="alert"
          className="p-5 rounded-2xl border border-red-900/60 bg-red-950/40 text-red-300 text-xs"
        >
          Failed to load similar builds: {error.message}
        </div>
      )}

      {/* Empty State */}
      {!isLoading && !error && similarBuilds.length === 0 && (
        <div className="p-12 text-center rounded-2xl border border-dashed border-zinc-800 bg-zinc-900/40 space-y-4">
          <div className="w-12 h-12 rounded-2xl bg-zinc-800 flex items-center justify-center mx-auto text-zinc-400">
            <Layers size={24} />
          </div>
          <div className="space-y-1">
            <h4 className="font-bold text-sm text-zinc-100">No Similar Builds Found</h4>
            <p className="text-xs text-zinc-400 max-w-sm mx-auto">
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
              className="px-4 py-2 rounded-xl border border-zinc-800 hover:bg-zinc-800 text-zinc-300 font-semibold text-xs transition-colors flex items-center gap-1.5"
            >
              <Compass size={14} />
              <span>Explore Build Feed</span>
            </Link>
          </div>
        </div>
      )}

      {/* Results List (Vertical stack of horizontal cards) */}
      {!isLoading && !error && similarBuilds.length > 0 && (
        <div className="space-y-4">
          {similarBuilds.map(
            (sb, idx) =>
              sb.build && (
                <BuildCard
                  key={sb.build.uuid || sb.build.id || idx}
                  build={sb.build}
                  variant="horizontal"
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
