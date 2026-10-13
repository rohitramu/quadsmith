import { describe, it, expect } from "vitest";
import {
  HARDWARE_COLLECTIONS,
  getCollectionPath,
  getHardwareCollection,
} from "../hardwareCollections";

describe("Hardware Collections Path Options", () => {
  it("defines paths for all 11 hardware collections matching proto options", () => {
    expect(HARDWARE_COLLECTIONS).toHaveLength(11);

    const expectedPaths: Record<string, string> = {
      antennas: "components/hardware/antennas",
      batteries: "components/hardware/batteries",
      cameras: "components/hardware/cameras",
      "electronic-speed-controllers": "components/hardware/electronic-speed-controllers",
      "flight-controllers": "components/hardware/flight-controllers",
      frames: "components/hardware/frames",
      "gps-receivers": "components/hardware/gps-receivers",
      motors: "components/hardware/motors",
      propellers: "components/hardware/propellers",
      receivers: "components/hardware/receivers",
      "video-transmitters": "components/hardware/video-transmitters",
    };

    for (const col of HARDWARE_COLLECTIONS) {
      const path = getCollectionPath(col);
      expect(path).toBe(expectedPaths[col.id]);
      expect(col.path).toBe(expectedPaths[col.id]);
    }
  });

  it("finds collections by ID, alias, or full collection path", () => {
    expect(getHardwareCollection("motors")?.id).toBe("motors");
    expect(getHardwareCollection("motor")?.id).toBe("motors");
    expect(getHardwareCollection("components/hardware/motors")?.id).toBe("motors");
    expect(getHardwareCollection("/components/hardware/motors")?.id).toBe("motors");

    expect(getHardwareCollection("electronic-speed-controllers")?.id).toBe(
      "electronic-speed-controllers",
    );
    expect(getHardwareCollection("escs")?.id).toBe("electronic-speed-controllers");
    expect(getHardwareCollection("components/hardware/electronic-speed-controllers")?.id).toBe(
      "electronic-speed-controllers",
    );
  });
});
