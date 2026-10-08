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
      expect(screen.getByText(`Browse all ${collection.name.toLowerCase()}`)).toBeInTheDocument();
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
