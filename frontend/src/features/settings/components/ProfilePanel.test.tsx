import { describe, it, expect, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import { ProfilePanel } from './ProfilePanel';
import type { AppContextValue } from '@/context/AppContext';

vi.mock('@/styles/tokens', async (importOriginal) => {
  const mod = await importOriginal<typeof import('@/styles/tokens')>();
  return {
    ...mod,
    buildTokens: vi.fn().mockReturnValue({
      primary: '#4285F4',
      primaryContainer: '#E8F0FE',
      onPrimaryContainer: '#001D35',
      onPrimary: '#ffffff',
    }),
  };
});

const MOCK_USER = {
  id: 'u1',
  name: 'Max Mustermann',
  email: 'max@example.com',
  photo: null,
  avatarColor: '#1565C0',
};

function makeApp(overrides: Record<string, unknown> = {}) {
  return {
    state: { primaryColor: '#1565C0', user: MOCK_USER, ...overrides },
    onFile: vi.fn(),
    uploadMyPhoto: vi.fn(),
  } as unknown as AppContextValue;
}

describe('ProfilePanel', () => {
  it('renders the user name', () => {
    render(<ProfilePanel app={makeApp()} />);
    expect(screen.getByText('Max Mustermann')).toBeTruthy();
  });

  it('renders the user email', () => {
    render(<ProfilePanel app={makeApp()} />);
    expect(screen.getByText('max@example.com')).toBeTruthy();
  });
});
