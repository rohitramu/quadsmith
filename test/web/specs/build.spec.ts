import { test, expect } from "@playwright/test";
import { preparePageForSnapshot } from "../helpers/stability";
import { loginAs, SEEDED_PILOT } from "../helpers/auth";

test.describe("Build Profile Showcase & Battery Simulator", () => {
  const BUILD_URL = "/builds/bando-basher-5-inch";

  test("renders build profile, hardware specs, and battery simulator", async ({
    page,
  }) => {
    await loginAs(page, SEEDED_PILOT);
    await page.goto(BUILD_URL);
    await preparePageForSnapshot(page);

    // Verify build name and hardware list
    await expect(page.getByRole("heading", { name: /Bando Basher/i })).toBeVisible();
    await expect(page.getByRole("heading", { name: "Hardware" })).toBeVisible();

    // Visual snapshot comparison
    await expect(page).toHaveScreenshot("build-profile.png", {
      fullPage: true,
    });
  });
});
