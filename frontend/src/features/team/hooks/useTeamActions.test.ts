import { describe, it, expect, vi, beforeEach } from 'vitest';
import { renderHook, act } from '@testing-library/react';
import { useTeamActions } from './useTeamActions';
import { AuthError } from '@/utils/errors';
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

function makeActiveTeam() {
  return {
    id: 'team1',
    name: 'Test Team',
    description: 'A test team',
    icon: '⚽',
    iconBg: '#000',
    iconFg: '#fff',
    logo: null,
    photo: null,
    memberCount: 5,
    reasonVisibilityRoles: [],
  };
}

function makeApi() {
  return {
    absences: {
      listMine: vi.fn().mockResolvedValue([]),
    },
    teams: {
      updateSettings: vi.fn().mockResolvedValue(undefined),
      create: vi.fn().mockResolvedValue({ id: 'new-team', name: 'New Team' }),
      createInvite: vi.fn().mockResolvedValue({ link: 'https://example.com/invite/abc123' }),
    },
    auth: {
      setPhoto: vi.fn().mockResolvedValue(undefined),
      currentUser: vi.fn().mockResolvedValue({ id: 'u1', name: 'Test User' }),
    },
    notifications: {},
  };
}

describe('useTeamActions', () => {
  let setState: ReturnType<typeof vi.fn>;
  let toastMsg: ReturnType<typeof vi.fn>;
  let refreshTeams: ReturnType<typeof vi.fn>;
  let refreshMembers: ReturnType<typeof vi.fn>;
  let setFormVal: ReturnType<typeof vi.fn>;
  let afterLoginLoad: ReturnType<typeof vi.fn>;
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
    refreshTeams = vi.fn().mockResolvedValue(undefined);
    refreshMembers = vi.fn().mockResolvedValue(undefined);
    setFormVal = vi.fn();
    afterLoginLoad = vi.fn().mockResolvedValue(undefined);
    logout = vi.fn();
    api = makeApi();
  });

  function renderActions() {
    return renderHook(() =>
      useTeamActions({
        api: api as never,
        S: () => stateRef,
        setState: setState as never,
        activeTeam: () => makeActiveTeam() as never,
        refreshTeams: refreshTeams as never,
        refreshMembers: refreshMembers as never,
        setFormVal: setFormVal as never,
        afterLoginLoad: afterLoginLoad as never,
        toastMsg: toastMsg as never,
        logout: logout as never,
      }),
    );
  }

  it('openTeamSwitcher sets teams sheet', () => {
    const { result } = renderActions();
    act(() => {
      result.current.openTeamSwitcher();
    });
    expect(setState).toHaveBeenCalledWith({ sheet: { type: 'teams' } });
  });

  it('openProfile sets profile sheet and loads absences', async () => {
    const { result } = renderActions();
    await act(async () => {
      result.current.openProfile();
    });
    expect(setState).toHaveBeenCalledWith({ sheet: { type: 'profile' } });
    expect(api.absences.listMine).toHaveBeenCalled();
  });

  it('openMore sets more sheet', () => {
    const { result } = renderActions();
    act(() => {
      result.current.openMore();
    });
    expect(setState).toHaveBeenCalledWith({ sheet: { type: 'more' } });
  });

  it('openTeamSettings sets teamSettings sheet with team form data', () => {
    const { result } = renderActions();
    act(() => {
      result.current.openTeamSettings();
    });
    expect(setState).toHaveBeenCalledWith(
      expect.objectContaining({
        sheet: { type: 'teamSettings' },
        form: expect.objectContaining({ name: 'Test Team', icon: '⚽' }),
      }),
    );
  });

  it('saveTeamPhoto calls updateSettings and shows toast', async () => {
    const { result } = renderActions();
    await act(async () => {
      await result.current.saveTeamPhoto('data:image/png;base64,abc');
    });
    expect(api.teams.updateSettings).toHaveBeenCalledWith('team1', { photo: 'data:image/png;base64,abc' });
    expect(toastMsg).toHaveBeenCalledWith('Gruppenbild aktualisiert');
  });

  it('removeTeamPhoto calls updateSettings with null and shows toast', async () => {
    // setFormVal is only applied while still on the teamSettings sheet for
    // the same team the removal was issued for -- see the team-switch race
    // regression test below.
    stateRef = makeState({ sheet: { type: 'teamSettings' } as never });
    const { result } = renderActions();
    await act(async () => {
      await result.current.removeTeamPhoto();
    });
    expect(api.teams.updateSettings).toHaveBeenCalledWith('team1', { photo: null });
    expect(setFormVal).toHaveBeenCalledWith({ photo: null });
    expect(toastMsg).toHaveBeenCalledWith('Gruppenbild entfernt');
  });

  // Regression test: setFormVal writes into the single shared, untyped form
  // buffer regardless of which sheet is open. A slow photo upload for team1
  // used to unconditionally apply its result even after the user had since
  // switched away from the teamSettings sheet (e.g. to CreateTeamSheet,
  // which reads the same form.photo field for its own, unrelated preview).
  it('saveTeamPhoto does not touch the form if the user left the teamSettings sheet while the save was in flight', async () => {
    let resolveUpdate!: () => void;
    api.teams.updateSettings = vi.fn(() => new Promise<void>((resolve) => (resolveUpdate = resolve)));
    stateRef = makeState({ sheet: { type: 'teamSettings' } as never });
    const { result } = renderActions();

    let savePromise!: Promise<void>;
    act(() => {
      savePromise = result.current.saveTeamPhoto('data:image/png;base64,fromTeam1');
    });
    expect(api.teams.updateSettings).toHaveBeenCalled();

    // User navigates away to a different sheet before the upload resolves.
    stateRef = { ...stateRef, sheet: { type: 'createTeam' } as never };

    await act(async () => {
      resolveUpdate();
      await savePromise;
    });

    expect(setFormVal).not.toHaveBeenCalled();
  });

  it('removeTeamPhoto ignores a second call while the first is still in flight', async () => {
    let resolveUpdate: () => void;
    api.teams.updateSettings.mockReturnValue(
      new Promise<void>((resolve) => {
        resolveUpdate = () => resolve(undefined);
      }),
    );
    const { result } = renderActions();

    let firstCall: Promise<void>;
    act(() => {
      firstCall = result.current.removeTeamPhoto();
      void result.current.removeTeamPhoto();
    });

    expect(api.teams.updateSettings).toHaveBeenCalledTimes(1);

    await act(async () => {
      resolveUpdate();
      await firstCall;
    });
  });

  it('removeTeamPhoto and saveTeamPhoto share the same in-flight guard key', async () => {
    let resolveUpdate: () => void;
    api.teams.updateSettings.mockReturnValue(
      new Promise<void>((resolve) => {
        resolveUpdate = () => resolve(undefined);
      }),
    );
    const { result } = renderActions();

    let firstCall: Promise<void>;
    act(() => {
      firstCall = result.current.saveTeamPhoto('data:image/png;base64,first');
      void result.current.removeTeamPhoto();
    });

    expect(api.teams.updateSettings).toHaveBeenCalledTimes(1);

    await act(async () => {
      resolveUpdate();
      await firstCall;
    });
  });

  it('saveTeamLogo calls updateSettings and shows toast', async () => {
    const { result } = renderActions();
    await act(async () => {
      await result.current.saveTeamLogo('data:image/png;base64,logo');
    });
    expect(api.teams.updateSettings).toHaveBeenCalledWith('team1', { logo: 'data:image/png;base64,logo' });
    expect(toastMsg).toHaveBeenCalledWith('Logo aktualisiert');
  });

  it('saveTeamPhoto ignores a second call while the first is still in flight', async () => {
    let resolveUpdate: () => void;
    api.teams.updateSettings.mockReturnValue(
      new Promise<void>((resolve) => {
        resolveUpdate = () => resolve(undefined);
      }),
    );
    const { result } = renderActions();

    let firstCall: Promise<void>;
    act(() => {
      firstCall = result.current.saveTeamPhoto('data:image/png;base64,first');
      void result.current.saveTeamPhoto('data:image/png;base64,second');
    });

    expect(api.teams.updateSettings).toHaveBeenCalledTimes(1);
    expect(api.teams.updateSettings).toHaveBeenCalledWith('team1', { photo: 'data:image/png;base64,first' });

    await act(async () => {
      resolveUpdate();
      await firstCall;
    });
  });

  it('saveTeamPhoto allows a new call once the previous one has settled', async () => {
    const { result } = renderActions();
    await act(async () => {
      await result.current.saveTeamPhoto('data:image/png;base64,first');
    });
    await act(async () => {
      await result.current.saveTeamPhoto('data:image/png;base64,second');
    });
    expect(api.teams.updateSettings).toHaveBeenCalledTimes(2);
  });

  it('saveTeamPhoto clears the in-flight guard even when the request fails', async () => {
    api.teams.updateSettings.mockRejectedValueOnce(new Error('boom'));
    const { result } = renderActions();
    await act(async () => {
      await result.current.saveTeamPhoto('data:image/png;base64,first');
    });
    await act(async () => {
      await result.current.saveTeamPhoto('data:image/png;base64,second');
    });
    expect(api.teams.updateSettings).toHaveBeenCalledTimes(2);
  });

  it('saveTeamLogo ignores a second call while the first is still in flight', async () => {
    let resolveUpdate: () => void;
    api.teams.updateSettings.mockReturnValue(
      new Promise<void>((resolve) => {
        resolveUpdate = () => resolve(undefined);
      }),
    );
    const { result } = renderActions();

    let firstCall: Promise<void>;
    act(() => {
      firstCall = result.current.saveTeamLogo('data:image/png;base64,first');
      void result.current.saveTeamLogo('data:image/png;base64,second');
    });

    expect(api.teams.updateSettings).toHaveBeenCalledTimes(1);
    expect(api.teams.updateSettings).toHaveBeenCalledWith('team1', { logo: 'data:image/png;base64,first' });

    await act(async () => {
      resolveUpdate();
      await firstCall;
    });
  });

  it('saveTeamPhoto and saveTeamLogo track in-flight state independently', async () => {
    let resolveUpdate: () => void;
    api.teams.updateSettings.mockReturnValue(
      new Promise<void>((resolve) => {
        resolveUpdate = () => resolve(undefined);
      }),
    );
    const { result } = renderActions();

    let firstCall: Promise<void>;
    act(() => {
      firstCall = result.current.saveTeamPhoto('data:image/png;base64,photo');
      void result.current.saveTeamLogo('data:image/png;base64,logo');
    });

    expect(api.teams.updateSettings).toHaveBeenCalledTimes(2);

    await act(async () => {
      resolveUpdate();
      await firstCall;
    });
  });

  it('setTeamIcon calls setFormVal and updateSettings', async () => {
    const { result } = renderActions();
    await act(async () => {
      result.current.setTeamIcon('🏆');
    });
    expect(setFormVal).toHaveBeenCalledWith({ icon: '🏆', logo: null });
    expect(api.teams.updateSettings).toHaveBeenCalledWith('team1', { icon: '🏆', logo: null });
  });

  it('toggleReasonRole adds role to reasonRoles', () => {
    stateRef = makeState({ form: { reasonRoles: ['r1'] } });
    const { result } = renderActions();
    act(() => {
      result.current.toggleReasonRole('r2');
    });
    expect(stateRef.form.reasonRoles).toContain('r2');
    expect(stateRef.form.reasonRoles).toContain('r1');
  });

  it('toggleReasonRole removes role from reasonRoles', () => {
    stateRef = makeState({ form: { reasonRoles: ['r1', 'r2'] } });
    const { result } = renderActions();
    act(() => {
      result.current.toggleReasonRole('r2');
    });
    expect(stateRef.form.reasonRoles).not.toContain('r2');
  });

  it('saveTeamSettings shows toast when name is empty', async () => {
    stateRef = makeState({ form: { name: '' } });
    const { result } = renderActions();
    await act(async () => {
      await result.current.saveTeamSettings();
    });
    expect(toastMsg).toHaveBeenCalledWith('Bitte Team-Namen angeben.');
    expect(api.teams.updateSettings).not.toHaveBeenCalled();
  });

  it('saveTeamSettings updates settings and shows toast', async () => {
    stateRef = makeState({ form: { name: 'Updated Team', description: 'Desc', reasonRoles: [] } });
    const { result } = renderActions();
    await act(async () => {
      await result.current.saveTeamSettings();
    });
    expect(api.teams.updateSettings).toHaveBeenCalledWith('team1', expect.objectContaining({ name: 'Updated Team' }));
    expect(toastMsg).toHaveBeenCalledWith('Team-Einstellungen gespeichert');
  });

  it('saveTeamSettings triggers logout on a 401 (expired session)', async () => {
    stateRef = makeState({ form: { name: 'Updated Team', description: 'Desc', reasonRoles: [] } });
    api.teams.updateSettings.mockRejectedValueOnce(new AuthError());
    const { result } = renderActions();
    await act(async () => {
      await result.current.saveTeamSettings();
    });
    expect(logout).toHaveBeenCalled();
  });

  it('openCreateTeam sets createTeam sheet', () => {
    const { result } = renderActions();
    act(() => {
      result.current.openCreateTeam();
    });
    expect(setState).toHaveBeenCalledWith({
      sheet: { type: 'createTeam' },
      form: { name: '', icon: '⭐', photo: null },
    });
  });

  it('createTeam shows toast when name is empty', async () => {
    stateRef = makeState({ form: { name: '' } });
    const { result } = renderActions();
    await act(async () => {
      await result.current.createTeam();
    });
    expect(toastMsg).toHaveBeenCalledWith('Bitte Team-Namen angeben.');
    expect(api.teams.create).not.toHaveBeenCalled();
  });

  it('createTeam creates team and shows toast', async () => {
    stateRef = makeState({ form: { name: 'My New Team', icon: '⭐', photo: null } });
    const { result } = renderActions();
    await act(async () => {
      await result.current.createTeam();
    });
    expect(api.teams.create).toHaveBeenCalledWith(expect.objectContaining({ name: 'My New Team' }));
    expect(toastMsg).toHaveBeenCalledWith('Team angelegt – du bist Admin');
    expect(afterLoginLoad).toHaveBeenCalledWith('new-team');
  });

  it('openInvite sets invite sheet and loads invite link', async () => {
    const { result } = renderActions();
    await act(async () => {
      await result.current.openInvite();
    });
    expect(api.teams.createInvite).toHaveBeenCalledWith('team1');
    expect(setState).toHaveBeenCalledWith({ sheet: { type: 'invite', invite: null } });
  });

  // Regression test: the response handler used to check only
  // `sheet.type === 'invite'`, never the team the invite was created for.
  // A slow createInvite() for team1, if the user switched to a different
  // team and opened ITS OWN invite sheet (also type 'invite') before it
  // resolved, would inject team1's invite link/code into the sheet the user
  // believes belongs to the new team -- a cross-team invite-token leak, not
  // just stale data.
  it('does not inject a stale invite into a different team\'s invite sheet opened afterward', async () => {
    let resolveCreate!: (v: { link: string; code: string }) => void;
    api.teams.createInvite = vi.fn(() => new Promise((resolve) => (resolveCreate = resolve)));
    const { result } = renderActions();

    let openPromise!: Promise<void>;
    act(() => {
      openPromise = result.current.openInvite();
    });
    expect(api.teams.createInvite).toHaveBeenCalledWith('team1');

    // User switches teams and opens THAT team's own (also empty) invite sheet.
    stateRef = { ...stateRef, activeTeamId: 'team2', sheet: { type: 'invite', invite: null } as never };

    await act(async () => {
      resolveCreate({ link: 'https://example.com/invite/team1-code', code: 'team1-code' });
      await openPromise;
    });

    expect(stateRef.sheet).toEqual({ type: 'invite', invite: null });
  });

  it('copyInvite does nothing when no invite in sheet', async () => {
    stateRef = makeState({ sheet: { type: 'invite', invite: null } as never });
    const { result } = renderActions();
    await act(async () => {
      await result.current.copyInvite();
    });
    expect(toastMsg).not.toHaveBeenCalled();
  });

  it('copyInvite shows toast when invite link exists', async () => {
    stateRef = makeState({ sheet: { type: 'invite', invite: { link: 'https://example.com/abc' } } as never });
    Object.assign(navigator, { clipboard: { writeText: vi.fn().mockResolvedValue(undefined) } });
    const { result } = renderActions();
    await act(async () => {
      await result.current.copyInvite();
    });
    expect(toastMsg).toHaveBeenCalledWith('Link kopiert');
  });

  it('copyInvite shows an error toast when the clipboard write fails', async () => {
    stateRef = makeState({ sheet: { type: 'invite', invite: { link: 'https://example.com/abc' } } as never });
    Object.assign(navigator, { clipboard: { writeText: vi.fn().mockRejectedValue(new Error('denied')) } });
    const { result } = renderActions();
    await act(async () => {
      await result.current.copyInvite();
    });
    expect(toastMsg).toHaveBeenCalledWith('Kopieren fehlgeschlagen');
  });

  it('uploadMyPhoto updates user photo and shows toast', async () => {
    const { result } = renderActions();
    await act(async () => {
      await result.current.uploadMyPhoto('data:image/png;base64,photo');
    });
    expect(api.auth.setPhoto).toHaveBeenCalledWith('data:image/png;base64,photo');
    expect(refreshTeams).toHaveBeenCalled();
    expect(refreshMembers).toHaveBeenCalled();
    expect(toastMsg).toHaveBeenCalledWith('Profilfoto aktualisiert');
  });

  // Regression test: unlike its siblings saveTeamPhoto/removeTeamPhoto/
  // saveTeamLogo, uploadMyPhoto had no in-flight guard -- a user reopening
  // the file picker before the first upload resolved could fire two
  // concurrent api.auth.setPhoto calls, and whichever response landed last
  // silently determined the final avatar regardless of which photo the user
  // picked last.
  it('uploadMyPhoto ignores a second call while the first is still in flight', async () => {
    let resolveSetPhoto: () => void;
    api.auth.setPhoto.mockReturnValue(
      new Promise<void>((resolve) => {
        resolveSetPhoto = () => resolve(undefined);
      }),
    );
    const { result } = renderActions();

    let firstCall: Promise<void>;
    act(() => {
      firstCall = result.current.uploadMyPhoto('data:image/png;base64,first');
      void result.current.uploadMyPhoto('data:image/png;base64,second');
    });

    expect(api.auth.setPhoto).toHaveBeenCalledTimes(1);
    expect(api.auth.setPhoto).toHaveBeenCalledWith('data:image/png;base64,first');

    await act(async () => {
      resolveSetPhoto();
      await firstCall;
    });
  });

  it('uploadMyPhoto allows a new call once the previous one has settled', async () => {
    const { result } = renderActions();
    await act(async () => {
      await result.current.uploadMyPhoto('data:image/png;base64,first');
    });
    await act(async () => {
      await result.current.uploadMyPhoto('data:image/png;base64,second');
    });
    expect(api.auth.setPhoto).toHaveBeenCalledTimes(2);
  });
});
