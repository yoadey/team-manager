import { useCallback } from 'react';
import type { api as defaultApi } from '@/services';
import type { Role } from '@/types';
import type { AppState, ConfirmConfig } from '@/context/AppContext';
import type { RoleFormValues } from '../components/roleFormSchema';
import { reportActionError } from '@/utils/errors';
import { t } from '@/i18n';

type SetState = (patch: Partial<AppState> | ((s: AppState) => Partial<AppState>)) => void;

type RoleDeps = {
  api: typeof defaultApi;
  S: () => AppState;
  setState: SetState;
  refreshRoles: () => Promise<void>;
  askConfirm: (cfg: ConfirmConfig) => void;
  toastMsg: (m: string, action?: { label: string; fn: () => void }, kind?: 'success' | 'error') => void;
  logout: () => void;
};

export function useRoleActions({ api, S, setState, refreshRoles, askConfirm, toastMsg, logout }: RoleDeps) {
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

  return { openRoles, openRoleForm, saveRole, removeRole };
}
