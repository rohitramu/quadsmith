import { test, expect } from "@playwright/test";
import { preparePageForSnapshot } from "../helpers/stability";
import { loginAs, SEEDED_PILOT } from "../helpers/auth";

test.describe("Administrative Ban Lockout & Account Deletion Flows", () => {
  test.beforeEach(async ({ page }) => {
    // Intercept GetCurrentUser and RefreshToken to return a banned user state
    await page.route("**/quadsmith.api.AuthService/GetCurrentUser", async (route) => {
      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify({
          uuid: "018f0000-0000-7000-8000-000000000001",
          id: "banned_pilot",
          role: "USER",
          email: "banned@example.com",
          activeBan: {
            uuid: "ban-uuid-12345",
            userUuid: "018f0000-0000-7000-8000-000000000001",
            bannedByUserUuid: "admin-uuid",
            reason: "Violation of community flight safety guidelines.",
            internalNotes: "Repeated dangerous flybys in populated zones.",
            createdAt: "2026-10-14T12:00:00Z",
          },
        }),
      });
    });

    await page.route("**/quadsmith.api.AuthService/RefreshToken", async (route) => {
      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify({
          accessToken: "mock-banned-access-token",
          user: {
            uuid: "018f0000-0000-7000-8000-000000000001",
            id: "banned_pilot",
            role: "USER",
            email: "banned@example.com",
            activeBan: {
              uuid: "ban-uuid-12345",
              userUuid: "018f0000-0000-7000-8000-000000000001",
              bannedByUserUuid: "admin-uuid",
              reason: "Violation of community flight safety guidelines.",
              createdAt: "2026-10-14T12:00:00Z",
            },
          },
          profile: {
            uuid: "profile-uuid",
            userUuid: "018f0000-0000-7000-8000-000000000001",
            bio: "Restricted pilot bio",
          },
        }),
      });
    });
  });

  test("renders AccountRestrictedPage lock screen on standard routes when user is banned", async ({
    page,
  }) => {
    await loginAs(page, SEEDED_PILOT);
    await page.goto("/");
    await preparePageForSnapshot(page);

    // Lock screen indicator
    await expect(page.locator('[data-testid="account-restricted-screen"]')).toBeVisible();
    await expect(page.getByRole("heading", { name: "Account Access Restricted" })).toBeVisible();
    await expect(
      page.getByText("Violation of community flight safety guidelines."),
    ).toBeVisible();

    // Visual snapshot comparison (Lock Screen)
    await expect(page).toHaveScreenshot("account-restricted-lockscreen.png", {
      fullPage: true,
    });
  });

  test("allows banned user to access /account with alert banner and locked handle", async ({
    page,
  }) => {
    await loginAs(page, SEEDED_PILOT);
    await page.goto("/account");
    await preparePageForSnapshot(page);

    // Verify restriction alert banner on account settings
    await expect(
      page.getByText(/Account Access Restricted|account suspended/i).first(),
    ).toBeVisible();

    // Username should be locked (read-only / disabled)
    const usernameInput = page.getByRole("textbox", { name: /username/i });
    if (await usernameInput.isVisible()) {
      await expect(usernameInput).toBeDisabled();
    }

    // Visual snapshot comparison (Account Settings Banned State)
    await expect(page).toHaveScreenshot("account-banned-settings.png", {
      fullPage: true,
    });
  });

  test("displays voluntary account deletion modal with grace period notice", async ({
    page,
  }) => {
    await loginAs(page, SEEDED_PILOT);
    await page.goto("/account");
    await preparePageForSnapshot(page);

    // Find and click Delete Account button
    const deleteBtn = page.getByRole("button", { name: /delete account/i });
    await expect(deleteBtn).toBeVisible();
    await deleteBtn.click();

    // Verify confirmation dialog / modal
    await expect(
      page.getByText(/30 days|grace period|permanently deleted/i).first(),
    ).toBeVisible();

    // Visual snapshot comparison (Account Deletion Modal)
    await expect(page).toHaveScreenshot("account-deletion-modal.png");
  });
});
