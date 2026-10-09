import { describe, it, expect } from "vitest";
import {
  COLLECTION_COLORS,
  DEFAULT_COLLECTION_COLOR,
  getCollectionColor,
  normalizeCollectionKey,
} from "../collectionColors";
import { HARDWARE_COLLECTIONS } from "../hardwareCollections";

describe("collectionColors module", () => {
  it("contains unique color definitions for all 11 hardware collections and builds", () => {
    const requiredCollections = [
      "antennas",
      "batteries",
      "builds",
      "cameras",
      "electronic-speed-controllers",
      "flight-controllers",
      "frames",
      "gps-receivers",
      "motors",
      "propellers",
      "receivers",
      "video-transmitters",
    ];

    expect(Object.keys(COLLECTION_COLORS)).toHaveLength(12);

    for (const key of requiredCollections) {
      expect(COLLECTION_COLORS[key]).toBeDefined();
      expect(COLLECTION_COLORS[key].hex).toMatch(/^#[0-9a-f]{6}$/i);
      expect(COLLECTION_COLORS[key].trimClass).toContain("bg-");
      expect(COLLECTION_COLORS[key].badgeClass).toContain("text-");
      expect(COLLECTION_COLORS[key].dotClass).toContain("bg-");
    }

    // Ensure all 12 collections have unique hex codes and color names
    const hexSet = new Set(Object.values(COLLECTION_COLORS).map((c) => c.hex));
    expect(hexSet.size).toBe(12);

    const colorNameSet = new Set(Object.values(COLLECTION_COLORS).map((c) => c.colorName));
    expect(colorNameSet.size).toBe(12);
  });

  it("assigns colors to every item in HARDWARE_COLLECTIONS", () => {
    for (const col of HARDWARE_COLLECTIONS) {
      expect(col.color).toBeDefined();
      expect(col.color?.id).toBe(col.id);
      expect(col.color?.hex).toBe(COLLECTION_COLORS[col.id].hex);
    }
  });

  it("normalizes paths, plural forms, and aliases correctly", () => {
    expect(normalizeCollectionKey("components/hardware/batteries")).toBe("batteries");
    expect(normalizeCollectionKey("esc")).toBe("electronic-speed-controllers");
    expect(getCollectionColor("components/hardware/batteries").id).toBe("batteries");
    expect(getCollectionColor("batteries").id).toBe("batteries");
    expect(getCollectionColor("battery").id).toBe("batteries");

    expect(getCollectionColor("components/hardware/electronic-speed-controllers").id).toBe(
      "electronic-speed-controllers",
    );
    expect(getCollectionColor("esc").id).toBe("electronic-speed-controllers");
    expect(getCollectionColor("escs").id).toBe("electronic-speed-controllers");

    expect(getCollectionColor("fc").id).toBe("flight-controllers");
    expect(getCollectionColor("flight_controllers").id).toBe("flight-controllers");

    expect(getCollectionColor("rx").id).toBe("receivers");
    expect(getCollectionColor("vtx").id).toBe("video-transmitters");
    expect(getCollectionColor("gps").id).toBe("gps-receivers");
    expect(getCollectionColor("prop").id).toBe("propellers");
    expect(getCollectionColor("build").id).toBe("builds");
    expect(getCollectionColor("builds").id).toBe("builds");
  });

  it("falls back gracefully for unknown or empty collection paths", () => {
    const fallbackEmpty = getCollectionColor("");
    expect(fallbackEmpty.id).toBe(DEFAULT_COLLECTION_COLOR.id);

    const fallbackNull = getCollectionColor(null);
    expect(fallbackNull.id).toBe(DEFAULT_COLLECTION_COLOR.id);

    const fallbackUnknown = getCollectionColor("unknown-resource");
    expect(fallbackUnknown.id).toBe(DEFAULT_COLLECTION_COLOR.id);
  });
});
