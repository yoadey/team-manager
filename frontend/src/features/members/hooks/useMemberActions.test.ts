import { describe, it, expect, vi, beforeEach } from 'vitest';
import { renderHook, act } from '@testing-library/react';
import { useMemberActions } from './useMemberActions';
import type { AppState } from '@/context/AppContext';

function makeState(overrides: Partial<AppState> = {}): AppState {
  return {
    phase: 'app',
    user: { id: 'u1', name: 'Test User', email: 'test@test.com', avatarColor: '#000', photo: null },
    activeTeamId: 'team1',
    sheet: null,
    form: {},
    formErrors: {},
    busy: null,
    toast: null,
    route: 'home',
    events: [],
    members: [
      {
        membershipId: 'ms1',
        userId: 'u2',
        name: 'Alice',
        email: 'alice@test.com',
        phone: '',
        birthday: '',
        address: '',
        group: '',
        photo: null,
        roles: [{ id: 'r1', name: 'Mitglied' }],
        avatarColor: '#aaa',
      },
    ],
    finances: null,
    stats: null,
    statsRange: null,
    news: [],
    polls: [],
    teams: [],
    roles: [],
    notifUnread: 0,
    notifications: [],
    primaryColor: '#000',
    ...overrides,
  } as unknown as AppState;
}

function makeApi() {
  return {
    stats: {
      attendanceFor: vi.fn().mockResolvedValue({ quote: 0.8, counted: 10, yes: 8 }),
    },
    members: {
      update: vi.fn().mockResolvedValue(undefined),
      setRoles: vi.fn().mockResolvedValue(undefined),
      remove: vi.fn().mockResolvedValue(undefined),
    },
    auth: {
      currentUser: vi.fn().mockResolvedValue({ id: 'u1', name: 'Updated User' }),
      setPhoto: vi.fn().mockResolvedValue(undefined),
    },
  };
}

describe('useMemberActions', () => {
  let setState: ReturnType<typeof vi.fn>;
  let toastMsg: ReturnType<typeof vi.fn>;
  let refreshMembers: ReturnType<typeof vi.fn>;
  let refreshTeams: ReturnType<typeof vi.fn>;
  let askConfirm: ReturnType<typeof vi.fn>;
  let logout: ReturnType<typeof vi.fn>;
  let api: ReturnType<typeof makeApi>;
  let stateRef: AppState;

  beforeEach(() => {
    stateRef = makeState();
    setState = vi.fn((patch) => {
      if (typeof patch === 'function') {
        const result = patch(stateRef);
        stateRef = { ...stateRef, ...result };
      } else {
        stateRef = { ...stateRef, ...patch };
      }
    });
    toastMsg = vi.fn();
    refreshMembers = vi.fn().mockResolvedValue(undefined);
    refreshTeams = vi.fn().mockResolvedValue(undefined);
    askConfirm = vi.fn();
    logout = vi.fn();
    api = makeApi();
  });

  function renderActions() {
    return renderHook(() =>
      useMemberActions({
        api: api as never,
        S: () => stateRef,
        setState: setState as never,
        refreshMembers: refreshMembers as never,
        refreshTeams: refreshTeams as never,
        askConfirm: askConfirm as never,
        toastMsg: toastMsg as never,
        logout: logout as never,
      }),
    );
  }

  it('openMemberDetail sets sheet and loads stats', async () => {
    const { result } = renderActions();
    await act(async () => {
      await result.current.openMemberDetail('ms1');
    });
    expect(setState).toHaveBeenCalledWith(
      expect.objectContaining({
        sheet: expect.objectContaining({ type: 'memberDetail', membershipId: 'ms1' }),
      }),
    );
    expect(api.stats.attendanceFor).toHaveBeenCalledWith('team1', 'u2');
  });

  it('openMemberDetail handles API errors gracefully', async () => {
    api.stats.attendanceFor = vi.fn().mockRejectedValue(new Error('Network error'));
    const { result } = renderActions();
    await act(async () => {
      await result.current.openMemberDetail('ms1');
    });
    expect(toastMsg).toHaveBeenCalled();
  });

  it('openMemberForm sets memberForm sheet with member data', () => {
    const member = stateRef.members![0];
    const { result } = renderActions();
    act(() => {
      result.current.openMemberForm(member as never);
    });
    expect(setState).toHaveBeenCalled();
    const call = setState.mock.calls[0][0];
    const patch = typeof call === 'function' ? call(stateRef) : call;
    expect(patch.sheet).toMatchObject({ type: 'memberForm', mode: 'edit' });
    expect(patch.form).toMatchObject({ name: 'Alice', email: 'alice@test.com' });
  });

  it('toggleFormRole adds role to form', () => {
    stateRef = makeState({ form: { roleIds: ['r1'] } });
    const { result } = renderActions();
    act(() => {
      result.current.toggleFormRole('r2');
    });
    // stateRef is updated by the mock setState
    expect(stateRef.form.roleIds).toContain('r2');
    expect(stateRef.form.roleIds).toContain('r1');
  });

  it('toggleFormRole removes role from form', () => {
    stateRef = makeState({ form: { roleIds: ['r1', 'r2'] } });
    const { result } = renderActions();
    act(() => {
      result.current.toggleFormRole('r2');
    });
    expect(stateRef.form.roleIds).not.toContain('r2');
    expect(stateRef.form.roleIds).toContain('r1');
  });

  it('toggleFormRole does not remove last role', () => {
    stateRef = makeState({ form: { roleIds: ['r1'] } });
    const { result } = renderActions();
    act(() => {
      result.current.toggleFormRole('r1');
    });
    expect(stateRef.form.roleIds).toContain('r1');
  });

  it('saveMember shows toast when name is empty', async () => {
    stateRef = makeState({
      form: { name: '', membershipId: 'ms1' },
      sheet: { type: 'memberForm', mode: 'edit', self: false } as never,
    });
    const { result } = renderActions();
    await act(async () => {
      await result.current.saveMember();
    });
    expect(toastMsg).toHaveBeenCalledWith('Name fehlt.');
    expect(api.members.update).not.toHaveBeenCalled();
  });

  it('saveMember shows toast when name is whitespace-only, without calling the API', async () => {
    stateRef = makeState({
      form: { name: '   ', membershipId: 'ms1' },
      sheet: { type: 'memberForm', mode: 'edit', self: false } as never,
    });
    const { result } = renderActions();
    await act(async () => {
      await result.current.saveMember();
    });
    expect(toastMsg).toHaveBeenCalledWith('Name fehlt.');
    expect(api.members.update).not.toHaveBeenCalled();
  });

  it('saveMember updates member data and shows toast', async () => {
    stateRef = makeState({
      form: { name: 'Alice Updated', email: 'alice@test.com', membershipId: 'ms1', roleIds: ['r1'] },
      sheet: { type: 'memberForm', mode: 'edit', self: false, back: null } as never,
    });
    const { result } = renderActions();
    await act(async () => {
      await result.current.saveMember();
    });
    expect(api.members.update).toHaveBeenCalledWith(
      'ms1',
      expect.objectContaining({ name: 'Alice Updated' }),
      'team1',
    );
    expect(toastMsg).toHaveBeenCalledWith('Profil gespeichert');
  });

  it('saveMember calls setRoles when the form roleIds differ from the member\'s current roles', async () => {
    stateRef = makeState({
      form: { name: 'Alice', email: 'alice@test.com', membershipId: 'ms1', roleIds: ['r1', 'r2'] },
      sheet: { type: 'memberForm', mode: 'edit', self: false, back: null } as never,
    });
    const { result } = renderActions();
    await act(async () => {
      await result.current.saveMember();
    });
    expect(api.members.setRoles).toHaveBeenCalledWith('ms1', ['r1', 'r2'], 'team1');
  });

  it('saveMember does not call setRoles when the form roleIds match the member\'s current roles (order-independent)', async () => {
    stateRef = makeState({
      members: [
        {
          membershipId: 'ms1',
          userId: 'u2',
          name: 'Alice',
          email: 'alice@test.com',
          phone: '',
          birthday: '',
          address: '',
          group: '',
          photo: null,
          roles: [
            { id: 'r1', name: 'Mitglied' },
            { id: 'r2', name: 'Trainer' },
          ],
          avatarColor: '#aaa',
        },
      ] as never,
      form: { name: 'Alice', email: 'alice@test.com', membershipId: 'ms1', roleIds: ['r2', 'r1'] },
      sheet: { type: 'memberForm', mode: 'edit', self: false, back: null } as never,
    });
    const { result } = renderActions();
    await act(async () => {
      await result.current.saveMember();
    });
    expect(api.members.setRoles).not.toHaveBeenCalled();
  });

  it('saveMember calls auth.setPhoto (not members.update) when saving your own changed photo', async () => {
    stateRef = makeState({
      form: {
        name: 'Alice',
        email: 'alice@test.com',
        membershipId: 'ms1',
        roleIds: ['r1'],
        photo: 'data:image/png;base64,newphoto',
      },
      sheet: { type: 'memberForm', mode: 'edit', self: true, back: null } as never,
    });
    const { result } = renderActions();
    await act(async () => {
      await result.current.saveMember();
    });
    expect(api.auth.setPhoto).toHaveBeenCalledWith('data:image/png;base64,newphoto');
    expect(api.members.update).toHaveBeenCalledWith(
      'ms1',
      expect.not.objectContaining({ photo: expect.anything() }),
      'team1',
    );
  });

  it('saveMember does not call auth.setPhoto when your own photo is unchanged', async () => {
    stateRef = makeState({
      form: { name: 'Alice', email: 'alice@test.com', membershipId: 'ms1', roleIds: ['r1'], photo: null },
      sheet: { type: 'memberForm', mode: 'edit', self: true, back: null } as never,
    });
    const { result } = renderActions();
    await act(async () => {
      await result.current.saveMember();
    });
    expect(api.auth.setPhoto).not.toHaveBeenCalled();
  });

  it('saveMember does not call auth.setPhoto when an admin edits someone else, even if the photo field changed', async () => {
    stateRef = makeState({
      form: {
        name: 'Alice',
        email: 'alice@test.com',
        membershipId: 'ms1',
        roleIds: ['r1'],
        photo: 'data:image/png;base64,newphoto',
      },
      sheet: { type: 'memberForm', mode: 'edit', self: false, back: null } as never,
    });
    const { result } = renderActions();
    await act(async () => {
      await result.current.saveMember();
    });
    expect(api.auth.setPhoto).not.toHaveBeenCalled();
  });

  it('saveMember trims leading/trailing whitespace from the name before saving', async () => {
    stateRef = makeState({
      form: { name: '  Alice Updated  ', email: 'alice@test.com', membershipId: 'ms1', roleIds: ['r1'] },
      sheet: { type: 'memberForm', mode: 'edit', self: false, back: null } as never,
    });
    const { result } = renderActions();
    await act(async () => {
      await result.current.saveMember();
    });
    expect(api.members.update).toHaveBeenCalledWith(
      'ms1',
      expect.objectContaining({ name: 'Alice Updated' }),
      'team1',
    );
  });

  it('saveMember refreshes teams and user when saving own profile', async () => {
    stateRef = makeState({
      form: { name: 'Self Updated', membershipId: 'ms1', roleIds: ['r1'] },
      sheet: { type: 'memberForm', mode: 'edit', self: true, back: null } as never,
    });
    const { result } = renderActions();
    await act(async () => {
      await result.current.saveMember();
    });
    expect(api.auth.currentUser).toHaveBeenCalled();
    expect(refreshTeams).toHaveBeenCalled();
  });

  it('removeMember calls askConfirm with member name', () => {
    const { result } = renderActions();
    act(() => {
      result.current.removeMember('ms1');
    });
    expect(askConfirm).toHaveBeenCalledWith(
      expect.objectContaining({
        title: 'Mitglied entfernen?',
        danger: true,
      }),
    );
    const cfg = askConfirm.mock.calls[0][0];
    expect(cfg.message).toContain('Alice');
  });

  it('removeMember onConfirm removes member and shows toast', async () => {
    const { result } = renderActions();
    act(() => {
      result.current.removeMember('ms1');
    });
    const cfg = askConfirm.mock.calls[0][0];
    await act(async () => {
      await cfg.onConfirm();
    });
    expect(api.members.remove).toHaveBeenCalledWith('ms1', 'team1');
    expect(toastMsg).toHaveBeenCalledWith('Mitglied entfernt');
  });
});
