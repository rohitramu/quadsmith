import { describe, it, expect } from "vitest";
import { screen, waitFor } from "@testing-library/react";
import { Routes, Route, useLocation } from "react-router-dom";
import { renderWithProviders } from "../../test/test-utils";
import { CollectionPage } from "../CollectionPage";

function LocationDisplay() {
  const location = useLocation();
  return <div data-testid="location-search">{location.search}</div>;
}

describe("CollectionPage Component", () => {
  it("renders breadcrumbs, page title, and table data with mock motors", async () => {
    renderWithProviders(
      <Routes>
        <Route path="/components/:categoryId/:collectionId" element={<CollectionPage />} />
      </Routes>,
      { route: "/components/hardware/motors" },
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
      { route: "/components/hardware/motors" },
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
      { route: "/components/hardware/motors" },
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
      },
    );

    await waitFor(() => {
      expect(screen.getByText("0 components found")).toBeInTheDocument();
      expect(screen.getByText(/no motors found matching the active filters/i)).toBeInTheDocument();
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
      },
    );

    await waitFor(() => {
      expect(screen.getByText(/failed to fetch motors from server/i)).toBeInTheDocument();
    });
  });

  it("renders coming soon message for unknown collection", () => {
    renderWithProviders(
      <Routes>
        <Route path="/components/:categoryId/:collectionId" element={<CollectionPage />} />
      </Routes>,
      { route: "/components/hardware/non-existent-collection" },
    );

    expect(
      screen.getByText("Implementation for non existent collection coming soon."),
    ).toBeInTheDocument();
  });

  it("filters out internal-only products by default and shows them when toggled", async () => {
    const { user } = renderWithProviders(
      <Routes>
        <Route path="/components/:categoryId/:collectionId" element={<CollectionPage />} />
      </Routes>,
      { route: "/components/hardware/flight-controllers" },
    );

    // Standalone FC should be visible
    await waitFor(() => {
      expect(screen.getByText("F405 V4 FC")).toBeInTheDocument();
    });

    // Internal FC should be filtered out by default
    expect(screen.queryByText("Integrated FC AIO")).not.toBeInTheDocument();

    // The 'Show internal' toggle should be present and unchecked
    const showInternalCheckbox = screen.getByRole("checkbox", { name: /show internal/i });
    expect(showInternalCheckbox).toBeInTheDocument();
    expect(showInternalCheckbox).not.toBeChecked();

    // Check the box to show internal components
    await user.click(showInternalCheckbox);

    // Both should now be visible
    await waitFor(() => {
      expect(screen.getByText("Integrated FC AIO")).toBeInTheDocument();
    });
    expect(screen.getByText("F405 V4 FC")).toBeInTheDocument();

    // Uncheck to filter them out again
    await user.click(showInternalCheckbox);
    await waitFor(() => {
      expect(screen.queryByText("Integrated FC AIO")).not.toBeInTheDocument();
    });
    expect(screen.getByText("F405 V4 FC")).toBeInTheDocument();
  });

  it("shows internal products when filtered explicitly with is_internal_only in smart filter", async () => {
    const { user } = renderWithProviders(
      <Routes>
        <Route path="/components/:categoryId/:collectionId" element={<CollectionPage />} />
      </Routes>,
      { route: "/components/hardware/flight-controllers" },
    );

    await waitFor(() => {
      expect(screen.getByText("F405 V4 FC")).toBeInTheDocument();
    });
    expect(screen.queryByText("Integrated FC AIO")).not.toBeInTheDocument();

    const filterInput = screen.getByPlaceholderText(/filter components/i);
    await user.type(filterInput, "is_internal_only == true{Enter}");

    await waitFor(() => {
      expect(screen.getByText("Integrated FC AIO")).toBeInTheDocument();
    });
  });

  it("updates URL query parameters when sorting and filtering", async () => {
    const { user } = renderWithProviders(
      <Routes>
        <Route
          path="/components/:categoryId/:collectionId"
          element={
            <>
              <CollectionPage />
              <LocationDisplay />
            </>
          }
        />
      </Routes>,
      { route: "/components/hardware/motors" },
    );

    await waitFor(() => {
      expect(screen.getByText("ECO II 2207")).toBeInTheDocument();
    });

    const locationSearch = screen.getByTestId("location-search");
    expect(locationSearch.textContent).toBe("");

    // Click sort button for Name column -> asc
    const sortBtn = screen.getByRole("button", { name: /sort by name/i });
    await user.click(sortBtn);
    await waitFor(() => {
      expect(locationSearch.textContent).toBe("?sort=name");
    });

    // Click again -> desc (^name)
    await user.click(sortBtn);
    await waitFor(() => {
      expect(locationSearch.textContent).toBe("?sort=%5Ename");
    });

    // Apply a filter
    const filterInput = screen.getByPlaceholderText(/filter components/i);
    await user.type(filterInput, "kv >= 2000{Enter}");
    await waitFor(() => {
      expect(locationSearch.textContent).toContain("filter=kv+%3E%3D+2000");
      expect(locationSearch.textContent).toContain("sort=%5Ename");
    });

    // Click sort button again -> removes sort from URL
    await user.click(sortBtn);
    await waitFor(() => {
      expect(locationSearch.textContent).not.toContain("sort=");
      expect(locationSearch.textContent).toContain("filter=kv+%3E%3D+2000");
    });
  });

  it("initializes sorting, filtering, and column selection from URL query parameters", async () => {
    renderWithProviders(
      <Routes>
        <Route
          path="/components/:categoryId/:collectionId"
          element={
            <>
              <CollectionPage />
              <LocationDisplay />
            </>
          }
        />
      </Routes>,
      {
        initialEntries: ["/components/hardware/motors?sort=%5Ename&filter=ECO&columns=name,kv"],
      },
    );

    await waitFor(() => {
      expect(screen.getByText("ECO II 2207")).toBeInTheDocument();
    });

    // Filter input should be populated with "ECO"
    const filterInput = screen.getByPlaceholderText(/filter components/i) as HTMLInputElement;
    expect(filterInput.value).toBe("ECO");

    // Table header should show Name and KV columns
    expect(screen.getByText("Name")).toBeInTheDocument();
    expect(screen.getByText("KV")).toBeInTheDocument();

    // Table header should NOT show Manufacturer or Stator Size
    expect(screen.queryByText("Manufacturer")).not.toBeInTheDocument();
    expect(screen.queryByText("Stator Size")).not.toBeInTheDocument();
  });

  it("updates URL columns parameter when modifying column selection and removes it on Reset to Default", async () => {
    const { user } = renderWithProviders(
      <Routes>
        <Route
          path="/components/:categoryId/:collectionId"
          element={
            <>
              <CollectionPage />
              <LocationDisplay />
            </>
          }
        />
      </Routes>,
      { route: "/components/hardware/motors" },
    );

    await waitFor(() => {
      expect(screen.getByText("ECO II 2207")).toBeInTheDocument();
    });

    const locationSearch = screen.getByTestId("location-search");
    expect(locationSearch.textContent).toBe("");

    // Open ColumnSelector
    const columnsButton = screen.getByRole("button", { name: /columns \(/i });
    await user.click(columnsButton);

    // Hide Stator Size column by clicking its checkbox
    const statorCheckbox = screen.getAllByRole("checkbox")[2]; // In visible columns
    await user.click(statorCheckbox);

    await waitFor(() => {
      expect(locationSearch.textContent).toContain("columns=");
    });

    // Reset to Default
    const resetButton = screen.getByRole("button", { name: /reset to default/i });
    await user.click(resetButton);

    await waitFor(() => {
      expect(locationSearch.textContent).not.toContain("columns=");
    });
  });

  it("displays product hover card when hovering over any cell in the table row", async () => {
    const { user } = renderWithProviders(
      <Routes>
        <Route path="/components/:categoryId/:collectionId" element={<CollectionPage />} />
      </Routes>,
      { route: "/components/hardware/motors" },
    );

    await waitFor(() => {
      expect(screen.getByText("ECO II 2207")).toBeInTheDocument();
    });

    // Find the row triggers
    const rowTriggers = screen.getAllByTestId("product-hover-trigger");
    expect(rowTriggers.length).toBeGreaterThan(0);

    // Hover over the first row trigger (a <tr> element)
    await user.hover(rowTriggers[0]);

    // Fast-forward or wait for hover card
    await waitFor(
      () => {
        expect(screen.getByTestId("product-hover-card")).toBeInTheDocument();
      },
      { timeout: 1000 },
    );

    expect(screen.getByTestId("product-hover-card")).toBeInTheDocument();
    expect(screen.getByText("Details")).toBeInTheDocument();
  });
});
