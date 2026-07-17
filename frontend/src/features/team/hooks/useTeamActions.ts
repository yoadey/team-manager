import { useCallback, useRef } from 'react';
import { useQueryClient } from '@tanstack/react-query';
import type { api as defaultApi } from '@/services';
import type { Invite } from '@/types';
import type { AppState } from '@/context/AppContext';
import type { CreateTeamFormValues } from '../components/createTeamSchema';
import type { TeamSettingsFormValues } from '../components/teamSettingsSchema';
import { queryKeys } from '@/query/keys';
import { reportActionError } from '@/utils/errors';
import { t } from '@/i18n';

type SetState = (patch: Partial<AppState> | ((s: AppState) => Partial<AppState>)) => void;

type TeamDeps = {
  api: typeof defaultApi;
  S: () => AppState;
  setState: SetState;
  refreshTeams: () => Promise<void>;
  /** Invalidates the members query cache (React Query) -- a photo change is
   * visible in the member list too, so it needs to be refetched alongside
   * the team list. */
  invalidateMembers: () => Promise<void>;
  afterLoginLoad: (teamId: string) => Promise<void>;
  toastMsg: (m: string, action?: { label: string; fn: () => void }, kind?: 'success' | 'error') => void;
  logout: () => void;
};

export function useTeamActions({
  api,
  S,
  setState,
  refreshTeams,
  invalidateMembers,
  afterLoginLoad,
  toastMsg,
  logout,
}: TeamDeps) {
  const openTeamSwitcher = useCallback(() => setState({ sheet: { type: 'teams' } }), [setState]);

  const qc = useQueryClient();

  // No component currently reads this -- kept prefetched into the React
  // Query cache (rather than dropped) on the same "profile opened" trigger
  // the pre-migration state.myAbsences fetch used, so the data is a cache-hit
  // away the moment something starts reading it. A team-scoped query key
  // makes the pre-migration activeTeamId re-check after the fetch resolves
  // unnecessary -- a stale prefetch for a team the user has since switched
  // away from just populates a cache entry nothing reads instead of racing to
  // overwrite the new team's state.
  const openProfile = useCallback(() => {
    setState({ sheet: { type: 'profile' } });
    const teamId = S().activeTeamId!;
    void qc.prefetchQuery({ queryKey: queryKeys.myAbsences(teamId), queryFn: () => api.absences.listMine(teamId) });
  }, [api, S, setState, qc]);

  const openMore = useCallback(() => setState({ sheet: { type: 'more' } }), [setState]);

  const openTeamSettings = useCallback(() => {
    setState({ sheet: { type: 'teamSettings' } });
  }, [setState]);

  const photoLogoInFlight = useRef(new Set<string>());

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
        toastMsg(t('team.toastPhotoSaved'));
      } catch (err) {
        reportActionError({ setState, toastMsg, onAuthError: logout }, err, 'error.save');
      } finally {
        photoLogoInFlight.current.delete(key);
      }
    },
    [api, refreshTeams, setState, toastMsg, logout],
  );

  const removeTeamPhoto = useCallback(async () => {
    const key = 'photo';
    if (photoLogoInFlight.current.has(key)) return;
    photoLogoInFlight.current.add(key);
    const teamId = S().activeTeamId!;
    try {
      await api.teams.updateSettings(teamId, { photo: null });
      await refreshTeams();
      toastMsg(t('team.toastPhotoRemoved'));
    } catch (err) {
      reportActionError({ setState, toastMsg, onAuthError: logout }, err, 'error.save');
    } finally {
      photoLogoInFlight.current.delete(key);
    }
  }, [api, S, refreshTeams, setState, toastMsg, logout]);

  const saveTeamLogo = useCallback(
    async (dataUrl: string, teamId: string) => {
      const key = 'logo';
      if (photoLogoInFlight.current.has(key)) return;
      photoLogoInFlight.current.add(key);
      try {
        await api.teams.updateSettings(teamId, { logo: dataUrl });
        await refreshTeams();
        toastMsg(t('team.toastLogoSaved'));
      } catch (err) {
        reportActionError({ setState, toastMsg, onAuthError: logout }, err, 'error.save');
      } finally {
        photoLogoInFlight.current.delete(key);
      }
    },
    [api, refreshTeams, setState, toastMsg, logout],
  );

  const setTeamIcon = useCallback(
    async (em: string) => {
      const key = 'logo';
      if (photoLogoInFlight.current.has(key)) return;
      photoLogoInFlight.current.add(key);
      const teamId = S().activeTeamId!;
      try {
        await api.teams.updateSettings(teamId, { icon: em, logo: null });
        await refreshTeams();
      } catch (err) {
        reportActionError({ setState, toastMsg, onAuthError: logout }, err, 'error.save');
      } finally {
        photoLogoInFlight.current.delete(key);
      }
    },
    [api, S, refreshTeams, setState, toastMsg, logout],
  );

  const saveTeamSettings = useCallback(
    async (f: TeamSettingsFormValues) => {
      try {
        await api.teams.updateSettings(S().activeTeamId!, {
          name: f.name.trim(),
          description: f.description || '',
          reasonVisibilityRoles: f.reasonRoles || [],
        });
        await refreshTeams();
        toastMsg(t('team.toastSettingsSaved'));
      } catch (err) {
        reportActionError({ setState, toastMsg, onAuthError: logout }, err, 'error.save');
        throw err;
      }
    },
    [api, S, setState, refreshTeams, toastMsg, logout],
  );

  const openCreateTeam = useCallback(() => {
    setState({ sheet: { type: 'createTeam' } });
  }, [setState]);

  const createTeam = useCallback(
    async (f: CreateTeamFormValues) => {
      const sh = S().sheet;
      try {
        const team = await api.teams.create({
          name: f.name.trim(),
          icon: f.icon || '⭐',
          iconBg: '#1A1A1A',
          iconFg: '#F5C518',
          photo: f.photo ?? null,
        });
        await refreshTeams();
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
        reportActionError({ setState, toastMsg, onAuthError: logout }, err, 'error.save');
        throw err;
      }
    },
    [api, S, setState, refreshTeams, afterLoginLoad, toastMsg, logout],
  );

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
        await Promise.all([refreshTeams(), invalidateMembers()]);
        setState({ user });
        toastMsg(t('team.toastMyPhotoSaved'));
      } catch (err) {
        reportActionError({ setState, toastMsg, onAuthError: logout }, err, 'error.save');
      } finally {
        photoLogoInFlight.current.delete(key);
      }
    },
    [api, refreshTeams, invalidateMembers, setState, toastMsg, logout],
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
    saveTeamSettings,
    openCreateTeam,
    createTeam,
    openInvite,
    copyInvite,
    uploadMyPhoto,
  };
}
