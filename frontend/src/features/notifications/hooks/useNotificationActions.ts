import { useCallback } from 'react';
import type { api as defaultApi } from '@/services';
import type { AppState } from '@/context/AppContext';
import { reportActionError } from '@/utils/errors';

type SetState = (patch: Partial<AppState> | ((s: AppState) => Partial<AppState>)) => void;

type NotifDeps = {
  api: typeof defaultApi;
  S: () => AppState;
  setState: SetState;
  loadNotifications: () => Promise<void>;
  toastMsg: (m: string, action?: { label: string; fn: () => void }, kind?: 'success' | 'error') => void;
  logout: () => void;
};

export function useNotificationActions({ api, S, setState, loadNotifications, toastMsg, logout }: NotifDeps) {
  const openNotifications = useCallback(() => {
    const teamId = S().activeTeamId;
    if (!teamId) return;
    setState({ sheet: { type: 'notifications' }, notifFilter: 'all' });
    void (async () => {
      try {
        if (!S().notifications) await loadNotifications();
        await api.notifications.markSeen(teamId);
        setState((s) => {
          if (s.activeTeamId !== teamId) return {};
          return {
            notifications: s.notifications ? s.notifications.map((n) => ({ ...n, unread: false })) : s.notifications,
            notifUnread: 0,
          };
        });
      } catch (err) {
        reportActionError({ setState, toastMsg, onAuthError: logout }, err, 'notifications.markReadError');
      }
    })();
  }, [api, S, setState, loadNotifications, toastMsg, logout]);

  const setNotifFilter = useCallback((f: AppState['notifFilter']) => setState({ notifFilter: f }), [setState]);

  return { openNotifications, setNotifFilter };
}
