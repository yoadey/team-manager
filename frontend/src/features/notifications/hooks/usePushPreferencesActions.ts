import { useCallback } from 'react';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import type { api as defaultApi } from '@/services';
import { queryKeys } from '@/query/keys';
import { t } from '@/i18n';
import { usePushPreferencesQuery } from './usePushPreferencesQuery';
import type { PushCategoryPreferences } from '../types';

const DEFAULT_PREFERENCES: PushCategoryPreferences = {
  attendance: true,
  events: true,
  news: true,
  polls: true,
  absence: true,
  eventReminderEnabled: true,
  eventReminderHoursBefore: 6,
};

type ToastFn = (m: string, action?: { label: string; fn: () => void }, kind?: 'success' | 'error') => void;

/**
 * Drives the per-team push-category toggles in Settings' NotificationsPanel: reads the
 * active team's current preferences (defaulting to everything enabled, same
 * as a member who's never customized anything) and exposes a single-category
 * toggle that persists the full preference object -- the API is a whole-object
 * PUT, not a per-field PATCH. Only enabled while `enabled` is true (the caller
 * gates this on the browser already having an active push subscription --
 * per-team preferences are meaningless before that).
 */
export function usePushPreferencesActions(api: typeof defaultApi, teamId: string | null, toastMsg: ToastFn, enabled: boolean) {
  const query = usePushPreferencesQuery(api, teamId, enabled);
  const qc = useQueryClient();

  const mutation = useMutation({
    mutationFn: (next: PushCategoryPreferences) => api.push.setPreferences(teamId!, next),
    onSuccess: (_void, next) => {
      qc.setQueryData<PushCategoryPreferences>(queryKeys.pushPreferences(teamId ?? ''), next);
    },
    onError: () => {
      toastMsg(t('push.preferencesSaveFailed'), undefined, 'error');
    },
  });

  const setCategory = useCallback(
    (category: keyof PushCategoryPreferences, categoryEnabled: boolean) => {
      const current = query.data ?? DEFAULT_PREFERENCES;
      mutation.mutate({ ...current, [category]: categoryEnabled });
    },
    [query.data, mutation],
  );

  const setEventReminderHoursBefore = useCallback(
    (hours: number) => {
      const current = query.data ?? DEFAULT_PREFERENCES;
      mutation.mutate({ ...current, eventReminderHoursBefore: hours });
    },
    [query.data, mutation],
  );

  return {
    prefs: query.data ?? DEFAULT_PREFERENCES,
    isLoading: query.isLoading,
    busy: mutation.isPending,
    setCategory,
    setEventReminderHoursBefore,
  };
}
