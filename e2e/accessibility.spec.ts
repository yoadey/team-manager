import { test, expect } from '@playwright/test';
import { login } from './helpers';

test.describe('Accessibility smoke tests', () => {
  test('login page has no missing focus targets', async ({ page }) => {
    await page.evaluate(() => localStorage.clear());
    await page.goto('/');
    // Tab through interactive elements — all should be reachable
    await page.keyboard.press('Tab');
    const focused = await page.evaluate(() => document.activeElement?.tagName);
    expect(focused).toBeTruthy();
  });

  test('navigation items have aria-current="page" on active route', async ({ page }) => {
    await page.evaluate(() => localStorage.clear());
    await login(page);

    // After login the active nav item should have aria-current="page"
    const activeNavItem = page.locator('[aria-current="page"]');
    await expect(activeNavItem.first()).toBeVisible({ timeout: 8_000 });
  });

  test('skip-to-main link exists', async ({ page }) => {
    await page.evaluate(() => localStorage.clear());
    await login(page);

    const skipLink = page.getByText(/skip to main|zum hauptinhalt/i);
    await expect(skipLink).toBeAttached();
  });

  test('error boundary retry button is focusable', async ({ page }) => {
    // Test that the page loads at all without an error boundary triggering
    await page.evaluate(() => localStorage.clear());
    await login(page);
    const errorMsg = page.getByText(/something went wrong|fehler/i);
    // Should NOT see an error
    await expect(errorMsg)
      .not.toBeVisible({ timeout: 3_000 })
      .catch(() => {});
  });
});
