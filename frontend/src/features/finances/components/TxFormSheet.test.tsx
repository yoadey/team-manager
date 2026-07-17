import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { TxFormSheet } from './TxFormSheet';

vi.mock('@/context/AppContext', () => {
  const useApp = vi.fn();
  return { useApp };
});

vi.mock('../hooks/useFinanceQueries', () => ({
  useFinanceOverviewQuery: vi.fn(),
}));

import { useApp } from '@/context/AppContext';
import { useFinanceOverviewQuery } from '../hooks/useFinanceQueries';
const mockUseApp = vi.mocked(useApp);
const mockUseFinanceOverviewQuery = useFinanceOverviewQuery as ReturnType<typeof vi.fn>;

function makeApp(formOverrides: Record<string, unknown> = {}, transactions: { category: string }[] = []) {
  mockUseFinanceOverviewQuery.mockReturnValue({ data: { transactions } });
  const app = {
    api: {},
    state: {
      primaryColor: '#4285F4',
      activeTeamId: 't1',
    },
    saveTx: vi.fn(),
    deleteTx: vi.fn(),
    askConfirm: vi.fn(),
  };
  mockUseApp.mockReturnValue(app as unknown as ReturnType<typeof useApp>);
  return { app, formInitial: { type: 'income', title: '', amount: '', category: '', id: null, ...formOverrides } };
}

function makeSheet(mode: 'create' | 'edit', formInitial: Record<string, unknown>) {
  return { mode, formInitial } as never;
}

describe('TxFormSheet', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('renders income and expense type buttons', () => {
    const { app, formInitial } = makeApp();
    render(<TxFormSheet app={app as never} sheet={makeSheet('create', formInitial)} />);
    expect(screen.getByText('Einnahme')).toBeTruthy();
    expect(screen.getByText('Ausgabe')).toBeTruthy();
  });

  it('exposes the selected type via aria-pressed for screen-reader users', () => {
    const { app, formInitial } = makeApp({ type: 'income' });
    render(<TxFormSheet app={app as never} sheet={makeSheet('create', formInitial)} />);
    expect(screen.getByText('Einnahme').closest('button')).toHaveAttribute('aria-pressed', 'true');
    expect(screen.getByText('Ausgabe').closest('button')).toHaveAttribute('aria-pressed', 'false');
  });

  it('clicking income type button selects income', () => {
    const { app, formInitial } = makeApp({ type: 'expense' });
    render(<TxFormSheet app={app as never} sheet={makeSheet('create', formInitial)} />);
    const btn = screen.getByText('Einnahme').closest('button')!;
    fireEvent.click(btn);
    expect(btn).toHaveAttribute('aria-pressed', 'true');
  });

  it('clicking expense type button selects expense', () => {
    const { app, formInitial } = makeApp({ type: 'income' });
    render(<TxFormSheet app={app as never} sheet={makeSheet('create', formInitial)} />);
    const btn = screen.getByText('Ausgabe').closest('button')!;
    fireEvent.click(btn);
    expect(btn).toHaveAttribute('aria-pressed', 'true');
  });

  it('caps title and category inputs at 255 characters matching the backend limit', () => {
    const { app, formInitial } = makeApp();
    render(<TxFormSheet app={app as never} sheet={makeSheet('create', formInitial)} />);
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
    const { app, formInitial } = makeApp();
    render(<TxFormSheet app={app as never} sheet={makeSheet('create', formInitial)} />);
    const categoryInput = document.querySelector('input[name="category"]') as HTMLInputElement;
    expect(categoryInput.parentElement?.tagName).toBe('LABEL');
  });

  it('shows title error when title is blank on blur', async () => {
    const { app, formInitial } = makeApp({ title: '' });
    render(<TxFormSheet app={app as never} sheet={makeSheet('create', formInitial)} />);
    const titleInput = screen.getByPlaceholderText('z. B. Mitgliedsbeiträge');
    fireEvent.blur(titleInput);
    await waitFor(() => {
      expect(screen.getByText('Bezeichnung fehlt.')).toBeTruthy();
    });
  });

  it('clears title error when title has value on blur', async () => {
    const { app, formInitial } = makeApp({ title: 'Mitgliedsbeiträge' });
    render(<TxFormSheet app={app as never} sheet={makeSheet('create', formInitial)} />);
    const titleInput = screen.getByPlaceholderText('z. B. Mitgliedsbeiträge');
    fireEvent.blur(titleInput);
    await waitFor(() => {
      expect(screen.queryByText('Bezeichnung fehlt.')).toBeNull();
    });
  });

  it('shows amount error when amount is blank on blur', async () => {
    const { app, formInitial } = makeApp({ title: 'Test', amount: '' });
    render(<TxFormSheet app={app as never} sheet={makeSheet('create', formInitial)} />);
    const amountInput = screen.getByRole('spinbutton');
    fireEvent.blur(amountInput);
    await waitFor(() => {
      expect(screen.getByText('Betrag fehlt.')).toBeTruthy();
    });
  });

  it('shows amount error when amount is negative', async () => {
    const { app, formInitial } = makeApp({ title: 'Test', amount: '-5' });
    render(<TxFormSheet app={app as never} sheet={makeSheet('create', formInitial)} />);
    const amountInput = screen.getByRole('spinbutton');
    fireEvent.blur(amountInput);
    await waitFor(() => {
      expect(screen.getByText('Betrag muss größer als 0 € sein.')).toBeTruthy();
    });
  });

  it('shows amount error when amount is zero', async () => {
    const { app, formInitial } = makeApp({ title: 'Test', amount: '0' });
    render(<TxFormSheet app={app as never} sheet={makeSheet('create', formInitial)} />);
    const amountInput = screen.getByRole('spinbutton');
    fireEvent.blur(amountInput);
    await waitFor(() => {
      expect(screen.getByText('Betrag muss größer als 0 € sein.')).toBeTruthy();
    });
  });

  it('clears amount error when amount is a valid positive number', async () => {
    const { app, formInitial } = makeApp({ title: 'Test', amount: '50' });
    render(<TxFormSheet app={app as never} sheet={makeSheet('create', formInitial)} />);
    const amountInput = screen.getByRole('spinbutton');
    fireEvent.blur(amountInput);
    await waitFor(() => {
      expect(screen.queryByText('Betrag muss größer als 0 € sein.')).toBeNull();
    });
  });

  it('shows amount error and disables submit when amount exceeds the €1,000,000 cap', async () => {
    const { app, formInitial } = makeApp({ title: 'Test', amount: '5000000' });
    render(<TxFormSheet app={app as never} sheet={makeSheet('create', formInitial)} />);
    const amountInput = screen.getByRole('spinbutton');
    fireEvent.blur(amountInput);
    await waitFor(() => {
      expect(screen.getByText('Betrag darf höchstens 1.000.000 € betragen.')).toBeTruthy();
      expect(screen.getByText(/erfassen/i).closest('button')).toBeDisabled();
    });
  });

  it('exposes a max attribute on the amount input matching the backend cap', () => {
    const { app, formInitial } = makeApp();
    render(<TxFormSheet app={app as never} sheet={makeSheet('create', formInitial)} />);
    const amountInput = screen.getByRole('spinbutton') as HTMLInputElement;
    expect(amountInput.max).toBe('1000000');
  });

  it('submit button is disabled when form is empty', () => {
    const { app, formInitial } = makeApp({ title: '', amount: '' });
    render(<TxFormSheet app={app as never} sheet={makeSheet('create', formInitial)} />);
    const btn = screen.getByRole('button', { name: /Buchung erfassen/i });
    expect(btn).toBeDisabled();
  });

  it('submit button is enabled when form has title and positive amount', () => {
    const { app, formInitial } = makeApp({ title: 'Mitgliedsbeiträge', amount: '25' });
    render(<TxFormSheet app={app as never} sheet={makeSheet('create', formInitial)} />);
    const btn = screen.getByRole('button', { name: /Buchung erfassen/i });
    expect(btn).not.toBeDisabled();
  });

  it('does NOT show delete button in create mode', () => {
    const { app, formInitial } = makeApp({ title: 'Test', amount: '10' });
    render(<TxFormSheet app={app as never} sheet={makeSheet('create', formInitial)} />);
    expect(screen.queryByText('Buchung löschen')).toBeNull();
  });

  it('shows delete button in edit mode', () => {
    const { app, formInitial } = makeApp({ title: 'Test', amount: '10', id: 'tx-1' });
    render(<TxFormSheet app={app as never} sheet={makeSheet('edit', formInitial)} />);
    expect(screen.getByText('Buchung löschen')).toBeTruthy();
  });

  it('shows "Buchung erfassen" submit label in create mode', () => {
    const { app, formInitial } = makeApp({ title: '', amount: '' });
    render(<TxFormSheet app={app as never} sheet={makeSheet('create', formInitial)} />);
    expect(screen.getByRole('button', { name: /Buchung erfassen/i })).toBeTruthy();
  });

  it('shows "Änderungen speichern" submit label in edit mode', () => {
    const { app, formInitial } = makeApp({ title: 'Test', amount: '10', id: 'tx-1' });
    render(<TxFormSheet app={app as never} sheet={makeSheet('edit', formInitial)} />);
    expect(screen.getByRole('button', { name: /Änderungen speichern/i })).toBeTruthy();
  });

  it('shows category chips when transactions have categories', () => {
    const { app, formInitial } = makeApp({ title: '', amount: '' }, [
      { category: 'Mitgliedsbeiträge' },
      { category: 'Sponsoring' },
      { category: 'Mitgliedsbeiträge' },
    ]);
    render(<TxFormSheet app={app as never} sheet={makeSheet('create', formInitial)} />);
    expect(screen.getByText('Mitgliedsbeiträge')).toBeTruthy();
    expect(screen.getByText('Sponsoring')).toBeTruthy();
  });

  it('clicking a category chip updates category value', () => {
    const { app, formInitial } = makeApp({ title: '', amount: '', category: '' }, [{ category: 'Sponsoring' }]);
    render(<TxFormSheet app={app as never} sheet={makeSheet('create', formInitial)} />);
    const chip = screen.getByText('Sponsoring');
    fireEvent.click(chip);
    const input = document.querySelector('input[name="category"]') as HTMLInputElement;
    expect(input.value).toBe('Sponsoring');
  });

  it('sorts category chips using the current locale rather than a hardcoded one', async () => {
    const i18n = await import('@/i18n');
    const spy = vi.spyOn(i18n, 'getIntlLocale').mockReturnValue('en-US');
    const localeCompareSpy = vi.spyOn(String.prototype, 'localeCompare');
    const { app, formInitial } = makeApp({ title: '', amount: '' }, [{ category: 'Alpha' }, { category: 'Beta' }]);
    render(<TxFormSheet app={app as never} sheet={makeSheet('create', formInitial)} />);

    expect(spy).toHaveBeenCalled();
    const usedLocaleArgs = localeCompareSpy.mock.calls.map((c) => c[1]);
    expect(usedLocaleArgs).toContain('en-US');
    expect(usedLocaleArgs).not.toContain('de');

    spy.mockRestore();
    localeCompareSpy.mockRestore();
  });

  it('does not render category chips when transactions have no categories', () => {
    const { app, formInitial } = makeApp({ title: '', amount: '' }, []);
    render(<TxFormSheet app={app as never} sheet={makeSheet('create', formInitial)} />);
    const buttons = screen.getAllByRole('button');
    expect(buttons).toHaveLength(3);
  });

  it('clicking delete button in edit mode calls askConfirm', () => {
    const { app, formInitial } = makeApp({ title: 'Alte Buchung', amount: '100', id: 'tx-42' });
    render(<TxFormSheet app={app as never} sheet={makeSheet('edit', formInitial)} />);
    fireEvent.click(screen.getByText('Buchung löschen'));
    expect(app.askConfirm).toHaveBeenCalled();
  });
});
