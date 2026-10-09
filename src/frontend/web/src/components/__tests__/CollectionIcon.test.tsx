import { describe, it, expect } from "vitest";
import { render } from "@testing-library/react";
import {
  CollectionIcon,
  getCollectionIconComponent,
  DroneFrame,
  Propeller,
} from "../CollectionIcon";
import {
  Antenna,
  Battery,
  Camera,
  Satellite,
  Cpu,
  Radio,
  Rocket,
  Tv,
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
    expect(getCollectionIconComponent("frames")).toBe(DroneFrame);
    expect(getCollectionIconComponent("gps-receivers")).toBe(Satellite);
    expect(getCollectionIconComponent("gps")).toBe(Satellite);
    expect(getCollectionIconComponent("motors")).toBe(Rocket);
    expect(getCollectionIconComponent("propellers")).toBe(Propeller);
    expect(getCollectionIconComponent("props")).toBe(Propeller);
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
