import { useState, useRef, useEffect, useCallback } from "react";
import { createPortal } from "react-dom";
import { Link } from "react-router-dom";
import { useQuery } from "@connectrpc/connect-query";
import { getPublicProfile } from "@quadsmith/shared/gen/quadsmith/api/user-UserService_connectquery";
import { getAvatarPalette, getInitial } from "@quadsmith/shared";
import { Calendar, Wrench, Star, ArrowRight } from "lucide-react";

export interface UserHoverCardProps {
  username?: string | null;
  profile?: any;
  children: React.ReactNode;
  className?: string;
  as?: React.ElementType;
  placement?: "auto" | "top" | "bottom" | "left" | "right";
}

function formatJoinedDate(seconds?: bigint | number): string {
  if (!seconds) return "Recently";
  try {
    const ms = Number(seconds) * 1000;
    const date = new Date(ms);
    return date.toLocaleDateString(undefined, { month: "short", year: "numeric" });
  } catch {
    return "Recently";
  }
}

export function UserHoverCard({
  username,
  profile: initialProfile,
  children,
  className = "",
  as = "span",
  placement = "auto",
}: UserHoverCardProps) {
  const cleanUsername = username?.replace(/^@/, "").trim() || "";
  const isValidUser = cleanUsername.length > 0 && cleanUsername !== "community";

  const [isOpen, setIsOpen] = useState(false);
  const [coords, setCoords] = useState<{ top: number; left: number; placeAbove: boolean }>({
    top: 0,
    left: 0,
    placeAbove: false,
  });

  const triggerRef = useRef<HTMLElement>(null);
  const cardRef = useRef<HTMLDivElement>(null);
  const openTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const closeTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  // Fetch Public Profile on hover if not provided
  const { data: fetchedProfile, isLoading: isQueryLoading } = useQuery(
    getPublicProfile,
    { id: cleanUsername },
    { enabled: isOpen && isValidUser && !initialProfile },
  );

  const profile = initialProfile || fetchedProfile;
  const isLoading = !profile && isQueryLoading;

  const updatePosition = useCallback(() => {
    if (!triggerRef.current) return;
    const triggerRect = triggerRef.current.getBoundingClientRect();
    const cardWidth = 280;
    const estimatedHeight = 180;

    let left = triggerRect.left;
    let top = triggerRect.bottom + 8;
    let placeAbove = false;

    if (placement === "top") {
      placeAbove = true;
      top = triggerRect.top - estimatedHeight - 8;
    } else if (placement === "bottom") {
      placeAbove = false;
      top = triggerRect.bottom + 8;
    } else {
      // Auto placement: place above if near bottom of screen
      const spaceBelow = window.innerHeight - triggerRect.bottom;
      if (spaceBelow < estimatedHeight + 16 && triggerRect.top > estimatedHeight + 16) {
        placeAbove = true;
        top = triggerRect.top - estimatedHeight - 8;
      } else {
        placeAbove = false;
        top = triggerRect.bottom + 8;
      }
    }

    // Clamp horizontally within viewport
    if (left + cardWidth > window.innerWidth - 16) {
      left = window.innerWidth - cardWidth - 16;
    }
    if (left < 16) {
      left = 16;
    }

    setCoords({
      top: top + window.scrollY,
      left: left + window.scrollX,
      placeAbove,
    });
  }, [placement]);

  const handleMouseEnter = () => {
    if (!isValidUser) return;
    if (typeof window !== "undefined" && window.matchMedia) {
      if (window.matchMedia("(hover: none) and (pointer: coarse)").matches) {
        return;
      }
    }
    if (closeTimerRef.current) {
      clearTimeout(closeTimerRef.current);
      closeTimerRef.current = null;
    }
    openTimerRef.current = setTimeout(() => {
      updatePosition();
      setIsOpen(true);
    }, 200); // 200ms hover intent
  };

  const handleMouseLeave = () => {
    if (openTimerRef.current) {
      clearTimeout(openTimerRef.current);
      openTimerRef.current = null;
    }
    closeTimerRef.current = setTimeout(() => {
      setIsOpen(false);
    }, 150); // 150ms grace period
  };

  useEffect(() => {
    return () => {
      if (openTimerRef.current) clearTimeout(openTimerRef.current);
      if (closeTimerRef.current) clearTimeout(closeTimerRef.current);
    };
  }, []);

  const Component = as;

  if (!isValidUser) {
    return <Component className={className}>{children}</Component>;
  }

  const palette = getAvatarPalette(cleanUsername);
  const initial = getInitial(cleanUsername);

  const hoverCard =
    isOpen && typeof document !== "undefined"
      ? createPortal(
          <div
            ref={cardRef}
            data-testid="user-hover-card"
            className="fixed z-[9999] pointer-events-auto transition-opacity duration-150"
            style={{
              top: coords.top - window.scrollY,
              left: coords.left - window.scrollX,
              width: 280,
              maxWidth: "calc(100vw - 2rem)",
            }}
            onMouseEnter={() => {
              if (closeTimerRef.current) {
                clearTimeout(closeTimerRef.current);
                closeTimerRef.current = null;
              }
            }}
            onMouseLeave={handleMouseLeave}
          >
            <div className="rounded-2xl border border-zinc-200 dark:border-zinc-800 bg-white dark:bg-zinc-900 shadow-xl p-4 text-left backdrop-blur-md">
              {isLoading ? (
                <div className="space-y-3 animate-pulse">
                  <div className="flex items-center gap-3">
                    <div className="w-10 h-10 rounded-full bg-zinc-200 dark:bg-zinc-800" />
                    <div className="space-y-1.5 flex-1">
                      <div className="h-4 bg-zinc-200 dark:bg-zinc-800 rounded w-24" />
                      <div className="h-3 bg-zinc-200 dark:bg-zinc-800 rounded w-16" />
                    </div>
                  </div>
                  <div className="h-8 bg-zinc-100 dark:bg-zinc-800/50 rounded-lg" />
                </div>
              ) : (
                <>
                  {/* Header row */}
                  <div className="flex items-center gap-3">
                    {profile?.avatarUrl ? (
                      <img
                        src={profile.avatarUrl}
                        alt={cleanUsername}
                        className="w-10 h-10 rounded-full object-cover border border-zinc-200 dark:border-zinc-800 shrink-0"
                      />
                    ) : (
                      <div
                        className={`w-10 h-10 rounded-full border flex items-center justify-center font-bold text-sm shadow-2xs shrink-0 ${palette.badgeClass}`}
                      >
                        {initial}
                      </div>
                    )}
                    <div className="min-w-0 flex-1">
                      <Link
                        to={`/users/${cleanUsername}`}
                        className="font-bold text-sm text-zinc-900 dark:text-zinc-100 hover:text-blue-600 dark:hover:text-blue-400 hover:underline block truncate"
                      >
                        @{cleanUsername}
                      </Link>
                      <span className="flex items-center gap-1 text-[11px] text-zinc-500 dark:text-zinc-400 mt-0.5">
                        <Calendar size={11} className="shrink-0" />
                        <span>Joined {formatJoinedDate(profile?.createdAt?.seconds)}</span>
                      </span>
                    </div>
                  </div>

                  {/* Bio (if available) */}
                  {profile?.bio && (
                    <p className="mt-2.5 text-xs text-zinc-600 dark:text-zinc-300 line-clamp-2 leading-relaxed">
                      {profile.bio}
                    </p>
                  )}

                  {/* Pilot Stats */}
                  <div className="grid grid-cols-2 gap-2 mt-3">
                    <div className="flex items-center gap-1.5 px-2.5 py-1.5 rounded-lg bg-zinc-50 dark:bg-zinc-800/60 border border-zinc-200/60 dark:border-zinc-800 text-xs font-medium text-zinc-700 dark:text-zinc-300">
                      <Wrench size={13} className="text-blue-500 shrink-0" />
                      <span>
                        {profile?.buildCount ?? 0} {profile?.buildCount === 1 ? "build" : "builds"}
                      </span>
                    </div>
                    <div className="flex items-center gap-1.5 px-2.5 py-1.5 rounded-lg bg-zinc-50 dark:bg-zinc-800/60 border border-zinc-200/60 dark:border-zinc-800 text-xs font-medium text-zinc-700 dark:text-zinc-300">
                      <Star size={13} className="text-amber-500 fill-amber-500/20 shrink-0" />
                      <span>{profile?.reputationScore ?? 0} rep</span>
                    </div>
                  </div>

                  {/* Action Link */}
                  <div className="mt-3 pt-2 border-t border-zinc-100 dark:border-zinc-800/80 flex justify-end">
                    <Link
                      to={`/users/${cleanUsername}`}
                      className="text-xs font-medium text-blue-600 dark:text-blue-400 hover:underline inline-flex items-center gap-1 transition-colors"
                    >
                      View profile
                      <ArrowRight size={12} />
                    </Link>
                  </div>
                </>
              )}
            </div>
          </div>,
          document.body,
        )
      : null;

  return (
    <>
      <Component
        ref={triggerRef}
        className={className}
        onMouseEnter={handleMouseEnter}
        onMouseLeave={handleMouseLeave}
      >
        {children}
      </Component>
      {hoverCard}
    </>
  );
}
