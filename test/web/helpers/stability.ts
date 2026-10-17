import { Page, expect } from "@playwright/test";

export interface StabilityOptions {
  theme?: "light" | "dark";
  fixedTime?: string;
  disableAnimations?: boolean;
}

export async function preparePageForSnapshot(
  page: Page,
  options: StabilityOptions = {},
): Promise<void> {
  const { theme = "dark", fixedTime = "2026-10-15T12:00:00Z", disableAnimations = true } = options;

  // Freeze time clock for deterministic relative dates and timestamps
  try {
    await page.clock.setFixedTime(new Date(fixedTime));
  } catch {
    // Clock may already be installed
  }

  const applyTheme = () =>
    page.evaluate((targetTheme) => {
      try {
        localStorage.setItem("theme", targetTheme);
        if (targetTheme === "dark") {
          document.documentElement.classList.add("dark");
        } else {
          document.documentElement.classList.remove("dark");
        }
        window.dispatchEvent(new CustomEvent("qs-theme-change", { detail: targetTheme }));
        window.dispatchEvent(
          new StorageEvent("storage", {
            key: "theme",
            newValue: targetTheme,
            storageArea: localStorage,
          }),
        );
      } catch {
        // ignore
      }
    }, theme);

  await applyTheme();

  try {
    if (theme === "dark") {
      await expect(page.locator("html")).toHaveClass(/(^|\s)dark(\s|$)/, { timeout: 2000 });
    } else {
      await expect(page.locator("html")).not.toHaveClass(/(^|\s)dark(\s|$)/, { timeout: 2000 });
    }
  } catch {
    await applyTheme();
    if (theme === "dark") {
      await expect(page.locator("html")).toHaveClass(/(^|\s)dark(\s|$)/, { timeout: 5000 });
    } else {
      await expect(page.locator("html")).not.toHaveClass(/(^|\s)dark(\s|$)/, { timeout: 5000 });
    }
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
