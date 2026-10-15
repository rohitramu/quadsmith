import { Page } from "@playwright/test";

export async function simulateNetworkDrop(
  page: Page,
  urlPattern: string | RegExp,
): Promise<() => Promise<void>> {
  const handler = async (route: any) => {
    await route.abort("failed");
  };
  await page.route(urlPattern, handler);
  return async () => {
    await page.unroute(urlPattern, handler);
  };
}

export async function simulateServerError(
  page: Page,
  urlPattern: string | RegExp,
  status: number = 500,
  errorMessage: string = "Internal server error occurred",
): Promise<() => Promise<void>> {
  const handler = async (route: any) => {
    await route.fulfill({
      status,
      contentType: "application/json",
      body: JSON.stringify({
        code: "internal",
        message: errorMessage,
      }),
    });
  };
  await page.route(urlPattern, handler);
  return async () => {
    await page.unroute(urlPattern, handler);
  };
}

export async function simulateSlowNetwork(
  page: Page,
  urlPattern: string | RegExp,
  delayMs: number = 2000,
): Promise<() => Promise<void>> {
  const handler = async (route: any) => {
    await new Promise((resolve) => setTimeout(resolve, delayMs));
    await route.continue();
  };
  await page.route(urlPattern, handler);
  return async () => {
    await page.unroute(urlPattern, handler);
  };
}
