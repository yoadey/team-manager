import { useCallback } from 'react';
import type { api as defaultApi } from '@/services/serviceLayer';
import type { ModuleKey, PermLevel, TeamForUser } from '@/types';
import type { AppState } from '@/context/AppContext';
import type { RoleFormValues } from '../types';
import { formValues } from '@/utils/forms';
import { reportActionError } from '@/utils/errors';
import { validateRequiredText } from '@/utils/validation';
import { t } from '@/i18n';

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

  const openCreateRole = useCallback(() => {
    const form: RoleFormValues = {
      name: '',
      perms: { events: 'read', members: 'read', finances: 'none', news: 'read', polls: 'read', settings: 'none' },
    };
    setState((st) => ({ sheet: { type: 'roleForm', back: st.sheet }, form }));
  }, [setState]);

  const setRolePerm = useCallback(
    (module: ModuleKey, level: PermLevel) =>
      setState((s) => {
        const perms = formValues<RoleFormValues>(s).perms;
        return { form: { ...s.form, perms: { ...perms, [module]: level } } };
      }),
    [setState],
  );

  const saveRole = useCallback(async () => {
    const f = S().form as RoleFormValues;
    const nameResult = validateRequiredText(f.name, t('team.roleNameRequired'));
    if (!nameResult.ok) {
      toastMsg(nameResult.message!);
      return;
    }
    setState({ busy: 'save' });
    try {
      await api.roles.create(S().activeTeamId!, { name: nameResult.value!, permissions: f.perms });
      await refreshRoles();
      setState({ busy: null, sheet: { type: 'roles' } });
      toastMsg(t('team.toastRoleCreated'));
    } catch (err) {
      reportActionError({ setState, toastMsg }, err, 'error.save');
    }
  }, [api, S, setState, refreshRoles, toastMsg]);

  const toggleMyRole = useCallback(
    async (roleId: string) => {
      const team = activeTeam()!;
      const cur = team.myRoles.map((r) => r.id);
      const next = cur.includes(roleId) ? cur.filter((x) => x !== roleId) : cur.concat(roleId);
      if (!next.length) {
        toastMsg(t('team.roleAtLeastOne'));
        return;
      }
      try {
        await api.members.setRoles(team.membershipId, next, team.id);
        await refreshTeams();
        toastMsg(t('team.toastRolesSaved'));
      } catch (err) {
        reportActionError({ setState, toastMsg }, err);
      }
    },
    [api, activeTeam, refreshTeams, setState, toastMsg],
  );

  return { openRoles, openCreateRole, setRolePerm, saveRole, toggleMyRole };
}
