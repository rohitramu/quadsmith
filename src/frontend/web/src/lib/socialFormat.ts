import { ReviewTier } from "../gen/quadsmith/api/social_pb";

export function getReviewTierLabel(tier: ReviewTier): string {
  switch (tier) {
    case ReviewTier.OVERWHELMINGLY_POSITIVE:
      return "Overwhelmingly Positive";
    case ReviewTier.VERY_POSITIVE:
      return "Very Positive";
    case ReviewTier.POSITIVE:
      return "Positive";
    case ReviewTier.MIXED:
      return "Mixed";
    case ReviewTier.MOSTLY_NEGATIVE:
      return "Mostly Negative";
    case ReviewTier.OVERWHELMINGLY_NEGATIVE:
      return "Overwhelmingly Negative";
    case ReviewTier.FEW_REVIEWS:
      return "Need More Reviews";
    default:
      return "No Reviews Yet";
  }
}

export function getReviewTierBadgeClass(tier: ReviewTier): {
  text: string;
  bg: string;
  border: string;
} {
  switch (tier) {
    case ReviewTier.OVERWHELMINGLY_POSITIVE:
    case ReviewTier.VERY_POSITIVE:
    case ReviewTier.POSITIVE:
      return {
        text: "text-emerald-500 dark:text-emerald-400",
        bg: "bg-emerald-500/10 dark:bg-emerald-950/30",
        border: "border-emerald-500/20 dark:border-emerald-500/30",
      };
    case ReviewTier.MIXED:
      return {
        text: "text-amber-500 dark:text-amber-400",
        bg: "bg-amber-500/10 dark:bg-amber-950/30",
        border: "border-amber-500/20 dark:border-amber-500/30",
      };
    case ReviewTier.MOSTLY_NEGATIVE:
    case ReviewTier.OVERWHELMINGLY_NEGATIVE:
      return {
        text: "text-rose-500 dark:text-rose-400",
        bg: "bg-rose-500/10 dark:bg-rose-950/30",
        border: "border-rose-500/20 dark:border-rose-500/30",
      };
    case ReviewTier.FEW_REVIEWS:
    default:
      return {
        text: "text-zinc-500 dark:text-zinc-400",
        bg: "bg-zinc-500/10 dark:bg-zinc-800/50",
        border: "border-zinc-500/20 dark:border-zinc-700/50",
      };
  }
}

export function formatRelativeDate(seconds?: bigint | number): string {
  if (!seconds) return "";
  const sec = typeof seconds === "bigint" ? Number(seconds) : seconds;
  const d = new Date(sec * 1000);
  return d.toLocaleDateString(undefined, {
    month: "short",
    day: "numeric",
    year: "numeric",
  });
}
