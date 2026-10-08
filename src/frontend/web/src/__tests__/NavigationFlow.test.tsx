import { describe, it, expect } from "vitest";
import { screen, waitFor } from "@testing-library/react";
import { renderApp } from "../test/test-utils";

describe("Full Application Navigation Flow", () => {
  it("navigates seamlessly from Home to Category, Collection Table, Product Details, and back via Breadcrumbs and Navbar", async () => {
    const { user } = renderApp("/");

    // 1. Initial State: Home Page
    expect(
      screen.getByRole("heading", { level: 1, name: "Welcome to Quadsmith" }),
    ).toBeInTheDocument();

    // 2. Click "Browse Hardware" card to navigate to /components/hardware
    const browseCard = screen.getByRole("link", { name: /browse hardware/i });
    await user.click(browseCard);

    // 3. Verify on Category Page (/components/hardware)
    expect(await screen.findByRole("heading", { level: 1, name: "hardware" })).toBeInTheDocument();

    // 4. Click "Motors" card to navigate to /components/hardware/motors
    const motorsCard = screen.getByRole("link", { name: /browse all motors/i });
    await user.click(motorsCard);

    // 5. Verify on Collection Page (/components/hardware/motors)
    expect(await screen.findByRole("heading", { level: 1, name: "Motors" })).toBeInTheDocument();

    // Wait for table to load
    await waitFor(() => {
      expect(screen.getByText("ECO II 2207")).toBeInTheDocument();
    });

    // 6. Click table row for "ECO II 2207" to navigate to /components/hardware/motors/emax-eco-ii-2207
    const rowLink = screen.getByRole("link", { name: /view eco ii 2207/i });
    await user.click(rowLink);

    // 7. Verify on Product Details Page
    expect(
      await screen.findByRole("heading", { level: 1, name: "ECO II 2207" }),
    ).toBeInTheDocument();
    expect(screen.getByText("EMAX")).toBeInTheDocument();
    expect(screen.getAllByText("1900").length).toBeGreaterThan(0);

    // 8. Click "Motors" in breadcrumbs to navigate back to collection table
    const motorsBreadcrumb = screen.getByRole("link", { name: "Motors" });
    await user.click(motorsBreadcrumb);

    expect(await screen.findByRole("heading", { level: 1, name: "Motors" })).toBeInTheDocument();

    // 9. Click "Hardware" in sidebar to navigate back to hardware category
    const hardwareSidebarLink = screen.getByRole("link", { name: "Hardware" });
    await user.click(hardwareSidebarLink);

    expect(await screen.findByRole("heading", { level: 1, name: "hardware" })).toBeInTheDocument();

    // 10. Click Quadsmith Logo to navigate back to Home Page
    const logoLink = screen.getByAltText("Quadsmith").closest("a");
    expect(logoLink).toBeInTheDocument();
    await user.click(logoLink!);

    expect(
      await screen.findByRole("heading", {
        level: 1,
        name: "Welcome to Quadsmith",
      }),
    ).toBeInTheDocument();
  });
});
