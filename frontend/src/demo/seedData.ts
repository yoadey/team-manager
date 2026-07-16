// Demo seed data for the local mock service layer.

import type {
  AttendanceStatus,
  Invite,
  Membership,
  Permissions,
  PermLevel,
  ReasonVisibility,
  Role,
  RoleDto,
  Team,
  User,
} from '@/types';
import type { Absence, AttendanceDto, EventComment, EventDto, ResponseMode } from '@/features/events';
import type { Contribution, Penalty, PenaltyAssignment, Transaction } from '@/features/finances';
import type { NewsItem } from '@/features/news';
import type { AppNotification } from '@/features/notifications';
import type { PollDto } from '@/features/polls';
import { formatDateOnly, todayLocalDate } from '@/utils/date';

const rid = (p: string) => p + '_' + Math.random().toString(36).slice(2, 9);
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
  d.setDate(d.getDate() + diff + weeks * 7);
  return d;
}
function plusDays(n: number) {
  const d = new Date();
  d.setHours(0, 0, 0, 0);
  d.setDate(d.getDate() + n);
  return formatDateOnly(d);
}

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

export interface DemoDb {
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
  polls: PollDto[];
  eventComments: EventComment[];
  notifications: AppNotification[];
  notifSeen: Record<string, string>;
  meta?: { seededAt: string; version: number };
}

// ---- Seed -------------------------------------------------------------------
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
  };

  const U = (id: string, name: string, email: string, phone: string, color: string): User => ({
    id,
    name,
    email,
    phone,
    avatarColor: color,
    photo: null,
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

  const tA: Team = {
    id: 't_a',
    name: 'A-Team TSC Schwarz-Gelb Aachen',
    short: 'A',
    icon: '🏆',
    iconBg: '#1A1A1A',
    iconFg: '#F5C518',
    photo: null,
    logo: null,
    description: '',
  };
  const tB: Team = {
    id: 't_b',
    name: 'B-Team TSC Schwarz-Gelb Aachen',
    short: 'B',
    icon: '⭐',
    iconBg: '#1A1A1A',
    iconFg: '#F5C518',
    photo: null,
    logo: null,
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
  tA.description = 'A-Formation Latein – aktuell in der NRW-Liga. Training Di & Do in Eilendorf.';
  tB.description = 'B-Formation – Nachwuchs & Aufbau. Wir freuen uns über jede neue Tänzerin und jeden neuen Tänzer.';
  const setProfile = (uid: string, bd: string, addr: string) => {
    const u = db.users.find((x) => x.id === uid);
    if (u) {
      u.birthday = bd;
      u.address = addr;
    }
  };
  setProfile('u1', '1998-04-12', 'Jülicher Straße 12, 52070 Aachen');
  setProfile('u2', '1996-09-30', 'Adalbertsteinweg 88, 52070 Aachen');
  setProfile('u3', '2000-01-23', 'Pontstraße 45, 52062 Aachen');
  setProfile('u4', '1999-07-08', 'Vaalser Straße 210, 52074 Aachen');

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

  // ---- Termine A-Team ----
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
        meetTime: atTime(d, 19, 15),
        startTime: atTime(d, 19, 30),
        endTime: atTime(d, 21, 30),
        meetTimeMandatory: true,
        location: 'Tanzsporthalle Eilendorf',
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
    meetTime: atTime(turnierD, 9, 0),
    startTime: atTime(turnierD, 11, 0),
    endTime: atTime(turnierD, 18, 0),
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
    meetTime: atTime(grillD, 18, 0),
    startTime: atTime(grillD, 18, 0),
    endTime: atTime(grillD, 23, 0),
    meetTimeMandatory: false,
    location: 'Vereinsheim, Aachen-Brand',
    responseMode: 'opt_in',
    note: 'Bitte Salate/Beilagen in der Umfrage eintragen.',
  });
  const pastTurnier = nextWeekday(6, -2);
  E({
    type: 'auftritt',
    title: 'DM-Qualifikation Latein – 1. Wertung',
    date: dstr(pastTurnier),
    meetTime: atTime(pastTurnier, 8, 30),
    startTime: atTime(pastTurnier, 10, 30),
    endTime: atTime(pastTurnier, 17, 0),
    meetTimeMandatory: true,
    location: 'Stadthalle Braunschweig',
    responseMode: 'opt_in',
    result: 'Platz 3 von 11 – Aufstieg gesichert',
  });
  db.events = ev;

  // ---- Anwesenheit ----
  const att: any[] = [];
  const A = (
    eventId: string,
    userId: string,
    status: AttendanceStatus,
    reason?: string,
    reasonId?: string | null,
    vis?: ReasonVisibility,
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

  // ---- Abwesenheiten (geplant) ----
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

  // ---- News ----
  db.news = [
    {
      id: rid('news'),
      teamId: 't_a',
      title: 'Aufstieg in die NRW-Liga! 🎉',
      body: 'Mit Platz 3 bei der DM-Qualifikation haben wir den Aufstieg sicher. Riesen Kompliment an die ganze Formation – jetzt greifen wir in der NRW-Liga an!',
      authorId: 'u1',
      pinned: true,
      createdAt: iso(new Date(Date.now() - 2 * DAY)),
    },
    {
      id: rid('news'),
      teamId: 't_a',
      title: 'Neue Trainingszeiten ab Juli',
      body: 'Ab Juli trainieren wir zusätzlich freitags. Details folgen, bitte Kalender im Blick behalten.',
      authorId: 'u2',
      pinned: false,
      createdAt: iso(new Date(Date.now() - 5 * DAY)),
    },
    {
      id: rid('news'),
      teamId: 't_a',
      title: 'Turnieranmeldung Düsseldorf abgeschlossen',
      body: 'Alle gemeldeten Paare sind bestätigt. Treffpunkt und Ablauf stehen im Termin zur 2. Wertung.',
      authorId: 'u1',
      pinned: false,
      createdAt: iso(new Date(Date.now() - 8 * DAY)),
    },
  ];

  // ---- Finanzen ----
  const T = (type: 'income' | 'expense', title: string, amount: number, daysAgo: number, cat: string) =>
    db.transactions.push({
      id: rid('tx'),
      teamId: 't_a',
      type,
      title,
      amount,
      date: plusDays(-daysAgo),
      category: cat,
    });
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
    db.penaltyAssignments.push({
      id: rid('pa'),
      teamId: 't_a',
      userId,
      penaltyId: pid(penIdx),
      paid,
      date: plusDays(-daysAgo),
    });
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
      amount: 25,
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

  // ---- Umfragen ----
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

  const _ft = ev.find((e) => e.type === 'training' && e.date >= todayLocalDate());
  if (_ft) {
    db.eventComments = [
      {
        id: rid('cm'),
        eventId: _ft.id,
        userId: 'u2',
        text: 'Bringe die neue Musik für die Kür mit.',
        createdAt: iso(new Date(Date.now() - 2 * 3600 * 1000)),
      },
      {
        id: rid('cm'),
        eventId: _ft.id,
        userId: 'u1',
        text: 'Perfekt – wir starten mit der Eröffnung, bitte pünktlich.',
        createdAt: iso(new Date(Date.now() - 1 * 3600 * 1000)),
      },
    ];
  }

  // ---- Benachrichtigungen (letzte ~2 Monate) ----
  const ntf: AppNotification[] = [];
  const hAgo = (h: number) => iso(new Date(Date.now() - h * 3600 * 1000));
  const dAgo = (d: number) => iso(new Date(Date.now() - d * DAY));
  const N = (o: Partial<AppNotification>) =>
    ntf.push(Object.assign({ id: rid('ntf'), teamId: 't_a' }, o) as AppNotification);
  const upTrain = ev
    .filter((e) => e.type === 'training' && e.date >= todayLocalDate())
    .sort((a, b) => a.date.localeCompare(b.date))[0];
  const trn = ev.find((e) => e.title.startsWith('NRW-Liga'));
  const grl = ev.find((e) => e.type === 'event');
  N({ type: 'event_created', actorId: 'u1', title: 'Lateinformation – Training (Serie Di & Do)', createdAt: dAgo(48) });
  N({
    type: 'event_created',
    actorId: 'u1',
    title: trn ? trn.title : 'NRW-Liga',
    eventId: trn ? trn.id : null,
    eventTitle: trn ? trn.title : '',
    eventDate: trn ? trn.date : '',
    createdAt: dAgo(11),
  });
  N({
    type: 'event_updated',
    actorId: 'u2',
    title: 'Lateinformation – Training',
    createdAt: dAgo(9),
    note: 'Treffzeit auf 19:15 vorgezogen',
  });
  N({
    type: 'event_created',
    actorId: 'u1',
    title: grl ? grl.title : 'Saison-Kickoff Grillen',
    eventId: grl ? grl.id : null,
    eventTitle: grl ? grl.title : '',
    eventDate: grl ? grl.date : '',
    createdAt: dAgo(6),
  });
  db.news.forEach((n) => N({ type: 'news', actorId: n.authorId, title: n.title, createdAt: n.createdAt }));
  N({ type: 'poll', actorId: 'u1', title: 'Neue Turnierkleidung – welche Farbe?', createdAt: dAgo(4) });
  N({ type: 'poll', actorId: 'u2', title: 'Was bringst du zum Grillen mit?', createdAt: dAgo(1) });
  N({ type: 'absence', actorId: 'u6', title: 'Urlaub (Italien)', createdAt: dAgo(7) });
  N({ type: 'absence', actorId: 'u10', title: 'Klassenfahrt', createdAt: dAgo(3) });
  if (trn) {
    const trnResp: [string, AttendanceStatus, number][] = [
      ['u2', 'yes', 30],
      ['u3', 'yes', 28],
      ['u4', 'yes', 26],
      ['u5', 'yes', 22],
      ['u8', 'no', 20],
      ['u9', 'maybe', 14],
      ['u7', 'yes', 10],
      ['u12', 'yes', 6],
      ['u8', 'yes', 4],
    ];
    trnResp.forEach(([uid, st, h]) =>
      N({
        type: 'attendance',
        actorId: uid,
        status: st,
        eventId: trn.id,
        eventTitle: trn.title,
        eventDate: trn.date,
        createdAt: hAgo(h),
      }),
    );
  }
  if (upTrain) {
    const trResp: [string, AttendanceStatus, number][] = [
      ['u4', 'yes', 18],
      ['u5', 'no', 16],
      ['u7', 'maybe', 12],
      ['u8', 'yes', 5],
      ['u9', 'maybe', 3],
      ['u12', 'yes', 2],
    ];
    trResp.forEach(([uid, st, h]) =>
      N({
        type: 'attendance',
        actorId: uid,
        status: st,
        eventId: upTrain.id,
        eventTitle: upTrain.title,
        eventDate: upTrain.date,
        createdAt: hAgo(h),
      }),
    );
  }
  db.notifications = ntf;
  db.notifSeen = { t_a: hAgo(30) };

  db.meta = { seededAt: iso(new Date()), version: 6 };
  return db;
}
