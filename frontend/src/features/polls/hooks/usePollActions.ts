import { useCallback, useRef } from 'react';
import type { api as defaultApi } from '@/services/serviceLayer';
import type { Poll, PollFormValues } from '../types';
import type { AppState } from '@/context/AppContext';
import { validatePollForm } from '@/utils/validation';
import { reportActionError } from '@/utils/errors';
import { clearBusyIfOwned } from '@/utils/forms';
import { t } from '@/i18n';

type SetState = (patch: Partial<AppState> | ((s: AppState) => Partial<AppState>)) => void;

type PollFeatureDeps = {
  api: typeof defaultApi;
  S: () => AppState;
  setState: SetState;
  loadPolls: () => Promise<void>;
  toastMsg: (m: string) => void;
  askConfirm: (cfg: {
    title: string;
    message: string;
    confirmLabel?: string;
    danger?: boolean;
    onConfirm: () => void | Promise<void>;
  }) => void;
  logout: () => void;
};

export function usePollActions({ api, S, setState, loadPolls, toastMsg, askConfirm, logout }: PollFeatureDeps) {
  const openPollForm = useCallback(() => {
    const form: PollFormValues = {
      question: '',
      opt0: '',
      opt1: '',
      opt2: '',
      opt3: '',
      multiple: false,
      anonymous: false,
    };
    setState({ sheet: { type: 'pollForm' }, form, formErrors: {} });
  }, [setState]);

  // Guards against the lost-update race where two quick clicks on different
  // options of the same multi-select poll both read the same stale
  // poll.myVote and fire overlapping vote requests — whichever response
  // lands last would silently overwrite the other's selection. Dropping a
  // second click while the first is still in flight (rather than queuing it)
  // matches the same-class guards elsewhere (e.g. useFinanceActions'
  // togglePenalty/toggleContribution).
  const voteInFlight = useRef(new Set<string>());

  const votePoll = useCallback(
    async (pollId: string, optionIds: string[]) => {
      if (voteInFlight.current.has(pollId)) return;
      voteInFlight.current.add(pollId);
      try {
        await api.polls.vote(pollId, optionIds, S().activeTeamId!);
        await loadPolls();
      } catch (err) {
        reportActionError({ setState, toastMsg, onAuthError: logout }, err);
      } finally {
        voteInFlight.current.delete(pollId);
      }
    },
    [api, S, loadPolls, setState, toastMsg, logout],
  );

  const savePoll = useCallback(async () => {
    const f = S().form as PollFormValues;
    const poll = validatePollForm(f);
    if (!poll.ok) {
      toastMsg(poll.message!);
      return;
    }
    const sh = S().sheet;
    const teamId = S().activeTeamId!;
    setState({ busy: 'save' });
    try {
      await api.polls.create(teamId, {
        question: poll.value!.question,
        options: poll.value!.options,
        multiple: f.multiple,
        anonymous: f.anonymous,
      });
      await loadPolls();
      clearBusyIfOwned(S, setState, 'save');
      // Don't close a sheet the user has since opened for a different team
      // after switching away mid-request, or one they've since opened for a
      // different entity (same team) while this save was in flight.
      if (S().activeTeamId === teamId && S().sheet === sh) setState({ sheet: null });
      toastMsg(t('polls.toastCreated'));
    } catch (err) {
      reportActionError({ setState, toastMsg, onAuthError: logout }, err, 'error.save');
    }
  }, [api, S, setState, loadPolls, toastMsg, logout]);

  const togglePollOption = useCallback(
    (poll: Poll, optId: string) => {
      const cur = poll.myVote || [];
      let next: string[];
      if (poll.multiple) next = cur.includes(optId) ? cur.filter((x) => x !== optId) : cur.concat(optId);
      else next = [optId];
      votePoll(poll.id, next);
    },
    [votePoll],
  );

  const removePoll = useCallback(
    (id: string) =>
      askConfirm({
        title: t('polls.deleteConfirmTitle'),
        message: t('polls.deleteConfirmMsg'),
        confirmLabel: t('common.delete'),
        danger: true,
        onConfirm: async () => {
          try {
            await api.polls.remove(id, S().activeTeamId!);
            await loadPolls();
            toastMsg(t('polls.toastDeleted'));
          } catch (err) {
            reportActionError({ setState, toastMsg, onAuthError: logout }, err, 'error.delete');
          }
        },
      }),
    [api, S, askConfirm, loadPolls, setState, toastMsg, logout],
  );

  return { openPollForm, savePoll, togglePollOption, removePoll };
}
