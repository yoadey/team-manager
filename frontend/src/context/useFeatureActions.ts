import {
  useEventActionFeatures,
  useEventDetailActions,
  useAbsenceActions,
  useCalExportActions,
  useEventFormActions,
} from '@/features/events';
import { useFinanceActions } from '@/features/finances';
import { useMemberActions, useInvalidateMembers } from '@/features/members';
import { useNewsActions } from '@/features/news';
import { useNotificationActions } from '@/features/notifications';
import { usePollActions } from '@/features/polls';
import { useTeamActions, useRoleActions } from '@/features/team';
import type { AppState, ConfirmConfig } from './AppContext';
import type { api as defaultApi } from '@/services';
import type { Role, TeamForUser } from '@/types';

type SetState = (patch: Partial<AppState> | ((s: AppState) => Partial<AppState>)) => void;

type FeatureActionDeps = {
  api: typeof defaultApi;
  S: () => AppState;
  setState: SetState;
  activeTeam: () => TeamForUser | null;
  myRoles: () => Role[];
  /** Reactive active team id, for the events/members/finances/polls/news/absences/stats verticals' query/mutation hooks. */
  teamId: string | null;
  refreshRoles: () => Promise<void>;
  refreshTeams: () => Promise<void>;
  loadNotifications: () => Promise<void>;
  afterLoginLoad: (teamId: string) => Promise<void>;
  toastMsg: (m: string, action?: { label: string; fn: () => void }, kind?: 'success' | 'error') => void;
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
    teamId,
    refreshRoles,
    refreshTeams,
    loadNotifications,
    afterLoginLoad,
    toastMsg,
    setFormVal,
    askConfirm,
    logout,
  } = deps;

  // Used by useTeamActions' uploadMyPhoto (not migrated yet), whose own-photo
  // change is visible in the member list too.
  const invalidateMembers = useInvalidateMembers(teamId);

  const eventDetailActions = useEventDetailActions({
    api,
    S,
    setState,
    activeTeam,
    myRoles,
    teamId,
    loadNotifications,
    setFormVal,
    askConfirm,
    toastMsg,
    logout,
  });
  const eventActions = useEventActionFeatures({
    api,
    S,
    setState,
    loadNotifications,
    toastMsg,
    logout,
    askConfirm,
    openEventDetail: eventDetailActions.openEventDetail,
  });
  const notifActions = useNotificationActions({ api, setState, teamId, toastMsg, logout });
  const memberActions = useMemberActions({
    api,
    S,
    setState,
    teamId,
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
    invalidateMembers,
    setFormVal,
    afterLoginLoad,
    toastMsg,
    logout,
  });
  const absenceActions = useAbsenceActions({
    api,
    S,
    setState,
    teamId,
    loadNotifications,
    askConfirm,
    toastMsg,
    logout,
  });
  const calExportActions = useCalExportActions({ api, S, setState, activeTeam, teamId, toastMsg });
  const newsActions = useNewsActions({ api, S, setState, teamId, loadNotifications, askConfirm, toastMsg, logout });
  const financeActions = useFinanceActions({
    api,
    S,
    setState,
    teamId,
    askConfirm,
    toastMsg,
    logout,
  });
  const pollActions = usePollActions({ api, S, setState, teamId, loadNotifications, toastMsg, askConfirm, logout });
  const eventFormActions = useEventFormActions({
    api,
    S,
    setState,
    teamId,
    loadNotifications,
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
