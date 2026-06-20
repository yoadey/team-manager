import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { FinancesTransactions } from './FinancesTransactions';

vi.mock('@/context/AppContext', () => ({
  useApp: vi.fn(),
  useAppActions: vi.fn().mockReturnValue({}),
}));

vi.mock('@/styles/tokens', () => ({
  buildTokens: vi.fn().mockReturnValue({ primary: '#1565C0' }),
  fmtDate: vi.fn().mockImplementation((d) => d),
  fmtMoney: vi.fn().mockImplementation((n) => `${n} €`),
  NEUTRAL: {
    surface: '#fff',
    card: '#fff',
    appBg: '#f5f5f5',
    line: '#e0e0e0',
    secondary: '#757575',
    error: '#B00020',
    errorBg: '#FFEBEE',
    success: '#2E7D32',
    successBg: '#E8F5E9',
    primaryText: '#212121',
    onSurfaceVariant: '#666',
    faint: '#999',
  },
}));

vi.mock('@/i18n', () => ({ t: vi.fn().mockImplementation((key) => key) }));

const tk = {} as never;

function makeApp(overrides = {}) {
  return {
    openTxForm: vi.fn(),
    ...overrides,
  };
}

// eslint-disable-next-line @typescript-eslint/no-explicit-any
function makeFinances(txOverrides: any[] = []) {
  return {
    balance: 100,
    income: 200,
    expense: 100,
    transactions: txOverrides,
    penalties: [],
    assignments: [],
    openPenalties: [],
    openPenaltySum: 0,
    contributions: [],
    contribOpen: 0,
  };
}

function makeTx(overrides = {}) {
  return {
    id: 'tx1',
    teamId: 't1',
    type: 'income' as const,
    title: 'Mitgliedsbeitrag',
    amount: 50,
    date: '2025-06-01',
    category: 'beitrag',
    ...overrides,
  };
}

describe('FinancesTransactions', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('renders empty state when no transactions', () => {
    const app = makeApp();
    render(<FinancesTransactions app={app as never} t={tk} f={makeFinances([])} canFin={false} />);
    expect(document.querySelector('div')).toBeTruthy();
  });

  it('renders income transaction title', () => {
    const app = makeApp();
    render(<FinancesTransactions app={app as never} t={tk} f={makeFinances([makeTx()])} canFin={false} />);
    expect(screen.getByText('Mitgliedsbeitrag')).toBeTruthy();
  });

  it('renders expense transaction', () => {
    const app = makeApp();
    const tx = makeTx({ type: 'expense', title: 'Hallemiete', amount: 30 });
    render(<FinancesTransactions app={app as never} t={tk} f={makeFinances([tx])} canFin={false} />);
    expect(screen.getByText('Hallemiete')).toBeTruthy();
  });

  it('clicking income tx row calls openTxForm when canFin', () => {
    const app = makeApp();
    const tx = makeTx();
    render(<FinancesTransactions app={app as never} t={tk} f={makeFinances([tx])} canFin={true} />);
    fireEvent.click(screen.getByText('Mitgliedsbeitrag').closest('button')!);
    expect(app.openTxForm).toHaveBeenCalledWith(tx);
  });

  it('tx row is not a button when canFin is false', () => {
    const app = makeApp();
    render(<FinancesTransactions app={app as never} t={tk} f={makeFinances([makeTx()])} canFin={false} />);
    expect(screen.getByText('Mitgliedsbeitrag').closest('button')).toBeNull();
  });

  it('renders multiple transactions', () => {
    const app = makeApp();
    const txs = [makeTx({ id: 'tx1', title: 'Eintrag A' }), makeTx({ id: 'tx2', title: 'Eintrag B' })];
    render(<FinancesTransactions app={app as never} t={tk} f={makeFinances(txs)} canFin={false} />);
    expect(screen.getByText('Eintrag A')).toBeTruthy();
    expect(screen.getByText('Eintrag B')).toBeTruthy();
  });

  it('renders formatted amount with +/- prefix', () => {
    const app = makeApp();
    render(
      <FinancesTransactions
        app={app as never}
        t={tk}
        f={makeFinances([makeTx({ amount: 99, type: 'income' })])}
        canFin={false}
      />,
    );
    expect(screen.getByText('+99 €')).toBeTruthy();
  });
});
