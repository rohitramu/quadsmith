import { Link } from "react-router-dom";
import { getCollectionColor, COLLECTION_SHORT_NAMES } from "../lib/collectionColors";

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
  to,
  className = "",
}: CollectionBadgeProps) {
  const color = getCollectionColor(collection);

  // Prefer short chip name for collections like FC, ESC, VTX, Props, RX, TX
  let displayText = label;
  if (!displayText) {
    displayText = color.shortName || color.name;
  } else {
    const key = displayText.trim().toLowerCase();
    if (COLLECTION_SHORT_NAMES[key]) {
      displayText = COLLECTION_SHORT_NAMES[key];
    }
  }

  const sizeClasses = {
    xs: "text-[10px] px-1.5 py-0.5 rounded-md",
    sm: "text-xs px-2 py-0.5 rounded-md",
    md: "text-sm px-2 py-0.5 rounded-md",
  }[size];

  const badgeContent = (
    <span
      data-testid="collection-badge"
      title={color.name}
      className={`inline-flex items-center font-medium border font-mono shrink-0 transition-colors ${color.badgeClass} ${sizeClasses} ${className}`}
    >
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
