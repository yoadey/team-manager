// MSW request handlers for the demo/test backend — one per OpenAPI operation
// actually called by serviceLayerReal.ts. Response bodies are explicitly
// typed against `components['schemas']` (src/api/types.gen.ts) so a future
// OpenAPI spec change that isn't mirrored here surfaces as a TypeScript error,
// not a silent runtime drift.
import { http, HttpResponse } from 'msw';
import type { components } from '@/api/types.gen';
import {
  db,
  session,
  rid,
  rolesOf,
  primaryRole,
  mergePerms,
  effectiveStatus,
  rawCountedStatus,
  threeMonthsBeforeLocal,
  applyNominations,
  pushNotif,
  DEFAULT_MEMBER_ROLE_NAME,
  DEMO_PASSWORD,
  DEMO_LOGIN_EMAIL,
  DEMO_LOGIN_USER_ID,
  DEMO_SSO_PROVIDER_IDS,
} from './db';
import type { UserRow, TeamRow } from './db';
import type { RoleDto } from '@/types';
import type { EventDto } from '@/features/events';
import { formatDateOnly, parseDateOnlyLocal, todayLocalDate } from '@/utils/date';

type S = components['schemas'];

// A small artificial per-request delay, applied to every handler below.
// Without it, two chained requests issued a tick apart (e.g. AppContext's
// session-restore effect awaiting `teams.listForCurrentUser()` then, once
// that resolves, `openEventDetail()` awaiting `events.get()`) can resolve
// within the same microtask turn and get batched into a single React commit
// — collapsing state transitions that calling code (e.g. the URL-sync
// effect's `lastSyncedPath` ref diffing) assumes happen as observably
// separate steps. A real backend never resolves same-tick, so this restores
// that same-origin-but-not-synchronous realism (mirroring the deleted
// localStorage mock's `delay()`, and honoring VITE_MOCK_DELAY_MIN/MAX).
const loginDelay = () => new Promise<void>((r) => setTimeout(r, 40));
const mockDelay = () => new Promise<void>((r) => setTimeout(r, 5));

// Every route is registered relative (no origin) so it matches both the
// browser worker (resolved against location.origin) and the node
// setupServer used in tests (resolved against jsdom's default origin) —
// see src/api/client.ts's baseUrl (`config.apiBaseUrl + '/api/v1'`, empty in
// demo mode, so requests are same-origin relative URLs).
const P = (path: string) => `*/api/v1${path}`;

function problem(status: number, detail: string): HttpResponse<S['Problem']> {
  const body: S['Problem'] = { status, title: detail, detail };
  return HttpResponse.json(body, { status });
}

function requireAuth(): string | HttpResponse<S['Problem']> {
  if (!session.userId) return problem(401, 'Not authenticated');
  return session.userId;
}

function requireUser(id: string): UserRow {
  const u = db.users.find((x) => x.id === id);
  if (!u) throw new Error(`mocks/handlers: unknown user ${id}`);
  return u;
}

function toWireUser(u: UserRow): S['User'] {
  return { id: u.id, name: u.name, email: u.email, phone: u.phone || undefined, avatarColor: u.avatarColor, birthday: u.birthday || undefined, address: u.address || undefined, hasPhoto: u.hasPhoto };
}

// ---- self-registration (mock equivalent of auth.Service.Register etc.) ----

// Fixed, generic response for register/resend-verification, returned
// identically regardless of account state -- mirrors the real backend's
// enumeration-safety contract (see openspec/changes/self-service-registration).
const REGISTRATION_ACCEPTED_MESSAGE = 'If this email can be registered, a verification link has been sent.';
const verificationTokenTTLMs = 48 * 60 * 60 * 1000;

function issueVerificationToken(userId: string): void {
  // rid() is fine for demo entity ids but not for this: the token stands in
  // for a secret, unguessable value, the same property the real backend's
  // crypto/rand-generated token has. crypto.randomUUID() is a CSPRNG, so it
  // is used here instead of rid().
  const token = crypto.randomUUID();
  db.verificationTokens[token] = { userId, expiresAt: new Date(Date.now() + verificationTokenTTLMs).toISOString() };
}

function toWireRole(r: RoleDto): S['Role'] {
  return { id: r.id, teamId: r.teamId, name: r.name, system: r.system, color: r.color, permissions: r.permissions };
}

function toWireTeam(t: TeamRow): S['Team'] {
  return {
    id: t.id,
    name: t.name,
    short: t.short,
    icon: t.icon,
    iconBg: t.iconBg,
    iconFg: t.iconFg,
    description: t.description,
    hasPhoto: t.hasPhoto,
    hasLogo: t.hasLogo,
    reasonVisibilityRoleIds: t.reasonVisibilityRoles,
  };
}

function toWireTeamForUser(t: TeamRow, userId: string): S['TeamForUser'] {
  const m = db.memberships.find((x) => x.teamId === t.id && x.userId === userId)!;
  const roles = rolesOf(m);
  return {
    ...toWireTeam(t),
    myRoles: roles.map(toWireRole),
    myPerms: mergePerms(roles),
    membershipId: m.id,
    memberCount: db.memberships.filter((x) => x.teamId === t.id).length,
  };
}

function toWireMember(m: (typeof db.memberships)[number]): S['Member'] {
  const u = requireUser(m.userId);
  const roles = rolesOf(m);
  const pr = primaryRole(roles);
  return {
    membershipId: m.id,
    userId: u.id,
    name: u.name,
    email: u.email,
    phone: u.phone || undefined,
    birthday: u.birthday || undefined,
    address: u.address || undefined,
    avatarColor: u.avatarColor,
    hasPhoto: u.hasPhoto,
    group: m.group || undefined,
    roles: roles.map(toWireRole),
    primaryRole: pr ? toWireRole(pr) : undefined,
    perms: mergePerms(roles),
    joinedAt: m.joinedAt,
  };
}

function eventSummary(e: EventDto, teamId: string): S['EventSummary'] {
  const memberIds = db.memberships.filter((m) => m.teamId === teamId).map((m) => m.userId);
  let yes = 0, no = 0, maybe = 0, pending = 0, notNom = 0;
  memberIds.forEach((uid) => {
    const s = effectiveStatus(e, uid).status;
    if (s === 'yes') yes++;
    else if (s === 'no') no++;
    else if (s === 'maybe') maybe++;
    else if (s === 'not_nominated') notNom++;
    else pending++;
  });
  return { yes, no, maybe, pending, notNominated: notNom, nominated: memberIds.length - notNom, total: memberIds.length };
}

function toWireEvent(e: EventDto): S['TeamEvent'] {
  const mine = effectiveStatus(e, session.userId);
  return {
    id: e.id,
    teamId: e.teamId,
    seriesId: e.seriesId ?? undefined,
    type: e.type,
    title: e.title,
    date: e.date,
    location: e.location || undefined,
    note: e.note || undefined,
    result: e.result,
    meetTime: e.meetTime ?? undefined,
    startTime: e.startTime ?? undefined,
    endTime: e.endTime ?? undefined,
    meetTimeMandatory: e.meetTimeMandatory,
    responseMode: e.responseMode,
    nominatedRoleIds: e.nominatedRoleIds,
    recurring: e.recurring,
    status: e.status,
    summary: eventSummary(e, e.teamId),
    myStatus: mine.status,
    myAuto: mine.auto,
    myReason: mine.reason,
  };
}

function toWireAttendanceRow(e: EventDto, m: (typeof db.memberships)[number]): S['AttendanceRow'] {
  const u = requireUser(m.userId);
  const roles = rolesOf(m);
  const es = effectiveStatus(e, m.userId);
  const pr = primaryRole(roles);
  return {
    userId: u.id,
    name: u.name,
    avatarColor: u.avatarColor,
    hasPhoto: u.hasPhoto,
    group: m.group || undefined,
    primaryRole: pr ? toWireRole(pr) : undefined,
    status: es.status,
    reason: es.reason || undefined,
    reasonId: es.reasonId ?? undefined,
    reasonVisibility: es.reasonVisibility ?? undefined,
    auto: es.auto,
    absent: es.absent,
  };
}

function toWireComment(c: (typeof db.eventComments)[number]): S['EventComment'] {
  const u = db.users.find((x) => x.id === c.userId);
  return { id: c.id, eventId: c.eventId, userId: c.userId, text: c.text, createdAt: c.createdAt, authorName: u?.name, authorColor: u?.avatarColor, hasAuthorPhoto: u?.hasPhoto };
}

function toWireAbsence(a: (typeof db.absences)[number], teamId: string): S['Absence'] {
  const u = db.users.find((x) => x.id === a.userId);
  const m = db.memberships.find((x) => x.teamId === teamId && x.userId === a.userId);
  const pr = m ? primaryRole(rolesOf(m)) : null;
  return { id: a.id, userId: a.userId, from: a.from, to: a.to, reason: a.reason || undefined, createdAt: a.createdAt, memberName: u?.name, memberAvatarColor: u?.avatarColor, hasPhoto: u?.hasPhoto, roleColor: pr?.color, roleName: pr?.name };
}

function toWireNews(n: (typeof db.news)[number]): S['NewsItem'] {
  const u = db.users.find((x) => x.id === n.authorId);
  return { id: n.id, teamId: n.teamId, authorId: n.authorId, title: n.title, body: n.body, pinned: n.pinned, createdAt: n.createdAt, authorName: u?.name, authorColor: u?.avatarColor, hasAuthorPhoto: u?.hasPhoto };
}

function toWirePoll(p: (typeof db.polls)[number]): S['Poll'] {
  const total = p.votes.length;
  const counts: Record<string, number> = {};
  p.options.forEach((o) => { counts[o.id] = 0; });
  p.votes.forEach((v) => v.optionIds.forEach((oid) => { if (counts[oid] !== undefined) counts[oid]++; }));
  const mine = p.votes.find((v) => v.userId === session.userId);
  return {
    id: p.id,
    question: p.question,
    multiple: p.multiple,
    anonymous: p.anonymous,
    createdAt: p.createdAt,
    totalVotes: total,
    myVote: mine ? [...mine.optionIds] : undefined,
    options: p.options.map((o) => ({
      id: o.id,
      text: o.text,
      count: counts[o.id],
      pct: total ? Math.round((counts[o.id] / total) * 100) : 0,
      voters: p.anonymous
        ? []
        : p.votes.filter((v) => v.optionIds.includes(o.id)).map((v) => {
            const u = db.users.find((x) => x.id === v.userId);
            return { name: u?.name, color: u?.avatarColor, hasPhoto: u?.hasPhoto };
          }),
    })),
  };
}

function toWireNotification(n: (typeof db.notifications)[number], seen: string | null): S['AppNotification'] {
  const u = db.users.find((x) => x.id === n.actorId);
  return {
    id: n.id,
    teamId: n.teamId,
    type: n.type,
    actorId: n.actorId,
    status: n.status,
    title: n.title,
    eventId: n.eventId ?? undefined,
    eventTitle: n.eventTitle,
    eventDate: n.eventDate,
    note: n.note,
    createdAt: n.createdAt,
    actorName: u?.name,
    actorColor: u?.avatarColor,
    hasActorPhoto: u?.hasPhoto,
    unread: seen ? n.createdAt > seen : true,
  };
}

function toWireTransaction(t: (typeof db.transactions)[number]): S['Transaction'] {
  return { id: t.id, teamId: t.teamId, type: t.type, title: t.title, amount: t.amount, date: t.date, category: t.category || undefined };
}
function toWirePenalty(p: (typeof db.penalties)[number]): S['Penalty'] {
  return { id: p.id, teamId: p.teamId, label: p.label, amount: p.amount };
}
function toWireAssignment(a: (typeof db.penaltyAssignments)[number]): S['PenaltyAssignment'] {
  const u = db.users.find((x) => x.id === a.userId);
  return { id: a.id, teamId: a.teamId, userId: a.userId, penaltyId: a.penaltyId, paid: a.paid, date: a.date, memberName: u?.name, memberAvatarColor: u?.avatarColor, hasPhoto: u?.hasPhoto, label: a.label, amount: a.amount };
}
function toWireContribution(c: (typeof db.contributions)[number]): S['Contribution'] {
  const u = db.users.find((x) => x.id === c.userId);
  return { id: c.id, teamId: c.teamId, userId: c.userId, month: c.month, label: c.label || undefined, amount: c.amount, status: c.status, memberName: u?.name, memberAvatarColor: u?.avatarColor, hasPhoto: u?.hasPhoto };
}

function eventDate(id: string): EventDto | undefined {
  return db.events.find((e) => e.id === id);
}

export const handlers = [
  // ---- auth ----
  http.get(P('/auth/providers'), async () => {
    await mockDelay();
    const body: S['Provider'][] = [
      { id: 'password', name: 'Passwort', sub: DEMO_LOGIN_EMAIL + ' / ' + DEMO_PASSWORD, glyph: 'P', bg: '#1565C0', fg: '#FFFFFF' },
    ];
    return HttpResponse.json(body);
  }),

  // Demo-only. Two accepted paths:
  //  1. The fixed demo email + password (see db.ts's DEMO_PASSWORD/
  //     DEMO_LOGIN_EMAIL) — a wrong/missing password is rejected with 401,
  //     unlike the old localStorage mock, which ignored the password
  //     entirely (see proposal.md's security smell).
  //  2. One of DEMO_SSO_PROVIDER_IDS as the "email" field with no password —
  //     the app's Login screen historically offered one-tap SSO buttons
  //     (never backed by a real OIDC flow; /auth/providers now only
  //     advertises "password", so these no longer render) that call
  //     `api.auth.login(providerId)`; kept working here as a demo
  //     convenience distinct from (and not weakening) the password path.
  http.post(P('/auth/login'), async ({ request }) => {
    const body = (await request.json()) as S['LoginRequest'];
    await loginDelay();
    if (DEMO_SSO_PROVIDER_IDS.includes(body.email)) {
      const u = requireUser(DEMO_LOGIN_USER_ID);
      session.userId = u.id;
      const resp: S['LoginResponse'] = { token: 'demo.' + crypto.randomUUID(), user: toWireUser(u) };
      return HttpResponse.json(resp, { headers: { 'Set-Cookie': 'tv_session=demo; Path=/; SameSite=Lax' } });
    }
    const u = db.users.find((x) => x.email.toLowerCase() === body.email?.toLowerCase());
    // Self-registered accounts (created via POST /auth/register) carry their
    // own password on the row; the fixed demo account never has one set.
    if (u?.password !== undefined) {
      if (body.password !== u.password) return problem(401, 'Invalid email or password');
      if (!u.emailVerifiedAt) return problem(403, 'please verify your email before logging in');
      session.userId = u.id;
      const resp: S['LoginResponse'] = { token: 'demo.' + crypto.randomUUID(), user: toWireUser(u) };
      return HttpResponse.json(resp, { headers: { 'Set-Cookie': 'tv_session=demo; Path=/; SameSite=Lax' } });
    }
    if (!u || u.id !== DEMO_LOGIN_USER_ID || body.password !== DEMO_PASSWORD) {
      return problem(401, 'Invalid email or password');
    }
    session.userId = u.id;
    const resp: S['LoginResponse'] = { token: 'demo.' + crypto.randomUUID(), user: toWireUser(u) };
    return HttpResponse.json(resp, { headers: { 'Set-Cookie': 'tv_session=demo; Path=/; SameSite=Lax' } });
  }),

  // Enumeration-safe: the response is identical whether the email was
  // available, already registered and verified, or already registered and
  // still pending -- see Service.Register's design on the backend.
  http.post(P('/auth/register'), async ({ request }) => {
    const body = (await request.json()) as S['RegisterRequest'];
    await mockDelay();
    const email = (body.email ?? '').toLowerCase();
    const resp: S['RegisterResponse'] = { message: REGISTRATION_ACCEPTED_MESSAGE };

    const existing = db.users.find((x) => x.email.toLowerCase() === email);
    if (existing) {
      // Verified: leave the account and its password untouched, no token issued.
      // Still-pending: issue a fresh token, but never overwrite the password.
      if (!existing.emailVerifiedAt) issueVerificationToken(existing.id);
      return HttpResponse.json(resp, { status: 202 });
    }

    const id = rid('u');
    const newUser: UserRow = {
      id,
      name: email.split('@')[0] || email,
      email: body.email,
      phone: '',
      avatarColor: '#6366f1',
      photo: null,
      hasPhoto: false,
      birthday: '',
      address: '',
      password: body.password,
      emailVerifiedAt: null,
    };
    db.users.push(newUser);
    issueVerificationToken(id);
    return HttpResponse.json(resp, { status: 202 });
  }),

  http.post(P('/auth/verify-email'), async ({ request }) => {
    const body = (await request.json()) as S['VerifyEmailRequest'];
    await mockDelay();
    const entry = body.token ? db.verificationTokens[body.token] : undefined;
    if (!entry || new Date(entry.expiresAt).getTime() < Date.now()) {
      return problem(401, 'Invalid or expired verification token');
    }
    delete db.verificationTokens[body.token];
    const u = requireUser(entry.userId);
    u.emailVerifiedAt = new Date().toISOString();
    session.userId = u.id;
    const resp: S['LoginResponse'] = { token: 'demo.' + crypto.randomUUID(), user: toWireUser(u) };
    return HttpResponse.json(resp, { headers: { 'Set-Cookie': 'tv_session=demo; Path=/; SameSite=Lax' } });
  }),

  http.post(P('/auth/resend-verification'), async ({ request }) => {
    const body = (await request.json()) as S['ResendVerificationRequest'];
    await mockDelay();
    const email = (body.email ?? '').toLowerCase();
    const u = db.users.find((x) => x.email.toLowerCase() === email);
    if (u && !u.emailVerifiedAt) issueVerificationToken(u.id);
    const resp: S['RegisterResponse'] = { message: REGISTRATION_ACCEPTED_MESSAGE };
    return HttpResponse.json(resp, { status: 202 });
  }),

  http.post(P('/auth/logout'), async () => {
    await mockDelay();
    session.userId = null;
    return new HttpResponse(null, { status: 204 });
  }),

  http.get(P('/auth/me'), async () => {
    await mockDelay();
    if (!session.userId) return problem(401, 'Not authenticated');
    return HttpResponse.json(toWireUser(requireUser(session.userId)));
  }),

  http.delete(P('/auth/me'), async ({ request }) => {
    await mockDelay();
    const auth = requireAuth();
    if (typeof auth !== 'string') return auth;
    const body = (await request.json()) as S['DeleteAccountRequest'];
    const u = requireUser(auth);
    if (body.confirmEmail?.toLowerCase() !== u.email.toLowerCase()) return problem(400, 'Email does not match');
    u.name = 'Gelöschtes Mitglied';
    u.email = `deleted+${u.id}@invalid`;
    u.phone = '';
    u.birthday = '';
    u.address = '';
    u.hasPhoto = false;
    session.userId = null;
    return new HttpResponse(null, { status: 204 });
  }),

  http.get(P('/auth/me/data-export'), async () => {
    await mockDelay();
    const auth = requireAuth();
    if (typeof auth !== 'string') return auth;
    const u = requireUser(auth);
    return HttpResponse.json({
      exportedAt: new Date().toISOString(),
      profile: toWireUser(u),
      memberships: db.memberships.filter((m) => m.userId === auth),
    });
  }),

  http.put(P('/auth/me/photo'), async () => {
    await mockDelay();
    const auth = requireAuth();
    if (typeof auth !== 'string') return auth;
    const u = requireUser(auth);
    u.hasPhoto = true;
    return HttpResponse.json(toWireUser(u));
  }),

  // ---- teams ----
  http.get(P('/teams'), async () => {
    await mockDelay();
    const auth = requireAuth();
    if (typeof auth !== 'string') return auth;
    const teamIds = db.memberships.filter((m) => m.userId === auth).map((m) => m.teamId);
    const body: S['TeamForUser'][] = db.teams.filter((t) => teamIds.includes(t.id)).map((t) => toWireTeamForUser(t, auth));
    return HttpResponse.json(body);
  }),

  http.post(P('/teams'), async ({ request }) => {
    await mockDelay();
    const auth = requireAuth();
    if (typeof auth !== 'string') return auth;
    const body = (await request.json()) as S['CreateTeamRequest'];
    const team: TeamRow = {
      id: rid('t'),
      name: body.name,
      short: (body.name || 'T').trim().charAt(0).toUpperCase(),
      icon: body.icon || '⭐',
      iconBg: body.iconBg || '#1565C0',
      iconFg: body.iconFg || '#FFFFFF',
      photo: null,
      logo: null,
      hasPhoto: false,
      hasLogo: false,
      description: '',
    };
    db.teams.push(team);
    const roles: RoleDto[] = [
      { id: rid('role'), teamId: team.id, name: 'Admin / Trainer', system: true, color: '#1565C0', permissions: { events: 'write', members: 'write', finances: 'write', news: 'write', polls: 'write', settings: 'write' } },
      { id: rid('role'), teamId: team.id, name: 'Mitglied', system: true, color: '#5B6470', permissions: { events: 'read', members: 'read', finances: 'read', news: 'read', polls: 'read', settings: 'none' } },
    ];
    db.roles.push(...roles);
    team.reasonVisibilityRoles = [roles[0].id];
    db.memberships.push({ id: rid('mem'), teamId: team.id, userId: auth, roleIds: [roles[0].id], group: '', joinedAt: new Date().toISOString() });
    return HttpResponse.json(toWireTeamForUser(team, auth), { status: 201 });
  }),

  http.get(P('/teams/:teamId'), async ({ params }) => {
    await mockDelay();
    const t = db.teams.find((x) => x.id === params.teamId);
    if (!t) return problem(404, 'Team not found');
    return HttpResponse.json(toWireTeam(t));
  }),

  http.patch(P('/teams/:teamId'), async ({ params, request }) => {
    await mockDelay();
    const t = db.teams.find((x) => x.id === params.teamId);
    if (!t) return problem(404, 'Team not found');
    const body = (await request.json()) as S['UpdateTeamRequest'];
    if (body.name !== undefined) t.name = body.name;
    if (body.short !== undefined) t.short = body.short;
    if (body.icon !== undefined) t.icon = body.icon;
    if (body.iconBg !== undefined) t.iconBg = body.iconBg;
    if (body.iconFg !== undefined) t.iconFg = body.iconFg;
    if (body.description !== undefined) t.description = body.description;
    if (body.reasonVisibilityRoleIds !== undefined) t.reasonVisibilityRoles = body.reasonVisibilityRoleIds;
    return HttpResponse.json(toWireTeam(t));
  }),

  http.put(P('/teams/:teamId/photo'), async ({ params }) => {
    await mockDelay();
    const t = db.teams.find((x) => x.id === params.teamId);
    if (!t) return problem(404, 'Team not found');
    t.hasPhoto = true;
    return HttpResponse.json(toWireTeam(t));
  }),
  http.delete(P('/teams/:teamId/photo'), async ({ params }) => {
    await mockDelay();
    const t = db.teams.find((x) => x.id === params.teamId);
    if (t) t.hasPhoto = false;
    return new HttpResponse(null, { status: 204 });
  }),
  http.put(P('/teams/:teamId/logo'), async ({ params }) => {
    await mockDelay();
    const t = db.teams.find((x) => x.id === params.teamId);
    if (!t) return problem(404, 'Team not found');
    t.hasLogo = true;
    return HttpResponse.json(toWireTeam(t));
  }),
  http.delete(P('/teams/:teamId/logo'), async ({ params }) => {
    await mockDelay();
    const t = db.teams.find((x) => x.id === params.teamId);
    if (t) t.hasLogo = false;
    return new HttpResponse(null, { status: 204 });
  }),

  http.post(P('/teams/:teamId/invite'), async ({ params }) => {
    await mockDelay();
    const teamId = params.teamId as string;
    const code = Math.random().toString(36).slice(2, 8).toUpperCase();
    const inv: S['Invite'] = {
      id: rid('inv'),
      teamId,
      code,
      link: 'https://teamverwaltung.app/join/' + teamId + '/' + code,
      createdAt: new Date().toISOString(),
      expiresAt: new Date(Date.now() + 7 * 86400000).toISOString(),
    };
    db.invites.push(inv);
    return HttpResponse.json(inv, { status: 201 });
  }),

  http.post(P('/invites/:code/accept'), async ({ params }) => {
    await mockDelay();
    const auth = requireAuth();
    if (typeof auth !== 'string') return auth;
    const inv = db.invites.find((i) => i.code === params.code);
    if (!inv || new Date(inv.expiresAt).getTime() <= Date.now()) return problem(404, 'Invite not found or expired');
    const existing = db.memberships.find((m) => m.teamId === inv.teamId && m.userId === auth);
    const alreadyMember = !!existing;
    if (!existing) {
      // Matches backend/internal/teams/repository.go's AcceptInvite: look up
      // the seeded default member role by its stable name, not "any role
      // that isn't Admin" — a team with several non-admin system roles
      // (Kassenwart, Teamkapitän, ...) would otherwise risk handing a new
      // member a privileged role if the true default role were ever deleted.
      const memberRole = db.roles.find((r) => r.teamId === inv.teamId && r.name === DEFAULT_MEMBER_ROLE_NAME);
      db.memberships.push({ id: rid('mem'), teamId: inv.teamId, userId: auth, roleIds: memberRole ? [memberRole.id] : [], group: '', joinedAt: new Date().toISOString() });
    }
    const t = db.teams.find((x) => x.id === inv.teamId)!;
    const body: S['AcceptInviteResponse'] = { ...toWireTeamForUser(t, auth), alreadyMember };
    return HttpResponse.json(body);
  }),

  // ---- members ----
  http.get(P('/teams/:teamId/members'), async ({ params }) => {
    await mockDelay();
    const items = db.memberships.filter((m) => m.teamId === params.teamId).map(toWireMember).sort((a, b) => a.name.localeCompare(b.name, 'de'));
    return HttpResponse.json({ items, nextCursor: null });
  }),

  http.patch(P('/teams/:teamId/members/:membershipId'), async ({ params, request }) => {
    await mockDelay();
    const m = db.memberships.find((x) => x.id === params.membershipId);
    if (!m) return problem(404, 'Member not found');
    const u = requireUser(m.userId);
    const body = (await request.json()) as S['UpdateMemberRequest'];
    if (body.name !== undefined) u.name = body.name;
    if (body.email !== undefined) u.email = body.email;
    if (body.phone !== undefined) u.phone = body.phone;
    if (body.birthday !== undefined) u.birthday = body.birthday;
    if (body.address !== undefined) u.address = body.address;
    if (body.group !== undefined) m.group = body.group;
    // An explicitly empty array clears all roles (matches the real backend's
    // SetRoles, only guarded by ErrLastSettingsAdmin server-side) — only an
    // absent field should be a no-op, not an empty one.
    if (body.roleIds !== undefined) m.roleIds = body.roleIds;
    return HttpResponse.json(toWireMember(m));
  }),

  http.put(P('/teams/:teamId/members/:membershipId/roles'), async ({ params, request }) => {
    await mockDelay();
    const m = db.memberships.find((x) => x.id === params.membershipId);
    if (!m) return problem(404, 'Member not found');
    const body = (await request.json()) as S['SetRolesRequest'];
    m.roleIds = body.roleIds;
    return HttpResponse.json(toWireMember(m));
  }),

  http.delete(P('/teams/:teamId/members/:membershipId'), async ({ params }) => {
    await mockDelay();
    if (!db.memberships.some((x) => x.id === params.membershipId)) return problem(404, 'Member not found');
    db.memberships = db.memberships.filter((x) => x.id !== params.membershipId);
    return new HttpResponse(null, { status: 204 });
  }),

  // ---- roles ----
  http.get(P('/teams/:teamId/roles'), async ({ params }) => {
    await mockDelay();
    const body = db.roles.filter((r) => r.teamId === params.teamId).map(toWireRole);
    return HttpResponse.json(body);
  }),

  http.post(P('/teams/:teamId/roles'), async ({ params, request }) => {
    await mockDelay();
    const teamId = params.teamId as string;
    const body = (await request.json()) as S['CreateRoleRequest'];
    const r: RoleDto = { id: rid('role'), teamId, name: body.name, system: false, color: body.color || '#888888', permissions: body.permissions };
    db.roles.push(r);
    return HttpResponse.json(toWireRole(r), { status: 201 });
  }),

  http.patch(P('/teams/:teamId/roles/:roleId'), async ({ params, request }) => {
    await mockDelay();
    const r = db.roles.find((x) => x.id === params.roleId);
    if (!r) return problem(404, 'Role not found');
    const body = (await request.json()) as S['UpdateRoleRequest'];
    if (body.name !== undefined) r.name = body.name;
    if (body.color !== undefined) r.color = body.color;
    if (body.permissions !== undefined) r.permissions = body.permissions;
    return HttpResponse.json(toWireRole(r));
  }),

  http.delete(P('/teams/:teamId/roles/:roleId'), async ({ params }) => {
    await mockDelay();
    if (!db.roles.some((x) => x.id === params.roleId)) return problem(404, 'Role not found');
    db.roles = db.roles.filter((x) => x.id !== params.roleId);
    return new HttpResponse(null, { status: 204 });
  }),

  // ---- events ----
  http.get(P('/teams/:teamId/events'), async ({ params, request }) => {
    await mockDelay();
    const url = new URL(request.url);
    const scope = (url.searchParams.get('scope') as 'all' | 'upcoming' | 'past' | null) ?? 'all';
    const today = todayLocalDate();
    // Drift-bug fix #4: "upcoming" is date >= today (today counts as
    // upcoming), matching backend/internal/events/repository.go's
    // `WHERE date >= $2` for scope=upcoming (and `date < $2` for scope=past).
    let list = db.events.filter((e) => e.teamId === params.teamId);
    if (scope === 'upcoming') list = list.filter((e) => e.date >= today);
    if (scope === 'past') list = list.filter((e) => e.date < today);
    list = [...list].sort((a, b) => (scope === 'past' ? b.date.localeCompare(a.date) : a.date.localeCompare(b.date)));
    return HttpResponse.json({ items: list.map(toWireEvent), nextCursor: null });
  }),

  http.post(P('/teams/:teamId/events'), async ({ params, request }) => {
    await mockDelay();
    const teamId = params.teamId as string;
    const body = (await request.json()) as S['CreateEventRequest'];
    const created: EventDto[] = [];
    const mk = (date: string): EventDto => ({
      id: rid('ev'),
      teamId,
      type: body.type,
      title: body.title,
      date,
      location: body.location || '',
      note: body.note || '',
      result: undefined,
      meetTime: body.meetTime ?? null,
      startTime: body.startTime ?? null,
      endTime: body.endTime ?? null,
      meetTimeMandatory: body.meetTimeMandatory ?? false,
      responseMode: body.responseMode || 'opt_in',
      nominatedRoleIds: body.nominatedRoleIds ? [...body.nominatedRoleIds] : undefined,
      recurring: !!body.recurring,
      seriesId: null,
      status: 'active',
    });
    if (body.recurring && body.repeatWeeks && body.repeatWeeks > 1) {
      const seriesId = rid('series');
      for (let w = 0; w < body.repeatWeeks; w++) {
        const d = parseDateOnlyLocal(body.date);
        d.setDate(d.getDate() + w * 7);
        const e = mk(formatDateOnly(d));
        e.seriesId = seriesId;
        created.push(e);
      }
    } else {
      created.push(mk(body.date));
    }
    created.forEach((e) => db.events.push(e));
    if (body.nominatedRoleIds) created.forEach((e) => applyNominations(e, body.nominatedRoleIds!));
    pushNotif({
      teamId,
      type: 'event_created',
      actorId: session.userId ?? undefined,
      title: created[0].title,
      eventId: created[0].id,
      eventTitle: created[0].title,
      eventDate: created[0].date,
      note: created.length > 1 ? `Serie mit ${created.length} Terminen` : '',
    });
    return HttpResponse.json(toWireEvent(created[0]), { status: 201 });
  }),

  http.get(P('/teams/:teamId/events/:eventId'), async ({ params }) => {
    await mockDelay();
    const e = eventDate(params.eventId as string);
    if (!e) return problem(404, 'Event not found');
    return HttpResponse.json(toWireEvent(e));
  }),

  http.patch(P('/teams/:teamId/events/:eventId'), async ({ params, request }) => {
    await mockDelay();
    const e = eventDate(params.eventId as string);
    if (!e) return problem(404, 'Event not found');
    const url = new URL(request.url);
    const scope = (url.searchParams.get('scope') as 'single' | 'series' | null) ?? 'single';
    const body = (await request.json()) as S['UpdateEventRequest'];
    const targets = scope === 'series' && e.seriesId ? db.events.filter((x) => x.seriesId === e.seriesId) : [e];
    targets.forEach((ev) => {
      if (scope !== 'series' && body.date !== undefined) ev.date = body.date;
      if (body.type !== undefined) ev.type = body.type;
      if (body.title !== undefined) ev.title = body.title;
      if (body.location !== undefined) ev.location = body.location;
      if (body.note !== undefined) ev.note = body.note;
      if (body.meetTimeMandatory !== undefined) ev.meetTimeMandatory = body.meetTimeMandatory;
      if (body.responseMode !== undefined) ev.responseMode = body.responseMode;
      if (body.meetTime !== undefined) ev.meetTime = body.meetTime || null;
      if (body.startTime !== undefined) ev.startTime = body.startTime || null;
      if (body.endTime !== undefined) ev.endTime = body.endTime || null;
      if (body.nominatedRoleIds !== undefined) applyNominations(ev, body.nominatedRoleIds);
    });
    pushNotif({ teamId: e.teamId, type: 'event_updated', actorId: session.userId ?? undefined, title: e.title, eventId: e.id, eventTitle: e.title, eventDate: e.date, note: scope === 'series' ? 'ganze Serie' : '' });
    return HttpResponse.json(toWireEvent(e));
  }),

  http.post(P('/teams/:teamId/events/:eventId/status'), async ({ params, request }) => {
    await mockDelay();
    const e = eventDate(params.eventId as string);
    if (!e) return problem(404, 'Event not found');
    const url = new URL(request.url);
    const scope = (url.searchParams.get('scope') as 'single' | 'series' | null) ?? 'single';
    const body = (await request.json()) as S['SetEventStatusRequest'];
    const targets = scope === 'series' && e.seriesId ? db.events.filter((x) => x.seriesId === e.seriesId) : [e];
    targets.forEach((ev) => { ev.status = body.status; });
    pushNotif({ teamId: e.teamId, type: body.status === 'cancelled' ? 'event_cancelled' : 'event_reactivated', actorId: session.userId ?? undefined, title: e.title, eventId: e.id, eventTitle: e.title, eventDate: e.date, note: scope === 'series' ? 'ganze Serie' : '' });
    return HttpResponse.json(toWireEvent(e));
  }),

  http.delete(P('/teams/:teamId/events/:eventId'), async ({ params, request }) => {
    await mockDelay();
    const e = eventDate(params.eventId as string);
    const url = new URL(request.url);
    const scope = (url.searchParams.get('scope') as 'single' | 'series' | null) ?? 'single';
    const ids = e && scope === 'series' && e.seriesId ? db.events.filter((x) => x.seriesId === e.seriesId).map((x) => x.id) : [params.eventId as string];
    if (e) pushNotif({ teamId: e.teamId, type: 'event_deleted', actorId: session.userId ?? undefined, title: e.title, eventTitle: e.title, eventDate: e.date, note: scope === 'series' ? 'ganze Serie' : '' });
    db.events = db.events.filter((x) => !ids.includes(x.id));
    db.attendance = db.attendance.filter((a) => !ids.includes(a.eventId));
    db.eventComments = db.eventComments.filter((c) => !ids.includes(c.eventId));
    return new HttpResponse(null, { status: 204 });
  }),

  http.get(P('/teams/:teamId/events/:eventId/comments'), async ({ params }) => {
    await mockDelay();
    // Keyset { items, nextCursor } envelope (oldest-first). The mock returns
    // everything in one page (nextCursor: null), which fetchAllPages consumes
    // exactly like the real multi-page envelope.
    const items = db.eventComments.filter((c) => c.eventId === params.eventId).sort((a, b) => a.createdAt.localeCompare(b.createdAt)).map(toWireComment);
    return HttpResponse.json({ items, nextCursor: null });
  }),

  http.post(P('/teams/:teamId/events/:eventId/comments'), async ({ params, request }) => {
    await mockDelay();
    const auth = requireAuth();
    if (typeof auth !== 'string') return auth;
    const body = (await request.json()) as S['AddCommentRequest'];
    const c = { id: rid('cm'), eventId: params.eventId as string, userId: auth, text: body.text, createdAt: new Date().toISOString() };
    db.eventComments.push(c);
    return HttpResponse.json(toWireComment(c), { status: 201 });
  }),

  http.delete(P('/teams/:teamId/events/:eventId/comments/:commentId'), async ({ params }) => {
    await mockDelay();
    db.eventComments = db.eventComments.filter((x) => x.id !== params.commentId);
    return new HttpResponse(null, { status: 204 });
  }),

  http.get(P('/teams/:teamId/events/:eventId/attendance'), async ({ params }) => {
    await mockDelay();
    const e = eventDate(params.eventId as string);
    if (!e) return problem(404, 'Event not found');
    const rows = db.memberships.filter((m) => m.teamId === e.teamId).map((m) => toWireAttendanceRow(e, m));
    return HttpResponse.json(rows);
  }),

  http.post(P('/teams/:teamId/events/:eventId/attendance'), async ({ params, request }) => {
    await mockDelay();
    const eventId = params.eventId as string;
    const body = (await request.json()) as S['SetAttendanceRequest'];
    let a = db.attendance.find((x) => x.eventId === eventId && x.userId === body.userId);
    if (!a) {
      a = { id: rid('att'), eventId, userId: body.userId, status: body.status, reason: '', reasonId: null, reasonVisibility: null };
      db.attendance.push(a);
    }
    a.status = body.status;
    a.reason = body.reason || '';
    a.reasonId = body.reasonId || null;
    a.reasonVisibility = (body.reasonVisibility as S['AttendanceRow']['reasonVisibility']) || null;
    a.at = new Date().toISOString();
    const e = eventDate(eventId);
    if (e && (body.status === 'yes' || body.status === 'no' || body.status === 'maybe')) {
      pushNotif({ teamId: e.teamId, type: 'attendance', actorId: body.userId, status: body.status, eventId: e.id, eventTitle: e.title, eventDate: e.date });
    }
    const record: S['AttendanceRecord'] = { id: a.id, eventId: a.eventId, userId: a.userId, status: a.status, reason: a.reason || undefined, reasonId: a.reasonId ?? undefined, reasonVisibility: a.reasonVisibility ?? undefined, at: a.at };
    return HttpResponse.json(record);
  }),

  http.put(P('/teams/:teamId/events/:eventId/attendance/nominations'), async ({ params, request }) => {
    await mockDelay();
    const eventId = params.eventId as string;
    const body = (await request.json()) as S['SetNominationRequest'];
    if (body.nominated) {
      // Only clear the synthetic "not_nominated" placeholder — mirrors
      // applyNominations() in db.ts. A member who already has a real RSVP
      // ('yes'/'no'/'maybe') keeps it; re-nominating them must not silently
      // revert an actual response back to pending.
      db.attendance = db.attendance.filter((x) => !(x.eventId === eventId && x.userId === body.userId && x.status === 'not_nominated'));
    } else {
      let a = db.attendance.find((x) => x.eventId === eventId && x.userId === body.userId);
      if (!a) {
        a = { id: rid('att'), eventId, userId: body.userId, status: 'not_nominated', reason: '', reasonId: null, reasonVisibility: null };
        db.attendance.push(a);
      }
      a.status = 'not_nominated';
      a.reason = '';
      a.reasonId = null;
    }
    return new HttpResponse(null, { status: 204 });
  }),

  // ---- absences ----
  http.get(P('/teams/:teamId/absences'), async ({ params }) => {
    await mockDelay();
    const memberIds = db.memberships.filter((m) => m.teamId === params.teamId).map((m) => m.userId);
    const items = db.absences.filter((a) => memberIds.includes(a.userId)).map((a) => toWireAbsence(a, params.teamId as string));
    return HttpResponse.json({ items, nextCursor: null });
  }),

  http.get(P('/teams/:teamId/absences/mine'), async ({ params }) => {
    await mockDelay();
    const auth = requireAuth();
    if (typeof auth !== 'string') return auth;
    const items = db.absences.filter((a) => a.userId === auth).map((a) => toWireAbsence(a, params.teamId as string));
    return HttpResponse.json({ items, nextCursor: null });
  }),

  http.post(P('/teams/:teamId/absences'), async ({ params, request }) => {
    await mockDelay();
    const body = (await request.json()) as S['CreateAbsenceRequest'];
    const a = { id: rid('abs'), userId: body.userId, from: body.from, to: body.to, reason: body.reason || '', createdAt: new Date().toISOString() };
    db.absences.push(a);
    const mem = db.memberships.find((m) => m.userId === body.userId && m.teamId === params.teamId);
    if (mem) pushNotif({ teamId: mem.teamId, type: 'absence', actorId: body.userId, title: a.reason });
    return HttpResponse.json(toWireAbsence(a, params.teamId as string), { status: 201 });
  }),

  http.patch(P('/teams/:teamId/absences/:absenceId'), async ({ params, request }) => {
    await mockDelay();
    const a = db.absences.find((x) => x.id === params.absenceId);
    if (!a) return problem(404, 'Absence not found');
    const body = (await request.json()) as S['UpdateAbsenceRequest'];
    if (body.from !== undefined) a.from = body.from;
    if (body.to !== undefined) a.to = body.to;
    if (body.reason !== undefined) a.reason = body.reason;
    return HttpResponse.json(toWireAbsence(a, params.teamId as string));
  }),

  http.delete(P('/teams/:teamId/absences/:absenceId'), async ({ params }) => {
    await mockDelay();
    if (!db.absences.some((x) => x.id === params.absenceId)) return problem(404, 'Absence not found');
    db.absences = db.absences.filter((x) => x.id !== params.absenceId);
    return new HttpResponse(null, { status: 204 });
  }),

  // ---- news ----
  http.get(P('/teams/:teamId/news'), async ({ params }) => {
    await mockDelay();
    const items = db.news
      .filter((n) => n.teamId === params.teamId)
      .sort((a, b) => Number(b.pinned) - Number(a.pinned) || b.createdAt.localeCompare(a.createdAt))
      .map(toWireNews);
    return HttpResponse.json({ items, nextCursor: null });
  }),

  http.post(P('/teams/:teamId/news'), async ({ params, request }) => {
    await mockDelay();
    const auth = requireAuth();
    if (typeof auth !== 'string') return auth;
    const teamId = params.teamId as string;
    const body = (await request.json()) as S['CreateNewsRequest'];
    const n = { id: rid('news'), teamId, authorId: auth, title: body.title, body: body.body, pinned: !!body.pinned, createdAt: new Date().toISOString() };
    db.news.push(n);
    pushNotif({ teamId, type: 'news', actorId: auth, title: body.title });
    return HttpResponse.json(toWireNews(n), { status: 201 });
  }),

  http.patch(P('/teams/:teamId/news/:newsId'), async ({ params, request }) => {
    await mockDelay();
    const n = db.news.find((x) => x.id === params.newsId);
    if (!n) return problem(404, 'News not found');
    const body = (await request.json()) as S['UpdateNewsRequest'];
    if (body.title !== undefined) n.title = body.title;
    if (body.body !== undefined) n.body = body.body;
    if (body.pinned !== undefined) n.pinned = body.pinned;
    return HttpResponse.json(toWireNews(n));
  }),

  http.delete(P('/teams/:teamId/news/:newsId'), async ({ params }) => {
    await mockDelay();
    if (!db.news.some((x) => x.id === params.newsId)) return problem(404, 'News not found');
    db.news = db.news.filter((x) => x.id !== params.newsId);
    return new HttpResponse(null, { status: 204 });
  }),

  // ---- polls ----
  http.get(P('/teams/:teamId/polls'), async ({ params }) => {
    await mockDelay();
    const items = db.polls.filter((p) => p.teamId === params.teamId).sort((a, b) => b.createdAt.localeCompare(a.createdAt)).map(toWirePoll);
    return HttpResponse.json({ items, nextCursor: null });
  }),

  http.post(P('/teams/:teamId/polls'), async ({ params, request }) => {
    await mockDelay();
    const teamId = params.teamId as string;
    const body = (await request.json()) as S['CreatePollRequest'];
    const p = {
      id: rid('poll'),
      teamId,
      question: body.question,
      multiple: !!body.multiple,
      anonymous: !!body.anonymous,
      createdAt: new Date().toISOString(),
      options: body.options.filter((o) => o.trim()).map((o, i) => ({ id: 'opt' + i + '_' + rid('o'), text: o.trim() })),
      votes: [],
    };
    db.polls.push(p);
    pushNotif({ teamId, type: 'poll', actorId: session.userId ?? undefined, title: body.question });
    return HttpResponse.json(toWirePoll(p), { status: 201 });
  }),

  http.delete(P('/teams/:teamId/polls/:pollId'), async ({ params }) => {
    await mockDelay();
    if (!db.polls.some((x) => x.id === params.pollId)) return problem(404, 'Poll not found');
    db.polls = db.polls.filter((x) => x.id !== params.pollId);
    return new HttpResponse(null, { status: 204 });
  }),

  // Drift-bug fix #3: backend/internal/polls/service.go rejects a vote with
  // >1 optionIds on a single-choice (multiple=false) poll with
  // ErrSingleChoiceMultipleOptions (422) — the old localStorage mock instead
  // silently truncated to `optionIds[0]`.
  http.post(P('/teams/:teamId/polls/:pollId/vote'), async ({ params, request }) => {
    await mockDelay();
    const auth = requireAuth();
    if (typeof auth !== 'string') return auth;
    const p = db.polls.find((x) => x.id === params.pollId);
    if (!p) return problem(404, 'Poll not found');
    const body = (await request.json()) as S['VotePollRequest'];
    if (!p.multiple && body.optionIds.length > 1) {
      return problem(422, 'cannot select multiple options on a single-choice poll');
    }
    p.votes = p.votes.filter((v) => v.userId !== auth);
    if (body.optionIds.length) p.votes.push({ userId: auth, optionIds: [...body.optionIds] });
    return HttpResponse.json(toWirePoll(p));
  }),

  // ---- notifications ----
  http.get(P('/teams/:teamId/notifications'), async ({ params }) => {
    await mockDelay();
    const teamId = params.teamId as string;
    const since = Date.now() - 62 * 86400000;
    const seen = db.notifSeen[teamId] || null;
    const items = db.notifications
      .filter((n) => n.teamId === teamId && new Date(n.createdAt).getTime() >= since)
      .sort((a, b) => b.createdAt.localeCompare(a.createdAt))
      .map((n) => toWireNotification(n, seen));
    const body: S['NotificationsResult'] = { items, unreadCount: items.filter((n) => n.unread).length };
    return HttpResponse.json(body);
  }),

  http.post(P('/teams/:teamId/notifications/seen'), async ({ params }) => {
    await mockDelay();
    db.notifSeen[params.teamId as string] = new Date().toISOString();
    return new HttpResponse(null, { status: 204 });
  }),

  // ---- finances ----
  http.get(P('/teams/:teamId/finances'), async ({ params }) => {
    await mockDelay();
    const teamId = params.teamId as string;
    const tx = db.transactions.filter((x) => x.teamId === teamId).sort((a, b) => b.date.localeCompare(a.date));
    const income = tx.filter((t) => t.type === 'income').reduce((s, t) => s + t.amount, 0);
    const expense = tx.filter((t) => t.type === 'expense').reduce((s, t) => s + t.amount, 0);
    const penalties = db.penalties.filter((p) => p.teamId === teamId);
    const assignments = db.penaltyAssignments.filter((p) => p.teamId === teamId);
    const openByUser: Record<string, number> = {};
    assignments.filter((a) => !a.paid).forEach((a) => { openByUser[a.userId] = (openByUser[a.userId] || 0) + (a.amount || 0); });
    const openPenalties: S['OpenPenalty'][] = Object.keys(openByUser)
      .map((uid) => {
        const u = requireUser(uid);
        return { userId: uid, name: u.name, avatarColor: u.avatarColor, hasPhoto: u.hasPhoto, amount: openByUser[uid] };
      })
      .sort((a, b) => b.amount - a.amount);
    const contributions = db.contributions.filter((c) => c.teamId === teamId);
    const body: S['FinanceOverview'] = {
      balance: income - expense,
      income,
      expense,
      transactions: tx.map(toWireTransaction),
      penalties: penalties.map(toWirePenalty),
      assignments: assignments.map(toWireAssignment),
      openPenalties,
      openPenaltySum: Object.values(openByUser).reduce((s, v) => s + v, 0),
      contributions: contributions.map(toWireContribution),
      contribOpen: contributions.filter((c) => c.status === 'open').length,
    };
    return HttpResponse.json(body);
  }),

  // Keyset-paginated transaction list. Removes the overview's row cap by
  // exposing the full history; the mock returns everything in one page
  // (nextCursor: null), which fetchAllPages consumes exactly like the real
  // multi-page envelope.
  http.get(P('/teams/:teamId/finances/transactions'), async ({ params }) => {
    await mockDelay();
    const teamId = params.teamId as string;
    const tx = db.transactions
      .filter((x) => x.teamId === teamId)
      .sort((a, b) => b.date.localeCompare(a.date));
    return HttpResponse.json({ items: tx.map(toWireTransaction), nextCursor: null });
  }),

  http.post(P('/teams/:teamId/finances/transactions'), async ({ params, request }) => {
    await mockDelay();
    const teamId = params.teamId as string;
    const body = (await request.json()) as S['CreateTransactionRequest'];
    const t = { id: rid('tx'), teamId, type: body.type, title: body.title, amount: body.amount, date: body.date || todayLocalDate(), category: body.category || '' };
    db.transactions.push(t);
    return HttpResponse.json(toWireTransaction(t), { status: 201 });
  }),

  http.patch(P('/teams/:teamId/finances/transactions/:transactionId'), async ({ params, request }) => {
    await mockDelay();
    const t = db.transactions.find((x) => x.id === params.transactionId);
    if (!t) return problem(404, 'Transaction not found');
    const body = (await request.json()) as S['UpdateTransactionRequest'];
    if (body.type !== undefined) t.type = body.type;
    if (body.title !== undefined) t.title = body.title;
    if (body.amount !== undefined) t.amount = body.amount;
    if (body.category !== undefined) t.category = body.category;
    if (body.date !== undefined) t.date = body.date;
    return HttpResponse.json(toWireTransaction(t));
  }),

  http.delete(P('/teams/:teamId/finances/transactions/:transactionId'), async ({ params }) => {
    await mockDelay();
    if (!db.transactions.some((x) => x.id === params.transactionId)) return problem(404, 'Transaction not found');
    db.transactions = db.transactions.filter((x) => x.id !== params.transactionId);
    return new HttpResponse(null, { status: 204 });
  }),

  http.post(P('/teams/:teamId/finances/penalties'), async ({ params, request }) => {
    await mockDelay();
    const teamId = params.teamId as string;
    const body = (await request.json()) as S['CreatePenaltyRequest'];
    const p = { id: rid('pen'), teamId, label: body.label, amount: body.amount };
    db.penalties.push(p);
    return HttpResponse.json(toWirePenalty(p), { status: 201 });
  }),

  http.patch(P('/teams/:teamId/finances/penalties/:penaltyId'), async ({ params, request }) => {
    await mockDelay();
    const p = db.penalties.find((x) => x.id === params.penaltyId);
    if (!p) return problem(404, 'Penalty not found');
    const body = (await request.json()) as S['UpdatePenaltyRequest'];
    // Drift-bug fix #1: editing a penalty template must NOT retroactively
    // change amounts already snapshotted onto PenaltyAssignment rows (see
    // createPenaltyAssignment below / backend/internal/finances/repository.go's
    // `INSERT INTO penalty_assignments (..., amount, label) SELECT amount,
    // label FROM penalties WHERE id = $1`) — only future assignments see it.
    if (body.label !== undefined) p.label = body.label;
    if (body.amount !== undefined) p.amount = body.amount;
    return HttpResponse.json(toWirePenalty(p));
  }),

  http.delete(P('/teams/:teamId/finances/penalties/:penaltyId'), async ({ params }) => {
    await mockDelay();
    if (!db.penalties.some((x) => x.id === params.penaltyId)) return problem(404, 'Penalty not found');
    db.penalties = db.penalties.filter((x) => x.id !== params.penaltyId);
    db.penaltyAssignments = db.penaltyAssignments.filter((x) => x.penaltyId !== params.penaltyId);
    return new HttpResponse(null, { status: 204 });
  }),

  http.post(P('/teams/:teamId/finances/penalty-assignments'), async ({ params, request }) => {
    await mockDelay();
    const teamId = params.teamId as string;
    const body = (await request.json()) as S['CreatePenaltyAssignmentRequest'];
    const penalty = db.penalties.find((p) => p.id === body.penaltyId);
    if (!penalty) return problem(404, 'Penalty not found');
    // Snapshot label/amount at assignment time (drift-bug fix #1).
    const a = { id: rid('pa'), teamId, userId: body.userId, penaltyId: body.penaltyId, paid: false, date: todayLocalDate(), label: penalty.label, amount: penalty.amount };
    db.penaltyAssignments.push(a);
    return HttpResponse.json(toWireAssignment(a), { status: 201 });
  }),

  http.delete(P('/teams/:teamId/finances/penalty-assignments/:assignmentId'), async ({ params }) => {
    await mockDelay();
    if (!db.penaltyAssignments.some((x) => x.id === params.assignmentId)) return problem(404, 'Penalty assignment not found');
    db.penaltyAssignments = db.penaltyAssignments.filter((x) => x.id !== params.assignmentId);
    return new HttpResponse(null, { status: 204 });
  }),

  http.put(P('/teams/:teamId/finances/penalty-assignments/:assignmentId/paid'), async ({ params, request }) => {
    await mockDelay();
    const a = db.penaltyAssignments.find((x) => x.id === params.assignmentId);
    if (!a) return problem(404, 'Assignment not found');
    const body = (await request.json()) as S['SetPaidRequest'];
    a.paid = body.paid;
    return HttpResponse.json(toWireAssignment(a));
  }),

  http.patch(P('/teams/:teamId/finances/contributions/:contributionId'), async ({ params, request }) => {
    await mockDelay();
    const c = db.contributions.find((x) => x.id === params.contributionId);
    if (!c) return problem(404, 'Contribution not found');
    const body = (await request.json()) as S['UpdateContributionRequest'];
    if (body.label !== undefined) c.label = body.label;
    if (body.amount !== undefined) c.amount = body.amount;
    return HttpResponse.json(toWireContribution(c));
  }),

  http.put(P('/teams/:teamId/finances/contributions/:contributionId/paid'), async ({ params, request }) => {
    await mockDelay();
    const c = db.contributions.find((x) => x.id === params.contributionId);
    if (!c) return problem(404, 'Contribution not found');
    const body = (await request.json()) as S['SetPaidRequest'];
    c.status = body.paid ? 'paid' : 'open';
    return HttpResponse.json(toWireContribution(c));
  }),

  // ---- stats ----
  http.get(P('/teams/:teamId/stats'), async ({ params, request }) => {
    await mockDelay();
    const teamId = params.teamId as string;
    const url = new URL(request.url);
    const today = todayLocalDate();
    const from = url.searchParams.get('from') || threeMonthsBeforeLocal(today);
    const to = url.searchParams.get('to') || today;
    const memberIds = db.memberships.filter((m) => m.teamId === teamId).map((m) => m.userId);
    const events = db.events.filter((e) => e.teamId === teamId && e.status !== 'cancelled' && e.date >= from && e.date <= to).sort((a, b) => a.date.localeCompare(b.date));

    const memberStats: S['MemberStat'][] = memberIds
      .map((uid) => {
        const u = requireUser(uid);
        let yes = 0, counted = 0;
        events.forEach((e) => {
          const s = rawCountedStatus(e.id, uid);
          if (!s) return;
          counted++;
          if (s === 'yes') yes++;
        });
        return { userId: u.id, name: u.name, avatarColor: u.avatarColor, hasPhoto: u.hasPhoto, quote: counted ? yes / counted : 0, counted, yes };
      })
      .sort((a, b) => b.yes - a.yes || a.name.localeCompare(b.name, 'de'));
    const avg = memberStats.length ? memberStats.reduce((s, m) => s + m.quote, 0) / memberStats.length : 0;

    const eventStats: S['EventStat'][] = events.map((e) => {
      let yes = 0, counted = 0;
      memberIds.forEach((uid) => {
        const s = rawCountedStatus(e.id, uid);
        if (!s) return;
        counted++;
        if (s === 'yes') yes++;
      });
      const pct = counted ? yes / counted : 0;
      return { id: e.id, title: e.title, type: e.type, date: e.date, yes, nominated: counted, pct, enough: pct >= 0.5 };
    });

    const body: S['StatsOverview'] = { avg, members: memberStats, events: eventStats, pastCount: events.length, from, to };
    return HttpResponse.json(body);
  }),

  http.get(P('/teams/:teamId/stats/members/:userId'), async ({ params }) => {
    await mockDelay();
    const teamId = params.teamId as string;
    const userId = params.userId as string;
    const to = todayLocalDate();
    const from = threeMonthsBeforeLocal(to);
    const events = db.events.filter((e) => e.teamId === teamId && e.status !== 'cancelled' && e.date >= from && e.date <= to);
    let yes = 0, counted = 0;
    events.forEach((e) => {
      const s = rawCountedStatus(e.id, userId);
      if (!s) return;
      counted++;
      if (s === 'yes') yes++;
    });
    const body: S['MemberAttendanceStats'] = { quote: counted ? yes / counted : 0, counted, yes };
    return HttpResponse.json(body);
  }),
];
