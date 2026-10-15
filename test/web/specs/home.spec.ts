import { test, expect } from "@playwright/test";
import { preparePageForSnapshot } from "../helpers/stability";
import { loginAs, SEEDED_PILOT } from "../helpers/auth";

test.describe("Home Page & Core Layout", () => {
  test("renders home hero, navigation header, and theme toggle in dark mode", async ({
    page,
  }) => {
    await loginAs(page, SEEDED_PILOT);
    await page.goto("/");
    await preparePageForSnapshot(page, { theme: "dark" });

    // Verify presence of core landing landmarks
    await expect(page.locator("header")).toBeVisible();
    await expect(page.getByRole("heading", { level: 1 })).toBeVisible();

    // Verify main content container is rendered
    await expect(page.locator("main")).toBeVisible();

    // Visual snapshot comparison (Dark Mode)
    await expect(page).toHaveScreenshot("home-page-dark.png", {
      fullPage: true,
    });
  });

  test("renders home page cleanly in light mode", async ({ page }) => {
    await loginAs(page, SEEDED_PILOT);
    await page.goto("/");
    await preparePageForSnapshot(page, { theme: "light" });

    // Verify header navigation elements
    await expect(page.locator("header")).toBeVisible();
    await expect(page.getByRole("heading", { level: 1 })).toBeVisible();
    await expect(page.locator("main")).toBeVisible();

    // Visual snapshot comparison (Light Mode)
    await expect(page).toHaveScreenshot("home-page-light.png", {
      fullPage: true,
    });
  });
});
