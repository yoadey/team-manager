import type { Page } from '@playwright/test';

/** Click the first available login provider and wait for the app shell to appear. */
export async function login(page: Page) {
  await page.goto('/');
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
