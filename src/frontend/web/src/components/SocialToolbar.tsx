import { useState } from "react";
import { useQuery, useMutation } from "@connectrpc/connect-query";
import { useQueryClient } from "@tanstack/react-query";
import {
  getEntitySocialSummary,
  toggleLikeItem,
} from "../gen/quadsmith/social-SocialService_connectquery";
import { useAuth } from "../context/AuthContext";
import { Heart, Bookmark, Share2, ThumbsUp, HelpCircle, Check, Loader2 } from "lucide-react";
import { getReviewTierBadgeClass, getReviewTierLabel } from "../lib/socialFormat";
import { SaveToCollectionModal } from "./SaveToCollectionModal";

export interface SocialToolbarProps {
  target: { type?: "build" | "component"; id: string; collection?: string };
  targetName?: string;
  onJumpToReviews?: () => void;
  onJumpToQa?: () => void;
}

export function SocialToolbar({
  target,
  targetName,
  onJumpToReviews,
  onJumpToQa,
}: SocialToolbarProps) {
  const { isAuthenticated } = useAuth();
  const queryClient = useQueryClient();

  const [isCollectionModalOpen, setIsCollectionModalOpen] = useState(false);
  const [copiedShare, setCopiedShare] = useState(false);

  const queryPayload = {
    entityId: target.id,
    collection: target.collection || (target.type === "build" ? "builds" : undefined),
  };

  const { data: summary, refetch: refetchSummary } = useQuery(
    getEntitySocialSummary,
    queryPayload,
    { staleTime: 60_000 },
  );

  const { mutateAsync: mutateToggleLike, isPending: isLiking } = useMutation(toggleLikeItem);

  async function handleToggleLike() {
    if (!isAuthenticated) {
      return;
    }
    try {
      await mutateToggleLike({
        entityId: target.id,
        collection: target.collection || (target.type === "build" ? "builds" : undefined),
      });
      await refetchSummary();
      await queryClient.invalidateQueries();
    } catch {
      // Ignore or notify
    }
  }

  function handleShare() {
    if (navigator.clipboard) {
      navigator.clipboard.writeText(window.location.href);
      setCopiedShare(true);
      setTimeout(() => setCopiedShare(false), 2000);
    }
  }

  const isLiked = summary?.isLiked || false;
  const likeCount = summary?.likeCount ?? 0;
  const reviewSummary = summary?.reviewSummary;
  const reviewCount = reviewSummary?.totalReviews ?? 0;
  const questionCount = summary?.questionCount ?? 0;

  const tierColors = reviewSummary ? getReviewTierBadgeClass(reviewSummary.tier) : null;
  const tierLabel = reviewSummary ? getReviewTierLabel(reviewSummary.tier) : "";

  return (
    <>
      <div className="pt-4 border-t border-zinc-200 dark:border-zinc-800/80 flex flex-wrap items-center justify-between gap-4">
        {/* Action Buttons: Like, Save to Collection, Share */}
        <div className="flex items-center gap-3">
          {/* 1-Click Like Button */}
          <button
            type="button"
            onClick={handleToggleLike}
            disabled={isLiking}
            className={`flex items-center gap-2 px-4 py-2 rounded-xl border transition-all text-xs font-semibold cursor-pointer active:scale-95 group ${
              isLiked
                ? "border-rose-500/40 bg-rose-500/10 text-rose-500 hover:bg-rose-500/20"
                : "border-zinc-300 dark:border-zinc-700 bg-zinc-100/80 dark:bg-zinc-800/70 hover:bg-zinc-200 dark:hover:bg-zinc-700 text-zinc-700 dark:text-zinc-200"
            }`}
            title={isAuthenticated ? (isLiked ? "Unlike" : "Like") : "Log in to like"}
          >
            {isLiking ? (
              <Loader2 size={16} className="animate-spin text-zinc-400" />
            ) : (
              <Heart
                size={16}
                className={`transition-transform group-hover:scale-110 ${
                  isLiked ? "fill-rose-500 text-rose-500" : "text-zinc-500 dark:text-zinc-400"
                }`}
              />
            )}
            <span>{isLiked ? "Liked" : "Like"}</span>
            <span className="px-1.5 py-0.5 rounded-md bg-white dark:bg-zinc-900 text-zinc-700 dark:text-zinc-300 font-mono text-[11px] border border-zinc-200 dark:border-zinc-800">
              {likeCount}
            </span>
          </button>

          {/* Save to Collection Modal Trigger */}
          <button
            type="button"
            onClick={() => setIsCollectionModalOpen(true)}
            className="flex items-center gap-2 px-4 py-2 rounded-xl border border-zinc-300 dark:border-zinc-700 bg-zinc-100/80 dark:bg-zinc-800/70 hover:bg-zinc-200 dark:hover:bg-zinc-700 text-zinc-700 dark:text-zinc-200 transition-all text-xs font-semibold cursor-pointer active:scale-95"
          >
            <Bookmark size={16} className="text-zinc-500 dark:text-zinc-400" />
            <span>Save to Collection</span>
          </button>

          {/* Share Button */}
          <button
            type="button"
            onClick={handleShare}
            className="p-2 rounded-xl border border-zinc-300 dark:border-zinc-700 bg-zinc-100/80 dark:bg-zinc-800/70 hover:bg-zinc-200 dark:hover:bg-zinc-700 text-zinc-500 dark:text-zinc-400 hover:text-zinc-900 dark:hover:text-zinc-200 transition-colors"
            title="Share Link"
          >
            {copiedShare ? <Check size={16} className="text-emerald-500" /> : <Share2 size={16} />}
          </button>
        </div>

        {/* Sentiment Badge & Jump Links */}
        <div className="flex items-center gap-3 flex-wrap">
          {reviewSummary && reviewCount > 0 && tierColors && (
            <div
              className={`flex items-center gap-1.5 px-3 py-1 rounded-xl border text-xs font-bold ${tierColors.bg} ${tierColors.border} ${tierColors.text}`}
            >
              <ThumbsUp size={13} className="fill-current" />
              <span>{tierLabel}</span>
              <span className="font-mono ml-1 text-[11px] opacity-80">
                {Math.round(reviewSummary.scorePercentage)}%
              </span>
            </div>
          )}

          {/* Jump to Reviews */}
          <button
            type="button"
            onClick={onJumpToReviews}
            className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-xs font-medium text-zinc-600 dark:text-zinc-400 hover:text-zinc-900 dark:hover:text-white hover:bg-zinc-100 dark:hover:bg-zinc-800 transition-colors"
          >
            <ThumbsUp size={13} />
            <span>Reviews ({reviewCount})</span>
          </button>

          {/* Jump to Q&A */}
          <button
            type="button"
            onClick={onJumpToQa}
            className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-xs font-medium text-zinc-600 dark:text-zinc-400 hover:text-zinc-900 dark:hover:text-white hover:bg-zinc-100 dark:hover:bg-zinc-800 transition-colors"
          >
            <HelpCircle size={13} />
            <span>Q&A ({questionCount})</span>
          </button>
        </div>
      </div>

      <SaveToCollectionModal
        isOpen={isCollectionModalOpen}
        onClose={() => setIsCollectionModalOpen(false)}
        target={target}
        targetName={targetName}
      />
    </>
  );
}
