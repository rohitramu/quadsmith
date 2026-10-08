import { describe, it, expect } from "vitest";
import { screen } from "@testing-library/react";
import { renderWithProviders } from "../../test/test-utils";
import { HomePage } from "../HomePage";

describe("HomePage Component", () => {
  it("renders welcome heading and description text", () => {
    renderWithProviders(<HomePage />);

    expect(
      screen.getByRole("heading", { level: 1, name: "Welcome to Quadsmith" })
    ).toBeInTheDocument();

    expect(
      screen.getByText(/ultimate hardware data sourcing and component browser for FPV drone builders/i)
    ).toBeInTheDocument();
  });

  it("renders Browse Hardware card linking to /components/hardware", () => {
    renderWithProviders(<HomePage />);

    const browseHardwareLink = screen.getByRole("link", { name: /browse hardware/i });
    expect(browseHardwareLink).toBeInTheDocument();
    expect(browseHardwareLink).toHaveAttribute("href", "/components/hardware");
    expect(screen.getByText(/explore our extensive catalog of fpv drone components/i)).toBeInTheDocument();
  });

  it("renders Build Planner placeholder card with coming soon message", () => {
    renderWithProviders(<HomePage />);

    expect(screen.getByText("Build Planner (Coming Soon)")).toBeInTheDocument();
    expect(
      screen.getByText(/design your dream drone and let quadsmith automatically check for component compatibility/i)
    ).toBeInTheDocument();
  });
});
