import { describe, it, expect, beforeEach, vi } from "vitest";
import { screen, fireEvent, waitFor } from "@testing-library/react";
import { renderApp } from "../../test/test-utils";

describe("AccountSettingsPage Component", () => {
  beforeEach(() => {
    localStorage.setItem("qs_access_token", "mock-jwt-token");
    vi.spyOn(window, "confirm").mockImplementation(() => true);
  });

  it("renders account settings with pilot handle, profile link, and logout button", async () => {
    renderApp("/account");

    // Verify Heading & Handle
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

    // Verify Save Changes button
    expect(screen.getByRole("button", { name: /save changes/i })).toBeInTheDocument();
  });

  it("renders associated emails with primary and secondary status badges", async () => {
    renderApp("/account");

    expect(await screen.findByText("Associated Email Addresses")).toBeInTheDocument();
    expect(screen.getByText("Multi-Email Login")).toBeInTheDocument();

    // Both mock emails should be in the table
    await waitFor(() => {
      expect(screen.getAllByText("alex@quadsmith.dev").length).toBeGreaterThan(0);
      expect(screen.getByText("alex.secondary@example.com")).toBeInTheDocument();
    });

    expect(screen.getAllByText("Primary").length).toBeGreaterThan(0);
    expect(screen.getByText("Secondary")).toBeInTheDocument();
    expect(screen.getByText("Make Primary")).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: /remove alex\.secondary@example\.com/i }),
    ).toBeInTheDocument();
  });

  it("allows adding a new email through OTP verification", async () => {
    renderApp("/account");

    const addEmailBtn = await screen.findByRole("button", { name: /add email/i });
    fireEvent.click(addEmailBtn);

    // Form inputs appear
    expect(screen.getByPlaceholderText(/pilot\.secondary@example\.com/i)).toBeInTheDocument();
    const emailInput = screen.getByPlaceholderText(/pilot\.secondary@example\.com/i);
    fireEvent.change(emailInput, { target: { value: "new.pilot@fpv.org" } });

    const sendCodeBtn = screen.getByRole("button", { name: /send code/i });
    fireEvent.click(sendCodeBtn);

    // Code input should appear
    await waitFor(() => {
      expect(screen.getByPlaceholderText("123456")).toBeInTheDocument();
    });

    const codeInput = screen.getByPlaceholderText("123456");
    fireEvent.change(codeInput, { target: { value: "123456" } });

    const verifyBtn = screen.getByRole("button", { name: /verify & associate/i });
    fireEvent.click(verifyBtn);

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

    // Enter email of an existing account
    const emailInput = screen.getByPlaceholderText(/pilot\.secondary@example\.com/i);
    fireEvent.change(emailInput, { target: { value: "old.racer@fpv.org" } });

    // Click Send Code
    const sendCodeBtn = screen.getByRole("button", { name: /send code/i });
    expect(sendCodeBtn).not.toBeDisabled();
    fireEvent.click(sendCodeBtn);

    await waitFor(() => {
      expect(screen.getByText("Verify Account Ownership")).toBeInTheDocument();
      expect(screen.getByText("Existing Account Detected")).toBeInTheDocument();
      expect(screen.getByText("Two Ways to Verify:")).toBeInTheDocument();
      expect(
        screen.getByText(/enter the verification code sent to your email to continue/i),
      ).toBeInTheDocument();
      // Crucial: ensure @old_racer is NOT leaked before ownership is verified
      expect(screen.queryByText(/old_racer/i)).not.toBeInTheDocument();
    });

    // Enter verification code into segmented OTP inputs
    const firstDigit = screen.getByLabelText("Digit 1");
    expect(firstDigit).toBeInTheDocument();
    fireEvent.change(firstDigit, { target: { value: "123456" } });

    // Verify Ownership & Review
    const verifyBtn = screen.getByRole("button", { name: /verify ownership & review/i });
    expect(verifyBtn).not.toBeDisabled();
    fireEvent.click(verifyBtn);

    // Summary screen displayed after ownership check
    await waitFor(() => {
      expect(screen.getByText("Confirm Account Merge")).toBeInTheDocument();
      expect(screen.getByText(/ownership verified for/i)).toBeInTheDocument();
      expect(screen.getByText(/delete username:/i)).toBeInTheDocument();
      expect(screen.getByText(/held for 30 days before being released/i)).toBeInTheDocument();
      expect(screen.getByText(/associate email address/i)).toBeInTheDocument();
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
    });

    const doneBtn = screen.getByRole("button", { name: /done/i });
    fireEvent.click(doneBtn);

    // Modal closes and email is in the list
    await waitFor(() => {
      expect(screen.queryByText(/accounts merged successfully!/i)).not.toBeInTheDocument();
      expect(screen.getByText("old.racer@fpv.org")).toBeInTheDocument();
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
    });
  });
});
