// @vitest-environment node
import { describe, it, expect } from "vitest";
import {
  COLOR_PALETTE,
  COLLECTION_COLORS,
  DEFAULT_COLLECTION_COLOR,
  getCollectionColor,
  getCollectionColorByCode,
  normalizeCollectionKey,
} from "@quadsmith/shared";
import { HARDWARE_COLLECTIONS } from "../hardwareCollections";

describe("collectionColors module", () => {
  it("defines an authoritative color palette of at least 25 distinct colors with light and dark variants", () => {
    expect(COLOR_PALETTE.length).toBeGreaterThanOrEqual(25);

    // Verify all hex codes in palette are unique
    const hexCodes = new Set(COLOR_PALETTE.map((c) => c.hex.toLowerCase()));
    expect(hexCodes.size).toBe(COLOR_PALETTE.length);

    // Verify all color names are unique
    const colorNames = new Set(COLOR_PALETTE.map((c) => c.colorName.toLowerCase()));
    expect(colorNames.size).toBe(COLOR_PALETTE.length);

    // Verify each color has both light mode and dark mode styling classes
    for (const color of COLOR_PALETTE) {
      expect(color.badgeClass).toContain("bg-");
      expect(color.badgeClass).toContain("dark:bg-");
      expect(color.badgeClass).toContain("text-");
      expect(color.badgeClass).toContain("dark:text-");
      expect(color.bgClass).toContain("dark:bg-");
      expect(color.textClass).toContain("dark:text-");
      expect(color.borderClass).toContain("dark:border-");
      expect(color.trimClass).toContain("bg-");
      expect(color.dotClass).toContain("bg-");
      expect(color.iconBgClass).toContain("dark:bg-");
      expect(color.hoverBorderClass).toContain("hover:border-");
    }
  });

  it("contains color definitions for all hardware collections, builds, firmware, and tools", () => {
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
      "fc-firmware",
      "esc-firmware",
      "vtx-firmware",
      "radio-firmware",
      "software-tools",
      "transmitters",
      "goggles",
      "chargers",
      "field-tools",
    ];

    for (const key of requiredCollections) {
      expect(COLLECTION_COLORS[key]).toBeDefined();
      expect(COLLECTION_COLORS[key].hex).toMatch(/^#[0-9a-f]{6}$/i);
      expect(COLLECTION_COLORS[key].trimClass).toContain("bg-");
      expect(COLLECTION_COLORS[key].badgeClass).toContain("text-");
      expect(COLLECTION_COLORS[key].dotClass).toContain("bg-");
    }

    // Ensure all 11 hardware collections have unique hex codes
    const hardwareHexSet = new Set(HARDWARE_COLLECTIONS.map((c) => c.color?.hex));
    expect(hardwareHexSet.size).toBe(11);

    // Builds shares the blue theme (#3b82f6) with flight controllers
    expect(COLLECTION_COLORS["builds"].hex).toBe("#3b82f6");
    expect(COLLECTION_COLORS["flight-controllers"].hex).toBe("#3b82f6");
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
    expect(getCollectionColor("builds").hex).toBe("#3b82f6");
    expect(getCollectionColor("builds").textClass).toBe("text-blue-600 dark:text-blue-400");
    expect(getCollectionColor("the-forge").id).toBe("the-forge");
    expect(getCollectionColor("the-forge").hex).toBe("#FF6D00");
    expect(getCollectionColor("the-forge").textClass).toBe("text-[#FF6D00] dark:text-[#FF6D00]");
    expect(getCollectionColor("forge").id).toBe("the-forge");
    expect(getCollectionColor("build-wizard").id).toBe("the-forge");
    expect(getCollectionColor("build-wizard").hex).toBe("#FF6D00");
    expect(getCollectionColor("build-wizard").textClass).toBe("text-[#FF6D00] dark:text-[#FF6D00]");
    expect(getCollectionColor("builds/new").id).toBe("the-forge");
    expect(getCollectionColor("builds/new").hex).toBe("#FF6D00");

    // Software & Gear collections
    expect(normalizeCollectionKey("components/software/fc-firmware")).toBe("fc-firmware");
    expect(getCollectionColor("fc-firmware").id).toBe("fc-firmware");
    expect(getCollectionColor("esc-firmware").id).toBe("esc-firmware");
    expect(getCollectionColor("vtx-firmware").id).toBe("vtx-firmware");
    expect(getCollectionColor("radio-firmware").id).toBe("radio-firmware");
    expect(getCollectionColor("software-tools").id).toBe("software-tools");
    expect(getCollectionColor("configurator").id).toBe("software-tools");
    expect(getCollectionColor("radio").id).toBe("transmitters");
    expect(getCollectionColor("transmitters").id).toBe("transmitters");
    expect(getCollectionColor("goggles").id).toBe("goggles");
    expect(getCollectionColor("fpv-goggles").id).toBe("goggles");
    expect(getCollectionColor("chargers").id).toBe("chargers");
    expect(getCollectionColor("field-tools").id).toBe("field-tools");
    expect(getCollectionColor("tools").id).toBe("field-tools");
  });

  it("falls back gracefully to default grey for unknown, missing, or empty color codes", () => {
    const fallbackEmpty = getCollectionColor("");
    expect(fallbackEmpty.id).toBe(DEFAULT_COLLECTION_COLOR.id);
    expect(fallbackEmpty.hex).toBe("#64748b");
    expect(fallbackEmpty.colorName).toBe("slate");

    const fallbackNull = getCollectionColor(null);
    expect(fallbackNull.id).toBe(DEFAULT_COLLECTION_COLOR.id);
    expect(fallbackNull.hex).toBe("#64748b");

    const fallbackUnknown = getCollectionColor("unknown-resource");
    expect(fallbackUnknown.id).toBe(DEFAULT_COLLECTION_COLOR.id);
    expect(fallbackUnknown.hex).toBe("#64748b");

    // Testing getCollectionColorByCode
    expect(getCollectionColorByCode("").hex).toBe("#64748b");
    expect(getCollectionColorByCode(null).hex).toBe("#64748b");
    expect(getCollectionColorByCode(undefined).hex).toBe("#64748b");
    expect(getCollectionColorByCode("#999999").hex).toBe("#64748b");

    // Case-insensitivity and prefix handling
    expect(getCollectionColorByCode("#10B981").colorName).toBe("emerald");
    expect(getCollectionColorByCode("10b981").colorName).toBe("emerald");
  });
});
