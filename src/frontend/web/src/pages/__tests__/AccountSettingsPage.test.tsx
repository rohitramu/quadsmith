import { describe, it, expect, beforeEach } from "vitest";
import { screen } from "@testing-library/react";
import { renderApp } from "../../test/test-utils";

describe("AccountSettingsPage Component", () => {
  beforeEach(() => {
    localStorage.setItem("qs_access_token", "mock-jwt-token");
  });

  it("renders account settings with pilot handle, private email, profile link, and logout button", async () => {
    renderApp("/account");

    // Verify Heading & Handle
    expect(await screen.findByRole("heading", { name: /account settings/i })).toBeInTheDocument();
    expect(screen.getByText(/@fpv_pilot/i)).toBeInTheDocument();
    expect(screen.getByText(/alex@quadsmith\.dev/i)).toBeInTheDocument();

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
});
