import { test, expect } from "@playwright/test";

test.describe("PWA", () => {
  test("should serve manifest.webmanifest", async ({ page }) => {
    const response = await page.goto("/manifest.webmanifest");
    expect(response?.status()).toBe(200);
    const json = await response?.json();
    expect(json).toHaveProperty("short_name", "Opus");
  });

  test("should have a service worker script", async ({ page }) => {
    // In dev mode, sw might not be active, but we check if it's served
    const response = await page.goto("/sw.js");
    expect(response?.status()).toBe(200);
  });

  test("should render offline page", async ({ page }) => {
    await page.goto("/offline");
    await expect(page.getByText(/you are offline/i)).toBeVisible();
  });
});
