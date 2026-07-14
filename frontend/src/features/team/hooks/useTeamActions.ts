import { useCallback, useRef } from 'react';
import type { api as defaultApi } from '@/services';
import type { Invite, TeamForUser } from '@/types';
import type { AppState } from '@/context/AppContext';
import type { CreateTeamFormValues, TeamSettingsFormValues } from '../types';
import { formValues, clearBusyIfOwned } from '@/utils/forms';
import { validateRequiredText } from '@/utils/validation';
import { reportActionError } from '@/utils/errors';
import { t } from '@/i18n';

type SetState = (patch: Partial<AppState> | ((s: AppState) => Partial<AppState>)) => void;

type TeamDeps = {
  api: typeof defaultApi;
  S: () => AppState;
  setState: SetState;
  activeTeam: () => TeamForUser | null;
  refreshTeams: () => Promise<void>;
  refreshMembers: () => Promise<void>;
  setFormVal: (patch: Record<string, unknown>) => void;
  afterLoginLoad: (teamId: string) => Promise<void>;
  toastMsg: (m: string, action?: { label: string; fn: () => void }, kind?: 'success' | 'error') => void;
  logout: () => void;
};

export function useTeamActions({
  api,
  S,
  setState,
  activeTeam,
  refreshTeams,
  refreshMembers,
  setFormVal,
  afterLoginLoad,
  toastMsg,
  logout,
}: TeamDeps) {
  const openTeamSwitcher = useCallback(() => setState({ sheet: { type: 'teams' } }), [setState]);

  const openProfile = useCallback(() => {
    setState({ sheet: { type: 'profile' } });
    const teamId = S().activeTeamId!;
    api.absences
      .listMine(teamId)
      .then((myAbsences) => setState((s) => (s.activeTeamId === teamId ? { myAbsences } : {})))
      .catch((err) => reportActionError({ setState, toastMsg, onAuthError: logout }, err, 'error.load'));
  }, [api, S, setState, toastMsg, logout]);

  const openMore = useCallback(() => setState({ sheet: { type: 'more' } }), [setState]);

  const openTeamSettings = useCallback(() => {
    const team = activeTeam()!;
    const form: TeamSettingsFormValues = {
      name: team.name,
      description: team.description || '',
      icon: team.icon,
      logo: team.logo || null,
      photo: team.photo,
      reasonRoles: (team.reasonVisibilityRoles || []).slice(),
    };
    setState({ sheet: { type: 'teamSettings' }, form });
  }, [activeTeam, setState]);

  const photoLogoInFlight = useRef(new Set<string>());

  // setFormVal writes into the single shared, untyped form buffer regardless
  // of which sheet is currently open, so an upload that resolves after the
  // user has since switched teams and/or opened a different sheet must not
  // apply its result -- otherwise it silently overwrites whatever the OTHER
  // sheet's form fields (e.g. CreateTeamSheet's own photo preview) currently
  // hold with stale data from a completely different team.
  const stillOnTeamSettingsFor = useCallback(
    (teamId: string) => S().activeTeamId === teamId && S().sheet?.type === 'teamSettings',
    [S],
  );

  // teamId is passed in by the caller rather than read from S() here --
  // saveTeamPhoto/saveTeamLogo are invoked from a FileReader.onload callback
  // (see TeamSheets.tsx), so by the time this function runs, an arbitrary
  // amount of time (and possibly a team switch) may have passed since the
  // user picked the file. Reading S().activeTeamId! at THIS point would
  // capture whatever team is active when the read finishes, not the team
  // the upload was actually for -- the caller must snapshot it synchronously
  // when the file was selected, before the async read even starts.
  const saveTeamPhoto = useCallback(
    async (dataUrl: string, teamId: string) => {
      const key = 'photo';
      if (photoLogoInFlight.current.has(key)) return;
      photoLogoInFlight.current.add(key);
      try {
        await api.teams.updateSettings(teamId, { photo: dataUrl });
        await refreshTeams();
        if (stillOnTeamSettingsFor(teamId)) setFormVal({ photo: dataUrl });
        toastMsg(t('team.toastPhotoSaved'));
      } catch (err) {
        reportActionError({ setState, toastMsg, onAuthError: logout }, err, 'error.save');
      } finally {
        photoLogoInFlight.current.delete(key);
      }
    },
    [api, refreshTeams, setFormVal, stillOnTeamSettingsFor, setState, toastMsg, logout],
  );

  const removeTeamPhoto = useCallback(async () => {
    const key = 'photo';
    if (photoLogoInFlight.current.has(key)) return;
    photoLogoInFlight.current.add(key);
    const teamId = S().activeTeamId!;
    try {
      await api.teams.updateSettings(teamId, { photo: null });
      await refreshTeams();
      if (stillOnTeamSettingsFor(teamId)) setFormVal({ photo: null });
      toastMsg(t('team.toastPhotoRemoved'));
    } catch (err) {
      reportActionError({ setState, toastMsg, onAuthError: logout }, err, 'error.save');
    } finally {
      photoLogoInFlight.current.delete(key);
    }
  }, [api, S, refreshTeams, setFormVal, stillOnTeamSettingsFor, setState, toastMsg, logout]);

  const saveTeamLogo = useCallback(
    async (dataUrl: string, teamId: string) => {
      const key = 'logo';
      if (photoLogoInFlight.current.has(key)) return;
      photoLogoInFlight.current.add(key);
      try {
        await api.teams.updateSettings(teamId, { logo: dataUrl });
        await refreshTeams();
        if (stillOnTeamSettingsFor(teamId)) setFormVal({ logo: dataUrl });
        toastMsg(t('team.toastLogoSaved'));
      } catch (err) {
        reportActionError({ setState, toastMsg, onAuthError: logout }, err, 'error.save');
      } finally {
        photoLogoInFlight.current.delete(key);
      }
    },
    [api, refreshTeams, setFormVal, stillOnTeamSettingsFor, setState, toastMsg, logout],
  );

  const setTeamIcon = useCallback(
    async (em: string) => {
      // Was optimistic (setFormVal before the API call, no rollback on
      // failure) -- unlike every other photo/logo mutation in this file,
      // which all await the API call first and only touch form state on
      // success. A failed updateSettings (permission revoked mid-session,
      // network blip) left the settings sheet showing the new icon as
      // selected even though the backend still had the old one, with only a
      // transient toast as the (easy-to-miss) signal it didn't save.
      const key = 'logo';
      if (photoLogoInFlight.current.has(key)) return;
      photoLogoInFlight.current.add(key);
      const teamId = S().activeTeamId!;
      try {
        await api.teams.updateSettings(teamId, { icon: em, logo: null });
        await refreshTeams();
        if (stillOnTeamSettingsFor(teamId)) setFormVal({ icon: em, logo: null });
      } catch (err) {
        reportActionError({ setState, toastMsg, onAuthError: logout }, err, 'error.save');
      } finally {
        photoLogoInFlight.current.delete(key);
      }
    },
    [api, S, refreshTeams, setFormVal, stillOnTeamSettingsFor, setState, toastMsg, logout],
  );

  const toggleReasonRole = useCallback(
    (roleId: string) =>
      setState((s) => {
        const cur = formValues<TeamSettingsFormValues>(s).reasonRoles ?? [];
        const next = cur.includes(roleId) ? cur.filter((x) => x !== roleId) : cur.concat(roleId);
        return { form: { ...s.form, reasonRoles: next } };
      }),
    [setState],
  );

  const saveTeamSettings = useCallback(async () => {
    const f = S().form as TeamSettingsFormValues;
    if (!f.name || !f.name.trim()) {
      toastMsg(t('team.nameRequired'), undefined, 'error');
      return;
    }
    setState({ busy: 'save' });
    try {
      await api.teams.updateSettings(S().activeTeamId!, {
        name: f.name.trim(),
        description: f.description || '',
        reasonVisibilityRoles: f.reasonRoles || [],
      });
      await refreshTeams();
      clearBusyIfOwned(S, setState, 'save');
      toastMsg(t('team.toastSettingsSaved'));
    } catch (err) {
      reportActionError({ setState, toastMsg, onAuthError: logout, S, busyOwner: 'save' }, err, 'error.save');
    }
  }, [api, S, setState, refreshTeams, toastMsg, logout]);

  const openCreateTeam = useCallback(() => {
    const form: CreateTeamFormValues = { name: '', icon: '⭐', photo: null };
    setState({ sheet: { type: 'createTeam' }, form });
  }, [setState]);

  const createTeam = useCallback(async () => {
    const f = S().form as CreateTeamFormValues;
    const name = validateRequiredText(f.name, t('team.nameRequired'));
    if (!name.ok) {
      toastMsg(name.message!, undefined, 'error');
      return;
    }
    const sh = S().sheet;
    setState({ busy: 'save' });
    try {
      const team = await api.teams.create({
        name: name.value!,
        icon: f.icon,
        iconBg: '#1A1A1A',
        iconFg: '#F5C518',
        photo: f.photo,
      });
      await refreshTeams();
      clearBusyIfOwned(S, setState, 'save');
      // Don't navigate the user into the new team, or clobber whatever
      // sheet they've since opened, if they closed the create-team sheet
      // (or opened something else) while this request was in flight -- the
      // new team still exists and is now in their team list either way.
      if (S().sheet === sh) {
        setState({ sheet: null, activeTeamId: team.id, route: 'home', eventScope: 'upcoming', phase: 'app' });
        await afterLoginLoad(team.id);
      }
      toastMsg(t('team.toastTeamCreated'));
    } catch (err) {
      reportActionError({ setState, toastMsg, onAuthError: logout, S, busyOwner: 'save' }, err, 'error.save');
    }
  }, [api, S, setState, refreshTeams, afterLoginLoad, toastMsg, logout]);

  const openInvite = useCallback(async () => {
    const teamId = S().activeTeamId!;
    // Captured by reference, not just sheet.type/activeTeamId -- openInvite
    // has no busy flag, so it can be invoked twice in a row for the same
    // team (open, close, reopen before the first request resolves). A
    // type+team check alone can't tell those two invite sheets apart: the
    // first request's success/failure handler would silently overwrite or
    // close the SECOND (still in-flight, possibly about-to-succeed) sheet.
    // Reference equality also subsumes the original team-switch guard, since
    // selectTeam clears sheet to null (!== sh) on any team change.
    const sh = { type: 'invite' as const, invite: null };
    setState({ sheet: sh });
    try {
      const invite = await api.teams.createInvite(teamId);
      setState((s) => (s.sheet === sh ? { sheet: { ...sh, invite } } : {}));
    } catch (err) {
      reportActionError({ setState, toastMsg, onAuthError: logout }, err);
      // InviteSheet shows an eternal "wird generiert..." placeholder while
      // sheet.invite is null -- there's no error state to fall back to, so
      // a failed fetch (permission downgrade mid-flight, network blip) left
      // it stuck forever. The toast above already explains what went wrong;
      // just close the sheet instead, mirroring reloadDetail's same fix.
      setState((s) => (s.sheet === sh ? { sheet: null } : {}));
    }
  }, [api, S, setState, toastMsg, logout]);

  const copyInvite = useCallback(async () => {
    const teamId = S().activeTeamId;
    const inv: Invite | null | undefined = S().sheet?.invite;
    if (!inv) return;
    try {
      await navigator.clipboard.writeText(inv.link);
    } catch {
      toastMsg(t('error.copy'), undefined, 'error');
      return;
    }
    // Must check the team and sheet type too, not just splice onto whatever
    // sheet is current at resolution time -- if the user closed the sheet
    // (or switched teams, which clears it) while the clipboard write was
    // pending, `{ ...s.sheet!, copied: true }` on a null sheet produces a
    // typeless `{ copied: true }` object that still passes SheetHost's
    // truthy check, popping up an empty untitled modal out of nowhere.
    setState((s) =>
      s.activeTeamId === teamId && s.sheet?.type === 'invite' ? { sheet: { ...s.sheet, copied: true } } : {},
    );
    toastMsg(t('team.toastLinkCopied'));
  }, [S, setState, toastMsg]);

  const uploadMyPhoto = useCallback(
    async (dataUrl: string) => {
      const key = 'myPhoto';
      if (photoLogoInFlight.current.has(key)) return;
      photoLogoInFlight.current.add(key);
      try {
        await api.auth.setPhoto(dataUrl);
        const user = await api.auth.currentUser();
        await Promise.all([refreshTeams(), refreshMembers()]);
        setState({ user });
        toastMsg(t('team.toastMyPhotoSaved'));
      } catch (err) {
        reportActionError({ setState, toastMsg, onAuthError: logout }, err, 'error.save');
      } finally {
        photoLogoInFlight.current.delete(key);
      }
    },
    [api, refreshTeams, refreshMembers, setState, toastMsg, logout],
  );

  return {
    openTeamSwitcher,
    openProfile,
    openMore,
    openTeamSettings,
    saveTeamPhoto,
    removeTeamPhoto,
    saveTeamLogo,
    setTeamIcon,
    toggleReasonRole,
    saveTeamSettings,
    openCreateTeam,
    createTeam,
    openInvite,
    copyInvite,
    uploadMyPhoto,
  };
}
