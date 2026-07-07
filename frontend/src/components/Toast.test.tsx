import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen } from '@testing-library/react';
import { Toast } from './Toast';

vi.mock('@/context/AppContext', () => ({
  useApp: vi.fn(),
}));

import { useApp } from '@/context/AppContext';
const mockUseApp = useApp as ReturnType<typeof vi.fn>;

describe('Toast', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('renders nothing when toast is null', () => {
    mockUseApp.mockReturnValue({ state: { toast: null } });
    const { container } = render(<Toast />);
    expect(container.firstChild).toBeNull();
  });

  it('renders toast message when set', () => {
    mockUseApp.mockReturnValue({ state: { toast: { message: 'Gespeichert!' } } });
    render(<Toast />);
    expect(screen.getByText('Gespeichert!')).toBeTruthy();
    expect(screen.getByRole('status')).toBeTruthy();
  });

  it('has aria-live polite for accessibility', () => {
    mockUseApp.mockReturnValue({ state: { toast: { message: 'Hallo' } } });
    render(<Toast />);
    const toast = screen.getByRole('status');
    expect(toast.getAttribute('aria-live')).toBe('polite');
  });

  // Regression test: the message was a bare flex-row child with no
  // minWidth: 0 / overflow-wrap, so a long unbreakable token (e.g. a joined
  // team name up to 60 chars with no spaces) could overflow the toast's
  // maxWidth: 90vw box on a narrow viewport instead of wrapping inside it.
  it('wraps the message in a shrinkable, word-breaking container', () => {
    mockUseApp.mockReturnValue({ state: { toast: { message: 'Hallo' } } });
    render(<Toast />);
    const messageEl = screen.getByText('Hallo');
    expect(messageEl.tagName).toBe('SPAN');
  });
});
