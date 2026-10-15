import React from "react";
import { Link } from "react-router-dom";
import { getAvatarPalette, getInitial } from "@quadsmith/shared";
import { UserHoverCard } from "./UserHoverCard";

export interface UserBadgeProps {
  username?: string | null;
  size?: "xs" | "sm" | "md";
  prefix?: string;
  withAt?: boolean;
  withAvatar?: boolean;
  withHoverCard?: boolean;
  to?: string;
  className?: string;
  onClick?: (e: React.MouseEvent) => void;
}

export function UserBadge({
  username,
  size = "xs",
  prefix,
  withAt = true,
  withAvatar = false,
  withHoverCard = true,
  to,
  className = "",
  onClick,
}: UserBadgeProps) {
  const raw = (username || "").trim();
  const cleanUsername = raw.startsWith("@") ? raw.slice(1) : raw;
  const isCommunity = cleanUsername.toLowerCase() === "community";
  const isValidUser =
    cleanUsername !== "" &&
    !isCommunity &&
    cleanUsername.toLowerCase() !== "anonymous" &&
    cleanUsername.toLowerCase() !== "anonymous pilot";

  const palette = isValidUser
    ? getAvatarPalette(cleanUsername)
    : {
        name: "zinc",
        badgeClass:
          "bg-zinc-100 text-zinc-700 border-zinc-200 dark:bg-zinc-800 dark:text-zinc-300 dark:border-zinc-700",
        bgClass: "bg-zinc-100 dark:bg-zinc-800",
        textClass: "text-zinc-700 dark:text-zinc-300",
        borderClass: "border-zinc-200 dark:border-zinc-700",
      };

  const initial = getInitial(cleanUsername);

  const sizeClasses = {
    xs: "text-[10px] px-1.5 py-0.5 rounded-md gap-1",
    sm: "text-xs px-2 py-0.5 rounded-md gap-1",
    md: "text-sm px-2.5 py-1 rounded-md gap-1.5",
  }[size];

  const fallbackName = cleanUsername || "community";
  const nameWithAt = withAt && !fallbackName.startsWith("@") ? `@${fallbackName}` : fallbackName;
  const displayText = prefix ? `${prefix}${nameWithAt}` : nameWithAt;

  const badgeContent = (
    <span
      data-testid="user-badge"
      className={`inline-flex items-center font-medium border font-mono shrink-0 transition-all select-none ${palette.badgeClass} ${sizeClasses} ${className}`}
    >
      {withAvatar && (
        <span
          className={`w-3.5 h-3.5 rounded-full flex items-center justify-center font-bold text-[8px] border shrink-0 ${palette.borderClass} ${palette.bgClass} ${palette.textClass}`}
        >
          {initial}
        </span>
      )}
      <span className="truncate">{displayText}</span>
    </span>
  );

  const linkTo = to !== undefined ? to : isValidUser ? `/users/${cleanUsername}` : undefined;

  let content: React.ReactNode;
  if (linkTo) {
    content = (
      <Link
        to={linkTo}
        onClick={(e) => {
          e.stopPropagation();
          onClick?.(e);
        }}
        className="inline-flex hover:opacity-85 hover:shadow-2xs transition-all cursor-pointer no-underline text-inherit"
      >
        {badgeContent}
      </Link>
    );
  } else {
    content = badgeContent;
  }

  if (withHoverCard && isValidUser) {
    return <UserHoverCard username={cleanUsername}>{content}</UserHoverCard>;
  }

  return <>{content}</>;
}
