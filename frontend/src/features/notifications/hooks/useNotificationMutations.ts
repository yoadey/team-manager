import { useMutation, useQueryClient } from '@tanstack/react-query';
import type { api as defaultApi } from '@/services';
import { queryKeys } from '@/query/keys';
import type { NotificationsResult } from '../types';

/**
 * Marks every notification as read. Writes the result directly into the
 * query cache (rather than invalidating and refetching) to mirror the
 * pre-migration behavior exactly: markSeen doesn't change which
 * notifications exist, only their `unread` flag, so there's nothing a
 * refetch would learn that this optimistic local update doesn't already
 * know.
 */
export function useMarkNotificationsSeenMutation(api: typeof defaultApi, teamId: string | null) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: () => api.notifications.markSeen(teamId!),
    onSuccess: () => {
      qc.setQueryData<NotificationsResult>(queryKeys.notifications(teamId ?? ''), (old) =>
        old ? { items: old.items.map((n) => ({ ...n, unread: false })), unreadCount: 0 } : old,
      );
    },
  });
}
