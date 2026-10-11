import { useState, useEffect } from "react";
import { useQuery } from "@connectrpc/connect-query";
import {
  getEntitySocialSummary,
  listUserCollections,
} from "../gen/quadsmith/social-SocialService_connectquery";
import { useAuth } from "../context/AuthContext";
import { ThumbsUp, HelpCircle, Bookmark, Star, ArrowUpRight, Plus, Archive } from "lucide-react";
import { SocialReviewSection } from "./SocialReviewSection";
import { SocialQaSection } from "./SocialQaSection";
import { SaveToCollectionModal } from "./SaveToCollectionModal";
import { Link } from "react-router-dom";

export interface SocialTabsProps {
  target: { type?: "build" | "component"; id: string; collection?: string };
  targetAuthorUuid?: string;
  entityName?: string;
  defaultTab?: "reviews" | "qa" | "collections";
  activeTab?: "reviews" | "qa" | "collections";
  onTabChange?: (tab: "reviews" | "qa" | "collections") => void;
}

export function SocialTabs({
  target,
  targetAuthorUuid,
  entityName,
  defaultTab = "reviews",
  activeTab: controlledActiveTab,
  onTabChange,
}: SocialTabsProps) {
  const [internalActiveTab, setInternalActiveTab] = useState<"reviews" | "qa" | "collections">(
    controlledActiveTab || defaultTab,
  );
  const activeTab = controlledActiveTab !== undefined ? controlledActiveTab : internalActiveTab;
  const [visitedTabs, setVisitedTabs] = useState<Set<string>>(
    () => new Set([controlledActiveTab || defaultTab]),
  );
  const [isNewColModalOpen, setIsNewColModalOpen] = useState(false);
  const { user } = useAuth();

  // Sync internal active tab if controlled prop changes
  useEffect(() => {
    if (controlledActiveTab !== undefined) {
      setInternalActiveTab(controlledActiveTab);
    }
  }, [controlledActiveTab]);

  // Keep track of visited tabs so unvisited tabs are lazy loaded
  useEffect(() => {
    setVisitedTabs((prev) => {
      if (prev.has(activeTab)) return prev;
      const next = new Set(prev);
      next.add(activeTab);
      return next;
    });
  }, [activeTab]);

  const handleTabSelect = (tab: "reviews" | "qa" | "collections") => {
    if (controlledActiveTab === undefined) {
      setInternalActiveTab(tab);
    }
    onTabChange?.(tab);
  };

  const queryPayload = {
    entityId: target.id,
    collection: target.collection || (target.type === "build" ? "builds" : undefined),
  };

  const { data: summary } = useQuery(getEntitySocialSummary, queryPayload, {
    staleTime: 60_000,
  });
  const { data: userColsRes } = useQuery(
    listUserCollections,
    { userId: user?.id || "" },
    { enabled: !!user?.id, staleTime: 60_000 },
  );

  const reviewCount = summary?.reviewSummary?.totalReviews ?? 0;
  const questionCount = summary?.questionCount ?? 0;
  const collections = userColsRes?.collections || [];
  const isArchived = summary?.isArchived || false;

  return (
    <section id="social-hub" className="mt-12 space-y-6">
      {/* Archived Read-Only Banner */}
      {isArchived && (
        <div className="p-4 rounded-2xl border border-amber-500/30 bg-amber-500/10 text-amber-700 dark:text-amber-300 text-xs flex items-center gap-3">
          <Archive size={18} className="shrink-0 text-amber-500" />
          <div>
            <span className="font-bold">Archived Entity (Read-Only): </span>
            This entity has been archived for historical reference. New reviews, questions, answers,
            and votes are disabled.
          </div>
        </div>
      )}

      {/* Tab Navigation Bar */}
      <div className="flex items-center gap-2 border-b border-zinc-200 dark:border-zinc-800 pb-px flex-wrap">
        <button
          type="button"
          onClick={() => handleTabSelect("reviews")}
          className={`flex items-center gap-2 px-5 py-3 border-b-2 font-bold text-sm transition-colors cursor-pointer ${
            activeTab === "reviews"
              ? "border-blue-500 text-blue-600 dark:text-blue-400"
              : "border-transparent text-zinc-500 dark:text-zinc-400 hover:text-zinc-800 dark:hover:text-zinc-200 hover:border-zinc-300 dark:hover:border-zinc-700"
          }`}
        >
          <ThumbsUp size={16} />
          <span>User Reviews</span>
          <span className="px-2 py-0.5 rounded-full text-[11px] bg-blue-500/10 text-blue-600 dark:text-blue-400 font-mono">
            {reviewCount}
          </span>
        </button>

        <button
          type="button"
          onClick={() => handleTabSelect("qa")}
          className={`flex items-center gap-2 px-5 py-3 border-b-2 font-bold text-sm transition-colors cursor-pointer ${
            activeTab === "qa"
              ? "border-blue-500 text-blue-600 dark:text-blue-400"
              : "border-transparent text-zinc-500 dark:text-zinc-400 hover:text-zinc-800 dark:hover:text-zinc-200 hover:border-zinc-300 dark:hover:border-zinc-700"
          }`}
        >
          <HelpCircle size={16} />
          <span>Technical Q&A</span>
          <span className="px-2 py-0.5 rounded-full text-[11px] bg-zinc-100 dark:bg-zinc-800 text-zinc-600 dark:text-zinc-400 font-mono">
            {questionCount}
          </span>
        </button>

        <button
          type="button"
          onClick={() => handleTabSelect("collections")}
          className={`flex items-center gap-2 px-5 py-3 border-b-2 font-bold text-sm transition-colors cursor-pointer ${
            activeTab === "collections"
              ? "border-blue-500 text-blue-600 dark:text-blue-400"
              : "border-transparent text-zinc-500 dark:text-zinc-400 hover:text-zinc-800 dark:hover:text-zinc-200 hover:border-zinc-300 dark:hover:border-zinc-700"
          }`}
        >
          <Bookmark size={16} />
          <span>Collections</span>
          <span className="px-2 py-0.5 rounded-full text-[11px] bg-zinc-100 dark:bg-zinc-800 text-zinc-600 dark:text-zinc-400 font-mono">
            {collections.length}
          </span>
        </button>
      </div>

      {/* Tab 1: Reviews */}
      {visitedTabs.has("reviews") && (
        <div
          id="reviews"
          className={`animate-in fade-in duration-150 ${activeTab === "reviews" ? "" : "hidden"}`}
        >
          <SocialReviewSection
            target={{
              ...target,
              collection: target.collection || (target.type === "build" ? "builds" : undefined),
            }}
            targetAuthorUuid={targetAuthorUuid}
            entityName={entityName}
            isArchived={isArchived}
          />
        </div>
      )}

      {/* Tab 2: Q&A */}
      {visitedTabs.has("qa") && (
        <div
          id="qa"
          className={`animate-in fade-in duration-150 ${activeTab === "qa" ? "" : "hidden"}`}
        >
          <SocialQaSection
            target={{
              ...target,
              collection: target.collection || (target.type === "build" ? "builds" : undefined),
            }}
            targetAuthorUuid={targetAuthorUuid}
            entityName={entityName}
            isArchived={isArchived}
          />
        </div>
      )}

      {/* Tab 3: Collections */}
      {visitedTabs.has("collections") && (
        <div
          id="collections"
          className={`space-y-6 animate-in fade-in duration-150 ${activeTab === "collections" ? "" : "hidden"}`}
        >
          <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4 p-6 rounded-2xl border border-zinc-200 dark:border-zinc-800 bg-zinc-50/70 dark:bg-zinc-900/40">
            <div className="space-y-1">
              <h3 className="font-bold text-base text-zinc-900 dark:text-zinc-100 flex items-center gap-2">
                <Bookmark size={18} className="text-blue-500" />
                <span>Collections Featuring This Item</span>
              </h3>
              <p className="text-xs text-zinc-500 dark:text-zinc-400">
                Users organize builds and components into custom public and private collections for
                project inspiration, bando parts lists, and gear bags.
              </p>
            </div>
            <button
              type="button"
              onClick={() => setIsNewColModalOpen(true)}
              className="px-5 py-2.5 rounded-xl bg-blue-600 hover:bg-blue-500 text-white font-semibold text-xs transition-colors shrink-0 shadow-lg shadow-blue-500/20 flex items-center gap-1.5 justify-center cursor-pointer"
            >
              <Plus size={14} />
              <span>Save / Create Collection</span>
            </button>
          </div>

          <div className="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 gap-4">
            {collections.length === 0 ? (
              <div className="col-span-full p-8 text-center rounded-2xl border border-dashed border-zinc-200 dark:border-zinc-800 text-zinc-400 text-xs">
                No collections created yet. Use the Save to Collection button to start a new
                collection!
              </div>
            ) : (
              collections.map((col) => (
                <div
                  key={col.id}
                  className="rounded-2xl border border-zinc-200 dark:border-zinc-800 bg-white dark:bg-zinc-900/60 p-5 space-y-3 hover:border-zinc-300 dark:hover:border-zinc-700 transition-colors shadow-2xs"
                >
                  <div className="flex items-center justify-between">
                    <div className="flex items-center gap-2">
                      <div className="w-8 h-8 rounded-lg bg-blue-500/10 text-blue-500 flex items-center justify-center font-bold text-xs">
                        {col.isDefaultLiked ? "❤️" : col.name.slice(0, 2).toUpperCase()}
                      </div>
                      <div>
                        <h4 className="font-bold text-sm text-zinc-900 dark:text-zinc-100">
                          {col.name}
                        </h4>
                        <p className="text-[10px] text-zinc-500">
                          {col.isDefaultLiked ? "Default Likes" : `@${col.authorUsername}`}
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

                  <div className="flex items-center justify-between text-xs text-zinc-500 pt-2 border-t border-zinc-100 dark:border-zinc-800">
                    <span className="font-mono">{col.itemCount} items</span>
                    <Link
                      to={`/users/${col.authorUsername}/collections/${col.id}`}
                      className="text-blue-500 hover:underline flex items-center gap-1 font-medium"
                    >
                      <span>View</span>
                      <ArrowUpRight size={12} />
                    </Link>
                  </div>
                </div>
              ))
            )}
          </div>
        </div>
      )}

      {/* Reputation Economy Explainer Card (from preview design) */}
      <div className="rounded-2xl border border-zinc-200 dark:border-zinc-800 bg-zinc-50/50 dark:bg-zinc-900/30 p-6 space-y-4">
        <div className="flex items-center justify-between flex-wrap gap-2">
          <h4 className="text-xs font-bold uppercase tracking-wider text-amber-500 dark:text-amber-400 flex items-center gap-1.5">
            <Star size={14} className="fill-current" />
            <span>Community Reputation Economy (Karma Rules)</span>
          </h4>
          <span className="text-[11px] font-mono text-zinc-400">
            Derives from relational vote & acceptance tables
          </span>
        </div>

        <div className="grid grid-cols-2 sm:grid-cols-5 gap-3 text-xs">
          <div className="p-3 rounded-xl bg-white dark:bg-zinc-950/60 border border-zinc-200 dark:border-zinc-800">
            <div className="text-zinc-500 dark:text-zinc-400">Review Author</div>
            <div className="text-emerald-600 dark:text-emerald-400 font-bold font-mono text-base mt-0.5">
              +3 Rep
            </div>
            <div className="text-[10px] text-zinc-400 mt-1">Written evaluation reward</div>
          </div>

          <div className="p-3 rounded-xl bg-white dark:bg-zinc-950/60 border border-zinc-200 dark:border-zinc-800">
            <div className="text-zinc-500 dark:text-zinc-400">Accepted Answer</div>
            <div className="text-emerald-600 dark:text-emerald-400 font-bold font-mono text-base mt-0.5">
              +30 Rep
            </div>
            <div className="text-[10px] text-zinc-400 mt-1">+10 bonus to asker</div>
          </div>

          <div className="p-3 rounded-xl bg-white dark:bg-zinc-950/60 border border-zinc-200 dark:border-zinc-800">
            <div className="text-zinc-500 dark:text-zinc-400">Author Endorsed</div>
            <div className="text-amber-600 dark:text-amber-400 font-bold font-mono text-base mt-0.5">
              +20 Rep
            </div>
            <div className="text-[10px] text-zinc-400 mt-1">Conferred by build author</div>
          </div>

          <div className="p-3 rounded-xl bg-white dark:bg-zinc-950/60 border border-zinc-200 dark:border-zinc-800">
            <div className="text-zinc-500 dark:text-zinc-400">Post / Review Upvote</div>
            <div className="text-emerald-600 dark:text-emerald-400 font-bold font-mono text-base mt-0.5">
              +1 Rep
            </div>
            <div className="text-[10px] text-zinc-400 mt-1">Daily Cap: max 20 upvotes / 24h</div>
          </div>

          <div className="p-3 rounded-xl bg-white dark:bg-zinc-950/60 border border-zinc-200 dark:border-zinc-800">
            <div className="text-zinc-500 dark:text-zinc-400">Downvote Stakes</div>
            <div className="text-rose-600 dark:text-rose-400 font-bold font-mono text-base mt-0.5">
              -2 Rep (Both)
            </div>
            <div className="text-[10px] text-zinc-400 mt-1">
              Costs voter -2 • Unlock at 100 rep • Max 3/24h (max 3/wk per target author)
            </div>
          </div>
        </div>
      </div>

      <SaveToCollectionModal
        isOpen={isNewColModalOpen}
        onClose={() => setIsNewColModalOpen(false)}
        target={target}
        targetName={entityName}
      />
    </section>
  );
}
