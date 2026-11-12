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

// The calendar is the one events view that fills the screen. These run in a
// real browser on purpose: the row sizing is pure layout (grid track sizing
// against the viewport), which jsdom-based unit tests cannot observe.
test.describe('Events calendar layout', () => {
  test.use({ viewport: { width: 1440, height: 900 } });

  // Height of the first day cell of each of the six week rows, plus how far
  // the page's scroll container overflows and how wide the grid is relative
  // to the content area.
  async function measure(page: import('@playwright/test').Page) {
    return page.evaluate(() => {
      const main = document.querySelector('main')!;
      const grid = document.querySelector('[data-testid="calendar-grid"]')!;
      const cells = [...grid.children].slice(7);
      const rows = [0, 1, 2, 3, 4, 5].map((r) => cells[r * 7]!.getBoundingClientRect().height);
      const style = getComputedStyle(main);
      const contentWidth = main.clientWidth - parseFloat(style.paddingLeft) - parseFloat(style.paddingRight);
      return {
        rows,
        overflow: main.scrollHeight - main.clientHeight,
        widthRatio: grid.getBoundingClientRect().width / contentWidth,
      };
    });
  }

  async function openCalendar(page: import('@playwright/test').Page) {
    await page
      .getByRole('button', { name: /events|termine/i })
      .first()
      .click();
    const calTab = page.getByRole('tab', { name: /calendar|kalender/i });
    await calTab.waitFor({ state: 'visible', timeout: NAV_TIMEOUT });
    await calTab.click();
    await expect(page.getByTestId('calendar-grid')).toBeVisible({ timeout: 6_000 });
  }

  test('fills the viewport, and a busy day grows only its own week', async ({ page }) => {
    await login(page);
    await openCalendar(page);

    // Go three years back: a month the demo data has no events in, so the
    // layout under test doesn't depend on what the seed puts in "this month".
    const prev = page.getByRole('button', { name: /vorheriger monat|previous month/i });
    for (let i = 0; i < 36; i++) await prev.click();
    const now = new Date();
    const target = new Date(now.getFullYear(), now.getMonth() - 36, 15);
    const date = `${target.getFullYear()}-${String(target.getMonth() + 1).padStart(2, '0')}-15`;

    // Sparse month: the six week rows share the free height equally, taller
    // than the 76px minimum, and the calendar ends where the page does.
    await expect.poll(async () => (await measure(page)).rows[0]!, { timeout: 6_000 }).toBeGreaterThan(76);
    const sparse = await measure(page);
    expect(Math.max(...sparse.rows) - Math.min(...sparse.rows)).toBeLessThanOrEqual(1);
    expect(sparse.overflow).toBeLessThanOrEqual(1);
    expect(sparse.widthRatio).toBeGreaterThan(0.9);

    // Put six events on the 15th of that month (the demo backend accepts
    // them through the same API the app uses), then re-enter the page so the
    // event list is refetched; the selected month is kept.
    await page.evaluate(async (d) => {
      for (let i = 0; i < 6; i++) {
        const res = await fetch('/api/v1/teams/t_a/events', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          credentials: 'include',
          body: JSON.stringify({ type: 'event', title: 'Layout-Test ' + (i + 1), date: d, startTime: `1${i}:00` }),
        });
        if (!res.ok) throw new Error('create event failed: ' + res.status);
      }
    }, date);

    await page
      .getByRole('button', { name: /^(start|home)$/i })
      .first()
      .click();
    await openCalendar(page);
    await expect(page.getByText('Layout-Test 1').first()).toBeVisible({ timeout: 8_000 });

    // Busy month: exactly one week row is taller than the others -- the
    // other five keep the height they had (one busy day must not inflate
    // every week), and every row is still at least the shared height.
    const busy = await measure(page);
    const shared = Math.min(...busy.rows);
    expect(Math.abs(shared - Math.min(...sparse.rows))).toBeLessThanOrEqual(1);
    expect(busy.rows.filter((h) => h > shared + 1)).toHaveLength(1);
    // The page scrolls by no more than what that one week grew.
    expect(busy.overflow).toBeLessThanOrEqual(Math.max(...busy.rows) - shared + 2);
  });
});
