import { Link } from "react-router-dom";
import { getCollectionColor, COLLECTION_SHORT_NAMES } from "../lib/collectionColors";
import { CollectionIcon } from "./CollectionIcon";

export interface CollectionBadgeProps {
  collection?: string | null;
  label?: string | null;
  size?: "xs" | "sm" | "md";
  variant?: "text" | "pill";
  withIcon?: boolean;
  withDot?: boolean;
  to?: string;
  className?: string;
}

export function CollectionBadge({
  collection,
  label,
  size = "xs",
  variant = "text",
  withIcon = true,
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

  const iconSizes = {
    xs: 12,
    sm: 14,
    md: 16,
  };

  const textSizes = {
    xs: "text-[10px]",
    sm: "text-xs",
    md: "text-sm",
  };

  const pillSizes = {
    xs: "text-[10px] px-1.5 py-0.5 rounded-md",
    sm: "text-xs px-2 py-0.5 rounded-md",
    md: "text-sm px-2.5 py-0.5 rounded-md",
  };

  const badgeContent =
    variant === "pill" ? (
      <span
        data-testid="collection-badge"
        title={color.name}
        className={`inline-flex items-center font-medium border font-mono shrink-0 transition-colors ${color.badgeClass} ${pillSizes[size]} ${className}`}
      >
        <span className="truncate">{displayText}</span>
      </span>
    ) : (
      <span
        data-testid="collection-badge"
        title={color.name}
        className={`inline-flex items-center gap-1.5 font-medium font-mono shrink-0 transition-colors text-zinc-700 dark:text-zinc-300 select-none ${textSizes[size]} ${className}`}
      >
        {withIcon && <CollectionIcon collection={collection} color={true} size={iconSizes[size]} />}
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
