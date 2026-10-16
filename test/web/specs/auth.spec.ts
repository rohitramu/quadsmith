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

    const submitBtn = page.getByRole("button", { name: /send verification code/i });
    if (await submitBtn.isVisible()) {
      await submitBtn.click();
    } else {
      await emailInput.press("Enter");
    }

    // In parallel test runs across browser profiles, multiple workers submit pilot@example.com simultaneously.
    // If rate-limited by email cooldown, wait for cooldown to expire and re-submit.
    const step2Heading = page.getByRole("heading", { name: /verify your email/i });
    const cooldownNotice = page.getByText(/please wait \d+ seconds? before requesting another code/i);

    for (let attempt = 0; attempt < 5; attempt++) {
      if (await step2Heading.isVisible()) {
        break;
      }
      const isCooldown = await cooldownNotice.isVisible({ timeout: 1500 }).catch(() => false);
      if (isCooldown) {
        await page.waitForTimeout(1200);
        await submitBtn.click();
      } else {
        const result = await Promise.race([
          step2Heading.waitFor({ state: "visible", timeout: 2000 }).then(() => "step2").catch(() => null),
          cooldownNotice.waitFor({ state: "visible", timeout: 2000 }).then(() => "cooldown").catch(() => null),
        ]);
        if (result === "step2") {
          break;
        }
        if (result === "cooldown") {
          await page.waitForTimeout(1200);
          await submitBtn.click();
        }
      }
    }

    // Wait for step 2 heading and OTP input
    await expect(step2Heading).toBeVisible({ timeout: 10000 });
    await expect(page.locator('input[autocomplete="one-time-code"]')).toBeVisible({
      timeout: 10000,
    });

    await preparePageForSnapshot(page, { theme: "dark" });

    // Visual snapshot comparison (Step 2 OTP Entry)
    await expect(page).toHaveScreenshot("login-step2-dark.png", {
      fullPage: true,
    });
  });
});
