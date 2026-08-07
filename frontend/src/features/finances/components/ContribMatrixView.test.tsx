import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { ContribMatrixView } from './ContribMatrixView';
import type { Contribution } from '../types';

vi.mock('@/styles/tokens', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/styles/tokens')>();
  return {
    ...actual,
    fmtMoney: vi.fn().mockImplementation((n: number) => `${n} €`),
    fmtDate: vi.fn().mockImplementation((d: string) => d),
  };
});

vi.mock('@/i18n', () => ({
  t: vi.fn().mockImplementation((key: string) => key),
  getIntlLocale: vi.fn().mockReturnValue('de-DE'),
}));

function makeContrib(overrides: Partial<Contribution> = {}): Contribution {
  return {
    id: 'c1',
    teamId: 't1',
    userId: 'u1',
    label: 'Monatsbeitrag',
    amount: 20,
    paidAmount: 0,
    status: 'open',
    archived: false,
    name: 'Anna Müller',
    ...overrides,
  };
}

function makeApp() {
  return {
    openTxFormForContribution: vi.fn(),
    openContribForm: vi.fn(),
  };
}

describe('ContribMatrixView', () => {
  it('shows an empty state when there are no contributions', () => {
    render(<ContribMatrixView app={makeApp() as never} contributions={[]} />);
    expect(screen.getByText('finances.contribMatrixEmpty')).toBeTruthy();
  });

  it('renders one row per member and one column per fee group', () => {
    const contribs = [
      makeContrib({ id: 'c1', userId: 'u1', name: 'Anna' }),
      makeContrib({ id: 'c2', userId: 'u2', name: 'Bob', label: 'Turniergebühr' }),
    ];
    render(<ContribMatrixView app={makeApp() as never} contributions={contribs} />);
    expect(screen.getByText('Anna')).toBeTruthy();
    expect(screen.getByText('Bob')).toBeTruthy();
    expect(screen.getByText('Monatsbeitrag')).toBeTruthy();
    expect(screen.getByText('Turniergebühr')).toBeTruthy();
  });

  it('shows a dash icon for a member with no contribution in a given fee group', () => {
    const contribs = [
      makeContrib({ id: 'c1', userId: 'u1', name: 'Anna', label: 'Monatsbeitrag' }),
      makeContrib({ id: 'c2', userId: 'u2', name: 'Bob', label: 'Turniergebühr' }),
    ];
    render(<ContribMatrixView app={makeApp() as never} contributions={contribs} />);
    // Anna has no row for "Turniergebühr" -- rendered as a dash icon,
    // distinct from any real paid/open/partial/overpaid status.
    const cells = screen.getAllByRole('img');
    expect(cells.some((c) => c.getAttribute('data-testid') === 'RemoveOutlinedIcon')).toBe(true);
  });

  it('distinguishes an overpaid cell (savings icon) from an exactly paid one (check icon)', () => {
    const overpaid = [makeContrib({ id: 'c1', userId: 'u1', name: 'Anna', amount: 20, paidAmount: 25 })];
    const { unmount } = render(<ContribMatrixView app={makeApp() as never} contributions={overpaid} />);
    expect(screen.getByRole('img').getAttribute('data-testid')).toBe('SavingsOutlinedIcon');
    unmount();

    const exact = [makeContrib({ id: 'c2', userId: 'u1', name: 'Anna', amount: 20, paidAmount: 20 })];
    render(<ContribMatrixView app={makeApp() as never} contributions={exact} />);
    expect(screen.getByRole('img').getAttribute('data-testid')).toBe('CheckOutlinedIcon');
  });

  it('clicking a cell with no payment yet opens the pre-linked transaction form', () => {
    const app = makeApp();
    const c = makeContrib({ paidAmount: 0 });
    render(<ContribMatrixView app={app as never} contributions={[c]} />);
    fireEvent.click(screen.getByRole('button'));
    expect(app.openTxFormForContribution).toHaveBeenCalledWith(c);
    expect(app.openContribForm).not.toHaveBeenCalled();
  });

  it('clicking a cell with at least one payment opens the contribution detail sheet', () => {
    const app = makeApp();
    const c = makeContrib({ paidAmount: 10 });
    render(<ContribMatrixView app={app as never} contributions={[c]} />);
    fireEvent.click(screen.getByRole('button'));
    expect(app.openContribForm).toHaveBeenCalledWith(c);
    expect(app.openTxFormForContribution).not.toHaveBeenCalled();
  });

  it('an empty cell (no contribution row) is not clickable', () => {
    const app = makeApp();
    const contribs = [
      makeContrib({ id: 'c1', userId: 'u1', name: 'Anna', label: 'Monatsbeitrag' }),
      makeContrib({ id: 'c2', userId: 'u2', name: 'Bob', label: 'Turniergebühr' }),
    ];
    render(<ContribMatrixView app={app as never} contributions={contribs} />);
    // 2 members x 2 groups = 4 cells, only 2 are populated -> only 2 buttons.
    expect(screen.getAllByRole('button')).toHaveLength(2);
  });
});
