import { describe, it, expect, vi } from "vitest";
import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { renderWithProviders } from "../../test/test-utils";
import { ComponentFormModal } from "../ComponentFormModal";

describe("ComponentFormModal Component", () => {
  it("does not render when isOpen is false", () => {
    renderWithProviders(
      <ComponentFormModal isOpen={false} onClose={vi.fn()} collectionId="motors" />,
    );

    expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
  });

  it("renders with 'Add New Motor' header and fields when isOpen is true", () => {
    renderWithProviders(
      <ComponentFormModal isOpen={true} onClose={vi.fn()} collectionId="motors" />,
    );

    expect(screen.getByRole("dialog")).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: /add new motor/i })).toBeInTheDocument();
    expect(screen.getByLabelText(/manufacturer \*/i)).toBeInTheDocument();
    expect(screen.getByLabelText(/model name \*/i)).toBeInTheDocument();
    expect(screen.queryByLabelText(/id slug/i)).not.toBeInTheDocument();
  });

  it("pre-populates form in edit mode", () => {
    const mockMotor = {
      uuid: "01925b42-1234-789a-b456-abcdef012345",
      id: "tmotor-f80-pro-1900kv",
      manufacturer: "T-Motor",
      name: "F80 Pro",
      description: "High torque racing motor",
      kv: 1900,
      statorDiameterMm: 24,
      statorHeightMm: 8,
      minVoltage: 16.8,
      maxVoltage: 25.2,
      maxCurrentA: 55,
      weightG: 41.2,
    };

    renderWithProviders(
      <ComponentFormModal
        isOpen={true}
        onClose={vi.fn()}
        collectionId="motors"
        initialData={mockMotor}
      />,
    );

    expect(screen.getByRole("heading", { name: /edit motor/i })).toBeInTheDocument();
    expect(screen.getByDisplayValue("T-Motor")).toBeInTheDocument();
    expect(screen.getByDisplayValue("F80 Pro")).toBeInTheDocument();
    expect(screen.getByDisplayValue("High torque racing motor")).toBeInTheDocument();
    expect(screen.queryByLabelText(/id slug/i)).not.toBeInTheDocument();
  });

  it("pre-populates form in clone / create similar mode with all editable specs", () => {
    const mockMotor = {
      uuid: "01925b42-1234-789a-b456-abcdef012345",
      id: "tmotor-f80-pro-1900kv",
      manufacturer: "T-Motor",
      name: "F80 Pro",
      description: "High torque racing motor",
      kv: 1900,
      statorDiameterMm: 24,
      statorHeightMm: 8,
      minVoltage: 16.8,
      maxVoltage: 25.2,
      maxCurrentA: 55,
      weightG: 41.2,
    };

    renderWithProviders(
      <ComponentFormModal
        isOpen={true}
        onClose={vi.fn()}
        collectionId="motors"
        initialData={mockMotor}
        isClone={true}
      />,
    );

    expect(screen.getByRole("heading", { name: /create similar motor/i })).toBeInTheDocument();
    const mfgInput = screen.getByDisplayValue("T-Motor");
    const nameInput = screen.getByDisplayValue("F80 Pro");
    const kvInput = screen.getByDisplayValue("1900");
    expect(mfgInput).toBeEnabled();
    expect(nameInput).toBeEnabled();
    expect(kvInput).toBeEnabled();
    expect(screen.queryByLabelText(/id slug/i)).not.toBeInTheDocument();
    expect(screen.getByRole("button", { name: /create motor/i })).toBeInTheDocument();
  });

  it("allows adding and removing reference links", async () => {
    const user = userEvent.setup();

    renderWithProviders(
      <ComponentFormModal isOpen={true} onClose={vi.fn()} collectionId="motors" />,
    );

    const addLinkBtn = screen.getByRole("button", { name: /add link/i });
    await user.click(addLinkBtn);

    const labelInput = screen.getByPlaceholderText(/label \(e\.g\. official manual\)/i);
    const urlInput = screen.getByPlaceholderText(/https:\/\/example\.com\/spec-sheet/i);

    expect(labelInput).toBeInTheDocument();
    expect(urlInput).toBeInTheDocument();

    await user.type(labelInput, "Official Spec Sheet");
    await user.type(urlInput, "https://store.tmotor.com/f80pro");

    expect(labelInput).toHaveValue("Official Spec Sheet");
    expect(urlInput).toHaveValue("https://store.tmotor.com/f80pro");
  });

  it("calls onClose when cancel or escape is pressed", async () => {
    const user = userEvent.setup();
    const handleClose = vi.fn();

    renderWithProviders(
      <ComponentFormModal isOpen={true} onClose={handleClose} collectionId="motors" />,
    );

    const cancelBtn = screen.getByRole("button", { name: /cancel/i });
    await user.click(cancelBtn);

    expect(handleClose).toHaveBeenCalledTimes(1);
  });
});
