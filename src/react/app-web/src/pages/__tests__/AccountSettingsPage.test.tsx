import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import { screen, fireEvent, waitFor, within } from "@testing-library/react";
import { renderApp } from "../../test/test-utils";
import { mockUser } from "../../test/mocks/fixtures";
import { LoginChallengeStatus, EmailOtpPurpose } from "@quadsmith/shared/gen/quadsmith/api/auth_pb";

describe("AccountSettingsPage Component", () => {
  beforeEach(() => {
    mockUser.id = "fpv_pilot";
    delete mockUser.activeBan;
    localStorage.setItem("qs_access_token", "mock-jwt-token");
    vi.spyOn(window, "confirm").mockImplementation(() => true);
  });

  afterEach(() => {
    delete mockUser.activeBan;
  });

  it("renders account settings with username, profile link, and logout button", async () => {
    renderApp("/account");

    // Verify Heading & Username
    expect(await screen.findByRole("heading", { name: /account settings/i })).toBeInTheDocument();
    expect(screen.getAllByText(/@fpv_pilot/i).length).toBeGreaterThan(0);
    expect(screen.getAllByText(/alex@quadsmith\.dev/i).length).toBeGreaterThan(0);

    // Verify Public Profile button
    const profileLink = screen.getByRole("link", { name: /view public profile/i });
    expect(profileLink).toBeInTheDocument();
    expect(profileLink.getAttribute("href")).toBe("/users/fpv_pilot");

    // Verify Logout button
    const logoutBtn = screen.getByRole("button", { name: /log out/i });
    expect(logoutBtn).toBeInTheDocument();

    // Verify Save button (disabled by default when clean)
    const saveBtn = screen.getByRole("button", { name: /^save$/i });
    expect(saveBtn).toBeInTheDocument();
    expect(saveBtn).toBeDisabled();
  });

  it("renders associated emails with primary and secondary status badges", async () => {
    renderApp("/account");

    expect(await screen.findByText("Associated Email Addresses")).toBeInTheDocument();

    // Both mock emails should be in the table
    await waitFor(() => {
      expect(screen.getAllByText("alex@quadsmith.dev").length).toBeGreaterThan(0);
      expect(screen.getByText("alex.secondary@example.com")).toBeInTheDocument();
    });

    expect(screen.getAllByText("Primary").length).toBeGreaterThan(0);
    expect(screen.queryByText("Secondary")).not.toBeInTheDocument();
    expect(screen.getByText("Make Primary")).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: /remove alex\.secondary@example\.com/i }),
    ).toBeInTheDocument();
  });

  it("allows adding a new email through OTP verification", async () => {
    renderApp("/account");

    const addEmailBtn = await screen.findByRole("button", { name: /add email/i });
    fireEvent.click(addEmailBtn);

    // Modal appears
    expect(screen.getByRole("heading", { name: /add email address/i })).toBeInTheDocument();
    const emailInput = screen.getByPlaceholderText(/alex@quadsmith\.dev/i);
    fireEvent.change(emailInput, { target: { value: "new.pilot@fpv.org" } });

    const sendCodeBtn = screen.getByRole("button", { name: /send verification code/i });
    fireEvent.click(sendCodeBtn);

    // OTP inputs appear in modal
    await waitFor(() => {
      expect(screen.getByRole("heading", { name: /verify your email/i })).toBeInTheDocument();
    });

    const otpInputs = screen
      .getAllByRole("textbox")
      .filter((input) => input.getAttribute("inputmode") === "numeric");
    expect(otpInputs).toHaveLength(6);

    // Entering the 6-digit code auto-submits verification
    fireEvent.change(otpInputs[0], { target: { value: "123456" } });

    // Success notice and email appears in table
    await waitFor(() => {
      expect(screen.getByText(/successfully added new\.pilot@fpv\.org/i)).toBeInTheDocument();
      expect(screen.getByText("new.pilot@fpv.org")).toBeInTheDocument();
    });
  });

  it("allows setting an email as primary and removing an email", async () => {
    renderApp("/account");

    // Wait for secondary email actions
    const makePrimaryBtn = await screen.findByRole("button", { name: /make primary/i });
    fireEvent.click(makePrimaryBtn);

    // Secondary email should become primary
    await waitFor(() => {
      const emailRows = screen.getAllByRole("row");
      expect(emailRows.length).toBeGreaterThan(1);
    });

    // Remove secondary email
    const removeBtn = screen.getByRole("button", { name: /remove/i });
    fireEvent.click(removeBtn);

    await waitFor(() => {
      expect(screen.queryByText("alex@quadsmith.dev")).toBeInTheDocument();
    });
  });

  it("brings up the merge modal with instructions and confirmation when an existing account email is entered in Add Email", async () => {
    renderApp("/account");

    // Click Add Email in Section 1
    const addEmailBtn = await screen.findByRole("button", { name: /add email/i });
    fireEvent.click(addEmailBtn);

    // Enter email of an existing account in modal
    const emailInput = screen.getByPlaceholderText(/alex@quadsmith\.dev/i);
    fireEvent.change(emailInput, { target: { value: "old.racer@fpv.org" } });

    // Click Send Verification Code
    const sendCodeBtn = screen.getByRole("button", { name: /send verification code/i });
    expect(sendCodeBtn).not.toBeDisabled();
    fireEvent.click(sendCodeBtn);

    // Uses the same Add Email modal without ANY early indication that an account already exists
    await waitFor(() => {
      expect(screen.getByRole("heading", { name: /verify your email/i })).toBeInTheDocument();
      expect(
        screen.getByText(/we sent a single-use verification code to old\.racer@fpv\.org/i),
      ).toBeInTheDocument();
      expect(screen.queryByText(/existing account/i)).not.toBeInTheDocument();
      expect(screen.queryByText(/merge/i)).not.toBeInTheDocument();
      expect(screen.queryByText(/old_racer/i)).not.toBeInTheDocument();
    });

    // Enter verification code into Add Email OTP inputs
    const otpInputs = screen
      .getAllByRole("textbox")
      .filter((input) => input.getAttribute("inputmode") === "numeric");
    expect(otpInputs).toHaveLength(6);
    fireEvent.change(otpInputs[0], { target: { value: "123456" } });

    // After successful verification, transition to the "Confirm Account Merge" modal
    await waitFor(() => {
      expect(screen.getByText("Confirm Account Merge")).toBeInTheDocument();
      expect(screen.getByText(/ownership verified for/i)).toBeInTheDocument();
      expect(screen.getByText(/delete username:/i)).toBeInTheDocument();
      expect(screen.getByText(/held for 30 days before being released/i)).toBeInTheDocument();
      expect(screen.getByText(/associate \d+ email addresses/i)).toBeInTheDocument();
      expect(screen.getByText(/migrated resources \(4 drone builds\)/i)).toBeInTheDocument();
      expect(screen.getByText(/bando basher 5-inch/i)).toBeInTheDocument();
      expect(
        screen.getByText(/this action is permanent and cannot be undone/i),
      ).toBeInTheDocument();
    });

    // Confirm Merge
    const confirmBtn = screen.getByRole("button", { name: /^merge$/i });
    expect(confirmBtn).not.toBeDisabled();
    fireEvent.click(confirmBtn);

    // Success state in modal
    await waitFor(() => {
      expect(screen.getByText(/accounts merged successfully!/i)).toBeInTheDocument();
      expect(
        screen.getByText(/all 4 builds from @old_racer have been reassigned to your account/i),
      ).toBeInTheDocument();
      expect(screen.getByText(/all 2 email addresses/i)).toBeInTheDocument();
    });

    const doneBtn = screen.getByRole("button", { name: /done/i });
    fireEvent.click(doneBtn);

    // Modal closes and all emails are in the list
    await waitFor(() => {
      expect(screen.queryByText(/accounts merged successfully!/i)).not.toBeInTheDocument();
      expect(screen.getByText("old.racer@fpv.org")).toBeInTheDocument();
      expect(screen.getByText("old.racer.backup@example.com")).toBeInTheDocument();
    });
  });

  it("verifies ownership via merge_token and presents summary before confirming merge", async () => {
    renderApp("/account?merge_token=valid-mock-token");

    // Arrives with token, verifies ownership, and displays summary
    await waitFor(() => {
      expect(screen.getByText("Confirm Account Merge")).toBeInTheDocument();
      expect(screen.getByText(/ownership verified for/i)).toBeInTheDocument();
      expect(screen.getByText(/delete username:/i)).toBeInTheDocument();
      expect(screen.getByText(/migrated resources \(4 drone builds\)/i)).toBeInTheDocument();
    });

    // User clicks Confirm Merge
    const confirmBtn = screen.getByRole("button", { name: /^merge$/i });
    expect(confirmBtn).not.toBeDisabled();
    fireEvent.click(confirmBtn);

    // Success state in modal
    await waitFor(() => {
      expect(screen.getByText(/accounts merged successfully!/i)).toBeInTheDocument();
      expect(
        screen.getByText(/all 4 builds from @old_racer have been reassigned to your account/i),
      ).toBeInTheDocument();
    });

    const doneBtn = screen.getByRole("button", { name: /done/i });
    fireEvent.click(doneBtn);

    await waitFor(() => {
      expect(screen.queryByText(/accounts merged successfully!/i)).not.toBeInTheDocument();
      expect(screen.getByText("old.racer@fpv.org")).toBeInTheDocument();
      expect(screen.getByText("old.racer.backup@example.com")).toBeInTheDocument();
    });
  });

  it("opens Change Username modal, shows held archived usernames, and allows 1-click reclaim", async () => {
    renderApp("/account");

    const changeUsernameBtn = await screen.findByRole("button", { name: /change username/i });
    expect(changeUsernameBtn).toBeInTheDocument();
    fireEvent.click(changeUsernameBtn);

    // Modal opens
    expect(await screen.findByRole("heading", { name: /change username/i })).toBeInTheDocument();
    expect(await screen.findByText("Your Held Usernames (30d Hold)")).toBeInTheDocument();
    expect(screen.getByText("@old_racer")).toBeInTheDocument();
    expect(screen.getByText("Reclaim Available")).toBeInTheDocument();

    // Click Reclaim button
    const reclaimBtn = screen.getByRole("button", { name: /^reclaim$/i });
    expect(reclaimBtn).toBeInTheDocument();
    expect(reclaimBtn).not.toBeDisabled();
    fireEvent.click(reclaimBtn);

    // Success notice appears
    await waitFor(() => {
      expect(screen.getByText(/changed to @old_racer/i)).toBeInTheDocument();
    });
  });

  it("allows entering a new username and submitting the change", async () => {
    renderApp("/account");

    const changeUsernameBtn = await screen.findByRole("button", { name: /change username/i });
    fireEvent.click(changeUsernameBtn);

    expect(await screen.findByRole("heading", { name: /change username/i })).toBeInTheDocument();

    const usernameInput = screen.getByPlaceholderText("new_username");
    expect(usernameInput).toBeInTheDocument();

    // Type a new valid username
    fireEvent.change(usernameInput, { target: { value: "aero_glide" } });

    // Live availability check
    await waitFor(() => {
      expect(screen.getByText("✓ Available")).toBeInTheDocument();
    });

    const dialog = screen.getByRole("dialog", { name: /change username/i });
    const submitBtn = within(dialog).getByRole("button", { name: /^change username$/i });
    expect(submitBtn).not.toBeDisabled();
    fireEvent.click(submitBtn);

    await waitFor(() => {
      expect(screen.getByText(/changed to @aero_glide/i)).toBeInTheDocument();
    });
  });

  it("indicates invalid username via border styling without showing redundant error message", async () => {
    renderApp("/account");

    const changeUsernameBtn = await screen.findByRole("button", { name: /change username/i });
    fireEvent.click(changeUsernameBtn);

    expect(await screen.findByRole("heading", { name: /change username/i })).toBeInTheDocument();

    const usernameInput = screen.getByPlaceholderText("new_username");
    const container = screen.getByTestId("username-input-container");

    // Type an invalid username (too short: "ab")
    fireEvent.change(usernameInput, { target: { value: "ab" } });

    // The container border indicates invalidity
    expect(container.className).toContain("border-rose-500");
    expect(usernameInput).toHaveAttribute("aria-invalid", "true");

    // The redundant error message is NOT displayed
    expect(
      screen.queryByText(
        "3-20 characters: letters, numbers, and underscores (must start with letter and end with alphanumeric)",
      ),
    ).not.toBeInTheDocument();

    // The explanation below the text box is still present
    expect(
      screen.getByText(/3-20 characters: lowercase letters, numbers, and single underscores/i),
    ).toBeInTheDocument();

    // Submit button is disabled
    const dialog = screen.getByRole("dialog", { name: /change username/i });
    const submitBtn = within(dialog).getByRole("button", { name: /^change username$/i });
    expect(submitBtn).toBeDisabled();
  });

  it("disables Save button initially, enables it when fields are modified, and shows modified indicators", async () => {
    renderApp("/account");

    const saveBtn = await screen.findByRole("button", { name: /^save$/i });
    expect(saveBtn).toBeDisabled();
    expect(screen.getByText("No unsaved changes")).toBeInTheDocument();
    expect(screen.queryByText("Modified")).not.toBeInTheDocument();

    // Modify bio
    const bioTextarea = screen.getByPlaceholderText(/tell other users about your flying style/i);
    fireEvent.change(bioTextarea, { target: { value: "FPV freestyle pilot and bando crusher." } });

    // Now save button is enabled and shows modified indicators
    await waitFor(() => {
      expect(saveBtn).not.toBeDisabled();
    });
    expect(screen.getByText("Unsaved changes")).toBeInTheDocument();
    expect(screen.getByText("Modified")).toBeInTheDocument();

    // Click Save
    fireEvent.click(saveBtn);

    // After save, success notice appears and button becomes disabled
    await waitFor(() => {
      expect(screen.getByText("Saved successfully")).toBeInTheDocument();
      expect(saveBtn).toBeDisabled();
    });
  });

  it("renders Privacy section with Download My Data button and handles OTP export flow", async () => {
    renderApp("/account");

    expect(await screen.findByText("Privacy")).toBeInTheDocument();
    const downloadBtn = screen.getByTestId("download-data-button");
    expect(downloadBtn).toBeInTheDocument();
    expect(downloadBtn).toHaveTextContent("Download My Data");

    // Click button to open OTP modal
    fireEvent.click(downloadBtn);

    expect(await screen.findByRole("heading", { name: /verify your email/i })).toBeInTheDocument();
    expect(screen.getByText(/we sent a single-use verification code to/i)).toBeInTheDocument();

    // Verify modal inputs are rendered
    const inputs = screen.getAllByRole("textbox");
    const otpInputs = inputs.filter((el) => (el as HTMLInputElement).maxLength === 1);
    expect(otpInputs.length).toBe(6);

    // Enter valid 6-digit code
    for (let i = 0; i < 6; i++) {
      fireEvent.change(otpInputs[i], { target: { value: "123456"[i] } });
    }

    // Modal closes and success message appears
    await waitFor(() => {
      expect(screen.queryByRole("heading", { name: /verify your email/i })).not.toBeInTheDocument();
      expect(screen.getByText(/your data download .* has started/i)).toBeInTheDocument();
    });
  });

  it("polls and automatically initiates data download on Device A when export is approved on Device B", async () => {
    let polled = false;
    renderApp("/account", {
      transportOptions: {
        authService: {
          sendEmailOtp: () => ({
            success: true,
            challengeId: "export-cross-device-challenge",
            cooldownSeconds: 20,
            email: "alex@quadsmith.dev",
          }),
          getLoginChallengeStatus: () => {
            if (!polled) {
              polled = true;
              return { status: LoginChallengeStatus.PENDING };
            }
            return {
              status: LoginChallengeStatus.APPROVED,
              verifyResponse: {
                success: true,
                purpose: EmailOtpPurpose.EXPORT_DATA,
                email: "alex@quadsmith.dev",
                message: "Data export authorized",
                exportToken: "mock-signed-export-token",
                downloadUrl: "/api/account/export?token=mock-signed-export-token",
                filename: "quadsmith-data-fpv_pilot-2026-10-14.zip",
              },
            };
          },
        },
      },
    });

    const downloadBtn = await screen.findByTestId("download-data-button");
    fireEvent.click(downloadBtn);

    expect(await screen.findByRole("heading", { name: /verify your email/i })).toBeInTheDocument();

    // Device A polls and auto-completes download without typing OTP
    await waitFor(
      () => {
        expect(
          screen.queryByRole("heading", { name: /verify your email/i }),
        ).not.toBeInTheDocument();
        expect(screen.getByText(/your data download .* has started/i)).toBeInTheDocument();
      },
      { timeout: 8000 },
    );
  });

  it("renders Active Sessions section with This Device badge, session count, and scrollable container", async () => {
    renderApp("/account");

    expect(await screen.findByRole("heading", { name: /active sessions/i })).toBeInTheDocument();
    expect(await screen.findByText(/4 Sessions/i)).toBeInTheDocument();

    // Verify current device card
    expect(screen.getByText("Chrome on macOS")).toBeInTheDocument();
    expect(screen.getByText("This Device")).toBeInTheDocument();
    expect(screen.getByText("Current")).toBeInTheDocument();

    // Verify remote sessions
    expect(screen.getByText("Safari on iOS")).toBeInTheDocument();
    expect(screen.getByText("Firefox on Linux")).toBeInTheDocument();
    expect(screen.getByText("Chrome on Android")).toBeInTheDocument();

    // Bulk sign out button is present
    expect(
      screen.getByRole("button", { name: /sign out all other sessions/i }),
    ).toBeInTheDocument();

    // Verify scrollable container styling
    const scrollContainer = document.querySelector(".session-scroll-list");
    expect(scrollContainer).toBeInTheDocument();
    expect(scrollContainer).toHaveClass("max-h-[224px]");
    expect(scrollContainer).toHaveClass("overflow-y-auto");
  });

  it("allows revoking a single remote session through confirmation modal", async () => {
    renderApp("/account");

    expect(await screen.findByRole("heading", { name: /active sessions/i })).toBeInTheDocument();
    expect(await screen.findByText("Safari on iOS")).toBeInTheDocument();

    // Find revoke buttons (should be 3 since current device has none)
    const revokeButtons = screen.getAllByRole("button", { name: /revoke/i });
    expect(revokeButtons.length).toBe(3);

    // Click revoke on first remote session (Safari on iOS)
    fireEvent.click(revokeButtons[0]);

    // Modal appears
    const modal = await screen.findByRole("dialog", { name: /revoke session/i });
    expect(within(modal).getByText(/safari on ios/i)).toBeInTheDocument();
    expect(within(modal).getByText(/172\.56\.21\.89/i)).toBeInTheDocument();

    // Confirm revocation
    const confirmBtn = within(modal).getByRole("button", { name: /^revoke session$/i });
    fireEvent.click(confirmBtn);

    // Modal closes and session is removed
    await waitFor(() => {
      expect(screen.queryByRole("dialog", { name: /revoke session/i })).not.toBeInTheDocument();
      expect(screen.queryByText("Safari on iOS")).not.toBeInTheDocument();
    });
  });

  it("allows signing out all other sessions through bulk confirmation modal", async () => {
    renderApp("/account");

    expect(await screen.findByRole("heading", { name: /active sessions/i })).toBeInTheDocument();

    const signOutAllBtn = await screen.findByRole("button", {
      name: /sign out all other sessions/i,
    });
    fireEvent.click(signOutAllBtn);

    // Modal appears
    expect(
      await screen.findByRole("dialog", { name: /sign out all other sessions/i }),
    ).toBeInTheDocument();
    expect(
      screen.getByText(
        /all active login sessions on your other devices will be immediately revoked/i,
      ),
    ).toBeInTheDocument();

    // Confirm bulk sign out
    const confirmBtn = screen.getByRole("button", { name: /sign out others/i });
    fireEvent.click(confirmBtn);

    // Modal closes, other sessions removed, current device remains, bulk button disappears
    await waitFor(() => {
      expect(
        screen.queryByRole("dialog", { name: /sign out all other sessions/i }),
      ).not.toBeInTheDocument();
      expect(screen.getByText("Chrome on macOS")).toBeInTheDocument();
      expect(screen.queryByText("Safari on iOS")).not.toBeInTheDocument();
      expect(screen.queryByText("Firefox on Linux")).not.toBeInTheDocument();
      expect(screen.queryByText("Chrome on Android")).not.toBeInTheDocument();
      expect(
        screen.queryByRole("button", { name: /sign out all other sessions/i }),
      ).not.toBeInTheDocument();
    });
  });

  describe("banned account behavior", () => {
    beforeEach(() => {
      mockUser.activeBan = {
        uuid: "018f0000-0000-7000-0000-000000000055",
        userUuid: mockUser.uuid,
        reason: "Repeated spam and abusive language in reviews",
        notes: "Internal staff note",
        createdAt: { seconds: BigInt(Math.floor(Date.now() / 1000)) },
        expiresAt: { seconds: BigInt(Math.floor(Date.now() / 1000) + 86400 * 7) },
      };
    });

    afterEach(() => {
      delete mockUser.activeBan;
    });

    it("displays prominent ban warning banner, disables handle change, and locks email modifications while keeping bio editable and sessions functional", async () => {
      renderApp("/account");

      // Verify Ban Banner
      expect(await screen.findByText("Account Suspended")).toBeInTheDocument();
      expect(
        screen.getByText(/repeated spam and abusive language in reviews/i),
      ).toBeInTheDocument();
      expect(screen.getByText("TEMPORARY")).toBeInTheDocument();

      // Verify Handle Locked
      expect(screen.getByText("Handle locked")).toBeInTheDocument();
      expect(screen.queryByRole("button", { name: /^change$/i })).not.toBeInTheDocument();

      // Verify Add Email is hidden
      expect(screen.queryByRole("button", { name: /add email/i })).not.toBeInTheDocument();

      // Verify Make Primary and Remove buttons are absent, replaced by Read-only status
      expect(screen.getByText("Read-only (Suspended)")).toBeInTheDocument();
      expect(screen.queryByRole("button", { name: /make primary/i })).not.toBeInTheDocument();
      expect(
        screen.queryByRole("button", {
          name: /remove alex\.secondary@example\.com/i,
        }),
      ).not.toBeInTheDocument();

      // Verify Bio is still editable and saveable
      const bioTextarea = screen.getByPlaceholderText(/tell other users about your flying style/i);
      expect(bioTextarea).toBeEnabled();
      fireEvent.change(bioTextarea, { target: { value: "Updated bio while restricted." } });
      const saveBtn = screen.getByRole("button", { name: /^save$/i });
      expect(saveBtn).toBeEnabled();

      // Verify Active Sessions is still rendered
      expect(screen.getByRole("heading", { name: /active sessions/i })).toBeInTheDocument();
      expect(
        screen.getByRole("button", { name: /sign out all other sessions/i }),
      ).toBeInTheDocument();
    });

    it("renders AccountRestrictedPage when banned user accesses other pages", async () => {
      renderApp("/builds");

      expect(
        await screen.findByRole("heading", { name: /account access restricted/i }),
      ).toBeInTheDocument();
      expect(
        screen.getByText(/your quadsmith account has been suspended by community moderation/i),
      ).toBeInTheDocument();
      expect(screen.getByText("Repeated spam and abusive language in reviews")).toBeInTheDocument();
      expect(
        screen.getAllByRole("link", { name: /account settings/i }).length,
      ).toBeGreaterThanOrEqual(1);
    });
  });
});
