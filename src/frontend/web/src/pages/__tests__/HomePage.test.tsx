import { describe, it, expect } from "vitest";
import { screen } from "@testing-library/react";
import { renderWithProviders } from "../../test/test-utils";
import { HomePage } from "../HomePage";

describe("HomePage Component", () => {
  it("renders welcome heading and description text", () => {
    renderWithProviders(<HomePage />);

    expect(
      screen.getByRole("heading", { level: 1, name: "Welcome to Quadsmith" }),
    ).toBeInTheDocument();

    expect(
      screen.getByText(
        /ultimate hardware data sourcing and component browser for FPV drone builders/i,
      ),
    ).toBeInTheDocument();
  });

  it("renders Browse Hardware link pointing to /components/hardware", () => {
    renderWithProviders(<HomePage />);

    const browseHardwareLink = screen.getByRole("link", {
      name: /browse hardware/i,
    });
    expect(browseHardwareLink).toBeInTheDocument();
    expect(browseHardwareLink).toHaveAttribute("href", "/components/hardware");
  });

  it("renders The Forge link pointing to /builds/new with hammer icon in fiery orange", () => {
    renderWithProviders(<HomePage />);

    const forgeLink = screen.getByRole("link", {
      name: /the forge/i,
    });
    expect(forgeLink).toBeInTheDocument();
    expect(forgeLink).toHaveAttribute("href", "/builds/new");
    const icon = forgeLink.querySelector("svg");
    expect(icon).toBeInTheDocument();
    expect(icon).toHaveClass("text-[#FF6D00]");
  });

  it("renders Community & Curated Builds feed heading and loaded build cards", async () => {
    renderWithProviders(<HomePage />);

    expect(
      screen.getByRole("heading", {
        level: 2,
        name: /community & curated builds/i,
      }),
    ).toBeInTheDocument();

    // Verify build cards are rendered from mock data
    expect(await screen.findByText("Bando Basher 5 inch")).toBeInTheDocument();
    expect(await screen.findByText("Long Range Explorer 7 inch")).toBeInTheDocument();

    // Verify link to build profile
    const bandoProfileLinks = screen.getAllByRole("link", {
      name: /bando basher 5 inch/i,
    });
    expect(bandoProfileLinks.length).toBeGreaterThan(0);
    expect(bandoProfileLinks[0]).toHaveAttribute("href", "/builds/bando-basher-5-inch");
  });

  it("displays end of feed caught up message when all builds have loaded", async () => {
    renderWithProviders(<HomePage />);

    expect(await screen.findByText(/you've caught up with all builds!/i)).toBeInTheDocument();
  });
});
