import { Link } from "react-router-dom";
import { getCollectionColor } from "../lib/collectionColors";

export interface CollectionBadgeProps {
  collection?: string | null;
  label?: string | null;
  size?: "xs" | "sm" | "md";
  withDot?: boolean;
  to?: string;
  className?: string;
}

export function CollectionBadge({
  collection,
  label,
  size = "xs",
  withDot = false,
  to,
  className = "",
}: CollectionBadgeProps) {
  const color = getCollectionColor(collection);
  const displayText = label || color.name;

  const sizeClasses = {
    xs: "text-[10px] px-1.5 py-0.5 rounded-md",
    sm: "text-xs px-2 py-0.5 rounded-md",
    md: "text-sm px-2.5 py-1 rounded-lg",
  }[size];

  const dotSizeClasses = {
    xs: "w-1.5 h-1.5",
    sm: "w-2 h-2",
    md: "w-2.5 h-2.5",
  }[size];

  const badgeContent = (
    <span
      data-testid="collection-badge"
      className={`inline-flex items-center gap-1.5 font-medium border font-mono shrink-0 transition-colors ${color.badgeClass} ${sizeClasses} ${className}`}
    >
      {withDot && (
        <span
          className={`rounded-full shrink-0 ${color.dotClass} ${dotSizeClasses}`}
          aria-hidden="true"
        />
      )}
      <span className="truncate">{displayText}</span>
    </span>
  );

  if (to) {
    return (
      <Link to={to} className="inline-flex hover:opacity-85 transition-opacity">
        {badgeContent}
      </Link>
    );
  }

  return badgeContent;
}
