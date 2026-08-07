import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { FinancesContributions } from './FinancesContributions';

vi.mock('@/context/AppContext', () => ({
  useApp: vi.fn(),
  useAppActions: vi.fn().mockReturnValue({}),
}));

vi.mock('@/styles/tokens', () => ({
  buildTokens: vi
    .fn()
    .mockReturnValue({
      primary: '#1565C0',
      onPrimary: '#fff',
      primaryContainer: '#E3F2FD',
      onPrimaryContainer: '#0D47A1',
    }),
  fmtMoney: vi.fn().mockImplementation((n) => `${n} €`),
  fmtDate: vi.fn().mockImplementation((d) => d),
  monthName: vi.fn().mockImplementation((m) => m),
  initials: vi.fn().mockImplementation((n: string) => n.slice(0, 2).toUpperCase()),
  NEUTRAL: {
    surface: '#fff',
    card: '#fff',
    line: '#e0e0e0',
    secondary: '#757575',
    error: '#B00020',
    success: '#2E7D32',
    successBg: '#E8F5E9',
    faint: '#999',
    onSurfaceVariant: '#666',
    warn: '#B26A00',
  },
}));

vi.mock('@/i18n', () => ({
  t: vi.fn().mockImplementation((key) => key),
  getIntlLocale: vi.fn().mockReturnValue('de-DE'),
}));

const tk = {
  primary: '#1565C0',
  onPrimary: '#fff',
  primaryContainer: '#E3F2FD',
  onPrimaryContainer: '#0D47A1',
} as never;

function makeApp(overrides = {}) {
  return {
    setState: vi.fn(),
    state: { contribGroup: null },
    openContribForm: vi.fn(),
    openContribCreate: vi.fn(),
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
    openPenaltySum: 0,
    contributions: [],
    contribOpen: 0,
    ...overrides,
  };
}

function makeContrib(overrides = {}) {
  return {
    id: 'c1',
    teamId: 't1',
    userId: 'u1',
    label: 'Monatsbeitrag',
    dueDate: null,
    amount: 20,
    paidAmount: 0,
    status: 'open' as const,
    name: 'Anna Müller',
    avatarColor: '#4285F4',
    photo: null,
    ...overrides,
  };
}

describe('FinancesContributions', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('renders empty state when no contributions', () => {
    const app = makeApp();
    render(<FinancesContributions app={app as never} t={tk} f={makeFinances()} canFin={false} />);
    expect(screen.getByText('finances.contribEmpty')).toBeTruthy();
  });

  it('renders a fee-name chip for contributions', () => {
    const app = makeApp();
    render(
      <FinancesContributions
        app={app as never}
        t={tk}
        f={makeFinances({ contributions: [makeContrib()] })}
        canFin={false}
      />,
    );
    expect(screen.getAllByText('Monatsbeitrag').length).toBeGreaterThan(0);
  });

  it('renders contribution member name', () => {
    const app = makeApp();
    render(
      <FinancesContributions
        app={app as never}
        t={tk}
        f={makeFinances({ contributions: [makeContrib()] })}
        canFin={false}
      />,
    );
    expect(screen.getByText('Anna Müller')).toBeTruthy();
  });

  it('renders paid contribution row', () => {
    const app = makeApp();
    render(
      <FinancesContributions
        app={app as never}
        t={tk}
        f={makeFinances({ contributions: [makeContrib({ status: 'paid', paidAmount: 20 })] })}
        canFin={false}
      />,
    );
    expect(screen.getByText('Anna Müller')).toBeTruthy();
  });

  it('renders a partially paid contribution row with a paid/total amount', () => {
    const app = makeApp();
    render(
      <FinancesContributions
        app={app as never}
        t={tk}
        f={makeFinances({ contributions: [makeContrib({ status: 'partial', paidAmount: 10 })] })}
        canFin={false}
      />,
    );
    expect(screen.getByText('10 € / 20 €')).toBeTruthy();
    expect(screen.getAllByText('finances.contribPartial').length).toBeGreaterThan(0);
  });

  it('clicking a fee-name chip calls setState with the group key', () => {
    const app = makeApp();
    render(
      <FinancesContributions
        app={app as never}
        t={tk}
        f={makeFinances({ contributions: [makeContrib()] })}
        canFin={false}
      />,
    );
    fireEvent.click(screen.getAllByText('Monatsbeitrag')[0]!.closest('button')!);
    expect(app.setState).toHaveBeenCalledWith({ contribGroup: 'Monatsbeitrag' });
  });

  it('renders multiple fee groups', () => {
    const app = makeApp();
    const contribs = [
      makeContrib({ id: 'c1', label: 'Mitgliedsbeitrag Juni', name: 'Anna' }),
      makeContrib({ id: 'c2', label: 'Mitgliedsbeitrag Mai', name: 'Bob', userId: 'u2' }),
    ];
    render(
      <FinancesContributions app={app as never} t={tk} f={makeFinances({ contributions: contribs })} canFin={false} />,
    );
    expect(screen.getAllByText('Mitgliedsbeitrag Juni').length).toBeGreaterThan(0);
    expect(screen.getAllByText('Mitgliedsbeitrag Mai').length).toBeGreaterThan(0);
  });

  it('sorts groups by soonest due date first', () => {
    const app = makeApp();
    const contribs = [
      makeContrib({ id: 'c1', label: 'Später fällig', name: 'Anna', dueDate: '2026-08-31' }),
      makeContrib({ id: 'c2', label: 'Bald fällig', name: 'Bob', userId: 'u2', dueDate: '2026-01-31' }),
    ];
    render(
      <FinancesContributions app={app as never} t={tk} f={makeFinances({ contributions: contribs })} canFin={false} />,
    );
    // The soonest-due group is selected by default, so its member renders.
    expect(screen.getByText('Bob')).toBeTruthy();
  });

  // Regression test: a recurring fee re-created period after period (no
  // catalog forces period-differentiated names, per design.md) used to
  // group solely by name, merging unrelated batches -- two same-named
  // batches with different due dates must now stay in separate groups.
  it('keeps two same-named batches with different due dates in separate groups', () => {
    const app = makeApp();
    const contribs = [
      makeContrib({ id: 'c1', label: 'Mitgliedsbeitrag', name: 'Anna', dueDate: '2026-01-31', amount: 25 }),
      makeContrib({ id: 'c2', label: 'Mitgliedsbeitrag', name: 'Bob', userId: 'u2', dueDate: '2026-02-28', amount: 25 }),
    ];
    render(
      <FinancesContributions app={app as never} t={tk} f={makeFinances({ contributions: contribs })} canFin={false} />,
    );
    // The soonest-due group (January) is selected by default -- only Anna's
    // row shows, Bob's February batch must not be blended in.
    expect(screen.getByText('Anna')).toBeTruthy();
    expect(screen.queryByText('Bob')).toBeNull();

    // Two distinct chips exist (disambiguated by due date in the chip's
    // secondary line), not one merged group.
    const febChip = screen.getAllByRole('button').find((btn) => btn.textContent?.includes('2026-02-28'));
    expect(febChip).toBeTruthy();
    fireEvent.click(febChip!);
    expect(app.setState).toHaveBeenCalledWith({ contribGroup: 'Mitgliedsbeitrag 2026-02-28' });
  });

  // Two same-named batches sharing the exact same due date ARE the same
  // fee re-touched (e.g. edited after creation) and must still merge into
  // one group, same as before this change.
  it('still merges same-named batches that share the same due date', () => {
    const app = makeApp();
    const contribs = [
      makeContrib({ id: 'c1', label: 'Mitgliedsbeitrag', name: 'Anna', dueDate: '2026-01-31' }),
      makeContrib({ id: 'c2', label: 'Mitgliedsbeitrag', name: 'Bob', userId: 'u2', dueDate: '2026-01-31' }),
    ];
    render(
      <FinancesContributions app={app as never} t={tk} f={makeFinances({ contributions: contribs })} canFin={false} />,
    );
    expect(screen.getByText('Anna')).toBeTruthy();
    expect(screen.getByText('Bob')).toBeTruthy();
  });

  it('uses contribGroup from state when it matches an existing group', () => {
    const app = makeApp({ state: { contribGroup: 'Mitgliedsbeitrag Mai' } });
    const contribs = [
      makeContrib({ id: 'c1', label: 'Mitgliedsbeitrag Juni', name: 'Anna' }),
      makeContrib({ id: 'c2', label: 'Mitgliedsbeitrag Mai', name: 'Bob', userId: 'u2' }),
    ];
    render(
      <FinancesContributions app={app as never} t={tk} f={makeFinances({ contributions: contribs })} canFin={false} />,
    );
    expect(screen.getByText('Bob')).toBeTruthy();
  });

  it('shows open count in the fee-name chip when there are open contribs', () => {
    const app = makeApp();
    render(
      <FinancesContributions
        app={app as never}
        t={tk}
        f={makeFinances({ contributions: [makeContrib({ status: 'open' })] })}
        canFin={false}
      />,
    );
    expect(screen.getAllByText('finances.contribOpen').length).toBeGreaterThan(0);
  });

  it('shows the create-fee button when canFin is true', () => {
    const app = makeApp();
    render(<FinancesContributions app={app as never} t={tk} f={makeFinances()} canFin={true} />);
    expect(screen.getByText('finances.contribCreateBtn')).toBeTruthy();
  });

  it('hides the create-fee button when canFin is false', () => {
    const app = makeApp();
    render(<FinancesContributions app={app as never} t={tk} f={makeFinances()} canFin={false} />);
    expect(screen.queryByText('finances.contribCreateBtn')).toBeNull();
  });

  it('clicking the create-fee button calls openContribCreate', () => {
    const app = makeApp();
    render(<FinancesContributions app={app as never} t={tk} f={makeFinances()} canFin={true} />);
    fireEvent.click(screen.getByText('finances.contribCreateBtn'));
    expect(app.openContribCreate).toHaveBeenCalled();
  });

  // Regression: openContribForm/api.finances.updateContribution were fully
  // implemented (hook, sheet, service layer) but no rendered component ever
  // called openContribForm -- a contribution's label/amount could never be
  // corrected through the UI. Only visible with canFin=true.
  it('shows an edit action for each contribution when canFin is true', () => {
    const app = makeApp();
    render(
      <FinancesContributions
        app={app as never}
        t={tk}
        f={makeFinances({ contributions: [makeContrib()] })}
        canFin={true}
      />,
    );
    expect(screen.getByLabelText('finances.editContribLabel')).toBeTruthy();
  });

  it('hides the edit action when canFin is false', () => {
    const app = makeApp();
    render(
      <FinancesContributions
        app={app as never}
        t={tk}
        f={makeFinances({ contributions: [makeContrib()] })}
        canFin={false}
      />,
    );
    expect(screen.queryByLabelText('finances.editContribLabel')).toBeNull();
  });

  // Regression test: memberName is optional per the OpenAPI Contribution
  // schema (not in `required`), and the row-sort comparator used to call
  // `.name!.localeCompare(...)` unguarded -- a contribution with no name
  // (e.g. a left-outer-join for a member who left the team) would throw
  // "Cannot read properties of undefined" and crash the whole page.
  it('does not throw when a contribution has no member name', () => {
    const app = makeApp();
    const contribs = [
      makeContrib({ id: 'c1', name: undefined }),
      makeContrib({ id: 'c2', name: 'Bob', userId: 'u2' }),
    ];
    expect(() =>
      render(
        <FinancesContributions
          app={app as never}
          t={tk}
          f={makeFinances({ contributions: contribs })}
          canFin={false}
        />,
      ),
    ).not.toThrow();
    expect(screen.getByText('Bob')).toBeTruthy();
  });

  // Regression test: the contributor-row sort used to hardcode
  // localeCompare's locale argument to 'de' regardless of the active UI
  // locale, unlike every other locale-aware sort/format helper in the app.
  it('sorts contributor rows using the current locale rather than a hardcoded one', async () => {
    const i18n = await import('@/i18n');
    vi.mocked(i18n.getIntlLocale).mockReturnValue('en-US');
    const localeCompareSpy = vi.spyOn(String.prototype, 'localeCompare');
    const app = makeApp();
    const contribs = [makeContrib({ id: 'c1', name: 'Alice' }), makeContrib({ id: 'c2', name: 'Bob', userId: 'u2' })];
    render(
      <FinancesContributions app={app as never} t={tk} f={makeFinances({ contributions: contribs })} canFin={false} />,
    );

    const usedLocaleArgs = localeCompareSpy.mock.calls.map((c) => c[1]);
    expect(usedLocaleArgs).toContain('en-US');
    expect(usedLocaleArgs).not.toContain('de');

    localeCompareSpy.mockRestore();
    vi.mocked(i18n.getIntlLocale).mockReturnValue('de-DE');
  });

  it('clicking the edit action calls openContribForm with the contribution', () => {
    const app = makeApp();
    const contrib = makeContrib();
    render(
      <FinancesContributions
        app={app as never}
        t={tk}
        f={makeFinances({ contributions: [contrib] })}
        canFin={true}
      />,
    );
    fireEvent.click(screen.getByLabelText('finances.editContribLabel'));
    expect(app.openContribForm).toHaveBeenCalledWith(contrib);
  });
});
