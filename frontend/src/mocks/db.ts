// In-memory demo backend database for MSW handlers. Ported from the deleted
// src/services/mock/serviceLayerMock.ts + src/demo/seedData.ts, decoupled
// from that file's mock-only helper types. Rows are modeled on the frontend
// DTO types (which already mirror the OpenAPI wire shapes closely); handlers.ts
// is responsible for converting a row into the exact `components['schemas']`
// response shape.
import type { Invite, Membership, ModuleKey, Permissions, PermLevel, RoleDto, Team, User } from '@/types';
import type { Absence, AttendanceDto, EventComment, EventDto, ResponseMode } from '@/features/events';
import type { Contribution, Penalty, PenaltyAssignment, Transaction } from '@/features/finances';
import type { NewsItem } from '@/features/news';
import type { AppNotification } from '@/features/notifications';
import type { PollDto } from '@/features/polls';
import { formatDateOnly, monthsAgoLocal, todayLocalDate } from '@/utils/date';

export const rid = (p: string) => p + '_' + Math.random().toString(36).slice(2, 9);
const DAY = 86400000;
const iso = (d: Date) => d.toISOString();
const dstr = formatDateOnly;
export function atTime(h: number, m: number) {
  return String(h).padStart(2, '0') + ':' + String(m).padStart(2, '0');
}
function nextWeekday(weekday: number, weeks = 0) {
  const d = new Date();
  d.setHours(0, 0, 0, 0);
  const diff = (weekday - d.getDay() + 7) % 7;
  d.setDate(d.getDate() + diff + weeks * 7);
  return d;
}
function plusDays(n: number) {
  const d = new Date();
  d.setHours(0, 0, 0, 0);
  d.setDate(d.getDate() + n);
  return formatDateOnly(d);
}

export function perms(
  events: PermLevel,
  members: PermLevel,
  finances: PermLevel,
  news: PermLevel,
  polls: PermLevel,
  settings: PermLevel,
): Permissions {
  return { events, members, finances, news, polls, settings };
}

export const MODULES: ModuleKey[] = ['events', 'members', 'finances', 'news', 'polls', 'settings'];
const LEVEL: Record<PermLevel, number> = { none: 0, read: 1, write: 2 };

export function mergePerms(roles: RoleDto[]): Permissions {
  const out = perms('none', 'none', 'none', 'none', 'none', 'none');
  roles.forEach((r) =>
    MODULES.forEach((m) => {
      if (LEVEL[r.permissions[m]] > LEVEL[out[m]]) out[m] = r.permissions[m];
    }),
  );
  return out;
}

export function primaryRole(roles: RoleDto[]): RoleDto | null {
  const score = (r: RoleDto) => MODULES.reduce((s, m) => s + LEVEL[r.permissions[m]], 0);
  return [...roles].sort((a, b) => score(b) - score(a))[0] || null;
}

// The seeded default role newly-accepted invitees get (see handlers.ts's
// POST /invites/:code/accept) — a stable name, not derived by excluding the
// admin role, so it still resolves correctly even if a team has several
// non-admin system roles.
export const DEFAULT_MEMBER_ROLE_NAME = 'Tänzer / Mitglied';

function defaultRoles(teamId: string): RoleDto[] {
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
      name: DEFAULT_MEMBER_ROLE_NAME,
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

export interface UserRow extends User {
  hasPhoto: boolean;
  /**
   * Only set for self-registered (non-seed) accounts -- undefined for demo
   * seed users, which stay gated by the fixed DEMO_PASSWORD/DEMO_LOGIN_EMAIL
   * constants instead. Storing the plaintext password is fine here (this is
   * an in-memory demo/test double, not a real credential store).
   */
  password?: string;
  /** null/undefined = unverified. Mirrors the backend's users.email_verified_at. */
  emailVerifiedAt?: string | null;
}

/** A pending self-registration verification token (mock equivalent of the
 * backend's hashed email_verification_tokens table -- stored in the clear
 * here since there's nothing to protect in an in-memory test double). */
export interface VerificationToken {
  userId: string;
  expiresAt: string;
}

export interface TeamRow extends Team {
  hasPhoto: boolean;
  hasLogo: boolean;
}

export interface NewsRow extends NewsItem {
  teamId: string;
}

export interface DemoDb {
  users: UserRow[];
  teams: TeamRow[];
  memberships: Membership[];
  roles: RoleDto[];
  events: EventDto[];
  attendance: AttendanceDto[];
  invites: Invite[];
  absences: Absence[];
  news: NewsRow[];
  transactions: Transaction[];
  penalties: Penalty[];
  penaltyAssignments: PenaltyAssignment[];
  contributions: Contribution[];
  polls: PollDto[];
  eventComments: EventComment[];
  notifications: AppNotification[];
  notifSeen: Record<string, string>;
  /** Raw token -> pending verification, for self-registered accounts. */
  verificationTokens: Record<string, VerificationToken>;
}

// Fixed demo credentials for the MSW-only demo login (POST /auth/login).
// Deliberately NOT "any password works" — a mistyped/wrong password is
// rejected with 401, unlike the previous localStorage mock which ignored
// the password entirely. Documented here since there's no real user store.
export const DEMO_PASSWORD = 'demo-tanzsport';
export const DEMO_LOGIN_EMAIL = 'lena.bergmann@example.de';
export const DEMO_LOGIN_USER_ID = 'u1';
// Legacy one-tap "SSO" login ids, accepted with no password as a demo
// convenience (see handlers.ts's POST /auth/login) — distinct from, and not
// a weakening of, the DEMO_PASSWORD-gated email+password path.
export const DEMO_SSO_PROVIDER_IDS = ['google', 'apple', 'microsoft', 'vereins-sso'];

export function createSeedData(): DemoDb {
  const db: DemoDb = {
    users: [],
    teams: [],
    memberships: [],
    roles: [],
    events: [],
    attendance: [],
    invites: [],
    absences: [],
    news: [],
    transactions: [],
    penalties: [],
    penaltyAssignments: [],
    contributions: [],
    polls: [],
    eventComments: [],
    notifications: [],
    notifSeen: {},
    verificationTokens: {},
  };

  const U = (id: string, name: string, email: string, phone: string, color: string): UserRow => ({
    id,
    name,
    email,
    phone,
    avatarColor: color,
    photo: null,
    hasPhoto: false,
    birthday: '',
    address: '',
  });
  db.users = [
    U('u1', 'Lena Bergmann', 'lena.bergmann@example.de', '+49 151 2233445', '#1565C0'),
    U('u2', 'Jonas Krämer', 'jonas.kraemer@example.de', '+49 160 7788991', '#00796B'),
    U('u3', 'Marie Hoffmann', 'marie.hoffmann@example.de', '+49 152 3344556', '#C2185B'),
    U('u4', 'Tim Brauer', 'tim.brauer@example.de', '+49 171 9988776', '#5D4037'),
    U('u5', 'Sophie Klein', 'sophie.klein@example.de', '+49 159 1122334', '#7B1FA2'),
    U('u6', 'Niklas Wagner', 'niklas.wagner@example.de', '+49 176 5566778', '#455A64'),
    U('u7', 'Hannah Schäfer', 'hannah.schaefer@example.de', '+49 162 4455667', '#0277BD'),
    U('u8', 'David Möller', 'david.moeller@example.de', '+49 157 6677889', '#33691E'),
    U('u9', 'Clara Weiß', 'clara.weiss@example.de', '+49 155 2211009', '#E64A19'),
    U('u10', 'Felix Richter', 'felix.richter@example.de', '+49 173 8899001', '#0097A7'),
    U('u11', 'Anna Lehmann', 'anna.lehmann@example.de', '+49 151 3322110', '#AD1457'),
    U('u12', 'Paul Neumann', 'paul.neumann@example.de', '+49 170 1212343', '#283593'),
    U('u20', 'Greta Sommer', 'greta.sommer@example.de', '+49 152 9090901', '#00838F'),
    U('u21', 'Max Vogel', 'max.vogel@example.de', '+49 160 4343432', '#6A1B9A'),
    U('u22', 'Lina Fuchs', 'lina.fuchs@example.de', '+49 159 7676765', '#D84315'),
    U('u23', 'Ben Schulz', 'ben.schulz@example.de', '+49 176 1010102', '#1B5E20'),
  ];

  const tA: TeamRow = {
    id: 't_a',
    name: 'A-Team TSC Schwarz-Gelb Aachen',
    short: 'A',
    icon: '🏆',
    iconBg: '#1A1A1A',
    iconFg: '#F5C518',
    photo: null,
    logo: null,
    hasPhoto: false,
    hasLogo: false,
    description: '',
  };
  const tB: TeamRow = {
    id: 't_b',
    name: 'B-Team TSC Schwarz-Gelb Aachen',
    short: 'B',
    icon: '⭐',
    iconBg: '#1A1A1A',
    iconFg: '#F5C518',
    photo: null,
    logo: null,
    hasPhoto: false,
    hasLogo: false,
    description: '',
  };
  db.teams = [tA, tB];

  const rolesA = defaultRoles('t_a');
  const rolesB = defaultRoles('t_b');
  db.roles = [...rolesA, ...rolesB];
  const RA = (n: string) => rolesA.find((r) => r.name === n)!.id;
  const RB = (n: string) => rolesB.find((r) => r.name === n)!.id;
  tA.reasonVisibilityRoles = [RA('Admin / Trainer'), RA('Teamkapitän')];
  tB.reasonVisibilityRoles = [RB('Admin / Trainer'), RB('Teamkapitän')];
  tA.description = 'A-Formation Latein – aktuell in der NRW-Liga. Training Di & Do.';
  tB.description = 'B-Formation – Nachwuchs & Aufbau. Wir freuen uns über jede neue Tänzerin und jeden neuen Tänzer.';

  const M = (teamId: string, userId: string, roleIds: string[], group: string): Membership => ({
    id: rid('mem'),
    teamId,
    userId,
    roleIds,
    group,
    joinedAt: iso(new Date(Date.now() - 200 * DAY)),
  });
  db.memberships = [
    M('t_a', 'u1', [RA('Admin / Trainer'), RA('Tänzer / Mitglied')], 'A-Formation'),
    M('t_a', 'u2', [RA('Teamkapitän'), RA('Tänzer / Mitglied')], 'A-Formation'),
    M('t_a', 'u3', [RA('Kassenwart'), RA('Tänzer / Mitglied')], 'A-Formation'),
    M('t_a', 'u4', [RA('Tänzer / Mitglied')], 'A-Formation'),
    M('t_a', 'u5', [RA('Tänzer / Mitglied')], 'A-Formation'),
    M('t_a', 'u6', [RA('Tänzer / Mitglied')], 'A-Formation'),
    M('t_a', 'u7', [RA('Tänzer / Mitglied')], 'A-Formation'),
    M('t_a', 'u8', [RA('Tänzer / Mitglied')], 'A-Formation'),
    M('t_a', 'u9', [RA('Tänzer / Mitglied')], 'Nachwuchs'),
    M('t_a', 'u10', [RA('Tänzer / Mitglied')], 'Nachwuchs'),
    M('t_a', 'u11', [RA('Betreuer')], 'A-Formation'),
    M('t_a', 'u12', [RA('Tänzer / Mitglied')], 'Nachwuchs'),
    M('t_b', 'u1', [RB('Tänzer / Mitglied')], 'B-Formation'),
    M('t_b', 'u20', [RB('Admin / Trainer'), RB('Tänzer / Mitglied')], 'B-Formation'),
    M('t_b', 'u21', [RB('Teamkapitän'), RB('Tänzer / Mitglied')], 'B-Formation'),
    M('t_b', 'u22', [RB('Tänzer / Mitglied')], 'B-Formation'),
    M('t_b', 'u23', [RB('Tänzer / Mitglied')], 'B-Formation'),
  ];

  const ev: EventDto[] = [];
  const E = (o: Partial<EventDto>): EventDto => {
    const e = Object.assign(
      {
        id: rid('ev'),
        teamId: 't_a',
        recurring: false,
        meetTimeMandatory: true,
        location: '',
        note: '',
        responseMode: 'opt_in' as ResponseMode,
        status: 'active' as const,
        meetTime: null,
        startTime: null,
        endTime: null,
        seriesId: null,
      },
      o,
    ) as EventDto;
    ev.push(e);
    return e;
  };
  for (let w = -3; w <= 4; w++) {
    for (const wd of [2, 4]) {
      const d = nextWeekday(wd, w);
      E({
        type: 'training',
        title: 'Lateinformation – Training',
        date: dstr(d),
        meetTime: atTime(19, 15),
        startTime: atTime(19, 30),
        endTime: atTime(21, 30),
        meetTimeMandatory: true,
        location: 'Tanzsporthalle',
        recurring: true,
        seriesId: 'series_tue_thu',
        responseMode: 'opt_out',
      });
    }
  }
  const turnierD = nextWeekday(6, 1);
  E({
    type: 'auftritt',
    title: 'NRW-Liga Lateinformationen – 2. Wertung',
    date: dstr(turnierD),
    meetTime: atTime(9, 0),
    startTime: atTime(11, 0),
    endTime: atTime(18, 0),
    meetTimeMandatory: true,
    location: 'Castello Düsseldorf',
    responseMode: 'opt_in',
    note: 'Komplette Formation (16) erforderlich. Turnierkleidung & Haarteile mitbringen.',
  });
  const grillD = nextWeekday(5, 3);
  E({
    type: 'event',
    title: 'Saison-Kickoff Grillen',
    date: dstr(grillD),
    meetTime: atTime(18, 0),
    startTime: atTime(18, 0),
    endTime: atTime(23, 0),
    meetTimeMandatory: false,
    location: 'Vereinsheim',
    responseMode: 'opt_in',
    note: 'Bitte Salate/Beilagen in der Umfrage eintragen.',
  });
  const pastTurnier = nextWeekday(6, -2);
  E({
    type: 'auftritt',
    title: 'DM-Qualifikation Latein – 1. Wertung',
    date: dstr(pastTurnier),
    meetTime: atTime(8, 30),
    startTime: atTime(10, 30),
    endTime: atTime(17, 0),
    meetTimeMandatory: true,
    location: 'Stadthalle Braunschweig',
    responseMode: 'opt_in',
    result: 'Platz 3 von 11 – Aufstieg gesichert',
  });
  db.events = ev;

  const att: AttendanceDto[] = [];
  const A = (
    eventId: string,
    userId: string,
    status: AttendanceDto['status'],
    reason?: string,
    reasonId?: string | null,
    vis?: AttendanceDto['reasonVisibility'],
  ) =>
    att.push({
      id: rid('att'),
      eventId,
      userId,
      status,
      reason: reason || '',
      reasonId: reasonId || null,
      reasonVisibility: vis || null,
      at: iso(new Date()),
    });
  const aMembers = db.memberships.filter((m) => m.teamId === 't_a').map((m) => m.userId);
  const upcomingTraining = ev
    .filter((e) => e.type === 'training' && e.date >= todayLocalDate())
    .sort((a, b) => a.date.localeCompare(b.date))[0];
  const nominatedA = db.roles.filter((r) => r.teamId === 't_a' && r.name !== 'Betreuer').map((r) => r.id);
  ev.filter((e) => e.teamId === 't_a').forEach((e) => {
    e.nominatedRoleIds = [...nominatedA];
  });
  if (upcomingTraining) {
    const e = upcomingTraining.id;
    A(e, 'u1', 'yes');
    A(e, 'u4', 'yes');
    A(e, 'u5', 'no', 'Grippe, kuriere mich aus', 'cr1', 'trainers');
    A(e, 'u7', 'maybe');
    A(e, 'u8', 'yes');
    A(e, 'u9', 'maybe');
    A(e, 'u12', 'yes');
    A(e, 'u11', 'not_nominated');
  }
  const turnier = ev.find((e) => e.title.startsWith('NRW-Liga'));
  if (turnier) {
    aMembers.forEach((uid, i) => {
      if (uid === 'u11') return A(turnier.id, uid, 'not_nominated');
      if (i === 5) return A(turnier.id, uid, 'no', 'Familiäre Verpflichtung', 'cr2', 'team');
      if (i === 8) return A(turnier.id, uid, 'maybe');
      A(turnier.id, uid, 'yes');
    });
  }
  const pT = ev.find((e) => e.title.startsWith('DM-Qualifikation'));
  if (pT)
    aMembers.forEach((uid) =>
      A(
        pT.id,
        uid,
        uid === 'u9' ? 'no' : uid === 'u11' ? 'not_nominated' : 'yes',
        uid === 'u9' ? 'Krankheit' : '',
        uid === 'u9' ? 'cr1' : null,
        uid === 'u9' ? 'trainers' : null,
      ),
    );
  db.attendance = att;

  db.absences = [
    {
      id: rid('abs'),
      userId: 'u6',
      from: plusDays(5),
      to: plusDays(12),
      reason: 'Urlaub (Italien)',
      createdAt: iso(new Date()),
    },
    {
      id: rid('abs'),
      userId: 'u10',
      from: plusDays(1),
      to: plusDays(3),
      reason: 'Klassenfahrt',
      createdAt: iso(new Date()),
    },
    {
      id: rid('abs'),
      userId: 'u3',
      from: plusDays(20),
      to: plusDays(27),
      reason: 'Urlaub',
      createdAt: iso(new Date()),
    },
  ];

  db.news = [
    {
      id: rid('news'),
      teamId: 't_a',
      title: 'Aufstieg in die NRW-Liga! 🎉',
      body: 'Mit Platz 3 bei der DM-Qualifikation haben wir den Aufstieg sicher.',
      authorId: 'u1',
      pinned: true,
      createdAt: iso(new Date(Date.now() - 2 * DAY)),
    },
    {
      id: rid('news'),
      teamId: 't_a',
      title: 'Neue Trainingszeiten ab Juli',
      body: 'Ab Juli trainieren wir zusätzlich freitags.',
      authorId: 'u2',
      pinned: false,
      createdAt: iso(new Date(Date.now() - 5 * DAY)),
    },
  ];

  const T = (type: 'income' | 'expense', title: string, amountCents: number, daysAgo: number, cat: string) =>
    db.transactions.push({
      id: rid('tx'),
      teamId: 't_a',
      type,
      title,
      amount: amountCents,
      date: plusDays(-daysAgo),
      category: cat,
    });
  T('income', 'Mitgliedsbeiträge Mai', 30000, 30, 'Beiträge');
  T('income', 'Mitgliedsbeiträge Juni', 30000, 2, 'Beiträge');
  T('expense', 'Turnieranmeldung Düsseldorf', 18000, 6, 'Turniere');
  T('expense', 'Trikot-Nachbestellung', 24000, 15, 'Ausstattung');
  T('income', 'Spende Sponsor', 50000, 22, 'Spenden');
  T('expense', 'Hallenmiete Q2', 42000, 40, 'Halle');
  T('income', 'Strafenkasse', 4700, 3, 'Strafen');
  db.penalties = [
    { id: rid('pen'), teamId: 't_a', label: 'Zu spät zum Training', amount: 500 },
    { id: rid('pen'), teamId: 't_a', label: 'Training unentschuldigt verpasst', amount: 1000 },
    { id: rid('pen'), teamId: 't_a', label: 'Handy während Training', amount: 200 },
    { id: rid('pen'), teamId: 't_a', label: 'Tanzschuhe vergessen', amount: 300 },
  ];
  const pen = (i: number) => db.penalties[i];
  const PA = (userId: string, penIdx: number, paid: boolean, daysAgo: number) => {
    const p = pen(penIdx);
    db.penaltyAssignments.push({
      id: rid('pa'),
      teamId: 't_a',
      userId,
      penaltyId: p.id,
      paid,
      date: plusDays(-daysAgo),
      label: p.label,
      amount: p.amount,
    });
  };
  PA('u4', 0, false, 2);
  PA('u7', 1, false, 9);
  PA('u5', 2, true, 4);
  PA('u8', 0, false, 1);
  PA('u4', 2, false, 1);
  PA('u12', 3, true, 7);
  const monthKeyOff = (off: number) => {
    const d = new Date();
    d.setDate(1);
    d.setMonth(d.getMonth() - off);
    return formatDateOnly(d).slice(0, 7);
  };
  const CO = (userId: string, month: string, paid: boolean) =>
    db.contributions.push({
      id: rid('co'),
      teamId: 't_a',
      userId,
      month,
      label: 'Mitgliedsbeitrag',
      amount: 2500,
      status: paid ? 'paid' : 'open',
    });
  aMembers.forEach((uid, i) => {
    for (let off = 0; off < 6; off++) {
      const month = monthKeyOff(off);
      let paid: boolean;
      if (off === 0) paid = i % 3 !== 0;
      else if (off === 1) paid = i % 5 !== 0;
      else paid = i % 7 !== 0;
      CO(uid, month, paid);
    }
  });

  db.polls = [
    {
      id: rid('poll'),
      teamId: 't_a',
      question: 'Was bringst du zum Grillen mit?',
      multiple: true,
      anonymous: false,
      createdAt: iso(new Date(Date.now() - 1 * DAY)),
      options: [
        { id: 'o1', text: 'Salat' },
        { id: 'o2', text: 'Nachtisch' },
        { id: 'o3', text: 'Getränke' },
        { id: 'o4', text: 'Grillgut' },
      ],
      votes: [
        { userId: 'u2', optionIds: ['o1'] },
        { userId: 'u3', optionIds: ['o2', 'o3'] },
        { userId: 'u4', optionIds: ['o4'] },
        { userId: 'u5', optionIds: ['o1'] },
        { userId: 'u8', optionIds: ['o3'] },
      ],
    },
    {
      id: rid('poll'),
      teamId: 't_a',
      question: 'Neue Turnierkleidung – welche Farbe?',
      multiple: false,
      anonymous: true,
      createdAt: iso(new Date(Date.now() - 4 * DAY)),
      options: [
        { id: 'a', text: 'Schwarz / Gold (klassisch)' },
        { id: 'b', text: 'Dunkelrot' },
        { id: 'c', text: 'Marineblau' },
      ],
      votes: [
        { userId: 'u2', optionIds: ['a'] },
        { userId: 'u3', optionIds: ['a'] },
        { userId: 'u5', optionIds: ['b'] },
        { userId: 'u7', optionIds: ['a'] },
        { userId: 'u8', optionIds: ['c'] },
        { userId: 'u9', optionIds: ['a'] },
      ],
    },
  ];

  const ft = ev.find((e) => e.type === 'training' && e.date >= todayLocalDate());
  if (ft) {
    db.eventComments = [
      {
        id: rid('cm'),
        eventId: ft.id,
        userId: 'u2',
        text: 'Bringe die neue Musik für die Kür mit.',
        createdAt: iso(new Date(Date.now() - 2 * 3600 * 1000)),
      },
      {
        id: rid('cm'),
        eventId: ft.id,
        userId: 'u1',
        text: 'Perfekt – wir starten mit der Eröffnung, bitte pünktlich.',
        createdAt: iso(new Date(Date.now() - 1 * 3600 * 1000)),
      },
    ];
  }

  const ntf: AppNotification[] = [];
  const hAgo = (h: number) => iso(new Date(Date.now() - h * 3600 * 1000));
  const dAgo = (d: number) => iso(new Date(Date.now() - d * DAY));
  const N = (o: Partial<AppNotification>) =>
    ntf.push(Object.assign({ id: rid('ntf'), teamId: 't_a' }, o) as AppNotification);
  const trn = ev.find((e) => e.title.startsWith('NRW-Liga'));
  N({ type: 'event_created', actorId: 'u1', title: 'Lateinformation – Training (Serie Di & Do)', createdAt: dAgo(48) });
  if (trn)
    N({
      type: 'event_created',
      actorId: 'u1',
      title: trn.title,
      eventId: trn.id,
      eventTitle: trn.title,
      eventDate: trn.date,
      createdAt: dAgo(11),
    });
  db.news.forEach((n) => N({ type: 'news', actorId: n.authorId, title: n.title, createdAt: n.createdAt }));
  N({ type: 'poll', actorId: 'u1', title: 'Neue Turnierkleidung – welche Farbe?', createdAt: dAgo(4) });
  N({ type: 'absence', actorId: 'u6', title: 'Urlaub (Italien)', createdAt: dAgo(7) });
  db.notifications = ntf;
  db.notifSeen = { t_a: hAgo(30) };

  return db;
}

// Mutable singleton, replaced in-place by resetDb() so existing references
// (captured at handler-registration time) keep working.
export const db: DemoDb = createSeedData();

// Demo auth session — a single logged-in-user singleton (MSW is one demo
// backend per page load, not a multi-client server), reset alongside the DB.
export const session: { userId: string | null } = { userId: null };

export function resetDb(): void {
  Object.assign(db, createSeedData());
  session.userId = null;
}

// ---- shared query helpers, used by handlers.ts ----

export function rolesOf(membership: Membership): RoleDto[] {
  return membership.roleIds.map((id) => db.roles.find((r) => r.id === id)).filter(Boolean) as RoleDto[];
}

export function absenceCovers(userId: string, date: string): boolean {
  return db.absences.some((a) => a.userId === userId && date >= a.from && date <= a.to);
}

export interface EffectiveAttendance {
  status: AttendanceDto['status'];
  reason: string;
  reasonId: string | null;
  reasonVisibility: AttendanceDto['reasonVisibility'];
  auto: boolean;
  absent: boolean;
}

// Mirrors backend events.computeEffectiveAttendance: an explicit row wins;
// otherwise a covering planned absence defaults to "no" (auto, absent); an
// opt_out event with no record defaults to "yes" (auto); everything else is
// "pending". Used for event summaries/attendance rows/roster views — NOT for
// stats (see rawCountedStatus below, drift-bug fix #2).
export function effectiveStatus(event: EventDto, userId: string | null): EffectiveAttendance {
  const rec = db.attendance.find((a) => a.eventId === event.id && a.userId === userId);
  if (rec)
    return {
      status: rec.status,
      reason: rec.reason,
      reasonId: rec.reasonId,
      reasonVisibility: rec.reasonVisibility,
      auto: false,
      absent: absenceCovers(userId!, event.date),
    };
  if (userId && absenceCovers(userId, event.date))
    return { status: 'no', reason: '', reasonId: null, reasonVisibility: null, auto: true, absent: true };
  if (event.responseMode === 'opt_out')
    return { status: 'yes', reason: '', reasonId: null, reasonVisibility: null, auto: true, absent: false };
  return { status: 'pending', reason: '', reasonId: null, reasonVisibility: null, auto: false, absent: false };
}

// Drift-bug fix #2: stats.Repository (backend/internal/stats/repository.go)
// joins the raw `attendance` table directly (`a.status IN ('yes','no','maybe')`)
// with no opt_out/absence defaulting — unlike the event-summary/roster views,
// a member who never explicitly responded to an opt_out event or who is
// auto-marked absent does NOT count toward a stats quota. Only an explicit
// attendance record counts.
export function rawCountedStatus(eventId: string, userId: string): 'yes' | 'no' | 'maybe' | null {
  const rec = db.attendance.find((a) => a.eventId === eventId && a.userId === userId);
  if (rec && (rec.status === 'yes' || rec.status === 'no' || rec.status === 'maybe')) return rec.status;
  return null;
}

export function threeMonthsBeforeLocal(dateStr: string): string {
  return monthsAgoLocal(dateStr, 3);
}

export function applyNominations(event: EventDto, nominatedRoleIds: string[]): void {
  event.nominatedRoleIds = [...nominatedRoleIds];
  const nomSet = new Set(nominatedRoleIds);
  const members = db.memberships.filter((m) => m.teamId === event.teamId);
  members.forEach((m) => {
    const nominated = m.roleIds.some((roleId) => nomSet.has(roleId));
    const a = db.attendance.find((x) => x.eventId === event.id && x.userId === m.userId);
    if (nominated) {
      if (a && a.status === 'not_nominated') db.attendance = db.attendance.filter((x) => x !== a);
      return;
    }
    if (!a) {
      db.attendance.push({
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

export function pushNotif(o: Partial<AppNotification>): void {
  db.notifications.push(Object.assign({ id: rid('ntf'), createdAt: iso(new Date()) }, o) as AppNotification);
}
