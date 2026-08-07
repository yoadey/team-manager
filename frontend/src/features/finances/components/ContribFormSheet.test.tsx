import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { ContribFormSheet } from './ContribFormSheet';

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

function makeApp(formOverrides: Record<string, unknown> = {}) {
  const app = {
    api: {},
    state: {
      primaryColor: '#1565C0',
      activeTeamId: 't1',
    },
    saveContrib: vi.fn(),
    deleteContrib: vi.fn(),
    openTxForm: vi.fn(),
  };
  mockUseApp.mockReturnValue(app as unknown as ReturnType<typeof useApp>);
  mockUseFinanceOverviewQuery.mockReturnValue({ data: { transactions: [] } });
  return { app, formInitial: { id: 'c1', label: '', amount: '', description: '', dueDate: '', archived: false, ...formOverrides } };
}

describe('ContribFormSheet', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('renders label and amount fields', () => {
    const { app, formInitial } = makeApp();
    render(<ContribFormSheet app={app as never} sheet={{ formInitial } as never} />);
    expect(screen.getByPlaceholderText('z. B. Mitgliedsbeitrag Januar 2026')).toBeTruthy();
  });

  it('caps the label input at 255 characters matching the backend limit', () => {
    const { app, formInitial } = makeApp();
    render(<ContribFormSheet app={app as never} sheet={{ formInitial } as never} />);
    const input = screen.getByPlaceholderText('z. B. Mitgliedsbeitrag Januar 2026') as HTMLInputElement;
    expect(input.maxLength).toBe(255);
  });

  it('shows label error when label is empty on blur', async () => {
    const { app, formInitial } = makeApp({ label: '' });
    render(<ContribFormSheet app={app as never} sheet={{ formInitial } as never} />);
    const input = screen.getByPlaceholderText('z. B. Mitgliedsbeitrag Januar 2026');
    fireEvent.blur(input);
    await waitFor(() => {
      expect(screen.getByText('Bezeichnung fehlt.')).toBeTruthy();
    });
  });

  it('shows amount required error when amount is blank on blur', async () => {
    const { app, formInitial } = makeApp({ label: 'Test', amount: '' });
    render(<ContribFormSheet app={app as never} sheet={{ formInitial } as never} />);
    const amountInputs = screen.getAllByRole('spinbutton');
    fireEvent.blur(amountInputs[0]!);
    await waitFor(() => {
      expect(screen.getByText('Betrag fehlt.')).toBeTruthy();
    });
  });

  it('shows error when amount is negative', async () => {
    const { app, formInitial } = makeApp({ label: 'Test', amount: '-5' });
    render(<ContribFormSheet app={app as never} sheet={{ formInitial } as never} />);
    const amountInputs = screen.getAllByRole('spinbutton');
    fireEvent.blur(amountInputs[0]!);
    await waitFor(() => {
      expect(screen.getByText('Betrag muss größer als 0 € sein.')).toBeTruthy();
    });
  });

  it('clears amount error when amount is valid', async () => {
    const { app, formInitial } = makeApp({ label: 'Test', amount: '120' });
    render(<ContribFormSheet app={app as never} sheet={{ formInitial } as never} />);
    const amountInputs = screen.getAllByRole('spinbutton');
    fireEvent.blur(amountInputs[0]!);
    await waitFor(() => {
      expect(screen.queryByText('Betrag muss größer als 0 € sein.')).toBeNull();
    });
  });

  it('submit button is disabled when form is empty', () => {
    const { app, formInitial } = makeApp({ label: '', amount: '' });
    render(<ContribFormSheet app={app as never} sheet={{ formInitial } as never} />);
    const btn = screen.getByRole('button', { name: /Änderungen speichern/i });
    expect(btn).toBeDisabled();
  });

  it('submit button is enabled when form is valid', () => {
    const { app, formInitial } = makeApp({ label: 'Monatsbeitrag', amount: '15' });
    render(<ContribFormSheet app={app as never} sheet={{ formInitial } as never} />);
    const btn = screen.getByRole('button', { name: /Änderungen speichern/i });
    expect(btn).not.toBeDisabled();
  });

  it('calls saveContrib when save button is clicked and form is valid', async () => {
    const { app, formInitial } = makeApp({ label: 'Monatsbeitrag', amount: '15' });
    render(<ContribFormSheet app={app as never} sheet={{ formInitial } as never} />);
    fireEvent.click(screen.getByRole('button', { name: /Änderungen speichern/i }));
    await waitFor(() => {
      expect(app.saveContrib).toHaveBeenCalled();
    });
  });

  it('renders a due-date field', () => {
    const { app, formInitial } = makeApp();
    render(<ContribFormSheet app={app as never} sheet={{ formInitial } as never} />);
    expect(document.querySelector('input[type="date"]')).toBeTruthy();
  });

  it('calls deleteContrib with the contribution id when the delete button is clicked', () => {
    const { app, formInitial } = makeApp();
    render(<ContribFormSheet app={app as never} sheet={{ formInitial } as never} />);
    fireEvent.click(screen.getByText('Löschen'));
    expect(app.deleteContrib).toHaveBeenCalledWith('c1');
  });

  it('renders a description field', () => {
    const { app, formInitial } = makeApp();
    render(<ContribFormSheet app={app as never} sheet={{ formInitial } as never} />);
    expect(screen.getByPlaceholderText('z. B. Deckt Startgeld und Transport ab.')).toBeTruthy();
  });

  it('shows an archive-not-a-delete action for a non-archived contribution', () => {
    const { app, formInitial } = makeApp();
    render(<ContribFormSheet app={app as never} sheet={{ formInitial } as never} />);
    expect(screen.getByText('Archivieren')).toBeTruthy();
    expect(screen.queryByText('Aus dem Archiv holen')).toBeNull();
  });

  it('toggling archive flips the button label to un-archive and saves with archived: true', async () => {
    const { app, formInitial } = makeApp({ label: 'Monatsbeitrag', amount: '15' });
    render(<ContribFormSheet app={app as never} sheet={{ formInitial } as never} />);
    fireEvent.click(screen.getByText('Archivieren'));
    expect(screen.getByText('Aus dem Archiv holen')).toBeTruthy();
    fireEvent.click(screen.getByRole('button', { name: /Änderungen speichern/i }));
    await waitFor(() => {
      expect(app.saveContrib).toHaveBeenCalledWith(expect.objectContaining({ archived: true }));
    });
  });

  it('shows an archived notice and un-archive action for an already-archived contribution', () => {
    const { app, formInitial } = makeApp({ archived: true });
    render(<ContribFormSheet app={app as never} sheet={{ formInitial } as never} />);
    expect(screen.getByText('Aus dem Archiv holen')).toBeTruthy();
  });

  it('lists transactions linked to this contribution', () => {
    const { app, formInitial } = makeApp();
    mockUseFinanceOverviewQuery.mockReturnValue({
      data: {
        transactions: [
          { id: 'tx1', title: 'Bar erhalten', date: '2026-06-02', amount: 15, contributionId: 'c1' },
          { id: 'tx2', title: 'Unrelated', date: '2026-06-03', amount: 5, contributionId: 'other' },
        ],
      },
    });
    render(<ContribFormSheet app={app as never} sheet={{ formInitial } as never} />);
    expect(screen.getByText('Bar erhalten')).toBeTruthy();
    expect(screen.queryByText('Unrelated')).toBeNull();
  });

  it('shows an empty state when no transaction is linked yet', () => {
    const { app, formInitial } = makeApp();
    render(<ContribFormSheet app={app as never} sheet={{ formInitial } as never} />);
    expect(screen.getByText('Keine verknüpften Buchungen')).toBeTruthy();
  });
});
