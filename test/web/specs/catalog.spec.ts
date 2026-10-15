import { test, expect } from "@playwright/test";
import { preparePageForSnapshot } from "../helpers/stability";
import { loginAs, SEEDED_PILOT } from "../helpers/auth";

test.describe("Component Catalog & Data Table", () => {
  test("renders motors catalog table with columns and product rows in dark mode", async ({
    page,
  }) => {
    await loginAs(page, SEEDED_PILOT);
    await page.goto("/components/hardware/motors");
    await preparePageForSnapshot(page, { theme: "dark" });

    // Table should be visible with rows
    await expect(page.locator("table")).toBeVisible();
    await expect(page.locator("table tbody tr").first()).toBeVisible();

    // Visual snapshot comparison (Dark Mode)
    await expect(page).toHaveScreenshot("catalog-motors-dark.png", {
      fullPage: true,
    });
  });

  test("filters catalog in real time via search input", async ({ page }) => {
    await loginAs(page, SEEDED_PILOT);
    await page.goto("/components/hardware/motors");
    await preparePageForSnapshot(page, { theme: "dark" });

    // Find and type into search filter input
    const filterInput = page.getByPlaceholder(/filter components/i);
    await expect(filterInput).toBeVisible();
    await filterInput.fill('manufacturer.contains("BETAFPV")');
    await filterInput.press("Enter");
    await page.waitForTimeout(500);
    await expect(page.locator("table tbody")).toContainText("BETAFPV");

    // Visual snapshot of filtered catalog
    await expect(page).toHaveScreenshot("catalog-filtered-dark.png", {
      fullPage: true,
    });
  });

  test("renders catalog cleanly in light mode", async ({ page }) => {
    await loginAs(page, SEEDED_PILOT);
    await page.goto("/components/hardware/motors");
    await preparePageForSnapshot(page, { theme: "light" });

    await expect(page.locator("table")).toBeVisible();
    await expect(page.locator("table tbody tr").first()).toBeVisible({ timeout: 10000 });

    // Visual snapshot comparison (Light Mode)
    await expect(page).toHaveScreenshot("catalog-motors-light.png", {
      fullPage: true,
    });
  });
});
