import {
  useEventActionFeatures,
  useEventDetailActions,
  useAbsenceActions,
  useCalExportActions,
  useEventFormActions,
} from '@/features/events';
import { useFinanceActions } from '@/features/finances';
import { useMemberActions } from '@/features/members';
import { useNewsActions } from '@/features/news';
import { useNotificationActions } from '@/features/notifications';
import { usePollActions } from '@/features/polls';
import { useTeamActions, useRoleActions } from '@/features/team';
import type { AppState, ConfirmConfig } from './AppContext';
import type { api as defaultApi } from '@/services/serviceLayer';
import type { DateRange, Role, TeamForUser } from '@/types';

type SetState = (patch: Partial<AppState> | ((s: AppState) => Partial<AppState>)) => void;

type FeatureActionDeps = {
  api: typeof defaultApi;
  S: () => AppState;
  setState: SetState;
  activeTeam: () => TeamForUser | null;
  myRoles: () => Role[];
  refreshEvents: () => Promise<void>;
  refreshMembers: () => Promise<void>;
  refreshRoles: () => Promise<void>;
  refreshTeams: () => Promise<void>;
  loadAbsences: () => Promise<void>;
  loadFinances: () => Promise<void>;
  loadStats: (range?: DateRange | null) => Promise<void>;
  loadNews: () => Promise<void>;
  loadPolls: () => Promise<void>;
  loadNotifications: () => Promise<void>;
  afterLoginLoad: (teamId: string) => Promise<void>;
  toastMsg: (m: string) => void;
  setFormVal: (patch: Record<string, unknown>) => void;
  askConfirm: (cfg: ConfirmConfig) => void;
  logout: () => void;
};

export function useFeatureActions(deps: FeatureActionDeps) {
  const {
    api,
    S,
    setState,
    activeTeam,
    myRoles,
    refreshEvents,
    refreshMembers,
    refreshRoles,
    refreshTeams,
    loadAbsences,
    loadFinances,
    loadStats,
    loadNews,
    loadPolls,
    loadNotifications,
    afterLoginLoad,
    toastMsg,
    setFormVal,
    askConfirm,
    logout,
  } = deps;

  const eventDetailActions = useEventDetailActions({
    api,
    S,
    setState,
    activeTeam,
    myRoles,
    refreshEvents,
    setFormVal,
    toastMsg,
    logout,
  });
  const eventActions = useEventActionFeatures({
    api,
    S,
    setState,
    activeTeam,
    myRoles,
    refreshEvents,
    setFormVal,
    toastMsg,
    logout,
    askConfirm,
    openEventDetail: eventDetailActions.openEventDetail,
  });
  const notifActions = useNotificationActions({ api, S, setState, loadNotifications, toastMsg, logout });
  const memberActions = useMemberActions({
    api,
    S,
    setState,
    refreshMembers,
    refreshTeams,
    askConfirm,
    toastMsg,
    logout,
  });
  const roleActions = useRoleActions({
    api,
    S,
    setState,
    activeTeam,
    refreshRoles,
    refreshTeams,
    askConfirm,
    toastMsg,
    logout,
  });
  const teamActions = useTeamActions({
    api,
    S,
    setState,
    activeTeam,
    refreshTeams,
    refreshMembers,
    setFormVal,
    afterLoginLoad,
    toastMsg,
    logout,
  });
  const absenceActions = useAbsenceActions({
    api,
    S,
    setState,
    refreshEvents,
    loadAbsences,
    askConfirm,
    toastMsg,
    logout,
  });
  const calExportActions = useCalExportActions({ S, setState, activeTeam, toastMsg });
  const newsActions = useNewsActions({ api, S, setState, loadNews, askConfirm, toastMsg, logout });
  const financeActions = useFinanceActions({
    api,
    S,
    setState,
    loadFinances,
    loadStats,
    refreshMembers,
    askConfirm,
    toastMsg,
    logout,
  });
  const pollActions = usePollActions({ api, S, setState, loadPolls, toastMsg, askConfirm, logout });
  const eventFormActions = useEventFormActions({
    api,
    S,
    setState,
    refreshEvents,
    openEventDetail: eventDetailActions.openEventDetail,
    toastMsg,
    logout,
  });

  return {
    ...eventDetailActions,
    ...eventActions,
    ...notifActions,
    ...memberActions,
    ...roleActions,
    ...teamActions,
    ...absenceActions,
    ...calExportActions,
    ...newsActions,
    ...financeActions,
    ...pollActions,
    ...eventFormActions,
  };
}
