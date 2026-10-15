import { describe, it, expect } from "vitest";
import { screen } from "@testing-library/react";
import { renderWithProviders } from "../../test/test-utils";
import { CollectionBadge } from "@quadsmith/shared";

describe("CollectionBadge Component", () => {
  it("renders collection name and colored icon in clean text variant by default", () => {
    renderWithProviders(<CollectionBadge collection="batteries" />);

    const badge = screen.getByText("Batteries");
    expect(badge).toBeInTheDocument();

    const container = screen.getByTestId("collection-badge");
    expect(container).toHaveClass("text-zinc-700");
    expect(container).not.toHaveClass("bg-lime-100");
    // Contains SVG icon and text span
    expect(container.querySelector("svg")).toBeInTheDocument();
  });

  it("renders styled pill badge with background and border when variant='pill'", () => {
    renderWithProviders(<CollectionBadge collection="batteries" variant="pill" />);

    const badge = screen.getByText("Batteries");
    expect(badge).toBeInTheDocument();

    const container = screen.getByTestId("collection-badge");
    expect(container).toHaveClass("bg-lime-100");
    expect(container.children.length).toBe(1);
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
    expect(container).toBeInTheDocument();
  });

  it("applies sizing classes correctly for text and pill variants", () => {
    const { unmount } = renderWithProviders(<CollectionBadge collection="motors" size="md" />);
    const containerText = screen.getByTestId("collection-badge");
    expect(containerText).toHaveClass("text-sm");
    unmount();

    renderWithProviders(<CollectionBadge collection="motors" size="md" variant="pill" />);
    const containerPill = screen.getByTestId("collection-badge");
    expect(containerPill).toHaveClass("text-sm");
    expect(containerPill).toHaveClass("px-2.5");
  });

  it("renders abbreviated chip names for FC, ESC, VTX, Props, RX, and TX", () => {
    const { unmount: u1 } = renderWithProviders(
      <CollectionBadge collection="flight-controllers" />,
    );
    expect(screen.getByText("FC")).toBeInTheDocument();
    u1();

    const { unmount: u2 } = renderWithProviders(
      <CollectionBadge collection="electronic-speed-controllers" />,
    );
    expect(screen.getByText("ESC")).toBeInTheDocument();
    u2();

    const { unmount: u3 } = renderWithProviders(
      <CollectionBadge collection="video-transmitters" />,
    );
    expect(screen.getByText("VTX")).toBeInTheDocument();
    u3();

    const { unmount: u4 } = renderWithProviders(<CollectionBadge collection="propellers" />);
    expect(screen.getByText("Props")).toBeInTheDocument();
    u4();

    const { unmount: u5 } = renderWithProviders(<CollectionBadge collection="receivers" />);
    expect(screen.getByText("RX")).toBeInTheDocument();
    u5();

    const { unmount: u6 } = renderWithProviders(<CollectionBadge collection="transmitters" />);
    expect(screen.getByText("TX")).toBeInTheDocument();
    u6();
  });

  it("converts full collection labels to chip abbreviations", () => {
    const { unmount: u1 } = renderWithProviders(
      <CollectionBadge collection="flight-controllers" label="Flight Controllers" />,
    );
    expect(screen.getByText("FC")).toBeInTheDocument();
    u1();

    const { unmount: u2 } = renderWithProviders(
      <CollectionBadge collection="video-transmitters" label="Video Transmitters" />,
    );
    expect(screen.getByText("VTX")).toBeInTheDocument();
    u2();
  });
});
