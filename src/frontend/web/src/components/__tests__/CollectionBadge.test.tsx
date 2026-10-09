import { describe, it, expect } from "vitest";
import { screen } from "@testing-library/react";
import { renderWithProviders } from "../../test/test-utils";
import { CollectionBadge } from "../CollectionBadge";

describe("CollectionBadge Component", () => {
  it("renders collection name and styled badge with dot", () => {
    renderWithProviders(<CollectionBadge collection="batteries" withDot />);

    const badge = screen.getByText("Batteries");
    expect(badge).toBeInTheDocument();

    const container = screen.getByTestId("collection-badge");
    expect(container).toHaveClass("bg-emerald-100");
  });

  it("supports custom label", () => {
    renderWithProviders(<CollectionBadge collection="motors" label="Custom Motor Label" />);

    expect(screen.getByText("Custom Motor Label")).toBeInTheDocument();
  });

  it("renders link when 'to' prop is provided", () => {
    renderWithProviders(<CollectionBadge collection="cameras" to="/components/hardware/cameras" />);

    const link = screen.getByRole("link");
    expect(link).toHaveAttribute("href", "/components/hardware/cameras");
    expect(screen.getByText("Cameras")).toBeInTheDocument();
  });

  it("renders fallback styling for unknown collections", () => {
    renderWithProviders(<CollectionBadge collection="unregistered" label="Special" />);

    const badge = screen.getByText("Special");
    expect(badge).toBeInTheDocument();
    const container = screen.getByTestId("collection-badge");
    expect(container).toHaveClass("bg-slate-100");
  });
});
