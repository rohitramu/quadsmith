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

  it("positions hover card next to the trigger element when placement is 'side'", async () => {
    renderWithProviders(
      <ProductHoverCard collectionId="motors" item={mockMotor1} placement="side">
        <span data-testid="side-trigger">Side Hover Target</span>
      </ProductHoverCard>,
    );

    const trigger = screen.getByTestId("product-hover-trigger");
    vi.spyOn(trigger, "getBoundingClientRect").mockReturnValue({
      left: 100,
      right: 350,
      top: 200,
      bottom: 236,
      width: 250,
      height: 36,
      x: 100,
      y: 200,
      toJSON: () => {},
    });

    act(() => {
      fireEvent.mouseEnter(trigger);
      vi.advanceTimersByTime(250);
    });

    const card = screen.getByTestId("product-hover-card");
    expect(card).toBeInTheDocument();
    // In placement="side", left should be right + 8 = 358px
    expect(card.style.left).toBe("358px");
    expect(card.style.top).toBe("200px");
    expect(card.style.transform).toBe("none");
  });

  it("positions hover card to the left when placement is 'left'", async () => {
    renderWithProviders(
      <ProductHoverCard collectionId="motors" item={mockMotor1} placement="left">
        <span data-testid="left-trigger">Left Hover Target</span>
      </ProductHoverCard>,
    );

    const trigger = screen.getByTestId("product-hover-trigger");
    vi.spyOn(trigger, "getBoundingClientRect").mockReturnValue({
      left: 500,
      right: 750,
      top: 200,
      bottom: 236,
      width: 250,
      height: 36,
      x: 500,
      y: 200,
      toJSON: () => {},
    });

    act(() => {
      fireEvent.mouseEnter(trigger);
      vi.advanceTimersByTime(250);
    });

    const card = screen.getByTestId("product-hover-card");
    expect(card).toBeInTheDocument();
    // In placement="left", left should be 500 - 320 - 8 = 172px
    expect(card.style.left).toBe("172px");
    expect(card.style.top).toBe("200px");
    expect(card.style.transform).toBe("none");
  });

  it("positions hover card to the right when placement is 'right'", async () => {
    renderWithProviders(
      <ProductHoverCard collectionId="motors" item={mockMotor1} placement="right">
        <span data-testid="right-trigger">Right Hover Target</span>
      </ProductHoverCard>,
    );

    const trigger = screen.getByTestId("product-hover-trigger");
    vi.spyOn(trigger, "getBoundingClientRect").mockReturnValue({
      left: 200,
      right: 450,
      top: 200,
      bottom: 236,
      width: 250,
      height: 36,
      x: 200,
      y: 200,
      toJSON: () => {},
    });

    act(() => {
      fireEvent.mouseEnter(trigger);
      vi.advanceTimersByTime(250);
    });

    const card = screen.getByTestId("product-hover-card");
    expect(card).toBeInTheDocument();
    // In placement="right", left should be 450 + 8 = 458px
    expect(card.style.left).toBe("458px");
    expect(card.style.top).toBe("200px");
    expect(card.style.transform).toBe("none");
  });
});
