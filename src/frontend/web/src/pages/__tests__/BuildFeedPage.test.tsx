import { describe, it, expect } from "vitest";
import { screen } from "@testing-library/react";
import { renderWithProviders } from "../../test/test-utils";
import { BuildFeedPage } from "../BuildFeedPage";

describe("BuildFeedPage Component", () => {
  it("renders Build Feed heading and description", () => {
    renderWithProviders(<BuildFeedPage />, { route: "/builds" });

    expect(screen.getByRole("heading", { level: 1, name: "Build Feed" })).toBeInTheDocument();

    expect(
      screen.getByText(/explore verified community and curated quadcopter builds/i),
    ).toBeInTheDocument();

    expect(screen.queryByRole("link", { name: /new build in the forge/i })).not.toBeInTheDocument();
  });

  it("renders build cards from mock data and links to build profiles", async () => {
    renderWithProviders(<BuildFeedPage />, { route: "/builds" });

    // Verify build cards are rendered
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
    renderWithProviders(<BuildFeedPage />, { route: "/builds" });

    expect(await screen.findByText(/you've caught up with all builds!/i)).toBeInTheDocument();
  });
});
