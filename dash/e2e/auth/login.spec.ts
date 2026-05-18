import { expect, test } from '@playwright/test';
import { LoginPage } from '../pages/login.page';

test.describe('Authentication Flow', () => {
  let loginPage: LoginPage;

  test.beforeEach(async ({ page }) => {
    loginPage = new LoginPage(page);
    await loginPage.goto();
  });

  test('should show the login page with all elements', async ({ page }) => {
    await expect(page).toHaveTitle(/Opus/);
    await expect(loginPage.welcomeHeading).toBeVisible();
    await expect(loginPage.emailInput).toBeVisible();
    await expect(loginPage.passwordInput).toBeVisible();
    await expect(loginPage.signInButton).toBeVisible();
    await expect(loginPage.githubButton).toBeVisible();
    await expect(loginPage.googleButton).toBeVisible();
  });

  test('should login successfully with valid credentials (mock)', async ({
    page,
  }) => {
    await loginPage.login('admin@opus.ai', 'any-password');

    // Should be redirected to the agent page
    await expect(page).toHaveURL(/\/agent/);

    // Header should show user name
    await expect(page.getByText('Pak Bos')).toBeVisible();
  });

  test('should show error with invalid credentials', async () => {
    await loginPage.login('wrong@example.com', 'password');
    await loginPage.expectLoginError();
  });

  test('should logout successfully', async ({ page }) => {
    // First login
    await loginPage.login('admin@opus.ai', 'password');
    await expect(page).toHaveURL(/\/agent/);

    // Logout
    await loginPage.logout();

    // Should be back at login
    await expect(page).toHaveURL(/\/login/);
  });
});
