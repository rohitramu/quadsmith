import { describe, it, expect, beforeEach } from "vitest";
import { screen } from "@testing-library/react";
import { renderApp } from "../../test/test-utils";

describe("LoginPage Component", () => {
  beforeEach(() => {
    localStorage.removeItem("qs_access_token");
  });

  it("renders step 1 with email form and no navbar or sidebar", async () => {
    renderApp("/login", {
      transportOptions: { isAuthenticated: false },
    });

    // Verify login title and input
    expect(
      await screen.findByRole("heading", { name: /sign in to quadsmith/i }),
    ).toBeInTheDocument();
    expect(screen.getByPlaceholderText(/alex@quadsmith.dev/i)).toBeInTheDocument();

    // Verify navbar / sidebar elements are NOT present
    expect(screen.queryByPlaceholderText(/search components/i)).not.toBeInTheDocument();
    expect(screen.queryByRole("link", { name: /hardware/i })).not.toBeInTheDocument();
  });

  it("submits email and transitions to step 2 with dual-path instructions", async () => {
    const { user } = renderApp("/login", {
      transportOptions: { isAuthenticated: false },
    });

    const emailInput = await screen.findByPlaceholderText(/alex@quadsmith.dev/i);
    await user.type(emailInput, "pilot@example.com");

    const submitBtn = screen.getByRole("button", { name: /send verification code/i });
    await user.click(submitBtn);

    // Verify transition to OTP step
    expect(await screen.findByText(/two ways to sign in/i)).toBeInTheDocument();
    expect(screen.getByText(/enter the/i)).toBeInTheDocument();
    expect(screen.getByText(/6-digit code/i)).toBeInTheDocument();
    expect(screen.getByText(/1-click magic link/i)).toBeInTheDocument();

    // Verify 6 OTP input boxes
    const inputs = screen.getAllByRole("textbox");
    expect(inputs.length).toBe(6);
  });
});
