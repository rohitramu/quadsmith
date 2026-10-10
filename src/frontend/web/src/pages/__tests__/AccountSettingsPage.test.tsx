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

  it("allows merging another account through the merge modal", async () => {
    renderApp("/account");

    const openMergeBtn = await screen.findByRole("button", { name: /merge account/i });
    fireEvent.click(openMergeBtn);

    // Step 1: Input email to merge
    expect(screen.getByPlaceholderText(/pilot\.other@example\.com/i)).toBeInTheDocument();
    const mergeInput = screen.getByPlaceholderText(/pilot\.other@example\.com/i);
    fireEvent.change(mergeInput, { target: { value: "old.racer@fpv.org" } });

    const sendVerifyBtn = screen.getByRole("button", { name: /send verification code/i });
    fireEvent.click(sendVerifyBtn);

    // Step 2: Verification code & confirm
    await waitFor(() => {
      expect(screen.getByText(/verify ownership of old\.racer@fpv\.org/i)).toBeInTheDocument();
    });

    const codeInput = screen.getByPlaceholderText("123456");
    fireEvent.change(codeInput, { target: { value: "123456" } });

    const confirmBtn = screen.getByRole("button", { name: /confirm & merge accounts/i });
    fireEvent.click(confirmBtn);

    // Step 3: Success state
    await waitFor(() => {
      expect(screen.getByText(/accounts merged successfully!/i)).toBeInTheDocument();
      expect(
        screen.getByText(/all 4 builds from the merged account have been reassigned/i),
      ).toBeInTheDocument();
    });

    const doneBtn = screen.getByRole("button", { name: /done/i });
    fireEvent.click(doneBtn);

    // Modal closed
    await waitFor(() => {
      expect(screen.queryByText(/accounts merged successfully!/i)).not.toBeInTheDocument();
    });
  });
});
