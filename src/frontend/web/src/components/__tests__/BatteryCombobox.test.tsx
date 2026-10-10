import { describe, it, expect, vi } from "vitest";
import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { renderWithProviders } from "../../test/test-utils";
import { BatteryCombobox } from "../BatteryCombobox";
import { mockBatteries, mockBattery1 } from "../../test/mocks/fixtures";

describe("BatteryCombobox Component", () => {
  it("renders with placeholder when no battery is selected", () => {
    renderWithProviders(
      <BatteryCombobox
        selectedBatteryId=""
        onSelectBattery={vi.fn()}
        batteries={mockBatteries}
        placeholder="Choose battery..."
      />,
    );

    const input = screen.getByRole("combobox", { name: /select battery/i });
    expect(input).toBeInTheDocument();
    expect(input).toHaveAttribute("placeholder", "Choose battery...");
  });

  it("renders formatted battery details when battery is selected", () => {
    renderWithProviders(
      <BatteryCombobox
        selectedBatteryId={mockBattery1.id}
        onSelectBattery={vi.fn()}
        batteries={mockBatteries}
      />,
    );

    const input = screen.getByRole("combobox", { name: /select battery/i });
    expect(input).toHaveValue("Black Series 1500mAh 6S 100C (220g, 6S, 1500mAh)");
  });

  it("opens dropdown on click and displays available batteries with specs", async () => {
    const user = userEvent.setup();
    renderWithProviders(
      <BatteryCombobox selectedBatteryId="" onSelectBattery={vi.fn()} batteries={mockBatteries} />,
    );

    const input = screen.getByRole("combobox", { name: /select battery/i });
    await user.click(input);

    expect(screen.getByRole("listbox", { name: /batteries/i })).toBeInTheDocument();
    expect(screen.getByText("Black Series 1500mAh 6S 100C")).toBeInTheDocument();
    expect(screen.getByText("6S")).toBeInTheDocument();
  });

  it("calls onSelectBattery when an option is clicked", async () => {
    const user = userEvent.setup();
    const handleSelect = vi.fn();

    renderWithProviders(
      <BatteryCombobox
        selectedBatteryId=""
        onSelectBattery={handleSelect}
        batteries={mockBatteries}
      />,
    );

    const input = screen.getByRole("combobox", { name: /select battery/i });
    await user.click(input);

    const option = screen.getByText("Black Series 1500mAh 6S 100C");
    await user.click(option);

    expect(handleSelect).toHaveBeenCalledWith(mockBattery1.id, expect.anything());
    // Dropdown closes after selection
    expect(screen.queryByRole("listbox")).not.toBeInTheDocument();
  });

  it("supports keyboard navigation with ArrowDown and Enter", async () => {
    const user = userEvent.setup();
    const handleSelect = vi.fn();

    renderWithProviders(
      <BatteryCombobox
        selectedBatteryId=""
        onSelectBattery={handleSelect}
        batteries={mockBatteries}
      />,
    );

    const input = screen.getByRole("combobox", { name: /select battery/i });
    await user.click(input);

    // Press ArrowDown to highlight first item, then Enter to select
    await user.keyboard("{ArrowDown}");
    await user.keyboard("{Enter}");

    expect(handleSelect).toHaveBeenCalledWith(mockBattery1.id, expect.anything());
  });

  it("allows typing search query and queries SearchService", async () => {
    const user = userEvent.setup();
    renderWithProviders(
      <BatteryCombobox selectedBatteryId="" onSelectBattery={vi.fn()} batteries={mockBatteries} />,
    );

    const input = screen.getByRole("combobox", { name: /select battery/i });
    await user.click(input);
    await user.type(input, "black");

    await waitFor(() => {
      expect(screen.getByText(/1 battery found/i)).toBeInTheDocument();
    });
    expect(screen.getByText("Black Series 1500mAh 6S 100C")).toBeInTheDocument();
  });
});
