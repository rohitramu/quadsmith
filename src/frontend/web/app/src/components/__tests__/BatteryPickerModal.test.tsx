import { describe, it, expect, vi } from "vitest";
import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { renderWithProviders } from "../../test/test-utils";
import { BatteryPickerModal } from "../BatteryPickerModal";
import { mockBattery1, mockBatteryIncompatible } from "../../test/mocks/fixtures";

describe("BatteryPickerModal Component", () => {
  const defaultLimits = {
    minVoltage: 14.8,
    maxVoltage: 25.2,
    maxCurrentA: 39.4,
    connector: "XT60",
    compatibleConnectors: ["XT60"],
  };

  it("does not render when isOpen is false", () => {
    renderWithProviders(
      <BatteryPickerModal
        isOpen={false}
        onClose={vi.fn()}
        selectedBatteryId=""
        onSelectBattery={vi.fn()}
        limits={defaultLimits}
        compatibleBatteries={[mockBattery1]}
        allBatteries={[mockBattery1, mockBatteryIncompatible]}
      />,
    );

    expect(screen.queryByText("Choose Compatible Battery")).not.toBeInTheDocument();
  });

  it("renders with 'Compatible only' checkbox checked by default and shows only compatible batteries", () => {
    renderWithProviders(
      <BatteryPickerModal
        isOpen={true}
        onClose={vi.fn()}
        selectedBatteryId=""
        onSelectBattery={vi.fn()}
        limits={defaultLimits}
        compatibleBatteries={[mockBattery1]}
        allBatteries={[mockBattery1, mockBatteryIncompatible]}
      />,
    );

    expect(screen.getByText("Choose Compatible Battery")).toBeInTheDocument();

    const checkbox = screen.getByRole("checkbox", { name: /compatible only/i });
    expect(checkbox).toBeInTheDocument();
    expect(checkbox).toBeChecked();

    // Compatible battery should be present
    expect(screen.getByText("Black Series 1500mAh 6S 100C")).toBeInTheDocument();
    // Incompatible battery should NOT be present when compatible only is checked
    expect(screen.queryByText("Tattu 450mAh 1S 95C")).not.toBeInTheDocument();
  });

  it("shows incompatible batteries with Forge styling and incompatibility badge when 'Compatible only' is unchecked", async () => {
    const user = userEvent.setup();

    renderWithProviders(
      <BatteryPickerModal
        isOpen={true}
        onClose={vi.fn()}
        selectedBatteryId=""
        onSelectBattery={vi.fn()}
        limits={defaultLimits}
        compatibleBatteries={[mockBattery1]}
        allBatteries={[mockBattery1, mockBatteryIncompatible]}
      />,
    );

    const checkbox = screen.getByRole("checkbox", { name: /compatible only/i });
    await user.click(checkbox);
    expect(checkbox).not.toBeChecked();

    // Both compatible and incompatible batteries should now be rendered
    expect(screen.getByText("Black Series 1500mAh 6S 100C")).toBeInTheDocument();
    expect(screen.getByText("Tattu 450mAh 1S 95C")).toBeInTheDocument();

    // Incompatible battery badge should be displayed
    const badge = screen.getByText(/Incompatible/i);
    expect(badge).toBeInTheDocument();
    expect(badge.className).toContain("text-amber-700");

    // Incompatible card should have the Forge styling classes
    const incompatibleCard = badge.closest(".group");
    expect(incompatibleCard?.className).toContain("opacity-60");
    expect(incompatibleCard?.className).toContain("saturate-50");
  });

  it("calls onSelectBattery and onClose when a battery is clicked", async () => {
    const user = userEvent.setup();
    const handleSelect = vi.fn();
    const handleClose = vi.fn();

    renderWithProviders(
      <BatteryPickerModal
        isOpen={true}
        onClose={handleClose}
        selectedBatteryId=""
        onSelectBattery={handleSelect}
        limits={defaultLimits}
        compatibleBatteries={[mockBattery1]}
        allBatteries={[mockBattery1, mockBatteryIncompatible]}
      />,
    );

    const option = screen.getByText("Black Series 1500mAh 6S 100C");
    await user.click(option);

    expect(handleSelect).toHaveBeenCalledWith(mockBattery1.id, mockBattery1);
    expect(handleClose).toHaveBeenCalled();
  });

  it("filters batteries by search query", async () => {
    const user = userEvent.setup();

    renderWithProviders(
      <BatteryPickerModal
        isOpen={true}
        onClose={vi.fn()}
        selectedBatteryId=""
        onSelectBattery={vi.fn()}
        limits={defaultLimits}
        compatibleBatteries={[mockBattery1]}
        allBatteries={[mockBattery1, mockBatteryIncompatible]}
      />,
    );

    const searchInput = screen.getByPlaceholderText(/search by name, brand, connector/i);
    await user.type(searchInput, "nonexistent");

    expect(screen.getByText(/no compatible batteries found/i)).toBeInTheDocument();
  });
});
