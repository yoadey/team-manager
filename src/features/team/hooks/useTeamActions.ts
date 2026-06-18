import { useCallback } from 'react';
import type { api as defaultApi } from '@/services/serviceLayer';
import type { Invite, TeamForUser } from '@/types';
import type { AppState } from '@/context/AppContext';
import { validateRequiredText } from '@/utils/validation';

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

export function useTeamActions({ api, S, setState, activeTeam, refreshTeams, refreshMembers, setFormVal, afterLoginLoad, toastMsg }: TeamDeps) {
  const openTeamSwitcher = useCallback(() => setState({ sheet: { type: 'teams' } }), [setState]);

  const openProfile = useCallback(() => {
    setState({ sheet: { type: 'profile' } });
    api.absences.listMine().then((myAbsences) => setState({ myAbsences }));
  }, [api, setState]);

  const openMore = useCallback(() => setState({ sheet: { type: 'more' } }), [setState]);

  const openTeamSettings = useCallback(() => {
    const t = activeTeam()!;
    setState({ sheet: { type: 'teamSettings' }, form: { name: t.name, description: t.description || '', icon: t.icon, logo: t.logo || null, photo: t.photo, reasonRoles: (t.reasonVisibilityRoles || []).slice() } });
  }, [activeTeam, setState]);

  const saveTeamPhoto = useCallback(async (dataUrl: string) => {
    await api.teams.updateSettings(S().activeTeamId!, { photo: dataUrl });
    await refreshTeams();
    setFormVal({ photo: dataUrl });
    toastMsg('Gruppenbild aktualisiert');
  }, [api, S, refreshTeams, setFormVal, toastMsg]);

  const saveTeamLogo = useCallback(async (dataUrl: string) => {
    await api.teams.updateSettings(S().activeTeamId!, { logo: dataUrl });
    await refreshTeams();
    setFormVal({ logo: dataUrl });
    toastMsg('Logo aktualisiert');
  }, [api, S, refreshTeams, setFormVal, toastMsg]);

  const setTeamIcon = useCallback((em: string) => {
    setFormVal({ icon: em, logo: null });
    api.teams.updateSettings(S().activeTeamId!, { icon: em, logo: null }).then(() => refreshTeams());
  }, [api, S, setFormVal, refreshTeams]);

  const toggleReasonRole = useCallback((roleId: string) => setState((s) => {
    const cur = s.form.reasonRoles || [];
    const next = cur.includes(roleId) ? cur.filter((x: string) => x !== roleId) : cur.concat(roleId);
    return { form: { ...s.form, reasonRoles: next } };
  }), [setState]);

  const saveTeamSettings = useCallback(async () => {
    const f = S().form;
    if (!f.name || !f.name.trim()) { toastMsg('Bitte Team-Namen angeben'); return; }
    setState({ busy: 'save' });
    await api.teams.updateSettings(S().activeTeamId!, { name: f.name.trim(), description: f.description || '', reasonVisibilityRoles: f.reasonRoles || [] });
    await refreshTeams();
    setState({ busy: null });
    toastMsg('Team-Einstellungen gespeichert');
  }, [api, S, setState, refreshTeams, toastMsg]);

  const openCreateTeam = useCallback(() => setState({ sheet: { type: 'createTeam' }, form: { name: '', icon: '⭐', photo: null } }), [setState]);

  const createTeam = useCallback(async () => {
    const f = S().form;
    const name = validateRequiredText(f.name, 'Team-Name fehlt.');
    if (!name.ok) { toastMsg(name.message!); return; }
    setState({ busy: 'save' });
    const team = await api.teams.create({ name: name.value!, icon: f.icon, iconBg: '#1A1A1A', iconFg: '#F5C518', photo: f.photo });
    await refreshTeams();
    setState({ busy: null, sheet: null, activeTeamId: team.id, route: 'home', eventScope: 'upcoming' });
    await afterLoginLoad(team.id);
    toastMsg('Team angelegt – du bist Admin');
  }, [api, S, setState, refreshTeams, afterLoginLoad, toastMsg]);

  const openInvite = useCallback(async () => {
    setState({ sheet: { type: 'invite', invite: null } });
    const invite = await api.teams.createInvite(S().activeTeamId!);
    setState((s) => (s.sheet && s.sheet.type === 'invite') ? { sheet: { ...s.sheet, invite } } : {});
  }, [api, S, setState]);

  const copyInvite = useCallback(() => {
    const inv: Invite = S().sheet!.invite;
    if (!inv) return;
    try { navigator.clipboard.writeText(inv.link); } catch { /* ignore */ }
    setState((s) => ({ sheet: { ...s.sheet!, copied: true } }));
    toastMsg('Link kopiert');
  }, [S, setState, toastMsg]);

  const uploadMyPhoto = useCallback(async (dataUrl: string) => {
    await api.auth.setPhoto(dataUrl);
    const user = await api.auth.currentUser();
    await Promise.all([refreshTeams(), refreshMembers()]);
    setState({ user });
    toastMsg('Profilfoto aktualisiert');
  }, [api, refreshTeams, refreshMembers, setState, toastMsg]);

  return { openTeamSwitcher, openProfile, openMore, openTeamSettings, saveTeamPhoto, saveTeamLogo, setTeamIcon, toggleReasonRole, saveTeamSettings, openCreateTeam, createTeam, openInvite, copyInvite, uploadMyPhoto };
}
