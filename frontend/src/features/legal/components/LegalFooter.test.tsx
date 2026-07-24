import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { LegalFooter } from './LegalFooter';

vi.mock('@/context/AppContext', () => ({
  useApp: vi.fn(),
  useAppActions: vi.fn().mockReturnValue({}),
}));

import { useApp } from '@/context/AppContext';
const mockUseApp = vi.mocked(useApp);

describe('LegalFooter', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('opens the legal notice when clicked', () => {
    const openLegal = vi.fn();
    mockUseApp.mockReturnValue({ openLegal } as unknown as ReturnType<typeof useApp>);
    render(<LegalFooter />);

    fireEvent.click(screen.getByText('Impressum'));
    expect(openLegal).toHaveBeenCalledWith('impressum');
  });

  it('opens the privacy policy when clicked', () => {
    const openLegal = vi.fn();
    mockUseApp.mockReturnValue({ openLegal } as unknown as ReturnType<typeof useApp>);
    render(<LegalFooter />);

    fireEvent.click(screen.getByText('Datenschutzerklärung'));
    expect(openLegal).toHaveBeenCalledWith('datenschutz');
  });
});
