import { describe, it, expect } from "vitest";
import { render } from "@testing-library/react";
import {
  CollectionIcon,
  getCollectionIconComponent,
  DroneFrame,
  Propeller,
  Motor,
} from "@quadsmith/shared";
import {
  Antenna,
  Battery,
  Camera,
  Satellite,
  Cpu,
  Radio,
  Tv,
  Wrench,
  Zap,
  Box,
  Hammer,
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
    expect(getCollectionIconComponent("motors")).toBe(Motor);
    expect(getCollectionIconComponent("propellers")).toBe(Propeller);
    expect(getCollectionIconComponent("props")).toBe(Propeller);
    expect(getCollectionIconComponent("receivers")).toBe(Radio);
    expect(getCollectionIconComponent("rx")).toBe(Radio);
    expect(getCollectionIconComponent("video-transmitters")).toBe(Tv);
    expect(getCollectionIconComponent("vtx")).toBe(Tv);
    expect(getCollectionIconComponent("the-forge")).toBe(Hammer);
    expect(getCollectionIconComponent("forge")).toBe(Hammer);
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

  it("safely resolves CollectionColorDef objects, hex strings, and boolean colors", () => {
    // 1. With CollectionColorDef object
    const colorDef = {
      id: "frames",
      hex: "#64748b",
      name: "Frames",
      colorName: "slate",
      badgeClass: "",
      trimClass: "",
      dotClass: "",
      borderTopClass: "",
      borderLeftClass: "",
      textClass: "",
      iconBgClass: "",
      hoverBorderClass: "",
    };
    const { container: c1 } = render(
      <CollectionIcon collection="frames" size={16} color={colorDef} />,
    );
    const svg1 = c1.querySelector("svg");
    expect(svg1).toHaveStyle({ color: "#64748b" });
    expect(svg1?.getAttribute("color")).toBe("#64748b");

    // 2. With boolean color={true}
    const { container: c2 } = render(<CollectionIcon collection="motors" size={16} color={true} />);
    const svg2 = c2.querySelector("svg");
    expect(svg2).toHaveStyle({ color: "#ef4444" });

    // 3. With hex string
    const { container: c3 } = render(
      <CollectionIcon collection="cameras" size={16} color="#06b6d4" />,
    );
    const svg3 = c3.querySelector("svg");
    expect(svg3).toHaveStyle({ color: "#06b6d4" });
  });
});
