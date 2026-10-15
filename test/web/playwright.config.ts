import { defineConfig, devices } from "@playwright/test";

const projects = [
  {
    name: "chromium-desktop",
    use: {
      ...devices["Desktop Chrome"],
      viewport: { width: 1280, height: 800 },
    },
  },
  {
    name: "chromium-mobile",
    use: {
      ...devices["Pixel 7"],
      viewport: { width: 390, height: 844 },
    },
  },
  {
    name: "firefox-desktop",
    use: {
      ...devices["Desktop Firefox"],
      viewport: { width: 1280, height: 800 },
    },
  },
  {
    name: "webkit-desktop",
    use: {
      ...devices["Desktop Safari"],
      viewport: { width: 1280, height: 800 },
    },
  },
  {
    name: "webkit-mobile",
    use: {
      ...devices["iPhone 14"],
      viewport: { width: 390, height: 844 },
    },
  },
];

export default defineConfig({
  testDir: "./specs",
  snapshotDir: "./snapshots",
  timeout: 30000,
  expect: {
    timeout: 15000,
    toHaveScreenshot: {
      maxDiffPixelRatio: 0.005,
      animations: "disabled",
    },
  },
  fullyParallel: true,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 1 : 0,
  // Default to 1 worker per browser profile (projects.length = 5)
  workers: process.env.PLAYWRIGHT_WORKERS
    ? parseInt(process.env.PLAYWRIGHT_WORKERS, 10)
    : projects.length,
  reporter: [
    ["list"],
    [
      "html",
      {
        outputFolder: process.env.PLAYWRIGHT_REPORT_DIR || "playwright-report",
        open: "never",
      },
    ],
  ],
  use: {
    baseURL: process.env.BASE_URL || "http://127.0.0.1:8080",
    trace: "on-first-retry",
    screenshot: "only-on-failure",
  },
  projects,
});
