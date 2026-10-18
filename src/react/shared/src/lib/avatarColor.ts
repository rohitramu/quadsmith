import { COLOR_PALETTE } from "./collectionColors";

export interface AvatarPalette {
  name: string;
  badgeClass: string;
  bgClass: string;
  textClass: string;
  borderClass: string;
}

/**
 * AVATAR_PALETTES provides all 32 distinct color presets derived directly from
 * the authoritative UI color palette in collectionColors.ts.
 */
export const AVATAR_PALETTES: AvatarPalette[] = COLOR_PALETTE.map((c) => ({
  name: c.colorName,
  badgeClass: c.badgeClass,
  bgClass: c.bgClass,
  textClass: c.textClass,
  borderClass: c.borderClass,
}));

/**
 * Deterministically maps a UUID (or fallback string) to one of the 32 color palettes.
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
