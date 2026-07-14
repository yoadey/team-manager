import { test, expect } from '@playwright/test';
import { login, NAV_TIMEOUT } from './helpers';

test.describe('Events', () => {
  test.beforeEach(async ({ page }) => {
    await login(page);
    // Navigate to events
    await page
      .getByRole('button', { name: /events|termine/i })
      .first()
      .click();
    await page.getByRole('tablist').first().waitFor({ state: 'visible', timeout: NAV_TIMEOUT });
  });

  test('event list loads and shows upcoming tab selected by default', async ({ page }) => {
    // The scope tabs (upcoming/archive) should be visible in list view.
    // Default locale is German, so the labels are "Anstehend" / "Archiv".
    const upcomingTab = page.getByRole('tab', { name: /anstehend|upcoming/i });
    await expect(upcomingTab).toBeVisible({ timeout: 8_000 });
    await expect(upcomingTab).toHaveAttribute('aria-selected', 'true');
  });

  test('can switch to Past events', async ({ page }) => {
    const pastTab = page.getByRole('tab', { name: /archiv|past|vergangen/i });
    await pastTab.waitFor({ state: 'visible', timeout: 8_000 });
    await pastTab.click();
    await expect(pastTab).toHaveAttribute('aria-selected', 'true');
  });

  test('calendar view renders month grid', async ({ page }) => {
    const calTab = page.getByRole('tab', { name: /calendar|kalender/i });
    await calTab.click();
    await expect(calTab).toHaveAttribute('aria-selected', 'true');
    // Calendar grid should have day cells
    await expect(page.getByTestId('calendar-grid')).toBeVisible({ timeout: 6_000 });
  });
});
