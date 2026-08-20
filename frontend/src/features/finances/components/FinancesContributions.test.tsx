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
    openContribDetail: vi.fn(),
    openContribGroupEdit: vi.fn(),
    openContribCreate: vi.fn(),
    openTxFormForContribution: vi.fn(),
    askConfirm: vi.fn((cfg: { onConfirm?: () => void }) => cfg.onConfirm?.()),
    archiveContribGroup: vi.fn(),
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

function makeContrib(overrides: Record<string, unknown> = {}) {
  // Cents fields default from the euro fields (possibly overridden below) so
  // callers that only pass `amount`/`paidAmount` still get consistent cents,
  // unless the test explicitly overrides `amountCents`/`paidAmountCents` too
  // (see the float-vs-cents summation regression test below).
  const amount = (overrides as { amount?: number }).amount ?? 20;
  const paidAmount = (overrides as { paidAmount?: number }).paidAmount ?? 0;
  return {
    id: 'c1',
    teamId: 't1',
    userId: 'u1',
    label: 'Monatsbeitrag',
    dueDate: null,
    amount,
    amountCents: Math.round(amount * 100),
    paidAmount,
    paidAmountCents: Math.round(paidAmount * 100),
    status: 'open' as const,
    archived: false,
    name: 'Anna Müller',
    avatarColor: '#4285F4',
    photo: null,
    ...overrides,
  };
}

/** Renders and switches to the list view -- the matrix view is the default
 * (see openspec/changes/finance-matrix-transactions), so list-specific
 * assertions need this instead of a bare render. */
function renderList(props: Parameters<typeof FinancesContributions>[0]) {
  const result = render(<FinancesContributions {...props} />);
  fireEvent.click(screen.getByText('finances.contribViewList'));
  return result;
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

  it('defaults to the matrix view', () => {
    const app = makeApp();
    render(
      <FinancesContributions
        app={app as never}
        t={tk}
        f={makeFinances({ contributions: [makeContrib()] })}
        canFin={false}
      />,
    );
    expect(screen.getByText('finances.contribMatrixMemberHeader')).toBeTruthy();
  });

  it('switching to the list view renders the list and hides the matrix', () => {
    const app = makeApp();
    renderList({
      app: app as never,
      t: tk,
      f: makeFinances({ contributions: [makeContrib()] }),
      canFin: false,
    });
    expect(screen.queryByText('finances.contribMatrixMemberHeader')).toBeNull();
    expect(screen.getAllByText('Monatsbeitrag').length).toBeGreaterThan(0);
  });

  it('renders a fee-name chip for contributions', () => {
    const app = makeApp();
    renderList({
      app: app as never,
      t: tk,
      f: makeFinances({ contributions: [makeContrib()] }),
      canFin: false,
    });
    expect(screen.getAllByText('Monatsbeitrag').length).toBeGreaterThan(0);
  });

  it('renders contribution member name', () => {
    const app = makeApp();
    renderList({
      app: app as never,
      t: tk,
      f: makeFinances({ contributions: [makeContrib()] }),
      canFin: false,
    });
    expect(screen.getByText('Anna Müller')).toBeTruthy();
  });

  it('renders paid contribution row', () => {
    const app = makeApp();
    renderList({
      app: app as never,
      t: tk,
      f: makeFinances({ contributions: [makeContrib({ status: 'paid', paidAmount: 20 })] }),
      canFin: false,
    });
    expect(screen.getByText('Anna Müller')).toBeTruthy();
  });

  it('renders a partially paid contribution row with a paid/total amount', () => {
    const app = makeApp();
    renderList({
      app: app as never,
      t: tk,
      f: makeFinances({ contributions: [makeContrib({ status: 'partial', paidAmount: 10 })] }),
      canFin: false,
    });
    expect(screen.getByText('10 € / 20 €')).toBeTruthy();
    expect(screen.getAllByText('finances.contribPartial').length).toBeGreaterThan(0);
  });

  it('clicking a fee-name chip calls setState with the group key', () => {
    const app = makeApp();
    renderList({
      app: app as never,
      t: tk,
      f: makeFinances({ contributions: [makeContrib()] }),
      canFin: false,
    });
    fireEvent.click(screen.getAllByText('Monatsbeitrag')[0]!.closest('button')!);
    expect(app.setState).toHaveBeenCalledWith({ contribGroup: 'Monatsbeitrag' });
  });

  it('renders multiple fee groups', () => {
    const app = makeApp();
    const contribs = [
      makeContrib({ id: 'c1', label: 'Mitgliedsbeitrag Juni', name: 'Anna' }),
      makeContrib({ id: 'c2', label: 'Mitgliedsbeitrag Mai', name: 'Bob', userId: 'u2' }),
    ];
    renderList({ app: app as never, t: tk, f: makeFinances({ contributions: contribs }), canFin: false });
    expect(screen.getAllByText('Mitgliedsbeitrag Juni').length).toBeGreaterThan(0);
    expect(screen.getAllByText('Mitgliedsbeitrag Mai').length).toBeGreaterThan(0);
  });

  it('sorts groups by soonest due date first', () => {
    const app = makeApp();
    const contribs = [
      makeContrib({ id: 'c1', label: 'Später fällig', name: 'Anna', dueDate: '2026-08-31' }),
      makeContrib({ id: 'c2', label: 'Bald fällig', name: 'Bob', userId: 'u2', dueDate: '2026-01-31' }),
    ];
    renderList({ app: app as never, t: tk, f: makeFinances({ contributions: contribs }), canFin: false });
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
    renderList({ app: app as never, t: tk, f: makeFinances({ contributions: contribs }), canFin: false });
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
    renderList({ app: app as never, t: tk, f: makeFinances({ contributions: contribs }), canFin: false });
    expect(screen.getByText('Anna')).toBeTruthy();
    expect(screen.getByText('Bob')).toBeTruthy();
  });

  it('uses contribGroup from state when it matches an existing group', () => {
    const app = makeApp({ state: { contribGroup: 'Mitgliedsbeitrag Mai' } });
    const contribs = [
      makeContrib({ id: 'c1', label: 'Mitgliedsbeitrag Juni', name: 'Anna' }),
      makeContrib({ id: 'c2', label: 'Mitgliedsbeitrag Mai', name: 'Bob', userId: 'u2' }),
    ];
    renderList({ app: app as never, t: tk, f: makeFinances({ contributions: contribs }), canFin: false });
    expect(screen.getByText('Bob')).toBeTruthy();
  });

  it('shows open count in the fee-name chip when there are open contribs', () => {
    const app = makeApp();
    renderList({
      app: app as never,
      t: tk,
      f: makeFinances({ contributions: [makeContrib({ status: 'open' })] }),
      canFin: false,
    });
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

  it('shows a view action for each contribution when canFin is true', () => {
    const app = makeApp();
    renderList({
      app: app as never,
      t: tk,
      f: makeFinances({ contributions: [makeContrib()] }),
      canFin: true,
    });
    expect(screen.getByLabelText('finances.viewContribLabel')).toBeTruthy();
  });

  it('hides the view action when canFin is false', () => {
    const app = makeApp();
    renderList({
      app: app as never,
      t: tk,
      f: makeFinances({ contributions: [makeContrib()] }),
      canFin: false,
    });
    expect(screen.queryByLabelText('finances.viewContribLabel')).toBeNull();
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
      renderList({ app: app as never, t: tk, f: makeFinances({ contributions: contribs }), canFin: false }),
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
    renderList({ app: app as never, t: tk, f: makeFinances({ contributions: contribs }), canFin: false });

    const usedLocaleArgs = localeCompareSpy.mock.calls.map((c) => c[1]);
    expect(usedLocaleArgs).toContain('en-US');
    expect(usedLocaleArgs).not.toContain('de');

    localeCompareSpy.mockRestore();
    vi.mocked(i18n.getIntlLocale).mockReturnValue('de-DE');
  });

  it('clicking the view action calls openContribDetail with the contribution', () => {
    const app = makeApp();
    const contrib = makeContrib();
    renderList({ app: app as never, t: tk, f: makeFinances({ contributions: [contrib] }), canFin: true });
    fireEvent.click(screen.getByLabelText('finances.viewContribLabel'));
    expect(app.openContribDetail).toHaveBeenCalledWith(contrib);
  });

  it('archived contributions are hidden by default', () => {
    const app = makeApp();
    renderList({
      app: app as never,
      t: tk,
      f: makeFinances({ contributions: [makeContrib({ archived: true })] }),
      canFin: false,
    });
    expect(screen.getByText('finances.contribEmpty')).toBeTruthy();
    expect(screen.queryByText('Anna Müller')).toBeNull();
  });

  it('the "show archived" toggle reveals archived contributions', () => {
    const app = makeApp();
    renderList({
      app: app as never,
      t: tk,
      f: makeFinances({ contributions: [makeContrib({ archived: true })] }),
      canFin: false,
    });
    fireEvent.click(screen.getByText('finances.contribShowArchived'));
    expect(screen.getByText('Anna Müller')).toBeTruthy();
  });

  it('shows an edit-group action for canFin users that opens the group edit sheet', () => {
    const app = makeApp();
    const contribs = [
      makeContrib({ id: 'c1', name: 'Anna', userId: 'u1' }),
      makeContrib({ id: 'c2', name: 'Bob', userId: 'u2' }),
    ];
    renderList({ app: app as never, t: tk, f: makeFinances({ contributions: contribs }), canFin: true });
    fireEvent.click(screen.getByText('finances.contribGroupEditBtn'));
    expect(app.openContribGroupEdit).toHaveBeenCalledWith(contribs);
  });

  it('hides the edit-group action when canFin is false', () => {
    const app = makeApp();
    renderList({
      app: app as never,
      t: tk,
      f: makeFinances({ contributions: [makeContrib()] }),
      canFin: false,
    });
    expect(screen.queryByText('finances.contribGroupEditBtn')).toBeNull();
  });

  it('shows an archive-group action for canFin users that archives every row in the group', () => {
    const app = makeApp();
    const contribs = [
      makeContrib({ id: 'c1', name: 'Anna', userId: 'u1' }),
      makeContrib({ id: 'c2', name: 'Bob', userId: 'u2' }),
    ];
    renderList({ app: app as never, t: tk, f: makeFinances({ contributions: contribs }), canFin: true });
    fireEvent.click(screen.getByText('finances.contribArchiveGroupBtn'));
    expect(app.archiveContribGroup).toHaveBeenCalledWith(contribs, true);
  });

  it('hides the archive-group action when canFin is false', () => {
    const app = makeApp();
    renderList({
      app: app as never,
      t: tk,
      f: makeFinances({ contributions: [makeContrib()] }),
      canFin: false,
    });
    expect(screen.queryByText('finances.contribArchiveGroupBtn')).toBeNull();
  });

  it('shows the paid-in-excess amount instead of capping display at the fee amount', () => {
    const app = makeApp();
    renderList({
      app: app as never,
      t: tk,
      f: makeFinances({ contributions: [makeContrib({ status: 'paid', paidAmount: 30, amount: 20 })] }),
      canFin: false,
    });
    expect(screen.getByText(/30 €/)).toBeTruthy();
    expect(screen.getAllByText('finances.contribOverpaid').length).toBeGreaterThan(0);
  });

  // Regression test: the group summary used to sum already-converted euro
  // floats (c.paidAmount/c.amount) directly, which accumulates visible
  // IEEE754 rounding error over many rows with amounts that don't divide
  // evenly (e.g. 0.29 € summed 100 times drifts to 28.99999999999994
  // instead of the exact 29). Summing the pre-conversion integer cents and
  // converting once must not exhibit that drift.
  it('sums group totals from integer cents, avoiding float drift across many rows', async () => {
    const i18n = await import('@/i18n');
    const app = makeApp();
    const rows = Array.from({ length: 100 }, (_, i) =>
      makeContrib({ id: `c${i}`, userId: `u${i}`, name: `M${i}`, amount: 0.29, paidAmount: 0.29, status: 'paid' }),
    );
    renderList({ app: app as never, t: tk, f: makeFinances({ contributions: rows }), canFin: false });

    const summaryCall = vi.mocked(i18n.t).mock.calls.find((call) => call[0] === 'finances.contribSummary');
    expect(summaryCall?.[1]).toMatchObject({ paidAmt: '29 €', totalAmt: '29 €' });
  });

  it('clicking a matrix cell opens the contribution detail sheet', () => {
    const app = makeApp();
    const contrib = makeContrib({ paidAmount: 0 });
    render(
      <FinancesContributions
        app={app as never}
        t={tk}
        f={makeFinances({ contributions: [contrib] })}
        canFin={false}
      />,
    );
    fireEvent.click(screen.getByRole('button', { name: /finances.contribMatrixCellAria/i }));
    expect(app.openContribDetail).toHaveBeenCalledWith(contrib);
  });
});
