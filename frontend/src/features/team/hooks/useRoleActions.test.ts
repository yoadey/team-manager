import { describe, it, expect, vi, beforeEach } from 'vitest';
import { renderHook, act } from '@testing-library/react';
import { useRoleActions } from './useRoleActions';
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
    members: [],
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

function makeActiveTeam(roleIds = ['r1']) {
  return {
    id: 'team1',
    name: 'Test Team',
    membershipId: 'ms1',
    myRoles: roleIds.map((id) => ({ id, name: `Role ${id}` })),
  };
}

describe('useRoleActions', () => {
  let setState: ReturnType<typeof vi.fn>;
  let toastMsg: ReturnType<typeof vi.fn>;
  let refreshRoles: ReturnType<typeof vi.fn>;
  let refreshTeams: ReturnType<typeof vi.fn>;
  let api: { roles: { create: ReturnType<typeof vi.fn> }; members: { setRoles: ReturnType<typeof vi.fn> } };
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
    refreshRoles = vi.fn().mockResolvedValue(undefined);
    refreshTeams = vi.fn().mockResolvedValue(undefined);
    api = {
      roles: { create: vi.fn().mockResolvedValue({ id: 'r-new' }) },
      members: { setRoles: vi.fn().mockResolvedValue(undefined) },
    };
  });

  function renderActions(roleIds = ['r1']) {
    return renderHook(() =>
      useRoleActions({
        api: api as never,
        S: () => stateRef,
        setState: setState as never,
        activeTeam: () => makeActiveTeam(roleIds) as never,
        refreshRoles: refreshRoles as never,
        refreshTeams: refreshTeams as never,
        toastMsg: toastMsg as never,
      }),
    );
  }

  it('openRoles sets roles sheet', () => {
    const { result } = renderActions();
    act(() => {
      result.current.openRoles();
    });
    expect(setState).toHaveBeenCalledWith({ sheet: { type: 'roles' } });
  });

  it('openCreateRole sets roleForm sheet with default permissions', () => {
    const { result } = renderActions();
    act(() => {
      result.current.openCreateRole();
    });
    expect(setState).toHaveBeenCalled();
    const patch = stateRef;
    expect(patch.sheet?.type).toBe('roleForm');
    expect(patch.form).toMatchObject({
      name: '',
      perms: expect.objectContaining({ events: 'read', finances: 'none' }),
    });
  });

  it('setRolePerm updates permission for a module', () => {
    stateRef = makeState({ form: { name: 'Test', perms: { events: 'read', finances: 'none' } } });
    const { result } = renderActions();
    act(() => {
      result.current.setRolePerm('finances', 'write');
    });
    expect((stateRef.form.perms as Record<string, string>).finances).toBe('write');
  });

  it('saveRole shows toast when name is empty', async () => {
    stateRef = makeState({ form: { name: '' } });
    const { result } = renderActions();
    await act(async () => {
      await result.current.saveRole();
    });
    expect(toastMsg).toHaveBeenCalledWith('Bitte Rollennamen angeben.');
    expect(api.roles.create).not.toHaveBeenCalled();
  });

  it('saveRole creates role and shows toast', async () => {
    stateRef = makeState({ form: { name: 'Trainer', perms: { events: 'write', finances: 'none' } } });
    const { result } = renderActions();
    await act(async () => {
      await result.current.saveRole();
    });
    expect(api.roles.create).toHaveBeenCalledWith('team1', expect.objectContaining({ name: 'Trainer' }));
    expect(toastMsg).toHaveBeenCalledWith('Rolle angelegt');
  });

  it('toggleMyRole adds role and shows toast', async () => {
    const { result } = renderActions(['r1']);
    await act(async () => {
      await result.current.toggleMyRole('r2');
    });
    expect(api.members.setRoles).toHaveBeenCalledWith('ms1', ['r1', 'r2'], 'team1');
    expect(toastMsg).toHaveBeenCalledWith('Rollen aktualisiert');
  });

  it('toggleMyRole removes role and shows toast', async () => {
    const { result } = renderActions(['r1', 'r2']);
    await act(async () => {
      await result.current.toggleMyRole('r2');
    });
    expect(api.members.setRoles).toHaveBeenCalledWith('ms1', ['r1'], 'team1');
  });

  it('toggleMyRole shows error when trying to remove last role', async () => {
    const { result } = renderActions(['r1']);
    await act(async () => {
      await result.current.toggleMyRole('r1');
    });
    expect(toastMsg).toHaveBeenCalledWith('Mindestens eine Rolle nötig.');
    expect(api.members.setRoles).not.toHaveBeenCalled();
  });
});
