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
  let askConfirm: ReturnType<typeof vi.fn>;
  let logout: ReturnType<typeof vi.fn>;
  let api: {
    roles: { create: ReturnType<typeof vi.fn>; update: ReturnType<typeof vi.fn>; remove: ReturnType<typeof vi.fn> };
    members: { setRoles: ReturnType<typeof vi.fn> };
  };
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
    askConfirm = vi.fn((cfg) => cfg.onConfirm());
    logout = vi.fn();
    api = {
      roles: {
        create: vi.fn().mockResolvedValue({ id: 'r-new' }),
        update: vi.fn().mockResolvedValue({ id: 'r1' }),
        remove: vi.fn().mockResolvedValue(undefined),
      },
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
        askConfirm: askConfirm as never,
        toastMsg: toastMsg as never,
        logout: logout as never,
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

  it('openRoleForm with no argument sets roleForm sheet in create mode with default permissions', () => {
    const { result } = renderActions();
    act(() => {
      result.current.openRoleForm();
    });
    expect(setState).toHaveBeenCalled();
    const patch = stateRef;
    expect(patch.sheet?.type).toBe('roleForm');
    expect(patch.sheet?.mode).toBe('create');
    expect(patch.form).toMatchObject({
      name: '',
      perms: expect.objectContaining({ events: 'read', finances: 'none' }),
    });
  });

  it('openRoleForm with a role pre-fills the form in edit mode', () => {
    const { result } = renderActions();
    const role = {
      id: 'r1',
      name: 'Trainer',
      permissions: { events: 'write', members: 'none', finances: 'none', news: 'none', polls: 'none', settings: 'none' },
    };
    act(() => {
      result.current.openRoleForm(role as never);
    });
    const patch = stateRef;
    expect(patch.sheet?.type).toBe('roleForm');
    expect(patch.sheet?.mode).toBe('edit');
    expect(patch.form).toMatchObject({ id: 'r1', name: 'Trainer', perms: role.permissions });
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

  it('saveRole updates an existing role (form has id) instead of creating', async () => {
    stateRef = makeState({ form: { id: 'r1', name: 'Renamed', perms: { events: 'write', finances: 'none' } } });
    const { result } = renderActions();
    await act(async () => {
      await result.current.saveRole();
    });
    expect(api.roles.update).toHaveBeenCalledWith(
      'r1',
      expect.objectContaining({ name: 'Renamed' }),
      'team1',
    );
    expect(api.roles.create).not.toHaveBeenCalled();
    expect(toastMsg).toHaveBeenCalledWith('Rolle aktualisiert');
  });

  it('saveRole shows toast when name is whitespace-only, without calling the API', async () => {
    stateRef = makeState({ form: { name: '   ' } });
    const { result } = renderActions();
    await act(async () => {
      await result.current.saveRole();
    });
    expect(toastMsg).toHaveBeenCalledWith('Bitte Rollennamen angeben.');
    expect(api.roles.create).not.toHaveBeenCalled();
  });

  it('saveRole trims leading/trailing whitespace from the name before saving', async () => {
    stateRef = makeState({ form: { name: '  Trainer  ', perms: { events: 'write', finances: 'none' } } });
    const { result } = renderActions();
    await act(async () => {
      await result.current.saveRole();
    });
    expect(api.roles.create).toHaveBeenCalledWith('team1', expect.objectContaining({ name: 'Trainer' }));
  });

  // Regression: saveRole used to navigate to the roles sheet unconditionally
  // (once the team still matched), so a slow save could clobber whatever
  // DIFFERENT sheet the user had since opened while it was in flight.
  it('saveRole does not touch the sheet if the user opened something else while in flight', async () => {
    let resolveUpdate!: (v: { id: string }) => void;
    api.roles.update = vi.fn(() => new Promise((resolve) => (resolveUpdate = resolve)));
    stateRef = makeState({
      form: { id: 'r1', name: 'Renamed', perms: { events: 'write', finances: 'none' } },
      sheet: { type: 'roleForm', mode: 'edit' } as never,
    });
    const { result } = renderActions();

    let savePromise!: Promise<void>;
    act(() => {
      savePromise = result.current.saveRole();
    });

    const somethingElse = { type: 'teams' } as never;
    stateRef = { ...stateRef, sheet: somethingElse };

    await act(async () => {
      resolveUpdate({ id: 'r1' });
      await savePromise;
    });

    expect(stateRef.sheet).toBe(somethingElse);
  });

  it('removeRole asks for confirmation, then removes the role and shows toast', async () => {
    const { result } = renderActions();
    await act(async () => {
      result.current.removeRole('r1');
      await Promise.resolve();
    });
    expect(askConfirm).toHaveBeenCalledWith(
      expect.objectContaining({ danger: true, onConfirm: expect.any(Function) }),
    );
    expect(api.roles.remove).toHaveBeenCalledWith('r1', 'team1');
    expect(refreshRoles).toHaveBeenCalled();
    expect(toastMsg).toHaveBeenCalledWith('Rolle gelöscht');
  });

  it('removeRole reports an error without removing on API failure', async () => {
    api.roles.remove.mockRejectedValueOnce(new Error('boom'));
    const { result } = renderActions();
    await act(async () => {
      result.current.removeRole('r1');
      await Promise.resolve();
    });
    expect(toastMsg).toHaveBeenCalled();
    expect(refreshRoles).not.toHaveBeenCalled();
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
