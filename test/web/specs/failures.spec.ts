import { test, expect } from "@playwright/test";
import { preparePageForSnapshot } from "../helpers/stability";
import { loginAs, SEEDED_PILOT } from "../helpers/auth";
import { simulateNetworkDrop, simulateServerError } from "../helpers/faults";

test.describe("Failure Modes, Input Validations & Fault Injection", () => {
  test("enforces email format validation and prevents submission on malformed inputs", async ({
    page,
  }) => {
    await page.goto("/login");
    await preparePageForSnapshot(page, { theme: "dark" });

    const emailInput = page.getByRole("textbox", { name: /email/i });
    await emailInput.fill("invalid-email-format");

    const submitBtn = page.getByRole("button", { name: /send verification code/i });
    await expect(submitBtn).toBeVisible();
    await submitBtn.click();
    await expect(
      page.getByText(/valid email|invalid email/i).first(),
    ).toBeVisible({ timeout: 5000 });

    // Visual snapshot comparison (Invalid Input Error State)
    await expect(page).toHaveScreenshot("login-invalid-email-error-dark.png");
  });

  test("suppresses clicks on disabled Forge wizard controls", async ({ page }) => {
    await loginAs(page, SEEDED_PILOT);
    await page.goto("/builds/new");
    await preparePageForSnapshot(page, { theme: "dark" });

    // In The Forge, "Next" or step advance should be disabled until a frame is selected
    const nextBtn = page.getByRole("button", { name: /next|continue/i });
    if (await nextBtn.isVisible()) {
      await expect(nextBtn).toBeDisabled();
      // Attempt click (force click should not advance step)
      await nextBtn.click({ force: true });
      // Should still be on Step 1
      await expect(page.getByText(/Frame Chassis/i).first()).toBeVisible();
    }
  });

  test("handles network connection drop during catalog retrieval gracefully with retry", async ({
    page,
  }) => {
    await loginAs(page, SEEDED_PILOT);

    // Abort motor list requests
    const restoreRoute = await simulateNetworkDrop(
      page,
      "**/quadsmith.api.MotorService/ListMotors",
    );

    await page.goto("/components/hardware/motors");
    await preparePageForSnapshot(page, { theme: "dark" });

    // Verify error state / empty state renders without white-screening
    await expect(
      page.getByText(/failed|error|unable to load|something went wrong|no motors found/i).first(),
    ).toBeVisible({ timeout: 15000 });

    // Restore network connection and reload to verify recovery
    await restoreRoute();
    await page.reload();
    await expect(page.locator('[data-testid="catalog-card"]').first()).toBeVisible({ timeout: 10000 });
  });

  test("renders graceful error card when build evaluator returns 500 server error", async ({
    page,
  }) => {
    await loginAs(page, SEEDED_PILOT);

    // Inject 500 server error on build evaluation
    await simulateServerError(
      page,
      "**/quadsmith.api.EvaluatorService/EvaluateBuild",
      500,
      "evaluator backend engine down",
    );

    await page.goto("/builds/bando-basher-5-inch");
    await preparePageForSnapshot(page, { theme: "dark" });

    // Verify page renders without crashing
    await expect(page.getByRole("heading", { name: /Bando Basher/i })).toBeVisible();

    // Visual snapshot comparison (Fault Injected Evaluator State)
    await expect(page).toHaveScreenshot("build-evaluator-server-error-dark.png");
  });
});
