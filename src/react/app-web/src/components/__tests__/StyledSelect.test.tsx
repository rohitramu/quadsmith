import { render, screen, fireEvent } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect, vi } from "vitest";
import { StyledSelect } from "../StyledSelect";

describe("StyledSelect", () => {
  const options = [
    { value: "opt1", label: "Option One" },
    { value: "opt2", label: "Option Two" },
    { value: "opt3", label: "Option Three" },
  ];

  it("renders trigger button with currently selected label", () => {
    render(
      <StyledSelect value="opt1" onChange={() => {}} options={options} ariaLabel="Test select" />,
    );

    const button = screen.getByRole("combobox", { name: "Test select" });
    expect(button).toBeInTheDocument();
    expect(button).toHaveTextContent("Option One");
    expect(button).toHaveAttribute("aria-expanded", "false");
  });

  it("opens menu when clicked and selects new option", async () => {
    const user = userEvent.setup();
    const handleChange = vi.fn();

    render(
      <StyledSelect
        value="opt1"
        onChange={handleChange}
        options={options}
        ariaLabel="Test select"
      />,
    );

    const button = screen.getByRole("combobox", { name: "Test select" });
    await user.click(button);

    expect(button).toHaveAttribute("aria-expanded", "true");
    const listbox = screen.getByRole("listbox");
    expect(listbox).toBeInTheDocument();

    const option2 = screen.getByRole("option", { name: "Option Two" });
    expect(option2).toBeInTheDocument();

    await user.click(option2);
    expect(handleChange).toHaveBeenCalledWith("opt2");
    expect(screen.queryByRole("listbox")).not.toBeInTheDocument();
  });

  it("navigates options via keyboard and closes on Escape", async () => {
    const user = userEvent.setup();
    const handleChange = vi.fn();

    render(
      <StyledSelect
        value="opt1"
        onChange={handleChange}
        options={options}
        ariaLabel="Test select"
      />,
    );

    const button = screen.getByRole("combobox", { name: "Test select" });
    button.focus();

    // Open via Enter
    await user.keyboard("{Enter}");
    expect(screen.getByRole("listbox")).toBeInTheDocument();

    // Close via Escape
    await user.keyboard("{Escape}");
    expect(screen.queryByRole("listbox")).not.toBeInTheDocument();

    // Open via ArrowDown
    await user.keyboard("{ArrowDown}");
    expect(screen.getByRole("listbox")).toBeInTheDocument();

    // Navigate to next and select with Enter
    await user.keyboard("{ArrowDown}");
    await user.keyboard("{Enter}");
    expect(handleChange).toHaveBeenCalledWith("opt2");
  });

  it("closes on outside click", async () => {
    const user = userEvent.setup();

    render(
      <div>
        <StyledSelect value="opt1" onChange={() => {}} options={options} ariaLabel="Test select" />
        <div data-testid="outside">Outside area</div>
      </div>,
    );

    const button = screen.getByRole("combobox", { name: "Test select" });
    await user.click(button);
    expect(screen.getByRole("listbox")).toBeInTheDocument();

    fireEvent.mouseDown(screen.getByTestId("outside"));
    expect(screen.queryByRole("listbox")).not.toBeInTheDocument();
  });
});
