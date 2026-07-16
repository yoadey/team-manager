import type { Page } from '@playwright/test';

// Timeout for waiting on the primary content of a page reached by clicking a
// nav item — the click triggers a route change, a mock-API fetch (120-320ms
// simulated latency, see VITE_MOCK_DELAY_MIN/MAX), and a render. 8s is
// normally plenty, but proved too tight under GitHub Actions' shared/
// contended runners: this exact class of wait (tablist/searchbox visibility
// right after a nav click) was the sole source of recurring E2E flakiness
// across several review rounds, while the actual app behavior underneath it
// checked out fine on every rerun. Matches login()'s existing 15s wait below
// for the same reason.
export const NAV_TIMEOUT = 15_000;

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
