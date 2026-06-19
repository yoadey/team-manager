import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { NewsPage } from './NewsPage';

vi.mock('@/context/AppContext', () => ({
  useApp: vi.fn(),
  useAppActions: vi.fn().mockReturnValue({}),
}));

import { useApp } from '@/context/AppContext';
const mockUseApp = useApp as ReturnType<typeof vi.fn>;

function makeApp(overrides: Record<string, unknown> = {}) {
  return {
    state: {
      primaryColor: '#4285F4',
      news: [],
      user: { id: 'u1' },
      ...overrides,
    },
    can: vi.fn().mockReturnValue(false),
    openNewsForm: vi.fn(),
    removeNews: vi.fn(),
  };
}

describe('NewsPage', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('shows spinner when news is null', () => {
    mockUseApp.mockReturnValue(makeApp({ news: null }));
    render(<NewsPage />);
    expect(screen.getByRole('status')).toBeTruthy();
  });

  it('shows empty state when no news', () => {
    mockUseApp.mockReturnValue(makeApp({ news: [] }));
    render(<NewsPage />);
    expect(screen.getByText('Noch keine Neuigkeiten')).toBeTruthy();
  });

  it('renders news items when they exist', () => {
    mockUseApp.mockReturnValue(
      makeApp({
        news: [
          {
            id: 'n1',
            title: 'Wichtige Mitteilung',
            body: 'Details hier',
            authorName: 'Admin',
            authorPhoto: null,
            authorColor: '#000',
            createdAt: '2026-01-15T10:00:00Z',
            pinned: false,
          },
        ],
      }),
    );
    render(<NewsPage />);
    expect(screen.getByText('Wichtige Mitteilung')).toBeTruthy();
  });

  it('calls openNewsForm when clicking edit button (admin)', async () => {
    const newsItem = {
      id: 'n1',
      title: 'Neuigkeit',
      body: 'Body',
      authorName: 'Coach',
      authorPhoto: null,
      authorColor: '#000',
      createdAt: '2026-01-15T10:00:00Z',
      pinned: false,
    };
    const app = makeApp({ news: [newsItem] });
    app.can = vi.fn().mockReturnValue(true);
    mockUseApp.mockReturnValue(app);
    render(<NewsPage />);
    await userEvent.click(screen.getByLabelText('News bearbeiten'));
    expect(app.openNewsForm).toHaveBeenCalledWith(newsItem);
  });

  it('calls removeNews when clicking delete button (admin)', async () => {
    const newsItem = {
      id: 'n1',
      title: 'Neuigkeit',
      body: 'Body',
      authorName: 'Coach',
      authorPhoto: null,
      authorColor: '#000',
      createdAt: '2026-01-15T10:00:00Z',
      pinned: false,
    };
    const app = makeApp({ news: [newsItem] });
    app.can = vi.fn().mockReturnValue(true);
    mockUseApp.mockReturnValue(app);
    render(<NewsPage />);
    await userEvent.click(screen.getByLabelText('News löschen'));
    expect(app.removeNews).toHaveBeenCalledWith('n1');
  });
});
