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
 * Supports both desktop cursor hover and mobile tap-to-toggle with outside dismissal.
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
  const containerRef = useRef<HTMLSpanElement>(null);
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

  // Outside click and Escape key dismissal
  useEffect(() => {
    function handlePointerDown(e: MouseEvent) {
      if (containerRef.current && !containerRef.current.contains(e.target as Node)) {
        setIsOpen(false);
      }
    }

    function handleKeyDown(e: KeyboardEvent) {
      if (e.key === "Escape") {
        setIsOpen(false);
      }
    }

    if (isOpen) {
      document.addEventListener("mousedown", handlePointerDown);
      document.addEventListener("keydown", handleKeyDown);
    }
    return () => {
      document.removeEventListener("mousedown", handlePointerDown);
      document.removeEventListener("keydown", handleKeyDown);
    };
  }, [isOpen]);

  if (!filter || !formattedFilter) {
    return <span className={className}>{children}</span>;
  }

  const updatePlacement = () => {
    if (triggerRef.current) {
      const rect = triggerRef.current.getBoundingClientRect();
      if (rect.top < 220) {
        setPlacement("bottom");
      } else {
        setPlacement("top");
      }
    }
  };

  const handleMouseEnter = () => {
    if (timerRef.current) {
      clearTimeout(timerRef.current);
      timerRef.current = null;
    }
    updatePlacement();
    setIsOpen(true);
  };

  const handleMouseLeave = () => {
    timerRef.current = setTimeout(() => {
      setIsOpen(false);
    }, 150);
  };

  const handleToggleClick = (e: React.MouseEvent) => {
    e.stopPropagation();
    updatePlacement();
    setIsOpen((prev) => !prev);
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
    <span ref={containerRef} className="relative inline-flex items-center">
      <span
        ref={triggerRef}
        onClick={handleToggleClick}
        onMouseEnter={handleMouseEnter}
        onMouseLeave={handleMouseLeave}
        className={`underline decoration-dotted underline-offset-2 decoration-zinc-500 hover:decoration-zinc-300 transition-colors cursor-help ${className}`}
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
          className={`absolute right-0 z-50 w-80 max-w-[calc(100vw-2rem)] p-3 rounded-xl bg-zinc-900 border border-zinc-800 shadow-xl transition-all duration-150 text-left ${
            placement === "top" ? "bottom-full mb-2.5" : "top-full mt-2.5"
          }`}
        >
          {/* Arrow */}
          <div
            className={`absolute right-4 w-2.5 h-2.5 rotate-45 bg-zinc-900 ${
              placement === "top"
                ? "bottom-[-6px] border-r border-b border-zinc-800"
                : "top-[-6px] border-l border-t border-zinc-800"
            }`}
          />

          <div className="flex items-center justify-between gap-2 mb-1.5">
            <span className="text-xs font-semibold text-zinc-100 flex items-center gap-1.5">
              <span>Compatibility Filter</span>
            </span>
            <button
              type="button"
              data-testid="copy-filter-button"
              onClick={handleCopy}
              title="Copy filter to clipboard"
              className="p-1 rounded-md hover:bg-zinc-800 text-zinc-500 hover:text-zinc-100 transition-colors cursor-pointer"
            >
              {copied ? <Check size={13} className="text-emerald-500" /> : <Copy size={13} />}
            </button>
          </div>

          <div className="bg-zinc-950 p-2 rounded-lg border border-zinc-800/80 overflow-x-auto">
            <pre className="text-[11px] font-mono text-zinc-200 whitespace-pre leading-relaxed selection:bg-blue-500/20">
              {formattedFilter}
            </pre>
          </div>
        </div>
      )}
    </span>
  );
}
