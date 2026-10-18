import { test, expect } from "@playwright/test";
import { preparePageForSnapshot } from "../helpers/stability";
import { loginAs, SEEDED_PILOT } from "../helpers/auth";

test.describe("Product Specification Detail", () => {
  const PRODUCT_URL = "/components/hardware/motors/sub250-1404-4500kv";

  test("renders product details, specs, and media gallery", async ({ page }) => {
    await loginAs(page, SEEDED_PILOT);
    await page.goto(PRODUCT_URL);
    await preparePageForSnapshot(page);

    // Verify product heading and specs
    await expect(page.getByRole("heading", { level: 1, name: /Sub250/i })).toBeVisible();

    // Visual snapshot comparison
    await expect(page).toHaveScreenshot("product-detail.png", {
      fullPage: true,
    });
  });
});
