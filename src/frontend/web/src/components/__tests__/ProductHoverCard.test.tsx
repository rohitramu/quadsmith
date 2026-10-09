import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { screen, act, fireEvent } from "@testing-library/react";
import { renderWithProviders } from "../../test/test-utils";
import { ProductHoverCard } from "../ProductHoverCard";
import { mockMotor1 } from "../../test/mocks/fixtures";

describe("ProductHoverCard", () => {
  beforeEach(() => {
    vi.useFakeTimers();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it("renders trigger and displays hover popover on mouse enter after delay", async () => {
    renderWithProviders(
      <ProductHoverCard collectionId="motors" item={mockMotor1}>
        <span data-testid="motor-trigger">EMAX ECO II 2207</span>
      </ProductHoverCard>,
    );

    const trigger = screen.getByTestId("motor-trigger");
    expect(trigger).toBeInTheDocument();

    // Hover card should not be visible before hover
    expect(screen.queryByTestId("product-hover-card")).not.toBeInTheDocument();

    // Fire mouseEnter
    act(() => {
      fireEvent.mouseEnter(trigger);
      vi.advanceTimersByTime(250);
    });

    // Popover should now be mounted
    const card = screen.getByTestId("product-hover-card");
    expect(card).toBeInTheDocument();
    expect(screen.getByText("EMAX")).toBeInTheDocument();
    expect(screen.getByText("ECO II 2207")).toBeInTheDocument();

    // Fire mouseLeave
    act(() => {
      fireEvent.mouseLeave(trigger);
      vi.advanceTimersByTime(200);
    });

    // Popover should be unmounted
    expect(screen.queryByTestId("product-hover-card")).not.toBeInTheDocument();
  });
});
