import { useCallback } from 'react';
import type { api as defaultApi } from '@/services';
import type { NewsItem } from '../types';
import type { AppState } from '@/context/AppContext';
import { reportActionError } from '@/utils/errors';
import { t } from '@/i18n';
import type { NewsFormValues } from '../components/newsFormSchema';
import { useDeleteNewsMutation, useSaveNewsMutation } from './useNewsMutations';

type SetState = (patch: Partial<AppState> | ((s: AppState) => Partial<AppState>)) => void;

type NewsDeps = {
  api: typeof defaultApi;
  S: () => AppState;
  setState: SetState;
  /** Reactive (render-time) active team id -- the query/mutation hooks key off this directly
   * rather than through `S()`, since a `useQuery`/`useMutation` call must re-run on every
   * render to pick up a team switch instead of only when some later callback fires. */
  teamId: string | null;
  /** A successful save/delete can flip a notification's read-worthy state
   * (e.g. a "new post" reminder), so each mutation also refreshes the
   * notification badge, mirroring the pre-migration loadNews(). */
  loadNotifications: () => Promise<void>;
  askConfirm: (cfg: {
    title: string;
    message: string;
    confirmLabel?: string;
    danger?: boolean;
    onConfirm: () => void | Promise<void>;
  }) => void;
  toastMsg: (m: string, action?: { label: string; fn: () => void }, kind?: 'success' | 'error') => void;
  logout: () => void;
};

export function useNewsActions({
  api,
  S,
  setState,
  teamId,
  loadNotifications,
  askConfirm,
  toastMsg,
  logout,
}: NewsDeps) {
  const { mutateAsync: saveNewsAsync, isPending: savingNews } = useSaveNewsMutation(api, teamId);
  const { mutateAsync: deleteNewsAsync } = useDeleteNewsMutation(api);

  const openNewsForm = useCallback(
    (n?: NewsItem) => {
      const form: NewsFormValues = n
        ? { id: n.id, title: n.title, body: n.body, pinned: n.pinned }
        : { title: '', body: '', pinned: false };
      setState({
        sheet: { type: 'newsForm', mode: n ? 'edit' : 'create', formInitial: form },
      });
    },
    [setState],
  );

  const saveNews = useCallback(
    async (f: NewsFormValues) => {
      const sh = S().sheet;
      const savedTeamId = teamId;
      const editing = !!f.id;
      try {
        await saveNewsAsync({
          id: f.id,
          payload: { title: f.title.trim(), body: f.body.trim(), pinned: !!f.pinned },
        });
        loadNotifications();
        // Don't close a sheet the user has since opened for a different
        // team after switching away mid-request, or one they've since
        // opened for a different entity (same team) while this save was in
        // flight.
        if (S().activeTeamId === savedTeamId && S().sheet === sh) setState({ sheet: null });
        toastMsg(editing ? t('news.toastUpdated') : t('news.toastPublished'));
      } catch (err) {
        reportActionError({ setState, toastMsg, onAuthError: logout }, err, 'error.save');
        throw err;
      }
    },
    [S, setState, saveNewsAsync, loadNotifications, teamId, toastMsg, logout],
  );

  const removeNews = useCallback(
    (id: string) => {
      const deletedTeamId = teamId!;
      askConfirm({
        title: t('news.deleteConfirmTitle'),
        message: t('news.deleteConfirmMsg'),
        confirmLabel: t('common.delete'),
        danger: true,
        onConfirm: async () => {
          try {
            await deleteNewsAsync({ id, teamId: deletedTeamId });
            loadNotifications();
            toastMsg(t('news.toastDeleted'));
          } catch (err) {
            reportActionError({ setState, toastMsg, onAuthError: logout }, err, 'error.delete');
          }
        },
      });
    },
    [askConfirm, deleteNewsAsync, loadNotifications, setState, teamId, toastMsg, logout],
  );

  return { openNewsForm, saveNews, removeNews, savingNews };
}
