import React, {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useRef,
  useState,
  useSyncExternalStore,
} from 'react';
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
import { reportActionError, retryable } from '@/utils/errors';
import { captureException, setSentryUser } from '@/monitoring';
import { t } from '@/i18n';
import { useFeatureActions } from './useFeatureActions';

export type Phase = 'loading' | 'login' | 'noTeam' | 'app';
export { ALL_ROUTES, routeFromPath } from './urlState';
export type { Route } from './urlState';
import { parseLocation, buildPath, currentPath, parsePendingInvite, type Route, type UrlState } from './urlState';

/** Map the active detail sheet (walking the back-stack) to a URL detail ref. */
function detailOfSheet(sheet: SheetState | null): UrlState['detail'] {
  let s = sheet;
  while (s) {
    if (s.type === 'eventDetail' && s.eventId) return { kind: 'event', id: s.eventId };
    if (s.type === 'memberDetail' && s.membershipId) return { kind: 'member', id: s.membershipId };
    s = s.back || null;
  }
  return null;
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
  colorScheme: 'system' | 'light' | 'dark';
  user: User | null;
  teams: TeamForUser[];
  activeTeamId: string | null;
  route: Route;
  eventScope: 'upcoming' | 'past';
  eventsView: 'list' | 'calendar' | 'absences';
  eventsOnlyPending: boolean;
  calShowAbsences: boolean;
  calMonth: Date | null;
  events: TeamEvent[] | null;
  members: Member[] | null;
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
  /** Shared editing buffer for the active sheet; read typed via formValues<T>() (src/utils/forms.ts). */
  form: Record<string, unknown>;
  formErrors: Record<string, string>;
  toast: { message: string; action?: { label: string; fn: () => void } } | null;
  error: string | null;
}

function loadColorScheme(): AppState['colorScheme'] {
  const stored = localStorage.getItem('tv_color_scheme');
  if (stored === 'light' || stored === 'dark' || stored === 'system') return stored;
  return 'system';
}

const initialLocation = parseLocation(window.location.pathname, window.location.search);

const initialState: AppState = {
  phase: 'loading',
  providers: [],
  busy: null,
  primaryColor: DEFAULT_PRESET_KEY,
  colorScheme: loadColorScheme(),
  user: null,
  teams: [],
  activeTeamId: null,
  route: initialLocation.route,
  eventScope: initialLocation.eventScope,
  eventsView: initialLocation.eventsView,
  eventsOnlyPending: initialLocation.eventsOnlyPending,
  calShowAbsences: false,
  calMonth: null,
  events: null,
  members: null,
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
  finTab: initialLocation.finTab,
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
  toastMsg: (m: string, action?: { label: string; fn: () => void }) => void;
  resetDemo: () => void;
  setPrimaryColor: (c: string) => void;
  setColorScheme: (scheme: AppState['colorScheme']) => void;
  // form
  onFormInput: (e: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement | HTMLSelectElement>) => void;
  setFormVal: (patch: Record<string, unknown>) => void;
  setFormErrors: (patch: Record<string, string>) => void;
  onFile: (e: React.ChangeEvent<HTMLInputElement>, cb: (dataUrl: string) => void) => void;
  setState: (patch: Partial<AppState> | ((s: AppState) => Partial<AppState>)) => void;
  // auth
  doLogin: (pid: string) => Promise<void>;
  doPasswordLogin: (email: string, password: string) => Promise<void>;
  logout: () => void;
  deleteAccount: (confirmEmail: string) => Promise<void>;
  exportMyData: () => Promise<void>;
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
  openRoleForm: (role?: Role) => void;
  setRolePerm: (module: ModuleKey, level: PermLevel) => void;
  saveRole: () => Promise<void>;
  removeRole: (roleId: string) => void;
  toggleMyRole: (roleId: string) => Promise<void>;
  // team
  openTeamSwitcher: () => void;
  openProfile: () => void;
  openMore: () => void;
  openTeamSettings: () => void;
  saveTeamPhoto: (dataUrl: string, teamId: string) => Promise<void>;
  removeTeamPhoto: () => Promise<void>;
  saveTeamLogo: (dataUrl: string, teamId: string) => Promise<void>;
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

/** External-store handle backing useAppSelector (fine-grained subscriptions). */
interface AppStore {
  subscribe: (cb: () => void) => () => void;
  get: () => AppState;
}
const AppStoreContext = createContext<AppStore | null>(null);

export const useApp = (): AppContextValue => {
  const actions = useContext(AppActionsContext);
  const state = useContext(AppStateContext);
  if (!actions || !state) throw new Error('useApp must be used within AppProvider');
  return useMemo(() => ({ ...actions, state }), [actions, state]);
};

/**
 * Subscribe to a *slice* of state. The component re-renders only when the
 * selected value changes (Object.is), instead of on every global state update
 * like `useApp()`. Pair with `useAppActions()` for dispatch.
 *
 * The selector must return a primitive or a stable reference — returning a
 * freshly-built object/array on each call defeats the equality check and can
 * loop. Combine `useApp`'s `state` for ad-hoc reads; reach for this in hot,
 * frequently-re-rendered leaves (lists, cards) that need one or two fields.
 */
export function useAppSelector<T>(selector: (s: AppState) => T): T {
  const store = useContext(AppStoreContext);
  if (!store) throw new Error('useAppSelector must be used within AppProvider');
  const sel = useRef(selector);
  sel.current = selector;
  return useSyncExternalStore(
    store.subscribe,
    useCallback(() => sel.current(store.get()), [store]),
  );
}

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

  // Fine-grained subscription store backing useAppSelector. Listeners are
  // notified after each committed state change (see effect below).
  const listeners = useRef(new Set<() => void>());
  const store = useMemo<AppStore>(
    () => ({
      subscribe: (cb) => {
        listeners.current.add(cb);
        return () => listeners.current.delete(cb);
      },
      get: () => stateRef.current,
    }),
    [],
  );
  useEffect(() => {
    listeners.current.forEach((l) => l());
  }, [state]);

  const setState = useCallback((patch: Partial<AppState> | ((s: AppState) => Partial<AppState>)) => {
    setRaw((prev) => {
      const p = typeof patch === 'function' ? patch(prev) : patch;
      const next = { ...prev, ...p };
      stateRef.current = next;
      return next;
    });
  }, []);

  const S = useCallback(() => stateRef.current, []);

  // ---------- helpers ----------
  const activeTeam = useCallback(() => S().teams.find((t) => t.id === S().activeTeamId) || null, [S]);
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
    (m: string, action?: { label: string; fn: () => void }) => {
      if (toastTimer.current) clearTimeout(toastTimer.current);
      setState({ toast: { message: m, action } });
      toastTimer.current = setTimeout(() => setState({ toast: null }), 2600);
    },
    [setState],
  );
  const resetDemo = useCallback(() => {
    resetDemoData();
    location.reload();
  }, []);
  const setPrimaryColor = useCallback((c: string) => setState({ primaryColor: c }), [setState]);
  const setColorScheme = useCallback(
    (scheme: AppState['colorScheme']) => {
      localStorage.setItem('tv_color_scheme', scheme);
      const html = document.documentElement;
      if (scheme === 'system') {
        html.removeAttribute('data-color-scheme');
      } else {
        html.dataset.colorScheme = scheme;
      }
      setState({ colorScheme: scheme });
    },
    [setState],
  );

  // Apply persisted color scheme whenever it changes
  useEffect(() => {
    const scheme = state.colorScheme;
    if (scheme === 'system') {
      document.documentElement.removeAttribute('data-color-scheme');
    } else {
      document.documentElement.dataset.colorScheme = scheme;
    }
  }, [state.colorScheme]);

  // ---------- auth ----------
  // logout is defined early so data-loader callbacks can reference it in their
  // onAuthError handler without a TDZ issue. doLogin depends on afterLoginLoad
  // (defined below) so it stays in the data-loaders section.
  const logout = useCallback(() => {
    // The session may already be invalid (e.g. this logout was triggered by an
    // AuthError from another call), in which case the server 401s here too;
    // client-side state is cleared regardless, so a failed server-side
    // invalidation isn't actionable — just report it instead of letting it
    // surface as an unhandled rejection.
    api.auth.logout().catch((err: unknown) => {
      captureException(err, { context: 'logout' });
    });
    setSentryUser(null);
    setState({
      phase: 'login',
      user: null,
      teams: [],
      activeTeamId: null,
      sheet: null,
      events: null,
      members: null,
      roles: [],
    });
    // Providers may not have been loaded if the session was restored from a
    // cookie at startup; refresh them so the login screen has its buttons.
    api.auth
      .providers()
      .then((providers) => setState({ providers }))
      .catch((err: unknown) => {
        captureException(err, { context: 'providers-on-logout' });
      });
  }, [api, setState]);

  // GDPR Art. 17: anonymize the account then drop to the login screen. The
  // caller passes the account email as an explicit confirmation; errors (e.g. a
  // mismatched email) propagate so the UI can surface them.
  const deleteAccount = useCallback(
    async (confirmEmail: string) => {
      await api.auth.deleteAccount(confirmEmail);
      logout();
    },
    [api, logout],
  );

  // GDPR Art. 15: fetch the personal-data export and download it as JSON.
  const exportMyData = useCallback(async () => {
    const data = await api.auth.exportData();
    const blob = new Blob([JSON.stringify(data, null, 2)], { type: 'application/json' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = `teamverwaltung-datenexport-${new Date().toISOString().slice(0, 10)}.json`;
    document.body.appendChild(a);
    a.click();
    setTimeout(() => {
      document.body.removeChild(a);
      URL.revokeObjectURL(url);
    }, 1500);
  }, [api]);

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
    (patch: Record<string, unknown>) => setState((s) => ({ form: { ...s.form, ...patch } })),
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

  // ---------- idle session timeout ----------
  // After IDLE_MS of no pointer/keyboard activity the user gets a toast warning,
  // then is automatically logged out WARN_MS later.  Both timers reset on any
  // activity so normal use is never interrupted.
  const IDLE_MS = 30 * 60 * 1000;
  const WARN_MS = 2 * 60 * 1000;

  useEffect(() => {
    if (state.phase !== 'app') return;

    let warnTimer: ReturnType<typeof setTimeout>;
    let logoutTimer: ReturnType<typeof setTimeout>;

    const resetTimers = () => {
      clearTimeout(warnTimer);
      clearTimeout(logoutTimer);
      warnTimer = setTimeout(() => {
        toastMsg(t('session.idleWarning'));
        logoutTimer = setTimeout(() => {
          toastMsg(t('session.loggedOut'));
          logout();
        }, WARN_MS);
      }, IDLE_MS - WARN_MS);
    };

    const events: (keyof WindowEventMap)[] = ['pointermove', 'pointerdown', 'keydown', 'wheel', 'touchstart'];
    events.forEach((e) => window.addEventListener(e, resetTimers, { passive: true }));
    resetTimers();

    return () => {
      events.forEach((e) => window.removeEventListener(e, resetTimers));
      clearTimeout(warnTimer);
      clearTimeout(logoutTimer);
    };
    // logout and toastMsg are stable useCallbacks; state.phase is the trigger
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [state.phase]);

  // ---------- data loaders ----------
  // Loaders report failures via toast + monitoring instead of leaving the UI
  // with a spinner or stale data. They never throw to their caller.
  const reportLoad = useCallback(
    (err: unknown) => reportActionError({ setState, toastMsg, onAuthError: logout }, err, 'error.load'),
    [setState, toastMsg, logout],
  );
  const loadNotifications = useCallback(async () => {
    const teamId = S().activeTeamId!;
    try {
      const r = await api.notifications.list(teamId);
      // Discard a stale response if the user switched teams while this was
      // in flight -- otherwise the previous team's notifications would
      // clobber the newly selected team's already-cleared state.
      setState((s) => (s.activeTeamId === teamId ? { notifications: r.items, notifUnread: r.unreadCount } : {}));
    } catch (err) {
      if (S().activeTeamId === teamId) reportLoad(err);
    }
  }, [api, S, setState, reportLoad]);
  // Monotonically increasing call counter so that, if afterLoginLoad is invoked
  // again (even for the same teamId) before an earlier call has finished, only
  // the most recently invoked call's results are ever applied. The activeTeamId
  // check alone doesn't catch this: re-entering the same team rapidly (A -> B ->
  // A) means s.activeTeamId === teamId is true for both in-flight calls, so
  // whichever happened to resolve last would win instead of whichever was
  // invoked last.
  const afterLoginLoadSeq = useRef(0);
  const afterLoginLoad = useCallback(
    async (teamId: string) => {
      const seq = ++afterLoginLoadSeq.current;
      setState({
        events: null,
        members: null,
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
        // Retry on transient network failures — this is the initial-load
        // read path for the whole app, so a single dropped connection
        // shouldn't fail the entire team switch/login when a retry would
        // likely succeed. All five calls are idempotent reads.
        const [events, members, roles, news, notif] = await Promise.all([
          retryable(() => api.events.list(teamId, 'all')),
          retryable(() => api.members.list(teamId)),
          retryable(() => api.roles.list(teamId)),
          retryable(() => api.news.list(teamId)),
          retryable(() => api.notifications.list(teamId)),
        ]);
        // Discard results if the user switched to a different team while loading,
        // or if a newer afterLoginLoad call (e.g. rapid re-entry into the same
        // team) has since been invoked.
        setState((s) => {
          if (s.activeTeamId !== teamId || afterLoginLoadSeq.current !== seq) return {};
          return { events, members, roles, news, notifications: notif.items, notifUnread: notif.unreadCount };
        });
      } catch (err) {
        if (afterLoginLoadSeq.current === seq && S().activeTeamId === teamId) reportLoad(err);
        setState((s) =>
          s.activeTeamId === teamId && afterLoginLoadSeq.current === seq ? { error: t('error.load') } : {},
        );
      }
    },
    [api, S, setState, reportLoad],
  );
  const refreshEvents = useCallback(async () => {
    const teamId = S().activeTeamId!;
    try {
      const events = await retryable(() => api.events.list(teamId, 'all'));
      // Guard against a team switch completing while this was in flight --
      // without it, a slow refreshEvents for team A could clobber team B's
      // freshly-cleared state with team A's stale event list.
      setState((s) => (s.activeTeamId === teamId ? { events } : {}));
      loadNotifications();
    } catch (err) {
      if (S().activeTeamId === teamId) reportLoad(err);
    }
  }, [api, S, setState, loadNotifications, reportLoad]);
  const refreshMembers = useCallback(async () => {
    const teamId = S().activeTeamId!;
    try {
      const members = await api.members.list(teamId);
      setState((s) => (s.activeTeamId === teamId ? { members } : {}));
    } catch (err) {
      if (S().activeTeamId === teamId) reportLoad(err);
    }
  }, [api, S, setState, reportLoad]);
  const refreshRoles = useCallback(async () => {
    const teamId = S().activeTeamId!;
    try {
      const roles = await api.roles.list(teamId);
      setState((s) => (s.activeTeamId === teamId ? { roles } : {}));
    } catch (err) {
      if (S().activeTeamId === teamId) reportLoad(err);
    }
  }, [api, S, setState, reportLoad]);
  const refreshTeams = useCallback(async () => {
    try {
      const teams = await api.teams.listForCurrentUser();
      setState({ teams });
    } catch (err) {
      reportLoad(err);
    }
  }, [api, setState, reportLoad]);
  const loadFinances = useCallback(async () => {
    const teamId = S().activeTeamId!;
    try {
      const finances = await api.finances.overview(teamId);
      setState((s) => (s.activeTeamId === teamId ? { finances } : {}));
    } catch (err) {
      if (S().activeTeamId === teamId) reportLoad(err);
    }
  }, [api, S, setState, reportLoad]);
  const loadStats = useCallback(
    async (range?: DateRange | null) => {
      const teamId = S().activeTeamId!;
      try {
        const r = range !== undefined ? range : S().statsRange;
        const stats = await api.stats.teamOverview(teamId, r);
        setState((s) => (s.activeTeamId === teamId ? { stats } : {}));
      } catch (err) {
        if (S().activeTeamId === teamId) reportLoad(err);
      }
    },
    [api, S, setState, reportLoad],
  );
  const loadNews = useCallback(async () => {
    const teamId = S().activeTeamId!;
    try {
      const news = await api.news.list(teamId);
      setState((s) => (s.activeTeamId === teamId ? { news } : {}));
      loadNotifications();
    } catch (err) {
      if (S().activeTeamId === teamId) reportLoad(err);
    }
  }, [api, S, setState, loadNotifications, reportLoad]);
  const loadPolls = useCallback(async () => {
    const teamId = S().activeTeamId!;
    try {
      const polls = await api.polls.list(teamId);
      setState((s) => (s.activeTeamId === teamId ? { polls } : {}));
      loadNotifications();
    } catch (err) {
      if (S().activeTeamId === teamId) reportLoad(err);
    }
  }, [api, S, setState, loadNotifications, reportLoad]);
  const loadAbsences = useCallback(async () => {
    const teamId = S().activeTeamId!;
    try {
      const [absences, myAbsences] = await Promise.all([
        api.absences.listForTeam(teamId),
        api.absences.listMine(teamId),
      ]);
      setState((s) => (s.activeTeamId === teamId ? { absences, myAbsences } : {}));
    } catch (err) {
      if (S().activeTeamId === teamId) reportLoad(err);
    }
  }, [api, S, setState, reportLoad]);
  const ensureRouteData = useCallback(
    (route: Route) => {
      if (route === 'finances' && !S().finances) loadFinances();
      if (route === 'stats' && !S().stats) loadStats();
      if (route === 'news' && !S().news) loadNews();
      if (route === 'polls' && !S().polls) loadPolls();
    },
    [S, loadFinances, loadStats, loadNews, loadPolls],
  );

  // ---------- auth ----------
  // establishSession takes an authenticated user, loads their teams, selects the
  // active team and transitions into the app. Shared by the login flows and the
  // startup session-restore effect.
  const establishSession = useCallback(
    async (user: User | null) => {
      // An invite link (/join/<teamId>/<code>) may have brought the user here,
      // whether they already had a session (redirected straight back in) or
      // just logged in for the first time. Redeem it before loading the team
      // list so the newly joined team is included and can be auto-selected.
      const invite = parsePendingInvite(window.location.pathname);
      let joinedTeamId: string | null = null;
      if (invite) {
        try {
          const joined = await api.teams.acceptInvite(invite.code);
          joinedTeamId = joined.id;
          // Idempotent redemption: a user re-visiting/re-clicking an old
          // invite link for a team they're already in must not see a
          // "joined" toast implying a state change that didn't happen.
          if (!joined.alreadyMember) {
            toastMsg(t('team.toastJoined', { name: joined.name }));
          }
        } catch {
          toastMsg(t('team.toastInviteInvalid'));
        }
      }

      const teams = await retryable(() => api.teams.listForCurrentUser());
      setSentryUser(user);
      if (!teams.length) {
        if (invite) history.replaceState({}, '', '/');
        setState({ user, teams: [], activeTeamId: null, phase: 'noTeam', busy: null });
        return;
      }
      const activeTeamId = joinedTeamId && teams.some((tm) => tm.id === joinedTeamId) ? joinedTeamId : teams[0].id;
      history.replaceState({ route: 'home' }, '', '/home');
      setState({ user, teams, activeTeamId, phase: 'app', busy: null, route: 'home' });
      await afterLoginLoad(activeTeamId);
    },
    [api, setState, afterLoginLoad, toastMsg],
  );

  const doLogin = useCallback(
    async (pid: string) => {
      setState({ busy: 'login:' + pid, error: null });
      try {
        await api.auth.login(pid);
        const user = await api.auth.currentUser();
        await establishSession(user);
      } catch (err) {
        const msg = err instanceof Error ? err.message : t('error.login');
        setState({ busy: null, error: msg });
      }
    },
    [api, setState, establishSession],
  );

  const doPasswordLogin = useCallback(
    async (email: string, password: string) => {
      setState({ busy: 'login:password', error: null });
      try {
        await api.auth.login(email, password);
        const user = await api.auth.currentUser();
        await establishSession(user);
      } catch (err) {
        const msg = err instanceof Error ? err.message : t('error.login');
        setState({ busy: null, error: msg });
      }
    },
    [api, setState, establishSession],
  );

  // ---------- nav ----------
  const closeSheet = useCallback(() => {
    const s = S().sheet;
    setState({ sheet: s && s.back ? s.back : null });
  }, [S, setState]);
  const go = useCallback(
    (route: Route) => {
      // History is mirrored centrally by the URL-sync effect (state -> URL).
      setState({ route, sheet: null, eventsOnlyPending: false });
      ensureRouteData(route);
    },
    [setState, ensureRouteData],
  );
  const goEventsPending = useCallback(() => {
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
  }, [S]);
  const selectTeam = useCallback(
    async (id: string) => {
      if (id === S().activeTeamId) {
        closeSheet();
        return;
      }
      setState({ activeTeamId: id, sheet: null, route: 'home', eventScope: 'upcoming', eventsView: 'list' });
      await afterLoginLoad(id);
    },
    [S, setState, closeSheet, afterLoginLoad],
  );
  const setEventsView = useCallback(
    (v: 'list' | 'calendar' | 'absences') => {
      setState({ eventsView: v });
      if (v === 'absences' && !S().absences) loadAbsences();
    },
    [S, setState, loadAbsences],
  );
  const toggleCalAbsences = useCallback(() => {
    const nv = !S().calShowAbsences;
    setState({ calShowAbsences: nv });
    if (nv && !S().absences) loadAbsences();
  }, [S, setState, loadAbsences]);

  // ---------- confirm ----------
  const askConfirm = useCallback(
    (cfg: ConfirmConfig) => setState((s) => ({ sheet: { type: 'confirm', cfg, back: s.sheet } })),
    [setState],
  );
  const cancelConfirm = useCallback(() => setState((s) => ({ sheet: (s.sheet && s.sheet.back) || null })), [setState]);
  const runConfirm = useCallback(async () => {
    const cfg = S().sheet?.cfg;
    setState({ sheet: null });
    if (!cfg?.onConfirm) return;
    try {
      await cfg.onConfirm();
    } catch (err) {
      reportActionError({ setState, toastMsg, onAuthError: logout }, err);
    }
  }, [S, setState, toastMsg, logout]);

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
    openRoleForm,
    setRolePerm,
    saveRole,
    removeRole,
    toggleMyRole,
    openTeamSwitcher,
    openProfile,
    openMore,
    openTeamSettings,
    saveTeamPhoto,
    removeTeamPhoto,
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
    logout,
  });

  // ---------- routing: state <-> URL ----------
  // Last path we wrote to / read from history, so the sync effect can tell a
  // real navigation from a no-op and choose push vs. replace.
  const lastSyncedPath = useRef(currentPath());

  // URL -> state: Back/Forward reconstructs route, list filters and any open
  // detail sheet from the address bar.
  useEffect(() => {
    const handler = () => {
      const p = parseLocation(window.location.pathname, window.location.search);
      lastSyncedPath.current = currentPath();
      setState({
        route: p.route,
        eventScope: p.eventScope,
        eventsView: p.eventsView,
        eventsOnlyPending: p.eventsOnlyPending,
        finTab: p.finTab,
        sheet: null,
      });
      ensureRouteData(p.route);
      if (p.detailId && p.route === 'events') void openEventDetail(p.detailId);
      else if (p.detailId && p.route === 'members') void openMemberDetail(p.detailId);
    };
    window.addEventListener('popstate', handler);
    return () => window.removeEventListener('popstate', handler);
  }, [setState, ensureRouteData, openEventDetail, openMemberDetail]);

  // state -> URL: mirror the bookmark-relevant slice into the address bar.
  // Route changes and opening a detail sheet create history entries (so Back
  // closes the sheet / returns to the previous view); filter tweaks replace.
  useEffect(() => {
    if (state.phase !== 'app') return;
    const next: UrlState = {
      route: state.route,
      eventScope: state.eventScope,
      eventsView: state.eventsView,
      eventsOnlyPending: state.eventsOnlyPending,
      finTab: state.finTab,
      detail: detailOfSheet(state.sheet),
    };
    const target = buildPath(next);
    if (target === lastSyncedPath.current) return;
    const [prevPath, prevQuery] = lastSyncedPath.current.split('?');
    const prev = parseLocation(prevPath, prevQuery ? '?' + prevQuery : '');
    const isNavigation = prev.route !== next.route || (!prev.detailId && !!next.detail);
    if (isNavigation) history.pushState(null, '', target);
    else history.replaceState(null, '', target);
    lastSyncedPath.current = target;
  }, [
    state.phase,
    state.route,
    state.eventScope,
    state.eventsView,
    state.eventsOnlyPending,
    state.finTab,
    state.sheet,
  ]);

  const bootstrapStarted = useRef(false);
  useEffect(() => {
    // React.StrictMode double-invokes effects on initial mount in dev; without
    // this guard that means two concurrent establishSession() calls (each
    // fanning out teams.listForCurrentUser() + 5 afterLoginLoad requests) on
    // every dev-mode app load. The ref (not state) survives both invocations
    // since StrictMode replays the same component instance.
    if (bootstrapStarted.current) return;
    bootstrapStarted.current = true;
    (async () => {
      try {
        // Restore an existing session from the HttpOnly cookie. If one is active,
        // the user stays logged in across reloads without seeing the login screen.
        const user = await api.auth.currentUser();
        if (user) {
          await establishSession(user);
          return;
        }
        const providers = await api.auth.providers();
        setState({ providers, phase: 'login' });
      } catch {
        // A valid session but a failed establishSession (e.g. a transient
        // network error loading the team list, already retried once inside
        // establishSession) must still land on a *usable* login screen --
        // otherwise providers stays [] forever, Login has no SSO buttons and
        // no password-provider button to reach the password form, and the
        // user is stuck with only a manual page reload to recover.
        try {
          const providers = await api.auth.providers();
          setState({ phase: 'login', providers, error: t('error.network') });
        } catch {
          setState({ phase: 'login', providers: [], error: t('error.network') });
        }
      }
    })();
  }, [api, setState, establishSession]);

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
      setColorScheme,
      onFormInput,
      setFormVal,
      setFormErrors,
      onFile,
      setState,
      doLogin,
      doPasswordLogin,
      logout,
      deleteAccount,
      exportMyData,
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
      openRoleForm,
      setRolePerm,
      saveRole,
      removeRole,
      toggleMyRole,
      openTeamSwitcher,
      openProfile,
      openMore,
      openTeamSettings,
      saveTeamPhoto,
      removeTeamPhoto,
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
    [
      api,
      activeTeam,
      myRoles,
      can,
      isStaff,
      toastMsg,
      resetDemo,
      setPrimaryColor,
      setColorScheme,
      onFormInput,
      setFormVal,
      setFormErrors,
      onFile,
      setState,
      doLogin,
      logout,
      deleteAccount,
      exportMyData,
      doPasswordLogin,
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
      openRoleForm,
      setRolePerm,
      saveRole,
      removeRole,
      toggleMyRole,
      openTeamSwitcher,
      openProfile,
      openMore,
      openTeamSettings,
      saveTeamPhoto,
      removeTeamPhoto,
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
    ],
  );

  return (
    <AppStoreContext.Provider value={store}>
      <AppActionsContext.Provider value={actions}>
        <AppStateContext.Provider value={state}>{children}</AppStateContext.Provider>
      </AppActionsContext.Provider>
    </AppStoreContext.Provider>
  );
}
