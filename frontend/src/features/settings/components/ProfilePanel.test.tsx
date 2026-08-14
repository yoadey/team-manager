import { describe, it, expect, vi } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';

vi.mock('@/context/AppContext', () => {
  const useApp = vi.fn();
  return {
    useApp,
    // PrimaryButton (rendered by ProfilePanel's save action) calls useApp()
    // internally for its theme -- same pattern as MemberSheets.test.tsx.
    useAppActions: vi.fn(() => useApp()),
    useAppSelector: (sel: (s: { form: Record<string, unknown> }) => unknown) => sel(useApp().state),
  };
});

import { useApp } from '@/context/AppContext';
import { ProfilePanel } from './ProfilePanel';
import { createQueryWrapper, createTestQueryClient } from '@/test/queryTestUtils';
import type { AppContextValue } from '@/context/AppContext';
import type { Member } from '@/features/members';

const mockUseApp = vi.mocked(useApp);

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

function makeMember(overrides: Partial<Member> = {}): Member {
  return {
    membershipId: 'ms1',
    userId: 'u1',
    name: 'Max Mustermann',
    email: 'max@example.com',
    phone: '',
    birthday: '',
    address: '',
    avatarColor: '#1565C0',
    photo: null,
    group: '',
    title: '',
    roles: [],
    joinedAt: '2024-01-01',
    primaryRole: null,
    perms: {} as never,
    excludeFromStats: false,
    ...overrides,
  };
}

function makeApp(members: Member[] = [makeMember()]): AppContextValue {
  const app = {
    state: { primaryColor: '#1565C0', user: MOCK_USER, activeTeamId: 't1' },
    onFile: vi.fn(),
    uploadMyPhoto: vi.fn(),
    setMyTitle: vi.fn().mockResolvedValue(undefined),
    api: { members: { list: vi.fn().mockResolvedValue(members) } },
  } as unknown as AppContextValue;
  mockUseApp.mockReturnValue(app);
  return app;
}

function renderPanel(app: AppContextValue) {
  return render(<ProfilePanel app={app} />, { wrapper: createQueryWrapper(createTestQueryClient()) });
}

describe('ProfilePanel', () => {
  it('renders the user name', () => {
    renderPanel(makeApp());
    expect(screen.getByText('Max Mustermann')).toBeTruthy();
  });

  it('renders the user email', () => {
    renderPanel(makeApp());
    expect(screen.getByText('max@example.com')).toBeTruthy();
  });

  it('prefills the title field with the caller own member title once loaded', async () => {
    const app = makeApp([makeMember({ title: 'Witzbeauftragter' })]);
    renderPanel(app);
    await waitFor(() => expect(screen.getByLabelText(/^Titel/)).toHaveValue('Witzbeauftragter'));
  });

  it('saves an edited title via the self-service action', async () => {
    const user = userEvent.setup();
    const app = makeApp([makeMember({ title: '' })]);
    renderPanel(app);

    const input = await screen.findByLabelText(/^Titel/);
    await user.type(input, 'Orgaente');
    await user.click(screen.getByText('Speichern'));

    await waitFor(() => expect(app.setMyTitle).toHaveBeenCalledWith('ms1', 'Orgaente'));
  });

  it('disables saving until the title is actually changed', async () => {
    const app = makeApp([makeMember({ title: 'Existing' })]);
    renderPanel(app);

    await waitFor(() => expect(screen.getByLabelText(/^Titel/)).toHaveValue('Existing'));
    expect(screen.getByText('Speichern').closest('button')).toBeDisabled();
  });
});
