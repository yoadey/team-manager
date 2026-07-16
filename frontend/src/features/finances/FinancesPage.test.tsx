import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen } from '@testing-library/react';
import { FinancesPage } from './FinancesPage';

vi.mock('@/context/AppContext', () => ({
  useApp: vi.fn(),
  useAppActions: vi.fn().mockReturnValue({}),
}));

// Mocked directly on the hooks module (not just a `@/features/finances`
// barrel re-export) -- FinancesPage.tsx imports `useFinanceOverviewQuery` via
// a relative import, so this must match that module path (see the identical
// comment/pattern in EventsPage.test.tsx/MembersPage.test.tsx).
vi.mock('./hooks/useFinanceQueries', () => ({
  useFinanceOverviewQuery: vi.fn(),
}));

import { useApp } from '@/context/AppContext';
import { useFinanceOverviewQuery } from './hooks/useFinanceQueries';
const mockUseApp = useApp as ReturnType<typeof vi.fn>;
const mockUseFinanceOverviewQuery = useFinanceOverviewQuery as ReturnType<typeof vi.fn>;

function makeFinances() {
  return {
    balance: 1250.5,
    income: 2500.0,
    expense: 1249.5,
    transactions: [],
    penalties: [],
    assignments: [],
    openPenalties: [],
    openPenaltySum: 0,
    contributions: [],
    contribOpen: 0,
  };
}

function makeApp(overrides: Record<string, unknown> = {}) {
  const { finances, ...stateOverrides } = overrides;
  mockUseFinanceOverviewQuery.mockReturnValue({ data: finances ?? null });
  return {
    api: {},
    state: {
      primaryColor: '#4285F4',
      activeTeamId: 't1',
      finTab: 'umsaetze',
      user: { id: 'u1' },
      roles: [],
      ...stateOverrides,
    },
    can: vi.fn().mockReturnValue(false),
    setState: vi.fn(),
    openTxForm: vi.fn(),
    deleteTx: vi.fn(),
    openPenaltyForm: vi.fn(),
    openPenaltyCatalog: vi.fn(),
    deletePenaltyDef: vi.fn(),
    openPenaltyAssign: vi.fn(),
    deleteAssignment: vi.fn(),
    togglePenalty: vi.fn(),
    openContribForm: vi.fn(),
    saveContrib: vi.fn(),
    toggleContribution: vi.fn(),
  };
}

describe('FinancesPage', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('shows spinner when finances is null', () => {
    mockUseApp.mockReturnValue(makeApp({ finances: null }));
    render(<FinancesPage />);
    expect(screen.getByRole('status')).toBeTruthy();
  });

  it('renders balance when finances loaded', () => {
    mockUseApp.mockReturnValue(makeApp({ finances: makeFinances() }));
    render(<FinancesPage />);
    expect(screen.getByText('Aktueller Kassenstand')).toBeTruthy();
  });

  it('renders tab buttons', () => {
    mockUseApp.mockReturnValue(makeApp({ finances: makeFinances() }));
    render(<FinancesPage />);
    expect(screen.getByText('Umsätze')).toBeTruthy();
    expect(screen.getByText('Strafen')).toBeTruthy();
    expect(screen.getByText('Beiträge')).toBeTruthy();
  });

  it('renders Einnahmen and Ausgaben stats', () => {
    mockUseApp.mockReturnValue(makeApp({ finances: makeFinances() }));
    render(<FinancesPage />);
    expect(screen.getByText('Einnahmen')).toBeTruthy();
    expect(screen.getByText('Ausgaben')).toBeTruthy();
  });

  it('renders transactions tab by default', () => {
    mockUseApp.mockReturnValue(makeApp({ finances: makeFinances() }));
    render(<FinancesPage />);
    // Transaction tab is active (umsaetze)
    expect(screen.getByText('Umsätze')).toBeTruthy();
  });

  it('renders Strafen tab when selected', () => {
    mockUseApp.mockReturnValue(makeApp({ finances: makeFinances(), finTab: 'strafen' }));
    render(<FinancesPage />);
    expect(screen.getByText('Strafen')).toBeTruthy();
  });

  it('renders Beiträge tab when selected', () => {
    mockUseApp.mockReturnValue(makeApp({ finances: makeFinances(), finTab: 'beitraege' }));
    render(<FinancesPage />);
    expect(screen.getByText('Beiträge')).toBeTruthy();
  });
});
