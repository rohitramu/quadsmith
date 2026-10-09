import { describe, it, expect } from "vitest";
import { formatStatorSize, getYouTubeEmbedUrl, getTwrDescription } from "../format";

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

  it("classifies cinelifter & heavy payload (1.8 <= TWR < 2.8)", () => {
    expect(getTwrDescription(1.8)).toBe("Cinelifter & Heavy Payload");
    expect(getTwrDescription(2.2)).toBe("Cinelifter & Heavy Payload");
    expect(getTwrDescription(2.7)).toBe("Cinelifter & Heavy Payload");
  });

  it("classifies sluggish / underpowered (1.0 <= TWR < 1.8)", () => {
    expect(getTwrDescription(1.0)).toBe("Sluggish / Underpowered");
    expect(getTwrDescription(1.4)).toBe("Sluggish / Underpowered");
    expect(getTwrDescription(1.79)).toBe("Sluggish / Underpowered");
  });

  it("classifies cannot take off (TWR < 1.0)", () => {
    expect(getTwrDescription(0.99)).toBe("Cannot Take Off");
    expect(getTwrDescription(0.5)).toBe("Cannot Take Off");
    expect(getTwrDescription(0)).toBe("Cannot Take Off");
  });
});
