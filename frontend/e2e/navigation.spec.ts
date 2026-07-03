import { test, expect } from '@playwright/test';
import { login, NAV_TIMEOUT } from './helpers';

test.describe('Navigation', () => {
  test.beforeEach(async ({ page }) => {
    await login(page);
  });

  test('navigates to Events page', async ({ page }) => {
    await page
      .getByRole('button', { name: /events|termine/i })
      .first()
      .click();
    await expect(page.getByRole('tablist').first()).toBeVisible({ timeout: NAV_TIMEOUT });
  });

  test('navigates to Members page', async ({ page }) => {
    await page
      .getByRole('button', { name: /members|mitglieder/i })
      .first()
      .click();
    // Members page has a search input
    await expect(page.getByRole('searchbox')).toBeVisible({ timeout: NAV_TIMEOUT });
  });

  test('Members page search filters the list', async ({ page }) => {
    await page
      .getByRole('button', { name: /members|mitglieder/i })
      .first()
      .click();
    const search = page.getByRole('searchbox');
    await search.waitFor({ state: 'visible', timeout: NAV_TIMEOUT });

    const memberRows = page.locator('[data-testid="member-row"]');
    await expect(memberRows.first()).toBeVisible({ timeout: NAV_TIMEOUT });
    const countBeforeFilter = await memberRows.count();

    // Type a query that will match nobody
    await search.fill('zzz_no_match_xyz');
    await expect(memberRows).toHaveCount(0);
    await expect(search).toHaveValue('zzz_no_match_xyz');

    // Clearing the query restores the full, unfiltered list.
    await search.fill('');
    await expect(memberRows).toHaveCount(countBeforeFilter);
  });

  test('tabs switch between list/calendar/absences on Events page', async ({ page }) => {
    await page
      .getByRole('button', { name: /events|termine/i })
      .first()
      .click();
    const tablist = page.getByRole('tablist').first();
    await tablist.waitFor({ state: 'visible', timeout: NAV_TIMEOUT });

    // Click Calendar tab
    const calTab = page.getByRole('tab', { name: /calendar|kalender/i });
    await calTab.click();
    await expect(calTab).toHaveAttribute('aria-selected', 'true');
  });
});
