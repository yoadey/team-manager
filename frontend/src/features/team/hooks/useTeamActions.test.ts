import { describe, it, expect, vi, beforeEach } from 'vitest';
import { renderHook, act } from '@testing-library/react';
import { useTeamActions } from './useTeamActions';
import type { QueryClient } from '@tanstack/react-query';
import { createQueryWrapper, createTestQueryClient } from '@/test/queryTestUtils';
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
    absences: {
      listMine: vi.fn().mockResolvedValue([]),
    },
    teams: {
      updateSettings: vi.fn().mockResolvedValue(undefined),
      create: vi.fn().mockResolvedValue({ id: 'new-team', name: 'New Team' }),
      createInvite: vi.fn().mockResolvedValue({ link: 'https://example.com/invite/abc123' }),
    },
    stats: {
      setGroupThresholds: vi.fn().mockResolvedValue([]),
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
  let invalidateMembers: ReturnType<typeof vi.fn>;
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
    invalidateMembers = vi.fn().mockResolvedValue(undefined);
    afterLoginLoad = vi.fn().mockResolvedValue(undefined);
    logout = vi.fn();
    api = makeApi();
  });

  function renderActions(client?: QueryClient) {
    return renderHook(
      () =>
        useTeamActions({
          api: api as never,
          S: () => stateRef,
          setState: setState as never,
          refreshTeams: refreshTeams as never,
          invalidateMembers: invalidateMembers as never,
          afterLoginLoad: afterLoginLoad as never,
          toastMsg: toastMsg as never,
          logout: logout as never,
        }),
      { wrapper: createQueryWrapper(client) },
    );
  }

  it('openTeamSwitcher sets teams sheet', () => {
    const { result } = renderActions();
    act(() => {
      result.current.openTeamSwitcher();
    });
    expect(setState).toHaveBeenCalledWith({ sheet: { type: 'teams' } });
  });

  it('openMore sets more sheet', () => {
    const { result } = renderActions();
    act(() => {
      result.current.openMore();
    });
    expect(setState).toHaveBeenCalledWith({ sheet: { type: 'more' } });
  });

  it('openTeamSettings sets teamSettings sheet', () => {
    const { result } = renderActions();
    act(() => {
      result.current.openTeamSettings();
    });
    expect(setState).toHaveBeenCalledWith({ sheet: { type: 'teamSettings' } });
  });

  it('saveTeamPhoto calls updateSettings and shows toast', async () => {
    const { result } = renderActions();
    await act(async () => {
      await result.current.saveTeamPhoto('data:image/png;base64,abc', 'team1');
    });
    expect(api.teams.updateSettings).toHaveBeenCalledWith('team1', { photo: 'data:image/png;base64,abc' });
    expect(toastMsg).toHaveBeenCalledWith('Gruppenbild aktualisiert');
  });

  it('removeTeamPhoto calls updateSettings with null and shows toast', async () => {
    stateRef = makeState({ sheet: { type: 'teamSettings' } as never });
    const { result } = renderActions();
    await act(async () => {
      await result.current.removeTeamPhoto();
    });
    expect(api.teams.updateSettings).toHaveBeenCalledWith('team1', { photo: null });
    expect(toastMsg).toHaveBeenCalledWith('Gruppenbild entfernt');
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
      firstCall = result.current.saveTeamPhoto('data:image/png;base64,first', 'team1');
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
      await result.current.saveTeamLogo('data:image/png;base64,logo', 'team1');
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
      firstCall = result.current.saveTeamPhoto('data:image/png;base64,first', 'team1');
      void result.current.saveTeamPhoto('data:image/png;base64,second', 'team1');
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
      await result.current.saveTeamPhoto('data:image/png;base64,first', 'team1');
    });
    await act(async () => {
      await result.current.saveTeamPhoto('data:image/png;base64,second', 'team1');
    });
    expect(api.teams.updateSettings).toHaveBeenCalledTimes(2);
  });

  it('saveTeamPhoto clears the in-flight guard even when the request fails', async () => {
    api.teams.updateSettings.mockRejectedValueOnce(new Error('boom'));
    const { result } = renderActions();
    await act(async () => {
      await result.current.saveTeamPhoto('data:image/png;base64,first', 'team1');
    });
    await act(async () => {
      await result.current.saveTeamPhoto('data:image/png;base64,second', 'team1');
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
      firstCall = result.current.saveTeamLogo('data:image/png;base64,first', 'team1');
      void result.current.saveTeamLogo('data:image/png;base64,second', 'team1');
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
      firstCall = result.current.saveTeamPhoto('data:image/png;base64,photo', 'team1');
      void result.current.saveTeamLogo('data:image/png;base64,logo', 'team1');
    });

    expect(api.teams.updateSettings).toHaveBeenCalledTimes(2);

    await act(async () => {
      resolveUpdate();
      await firstCall;
    });
  });

  it('setTeamIcon calls updateSettings', async () => {
    stateRef = makeState({ sheet: { type: 'teamSettings' } as never });
    const { result } = renderActions();
    await act(async () => {
      await result.current.setTeamIcon('🏆');
    });
    expect(api.teams.updateSettings).toHaveBeenCalledWith('team1', { icon: '🏆', logo: null });
  });

  it('setTeamIcon reports an error without crashing when updateSettings fails', async () => {
    stateRef = makeState({ sheet: { type: 'teamSettings' } as never });
    vi.mocked(api.teams.updateSettings).mockRejectedValueOnce(new Error('boom'));
    const { result } = renderActions();
    await act(async () => {
      await result.current.setTeamIcon('🏆');
    });
    expect(toastMsg).toHaveBeenCalled();
  });

  it('saveTeamSettings updates settings and shows toast', async () => {
    const { result } = renderActions();
    await act(async () => {
      await result.current.saveTeamSettings({ name: 'Updated Team', description: 'Desc', reasonRoles: [] } as never);
    });
    expect(api.teams.updateSettings).toHaveBeenCalledWith('team1', expect.objectContaining({ name: 'Updated Team' }));
    expect(toastMsg).toHaveBeenCalledWith('Team-Einstellungen gespeichert');
  });

  it('saveTeamSettings sends no threshold request when no threshold changed', async () => {
    const { result } = renderActions();
    await act(async () => {
      await result.current.saveTeamSettings({ name: 'Updated Team', description: '', reasonRoles: [] } as never);
      await result.current.saveTeamSettings({ name: 'Updated Team', description: '', reasonRoles: [] } as never, []);
    });
    expect(api.stats.setGroupThresholds).not.toHaveBeenCalled();
  });

  it('saveTeamSettings saves changed thresholds and invalidates the threshold + stats caches', async () => {
    const client = createTestQueryClient();
    const invalidate = vi.spyOn(client, 'invalidateQueries');
    const changed = [{ roleId: 'r1', mode: 'count' as const, value: 8 }];
    const { result } = renderActions(client);
    await act(async () => {
      await result.current.saveTeamSettings({ name: 'Updated Team', description: '', reasonRoles: [] } as never, changed);
    });
    expect(api.teams.updateSettings).toHaveBeenCalledWith('team1', expect.objectContaining({ name: 'Updated Team' }));
    expect(api.stats.setGroupThresholds).toHaveBeenCalledWith('team1', changed);
    expect(invalidate).toHaveBeenCalledWith({ queryKey: ['teams', 'team1', 'statsGroupThresholds'] });
    // Prefix of every range-scoped overview/matrix key of the team.
    expect(invalidate).toHaveBeenCalledWith({ queryKey: ['teams', 'team1', 'stats'] });
    expect(toastMsg).toHaveBeenCalledWith('Team-Einstellungen gespeichert');
  });

  // The two requests are not atomic: when the threshold save fails, the
  // team's own settings are already stored. The failure must say so (not
  // claim nothing was saved), the team list must still refresh, and the
  // threshold list must re-sync with the server.
  it('saveTeamSettings reports a failed threshold save as a partial save', async () => {
    api.stats.setGroupThresholds.mockRejectedValueOnce(new Error('boom'));
    const client = createTestQueryClient();
    const invalidate = vi.spyOn(client, 'invalidateQueries');
    const { result } = renderActions(client);
    await act(async () => {
      await expect(
        result.current.saveTeamSettings({ name: 'Updated Team', description: '', reasonRoles: [] } as never, [
          { roleId: 'r1', mode: 'percent', value: 60 },
        ]),
      ).rejects.toThrow('boom');
    });
    expect(api.teams.updateSettings).toHaveBeenCalled();
    expect(refreshTeams).toHaveBeenCalledTimes(1);
    expect(invalidate).toHaveBeenCalledWith({ queryKey: ['teams', 'team1', 'statsGroupThresholds'] });
    // Exactly one toast: the specific partial-save message, as an error.
    expect(toastMsg).toHaveBeenCalledTimes(1);
    expect(toastMsg).toHaveBeenCalledWith(
      expect.stringContaining('Team-Einstellungen gespeichert, aber die Mindest-Anwesenheit je Gruppe konnte nicht gespeichert werden'),
      undefined,
      'error',
    );
    expect(toastMsg).not.toHaveBeenCalledWith('Team-Einstellungen gespeichert');
  });

  it('saveTeamSettings sends no threshold request when the team settings themselves fail', async () => {
    api.teams.updateSettings.mockRejectedValueOnce(new Error('nope'));
    const { result } = renderActions();
    await act(async () => {
      await expect(
        result.current.saveTeamSettings({ name: 'Updated Team', description: '', reasonRoles: [] } as never, [
          { roleId: 'r1', mode: 'percent', value: 60 },
        ]),
      ).rejects.toThrow('nope');
    });
    expect(api.stats.setGroupThresholds).not.toHaveBeenCalled();
    expect(refreshTeams).not.toHaveBeenCalled();
    expect(toastMsg).toHaveBeenCalledTimes(1);
  });

  it('saveTeamSettings logs out on a 401 from the threshold save, reporting it once', async () => {
    api.stats.setGroupThresholds.mockRejectedValueOnce(new AuthError());
    const { result } = renderActions();
    await act(async () => {
      await expect(
        result.current.saveTeamSettings({ name: 'Updated Team', description: '', reasonRoles: [] } as never, [
          { roleId: 'r1', mode: 'percent', value: 60 },
        ]),
      ).rejects.toBeInstanceOf(AuthError);
    });
    expect(logout).toHaveBeenCalledTimes(1);
    expect(toastMsg).toHaveBeenCalledTimes(1);
  });

  it('saveTeamSettings triggers logout on a 401 (expired session)', async () => {
    api.teams.updateSettings.mockRejectedValueOnce(new AuthError());
    const { result } = renderActions();
    await act(async () => {
      await expect(
        result.current.saveTeamSettings({ name: 'Updated Team', description: 'Desc', reasonRoles: [] } as never),
      ).rejects.toThrow();
    });
    expect(logout).toHaveBeenCalled();
  });

  it('openCreateTeam sets createTeam sheet', () => {
    const { result } = renderActions();
    act(() => {
      result.current.openCreateTeam();
    });
    expect(setState).toHaveBeenCalledWith({ sheet: { type: 'createTeam' } });
  });

  it('createTeam creates team and shows toast', async () => {
    const { result } = renderActions();
    await act(async () => {
      await result.current.createTeam({ name: 'My New Team', icon: '⭐', photo: null } as never);
    });
    expect(api.teams.create).toHaveBeenCalledWith(expect.objectContaining({ name: 'My New Team' }));
    expect(toastMsg).toHaveBeenCalledWith('Team angelegt – du bist Admin');
    expect(afterLoginLoad).toHaveBeenCalledWith('new-team');
  });

  // Regression: createTeam used to navigate into the new team and close the
  // sheet unconditionally, so a slow create could clobber whatever DIFFERENT
  // sheet the user had since opened (or switch them away from where they
  // navigated to) while the request was in flight. The team still gets
  // created either way (refreshTeams already ran) -- only the forced
  // navigation/sheet-close is skipped.
  it('createTeam does not touch the sheet or navigate if the user opened something else while in flight', async () => {
    let resolveCreate!: (v: { id: string; name: string }) => void;
    api.teams.create = vi.fn(() => new Promise((resolve) => (resolveCreate = resolve)));
    stateRef = makeState({
      sheet: { type: 'createTeam' } as never,
    });
    const { result } = renderActions();

    let createPromise!: Promise<void>;
    act(() => {
      createPromise = result.current.createTeam({ name: 'My New Team', icon: '⭐', photo: null } as never);
    });

    const somethingElse = { type: 'teams' } as never;
    stateRef = { ...stateRef, sheet: somethingElse };

    await act(async () => {
      resolveCreate({ id: 'new-team', name: 'New Team' });
      await createPromise;
    });

    expect(stateRef.sheet).toBe(somethingElse);
    expect(stateRef.activeTeamId).not.toBe('new-team');
    expect(afterLoginLoad).not.toHaveBeenCalled();
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
  it("does not inject a stale invite into a different team's invite sheet opened afterward", async () => {
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

  // Regression test: openInvite has no busy flag, so it can be invoked twice
  // in a row for the SAME team (open, close, reopen before the first request
  // resolves) -- the old type+activeTeamId check couldn't tell the two
  // invite sheets apart, so the first (now-stale) request's late
  // success/failure handler would clobber the second (still in-flight)
  // sheet. Must key on the exact sheet object reference instead.
  it('a late-failing first openInvite call does not close a second, still-pending invite sheet', async () => {
    let rejectFirst!: (err: Error) => void;
    let resolveSecond!: (v: { link: string; code: string }) => void;
    api.teams.createInvite = vi
      .fn()
      .mockImplementationOnce(() => new Promise((_, reject) => (rejectFirst = reject)))
      .mockImplementationOnce(() => new Promise((resolve) => (resolveSecond = resolve)));
    const { result } = renderActions();

    let firstPromise!: Promise<void>;
    act(() => {
      firstPromise = result.current.openInvite();
    });
    let secondPromise!: Promise<void>;
    act(() => {
      secondPromise = result.current.openInvite();
    });

    await act(async () => {
      rejectFirst(new Error('boom'));
      await firstPromise;
    });

    // The first call's failure must not have closed the second's sheet.
    expect(stateRef.sheet).toEqual({ type: 'invite', invite: null });

    await act(async () => {
      resolveSecond({ link: 'https://example.com/invite/team1-code', code: 'team1-code' });
      await secondPromise;
    });
    expect(stateRef.sheet).toEqual({
      type: 'invite',
      invite: { link: 'https://example.com/invite/team1-code', code: 'team1-code' },
    });
  });

  // Regression test: InviteSheet shows an eternal "wird generiert..."
  // placeholder while sheet.invite is null, with no error state to fall
  // back to -- a failed createInvite() (permission downgrade mid-flight,
  // network blip) left the sheet stuck showing that forever, since the
  // catch block only toasted and never touched sheet state.
  it('openInvite closes the sheet instead of spinning forever when createInvite throws', async () => {
    api.teams.createInvite = vi.fn().mockRejectedValue(new Error('boom'));
    const { result } = renderActions();
    await act(async () => {
      await result.current.openInvite();
    });
    expect(stateRef.sheet).toBeNull();
    expect(toastMsg).toHaveBeenCalled();
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
    expect(toastMsg).toHaveBeenCalledWith('Kopieren fehlgeschlagen', undefined, 'error');
  });

  // Regression test: the sheet update used to splice `copied: true` onto
  // whatever `s.sheet` happened to be at resolution time with no team/type
  // check. If the user closed the invite sheet (activeTeamId/sheet both
  // reset) while the clipboard write was still pending, `{ ...null, copied:
  // true }` produced a typeless `{ copied: true }` object that still passed
  // SheetHost's truthy check -- an empty untitled modal popping up out of
  // nowhere over whatever screen the user has since navigated to.
  it('does not resurrect a closed sheet after a slow clipboard write resolves', async () => {
    stateRef = makeState({ sheet: { type: 'invite', invite: { link: 'https://example.com/abc' } } as never });
    let resolveWrite!: () => void;
    Object.assign(navigator, {
      clipboard: { writeText: vi.fn(() => new Promise<void>((resolve) => (resolveWrite = resolve))) },
    });
    const { result } = renderActions();

    let copyPromise!: Promise<void>;
    act(() => {
      copyPromise = result.current.copyInvite();
    });

    // User closes the sheet (e.g. navigates away) before the write resolves.
    stateRef = { ...stateRef, sheet: null };

    await act(async () => {
      resolveWrite();
      await copyPromise;
    });

    expect(stateRef.sheet).toBeNull();
  });

  it('uploadMyPhoto updates user photo and shows toast', async () => {
    const { result } = renderActions();
    await act(async () => {
      await result.current.uploadMyPhoto('data:image/png;base64,photo');
    });
    expect(api.auth.setPhoto).toHaveBeenCalledWith('data:image/png;base64,photo');
    expect(refreshTeams).toHaveBeenCalled();
    expect(invalidateMembers).toHaveBeenCalled();
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
