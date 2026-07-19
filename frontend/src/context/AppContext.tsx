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
import { api as defaultApi, resetDemoData } from '@/services';
import type {
  AttendanceStatus,
  DateRange,
  MemberAttendanceStats,
  ModuleKey,
  PermLevel,
  Provider,
  Role,
  TeamForUser,
  User,
} from '@/types';
import type { Absence, AttendanceRow, TeamEvent } from '@/features/events';
import type { EventFormValues } from '@/features/events/components/eventFormSchema';
import type { AbsenceFormValues } from '@/features/events/components/absenceFormSchema';
import type { Contribution, Penalty, Transaction } from '@/features/finances';
import type { ContribFormValues } from '@/features/finances/components/contribFormSchema';
import type { PenaltyFormValues } from '@/features/finances/components/penaltyFormSchema';
import type { PenaltyAssignFormValues } from '@/features/finances/components/penaltyAssignFormSchema';
import type { TxFormValues } from '@/features/finances/components/txFormSchema';
import type { Member } from '@/features/members';
import type { MemberFormValues } from '@/features/members/components/memberFormSchema';
import type { NewsItem } from '@/features/news';
import type { NewsFormValues } from '@/features/news/components/newsFormSchema';
import type { Poll } from '@/features/polls';
import type { PollFormValues } from '@/features/polls/components/pollFormSchema';
import type { RoleFormValues } from '@/features/team/components/roleFormSchema';
import type { CreateTeamFormValues } from '@/features/team/components/createTeamSchema';
import type { TeamSettingsFormValues } from '@/features/team/components/teamSettingsSchema';
import { queryKeys } from '@/query/keys';
import { useInvalidateTeamQuery } from '@/query/useInvalidateTeamQuery';
import { DEFAULT_PRESET_KEY } from '@/styles/tokens';
import { canForTeam, isStaffForTeam } from '@/utils/permissions';
import { ForbiddenError, reportActionError, retryable } from '@/utils/errors';
import { captureException, setSentryUser } from '@/monitoring';
import { t } from '@/i18n';
import { useFeatureActions } from './useFeatureActions';

export type Phase = 'loading' | 'login' | 'noTeam' | 'app';
export { ALL_ROUTES, routeFromPath } from './urlState';
export type { Route } from './urlState';
import {
  parseLocation,
  buildPath,
  currentPath,
  parsePendingInvite,
  ROUTE_MODULE,
  type Route,
  type UrlState,
} from './urlState';

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
  /** Used by the `seriesAction` confirm sheet only -- the `eventDetail` sheet
   * fetches its event via `useEventDetailQuery(eventId)` instead of carrying it. */
  event?: TeamEvent | null;
  eventId?: string;
  membershipId?: string;
  member?: Member | null;
  stats?: MemberAttendanceStats | null;
  userId?: string;
  name?: string;
  status?: AttendanceStatus;
  invite?: import('@/types').Invite | null;
  copied?: boolean;
  /** Per-sheet initial values for the form it opens with (e.g. `useForm({ defaultValues })`);
   * each sheet's own typed FormValues shape, cast at the read site. Replaces the old shared
   * `state.form` buffer -- scoped to the sheet instance instead of the whole app. */
  formInitial?: unknown;
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
  roles: Role[];
  notifFilter: 'all' | 'attendance' | 'events' | 'other';
  statsRange: DateRange | null;
  finTab: 'umsaetze' | 'strafen' | 'beitraege';
  contribMonth: string | null;
  sheet: SheetState | null;
  /**
   * `kind` defaults to 'success' when omitted (Toast.tsx) -- most of the ~70
   * toastMsg call sites across the app are success confirmations, so only
   * error paths (reportActionError) need to pass 'error' explicitly.
   */
  toast: { message: string; action?: { label: string; fn: () => void }; kind?: 'success' | 'error' } | null;
  error: string | null;
  /**
   * Per-operation mutation pending flags for the events/members/finances
   * verticals (React Query `mutation.isPending`), merged into the exposed
   * context value every render -- NOT part of the `useState` slice
   * `setState` manages, unlike every other field above. Replaces the shared
   * `busy` string for these actions so concurrent saves in different
   * verticals can't clear each other's spinner/disabled state.
   */
  savingEvent: boolean;
  savingComment: boolean;
  savingMember: boolean;
  savingTx: boolean;
  savingPenalty: boolean;
  savingPenaltyAssign: boolean;
  savingContrib: boolean;
  savingPoll: boolean;
  savingNews: boolean;
  savingAbsence: boolean;
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
  roles: [],
  notifFilter: 'all',
  statsRange: null,
  finTab: initialLocation.finTab,
  contribMonth: null,
  sheet: null,
  toast: null,
  error: null,
  // Overwritten every render by AppProvider's merged context value (see the
  // AppState doc comment above); these defaults are never actually read.
  savingEvent: false,
  savingComment: false,
  savingMember: false,
  savingTx: false,
  savingPenalty: false,
  savingPenaltyAssign: false,
  savingContrib: false,
  savingPoll: false,
  savingNews: false,
  savingAbsence: false,
};

// Photo/logo uploads (onFile) are read into a base64 data URL and sent as a
// JSON string field -- there is no server-side streaming upload path, so an
// oversized file has to be rejected client-side before FileReader reads it.
const MAX_UPLOAD_BYTES = 5 * 1024 * 1024;

const PAGE_SHEETS = ['eventDetail', 'eventForm', 'memberDetail', 'memberForm', 'teamSettings', 'roles', 'roleForm'];
export const isPageSheet = (type?: string | null) => !!type && PAGE_SHEETS.includes(type);

/**
 * Key for the ErrorBoundary wrapping a rendered sheet (AppShell/SheetHost).
 * React only resets an error boundary's caught-error state on remount (key
 * change) or an explicit reset -- keying on `sheet.type` alone means
 * switching between two DIFFERENT entities of the SAME sheet type (e.g. the
 * popstate handler going straight from eventDetail(evA) to eventDetail(evB)
 * without an intervening close, which React's automatic batching can collapse
 * into a single commit with no unmount in between) never remounts the
 * boundary, so a crash while rendering evA leaves evB stuck behind evA's
 * stale fallback. Including the entity id closes that gap.
 */
export function sheetErrorBoundaryKey(sheet: SheetState): string {
  return sheet.type + ':' + (sheet.eventId || sheet.membershipId || sheet.userId || '');
}

export interface AppContextValue {
  state: AppState;
  api: typeof defaultApi;
  // helpers
  activeTeam: () => TeamForUser | null;
  myRoles: () => Role[];
  can: (module: ModuleKey, level?: PermLevel) => boolean;
  isStaff: () => boolean;
  toastMsg: (m: string, action?: { label: string; fn: () => void }, kind?: 'success' | 'error') => void;
  resetDemo: () => void;
  setPrimaryColor: (c: string) => void;
  setColorScheme: (scheme: AppState['colorScheme']) => void;
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
  goEventsAbsences: () => void;
  closeSheet: () => void;
  activePageSheet: () => SheetState | null;
  selectTeam: (id: string) => Promise<void>;
  setEventsView: (v: 'list' | 'calendar' | 'absences') => void;
  toggleCalAbsences: () => void;
  // notifications
  openNotifications: () => void;
  setNotifFilter: (f: AppState['notifFilter']) => void;
  // attendance
  setMyStatus: (eventId: string, status: AttendanceStatus, currentReason?: string) => Promise<void>;
  setStatusFor: (e: TeamEvent, row: AttendanceRow, status: AttendanceStatus) => void;
  canSeeComment: (row: AttendanceRow) => boolean;
  openComment: (e: TeamEvent, row: { userId: string; name: string; status: AttendanceStatus; reason?: string }) => void;
  submitComment: (text: string) => Promise<void>;
  postEventComment: (eventId: string, text: string) => Promise<boolean>;
  removeEventComment: (eventId: string, cid: string) => void;
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
  openEventDetail: (eventId: string) => void;
  openEventForm: (event: TeamEvent | null) => void;
  saveEvent: (f: EventFormValues, scope?: 'single' | 'series') => Promise<void>;
  // members
  openMemberDetail: (membershipId: string) => Promise<void>;
  openMemberForm: (member: Member) => void;
  saveMember: (f: MemberFormValues) => Promise<void>;
  removeMember: (membershipId: string) => void;
  // roles
  openRoles: () => void;
  openRoleForm: (role?: Role) => void;
  saveRole: (f: RoleFormValues) => Promise<void>;
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
  saveTeamSettings: (f: TeamSettingsFormValues) => Promise<void>;
  openCreateTeam: () => void;
  createTeam: (f: CreateTeamFormValues) => Promise<void>;
  openInvite: () => Promise<void>;
  copyInvite: () => void;
  uploadMyPhoto: (dataUrl: string) => Promise<void>;
  // absences
  openAbsenceForm: (absence?: Absence | null) => void;
  saveAbsence: (f: AbsenceFormValues) => Promise<void>;
  removeAbsence: (id: string) => void;
  // calendar export
  openCalExport: () => void;
  downloadIcs: () => void;
  copyCalUrl: () => void;
  // news
  openNewsForm: (n?: NewsItem) => void;
  saveNews: (f: NewsFormValues) => Promise<void>;
  removeNews: (id: string) => void;
  // finances
  openTxForm: (tx?: Transaction) => void;
  saveTx: (f: TxFormValues) => Promise<void>;
  deleteTx: (id: string) => Promise<void>;
  openPenaltyCatalog: () => void;
  openPenaltyForm: (p?: Penalty) => void;
  savePenalty: (f: PenaltyFormValues) => Promise<void>;
  deletePenaltyDef: (id: string) => void;
  openPenaltyAssign: () => void;
  savePenaltyAssign: (f: PenaltyAssignFormValues) => Promise<void>;
  deleteAssignment: (id: string) => void;
  openContribForm: (c: Contribution) => void;
  saveContrib: (f: ContribFormValues) => Promise<void>;
  setPenaltyPaid: (id: string, paid: boolean) => Promise<void>;
  setContributionPaid: (id: string, paid: boolean) => Promise<void>;
  setStatsRange: (range: DateRange | null) => void;
  // polls
  openPollForm: () => void;
  savePoll: (f: PollFormValues) => Promise<void>;
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
  // Tracks the same merged value handed to AppStateContext (state plus the
  // per-render mutation-pending fields) so useAppSelector sees savingEvent/
  // savingComment too, not just the useState-managed slice. Updated in
  // render (see `exposedState` near the bottom), read only from effects/
  // handlers that run after render commits.
  const exposedStateRef = useRef<AppState>(initialState);
  const toastTimer = useRef<ReturnType<typeof setTimeout> | null>(null);

  // Fine-grained subscription store backing useAppSelector. Listeners are
  // notified after each committed state change (see effects below).
  const listeners = useRef(new Set<() => void>());
  const store = useMemo<AppStore>(
    () => ({
      subscribe: (cb) => {
        listeners.current.add(cb);
        return () => listeners.current.delete(cb);
      },
      get: () => exposedStateRef.current,
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
    (m: string, action?: { label: string; fn: () => void }, kind?: 'success' | 'error') => {
      if (toastTimer.current) clearTimeout(toastTimer.current);
      setState({ toast: { message: m, action, kind } });
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

  const onFile = useCallback(
    (e: React.ChangeEvent<HTMLInputElement>, cb: (dataUrl: string) => void) => {
      const f = e.target.files && e.target.files[0];
      if (!f) return;
      // The <input accept="image/*"> attribute is only a picker-UI filter
      // hint -- a user can switch to "All Files" or drag-and-drop past it,
      // so this is the only real gate against reading an arbitrary
      // non-image file (or something huge enough to freeze the tab) into a
      // base64 data URL and shipping it to the backend as a "photo"/"logo".
      if (!f.type.startsWith('image/')) {
        toastMsg(t('error.fileType'), undefined, 'error');
        e.target.value = '';
        return;
      }
      if (f.size > MAX_UPLOAD_BYTES) {
        toastMsg(t('error.fileTooLarge', { mb: MAX_UPLOAD_BYTES / (1024 * 1024) }), undefined, 'error');
        e.target.value = '';
        return;
      }
      const r = new FileReader();
      r.onload = () => cb(r.result as string);
      // Without this, a failed read (a corrupted file, a cloud-backed
      // picker file needing a network fetch that fails, a permission/
      // hardware error) leaves onload never firing and cb never called --
      // the user's "upload photo" click silently does nothing, with no
      // toast and no way to tell it failed at all.
      r.onerror = () => toastMsg(t('error.fileRead'), undefined, 'error');
      r.readAsDataURL(f);
    },
    [toastMsg],
  );

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
  // notifications itself is fetched via useNotificationsQuery (React Query,
  // consumed directly by AppShell/NotificationsSheet); `loadNotifications` is
  // kept as the SAME name/signature every other vertical's action hooks
  // already depend on (events/absences/news/polls call it after a save/vote/
  // delete that can flip a notification's read-worthy state), now
  // implemented as a cache invalidation instead of a manual fetch+setState --
  // none of those call sites needed to change when notifications itself
  // migrated.
  const invalidateNotifications = useInvalidateTeamQuery(state.activeTeamId, queryKeys.notifications);
  const loadNotifications = useCallback(() => invalidateNotifications(), [invalidateNotifications]);
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
        roles: [],
        eventsOnlyPending: false,
      });
      // events, members, finances, polls, news, absences, notifications, and
      // stats are deliberately not part of this bundle -- their respective
      // useXQuery hooks refetch on their own the moment activeTeamId (part of
      // their query key) changes. roles is the only server-state fetch left
      // here.
      //
      // Retry on transient network failures — this is the initial-load read
      // path for the whole app, so a single dropped connection shouldn't
      // fail the entire team switch/login when a retry would likely
      // succeed. This is an idempotent read.
      try {
        const roles = await retryable(() => api.roles.list(teamId));
        // Discard the result if the user switched to a different team while
        // loading, or if a newer afterLoginLoad call (e.g. rapid re-entry
        // into the same team) has since been invoked. Uses the functional
        // updater form (reading `s`, the state React actually commits this
        // update against) rather than S() -- S() reads a ref that isn't
        // guaranteed to reflect an activeTeamId change made earlier in this
        // same async call chain until React actually processes that pending
        // update, which can still be outstanding at this point.
        setState((s) => (s.activeTeamId === teamId && afterLoginLoadSeq.current === seq ? { roles } : {}));
      } catch (err) {
        // A ForbiddenError here just means the caller's role has roles:none --
        // an entirely ordinary, expected state, not a real failure. Surfacing
        // it as a toast on every single login/team switch for such a role
        // would be a false alarm; only report a genuine (non-permission)
        // failure.
        if (!(err instanceof ForbiddenError) && afterLoginLoadSeq.current === seq && S().activeTeamId === teamId) {
          reportLoad(err);
        }
      }
    },
    [api, S, setState, reportLoad],
  );
  // Each loader below gets its own monotonic call-sequence ref, mirroring
  // afterLoginLoadSeq above: the activeTeamId check alone only guards
  // against a TEAM SWITCH completing while a call is in flight, not against
  // two same-team refreshes of the SAME loader racing each other (e.g.
  // saving one role then quickly deleting another, both of which call
  // refreshRoles). If the network responds out of request order, an
  // unguarded loader would apply whichever response happened to arrive
  // last, silently reverting to stale data even though a newer request was
  // already in flight.
  //
  // events and members have no such loader here -- they're fetched via
  // useEventsQuery/useMembersQuery (React Query), whose team-scoped key
  // makes this class of race structurally impossible instead of needing a
  // manual sequence guard.
  const refreshRolesSeq = useRef(0);
  const refreshRoles = useCallback(async () => {
    const teamId = S().activeTeamId!;
    const seq = ++refreshRolesSeq.current;
    try {
      const roles = await api.roles.list(teamId);
      setState((s) => (s.activeTeamId === teamId && refreshRolesSeq.current === seq ? { roles } : {}));
    } catch (err) {
      if (S().activeTeamId === teamId && refreshRolesSeq.current === seq) reportLoad(err);
    }
  }, [api, S, setState, reportLoad]);
  // Unlike its sibling loaders above, refreshTeams has no activeTeamId scope
  // (teams are user-, not team-, scoped) -- but it still needs the same
  // monotonic-sequence guard, and for the same reason: it's invoked from
  // many independent, unserialized call sites (useTeamActions' saveTeamPhoto/
  // saveTeamLogo/setTeamIcon/removeTeamPhoto/saveTeamSettings/createTeam/
  // uploadMyPhoto, and useRoleActions' toggleMyRole), each guarded against
  // racing ITSELF (an in-flight key, or toggleMyRole's own chain) but not
  // against racing each other -- e.g. uploading a team photo while toggling
  // one's own role in a different sheet fires two concurrent refreshTeams()
  // calls. Without this, an out-of-order response applies whichever call
  // happened to resolve last, silently reverting the other's change (e.g. a
  // just-toggled own role, which feeds can()/myRoles() via activeTeam(),
  // appearing to un-toggle itself) until the next unrelated refresh.
  const refreshTeamsSeq = useRef(0);
  const refreshTeams = useCallback(async () => {
    const seq = ++refreshTeamsSeq.current;
    try {
      const teams = await api.teams.listForCurrentUser();
      if (refreshTeamsSeq.current === seq) setState({ teams });
    } catch (err) {
      if (refreshTeamsSeq.current === seq) reportLoad(err);
    }
  }, [api, setState, reportLoad]);
  // finances/stats have no such loader here -- they're fetched via
  // useFinanceOverviewQuery/useStatsQuery (React Query), whose team-scoped
  // (and, for stats, also range-scoped) key makes this class of race
  // structurally impossible instead of needing a manual sequence guard (same
  // as events/members).
  //
  // news, polls, and absences have no such loader here either -- they're
  // fetched via useNewsQuery/usePollsQuery/useAbsencesQuery (React Query),
  // for the same reason. Each old loader's paired loadNotifications()
  // refresh now lives in ensureRouteData below (news/polls) or directly in
  // the mutation's own action (absences, whose data only ever changes via
  // its own save/delete, never merely by navigating to it).
  const ensureRouteData = useCallback(
    (route: Route) => {
      // events, members, finances, polls, news, absences, and stats have no
      // data-fetch branch here: they're fetched by useEventsQuery/
      // useMembersQuery/useFinanceOverviewQuery/usePollsQuery/useNewsQuery/
      // useAbsencesQuery/useStatsQuery, which retry/refetch on their own.
      //
      // Skip entirely for a module the caller can't read (nav already hides
      // these routes, but a stale bookmark/URL or browser back/forward can
      // still land here) -- RouteScreen renders Home instead of the real
      // page for that case anyway, so fetching would just be a wasted 403
      // that reportLoad would then surface as a spurious forbidden toast.
      const module = ROUTE_MODULE[route];
      if (module && !can(module, 'read')) return;
      // polls'/news' own lists are fetched by usePollsQuery/useNewsQuery, but
      // the pre-migration loadPolls/loadNews loaders also refreshed
      // notifications on every (re-)navigation here -- e.g. a new poll/news
      // item clears its own "pending" notification once the user has viewed
      // it -- so that pairing stays, just without the list-fetch half.
      if (route === 'polls' || route === 'news') loadNotifications();
    },
    [can, loadNotifications],
  );

  // ---------- auth ----------
  // establishSession takes an authenticated user, loads their teams, selects the
  // active team and transitions into the app. Shared by the login flows and the
  // startup session-restore effect.
  const establishSession = useCallback(
    async (user: User | null, opts?: { restoreLocation?: boolean }) => {
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
          toastMsg(t('team.toastInviteInvalid'), undefined, 'error');
        }
      }

      const teams = await retryable(() => api.teams.listForCurrentUser());
      setSentryUser(user);
      if (!teams.length) {
        if (invite) history.replaceState({}, '', '/');
        setState({ user, teams: [], activeTeamId: null, phase: 'noTeam', busy: null });
        return null;
      }
      const activeTeamId = joinedTeamId && teams.some((tm) => tm.id === joinedTeamId) ? joinedTeamId : teams[0].id;
      if (opts?.restoreLocation) {
        // Session restore (a page reload, or a bookmarked/shared deep link
        // like /finances or /events?view=absences) must not silently bounce
        // the user to /home. Re-parse the URL fresh here (same pattern as
        // parsePendingInvite above) rather than trust the module-load-time
        // initialLocation/initialState snapshot, then fetch that route's
        // data the same way the popstate handler below does -- afterLoginLoad
        // only covers roles (events/members/finances/polls/news/absences/
        // notifications fetch via their own query hooks), so without this a
        // deep link into stats would render the right route with no data,
        // ever (the same "permanent skeleton loader" class events/members
        // no longer need a manual fix for). Left as `detail: null` here
        // deliberately --
        // the caller opens the detail sheet (see the bootstrap effect
        // below), and the state->URL sync effect restores the id segment
        // once state.sheet is set, so there's no need to duplicate that
        // logic here.
        const restored = parseLocation(window.location.pathname, window.location.search);
        history.replaceState(
          { route: restored.route },
          '',
          buildPath({
            route: restored.route,
            eventScope: restored.eventScope,
            eventsView: restored.eventsView,
            eventsOnlyPending: restored.eventsOnlyPending,
            finTab: restored.finTab,
            detail: null,
          }),
        );
        setState({
          user,
          teams,
          activeTeamId,
          phase: 'app',
          busy: null,
          route: restored.route,
          eventScope: restored.eventScope,
          eventsView: restored.eventsView,
          eventsOnlyPending: restored.eventsOnlyPending,
          finTab: restored.finTab,
        });
        await afterLoginLoad(activeTeamId);
        ensureRouteData(restored.route);
        return restored;
      }
      history.replaceState({ route: 'home' }, '', '/home');
      setState({ user, teams, activeTeamId, phase: 'app', busy: null, route: 'home' });
      await afterLoginLoad(activeTeamId);
      return null;
    },
    [api, setState, afterLoginLoad, toastMsg, ensureRouteData],
  );

  const doLogin = useCallback(
    async (pid: string) => {
      const owner = 'login:' + pid;
      setState({ busy: owner, error: null });
      try {
        await api.auth.login(pid);
        const user = await api.auth.currentUser();
        await establishSession(user);
      } catch (err) {
        const msg = err instanceof Error ? err.message : t('error.login');
        // Guard against a different, still-in-flight login (Login.tsx
        // normally prevents this by disabling every control while any login
        // is busy, but a defensive owner-check here costs nothing and keeps
        // this consistent with every other busy-setting flow in the app).
        if (S().busy === owner) setState({ busy: null, error: msg });
        else setState({ error: msg });
      }
    },
    [api, S, setState, establishSession],
  );

  const doPasswordLogin = useCallback(
    async (email: string, password: string) => {
      const owner = 'login:password';
      setState({ busy: owner, error: null });
      try {
        await api.auth.login(email, password);
        const user = await api.auth.currentUser();
        await establishSession(user);
      } catch (err) {
        const msg = err instanceof Error ? err.message : t('error.login');
        if (S().busy === owner) setState({ busy: null, error: msg });
        else setState({ error: msg });
      }
    },
    [api, S, setState, establishSession],
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
  // Dedicated nav action (rather than a raw setState) so an absence
  // notification's "jump to the Events > Absences tab" click goes through
  // ensureRouteData like every other route change, keeping this route
  // change consistent with how every other route arrives at the page
  // (permission pre-check below, plus events/absences data now lives in
  // useEventsQuery's/useAbsencesQuery's own React Query cache rather than
  // gating on state.events/state.absences here). RouteScreen itself bounces
  // to Home if the caller can't read events (a stale absence notification,
  // cached from before a permission downgrade, is the one way this route is
  // reachable without events:read), so EventAbsences never mounts -- and
  // thus never fetches -- in that case either.
  const goEventsAbsences = useCallback(() => {
    setState({ route: 'events', sheet: null, eventsView: 'absences' });
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
  // Absences data is now fetched by useAbsencesQuery directly in
  // EventAbsences/EventCalendar (enabled while the absences tab/overlay is
  // actually shown), so switching the view/toggle here is a pure state
  // update -- no imperative fetch trigger needed.
  const setEventsView = useCallback((v: 'list' | 'calendar' | 'absences') => setState({ eventsView: v }), [setState]);
  const toggleCalAbsences = useCallback(() => {
    setState((s) => ({ calShowAbsences: !s.calShowAbsences }));
  }, [setState]);

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
    saveMember,
    removeMember,
    openRoles,
    openRoleForm,
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
    setPenaltyPaid,
    setContributionPaid,
    setStatsRange,
    openPollForm,
    savePoll,
    togglePollOption,
    removePoll,
    openEventForm,
    saveEvent,
    savingEvent,
    savingComment,
    savingMember,
    savingTx,
    savingPenalty,
    savingPenaltyAssign,
    savingContrib,
    savingPoll,
    savingNews,
    savingAbsence,
  } = useFeatureActions({
    api,
    S,
    setState,
    activeTeam,
    myRoles,
    teamId: state.activeTeamId,
    refreshRoles,
    refreshTeams,
    loadNotifications,
    afterLoginLoad,
    toastMsg,
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
          const restored = await establishSession(user, { restoreLocation: true });
          // Mirrors the popstate handler below: a deep link into a specific
          // event/member detail sheet (e.g. /events/ev1) must survive a
          // page reload too, not just the route/list-filter portion of the
          // URL restoreLocation already restores above.
          if (restored?.detailId && restored.route === 'events') void openEventDetail(restored.detailId);
          else if (restored?.detailId && restored.route === 'members') void openMemberDetail(restored.detailId);
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
  }, [api, setState, establishSession, openEventDetail, openMemberDetail]);

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
      onFile,
      setState,
      doLogin,
      doPasswordLogin,
      logout,
      deleteAccount,
      exportMyData,
      go,
      goEventsPending,
      goEventsAbsences,
      closeSheet,
      activePageSheet,
      selectTeam,
      setEventsView,
      toggleCalAbsences,
      openNotifications,
      setNotifFilter,
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
      openEventForm,
      saveEvent,
      openMemberDetail,
      openMemberForm,
      saveMember,
      removeMember,
      openRoles,
      openRoleForm,
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
      setPenaltyPaid,
      setContributionPaid,
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
      onFile,
      setState,
      doLogin,
      logout,
      deleteAccount,
      exportMyData,
      doPasswordLogin,
      go,
      goEventsPending,
      goEventsAbsences,
      closeSheet,
      activePageSheet,
      selectTeam,
      setEventsView,
      toggleCalAbsences,
      openNotifications,
      setNotifFilter,
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
      openEventForm,
      saveEvent,
      openMemberDetail,
      openMemberForm,
      saveMember,
      removeMember,
      openRoles,
      openRoleForm,
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
      setPenaltyPaid,
      setContributionPaid,
      setStatsRange,
      openPollForm,
      savePoll,
      togglePollOption,
      removePoll,
    ],
  );

  // Per-operation mutation pending flags (React Query `isPending`) merged in
  // fresh every render -- see the AppState doc comment. Unlike every other
  // field on `state`, these never go through `setState` -- memoized so the
  // merged object's identity (and thus AppStateContext's value, and thus
  // every useApp() consumer's re-render) only changes when state or one of
  // these flags actually changes, not on every unrelated AppProvider render.
  const exposedState = useMemo<AppState>(
    () => ({
      ...state,
      savingEvent,
      savingComment,
      savingMember,
      savingTx,
      savingPenalty,
      savingPenaltyAssign,
      savingContrib,
      savingPoll,
      savingNews,
      savingAbsence,
    }),
    [
      state,
      savingEvent,
      savingComment,
      savingMember,
      savingTx,
      savingPenalty,
      savingPenaltyAssign,
      savingContrib,
      savingPoll,
      savingNews,
      savingAbsence,
    ],
  );
  exposedStateRef.current = exposedState;
  useEffect(() => {
    listeners.current.forEach((l) => l());
  }, [
    savingEvent,
    savingComment,
    savingMember,
    savingTx,
    savingPenalty,
    savingPenaltyAssign,
    savingContrib,
    savingPoll,
    savingNews,
    savingAbsence,
  ]);

  return (
    <AppStoreContext.Provider value={store}>
      <AppActionsContext.Provider value={actions}>
        <AppStateContext.Provider value={exposedState}>{children}</AppStateContext.Provider>
      </AppActionsContext.Provider>
    </AppStoreContext.Provider>
  );
}
