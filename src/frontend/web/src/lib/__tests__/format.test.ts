import { describe, it, expect } from "vitest";
import {
  formatStatorSize,
  getYouTubeEmbedUrl,
  getTwrDescription,
  getTwrColor,
  getHoverThrottleColor,
  formatProductTitle,
} from "../format";

describe("formatStatorSize", () => {
  it("formats integer stator dimensions as 4 digits", () => {
    expect(formatStatorSize(22, 7)).toBe("2207");
    expect(formatStatorSize(14, 4)).toBe("1404");
    expect(formatStatorSize(7, 2)).toBe("0702");
  });

  it("formats decimal stator dimensions correctly", () => {
    expect(formatStatorSize(22, 7.5)).toBe("2207.5");
    expect(formatStatorSize(12, 2.5)).toBe("1202.5");
    expect(formatStatorSize(22.6, 6.5)).toBe("22.6x6.5");
    expect(formatStatorSize(8.5, 20)).toBe("8.5x20");
  });

  it("handles null / undefined dimensions gracefully", () => {
    expect(formatStatorSize(null, null)).toBe("-");
    expect(formatStatorSize(22, null)).toBe("-");
    expect(formatStatorSize(undefined, 7)).toBe("-");
  });
});

describe("getYouTubeEmbedUrl", () => {
  it("converts watch URLs to embed URLs", () => {
    expect(getYouTubeEmbedUrl("https://www.youtube.com/watch?v=dQw4w9WgXcQ")).toBe(
      "https://www.youtube.com/embed/dQw4w9WgXcQ",
    );
  });

  it("converts youtu.be short URLs to embed URLs", () => {
    expect(getYouTubeEmbedUrl("https://youtu.be/dQw4w9WgXcQ")).toBe(
      "https://www.youtube.com/embed/dQw4w9WgXcQ",
    );
  });

  it("handles shorts and embed URLs", () => {
    expect(getYouTubeEmbedUrl("https://youtube.com/shorts/dQw4w9WgXcQ")).toBe(
      "https://www.youtube.com/embed/dQw4w9WgXcQ",
    );
    expect(getYouTubeEmbedUrl("https://youtube.com/embed/dQw4w9WgXcQ")).toBe(
      "https://youtube.com/embed/dQw4w9WgXcQ",
    );
  });

  it("returns null for invalid or null URLs", () => {
    expect(getYouTubeEmbedUrl(null)).toBeNull();
    expect(getYouTubeEmbedUrl("not-a-url")).toBeNull();
  });
});

describe("getTwrDescription", () => {
  it("classifies competition racing (TWR >= 8.0)", () => {
    expect(getTwrDescription(8.0)).toBe("Competition Racing");
    expect(getTwrDescription(10.5)).toBe("Competition Racing");
  });

  it("classifies freestyle acro (5.5 <= TWR < 8.0)", () => {
    expect(getTwrDescription(5.5)).toBe("Freestyle Acro");
    expect(getTwrDescription(6.4)).toBe("Freestyle Acro");
    expect(getTwrDescription(7.9)).toBe("Freestyle Acro");
  });

  it("classifies sport & toothpick (4.0 <= TWR < 5.5)", () => {
    expect(getTwrDescription(4.0)).toBe("Sport & Toothpick");
    expect(getTwrDescription(4.7)).toBe("Sport & Toothpick");
    expect(getTwrDescription(5.4)).toBe("Sport & Toothpick");
  });

  it("classifies long range cruiser (2.8 <= TWR < 4.0)", () => {
    expect(getTwrDescription(2.8)).toBe("Long Range Cruiser");
    expect(getTwrDescription(3.5)).toBe("Long Range Cruiser");
    expect(getTwrDescription(3.9)).toBe("Long Range Cruiser");
  });

  it("classifies underpowered (1.8 <= TWR < 2.8)", () => {
    expect(getTwrDescription(1.8)).toBe("Underpowered");
    expect(getTwrDescription(2.2)).toBe("Underpowered");
    expect(getTwrDescription(2.7)).toBe("Underpowered");
  });

  it("classifies unflyable (TWR < 1.8)", () => {
    expect(getTwrDescription(1.79)).toBe("Unflyable");
    expect(getTwrDescription(1.4)).toBe("Unflyable");
    expect(getTwrDescription(1.0)).toBe("Unflyable");
    expect(getTwrDescription(0.99)).toBe("Unflyable");
    expect(getTwrDescription(0.5)).toBe("Unflyable");
    expect(getTwrDescription(0)).toBe("Unflyable");
  });
});

describe("formatProductTitle", () => {
  it("formats manufacturer and product name with a space without dash", () => {
    expect(formatProductTitle("EMAX", "ECO II 2207")).toBe("EMAX ECO II 2207");
  });

  it("strips redundant manufacturer prefix from product name", () => {
    expect(formatProductTitle("Sub250", "Sub250 1404 4500KV")).toBe("Sub250 1404 4500KV");
    expect(formatProductTitle("Walksnail", "Walksnail Moonlight Camera")).toBe(
      "Walksnail Moonlight Camera",
    );
    expect(formatProductTitle("TBS", "TBS - Triumph Pro")).toBe("TBS Triumph Pro");
  });

  it("handles identical manufacturer and product name without duplication", () => {
    expect(formatProductTitle("Foxeer", "Foxeer")).toBe("Foxeer");
  });

  it("falls back to product name if manufacturer is missing or empty", () => {
    expect(formatProductTitle(null, "Custom Frame")).toBe("Custom Frame");
    expect(formatProductTitle("", "Custom Frame")).toBe("Custom Frame");
  });

  it("falls back to fallback string if name is missing", () => {
    expect(formatProductTitle(null, null, "fallback-id")).toBe("fallback-id");
    expect(formatProductTitle("EMAX", null, "eco-ii-2207")).toBe("EMAX eco-ii-2207");
  });
});

describe("getTwrColor", () => {
  it("assigns purple for competition racing (>= 8.0)", () => {
    expect(getTwrColor(8.5).text).toContain("text-purple");
    expect(getTwrColor(8.5).badge).toContain("bg-purple");
  });

  it("assigns emerald for freestyle acro (5.5 - 7.9)", () => {
    expect(getTwrColor(6.4).text).toContain("text-emerald");
    expect(getTwrColor(6.4).badge).toContain("bg-emerald");
  });

  it("assigns cyan for sport & toothpick (4.0 - 5.4)", () => {
    expect(getTwrColor(4.5).text).toContain("text-cyan");
    expect(getTwrColor(4.5).badge).toContain("bg-cyan");
  });

  it("assigns sky blue for long range cruiser (2.8 - 3.9)", () => {
    expect(getTwrColor(3.2).text).toContain("text-sky");
    expect(getTwrColor(3.2).badge).toContain("bg-sky");
  });

  it("assigns amber for underpowered (1.8 - 2.7)", () => {
    expect(getTwrColor(2.2).text).toContain("text-amber");
    expect(getTwrColor(2.2).badge).toContain("bg-amber");
  });

  it("assigns rose red for unflyable (< 1.8)", () => {
    expect(getTwrColor(1.79).text).toContain("text-rose");
    expect(getTwrColor(1.79).badge).toContain("bg-rose");
    expect(getTwrColor(1.4).text).toContain("text-rose");
    expect(getTwrColor(1.4).badge).toContain("bg-rose");
    expect(getTwrColor(0.8).text).toContain("text-rose");
    expect(getTwrColor(0.8).badge).toContain("bg-rose");
  });
});

describe("getHoverThrottleColor", () => {
  it("assigns emerald for Category 1 (< 35%)", () => {
    const res = getHoverThrottleColor(25);
    expect(res.text).toContain("text-emerald");
    expect(res.bar).toBe("bg-emerald-500");
    expect(res.category).toContain("<35%");
  });

  it("assigns subtle sky blue for Category 2 (35% - 49.9%)", () => {
    const res = getHoverThrottleColor(42);
    expect(res.text).toContain("text-sky");
    expect(res.bar).toBe("bg-sky-500");
    expect(res.category).toContain("35–50%");
  });

  it("assigns amber for Category 3 (50% - 64.9%)", () => {
    const res = getHoverThrottleColor(56);
    expect(res.text).toContain("text-amber");
    expect(res.bar).toBe("bg-amber-500");
    expect(res.category).toContain("50–65%");
  });

  it("assigns rose red for Category 4 (>= 65%)", () => {
    const res = getHoverThrottleColor(70);
    expect(res.text).toContain("text-rose");
    expect(res.bar).toBe("bg-rose-500");
    expect(res.category).toContain("≥65%");
  });
});
