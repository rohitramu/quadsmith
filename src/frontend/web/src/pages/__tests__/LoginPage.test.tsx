import { describe, it, expect, beforeEach } from "vitest";
import { screen, waitFor, fireEvent } from "@testing-library/react";
import { renderApp } from "../../test/test-utils";

function enterOtp(inputs: HTMLElement[], code: string) {
  code.split("").forEach((char, i) => {
    fireEvent.change(inputs[i], { target: { value: char } });
  });
}

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

    // Verify Back button is present and removed buttons are NOT present
    expect(screen.getByRole("button", { name: /back/i })).toBeInTheDocument();
    expect(screen.queryByText(/change email/i)).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /resend email/i })).not.toBeInTheDocument();
    expect(
      screen.queryByRole("button", { name: /request new verification code/i }),
    ).not.toBeInTheDocument();
    expect(
      screen.queryByRole("button", { name: /temporarily locked out/i }),
    ).not.toBeInTheDocument();
  });

  it("clears fields on incorrect code, displays incorrect code as placeholder hints, keeps neutral borders, and displays error below verification code and above buttons", async () => {
    const { user } = renderApp("/login", {
      transportOptions: { isAuthenticated: false },
    });

    const emailInput = await screen.findByPlaceholderText(/alex@quadsmith.dev/i);
    await user.type(emailInput, "pilot@example.com");

    const submitBtn = screen.getByRole("button", { name: /send verification code/i });
    await user.click(submitBtn);

    await screen.findByRole("heading", { name: /verify your email/i });

    // No Resend Email button
    expect(screen.queryByRole("button", { name: /resend email/i })).not.toBeInTheDocument();

    // Type incorrect 6-digit code: "888888"
    const inputs = screen.getAllByRole("textbox") as HTMLInputElement[];
    const wrongDigits = ["8", "8", "8", "8", "8", "8"];
    enterOtp(inputs, "888888");

    // Error banner appears with yellow (amber) styling for 1 failed attempt (2 attempts left)
    expect(await screen.findByText(/incorrect/i)).toBeInTheDocument();
    const banner = screen.getByTestId("otp-error-banner");
    expect(banner.className).toMatch(/amber/);
    expect(banner.className).not.toMatch(/rose/);

    // Verify DOM position: below the verification code inputs and above the buttons
    const backBtn = screen.getByRole("button", { name: /back/i });
    expect(
      inputs[5].compareDocumentPosition(banner) & Node.DOCUMENT_POSITION_FOLLOWING,
    ).toBeTruthy();
    expect(banner.compareDocumentPosition(backBtn) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();

    const attemptsBadge = screen.getByText(/^2 attempts left$/i);
    expect(attemptsBadge.className).toMatch(/text-amber-500/);

    // Inputs are cleared (empty value) with incorrect digits as placeholders, keeping neutral borders (NOT amber or rose)
    inputs.forEach((input, index) => {
      expect(input.value).toBe("");
      expect(input.placeholder).toBe(wrongDigits[index]);
      expect(input.className).toMatch(/border-zinc-300/);
      expect(input.className).not.toMatch(/border-amber-500/);
      expect(input.className).not.toMatch(/border-rose-500/);
    });

    // Type 2nd failed attempt: "777777" (1 attempt left)
    enterOtp(inputs, "777777");

    // Error banner switches to red (rose) styling for 2 failed attempts (1 attempt left)
    const redBanner = await screen.findByTestId("otp-error-banner");
    expect(redBanner.className).toMatch(/rose/);
    expect(redBanner.className).not.toMatch(/amber/);

    const finalBadge = screen.getByText(/^1 attempt left$/i);
    expect(finalBadge.className).toMatch(/text-rose-500/);

    inputs.forEach((input) => {
      expect(input.className).toMatch(/border-zinc-300/);
      expect(input.className).not.toMatch(/border-amber-500/);
      expect(input.className).not.toMatch(/border-rose-500/);
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
    enterOtp(inputs, "111111");

    // Auto-continues without clicking any button: token saved in localStorage
    await waitFor(() => {
      expect(localStorage.getItem("qs_access_token")).toBe("mock-jwt-token");
    });

    // Resend Email button is absent
    expect(screen.queryByRole("button", { name: /resend email/i })).not.toBeInTheDocument();
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
    enterOtp(inputs, "999999");

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

    // Resend email button is not present
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

  it("handles multi-code 60-minute lockout with disabled controls, without leaking IP in error, and with no locked out button", async () => {
    const { user } = renderApp("/login", {
      transportOptions: { isAuthenticated: false },
    });

    const emailInput = await screen.findByPlaceholderText(/alex@quadsmith.dev/i);
    await user.type(emailInput, "attacker@example.com");

    const submitBtn = screen.getByRole("button", { name: /send verification code/i });
    await user.click(submitBtn);

    await screen.findByRole("heading", { name: /verify your email/i });

    // Enter code 000000 to trigger multi-code lockout
    const inputs = screen.getAllByRole("textbox");
    enterOtp(inputs, "000000");

    // Displays the 60-minute lockout message with Lock icon, without mentioning IP
    expect(await screen.findByText(/too many failed verification attempts/i)).toBeInTheDocument();
    expect(screen.queryByText(/from this ip/i)).not.toBeInTheDocument();
    expect(screen.getByText(/60 minutes/i)).toBeInTheDocument();
    const banner = screen.getByTestId("otp-error-banner");
    expect(banner.querySelector("svg.lucide-lock")).toBeInTheDocument();

    // Temporarily Locked Out button is removed
    expect(
      screen.queryByRole("button", { name: /temporarily locked out/i }),
    ).not.toBeInTheDocument();
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

  it("preserves remaining attempts counter and placeholder hints across failed entries with no resend button", async () => {
    const { user } = renderApp("/login", {
      transportOptions: { isAuthenticated: false },
    });

    const emailInput = await screen.findByPlaceholderText(/alex@quadsmith.dev/i);
    await user.type(emailInput, "pilot@example.com");

    const submitBtn = screen.getByRole("button", { name: /send verification code/i });
    await user.click(submitBtn);

    await screen.findByRole("heading", { name: /verify your email/i });

    // Type incorrect 6-digit code: "888888" to fail once (2 attempts left)
    const inputs = screen.getAllByRole("textbox") as HTMLInputElement[];
    enterOtp(inputs, "888888");

    expect(await screen.findByText(/^2 attempts left$/i)).toBeInTheDocument();
    inputs.forEach((input) => {
      expect(input.value).toBe("");
      expect(input.placeholder).toBe("8");
    });

    // Resend Email button is completely removed
    expect(screen.queryByRole("button", { name: /resend email/i })).not.toBeInTheDocument();
  });
});
