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
});
