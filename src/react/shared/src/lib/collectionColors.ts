export interface CollectionColorDef {
  id: string;
  name: string;
  shortName?: string;
  colorName: string;
  hex: string;
  badgeClass: string;
  bgClass: string;
  textClass: string;
  borderClass: string;
  trimClass: string;
  dotClass: string;
  borderTopClass: string;
  borderLeftClass: string;
  iconBgClass: string;
  hoverBorderClass: string;
}

export type ColorStylePreset = Omit<CollectionColorDef, "id" | "name">;

/**
 * Authoritative UI color palette containing 32 distinct color presets,
 * each with comprehensive light mode and dark mode variants.
 */
export const COLOR_PALETTE: ColorStylePreset[] = [
  // 1. Red
  {
    colorName: "red",
    hex: "#ef4444",
    badgeClass:
      "bg-red-100 text-red-800 border-red-300 dark:bg-red-950/60 dark:text-red-300 dark:border-red-800/80",
    bgClass: "bg-red-100 dark:bg-red-950/60",
    textClass: "text-red-600 dark:text-red-400",
    borderClass: "border-red-300 dark:border-red-800/80",
    trimClass: "bg-red-500",
    dotClass: "bg-red-500",
    borderTopClass: "border-t-red-500",
    borderLeftClass: "border-l-red-500",
    iconBgClass: "bg-red-100 text-red-700 dark:bg-red-950/60 dark:text-red-300",
    hoverBorderClass: "hover:border-red-500",
  },
  // 2. Orange
  {
    colorName: "orange",
    hex: "#f97316",
    badgeClass:
      "bg-orange-100 text-orange-800 border-orange-300 dark:bg-orange-950/60 dark:text-orange-300 dark:border-orange-800/80",
    bgClass: "bg-orange-100 dark:bg-orange-950/60",
    textClass: "text-orange-600 dark:text-orange-400",
    borderClass: "border-orange-300 dark:border-orange-800/80",
    trimClass: "bg-orange-500",
    dotClass: "bg-orange-500",
    borderTopClass: "border-t-orange-500",
    borderLeftClass: "border-l-orange-500",
    iconBgClass: "bg-orange-100 text-orange-700 dark:bg-orange-950/60 dark:text-orange-300",
    hoverBorderClass: "hover:border-orange-500",
  },
  // 3. Amber
  {
    colorName: "amber",
    hex: "#f59e0b",
    badgeClass:
      "bg-amber-100 text-amber-800 border-amber-300 dark:bg-amber-950/60 dark:text-amber-300 dark:border-amber-800/80",
    bgClass: "bg-amber-100 dark:bg-amber-950/60",
    textClass: "text-amber-600 dark:text-amber-400",
    borderClass: "border-amber-300 dark:border-amber-800/80",
    trimClass: "bg-amber-500",
    dotClass: "bg-amber-500",
    borderTopClass: "border-t-amber-500",
    borderLeftClass: "border-l-amber-500",
    iconBgClass: "bg-amber-100 text-amber-700 dark:bg-amber-950/60 dark:text-amber-300",
    hoverBorderClass: "hover:border-amber-500",
  },
  // 4. Yellow
  {
    colorName: "yellow",
    hex: "#eab308",
    badgeClass:
      "bg-yellow-100 text-yellow-800 border-yellow-300 dark:bg-yellow-950/60 dark:text-yellow-300 dark:border-yellow-800/80",
    bgClass: "bg-yellow-100 dark:bg-yellow-950/60",
    textClass: "text-yellow-600 dark:text-yellow-400",
    borderClass: "border-yellow-300 dark:border-yellow-800/80",
    trimClass: "bg-yellow-500",
    dotClass: "bg-yellow-500",
    borderTopClass: "border-t-yellow-500",
    borderLeftClass: "border-l-yellow-500",
    iconBgClass: "bg-yellow-100 text-yellow-700 dark:bg-yellow-950/60 dark:text-yellow-300",
    hoverBorderClass: "hover:border-yellow-500",
  },
  // 5. Lime
  {
    colorName: "lime",
    hex: "#84cc16",
    badgeClass:
      "bg-lime-100 text-lime-800 border-lime-300 dark:bg-lime-950/60 dark:text-lime-300 dark:border-lime-800/80",
    bgClass: "bg-lime-100 dark:bg-lime-950/60",
    textClass: "text-lime-600 dark:text-lime-400",
    borderClass: "border-lime-300 dark:border-lime-800/80",
    trimClass: "bg-lime-500",
    dotClass: "bg-lime-500",
    borderTopClass: "border-t-lime-500",
    borderLeftClass: "border-l-lime-500",
    iconBgClass: "bg-lime-100 text-lime-700 dark:bg-lime-950/60 dark:text-lime-300",
    hoverBorderClass: "hover:border-lime-500",
  },
  // 6. Green
  {
    colorName: "green",
    hex: "#22c55e",
    badgeClass:
      "bg-green-100 text-green-800 border-green-300 dark:bg-green-950/60 dark:text-green-300 dark:border-green-800/80",
    bgClass: "bg-green-100 dark:bg-green-950/60",
    textClass: "text-green-600 dark:text-green-400",
    borderClass: "border-green-300 dark:border-green-800/80",
    trimClass: "bg-green-500",
    dotClass: "bg-green-500",
    borderTopClass: "border-t-green-500",
    borderLeftClass: "border-l-green-500",
    iconBgClass: "bg-green-100 text-green-700 dark:bg-green-950/60 dark:text-green-300",
    hoverBorderClass: "hover:border-green-500",
  },
  // 7. Emerald
  {
    colorName: "emerald",
    hex: "#10b981",
    badgeClass:
      "bg-emerald-100 text-emerald-800 border-emerald-300 dark:bg-emerald-950/60 dark:text-emerald-300 dark:border-emerald-800/80",
    bgClass: "bg-emerald-100 dark:bg-emerald-950/60",
    textClass: "text-emerald-600 dark:text-emerald-400",
    borderClass: "border-emerald-300 dark:border-emerald-800/80",
    trimClass: "bg-emerald-500",
    dotClass: "bg-emerald-500",
    borderTopClass: "border-t-emerald-500",
    borderLeftClass: "border-l-emerald-500",
    iconBgClass: "bg-emerald-100 text-emerald-700 dark:bg-emerald-950/60 dark:text-emerald-300",
    hoverBorderClass: "hover:border-emerald-500",
  },
  // 8. Teal
  {
    colorName: "teal",
    hex: "#14b8a6",
    badgeClass:
      "bg-teal-100 text-teal-800 border-teal-300 dark:bg-teal-950/60 dark:text-teal-300 dark:border-teal-800/80",
    bgClass: "bg-teal-100 dark:bg-teal-950/60",
    textClass: "text-teal-600 dark:text-teal-400",
    borderClass: "border-teal-300 dark:border-teal-800/80",
    trimClass: "bg-teal-500",
    dotClass: "bg-teal-500",
    borderTopClass: "border-t-teal-500",
    borderLeftClass: "border-l-teal-500",
    iconBgClass: "bg-teal-100 text-teal-700 dark:bg-teal-950/60 dark:text-teal-300",
    hoverBorderClass: "hover:border-teal-500",
  },
  // 9. Cyan
  {
    colorName: "cyan",
    hex: "#06b6d4",
    badgeClass:
      "bg-cyan-100 text-cyan-800 border-cyan-300 dark:bg-cyan-950/60 dark:text-cyan-300 dark:border-cyan-800/80",
    bgClass: "bg-cyan-100 dark:bg-cyan-950/60",
    textClass: "text-cyan-600 dark:text-cyan-400",
    borderClass: "border-cyan-300 dark:border-cyan-800/80",
    trimClass: "bg-cyan-500",
    dotClass: "bg-cyan-500",
    borderTopClass: "border-t-cyan-500",
    borderLeftClass: "border-l-cyan-500",
    iconBgClass: "bg-cyan-100 text-cyan-700 dark:bg-cyan-950/60 dark:text-cyan-300",
    hoverBorderClass: "hover:border-cyan-500",
  },
  // 10. Sky
  {
    colorName: "sky",
    hex: "#0ea5e9",
    badgeClass:
      "bg-sky-100 text-sky-800 border-sky-300 dark:bg-sky-950/60 dark:text-sky-300 dark:border-sky-800/80",
    bgClass: "bg-sky-100 dark:bg-sky-950/60",
    textClass: "text-sky-600 dark:text-sky-400",
    borderClass: "border-sky-300 dark:border-sky-800/80",
    trimClass: "bg-sky-500",
    dotClass: "bg-sky-500",
    borderTopClass: "border-t-sky-500",
    borderLeftClass: "border-l-sky-500",
    iconBgClass: "bg-sky-100 text-sky-700 dark:bg-sky-950/60 dark:text-sky-300",
    hoverBorderClass: "hover:border-sky-500",
  },
  // 11. Blue
  {
    colorName: "blue",
    hex: "#3b82f6",
    badgeClass:
      "bg-blue-100 text-blue-800 border-blue-300 dark:bg-blue-950/60 dark:text-blue-300 dark:border-blue-800/80",
    bgClass: "bg-blue-100 dark:bg-blue-950/60",
    textClass: "text-blue-600 dark:text-blue-400",
    borderClass: "border-blue-300 dark:border-blue-800/80",
    trimClass: "bg-blue-500",
    dotClass: "bg-blue-500",
    borderTopClass: "border-t-blue-500",
    borderLeftClass: "border-l-blue-500",
    iconBgClass: "bg-blue-100 text-blue-700 dark:bg-blue-950/60 dark:text-blue-300",
    hoverBorderClass: "hover:border-blue-500",
  },
  // 12. Indigo
  {
    colorName: "indigo",
    hex: "#6366f1",
    badgeClass:
      "bg-indigo-100 text-indigo-800 border-indigo-300 dark:bg-indigo-950/60 dark:text-indigo-300 dark:border-indigo-800/80",
    bgClass: "bg-indigo-100 dark:bg-indigo-950/60",
    textClass: "text-indigo-600 dark:text-indigo-400",
    borderClass: "border-indigo-300 dark:border-indigo-800/80",
    trimClass: "bg-indigo-500",
    dotClass: "bg-indigo-500",
    borderTopClass: "border-t-indigo-500",
    borderLeftClass: "border-l-indigo-500",
    iconBgClass: "bg-indigo-100 text-indigo-700 dark:bg-indigo-950/60 dark:text-indigo-300",
    hoverBorderClass: "hover:border-indigo-500",
  },
  // 13. Violet
  {
    colorName: "violet",
    hex: "#8b5cf6",
    badgeClass:
      "bg-violet-100 text-violet-800 border-violet-300 dark:bg-violet-950/60 dark:text-violet-300 dark:border-violet-800/80",
    bgClass: "bg-violet-100 dark:bg-violet-950/60",
    textClass: "text-violet-600 dark:text-violet-400",
    borderClass: "border-violet-300 dark:border-violet-800/80",
    trimClass: "bg-violet-500",
    dotClass: "bg-violet-500",
    borderTopClass: "border-t-violet-500",
    borderLeftClass: "border-l-violet-500",
    iconBgClass: "bg-violet-100 text-violet-700 dark:bg-violet-950/60 dark:text-violet-300",
    hoverBorderClass: "hover:border-violet-500",
  },
  // 14. Purple
  {
    colorName: "purple",
    hex: "#a855f7",
    badgeClass:
      "bg-purple-100 text-purple-800 border-purple-300 dark:bg-purple-950/60 dark:text-purple-300 dark:border-purple-800/80",
    bgClass: "bg-purple-100 dark:bg-purple-950/60",
    textClass: "text-purple-600 dark:text-purple-400",
    borderClass: "border-purple-300 dark:border-purple-800/80",
    trimClass: "bg-purple-500",
    dotClass: "bg-purple-500",
    borderTopClass: "border-t-purple-500",
    borderLeftClass: "border-l-purple-500",
    iconBgClass: "bg-purple-100 text-purple-700 dark:bg-purple-950/60 dark:text-purple-300",
    hoverBorderClass: "hover:border-purple-500",
  },
  // 15. Fuchsia
  {
    colorName: "fuchsia",
    hex: "#d946ef",
    badgeClass:
      "bg-fuchsia-100 text-fuchsia-800 border-fuchsia-300 dark:bg-fuchsia-950/60 dark:text-fuchsia-300 dark:border-fuchsia-800/80",
    bgClass: "bg-fuchsia-100 dark:bg-fuchsia-950/60",
    textClass: "text-fuchsia-600 dark:text-fuchsia-400",
    borderClass: "border-fuchsia-300 dark:border-fuchsia-800/80",
    trimClass: "bg-fuchsia-500",
    dotClass: "bg-fuchsia-500",
    borderTopClass: "border-t-fuchsia-500",
    borderLeftClass: "border-l-fuchsia-500",
    iconBgClass: "bg-fuchsia-100 text-fuchsia-700 dark:bg-fuchsia-950/60 dark:text-fuchsia-300",
    hoverBorderClass: "hover:border-fuchsia-500",
  },
  // 16. Pink
  {
    colorName: "pink",
    hex: "#ec4899",
    badgeClass:
      "bg-pink-100 text-pink-800 border-pink-300 dark:bg-pink-950/60 dark:text-pink-300 dark:border-pink-800/80",
    bgClass: "bg-pink-100 dark:bg-pink-950/60",
    textClass: "text-pink-600 dark:text-pink-400",
    borderClass: "border-pink-300 dark:border-pink-800/80",
    trimClass: "bg-pink-500",
    dotClass: "bg-pink-500",
    borderTopClass: "border-t-pink-500",
    borderLeftClass: "border-l-pink-500",
    iconBgClass: "bg-pink-100 text-pink-700 dark:bg-pink-950/60 dark:text-pink-300",
    hoverBorderClass: "hover:border-pink-500",
  },
  // 17. Rose
  {
    colorName: "rose",
    hex: "#f43f5e",
    badgeClass:
      "bg-rose-100 text-rose-800 border-rose-300 dark:bg-rose-950/60 dark:text-rose-300 dark:border-rose-800/80",
    bgClass: "bg-rose-100 dark:bg-rose-950/60",
    textClass: "text-rose-600 dark:text-rose-400",
    borderClass: "border-rose-300 dark:border-rose-800/80",
    trimClass: "bg-rose-500",
    dotClass: "bg-rose-500",
    borderTopClass: "border-t-rose-500",
    borderLeftClass: "border-l-rose-500",
    iconBgClass: "bg-rose-100 text-rose-700 dark:bg-rose-950/60 dark:text-rose-300",
    hoverBorderClass: "hover:border-rose-500",
  },
  // 18. Slate
  {
    colorName: "slate",
    hex: "#64748b",
    badgeClass:
      "bg-slate-100 text-slate-800 border-slate-300 dark:bg-slate-800 dark:text-slate-300 dark:border-slate-700",
    bgClass: "bg-slate-100 dark:bg-slate-800",
    textClass: "text-slate-600 dark:text-slate-400",
    borderClass: "border-slate-300 dark:border-slate-700",
    trimClass: "bg-slate-500",
    dotClass: "bg-slate-500",
    borderTopClass: "border-t-slate-500",
    borderLeftClass: "border-l-slate-500",
    iconBgClass: "bg-slate-100 text-slate-700 dark:bg-slate-800 dark:text-slate-300",
    hoverBorderClass: "hover:border-slate-500",
  },
  // 19. Zinc
  {
    colorName: "zinc",
    hex: "#71717a",
    badgeClass:
      "bg-zinc-100 text-zinc-800 border-zinc-300 dark:bg-zinc-800 dark:text-zinc-300 dark:border-zinc-700",
    bgClass: "bg-zinc-100 dark:bg-zinc-800",
    textClass: "text-zinc-600 dark:text-zinc-400",
    borderClass: "border-zinc-300 dark:border-zinc-700",
    trimClass: "bg-zinc-500",
    dotClass: "bg-zinc-500",
    borderTopClass: "border-t-zinc-500",
    borderLeftClass: "border-l-zinc-500",
    iconBgClass: "bg-zinc-100 text-zinc-700 dark:bg-zinc-800 dark:text-zinc-300",
    hoverBorderClass: "hover:border-zinc-500",
  },
  // 20. Neutral
  {
    colorName: "neutral",
    hex: "#737373",
    badgeClass:
      "bg-neutral-100 text-neutral-800 border-neutral-300 dark:bg-neutral-800 dark:text-neutral-300 dark:border-neutral-700",
    bgClass: "bg-neutral-100 dark:bg-neutral-800",
    textClass: "text-neutral-600 dark:text-neutral-400",
    borderClass: "border-neutral-300 dark:border-neutral-700",
    trimClass: "bg-neutral-500",
    dotClass: "bg-neutral-500",
    borderTopClass: "border-t-neutral-500",
    borderLeftClass: "border-l-neutral-500",
    iconBgClass: "bg-neutral-100 text-neutral-700 dark:bg-neutral-800 dark:text-neutral-300",
    hoverBorderClass: "hover:border-neutral-500",
  },
  // 21. Stone
  {
    colorName: "stone",
    hex: "#78716c",
    badgeClass:
      "bg-stone-100 text-stone-800 border-stone-300 dark:bg-stone-800 dark:text-stone-300 dark:border-stone-700",
    bgClass: "bg-stone-100 dark:bg-stone-800",
    textClass: "text-stone-600 dark:text-stone-400",
    borderClass: "border-stone-300 dark:border-stone-700",
    trimClass: "bg-stone-500",
    dotClass: "bg-stone-500",
    borderTopClass: "border-t-stone-500",
    borderLeftClass: "border-l-stone-500",
    iconBgClass: "bg-stone-100 text-stone-700 dark:bg-stone-800 dark:text-stone-300",
    hoverBorderClass: "hover:border-stone-500",
  },
  // 22. Fiery Orange
  {
    colorName: "fiery-orange",
    hex: "#FF6D00",
    badgeClass:
      "bg-orange-100 text-orange-900 border-orange-300 dark:bg-orange-950/60 dark:text-orange-300 dark:border-orange-800/80",
    bgClass: "bg-orange-100 dark:bg-orange-950/60",
    textClass: "text-[#FF6D00] dark:text-[#FF6D00]",
    borderClass: "border-orange-300 dark:border-orange-800/80",
    trimClass: "bg-[#FF6D00]",
    dotClass: "bg-[#FF6D00]",
    borderTopClass: "border-t-[#FF6D00]",
    borderLeftClass: "border-l-[#FF6D00]",
    iconBgClass: "bg-orange-100 text-[#FF6D00] dark:bg-orange-950/60 dark:text-[#FF6D00]",
    hoverBorderClass: "hover:border-[#FF6D00]",
  },
  // 23. Ruby (Dark Rose)
  {
    colorName: "ruby",
    hex: "#e11d48",
    badgeClass:
      "bg-rose-50 text-rose-900 border-rose-300 dark:bg-rose-950/80 dark:text-rose-200 dark:border-rose-700",
    bgClass: "bg-rose-50 dark:bg-rose-950/80",
    textClass: "text-rose-700 dark:text-rose-300",
    borderClass: "border-rose-300 dark:border-rose-700",
    trimClass: "bg-rose-600",
    dotClass: "bg-rose-600",
    borderTopClass: "border-t-rose-600",
    borderLeftClass: "border-l-rose-600",
    iconBgClass: "bg-rose-100 text-rose-800 dark:bg-rose-950/80 dark:text-rose-200",
    hoverBorderClass: "hover:border-rose-600",
  },
  // 24. Gold (Dark Amber)
  {
    colorName: "gold",
    hex: "#d97706",
    badgeClass:
      "bg-amber-50 text-amber-900 border-amber-300 dark:bg-amber-950/80 dark:text-amber-200 dark:border-amber-700",
    bgClass: "bg-amber-50 dark:bg-amber-950/80",
    textClass: "text-amber-700 dark:text-amber-300",
    borderClass: "border-amber-300 dark:border-amber-700",
    trimClass: "bg-amber-600",
    dotClass: "bg-amber-600",
    borderTopClass: "border-t-amber-600",
    borderLeftClass: "border-l-amber-600",
    iconBgClass: "bg-amber-100 text-amber-800 dark:bg-amber-950/80 dark:text-amber-200",
    hoverBorderClass: "hover:border-amber-600",
  },
  // 25. Chartreuse (Dark Lime)
  {
    colorName: "chartreuse",
    hex: "#65a30d",
    badgeClass:
      "bg-lime-50 text-lime-900 border-lime-300 dark:bg-lime-950/80 dark:text-lime-200 dark:border-lime-700",
    bgClass: "bg-lime-50 dark:bg-lime-950/80",
    textClass: "text-lime-700 dark:text-lime-300",
    borderClass: "border-lime-300 dark:border-lime-700",
    trimClass: "bg-lime-600",
    dotClass: "bg-lime-600",
    borderTopClass: "border-t-lime-600",
    borderLeftClass: "border-l-lime-600",
    iconBgClass: "bg-lime-100 text-lime-800 dark:bg-lime-950/80 dark:text-lime-200",
    hoverBorderClass: "hover:border-lime-600",
  },
  // 26. Mint (Dark Emerald)
  {
    colorName: "mint",
    hex: "#059669",
    badgeClass:
      "bg-emerald-50 text-emerald-900 border-emerald-300 dark:bg-emerald-950/80 dark:text-emerald-200 dark:border-emerald-700",
    bgClass: "bg-emerald-50 dark:bg-emerald-950/80",
    textClass: "text-emerald-700 dark:text-emerald-300",
    borderClass: "border-emerald-300 dark:border-emerald-700",
    trimClass: "bg-emerald-600",
    dotClass: "bg-emerald-600",
    borderTopClass: "border-t-emerald-600",
    borderLeftClass: "border-l-emerald-600",
    iconBgClass: "bg-emerald-100 text-emerald-800 dark:bg-emerald-950/80 dark:text-emerald-200",
    hoverBorderClass: "hover:border-emerald-600",
  },
  // 27. Cerulean (Dark Sky)
  {
    colorName: "cerulean",
    hex: "#0284c7",
    badgeClass:
      "bg-sky-50 text-sky-900 border-sky-300 dark:bg-sky-950/80 dark:text-sky-200 dark:border-sky-700",
    bgClass: "bg-sky-50 dark:bg-sky-950/80",
    textClass: "text-sky-700 dark:text-sky-300",
    borderClass: "border-sky-300 dark:border-sky-700",
    trimClass: "bg-sky-600",
    dotClass: "bg-sky-600",
    borderTopClass: "border-t-sky-600",
    borderLeftClass: "border-l-sky-600",
    iconBgClass: "bg-sky-100 text-sky-800 dark:bg-sky-950/80 dark:text-sky-200",
    hoverBorderClass: "hover:border-sky-600",
  },
  // 28. Royal Blue
  {
    colorName: "royal-blue",
    hex: "#2563eb",
    badgeClass:
      "bg-blue-50 text-blue-900 border-blue-300 dark:bg-blue-950/80 dark:text-blue-200 dark:border-blue-700",
    bgClass: "bg-blue-50 dark:bg-blue-950/80",
    textClass: "text-blue-700 dark:text-blue-300",
    borderClass: "border-blue-300 dark:border-blue-700",
    trimClass: "bg-blue-600",
    dotClass: "bg-blue-600",
    borderTopClass: "border-t-blue-600",
    borderLeftClass: "border-l-blue-600",
    iconBgClass: "bg-blue-100 text-blue-800 dark:bg-blue-950/80 dark:text-blue-200",
    hoverBorderClass: "hover:border-blue-600",
  },
  // 29. Plum (Dark Purple)
  {
    colorName: "plum",
    hex: "#9333ea",
    badgeClass:
      "bg-purple-50 text-purple-900 border-purple-300 dark:bg-purple-950/80 dark:text-purple-200 dark:border-purple-700",
    bgClass: "bg-purple-50 dark:bg-purple-950/80",
    textClass: "text-purple-700 dark:text-purple-300",
    borderClass: "border-purple-300 dark:border-purple-700",
    trimClass: "bg-purple-600",
    dotClass: "bg-purple-600",
    borderTopClass: "border-t-purple-600",
    borderLeftClass: "border-l-purple-600",
    iconBgClass: "bg-purple-100 text-purple-800 dark:bg-purple-950/80 dark:text-purple-200",
    hoverBorderClass: "hover:border-purple-600",
  },
  // 30. Magenta (Dark Fuchsia)
  {
    colorName: "magenta",
    hex: "#c026d3",
    badgeClass:
      "bg-fuchsia-50 text-fuchsia-900 border-fuchsia-300 dark:bg-fuchsia-950/80 dark:text-fuchsia-200 dark:border-fuchsia-700",
    bgClass: "bg-fuchsia-50 dark:bg-fuchsia-950/80",
    textClass: "text-fuchsia-700 dark:text-fuchsia-300",
    borderClass: "border-fuchsia-300 dark:border-fuchsia-700",
    trimClass: "bg-fuchsia-600",
    dotClass: "bg-fuchsia-600",
    borderTopClass: "border-t-fuchsia-600",
    borderLeftClass: "border-l-fuchsia-600",
    iconBgClass: "bg-fuchsia-100 text-fuchsia-800 dark:bg-fuchsia-950/80 dark:text-fuchsia-200",
    hoverBorderClass: "hover:border-fuchsia-600",
  },
  // 31. Coral (Dark Orange)
  {
    colorName: "coral",
    hex: "#ea580c",
    badgeClass:
      "bg-orange-50 text-orange-900 border-orange-300 dark:bg-orange-950/80 dark:text-orange-200 dark:border-orange-700",
    bgClass: "bg-orange-50 dark:bg-orange-950/80",
    textClass: "text-orange-700 dark:text-orange-300",
    borderClass: "border-orange-300 dark:border-orange-700",
    trimClass: "bg-orange-600",
    dotClass: "bg-orange-600",
    borderTopClass: "border-t-orange-600",
    borderLeftClass: "border-l-orange-600",
    iconBgClass: "bg-orange-100 text-orange-800 dark:bg-orange-950/80 dark:text-orange-200",
    hoverBorderClass: "hover:border-orange-600",
  },
  // 32. Crimson (Dark Red)
  {
    colorName: "crimson",
    hex: "#dc2626",
    badgeClass:
      "bg-red-50 text-red-900 border-red-300 dark:bg-red-950/80 dark:text-red-200 dark:border-red-700",
    bgClass: "bg-red-50 dark:bg-red-950/80",
    textClass: "text-red-700 dark:text-red-300",
    borderClass: "border-red-300 dark:border-red-700",
    trimClass: "bg-red-600",
    dotClass: "bg-red-600",
    borderTopClass: "border-t-red-600",
    borderLeftClass: "border-l-red-600",
    iconBgClass: "bg-red-100 text-red-800 dark:bg-red-950/80 dark:text-red-200",
    hoverBorderClass: "hover:border-red-600",
  },
];

export const COLOR_PRESETS_BY_HEX: Record<string, ColorStylePreset> = Object.fromEntries(
  COLOR_PALETTE.map((preset) => [preset.hex.toLowerCase(), preset]),
);

export const COLOR_PRESETS_BY_NAME: Record<string, ColorStylePreset> = Object.fromEntries(
  COLOR_PALETTE.map((preset) => [preset.colorName.toLowerCase(), preset]),
);

export const DEFAULT_COLLECTION_COLOR: CollectionColorDef = {
  id: "default",
  name: "Collection",
  shortName: "Collection",
  ...COLOR_PRESETS_BY_HEX["#64748b"],
};

export const COLLECTION_SHORT_NAMES: Record<string, string> = {
  "flight-controllers": "FC",
  "flight controllers": "FC",
  "flight-controller": "FC",
  "flight controller": "FC",
  fc: "FC",
  fcs: "FC",
  "electronic-speed-controllers": "ESC",
  "electronic speed controllers": "ESC",
  "electronic-speed-controller": "ESC",
  "electronic speed controller": "ESC",
  esc: "ESC",
  escs: "ESC",
  "video-transmitters": "VTX",
  "video transmitters": "VTX",
  "video-transmitter": "VTX",
  "video transmitter": "VTX",
  vtx: "VTX",
  vtxs: "VTX",
  propellers: "Props",
  propeller: "Props",
  props: "Props",
  prop: "Props",
  receivers: "RX",
  receiver: "RX",
  rx: "RX",
  rxs: "RX",
  transmitters: "TX",
  transmitter: "TX",
  tx: "TX",
  txs: "TX",
  "fc-firmware": "FC FW",
  "flight-controller-firmware": "FC FW",
  "esc-firmware": "ESC FW",
  "electronic-speed-controller-firmware": "ESC FW",
  "vtx-firmware": "VTX FW",
  "video-transmitter-firmware": "VTX FW",
  "radio-firmware": "Radio FW",
  "software-tools": "Tools",
  "software-tool": "Tools",
  goggles: "Goggles",
  goggle: "Goggles",
  chargers: "Chargers",
  charger: "Charger",
  "field-tools": "Tools",
  "field-tool": "Tools",
};

/**
 * Resolves a CollectionColorDef by hex code or color name string.
 * Falls back to the default slate color if the code is empty, null, or unrecognized.
 */
export function getCollectionColorByCode(
  hexOrColorCode?: string | null,
  id: string = "default",
  name: string = "Collection",
): CollectionColorDef {
  let normalized = (hexOrColorCode ?? "").trim().toLowerCase();
  if (normalized && !normalized.startsWith("#") && !COLOR_PRESETS_BY_NAME[normalized]) {
    normalized = "#" + normalized;
  }
  const preset =
    COLOR_PRESETS_BY_HEX[normalized] ||
    COLOR_PRESETS_BY_NAME[normalized] ||
    COLOR_PRESETS_BY_HEX["#64748b"];

  const shortName = COLLECTION_SHORT_NAMES[id] || COLLECTION_SHORT_NAMES[name] || name;
  return {
    id,
    name,
    shortName,
    colorName: preset.colorName,
    hex: preset.hex,
    badgeClass: preset.badgeClass,
    bgClass: preset.bgClass,
    textClass: preset.textClass,
    borderClass: preset.borderClass,
    trimClass: preset.trimClass,
    dotClass: preset.dotClass,
    borderTopClass: preset.borderTopClass,
    borderLeftClass: preset.borderLeftClass,
    iconBgClass: preset.iconBgClass,
    hoverBorderClass: preset.hoverBorderClass,
  };
}

/**
 * Map of canonical collection IDs to color definitions,
 * defined directly in the UI code as the single authoritative source of truth.
 */
export const COLLECTION_COLORS: Record<string, CollectionColorDef> = {
  antennas: getCollectionColorByCode("#eab308", "antennas", "Antennas"),
  batteries: getCollectionColorByCode("#84cc16", "batteries", "Batteries"),
  builds: getCollectionColorByCode("#3b82f6", "builds", "Builds"),
  cameras: getCollectionColorByCode("#06b6d4", "cameras", "Cameras"),
  "electronic-speed-controllers": getCollectionColorByCode(
    "#f97316",
    "electronic-speed-controllers",
    "Electronic Speed Controllers",
  ),
  "flight-controllers": getCollectionColorByCode(
    "#3b82f6",
    "flight-controllers",
    "Flight Controllers",
  ),
  frames: getCollectionColorByCode("#64748b", "frames", "Frames"),
  "gps-receivers": getCollectionColorByCode("#10b981", "gps-receivers", "GPS Receivers"),
  motors: getCollectionColorByCode("#ef4444", "motors", "Motors"),
  propellers: getCollectionColorByCode("#14b8a6", "propellers", "Propellers"),
  receivers: getCollectionColorByCode("#a855f7", "receivers", "Receivers"),
  "video-transmitters": getCollectionColorByCode(
    "#ec4899",
    "video-transmitters",
    "Video Transmitters",
  ),
  "fc-firmware": getCollectionColorByCode("#3b82f6", "fc-firmware", "Flight Controller Firmware"),
  "esc-firmware": getCollectionColorByCode("#10b981", "esc-firmware", "ESC Firmware"),
  "vtx-firmware": getCollectionColorByCode("#ec4899", "vtx-firmware", "Video Transmitter Firmware"),
  "radio-firmware": getCollectionColorByCode("#84cc16", "radio-firmware", "Radio Firmware"),
  "software-tools": getCollectionColorByCode("#a855f7", "software-tools", "Software Tools"),
  transmitters: getCollectionColorByCode("#0ea5e9", "transmitters", "Transmitters"),
  goggles: getCollectionColorByCode("#d946ef", "goggles", "FPV Goggles"),
  chargers: getCollectionColorByCode("#f59e0b", "chargers", "Battery Chargers"),
  "field-tools": getCollectionColorByCode("#f43f5e", "field-tools", "Field Tools"),
  "the-forge": getCollectionColorByCode("#ff6d00", "the-forge", "The Forge"),
};

/**
 * Convenience compatibility helper to retrieve color by schema or identifier.
 */
export function getCollectionColorFromSchema(
  _schema?: any,
  id: string = "default",
  name: string = "Collection",
): CollectionColorDef {
  return COLLECTION_COLORS[id] || getCollectionColorByCode(undefined, id, name);
}

/**
 * Normalizes any collection path, alias, plural, or singular identifier into a canonical key.
 */
export function normalizeCollectionKey(raw?: string | null): string {
  if (!raw) return "";
  let clean = raw.trim().toLowerCase();

  // Strip leading and trailing slashes
  clean = clean.replace(/^\/+|\/+$/g, "");

  // Special route aliases before splitting
  if (clean === "builds/new" || clean.startsWith("builds/new/")) {
    return "the-forge";
  }

  // If it's a full path, take the last segment (e.g. components/hardware/batteries -> batteries)
  if (clean.includes("/")) {
    const parts = clean.split("/");
    clean = parts[parts.length - 1];
  }

  // Convert underscores to dashes
  clean = clean.replace(/_/g, "-");

  // Alias lookup
  switch (clean) {
    case "antenna":
    case "antennas":
      return "antennas";

    case "battery":
    case "batteries":
      return "batteries";

    case "forge":
    case "the-forge":
    case "theforge":
    case "build-wizard":
    case "buildwizard":
    case "builds/new":
      return "the-forge";

    case "build":
    case "builds":
      return "builds";

    case "camera":
    case "cameras":
      return "cameras";

    case "esc":
    case "escs":
    case "electronic-speed-controller":
    case "electronic-speed-controllers":
    case "electronicspeedcontroller":
    case "electronicspeedcontrollers":
      return "electronic-speed-controllers";

    case "fc":
    case "fcs":
    case "flight-controller":
    case "flight-controllers":
    case "flightcontroller":
    case "flightcontrollers":
      return "flight-controllers";

    case "frame":
    case "frames":
      return "frames";

    case "gps":
    case "gps-receiver":
    case "gps-receivers":
    case "gpsreceiver":
    case "gpsreceivers":
      return "gps-receivers";

    case "motor":
    case "motors":
      return "motors";

    case "propeller":
    case "propellers":
    case "prop":
    case "props":
      return "propellers";

    case "rx":
    case "rxs":
    case "receiver":
    case "receivers":
      return "receivers";

    case "vtx":
    case "vtxs":
    case "video-transmitter":
    case "video-transmitters":
    case "videotransmitter":
    case "videotransmitters":
      return "video-transmitters";

    case "tx":
    case "txs":
    case "transmitter":
    case "transmitters":
    case "radio":
    case "radios":
      return "transmitters";

    case "fc-firmware":
    case "fc-firmwares":
    case "fcfirmware":
    case "flight-controller-firmware":
    case "flight-controller-firmwares":
      return "fc-firmware";

    case "esc-firmware":
    case "esc-firmwares":
    case "escfirmware":
    case "electronic-speed-controller-firmware":
    case "electronic-speed-controller-firmwares":
      return "esc-firmware";

    case "vtx-firmware":
    case "vtx-firmwares":
    case "vtxfirmware":
    case "video-transmitter-firmware":
    case "video-transmitter-firmwares":
      return "vtx-firmware";

    case "radio-firmware":
    case "radio-firmwares":
    case "radiofirmware":
    case "transmitters-firmware":
    case "tx-firmware":
      return "radio-firmware";

    case "software-tool":
    case "software-tools":
    case "softwaretool":
    case "softwaretools":
    case "configurator":
    case "configurators":
      return "software-tools";

    case "goggle":
    case "goggles":
    case "headset":
    case "headsets":
    case "fpv-goggles":
      return "goggles";

    case "charger":
    case "chargers":
    case "battery-charger":
    case "battery-chargers":
      return "chargers";

    case "field-tool":
    case "field-tools":
    case "fieldtool":
    case "fieldtools":
    case "tool":
    case "tools":
      return "field-tools";

    default:
      return clean;
  }
}

/**
 * Returns the color definition associated with a collection, or the default fallback.
 */
export function getCollectionColor(rawKey?: string | null): CollectionColorDef {
  const normalized = normalizeCollectionKey(rawKey);
  if (normalized && COLLECTION_COLORS[normalized]) {
    return COLLECTION_COLORS[normalized];
  }
  return DEFAULT_COLLECTION_COLOR;
}
