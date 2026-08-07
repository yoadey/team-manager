import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { ContribDetailSheet } from './ContribDetailSheet';

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

function makeContribution(overrides: Record<string, unknown> = {}) {
  return {
    id: 'c1',
    teamId: 't1',
    userId: 'u1',
    label: 'Mitgliedsbeitrag Januar',
    amount: 25,
    paidAmount: 10,
    status: 'partial',
    archived: false,
    name: 'Anna Müller',
    avatarColor: '#4285F4',
    photo: null,
    ...overrides,
  };
}

function makeApp(contribution: Record<string, unknown> | null, transactions: Record<string, unknown>[] = []) {
  const app = {
    api: {},
    state: { activeTeamId: 't1' },
    openTxFormForContribution: vi.fn(),
    openTxForm: vi.fn(),
  };
  mockUseApp.mockReturnValue(app as unknown as ReturnType<typeof useApp>);
  mockUseFinanceOverviewQuery.mockReturnValue({
    data: { transactions, contributions: contribution ? [contribution] : [] },
  });
  return { app, sheet: { formInitial: { id: 'c1' } } as never };
}

describe('ContribDetailSheet', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('shows the member name, paid amount, and required amount read-only', () => {
    const { app, sheet } = makeApp(makeContribution());
    render(<ContribDetailSheet app={app as never} sheet={sheet} />);
    expect(screen.getByText('Anna Müller')).toBeTruthy();
    expect(screen.queryByRole('textbox')).toBeNull();
    expect(screen.queryByRole('spinbutton')).toBeNull();
  });

  it('does not render any archive/unarchive/delete action', () => {
    const { app, sheet } = makeApp(makeContribution());
    render(<ContribDetailSheet app={app as never} sheet={sheet} />);
    expect(screen.queryByText('Löschen')).toBeNull();
    expect(screen.queryByText('Archivieren')).toBeNull();
    expect(screen.queryByText('Aus dem Archiv holen')).toBeNull();
  });

  it('lists linked transactions', () => {
    const contribution = makeContribution();
    const { app, sheet } = makeApp(contribution, [
      { id: 'tx1', title: 'Bar erhalten', date: '2026-01-05', amount: 10, contributionId: 'c1' },
    ]);
    render(<ContribDetailSheet app={app as never} sheet={sheet} />);
    expect(screen.getByText('Bar erhalten')).toBeTruthy();
  });

  it('clicking "Beitrag erfassen" opens the pre-linked transaction form', () => {
    const contribution = makeContribution();
    const { app, sheet } = makeApp(contribution);
    render(<ContribDetailSheet app={app as never} sheet={sheet} />);
    fireEvent.click(screen.getByText('Beitrag erfassen'));
    expect(app.openTxFormForContribution).toHaveBeenCalledWith(contribution);
  });

  it('shows the archived notice when the row is archived', () => {
    const { app, sheet } = makeApp(makeContribution({ archived: true }));
    render(<ContribDetailSheet app={app as never} sheet={sheet} />);
    expect(screen.getByText('Dieser Beitrag ist archiviert.')).toBeTruthy();
  });
});
