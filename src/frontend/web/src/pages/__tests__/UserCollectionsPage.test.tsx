import { describe, it, expect } from "vitest";
import { screen } from "@testing-library/react";
import { renderWithProviders } from "../../test/test-utils";
import { UserCollectionsPage } from "../UserCollectionsPage";
import { Route, Routes } from "react-router-dom";

describe("UserCollectionsPage", () => {
  function renderCollections(route: string = "/users/fpv_pilot/collections") {
    return renderWithProviders(
      <Routes>
        <Route path="/users/:id/collections" element={<UserCollectionsPage />} />
        <Route path="/users/:id/collections/:collectionId" element={<UserCollectionsPage />} />
      </Routes>,
      { route },
    );
  }

  it("renders collections list for a user", async () => {
    const { user } = renderCollections("/users/fpv_pilot/collections");

    // Header and collections list
    expect(await screen.findByText(/Collections by @fpv_pilot/i)).toBeInTheDocument();
    expect(screen.getByText("Liked")).toBeInTheDocument();
    expect(screen.getByText("Cinewhoop Favorites")).toBeInTheDocument();

    // Create collection button is present for profile owner
    const newBtn = screen.getByRole("button", { name: /create collection/i });
    expect(newBtn).toBeInTheDocument();

    // Click to open create form
    await user.click(newBtn);
    expect(
      await screen.findByRole("heading", { level: 3, name: /new custom collection/i }),
    ).toBeInTheDocument();
  });

  it("renders items inside a specific collection view", async () => {
    renderCollections("/users/fpv_pilot/collections/cinewhoop-favorites");

    // Single collection view title and items
    expect(
      await screen.findByRole("heading", { level: 1, name: "Cinewhoop Favorites" }),
    ).toBeInTheDocument();
    expect(screen.getByText(/Curated parts for indoor sub-250g filming/i)).toBeInTheDocument();

    // Collection item
    expect(await screen.findByText("Bando Basher 5 inch Freestyle")).toBeInTheDocument();
    expect(screen.getByText("Build")).toBeInTheDocument();
  });
});
