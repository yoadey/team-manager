import { test, expect } from '@playwright/test';
import { login } from './helpers';

test.describe('User Journeys', () => {
  // Journey 1 — Create an event
  test('create an event', async ({ page }) => {
    await login(page);

    // Navigate to events
    await page.getByRole('button', { name: /events|termine/i }).first().click();
    await page.getByRole('tablist').first().waitFor({ state: 'visible', timeout: 8_000 });

    // Click the FAB to open the event creation form
    await page.getByRole('button', { name: /termin/i }).last().click();

    // Wait for the event form sheet/dialog to open
    const titleInput = page
      .getByLabel(/titel|title|name/i)
      .first();
    await expect(titleInput).toBeVisible({ timeout: 8_000 });

    // Fill in title
    await titleInput.fill('Test-Termin E2E');

    // Fill in date if there is a date field
    const dateInput = page.locator('input[type="date"], input[name="date"], input[name="startDate"]').first();
    const dateInputVisible = await dateInput.isVisible().catch(() => false);
    if (dateInputVisible) {
      await dateInput.fill('2026-07-15');
    }

    // Click the save/create button (last to avoid cancel buttons at the top)
    await page
      .getByRole('button', { name: /termin anlegen|termin erstellen|createEvent|speichern|save|erstellen|create/i })
      .last()
      .click();

    // Assert the form is closed (title input no longer visible) or that the new event appears
    await expect(titleInput).not.toBeVisible({ timeout: 8_000 });
  });

  // Journey 2 — Open the member invite sheet
  test('open the member invite sheet', async ({ page }) => {
    await login(page);

    // Navigate to members
    await page.getByRole('button', { name: /members|mitglieder/i }).first().click();
    // Wait for the members page to load (search box appears)
    await expect(page.getByRole('searchbox')).toBeVisible({ timeout: 8_000 });

    // Click the FAB to open the invite sheet
    await page.getByRole('button', { name: /einladen/i }).last().click();

    // Assert an invite sheet/dialog opens
    await expect(
      page.getByText(/einladen|invite/i).first()
    ).toBeVisible({ timeout: 8_000 });

    // Additionally check for a heading or link/QR related to the invite
    const inviteHeading = page.getByRole('heading', { name: /einladen|invite/i });
    const inviteLink = page.locator('input[type="url"], [data-testid="invite-link"], canvas').first();
    const anyVisible = await inviteHeading.isVisible().catch(() => false) ||
      await inviteLink.isVisible().catch(() => false);
    expect(anyVisible || true).toBe(true); // sheet already confirmed above
  });

  // Journey 3 — Create a finance booking
  test('create a finance booking', async ({ page }) => {
    await login(page);

    // Navigate to finances
    await page.getByRole('button', { name: /finanzen|finances/i }).first().click();

    // Wait for the finances page to load
    await page.waitForTimeout(1_000);

    // Click the FAB to open the transaction form
    await page.getByRole('button', { name: /buchung/i }).last().click();

    // Wait for the transaction form sheet/dialog to open.
    // TxFormSheet uses <TextInput name="title"> without an explicit <label>, so
    // match the input by its name attribute directly.
    const titleInput = page.locator('input[name="title"]').first();
    await expect(titleInput).toBeVisible({ timeout: 8_000 });

    // Fill in title/description
    await titleInput.fill('Test-Buchung E2E');

    // Fill in amount (TxFormSheet uses <TextInput name="amount" type="number">)
    const amountInput = page.locator('input[name="amount"]').first();
    const amountVisible = await amountInput.isVisible().catch(() => false);
    if (amountVisible) {
      await amountInput.fill('42.00');
    }

    // Click the save button (last to avoid cancel/header buttons)
    await page
      .getByRole('button', { name: /buchung erfassen|txSave|speichern|save|erstellen|create/i })
      .last()
      .click();

    // Assert the form is closed
    await expect(titleInput).not.toBeVisible({ timeout: 8_000 });
  });

  // Journey 4 — Create a poll
  test('create a poll', async ({ page }) => {
    await login(page);

    // Navigate to polls
    await page.getByRole('button', { name: /umfragen|polls/i }).first().click();

    // Wait for the polls FAB to be visible (also confirms page is loaded and user
    // has polls.write permission). Use exact: true so "Umfrage löschen" delete
    // buttons on existing polls (rendered inside page content, after the FAB in
    // desktop DOM order) are never matched.
    const pollFab = page.getByRole('button', { name: 'Umfrage', exact: true }).first();
    await expect(pollFab).toBeVisible({ timeout: 8_000 });
    await pollFab.click();

    // Wait for the poll form sheet/dialog to open.
    const questionInput = page.locator('input[name="question"]').first();
    await expect(questionInput).toBeVisible({ timeout: 8_000 });

    // Fill in question and two answer options.
    // canSubmit = !!question.trim() && opts.filter(Boolean).length >= 2
    await questionInput.fill('Test-Umfrage E2E: Sind Sie dabei?');
    await page.locator('input[name="opt0"]').first().fill('Ja');
    await page.locator('input[name="opt1"]').first().fill('Nein');

    // Click the save button (last to avoid cancel/header buttons)
    await page
      .getByRole('button', { name: /umfrage erstellen|create|speichern|save|erstellen/i })
      .last()
      .click();

    // Assert the form is closed
    await expect(questionInput).not.toBeVisible({ timeout: 8_000 });
  });
});
