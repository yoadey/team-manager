import { useCallback } from 'react';
import type { api as defaultApi } from '../../../services/serviceLayer';
import type { AppState } from '../../../context/AppContext';

type SetState = (patch: Partial<AppState> | ((s: AppState) => Partial<AppState>)) => void;

type NewsDeps = {
  api: typeof defaultApi;
  S: () => AppState;
  setState: SetState;
  loadNews: () => Promise<void>;
  askConfirm: (cfg: { title: string; message: string; confirmLabel?: string; danger?: boolean; onConfirm: () => void | Promise<void> }) => void;
  toastMsg: (m: string) => void;
};

export function useNewsActions({ api, S, setState, loadNews, askConfirm, toastMsg }: NewsDeps) {
  const openNewsForm = useCallback(() => setState({ sheet: { type: 'newsForm' }, form: { title: '', body: '', pinned: false } }), [setState]);

  const saveNews = useCallback(async () => {
    const f = S().form;
    if (!f.title) { toastMsg('Bitte Titel angeben'); return; }
    setState({ busy: 'save' });
    await api.news.create(S().activeTeamId!, { title: f.title, body: f.body, pinned: f.pinned });
    await loadNews();
    setState({ busy: null, sheet: null });
    toastMsg('News veröffentlicht');
  }, [api, S, setState, loadNews, toastMsg]);

  const removeNews = useCallback((id: string) => askConfirm({
    title: 'News löschen?',
    message: 'Diese Neuigkeit wird dauerhaft entfernt.',
    confirmLabel: 'Löschen', danger: true,
    onConfirm: async () => { await api.news.remove(id); await loadNews(); toastMsg('News gelöscht'); },
  }), [api, askConfirm, loadNews, toastMsg]);

  return { openNewsForm, saveNews, removeNews };
}
