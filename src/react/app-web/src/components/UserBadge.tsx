import React from "react";
import { Link } from "react-router-dom";
import { getAvatarPalette, getInitial } from "@quadsmith/shared";
import { UserHoverCard } from "./UserHoverCard";
import { useOptionalAuth } from "../context/AuthContext";

export interface UserBadgeProps {
  username?: string | null;
  size?: "xs" | "sm" | "md" | "lg";
  prefix?: string;
  withAt?: boolean;
  withAvatar?: boolean;
  withHoverCard?: boolean;
  to?: string | null;
  className?: string;
  onClick?: (e: React.MouseEvent) => void;
  isAuthor?: boolean;
  isSelf?: boolean;
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
  isAuthor,
  isSelf,
}: UserBadgeProps) {
  const auth = useOptionalAuth();
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
        badgeClass: "bg-zinc-800 text-zinc-300 border-zinc-700",
        bgClass: "bg-zinc-800",
        textClass: "text-zinc-300",
        borderClass: "border-zinc-700",
      };

  const initial = getInitial(cleanUsername);

  const isSelfBadge =
    isSelf !== undefined
      ? isSelf
      : Boolean(
          auth?.user?.id &&
          cleanUsername &&
          auth.user.id.toLowerCase() === cleanUsername.toLowerCase(),
        );

  const fallbackName = cleanUsername || "community";
  const nameWithAt = withAt && !fallbackName.startsWith("@") ? `@${fallbackName}` : fallbackName;

  const paddingClasses = {
    xs: "text-[10px] px-2 py-0.5 rounded-md gap-1",
    sm: "text-xs px-2.5 py-1 rounded-md gap-1",
    md: "text-sm px-3 py-1.5 rounded-md gap-1.5",
    lg: "text-lg sm:text-xl px-4 py-1.5 rounded-xl gap-2",
  }[size];

  let borderClasses = `border-[1px] ${palette.borderClass}`;
  const fontClasses = "font-bold";
  let badgeTitle: string | undefined = undefined;
  let shadowClasses = "";

  if (isAuthor) {
    borderClasses = "border-[1px] border-[#fbbf24]";
    badgeTitle = "Author";
  }

  if (isSelfBadge) {
    shadowClasses = "shadow-[0_0_6px_2px_rgba(161,161,170,0.65)]";
    if (!isAuthor) {
      badgeTitle = "You";
    }
  }

  const badgeContent = (
    <span
      data-testid="user-badge"
      title={badgeTitle}
      className={`inline-flex items-center font-mono shrink-0 transition-all select-text ${palette.bgClass} ${palette.textClass} ${borderClasses} ${shadowClasses} ${fontClasses} ${paddingClasses} ${className}`}
    >
      {withAvatar && (
        <span
          className={`w-3.5 h-3.5 rounded-full flex items-center justify-center font-bold text-[8px] border shrink-0 ${palette.borderClass} ${palette.bgClass} ${palette.textClass}`}
        >
          {initial}
        </span>
      )}
      <span className="truncate">{nameWithAt}</span>
    </span>
  );

  const linkTo =
    to === null
      ? undefined
      : to !== undefined
        ? to
        : isValidUser
          ? `/users/${cleanUsername}`
          : undefined;

  let pillElement: React.ReactNode;
  if (linkTo) {
    pillElement = (
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
    pillElement = badgeContent;
  }

  if (withHoverCard && isValidUser) {
    pillElement = <UserHoverCard username={cleanUsername}>{pillElement}</UserHoverCard>;
  }

  if (prefix) {
    const prefixSize = {
      xs: "text-[10px]",
      sm: "text-xs",
      md: "text-sm",
      lg: "text-base",
    }[size];

    return (
      <span data-testid="user-badge-container" className="inline-flex items-center gap-1.5">
        <span className={`text-zinc-400 font-sans ${prefixSize}`}>{prefix.trim()} </span>
        {pillElement}
      </span>
    );
  }

  return <>{pillElement}</>;
}
