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
    state: { contribMonth: null },
    openContribForm: vi.fn(),
    toggleContribution: vi.fn(),
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
    month: '2025-06',
    label: 'Monatsbeitrag',
    amount: 20,
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
    expect(document.querySelector('div')).toBeTruthy();
  });

  it('renders month chip for contributions', () => {
    const app = makeApp();
    render(
      <FinancesContributions
        app={app as never}
        t={tk}
        f={makeFinances({ contributions: [makeContrib()] })}
        canFin={false}
      />,
    );
    expect(screen.getAllByText('2025-06').length).toBeGreaterThan(0);
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
        f={makeFinances({ contributions: [makeContrib({ status: 'paid' })] })}
        canFin={false}
      />,
    );
    expect(screen.getByText('Anna Müller')).toBeTruthy();
  });

  it('clicking month chip calls setState', () => {
    const app = makeApp();
    render(
      <FinancesContributions
        app={app as never}
        t={tk}
        f={makeFinances({ contributions: [makeContrib()] })}
        canFin={false}
      />,
    );
    fireEvent.click(screen.getAllByText('2025-06')[0]!.closest('button')!);
    expect(app.setState).toHaveBeenCalledWith({ contribMonth: '2025-06' });
  });

  it('renders multiple months and selects first by default', () => {
    const app = makeApp();
    const contribs = [
      makeContrib({ id: 'c1', month: '2025-06', name: 'Anna' }),
      makeContrib({ id: 'c2', month: '2025-05', name: 'Bob', userId: 'u2' }),
    ];
    render(
      <FinancesContributions app={app as never} t={tk} f={makeFinances({ contributions: contribs })} canFin={false} />,
    );
    expect(screen.getAllByText('2025-06').length).toBeGreaterThan(0);
    expect(screen.getAllByText('2025-05').length).toBeGreaterThan(0);
  });

  it('uses contribMonth from state when matching', () => {
    const app = makeApp({ state: { contribMonth: '2025-05' } });
    const contribs = [
      makeContrib({ id: 'c1', month: '2025-06', name: 'Anna' }),
      makeContrib({ id: 'c2', month: '2025-05', name: 'Bob', userId: 'u2' }),
    ];
    render(
      <FinancesContributions app={app as never} t={tk} f={makeFinances({ contributions: contribs })} canFin={false} />,
    );
    expect(screen.getByText('Bob')).toBeTruthy();
  });

  it('shows open count in month chip when there are open contribs', () => {
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
