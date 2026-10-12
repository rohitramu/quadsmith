import { describe, it, expect } from "vitest";
import { screen, within } from "@testing-library/react";
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

    // Liked is a system collection: should have no visibility toggle group and no edit button
    expect(screen.queryByRole("group", { name: /visibility for liked/i })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /edit liked/i })).not.toBeInTheDocument();

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

  it("allows owner to toggle visibility from public to private on collections list", async () => {
    const { user } = renderCollections("/users/fpv_pilot/collections");

    // Wait for collections to load
    expect(await screen.findByText("Cinewhoop Favorites")).toBeInTheDocument();

    // Group for Cinewhoop Favorites visibility
    const group = screen.getByRole("group", { name: /visibility for cinewhoop favorites/i });
    expect(group).toBeInTheDocument();

    const publicBtn = within(group).getByRole("button", { name: /^public$/i });
    const privateBtn = within(group).getByRole("button", { name: /^private$/i });

    expect(publicBtn).toHaveAttribute("aria-pressed", "true");
    expect(privateBtn).toHaveAttribute("aria-pressed", "false");

    // Click Private
    await user.click(privateBtn);

    // Private is now pressed
    expect(privateBtn).toHaveAttribute("aria-pressed", "true");
    expect(publicBtn).toHaveAttribute("aria-pressed", "false");
  });

  it("allows owner to toggle visibility from public to private in single collection view", async () => {
    const { user } = renderCollections("/users/fpv_pilot/collections/cinewhoop-favorites");

    expect(
      await screen.findByRole("heading", { level: 1, name: "Cinewhoop Favorites" }),
    ).toBeInTheDocument();

    const group = screen.getByRole("group", { name: /collection visibility/i });
    expect(group).toBeInTheDocument();

    const privateBtn = within(group).getByRole("button", { name: /^private$/i });
    await user.click(privateBtn);

    expect(privateBtn).toHaveAttribute("aria-pressed", "true");
  });

  it("allows owner to edit collection details and toggle visibility via modal", async () => {
    const { user } = renderCollections("/users/fpv_pilot/collections/cinewhoop-favorites");

    expect(
      await screen.findByRole("heading", { level: 1, name: "Cinewhoop Favorites" }),
    ).toBeInTheDocument();

    const editBtn = screen.getByRole("button", { name: /edit collection/i });
    await user.click(editBtn);

    // Modal opens
    expect(
      await screen.findByRole("heading", { level: 3, name: /edit collection/i }),
    ).toBeInTheDocument();

    // Switch visibility in modal
    const privateOption = screen.getByRole("button", { name: /private only you can see it/i });
    await user.click(privateOption);

    // Save changes
    const saveBtn = screen.getByRole("button", { name: /save changes/i });
    await user.click(saveBtn);

    // Modal closes
    expect(
      screen.queryByRole("heading", { level: 3, name: /edit collection/i }),
    ).not.toBeInTheDocument();
  });

  it("shows read-only visibility badge for non-owners", async () => {
    // Render collections of another user where logged in user is not the owner
    renderCollections("/users/other_pilot/collections");

    // No visibility button groups should exist
    expect(screen.queryByRole("group", { name: /visibility/i })).not.toBeInTheDocument();
  });
});
