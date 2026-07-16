import { describe, it, expect, vi, beforeEach } from 'vitest';
import { renderHook, act, waitFor } from '@testing-library/react';
import { useMemberActions } from './useMemberActions';
import { createQueryWrapper, createTestQueryClient } from '@/test/queryTestUtils';
import { queryKeys } from '@/query/keys';
import type { AppState } from '@/context/AppContext';
import type { Member } from '../types';
import type { QueryClient } from '@tanstack/react-query';

function makeMember(overrides: Partial<Member> = {}): Member {
  return {
    membershipId: 'ms1',
    userId: 'u2',
    name: 'Alice',
    email: 'alice@test.com',
    phone: '',
    birthday: '',
    address: '',
    group: '',
    photo: null,
    roles: [{ id: 'r1', name: 'Mitglied' }] as never,
    avatarColor: '#aaa',
    joinedAt: '2026-01-01',
    primaryRole: null,
    perms: {} as never,
    ...overrides,
  };
}

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
  let refreshTeams: ReturnType<typeof vi.fn>;
  let askConfirm: ReturnType<typeof vi.fn>;
  let logout: ReturnType<typeof vi.fn>;
  let api: ReturnType<typeof makeApi>;
  let stateRef: AppState;
  let client: QueryClient;

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
    askConfirm = vi.fn();
    logout = vi.fn();
    api = makeApi();
    client = createTestQueryClient();
    client.setQueryData(queryKeys.members('team1'), [makeMember()]);
  });

  function renderActions() {
    return renderHook(
      () =>
        useMemberActions({
          api: api as never,
          S: () => stateRef,
          setState: setState as never,
          teamId: stateRef.activeTeamId,
          refreshTeams: refreshTeams as never,
          askConfirm: askConfirm as never,
          toastMsg: toastMsg as never,
          logout: logout as never,
        }),
      { wrapper: createQueryWrapper(client) },
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

  // Regression test: a membershipId not present in the local member list
  // (e.g. a stale bookmarked/back-forward URL for a removed member) used to
  // force-unwrap `m!.userId` when fetching stats, throwing before the API
  // call was even made; the resulting sheet.member: undefined then crashed
  // MemberDetailSheet on render too (fixed separately). The hook must not
  // attempt the stats fetch, nor report a spurious error, for a member that
  // genuinely doesn't exist.
  it('openMemberDetail sets an undefined member and skips the stats fetch for an unknown membershipId', async () => {
    const { result } = renderActions();
    await act(async () => {
      await result.current.openMemberDetail('does-not-exist');
    });
    expect(setState).toHaveBeenCalledWith(
      expect.objectContaining({
        sheet: expect.objectContaining({ type: 'memberDetail', membershipId: 'does-not-exist', member: undefined }),
      }),
    );
    expect(api.stats.attendanceFor).not.toHaveBeenCalled();
    expect(toastMsg).not.toHaveBeenCalled();
  });

  it('openMemberDetail handles API errors gracefully', async () => {
    api.stats.attendanceFor = vi.fn().mockRejectedValue(new Error('Network error'));
    const { result } = renderActions();
    await act(async () => {
      await result.current.openMemberDetail('ms1');
    });
    expect(toastMsg).toHaveBeenCalled();
  });

  // Regression test: the sheet.membershipId check alone only guards against
  // the sheet having switched to a DIFFERENT member while a stats fetch was
  // in flight. It does not catch two openMemberDetail calls for the SAME
  // member racing each other -- e.g. rapid double-clicks on the same row (no
  // busy/disabled state blocks a second click) or mashing browser back/
  // forward between the same member's URL. If the network responds out of
  // request order, the older call's stale stats could overwrite the newer
  // call's fresher data even though the sheet never changed identity.
  it('a slow openMemberDetail cannot overwrite a newer call for the SAME member', async () => {
    let resolveFirst!: (v: { quote: number; counted: number; yes: number }) => void;
    const firstPromise = new Promise<{ quote: number; counted: number; yes: number }>((resolve) => {
      resolveFirst = resolve;
    });
    api.stats.attendanceFor = vi
      .fn()
      .mockReturnValueOnce(firstPromise)
      .mockResolvedValueOnce({ quote: 0.5, counted: 4, yes: 2 });

    const { result } = renderActions();

    const firstOpen = act(async () => {
      await result.current.openMemberDetail('ms1');
    });

    // Second (newer) call for the SAME member resolves immediately.
    await act(async () => {
      await result.current.openMemberDetail('ms1');
    });
    expect((stateRef.sheet as { stats?: { counted: number } } | null)?.stats?.counted).toBe(4);

    // The first call's stale response now arrives -- it must not overwrite
    // the second, fresher call's already-applied result.
    resolveFirst!({ quote: 0.8, counted: 10, yes: 8 });
    await firstOpen;

    expect((stateRef.sheet as { stats?: { counted: number } } | null)?.stats?.counted).toBe(4);
  });

  it('openMemberForm sets memberForm sheet with member data', () => {
    const member = makeMember();
    const { result } = renderActions();
    act(() => {
      result.current.openMemberForm(member);
    });
    expect(setState).toHaveBeenCalled();
    const call = setState.mock.calls[0][0];
    const patch = typeof call === 'function' ? call(stateRef) : call;
    expect(patch.sheet).toMatchObject({ type: 'memberForm', mode: 'edit' });
    expect(patch.form).toMatchObject({ name: 'Alice', email: 'alice@test.com' });
  });

  // Regression test: openMemberForm used to leave a prior sheet's
  // formErrors in place -- editing Member A with an invalid phone number,
  // closing, then editing Member B showed Member B's valid, freshly-loaded
  // phone field with Member A's stale error underneath it.
  it('openMemberForm clears a stale formErrors from a previous sheet', () => {
    stateRef = makeState({ formErrors: { phone: 'Ungültige Telefonnummer.' } });
    const member = makeMember();
    const { result } = renderActions();
    act(() => {
      result.current.openMemberForm(member);
    });
    expect(stateRef.formErrors).toEqual({});
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

  // Regression test: removing a member's last role chip used to silently
  // no-op (fall back to the unchanged list with no feedback), leaving the
  // admin thinking the click did nothing. It must now surface the same
  // "at least one role required" toast that toggleMyRole already shows for
  // the equivalent self-service case.
  it('toggleFormRole does not remove last role and shows a toast', () => {
    stateRef = makeState({ form: { roleIds: ['r1'] } });
    const { result } = renderActions();
    act(() => {
      result.current.toggleFormRole('r1');
    });
    expect(stateRef.form.roleIds).toContain('r1');
    expect(toastMsg).toHaveBeenCalledWith('Mindestens eine Rolle nötig.', undefined, 'error');
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
    expect(toastMsg).toHaveBeenCalledWith('Name fehlt.', undefined, 'error');
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
    expect(toastMsg).toHaveBeenCalledWith('Name fehlt.', undefined, 'error');
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

  it("saveMember calls setRoles when the form roleIds differ from the member's current roles", async () => {
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

  it("saveMember does not call setRoles when the form roleIds match the member's current roles (order-independent)", async () => {
    client.setQueryData(queryKeys.members('team1'), [
      makeMember({
        roles: [
          { id: 'r1', name: 'Mitglied' },
          { id: 'r2', name: 'Trainer' },
        ] as never,
      }),
    ]);
    stateRef = makeState({
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

  // Regression test: savingMember (mutation.isPending) must stay true for the
  // ENTIRE save, including the self-photo upload and the self-profile
  // session refresh (auth.currentUser + refreshTeams) that run after the
  // profile/roles write -- otherwise the Save button re-enables while those
  // are still in flight, letting a second click fire an overlapping save.
  it('savingMember stays true until the self-photo upload and session refresh both finish', async () => {
    let resolveCurrentUser!: (u: { id: string; name: string }) => void;
    api.auth.currentUser = vi.fn(
      () => new Promise<{ id: string; name: string }>((resolve) => (resolveCurrentUser = resolve)),
    );
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

    let savePromise!: Promise<void>;
    act(() => {
      savePromise = result.current.saveMember();
    });
    await waitFor(() => expect(result.current.savingMember).toBe(true));
    await waitFor(() => expect(api.auth.setPhoto).toHaveBeenCalled());
    // The profile patch, role check, and photo upload are done, but the
    // self-session refresh (auth.currentUser) is still pending -- the button
    // must still read busy here, not just up to saveMemberAsync's own resolution.
    expect(result.current.savingMember).toBe(true);

    await act(async () => {
      resolveCurrentUser({ id: 'u1', name: 'Alice' });
      await savePromise;
    });
    expect(refreshTeams).toHaveBeenCalled();
    await waitFor(() => expect(result.current.savingMember).toBe(false));
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

  // Regression test: saveMember used to unconditionally close/reopen the
  // sheet after its await resolved, with no check that the user was still
  // on the team the save was for. A slow members.update() for team1, if the
  // user switched to team2 (and opened some other sheet there) before it
  // resolved, would clobber that unrelated sheet.
  it('saveMember does not touch the sheet if the user switched teams while the save was in flight', async () => {
    let resolveUpdate!: () => void;
    api.members.update = vi.fn(() => new Promise<void>((resolve) => (resolveUpdate = resolve)));
    stateRef = makeState({
      form: { name: 'Alice Updated', email: 'alice@test.com', membershipId: 'ms1', roleIds: ['r1'] },
      sheet: { type: 'memberForm', mode: 'edit', self: false, back: null } as never,
    });
    const { result } = renderActions();

    let savePromise!: Promise<void>;
    act(() => {
      savePromise = result.current.saveMember();
    });
    await waitFor(() => expect(api.members.update).toHaveBeenCalled());

    // User switches teams and opens a different sheet before the save resolves.
    stateRef = { ...stateRef, activeTeamId: 'team2', sheet: { type: 'teams' } as never };

    await act(async () => {
      resolveUpdate();
      await savePromise;
    });

    expect(stateRef.sheet).toEqual({ type: 'teams' });
  });

  // Regression: a slow saveMember for one member used to unconditionally
  // close/reopen the sheet once it resolved, as long as the team hadn't
  // changed -- so closing this member's edit form and opening a DIFFERENT
  // member's form (same team) while the save was still in flight would get
  // silently clobbered by the stale save once it finally resolved, discarding
  // whatever the user was now doing.
  it("saveMember does not touch the sheet if the user opened a different member's form while the save was in flight", async () => {
    let resolveUpdate!: () => void;
    api.members.update = vi.fn(() => new Promise<void>((resolve) => (resolveUpdate = resolve)));
    stateRef = makeState({
      form: { name: 'Alice Updated', email: 'alice@test.com', membershipId: 'ms1', roleIds: ['r1'] },
      sheet: { type: 'memberForm', mode: 'edit', self: false, back: null } as never,
    });
    const { result } = renderActions();

    let savePromise!: Promise<void>;
    act(() => {
      savePromise = result.current.saveMember();
    });
    await waitFor(() => expect(api.members.update).toHaveBeenCalled());

    // User closes Alice's form and opens a different member's form (same
    // team) before Alice's save resolves.
    const otherMembersForm = { type: 'memberForm', mode: 'edit', self: false, back: null } as never;
    stateRef = { ...stateRef, sheet: otherMembersForm };

    await act(async () => {
      resolveUpdate();
      await savePromise;
    });

    expect(stateRef.sheet).toBe(otherMembersForm);
  });

  // Regression test: mirrors useDeleteEventMutation's per-call teamId
  // safeguard. The confirm sheet can still be open (and get confirmed) after
  // the user has switched to a different active team; the delete must still
  // target the team the member/confirm dialog belonged to, not whatever team
  // is active by the time the user actually confirms.
  it('removeMember onConfirm deletes against the team the confirm dialog was opened for, even after a team switch', async () => {
    const { result, rerender } = renderActions();
    act(() => {
      result.current.removeMember('ms1');
    });
    const cfg = askConfirm.mock.calls[0][0];

    // User switches the active team while the confirm dialog is still open.
    stateRef = { ...stateRef, activeTeamId: 'team2' };
    rerender();

    await act(async () => {
      await cfg.onConfirm();
    });
    expect(api.members.remove).toHaveBeenCalledWith('ms1', 'team1');
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

  // Regression: onConfirm used to close the sheet unconditionally (once the
  // team still matched), so a slow removeMember could clobber whatever
  // DIFFERENT sheet the user had since opened while the delete was in
  // flight (e.g. a real askConfirm flow clears the sheet immediately on
  // confirm, then the user opens something else before the delete
  // resolves).
  it('removeMember onConfirm does not touch the sheet if the user opened something else while the delete was in flight', async () => {
    let resolveRemove!: () => void;
    api.members.remove = vi.fn(() => new Promise<void>((resolve) => (resolveRemove = resolve)));
    const { result } = renderActions();
    act(() => {
      result.current.removeMember('ms1');
    });
    const cfg = askConfirm.mock.calls[0][0];

    let confirmPromise!: Promise<void>;
    act(() => {
      confirmPromise = cfg.onConfirm();
    });
    await waitFor(() => expect(api.members.remove).toHaveBeenCalled());

    const somethingElse = { type: 'teams' } as never;
    stateRef = { ...stateRef, sheet: somethingElse };

    await act(async () => {
      resolveRemove();
      await confirmPromise;
    });

    expect(stateRef.sheet).toBe(somethingElse);
  });
});
