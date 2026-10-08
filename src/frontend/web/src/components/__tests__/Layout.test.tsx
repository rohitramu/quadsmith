import { describe, it, expect, beforeEach } from "vitest";
import { screen } from "@testing-library/react";
import { renderWithProviders } from "../../test/test-utils";
import { Layout } from "../Layout";
import { HARDWARE_COLLECTIONS } from "../../lib/hardwareCollections";

describe("Layout Component", () => {
  beforeEach(() => {
    localStorage.clear();
    document.documentElement.classList.remove("dark");
  });

  it("renders top navbar with logo, search input, and theme toggle", () => {
    renderWithProviders(<Layout />, { route: "/" });

    // Logo
    const logoImg = screen.getByAltText("Quadsmith");
    expect(logoImg).toBeInTheDocument();
    const logoLink = logoImg.closest("a");
    expect(logoLink).toHaveAttribute("href", "/");

    // Search Bar
    const searchInput = screen.getByPlaceholderText("Search components...");
    expect(searchInput).toBeInTheDocument();

    // Theme Toggle Button
    const themeBtn = screen.getByRole("button", { name: "Toggle theme" });
    expect(themeBtn).toBeInTheDocument();
  });

  it("toggles light and dark themes correctly and persists to localStorage", async () => {
    const { user } = renderWithProviders(<Layout />, { route: "/" });

    // Initial state is dark by default
    expect(document.documentElement.classList.contains("dark")).toBe(true);
    expect(localStorage.getItem("theme")).toBe("dark");

    const themeButton = screen.getByRole("button", { name: "Toggle theme" });

    // Toggle to light mode
    await user.click(themeButton);
    expect(document.documentElement.classList.contains("dark")).toBe(false);
    expect(localStorage.getItem("theme")).toBe("light");

    // Toggle back to dark mode
    await user.click(themeButton);
    expect(document.documentElement.classList.contains("dark")).toBe(true);
    expect(localStorage.getItem("theme")).toBe("dark");
  });

  it("renders sidebar navigation links", () => {
    renderWithProviders(<Layout />, { route: "/" });

    // Category links
    expect(screen.getByRole("link", { name: "Hardware" })).toHaveAttribute(
      "href",
      "/components/hardware",
    );
    expect(screen.getByRole("link", { name: "Software" })).toHaveAttribute(
      "href",
      "/components/software",
    );
    expect(screen.getByRole("link", { name: "Gear" })).toHaveAttribute(
      "href",
      "/components/gear",
    );
  });

  it("expands and collapses hardware collection sub-links on toggle button click", async () => {
    const { user } = renderWithProviders(<Layout />, { route: "/" });

    // Initially on "/", hardware submenu is collapsed
    expect(
      screen.queryByRole("link", { name: "Motors" }),
    ).not.toBeInTheDocument();

    const chevronButton = screen.getByRole("button", {
      name: "Expand Hardware menu",
    });

    // Click to expand
    await user.click(chevronButton);

    // All hardware collections should now be visible
    for (const collection of HARDWARE_COLLECTIONS) {
      const link = screen.getByRole("link", { name: collection.name });
      expect(link).toBeInTheDocument();
      expect(link).toHaveAttribute(
        "href",
        `/components/hardware/${collection.id}`,
      );
    }

    // Click again to collapse
    await user.click(chevronButton);
    expect(
      screen.queryByRole("link", { name: "Motors" }),
    ).not.toBeInTheDocument();
  });

  it("auto-expands hardware submenu when navigating to a hardware route", () => {
    renderWithProviders(<Layout />, { route: "/components/hardware" });

    // Submenu should be auto-expanded
    expect(screen.getByRole("link", { name: "Motors" })).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Frames" })).toBeInTheDocument();
  });
});
