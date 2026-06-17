import { useCallback } from 'react';
import type { api as defaultApi } from '../../../services/serviceLayer';
import type { ModuleKey, PermLevel, TeamForUser } from '../../../types';
import type { AppState } from '../../../context/AppContext';

type SetState = (patch: Partial<AppState> | ((s: AppState) => Partial<AppState>)) => void;

type RoleDeps = {
  api: typeof defaultApi;
  S: () => AppState;
  setState: SetState;
  activeTeam: () => TeamForUser | null;
  refreshRoles: () => Promise<void>;
  refreshTeams: () => Promise<void>;
  toastMsg: (m: string) => void;
};

export function useRoleActions({ api, S, setState, activeTeam, refreshRoles, refreshTeams, toastMsg }: RoleDeps) {
  const openRoles = useCallback(() => setState({ sheet: { type: 'roles' } }), [setState]);

  const openCreateRole = useCallback(() => setState((st) => ({
    sheet: { type: 'roleForm', back: st.sheet },
    form: { name: '', perms: { events: 'read', members: 'read', finances: 'none', news: 'read', polls: 'read', settings: 'none' } },
  })), [setState]);

  const setRolePerm = useCallback((module: ModuleKey, level: PermLevel) => setState((s) => ({
    form: { ...s.form, perms: { ...s.form.perms, [module]: level } },
  })), [setState]);

  const saveRole = useCallback(async () => {
    const f = S().form;
    if (!f.name) { toastMsg('Bitte Rollennamen angeben'); return; }
    setState({ busy: 'save' });
    await api.roles.create(S().activeTeamId!, { name: f.name, permissions: f.perms });
    await refreshRoles();
    setState({ busy: null, sheet: { type: 'roles' } });
    toastMsg('Rolle angelegt');
  }, [api, S, setState, refreshRoles, toastMsg]);

  const toggleMyRole = useCallback(async (roleId: string) => {
    const team = activeTeam()!;
    const cur = team.myRoles.map((r) => r.id);
    const next = cur.includes(roleId) ? cur.filter((x) => x !== roleId) : cur.concat(roleId);
    if (!next.length) { toastMsg('Mindestens eine Rolle nötig'); return; }
    await api.members.setRoles(team.membershipId, next);
    await refreshTeams();
    toastMsg('Rollen aktualisiert');
  }, [api, activeTeam, refreshTeams, toastMsg]);

  return { openRoles, openCreateRole, setRolePerm, saveRole, toggleMyRole };
}
