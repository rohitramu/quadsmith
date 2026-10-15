// @vitest-environment node
import { describe, it, expect } from "vitest";
import { getAvatarPalette, getInitial, AVATAR_PALETTES } from "@quadsmith/shared";

describe("Avatar Color Utilities", () => {
  it("deterministically returns the same palette for a given UUID", () => {
    const uuid1 = "018f0000-0000-7000-0000-000000000001";
    const palette1a = getAvatarPalette(uuid1);
    const palette1b = getAvatarPalette(uuid1);

    expect(palette1a).toEqual(palette1b);
    expect(palette1a.badgeClass).toBeTruthy();
    expect(palette1a.bgClass).toBeTruthy();
    expect(palette1a.textClass).toBeTruthy();
  });

  it("returns different palettes for different UUIDs", () => {
    const uuid1 = "018f0000-0000-7000-0000-000000000001";
    const uuid2 = "018f0000-0000-7000-0000-000000000005";
    const p1 = getAvatarPalette(uuid1);
    const p2 = getAvatarPalette(uuid2);

    expect(AVATAR_PALETTES).toContain(p1);
    expect(AVATAR_PALETTES).toContain(p2);
  });

  it("handles null or undefined UUID with default fallback", () => {
    const fallback = getAvatarPalette(null);
    expect(fallback).toEqual(AVATAR_PALETTES[0]);
  });

  it("extracts uppercase initial correctly", () => {
    expect(getInitial("alex")).toBe("A");
    expect(getInitial("fpv_pilot")).toBe("F");
    expect(getInitial(" RotorNinja")).toBe("R");
    expect(getInitial("")).toBe("P");
    expect(getInitial(null)).toBe("P");
  });
});
