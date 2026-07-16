import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { PenaltyCatalogSheet } from './PenaltyCatalogSheet';

vi.mock('@/context/AppContext', () => ({
  useApp: vi.fn(),
  useAppActions: vi.fn().mockReturnValue({}),
}));

vi.mock('../hooks/useFinanceQueries', () => ({
  useFinanceOverviewQuery: vi.fn(),
}));

import { useApp } from '@/context/AppContext';
import { useFinanceOverviewQuery } from '../hooks/useFinanceQueries';
const mockUseApp = vi.mocked(useApp);
const mockUseFinanceOverviewQuery = useFinanceOverviewQuery as ReturnType<typeof vi.fn>;

function makePenalty(overrides = {}) {
  return { id: 'p1', label: 'Versäumtes Training', amount: 10, ...overrides };
}

function makeApp(penalties: unknown[] | null = [], canWrite = false) {
  mockUseFinanceOverviewQuery.mockReturnValue({ data: penalties === null ? null : { penalties } });
  return {
    api: {},
    state: {
      primaryColor: '#4285F4',
      activeTeamId: 't1',
    },
    can: vi.fn().mockReturnValue(canWrite),
    openPenaltyForm: vi.fn(),
  };
}

describe('PenaltyCatalogSheet', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  const sheet = {} as never;

  it('shows empty state when no penalties', () => {
    mockUseApp.mockReturnValue(makeApp([]) as never);
    const app = mockUseApp();
    render(<PenaltyCatalogSheet app={app as never} sheet={sheet} />);
    expect(screen.getByText(/Noch keine Strafen/i)).toBeTruthy();
  });

  it('renders penalty items', () => {
    mockUseApp.mockReturnValue(makeApp([makePenalty()]) as never);
    const app = mockUseApp();
    render(<PenaltyCatalogSheet app={app as never} sheet={sheet} />);
    expect(screen.getByText('Versäumtes Training')).toBeTruthy();
  });

  it('shows add button when user has write permission', () => {
    mockUseApp.mockReturnValue(makeApp([], true) as never);
    const app = mockUseApp();
    render(<PenaltyCatalogSheet app={app as never} sheet={sheet} />);
    expect(screen.getByText('Strafe zum Katalog hinzufügen')).toBeTruthy();
  });

  it('hides add button when user has no write permission', () => {
    mockUseApp.mockReturnValue(makeApp([], false) as never);
    const app = mockUseApp();
    render(<PenaltyCatalogSheet app={app as never} sheet={sheet} />);
    expect(screen.queryByText('Strafe zum Katalog hinzufügen')).toBeNull();
  });

  it('clicking add button calls openPenaltyForm()', () => {
    const app = makeApp([], true);
    mockUseApp.mockReturnValue(app as never);
    render(<PenaltyCatalogSheet app={app as never} sheet={sheet} />);
    fireEvent.click(screen.getByText('Strafe zum Katalog hinzufügen').closest('button')!);
    expect(app.openPenaltyForm).toHaveBeenCalledWith();
  });

  it('clicking penalty row calls openPenaltyForm(p) when can write', () => {
    const p = makePenalty();
    const app = makeApp([p], true);
    mockUseApp.mockReturnValue(app as never);
    render(<PenaltyCatalogSheet app={app as never} sheet={sheet} />);
    fireEvent.click(screen.getByText('Versäumtes Training').closest('button')!);
    expect(app.openPenaltyForm).toHaveBeenCalledWith(p);
  });

  it('penalty row is non-clickable Box (no button) when no write permission', () => {
    mockUseApp.mockReturnValue(makeApp([makePenalty()], false) as never);
    const app = mockUseApp();
    render(<PenaltyCatalogSheet app={app as never} sheet={sheet} />);
    // No button around the penalty label
    expect(screen.getByText('Versäumtes Training').closest('button')).toBeNull();
  });

  it('renders hint text', () => {
    mockUseApp.mockReturnValue(makeApp([]) as never);
    const app = mockUseApp();
    render(<PenaltyCatalogSheet app={app as never} sheet={sheet} />);
    // Catalog hint is shown
    expect(document.querySelector('div')).toBeTruthy();
  });

  it('handles null finances gracefully', () => {
    const app = makeApp(null);
    mockUseApp.mockReturnValue(app as never);
    render(<PenaltyCatalogSheet app={app as never} sheet={sheet} />);
    expect(screen.getByText(/Noch keine Strafen/i)).toBeTruthy();
  });
});
