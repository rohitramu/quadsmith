import { describe, it, expect, beforeEach, vi } from "vitest";
import { screen, waitFor, fireEvent, act } from "@testing-library/react";
import { renderApp, renderWithProviders } from "../../test/test-utils";
import { AppRoutes } from "../../App";
import { LoginChallengeStatus } from "../../gen/quadsmith/auth_pb";

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
    expect(
      screen.getByText(/waiting for magic link approval on your other device/i),
    ).toBeInTheDocument();
    expect(
      screen.getByText(/click the magic link in your email on your phone/i),
    ).toBeInTheDocument();

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
    await screen.findByText(/waiting for magic link approval on your other device/i);
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

    // Wait for lockout state (15s cooldown)
    const reqBtn = await screen.findByRole("button", {
      name: /request new verification code \(15s\)/i,
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

  it("handles multi-code 60-minute lockout with disabled controls, without leaking IP in error, and showing greyed out Request New Verification Code (60m0s) button", async () => {
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

    // Request New Verification Code (60m0s) button is present and disabled (greyed out)
    const reqBtn = screen.getByRole("button", {
      name: /request new verification code \(60m0s\)/i,
    });
    expect(reqBtn).toBeInTheDocument();
    expect(reqBtn).toBeDisabled();

    // Temporarily Locked Out button is removed
    expect(
      screen.queryByRole("button", { name: /temporarily locked out/i }),
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

  it("transitions into Account Setup with sliding container when OTP verification requires setup", async () => {
    const { user } = renderApp("/login", {
      transportOptions: { isAuthenticated: false },
    });

    const emailInput = await screen.findByPlaceholderText(/alex@quadsmith.dev/i);
    await user.type(emailInput, "newpilot@example.com");

    const submitBtn = screen.getByRole("button", { name: /send verification code/i });
    await user.click(submitBtn);

    await screen.findByRole("heading", { name: /verify your email/i });

    // Enter 222222 to trigger accountSetupRequired
    const inputs = screen.getAllByRole("textbox");
    enterOtp(inputs, "222222");

    // Sliding container is rendered with Account Setup screen
    expect(await screen.findByTestId("sliding-container")).toBeInTheDocument();
    expect(await screen.findByRole("heading", { name: /account setup/i })).toBeInTheDocument();
    expect(
      screen.getByText(/your username is your public display identifier across quadsmith/i),
    ).toBeInTheDocument();

    // Username input is empty by default and required
    const usernameInput = screen.getByTestId("username-input") as HTMLInputElement;
    expect(usernameInput.value).toBe("");
    expect(screen.getByText(/^required$/i)).toBeInTheDocument();

    // Complete setup button is disabled initially
    const completeBtn = screen.getByTestId("complete-setup-btn");
    expect(completeBtn).toBeDisabled();

    // Live avatar preview renders initial
    const avatarPreview = screen.getByTestId("avatar-preview");
    expect(avatarPreview).toBeInTheDocument();
    expect(avatarPreview.textContent).toBe("N"); // initial of newpilot
  });

  it("shows distinct errors for invalid naming rules vs unavailable taken usernames on Account Setup", async () => {
    const { user } = renderApp("/login", {
      transportOptions: { isAuthenticated: false },
    });

    const emailInput = await screen.findByPlaceholderText(/alex@quadsmith.dev/i);
    await user.type(emailInput, "newpilot@example.com");
    await user.click(screen.getByRole("button", { name: /send verification code/i }));
    await screen.findByRole("heading", { name: /verify your email/i });

    const inputs = screen.getAllByRole("textbox");
    enterOtp(inputs, "222222");

    await screen.findByRole("heading", { name: /account setup/i });
    const usernameInput = screen.getByTestId("username-input");
    const completeBtn = screen.getByTestId("complete-setup-btn");

    // 1. Taken username ("alex") -> shows Unavailable badge & already taken message
    fireEvent.change(usernameInput, { target: { value: "alex" } });
    expect(await screen.findByText(/^unavailable$/i, {}, { timeout: 3000 })).toBeInTheDocument();
    expect(screen.getByText(/this username is already taken/i)).toBeInTheDocument();
    expect(completeBtn).toBeDisabled();

    // 2. Invalid length ("ab" - too short) -> shows Invalid badge & naming rule message
    fireEvent.change(usernameInput, { target: { value: "ab" } });
    expect(await screen.findByText(/^invalid$/i, {}, { timeout: 3000 })).toBeInTheDocument();
    expect(screen.getByText(/3-20 characters/i)).toBeInTheDocument();
    expect(completeBtn).toBeDisabled();

    // 3. Invalid consecutive underscores ("rotor__ninja") -> shows Invalid badge & underscore rule message
    fireEvent.change(usernameInput, { target: { value: "rotor__ninja" } });
    expect(await screen.findByText(/^invalid$/i, {}, { timeout: 3000 })).toBeInTheDocument();
    expect(screen.getByText(/consecutive underscores/i)).toBeInTheDocument();
    expect(completeBtn).toBeDisabled();

    // 4. Reserved username ("admin") -> treated the same as unavailable! Shows Unavailable badge
    fireEvent.change(usernameInput, { target: { value: "admin" } });
    expect(await screen.findByText(/^unavailable$/i, {}, { timeout: 3000 })).toBeInTheDocument();
    expect(screen.getByText(/this username is already taken/i)).toBeInTheDocument();
    expect(completeBtn).toBeDisabled();

    // 5. Available username ("rotor_pilot") -> shows Available badge & enables complete setup
    fireEvent.change(usernameInput, { target: { value: "rotor_pilot" } });
    expect(await screen.findByText(/^✓ available$/i, {}, { timeout: 3000 })).toBeInTheDocument();
    expect(screen.getByText(/username is available/i)).toBeInTheDocument();
    expect(completeBtn).toBeEnabled();

    // Avatar preview dynamically updates to initial "R" with palette badge styling
    const avatarPreview = screen.getByTestId("avatar-preview");
    expect(avatarPreview.textContent).toBe("R");
    expect(avatarPreview.className).toMatch(/bg-teal-100/);
  });

  it("updates display image live preview and completes setup successfully", async () => {
    const { user } = renderApp("/login", {
      transportOptions: { isAuthenticated: false },
    });

    const emailInput = await screen.findByPlaceholderText(/alex@quadsmith.dev/i);
    await user.type(emailInput, "newpilot@example.com");
    await user.click(screen.getByRole("button", { name: /send verification code/i }));
    await screen.findByRole("heading", { name: /verify your email/i });

    const inputs = screen.getAllByRole("textbox");
    enterOtp(inputs, "222222");

    await screen.findByRole("heading", { name: /account setup/i });
    const usernameInput = screen.getByTestId("username-input");
    fireEvent.change(usernameInput, { target: { value: "rotor_pilot" } });
    expect(await screen.findByText(/^✓ available$/i, {}, { timeout: 3000 })).toBeInTheDocument();

    // Optional avatar URL
    const avatarUrlInput = screen.getByTestId("avatar-url-input");
    fireEvent.change(avatarUrlInput, { target: { value: "https://example.com/avatar.jpg" } });

    const avatarImg = screen.getByAltText("Avatar Preview") as HTMLImageElement;
    expect(avatarImg).toBeInTheDocument();
    expect(avatarImg.src).toBe("https://example.com/avatar.jpg");

    // Click Complete Setup
    const completeBtn = screen.getByTestId("complete-setup-btn");
    await user.click(completeBtn);

    // Setup completes and access token is saved in localStorage
    await waitFor(() => {
      expect(localStorage.getItem("qs_access_token")).toBe("mock-jwt-token");
    });
  });

  it("resets back to email step when 'Use a different email' is clicked on Account Setup", async () => {
    const { user } = renderApp("/login", {
      transportOptions: { isAuthenticated: false },
    });

    const emailInput = await screen.findByPlaceholderText(/alex@quadsmith.dev/i);
    await user.type(emailInput, "newpilot@example.com");
    await user.click(screen.getByRole("button", { name: /send verification code/i }));
    await screen.findByRole("heading", { name: /verify your email/i });

    const inputs = screen.getAllByRole("textbox");
    enterOtp(inputs, "222222");

    await screen.findByRole("heading", { name: /account setup/i });

    // Click "Use a different email"
    const diffEmailBtn = screen.getByRole("button", { name: /use a different email/i });
    await user.click(diffEmailBtn);

    // Transitions back to Step 1
    expect(
      await screen.findByRole("heading", { name: /sign in to quadsmith/i }),
    ).toBeInTheDocument();
    const resetEmailInput = screen.getByPlaceholderText(/alex@quadsmith.dev/i) as HTMLInputElement;
    expect(resetEmailInput.value).toBe("");
  });

  it("keeps Send Verification Code button disabled when email field is empty, before and after cooldown", async () => {
    vi.useFakeTimers();
    try {
      renderWithProviders(<AppRoutes />, {
        route: "/login",
        transportOptions: { isAuthenticated: false },
      });

      const emailInput = screen.getByPlaceholderText(/alex@quadsmith.dev/i);
      const submitBtn = screen.getByRole("button", { name: /^send verification code$/i });

      // Initially empty: disabled with cursor-not-allowed
      expect(submitBtn).toBeDisabled();
      expect(submitBtn.className).toMatch(/cursor-not-allowed/);
      expect(submitBtn.className).not.toMatch(/cursor-pointer/);
      expect(submitBtn.className).toMatch(/bg-zinc-100/);

      // Typing email enables the button
      fireEvent.change(emailInput, { target: { value: "pilot@example.com" } });
      expect(submitBtn).not.toBeDisabled();
      expect(submitBtn.className).toMatch(/cursor-pointer/);
      expect(submitBtn.className).toMatch(/bg-blue-600/);
      expect(screen.getByText("→")).toBeInTheDocument();

      // Clearing email disables it again
      fireEvent.change(emailInput, { target: { value: "" } });
      expect(submitBtn).toBeDisabled();
      expect(submitBtn.className).toMatch(/cursor-not-allowed/);
      expect(screen.queryByText("→")).not.toBeInTheDocument();

      // Re-type email and submit to trigger cooldown & transition to step 2
      fireEvent.change(emailInput, { target: { value: "pilot@example.com" } });
      fireEvent.click(submitBtn);

      // Advance mock promise tick
      await act(async () => {
        await Promise.resolve();
      });

      // Verify OTP screen reached
      expect(screen.getByRole("heading", { name: /verify your email/i })).toBeInTheDocument();

      // Click Back to return to Step 1
      const backBtn = screen.getByRole("button", { name: /back/i });
      fireEvent.click(backBtn);

      // Back on Step 1: email is reset to empty, cooldown is active (20s)
      expect(screen.getByRole("heading", { name: /sign in to quadsmith/i })).toBeInTheDocument();
      const backEmailInput = screen.getByPlaceholderText(/alex@quadsmith.dev/i) as HTMLInputElement;
      expect(backEmailInput.value).toBe("");

      // Button is in cooldown and disabled
      const cooldownBtn = screen.getByRole("button", {
        name: /send verification code \(\d+s\)/i,
      });
      expect(cooldownBtn).toBeDisabled();
      expect(cooldownBtn.className).toMatch(/cursor-not-allowed/);

      // Fast-forward 25 seconds so cooldown expires completely
      act(() => {
        vi.advanceTimersByTime(25000);
      });

      // After cooldown expires: email field is still empty, so button MUST remain disabled!
      const postCooldownBtn = screen.getByRole("button", {
        name: /^send verification code$/i,
      });
      expect(postCooldownBtn).toBeDisabled();
      expect(postCooldownBtn.className).toMatch(/cursor-not-allowed/);
      expect(postCooldownBtn.className).not.toMatch(/cursor-pointer/);
      expect(postCooldownBtn.className).toMatch(/bg-zinc-100/);
      expect(screen.queryByText("→")).not.toBeInTheDocument();

      // Now entering an email address after cooldown expires enables the button
      fireEvent.change(backEmailInput, { target: { value: "another@example.com" } });
      expect(postCooldownBtn).not.toBeDisabled();
      expect(postCooldownBtn.className).toMatch(/cursor-pointer/);
      expect(postCooldownBtn.className).toMatch(/bg-blue-600/);
      expect(screen.getByText("→")).toBeInTheDocument();
    } finally {
      vi.useRealTimers();
    }
  });

  it("renders Device B approval screen when token is present and approves login", async () => {
    const { user } = renderApp("/login?token=valid-magic-token", {
      transportOptions: { isAuthenticated: false },
    });

    expect(
      await screen.findByRole("heading", { name: /approve login request/i }),
    ).toBeInTheDocument();
    expect(await screen.findByText("alex@fpv.build")).toBeInTheDocument();
    expect(screen.getByText("Chrome on macOS")).toBeInTheDocument();
    expect(screen.getByText(/192.168.1.42 \(Local Network\)/i)).toBeInTheDocument();

    const approveBtn = await screen.findByTestId("approve-login-btn");
    await user.click(approveBtn);

    expect(await screen.findByTestId("approval-success-card")).toBeInTheDocument();
    expect(screen.getByText(/login approved!/i)).toBeInTheDocument();
    expect(screen.getByText(/you can safely close this window now/i)).toBeInTheDocument();
  });

  it("renders Device B approval screen when token is present and denies request", async () => {
    const { user } = renderApp("/login?token=valid-magic-token", {
      transportOptions: { isAuthenticated: false },
    });

    expect(
      await screen.findByRole("heading", { name: /approve login request/i }),
    ).toBeInTheDocument();
    expect(await screen.findByText("alex@fpv.build")).toBeInTheDocument();

    const denyBtn = await screen.findByTestId("deny-login-btn");
    await user.click(denyBtn);

    expect(await screen.findByTestId("approval-denied-card")).toBeInTheDocument();
    expect(screen.getByText(/login request denied/i)).toBeInTheDocument();
    expect(screen.getByText(/you can safely close this window now/i)).toBeInTheDocument();
  });

  it("polls and auto-logs in Device A when challenge is approved on Device B", async () => {
    let polled = false;
    const { user } = renderApp("/login", {
      transportOptions: {
        isAuthenticated: false,
        authService: {
          getLoginChallengeStatus: () => {
            if (!polled) {
              polled = true;
              return { status: LoginChallengeStatus.PENDING };
            }
            return {
              status: LoginChallengeStatus.APPROVED,
              authResponse: {
                accessToken: "cross-device-approved-token",
                refreshToken: "cross-device-refresh-token",
                user: { id: "pilot", email: "pilot@example.com" },
                accountSetupRequired: false,
              },
            };
          },
        },
      },
    });

    const emailInput = await screen.findByPlaceholderText(/alex@quadsmith.dev/i);
    await user.type(emailInput, "pilot@example.com");

    const submitBtn = screen.getByRole("button", { name: /send verification code/i });
    await user.click(submitBtn);

    await screen.findByText(/waiting for magic link approval on your other device/i);

    await waitFor(
      () => {
        expect(localStorage.getItem("qs_access_token")).toBe("cross-device-approved-token");
      },
      { timeout: 8000 },
    );
  });

  it("displays denial message and halts polling on Device A when challenge is denied on Device B", async () => {
    const { user } = renderApp("/login", {
      transportOptions: {
        isAuthenticated: false,
        authService: {
          getLoginChallengeStatus: () => {
            return {
              status: LoginChallengeStatus.DENIED,
            };
          },
        },
      },
    });

    const emailInput = await screen.findByPlaceholderText(/alex@quadsmith.dev/i);
    await user.type(emailInput, "pilot@example.com");

    const submitBtn = screen.getByRole("button", { name: /send verification code/i });
    await user.click(submitBtn);

    await screen.findByText(/waiting for magic link approval on your other device/i);

    await waitFor(
      () => {
        expect(screen.getByText(/login request was denied/i)).toBeInTheDocument();
      },
      { timeout: 8000 },
    );
  });
});
