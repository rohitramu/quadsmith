import { test, expect } from "@playwright/test";
import { preparePageForSnapshot } from "../helpers/stability";
import { loginAs, SEEDED_PILOT } from "../helpers/auth";

test.describe("Product Specification Detail", () => {
  const PRODUCT_URL = "/components/hardware/motors/sub250-1404-4500kv";

  test("renders product details, specs, and media gallery in dark mode", async ({
    page,
  }) => {
    await loginAs(page, SEEDED_PILOT);
    await page.goto(PRODUCT_URL);
    await preparePageForSnapshot(page, { theme: "dark" });

    // Verify product heading and specs
    await expect(page.getByRole("heading", { level: 1, name: /Sub250/i })).toBeVisible();

    // Visual snapshot comparison (Dark Mode)
    await expect(page).toHaveScreenshot("product-detail-dark.png", {
      fullPage: true,
    });
  });

  test("renders product detail page in light mode", async ({ page }) => {
    await loginAs(page, SEEDED_PILOT);
    await page.goto(PRODUCT_URL);
    await preparePageForSnapshot(page, { theme: "light" });

    await expect(page.getByRole("heading", { level: 1, name: /Sub250/i })).toBeVisible();

    // Visual snapshot comparison (Light Mode)
    await expect(page).toHaveScreenshot("product-detail-light.png", {
      fullPage: true,
    });
  });
});
