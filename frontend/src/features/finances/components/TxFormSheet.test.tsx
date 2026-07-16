import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { TxFormSheet } from './TxFormSheet';

vi.mock('@/context/AppContext', () => {
  const useApp = vi.fn();
  return {
    useApp,
    // Actions + selector derive from the per-test useApp mock so migrated
    // atoms (TextInput/TextArea via useAppActions/useAppSelector) resolve.
    useAppActions: vi.fn(() => useApp()),
    useAppSelector: (sel: (s: { form: Record<string, unknown> }) => unknown) => sel(useApp().state),
  };
});

vi.mock('../hooks/useFinanceQueries', () => ({
  useFinanceOverviewQuery: vi.fn(),
}));

import { useApp } from '@/context/AppContext';
import { useFinanceOverviewQuery } from '../hooks/useFinanceQueries';
const mockUseApp = vi.mocked(useApp);
const mockUseFinanceOverviewQuery = useFinanceOverviewQuery as ReturnType<typeof vi.fn>;

function makeApp(
  formOverrides: Record<string, unknown> = {},
  errOverrides: Record<string, string> = {},
  transactions: { category: string }[] = [],
) {
  mockUseFinanceOverviewQuery.mockReturnValue({ data: { transactions } });
  const setFormErrors = vi.fn();
  const app = {
    api: {},
    state: {
      primaryColor: '#4285F4',
      activeTeamId: 't1',
      form: { type: 'income', title: '', amount: '', category: '', id: null, ...formOverrides },
      formErrors: { title: '', amount: '', ...errOverrides },
    },
    setFormErrors,
    setFormVal: vi.fn(),
    onFormInput: vi.fn(),
    saveTx: vi.fn(),
    deleteTx: vi.fn(),
    askConfirm: vi.fn(),
  };
  mockUseApp.mockReturnValue(app as unknown as ReturnType<typeof useApp>);
  return app;
}

describe('TxFormSheet', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('renders income and expense type buttons', () => {
    const app = makeApp();
    const sheet = { mode: 'create' } as never;
    render(<TxFormSheet app={app as never} sheet={sheet} />);
    // German locale is the default
    expect(screen.getByText('Einnahme')).toBeTruthy();
    expect(screen.getByText('Ausgabe')).toBeTruthy();
  });

  it('exposes the selected type via aria-pressed for screen-reader users', () => {
    const app = makeApp({ type: 'income' });
    const sheet = { mode: 'create' } as never;
    render(<TxFormSheet app={app as never} sheet={sheet} />);
    expect(screen.getByText('Einnahme').closest('button')).toHaveAttribute('aria-pressed', 'true');
    expect(screen.getByText('Ausgabe').closest('button')).toHaveAttribute('aria-pressed', 'false');
  });

  it('clicking income type button calls setFormVal with type income', () => {
    const app = makeApp({ type: 'expense' });
    const sheet = { mode: 'create' } as never;
    render(<TxFormSheet app={app as never} sheet={sheet} />);
    fireEvent.click(screen.getByText('Einnahme'));
    expect(app.setFormVal).toHaveBeenCalledWith({ type: 'income' });
  });

  it('clicking expense type button calls setFormVal with type expense', () => {
    const app = makeApp({ type: 'income' });
    const sheet = { mode: 'create' } as never;
    render(<TxFormSheet app={app as never} sheet={sheet} />);
    fireEvent.click(screen.getByText('Ausgabe'));
    expect(app.setFormVal).toHaveBeenCalledWith({ type: 'expense' });
  });

  // Regression test: title/category had no client-side maxLength, matching
  // the backend's 255-char validate.MaxLen bound for both fields.
  it('caps title and category inputs at 255 characters matching the backend limit', () => {
    const app = makeApp();
    const sheet = { mode: 'create' } as never;
    render(<TxFormSheet app={app as never} sheet={sheet} />);
    const titleInput = screen.getByPlaceholderText('z. B. Mitgliedsbeiträge') as HTMLInputElement;
    const categoryInput = document.querySelector('input[name="category"]') as HTMLInputElement;
    expect(titleInput.maxLength).toBe(255);
    expect(categoryInput.maxLength).toBe(255);
  });

  // Regression test: the category field's <Field> used to wrap a <Box> that
  // in turn wrapped the real <input> (plus the datalist/quick-pick chips/hint
  // text), so Field's cloneElement-injected aria-required/aria-invalid/
  // aria-describedby landed on that wrapper Box, not the input a screen
  // reader actually focuses. Field must clone the <input> directly.
  it("renders the category input as Field's direct cloned child, not wrapped in an intermediate element", () => {
    const app = makeApp();
    const sheet = { mode: 'create' } as never;
    render(<TxFormSheet app={app as never} sheet={sheet} />);
    const categoryInput = document.querySelector('input[name="category"]') as HTMLInputElement;
    expect(categoryInput.parentElement?.tagName).toBe('LABEL');
  });

  it('shows title error when title is blank on blur', () => {
    const app = makeApp({ title: '' });
    const sheet = { mode: 'create' } as never;
    render(<TxFormSheet app={app as never} sheet={sheet} />);
    const titleInput = screen.getByPlaceholderText('z. B. Mitgliedsbeiträge');
    fireEvent.blur(titleInput);
    expect(app.setFormErrors).toHaveBeenCalledWith({ title: expect.stringMatching(/\S+/) });
  });

  it('clears title error when title has value on blur', () => {
    const app = makeApp({ title: 'Mitgliedsbeiträge' });
    const sheet = { mode: 'create' } as never;
    render(<TxFormSheet app={app as never} sheet={sheet} />);
    const titleInput = screen.getByPlaceholderText('z. B. Mitgliedsbeiträge');
    fireEvent.blur(titleInput);
    expect(app.setFormErrors).toHaveBeenCalledWith({ title: '' });
  });

  it('shows amount error when amount is blank on blur', () => {
    const app = makeApp({ title: 'Test', amount: '' });
    const sheet = { mode: 'create' } as never;
    render(<TxFormSheet app={app as never} sheet={sheet} />);
    const amountInput = screen.getByRole('spinbutton');
    fireEvent.blur(amountInput);
    expect(app.setFormErrors).toHaveBeenCalledWith({ amount: expect.stringMatching(/\S+/) });
  });

  it('shows amount error when amount is negative', () => {
    const app = makeApp({ title: 'Test', amount: '-5' });
    const sheet = { mode: 'create' } as never;
    render(<TxFormSheet app={app as never} sheet={sheet} />);
    const amountInput = screen.getByRole('spinbutton');
    fireEvent.blur(amountInput);
    expect(app.setFormErrors).toHaveBeenCalledWith({ amount: expect.stringMatching(/\S+/) });
  });

  it('shows amount error when amount is zero', () => {
    const app = makeApp({ title: 'Test', amount: '0' });
    const sheet = { mode: 'create' } as never;
    render(<TxFormSheet app={app as never} sheet={sheet} />);
    const amountInput = screen.getByRole('spinbutton');
    fireEvent.blur(amountInput);
    expect(app.setFormErrors).toHaveBeenCalledWith({ amount: expect.stringMatching(/\S+/) });
  });

  it('clears amount error when amount is a valid positive number', () => {
    const app = makeApp({ title: 'Test', amount: '50' });
    const sheet = { mode: 'create' } as never;
    render(<TxFormSheet app={app as never} sheet={sheet} />);
    const amountInput = screen.getByRole('spinbutton');
    fireEvent.blur(amountInput);
    expect(app.setFormErrors).toHaveBeenCalledWith({ amount: '' });
  });

  // Regression test: the inline blur validator and canSubmit only checked
  // "positive number", unlike the backend's €1,000,000 amount cap enforced
  // at submit time (useFinanceActions.ts's saveTx) -- so typing an over-cap
  // amount showed no inline error and left Save enabled, only failing with a
  // raw toast after clicking it.
  it('shows amount error and disables submit when amount exceeds the €1,000,000 cap', () => {
    const app = makeApp({ title: 'Test', amount: '5000000' });
    const sheet = { mode: 'create' } as never;
    render(<TxFormSheet app={app as never} sheet={sheet} />);
    const amountInput = screen.getByRole('spinbutton');
    fireEvent.blur(amountInput);
    expect(app.setFormErrors).toHaveBeenCalledWith({ amount: expect.stringMatching(/\S+/) });
    expect(screen.getByText(/erfassen|speichern/i).closest('button')).toBeDisabled();
  });

  it('exposes a max attribute on the amount input matching the backend cap', () => {
    const app = makeApp();
    render(<TxFormSheet app={app as never} sheet={{ mode: 'create' } as never} />);
    const amountInput = screen.getByRole('spinbutton') as HTMLInputElement;
    expect(amountInput.max).toBe('1000000');
  });

  it('submit button is disabled when form is empty', () => {
    const app = makeApp({ title: '', amount: '' });
    const sheet = { mode: 'create' } as never;
    render(<TxFormSheet app={app as never} sheet={sheet} />);
    // In create mode the button label is "Buchung erfassen"
    const btn = screen.getByRole('button', { name: /Buchung erfassen/i });
    expect(btn).toBeDisabled();
  });

  it('submit button is enabled when form has title and positive amount', () => {
    const app = makeApp({ title: 'Mitgliedsbeiträge', amount: '25' });
    const sheet = { mode: 'create' } as never;
    render(<TxFormSheet app={app as never} sheet={sheet} />);
    const btn = screen.getByRole('button', { name: /Buchung erfassen/i });
    expect(btn).not.toBeDisabled();
  });

  it('does NOT show delete button in create mode', () => {
    const app = makeApp({ title: 'Test', amount: '10' });
    const sheet = { mode: 'create' } as never;
    render(<TxFormSheet app={app as never} sheet={sheet} />);
    expect(screen.queryByText('Buchung löschen')).toBeNull();
  });

  it('shows delete button in edit mode', () => {
    const app = makeApp({ title: 'Test', amount: '10', id: 'tx-1' });
    const sheet = { mode: 'edit' } as never;
    render(<TxFormSheet app={app as never} sheet={sheet} />);
    expect(screen.getByText('Buchung löschen')).toBeTruthy();
  });

  it('shows "Buchung erfassen" submit label in create mode', () => {
    const app = makeApp({ title: '', amount: '' });
    const sheet = { mode: 'create' } as never;
    render(<TxFormSheet app={app as never} sheet={sheet} />);
    expect(screen.getByRole('button', { name: /Buchung erfassen/i })).toBeTruthy();
  });

  it('shows "Änderungen speichern" submit label in edit mode', () => {
    const app = makeApp({ title: 'Test', amount: '10', id: 'tx-1' });
    const sheet = { mode: 'edit' } as never;
    render(<TxFormSheet app={app as never} sheet={sheet} />);
    expect(screen.getByRole('button', { name: /Änderungen speichern/i })).toBeTruthy();
  });

  it('shows category chips when transactions have categories', () => {
    const app = makeApp({ title: '', amount: '' }, {}, [
      { category: 'Mitgliedsbeiträge' },
      { category: 'Sponsoring' },
      { category: 'Mitgliedsbeiträge' },
    ]);
    const sheet = { mode: 'create' } as never;
    render(<TxFormSheet app={app as never} sheet={sheet} />);
    // Unique categories should appear as chips
    expect(screen.getByText('Mitgliedsbeiträge')).toBeTruthy();
    expect(screen.getByText('Sponsoring')).toBeTruthy();
  });

  it('clicking a category chip calls setFormVal with the category', () => {
    const app = makeApp({ title: '', amount: '', category: '' }, {}, [{ category: 'Sponsoring' }]);
    const sheet = { mode: 'create' } as never;
    render(<TxFormSheet app={app as never} sheet={sheet} />);
    fireEvent.click(screen.getByText('Sponsoring'));
    expect(app.setFormVal).toHaveBeenCalledWith({ category: 'Sponsoring' });
  });

  // Regression test: the category chip sort used to hardcode localeCompare's
  // locale argument to 'de' regardless of the active UI locale, unlike every
  // other locale-aware sort/format helper in the app (which reads
  // getIntlLocale()). Spy on getIntlLocale to prove the sort now consults
  // it instead of a hardcoded value.
  it('sorts category chips using the current locale rather than a hardcoded one', async () => {
    const i18n = await import('@/i18n');
    const spy = vi.spyOn(i18n, 'getIntlLocale').mockReturnValue('en-US');
    const localeCompareSpy = vi.spyOn(String.prototype, 'localeCompare');
    const app = makeApp({ title: '', amount: '' }, {}, [{ category: 'Alpha' }, { category: 'Beta' }]);
    const sheet = { mode: 'create' } as never;
    render(<TxFormSheet app={app as never} sheet={sheet} />);

    expect(spy).toHaveBeenCalled();
    const usedLocaleArgs = localeCompareSpy.mock.calls.map((c) => c[1]);
    expect(usedLocaleArgs).toContain('en-US');
    expect(usedLocaleArgs).not.toContain('de');

    spy.mockRestore();
    localeCompareSpy.mockRestore();
  });

  it('does not render category chips when transactions have no categories', () => {
    const app = makeApp({ title: '', amount: '' }, {}, []);
    const sheet = { mode: 'create' } as never;
    render(<TxFormSheet app={app as never} sheet={sheet} />);
    // Only type buttons (Einnahme, Ausgabe) and the submit button should exist
    const buttons = screen.getAllByRole('button');
    expect(buttons).toHaveLength(3);
  });

  it('shows displayed error text when title error is present', () => {
    const app = makeApp({}, { title: 'Bezeichnung fehlt.' });
    const sheet = { mode: 'create' } as never;
    render(<TxFormSheet app={app as never} sheet={sheet} />);
    expect(screen.getByText('Bezeichnung fehlt.')).toBeTruthy();
  });

  it('shows displayed error text when amount error is present', () => {
    const app = makeApp({}, { amount: 'Betrag fehlt.' });
    const sheet = { mode: 'create' } as never;
    render(<TxFormSheet app={app as never} sheet={sheet} />);
    expect(screen.getByText('Betrag fehlt.')).toBeTruthy();
  });

  it('clicking delete button in edit mode calls askConfirm', () => {
    const app = makeApp({ title: 'Alte Buchung', amount: '100', id: 'tx-42' });
    const sheet = { mode: 'edit' } as never;
    render(<TxFormSheet app={app as never} sheet={sheet} />);
    fireEvent.click(screen.getByText('Buchung löschen'));
    expect(app.askConfirm).toHaveBeenCalled();
  });
});
