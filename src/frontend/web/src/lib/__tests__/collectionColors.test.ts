import { describe, it, expect } from "vitest";
import { getOption } from "@bufbuild/protobuf";
import { frontend } from "../../gen/quadsmith/_common_pb";
import { BatterySchema } from "../../gen/quadsmith/battery_pb";
import { BuildSchema } from "../../gen/quadsmith/build_pb";
import {
  COLLECTION_COLORS,
  DEFAULT_COLLECTION_COLOR,
  getCollectionColor,
  getCollectionColorByCode,
  getCollectionColorFromSchema,
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

    // Ensure all 11 hardware collections have unique hex codes
    const hardwareHexSet = new Set(HARDWARE_COLLECTIONS.map((c) => c.color?.hex));
    expect(hardwareHexSet.size).toBe(11);

    // Builds shares the blue theme (#3b82f6) with flight controllers
    expect(COLLECTION_COLORS["builds"].hex).toBe("#3b82f6");
    expect(COLLECTION_COLORS["flight-controllers"].hex).toBe("#3b82f6");
    const allHexSet = new Set(Object.values(COLLECTION_COLORS).map((c) => c.hex));
    expect(allHexSet.size).toBe(11);
  });

  it("extracts colors directly from proto schema options as the source of truth", () => {
    // BatterySchema defines color_code: "#84cc16" in battery.proto
    const batteryProtoColor = getOption(BatterySchema, frontend)?.colorCode;
    expect(batteryProtoColor).toBe("#84cc16");

    const batteryColor = getCollectionColorFromSchema(BatterySchema, "batteries", "Batteries");
    expect(batteryColor.hex).toBe("#84cc16");
    expect(batteryColor.colorName).toBe("lime");
    expect(batteryColor.trimClass).toBe("bg-lime-500");

    // BuildSchema defines color_code: "#3b82f6" in build.proto
    const buildProtoColor = getOption(BuildSchema, frontend)?.colorCode;
    expect(buildProtoColor).toBe("#3b82f6");

    const buildColor = getCollectionColorFromSchema(BuildSchema, "builds", "Builds");
    expect(buildColor.hex).toBe("#3b82f6");
    expect(buildColor.colorName).toBe("blue");
    expect(buildColor.textClass).toBe("text-blue-600 dark:text-blue-400");

    // Verify all HARDWARE_COLLECTIONS derive their color from their proto schema
    for (const col of HARDWARE_COLLECTIONS) {
      const protoColor = getOption(col.schema, frontend)?.colorCode;
      expect(protoColor).toBeDefined();
      expect(col.color?.hex).toBe(protoColor);
      expect(COLLECTION_COLORS[col.id].hex).toBe(protoColor);
    }
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
    expect(getCollectionColor("build-wizard").id).toBe("builds");
    expect(getCollectionColor("build-wizard").hex).toBe("#3b82f6");
    expect(getCollectionColor("build-wizard").textClass).toBe("text-blue-600 dark:text-blue-400");
    expect(getCollectionColor("builds/new").id).toBe("builds");
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

    // Schema without frontend color option falls back to slate/grey
    const dummySchemaWithoutColor = { name: "MockMessage" };
    const colorFromEmptySchema = getCollectionColorFromSchema(dummySchemaWithoutColor);
    expect(colorFromEmptySchema.hex).toBe("#64748b");
    expect(colorFromEmptySchema.colorName).toBe("slate");
  });
});
