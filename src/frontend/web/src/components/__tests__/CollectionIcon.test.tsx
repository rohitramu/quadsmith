import { describe, it, expect } from "vitest";
import { render } from "@testing-library/react";
import { CollectionIcon, getCollectionIconComponent } from "../CollectionIcon";
import {
  Antenna,
  Battery,
  Camera,
  Compass,
  Cpu,
  Layers,
  Radio,
  Rocket,
  Tv,
  Wind,
  Wrench,
  Zap,
  Box,
} from "lucide-react";

describe("CollectionIcon Component", () => {
  it("resolves the correct icon component for each collection", () => {
    expect(getCollectionIconComponent("antennas")).toBe(Antenna);
    expect(getCollectionIconComponent("batteries")).toBe(Battery);
    expect(getCollectionIconComponent("battery")).toBe(Battery);
    expect(getCollectionIconComponent("builds")).toBe(Wrench);
    expect(getCollectionIconComponent("cameras")).toBe(Camera);
    expect(getCollectionIconComponent("electronic-speed-controllers")).toBe(Zap);
    expect(getCollectionIconComponent("esc")).toBe(Zap);
    expect(getCollectionIconComponent("flight-controllers")).toBe(Cpu);
    expect(getCollectionIconComponent("fc")).toBe(Cpu);
    expect(getCollectionIconComponent("frames")).toBe(Layers);
    expect(getCollectionIconComponent("gps-receivers")).toBe(Compass);
    expect(getCollectionIconComponent("gps")).toBe(Compass);
    expect(getCollectionIconComponent("motors")).toBe(Rocket);
    expect(getCollectionIconComponent("propellers")).toBe(Wind);
    expect(getCollectionIconComponent("props")).toBe(Wind);
    expect(getCollectionIconComponent("receivers")).toBe(Radio);
    expect(getCollectionIconComponent("rx")).toBe(Radio);
    expect(getCollectionIconComponent("video-transmitters")).toBe(Tv);
    expect(getCollectionIconComponent("vtx")).toBe(Tv);
    expect(getCollectionIconComponent("unknown")).toBe(Box);
    expect(getCollectionIconComponent(null)).toBe(Box);
  });

  it("renders the SVG icon element", () => {
    const { container } = render(
      <CollectionIcon collection="batteries" size={20} className="text-emerald-500" />,
    );
    const svg = container.querySelector("svg");
    expect(svg).toBeInTheDocument();
    expect(svg).toHaveClass("text-emerald-500");
  });
});
