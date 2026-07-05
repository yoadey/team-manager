import { describe, it, expect, vi, beforeEach } from 'vitest';
import { renderHook, act } from '@testing-library/react';
import { useNewsActions } from './useNewsActions';
import type { AppState } from '@/context/AppContext';

function makeState(overrides: Partial<AppState> = {}): AppState {
  return {
    phase: 'app',
    user: { id: 'u1', name: 'Test User', email: 'test@test.com', avatarColor: '#000', photo: null },
    activeTeamId: 'team1',
    sheet: null,
    form: {},
    formErrors: {},
    busy: null,
    toast: null,
    route: 'home',
    events: [],
    members: [],
    finances: null,
    stats: null,
    statsRange: null,
    news: [],
    polls: [],
    teams: [],
    roles: [],
    notifUnread: 0,
    notifications: [],
    primaryColor: '#000',
    ...overrides,
  } as unknown as AppState;
}

describe('useNewsActions', () => {
  let setState: ReturnType<typeof vi.fn>;
  let toastMsg: ReturnType<typeof vi.fn>;
  let loadNews: ReturnType<typeof vi.fn>;
  let askConfirm: ReturnType<typeof vi.fn>;
  let logout: ReturnType<typeof vi.fn>;
  let api: {
    news: { create: ReturnType<typeof vi.fn>; update: ReturnType<typeof vi.fn>; remove: ReturnType<typeof vi.fn> };
  };
  let stateRef: AppState;

  beforeEach(() => {
    stateRef = makeState();
    setState = vi.fn((patch) => {
      if (typeof patch === 'function') {
        const result = patch(stateRef);
        stateRef = { ...stateRef, ...result };
      } else {
        stateRef = { ...stateRef, ...patch };
      }
    });
    toastMsg = vi.fn();
    loadNews = vi.fn().mockResolvedValue(undefined);
    askConfirm = vi.fn();
    logout = vi.fn();
    api = {
      news: {
        create: vi.fn().mockResolvedValue({ id: 'n1' }),
        update: vi.fn().mockResolvedValue(undefined),
        remove: vi.fn().mockResolvedValue(undefined),
      },
    };
  });

  function renderActions() {
    return renderHook(() =>
      useNewsActions({
        api: api as never,
        S: () => stateRef,
        setState: setState as never,
        loadNews: loadNews as never,
        askConfirm: askConfirm as never,
        toastMsg: toastMsg as never,
        logout: logout as never,
      }),
    );
  }

  it('openNewsForm sets create sheet with empty form', () => {
    const { result } = renderActions();
    act(() => {
      result.current.openNewsForm();
    });
    expect(setState).toHaveBeenCalledWith(
      expect.objectContaining({
        sheet: expect.objectContaining({ type: 'newsForm', mode: 'create' }),
        form: expect.objectContaining({ title: '', body: '' }),
      }),
    );
  });

  it('openNewsForm sets edit sheet when news item passed', () => {
    const news = { id: 'n1', title: 'Hello', body: 'World', pinned: false } as never;
    const { result } = renderActions();
    act(() => {
      result.current.openNewsForm(news);
    });
    expect(setState).toHaveBeenCalledWith(
      expect.objectContaining({
        sheet: expect.objectContaining({ type: 'newsForm', mode: 'edit' }),
        form: expect.objectContaining({ id: 'n1', title: 'Hello', body: 'World' }),
      }),
    );
  });

  it('saveNews shows toast when title is empty', async () => {
    stateRef = makeState({ form: { title: '', body: 'Content' } });
    const { result } = renderActions();
    await act(async () => {
      await result.current.saveNews();
    });
    expect(toastMsg).toHaveBeenCalledWith('Bitte Titel angeben.');
    expect(api.news.create).not.toHaveBeenCalled();
  });

  it('saveNews creates news in create mode (no id)', async () => {
    stateRef = makeState({ form: { title: 'New Article', body: 'Content', pinned: false } });
    const { result } = renderActions();
    await act(async () => {
      await result.current.saveNews();
    });
    expect(api.news.create).toHaveBeenCalledWith('team1', expect.objectContaining({ title: 'New Article' }));
    expect(toastMsg).toHaveBeenCalledWith('News veröffentlicht');
  });

  it('saveNews updates news in edit mode (has id)', async () => {
    stateRef = makeState({ form: { id: 'n1', title: 'Updated', body: 'New content', pinned: true } });
    const { result } = renderActions();
    await act(async () => {
      await result.current.saveNews();
    });
    expect(api.news.update).toHaveBeenCalledWith('n1', expect.objectContaining({ title: 'Updated' }), 'team1');
    expect(toastMsg).toHaveBeenCalledWith('News aktualisiert');
  });

  it('removeNews calls askConfirm', () => {
    const { result } = renderActions();
    act(() => {
      result.current.removeNews('n1');
    });
    expect(askConfirm).toHaveBeenCalledWith(
      expect.objectContaining({
        title: 'News löschen?',
        danger: true,
      }),
    );
  });

  it('removeNews onConfirm deletes news and shows toast', async () => {
    const { result } = renderActions();
    act(() => {
      result.current.removeNews('n1');
    });
    const cfg = askConfirm.mock.calls[0][0];
    await act(async () => {
      await cfg.onConfirm();
    });
    expect(api.news.remove).toHaveBeenCalledWith('n1', 'team1');
    expect(toastMsg).toHaveBeenCalledWith('News gelöscht');
  });

  it('saveNews handles API error gracefully', async () => {
    api.news.create = vi.fn().mockRejectedValue(new Error('Network error'));
    stateRef = makeState({ form: { title: 'Test', body: 'Content' } });
    const { result } = renderActions();
    await act(async () => {
      await result.current.saveNews();
    });
    expect(toastMsg).toHaveBeenCalled();
    expect(setState).toHaveBeenCalledWith(expect.objectContaining({ busy: 'save' }));
  });
});
