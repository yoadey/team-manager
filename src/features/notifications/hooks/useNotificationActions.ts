import { useCallback } from 'react';
import type { api as defaultApi } from '@/services/serviceLayer';
import type { AppState } from '@/context/AppContext';

type SetState = (patch: Partial<AppState> | ((s: AppState) => Partial<AppState>)) => void;

type NotifDeps = {
  api: typeof defaultApi;
  S: () => AppState;
  setState: SetState;
  loadNotifications: () => Promise<void>;
  toastMsg: (m: string) => void;
};

export function useNotificationActions({ api, S, setState, loadNotifications, toastMsg }: NotifDeps) {
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
      } catch {
        toastMsg('Benachrichtigungen konnten nicht als gelesen markiert werden.');
      }
    })();
  }, [api, S, setState, loadNotifications, toastMsg]);

  const setNotifFilter = useCallback((f: AppState['notifFilter']) => setState({ notifFilter: f }), [setState]);

  return { openNotifications, setNotifFilter };
}
