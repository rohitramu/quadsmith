import { describe, it, expect, beforeEach } from "vitest";
import { screen, waitFor } from "@testing-library/react";
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
    expect(screen.queryByText(/two ways to sign in/i)).not.toBeInTheDocument();
    expect(await screen.findByRole("heading", { name: /verify your email/i })).toBeInTheDocument();
    expect(screen.getByText(/enter the/i)).toBeInTheDocument();
    expect(screen.getByText(/6-digit code/i)).toBeInTheDocument();
    expect(screen.getByText(/^or$/i)).toBeInTheDocument();
    expect(screen.getByText(/click the link in the email/i)).toBeInTheDocument();

    // Verify 6 OTP input boxes
    const inputs = screen.getAllByRole("textbox");
    expect(inputs.length).toBe(6);

    // Verify Back button and Resend Email button are present
    expect(screen.getByRole("button", { name: /back/i })).toBeInTheDocument();
    expect(screen.queryByText(/change email/i)).not.toBeInTheDocument();
    expect(screen.getByRole("button", { name: /resend email/i })).toBeInTheDocument();
    expect(
      screen.queryByRole("button", { name: /request new verification code/i }),
    ).not.toBeInTheDocument();
    expect(
      screen.queryByRole("button", { name: /temporarily locked out/i }),
    ).not.toBeInTheDocument();
  });

  it("clears fields on incorrect code, displays incorrect code as placeholder hints, and preserves Resend Email button", async () => {
    const { user } = renderApp("/login", {
      transportOptions: { isAuthenticated: false },
    });

    const emailInput = await screen.findByPlaceholderText(/alex@quadsmith.dev/i);
    await user.type(emailInput, "pilot@example.com");

    const submitBtn = screen.getByRole("button", { name: /send verification code/i });
    await user.click(submitBtn);

    await screen.findByRole("heading", { name: /verify your email/i });

    // Resend Email button is visible initially
    expect(screen.getByRole("button", { name: /resend email/i })).toBeInTheDocument();

    // Type incorrect 6-digit code: "888888"
    const inputs = screen.getAllByRole("textbox") as HTMLInputElement[];
    const wrongDigits = ["8", "8", "8", "8", "8", "8"];
    for (let i = 0; i < 6; i++) {
      await user.type(inputs[i], wrongDigits[i]);
    }

    // Error banner appears with yellow (amber) styling for 1 failed attempt (2 attempts left)
    expect(await screen.findByText(/incorrect/i)).toBeInTheDocument();
    const banner = screen.getByTestId("otp-error-banner");
    expect(banner.className).toMatch(/amber/);
    expect(banner.className).not.toMatch(/rose/);

    const attemptsBadge = screen.getByText(/^2 attempts left$/i);
    expect(attemptsBadge.className).toMatch(/text-amber-500/);

    // Inputs are cleared (empty value) but have the incorrect digits as placeholders and amber border
    inputs.forEach((input, index) => {
      expect(input.value).toBe("");
      expect(input.placeholder).toBe(wrongDigits[index]);
      expect(input.className).toMatch(/border-amber-500/);
    });

    // Resend Email button is STILL visible while waiting for re-entry
    expect(screen.getByRole("button", { name: /resend email/i })).toBeInTheDocument();

    // Type 2nd failed attempt: "777777" (1 attempt left)
    for (let i = 0; i < 6; i++) {
      await user.type(inputs[i], "7");
    }

    // Error banner switches to red (rose) styling for 2 failed attempts (1 attempt left)
    const redBanner = await screen.findByTestId("otp-error-banner");
    expect(redBanner.className).toMatch(/rose/);
    expect(redBanner.className).not.toMatch(/amber/);

    const finalBadge = screen.getByText(/^1 attempt left$/i);
    expect(finalBadge.className).toMatch(/text-rose-500/);

    inputs.forEach((input) => {
      expect(input.className).toMatch(/border-rose-500/);
    });
  });

  it("does not render passwordless footer text, has no continue button, and automatically continues on 6-digit OTP entry", async () => {
    const { user } = renderApp("/login", {
      transportOptions: { isAuthenticated: false },
    });

    // Ensure footer text is absent on step 1
    expect(screen.queryByText(/secure, passwordless authentication/i)).not.toBeInTheDocument();

    const emailInput = await screen.findByPlaceholderText(/alex@quadsmith.dev/i);
    await user.type(emailInput, "pilot@example.com");

    const submitBtn = screen.getByRole("button", { name: /send verification code/i });
    await user.click(submitBtn);

    // Verify OTP step reached
    await screen.findByText(/click the link in the email/i);
    expect(screen.queryByText(/two ways to sign in/i)).not.toBeInTheDocument();

    // Ensure footer text is absent on step 2
    expect(screen.queryByText(/secure, passwordless authentication/i)).not.toBeInTheDocument();

    // Verify continue button is removed
    expect(
      screen.queryByRole("button", { name: /verify code & continue/i }),
    ).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /^continue$/i })).not.toBeInTheDocument();

    // Enter 6-digit code into OTP boxes
    const inputs = screen.getAllByRole("textbox");
    expect(inputs.length).toBe(6);

    for (let i = 0; i < 6; i++) {
      await user.type(inputs[i], "1");
    }

    // Auto-continues without clicking any button: token saved in localStorage
    await waitFor(() => {
      expect(localStorage.getItem("qs_access_token")).toBe("mock-jwt-token");
    });

    // Resend Email button is disabled on success and has NO countdown timer
    const resendBtn = screen.getByRole("button", { name: /^resend email$/i });
    expect(resendBtn).toBeInTheDocument();
    expect(resendBtn).toBeDisabled();
    expect(screen.queryByText(/resend email \(\d+s\)/i)).not.toBeInTheDocument();
  });

  it("greys out text boxes, removes locked out label above inputs, and displays Request New Verification Code upon lockout", async () => {
    const { user } = renderApp("/login", {
      transportOptions: { isAuthenticated: false },
    });

    const emailInput = await screen.findByPlaceholderText(/alex@quadsmith.dev/i);
    await user.type(emailInput, "pilot@example.com");

    const submitBtn = screen.getByRole("button", { name: /send verification code/i });
    await user.click(submitBtn);

    await screen.findByRole("heading", { name: /verify your email/i });

    // Enter incorrect code 999999 to trigger exhausted lockout
    const inputs = screen.getAllByRole("textbox");
    for (let i = 0; i < 6; i++) {
      await user.type(inputs[i], "9");
    }

    // Wait for lockout state (60s cooldown)
    const reqBtn = await screen.findByRole("button", {
      name: /request new verification code \(60s\)/i,
    });
    expect(reqBtn).toBeInTheDocument();
    expect(reqBtn).toBeDisabled();

    // Error banner renders Lock icon
    const banner = await screen.findByTestId("otp-error-banner");
    expect(banner.querySelector("svg.lucide-lock")).toBeInTheDocument();

    // Verification code label does not contain 3-min expiry
    expect(screen.getByText(/^verification code$/i)).toBeInTheDocument();
    expect(screen.queryByText(/3-min expiry/i)).not.toBeInTheDocument();

    // Resend email button is replaced by Request New Verification Code
    expect(screen.queryByRole("button", { name: /resend email/i })).not.toBeInTheDocument();

    // Text boxes are disabled
    inputs.forEach((input) => {
      expect(input).toBeDisabled();
    });

    // "Locked out" text is NOT shown above text boxes
    expect(screen.queryByText(/locked out/i)).not.toBeInTheDocument();

    // Back button is still present
    expect(screen.getByRole("button", { name: /back/i })).toBeInTheDocument();
  });

  it("handles multi-code 60-minute IP lockout with disabled controls and lockout warning", async () => {
    const { user } = renderApp("/login", {
      transportOptions: { isAuthenticated: false },
    });

    const emailInput = await screen.findByPlaceholderText(/alex@quadsmith.dev/i);
    await user.type(emailInput, "attacker@example.com");

    const submitBtn = screen.getByRole("button", { name: /send verification code/i });
    await user.click(submitBtn);

    await screen.findByRole("heading", { name: /verify your email/i });

    // Enter code 000000 to trigger multi-code IP lockout
    const inputs = screen.getAllByRole("textbox");
    for (let i = 0; i < 6; i++) {
      await user.type(inputs[i], "0");
    }

    // Displays the 60-minute IP lockout message with Lock icon
    expect(
      await screen.findByText(/too many failed verification attempts from this ip/i),
    ).toBeInTheDocument();
    expect(screen.getByText(/60 minutes/i)).toBeInTheDocument();
    const banner = screen.getByTestId("otp-error-banner");
    expect(banner.querySelector("svg.lucide-lock")).toBeInTheDocument();

    // Renders disabled "Temporarily Locked Out" button with Lock icon instead of Request New Code
    const lockedBtn = screen.getByRole("button", { name: /temporarily locked out/i });
    expect(lockedBtn).toBeInTheDocument();
    expect(lockedBtn).toBeDisabled();
    expect(lockedBtn.querySelector("svg.lucide-lock")).toBeInTheDocument();
    expect(
      screen.queryByRole("button", { name: /request new verification code/i }),
    ).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /resend email/i })).not.toBeInTheDocument();

    // Text inputs are disabled
    inputs.forEach((input) => {
      expect(input).toBeDisabled();
    });

    // Clicking "Back" returns to the email input step
    const backBtn = screen.getByRole("button", { name: /back/i });
    await user.click(backBtn);
    expect(
      await screen.findByRole("heading", { name: /sign in to quadsmith/i }),
    ).toBeInTheDocument();
  });
});
