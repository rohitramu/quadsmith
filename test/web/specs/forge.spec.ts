import { test, expect } from "@playwright/test";
import { preparePageForSnapshot } from "../helpers/stability";
import { loginAs, SEEDED_PILOT } from "../helpers/auth";

test.describe("The Forge (Build Wizard)", () => {
  test("renders Step 1 (Frame Chassis) with unlocked components", async ({
    page,
  }) => {
    await loginAs(page, SEEDED_PILOT);
    await page.goto("/builds/new");
    // Step 1: Frame should be active and visible
    await expect(page.getByText(/Frame Chassis/i).first()).toBeVisible();
    await preparePageForSnapshot(page);

    // Visual snapshot comparison
    await expect(page).toHaveScreenshot("forge-step1.png", {
      fullPage: true,
    });
  });

  test("loads existing build for editing on /builds/:id/edit", async ({ page }) => {
    await loginAs(page, SEEDED_PILOT);
    await page.goto("/builds/long-range-explorer/edit");
    await expect(page.getByText(/Frame Chassis/i).first()).toBeVisible({ timeout: 15000 });
    await expect(page.locator("#selected-frame-item")).toBeVisible({ timeout: 15000 });
    await preparePageForSnapshot(page);

    // Visual snapshot comparison (Edit Mode)
    await expect(page).toHaveScreenshot("forge-edit-build.png", {
      fullPage: true,
    });
  });
});
