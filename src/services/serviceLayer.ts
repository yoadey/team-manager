// =============================================================================
// serviceLayer.ts — Mock-Backend für die Teamverwaltungs-App
// -----------------------------------------------------------------------------
// Faithful TypeScript port of the prototype's serviceLayer.js. Simulates the
// future Go/PostgreSQL backend as an async API with artificial latency and
// localStorage persistence. Replace method bodies with HTTP calls later; the
// signatures (the API contract) stay the same.
// =============================================================================

import { mapAttendanceDtoToRow, mapEventDtoToTeamEvent, mapMemberDtoToMember } from './mappers';
import type { AttendanceStatus, DateRange, EventType, Invite, Membership, ModuleKey, Permissions, Provider, ReasonVisibility, Role, RoleDto, StatsOverview, Team, TeamForUser, User } from '@/types';
import type { Absence, AttendanceDto, AttendanceRow, EventComment, EventDto, ResponseMode, TeamEvent } from '@/features/events';
import type { Contribution, FinanceOverview, Penalty, PenaltyAssignment, Transaction } from '@/features/finances';
import type { Member, MemberDto } from '@/features/members';
import type { NewsItem } from '@/features/news';
import type { AppNotification, NotificationsResult } from '@/features/notifications';
import type { Poll } from '@/features/polls';
import { formatDateOnly, parseDateOnlyLocal, todayLocalDate } from '@/utils/date';

const rid = (p: string) => p + '_' + Math.random().toString(36).slice(2, 9);
const delay = (min = 120, max = 320) =>
  new Promise<void>((r) => setTimeout(r, min + Math.random() * (max - min)));
const clone = <T>(x: T): T => JSON.parse(JSON.stringify(x));
const DAY = 86400000;
const iso = (d: Date) => d.toISOString();
const dstr = formatDateOnly;
function atTime(_base: Date | string, h: number, m: number) {
  return String(h).padStart(2, '0') + ':' + String(m).padStart(2, '0');
}
function nextWeekday(weekday: number, weeks = 0) {
  const d = new Date();
  d.setHours(0, 0, 0, 0);
  const diff = (weekday - d.getDay() + 7) % 7;
  d.setTime(d.getTime() + (diff + weeks * 7) * DAY);
  return d;
}
function plusDays(n: number) {
  const d = new Date();
  d.setHours(0, 0, 0, 0);
  d.setTime(d.getTime() + n * DAY);
  return formatDateOnly(d);
}

// ---- Rechte-Modell ----------------------------------------------------------
const MODULES: ModuleKey[] = ['events', 'members', 'finances', 'news', 'polls', 'settings'];
function perms(
  events: any, members: any, finances: any, news: any, polls: any, settings: any,
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
    { id: rid('role'), teamId, name: 'Admin / Trainer', system: true, color: '#1565C0', permissions: perms('write', 'write', 'write', 'write', 'write', 'write') },
    { id: rid('role'), teamId, name: 'Tänzer / Mitglied', system: true, color: '#5B6470', permissions: perms('read', 'read', 'read', 'read', 'read', 'none') },
    { id: rid('role'), teamId, name: 'Kassenwart', system: true, color: '#2E7D32', permissions: perms('read', 'read', 'write', 'read', 'read', 'none') },
    { id: rid('role'), teamId, name: 'Teamkapitän', system: true, color: '#E8910C', permissions: perms('write', 'read', 'read', 'write', 'write', 'none') },
    { id: rid('role'), teamId, name: 'Betreuer', system: true, color: '#7A4FB6', permissions: perms('read', 'read', 'none', 'read', 'read', 'none') },
  ];
}

interface DB {
  users: User[];
  teams: Team[];
  memberships: Membership[];
  roles: RoleDto[];
  events: EventDto[];
  attendance: AttendanceDto[];
  invites: Invite[];
  absences: Absence[];
  news: NewsItem[];
  transactions: Transaction[];
  penalties: Penalty[];
  penaltyAssignments: PenaltyAssignment[];
  contributions: Contribution[];
  polls: any[];
  eventComments: EventComment[];
  notifications: AppNotification[];
  notifSeen: Record<string, string>;
  meta?: { seededAt: string; version: number };
}

// ---- Seed -------------------------------------------------------------------
function seed(): DB {
  const db: DB = {
    users: [], teams: [], memberships: [], roles: [], events: [], attendance: [], invites: [],
    absences: [], news: [], transactions: [], penalties: [], penaltyAssignments: [],
    contributions: [], polls: [], eventComments: [], notifications: [], notifSeen: {},
  };

  const U = (id: string, name: string, email: string, phone: string, color: string): User =>
    ({ id, name, email, phone, avatarColor: color, photo: null, birthday: '', address: '' });
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

  const tA: Team = { id: 't_a', name: 'A-Team TSC Schwarz-Gelb Aachen', short: 'A', icon: '🏆', iconBg: '#1A1A1A', iconFg: '#F5C518', photo: null, logo: null, description: '' };
  const tB: Team = { id: 't_b', name: 'B-Team TSC Schwarz-Gelb Aachen', short: 'B', icon: '⭐', iconBg: '#1A1A1A', iconFg: '#F5C518', photo: null, logo: null, description: '' };
  db.teams = [tA, tB];

  const rolesA = defaultRoles('t_a');
  const rolesB = defaultRoles('t_b');
  db.roles = [...rolesA, ...rolesB];
  const RA = (n: string) => rolesA.find((r) => r.name === n)!.id;
  const RB = (n: string) => rolesB.find((r) => r.name === n)!.id;
  tA.reasonVisibilityRoles = [RA('Admin / Trainer'), RA('Teamkapitän')];
  tB.reasonVisibilityRoles = [RB('Admin / Trainer'), RB('Teamkapitän')];
  tA.description = 'A-Formation Latein – aktuell in der NRW-Liga. Training Di & Do in Eilendorf.';
  tB.description = 'B-Formation – Nachwuchs & Aufbau. Wir freuen uns über jede neue Tänzerin und jeden neuen Tänzer.';
  const setProfile = (uid: string, bd: string, addr: string) => {
    const u = db.users.find((x) => x.id === uid);
    if (u) { u.birthday = bd; u.address = addr; }
  };
  setProfile('u1', '1998-04-12', 'Jülicher Straße 12, 52070 Aachen');
  setProfile('u2', '1996-09-30', 'Adalbertsteinweg 88, 52070 Aachen');
  setProfile('u3', '2000-01-23', 'Pontstraße 45, 52062 Aachen');
  setProfile('u4', '1999-07-08', 'Vaalser Straße 210, 52074 Aachen');

  const M = (teamId: string, userId: string, roleIds: string[], group: string): Membership =>
    ({ id: rid('mem'), teamId, userId, roleIds, group, joinedAt: iso(new Date(Date.now() - 200 * DAY)) });
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

  // ---- Termine A-Team ----
  const ev: EventDto[] = [];
  const E = (o: Partial<EventDto>): EventDto => {
    const e = Object.assign(
      { id: rid('ev'), teamId: 't_a', recurring: false, meetTimeMandatory: true, location: '', note: '', responseMode: 'opt_in' as ResponseMode, status: 'active' as const },
      o,
    ) as EventDto;
    ev.push(e);
    return e;
  };
  for (let w = -3; w <= 4; w++) {
    for (const wd of [2, 4]) {
      const d = nextWeekday(wd, w);
      E({ type: 'training', title: 'Lateinformation – Training', date: dstr(d), meetTime: atTime(d, 19, 15), startTime: atTime(d, 19, 30), endTime: atTime(d, 21, 30), meetTimeMandatory: true, location: 'Tanzsporthalle Eilendorf', recurring: true, seriesId: 'series_tue_thu', responseMode: 'opt_out' });
    }
  }
  const turnierD = nextWeekday(6, 1);
  E({ type: 'auftritt', title: 'NRW-Liga Lateinformationen – 2. Wertung', date: dstr(turnierD), meetTime: atTime(turnierD, 9, 0), startTime: atTime(turnierD, 11, 0), endTime: atTime(turnierD, 18, 0), meetTimeMandatory: true, location: 'Castello Düsseldorf', responseMode: 'opt_in', note: 'Komplette Formation (16) erforderlich. Turnierkleidung & Haarteile mitbringen.' });
  const grillD = nextWeekday(5, 3);
  E({ type: 'event', title: 'Saison-Kickoff Grillen', date: dstr(grillD), meetTime: atTime(grillD, 18, 0), startTime: atTime(grillD, 18, 0), endTime: atTime(grillD, 23, 0), meetTimeMandatory: false, location: 'Vereinsheim, Aachen-Brand', responseMode: 'opt_in', note: 'Bitte Salate/Beilagen in der Umfrage eintragen.' });
  const pastTurnier = nextWeekday(6, -2);
  E({ type: 'auftritt', title: 'DM-Qualifikation Latein – 1. Wertung', date: dstr(pastTurnier), meetTime: atTime(pastTurnier, 8, 30), startTime: atTime(pastTurnier, 10, 30), endTime: atTime(pastTurnier, 17, 0), meetTimeMandatory: true, location: 'Stadthalle Braunschweig', responseMode: 'opt_in', result: 'Platz 3 von 11 – Aufstieg gesichert' });
  db.events = ev;

  // ---- Anwesenheit ----
  const att: any[] = [];
  const A = (eventId: string, userId: string, status: AttendanceStatus, reason?: string, reasonId?: string | null, vis?: ReasonVisibility) =>
    att.push({ id: rid('att'), eventId, userId, status, reason: reason || '', reasonId: reasonId || null, reasonVisibility: vis || null, at: iso(new Date()) });
  const aMembers = db.memberships.filter((m) => m.teamId === 't_a').map((m) => m.userId);
  const upcomingTraining = ev.filter((e) => e.type === 'training' && e.date >= todayLocalDate()).sort((a, b) => a.date.localeCompare(b.date))[0];
  const nominatedA = db.roles.filter((r) => r.teamId === 't_a' && r.name !== 'Betreuer').map((r) => r.id);
  ev.filter((e) => e.teamId === 't_a').forEach((e) => { e.nominatedRoleIds = [...nominatedA]; });
  if (upcomingTraining) {
    const e = upcomingTraining.id;
    A(e, 'u1', 'yes'); A(e, 'u4', 'yes'); A(e, 'u5', 'no', 'Grippe, kuriere mich aus', 'cr1', 'trainers');
    A(e, 'u7', 'maybe'); A(e, 'u8', 'yes'); A(e, 'u9', 'maybe'); A(e, 'u12', 'yes');
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
  if (pT) aMembers.forEach((uid) => A(pT.id, uid, uid === 'u9' ? 'no' : (uid === 'u11' ? 'not_nominated' : 'yes'), uid === 'u9' ? 'Krankheit' : '', uid === 'u9' ? 'cr1' : null, uid === 'u9' ? 'trainers' : null));
  db.attendance = att;

  // ---- Abwesenheiten (geplant) ----
  db.absences = [
    { id: rid('abs'), userId: 'u6', from: plusDays(5), to: plusDays(12), reason: 'Urlaub (Italien)', createdAt: iso(new Date()) },
    { id: rid('abs'), userId: 'u10', from: plusDays(1), to: plusDays(3), reason: 'Klassenfahrt', createdAt: iso(new Date()) },
    { id: rid('abs'), userId: 'u3', from: plusDays(20), to: plusDays(27), reason: 'Urlaub', createdAt: iso(new Date()) },
  ];

  // ---- News ----
  db.news = [
    { id: rid('news'), teamId: 't_a', title: 'Aufstieg in die NRW-Liga! 🎉', body: 'Mit Platz 3 bei der DM-Qualifikation haben wir den Aufstieg sicher. Riesen Kompliment an die ganze Formation – jetzt greifen wir in der NRW-Liga an!', authorId: 'u1', pinned: true, createdAt: iso(new Date(Date.now() - 2 * DAY)) },
    { id: rid('news'), teamId: 't_a', title: 'Neue Trainingszeiten ab Juli', body: 'Ab Juli trainieren wir zusätzlich freitags. Details folgen, bitte Kalender im Blick behalten.', authorId: 'u2', pinned: false, createdAt: iso(new Date(Date.now() - 5 * DAY)) },
    { id: rid('news'), teamId: 't_a', title: 'Turnieranmeldung Düsseldorf abgeschlossen', body: 'Alle gemeldeten Paare sind bestätigt. Treffpunkt und Ablauf stehen im Termin zur 2. Wertung.', authorId: 'u1', pinned: false, createdAt: iso(new Date(Date.now() - 8 * DAY)) },
  ];

  // ---- Finanzen ----
  const T = (type: 'income' | 'expense', title: string, amount: number, daysAgo: number, cat: string) =>
    db.transactions.push({ id: rid('tx'), teamId: 't_a', type, title, amount, date: plusDays(-daysAgo), category: cat });
  T('income', 'Mitgliedsbeiträge Mai', 300, 30, 'Beiträge');
  T('income', 'Mitgliedsbeiträge Juni', 300, 2, 'Beiträge');
  T('expense', 'Turnieranmeldung Düsseldorf', 180, 6, 'Turniere');
  T('expense', 'Trikot-Nachbestellung', 240, 15, 'Ausstattung');
  T('income', 'Spende Sponsor „Tanzhaus"', 500, 22, 'Spenden');
  T('expense', 'Hallenmiete Q2', 420, 40, 'Halle');
  T('income', 'Strafenkasse', 47, 3, 'Strafen');
  db.penalties = [
    { id: rid('pen'), teamId: 't_a', label: 'Zu spät zum Training', amount: 5 },
    { id: rid('pen'), teamId: 't_a', label: 'Training unentschuldigt verpasst', amount: 10 },
    { id: rid('pen'), teamId: 't_a', label: 'Handy während Training', amount: 2 },
    { id: rid('pen'), teamId: 't_a', label: 'Tanzschuhe vergessen', amount: 3 },
  ];
  const pid = (i: number) => db.penalties[i].id;
  const PA = (userId: string, penIdx: number, paid: boolean, daysAgo: number) =>
    db.penaltyAssignments.push({ id: rid('pa'), teamId: 't_a', userId, penaltyId: pid(penIdx), paid, date: plusDays(-daysAgo) });
  PA('u4', 0, false, 2); PA('u7', 1, false, 9); PA('u5', 2, true, 4); PA('u8', 0, false, 1); PA('u4', 2, false, 1); PA('u12', 3, true, 7);
  const monthKeyOff = (off: number) => {
    const d = new Date(); d.setDate(1); d.setMonth(d.getMonth() - off); return formatDateOnly(d).slice(0, 7);
  };
  const CO = (userId: string, month: string, paid: boolean) =>
    db.contributions.push({ id: rid('co'), teamId: 't_a', userId, month, label: 'Mitgliedsbeitrag', amount: 25, status: paid ? 'paid' : 'open' });
  aMembers.forEach((uid, i) => {
    for (let off = 0; off < 6; off++) {
      const month = monthKeyOff(off);
      let paid: boolean;
      if (off === 0) paid = (i % 3 !== 0);
      else if (off === 1) paid = (i % 5 !== 0);
      else paid = (i % 7 !== 0);
      CO(uid, month, paid);
    }
  });

  // ---- Umfragen ----
  db.polls = [
    { id: rid('poll'), teamId: 't_a', question: 'Was bringst du zum Grillen mit?', multiple: true, anonymous: false, createdAt: iso(new Date(Date.now() - 1 * DAY)), options: [{ id: 'o1', text: 'Salat' }, { id: 'o2', text: 'Nachtisch' }, { id: 'o3', text: 'Getränke' }, { id: 'o4', text: 'Grillgut' }], votes: [{ userId: 'u2', optionIds: ['o1'] }, { userId: 'u3', optionIds: ['o2', 'o3'] }, { userId: 'u4', optionIds: ['o4'] }, { userId: 'u5', optionIds: ['o1'] }, { userId: 'u8', optionIds: ['o3'] }] },
    { id: rid('poll'), teamId: 't_a', question: 'Neue Turnierkleidung – welche Farbe?', multiple: false, anonymous: true, createdAt: iso(new Date(Date.now() - 4 * DAY)), options: [{ id: 'a', text: 'Schwarz / Gold (klassisch)' }, { id: 'b', text: 'Dunkelrot' }, { id: 'c', text: 'Marineblau' }], votes: [{ userId: 'u2', optionIds: ['a'] }, { userId: 'u3', optionIds: ['a'] }, { userId: 'u5', optionIds: ['b'] }, { userId: 'u7', optionIds: ['a'] }, { userId: 'u8', optionIds: ['c'] }, { userId: 'u9', optionIds: ['a'] }] },
  ];

  const _ft = ev.find((e) => e.type === 'training' && e.date >= todayLocalDate());
  if (_ft) {
    db.eventComments = [
      { id: rid('cm'), eventId: _ft.id, userId: 'u2', text: 'Bringe die neue Musik für die Kür mit.', createdAt: iso(new Date(Date.now() - 2 * 3600 * 1000)) },
      { id: rid('cm'), eventId: _ft.id, userId: 'u1', text: 'Perfekt – wir starten mit der Eröffnung, bitte pünktlich.', createdAt: iso(new Date(Date.now() - 1 * 3600 * 1000)) },
    ];
  }

  // ---- Benachrichtigungen (letzte ~2 Monate) ----
  const ntf: AppNotification[] = [];
  const hAgo = (h: number) => iso(new Date(Date.now() - h * 3600 * 1000));
  const dAgo = (d: number) => iso(new Date(Date.now() - d * DAY));
  const N = (o: Partial<AppNotification>) => ntf.push(Object.assign({ id: rid('ntf'), teamId: 't_a' }, o) as AppNotification);
  const upTrain = ev.filter((e) => e.type === 'training' && e.date >= todayLocalDate()).sort((a, b) => a.date.localeCompare(b.date))[0];
  const trn = ev.find((e) => e.title.startsWith('NRW-Liga'));
  const grl = ev.find((e) => e.type === 'event');
  N({ type: 'event_created', actorId: 'u1', title: 'Lateinformation – Training (Serie Di & Do)', createdAt: dAgo(48) });
  N({ type: 'event_created', actorId: 'u1', title: trn ? trn.title : 'NRW-Liga', eventId: trn ? trn.id : null, eventTitle: trn ? trn.title : '', eventDate: trn ? trn.date : '', createdAt: dAgo(11) });
  N({ type: 'event_updated', actorId: 'u2', title: 'Lateinformation – Training', createdAt: dAgo(9), note: 'Treffzeit auf 19:15 vorgezogen' });
  N({ type: 'event_created', actorId: 'u1', title: grl ? grl.title : 'Saison-Kickoff Grillen', eventId: grl ? grl.id : null, eventTitle: grl ? grl.title : '', eventDate: grl ? grl.date : '', createdAt: dAgo(6) });
  db.news.forEach((n) => N({ type: 'news', actorId: n.authorId, title: n.title, createdAt: n.createdAt }));
  N({ type: 'poll', actorId: 'u1', title: 'Neue Turnierkleidung – welche Farbe?', createdAt: dAgo(4) });
  N({ type: 'poll', actorId: 'u2', title: 'Was bringst du zum Grillen mit?', createdAt: dAgo(1) });
  N({ type: 'absence', actorId: 'u6', title: 'Urlaub (Italien)', createdAt: dAgo(7) });
  N({ type: 'absence', actorId: 'u10', title: 'Klassenfahrt', createdAt: dAgo(3) });
  if (trn) {
    const trnResp: [string, AttendanceStatus, number][] = [['u2', 'yes', 30], ['u3', 'yes', 28], ['u4', 'yes', 26], ['u5', 'yes', 22], ['u8', 'no', 20], ['u9', 'maybe', 14], ['u7', 'yes', 10], ['u12', 'yes', 6], ['u8', 'yes', 4]];
    trnResp.forEach(([uid, st, h]) => N({ type: 'attendance', actorId: uid, status: st, eventId: trn.id, eventTitle: trn.title, eventDate: trn.date, createdAt: hAgo(h) }));
  }
  if (upTrain) {
    const trResp: [string, AttendanceStatus, number][] = [['u4', 'yes', 18], ['u5', 'no', 16], ['u7', 'maybe', 12], ['u8', 'yes', 5], ['u9', 'maybe', 3], ['u12', 'yes', 2]];
    trResp.forEach(([uid, st, h]) => N({ type: 'attendance', actorId: uid, status: st, eventId: upTrain.id, eventTitle: upTrain.title, eventDate: upTrain.date, createdAt: hAgo(h) }));
  }
  db.notifications = ntf;
  db.notifSeen = { t_a: hAgo(30) };

  db.meta = { seededAt: iso(new Date()), version: 6 };
  return db;
}

// ---- Persistenz (tagesfrisch) ----------------------------------------------
function todayKey() { return 'tv_db_v7_' + todayLocalDate(); }
function loadDb(): DB {
  try {
    const raw = localStorage.getItem(todayKey());
    if (raw) return JSON.parse(raw);
  } catch { /* ignore */ }
  const db = seed();
  save(db);
  return db;
}
function save(db: DB) {
  try { localStorage.setItem(todayKey(), JSON.stringify(db)); } catch { /* ignore */ }
}
let DB = loadDb();
function persist() { save(DB); }
function pushNotif(o: Partial<AppNotification>) {
  DB.notifications.push(Object.assign({ id: rid('ntf'), createdAt: iso(new Date()) }, o) as AppNotification);
}
const session: { userId: string | null } = { userId: null };

export function resetDemoData() {
  try {
    Object.keys(localStorage).filter((k) => k.startsWith('tv_db_')).forEach((k) => localStorage.removeItem(k));
  } catch { /* ignore */ }
}

const PROVIDERS: Provider[] = [
  { id: 'vereins-sso', name: 'Vereins-SSO', sub: 'TSC Schwarz-Gelb', glyph: 'SG', bg: '#1A1A1A', fg: '#F5C518' },
  { id: 'google', name: 'Google', sub: 'Mit Google fortfahren', glyph: 'G', bg: '#FFFFFF', fg: '#4285F4', border: true },
  { id: 'microsoft', name: 'Microsoft', sub: 'Entra ID / Microsoft 365', glyph: 'M', bg: '#FFFFFF', fg: '#5E5E5E', border: true },
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
function effectiveStatus(event: EventDto, userId: string | null) {
  const rec = DB.attendance.find((a) => a.eventId === event.id && a.userId === userId);
  if (rec) return { status: rec.status as AttendanceStatus, reason: rec.reason, reasonId: rec.reasonId, reasonVisibility: rec.reasonVisibility, auto: false, absent: absenceCovers(userId!, event.date) };
  if (userId && absenceCovers(userId, event.date)) return { status: 'no' as AttendanceStatus, reason: 'Geplante Abwesenheit', reasonId: null, reasonVisibility: 'team' as ReasonVisibility, auto: true, absent: true };
  if (event.responseMode === 'opt_out') return { status: 'yes' as AttendanceStatus, reason: '', reasonId: null, reasonVisibility: null, auto: true, absent: false };
  return { status: 'pending' as AttendanceStatus, reason: '', reasonId: null, reasonVisibility: null, auto: false, absent: false };
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
      DB.attendance.push({ id: rid('att'), eventId: event.id, userId: m.userId, status: 'not_nominated', reason: '', reasonId: null, reasonVisibility: null, at: iso(new Date()) });
    } else if (a.status === 'not_nominated') {
      a.reason = ''; a.reasonId = null; a.reasonVisibility = null; a.at = iso(new Date());
    }
  });
}

// =============================================================================
export const SERVICE_ENDPOINTS = {
  // auth: thin wrappers for future authentication/session endpoints.
  auth: ['auth.providers', 'auth.login', 'auth.currentUser', 'auth.logout', 'auth.setPhoto'],
  // teams: thin wrappers for team CRUD and invite endpoints; listForCurrentUser is enriched with role permissions.
  teams: ['teams.listForCurrentUser', 'teams.get', 'teams.create', 'teams.updateSettings', 'teams.createInvite'],
  // members: list returns Member ViewModels mapped from MemberDto plus derived primaryRole/perms.
  members: ['members.list', 'members.add', 'members.update', 'members.setRoles', 'members.remove'],
  // roles: direct RoleDto CRUD endpoints.
  roles: ['roles.list', 'roles.create', 'roles.update', 'roles.remove'],
  // events: list/get map EventDto to TeamEvent with client-side summary/myStatus aggregation.
  events: ['events.list', 'events.get', 'events.create', 'events.update', 'events.setStatus', 'events.remove', 'events.listComments', 'events.addComment', 'events.removeComment'],
  // attendance: set/setNomination are endpoint-shaped mutations; listForEvent maps AttendanceDto to display rows.
  attendance: ['attendance.listForEvent', 'attendance.set', 'attendance.setNomination'],
  // absences/news/polls/notifications: endpoint-shaped resource operations with small display enrichments.
  absences: ['absences.listForTeam', 'absences.listMine', 'absences.create', 'absences.update', 'absences.remove'],
  news: ['news.list', 'news.create', 'news.remove'],
  polls: ['polls.list', 'polls.vote', 'polls.create'],
  notifications: ['notifications.list', 'notifications.markSeen'],
  // finances.overview is currently a client aggregation over finance collections; mutation methods map to future endpoints.
  finances: ['finances.overview', 'finances.addTransaction', 'finances.updateTransaction', 'finances.deleteTransaction', 'finances.updatePenalty', 'finances.deletePenalty', 'finances.addPenaltyAssignment', 'finances.togglePenaltyPaid', 'finances.updateContribution'],
  // stats.overview is a client-side aggregation until the backend exposes a reporting endpoint.
  stats: ['stats.overview'],
} as const;

export const CLIENT_AGGREGATIONS = ['events._withSummary', 'members.list', 'attendance.listForEvent', 'finances.overview', 'stats.overview'] as const;

export const API_ENDPOINT_METHODS = Object.values(SERVICE_ENDPOINTS).flat();

export const api = {
  auth: {
    async providers(): Promise<Provider[]> { await delay(80, 200); return clone(PROVIDERS); },
    async login(providerId: string) {
      await delay(380, 720);
      session.userId = 'u1';
      return { token: 'mock.jwt.' + rid('tk'), provider: providerId, user: clone(DB.users.find((u) => u.id === 'u1')!) };
    },
    async currentUser(): Promise<User | null> {
      await delay(50, 120);
      return session.userId ? clone(DB.users.find((u) => u.id === session.userId)!) : null;
    },
    logout() { session.userId = null; },
    async setPhoto(dataUrl: string): Promise<User> {
      await delay(150, 300);
      const u = DB.users.find((x) => x.id === session.userId)!;
      u.photo = dataUrl; persist(); return clone(u);
    },
  },

  teams: {
    async listForCurrentUser(): Promise<TeamForUser[]> {
      await delay();
      return DB.memberships.filter((m) => m.userId === session.userId).map((m) => {
        const t = DB.teams.find((x) => x.id === m.teamId)!;
        const roles = rolesOf(m);
        return Object.assign(clone(t), { myRoles: clone(roles), myPerms: mergePerms(roles), membershipId: m.id, memberCount: DB.memberships.filter((x) => x.teamId === t.id).length });
      });
    },
    async get(teamId: string): Promise<Team> { await delay(80, 160); return clone(DB.teams.find((t) => t.id === teamId)!); },
    async create({ name, icon, iconBg, iconFg, photo }: { name: string; icon?: string; iconBg?: string; iconFg?: string; photo?: string | null }): Promise<Team> {
      await delay(340, 600);
      const team: Team = { id: rid('t'), name, short: (name || 'T').trim().charAt(0).toUpperCase(), icon: icon || '⭐', iconBg: iconBg || '#1565C0', iconFg: iconFg || '#FFFFFF', photo: photo || null, logo: null, description: '' };
      DB.teams.push(team);
      const roles = defaultRoles(team.id);
      DB.roles.push(...roles);
      const admin = roles.find((r) => r.name === 'Admin / Trainer')!;
      team.reasonVisibilityRoles = [admin.id];
      DB.memberships.push({ id: rid('mem'), teamId: team.id, userId: session.userId!, roleIds: [admin.id], group: '', joinedAt: iso(new Date()) });
      persist(); return clone(team);
    },
    async updateSettings(teamId: string, patch: Partial<Team>): Promise<Team> {
      await delay();
      const t = DB.teams.find((x) => x.id === teamId)!;
      Object.assign(t, patch); persist(); return clone(t);
    },
    async createInvite(teamId: string): Promise<Invite> {
      await delay(180, 360);
      const code = Math.random().toString(36).slice(2, 8).toUpperCase();
      const inv: Invite = { id: rid('inv'), teamId, code, link: 'https://teamverwaltung.app/join/' + code, createdAt: iso(new Date()), expiresAt: iso(new Date(Date.now() + 7 * DAY)) };
      DB.invites.push(inv); persist(); return clone(inv);
    },
  },

  members: {
    async list(teamId: string): Promise<Member[]> {
      await delay();
      return DB.memberships.filter((m) => m.teamId === teamId).map((m) => {
        const u = DB.users.find((x) => x.id === m.userId)!;
        const roles = rolesOf(m);
        const dto: MemberDto = { membershipId: m.id, userId: u.id, name: u.name, email: u.email, phone: u.phone, birthday: u.birthday || '', address: u.address || '', avatarColor: u.avatarColor, photo: u.photo, group: m.group, roles: clone(roles), joinedAt: m.joinedAt };
        return mapMemberDtoToMember(dto, clone(primaryRole(roles)), mergePerms(roles));
      }).sort((a, b) => a.name.localeCompare(b.name, 'de'));
    },
    async add(teamId: string, { name, email, phone, roleIds, group, photo }: { name: string; email?: string; phone?: string; roleIds?: string[]; group?: string; photo?: string | null }) {
      await delay(280, 520);
      const u: User = { id: rid('u'), name, email: email || '', phone: phone || '', avatarColor: ['#1565C0', '#00796B', '#C2185B', '#5D4037', '#7B1FA2', '#455A64'][Math.floor(Math.random() * 6)], photo: photo || null, birthday: '', address: '' };
      DB.users.push(u);
      const mem: Membership = { id: rid('mem'), teamId, userId: u.id, roleIds: (roleIds && roleIds.length) ? roleIds : [DB.roles.find((r) => r.teamId === teamId && r.name === 'Tänzer / Mitglied')!.id], group: group || '', joinedAt: iso(new Date()) };
      DB.memberships.push(mem); persist(); return { membershipId: mem.id, userId: u.id };
    },
    async update(membershipId: string, { name, email, phone, birthday, address, roleIds, group, photo }: { name?: string; email?: string; phone?: string; birthday?: string; address?: string; roleIds?: string[]; group?: string; photo?: string | null }) {
      await delay(220, 420);
      const m = DB.memberships.find((x) => x.id === membershipId)!;
      const u = DB.users.find((x) => x.id === m.userId)!;
      if (name !== undefined) u.name = name;
      if (email !== undefined) u.email = email;
      if (phone !== undefined) u.phone = phone;
      if (birthday !== undefined) u.birthday = birthday;
      if (address !== undefined) u.address = address;
      if (photo !== undefined) u.photo = photo;
      if (roleIds !== undefined && roleIds.length) m.roleIds = roleIds;
      if (group !== undefined) m.group = group;
      persist(); return true;
    },
    async setRoles(membershipId: string, roleIds: string[]) {
      await delay(140, 300);
      const m = DB.memberships.find((x) => x.id === membershipId)!;
      if (roleIds.length) m.roleIds = roleIds;
      persist(); return true;
    },
    async remove(membershipId: string) {
      await delay(200, 400);
      DB.memberships = DB.memberships.filter((x) => x.id !== membershipId);
      persist(); return true;
    },
  },

  roles: {
    async list(teamId: string): Promise<Role[]> { await delay(90, 200); return clone(DB.roles.filter((r) => r.teamId === teamId)); },
    async create(teamId: string, { name, color, permissions }: { name: string; color?: string; permissions?: Permissions }): Promise<Role> {
      await delay(240, 440);
      const r: Role = { id: rid('role'), teamId, name, system: false, color: color || '#5B6470', permissions: permissions || perms('read', 'read', 'none', 'read', 'read', 'none') };
      DB.roles.push(r); persist(); return clone(r);
    },
    async update(roleId: string, patch: Partial<Role>): Promise<Role> {
      await delay(180, 360);
      const r = DB.roles.find((x) => x.id === roleId)!;
      Object.assign(r, patch); persist(); return clone(r);
    },
    async remove(roleId: string) {
      await delay(180, 360);
      DB.roles = DB.roles.filter((x) => x.id !== roleId);
      persist(); return true;
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
    async get(eventId: string): Promise<TeamEvent | null> {
      await delay(110, 220);
      const e = DB.events.find((x) => x.id === eventId);
      return e ? this._withSummary(e, e.teamId) : null;
    },
    _withSummary(e: EventDto, teamId: string): TeamEvent {
      const memberIds = DB.memberships.filter((m) => m.teamId === teamId).map((m) => m.userId);
      let yes = 0, no = 0, maybe = 0, pending = 0, notNom = 0;
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
      return mapEventDtoToTeamEvent(clone(e), { yes, no, maybe, pending, notNominated: notNom, nominated, total: memberIds.length }, { myStatus: mine.status, myAuto: mine.auto, myReason: mine.reason });
    },
    async create(teamId: string, payload: any): Promise<TeamEvent> {
      await delay(340, 600);
      const created: EventDto[] = [];
      const base = Object.assign({ teamId, recurring: false, meetTimeMandatory: true, location: '', note: '', responseMode: 'opt_in', status: 'active' }, payload);
      if (payload.recurring && payload.repeatWeeks > 1) {
        const seriesId = rid('series');
        for (let w = 0; w < payload.repeatWeeks; w++) {
          const d = parseDateOnlyLocal(payload.date);
          d.setDate(d.getDate() + w * 7);
          created.push(this._mk(Object.assign({}, base, { date: formatDateOnly(d), seriesId, recurring: true }), payload));
        }
      } else {
        created.push(this._mk(base, payload));
      }
      created.forEach((e) => DB.events.push(e));
      pushNotif({ teamId, type: 'event_created', actorId: session.userId!, title: created[0].title, eventId: created[0].id, eventTitle: created[0].title, eventDate: created[0].date, note: created.length > 1 ? ('Serie mit ' + created.length + ' Terminen') : '' });
      if (Array.isArray(payload.nominatedRoleIds)) {
        created.forEach((e) => applyNominations(e, payload.nominatedRoleIds));
      }
      persist(); return this._withSummary(created[0], teamId);
    },
    // Builds a persistable EventDto (not the enriched TeamEvent ViewModel) so the
    // result can be pushed straight into the DB.events collection.
    _mk(base: any, payload: any): EventDto {
      const mk = (h: string) => (h ? atTime(base.date, +h.slice(0, 2), +h.slice(3, 5)) : null);
      const nominatedRoleIds = Array.isArray(payload.nominatedRoleIds) ? [...payload.nominatedRoleIds] : undefined;
      return { id: rid('ev'), teamId: base.teamId, type: base.type, title: base.title, date: base.date, location: base.location, note: base.note || '', meetTime: payload.meetT ? mk(payload.meetT) : null, startTime: payload.startT ? mk(payload.startT) : null, endTime: payload.endT ? mk(payload.endT) : null, meetTimeMandatory: !!base.meetTimeMandatory, responseMode: base.responseMode || 'opt_in', nominatedRoleIds, recurring: !!base.recurring, seriesId: base.seriesId || null, status: 'active' } as EventDto;
    },
    async update(eventId: string, patch: any, scope: 'single' | 'series' = 'single'): Promise<TeamEvent> {
      await delay(260, 480);
      const e = DB.events.find((x) => x.id === eventId)!;
      const targets = (scope === 'series' && e.seriesId) ? DB.events.filter((x) => x.seriesId === e.seriesId) : [e];
      targets.forEach((ev) => {
        const baseDate = (scope !== 'series' && patch.date !== undefined) ? patch.date : ev.date;
        const mk = (h: string) => (h ? atTime(baseDate, +h.slice(0, 2), +h.slice(3, 5)) : null);
        if (scope !== 'series' && patch.date !== undefined) ev.date = patch.date;
        (['type', 'title', 'location', 'note', 'meetTimeMandatory', 'responseMode'] as const).forEach((k) => { if (patch[k] !== undefined) (ev as any)[k] = patch[k]; });
        if (patch.meetT !== undefined) ev.meetTime = patch.meetT ? mk(patch.meetT) : null;
        if (patch.startT !== undefined) ev.startTime = patch.startT ? mk(patch.startT) : null;
        if (patch.endT !== undefined) ev.endTime = patch.endT ? mk(patch.endT) : null;
        if (Array.isArray(patch.nominatedRoleIds)) applyNominations(ev, patch.nominatedRoleIds);
      });
      pushNotif({ teamId: e.teamId, type: 'event_updated', actorId: session.userId!, title: e.title, eventId: e.id, eventTitle: e.title, eventDate: e.date, note: scope === 'series' ? 'ganze Serie' : '' });
      persist(); return this._withSummary(e, e.teamId);
    },
    async setStatus(eventId: string, status: 'active' | 'cancelled', scope: 'single' | 'series' = 'single') {
      await delay(180, 360);
      const e = DB.events.find((x) => x.id === eventId);
      if (!e) return false;
      const targets = (scope === 'series' && e.seriesId) ? DB.events.filter((x) => x.seriesId === e.seriesId) : [e];
      targets.forEach((ev) => { ev.status = status; });
      pushNotif({ teamId: e.teamId, type: status === 'cancelled' ? 'event_cancelled' : 'event_reactivated', actorId: session.userId!, title: e.title, eventId: e.id, eventTitle: e.title, eventDate: e.date, note: scope === 'series' ? 'ganze Serie' : '' });
      persist(); return true;
    },
    async remove(eventId: string, scope: 'single' | 'series' = 'single') {
      await delay(200, 400);
      const e = DB.events.find((x) => x.id === eventId);
      const ids = (e && scope === 'series' && e.seriesId) ? DB.events.filter((x) => x.seriesId === e.seriesId).map((x) => x.id) : [eventId];
      if (e) pushNotif({ teamId: e.teamId, type: 'event_deleted', actorId: session.userId!, title: e.title, eventTitle: e.title, eventDate: e.date, note: scope === 'series' ? 'ganze Serie' : '' });
      DB.events = DB.events.filter((x) => !ids.includes(x.id));
      DB.attendance = DB.attendance.filter((a) => !ids.includes(a.eventId));
      DB.eventComments = DB.eventComments.filter((c) => !ids.includes(c.eventId));
      persist(); return true;
    },
    async listComments(eventId: string): Promise<EventComment[]> {
      await delay(110, 220);
      return DB.eventComments.filter((c) => c.eventId === eventId).sort((a, b) => a.createdAt.localeCompare(b.createdAt)).map((c) => {
        const u = DB.users.find((x) => x.id === c.userId);
        return Object.assign(clone(c), { name: u ? u.name : '', color: u ? u.avatarColor : '#888', photo: u ? u.photo : null });
      });
    },
    async addComment(eventId: string, text: string): Promise<EventComment> {
      await delay(160, 300);
      const c: EventComment = { id: rid('cm'), eventId, userId: session.userId!, text, createdAt: iso(new Date()) };
      DB.eventComments.push(c); persist(); return clone(c);
    },
    async removeComment(id: string) { await delay(140, 260); DB.eventComments = DB.eventComments.filter((x) => x.id !== id); persist(); return true; },
  },

  attendance: {
    async listForEvent(eventId: string): Promise<AttendanceRow[]> {
      await delay(130, 260);
      const e = DB.events.find((x) => x.id === eventId)!;
      const members = DB.memberships.filter((m) => m.teamId === e.teamId);
      const rows: AttendanceRow[] = members.map((m) => {
        const u = DB.users.find((x) => x.id === m.userId)!;
        const roles = rolesOf(m);
        const es = effectiveStatus(e, m.userId);
        const dto: AttendanceDto = { id: es.reasonId || `${eventId}:${u.id}`, eventId, userId: u.id, status: es.status, reason: es.reason, reasonId: es.reasonId, reasonVisibility: es.reasonVisibility };
        return mapAttendanceDtoToRow(dto, { userId: u.id, name: u.name, avatarColor: u.avatarColor, photo: u.photo, group: m.group, primaryRole: clone(primaryRole(roles)), auto: es.auto, absent: es.absent });
      });
      rows.sort((a, b) => (STATUS_ORDER[a.status] - STATUS_ORDER[b.status]) || a.name.localeCompare(b.name, 'de'));
      return rows;
    },
    async set(eventId: string, userId: string, { status, reason, reasonId, reasonVisibility }: { status: AttendanceStatus; reason?: string; reasonId?: string | null; reasonVisibility?: ReasonVisibility }) {
      await delay(160, 320);
      let a = DB.attendance.find((x) => x.eventId === eventId && x.userId === userId);
      if (!a) { a = { id: rid('att'), eventId, userId, status, reason: '', reasonId: null, reasonVisibility: null }; DB.attendance.push(a); }
      a.status = status; a.reason = reason || ''; a.reasonId = reasonId || null; a.reasonVisibility = reasonVisibility || null; a.at = iso(new Date());
      const e = DB.events.find((x) => x.id === eventId);
      if (e && (status === 'yes' || status === 'no' || status === 'maybe')) pushNotif({ teamId: e.teamId, type: 'attendance', actorId: userId, status, eventId: e.id, eventTitle: e.title, eventDate: e.date });
      persist(); return clone(a);
    },
    async setNomination(eventId: string, userId: string, nominated: boolean) {
      await delay(140, 280);
      if (nominated) {
        DB.attendance = DB.attendance.filter((x) => !(x.eventId === eventId && x.userId === userId));
      } else {
        let a = DB.attendance.find((x) => x.eventId === eventId && x.userId === userId);
        if (!a) { a = { id: rid('att'), eventId, userId, status: 'not_nominated', reason: '', reasonId: null, reasonVisibility: null }; DB.attendance.push(a); }
        a.status = 'not_nominated'; a.reason = ''; a.reasonId = null;
      }
      persist(); return true;
    },
  },

  absences: {
    async listForTeam(teamId: string): Promise<Absence[]> {
      await delay(120, 240);
      const memberIds = DB.memberships.filter((m) => m.teamId === teamId).map((m) => m.userId);
      return DB.absences.filter((a) => memberIds.includes(a.userId)).map((a) => {
        const u = DB.users.find((x) => x.id === a.userId)!;
        const m = DB.memberships.find((x) => x.teamId === teamId && x.userId === a.userId)!;
        const pr = primaryRole(rolesOf(m));
        return Object.assign(clone(a), { name: u.name, avatarColor: u.avatarColor, photo: u.photo, roleColor: pr ? pr.color : '#888', roleName: pr ? pr.name : '' });
      }).sort((a, b) => a.from.localeCompare(b.from));
    },
    async listMine(): Promise<Absence[]> { await delay(80, 180); return clone(DB.absences.filter((a) => a.userId === session.userId)).sort((a, b) => a.from.localeCompare(b.from)); },
    async create({ from, to, reason, userId }: { from: string; to: string; reason?: string; userId?: string }): Promise<Absence> {
      await delay(220, 420);
      const uid = userId || session.userId!;
      const a: Absence = { id: rid('abs'), userId: uid, from, to, reason: reason || 'Abwesend', createdAt: iso(new Date()) };
      DB.absences.push(a);
      const mem = DB.memberships.find((m) => m.userId === uid);
      if (mem) pushNotif({ teamId: mem.teamId, type: 'absence', actorId: uid, title: a.reason });
      persist(); return clone(a);
    },
    async update(id: string, { from, to, reason }: { from?: string; to?: string; reason?: string }): Promise<Absence> {
      await delay(180, 360);
      const a = DB.absences.find((x) => x.id === id)!;
      if (a) { if (from !== undefined) a.from = from; if (to !== undefined) a.to = to; if (reason !== undefined) a.reason = reason; }
      persist(); return clone(a);
    },
    async remove(id: string) { await delay(160, 300); DB.absences = DB.absences.filter((x) => x.id !== id); persist(); return true; },
  },

  news: {
    async list(teamId: string): Promise<NewsItem[]> {
      await delay(100, 220);
      return clone(DB.news.filter((n) => n.teamId === teamId)).map((n) => {
        const a = DB.users.find((u) => u.id === n.authorId);
        return Object.assign(n, { authorName: a ? a.name : '', authorColor: a ? a.avatarColor : '#888', authorPhoto: a ? a.photo : null });
      }).sort((a, b) => (Number(b.pinned) - Number(a.pinned)) || b.createdAt.localeCompare(a.createdAt));
    },
    async create(teamId: string, { title, body, pinned }: { title: string; body: string; pinned?: boolean }): Promise<NewsItem> {
      await delay(260, 480);
      const n: NewsItem = { id: rid('news'), teamId, title, body, authorId: session.userId!, pinned: !!pinned, createdAt: iso(new Date()) };
      DB.news.push(n); pushNotif({ teamId, type: 'news', actorId: session.userId!, title }); persist(); return clone(n);
    },
    async remove(id: string) { await delay(180, 340); DB.news = DB.news.filter((x) => x.id !== id); persist(); return true; },
  },

  finances: {
    async overview(teamId: string): Promise<FinanceOverview> {
      await delay(160, 320);
      const tx = DB.transactions.filter((x) => x.teamId === teamId).sort((a, b) => b.date.localeCompare(a.date));
      const income = tx.filter((t) => t.type === 'income').reduce((s, t) => s + t.amount, 0);
      const expense = tx.filter((t) => t.type === 'expense').reduce((s, t) => s + t.amount, 0);
      const penalties = clone(DB.penalties.filter((p) => p.teamId === teamId));
      const assignments = DB.penaltyAssignments.filter((p) => p.teamId === teamId).map((p) => {
        const u = DB.users.find((x) => x.id === p.userId)!;
        const pen = DB.penalties.find((x) => x.id === p.penaltyId)!;
        return Object.assign(clone(p), { name: u.name, avatarColor: u.avatarColor, photo: u.photo, label: pen.label, amount: pen.amount });
      });
      const openByUser: Record<string, number> = {};
      assignments.filter((a) => !a.paid).forEach((a) => { openByUser[a.userId] = (openByUser[a.userId] || 0) + (a.amount || 0); });
      const openPenalties = Object.keys(openByUser).map((uid) => {
        const u = DB.users.find((x) => x.id === uid)!;
        return { userId: uid, name: u.name, avatarColor: u.avatarColor, photo: u.photo, amount: openByUser[uid] };
      }).sort((a, b) => b.amount - a.amount);
      const contributions = DB.contributions.filter((c) => c.teamId === teamId).map((c) => {
        const u = DB.users.find((x) => x.id === c.userId)!;
        return Object.assign(clone(c), { name: u.name, avatarColor: u.avatarColor, photo: u.photo });
      }).sort((a, b) => a.name!.localeCompare(b.name!, 'de'));
      const contribOpen = contributions.filter((c) => c.status === 'open').length;
      return { balance: income - expense, income, expense, transactions: clone(tx), penalties, assignments, openPenalties, openPenaltySum: Object.values(openByUser).reduce((s, v) => s + v, 0), contributions, contribOpen };
    },
    async addTransaction(teamId: string, { type, title, amount, category }: { type: 'income' | 'expense'; title: string; amount: number | string; category?: string }): Promise<Transaction> {
      await delay(240, 440);
      const t: Transaction = { id: rid('tx'), teamId, type, title, amount: Number(amount), date: todayLocalDate(), category: category || 'Sonstiges' };
      DB.transactions.push(t); persist(); return clone(t);
    },
    async updateTransaction(id: string, { type, title, amount, category }: { type?: 'income' | 'expense'; title?: string; amount?: number | string; category?: string }): Promise<Transaction> {
      await delay(180, 360);
      const t = DB.transactions.find((x) => x.id === id)!;
      if (t) { if (type !== undefined) t.type = type; if (title !== undefined) t.title = title; if (amount !== undefined) t.amount = Number(amount); if (category !== undefined) t.category = category; }
      persist(); return clone(t);
    },
    async deleteTransaction(id: string) { await delay(160, 300); DB.transactions = DB.transactions.filter((x) => x.id !== id); persist(); return true; },
    async updatePenalty(id: string, { label, amount }: { label?: string; amount?: number | string }): Promise<Penalty> {
      await delay(160, 300);
      const p = DB.penalties.find((x) => x.id === id)!;
      if (p) { if (label !== undefined) p.label = label; if (amount !== undefined) p.amount = Number(amount); }
      persist(); return clone(p);
    },
    async updateContribution(id: string, { amount, label }: { amount?: number | string; label?: string }): Promise<Contribution> {
      await delay(160, 300);
      const c = DB.contributions.find((x) => x.id === id)!;
      if (c) { if (amount !== undefined) c.amount = Number(amount); if (label !== undefined) c.label = label; }
      persist(); return clone(c);
    },
    async createPenalty(teamId: string, { label, amount }: { label: string; amount: number | string }): Promise<Penalty> {
      await delay(200, 380);
      const p: Penalty = { id: rid('pen'), teamId, label, amount: Number(amount) };
      DB.penalties.push(p); persist(); return clone(p);
    },
    async deletePenalty(id: string) {
      await delay(160, 300);
      DB.penalties = DB.penalties.filter((x) => x.id !== id);
      DB.penaltyAssignments = DB.penaltyAssignments.filter((x) => x.penaltyId !== id);
      persist(); return true;
    },
    async assignPenalty(teamId: string, { userId, penaltyId }: { userId: string; penaltyId: string }): Promise<PenaltyAssignment> {
      await delay(200, 380);
      const a: PenaltyAssignment = { id: rid('pa'), teamId, userId, penaltyId, paid: false, date: todayLocalDate() };
      DB.penaltyAssignments.push(a); persist(); return clone(a);
    },
    async deleteAssignment(id: string) { await delay(160, 300); DB.penaltyAssignments = DB.penaltyAssignments.filter((x) => x.id !== id); persist(); return true; },
    async togglePenaltyPaid(assignmentId: string) {
      await delay(140, 280);
      const a = DB.penaltyAssignments.find((x) => x.id === assignmentId);
      if (a) a.paid = !a.paid;
      persist(); return true;
    },
    async toggleContribution(contribId: string) {
      await delay(140, 280);
      const c = DB.contributions.find((x) => x.id === contribId);
      if (c) c.status = c.status === 'paid' ? 'open' : 'paid';
      persist(); return true;
    },
  },

  stats: {
    async attendanceFor(teamId: string, userId: string) {
      await delay(110, 220);
      const today = todayLocalDate();
      const past = DB.events.filter((e) => e.teamId === teamId && e.date < today && e.status !== 'cancelled');
      let yes = 0, counted = 0;
      past.forEach((e) => { const s = effectiveStatus(e, userId).status; if (s === 'not_nominated') return; counted++; if (s === 'yes') yes++; });
      return { quote: counted ? Math.round((yes / counted) * 100) : null, counted, yes };
    },
    async teamOverview(teamId: string, range?: DateRange | null): Promise<StatsOverview> {
      await delay(180, 360);
      const today = todayLocalDate();
      const from = range && range.from ? range.from : null;
      const to = range && range.to ? range.to : null;
      const members = DB.memberships.filter((m) => m.teamId === teamId);
      let past = DB.events.filter((e) => e.teamId === teamId && e.date < today && e.status !== 'cancelled');
      if (from) past = past.filter((e) => e.date >= from);
      if (to) past = past.filter((e) => e.date <= to);
      past = past.sort((a, b) => b.date.localeCompare(a.date));
      const memberStats = members.map((m) => {
        const u = DB.users.find((x) => x.id === m.userId)!;
        let yes = 0, counted = 0;
        past.forEach((e) => { const s = effectiveStatus(e, u.id).status; if (s === 'not_nominated') return; counted++; if (s === 'yes') yes++; });
        return { userId: u.id, name: u.name, avatarColor: u.avatarColor, photo: u.photo, quote: counted ? Math.round((yes / counted) * 100) : null, counted, yes };
      }).sort((a, b) => (b.quote || 0) - (a.quote || 0));
      const quotes = memberStats.filter((s) => s.quote !== null).map((s) => s.quote!);
      const avg = quotes.length ? Math.round(quotes.reduce((s, q) => s + q, 0) / quotes.length) : 0;
      const eventStats = past.slice(0, 8).map((e) => {
        const sum = api.events._withSummary(e, teamId).summary;
        const pct = sum.nominated ? Math.round(sum.yes / sum.nominated * 100) : 0;
        return { id: e.id, title: e.title, type: e.type, date: e.date, yes: sum.yes, nominated: sum.nominated, pct, enough: pct >= 80 };
      }).reverse();
      return { avg, members: memberStats, events: eventStats, pastCount: past.length, from, to };
    },
  },

  polls: {
    async list(teamId: string): Promise<Poll[]> {
      await delay(120, 240);
      return DB.polls.filter((p) => p.teamId === teamId).map((p) => {
        const total = p.votes.length;
        const counts: Record<string, number> = {};
        p.options.forEach((o: any) => { counts[o.id] = 0; });
        p.votes.forEach((v: any) => v.optionIds.forEach((oid: string) => { if (counts[oid] !== undefined) counts[oid]++; }));
        const mine = p.votes.find((v: any) => v.userId === session.userId);
        return {
          id: p.id, question: p.question, multiple: p.multiple, anonymous: p.anonymous, createdAt: p.createdAt, totalVotes: total,
          myVote: mine ? clone(mine.optionIds) : null,
          options: p.options.map((o: any) => ({
            id: o.id, text: o.text, count: counts[o.id], pct: total ? Math.round(counts[o.id] / total * 100) : 0,
            voters: p.anonymous ? [] : p.votes.filter((v: any) => v.optionIds.includes(o.id)).map((v: any) => { const u = DB.users.find((x) => x.id === v.userId)!; return { name: u.name, color: u.avatarColor, photo: u.photo }; }),
          })),
        };
      }).sort((a, b) => b.createdAt.localeCompare(a.createdAt));
    },
    async vote(pollId: string, optionIds: string[]) {
      await delay(160, 320);
      const p = DB.polls.find((x) => x.id === pollId)!;
      p.votes = p.votes.filter((v: any) => v.userId !== session.userId);
      if (optionIds.length) p.votes.push({ userId: session.userId, optionIds: p.multiple ? optionIds : [optionIds[0]] });
      persist(); return true;
    },
    async create(teamId: string, { question, options, multiple, anonymous }: { question: string; options: string[]; multiple?: boolean; anonymous?: boolean }) {
      await delay(260, 480);
      const poll = { id: rid('poll'), teamId, question, multiple: !!multiple, anonymous: !!anonymous, createdAt: iso(new Date()), options: options.filter((o) => o.trim()).map((o, i) => ({ id: 'opt' + i + '_' + rid('o'), text: o.trim() })), votes: [] };
      DB.polls.push(poll); pushNotif({ teamId, type: 'poll', actorId: session.userId!, title: question }); persist(); return clone(poll);
    },
    async remove(id: string) { await delay(180, 340); DB.polls = DB.polls.filter((x) => x.id !== id); persist(); return true; },
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
          return Object.assign(clone(n), { actorName: u ? u.name : '', actorColor: u ? u.avatarColor : '#888', actorPhoto: u ? u.photo : null, unread: seen ? n.createdAt > seen : true });
        });
      const unreadCount = items.filter((n) => n.unread).length;
      return { items, unreadCount };
    },
    async markSeen(teamId: string) { await delay(40, 100); DB.notifSeen = DB.notifSeen || {}; DB.notifSeen[teamId] = iso(new Date()); persist(); return true; },
  },

  MODULES,
};

export const MODULE_LABELS: Record<ModuleKey, string> = { events: 'Termine', members: 'Mitglieder', finances: 'Finanzen', news: 'Neuigkeiten', polls: 'Umfragen', settings: 'Einstellungen' };
export const STATUS_ORDER_EXPORT = STATUS_ORDER;
export type Api = typeof api;
