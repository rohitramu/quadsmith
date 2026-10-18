import { useState, useMemo, useRef } from "react";
import { Copy, Check } from "lucide-react";
import { formatCelFilterForDisplay } from "../lib/batteryFilter";

export interface CopyFilterButtonProps {
  filter?: string | null;
  className?: string;
  ariaLabel?: string;
}

/**
 * An icon-only button that copies a CEL filter string to the clipboard.
 * Hovering over the button reveals a floating hint showing the complete multiline filter string.
 */
export function CopyFilterButton({
  filter,
  className = "",
  ariaLabel = "Copy filter",
}: CopyFilterButtonProps) {
  const [copied, setCopied] = useState(false);
  const [isHovered, setIsHovered] = useState(false);
  const [placement, setPlacement] = useState<"top" | "bottom">("top");
  const buttonRef = useRef<HTMLButtonElement>(null);

  const formattedFilter = useMemo(() => formatCelFilterForDisplay(filter), [filter]);

  if (!filter || !formattedFilter) return null;

  const handleMouseEnter = () => {
    if (buttonRef.current) {
      const rect = buttonRef.current.getBoundingClientRect();
      if (rect.top < 130) {
        setPlacement("bottom");
      } else {
        setPlacement("top");
      }
    }
    setIsHovered(true);
  };

  const handleMouseLeave = () => {
    setIsHovered(false);
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
    <div
      className="relative inline-flex items-center"
      onMouseEnter={handleMouseEnter}
      onMouseLeave={handleMouseLeave}
      onFocus={handleMouseEnter}
      onBlur={handleMouseLeave}
    >
      <button
        ref={buttonRef}
        type="button"
        onClick={handleCopy}
        aria-label={ariaLabel}
        data-testid="copy-filter-button"
        title={formattedFilter}
        className={`p-1 rounded-md text-zinc-400 hover:text-zinc-100 hover:bg-zinc-700/50 transition-colors cursor-pointer inline-flex items-center justify-center shrink-0 ${className}`}
      >
        {copied ? <Check size={13} className="text-emerald-400" /> : <Copy size={13} />}
      </button>

      {/* Floating Hover Hint */}
      {isHovered && (
        <div
          role="tooltip"
          className={`absolute left-0 z-50 pointer-events-none px-3 py-2 rounded-lg text-[11px] font-mono whitespace-pre-wrap leading-relaxed bg-zinc-950 text-zinc-100 border border-zinc-800 shadow-xl max-w-[calc(100vw-3rem)] sm:max-w-md w-max break-words select-none transition-all duration-150 ${
            placement === "top" ? "bottom-full mb-2" : "top-full mt-2"
          }`}
        >
          {formattedFilter}
          <div
            className={`absolute left-2.5 border-4 border-transparent ${
              placement === "top"
                ? "top-full -mt-[1px] border-t-zinc-950"
                : "bottom-full -mb-[1px] border-b-zinc-950"
            }`}
          />
        </div>
      )}
    </div>
  );
}
