import { useCallback } from 'react';
import type { api as defaultApi } from '@/services/serviceLayer';
import type { Invite, TeamForUser } from '@/types';
import type { AppState } from '@/context/AppContext';
import type { CreateTeamFormValues, TeamSettingsFormValues } from '../types';
import { formValues } from '@/utils/forms';
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
  toastMsg: (m: string) => void;
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
}: TeamDeps) {
  const openTeamSwitcher = useCallback(() => setState({ sheet: { type: 'teams' } }), [setState]);

  const openProfile = useCallback(() => {
    setState({ sheet: { type: 'profile' } });
    api.absences
      .listMine(S().activeTeamId!)
      .then((myAbsences) => setState({ myAbsences }))
      .catch((err) => reportActionError({ setState, toastMsg }, err, 'error.load'));
  }, [api, S, setState, toastMsg]);

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

  const saveTeamPhoto = useCallback(
    async (dataUrl: string) => {
      try {
        await api.teams.updateSettings(S().activeTeamId!, { photo: dataUrl });
        await refreshTeams();
        setFormVal({ photo: dataUrl });
        toastMsg(t('team.toastPhotoSaved'));
      } catch (err) {
        reportActionError({ setState, toastMsg }, err, 'error.save');
      }
    },
    [api, S, refreshTeams, setFormVal, setState, toastMsg],
  );

  const saveTeamLogo = useCallback(
    async (dataUrl: string) => {
      try {
        await api.teams.updateSettings(S().activeTeamId!, { logo: dataUrl });
        await refreshTeams();
        setFormVal({ logo: dataUrl });
        toastMsg(t('team.toastLogoSaved'));
      } catch (err) {
        reportActionError({ setState, toastMsg }, err, 'error.save');
      }
    },
    [api, S, refreshTeams, setFormVal, setState, toastMsg],
  );

  const setTeamIcon = useCallback(
    (em: string) => {
      setFormVal({ icon: em, logo: null });
      api.teams
        .updateSettings(S().activeTeamId!, { icon: em, logo: null })
        .then(() => refreshTeams())
        .catch((err) => reportActionError({ setState, toastMsg }, err, 'error.save'));
    },
    [api, S, setFormVal, refreshTeams, setState, toastMsg],
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
      toastMsg(t('team.nameRequired'));
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
      setState({ busy: null });
      toastMsg(t('team.toastSettingsSaved'));
    } catch (err) {
      reportActionError({ setState, toastMsg }, err, 'error.save');
    }
  }, [api, S, setState, refreshTeams, toastMsg]);

  const openCreateTeam = useCallback(() => {
    const form: CreateTeamFormValues = { name: '', icon: '⭐', photo: null };
    setState({ sheet: { type: 'createTeam' }, form });
  }, [setState]);

  const createTeam = useCallback(async () => {
    const f = S().form as CreateTeamFormValues;
    const name = validateRequiredText(f.name, t('team.nameRequired'));
    if (!name.ok) {
      toastMsg(name.message!);
      return;
    }
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
      setState({ busy: null, sheet: null, activeTeamId: team.id, route: 'home', eventScope: 'upcoming', phase: 'app' });
      await afterLoginLoad(team.id);
      toastMsg(t('team.toastTeamCreated'));
    } catch (err) {
      reportActionError({ setState, toastMsg }, err, 'error.save');
    }
  }, [api, S, setState, refreshTeams, afterLoginLoad, toastMsg]);

  const openInvite = useCallback(async () => {
    setState({ sheet: { type: 'invite', invite: null } });
    try {
      const invite = await api.teams.createInvite(S().activeTeamId!);
      setState((s) => (s.sheet && s.sheet.type === 'invite' ? { sheet: { ...s.sheet, invite } } : {}));
    } catch (err) {
      reportActionError({ setState, toastMsg }, err);
    }
  }, [api, S, setState, toastMsg]);

  const copyInvite = useCallback(() => {
    const inv: Invite | null | undefined = S().sheet?.invite;
    if (!inv) return;
    try {
      navigator.clipboard.writeText(inv.link);
    } catch {
      /* ignore */
    }
    setState((s) => ({ sheet: { ...s.sheet!, copied: true } }));
    toastMsg(t('team.toastLinkCopied'));
  }, [S, setState, toastMsg]);

  const uploadMyPhoto = useCallback(
    async (dataUrl: string) => {
      try {
        await api.auth.setPhoto(dataUrl);
        const user = await api.auth.currentUser();
        await Promise.all([refreshTeams(), refreshMembers()]);
        setState({ user });
        toastMsg(t('team.toastMyPhotoSaved'));
      } catch (err) {
        reportActionError({ setState, toastMsg }, err, 'error.save');
      }
    },
    [api, refreshTeams, refreshMembers, setState, toastMsg],
  );

  return {
    openTeamSwitcher,
    openProfile,
    openMore,
    openTeamSettings,
    saveTeamPhoto,
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
