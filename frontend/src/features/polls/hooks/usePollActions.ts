import { useCallback } from 'react';
import type { api as defaultApi } from '@/services/serviceLayer';
import type { Poll, PollFormValues } from '../types';
import type { AppState } from '@/context/AppContext';
import { validatePollForm } from '@/utils/validation';
import { reportActionError } from '@/utils/errors';
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
};

export function usePollActions({ api, S, setState, loadPolls, toastMsg, askConfirm }: PollFeatureDeps) {
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

  const votePoll = useCallback(
    async (pollId: string, optionIds: string[]) => {
      try {
        await api.polls.vote(pollId, optionIds);
        await loadPolls();
      } catch (err) {
        reportActionError({ setState, toastMsg }, err);
      }
    },
    [api, loadPolls, setState, toastMsg],
  );

  const savePoll = useCallback(async () => {
    const f = S().form as PollFormValues;
    const poll = validatePollForm(f);
    if (!poll.ok) {
      toastMsg(poll.message!);
      return;
    }
    setState({ busy: 'save' });
    try {
      await api.polls.create(S().activeTeamId!, {
        question: poll.value!.question,
        options: poll.value!.options,
        multiple: f.multiple,
        anonymous: f.anonymous,
      });
      await loadPolls();
      setState({ busy: null, sheet: null });
      toastMsg(t('polls.toastCreated'));
    } catch (err) {
      reportActionError({ setState, toastMsg }, err, 'error.save');
    }
  }, [api, S, setState, loadPolls, toastMsg]);

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
            await api.polls.remove(id);
            await loadPolls();
            toastMsg(t('polls.toastDeleted'));
          } catch (err) {
            reportActionError({ setState, toastMsg }, err, 'error.delete');
          }
        },
      }),
    [api, askConfirm, loadPolls, setState, toastMsg],
  );

  return { openPollForm, savePoll, togglePollOption, removePoll };
}
