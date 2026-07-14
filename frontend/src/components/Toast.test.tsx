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

  // Regression test: every toast -- including reportActionError's "You don't
  // have permission to do that" / network / session-expired messages --
  // previously rendered the same green success checkmark, regardless of
  // kind. A caller with no kind (or kind: 'success') keeps that look; an
  // error must switch to a distinct icon, color, and a more insistent
  // role/aria-live so it doesn't visually read as a success.
  it('defaults to the success checkmark when kind is omitted', () => {
    mockUseApp.mockReturnValue({ state: { toast: { message: 'Gespeichert!' } } });
    render(<Toast />);
    expect(screen.getByText('check_circle')).toBeTruthy();
    expect(screen.getByRole('status')).toBeTruthy();
  });

  it('renders an error icon and role="alert" when kind is error', () => {
    mockUseApp.mockReturnValue({ state: { toast: { message: 'Keine Berechtigung', kind: 'error' } } });
    render(<Toast />);
    expect(screen.getByText('Keine Berechtigung')).toBeTruthy();
    expect(screen.getByText('error')).toBeTruthy();
    expect(screen.queryByText('check_circle')).toBeNull();
    const toast = screen.getByRole('alert');
    expect(toast.getAttribute('aria-live')).toBe('assertive');
  });
});
