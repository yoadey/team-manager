import { useCallback } from 'react';
import type { api as defaultApi } from '@/services/serviceLayer';
import type { ModuleKey, PermLevel, Role, TeamForUser } from '@/types';
import type { AppState, ConfirmConfig } from '@/context/AppContext';
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
  askConfirm: (cfg: ConfirmConfig) => void;
  toastMsg: (m: string) => void;
};

export function useRoleActions({
  api,
  S,
  setState,
  activeTeam,
  refreshRoles,
  refreshTeams,
  askConfirm,
  toastMsg,
}: RoleDeps) {
  const openRoles = useCallback(() => setState({ sheet: { type: 'roles' } }), [setState]);

  const openRoleForm = useCallback(
    (role?: Role) => {
      const form: RoleFormValues = role
        ? { id: role.id, name: role.name, perms: role.permissions }
        : {
            name: '',
            perms: { events: 'read', members: 'read', finances: 'none', news: 'read', polls: 'read', settings: 'none' },
          };
      setState((st) => ({ sheet: { type: 'roleForm', mode: role ? 'edit' : 'create', back: st.sheet }, form }));
    },
    [setState],
  );

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
      if (f.id) {
        await api.roles.update(f.id, { name: nameResult.value!, permissions: f.perms }, S().activeTeamId!);
        await refreshRoles();
        setState({ busy: null, sheet: { type: 'roles' } });
        toastMsg(t('team.toastRoleUpdated'));
      } else {
        await api.roles.create(S().activeTeamId!, { name: nameResult.value!, permissions: f.perms });
        await refreshRoles();
        setState({ busy: null, sheet: { type: 'roles' } });
        toastMsg(t('team.toastRoleCreated'));
      }
    } catch (err) {
      reportActionError({ setState, toastMsg }, err, 'error.save');
    }
  }, [api, S, setState, refreshRoles, toastMsg]);

  const removeRole = useCallback(
    (roleId: string) =>
      askConfirm({
        title: t('team.deleteRoleConfirmTitle'),
        message: t('team.deleteRoleConfirmMsg'),
        confirmLabel: t('common.delete'),
        danger: true,
        onConfirm: async () => {
          try {
            await api.roles.remove(roleId, S().activeTeamId!);
            await refreshRoles();
            toastMsg(t('team.toastRoleDeleted'));
          } catch (err) {
            reportActionError({ setState, toastMsg }, err, 'error.delete');
          }
        },
      }),
    [api, S, askConfirm, refreshRoles, setState, toastMsg],
  );

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

  return { openRoles, openRoleForm, setRolePerm, saveRole, removeRole, toggleMyRole };
}
