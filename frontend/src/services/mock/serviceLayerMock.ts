// =============================================================================
// serviceLayerMock.ts — Mock-Backend für die Teamverwaltungs-App
// -----------------------------------------------------------------------------
// Faithful TypeScript port of the prototype's serviceLayer.js. Simulates the
// future Go/PostgreSQL backend as an async API with artificial latency and
// localStorage persistence. Replace method bodies with HTTP calls later; the
// signatures (the API contract) stay the same.
// =============================================================================

/* eslint-disable @typescript-eslint/no-explicit-any */
import { mapAttendanceDtoToRow, mapEventDtoToTeamEvent, mapMemberDtoToMember } from '../mappers';
import type {
  AttendanceStatus,
  DateRange,
  Invite,
  Membership,
  ModuleKey,
  Permissions,
  PermLevel,
  Provider,
  ReasonVisibility,
  Role,
  StatsOverview,
  Team,
  TeamForUser,
  User,
} from '@/types';
import type { Absence, AttendanceDto, AttendanceRow, EventComment, EventDto, TeamEvent } from '@/features/events';
import type { Contribution, FinanceOverview, Penalty, PenaltyAssignment, Transaction } from '@/features/finances';
import type { Member, MemberDto } from '@/features/members';
import type { NewsItem } from '@/features/news';
import type { AppNotification, NotificationsResult } from '@/features/notifications';
import type { Poll, PollOptionDto, PollVoteDto } from '@/features/polls';
import { config } from '@/config';
import { createSeedData, type DemoDb } from '@/demo/seedData';
import { formatDateOnly, parseDateOnlyLocal, todayLocalDate } from '@/utils/date';

const rid = (p: string) => p + '_' + Math.random().toString(36).slice(2, 9);
const delay = (min = 120, max = 320) => new Promise<void>((r) => setTimeout(r, min + Math.random() * (max - min)));
const clone = <T>(x: T): T => JSON.parse(JSON.stringify(x));
const DAY = 86400000;
const iso = (d: Date) => d.toISOString();
function atTime(_base: Date | string, h: number, m: number) {
  return String(h).padStart(2, '0') + ':' + String(m).padStart(2, '0');
}

// ---- Rechte-Modell ----------------------------------------------------------
const MODULES: ModuleKey[] = ['events', 'members', 'finances', 'news', 'polls', 'settings'];
function perms(
  events: PermLevel,
  members: PermLevel,
  finances: PermLevel,
  news: PermLevel,
  polls: PermLevel,
  settings: PermLevel,
): Permissions {
  return { events, members, finances, news, polls, settings };
}
const LEVEL: Record<string, number> = { none: 0, read: 1, write: 2 };
function mergePerms(roles: Role[]): Permissions {
  const out = perms('none', 'none', 'none', 'none', 'none', 'none');
  roles.forEach((r) =>
    MODULES.forEach((m) => {
      if (LEVEL[r.permissions[m]] > LEVEL[out[m]]) out[m] = r.permissions[m];
    }),
  );
  return out;
}
function defaultRoles(teamId: string): Role[] {
  return [
    {
      id: rid('role'),
      teamId,
      name: 'Admin / Trainer',
      system: true,
      color: '#1565C0',
      permissions: perms('write', 'write', 'write', 'write', 'write', 'write'),
    },
    {
      id: rid('role'),
      teamId,
      name: 'Tänzer / Mitglied',
      system: true,
      color: '#5B6470',
      permissions: perms('read', 'read', 'read', 'read', 'read', 'none'),
    },
    {
      id: rid('role'),
      teamId,
      name: 'Kassenwart',
      system: true,
      color: '#2E7D32',
      permissions: perms('read', 'read', 'write', 'read', 'read', 'none'),
    },
    {
      id: rid('role'),
      teamId,
      name: 'Teamkapitän',
      system: true,
      color: '#E8910C',
      permissions: perms('write', 'read', 'read', 'write', 'write', 'none'),
    },
    {
      id: rid('role'),
      teamId,
      name: 'Betreuer',
      system: true,
      color: '#7A4FB6',
      permissions: perms('read', 'read', 'none', 'read', 'read', 'none'),
    },
  ];
}

// ---- Persistenz (tagesfrisch) ----------------------------------------------
function todayKey() {
  return config.storageKeyPrefix + 'v7_' + todayLocalDate();
}
function loadDb(): DemoDb {
  try {
    const raw = localStorage.getItem(todayKey());
    if (raw) return JSON.parse(raw);
  } catch {
    /* ignore */
  }
  const db = createSeedData();
  save(db);
  return db;
}
function save(db: DemoDb) {
  try {
    localStorage.setItem(todayKey(), JSON.stringify(db));
  } catch {
    /* ignore */
  }
}
let DB = loadDb();
function persist() {
  save(DB);
}
// persist() always overwrites the whole stored blob with this tab's
// in-memory snapshot -- without this listener, a stale tab's next mutation
// (however unrelated) would silently clobber/resurrect data written by
// another tab in the meantime, since the two never otherwise communicate.
// Reloading DB in-place from the fresh value on every cross-tab write closes
// that window for all but genuinely simultaneous edits in both tabs, which
// this mock backend (unlike the real API) has no way to serialize anyway.
if (typeof window !== 'undefined') {
  window.addEventListener('storage', (e) => {
    if (e.key !== todayKey()) return;
    try {
      DB = e.newValue ? JSON.parse(e.newValue) : createSeedData();
    } catch {
      /* ignore a malformed write from another tab */
    }
  });
}
function pushNotif(o: Partial<AppNotification>) {
  DB.notifications.push(Object.assign({ id: rid('ntf'), createdAt: iso(new Date()) }, o) as AppNotification);
}
const session: { userId: string | null } = { userId: null };

export function resetDemoData() {
  try {
    Object.keys(localStorage)
      .filter((k) => k.startsWith(config.storageKeyPrefix))
      .forEach((k) => localStorage.removeItem(k));
  } catch {
    /* ignore */
  }
}

const PROVIDERS: Provider[] = [
  { id: 'vereins-sso', name: 'Vereins-SSO', sub: 'TSC Schwarz-Gelb', glyph: 'SG', bg: '#1A1A1A', fg: '#F5C518' },
  {
    id: 'google',
    name: 'Google',
    sub: 'Mit Google fortfahren',
    glyph: 'G',
    bg: '#FFFFFF',
    fg: '#4285F4',
    border: true,
  },
  {
    id: 'microsoft',
    name: 'Microsoft',
    sub: 'Entra ID / Microsoft 365',
    glyph: 'M',
    bg: '#FFFFFF',
    fg: '#5E5E5E',
    border: true,
  },
  { id: 'apple', name: 'Apple', sub: 'Mit Apple anmelden', glyph: '', bg: '#000000', fg: '#FFFFFF' },
];

// ---- interne Helfer ---------------------------------------------------------
function rolesOf(membership: Membership): Role[] {
  return membership.roleIds.map((id) => DB.roles.find((r) => r.id === id)).filter(Boolean) as Role[];
}
function primaryRole(roles: Role[]): Role | null {
  const score = (r: Role) => MODULES.reduce((s, m) => s + LEVEL[r.permissions[m]], 0);
  return [...roles].sort((a, b) => score(b) - score(a))[0] || null;
}
function absenceCovers(userId: string, date: string) {
  return DB.absences.some((a) => a.userId === userId && date >= a.from && date <= a.to);
}
// Matches stats.Repository's `COUNT(*) FILTER (WHERE a.status IN
// ('yes','no','maybe'))` — the attendance-stats denominator excludes both
// 'pending' and 'not_nominated', unlike the general-purpose "nominated"
// count used elsewhere (events._withSummary) which only excludes
// 'not_nominated'.
function isCountedStatus(status: string): boolean {
  return status === 'yes' || status === 'no' || status === 'maybe';
}
// Matches stats.Service.defaultDateRange's `now.AddDate(0, -3, 0)` — 3
// calendar months before `dateStr`, not literally 90 days.
function threeMonthsBeforeLocal(dateStr: string): string {
  const d = parseDateOnlyLocal(dateStr);
  d.setMonth(d.getMonth() - 3);
  return formatDateOnly(d);
}
function effectiveStatus(event: EventDto, userId: string | null) {
  const rec = DB.attendance.find((a) => a.eventId === event.id && a.userId === userId);
  if (rec)
    return {
      status: rec.status as AttendanceStatus,
      reason: rec.reason,
      reasonId: rec.reasonId,
      reasonVisibility: rec.reasonVisibility,
      auto: false,
      absent: absenceCovers(userId!, event.date),
    };
  if (userId && absenceCovers(userId, event.date))
    // Matches events.computeEffectiveAttendance (backend/internal/events/attendance.go):
    // no reason/reasonId/reasonVisibility is synthesized here -- the UI already
    // renders a dedicated, localized "absent" badge (events.absent) for
    // Absent=true instead of a raw reason field, so hardcoding a single-locale
    // string here would both diverge from the real backend and show German
    // text regardless of the active locale.
    return {
      status: 'no' as AttendanceStatus,
      reason: '',
      reasonId: null,
      reasonVisibility: null,
      auto: true,
      absent: true,
    };
  if (event.responseMode === 'opt_out')
    return {
      status: 'yes' as AttendanceStatus,
      reason: '',
      reasonId: null,
      reasonVisibility: null,
      auto: true,
      absent: false,
    };
  return {
    status: 'pending' as AttendanceStatus,
    reason: '',
    reasonId: null,
    reasonVisibility: null,
    auto: false,
    absent: false,
  };
}
const STATUS_ORDER: Record<string, number> = { yes: 0, maybe: 1, pending: 2, no: 3, not_nominated: 4 };
// Operates on the raw EventDto: nominations are persisted on the stored record
// (id/teamId/nominatedRoleIds), independent of the UI TeamEvent enrichment.
function applyNominations(event: EventDto, nominatedRoleIds: string[]) {
  event.nominatedRoleIds = [...nominatedRoleIds];
  const nomSet = new Set(nominatedRoleIds);
  const members = DB.memberships.filter((m) => m.teamId === event.teamId);
  members.forEach((m) => {
    const nominated = m.roleIds.some((roleId) => nomSet.has(roleId));
    const a = DB.attendance.find((x) => x.eventId === event.id && x.userId === m.userId);
    if (nominated) {
      if (a && a.status === 'not_nominated') DB.attendance = DB.attendance.filter((x) => x !== a);
      return;
    }
    if (!a) {
      DB.attendance.push({
        id: rid('att'),
        eventId: event.id,
        userId: m.userId,
        status: 'not_nominated',
        reason: '',
        reasonId: null,
        reasonVisibility: null,
        at: iso(new Date()),
      });
    } else if (a.status === 'not_nominated') {
      a.reason = '';
      a.reasonId = null;
      a.reasonVisibility = null;
      a.at = iso(new Date());
    }
  });
}

// =============================================================================
export const SERVICE_ENDPOINTS = {
  // auth: thin wrappers for future authentication/session endpoints.
  auth: ['auth.providers', 'auth.login', 'auth.currentUser', 'auth.logout', 'auth.setPhoto'],
  // teams: thin wrappers for team CRUD and invite endpoints; listForCurrentUser is enriched with role permissions.
  teams: [
    'teams.listForCurrentUser',
    'teams.get',
    'teams.create',
    'teams.updateSettings',
    'teams.createInvite',
    'teams.acceptInvite',
  ],
  // members: list returns Member ViewModels mapped from MemberDto plus derived primaryRole/perms.
  members: ['members.list', 'members.update', 'members.setRoles', 'members.remove'],
  // roles: direct RoleDto CRUD endpoints.
  roles: ['roles.list', 'roles.create', 'roles.update', 'roles.remove'],
  // events: list/get map EventDto to TeamEvent with client-side summary/myStatus aggregation.
  events: [
    'events.list',
    'events.get',
    'events.create',
    'events.update',
    'events.setStatus',
    'events.remove',
    'events.listComments',
    'events.addComment',
    'events.removeComment',
  ],
  // attendance: set/setNomination are endpoint-shaped mutations; listForEvent maps AttendanceDto to display rows.
  attendance: ['attendance.listForEvent', 'attendance.set', 'attendance.setNomination'],
  // absences/news/polls/notifications: endpoint-shaped resource operations with small display enrichments.
  absences: ['absences.listForTeam', 'absences.listMine', 'absences.create', 'absences.update', 'absences.remove'],
  news: ['news.list', 'news.create', 'news.remove'],
  polls: ['polls.list', 'polls.vote', 'polls.create'],
  notifications: ['notifications.list', 'notifications.markSeen'],
  // finances.overview is currently a client aggregation over finance collections; mutation methods map to future endpoints.
  finances: [
    'finances.overview',
    'finances.addTransaction',
    'finances.updateTransaction',
    'finances.deleteTransaction',
    'finances.updatePenalty',
    'finances.deletePenalty',
    'finances.addPenaltyAssignment',
    'finances.togglePenaltyPaid',
    'finances.updateContribution',
  ],
  // stats.overview is a client-side aggregation until the backend exposes a reporting endpoint.
  stats: ['stats.overview'],
} as const;

export const CLIENT_AGGREGATIONS = [
  'events._withSummary',
  'members.list',
  'attendance.listForEvent',
  'finances.overview',
  'stats.overview',
] as const;

export const API_ENDPOINT_METHODS = Object.values(SERVICE_ENDPOINTS).flat();

export const mockApi = {
  auth: {
    async providers(): Promise<Provider[]> {
      await delay(80, 200);
      return clone(PROVIDERS);
    },
    async login(providerId: string, _password?: string) {
      await delay(380, 720);
      session.userId = 'u1';
      return {
        token: 'mock.jwt.' + rid('tk'),
        provider: providerId,
        user: clone(DB.users.find((u) => u.id === 'u1')!),
      };
    },
    async currentUser(): Promise<User | null> {
      await delay(50, 120);
      return session.userId ? clone(DB.users.find((u) => u.id === session.userId)!) : null;
    },
    async logout() {
      session.userId = null;
    },
    // GDPR Art. 15: a minimal personal-data export for the current mock user.
    async exportData(): Promise<unknown> {
      await delay(80, 200);
      const u = DB.users.find((x) => x.id === session.userId);
      return {
        exportedAt: new Date().toISOString(),
        profile: u ? clone(u) : null,
        memberships: DB.memberships.filter((m) => m.userId === session.userId).map(clone),
      };
    },
    async setPhoto(dataUrl: string): Promise<User> {
      await delay(150, 300);
      const u = DB.users.find((x) => x.id === session.userId)!;
      u.photo = dataUrl;
      persist();
      return clone(u);
    },
    // GDPR Art. 17 erasure by anonymization: overwrite the current user's PII
    // and end the session (mirrors the backend's DELETE /auth/me behavior).
    async deleteAccount(_confirmEmail?: string): Promise<void> {
      await delay(150, 300);
      const u = DB.users.find((x) => x.id === session.userId);
      if (u) {
        u.name = 'Gelöschtes Mitglied';
        u.email = `deleted+${u.id}@invalid`;
        u.phone = '';
        u.birthday = '';
        u.address = '';
        u.photo = null;
        persist();
      }
      session.userId = null;
    },
  },

  teams: {
    async listForCurrentUser(): Promise<TeamForUser[]> {
      await delay();
      return DB.memberships
        .filter((m) => m.userId === session.userId)
        .map((m) => {
          const t = DB.teams.find((x) => x.id === m.teamId)!;
          const roles = rolesOf(m);
          return Object.assign(clone(t), {
            myRoles: clone(roles),
            myPerms: mergePerms(roles),
            membershipId: m.id,
            memberCount: DB.memberships.filter((x) => x.teamId === t.id).length,
          });
        });
    },
    async get(teamId: string): Promise<Team> {
      await delay(80, 160);
      return clone(DB.teams.find((t) => t.id === teamId)!);
    },
    async create({
      name,
      icon,
      iconBg,
      iconFg,
      photo,
    }: {
      name: string;
      icon?: string;
      iconBg?: string;
      iconFg?: string;
      photo?: string | null;
    }): Promise<Team> {
      await delay(340, 600);
      const team: Team = {
        id: rid('t'),
        name,
        short: (name || 'T').trim().charAt(0).toUpperCase(),
        icon: icon || '⭐',
        iconBg: iconBg || '#1565C0',
        iconFg: iconFg || '#FFFFFF',
        photo: photo || null,
        logo: null,
        description: '',
      };
      DB.teams.push(team);
      const roles = defaultRoles(team.id);
      DB.roles.push(...roles);
      const admin = roles.find((r) => r.name === 'Admin / Trainer')!;
      team.reasonVisibilityRoles = [admin.id];
      DB.memberships.push({
        id: rid('mem'),
        teamId: team.id,
        userId: session.userId!,
        roleIds: [admin.id],
        group: '',
        joinedAt: iso(new Date()),
      });
      persist();
      return clone(team);
    },
    async updateSettings(teamId: string, patch: Partial<Team>): Promise<Team> {
      await delay();
      const t = DB.teams.find((x) => x.id === teamId)!;
      Object.assign(t, patch);
      persist();
      return clone(t);
    },
    async createInvite(teamId: string): Promise<Invite> {
      await delay(180, 360);
      const code = Math.random().toString(36).slice(2, 8).toUpperCase();
      const inv: Invite = {
        id: rid('inv'),
        teamId,
        code,
        // Matches serviceLayerReal.ts's `${publicBaseURL}/join/{teamId}/{code}`
        // shape (teams.Service.CreateInvite on the real backend), since
        // acceptInvite's URL-parsing on the frontend expects that format
        // regardless of which service layer generated the link.
        link: 'https://teamverwaltung.app/join/' + teamId + '/' + code,
        createdAt: iso(new Date()),
        expiresAt: iso(new Date(Date.now() + 7 * DAY)),
      };
      DB.invites.push(inv);
      persist();
      return clone(inv);
    },

    async acceptInvite(code: string): Promise<TeamForUser & { alreadyMember: boolean }> {
      await delay(180, 360);
      const inv = DB.invites.find((i) => i.code === code);
      if (!inv || new Date(inv.expiresAt).getTime() <= Date.now()) {
        throw new Error('invite not found or expired');
      }

      const existing = DB.memberships.find((m) => m.teamId === inv.teamId && m.userId === session.userId);
      const alreadyMember = !!existing;
      if (!existing) {
        const memberRole = DB.roles.find((r) => r.teamId === inv.teamId && r.name === 'Tänzer / Mitglied');
        DB.memberships.push({
          id: rid('mem'),
          teamId: inv.teamId,
          userId: session.userId!,
          roleIds: memberRole ? [memberRole.id] : [],
          group: '',
          joinedAt: iso(new Date()),
        });
        persist();
      }

      const t = DB.teams.find((x) => x.id === inv.teamId)!;
      const m = DB.memberships.find((x) => x.teamId === inv.teamId && x.userId === session.userId)!;
      const roles = rolesOf(m);
      return Object.assign(clone(t), {
        myRoles: clone(roles),
        myPerms: mergePerms(roles),
        membershipId: m.id,
        memberCount: DB.memberships.filter((x) => x.teamId === t.id).length,
        alreadyMember,
      });
    },
  },

  members: {
    async list(teamId: string): Promise<Member[]> {
      await delay();
      return DB.memberships
        .filter((m) => m.teamId === teamId)
        .map((m) => {
          const u = DB.users.find((x) => x.id === m.userId)!;
          const roles = rolesOf(m);
          const dto: MemberDto = {
            membershipId: m.id,
            userId: u.id,
            name: u.name,
            email: u.email,
            phone: u.phone,
            birthday: u.birthday || '',
            address: u.address || '',
            avatarColor: u.avatarColor,
            photo: u.photo,
            group: m.group,
            roles: clone(roles),
            joinedAt: m.joinedAt,
          };
          return mapMemberDtoToMember(dto, clone(primaryRole(roles)), mergePerms(roles));
        })
        .sort((a, b) => a.name.localeCompare(b.name, 'de'));
    },
    async update(
      membershipId: string,
      {
        name,
        email,
        phone,
        birthday,
        address,
        group,
      }: {
        name?: string;
        email?: string;
        phone?: string;
        birthday?: string;
        address?: string;
        group?: string;
      },
      _teamId: string,
    ) {
      await delay(220, 420);
      const m = DB.memberships.find((x) => x.id === membershipId)!;
      const u = DB.users.find((x) => x.id === m.userId)!;
      if (name !== undefined) u.name = name;
      if (email !== undefined) u.email = email;
      if (phone !== undefined) u.phone = phone;
      if (birthday !== undefined) u.birthday = birthday;
      if (address !== undefined) u.address = address;
      if (group !== undefined) m.group = group;
      persist();
      return true;
    },
    async setRoles(membershipId: string, roleIds: string[], _teamId: string) {
      await delay(140, 300);
      const m = DB.memberships.find((x) => x.id === membershipId)!;
      if (roleIds.length) m.roleIds = roleIds;
      persist();
      return true;
    },
    async remove(membershipId: string, _teamId: string) {
      await delay(200, 400);
      DB.memberships = DB.memberships.filter((x) => x.id !== membershipId);
      persist();
      return true;
    },
  },

  roles: {
    async list(teamId: string): Promise<Role[]> {
      await delay(90, 200);
      return clone(DB.roles.filter((r) => r.teamId === teamId));
    },
    async create(
      teamId: string,
      { name, color, permissions }: { name: string; color?: string; permissions?: Permissions },
    ): Promise<Role> {
      await delay(240, 440);
      const r: Role = {
        id: rid('role'),
        teamId,
        name,
        system: false,
        // Matches api/map.ts's mapRole fallback (r.color ?? '#888888') — the
        // real backend stores an omitted color as NULL and that's the
        // default applied on read, so the mock must agree or a role created
        // without an explicit color (RoleFormSheet has no color picker)
        // renders a different color depending on which backend is active.
        color: color || '#888888',
        permissions: permissions || perms('read', 'read', 'none', 'read', 'read', 'none'),
      };
      DB.roles.push(r);
      persist();
      return clone(r);
    },
    async update(roleId: string, patch: Partial<Role>, _teamId: string): Promise<Role> {
      await delay(180, 360);
      const r = DB.roles.find((x) => x.id === roleId)!;
      Object.assign(r, patch);
      persist();
      return clone(r);
    },
    async remove(roleId: string, _teamId: string) {
      await delay(180, 360);
      DB.roles = DB.roles.filter((x) => x.id !== roleId);
      persist();
      return true;
    },
  },

  events: {
    async list(teamId: string, scope: 'all' | 'upcoming' | 'past' = 'all'): Promise<TeamEvent[]> {
      await delay();
      const today = todayLocalDate();
      let list = DB.events.filter((e) => e.teamId === teamId);
      if (scope === 'upcoming') list = list.filter((e) => e.date >= today);
      if (scope === 'past') list = list.filter((e) => e.date < today);
      list = list.sort((a, b) => (scope === 'past' ? b.date.localeCompare(a.date) : a.date.localeCompare(b.date)));
      return list.map((e) => this._withSummary(e, teamId));
    },
    async get(eventId: string, _teamId: string): Promise<TeamEvent | null> {
      await delay(110, 220);
      const e = DB.events.find((x) => x.id === eventId);
      return e ? this._withSummary(e, e.teamId) : null;
    },
    _withSummary(e: EventDto, teamId: string): TeamEvent {
      const memberIds = DB.memberships.filter((m) => m.teamId === teamId).map((m) => m.userId);
      let yes = 0,
        no = 0,
        maybe = 0,
        pending = 0,
        notNom = 0;
      memberIds.forEach((uid) => {
        const s = effectiveStatus(e, uid).status;
        if (s === 'yes') yes++;
        else if (s === 'no') no++;
        else if (s === 'maybe') maybe++;
        else if (s === 'not_nominated') notNom++;
        else pending++;
      });
      const mine = effectiveStatus(e, session.userId);
      const nominated = memberIds.length - notNom;
      return mapEventDtoToTeamEvent(
        clone(e),
        { yes, no, maybe, pending, notNominated: notNom, nominated, total: memberIds.length },
        { myStatus: mine.status, myAuto: mine.auto, myReason: mine.reason },
      );
    },
    async create(teamId: string, payload: any): Promise<TeamEvent> {
      await delay(340, 600);
      const created: EventDto[] = [];
      const base = Object.assign(
        {
          teamId,
          recurring: false,
          // Matches the real backend: CreateEventRequest.meetTimeMandatory is
          // optional, and internal/events/repository.go's boolVal(nil) stores
          // `false` when the field is omitted (see also api/map.ts's
          // `e.meetTimeMandatory ?? false` read-side fallback). The event
          // creation UI (useEventFormActions.ts) always sends an explicit
          // boolean, so this default only bites callers that omit the field
          // (e.g. serviceLayer.test.ts's `api.events.create(..., { type,
          // title, date })` calls) — but it must still agree with the real
          // backend, or such a caller sees a different value depending on
          // which backend is active.
          meetTimeMandatory: false,
          location: '',
          note: '',
          responseMode: 'opt_in',
          status: 'active',
        },
        payload,
      );
      if (payload.recurring && payload.repeatWeeks > 1) {
        const seriesId = rid('series');
        for (let w = 0; w < payload.repeatWeeks; w++) {
          const d = parseDateOnlyLocal(payload.date);
          d.setDate(d.getDate() + w * 7);
          created.push(
            this._mk(Object.assign({}, base, { date: formatDateOnly(d), seriesId, recurring: true }), payload),
          );
        }
      } else {
        created.push(this._mk(base, payload));
      }
      created.forEach((e) => DB.events.push(e));
      pushNotif({
        teamId,
        type: 'event_created',
        actorId: session.userId!,
        title: created[0].title,
        eventId: created[0].id,
        eventTitle: created[0].title,
        eventDate: created[0].date,
        note: created.length > 1 ? 'Serie mit ' + created.length + ' Terminen' : '',
      });
      if (Array.isArray(payload.nominatedRoleIds)) {
        created.forEach((e) => applyNominations(e, payload.nominatedRoleIds));
      }
      persist();
      return this._withSummary(created[0], teamId);
    },
    // Builds a persistable EventDto (not the enriched TeamEvent ViewModel) so the
    // result can be pushed straight into the DB.events collection.
    _mk(base: any, payload: any): EventDto {
      const mk = (h: string) => (h ? atTime(base.date, +h.slice(0, 2), +h.slice(3, 5)) : null);
      const nominatedRoleIds = Array.isArray(payload.nominatedRoleIds) ? [...payload.nominatedRoleIds] : undefined;
      return {
        id: rid('ev'),
        teamId: base.teamId,
        type: base.type,
        title: base.title,
        date: base.date,
        location: base.location,
        note: base.note || '',
        meetTime: payload.meetT ? mk(payload.meetT) : null,
        startTime: payload.startT ? mk(payload.startT) : null,
        endTime: payload.endT ? mk(payload.endT) : null,
        meetTimeMandatory: !!base.meetTimeMandatory,
        responseMode: base.responseMode || 'opt_in',
        nominatedRoleIds,
        recurring: !!base.recurring,
        seriesId: base.seriesId || null,
        status: 'active',
      } as EventDto;
    },
    async update(eventId: string, patch: any, scope: 'single' | 'series', _teamId: string): Promise<TeamEvent> {
      await delay(260, 480);
      const e = DB.events.find((x) => x.id === eventId)!;
      const targets = scope === 'series' && e.seriesId ? DB.events.filter((x) => x.seriesId === e.seriesId) : [e];
      targets.forEach((ev) => {
        const baseDate = scope !== 'series' && patch.date !== undefined ? patch.date : ev.date;
        const mk = (h: string) => (h ? atTime(baseDate, +h.slice(0, 2), +h.slice(3, 5)) : null);
        if (scope !== 'series' && patch.date !== undefined) ev.date = patch.date;
        (['type', 'title', 'location', 'note', 'meetTimeMandatory', 'responseMode'] as const).forEach((k) => {
          if (patch[k] !== undefined) (ev as any)[k] = patch[k];
        });
        if (patch.meetT !== undefined) ev.meetTime = patch.meetT ? mk(patch.meetT) : null;
        if (patch.startT !== undefined) ev.startTime = patch.startT ? mk(patch.startT) : null;
        if (patch.endT !== undefined) ev.endTime = patch.endT ? mk(patch.endT) : null;
        if (Array.isArray(patch.nominatedRoleIds)) applyNominations(ev, patch.nominatedRoleIds);
      });
      pushNotif({
        teamId: e.teamId,
        type: 'event_updated',
        actorId: session.userId!,
        title: e.title,
        eventId: e.id,
        eventTitle: e.title,
        eventDate: e.date,
        note: scope === 'series' ? 'ganze Serie' : '',
      });
      persist();
      return this._withSummary(e, e.teamId);
    },
    async setStatus(eventId: string, status: 'active' | 'cancelled', scope: 'single' | 'series', _teamId: string) {
      await delay(180, 360);
      const e = DB.events.find((x) => x.id === eventId);
      if (!e) return false;
      const targets = scope === 'series' && e.seriesId ? DB.events.filter((x) => x.seriesId === e.seriesId) : [e];
      targets.forEach((ev) => {
        ev.status = status;
      });
      pushNotif({
        teamId: e.teamId,
        type: status === 'cancelled' ? 'event_cancelled' : 'event_reactivated',
        actorId: session.userId!,
        title: e.title,
        eventId: e.id,
        eventTitle: e.title,
        eventDate: e.date,
        note: scope === 'series' ? 'ganze Serie' : '',
      });
      persist();
      return true;
    },
    async remove(eventId: string, scope: 'single' | 'series', _teamId: string) {
      await delay(200, 400);
      const e = DB.events.find((x) => x.id === eventId);
      const ids =
        e && scope === 'series' && e.seriesId
          ? DB.events.filter((x) => x.seriesId === e.seriesId).map((x) => x.id)
          : [eventId];
      if (e)
        pushNotif({
          teamId: e.teamId,
          type: 'event_deleted',
          actorId: session.userId!,
          title: e.title,
          eventTitle: e.title,
          eventDate: e.date,
          note: scope === 'series' ? 'ganze Serie' : '',
        });
      DB.events = DB.events.filter((x) => !ids.includes(x.id));
      DB.attendance = DB.attendance.filter((a) => !ids.includes(a.eventId));
      DB.eventComments = DB.eventComments.filter((c) => !ids.includes(c.eventId));
      persist();
      return true;
    },
    async listComments(eventId: string, _teamId: string): Promise<EventComment[]> {
      await delay(110, 220);
      return DB.eventComments
        .filter((c) => c.eventId === eventId)
        .sort((a, b) => a.createdAt.localeCompare(b.createdAt))
        .map((c) => {
          const u = DB.users.find((x) => x.id === c.userId);
          return Object.assign(clone(c), {
            name: u ? u.name : '',
            color: u ? u.avatarColor : '#888',
            photo: u ? u.photo : null,
          });
        });
    },
    async addComment(eventId: string, text: string, _teamId: string): Promise<EventComment> {
      await delay(160, 300);
      const c: EventComment = { id: rid('cm'), eventId, userId: session.userId!, text, createdAt: iso(new Date()) };
      DB.eventComments.push(c);
      persist();
      return clone(c);
    },
    async removeComment(id: string, _eventId: string, _teamId: string) {
      await delay(140, 260);
      DB.eventComments = DB.eventComments.filter((x) => x.id !== id);
      persist();
      return true;
    },
  },

  attendance: {
    async listForEvent(eventId: string, _teamId: string): Promise<AttendanceRow[]> {
      await delay(130, 260);
      const e = DB.events.find((x) => x.id === eventId)!;
      const members = DB.memberships.filter((m) => m.teamId === e.teamId);
      const rows: AttendanceRow[] = members.map((m) => {
        const u = DB.users.find((x) => x.id === m.userId)!;
        const roles = rolesOf(m);
        const es = effectiveStatus(e, m.userId);
        const dto: AttendanceDto = {
          id: es.reasonId || `${eventId}:${u.id}`,
          eventId,
          userId: u.id,
          status: es.status,
          reason: es.reason,
          reasonId: es.reasonId,
          reasonVisibility: es.reasonVisibility,
        };
        return mapAttendanceDtoToRow(dto, {
          userId: u.id,
          name: u.name,
          avatarColor: u.avatarColor,
          photo: u.photo,
          group: m.group,
          primaryRole: clone(primaryRole(roles)),
          auto: es.auto,
          absent: es.absent,
        });
      });
      rows.sort((a, b) => STATUS_ORDER[a.status] - STATUS_ORDER[b.status] || a.name.localeCompare(b.name, 'de'));
      return rows;
    },
    async set(
      eventId: string,
      userId: string,
      {
        status,
        reason,
        reasonId,
        reasonVisibility,
      }: { status: AttendanceStatus; reason?: string; reasonId?: string | null; reasonVisibility?: ReasonVisibility },
      _teamId: string,
    ) {
      await delay(160, 320);
      let a = DB.attendance.find((x) => x.eventId === eventId && x.userId === userId);
      if (!a) {
        a = { id: rid('att'), eventId, userId, status, reason: '', reasonId: null, reasonVisibility: null };
        DB.attendance.push(a);
      }
      a.status = status;
      a.reason = reason || '';
      a.reasonId = reasonId || null;
      a.reasonVisibility = reasonVisibility || null;
      a.at = iso(new Date());
      const e = DB.events.find((x) => x.id === eventId);
      if (e && (status === 'yes' || status === 'no' || status === 'maybe'))
        pushNotif({
          teamId: e.teamId,
          type: 'attendance',
          actorId: userId,
          status,
          eventId: e.id,
          eventTitle: e.title,
          eventDate: e.date,
        });
      persist();
      return clone(a);
    },
    async setNomination(eventId: string, userId: string, nominated: boolean, _teamId: string) {
      await delay(140, 280);
      if (nominated) {
        DB.attendance = DB.attendance.filter((x) => !(x.eventId === eventId && x.userId === userId));
      } else {
        let a = DB.attendance.find((x) => x.eventId === eventId && x.userId === userId);
        if (!a) {
          a = {
            id: rid('att'),
            eventId,
            userId,
            status: 'not_nominated',
            reason: '',
            reasonId: null,
            reasonVisibility: null,
          };
          DB.attendance.push(a);
        }
        a.status = 'not_nominated';
        a.reason = '';
        a.reasonId = null;
      }
      persist();
      return true;
    },
  },

  absences: {
    async listForTeam(teamId: string): Promise<Absence[]> {
      await delay(120, 240);
      const memberIds = DB.memberships.filter((m) => m.teamId === teamId).map((m) => m.userId);
      return DB.absences
        .filter((a) => memberIds.includes(a.userId))
        .map((a) => {
          const u = DB.users.find((x) => x.id === a.userId)!;
          const m = DB.memberships.find((x) => x.teamId === teamId && x.userId === a.userId)!;
          const pr = primaryRole(rolesOf(m));
          return Object.assign(clone(a), {
            name: u.name,
            avatarColor: u.avatarColor,
            photo: u.photo,
            roleColor: pr ? pr.color : '#888',
            roleName: pr ? pr.name : '',
          });
        })
        .sort((a, b) => a.from.localeCompare(b.from));
    },
    async listMine(_teamId: string): Promise<Absence[]> {
      await delay(80, 180);
      return clone(DB.absences.filter((a) => a.userId === session.userId)).sort((a, b) => a.from.localeCompare(b.from));
    },
    async create({
      from,
      to,
      reason,
      userId,
    }: {
      teamId: string;
      from: string;
      to: string;
      reason?: string;
      userId: string;
    }): Promise<Absence> {
      await delay(220, 420);
      const uid = userId;
      const a: Absence = {
        id: rid('abs'),
        userId: uid,
        from,
        to,
        reason: reason || '',
        createdAt: iso(new Date()),
      };
      DB.absences.push(a);
      const mem = DB.memberships.find((m) => m.userId === uid);
      if (mem) pushNotif({ teamId: mem.teamId, type: 'absence', actorId: uid, title: a.reason });
      persist();
      return clone(a);
    },
    async update(
      id: string,
      { from, to, reason }: { from?: string; to?: string; reason?: string },
      _teamId: string,
    ): Promise<Absence> {
      await delay(180, 360);
      const a = DB.absences.find((x) => x.id === id)!;
      if (a) {
        if (from !== undefined) a.from = from;
        if (to !== undefined) a.to = to;
        if (reason !== undefined) a.reason = reason;
      }
      persist();
      return clone(a);
    },
    async remove(id: string, _teamId: string) {
      await delay(160, 300);
      DB.absences = DB.absences.filter((x) => x.id !== id);
      persist();
      return true;
    },
  },

  news: {
    async list(teamId: string): Promise<NewsItem[]> {
      await delay(100, 220);
      return clone(DB.news.filter((n) => n.teamId === teamId))
        .map((n) => {
          const a = DB.users.find((u) => u.id === n.authorId);
          return Object.assign(n, {
            authorName: a ? a.name : '',
            authorColor: a ? a.avatarColor : '#888',
            authorPhoto: a ? a.photo : null,
          });
        })
        .sort((a, b) => Number(b.pinned) - Number(a.pinned) || b.createdAt.localeCompare(a.createdAt));
    },
    async create(
      teamId: string,
      { title, body, pinned }: { title: string; body: string; pinned?: boolean },
    ): Promise<NewsItem> {
      await delay(260, 480);
      const n: NewsItem = {
        id: rid('news'),
        teamId,
        title,
        body,
        authorId: session.userId!,
        pinned: !!pinned,
        createdAt: iso(new Date()),
      };
      DB.news.push(n);
      pushNotif({ teamId, type: 'news', actorId: session.userId!, title });
      persist();
      return clone(n);
    },
    async update(
      id: string,
      { title, body, pinned }: { title: string; body: string; pinned?: boolean },
      _teamId: string,
    ): Promise<boolean> {
      await delay(200, 360);
      const n = DB.news.find((x) => x.id === id);
      if (!n) throw new Error('News not found');
      n.title = title;
      n.body = body;
      n.pinned = !!pinned;
      persist();
      return true;
    },
    async remove(id: string, _teamId: string) {
      await delay(180, 340);
      DB.news = DB.news.filter((x) => x.id !== id);
      persist();
      return true;
    },
  },

  finances: {
    async overview(teamId: string): Promise<FinanceOverview> {
      await delay(160, 320);
      const tx = DB.transactions.filter((x) => x.teamId === teamId).sort((a, b) => b.date.localeCompare(a.date));
      const income = tx.filter((t) => t.type === 'income').reduce((s, t) => s + t.amount, 0);
      const expense = tx.filter((t) => t.type === 'expense').reduce((s, t) => s + t.amount, 0);
      const penalties = clone(DB.penalties.filter((p) => p.teamId === teamId));
      const assignments = DB.penaltyAssignments
        .filter((p) => p.teamId === teamId)
        .map((p) => {
          const u = DB.users.find((x) => x.id === p.userId)!;
          const pen = DB.penalties.find((x) => x.id === p.penaltyId)!;
          return Object.assign(clone(p), {
            name: u.name,
            avatarColor: u.avatarColor,
            photo: u.photo,
            label: pen.label,
            amount: pen.amount,
          });
        });
      const openByUser: Record<string, number> = {};
      assignments
        .filter((a) => !a.paid)
        .forEach((a) => {
          openByUser[a.userId] = (openByUser[a.userId] || 0) + (a.amount || 0);
        });
      const openPenalties = Object.keys(openByUser)
        .map((uid) => {
          const u = DB.users.find((x) => x.id === uid)!;
          return { userId: uid, name: u.name, avatarColor: u.avatarColor, photo: u.photo, amount: openByUser[uid] };
        })
        .sort((a, b) => b.amount - a.amount);
      const contributions = DB.contributions
        .filter((c) => c.teamId === teamId)
        .map((c) => {
          const u = DB.users.find((x) => x.id === c.userId)!;
          return Object.assign(clone(c), { name: u.name, avatarColor: u.avatarColor, photo: u.photo });
        })
        .sort((a, b) => a.name!.localeCompare(b.name!, 'de'));
      const contribOpen = contributions.filter((c) => c.status === 'open').length;
      return {
        balance: income - expense,
        income,
        expense,
        transactions: clone(tx),
        penalties,
        assignments,
        openPenalties,
        openPenaltySum: Object.values(openByUser).reduce((s, v) => s + v, 0),
        contributions,
        contribOpen,
      };
    },
    async addTransaction(
      teamId: string,
      {
        type,
        title,
        amount,
        category,
      }: { type: 'income' | 'expense'; title: string; amount: number | string; category?: string },
    ): Promise<Transaction> {
      await delay(240, 440);
      const t: Transaction = {
        id: rid('tx'),
        teamId,
        type,
        title,
        amount: Number(amount),
        date: todayLocalDate(),
        category: category || '',
      };
      DB.transactions.push(t);
      persist();
      return clone(t);
    },
    async updateTransaction(
      id: string,
      {
        type,
        title,
        amount,
        category,
      }: { type?: 'income' | 'expense'; title?: string; amount?: number | string; category?: string },
      _teamId: string,
    ): Promise<Transaction> {
      await delay(180, 360);
      const t = DB.transactions.find((x) => x.id === id)!;
      if (t) {
        if (type !== undefined) t.type = type;
        if (title !== undefined) t.title = title;
        if (amount !== undefined) t.amount = Number(amount);
        if (category !== undefined) t.category = category;
      }
      persist();
      return clone(t);
    },
    async deleteTransaction(id: string, _teamId: string) {
      await delay(160, 300);
      DB.transactions = DB.transactions.filter((x) => x.id !== id);
      persist();
      return true;
    },
    async updatePenalty(
      id: string,
      { label, amount }: { label?: string; amount?: number | string },
      _teamId: string,
    ): Promise<Penalty> {
      await delay(160, 300);
      const p = DB.penalties.find((x) => x.id === id)!;
      if (p) {
        if (label !== undefined) p.label = label;
        if (amount !== undefined) p.amount = Number(amount);
      }
      persist();
      return clone(p);
    },
    async updateContribution(
      id: string,
      { amount, label }: { amount?: number | string; label?: string },
      _teamId: string,
    ): Promise<Contribution> {
      await delay(160, 300);
      const c = DB.contributions.find((x) => x.id === id)!;
      if (c) {
        if (amount !== undefined) c.amount = Number(amount);
        if (label !== undefined) c.label = label;
      }
      persist();
      return clone(c);
    },
    async createPenalty(
      teamId: string,
      { label, amount }: { label: string; amount: number | string },
    ): Promise<Penalty> {
      await delay(200, 380);
      const p: Penalty = { id: rid('pen'), teamId, label, amount: Number(amount) };
      DB.penalties.push(p);
      persist();
      return clone(p);
    },
    async deletePenalty(id: string, _teamId: string) {
      await delay(160, 300);
      DB.penalties = DB.penalties.filter((x) => x.id !== id);
      DB.penaltyAssignments = DB.penaltyAssignments.filter((x) => x.penaltyId !== id);
      persist();
      return true;
    },
    async assignPenalty(
      teamId: string,
      { userId, penaltyId }: { userId: string; penaltyId: string },
    ): Promise<PenaltyAssignment> {
      await delay(200, 380);
      const a: PenaltyAssignment = { id: rid('pa'), teamId, userId, penaltyId, paid: false, date: todayLocalDate() };
      DB.penaltyAssignments.push(a);
      persist();
      return clone(a);
    },
    async deleteAssignment(id: string, _teamId: string) {
      await delay(160, 300);
      DB.penaltyAssignments = DB.penaltyAssignments.filter((x) => x.id !== id);
      persist();
      return true;
    },
    async togglePenaltyPaid(assignmentId: string, _teamId: string) {
      await delay(140, 280);
      const a = DB.penaltyAssignments.find((x) => x.id === assignmentId);
      if (a) a.paid = !a.paid;
      persist();
      return true;
    },
    async toggleContribution(contribId: string, _teamId: string) {
      await delay(140, 280);
      const c = DB.contributions.find((x) => x.id === contribId);
      if (c) c.status = c.status === 'paid' ? 'open' : 'paid';
      persist();
      return true;
    },
  },

  stats: {
    async attendanceFor(teamId: string, userId: string) {
      await delay(110, 220);
      // Matches stats.Service.GetMemberStats / SingleMemberStats: default
      // date range is 3 calendar months ago -> today when none is supplied
      // (this mock method takes no range param, so it always uses the
      // default), and `counted` is yes/no/maybe responses only — excludes
      // both 'pending' and 'not_nominated' (COUNT(*) FILTER (WHERE a.status
      // IN ('yes','no','maybe'))), not just 'not_nominated'.
      const to = todayLocalDate();
      const from = threeMonthsBeforeLocal(to);
      const inRange = DB.events.filter(
        (e) => e.teamId === teamId && e.status !== 'cancelled' && e.date >= from && e.date <= to,
      );
      let yes = 0,
        counted = 0;
      inRange.forEach((e) => {
        const s = effectiveStatus(e, userId).status;
        if (!isCountedStatus(s)) return;
        counted++;
        if (s === 'yes') yes++;
      });
      return { quote: counted ? Math.round((yes / counted) * 100) : null, counted, yes };
    },
    async teamOverview(teamId: string, range?: DateRange | null): Promise<StatsOverview> {
      await delay(180, 360);
      // Matches stats.Service.defaultDateRange (90 days ago exactly? no —
      // 3 calendar months, i.e. Go's `now.AddDate(0, -3, 0)`) -> today.
      const today = todayLocalDate();
      const from = range && range.from ? range.from : threeMonthsBeforeLocal(today);
      const to = range && range.to ? range.to : today;
      const memberIds = DB.memberships.filter((m) => m.teamId === teamId).map((m) => m.userId);
      // stats.Repository.{MemberStats,EventStats} filter status='active'
      // (this mock's only other status is 'cancelled', so `!== 'cancelled'`
      // is equivalent) and date BETWEEN from AND to — no "date < today"
      // requirement, unlike this mock's previous "past events only" filter,
      // which excluded today's/future-dated events within the range that
      // the real backend would include.
      const events = DB.events
        .filter((e) => e.teamId === teamId && e.status !== 'cancelled' && e.date >= from && e.date <= to)
        .sort((a, b) => a.date.localeCompare(b.date)); // ORDER BY e.date
      const memberStats = memberIds
        .map((uid) => {
          const u = DB.users.find((x) => x.id === uid)!;
          let yes = 0,
            counted = 0;
          events.forEach((e) => {
            const s = effectiveStatus(e, uid).status;
            if (!isCountedStatus(s)) return;
            counted++;
            if (s === 'yes') yes++;
          });
          return {
            userId: u.id,
            name: u.name,
            avatarColor: u.avatarColor,
            photo: u.photo,
            quote: counted ? Math.round((yes / counted) * 100) : null,
            counted,
            yes,
          };
        })
        .sort((a, b) => b.yes - a.yes || a.name.localeCompare(b.name, 'de')); // ORDER BY yes_count DESC, u.name
      // Matches stats.Service.GetOverview: `avg` sums every member's quote
      // (quote() returns 0, not skipped, for a member with 0 counted events)
      // and divides by the total member count — a member with no counted
      // attendance in range still drags the average down instead of being
      // excluded from both the numerator and the denominator.
      const avg = memberStats.length
        ? Math.round(memberStats.reduce((s, m) => s + (m.quote ?? 0), 0) / memberStats.length)
        : 0;
      const eventStats = events.map((e) => {
        let yes = 0,
          counted = 0;
        memberIds.forEach((uid) => {
          const s = effectiveStatus(e, uid).status;
          if (!isCountedStatus(s)) return;
          counted++;
          if (s === 'yes') yes++;
        });
        const pct = counted ? Math.round((yes / counted) * 100) : 0;
        return {
          id: e.id,
          title: e.title,
          type: e.type,
          date: e.date,
          yes,
          nominated: counted,
          pct,
          enough: pct >= 50, // matches stats.Service.GetOverview's `pct >= 0.5`
        };
      });
      return { avg, members: memberStats, events: eventStats, pastCount: events.length, from, to };
    },
  },

  polls: {
    async list(teamId: string): Promise<Poll[]> {
      await delay(120, 240);
      return DB.polls
        .filter((p) => p.teamId === teamId)
        .map((p) => {
          const total = p.votes.length;
          const counts: Record<string, number> = {};
          p.options.forEach((o: PollOptionDto) => {
            counts[o.id] = 0;
          });
          p.votes.forEach((v: PollVoteDto) =>
            v.optionIds.forEach((oid: string) => {
              if (counts[oid] !== undefined) counts[oid]++;
            }),
          );
          const mine = p.votes.find((v: PollVoteDto) => v.userId === session.userId);
          return {
            id: p.id,
            question: p.question,
            multiple: p.multiple,
            anonymous: p.anonymous,
            createdAt: p.createdAt,
            totalVotes: total,
            myVote: mine ? clone(mine.optionIds) : null,
            options: p.options.map((o: PollOptionDto) => ({
              id: o.id,
              text: o.text,
              count: counts[o.id],
              pct: total ? Math.round((counts[o.id] / total) * 100) : 0,
              voters: p.anonymous
                ? []
                : p.votes
                    .filter((v: PollVoteDto) => v.optionIds.includes(o.id))
                    .map((v: PollVoteDto) => {
                      const u = DB.users.find((x) => x.id === v.userId)!;
                      return { name: u.name, color: u.avatarColor, photo: u.photo };
                    }),
            })),
          };
        })
        .sort((a, b) => b.createdAt.localeCompare(a.createdAt));
    },
    async vote(pollId: string, optionIds: string[], _teamId: string) {
      await delay(160, 320);
      const p = DB.polls.find((x) => x.id === pollId)!;
      p.votes = p.votes.filter((v: PollVoteDto) => v.userId !== session.userId);
      if (optionIds.length)
        p.votes.push({ userId: session.userId!, optionIds: p.multiple ? optionIds : [optionIds[0]] });
      persist();
      return true;
    },
    async create(
      teamId: string,
      {
        question,
        options,
        multiple,
        anonymous,
      }: { question: string; options: string[]; multiple?: boolean; anonymous?: boolean },
    ) {
      await delay(260, 480);
      const poll = {
        id: rid('poll'),
        teamId,
        question,
        multiple: !!multiple,
        anonymous: !!anonymous,
        createdAt: iso(new Date()),
        options: options.filter((o) => o.trim()).map((o, i) => ({ id: 'opt' + i + '_' + rid('o'), text: o.trim() })),
        votes: [],
      };
      DB.polls.push(poll);
      pushNotif({ teamId, type: 'poll', actorId: session.userId!, title: question });
      persist();
      return clone(poll);
    },
    async remove(id: string, _teamId: string) {
      await delay(180, 340);
      DB.polls = DB.polls.filter((x) => x.id !== id);
      persist();
      return true;
    },
  },

  notifications: {
    async list(teamId: string): Promise<NotificationsResult> {
      await delay(110, 240);
      const since = Date.now() - 62 * DAY;
      const seen = (DB.notifSeen && DB.notifSeen[teamId]) || null;
      const items = DB.notifications
        .filter((n) => n.teamId === teamId && new Date(n.createdAt).getTime() >= since)
        .sort((a, b) => b.createdAt.localeCompare(a.createdAt))
        .map((n) => {
          const u = DB.users.find((x) => x.id === n.actorId);
          return Object.assign(clone(n), {
            actorName: u ? u.name : '',
            actorColor: u ? u.avatarColor : '#888',
            actorPhoto: u ? u.photo : null,
            unread: seen ? n.createdAt > seen : true,
          });
        });
      const unreadCount = items.filter((n) => n.unread).length;
      return { items, unreadCount };
    },
    async markSeen(teamId: string) {
      await delay(40, 100);
      DB.notifSeen = DB.notifSeen || {};
      DB.notifSeen[teamId] = iso(new Date());
      persist();
      return true;
    },
  },

  MODULES,
};

export const MODULE_LABELS: Record<ModuleKey, string> = {
  events: 'Termine',
  members: 'Mitglieder',
  finances: 'Finanzen',
  news: 'Neuigkeiten',
  polls: 'Umfragen',
  settings: 'Einstellungen',
};
export const STATUS_ORDER_EXPORT = STATUS_ORDER;

export type MockApi = typeof mockApi;
