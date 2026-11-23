import { describe, it, expect } from "vitest";
import { screen, waitFor, within } from "@testing-library/react";
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

    // Check bounded container & single-column row-card links to product detail pages
    const pageContainer = screen.getByTestId("collection-page-container");
    expect(pageContainer.className).toContain("max-w-[110rem]");
    expect(pageContainer.className).toContain("mx-auto");

    const catalogGrid = screen.getByTestId("catalog-grid");
    expect(catalogGrid.className).toContain("grid-cols-1");
    expect(catalogGrid.className).not.toContain("2xl:grid-cols-3");

    const emaxLink = screen.getByRole("link", { name: /view eco ii 2207/i });
    expect(emaxLink).toHaveAttribute("href", "/components/hardware/motors/emax-eco-ii-2207");
  });

  it("handles sorting via toolbar sort selector and direction toggle", async () => {
    const { user } = renderWithProviders(
      <Routes>
        <Route path="/components/:categoryId/:collectionId" element={<CollectionPage />} />
      </Routes>,
      { route: "/components/hardware/motors" },
    );

    await waitFor(() => {
      expect(screen.getByText("ECO II 2207")).toBeInTheDocument();
    });

    // Find sort combobox
    const sortCombobox = screen.getByRole("combobox", { name: /sort by field/i });
    expect(sortCombobox).toBeInTheDocument();
    expect(sortCombobox).toHaveTextContent("Sort: Name");

    // Find direction button (initially ascending)
    const dirBtn = screen.getByRole("button", { name: /sort ascending/i });
    expect(dirBtn).toBeInTheDocument();

    // Toggle direction to descending
    await user.click(dirBtn);
    expect(await screen.findByRole("button", { name: /sort descending/i })).toBeInTheDocument();
  });

  it("opens property selector and allows toggling visible properties", async () => {
    const { user } = renderWithProviders(
      <Routes>
        <Route path="/components/:categoryId/:collectionId" element={<CollectionPage />} />
      </Routes>,
      { route: "/components/hardware/motors" },
    );

    await waitFor(() => {
      expect(screen.getByText("ECO II 2207")).toBeInTheDocument();
    });

    // Find and click the Properties dropdown button (5 default spec properties on Motors)
    const propertiesButton = screen.getByRole("button", { name: /properties \(5\)/i });
    await user.click(propertiesButton);

    // Popover should be open without duplicating Name or Manufacturer
    const visibleHeader = screen.getByText(/visible \(drag or arrows\)/i);
    expect(visibleHeader).toBeInTheDocument();
    const popover = visibleHeader.closest("div.absolute") as HTMLElement;
    expect(within(popover).queryByText("Name")).not.toBeInTheDocument();
    expect(within(popover).queryByText("Manufacturer")).not.toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Reset to Default" })).toBeInTheDocument();

    // Find checkboxes for visible spec properties
    const checkboxes = screen.getAllByRole("checkbox");
    expect(checkboxes.length).toBeGreaterThanOrEqual(5);
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

  it("filters out internal-only products by default and does not render the show internal checkbox", async () => {
    renderWithProviders(
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

    // The 'Show internal' toggle should not be present
    expect(screen.queryByRole("checkbox", { name: /show internal/i })).not.toBeInTheDocument();
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

    // Click sort direction button -> desc (^name)
    const dirBtn = screen.getByRole("button", { name: /sort ascending/i });
    await user.click(dirBtn);
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

    // Toggle direction button again -> asc (name)
    const descDirBtn = screen.getByRole("button", { name: /sort descending/i });
    await user.click(descDirBtn);
    await waitFor(() => {
      expect(locationSearch.textContent).toContain("sort=name");
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

    // Sort dropdown should show Name, and spec card should show KV
    expect(screen.getByText("Sort: Name")).toBeInTheDocument();
    expect(screen.getByText("KV")).toBeInTheDocument();

    // Spec card should NOT show Stator Size
    expect(screen.queryByText("Stator Size")).not.toBeInTheDocument();
  });

  it("updates URL properties parameter when modifying property selection and removes it on Reset to Default", async () => {
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

    // Open PropertySelector
    const propertiesButton = screen.getByRole("button", { name: /properties \(/i });
    await user.click(propertiesButton);

    // Hide Stator Size property by clicking its checkbox
    const statorCheckbox = screen.getAllByRole("checkbox")[1]; // In visible properties
    await user.click(statorCheckbox);

    await waitFor(() => {
      expect(locationSearch.textContent).toContain("properties=");
    });

    // Reset to Default
    const resetButton = screen.getByRole("button", { name: /reset to default/i });
    await user.click(resetButton);

    await waitFor(() => {
      expect(locationSearch.textContent).not.toContain("properties=");
    });
  });

  it("renders single-column clickable row-cards with ~5 default spec pills and a square right-side photo pane", async () => {
    renderWithProviders(
      <Routes>
        <Route path="/components/:categoryId/:collectionId" element={<CollectionPage />} />
      </Routes>,
      { route: "/components/hardware/motors" },
    );

    await waitFor(() => {
      expect(screen.getByText("ECO II 2207")).toBeInTheDocument();
    });

    // Check whole-card clickable links are rendered without hover cards or footer Details links
    const cards = screen.getAllByTestId("catalog-card");
    expect(cards.length).toBeGreaterThan(0);
    expect(cards[0].tagName).toBe("A");
    expect(cards[0]).toHaveAttribute("href", "/components/hardware/motors/emax-eco-ii-2207");

    expect(screen.queryByTestId("product-hover-card")).not.toBeInTheDocument();
    expect(screen.queryByRole("link", { name: /details/i })).not.toBeInTheDocument();

    // Verify 5 default spec pills are shown without duplicating Manufacturer
    expect(within(cards[0]).getByText("KV")).toBeInTheDocument();
    expect(within(cards[0]).getByText("Stator Size")).toBeInTheDocument();
    expect(within(cards[0]).getByText("Max Current (A)")).toBeInTheDocument();
    expect(within(cards[0]).getByText("Prop Mounts")).toBeInTheDocument();
    expect(within(cards[0]).getByText("Weight (g)")).toBeInTheDocument();
    expect(within(cards[0]).queryByText("Manufacturer")).not.toBeInTheDocument();

    // Verify full-height 1:1 square right-side photo pane
    const photoPanes = screen.getAllByTestId("catalog-card-photo");
    expect(photoPanes.length).toBe(cards.length);
    expect(photoPanes[0].className).toContain("aspect-square");
    expect(photoPanes[0].className).toContain("self-stretch");
  });
});
