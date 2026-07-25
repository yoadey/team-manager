import { describe, it, expect, vi, beforeEach } from 'vitest';
import { renderHook, act } from '@testing-library/react';
import { useNewsActions } from './useNewsActions';
import { createQueryWrapper } from '@/test/queryTestUtils';
import type { AppState } from '@/context/AppContext';
import type { NewsFormValues } from '../components/newsFormSchema';

function makeState(overrides: Partial<AppState> = {}): AppState {
  return {
    phase: 'app',
    user: { id: 'u1', name: 'Test User', email: 'test@test.com', avatarColor: '#000', photo: null },
    activeTeamId: 'team1',
    sheet: null,
    busy: null,
    toast: null,
    route: 'home',
    events: [],
    members: [],
    finances: null,
    stats: null,
    statsRange: null,
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
  let askConfirm: ReturnType<typeof vi.fn>;
  let loadNotifications: ReturnType<typeof vi.fn>;
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
    askConfirm = vi.fn();
    loadNotifications = vi.fn().mockResolvedValue(undefined);
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
    return renderHook(
      () =>
        useNewsActions({
          api: api as never,
          S: () => stateRef,
          setState: setState as never,
          teamId: stateRef.activeTeamId,
          loadNotifications: loadNotifications as never,
          askConfirm: askConfirm as never,
          toastMsg: toastMsg as never,
          logout: logout as never,
        }),
      { wrapper: createQueryWrapper() },
    );
  }

  it('openNewsForm sets create sheet with empty form', () => {
    const { result } = renderActions();
    act(() => {
      result.current.openNewsForm();
    });
    expect(setState).toHaveBeenCalledWith(
      expect.objectContaining({
        sheet: expect.objectContaining({
          type: 'newsForm',
          mode: 'create',
          formInitial: expect.objectContaining({ title: '', body: '' }),
        }),
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
        sheet: expect.objectContaining({
          type: 'newsForm',
          mode: 'edit',
          formInitial: expect.objectContaining({ id: 'n1', title: 'Hello', body: 'World' }),
        }),
      }),
    );
  });

  it('saveNews creates news in create mode (no id)', async () => {
    const { result } = renderActions();
    await act(async () => {
      await result.current.saveNews({ title: 'New Article', body: 'Content', pinned: false } as NewsFormValues);
    });
    expect(api.news.create).toHaveBeenCalledWith('team1', expect.objectContaining({ title: 'New Article' }));
    expect(toastMsg).toHaveBeenCalledWith('News veröffentlicht');
    expect(loadNotifications).toHaveBeenCalled();
  });

  it('saveNews updates news in edit mode (has id)', async () => {
    const { result } = renderActions();
    await act(async () => {
      await result.current.saveNews({
        id: 'n1',
        title: 'Updated',
        body: 'New content',
        pinned: true,
      } as NewsFormValues);
    });
    expect(api.news.update).toHaveBeenCalledWith('n1', expect.objectContaining({ title: 'Updated' }), 'team1');
    expect(toastMsg).toHaveBeenCalledWith('News aktualisiert');
    expect(loadNotifications).toHaveBeenCalled();
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
    const cfg = askConfirm.mock.calls[0]![0];
    await act(async () => {
      await cfg.onConfirm();
    });
    expect(api.news.remove).toHaveBeenCalledWith('n1', 'team1');
    expect(toastMsg).toHaveBeenCalledWith('News gelöscht');
    expect(loadNotifications).toHaveBeenCalled();
  });

  it('saveNews handles API error gracefully', async () => {
    api.news.create = vi.fn().mockRejectedValue(new Error('Network error'));
    const { result } = renderActions();
    await act(async () => {
      await expect(
        result.current.saveNews({ title: 'Test', body: 'Content' } as NewsFormValues),
      ).rejects.toThrow('Network error');
    });
    expect(toastMsg).toHaveBeenCalledWith(expect.stringContaining('Network error'), undefined, 'error');
  });

  // Regression test: mirrors useDeleteEventMutation/useRemoveMemberMutation's
  // per-call teamId safeguard. The confirm sheet can still be open (and get
  // confirmed) after the user has switched to a different active team; the
  // delete must still target the team the confirm dialog was opened for.
  it('removeNews onConfirm deletes against the team the confirm dialog was opened for, even after a team switch', async () => {
    const { result, rerender } = renderActions();
    act(() => {
      result.current.removeNews('n1');
    });
    const cfg = askConfirm.mock.calls[0]![0];

    stateRef = { ...stateRef, activeTeamId: 'team2' };
    rerender();

    await act(async () => {
      await cfg.onConfirm();
    });
    expect(api.news.remove).toHaveBeenCalledWith('n1', 'team1');
  });
});
