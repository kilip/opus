import { expect, type Locator, type Page } from '@playwright/test';

/**
 * Page Object Model for the Login Page.
 * Encapsulates selectors and actions for the login flow.
 */
export class LoginPage {
  readonly page: Page;
  readonly emailInput: Locator;
  readonly passwordInput: Locator;
  readonly signInButton: Locator;
  readonly welcomeHeading: Locator;
  readonly githubButton: Locator;
  readonly googleButton: Locator;
  readonly logoutButton: Locator;

  constructor(page: Page) {
    this.page = page;
    this.emailInput = page.getByLabel('Email Address');
    this.passwordInput = page.getByLabel('Password');
    this.signInButton = page.getByRole('button', { name: 'Sign In' });
    this.welcomeHeading = page.getByText('Welcome Back');
    this.githubButton = page.getByRole('button', { name: 'GitHub' });
    this.googleButton = page.getByRole('button', { name: 'Google' });
    this.logoutButton = page.getByTitle('Sign out');
  }

  async goto() {
    await this.page.goto('/login');
  }

  async login(email: string, password: string) {
    await this.emailInput.fill(email);
    await this.passwordInput.fill(password);
    await this.signInButton.click();
  }

  async expectLoginError() {
    await expect(this.page.getByText('Invalid credentials')).toBeVisible();
  }

  async logout() {
    await this.logoutButton.click();
  }
}
