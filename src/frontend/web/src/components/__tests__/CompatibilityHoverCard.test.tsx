import { describe, it, expect, vi, beforeEach } from "vitest";
import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { renderWithProviders } from "../../test/test-utils";
import { CompatibilityHoverCard } from "../CompatibilityHoverCard";

describe("CompatibilityHoverCard Component", () => {
  beforeEach(() => {
    vi.restoreAllMocks();
  });

  it("renders plain children without dotted underline or hover card when filter is null or empty", () => {
    const { container: emptyContainer } = renderWithProviders(
      <CompatibilityHoverCard filter="">Compatible only</CompatibilityHoverCard>,
    );
    expect(emptyContainer.textContent).toBe("Compatible only");
    expect(emptyContainer.querySelector(".decoration-dotted")).toBeNull();

    const { container: nullContainer } = renderWithProviders(
      <CompatibilityHoverCard filter={null}>Compatible only</CompatibilityHoverCard>,
    );
    expect(nullContainer.textContent).toBe("Compatible only");
    expect(nullContainer.querySelector(".decoration-dotted")).toBeNull();
  });

  it("renders trigger with dotted underline when filter is present", () => {
    renderWithProviders(
      <CompatibilityHoverCard filter="diameter_mm <= 127.00">
        Compatible only
      </CompatibilityHoverCard>,
    );

    const trigger = screen.getByText("Compatible only");
    expect(trigger).toBeInTheDocument();
    expect(trigger.className).toContain("underline");
    expect(trigger.className).toContain("decoration-dotted");
  });

  it("displays hover card on hover with formatted filter text and copy button", async () => {
    const user = userEvent.setup();
    const filter = 'min_voltage >= 14.80 && max_voltage <= 25.20 && connector == "XT60"';

    renderWithProviders(
      <CompatibilityHoverCard filter={filter}>Compatible only</CompatibilityHoverCard>,
    );

    const trigger = screen.getByText("Compatible only");
    expect(screen.queryByTestId("compatibility-hover-card")).not.toBeInTheDocument();

    await user.hover(trigger);

    const hoverCard = await screen.findByTestId("compatibility-hover-card");
    expect(hoverCard).toBeInTheDocument();

    // Verify header title inside hover card
    expect(screen.getByText("Compatibility Filter")).toBeInTheDocument();

    // Verify copy button is inside hover card
    const copyBtn = screen.getByTestId("copy-filter-button");
    expect(copyBtn).toBeInTheDocument();

    // Verify formatted filter text is inside pre with whitespace-pre
    const pre = hoverCard.querySelector("pre");
    expect(pre).toBeInTheDocument();
    expect(pre?.className).toContain("whitespace-pre");
    expect(pre?.textContent).toBe(
      'min_voltage >= 14.80 &&\nmax_voltage <= 25.20 &&\nconnector == "XT60"',
    );

    // Verify horizontal scrolling container is present around pre
    const scrollBox = pre?.parentElement;
    expect(scrollBox?.className).toContain("overflow-x-auto");

    // Unhover trigger
    await user.unhover(trigger);
    await waitFor(() => {
      expect(screen.queryByTestId("compatibility-hover-card")).not.toBeInTheDocument();
    });
  });

  it("copies the multiline filter to clipboard when copy button in hover card is clicked", async () => {
    const user = userEvent.setup();
    const writeTextSpy = vi.spyOn(navigator.clipboard, "writeText");
    const filter = 'diameter_mm <= 127.00 && prop_mount_interface == "M5"';

    renderWithProviders(
      <CompatibilityHoverCard filter={filter}>Compatible only</CompatibilityHoverCard>,
    );

    const trigger = screen.getByText("Compatible only");
    await user.hover(trigger);

    const copyBtn = await screen.findByTestId("copy-filter-button");
    await user.click(copyBtn);

    await waitFor(() => {
      expect(writeTextSpy).toHaveBeenCalledWith(
        'diameter_mm <= 127.00 &&\nprop_mount_interface == "M5"',
      );
    });
  });

  it("stops click propagation so it does not toggle parent checkbox", async () => {
    const user = userEvent.setup();
    const handleCheckboxChange = vi.fn();

    renderWithProviders(
      <label>
        <input type="checkbox" onChange={handleCheckboxChange} data-testid="test-checkbox" />
        <CompatibilityHoverCard filter="diameter_mm <= 127.00">
          Compatible only
        </CompatibilityHoverCard>
      </label>,
    );

    const trigger = screen.getByText("Compatible only");
    await user.hover(trigger);

    const hoverCard = await screen.findByTestId("compatibility-hover-card");
    const copyBtn = screen.getByTestId("copy-filter-button");

    // Clicking inside the hover card should NOT trigger the parent checkbox change
    await user.click(hoverCard);
    expect(handleCheckboxChange).not.toHaveBeenCalled();

    await user.click(copyBtn);
    expect(handleCheckboxChange).not.toHaveBeenCalled();
  });
});
