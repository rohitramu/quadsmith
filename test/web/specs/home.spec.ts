import { test, expect } from "@playwright/test";
import { preparePageForSnapshot } from "../helpers/stability";
import { loginAs, SEEDED_PILOT } from "../helpers/auth";

test.describe("Home Page & Core Layout", () => {
  test("renders home hero and navigation header", async ({ page }) => {
    await loginAs(page, SEEDED_PILOT);
    await page.goto("/");
    await preparePageForSnapshot(page);

    // Verify presence of core landing landmarks
    await expect(page.locator("header")).toBeVisible();
    await expect(page.getByRole("heading", { level: 1 })).toBeVisible();

    // Verify main content container is rendered
    await expect(page.locator("main")).toBeVisible();

    // Visual snapshot comparison
    await expect(page).toHaveScreenshot("home-page.png", {
      fullPage: true,
    });
  });
});
