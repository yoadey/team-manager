import { useCallback } from 'react';
import type { api as defaultApi } from '../../../services/serviceLayer';
import type { Poll } from '../../../types';
import type { AppState } from '../../../store/AppContext';

type SetState = (patch: Partial<AppState> | ((s: AppState) => Partial<AppState>)) => void;

type PollFeatureDeps = { api: typeof defaultApi; S: () => AppState; setState: SetState; loadPolls: () => Promise<void>; toastMsg: (m: string) => void };

export function usePollActions({ api, S, setState, loadPolls, toastMsg }: PollFeatureDeps) {
  const openPollForm = useCallback(() => setState({ sheet: { type: 'pollForm' }, form: { question: '', opt0: '', opt1: '', opt2: '', opt3: '', multiple: false, anonymous: false } }), [setState]);
  const votePoll = useCallback(async (pollId: string, optionIds: string[]) => { await api.polls.vote(pollId, optionIds); await loadPolls(); }, [api, loadPolls]);
  const savePoll = useCallback(async () => { const f = S().form; const opts = [f.opt0, f.opt1, f.opt2, f.opt3].filter((o) => o && o.trim()); if (!f.question || opts.length < 2) { toastMsg('Frage und mind. 2 Optionen'); return; } setState({ busy: 'save' }); await api.polls.create(S().activeTeamId!, { question: f.question, options: opts, multiple: f.multiple, anonymous: f.anonymous }); await loadPolls(); setState({ busy: null, sheet: null }); toastMsg('Umfrage erstellt'); }, [api, S, setState, loadPolls, toastMsg]);
  const togglePollOption = useCallback((poll: Poll, optId: string) => { const cur = poll.myVote || []; let next: string[]; if (poll.multiple) next = cur.includes(optId) ? cur.filter((x) => x !== optId) : cur.concat(optId); else next = [optId]; votePoll(poll.id, next); }, [votePoll]);

  return { openPollForm, savePoll, togglePollOption };
}
