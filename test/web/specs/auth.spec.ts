import { test, expect } from "@playwright/test";
import { preparePageForSnapshot } from "../helpers/stability";

test.describe("Authentication Flows", () => {
  test("renders Login Step 1 (email form) without navigation header in dark mode", async ({
    page,
  }) => {
    await page.goto("/login");
    await preparePageForSnapshot(page, { theme: "dark" });

    // Header and navigation should NOT be rendered on login page
    await expect(page.locator("nav")).not.toBeVisible();
    await expect(page.getByRole("textbox", { name: /email/i })).toBeVisible();

    // Visual snapshot comparison (Dark Mode)
    await expect(page).toHaveScreenshot("login-step1-dark.png", {
      fullPage: true,
    });
  });

  test("renders Login Step 1 in light mode", async ({ page }) => {
    await page.goto("/login");
    await preparePageForSnapshot(page, { theme: "light" });

    await expect(page.getByRole("textbox", { name: /email/i })).toBeVisible();

    // Visual snapshot comparison (Light Mode)
    await expect(page).toHaveScreenshot("login-step1-light.png", {
      fullPage: true,
    });
  });

  test("transitions to Step 2 upon email submission and renders OTP entry", async ({
    page,
  }) => {
    await page.goto("/login");
    await preparePageForSnapshot(page, { theme: "dark" });

    const emailInput = page.getByRole("textbox", { name: /email/i });
    await emailInput.fill("pilot@example.com");

    const submitBtn = page.getByRole("button", { name: /continue|send|submit/i });
    if (await submitBtn.isVisible()) {
      await submitBtn.click();
    } else {
      await emailInput.press("Enter");
    }

    // Wait for step 2 (verification code prompt)
    await expect(
      page.getByText(/verification code|check your email|enter the code/i),
    ).toBeVisible({ timeout: 10000 });

    await preparePageForSnapshot(page, { theme: "dark" });

    // Visual snapshot comparison (Step 2 OTP Entry)
    await expect(page).toHaveScreenshot("login-step2-dark.png", {
      fullPage: true,
    });
  });
});
