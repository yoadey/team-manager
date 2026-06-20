import { test, expect } from '@playwright/test';
import { login } from './helpers';

test.describe('Authentication', () => {
  test('shows login screen on first visit', async ({ page }) => {
    // Clear localStorage so there's no persisted session
    await page.goto('/');
    await page.evaluate(() => localStorage.clear());
    await page.reload();

    // Login card should be visible
    await expect(page.getByText('Teamverwaltung').first()).toBeVisible({ timeout: 10_000 });
    await expect(page.locator('button, [role="button"]').filter({ hasText: 'Vereins-SSO' })).toBeVisible();
  });

  test('logs in with Vereins-SSO and shows home page', async ({ page }) => {
    await page.evaluate(() => localStorage.clear());
    await login(page);

    // After login the home page should show
    await expect(page.getByRole('navigation')).toBeVisible();
  });
});
