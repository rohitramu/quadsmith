import { useState, useEffect, useLayoutEffect, useRef } from "react";
import { createPortal } from "react-dom";

export interface EllipsisHintTarget {
  element: HTMLElement;
  text: string;
}

export function isElementEllipsized(el: HTMLElement): boolean {
  if (!el || !el.textContent || !el.textContent.trim()) return false;

  const tag = el.tagName.toUpperCase();
  if (tag === "INPUT" || tag === "TEXTAREA" || tag === "SELECT") {
    return false;
  }

  const style = window.getComputedStyle(el);
  const textOverflow = style.textOverflow;
  const webkitLineClamp = style.webkitLineClamp;
  const hasLineClamp =
    webkitLineClamp !== "none" && webkitLineClamp !== "" && webkitLineClamp !== undefined;

  const isOverflowHidden =
    style.overflow === "hidden" ||
    style.overflow === "clip" ||
    style.overflowX === "hidden" ||
    style.overflowX === "clip" ||
    style.overflowY === "hidden" ||
    style.overflowY === "clip";

  const hasTruncateClass =
    el.classList.contains("truncate") ||
    Array.from(el.classList).some((c) => c.startsWith("line-clamp-"));

  const isHorizontallyOverflowing = el.scrollWidth > el.clientWidth + 1;
  const isVerticallyOverflowing = el.scrollHeight > el.clientHeight + 1;

  if (hasLineClamp && isOverflowHidden && isVerticallyOverflowing) {
    return true;
  }

  if (
    (textOverflow === "ellipsis" || hasTruncateClass) &&
    isOverflowHidden &&
    (isHorizontallyOverflowing || isVerticallyOverflowing)
  ) {
    return true;
  }

  if (textOverflow === "ellipsis" && isHorizontallyOverflowing) {
    return true;
  }

  return false;
}

export function getElementFullText(el: HTMLElement): string {
  const dataFullText = el.getAttribute("data-full-text");
  if (dataFullText && dataFullText.trim()) return dataFullText.trim();

  // If there's an existing title attribute, save it
  const title = el.getAttribute("title");
  if (title && title.trim()) return title.trim();

  const text = el.innerText || el.textContent || "";
  return text.replace(/\s+/g, " ").trim();
}

export function findEllipsizedTarget(target: EventTarget | null): EllipsisHintTarget | null {
  if (!target || !(target instanceof HTMLElement)) return null;
  if (target === document.body || target === document.documentElement) return null;

  // 1. Check target itself and ancestors up to 4 levels
  let current: HTMLElement | null = target;
  let depth = 0;
  while (
    current &&
    current !== document.body &&
    current !== document.documentElement &&
    depth < 4
  ) {
    if (isElementEllipsized(current)) {
      const text = getElementFullText(current);
      if (text) return { element: current, text };
    }
    current = current.parentElement;
    depth++;
  }

  // 2. Check direct children with truncate or line-clamp classes
  const child = target.querySelector<HTMLElement>(".truncate, [class*='line-clamp-']");
  if (child && isElementEllipsized(child)) {
    const text = getElementFullText(child);
    if (text) return { element: child, text };
  }

  return null;
}

interface ActiveHint {
  element: HTMLElement;
  text: string;
  rect: DOMRect;
  placeAbove: boolean;
}

export function EllipsisHint() {
  const [activeHint, setActiveHint] = useState<ActiveHint | null>(null);
  const [coords, setCoords] = useState<{
    top: number;
    left: number;
    arrowOffset: number;
  } | null>(null);

  const hoverTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const currentTargetRef = useRef<HTMLElement | null>(null);
  const tooltipRef = useRef<HTMLDivElement>(null);

  const clearHoverTimer = () => {
    if (hoverTimerRef.current) {
      clearTimeout(hoverTimerRef.current);
      hoverTimerRef.current = null;
    }
  };

  const restoreElementTitle = (el: HTMLElement | null) => {
    if (el && el.dataset.storedTitle !== undefined) {
      el.title = el.dataset.storedTitle;
      delete el.dataset.storedTitle;
    }
  };

  useEffect(() => {
    const handlePointerOver = (e: PointerEvent | MouseEvent) => {
      const found = findEllipsizedTarget(e.target);
      if (!found) return;

      const { element, text } = found;

      if (currentTargetRef.current === element) {
        return;
      }

      clearHoverTimer();
      restoreElementTitle(currentTargetRef.current);
      currentTargetRef.current = element;

      // Temporarily stash native title to prevent default OS tooltip collision
      if (element.hasAttribute("title") && element.title) {
        element.dataset.storedTitle = element.title;
        element.title = "";
      }

      hoverTimerRef.current = setTimeout(() => {
        if (!element.isConnected) return;
        const rect = element.getBoundingClientRect();
        if (rect.width === 0 && rect.height === 0) return;

        const placeAbove = rect.top > 60;
        setActiveHint({
          element,
          text,
          rect,
          placeAbove,
        });
      }, 150); // 150ms hover intent
    };

    const handlePointerOut = (e: PointerEvent | MouseEvent) => {
      const current = currentTargetRef.current;
      if (!current) return;

      const related = e.relatedTarget;
      if (related && related instanceof Node && current.contains(related)) {
        return;
      }

      clearHoverTimer();
      restoreElementTitle(current);
      currentTargetRef.current = null;
      setActiveHint(null);
      setCoords(null);
    };

    const handleScrollOrEscape = (e: Event) => {
      if (e instanceof KeyboardEvent && e.key !== "Escape") {
        return;
      }
      clearHoverTimer();
      restoreElementTitle(currentTargetRef.current);
      currentTargetRef.current = null;
      setActiveHint(null);
      setCoords(null);
    };

    document.addEventListener("pointerover", handlePointerOver, true);
    document.addEventListener("pointerout", handlePointerOut, true);
    window.addEventListener("scroll", handleScrollOrEscape, { capture: true, passive: true });
    window.addEventListener("keydown", handleScrollOrEscape);

    return () => {
      clearHoverTimer();
      restoreElementTitle(currentTargetRef.current);
      document.removeEventListener("pointerover", handlePointerOver, true);
      document.removeEventListener("pointerout", handlePointerOut, true);
      window.removeEventListener("scroll", handleScrollOrEscape, true);
      window.removeEventListener("keydown", handleScrollOrEscape);
    };
  }, []);

  useLayoutEffect(() => {
    if (!activeHint) {
      setCoords(null);
      return;
    }

    if (!tooltipRef.current) return;

    const tooltipRect = tooltipRef.current.getBoundingClientRect();
    const targetRect = activeHint.rect;

    // Center horizontally over target
    const targetCenter = targetRect.left + targetRect.width / 2;
    let left = targetCenter - tooltipRect.width / 2;

    // Clamp within viewport margins
    const minLeft = 12;
    const maxLeft = Math.max(minLeft, window.innerWidth - tooltipRect.width - 12);
    const clampedLeft = Math.max(minLeft, Math.min(left, maxLeft));

    // Calculate arrow position relative to tooltip left edge
    const arrowOffset = Math.max(12, Math.min(targetCenter - clampedLeft, tooltipRect.width - 12));

    // Place above or below target
    const placeAbove =
      targetRect.top > tooltipRect.height + 16 || targetRect.bottom > window.innerHeight - 60;

    const top = placeAbove ? targetRect.top - tooltipRect.height - 8 : targetRect.bottom + 8;

    setCoords({
      top,
      left: clampedLeft,
      arrowOffset,
    });
  }, [activeHint]);

  if (!activeHint) return null;

  return createPortal(
    <div
      ref={tooltipRef}
      role="tooltip"
      data-testid="ellipsis-hint"
      style={{
        position: "fixed",
        top: coords
          ? `${coords.top}px`
          : `${activeHint.placeAbove ? activeHint.rect.top - 40 : activeHint.rect.bottom + 8}px`,
        left: coords ? `${coords.left}px` : `${Math.max(12, activeHint.rect.left)}px`,
        maxWidth: "min(calc(100vw - 32px), 480px)",
        zIndex: 99999,
        opacity: coords ? 1 : 0,
      }}
      className="fixed z-[99999] pointer-events-none px-3 py-1.5 rounded-lg text-xs font-medium leading-relaxed bg-zinc-900/95 dark:bg-zinc-800/95 text-zinc-100 dark:text-zinc-100 shadow-xl border border-zinc-700/80 dark:border-zinc-600/80 backdrop-blur-xs select-none break-words transition-opacity duration-150"
    >
      {activeHint.text}
      {coords && (
        <div
          style={{ left: `${coords.arrowOffset}px` }}
          className={`absolute -translate-x-1/2 w-2 h-2 rotate-45 bg-zinc-900 dark:bg-zinc-800 border-zinc-700/80 dark:border-zinc-600/80 ${
            activeHint.placeAbove
              ? "bottom-[-5px] border-r border-b"
              : "top-[-5px] border-l border-t"
          }`}
        />
      )}
    </div>,
    document.body,
  );
}
