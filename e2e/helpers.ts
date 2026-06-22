import type { Page } from '@playwright/test';

/**
 * Navigate to the app on a clean origin (no persisted localStorage/session).
 *
 * Note: `localStorage` can only be accessed once a real document is loaded, so
 * the clear runs *after* `goto` — clearing on the initial `about:blank` page
 * throws `SecurityError: Access is denied for this document`. A reload then
 * boots the app against the cleared storage.
 */
export async function gotoFresh(page: Page) {
  await page.goto('/');
  await page.evaluate(() => localStorage.clear());
  await page.reload();
}

/** Open the app fresh, click the first login provider and wait for the app shell. */
export async function login(page: Page) {
  await gotoFresh(page);
  // Wait for providers to load and click the first one (Vereins-SSO / mock)
  const providerBtn = page.locator('button, [role="button"]').filter({ hasText: 'Vereins-SSO' }).first();
  await providerBtn.waitFor({ timeout: 10_000 });
  await providerBtn.click();
  // App shell is ready when the main nav is visible
  await page.locator('[aria-label="Hauptnavigation"], [aria-label="Main navigation"]').first().waitFor({
    state: 'visible',
    timeout: 15_000,
  });
}
