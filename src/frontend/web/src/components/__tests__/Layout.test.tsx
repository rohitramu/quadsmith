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

  it("renders sidebar navigation links with hardware collections permanently expanded", () => {
    renderWithProviders(<Layout />, { route: "/" });

    // Builds link is removed altogether from sidebar and top navbar
    expect(screen.queryByRole("link", { name: "Builds" })).not.toBeInTheDocument();

    const wizardLink = screen.getByRole("link", { name: /The Forge/i });
    expect(wizardLink).toHaveAttribute("href", "/builds/new");
    const wizardIcon = wizardLink.querySelector("svg");
    expect(wizardIcon).toBeInTheDocument();
    expect(wizardIcon).toHaveClass("text-[#FF6D00]");
    expect(screen.queryByText("New")).not.toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Hardware" })).toHaveAttribute(
      "href",
      "/components/hardware",
    );
    expect(screen.getByRole("link", { name: "Software" })).toHaveAttribute(
      "href",
      "/components/software",
    );
    expect(screen.getByRole("link", { name: "Gear" })).toHaveAttribute("href", "/components/gear");

    // No expand/collapse toggle buttons
    expect(screen.queryByRole("button", { name: /Expand Hardware menu/i })).not.toBeInTheDocument();
    expect(
      screen.queryByRole("button", { name: /Collapse Hardware menu/i }),
    ).not.toBeInTheDocument();

    // All hardware collections are permanently visible on initial render with icons
    for (const collection of HARDWARE_COLLECTIONS) {
      const link = screen.getByRole("link", { name: collection.name });
      expect(link).toBeInTheDocument();
      expect(link).toHaveAttribute("href", `/components/hardware/${collection.id}`);
      expect(link.querySelector("svg")).toBeInTheDocument();
      expect(link.querySelector(".rounded-full")).not.toBeInTheDocument();
    }
  });
});
