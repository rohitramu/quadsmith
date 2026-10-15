import { test, expect } from "@playwright/test";
import { preparePageForSnapshot } from "../helpers/stability";
import { loginAs, SEEDED_PILOT } from "../helpers/auth";

test.describe("The Forge (Build Wizard)", () => {
  test("renders Step 1 (Frame Chassis) with unlocked components in dark mode", async ({
    page,
  }) => {
    await loginAs(page, SEEDED_PILOT);
    await page.goto("/builds/new");
    await preparePageForSnapshot(page, { theme: "dark" });

    // Step 1: Frame should be active and visible
    await expect(page.getByText(/Frame/i).first()).toBeVisible();

    // Visual snapshot comparison (Dark Mode)
    await expect(page).toHaveScreenshot("forge-step1-dark.png", {
      fullPage: true,
    });
  });

  test("renders The Forge build wizard in light mode", async ({ page }) => {
    await loginAs(page, SEEDED_PILOT);
    await page.goto("/builds/new");
    await preparePageForSnapshot(page, { theme: "light" });

    await expect(page.getByText(/Frame/i).first()).toBeVisible();

    // Visual snapshot comparison (Light Mode)
    await expect(page).toHaveScreenshot("forge-step1-light.png", {
      fullPage: true,
    });
  });

  test("loads existing build for editing on /builds/:id/edit", async ({ page }) => {
    await loginAs(page, SEEDED_PILOT);
    await page.goto("/builds/long-range-explorer/edit");
    await expect(page.getByText(/Frame/i).first()).toBeVisible({ timeout: 10000 });
    await expect(page.getByRole("button", { name: /save/i })).toBeVisible({ timeout: 10000 });
    await preparePageForSnapshot(page, { theme: "dark" });

    // Visual snapshot comparison (Edit Mode)
    await expect(page).toHaveScreenshot("forge-edit-build-dark.png", {
      fullPage: true,
    });
  });
});
