import { useCallback } from 'react';
import { useQueryClient } from '@tanstack/react-query';
import type { api as defaultApi } from '@/services';
import type { AppState, ConfirmConfig } from '@/context/AppContext';
import { queryKeys } from '@/query/keys';
import { reportActionError } from '@/utils/errors';
import { t } from '@/i18n';

type SetState = (patch: Partial<AppState> | ((s: AppState) => Partial<AppState>)) => void;

type CalendarShareDeps = {
  api: typeof defaultApi;
  S: () => AppState;
  setState: SetState;
  askConfirm: (cfg: ConfirmConfig) => void;
  toastMsg: (m: string, action?: { label: string; fn: () => void }, kind?: 'success' | 'error') => void;
  logout: () => void;
};

export function useCalendarShareActions({ api, S, setState, askConfirm, toastMsg, logout }: CalendarShareDeps) {
  const qc = useQueryClient();

  const openCalendarShares = useCallback(() => setState({ sheet: { type: 'calendarShares' } }), [setState]);
  const openSharedCalendars = useCallback(() => setState({ sheet: { type: 'sharedCalendars' } }), [setState]);

  const grantCalendarShare = useCallback(
    async (viewerTeamId: string) => {
      const teamId = S().activeTeamId!;
      try {
        await api.calendarShares.create(teamId, viewerTeamId);
        await qc.invalidateQueries({ queryKey: queryKeys.calendarShares(teamId) });
        toastMsg(t('team.toastCalendarShareCreated'));
      } catch (err) {
        reportActionError({ setState, toastMsg, onAuthError: logout }, err, 'error.save');
        throw err;
      }
    },
    [api, S, qc, setState, toastMsg, logout],
  );

  const revokeCalendarShare = useCallback(
    (viewerTeamId: string, viewerTeamName: string) =>
      askConfirm({
        title: t('team.revokeCalendarShareConfirmTitle'),
        message: t('team.revokeCalendarShareConfirmMsg', { name: viewerTeamName }),
        confirmLabel: t('common.delete'),
        danger: true,
        onConfirm: async () => {
          const teamId = S().activeTeamId!;
          try {
            await api.calendarShares.remove(teamId, viewerTeamId);
            await qc.invalidateQueries({ queryKey: queryKeys.calendarShares(teamId) });
            toastMsg(t('team.toastCalendarShareRevoked'));
          } catch (err) {
            reportActionError({ setState, toastMsg, onAuthError: logout }, err, 'error.delete');
          }
        },
      }),
    [api, S, qc, askConfirm, setState, toastMsg, logout],
  );

  return { openCalendarShares, openSharedCalendars, grantCalendarShare, revokeCalendarShare };
}
