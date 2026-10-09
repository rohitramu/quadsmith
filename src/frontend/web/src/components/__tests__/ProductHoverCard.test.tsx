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
    expect(screen.getByRole("link", { name: /details/i })).toBeInTheDocument();
    // Product ID should be omitted from the hover card
    expect(screen.queryByText(mockMotor1.id)).not.toBeInTheDocument();

    // Fire mouseLeave
    act(() => {
      fireEvent.mouseLeave(trigger);
      vi.advanceTimersByTime(200);
    });

    // Popover should be unmounted
    expect(screen.queryByTestId("product-hover-card")).not.toBeInTheDocument();
  });

  it("supports rendering as a table row and displays popover when hovering anywhere on row", async () => {
    renderWithProviders(
      <table>
        <tbody>
          <ProductHoverCard
            as="tr"
            collectionId="motors"
            item={mockMotor1}
            data-testid="row-trigger"
          >
            <td>Cell 1: Name</td>
            <td data-testid="row-cell-2">Cell 2: KV</td>
          </ProductHoverCard>
        </tbody>
      </table>,
    );

    const row = screen.getByTestId("product-hover-trigger");
    expect(row.tagName).toBe("TR");

    const cell2 = screen.getByTestId("row-cell-2");

    // Hover over any cell in the row
    act(() => {
      fireEvent.mouseEnter(row, { clientX: 250, clientY: 100 });
      fireEvent.mouseMove(cell2, { clientX: 300, clientY: 100 });
      vi.advanceTimersByTime(250);
    });

    const card = screen.getByTestId("product-hover-card");
    expect(card).toBeInTheDocument();
    expect(screen.getByText("ECO II 2207")).toBeInTheDocument();
  });
});
