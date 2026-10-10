import { describe, it, expect } from "vitest";
import { screen, waitFor } from "@testing-library/react";
import { Routes, Route } from "react-router-dom";
import { renderWithProviders } from "../../test/test-utils";
import { BuildsPage } from "../BuildsPage";

describe("BuildsPage Component", () => {
  it("renders breadcrumbs, page title, and table data with mock builds", async () => {
    renderWithProviders(
      <Routes>
        <Route path="/builds" element={<BuildsPage />} />
      </Routes>,
      { route: "/builds" },
    );

    // Breadcrumbs: Home > Builds
    const homeBreadcrumb = screen.getByRole("link", { name: "Home" });
    expect(homeBreadcrumb).toBeInTheDocument();
    expect(homeBreadcrumb).toHaveAttribute("href", "/");

    // Title
    expect(screen.getByRole("heading", { level: 1, name: "Builds" })).toBeInTheDocument();

    // SmartFilterInput exists
    expect(screen.getByPlaceholderText(/filter components/i)).toBeInTheDocument();

    // Wait for mock data to populate table
    await waitFor(() => {
      expect(screen.getByText("Bando Basher 5 inch")).toBeInTheDocument();
    });

    expect(screen.getAllByText(/showing/i).length).toBeGreaterThan(0);
    expect(screen.getByText("Long Range Explorer 7 inch")).toBeInTheDocument();

    // Check row links to build profile page
    const bandoLink = screen.getByRole("link", { name: /view bando basher 5 inch/i });
    expect(bandoLink).toHaveAttribute("href", "/builds/bando-basher-5-inch");
    expect(screen.queryByRole("button", { name: "All Builds" })).not.toBeInTheDocument();
  });

  it("handles sorting by clicking column headers", async () => {
    const { user } = renderWithProviders(
      <Routes>
        <Route path="/builds" element={<BuildsPage />} />
      </Routes>,
      { route: "/builds" },
    );

    await waitFor(() => {
      expect(screen.getByText("Bando Basher 5 inch")).toBeInTheDocument();
    });

    // Find sort button for Name column
    const sortBtn = screen.getByRole("button", { name: /sort by name/i });
    expect(sortBtn).toBeInTheDocument();

    // Click to sort
    await user.click(sortBtn);
    expect(sortBtn).toHaveAttribute("title", "Sort by Name");
  });

  it("filters builds using presets", async () => {
    const { user } = renderWithProviders(
      <Routes>
        <Route path="/builds" element={<BuildsPage />} />
      </Routes>,
      { route: "/builds" },
    );

    await waitFor(() => {
      expect(screen.getByText("Bando Basher 5 inch")).toBeInTheDocument();
    });

    // Quick preset buttons
    const freestylePreset = screen.getByRole("button", { name: '5" Freestyle' });
    expect(freestylePreset).toBeInTheDocument();

    await user.click(freestylePreset);
    const filterInput = screen.getByPlaceholderText(/filter components/i);
    expect((filterInput as HTMLInputElement).value).toContain("5");
  });
});
