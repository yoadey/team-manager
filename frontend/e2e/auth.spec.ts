import { test, expect } from '@playwright/test';
import { login, gotoFresh } from './helpers';

test.describe('Authentication', () => {
  test('shows login screen on first visit', async ({ page }) => {
    // Clear localStorage so there's no persisted session
    await gotoFresh(page);

    // Login card should be visible
    await expect(page.getByText('Teamverwaltung').first()).toBeVisible({ timeout: 10_000 });
    await expect(page.locator('button, [role="button"]').filter({ hasText: 'Passwort' })).toBeVisible();
  });

  test('logs in with the password provider and shows home page', async ({ page }) => {
    await login(page);

    // After login the home page should show
    await expect(page.getByRole('navigation')).toBeVisible();
  });
});
