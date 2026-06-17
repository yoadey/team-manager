/* eslint-disable @typescript-eslint/no-explicit-any */
import React, { createContext, useCallback, useContext, useEffect, useMemo, useRef, useState } from 'react';
import { api as defaultApi, resetDemoData } from '../services/serviceLayer';
import type {
  AppNotification, Absence, AttendanceRow, AttendanceStatus, DateRange, EventComment,
  FinanceOverview, Invite, Member, ModuleKey, NewsItem, PermLevel, Poll, Provider, Role,
  StatsOverview, TeamEvent, TeamForUser, User,
} from '../types';
import { DEFAULT_PRESET_KEY, hhmm, todayStr } from '../styles/tokens';
import { validateDateRange, validateEventForm, validateMoneyAmount, validatePollForm, validateRequiredText } from '../utils/validation';
import { combineDateAndTimeLocal } from '../utils/date';
import { useEventActionFeatures, useEventDetailActions } from '../features/events/hooks/useEventActions';
import { useFinanceActions } from '../features/finances/hooks/useFinanceActions';
import { usePollActions } from '../features/polls/hooks/usePollActions';

export type Phase = 'loading' | 'login' | 'app';
export type Route = 'home' | 'events' | 'members' | 'finances' | 'stats' | 'news' | 'polls' | 'team';
export interface SheetState { type: string; back?: SheetState | null; [k: string]: any; }

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
  form: Record<string, any>;
  toast: string | null;
}

const initialState: AppState = {
  phase: 'loading', providers: [], busy: null, primaryColor: DEFAULT_PRESET_KEY,
  user: null, teams: [], activeTeamId: null, route: 'home',
  eventScope: 'upcoming', eventsView: 'list', eventsOnlyPending: false,
  calShowAbsences: false, calMonth: null,
  events: [], members: [], roles: [], news: null, finances: null, stats: null,
  polls: null, absences: null, myAbsences: null,
  notifications: null, notifUnread: 0, notifFilter: 'all',
  statsRange: null, finTab: 'umsaetze', contribMonth: null,
  sheet: null, form: {}, toast: null,
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
  setFormVal: (patch: Record<string, any>) => void;
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
  askConfirm: (cfg: { title: string; message: string; confirmLabel?: string; danger?: boolean; onConfirm: () => void | Promise<void> }) => void;
  cancelConfirm: () => void;
  runConfirm: () => Promise<void>;
  // events
  askEventAction: (action: 'cancel' | 'delete' | 'reactivate', event: TeamEvent) => void;
  runEventAction: (action: 'cancel' | 'delete' | 'reactivate', event: TeamEvent, scope: 'single' | 'series') => Promise<void>;
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
  openNewsForm: () => void;
  saveNews: () => Promise<void>;
  removeNews: (id: string) => void;
  // finances
  openTxForm: (tx?: any) => void;
  saveTx: () => Promise<void>;
  deleteTx: (id: string) => Promise<void>;
  openPenaltyCatalog: () => void;
  openPenaltyForm: (p?: any) => void;
  savePenalty: () => Promise<void>;
  deletePenaltyDef: (id: string) => void;
  openPenaltyAssign: () => void;
  savePenaltyAssign: () => Promise<void>;
  deleteAssignment: (id: string) => Promise<void>;
  openContribForm: (c: any) => void;
  saveContrib: () => Promise<void>;
  togglePenalty: (id: string) => Promise<void>;
  toggleContribution: (id: string) => Promise<void>;
  setStatsRange: (range: DateRange | null) => void;
  // polls
  openPollForm: () => void;
  savePoll: () => Promise<void>;
  togglePollOption: (poll: Poll, optId: string) => void;
}

const AppContext = createContext<AppContextValue | null>(null);
export const useApp = () => {
  const ctx = useContext(AppContext);
  if (!ctx) throw new Error('useApp must be used within AppProvider');
  return ctx;
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
  const myRoles = useCallback(() => { const t = activeTeam(); return t ? t.myRoles : []; }, [activeTeam]);
  const can = useCallback((module: ModuleKey, level: PermLevel = 'write') => {
    const t = activeTeam();
    if (!t || !t.myPerms) return false;
    const p = t.myPerms[module];
    if (level === 'read') return p === 'read' || p === 'write';
    return p === 'write';
  }, [activeTeam]);
  const isStaff = useCallback(() => can('events', 'write') || can('members', 'write'), [can]);
  const toastMsg = useCallback((m: string) => {
    if (toastTimer.current) clearTimeout(toastTimer.current);
    setState({ toast: m });
    toastTimer.current = setTimeout(() => setState({ toast: null }), 2600);
  }, [setState]);
  const resetDemo = useCallback(() => { resetDemoData(); location.reload(); }, []);
  const setPrimaryColor = useCallback((c: string) => setState({ primaryColor: c }), [setState]);

  // ---------- form ----------
  const onFormInput = useCallback((e: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement | HTMLSelectElement>) => {
    const target = e.target as HTMLInputElement;
    const name = target.name;
    const val = target.type === 'checkbox' ? target.checked : target.value;
    setState((s) => ({ form: { ...s.form, [name]: val } }));
  }, [setState]);
  const setFormVal = useCallback((patch: Record<string, any>) => setState((s) => ({ form: { ...s.form, ...patch } })), [setState]);
  const onFile = useCallback((e: React.ChangeEvent<HTMLInputElement>, cb: (dataUrl: string) => void) => {
    const f = e.target.files && e.target.files[0];
    if (!f) return;
    const r = new FileReader();
    r.onload = () => cb(r.result as string);
    r.readAsDataURL(f);
  }, []);

  // ---------- data loaders ----------
  const loadNotifications = useCallback(async () => {
    const r = await api.notifications.list(S().activeTeamId!);
    setState({ notifications: r.items, notifUnread: r.unreadCount });
  }, [api, setState]);
  const afterLoginLoad = useCallback(async (teamId: string) => {
    setState({ events: [], members: [], roles: [], news: null, finances: null, stats: null, polls: null, absences: null, myAbsences: null, notifications: null, eventsOnlyPending: false });
    const [events, members, roles, news, notif] = await Promise.all([
      api.events.list(teamId, 'all'), api.members.list(teamId), api.roles.list(teamId), api.news.list(teamId), api.notifications.list(teamId),
    ]);
    setState({ events, members, roles, news, notifications: notif.items, notifUnread: notif.unreadCount });
  }, [api, setState]);
  const refreshEvents = useCallback(async () => { const events = await api.events.list(S().activeTeamId!, 'all'); setState({ events }); loadNotifications(); }, [api, setState, loadNotifications]);
  const refreshMembers = useCallback(async () => { const members = await api.members.list(S().activeTeamId!); setState({ members }); }, [api, setState]);
  const refreshRoles = useCallback(async () => { const roles = await api.roles.list(S().activeTeamId!); setState({ roles }); }, [api, setState]);
  const refreshTeams = useCallback(async () => { const teams = await api.teams.listForCurrentUser(); setState({ teams }); }, [api, setState]);
  const loadFinances = useCallback(async () => { const finances = await api.finances.overview(S().activeTeamId!); setState({ finances }); }, [api, setState]);
  const loadStats = useCallback(async (range?: DateRange | null) => { const r = range !== undefined ? range : S().statsRange; const stats = await api.stats.teamOverview(S().activeTeamId!, r); setState({ stats }); }, [api, setState]);
  const loadNews = useCallback(async () => { const news = await api.news.list(S().activeTeamId!); setState({ news }); loadNotifications(); }, [api, setState, loadNotifications]);
  const loadPolls = useCallback(async () => { const polls = await api.polls.list(S().activeTeamId!); setState({ polls }); loadNotifications(); }, [api, setState, loadNotifications]);
  const loadAbsences = useCallback(async () => { const [absences, myAbsences] = await Promise.all([api.absences.listForTeam(S().activeTeamId!), api.absences.listMine()]); setState({ absences, myAbsences }); }, [api, setState]);
  const ensureRouteData = useCallback((route: Route) => {
    if (route === 'finances' && !S().finances) loadFinances();
    if (route === 'stats' && !S().stats) loadStats();
    if (route === 'news' && !S().news) loadNews();
    if (route === 'polls' && !S().polls) loadPolls();
  }, [loadFinances, loadStats, loadNews, loadPolls]);

  // ---------- auth ----------
  const doLogin = useCallback(async (pid: string) => {
    setState({ busy: 'login:' + pid });
    await api.auth.login(pid);
    const user = await api.auth.currentUser();
    const teams = await api.teams.listForCurrentUser();
    const activeTeamId = teams[0].id;
    setState({ user, teams, activeTeamId, phase: 'app', busy: null, route: 'home' });
    await afterLoginLoad(activeTeamId);
  }, [api, setState, afterLoginLoad]);
  const logout = useCallback(() => {
    api.auth.logout();
    setState({ phase: 'login', user: null, teams: [], activeTeamId: null, sheet: null, events: [], members: [], roles: [] });
  }, [api, setState]);

  // ---------- nav ----------
  const closeSheet = useCallback(() => { const s = S().sheet; setState({ sheet: (s && s.back) ? s.back : null }); }, [setState]);
  const go = useCallback((route: Route) => { setState({ route, sheet: null, eventsOnlyPending: false }); ensureRouteData(route); }, [setState, ensureRouteData]);
  const goEventsPending = useCallback(() => { setState({ route: 'events', sheet: null, eventsView: 'list', eventScope: 'upcoming', eventsOnlyPending: true }); ensureRouteData('events'); }, [setState, ensureRouteData]);
  const activePageSheet = useCallback(() => { let s = S().sheet; while (s) { if (isPageSheet(s.type)) return s; s = s.back || null; } return null; }, []);
  const selectTeam = useCallback(async (id: string) => {
    if (id === S().activeTeamId) { closeSheet(); return; }
    setState({ activeTeamId: id, sheet: null, route: 'home', eventScope: 'upcoming', eventsView: 'list' });
    await afterLoginLoad(id);
  }, [setState, closeSheet, afterLoginLoad]);
  const setEventsView = useCallback((v: 'list' | 'calendar' | 'absences') => { setState({ eventsView: v }); if (v === 'absences' && !S().absences) loadAbsences(); }, [setState, loadAbsences]);
  const toggleCalAbsences = useCallback(() => { const nv = !S().calShowAbsences; setState({ calShowAbsences: nv }); if (nv && !S().absences) loadAbsences(); }, [setState, loadAbsences]);

  // ---------- notifications ----------
  const openNotifications = useCallback(() => {
    const teamId = S().activeTeamId;
    if (!teamId) return;

    setState({ sheet: { type: 'notifications' }, notifFilter: 'all' });

    void (async () => {
      try {
        if (!S().notifications) await loadNotifications();
        await api.notifications.markSeen(teamId);
        setState((s) => {
          if (s.activeTeamId !== teamId) return {};
          return {
            notifications: s.notifications ? s.notifications.map((n) => ({ ...n, unread: false })) : s.notifications,
            notifUnread: 0,
          };
        });
      } catch {
        toastMsg('Benachrichtigungen konnten nicht als gelesen markiert werden.');
      }
    })();
  }, [api, setState, loadNotifications, toastMsg]);
  const setNotifFilter = useCallback((f: AppState['notifFilter']) => setState({ notifFilter: f }), [setState]);

  // ---------- confirm ----------
  const askConfirm = useCallback((cfg: any) => setState((s) => ({ sheet: { type: 'confirm', cfg, back: s.sheet } })), [setState]);
  const cancelConfirm = useCallback(() => setState((s) => ({ sheet: (s.sheet && s.sheet.back) || null })), [setState]);
  const runConfirm = useCallback(async () => { const cfg = S().sheet && S().sheet!.cfg; setState({ sheet: null }); if (cfg && cfg.onConfirm) await cfg.onConfirm(); }, [setState]);

  const { reloadDetail, openEventDetail, setMyStatus, setStatusFor, canSeeComment, openComment, submitComment, postEventComment, removeEventComment, toggleNomination } = useEventDetailActions({ api, S, setState, activeTeam, myRoles, refreshEvents, setFormVal, toastMsg });

  const { askEventAction, runEventAction } = useEventActionFeatures({ api, S, setState, activeTeam, myRoles, refreshEvents, setFormVal, toastMsg, askConfirm, openEventDetail });

  // ---------- event form ----------
  const openEventForm = useCallback((event: TeamEvent | null) => {
    const f = event ? {
      id: event.id, seriesId: event.seriesId || null, type: event.type, title: event.title, date: event.date,
      meetT: hhmm(event.meetTime), startT: hhmm(event.startTime), endT: hhmm(event.endTime), location: event.location || '',
      note: event.note || '', meetTimeMandatory: !!event.meetTimeMandatory, responseMode: event.responseMode || 'opt_in',
      nominatedRoleIds: event.nominatedRoleIds || S().roles.map((r) => r.id), recurring: false, repeatWeeks: 8,
    } : {
      type: 'training', title: '', date: todayStr(), meetT: '19:15', startT: '19:30', endT: '21:30', location: 'Tanzsporthalle Eilendorf',
      note: '', meetTimeMandatory: true, responseMode: 'opt_out', nominatedRoleIds: S().roles.map((r) => r.id), recurring: false, repeatWeeks: 8,
    };
    setState((st) => ({ sheet: { type: 'eventForm', mode: event ? 'edit' : 'create', back: (st.sheet && st.sheet.type === 'eventDetail') ? st.sheet : null }, form: f }));
  }, [setState]);
  const saveEvent = useCallback(async (scope: 'single' | 'series' = 'single') => {
    const f = S().form;
    const sh = S().sheet!;
    const mode = sh.mode;
    const validation = validateEventForm(f, mode);
    if (!validation.ok) { toastMsg(validation.message!); return; }
    const back = sh.back;
    setState({ busy: 'save' });
    const payload = { type: f.type, title: f.title.trim(), date: f.date, location: f.location, note: f.note, meetTimeMandatory: f.meetTimeMandatory, responseMode: f.responseMode, meetT: f.meetT, startT: f.startT, endT: f.endT, nominatedRoleIds: f.nominatedRoleIds };
    if (mode === 'edit') await api.events.update(f.id, payload, scope);
    else await api.events.create(S().activeTeamId!, { ...payload, recurring: f.recurring, repeatWeeks: validation.value!.repeatWeeks, nominatedRoleIds: f.nominatedRoleIds });
    await refreshEvents();
    setState({ busy: null, sheet: null });
    if (mode === 'edit' && back && back.type === 'eventDetail') openEventDetail(f.id);
    toastMsg(mode === 'edit' ? (scope === 'series' ? 'Ganze Serie aktualisiert' : 'Termin aktualisiert') : 'Termin angelegt');
  }, [api, setState, refreshEvents, openEventDetail, toastMsg]);
  const toggleFormNomRole = useCallback((roleId: string) => setState((s) => { const cur = s.form.nominatedRoleIds || []; const next = cur.includes(roleId) ? cur.filter((x: string) => x !== roleId) : cur.concat(roleId); return { form: { ...s.form, nominatedRoleIds: next } }; }), [setState]);

  // ---------- members ----------
  const openMemberDetail = useCallback(async (membershipId: string) => {
    const m = S().members.find((x) => x.membershipId === membershipId);
    setState({ sheet: { type: 'memberDetail', membershipId, member: m, stats: null } });
    const stats = await api.stats.attendanceFor(S().activeTeamId!, m!.userId);
    setState((s) => (s.sheet && s.sheet.type === 'memberDetail') ? { sheet: { ...s.sheet, stats } } : {});
  }, [api, setState]);
  const openMemberForm = useCallback((member: Member) => {
    const f = { membershipId: member.membershipId, name: member.name, email: member.email, phone: member.phone, birthday: member.birthday || '', address: member.address || '', roleIds: member.roles.map((r) => r.id), group: member.group, photo: member.photo };
    setState((st) => ({ sheet: { type: 'memberForm', mode: 'edit', self: member.userId === st.user!.id, back: (st.sheet && st.sheet.type === 'memberDetail') ? st.sheet : null }, form: f }));
  }, [setState]);
  const toggleFormRole = useCallback((roleId: string) => setState((s) => { const cur = s.form.roleIds || []; const next = cur.includes(roleId) ? cur.filter((x: string) => x !== roleId) : cur.concat(roleId); return { form: { ...s.form, roleIds: next.length ? next : cur } }; }), [setState]);
  const saveMember = useCallback(async () => {
    const f = S().form;
    if (!f.name) { toastMsg('Bitte einen Namen angeben'); return; }
    const sh = S().sheet!;
    const back = sh.back; const self = sh.self;
    setState({ busy: 'save' });
    await api.members.update(f.membershipId, { name: f.name, email: f.email, phone: f.phone, birthday: f.birthday, address: f.address, roleIds: f.roleIds, group: f.group, photo: f.photo });
    await refreshMembers();
    if (self) { const u = await api.auth.currentUser(); await refreshTeams(); setState({ user: u }); }
    setState({ busy: null, sheet: null });
    if (back && back.type === 'memberDetail') openMemberDetail(f.membershipId);
    toastMsg('Profil gespeichert');
  }, [api, setState, refreshMembers, refreshTeams, openMemberDetail, toastMsg]);
  const removeMember = useCallback((membershipId: string) => {
    const m = S().members.find((x) => x.membershipId === membershipId);
    askConfirm({ title: 'Mitglied entfernen?', message: '„' + (m ? m.name : 'Das Mitglied') + '" wird aus dem Team entfernt und verliert den Zugriff. Diese Aktion kann nicht rückgängig gemacht werden.', confirmLabel: 'Entfernen', danger: true, onConfirm: async () => { await api.members.remove(membershipId); await refreshMembers(); setState({ sheet: null }); toastMsg('Mitglied entfernt'); } });
  }, [api, askConfirm, refreshMembers, setState, toastMsg]);

  // ---------- roles ----------
  const openRoles = useCallback(() => setState({ sheet: { type: 'roles' } }), [setState]);
  const openCreateRole = useCallback(() => setState((st) => ({ sheet: { type: 'roleForm', back: st.sheet }, form: { name: '', perms: { events: 'read', members: 'read', finances: 'none', news: 'read', polls: 'read', settings: 'none' } } })), [setState]);
  const setRolePerm = useCallback((module: ModuleKey, level: PermLevel) => setState((s) => ({ form: { ...s.form, perms: { ...s.form.perms, [module]: level } } })), [setState]);
  const saveRole = useCallback(async () => {
    const f = S().form;
    if (!f.name) { toastMsg('Bitte Rollennamen angeben'); return; }
    setState({ busy: 'save' });
    await api.roles.create(S().activeTeamId!, { name: f.name, permissions: f.perms });
    await refreshRoles();
    setState({ busy: null, sheet: { type: 'roles' } });
    toastMsg('Rolle angelegt');
  }, [api, setState, refreshRoles, toastMsg]);
  const toggleMyRole = useCallback(async (roleId: string) => {
    const team = activeTeam()!;
    const cur = team.myRoles.map((r) => r.id);
    const next = cur.includes(roleId) ? cur.filter((x) => x !== roleId) : cur.concat(roleId);
    if (!next.length) { toastMsg('Mindestens eine Rolle nötig'); return; }
    await api.members.setRoles(team.membershipId, next);
    await refreshTeams();
    toastMsg('Rollen aktualisiert');
  }, [api, activeTeam, refreshTeams, toastMsg]);

  // ---------- team ----------
  const openTeamSwitcher = useCallback(() => setState({ sheet: { type: 'teams' } }), [setState]);
  const openProfile = useCallback(() => { setState({ sheet: { type: 'profile' } }); api.absences.listMine().then((myAbsences) => setState({ myAbsences })); }, [api, setState]);
  const openMore = useCallback(() => setState({ sheet: { type: 'more' } }), [setState]);
  const openTeamSettings = useCallback(() => { const t = activeTeam()!; setState({ sheet: { type: 'teamSettings' }, form: { name: t.name, description: t.description || '', icon: t.icon, logo: t.logo || null, photo: t.photo, reasonRoles: (t.reasonVisibilityRoles || []).slice() } }); }, [activeTeam, setState]);
  const saveTeamPhoto = useCallback(async (dataUrl: string) => { await api.teams.updateSettings(S().activeTeamId!, { photo: dataUrl }); await refreshTeams(); setFormVal({ photo: dataUrl }); toastMsg('Gruppenbild aktualisiert'); }, [api, refreshTeams, setFormVal, toastMsg]);
  const saveTeamLogo = useCallback(async (dataUrl: string) => { await api.teams.updateSettings(S().activeTeamId!, { logo: dataUrl }); await refreshTeams(); setFormVal({ logo: dataUrl }); toastMsg('Logo aktualisiert'); }, [api, refreshTeams, setFormVal, toastMsg]);
  const setTeamIcon = useCallback((em: string) => { setFormVal({ icon: em, logo: null }); api.teams.updateSettings(S().activeTeamId!, { icon: em, logo: null }).then(() => refreshTeams()); }, [api, setFormVal, refreshTeams]);
  const toggleReasonRole = useCallback((roleId: string) => setState((s) => { const cur = s.form.reasonRoles || []; const next = cur.includes(roleId) ? cur.filter((x: string) => x !== roleId) : cur.concat(roleId); return { form: { ...s.form, reasonRoles: next } }; }), [setState]);
  const saveTeamSettings = useCallback(async () => {
    const f = S().form;
    if (!f.name || !f.name.trim()) { toastMsg('Bitte Team-Namen angeben'); return; }
    setState({ busy: 'save' });
    await api.teams.updateSettings(S().activeTeamId!, { name: f.name.trim(), description: f.description || '', reasonVisibilityRoles: f.reasonRoles || [] });
    await refreshTeams();
    setState({ busy: null });
    toastMsg('Team-Einstellungen gespeichert');
  }, [api, setState, refreshTeams, toastMsg]);
  const openCreateTeam = useCallback(() => setState({ sheet: { type: 'createTeam' }, form: { name: '', icon: '⭐', photo: null } }), [setState]);
  const createTeam = useCallback(async () => {
    const f = S().form;
    const name = validateRequiredText(f.name, 'Team-Name fehlt.');
    if (!name.ok) { toastMsg(name.message!); return; }
    setState({ busy: 'save' });
    const team = await api.teams.create({ name: name.value!, icon: f.icon, iconBg: '#1A1A1A', iconFg: '#F5C518', photo: f.photo });
    await refreshTeams();
    setState({ busy: null, sheet: null, activeTeamId: team.id, route: 'home', eventScope: 'upcoming' });
    await afterLoginLoad(team.id);
    toastMsg('Team angelegt – du bist Admin');
  }, [api, setState, refreshTeams, afterLoginLoad, toastMsg]);
  const openInvite = useCallback(async () => { setState({ sheet: { type: 'invite', invite: null } }); const invite = await api.teams.createInvite(S().activeTeamId!); setState((s) => (s.sheet && s.sheet.type === 'invite') ? { sheet: { ...s.sheet, invite } } : {}); }, [api, setState]);
  const copyInvite = useCallback(() => { const inv: Invite = S().sheet!.invite; if (!inv) return; try { navigator.clipboard.writeText(inv.link); } catch { /* ignore */ } setState((s) => ({ sheet: { ...s.sheet!, copied: true } })); toastMsg('Link kopiert'); }, [setState, toastMsg]);
  const uploadMyPhoto = useCallback(async (dataUrl: string) => { await api.auth.setPhoto(dataUrl); const user = await api.auth.currentUser(); await Promise.all([refreshTeams(), refreshMembers()]); setState({ user }); toastMsg('Profilfoto aktualisiert'); }, [api, refreshTeams, refreshMembers, setState, toastMsg]);

  // ---------- absences ----------
  const openAbsenceForm = useCallback((absence?: Absence | null) => {
    const f = absence ? { id: absence.id, from: absence.from, to: absence.to, reason: absence.reason } : { from: todayStr(), to: todayStr(), reason: 'Urlaub' };
    setState({ sheet: { type: 'absenceForm', mode: absence ? 'edit' : 'create' }, form: f });
  }, [setState]);
  const saveAbsence = useCallback(async () => {
    const f = S().form;
    const range = validateDateRange(f.from, f.to);
    if (!range.ok) { toastMsg(range.message!); return; }
    const mode = S().sheet!.mode;
    setState({ busy: 'save' });
    if (mode === 'edit') await api.absences.update(f.id, { from: range.value!.from, to: range.value!.to, reason: f.reason });
    else await api.absences.create({ from: range.value!.from, to: range.value!.to, reason: f.reason });
    await Promise.all([refreshEvents(), loadAbsences()]);
    setState({ busy: null, sheet: null });
    toastMsg(mode === 'edit' ? 'Abwesenheit aktualisiert' : 'Abwesenheit eingetragen');
  }, [api, setState, refreshEvents, loadAbsences, toastMsg]);
  const removeAbsence = useCallback((id: string) => {
    askConfirm({ title: 'Abwesenheit löschen?', message: 'Der Zeitraum wird entfernt. Automatisch gesetzte Absagen in diesem Zeitraum werden zurückgenommen.', confirmLabel: 'Löschen', danger: true, onConfirm: async () => { await api.absences.remove(id); await Promise.all([refreshEvents(), loadAbsences()]); toastMsg('Abwesenheit entfernt'); } });
  }, [api, askConfirm, refreshEvents, loadAbsences, toastMsg]);

  // ---------- calendar export ----------
  const openCalExport = useCallback(() => setState({ sheet: { type: 'calExport' } }), [setState]);
  const buildIcs = useCallback(() => {
    const team = activeTeam();
    const pad = (n: number) => String(n).padStart(2, '0');
    const fmt = (d: Date) => d.getUTCFullYear() + pad(d.getUTCMonth() + 1) + pad(d.getUTCDate()) + 'T' + pad(d.getUTCHours()) + pad(d.getUTCMinutes()) + '00Z';
    const esc = (s: string) => String(s || '').replace(/\\/g, '\\\\').replace(/\n/g, '\\n').replace(/,/g, '\\,').replace(/;/g, '\\;');
    const fold = (l: string) => (l.length <= 73 ? l : (l.match(/.{1,73}/g) || []).join('\r\n '));
    const evs = (S().events || []).filter((e) => e.status !== 'cancelled');
    const lines = ['BEGIN:VCALENDAR', 'VERSION:2.0', 'PRODID:-//Teamverwaltung//Termine//DE', 'CALSCALE:GREGORIAN', 'METHOD:PUBLISH', 'X-WR-CALNAME:' + esc(team ? team.name : 'Team'), 'X-WR-TIMEZONE:Europe/Berlin'];
    const now = new Date();
    const tMeta: Record<string, string> = { training: 'Training', auftritt: 'Auftritt / Turnier', event: 'Team-Event' };
    evs.forEach((e) => {
      const start = combineDateAndTimeLocal(e.date, hhmm(e.startTime) || hhmm(e.meetTime) || '18:00');
      const end = e.endTime ? combineDateAndTimeLocal(e.date, hhmm(e.endTime)) : new Date(start.getTime() + 2 * 3600 * 1000);
      const descParts: string[] = [];
      if (e.meetTime) descParts.push('Treffen: ' + hhmm(e.meetTime));
      if (e.note) descParts.push(e.note);
      descParts.push('Typ: ' + (tMeta[e.type] || 'Team-Event'));
      lines.push('BEGIN:VEVENT', 'UID:' + e.id + '@teamverwaltung.app', 'DTSTAMP:' + fmt(now), 'DTSTART:' + fmt(start), 'DTEND:' + fmt(end), fold('SUMMARY:' + esc(e.title)));
      if (e.location) lines.push(fold('LOCATION:' + esc(e.location)));
      lines.push(fold('DESCRIPTION:' + esc(descParts.join('\n'))), 'END:VEVENT');
    });
    lines.push('END:VCALENDAR');
    return { text: lines.join('\r\n'), count: evs.length };
  }, [activeTeam]);
  const downloadIcs = useCallback(() => {
    const team = activeTeam();
    const ics = buildIcs();
    try {
      const blob = new Blob([ics.text], { type: 'text/calendar;charset=utf-8' });
      const url = URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url; a.download = ((team && team.short) ? team.short.toLowerCase() : 'team') + '-termine.ics';
      document.body.appendChild(a); a.click();
      setTimeout(() => { document.body.removeChild(a); URL.revokeObjectURL(url); }, 1500);
      toastMsg(ics.count + ' Termine als .ics exportiert');
    } catch { toastMsg('Export nicht möglich'); }
  }, [activeTeam, buildIcs, toastMsg]);
  const copyCalUrl = useCallback(() => {
    const team = activeTeam();
    const url = 'webcal://teamverwaltung.app/cal/' + ((team && team.id) || 'team') + '.ics';
    try { navigator.clipboard.writeText(url.replace('webcal://', 'https://')); } catch { /* ignore */ }
    setState((s) => (s.sheet && s.sheet.type === 'calExport') ? { sheet: { ...s.sheet, copied: true } } : {});
    toastMsg('Abo-Link kopiert');
  }, [activeTeam, setState, toastMsg]);

  // ---------- news ----------
  const openNewsForm = useCallback(() => setState({ sheet: { type: 'newsForm' }, form: { title: '', body: '', pinned: false } }), [setState]);
  const saveNews = useCallback(async () => { const f = S().form; if (!f.title) { toastMsg('Bitte Titel angeben'); return; } setState({ busy: 'save' }); await api.news.create(S().activeTeamId!, { title: f.title, body: f.body, pinned: f.pinned }); await loadNews(); setState({ busy: null, sheet: null }); toastMsg('News veröffentlicht'); }, [api, setState, loadNews, toastMsg]);
  const removeNews = useCallback((id: string) => askConfirm({ title: 'News löschen?', message: 'Diese Neuigkeit wird dauerhaft entfernt.', confirmLabel: 'Löschen', danger: true, onConfirm: async () => { await api.news.remove(id); await loadNews(); toastMsg('News gelöscht'); } }), [api, askConfirm, loadNews, toastMsg]);

  // ---------- finances ----------
  const openTxForm = useCallback((tx?: any) => { const f = tx ? { id: tx.id, type: tx.type, title: tx.title, amount: String(tx.amount), category: tx.category } : { type: 'income', title: '', amount: '', category: 'Beiträge' }; setState({ sheet: { type: 'txForm', mode: tx ? 'edit' : 'create' }, form: f }); }, [setState]);
  const saveTx = useCallback(async () => { const f = S().form; const title = validateRequiredText(f.title, 'Bezeichnung der Buchung fehlt.'); if (!title.ok) { toastMsg(title.message!); return; } const amount = validateMoneyAmount(f.amount, { field: 'Betrag der Buchung', positive: true }); if (!amount.ok) { toastMsg(amount.message!); return; } setState({ busy: 'save' }); if (S().sheet!.mode === 'edit') await api.finances.updateTransaction(f.id, { type: f.type, title: title.value!, amount: amount.value!, category: f.category }); else await api.finances.addTransaction(S().activeTeamId!, { type: f.type, title: title.value!, amount: amount.value!, category: f.category }); await loadFinances(); setState({ busy: null, sheet: null }); toastMsg('Buchung gespeichert'); }, [api, setState, loadFinances, toastMsg]);
  const deleteTx = useCallback(async (id: string) => { await api.finances.deleteTransaction(id); await loadFinances(); setState({ sheet: null }); toastMsg('Buchung gelöscht'); }, [api, loadFinances, setState, toastMsg]);
  const openPenaltyCatalog = useCallback(() => setState({ sheet: { type: 'penaltyCatalog' } }), [setState]);
  const openPenaltyForm = useCallback((p?: any) => setState((st) => ({ sheet: { type: 'penaltyForm', mode: p ? 'edit' : 'create', back: (st.sheet && st.sheet.type === 'penaltyCatalog') ? st.sheet : null }, form: p ? { id: p.id, label: p.label, amount: String(p.amount) } : { label: '', amount: '' } })), [setState]);
  const savePenalty = useCallback(async () => { const f = S().form; const label = validateRequiredText(f.label, 'Bezeichnung der Strafe fehlt.'); if (!label.ok) { toastMsg(label.message!); return; } const amount = validateMoneyAmount(f.amount, { field: 'Betrag der Strafe', positive: true }); if (!amount.ok) { toastMsg(amount.message!); return; } const sh = S().sheet!; const back = sh.back || null; const create = sh.mode === 'create'; setState({ busy: 'save' }); if (create) await api.finances.createPenalty(S().activeTeamId!, { label: label.value!, amount: amount.value! }); else await api.finances.updatePenalty(f.id, { label: label.value!, amount: amount.value! }); await loadFinances(); setState({ busy: null, sheet: back }); toastMsg(create ? 'Strafe hinzugefügt' : 'Strafe gespeichert'); }, [api, setState, loadFinances, toastMsg]);
  const deletePenaltyDef = useCallback((id: string) => askConfirm({ title: 'Strafe entfernen?', message: 'Diese Strafe wird aus dem Katalog entfernt. Bereits erfasste Strafen bleiben erhalten.', confirmLabel: 'Entfernen', danger: true, onConfirm: async () => { await api.finances.deletePenalty(id); await loadFinances(); setState({ sheet: { type: 'penaltyCatalog' } }); toastMsg('Strafe entfernt'); } }), [api, askConfirm, loadFinances, setState, toastMsg]);
  const openPenaltyAssign = useCallback(() => { if (!S().members || !S().members.length) refreshMembers(); const f = S().finances; const first = (f && f.penalties[0]) ? f.penalties[0].id : null; setState({ sheet: { type: 'penaltyAssign' }, form: { userId: '', penaltyId: first } }); }, [refreshMembers, setState]);
  const savePenaltyAssign = useCallback(async () => { const f = S().form; if (!f.userId) { toastMsg('Bitte Person wählen'); return; } if (!f.penaltyId) { toastMsg('Bitte Strafe wählen'); return; } setState({ busy: 'save' }); await api.finances.assignPenalty(S().activeTeamId!, { userId: f.userId, penaltyId: f.penaltyId }); await loadFinances(); setState({ busy: null, sheet: null }); toastMsg('Strafe erfasst'); }, [api, setState, loadFinances, toastMsg]);
  const deleteAssignment = useCallback(async (id: string) => { await api.finances.deleteAssignment(id); await loadFinances(); toastMsg('Strafe gelöscht'); }, [api, loadFinances, toastMsg]);
  const openContribForm = useCallback((c: any) => setState({ sheet: { type: 'contribForm' }, form: { id: c.id, label: c.label, amount: String(c.amount) } }), [setState]);
  const saveContrib = useCallback(async () => { const f = S().form; const label = validateRequiredText(f.label, 'Bezeichnung des Beitrags fehlt.'); if (!label.ok) { toastMsg(label.message!); return; } const amount = validateMoneyAmount(f.amount, { field: 'Betrag des Beitrags', positive: true }); if (!amount.ok) { toastMsg(amount.message!); return; } setState({ busy: 'save' }); await api.finances.updateContribution(f.id, { label: label.value!, amount: amount.value! }); await loadFinances(); setState({ busy: null, sheet: null }); toastMsg('Beitrag gespeichert'); }, [api, setState, loadFinances, toastMsg]);
  const togglePenalty = useCallback(async (id: string) => { await api.finances.togglePenaltyPaid(id); await loadFinances(); }, [api, loadFinances]);
  const toggleContribution = useCallback(async (id: string) => { await api.finances.toggleContribution(id); await loadFinances(); }, [api, loadFinances]);
  const setStatsRange = useCallback((range: DateRange | null) => { setState({ statsRange: range, stats: null }); loadStats(range); }, [setState, loadStats]);

  // ---------- polls ----------
  const openPollForm = useCallback(() => setState({ sheet: { type: 'pollForm' }, form: { question: '', opt0: '', opt1: '', opt2: '', opt3: '', multiple: false, anonymous: false } }), [setState]);
  const votePoll = useCallback(async (pollId: string, optionIds: string[]) => { await api.polls.vote(pollId, optionIds); await loadPolls(); }, [api, loadPolls]);
  const savePoll = useCallback(async () => { const f = S().form; const poll = validatePollForm(f); if (!poll.ok) { toastMsg(poll.message!); return; } setState({ busy: 'save' }); await api.polls.create(S().activeTeamId!, { question: poll.value!.question, options: poll.value!.options, multiple: f.multiple, anonymous: f.anonymous }); await loadPolls(); setState({ busy: null, sheet: null }); toastMsg('Umfrage erstellt'); }, [api, setState, loadPolls, toastMsg]);
  const togglePollOption = useCallback((poll: Poll, optId: string) => { const cur = poll.myVote || []; let next: string[]; if (poll.multiple) next = cur.includes(optId) ? cur.filter((x) => x !== optId) : cur.concat(optId); else next = [optId]; votePoll(poll.id, next); }, [votePoll]);

  // ---------- bootstrap ----------
  useEffect(() => {
    (async () => {
      const providers = await api.auth.providers();
      setState({ providers, phase: 'login' });
    })();
  }, [api, setState]);

  const value: AppContextValue = useMemo(() => ({
    state, api, activeTeam, myRoles, can, isStaff, toastMsg, resetDemo, setPrimaryColor,
    onFormInput, setFormVal, onFile, setState,
    doLogin, logout, go, goEventsPending, closeSheet, activePageSheet, selectTeam, setEventsView, toggleCalAbsences,
    openNotifications, setNotifFilter, loadAbsences, loadFinances, loadStats,
    setMyStatus, setStatusFor, canSeeComment, openComment, submitComment, postEventComment, removeEventComment, toggleNomination,
    askConfirm, cancelConfirm, runConfirm, askEventAction, runEventAction, openEventDetail, reloadDetail, openEventForm, saveEvent, toggleFormNomRole,
    openMemberDetail, openMemberForm, toggleFormRole, saveMember, removeMember,
    openRoles, openCreateRole, setRolePerm, saveRole, toggleMyRole,
    openTeamSwitcher, openProfile, openMore, openTeamSettings, saveTeamPhoto, saveTeamLogo, setTeamIcon, toggleReasonRole, saveTeamSettings, openCreateTeam, createTeam, openInvite, copyInvite, uploadMyPhoto,
    openAbsenceForm, saveAbsence, removeAbsence,
    openCalExport, downloadIcs, copyCalUrl,
    openNewsForm, saveNews, removeNews,
    openTxForm, saveTx, deleteTx, openPenaltyCatalog, openPenaltyForm, savePenalty, deletePenaltyDef, openPenaltyAssign, savePenaltyAssign, deleteAssignment, openContribForm, saveContrib, togglePenalty, toggleContribution, setStatsRange,
    openPollForm, savePoll, togglePollOption,
  }), [state]); // eslint-disable-line react-hooks/exhaustive-deps

  return <AppContext.Provider value={value}>{children}</AppContext.Provider>;
}
