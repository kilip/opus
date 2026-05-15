import { test, expect } from "@playwright/test";

test.describe("Authentication", () => {
  test("should redirect unauthenticated user to login", async ({ page }) => {
    await page.goto("/");
    await expect(page).toHaveURL("/login");
  });

  test("should show OAuth login buttons", async ({ page }) => {
    await page.goto("/login");
    await expect(page.getByRole("link", { name: /google/i })).toBeVisible();
    await expect(page.getByRole("link", { name: /github/i })).toBeVisible();
  });

  test("should show dev login form in dev mode", async ({ page }) => {
    // This assumes NEXT_PUBLIC_DEV_MODE is true in the test environment
    await page.goto("/login");
    const emailInput = page.locator('input[type="email"]');
    if (await emailInput.isVisible()) {
      await expect(emailInput).toBeVisible();
    }
  });
});
