import React, { createContext, useCallback, useContext, useEffect, useMemo, useRef, useState } from 'react';
import { api as defaultApi, resetDemoData } from '@/services/serviceLayer';
import type {
  AttendanceStatus,
  DateRange,
  MemberAttendanceStats,
  ModuleKey,
  PermLevel,
  Provider,
  Role,
  StatsOverview,
  TeamForUser,
  User,
} from '@/types';
import type { Absence, AttendanceRow, TeamEvent } from '@/features/events';
import type { Contribution, FinanceOverview, Penalty, Transaction } from '@/features/finances';
import type { Member } from '@/features/members';
import type { NewsItem } from '@/features/news';
import type { AppNotification } from '@/features/notifications';
import type { Poll } from '@/features/polls';
import { DEFAULT_PRESET_KEY } from '@/styles/tokens';
import { canForTeam, isStaffForTeam } from '@/utils/permissions';
import { reportActionError } from '@/utils/errors';
import { setSentryUser } from '@/monitoring';
import { t } from '@/i18n';
import { useFeatureActions } from './useFeatureActions';

export type Phase = 'loading' | 'login' | 'app';
export type Route = 'home' | 'events' | 'members' | 'finances' | 'stats' | 'news' | 'polls' | 'team';

const ALL_ROUTES: Route[] = ['home', 'events', 'members', 'finances', 'stats', 'news', 'polls', 'team'];
function routeFromPath(path: string): Route {
  const seg = path.replace(/^\//, '').split('/')[0] as Route;
  return ALL_ROUTES.includes(seg) ? seg : 'home';
}
function pushRoute(route: Route) {
  history.pushState({ route }, '', '/' + route);
}
export type SheetType =
  | 'teams'
  | 'profile'
  | 'more'
  | 'teamSettings'
  | 'createTeam'
  | 'invite'
  | 'notifications'
  | 'calExport'
  | 'eventDetail'
  | 'eventForm'
  | 'seriesAction'
  | 'comment'
  | 'absenceForm'
  | 'memberDetail'
  | 'memberForm'
  | 'newsForm'
  | 'txForm'
  | 'penaltyCatalog'
  | 'penaltyAssign'
  | 'penaltyForm'
  | 'contribForm'
  | 'pollForm'
  | 'roles'
  | 'roleForm'
  | 'confirm';

export interface ConfirmConfig {
  title?: string;
  message?: string;
  confirmLabel?: string;
  danger?: boolean;
  onConfirm?: () => void | Promise<void>;
}

export interface SheetState {
  type: SheetType;
  back?: SheetState | null;
  // Typed payload properties shared across sheet variants (all optional)
  cfg?: ConfirmConfig;
  mode?: 'edit' | 'create';
  self?: boolean;
  action?: 'cancel' | 'delete' | 'reactivate';
  event?: TeamEvent | null;
  eventId?: string;
  rows?: AttendanceRow[];
  comments?: import('@/features/events').EventComment[];
  membershipId?: string;
  member?: Member | null;
  stats?: MemberAttendanceStats | null;
  userId?: string;
  name?: string;
  status?: AttendanceStatus;
  invite?: import('@/types').Invite | null;
  copied?: boolean;
}

export interface AppState {
  phase: Phase;
  providers: Provider[];
  busy: string | null;
  primaryColor: string;
  user: User | null;
  teams: TeamForUser[];
  activeTeamId: string | null;
  route: Route;
  eventScope: 'upcoming' | 'past';
  eventsView: 'list' | 'calendar' | 'absences';
  eventsOnlyPending: boolean;
  calShowAbsences: boolean;
  calMonth: Date | null;
  events: TeamEvent[];
  members: Member[];
  roles: Role[];
  news: NewsItem[] | null;
  finances: FinanceOverview | null;
  stats: StatsOverview | null;
  polls: Poll[] | null;
  absences: Absence[] | null;
  myAbsences: Absence[] | null;
  notifications: AppNotification[] | null;
  notifUnread: number;
  notifFilter: 'all' | 'attendance' | 'events' | 'other';
  statsRange: DateRange | null;
  finTab: 'umsaetze' | 'strafen' | 'beitraege';
  contribMonth: string | null;
  sheet: SheetState | null;
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  form: Record<string, any>;
  formErrors: Record<string, string>;
  toast: string | null;
  error: string | null;
}

const initialState: AppState = {
  phase: 'loading',
  providers: [],
  busy: null,
  primaryColor: DEFAULT_PRESET_KEY,
  user: null,
  teams: [],
  activeTeamId: null,
  route: routeFromPath(window.location.pathname),
  eventScope: 'upcoming',
  eventsView: 'list',
  eventsOnlyPending: false,
  calShowAbsences: false,
  calMonth: null,
  events: [],
  members: [],
  roles: [],
  news: null,
  finances: null,
  stats: null,
  polls: null,
  absences: null,
  myAbsences: null,
  notifications: null,
  notifUnread: 0,
  notifFilter: 'all',
  statsRange: null,
  finTab: 'umsaetze',
  contribMonth: null,
  sheet: null,
  form: {},
  formErrors: {},
  toast: null,
  error: null,
};

const PAGE_SHEETS = ['eventDetail', 'eventForm', 'memberDetail', 'memberForm', 'teamSettings', 'roles', 'roleForm'];
export const isPageSheet = (type?: string | null) => !!type && PAGE_SHEETS.includes(type);

export interface AppContextValue {
  state: AppState;
  api: typeof defaultApi;
  // helpers
  activeTeam: () => TeamForUser | null;
  myRoles: () => Role[];
  can: (module: ModuleKey, level?: PermLevel) => boolean;
  isStaff: () => boolean;
  toastMsg: (m: string) => void;
  resetDemo: () => void;
  setPrimaryColor: (c: string) => void;
  // form
  onFormInput: (e: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement | HTMLSelectElement>) => void;
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  setFormVal: (patch: Record<string, any>) => void;
  setFormErrors: (patch: Record<string, string>) => void;
  onFile: (e: React.ChangeEvent<HTMLInputElement>, cb: (dataUrl: string) => void) => void;
  setState: (patch: Partial<AppState> | ((s: AppState) => Partial<AppState>)) => void;
  // auth
  doLogin: (pid: string) => Promise<void>;
  logout: () => void;
  // nav
  go: (route: Route) => void;
  goEventsPending: () => void;
  closeSheet: () => void;
  activePageSheet: () => SheetState | null;
  selectTeam: (id: string) => Promise<void>;
  setEventsView: (v: 'list' | 'calendar' | 'absences') => void;
  toggleCalAbsences: () => void;
  // notifications
  openNotifications: () => void;
  setNotifFilter: (f: AppState['notifFilter']) => void;
  loadAbsences: () => Promise<void>;
  // data refresh
  loadFinances: () => Promise<void>;
  loadStats: (range?: DateRange | null) => Promise<void>;
  // attendance
  setMyStatus: (eventId: string, status: AttendanceStatus) => Promise<void>;
  setStatusFor: (e: TeamEvent, row: AttendanceRow, status: AttendanceStatus) => void;
  canSeeComment: (row: AttendanceRow) => boolean;
  openComment: (e: TeamEvent, row: { userId: string; name: string; status: AttendanceStatus; reason?: string }) => void;
  submitComment: () => Promise<void>;
  postEventComment: (eventId: string) => Promise<void>;
  removeEventComment: (eventId: string, cid: string) => Promise<void>;
  toggleNomination: (eventId: string, userId: string, currentlyNominated: boolean) => Promise<void>;
  // confirm
  askConfirm: (cfg: {
    title: string;
    message: string;
    confirmLabel?: string;
    danger?: boolean;
    onConfirm: () => void | Promise<void>;
  }) => void;
  cancelConfirm: () => void;
  runConfirm: () => Promise<void>;
  // events
  askEventAction: (action: 'cancel' | 'delete' | 'reactivate', event: TeamEvent) => void;
  runEventAction: (
    action: 'cancel' | 'delete' | 'reactivate',
    event: TeamEvent,
    scope: 'single' | 'series',
  ) => Promise<void>;
  openEventDetail: (eventId: string) => Promise<void>;
  reloadDetail: (eventId: string) => Promise<void>;
  openEventForm: (event: TeamEvent | null) => void;
  saveEvent: (scope?: 'single' | 'series') => Promise<void>;
  toggleFormNomRole: (roleId: string) => void;
  // members
  openMemberDetail: (membershipId: string) => Promise<void>;
  openMemberForm: (member: Member) => void;
  toggleFormRole: (roleId: string) => void;
  saveMember: () => Promise<void>;
  removeMember: (membershipId: string) => void;
  // roles
  openRoles: () => void;
  openCreateRole: () => void;
  setRolePerm: (module: ModuleKey, level: PermLevel) => void;
  saveRole: () => Promise<void>;
  toggleMyRole: (roleId: string) => Promise<void>;
  // team
  openTeamSwitcher: () => void;
  openProfile: () => void;
  openMore: () => void;
  openTeamSettings: () => void;
  saveTeamPhoto: (dataUrl: string) => Promise<void>;
  saveTeamLogo: (dataUrl: string) => Promise<void>;
  setTeamIcon: (em: string) => void;
  toggleReasonRole: (roleId: string) => void;
  saveTeamSettings: () => Promise<void>;
  openCreateTeam: () => void;
  createTeam: () => Promise<void>;
  openInvite: () => Promise<void>;
  copyInvite: () => void;
  uploadMyPhoto: (dataUrl: string) => Promise<void>;
  // absences
  openAbsenceForm: (absence?: Absence | null) => void;
  saveAbsence: () => Promise<void>;
  removeAbsence: (id: string) => void;
  // calendar export
  openCalExport: () => void;
  downloadIcs: () => void;
  copyCalUrl: () => void;
  // news
  openNewsForm: (n?: import('@/features/news').NewsItem) => void;
  saveNews: () => Promise<void>;
  removeNews: (id: string) => void;
  // finances
  openTxForm: (tx?: Transaction) => void;
  saveTx: () => Promise<void>;
  deleteTx: (id: string) => Promise<void>;
  openPenaltyCatalog: () => void;
  openPenaltyForm: (p?: Penalty) => void;
  savePenalty: () => Promise<void>;
  deletePenaltyDef: (id: string) => void;
  openPenaltyAssign: () => void;
  savePenaltyAssign: () => Promise<void>;
  deleteAssignment: (id: string) => Promise<void>;
  openContribForm: (c: Contribution) => void;
  saveContrib: () => Promise<void>;
  togglePenalty: (id: string) => Promise<void>;
  toggleContribution: (id: string) => Promise<void>;
  setStatsRange: (range: DateRange | null) => void;
  // polls
  openPollForm: () => void;
  savePoll: () => Promise<void>;
  togglePollOption: (poll: Poll, optId: string) => void;
  removePoll: (id: string) => void;
}

/** Actions + helpers, without the mutable `state`. Stable across renders. */
export type AppActions = Omit<AppContextValue, 'state'>;

// State and actions are kept in separate contexts so components that only
// dispatch actions (via useAppActions) do not re-render on every state change.
// Actions have a stable identity; only state-subscribed consumers re-render.
const AppStateContext = createContext<AppState | null>(null);
const AppActionsContext = createContext<AppActions | null>(null);

export const useApp = (): AppContextValue => {
  const actions = useContext(AppActionsContext);
  const state = useContext(AppStateContext);
  if (!actions || !state) throw new Error('useApp must be used within AppProvider');
  return useMemo(() => ({ ...actions, state }), [actions, state]);
};

/**
 * Actions/helpers only — does NOT subscribe to state changes. Use in components
 * that merely dispatch and read their display data from props, so they skip
 * re-renders triggered by unrelated state updates.
 */
export const useAppActions = (): AppActions => {
  const actions = useContext(AppActionsContext);
  if (!actions) throw new Error('useAppActions must be used within AppProvider');
  return actions;
};

export function AppProvider({ children }: { children: React.ReactNode }) {
  const api = defaultApi;
  const [state, setRaw] = useState<AppState>(initialState);
  const stateRef = useRef(state);
  stateRef.current = state;
  const toastTimer = useRef<ReturnType<typeof setTimeout> | null>(null);

  const setState = useCallback((patch: Partial<AppState> | ((s: AppState) => Partial<AppState>)) => {
    setRaw((prev) => {
      const p = typeof patch === 'function' ? patch(prev) : patch;
      const next = { ...prev, ...p };
      stateRef.current = next;
      return next;
    });
  }, []);

  const S = () => stateRef.current;

  // ---------- helpers ----------
  const activeTeam = useCallback(() => S().teams.find((t) => t.id === S().activeTeamId) || null, []);
  const myRoles = useCallback(() => {
    const t = activeTeam();
    return t ? t.myRoles : [];
  }, [activeTeam]);
  const can = useCallback(
    (module: ModuleKey, level: PermLevel = 'write') => canForTeam(activeTeam(), module, level),
    [activeTeam],
  );
  const isStaff = useCallback(() => isStaffForTeam(activeTeam()), [activeTeam]);
  const toastMsg = useCallback(
    (m: string) => {
      if (toastTimer.current) clearTimeout(toastTimer.current);
      setState({ toast: m });
      toastTimer.current = setTimeout(() => setState({ toast: null }), 2600);
    },
    [setState],
  );
  const resetDemo = useCallback(() => {
    resetDemoData();
    location.reload();
  }, []);
  const setPrimaryColor = useCallback((c: string) => setState({ primaryColor: c }), [setState]);

  // ---------- form ----------
  const onFormInput = useCallback(
    (e: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement | HTMLSelectElement>) => {
      const target = e.target as HTMLInputElement;
      const name = target.name;
      const val = target.type === 'checkbox' ? target.checked : target.value;
      setState((s) => ({ form: { ...s.form, [name]: val } }));
    },
    [setState],
  );
  const setFormVal = useCallback(
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    (patch: Record<string, any>) => setState((s) => ({ form: { ...s.form, ...patch } })),
    [setState],
  );
  const setFormErrors = useCallback(
    (patch: Record<string, string>) => setState((s) => ({ formErrors: { ...s.formErrors, ...patch } })),
    [setState],
  );
  const onFile = useCallback((e: React.ChangeEvent<HTMLInputElement>, cb: (dataUrl: string) => void) => {
    const f = e.target.files && e.target.files[0];
    if (!f) return;
    const r = new FileReader();
    r.onload = () => cb(r.result as string);
    r.readAsDataURL(f);
  }, []);

  // ---------- data loaders ----------
  // Loaders report failures via toast + monitoring instead of leaving the UI
  // with a spinner or stale data. They never throw to their caller.
  const reportLoad = useCallback(
    (err: unknown) => reportActionError({ setState, toastMsg }, err, 'error.load'),
    [setState, toastMsg],
  );
  const loadNotifications = useCallback(async () => {
    try {
      const r = await api.notifications.list(S().activeTeamId!);
      setState({ notifications: r.items, notifUnread: r.unreadCount });
    } catch (err) {
      reportLoad(err);
    }
  }, [api, setState, reportLoad]);
  const afterLoginLoad = useCallback(
    async (teamId: string) => {
      setState({
        events: [],
        members: [],
        roles: [],
        news: null,
        finances: null,
        stats: null,
        polls: null,
        absences: null,
        myAbsences: null,
        notifications: null,
        eventsOnlyPending: false,
      });
      try {
        const [events, members, roles, news, notif] = await Promise.all([
          api.events.list(teamId, 'all'),
          api.members.list(teamId),
          api.roles.list(teamId),
          api.news.list(teamId),
          api.notifications.list(teamId),
        ]);
        // Discard results if the user switched to a different team while loading
        setState((s) => {
          if (s.activeTeamId !== teamId) return {};
          return { events, members, roles, news, notifications: notif.items, notifUnread: notif.unreadCount };
        });
      } catch (err) {
        if (S().activeTeamId === teamId) {
          reportLoad(err);
          setState({ error: t('error.load') });
        }
      }
    },
    [api, setState, reportLoad],
  );
  const refreshEvents = useCallback(async () => {
    try {
      const events = await api.events.list(S().activeTeamId!, 'all');
      setState({ events });
      loadNotifications();
    } catch (err) {
      reportLoad(err);
    }
  }, [api, setState, loadNotifications, reportLoad]);
  const refreshMembers = useCallback(async () => {
    try {
      const members = await api.members.list(S().activeTeamId!);
      setState({ members });
    } catch (err) {
      reportLoad(err);
    }
  }, [api, setState, reportLoad]);
  const refreshRoles = useCallback(async () => {
    try {
      const roles = await api.roles.list(S().activeTeamId!);
      setState({ roles });
    } catch (err) {
      reportLoad(err);
    }
  }, [api, setState, reportLoad]);
  const refreshTeams = useCallback(async () => {
    try {
      const teams = await api.teams.listForCurrentUser();
      setState({ teams });
    } catch (err) {
      reportLoad(err);
    }
  }, [api, setState, reportLoad]);
  const loadFinances = useCallback(async () => {
    try {
      const finances = await api.finances.overview(S().activeTeamId!);
      setState({ finances });
    } catch (err) {
      reportLoad(err);
    }
  }, [api, setState, reportLoad]);
  const loadStats = useCallback(
    async (range?: DateRange | null) => {
      try {
        const r = range !== undefined ? range : S().statsRange;
        const stats = await api.stats.teamOverview(S().activeTeamId!, r);
        setState({ stats });
      } catch (err) {
        reportLoad(err);
      }
    },
    [api, setState, reportLoad],
  );
  const loadNews = useCallback(async () => {
    try {
      const news = await api.news.list(S().activeTeamId!);
      setState({ news });
      loadNotifications();
    } catch (err) {
      reportLoad(err);
    }
  }, [api, setState, loadNotifications, reportLoad]);
  const loadPolls = useCallback(async () => {
    try {
      const polls = await api.polls.list(S().activeTeamId!);
      setState({ polls });
      loadNotifications();
    } catch (err) {
      reportLoad(err);
    }
  }, [api, setState, loadNotifications, reportLoad]);
  const loadAbsences = useCallback(async () => {
    try {
      const [absences, myAbsences] = await Promise.all([
        api.absences.listForTeam(S().activeTeamId!),
        api.absences.listMine(),
      ]);
      setState({ absences, myAbsences });
    } catch (err) {
      reportLoad(err);
    }
  }, [api, setState, reportLoad]);
  const ensureRouteData = useCallback(
    (route: Route) => {
      if (route === 'finances' && !S().finances) loadFinances();
      if (route === 'stats' && !S().stats) loadStats();
      if (route === 'news' && !S().news) loadNews();
      if (route === 'polls' && !S().polls) loadPolls();
    },
    [loadFinances, loadStats, loadNews, loadPolls],
  );

  // ---------- auth ----------
  const doLogin = useCallback(
    async (pid: string) => {
      setState({ busy: 'login:' + pid, error: null });
      try {
        await api.auth.login(pid);
        const user = await api.auth.currentUser();
        const teams = await api.teams.listForCurrentUser();
        if (!teams.length) {
          setState({ busy: null, error: t('error.login') });
          return;
        }
        const activeTeamId = teams[0].id;
        history.replaceState({ route: 'home' }, '', '/home');
        setState({ user, teams, activeTeamId, phase: 'app', busy: null, route: 'home' });
        setSentryUser(user);
        await afterLoginLoad(activeTeamId);
      } catch (err) {
        const msg = err instanceof Error ? err.message : t('error.login');
        setState({ busy: null, error: msg });
      }
    },
    [api, setState, afterLoginLoad],
  );
  const logout = useCallback(() => {
    api.auth.logout();
    setSentryUser(null);
    setState({
      phase: 'login',
      user: null,
      teams: [],
      activeTeamId: null,
      sheet: null,
      events: [],
      members: [],
      roles: [],
    });
  }, [api, setState]);

  // ---------- nav ----------
  const closeSheet = useCallback(() => {
    const s = S().sheet;
    setState({ sheet: s && s.back ? s.back : null });
  }, [setState]);
  const go = useCallback(
    (route: Route) => {
      pushRoute(route);
      setState({ route, sheet: null, eventsOnlyPending: false });
      ensureRouteData(route);
    },
    [setState, ensureRouteData],
  );
  const goEventsPending = useCallback(() => {
    pushRoute('events');
    setState({ route: 'events', sheet: null, eventsView: 'list', eventScope: 'upcoming', eventsOnlyPending: true });
    ensureRouteData('events');
  }, [setState, ensureRouteData]);
  const activePageSheet = useCallback(() => {
    let s = S().sheet;
    while (s) {
      if (isPageSheet(s.type)) return s;
      s = s.back || null;
    }
    return null;
  }, []);
  const selectTeam = useCallback(
    async (id: string) => {
      if (id === S().activeTeamId) {
        closeSheet();
        return;
      }
      pushRoute('home');
      setState({ activeTeamId: id, sheet: null, route: 'home', eventScope: 'upcoming', eventsView: 'list' });
      await afterLoginLoad(id);
    },
    [setState, closeSheet, afterLoginLoad],
  );
  const setEventsView = useCallback(
    (v: 'list' | 'calendar' | 'absences') => {
      setState({ eventsView: v });
      if (v === 'absences' && !S().absences) loadAbsences();
    },
    [setState, loadAbsences],
  );
  const toggleCalAbsences = useCallback(() => {
    const nv = !S().calShowAbsences;
    setState({ calShowAbsences: nv });
    if (nv && !S().absences) loadAbsences();
  }, [setState, loadAbsences]);

  // ---------- confirm ----------
  const askConfirm = useCallback(
    (cfg: ConfirmConfig) => setState((s) => ({ sheet: { type: 'confirm', cfg, back: s.sheet } })),
    [setState],
  );
  const cancelConfirm = useCallback(() => setState((s) => ({ sheet: (s.sheet && s.sheet.back) || null })), [setState]);
  const runConfirm = useCallback(async () => {
    const cfg = S().sheet && S().sheet!.cfg;
    setState({ sheet: null });
    if (cfg && cfg.onConfirm) await cfg.onConfirm();
  }, [setState]);

  // ---------- feature hooks ----------
  const {
    reloadDetail,
    openEventDetail,
    setMyStatus,
    setStatusFor,
    canSeeComment,
    openComment,
    submitComment,
    postEventComment,
    removeEventComment,
    toggleNomination,
    askEventAction,
    runEventAction,
    openNotifications,
    setNotifFilter,
    openMemberDetail,
    openMemberForm,
    toggleFormRole,
    saveMember,
    removeMember,
    openRoles,
    openCreateRole,
    setRolePerm,
    saveRole,
    toggleMyRole,
    openTeamSwitcher,
    openProfile,
    openMore,
    openTeamSettings,
    saveTeamPhoto,
    saveTeamLogo,
    setTeamIcon,
    toggleReasonRole,
    saveTeamSettings,
    openCreateTeam,
    createTeam,
    openInvite,
    copyInvite,
    uploadMyPhoto,
    openAbsenceForm,
    saveAbsence,
    removeAbsence,
    openCalExport,
    downloadIcs,
    copyCalUrl,
    openNewsForm,
    saveNews,
    removeNews,
    openTxForm,
    saveTx,
    deleteTx,
    openPenaltyCatalog,
    openPenaltyForm,
    savePenalty,
    deletePenaltyDef,
    openPenaltyAssign,
    savePenaltyAssign,
    deleteAssignment,
    openContribForm,
    saveContrib,
    togglePenalty,
    toggleContribution,
    setStatsRange,
    openPollForm,
    savePoll,
    togglePollOption,
    removePoll,
    openEventForm,
    saveEvent,
    toggleFormNomRole,
  } = useFeatureActions({
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
  });

  // ---------- bootstrap ----------
  useEffect(() => {
    const handler = (e: PopStateEvent) => {
      const route: Route =
        e.state && ALL_ROUTES.includes(e.state.route) ? e.state.route : routeFromPath(window.location.pathname);
      setState({ route, sheet: null });
      ensureRouteData(route);
    };
    window.addEventListener('popstate', handler);
    return () => window.removeEventListener('popstate', handler);
  }, [setState, ensureRouteData]);

  useEffect(() => {
    (async () => {
      try {
        const providers = await api.auth.providers();
        setState({ providers, phase: 'login' });
      } catch {
        setState({ phase: 'login', providers: [], error: t('error.network') });
      }
    })();
  }, [api, setState]);

  // All actions/helpers are stable (useCallback), so the actions object is built
  // once and never changes identity — this is what lets useAppActions consumers
  // avoid re-rendering on state changes.
  const actions: AppActions = useMemo(
    () => ({
      api,
      activeTeam,
      myRoles,
      can,
      isStaff,
      toastMsg,
      resetDemo,
      setPrimaryColor,
      onFormInput,
      setFormVal,
      setFormErrors,
      onFile,
      setState,
      doLogin,
      logout,
      go,
      goEventsPending,
      closeSheet,
      activePageSheet,
      selectTeam,
      setEventsView,
      toggleCalAbsences,
      openNotifications,
      setNotifFilter,
      loadAbsences,
      loadFinances,
      loadStats,
      setMyStatus,
      setStatusFor,
      canSeeComment,
      openComment,
      submitComment,
      postEventComment,
      removeEventComment,
      toggleNomination,
      askConfirm,
      cancelConfirm,
      runConfirm,
      askEventAction,
      runEventAction,
      openEventDetail,
      reloadDetail,
      openEventForm,
      saveEvent,
      toggleFormNomRole,
      openMemberDetail,
      openMemberForm,
      toggleFormRole,
      saveMember,
      removeMember,
      openRoles,
      openCreateRole,
      setRolePerm,
      saveRole,
      toggleMyRole,
      openTeamSwitcher,
      openProfile,
      openMore,
      openTeamSettings,
      saveTeamPhoto,
      saveTeamLogo,
      setTeamIcon,
      toggleReasonRole,
      saveTeamSettings,
      openCreateTeam,
      createTeam,
      openInvite,
      copyInvite,
      uploadMyPhoto,
      openAbsenceForm,
      saveAbsence,
      removeAbsence,
      openCalExport,
      downloadIcs,
      copyCalUrl,
      openNewsForm,
      saveNews,
      removeNews,
      openTxForm,
      saveTx,
      deleteTx,
      openPenaltyCatalog,
      openPenaltyForm,
      savePenalty,
      deletePenaltyDef,
      openPenaltyAssign,
      savePenaltyAssign,
      deleteAssignment,
      openContribForm,
      saveContrib,
      togglePenalty,
      toggleContribution,
      setStatsRange,
      openPollForm,
      savePoll,
      togglePollOption,
      removePoll,
    }),
    // All referenced actions are stable useCallback identities, so the object is
    // intentionally built once. Listing ~90 stable deps would add no safety.
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [],
  );

  return (
    <AppActionsContext.Provider value={actions}>
      <AppStateContext.Provider value={state}>{children}</AppStateContext.Provider>
    </AppActionsContext.Provider>
  );
}
