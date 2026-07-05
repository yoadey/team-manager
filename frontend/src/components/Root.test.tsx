import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen } from '@testing-library/react';
import { Root } from './Root';

vi.mock('@/context/AppContext', () => ({
  useApp: vi.fn(),
}));

vi.mock('@/features/auth', () => ({
  Login: () => <div data-testid="login">Login</div>,
}));

vi.mock('@/layouts/AppShell', () => ({
  Shell: () => <div data-testid="shell">Shell</div>,
  useCompact: vi.fn().mockReturnValue(false),
  COMPACT_BP: 760,
  shortName: (n: string) => n,
}));

vi.mock('./SheetHost', () => ({
  SheetHost: () => null,
}));

vi.mock('./Toast', () => ({
  Toast: () => <div data-testid="toast">Toast</div>,
}));

import { useApp } from '@/context/AppContext';
const mockUseApp = useApp as ReturnType<typeof vi.fn>;

describe('Root', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('renders loading spinner when phase is loading', () => {
    mockUseApp.mockReturnValue({ state: { phase: 'loading', primaryColor: '#4285F4' } });
    render(<Root />);
    expect(screen.getByRole('status')).toBeTruthy();
  });

  it('mounts Toast even while phase is loading, so a toast set during the ' +
    'cookie-restore establishSession call (e.g. invite-link redemption) is not silently swallowed', () => {
    mockUseApp.mockReturnValue({ state: { phase: 'loading', primaryColor: '#4285F4' } });
    render(<Root />);
    expect(screen.getByTestId('toast')).toBeTruthy();
  });

  it('renders Login component when phase is login', () => {
    mockUseApp.mockReturnValue({ state: { phase: 'login', primaryColor: '#4285F4' } });
    render(<Root />);
    expect(screen.getByTestId('login')).toBeTruthy();
  });

  it('renders Shell when phase is app', () => {
    mockUseApp.mockReturnValue({ state: { phase: 'app', primaryColor: '#4285F4' } });
    render(<Root />);
    expect(screen.getByTestId('shell')).toBeTruthy();
  });
});
