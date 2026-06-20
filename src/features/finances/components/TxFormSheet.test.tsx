import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { TxFormSheet } from './TxFormSheet';

vi.mock('@/context/AppContext', () => ({
  useApp: vi.fn(),
  useAppActions: vi.fn().mockReturnValue({}),
}));

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
    // German locale is the default
    expect(screen.getByText('Einnahme')).toBeTruthy();
    expect(screen.getByText('Ausgabe')).toBeTruthy();
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
