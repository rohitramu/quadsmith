import React, { useState, useMemo, useRef, useEffect } from "react";
import { Copy, Check } from "lucide-react";
import { formatCelFilterForDisplay } from "@quadsmith/shared";

export interface CompatibilityHoverCardProps {
  filter?: string | null;
  children?: React.ReactNode;
  className?: string;
}

/**
 * Renders trigger text with a dotted underline and a hover card.
 * The hover card contains a text box showing the full filter string
 * with a horizontal scrollbar if needed to avoid text wrapping,
 * as well as a copy button to copy the filter string to clipboard.
 */
export function CompatibilityHoverCard({
  filter,
  children = "Compatible only",
  className = "",
}: CompatibilityHoverCardProps) {
  const [isOpen, setIsOpen] = useState(false);
  const [copied, setCopied] = useState(false);
  const [placement, setPlacement] = useState<"top" | "bottom">("top");
  const triggerRef = useRef<HTMLSpanElement>(null);
  const timerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  const formattedFilter = useMemo(() => formatCelFilterForDisplay(filter), [filter]);

  useEffect(() => {
    return () => {
      if (timerRef.current) {
        clearTimeout(timerRef.current);
      }
    };
  }, []);

  if (!filter || !formattedFilter) {
    return <span className={className}>{children}</span>;
  }

  const handleMouseEnter = () => {
    if (timerRef.current) {
      clearTimeout(timerRef.current);
      timerRef.current = null;
    }
    if (triggerRef.current) {
      const rect = triggerRef.current.getBoundingClientRect();
      if (rect.top < 220) {
        setPlacement("bottom");
      } else {
        setPlacement("top");
      }
    }
    setIsOpen(true);
  };

  const handleMouseLeave = () => {
    timerRef.current = setTimeout(() => {
      setIsOpen(false);
    }, 150);
  };

  const handleCopy = async (e: React.MouseEvent) => {
    e.stopPropagation();
    e.preventDefault();
    try {
      const nav = typeof window !== "undefined" ? window.navigator : navigator;
      if (nav?.clipboard?.writeText) {
        await nav.clipboard.writeText(formattedFilter);
      } else {
        const textarea = document.createElement("textarea");
        textarea.value = formattedFilter;
        textarea.style.position = "fixed";
        textarea.style.opacity = "0";
        document.body.appendChild(textarea);
        textarea.select();
        document.execCommand("copy");
        document.body.removeChild(textarea);
      }
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch (err) {
      console.error("Failed to copy filter:", err);
    }
  };

  return (
    <span className="relative inline-flex items-center">
      <span
        ref={triggerRef}
        onMouseEnter={handleMouseEnter}
        onMouseLeave={handleMouseLeave}
        className={`underline decoration-dotted underline-offset-2 decoration-zinc-400 dark:decoration-zinc-500 hover:decoration-zinc-700 dark:hover:decoration-zinc-300 transition-colors cursor-help ${className}`}
      >
        {children}
      </span>

      {isOpen && (
        <div
          role="tooltip"
          aria-label="Compatibility filter details"
          data-testid="compatibility-hover-card"
          onMouseEnter={handleMouseEnter}
          onMouseLeave={handleMouseLeave}
          onClick={(e) => {
            e.stopPropagation();
            e.preventDefault();
          }}
          onMouseDown={(e) => {
            e.stopPropagation();
          }}
          className={`absolute right-0 z-50 w-80 sm:w-96 max-w-[calc(100vw-2.5rem)] p-3 rounded-xl bg-white dark:bg-zinc-900 border border-zinc-200 dark:border-zinc-800 shadow-xl transition-all duration-150 text-left ${
            placement === "top" ? "bottom-full mb-2.5" : "top-full mt-2.5"
          }`}
        >
          {/* Arrow */}
          <div
            className={`absolute right-4 w-2.5 h-2.5 rotate-45 bg-white dark:bg-zinc-900 ${
              placement === "top"
                ? "-bottom-[6px] border-r border-b border-zinc-200 dark:border-zinc-800"
                : "-top-[6px] border-l border-t border-zinc-200 dark:border-zinc-800"
            }`}
          />

          {/* Header */}
          <div className="flex items-center justify-between pb-2 mb-2 border-b border-zinc-100 dark:border-zinc-800">
            <span className="font-semibold text-[10px] uppercase tracking-wider text-zinc-500 dark:text-zinc-400 select-none">
              Compatibility Filter
            </span>
            <button
              type="button"
              onClick={handleCopy}
              aria-label="Copy filter"
              title="Copy filter"
              data-testid="copy-filter-button"
              className="p-1 rounded-md text-zinc-500 hover:text-zinc-800 dark:text-zinc-400 dark:hover:text-zinc-100 hover:bg-zinc-100 dark:hover:bg-zinc-800 transition-colors cursor-pointer inline-flex items-center justify-center shrink-0"
            >
              {copied ? (
                <Check size={13} className="text-emerald-600 dark:text-emerald-400" />
              ) : (
                <Copy size={13} />
              )}
            </button>
          </div>

          {/* Text box with horizontal scrollbar if needed to avoid text wrapping */}
          <div className="overflow-x-auto max-h-48 rounded-lg bg-zinc-50 dark:bg-zinc-950 p-2.5 border border-zinc-200 dark:border-zinc-800/80">
            <pre className="font-mono text-[11px] leading-relaxed text-zinc-800 dark:text-zinc-200 whitespace-pre select-text">
              {formattedFilter}
            </pre>
          </div>
        </div>
      )}
    </span>
  );
}
