import { describe, it, expect, vi } from "vitest";
import { screen } from "@testing-library/react";
import { renderWithProviders } from "../../test/test-utils";
import { UserCollectionsPage } from "../UserCollectionsPage";
import { Route, Routes } from "react-router-dom";

describe("UserCollectionsPage", () => {
  function renderCollections(route: string = "/users/fpv_pilot/collections/cinewhoop-favorites") {
    return renderWithProviders(
      <Routes>
        <Route path="/users/:id/collections" element={<UserCollectionsPage />} />
        <Route path="/users/:id/collections/:collectionId" element={<UserCollectionsPage />} />
        <Route path="/users/:id" element={<div>User Profile Mock</div>} />
      </Routes>,
      { route },
    );
  }

  it("redirects /users/:id/collections to user profile collections tab", async () => {
    renderCollections("/users/fpv_pilot/collections");

    // Because there is no collectionId, it redirects to /users/fpv_pilot?tab=collections
    expect(await screen.findByText("User Profile Mock")).toBeInTheDocument();
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

  it("has a rename button next to the name and allows owner to rename the collection", async () => {
    const { user } = renderCollections("/users/fpv_pilot/collections/cinewhoop-favorites");

    expect(
      await screen.findByRole("heading", { level: 1, name: "Cinewhoop Favorites" }),
    ).toBeInTheDocument();

    // Button next to name to rename collection
    const renameBtn = screen.getByRole("button", { name: /rename collection/i });
    expect(renameBtn).toBeInTheDocument();

    // Click to open rename modal
    await user.click(renameBtn);

    expect(
      await screen.findByRole("heading", { level: 3, name: /rename collection/i }),
    ).toBeInTheDocument();

    const input = screen.getByRole("textbox", { name: /collection name/i });
    expect(input).toHaveValue("Cinewhoop Favorites");

    // Change name and submit
    await user.clear(input);
    await user.type(input, "Indoor Cinewhoops Pro");

    const saveBtn = screen.getByRole("button", { name: /save name/i });
    await user.click(saveBtn);

    // Modal closes
    expect(
      screen.queryByRole("heading", { level: 3, name: /rename collection/i }),
    ).not.toBeInTheDocument();
  });

  it("allows owner to delete collection with garbage bin icon", async () => {
    vi.spyOn(window, "confirm").mockReturnValue(true);
    const { user } = renderCollections("/users/fpv_pilot/collections/cinewhoop-favorites");

    expect(
      await screen.findByRole("heading", { level: 1, name: "Cinewhoop Favorites" }),
    ).toBeInTheDocument();

    const deleteBtn = screen.getByRole("button", { name: /delete collection/i });
    expect(deleteBtn).toBeInTheDocument();

    await user.click(deleteBtn);

    expect(window.confirm).toHaveBeenCalled();
    // After deletion, navigates to user profile
    expect(await screen.findByText("User Profile Mock")).toBeInTheDocument();
  });

  it("does not show edit or delete controls for non-owners", async () => {
    renderCollections("/users/other_pilot/collections/other-col");

    // No rename or delete buttons should exist for non-owners
    expect(screen.queryByRole("button", { name: /rename collection/i })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /delete collection/i })).not.toBeInTheDocument();
  });
});
