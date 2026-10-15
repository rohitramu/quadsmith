import { describe, it, expect } from "vitest";
import { screen } from "@testing-library/react";
import { Routes, Route } from "react-router-dom";
import { renderWithProviders } from "../../test/test-utils";
import { CategoryPage } from "../CategoryPage";
import { HARDWARE_COLLECTIONS } from "../../lib/hardwareCollections";

describe("CategoryPage Component", () => {
  it("renders category title and all 11 hardware collection cards for 'hardware'", () => {
    renderWithProviders(
      <Routes>
        <Route path="/components/:categoryId" element={<CategoryPage />} />
      </Routes>,
      { route: "/components/hardware" },
    );

    // Title
    expect(screen.getByRole("heading", { level: 1, name: "hardware" })).toBeInTheDocument();

    // Verify all hardware collections are present
    expect(HARDWARE_COLLECTIONS).toHaveLength(11);
    for (const collection of HARDWARE_COLLECTIONS) {
      const heading = screen.getByRole("heading", {
        level: 2,
        name: collection.name,
      });
      expect(heading).toBeInTheDocument();
      const cardLink = heading.closest("a");
      expect(cardLink).toHaveAttribute("href", `/components/hardware/${collection.id}`);
      // Collection icon should be present in the card
      const iconSvg = cardLink?.querySelector("svg");
      expect(iconSvg).toBeInTheDocument();
      if (collection.description) {
        expect(screen.getByText(collection.description)).toBeInTheDocument();
      }
    }
  });

  it("renders category title and all 5 software collection cards for 'software'", () => {
    renderWithProviders(
      <Routes>
        <Route path="/components/:categoryId" element={<CategoryPage />} />
      </Routes>,
      { route: "/components/software" },
    );

    // Title
    expect(screen.getByRole("heading", { level: 1, name: "software" })).toBeInTheDocument();

    const expectedCollections = [
      { id: "fc-firmware", name: "Flight Controller Firmware" },
      { id: "esc-firmware", name: "ESC Firmware" },
      { id: "vtx-firmware", name: "Video Transmitter Firmware" },
      { id: "radio-firmware", name: "Radio Firmware" },
      { id: "software-tools", name: "Software Tools" },
    ];

    for (const col of expectedCollections) {
      const heading = screen.getByRole("heading", {
        level: 2,
        name: col.name,
      });
      expect(heading).toBeInTheDocument();
      const cardLink = heading.closest("a");
      expect(cardLink).toHaveAttribute("href", `/components/software/${col.id}`);
      expect(cardLink?.querySelector("svg")).toBeInTheDocument();
    }
  });

  it("renders category title and all 4 gear collection cards for 'gear'", () => {
    renderWithProviders(
      <Routes>
        <Route path="/components/:categoryId" element={<CategoryPage />} />
      </Routes>,
      { route: "/components/gear" },
    );

    // Title
    expect(screen.getByRole("heading", { level: 1, name: "gear" })).toBeInTheDocument();

    const expectedCollections = [
      { id: "transmitters", name: "Transmitters" },
      { id: "goggles", name: "FPV Goggles" },
      { id: "chargers", name: "Battery Chargers" },
      { id: "field-tools", name: "Field Tools" },
    ];

    for (const col of expectedCollections) {
      const heading = screen.getByRole("heading", {
        level: 2,
        name: col.name,
      });
      expect(heading).toBeInTheDocument();
      const cardLink = heading.closest("a");
      expect(cardLink).toHaveAttribute("href", `/components/gear/${col.id}`);
      expect(cardLink?.querySelector("svg")).toBeInTheDocument();
    }
  });

  it("renders empty state message when category has no collections", () => {
    renderWithProviders(
      <Routes>
        <Route path="/components/:categoryId" element={<CategoryPage />} />
      </Routes>,
      { route: "/components/unknown-category" },
    );

    expect(screen.getByRole("heading", { level: 1, name: "unknown-category" })).toBeInTheDocument();
    expect(screen.getByText("No collections found in this category.")).toBeInTheDocument();
  });
});
