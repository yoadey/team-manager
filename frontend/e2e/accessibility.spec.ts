import { test, expect } from '@playwright/test';
import { login, gotoFresh } from './helpers';

test.describe('Accessibility smoke tests', () => {
  test('login page has no missing focus targets', async ({ page }) => {
    await gotoFresh(page);
    // Tab through interactive elements — all should be reachable
    await page.keyboard.press('Tab');
    const focused = await page.evaluate(() => document.activeElement?.tagName);
    expect(focused).toBeTruthy();
  });

  test('navigation items have aria-current="page" on active route', async ({ page }) => {
    await login(page);

    // After login the active nav item should have aria-current="page"
    const activeNavItem = page.locator('[aria-current="page"]');
    await expect(activeNavItem.first()).toBeVisible({ timeout: 8_000 });
  });

  test('skip-to-main link exists', async ({ page }) => {
    await login(page);

    const skipLink = page.getByText(/skip to main|zum hauptinhalt/i);
    await expect(skipLink).toBeAttached();
  });

  test('login boots the app without hitting an error boundary', async ({ page }) => {
    // Was previously a tautological assertion: `.not.toBeVisible(...).catch(() => {})`
    // swallows the one rejection this assertion exists to surface, so it could
    // never fail even if the app actually crashed into ErrorBoundary's
    // DefaultFallback (components/ErrorBoundary.tsx) on login. There is no
    // dev/test-only hook to deliberately trigger the boundary and check its
    // retry button's focusability (what this test's name used to claim), so
    // this is scoped to what it can actually verify without adding one.
    await login(page);
    await expect(page.getByText(/something went wrong|schiefgelaufen|konnte nicht geladen werden/i)).not.toBeVisible();
  });
});
