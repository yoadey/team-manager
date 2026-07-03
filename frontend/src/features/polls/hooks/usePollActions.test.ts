import { describe, it, expect, vi, beforeEach } from 'vitest';
import { renderHook, act } from '@testing-library/react';
import { usePollActions } from './usePollActions';
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

describe('usePollActions', () => {
  let setState: ReturnType<typeof vi.fn>;
  let toastMsg: ReturnType<typeof vi.fn>;
  let loadPolls: ReturnType<typeof vi.fn>;
  let askConfirm: ReturnType<typeof vi.fn>;
  let api: {
    polls: { vote: ReturnType<typeof vi.fn>; create: ReturnType<typeof vi.fn>; remove: ReturnType<typeof vi.fn> };
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
    loadPolls = vi.fn().mockResolvedValue(undefined);
    askConfirm = vi.fn();
    api = {
      polls: {
        vote: vi.fn().mockResolvedValue(undefined),
        create: vi.fn().mockResolvedValue({ id: 'poll1' }),
        remove: vi.fn().mockResolvedValue(undefined),
      },
    };
  });

  function renderActions() {
    return renderHook(() =>
      usePollActions({
        api: api as never,
        S: () => stateRef,
        setState: setState as never,
        loadPolls: loadPolls as never,
        toastMsg: toastMsg as never,
        askConfirm: askConfirm as never,
      }),
    );
  }

  it('openPollForm sets pollForm sheet with empty form', () => {
    const { result } = renderActions();
    act(() => {
      result.current.openPollForm();
    });
    expect(setState).toHaveBeenCalledWith(
      expect.objectContaining({
        sheet: { type: 'pollForm' },
        form: expect.objectContaining({ question: '', opt0: '', opt1: '' }),
      }),
    );
  });

  it('savePoll shows toast when form is invalid (no question)', async () => {
    stateRef = makeState({ form: { question: '', opt0: 'A', opt1: 'B' } });
    const { result } = renderActions();
    await act(async () => {
      await result.current.savePoll();
    });
    expect(toastMsg).toHaveBeenCalled();
    expect(api.polls.create).not.toHaveBeenCalled();
  });

  it('savePoll shows toast when less than 2 options', async () => {
    stateRef = makeState({ form: { question: 'Which?', opt0: 'A', opt1: '' } });
    const { result } = renderActions();
    await act(async () => {
      await result.current.savePoll();
    });
    expect(toastMsg).toHaveBeenCalled();
    expect(api.polls.create).not.toHaveBeenCalled();
  });

  it('savePoll creates poll when valid', async () => {
    stateRef = makeState({
      form: {
        question: 'Treffen heute?',
        opt0: 'Ja',
        opt1: 'Nein',
        opt2: '',
        opt3: '',
        multiple: false,
        anonymous: false,
      },
    });
    const { result } = renderActions();
    await act(async () => {
      await result.current.savePoll();
    });
    expect(api.polls.create).toHaveBeenCalledWith(
      'team1',
      expect.objectContaining({
        question: 'Treffen heute?',
        options: expect.arrayContaining(['Ja', 'Nein']),
      }),
    );
    expect(toastMsg).toHaveBeenCalledWith('Umfrage erstellt');
  });

  it('togglePollOption sets single option for non-multiple poll', async () => {
    const poll = { id: 'poll1', multiple: false, myVote: ['opt1'] } as never;
    const { result } = renderActions();
    await act(async () => {
      result.current.togglePollOption(poll, 'opt2');
    });
    expect(api.polls.vote).toHaveBeenCalledWith('poll1', ['opt2'], 'team1');
  });

  it('togglePollOption toggles multiple options for multiple poll', async () => {
    const poll = { id: 'poll1', multiple: true, myVote: ['opt1'] } as never;
    const { result } = renderActions();
    await act(async () => {
      result.current.togglePollOption(poll, 'opt2');
    });
    expect(api.polls.vote).toHaveBeenCalledWith('poll1', ['opt1', 'opt2'], 'team1');
  });

  it('togglePollOption removes option from multiple poll when already selected', async () => {
    const poll = { id: 'poll1', multiple: true, myVote: ['opt1', 'opt2'] } as never;
    const { result } = renderActions();
    await act(async () => {
      result.current.togglePollOption(poll, 'opt1');
    });
    expect(api.polls.vote).toHaveBeenCalledWith('poll1', ['opt2'], 'team1');
  });

  it('votePoll drops a second concurrent vote for the same poll to avoid a lost-update race', async () => {
    let resolveFirstVote: () => void = () => {};
    api.polls.vote.mockImplementationOnce(
      () =>
        new Promise<void>((resolve) => {
          resolveFirstVote = resolve;
        }),
    );

    const poll = { id: 'poll1', multiple: true, myVote: ['opt1'] } as never;
    const { result } = renderActions();

    await act(async () => {
      // Fire two rapid clicks on different options before the first request
      // resolves — both read the same stale poll.myVote via togglePollOption.
      result.current.togglePollOption(poll, 'opt2');
      result.current.togglePollOption(poll, 'opt3');
      resolveFirstVote();
      // Flush the pending vote/loadPolls microtasks.
      await new Promise((resolve) => setTimeout(resolve, 0));
    });

    expect(api.polls.vote).toHaveBeenCalledTimes(1);
    expect(api.polls.vote).toHaveBeenCalledWith('poll1', ['opt1', 'opt2'], 'team1');
  });

  it('removePoll calls askConfirm', () => {
    const { result } = renderActions();
    act(() => {
      result.current.removePoll('poll1');
    });
    expect(askConfirm).toHaveBeenCalledWith(
      expect.objectContaining({
        title: 'Umfrage löschen?',
        danger: true,
      }),
    );
  });

  it('removePoll onConfirm removes poll and shows toast', async () => {
    const { result } = renderActions();
    act(() => {
      result.current.removePoll('poll1');
    });
    const cfg = askConfirm.mock.calls[0][0];
    await act(async () => {
      await cfg.onConfirm();
    });
    expect(api.polls.remove).toHaveBeenCalledWith('poll1', 'team1');
    expect(toastMsg).toHaveBeenCalledWith('Umfrage gelöscht');
  });
});
