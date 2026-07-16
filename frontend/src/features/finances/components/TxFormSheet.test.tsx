import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { TxFormSheet } from './TxFormSheet';

vi.mock('@/context/AppContext', () => {
  const useApp = vi.fn();
  return {
    useApp,
    useAppActions: vi.fn(() => useApp()),
    useAppSelector: (sel: (s: { form: Record<string, unknown> }) => unknown) => sel(useApp().state),
  };
});

import { useApp } from '@/context/AppContext';
const mockUseApp = vi.mocked(useApp);

function makeApp(
  formOverrides: Record<string, unknown> = {},
  errOverrides: Record<string, string> = {},
  transactions: { category: string }[] = [],
) {
  const setFormErrors = vi.fn();
  const app = {
    state: {
      primaryColor: '#4285F4',
      form: { type: 'income', title: '', amount: '', category: '', id: null, ...formOverrides },
      formErrors: { title: '', amount: '', ...errOverrides },
      busy: null,
      finances: { transactions },
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

  it('clicking income type button selects income', () => {
    const app = makeApp({ type: 'expense' });
    const sheet = { mode: 'create' } as never;
    render(<TxFormSheet app={app as never} sheet={sheet} />);
    const btn = screen.getByText('Einnahme').closest('button')!;
    fireEvent.click(btn);
    expect(btn).toHaveAttribute('aria-pressed', 'true');
  });

  it('clicking expense type button selects expense', () => {
    const app = makeApp({ type: 'income' });
    const sheet = { mode: 'create' } as never;
    render(<TxFormSheet app={app as never} sheet={sheet} />);
    const btn = screen.getByText('Ausgabe').closest('button')!;
    fireEvent.click(btn);
    expect(btn).toHaveAttribute('aria-pressed', 'true');
  });

  it('caps title and category inputs at 255 characters matching the backend limit', () => {
    const app = makeApp();
    const sheet = { mode: 'create' } as never;
    render(<TxFormSheet app={app as never} sheet={sheet} />);
    const titleInput = screen.getByPlaceholderText('z. B. Mitgliedsbeiträge') as HTMLInputElement;
    const categoryInput = document.querySelector('input[name="category"]') as HTMLInputElement;
    expect(titleInput.maxLength).toBe(255);
    expect(categoryInput.maxLength).toBe(255);
  });

  it('renders the category input as Field\'s direct cloned child, not wrapped in an intermediate element', () => {
    const app = makeApp();
    const sheet = { mode: 'create' } as never;
    render(<TxFormSheet app={app as never} sheet={sheet} />);
    const categoryInput = document.querySelector('input[name="category"]') as HTMLInputElement;
    expect(categoryInput.parentElement?.tagName).toBe('LABEL');
  });

  it('shows title error when title is blank on blur', async () => {
    const app = makeApp({ title: '' });
    const sheet = { mode: 'create' } as never;
    render(<TxFormSheet app={app as never} sheet={sheet} />);
    const titleInput = screen.getByPlaceholderText('z. B. Mitgliedsbeiträge');
    fireEvent.blur(titleInput);
    await waitFor(() => {
      expect(screen.getByText('Bezeichnung fehlt.')).toBeTruthy();
    });
  });

  it('clears title error when title has value on blur', async () => {
    const app = makeApp({ title: 'Mitgliedsbeiträge' });
    const sheet = { mode: 'create' } as never;
    render(<TxFormSheet app={app as never} sheet={sheet} />);
    const titleInput = screen.getByPlaceholderText('z. B. Mitgliedsbeiträge');
    fireEvent.blur(titleInput);
    await waitFor(() => {
      expect(screen.queryByText('Bezeichnung fehlt.')).toBeNull();
    });
  });

  it('shows amount error when amount is blank on blur', async () => {
    const app = makeApp({ title: 'Test', amount: '' });
    const sheet = { mode: 'create' } as never;
    render(<TxFormSheet app={app as never} sheet={sheet} />);
    const amountInput = screen.getByRole('spinbutton');
    fireEvent.blur(amountInput);
    await waitFor(() => {
      expect(screen.getByText('Betrag fehlt.')).toBeTruthy();
    });
  });

  it('shows amount error when amount is negative', async () => {
    const app = makeApp({ title: 'Test', amount: '-5' });
    const sheet = { mode: 'create' } as never;
    render(<TxFormSheet app={app as never} sheet={sheet} />);
    const amountInput = screen.getByRole('spinbutton');
    fireEvent.blur(amountInput);
    await waitFor(() => {
      expect(screen.getByText('Betrag muss größer als 0 € sein.')).toBeTruthy();
    });
  });

  it('shows amount error when amount is zero', async () => {
    const app = makeApp({ title: 'Test', amount: '0' });
    const sheet = { mode: 'create' } as never;
    render(<TxFormSheet app={app as never} sheet={sheet} />);
    const amountInput = screen.getByRole('spinbutton');
    fireEvent.blur(amountInput);
    await waitFor(() => {
      expect(screen.getByText('Betrag muss größer als 0 € sein.')).toBeTruthy();
    });
  });

  it('clears amount error when amount is a valid positive number', async () => {
    const app = makeApp({ title: 'Test', amount: '50' });
    const sheet = { mode: 'create' } as never;
    render(<TxFormSheet app={app as never} sheet={sheet} />);
    const amountInput = screen.getByRole('spinbutton');
    fireEvent.blur(amountInput);
    await waitFor(() => {
      expect(screen.queryByText('Betrag muss größer als 0 € sein.')).toBeNull();
    });
  });

  it('shows amount error and disables submit when amount exceeds the €1,000,000 cap', async () => {
    const app = makeApp({ title: 'Test', amount: '5000000' });
    const sheet = { mode: 'create' } as never;
    render(<TxFormSheet app={app as never} sheet={sheet} />);
    const amountInput = screen.getByRole('spinbutton');
    fireEvent.blur(amountInput);
    await waitFor(() => {
      expect(screen.getByText('Betrag darf höchstens 1.000.000 € betragen.')).toBeTruthy();
      expect(screen.getByText(/erfassen/i).closest('button')).toBeDisabled();
    });
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
    expect(screen.getByText('Mitgliedsbeiträge')).toBeTruthy();
    expect(screen.getByText('Sponsoring')).toBeTruthy();
  });

  it('clicking a category chip updates category value', () => {
    const app = makeApp({ title: '', amount: '', category: '' }, {}, [{ category: 'Sponsoring' }]);
    const sheet = { mode: 'create' } as never;
    render(<TxFormSheet app={app as never} sheet={sheet} />);
    const chip = screen.getByText('Sponsoring');
    fireEvent.click(chip);
    const input = document.querySelector('input[name="category"]') as HTMLInputElement;
    expect(input.value).toBe('Sponsoring');
  });

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
