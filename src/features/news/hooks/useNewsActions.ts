import { useCallback } from 'react';
import type { api as defaultApi } from '@/services/serviceLayer';
import type { NewsItem, NewsFormValues } from '../types';
import type { AppState } from '@/context/AppContext';
import { reportActionError } from '@/utils/errors';
import { t } from '@/i18n';

type SetState = (patch: Partial<AppState> | ((s: AppState) => Partial<AppState>)) => void;

type NewsDeps = {
  api: typeof defaultApi;
  S: () => AppState;
  setState: SetState;
  loadNews: () => Promise<void>;
  askConfirm: (cfg: {
    title: string;
    message: string;
    confirmLabel?: string;
    danger?: boolean;
    onConfirm: () => void | Promise<void>;
  }) => void;
  toastMsg: (m: string) => void;
};

export function useNewsActions({ api, S, setState, loadNews, askConfirm, toastMsg }: NewsDeps) {
  const openNewsForm = useCallback(
    (n?: NewsItem) => {
      const form: NewsFormValues = n
        ? { id: n.id, title: n.title, body: n.body, pinned: n.pinned }
        : { title: '', body: '', pinned: false };
      setState({
        sheet: { type: 'newsForm', mode: n ? 'edit' : 'create' },
        form,
        formErrors: {},
      });
    },
    [setState],
  );

  const saveNews = useCallback(async () => {
    const f = S().form as NewsFormValues;
    if (!f.title) {
      toastMsg(t('news.titleRequired'));
      return;
    }
    setState({ busy: 'save' });
    try {
      if (f.id) {
        await api.news.update(f.id, { title: f.title, body: f.body, pinned: f.pinned });
        await loadNews();
        setState({ busy: null, sheet: null });
        toastMsg(t('news.toastUpdated'));
      } else {
        await api.news.create(S().activeTeamId!, { title: f.title, body: f.body, pinned: f.pinned });
        await loadNews();
        setState({ busy: null, sheet: null });
        toastMsg(t('news.toastPublished'));
      }
    } catch (err) {
      reportActionError({ setState, toastMsg }, err, 'error.save');
    }
  }, [api, S, setState, loadNews, toastMsg]);

  const removeNews = useCallback(
    (id: string) =>
      askConfirm({
        title: t('news.deleteConfirmTitle'),
        message: t('news.deleteConfirmMsg'),
        confirmLabel: t('common.delete'),
        danger: true,
        onConfirm: async () => {
          try {
            await api.news.remove(id);
            await loadNews();
            toastMsg(t('news.toastDeleted'));
          } catch (err) {
            reportActionError({ setState, toastMsg }, err, 'error.delete');
          }
        },
      }),
    [api, askConfirm, loadNews, setState, toastMsg],
  );

  return { openNewsForm, saveNews, removeNews };
}
