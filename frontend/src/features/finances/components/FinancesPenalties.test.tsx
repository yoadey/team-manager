import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { FinancesPenalties } from './FinancesPenalties';

vi.mock('@/context/AppContext', () => ({
  useApp: vi.fn(),
  useAppActions: vi.fn().mockReturnValue({}),
}));

vi.mock('@/styles/tokens', () => ({
  buildTokens: vi.fn().mockReturnValue({ primary: '#1565C0', onPrimary: '#fff' }),
  fmtDate: vi.fn().mockImplementation((d) => d),
  fmtMoney: vi.fn().mockImplementation((n) => `${n} €`),
  initials: vi.fn().mockImplementation((n: string) => n.slice(0, 2).toUpperCase()),
  NEUTRAL: {
    surface: '#fff',
    card: '#fff',
    line: '#e0e0e0',
    secondary: '#757575',
    error: '#B00020',
    errorBg: '#FFEBEE',
    success: '#2E7D32',
    successBg: '#E8F5E9',
    faint: '#999',
    onSurfaceVariant: '#666',
  },
}));

vi.mock('@/i18n', () => ({ t: vi.fn().mockImplementation((key) => key) }));

const tk = { primary: '#1565C0', onPrimary: '#fff' } as never;

function makeApp(overrides = {}) {
  return {
    openPenaltyCatalog: vi.fn(),
    openPenaltyAssign: vi.fn(),
    setPenaltyPaid: vi.fn(),
    deleteAssignment: vi.fn(),
    ...overrides,
  };
}

function makeFinances(overrides = {}) {
  return {
    balance: 0,
    income: 0,
    expense: 0,
    transactions: [],
    penalties: [],
    assignments: [],
    openPenalties: [],
    openPenaltySum: 25,
    contributions: [],
    contribOpen: 0,
    ...overrides,
  };
}

function makeAssignment(overrides = {}) {
  return {
    id: 'a1',
    teamId: 't1',
    userId: 'u1',
    penaltyId: 'p1',
    paid: false,
    date: '2025-06-01',
    name: 'Anna Müller',
    avatarColor: '#4285F4',
    photo: null,
    label: 'Versäumtes Training',
    amount: 10,
    ...overrides,
  };
}

describe('FinancesPenalties', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('renders empty state when no assignments', () => {
    const app = makeApp();
    render(<FinancesPenalties app={app as never} t={tk} f={makeFinances()} canFin={false} />);
    expect(document.querySelector('div')).toBeTruthy();
  });

  it('renders catalog button and calls openPenaltyCatalog on click', () => {
    const app = makeApp();
    render(<FinancesPenalties app={app as never} t={tk} f={makeFinances()} canFin={false} />);
    const btn = document.querySelector('button')!;
    expect(btn).toBeTruthy();
    fireEvent.click(btn);
    expect(app.openPenaltyCatalog).toHaveBeenCalled();
  });

  it('shows assign button when canFin is true', () => {
    const app = makeApp();
    render(<FinancesPenalties app={app as never} t={tk} f={makeFinances()} canFin={true} />);
    const btns = document.querySelectorAll('button');
    expect(btns.length).toBeGreaterThanOrEqual(2);
  });

  it('hides assign button when canFin is false', () => {
    const app = makeApp();
    render(<FinancesPenalties app={app as never} t={tk} f={makeFinances()} canFin={false} />);
    const btns = document.querySelectorAll('button');
    expect(btns.length).toBe(1);
  });

  it('calls openPenaltyAssign on assign button click', () => {
    const app = makeApp();
    render(<FinancesPenalties app={app as never} t={tk} f={makeFinances()} canFin={true} />);
    const btns = document.querySelectorAll('button');
    fireEvent.click(btns[1]);
    expect(app.openPenaltyAssign).toHaveBeenCalled();
  });

  it('renders assignment member name', () => {
    const app = makeApp();
    render(
      <FinancesPenalties
        app={app as never}
        t={tk}
        f={makeFinances({ assignments: [makeAssignment()] })}
        canFin={false}
      />,
    );
    expect(screen.getByText('Anna Müller')).toBeTruthy();
  });

  it('calls setPenaltyPaid with the toggled value when the button is clicked', () => {
    const app = makeApp();
    render(
      <FinancesPenalties
        app={app as never}
        t={tk}
        f={makeFinances({ assignments: [makeAssignment()] })}
        canFin={true}
      />,
    );
    fireEvent.click(screen.getByText('finances.penaltyPaid'));
    // The assignment is unpaid (makeAssignment defaults paid:false), so the
    // click requests the opposite value explicitly (idempotent set, not toggle).
    expect(app.setPenaltyPaid).toHaveBeenCalledWith('a1', true);
  });

  it('calls deleteAssignment when delete button clicked', () => {
    const app = makeApp();
    render(
      <FinancesPenalties
        app={app as never}
        t={tk}
        f={makeFinances({ assignments: [makeAssignment()] })}
        canFin={true}
      />,
    );
    fireEvent.click(screen.getByLabelText('common.delete'));
    expect(app.deleteAssignment).toHaveBeenCalledWith('a1');
  });

  it('renders paid assignment with different label', () => {
    const app = makeApp();
    render(
      <FinancesPenalties
        app={app as never}
        t={tk}
        f={makeFinances({ assignments: [makeAssignment({ paid: true })] })}
        canFin={true}
      />,
    );
    expect(screen.getByText('finances.penaltyOpen')).toBeTruthy();
  });
});
