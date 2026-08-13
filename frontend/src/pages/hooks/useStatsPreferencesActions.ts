import { useCallback } from 'react';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import type { api as defaultApi } from '@/services';
import { queryKeys } from '@/query/keys';
import { t } from '@/i18n';
import { useStatsPreferencesQuery, useStatsPresetsQuery } from './useStatsPreferencesQuery';
import type { DateRange, StatsPreferences, StatsPreset } from '@/types';

type ToastFn = (m: string, action?: { label: string; fn: () => void }, kind?: 'success' | 'error') => void;

/**
 * Drives the Stats page's persisted range + named presets: reads the
 * caller's last-saved selection and preset list for the active team, and
 * exposes mutations to save the current selection (fired on every range
 * change, mirroring usePushPreferencesActions' whole-object-PUT pattern)
 * and to create/rename/delete a preset.
 */
export function useStatsPreferencesActions(api: typeof defaultApi, teamId: string | null, toastMsg: ToastFn) {
  const preferencesQuery = useStatsPreferencesQuery(api, teamId);
  const presetsQuery = useStatsPresetsQuery(api, teamId);
  const qc = useQueryClient();

  const saveSelectionMutation = useMutation({
    mutationFn: ({ range, presetId }: { range: DateRange; presetId: string | null }) =>
      api.statsPrefs.setPreferences(teamId!, range, presetId),
    onSuccess: (_void, { range, presetId }) => {
      qc.setQueryData<StatsPreferences>(queryKeys.statsPreferences(teamId ?? ''), { range, presetId });
    },
  });

  const createPresetMutation = useMutation({
    mutationFn: ({ name, range }: { name: string; range: DateRange }) => api.statsPrefs.createPreset(teamId!, name, range),
    onSuccess: (preset) => {
      qc.setQueryData<StatsPreset[]>(queryKeys.statsPresets(teamId ?? ''), (prev) => [preset, ...(prev ?? [])]);
    },
    onError: () => {
      toastMsg(t('stats.presetSaveFailed'), undefined, 'error');
    },
  });

  const updatePresetMutation = useMutation({
    mutationFn: ({ id, patch }: { id: string; patch: { name?: string; range?: DateRange } }) =>
      api.statsPrefs.updatePreset(teamId!, id, patch),
    onSuccess: (preset) => {
      qc.setQueryData<StatsPreset[]>(queryKeys.statsPresets(teamId ?? ''), (prev) =>
        (prev ?? []).map((p) => (p.id === preset.id ? preset : p)),
      );
    },
    onError: () => {
      toastMsg(t('stats.presetSaveFailed'), undefined, 'error');
    },
  });

  const deletePresetMutation = useMutation({
    mutationFn: (id: string) => api.statsPrefs.deletePreset(teamId!, id),
    onSuccess: (_void, id) => {
      qc.setQueryData<StatsPreset[]>(queryKeys.statsPresets(teamId ?? ''), (prev) => (prev ?? []).filter((p) => p.id !== id));
      // Deleting the active preset degrades the saved selection's presetId to
      // null server-side (ON DELETE SET NULL) -- mirror that locally so a
      // stale presetId doesn't linger in the cache.
      qc.setQueryData<StatsPreferences>(queryKeys.statsPreferences(teamId ?? ''), (prev) =>
        prev && prev.presetId === id ? { ...prev, presetId: null } : prev,
      );
    },
  });

  const saveSelection = useCallback(
    (range: DateRange, presetId: string | null) => {
      if (!range.from || !range.to) return;
      saveSelectionMutation.mutate({ range, presetId });
    },
    [saveSelectionMutation],
  );

  const createPreset = useCallback(
    (name: string, range: DateRange) => createPresetMutation.mutateAsync({ name, range }),
    [createPresetMutation],
  );

  const renamePreset = useCallback(
    (id: string, name: string) => updatePresetMutation.mutateAsync({ id, patch: { name } }),
    [updatePresetMutation],
  );

  const deletePreset = useCallback((id: string) => deletePresetMutation.mutate(id), [deletePresetMutation]);

  return {
    preferences: preferencesQuery.data,
    preferencesLoaded: preferencesQuery.isSuccess,
    presets: presetsQuery.data ?? [],
    saveSelection,
    createPreset,
    creatingPreset: createPresetMutation.isPending,
    renamePreset,
    deletePreset,
  };
}
