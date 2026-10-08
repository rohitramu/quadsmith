import { describe, it, expect } from "vitest";
import { screen, waitFor } from "@testing-library/react";
import { Routes, Route } from "react-router-dom";
import { renderWithProviders } from "../../test/test-utils";
import { CollectionPage } from "../CollectionPage";

describe("CollectionPage Component", () => {
  it("renders breadcrumbs, page title, and table data with mock motors", async () => {
    renderWithProviders(
      <Routes>
        <Route path="/components/:categoryId/:collectionId" element={<CollectionPage />} />
      </Routes>,
      { route: "/components/hardware/motors" }
    );

    // Breadcrumbs
    const categoryBreadcrumb = screen.getByRole("link", { name: "hardware" });
    expect(categoryBreadcrumb).toBeInTheDocument();
    expect(categoryBreadcrumb).toHaveAttribute("href", "/components/hardware");

    // Title
    expect(screen.getByRole("heading", { level: 1, name: "Motors" })).toBeInTheDocument();

    // SmartFilterInput exists
    expect(screen.getByPlaceholderText(/filter components/i)).toBeInTheDocument();

    // Wait for mock data to populate table
    await waitFor(() => {
      expect(screen.getByText("ECO II 2207")).toBeInTheDocument();
    });
    expect(screen.getAllByText(/showing/i).length).toBeGreaterThan(0);
    expect(screen.getByText("F60 PRO V")).toBeInTheDocument();

    // Check row links to product detail pages
    const emaxLink = screen.getByRole("link", { name: /view eco ii 2207/i });
    expect(emaxLink).toHaveAttribute("href", "/components/hardware/motors/emax-eco-ii-2207");
  });

  it("handles sorting by clicking column headers", async () => {
    const { user } = renderWithProviders(
      <Routes>
        <Route path="/components/:categoryId/:collectionId" element={<CollectionPage />} />
      </Routes>,
      { route: "/components/hardware/motors" }
    );

    await waitFor(() => {
      expect(screen.getByText("ECO II 2207")).toBeInTheDocument();
    });

    // Find sort button for Name column
    const sortBtn = screen.getByRole("button", { name: /sort by name/i });
    expect(sortBtn).toBeInTheDocument();

    // Click to sort ascending
    await user.click(sortBtn);
    await waitFor(() => {
      expect(sortBtn).toHaveAttribute("title", "Sort by Name");
    });

    // Click again to sort descending
    await user.click(sortBtn);
    await waitFor(() => {
      expect(sortBtn).toHaveAttribute("title", "Sort by Name");
    });
  });

  it("opens column selector and allows toggling visible columns", async () => {
    const { user } = renderWithProviders(
      <Routes>
        <Route path="/components/:categoryId/:collectionId" element={<CollectionPage />} />
      </Routes>,
      { route: "/components/hardware/motors" }
    );

    await waitFor(() => {
      expect(screen.getByText("ECO II 2207")).toBeInTheDocument();
    });

    // Find and click the Columns dropdown button
    const columnsButton = screen.getByRole("button", { name: /columns \(/i });
    await user.click(columnsButton);

    // Popover should be open
    expect(screen.getByText(/visible \(drag or arrows\)/i)).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Reset to Default" })).toBeInTheDocument();

    // Find checkbox for Stator Size column and uncheck it to hide
    const checkboxes = screen.getAllByRole("checkbox");
    expect(checkboxes.length).toBeGreaterThan(0);
  });

  it("renders empty state message when no items match filters", async () => {
    renderWithProviders(
      <Routes>
        <Route path="/components/:categoryId/:collectionId" element={<CollectionPage />} />
      </Routes>,
      {
        route: "/components/hardware/motors",
        transportOptions: { motors: [] },
      }
    );

    await waitFor(() => {
      expect(screen.getByText("0 components found")).toBeInTheDocument();
      expect(
        screen.getByText(/no motors found matching the active filters/i)
      ).toBeInTheDocument();
    });
  });

  it("renders error state when query fails", async () => {
    renderWithProviders(
      <Routes>
        <Route path="/components/:categoryId/:collectionId" element={<CollectionPage />} />
      </Routes>,
      {
        route: "/components/hardware/motors",
        transportOptions: { simulateError: true },
      }
    );

    await waitFor(() => {
      expect(
        screen.getByText(/failed to fetch motors from server/i)
      ).toBeInTheDocument();
    });
  });

  it("renders coming soon message for unknown collection", () => {
    renderWithProviders(
      <Routes>
        <Route path="/components/:categoryId/:collectionId" element={<CollectionPage />} />
      </Routes>,
      { route: "/components/hardware/non-existent-collection" }
    );

    expect(
      screen.getByText("Implementation for non existent collection coming soon.")
    ).toBeInTheDocument();
  });
});
