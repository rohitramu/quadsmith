import { describe, it, expect, beforeEach } from "vitest";
import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { renderWithProviders } from "../../test/test-utils";
import { Layout } from "../Layout";
import { HARDWARE_COLLECTIONS } from "../../lib/hardwareCollections";

describe("Layout Component", () => {
  beforeEach(() => {
    localStorage.clear();
  });

  it("renders top navbar with logo and search input", () => {
    renderWithProviders(<Layout />, { route: "/" });

    // Logo
    const logoImg = screen.getByAltText("Quadsmith");
    expect(logoImg).toBeInTheDocument();
    const logoLink = logoImg.closest("a");
    expect(logoLink).toHaveAttribute("href", "/");

    // Search Bar
    const searchInput = screen.getByPlaceholderText(/search/i);
    expect(searchInput).toBeInTheDocument();
  });

  it("focuses search bar on '/' keyboard shortcut", async () => {
    const user = userEvent.setup();
    renderWithProviders(<Layout />, { route: "/" });

    const searchInput = screen.getByPlaceholderText(/search/i);
    expect(searchInput).not.toHaveFocus();

    await user.keyboard("/");
    expect(searchInput).toHaveFocus();
  });

  it("opens mobile navigation drawer on hamburger menu click and closes on close button or backdrop click", async () => {
    const { user } = renderWithProviders(<Layout />, { route: "/" });

    const menuButton = screen.getByRole("button", { name: "Open navigation menu" });
    expect(menuButton).toBeInTheDocument();

    // Initially drawer dialog is not present
    expect(screen.queryByRole("dialog", { name: "Navigation drawer" })).not.toBeInTheDocument();

    // Click hamburger to open drawer
    await user.click(menuButton);
    const drawer = screen.getByRole("dialog", { name: "Navigation drawer" });
    expect(drawer).toBeInTheDocument();

    // Close button inside drawer closes it
    const closeBtn = screen.getByRole("button", { name: "Close navigation menu" });
    await user.click(closeBtn);
    expect(screen.queryByRole("dialog", { name: "Navigation drawer" })).not.toBeInTheDocument();

    // Reopen and close via Escape
    await user.click(menuButton);
    expect(screen.getByRole("dialog", { name: "Navigation drawer" })).toBeInTheDocument();
    await user.keyboard("{Escape}");
    expect(screen.queryByRole("dialog", { name: "Navigation drawer" })).not.toBeInTheDocument();
  });

  it("renders sidebar navigation links with hardware collections permanently expanded", () => {
    renderWithProviders(<Layout />, { route: "/" });

    // Generic "Builds" link is removed in favor of "Build Feed"
    expect(screen.queryByRole("link", { name: /^Builds$/ })).not.toBeInTheDocument();

    const feedLink = screen.getByRole("link", { name: /Build Feed/i });
    expect(feedLink).toHaveAttribute("href", "/builds");
    const feedIcon = feedLink.querySelector("svg");
    expect(feedIcon).toBeInTheDocument();
    expect(feedIcon).toHaveClass("text-blue-400");

    const wizardLink = screen.getByRole("link", { name: /The Forge/i });
    expect(wizardLink).toHaveAttribute("href", "/builds/new");
    const wizardIcon = wizardLink.querySelector("svg");
    expect(wizardIcon).toBeInTheDocument();
    expect(wizardIcon).toHaveClass("text-[#FF6D00]");

    // Verify Build Feed is positioned above The Forge in Discover section
    const discoverNav = wizardLink.closest("ul")!;
    const navItems = discoverNav.querySelectorAll("li a");
    expect(navItems[0]).toHaveAttribute("href", "/builds");
    expect(navItems[1]).toHaveAttribute("href", "/builds/new");
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
      const links = screen.getAllByRole("link", { name: collection.name });
      expect(links.length).toBeGreaterThanOrEqual(1);
      const link = links[0];
      expect(link).toHaveAttribute("href", `/components/hardware/${collection.id}`);
      expect(link.querySelector("svg")).toBeInTheDocument();
      expect(link.querySelector(".rounded-full")).not.toBeInTheDocument();
    }
  });

  it("styles The Forge link with blue active text when selected, while keeping the fiery orange icon", () => {
    renderWithProviders(<Layout />, { route: "/builds/new" });

    const forgeLink = screen.getByRole("link", { name: /The Forge/i });
    expect(forgeLink).toHaveClass("text-blue-400");
    const forgeIcon = forgeLink.querySelector("svg");
    expect(forgeIcon).toHaveClass("text-[#FF6D00]");
  });

  it("styles Build Feed link with blue active text when selected, while keeping blue icon", () => {
    renderWithProviders(<Layout />, { route: "/builds" });

    const feedLink = screen.getByRole("link", { name: /Build Feed/i });
    expect(feedLink).toHaveClass("text-blue-400");
    const feedIcon = feedLink.querySelector("svg");
    expect(feedIcon).toHaveClass("text-blue-400");
  });

  it("shows account dropdown menu only on click and provides distinct Public Profile button", async () => {
    const { user } = renderWithProviders(<Layout />, { route: "/" });

    // Wait for auth to resolve and find the account settings avatar button
    const accountBtn = await screen.findByRole("button", { name: /Account Settings/i });
    expect(accountBtn).toBeInTheDocument();

    // Menu is not visible initially
    expect(screen.queryByRole("menu", { name: /account menu/i })).not.toBeInTheDocument();

    // Hover over avatar does NOT open the menu
    await user.hover(accountBtn);
    expect(screen.queryByRole("menu", { name: /account menu/i })).not.toBeInTheDocument();

    // Click on avatar opens menu
    await user.click(accountBtn);
    const menu = await screen.findByRole("menu", { name: /account menu/i });
    expect(menu).toBeInTheDocument();

    // The username is rendered as a badge pill and is NOT clickable
    expect(screen.getByTestId("user-badge")).toHaveTextContent("@fpv_pilot");
    const usernameProfileLink = screen.queryByRole("link", { name: /@fpv_pilot/i });
    expect(usernameProfileLink).not.toBeInTheDocument();

    // Separate "Public Profile" button is rendered above "Account Settings"
    const publicProfileLink = screen.getByRole("menuitem", { name: /public profile/i });
    expect(publicProfileLink).toBeInTheDocument();
    expect(publicProfileLink).toHaveAttribute("href", "/users/fpv_pilot");

    const settingsLink = screen.getByRole("menuitem", { name: /account settings/i });
    expect(settingsLink).toBeInTheDocument();
    expect(settingsLink).toHaveAttribute("href", "/account");

    const logoutBtn = screen.getByRole("menuitem", { name: /log out/i });
    expect(logoutBtn).toBeInTheDocument();
    expect(logoutBtn).toHaveClass("text-rose-400");

    // Clicking avatar again toggles menu closed
    await user.click(accountBtn);
    expect(screen.queryByRole("menu", { name: /account menu/i })).not.toBeInTheDocument();
  });
});
