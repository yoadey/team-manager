import { useCallback } from 'react';
import type { api as defaultApi } from '@/services';
import type { AppState } from '@/context/AppContext';
import { reportActionError } from '@/utils/errors';
import { useMarkNotificationsSeenMutation } from './useNotificationMutations';

type SetState = (patch: Partial<AppState> | ((s: AppState) => Partial<AppState>)) => void;

type NotifDeps = {
  api: typeof defaultApi;
  setState: SetState;
  /** Reactive (render-time) active team id -- the mutation hook keys off this directly
   * rather than through `S()`, since a `useMutation` call must re-run on every render to
   * pick up a team switch instead of only when some later callback fires. */
  teamId: string | null;
  toastMsg: (m: string, action?: { label: string; fn: () => void }, kind?: 'success' | 'error') => void;
  logout: () => void;
};

export function useNotificationActions({ api, setState, teamId, toastMsg, logout }: NotifDeps) {
  const { mutateAsync: markSeenAsync } = useMarkNotificationsSeenMutation(api, teamId);

  const openNotifications = useCallback(() => {
    if (!teamId) return;
    setState({ sheet: { type: 'notifications' }, notifFilter: 'all' });
    // Fire-and-forget: the sheet renders from useNotificationsQuery's own
    // cache (already warm from AppShell's badge, or loading its own spinner
    // if not) regardless of when this resolves.
    markSeenAsync().catch((err) =>
      reportActionError({ setState, toastMsg, onAuthError: logout }, err, 'notifications.markReadError'),
    );
  }, [teamId, setState, markSeenAsync, toastMsg, logout]);

  const setNotifFilter = useCallback((f: AppState['notifFilter']) => setState({ notifFilter: f }), [setState]);

  return { openNotifications, setNotifFilter };
}
