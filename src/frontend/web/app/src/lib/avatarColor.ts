export interface AvatarPalette {
  name: string;
  badgeClass: string;
  bgClass: string;
  textClass: string;
  borderClass: string;
}

export const AVATAR_PALETTES: AvatarPalette[] = [
  {
    name: "purple",
    badgeClass:
      "bg-purple-100 text-purple-800 border-purple-300 dark:bg-purple-950/60 dark:text-purple-300 dark:border-purple-800/80",
    bgClass: "bg-purple-100 dark:bg-purple-950/60",
    textClass: "text-purple-800 dark:text-purple-300",
    borderClass: "border-purple-300 dark:border-purple-800/80",
  },
  {
    name: "blue",
    badgeClass:
      "bg-blue-100 text-blue-800 border-blue-300 dark:bg-blue-950/60 dark:text-blue-300 dark:border-blue-800/80",
    bgClass: "bg-blue-100 dark:bg-blue-950/60",
    textClass: "text-blue-800 dark:text-blue-300",
    borderClass: "border-blue-300 dark:border-blue-800/80",
  },
  {
    name: "emerald",
    badgeClass:
      "bg-emerald-100 text-emerald-800 border-emerald-300 dark:bg-emerald-950/60 dark:text-emerald-300 dark:border-emerald-800/80",
    bgClass: "bg-emerald-100 dark:bg-emerald-950/60",
    textClass: "text-emerald-800 dark:text-emerald-300",
    borderClass: "border-emerald-300 dark:border-emerald-800/80",
  },
  {
    name: "amber",
    badgeClass:
      "bg-amber-100 text-amber-800 border-amber-300 dark:bg-amber-950/60 dark:text-amber-300 dark:border-amber-800/80",
    bgClass: "bg-amber-100 dark:bg-amber-950/60",
    textClass: "text-amber-800 dark:text-amber-300",
    borderClass: "border-amber-300 dark:border-amber-800/80",
  },
  {
    name: "rose",
    badgeClass:
      "bg-rose-100 text-rose-800 border-rose-300 dark:bg-rose-950/60 dark:text-rose-300 dark:border-rose-800/80",
    bgClass: "bg-rose-100 dark:bg-rose-950/60",
    textClass: "text-rose-800 dark:text-rose-300",
    borderClass: "border-rose-300 dark:border-rose-800/80",
  },
  {
    name: "cyan",
    badgeClass:
      "bg-cyan-100 text-cyan-800 border-cyan-300 dark:bg-cyan-950/60 dark:text-cyan-300 dark:border-cyan-800/80",
    bgClass: "bg-cyan-100 dark:bg-cyan-950/60",
    textClass: "text-cyan-800 dark:text-cyan-300",
    borderClass: "border-cyan-300 dark:border-cyan-800/80",
  },
  {
    name: "teal",
    badgeClass:
      "bg-teal-100 text-teal-800 border-teal-300 dark:bg-teal-950/60 dark:text-teal-300 dark:border-teal-800/80",
    bgClass: "bg-teal-100 dark:bg-teal-950/60",
    textClass: "text-teal-800 dark:text-teal-300",
    borderClass: "border-teal-300 dark:border-teal-800/80",
  },
  {
    name: "indigo",
    badgeClass:
      "bg-indigo-100 text-indigo-800 border-indigo-300 dark:bg-indigo-950/60 dark:text-indigo-300 dark:border-indigo-800/80",
    bgClass: "bg-indigo-100 dark:bg-indigo-950/60",
    textClass: "text-indigo-800 dark:text-indigo-300",
    borderClass: "border-indigo-300 dark:border-indigo-800/80",
  },
  {
    name: "orange",
    badgeClass:
      "bg-orange-100 text-orange-800 border-orange-300 dark:bg-orange-950/60 dark:text-orange-300 dark:border-orange-800/80",
    bgClass: "bg-orange-100 dark:bg-orange-950/60",
    textClass: "text-orange-800 dark:text-orange-300",
    borderClass: "border-orange-300 dark:border-orange-800/80",
  },
  {
    name: "pink",
    badgeClass:
      "bg-pink-100 text-pink-800 border-pink-300 dark:bg-pink-950/60 dark:text-pink-300 dark:border-pink-800/80",
    bgClass: "bg-pink-100 dark:bg-pink-950/60",
    textClass: "text-pink-800 dark:text-pink-300",
    borderClass: "border-pink-300 dark:border-pink-800/80",
  },
  {
    name: "lime",
    badgeClass:
      "bg-lime-100 text-lime-800 border-lime-300 dark:bg-lime-950/60 dark:text-lime-300 dark:border-lime-800/80",
    bgClass: "bg-lime-100 dark:bg-lime-950/60",
    textClass: "text-lime-800 dark:text-lime-300",
    borderClass: "border-lime-300 dark:border-lime-800/80",
  },
];

/**
 * Deterministically maps a UUID (or fallback string) to one of the component category chip color palettes.
 */
export function getAvatarPalette(uuid?: string | null): AvatarPalette {
  if (!uuid) return AVATAR_PALETTES[0];
  let hash = 0;
  for (let i = 0; i < uuid.length; i++) {
    hash = (hash << 5) - hash + uuid.charCodeAt(i);
    hash |= 0;
  }
  const index = Math.abs(hash) % AVATAR_PALETTES.length;
  return AVATAR_PALETTES[index];
}

/**
 * Gets the first uppercase initial from a user's handle or display name.
 */
export function getInitial(nameOrHandle?: string | null): string {
  if (!nameOrHandle) return "P";
  const trimmed = nameOrHandle.trim();
  if (!trimmed) return "P";
  return trimmed.charAt(0).toUpperCase();
}
