import { useCallback, useRef } from 'react';
import type { api as defaultApi } from '@/services';
import type { Role, TeamForUser } from '@/types';
import type { AppState, ConfirmConfig } from '@/context/AppContext';
import type { RoleFormValues } from '../components/roleFormSchema';
import { reportActionError } from '@/utils/errors';
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
  toastMsg: (m: string, action?: { label: string; fn: () => void }, kind?: 'success' | 'error') => void;
  logout: () => void;
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
  logout,
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
      setState((st) => ({
        sheet: { type: 'roleForm', mode: role ? 'edit' : 'create', back: st.sheet, formInitial: form },
      }));
    },
    [setState],
  );

  const saveRole = useCallback(
    async (f: RoleFormValues) => {
      const teamId = S().activeTeamId!;
      const sh = S().sheet;
      try {
        if (f.id) {
          await api.roles.update(f.id, { name: f.name.trim(), permissions: f.perms }, teamId);
          await refreshRoles();
          // Don't navigate to the roles sheet for a different team than the
          // one the user has since switched to, or clobber a different sheet
          // the user has since opened while this save was in flight.
          if (S().activeTeamId === teamId && S().sheet === sh) setState({ sheet: { type: 'roles' } });
          toastMsg(t('team.toastRoleUpdated'));
        } else {
          await api.roles.create(teamId, { name: f.name.trim(), permissions: f.perms });
          await refreshRoles();
          if (S().activeTeamId === teamId && S().sheet === sh) setState({ sheet: { type: 'roles' } });
          toastMsg(t('team.toastRoleCreated'));
        }
      } catch (err) {
        reportActionError({ setState, toastMsg, onAuthError: logout }, err, 'error.save');
        throw err;
      }
    },
    [api, S, setState, refreshRoles, toastMsg, logout],
  );

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
            reportActionError({ setState, toastMsg, onAuthError: logout }, err, 'error.delete');
          }
        },
      }),
    [api, S, askConfirm, refreshRoles, setState, toastMsg, logout],
  );

  // Chains toggleMyRole calls so each one reads activeTeam().myRoles only
  // once the previous toggle's refreshTeams() has applied -- reading it at
  // click time instead let two rapid toggles of different roles both start
  // from the same stale role list, so the second request's PUT silently
  // overwrote the first's change instead of building on it.
  const toggleChain = useRef<Promise<void>>(Promise.resolve());

  const toggleMyRole = useCallback(
    (roleId: string) => {
      const run = async () => {
        const team = activeTeam();
        if (!team) return;
        const cur = team.myRoles.map((r) => r.id);
        const next = cur.includes(roleId) ? cur.filter((x) => x !== roleId) : cur.concat(roleId);
        if (!next.length) {
          toastMsg(t('team.roleAtLeastOne'), undefined, 'error');
          return;
        }
        try {
          await api.members.setRoles(team.membershipId, next, team.id);
          await refreshTeams();
          toastMsg(t('team.toastRolesSaved'));
        } catch (err) {
          reportActionError({ setState, toastMsg, onAuthError: logout }, err);
        }
      };
      const step = toggleChain.current.then(run);
      toggleChain.current = step;
      return step;
    },
    [api, activeTeam, refreshTeams, setState, toastMsg, logout],
  );

  return { openRoles, openRoleForm, saveRole, removeRole, toggleMyRole };
}
