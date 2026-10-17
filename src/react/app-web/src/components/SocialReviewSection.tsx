import { useState } from "react";
import { useQuery, useMutation } from "@connectrpc/connect-query";
import { UserBadge } from "./UserBadge";
import { useQueryClient } from "@tanstack/react-query";
import {
  listReviews,
  listReviewComments,
  createReview,
  deleteReview,
  voteReview,
  createReviewComment,
  deleteReviewComment,
} from "@quadsmith/shared/gen/quadsmith/api/social-SocialService_connectquery";
import { ReviewVoteType, type Review } from "@quadsmith/shared/gen/quadsmith/api/social_pb";
import { useAuth } from "../context/AuthContext";
import {
  ThumbsUp,
  ThumbsDown,
  MessageSquare,
  Trash2,
  Send,
  Loader2,
  ChevronDown,
  ChevronUp,
  AlertCircle,
} from "lucide-react";
import { getReviewTierBadgeClass, getReviewTierLabel, formatRelativeDate } from "@quadsmith/shared";
import { getAvatarPalette, getInitial } from "@quadsmith/shared";
import { StyledSelect } from "./StyledSelect";

const reviewSortOptions = [
  { value: "helpful", label: "Most Helpful" },
  { value: "recent", label: "Most Recent" },
];

export interface SocialReviewSectionProps {
  target: { type?: "build" | "component"; id: string; collection?: string };
  targetAuthorUuid?: string;
  entityName?: string;
  isArchived?: boolean;
}

export function SocialReviewSection({
  target,
  targetAuthorUuid,
  entityName,
  isArchived = false,
}: SocialReviewSectionProps) {
  const { user, isAuthenticated } = useAuth();
  const queryClient = useQueryClient();

  const [sortBy, setSortBy] = useState<string>("helpful");
  const [isComposerOpen, setIsComposerOpen] = useState(false);
  const [isRecommended, setIsRecommended] = useState<boolean>(true);
  const [reviewBody, setReviewBody] = useState("");
  const [composerError, setComposerError] = useState<string | null>(null);
  const [isSubmitting, setIsSubmitting] = useState(false);

  // Discussion reply inputs keyed by review UUID/ID
  const [openThreads, setOpenThreads] = useState<Record<string, boolean>>({});

  const queryPayload = {
    entityId: target.id,
    collection: target.collection || (target.type === "build" ? "builds" : undefined),
    sortBy,
  };

  const {
    data: reviewsResponse,
    isLoading: isLoadingReviews,
    refetch: refetchReviews,
  } = useQuery(listReviews, queryPayload, { staleTime: 60_000 });

  const { mutateAsync: mutateCreateReview } = useMutation(createReview);
  const { mutateAsync: mutateDeleteReview } = useMutation(deleteReview);
  const { mutateAsync: mutateVoteReview } = useMutation(voteReview);

  const reviews = reviewsResponse?.reviews || [];
  const summary = reviewsResponse?.summary;
  const isAuthorOfBuild =
    target.type === "build" && !!user?.uuid && !!targetAuthorUuid && user.uuid === targetAuthorUuid;

  const existingUserReview = reviews.find((r) => r.userUuid === user?.uuid);

  async function handleSubmitReview(e: React.FormEvent) {
    e.preventDefault();
    if (!reviewBody.trim()) {
      setComposerError("Please write a few words about your experience.");
      return;
    }
    setComposerError(null);
    setIsSubmitting(true);
    try {
      await mutateCreateReview({
        entityId: target.id,
        collection: target.collection || (target.type === "build" ? "builds" : undefined),
        isRecommended,
        body: reviewBody.trim(),
      });
      setReviewBody("");
      setIsComposerOpen(false);
      await refetchReviews();
      await queryClient.invalidateQueries();
    } catch (err: unknown) {
      setComposerError(err instanceof Error ? err.message : "Failed to post review");
    } finally {
      setIsSubmitting(false);
    }
  }

  async function handleDeleteReview(reviewId: string) {
    if (!window.confirm("Are you sure you want to delete your review?")) return;
    try {
      await mutateDeleteReview({ reviewId });
      await refetchReviews();
      await queryClient.invalidateQueries();
    } catch (err: unknown) {
      alert(err instanceof Error ? err.message : "Failed to delete review");
    }
  }

  async function handleVote(
    reviewId: string,
    currentVote: ReviewVoteType,
    voteForHelpful: boolean,
  ) {
    if (!isAuthenticated) return;
    const newVoteType = voteForHelpful
      ? currentVote === ReviewVoteType.HELPFUL
        ? ReviewVoteType.UNSPECIFIED
        : ReviewVoteType.HELPFUL
      : currentVote === ReviewVoteType.UNHELPFUL
        ? ReviewVoteType.UNSPECIFIED
        : ReviewVoteType.UNHELPFUL;

    try {
      await mutateVoteReview({
        reviewId,
        voteType: newVoteType,
      });
      await refetchReviews();
    } catch (err: unknown) {
      if (err instanceof Error && err.message.includes("cannot vote on your own review")) {
        return;
      }
      alert(err instanceof Error ? err.message : "Failed to register vote");
    }
  }

  function toggleDiscussionThread(reviewId: string) {
    setOpenThreads((prev) => ({ ...prev, [reviewId]: !prev[reviewId] }));
  }

  const positivePercent = summary ? Math.round(summary.scorePercentage) : 0;
  const totalReviews = summary?.totalReviews || 0;
  const recommendedCount = summary?.recommendedCount || 0;
  const notRecommendedCount = totalReviews - recommendedCount;
  const tierColors = summary ? getReviewTierBadgeClass(summary.tier) : null;
  const tierLabel = summary ? getReviewTierLabel(summary.tier) : "No Reviews";

  return (
    <div className="space-y-6">
      {/* Community Sentiment Aggregate Breakdown & Write Review CTA */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-6 p-6 rounded-2xl border border-zinc-200 dark:border-zinc-800 bg-zinc-50/70 dark:bg-zinc-900/40">
        <div className="md:col-span-2 space-y-3">
          <div className="flex items-center gap-2">
            <h3 className="text-xs font-bold uppercase tracking-wider text-zinc-500 dark:text-zinc-400">
              Community Sentiment
            </h3>
            {totalReviews > 0 && tierColors ? (
              <span
                className={`px-2 py-0.5 rounded text-[10px] font-bold border ${tierColors.bg} ${tierColors.border} ${tierColors.text}`}
              >
                {tierLabel}
              </span>
            ) : (
              <span className="px-2 py-0.5 rounded text-[10px] font-semibold border border-zinc-200 dark:border-zinc-700 bg-zinc-100 dark:bg-zinc-800 text-zinc-500">
                No Reviews
              </span>
            )}
          </div>

          <div className="flex items-baseline gap-3">
            <span className="text-3xl font-extrabold text-zinc-900 dark:text-zinc-50">
              {totalReviews > 0 ? `${positivePercent}% Recommended` : "No reviews yet"}
            </span>
            {totalReviews > 0 && (
              <span className="text-xs text-zinc-500 dark:text-zinc-400 font-mono">
                ({recommendedCount} positive / {notRecommendedCount} negative)
              </span>
            )}
          </div>

          <p className="text-xs text-zinc-600 dark:text-zinc-400 leading-relaxed">
            Reviews on Quadsmith use Steam-style binary recommendations to provide clear, actionable
            purchasing and flight consensus.
          </p>

          {/* Qualitative Tier Progress */}
          <div className="space-y-1.5 pt-2">
            <div className="flex justify-between text-[11px] text-zinc-500 dark:text-zinc-400 font-mono">
              <span>Consensus</span>
              <span>
                {totalReviews > 0 ? `${positivePercent}% (${tierLabel})` : "No consensus yet"}
              </span>
            </div>
            <div className="h-2 w-full bg-zinc-200 dark:bg-zinc-800 rounded-full overflow-hidden flex">
              {totalReviews > 0 ? (
                <>
                  <div
                    className="bg-emerald-500 h-full transition-all duration-300"
                    style={{ width: `${positivePercent}%` }}
                  />
                  <div
                    className="bg-rose-500 h-full transition-all duration-300"
                    style={{ width: `${100 - positivePercent}%` }}
                  />
                </>
              ) : (
                <div className="bg-zinc-300 dark:bg-zinc-700 h-full w-full opacity-30" />
              )}
            </div>
          </div>
        </div>

        {/* Write Review Button Box */}
        <div className="flex flex-col justify-center items-start md:items-end border-t md:border-t-0 md:border-l border-zinc-200 dark:border-zinc-800 pt-4 md:pt-0 md:pl-6">
          {isArchived ? (
            <div className="p-3 rounded-xl bg-amber-500/10 border border-amber-500/20 text-amber-600 dark:text-amber-400 text-xs text-center md:text-right">
              <AlertCircle size={16} className="inline mr-1 mb-0.5" />
              Archived entity: Reviews are read-only.
            </div>
          ) : isAuthorOfBuild ? (
            <div className="p-3 rounded-xl bg-amber-500/10 border border-amber-500/20 text-amber-600 dark:text-amber-400 text-xs text-center md:text-right">
              <AlertCircle size={16} className="inline mr-1 mb-0.5" />
              Build creators cannot review their own builds.
            </div>
          ) : !isAuthenticated ? (
            <div className="text-xs text-zinc-500 text-center md:text-right">
              <p>Log in to write a review and help the community.</p>
            </div>
          ) : existingUserReview ? (
            <div className="text-xs text-zinc-500 text-center md:text-right space-y-1">
              <p className="font-semibold text-emerald-600 dark:text-emerald-400">
                You reviewed this {target.type}.
              </p>
              <button
                type="button"
                onClick={() => handleDeleteReview(existingUserReview.id)}
                className="text-xs text-rose-500 hover:underline flex items-center gap-1 mx-auto md:ml-auto"
              >
                <Trash2 size={12} />
                <span>Delete your review</span>
              </button>
            </div>
          ) : (
            <>
              <button
                type="button"
                onClick={() => setIsComposerOpen((prev) => !prev)}
                className="w-full md:w-auto px-5 py-2.5 rounded-xl bg-blue-600 hover:bg-blue-500 text-white font-semibold text-xs transition-colors shadow-lg shadow-blue-500/20 flex items-center justify-center gap-2 cursor-pointer"
              >
                <ThumbsUp size={14} />
                <span>{isComposerOpen ? "Close Composer" : "Write a Review"}</span>
              </button>
              <span className="text-[11px] text-zinc-400 dark:text-zinc-500 mt-2">
                1 review per user • Editable
              </span>
            </>
          )}
        </div>
      </div>

      {/* Inline Review Composer */}
      {isComposerOpen && !isAuthorOfBuild && (
        <form
          onSubmit={handleSubmitReview}
          className="rounded-2xl border-2 border-blue-500/30 bg-white dark:bg-zinc-900/90 p-6 space-y-4 shadow-xl"
        >
          <div className="flex items-center justify-between">
            <h3 className="font-bold text-sm text-zinc-900 dark:text-zinc-100">
              Your Review {entityName ? `of ${entityName}` : ""}
            </h3>
            <button
              type="button"
              onClick={() => setIsComposerOpen(false)}
              className="text-zinc-400 hover:text-zinc-600 dark:hover:text-zinc-200 text-xs"
            >
              Cancel
            </button>
          </div>

          {composerError && (
            <div className="p-3 rounded-xl bg-rose-500/10 border border-rose-500/20 text-rose-500 text-xs">
              {composerError}
            </div>
          )}

          {/* Recommendation Toggle Buttons */}
          <div className="space-y-2">
            <label className="block text-xs font-semibold uppercase tracking-wider text-zinc-500 dark:text-zinc-400">
              Do you recommend this {target.type}?
            </label>
            <div className="grid grid-cols-2 gap-3 max-w-md">
              <button
                type="button"
                onClick={() => setIsRecommended(true)}
                className={`flex items-center justify-center gap-2 py-3 px-4 rounded-xl border text-xs font-bold transition-all cursor-pointer ${
                  isRecommended
                    ? "border-emerald-500 bg-emerald-500/10 text-emerald-600 dark:text-emerald-400 shadow-sm"
                    : "border-zinc-200 dark:border-zinc-800 bg-zinc-100 dark:bg-zinc-800/50 text-zinc-500 dark:text-zinc-400 hover:bg-zinc-200 dark:hover:bg-zinc-800"
                }`}
              >
                <ThumbsUp size={16} className={isRecommended ? "fill-current" : ""} />
                <span>Recommended</span>
              </button>

              <button
                type="button"
                onClick={() => setIsRecommended(false)}
                className={`flex items-center justify-center gap-2 py-3 px-4 rounded-xl border text-xs font-bold transition-all cursor-pointer ${
                  !isRecommended
                    ? "border-rose-500 bg-rose-500/10 text-rose-600 dark:text-rose-400 shadow-sm"
                    : "border-zinc-200 dark:border-zinc-800 bg-zinc-100 dark:bg-zinc-800/50 text-zinc-500 dark:text-zinc-400 hover:bg-zinc-200 dark:hover:bg-zinc-800"
                }`}
              >
                <ThumbsDown size={16} className={!isRecommended ? "fill-current" : ""} />
                <span>Not Recommended</span>
              </button>
            </div>
          </div>

          {/* Review Text */}
          <div className="space-y-1.5">
            <label className="block text-xs font-semibold text-zinc-600 dark:text-zinc-400">
              Review Writeup
            </label>
            <textarea
              rows={4}
              value={reviewBody}
              onChange={(e) => setReviewBody(e.target.value)}
              placeholder="Describe flight characteristics, build assembly notes, motor thermals, durability, or component fitment..."
              className="w-full rounded-xl border border-zinc-200 dark:border-zinc-700 bg-zinc-50 dark:bg-zinc-950 p-3 text-sm text-zinc-900 dark:text-zinc-100 placeholder-zinc-400 dark:placeholder-zinc-500 focus:outline-hidden focus:border-blue-500"
            />
          </div>

          <div className="flex justify-end gap-3 pt-1">
            <button
              type="submit"
              disabled={isSubmitting}
              className="px-5 py-2 rounded-xl bg-blue-600 hover:bg-blue-500 text-white font-semibold text-xs shadow-md transition-colors flex items-center gap-1.5 disabled:opacity-50"
            >
              {isSubmitting && <Loader2 size={13} className="animate-spin" />}
              <span>Post Review</span>
            </button>
          </div>
        </form>
      )}

      {/* Reviews List & Sort Header */}
      <div className="space-y-4">
        <div className="flex items-center justify-between">
          <span className="text-xs font-bold uppercase tracking-wider text-zinc-500 dark:text-zinc-400">
            User Reviews ({reviews.length})
          </span>
          <div className="flex items-center gap-2 text-xs">
            <span className="text-zinc-400">Sort by:</span>
            <StyledSelect
              value={sortBy}
              onChange={(val) => setSortBy(val)}
              options={reviewSortOptions}
              ariaLabel="Sort reviews"
              size="sm"
            />
          </div>
        </div>

        {isLoadingReviews ? (
          <div className="py-12 flex justify-center text-zinc-400">
            <Loader2 className="animate-spin text-blue-500" size={24} />
          </div>
        ) : reviews.length === 0 ? (
          <div className="p-8 text-center rounded-2xl border border-dashed border-zinc-200 dark:border-zinc-800 text-zinc-400 text-xs">
            No reviews yet. Be the first to share your evaluation!
          </div>
        ) : (
          reviews.map((review) => {
            const palette = getAvatarPalette(review.authorUsername || review.userUuid);
            const initial = getInitial(review.authorUsername || review.userUuid);
            const isMyReview = Boolean(user?.uuid && user.uuid === review.userUuid);
            const isThreadOpen = openThreads[review.id] ?? false;
            const comments = review.commentCount;

            return (
              <div
                key={review.id}
                className="rounded-2xl border border-zinc-200 dark:border-zinc-800 bg-white dark:bg-zinc-900/60 p-6 space-y-4 shadow-2xs"
              >
                {/* Header: User Info & Recommendation Badge */}
                <div className="flex items-start justify-between gap-4">
                  <div className="flex items-center gap-3">
                    {review.authorAvatarUrl ? (
                      <img
                        src={review.authorAvatarUrl}
                        alt={review.authorUsername}
                        className="w-10 h-10 rounded-full object-cover border border-zinc-200 dark:border-zinc-800"
                      />
                    ) : (
                      <div
                        className={`w-10 h-10 rounded-full border flex items-center justify-center font-bold text-sm ${palette.badgeClass}`}
                      >
                        {initial}
                      </div>
                    )}
                    <div>
                      <div className="flex items-center gap-2">
                        <UserBadge
                          username={review.authorUsername}
                          size="sm"
                          isAuthor={review.isAuthorVerifiedOwner}
                        />
                      </div>
                      <div className="text-[11px] text-zinc-400">
                        Reviewed on {formatRelativeDate(review.createdAt?.seconds)}
                        {review.isEdited && " • (edited)"}
                      </div>
                    </div>
                  </div>

                  {/* Recommendation Badge */}
                  <div
                    className={`flex items-center gap-1.5 px-3 py-1.5 rounded-xl border font-bold text-xs ${
                      review.isRecommended
                        ? "border-emerald-500/30 bg-emerald-500/10 text-emerald-600 dark:text-emerald-400"
                        : "border-rose-500/30 bg-rose-500/10 text-rose-600 dark:text-rose-400"
                    }`}
                  >
                    {review.isRecommended ? (
                      <>
                        <ThumbsUp size={14} className="fill-current" />
                        <span>Recommended</span>
                      </>
                    ) : (
                      <>
                        <ThumbsDown size={14} className="fill-current" />
                        <span>Not Recommended</span>
                      </>
                    )}
                  </div>
                </div>

                {/* Review Body */}
                <div className="text-sm text-zinc-700 dark:text-zinc-300 leading-relaxed whitespace-pre-line">
                  {review.body}
                </div>

                {/* Helpfulness Bar & Thread Toggle */}
                <div className="pt-3 border-t border-zinc-100 dark:border-zinc-800/80 flex items-center justify-between text-xs flex-wrap gap-2">
                  <div className="flex items-center gap-3">
                    <span className="text-zinc-500">Was this review helpful?</span>
                    {/* Yes button */}
                    <button
                      type="button"
                      disabled={isArchived || isMyReview}
                      onClick={() => handleVote(review.id, review.currentUserVote, true)}
                      title={
                        isArchived
                          ? "Reviews are archived"
                          : isMyReview
                            ? "You cannot vote on your own review"
                            : undefined
                      }
                      className={`flex items-center gap-1.5 px-3 py-1 rounded-lg border transition-colors ${
                        isArchived || isMyReview
                          ? "cursor-not-allowed opacity-50"
                          : "cursor-pointer hover:bg-zinc-100 dark:hover:bg-zinc-700"
                      } ${
                        review.currentUserVote === ReviewVoteType.HELPFUL
                          ? "border-emerald-500 bg-emerald-500/10 text-emerald-600 dark:text-emerald-400 font-semibold"
                          : "border-zinc-200 dark:border-zinc-700 bg-zinc-50 dark:bg-zinc-800 text-zinc-600 dark:text-zinc-300"
                      }`}
                    >
                      <ThumbsUp size={13} />
                      <span>
                        {review.currentUserVote === ReviewVoteType.HELPFUL
                          ? "You voted Yes"
                          : "Yes"}
                      </span>
                      <span className="font-mono text-zinc-400 ml-0.5">{review.helpfulCount}</span>
                    </button>

                    {/* No button */}
                    <button
                      type="button"
                      disabled={isArchived || isMyReview}
                      onClick={() => handleVote(review.id, review.currentUserVote, false)}
                      title={
                        isArchived
                          ? "Reviews are archived"
                          : isMyReview
                            ? "You cannot vote on your own review"
                            : undefined
                      }
                      className={`flex items-center gap-1.5 px-3 py-1 rounded-lg border transition-colors ${
                        isArchived || isMyReview
                          ? "cursor-not-allowed opacity-50"
                          : "cursor-pointer hover:bg-zinc-100 dark:hover:bg-zinc-700"
                      } ${
                        review.currentUserVote === ReviewVoteType.UNHELPFUL
                          ? "border-rose-500 bg-rose-500/10 text-rose-600 dark:text-rose-400 font-semibold"
                          : "border-zinc-200 dark:border-zinc-700 bg-zinc-50 dark:bg-zinc-800 text-zinc-500 dark:text-zinc-400"
                      }`}
                    >
                      <ThumbsDown size={13} />
                      <span>
                        {review.currentUserVote === ReviewVoteType.UNHELPFUL
                          ? "You voted No"
                          : "No"}
                      </span>
                      <span className="font-mono text-zinc-400 ml-0.5">
                        {review.unhelpfulCount}
                      </span>
                    </button>
                  </div>

                  <div className="flex items-center gap-3">
                    {isMyReview && (
                      <button
                        type="button"
                        onClick={() => handleDeleteReview(review.id)}
                        className="text-zinc-400 hover:text-rose-500 transition-colors p-1"
                        title="Delete review"
                      >
                        <Trash2 size={14} />
                      </button>
                    )}

                    <button
                      type="button"
                      onClick={() => toggleDiscussionThread(review.id)}
                      className="text-zinc-500 hover:text-blue-500 flex items-center gap-1 cursor-pointer font-medium"
                    >
                      <MessageSquare size={13} />
                      <span>
                        {comments} {comments === 1 ? "Discussion reply" : "Discussion replies"}
                      </span>
                      {isThreadOpen ? <ChevronUp size={14} /> : <ChevronDown size={14} />}
                    </button>
                  </div>
                </div>

                {/* Flat discussion comments thread */}
                {isThreadOpen && (
                  <ReviewCommentsThread review={review} onCommentChanged={() => refetchReviews()} />
                )}
              </div>
            );
          })
        )}
      </div>
    </div>
  );
}

function ReviewCommentsThread({
  review,
  onCommentChanged,
}: {
  review: Review;
  onCommentChanged: () => void;
}) {
  const { user, isAuthenticated } = useAuth();
  const [replyText, setReplyText] = useState("");
  const [isPosting, setIsPosting] = useState(false);

  const {
    data: commentsRes,
    isLoading,
    refetch,
  } = useQuery(listReviewComments, {
    reviewId: review.id,
  });
  const { mutateAsync: mutateCreateComment } = useMutation(createReviewComment);
  const { mutateAsync: mutateDeleteComment } = useMutation(deleteReviewComment);

  const comments = commentsRes?.comments || [];

  async function handlePost() {
    if (!replyText.trim()) return;
    setIsPosting(true);
    try {
      await mutateCreateComment({
        reviewId: review.id,
        body: replyText.trim(),
      });
      setReplyText("");
      await refetch();
      onCommentChanged();
    } catch (err: unknown) {
      alert(err instanceof Error ? err.message : "Failed to post comment");
    } finally {
      setIsPosting(false);
    }
  }

  async function handleDelete(commentUuid: string) {
    if (!window.confirm("Delete this comment?")) return;
    try {
      await mutateDeleteComment({ commentUuid });
      await refetch();
      onCommentChanged();
    } catch (err: unknown) {
      alert(err instanceof Error ? err.message : "Failed to delete comment");
    }
  }

  return (
    <div className="space-y-3 pt-3 pl-4 border-l-2 border-zinc-200 dark:border-zinc-800">
      {isLoading ? (
        <div className="py-2 flex justify-center text-zinc-400">
          <Loader2 size={14} className="animate-spin text-blue-500" />
        </div>
      ) : (
        comments.map((c) => (
          <div
            key={c.uuid}
            className="p-3 rounded-xl bg-zinc-50 dark:bg-zinc-950/60 border border-zinc-200 dark:border-zinc-800/80 text-xs space-y-1"
          >
            <div className="flex items-center justify-between text-zinc-500">
              <UserBadge username={c.authorUsername} size="xs" />
              <div className="flex items-center gap-2">
                <span className="text-[10px] text-zinc-400">
                  {formatRelativeDate(c.createdAt?.seconds)}
                </span>
                {user?.uuid === c.userUuid && (
                  <button
                    type="button"
                    onClick={() => handleDelete(c.uuid)}
                    className="text-zinc-400 hover:text-rose-500 p-0.5"
                    title="Delete comment"
                  >
                    <Trash2 size={11} />
                  </button>
                )}
              </div>
            </div>
            <p className="text-zinc-700 dark:text-zinc-300">{c.body}</p>
          </div>
        ))
      )}

      {/* Add reply input */}
      {isAuthenticated && (
        <div className="flex gap-2 pt-1">
          <input
            type="text"
            value={replyText}
            onChange={(e) => setReplyText(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === "Enter") {
                e.preventDefault();
                handlePost();
              }
            }}
            placeholder={`Reply to @${review.authorUsername || "author"}'s review...`}
            className="flex-1 rounded-lg border border-zinc-200 dark:border-zinc-800 bg-zinc-50 dark:bg-zinc-950 px-3 py-1.5 text-xs text-zinc-900 dark:text-zinc-200 placeholder-zinc-400 focus:outline-hidden focus:border-blue-500"
          />
          <button
            type="button"
            onClick={handlePost}
            disabled={isPosting || !replyText.trim()}
            className="px-3 py-1.5 rounded-lg bg-zinc-900 hover:bg-zinc-800 dark:bg-zinc-800 dark:hover:bg-zinc-700 text-white text-xs font-medium disabled:opacity-50 transition-colors flex items-center gap-1 cursor-pointer"
          >
            {isPosting ? <Loader2 size={12} className="animate-spin" /> : <Send size={12} />}
            <span>Post</span>
          </button>
        </div>
      )}
    </div>
  );
}
