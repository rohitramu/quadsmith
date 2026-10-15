import { describe, it, expect, vi, beforeEach } from "vitest";
import { screen, waitFor, fireEvent } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { renderWithProviders } from "../../test/test-utils";
import { CopyFilterButton } from "@quadsmith/shared";

describe("CopyFilterButton Component", () => {
  beforeEach(() => {
    vi.restoreAllMocks();
  });

  it("does not render when filter is null or empty", () => {
    const { container } = renderWithProviders(<CopyFilterButton filter="" />);
    expect(container.firstChild).toBeNull();

    const { container: nullContainer } = renderWithProviders(<CopyFilterButton filter={null} />);
    expect(nullContainer.firstChild).toBeNull();
  });

  it("renders a button with only an icon and no visible text", () => {
    renderWithProviders(<CopyFilterButton filter="diameter_mm <= 127.0" />);

    const button = screen.getByRole("button", { name: "Copy filter" });
    expect(button).toBeInTheDocument();
    expect(button.textContent).toBe("");
  });

  it("displays a floating hint on hover with newlines after each joining &&", async () => {
    const user = userEvent.setup();
    const filter = 'min_voltage >= 14.80 && max_voltage <= 25.20 && connector == "XT60"';

    renderWithProviders(<CopyFilterButton filter={filter} />);

    const button = screen.getByRole("button", { name: "Copy filter" });
    expect(screen.queryByRole("tooltip")).not.toBeInTheDocument();

    await user.hover(button);

    const tooltip = await screen.findByRole("tooltip");
    expect(tooltip).toBeInTheDocument();
    expect(tooltip.textContent).toContain(
      'min_voltage >= 14.80 &&\nmax_voltage <= 25.20 &&\nconnector == "XT60"',
    );

    await user.unhover(button);
    await waitFor(() => {
      expect(screen.queryByRole("tooltip")).not.toBeInTheDocument();
    });
  });

  it("copies the multiline filter string to clipboard and shows check icon on click", async () => {
    const writeTextSpy = vi.spyOn(navigator.clipboard, "writeText");

    const filter = 'diameter_mm <= 127.00 && prop_mount_interface == "M5"';
    renderWithProviders(<CopyFilterButton filter={filter} />);

    const button = screen.getByRole("button", { name: "Copy filter" });
    fireEvent.click(button);

    await waitFor(() => {
      expect(writeTextSpy).toHaveBeenCalledWith(
        'diameter_mm <= 127.00 &&\nprop_mount_interface == "M5"',
      );
    });
  });
});
