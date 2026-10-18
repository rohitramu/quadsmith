import { Page } from "@playwright/test";

export interface StabilityOptions {
  fixedTime?: string;
  disableAnimations?: boolean;
}

export async function preparePageForSnapshot(
  page: Page,
  options: StabilityOptions = {},
): Promise<void> {
  const { fixedTime = "2026-10-15T12:00:00Z", disableAnimations = true } = options;

  // Freeze time clock for deterministic relative dates and timestamps
  try {
    await page.clock.setFixedTime(new Date(fixedTime));
  } catch {
    // Clock may already be installed
  }

  // Disable CSS animations, keyframes, transitions, and smooth scrolling.
  // Use page.evaluate instead of page.addStyleTag because Playwright's WebKit
  // addStyleTag implementation listens for document securitypolicyviolation
  // events and falsely rejects if an external image redirect triggers img-src CSP.
  if (disableAnimations) {
    await page.evaluate(() => {
      if (document.getElementById("qs-stability-style")) return;
      const style = document.createElement("style");
      style.id = "qs-stability-style";
      style.textContent = `
        *, *::before, *::after {
          -moz-animation: none !important;
          -webkit-animation: none !important;
          animation: none !important;
          -moz-transition: none !important;
          -webkit-transition: none !important;
          transition: none !important;
          scroll-behavior: auto !important;
        }
        input, textarea {
          caret-color: transparent !important;
        }
      `;
      document.head.appendChild(style);
    });
  }

  // Ensure web fonts are completely loaded
  await page.evaluate(() => document.fonts?.ready);

  // Small micro-settle delay for layout completion
  await page.waitForTimeout(200);
}
