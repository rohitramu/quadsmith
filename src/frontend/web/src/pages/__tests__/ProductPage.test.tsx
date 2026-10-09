import { describe, it, expect } from "vitest";
import { screen, waitFor } from "@testing-library/react";
import { Routes, Route } from "react-router-dom";
import { renderWithProviders } from "../../test/test-utils";
import { ProductPage } from "../ProductPage";

describe("ProductPage Component", () => {
  it("renders breadcrumbs, title, highlights, specs, and reference links", async () => {
    renderWithProviders(
      <Routes>
        <Route path="/components/:categoryId/:collectionId/:productId" element={<ProductPage />} />
      </Routes>,
      { route: "/components/hardware/motors/emax-eco-ii-2207" },
    );

    // Verify breadcrumbs
    const categoryLink = screen.getByRole("link", { name: "hardware" });
    expect(categoryLink).toHaveAttribute("href", "/components/hardware");

    const collectionLink = screen.getByRole("link", { name: "Motors" });
    expect(collectionLink).toHaveAttribute("href", "/components/hardware/motors");

    // Wait for product details to load
    await waitFor(() => {
      expect(screen.getByRole("heading", { level: 1, name: "ECO II 2207" })).toBeInTheDocument();
    });

    // Manufacturer
    expect(screen.getByText("EMAX")).toBeInTheDocument();

    // Highlights & Specs
    expect(screen.getAllByText("Stator Size").length).toBeGreaterThan(0);
    expect(screen.getAllByText("2207").length).toBeGreaterThan(0);
    expect(screen.getAllByText("KV").length).toBeGreaterThan(0);
    expect(screen.getAllByText(/Weight/i).length).toBeGreaterThan(0);
    expect(screen.getAllByText(/33\.6|33\.59/).length).toBeGreaterThan(0);

    // Description
    expect(
      screen.getByText(
        "Durable and affordable 2207 brushless motor for 5-inch freestyle quadcopters.",
      ),
    ).toBeInTheDocument();

    // Reference Links
    expect(screen.getByText("Purchase")).toBeInTheDocument();
    expect(screen.getByText("https://store.example.com/emax-eco-ii")).toBeInTheDocument();
    expect(screen.getByText("Official Product Page")).toBeInTheDocument();
    expect(screen.getByText("https://emax-usa.com/products/eco-ii-2207")).toBeInTheDocument();

    // Primary Display Image & Media Gallery
    expect(screen.getByAltText("Primary Display")).toBeInTheDocument();
    expect(screen.getByText("Media Gallery")).toBeInTheDocument();
    expect(screen.getByText("Motor Dimensions & Specs")).toBeInTheDocument();
    expect(screen.getByText("EMAX ECO II Thrust Bench Test")).toBeInTheDocument();
    expect(screen.getByTitle("EMAX ECO II Thrust Bench Test")).toBeInTheDocument();
  });

  it("renders error state when product is not found", async () => {
    renderWithProviders(
      <Routes>
        <Route path="/components/:categoryId/:collectionId/:productId" element={<ProductPage />} />
      </Routes>,
      { route: "/components/hardware/motors/non-existent-motor" },
    );

    await waitFor(() => {
      expect(screen.getByText(/Motor with ID 'non-existent-motor' not found/i)).toBeInTheDocument();
    });
  });

  it("renders coming soon message for unknown collection", () => {
    renderWithProviders(
      <Routes>
        <Route path="/components/:categoryId/:collectionId/:productId" element={<ProductPage />} />
      </Routes>,
      { route: "/components/hardware/unknown-collection/some-id" },
    );

    expect(
      screen.getByText("Product page for unknown collection coming soon."),
    ).toBeInTheDocument();
  });
});
