// Mapping functions: API schema types → frontend domain types.
// Photos: the backend stores raw bytes and returns hasPhoto:boolean. Only two
// GET-by-bytes endpoints exist — the current user's own photo (/auth/me/photo)
// and a team's photo/logo (/teams/{teamId}/photo|logo) — so only those two
// entities can be resolved to a display URL here. Other members' photos
// (attendance rows, comments, poll voters, finance rows, stats, ...) have no
// per-arbitrary-user photo endpoint on the backend, so they stay null
// (initials/avatar fallback) until such an endpoint exists.

import { config } from '@/config';
import type { components } from './types.gen';
import type {
  User,
  Team,
  TeamForUser,
  Role,
  Permissions,
  Invite,
  MemberStat,
  EventStat,
  StatsOverview,
  Provider,
} from '@/types';
import type { Member, MemberDto } from '@/features/members';
import type {
  TeamEvent,
  AttendanceRow,
  EventSummary,
  EventComment,
  Absence,
} from '@/features/events';
import type { NewsItem } from '@/features/news';
import type { Poll } from '@/features/polls';
import type { AppNotification, NotificationsResult } from '@/features/notifications';
import type {
  FinanceOverview,
  Transaction,
  Penalty,
  PenaltyAssignment,
  Contribution,
} from '@/features/finances';

type S = components['schemas'];

// The backend represents money as integer cents (see openapi.yaml amount
// fields); the frontend domain model and UI keep working in euros (floats)
// to minimize changes to components/validation, so the conversion happens
// once at this API-mapping boundary.
export function centsToEuros(cents: number): number {
  return cents / 100;
}

export function eurosToCents(euros: number): number {
  return Math.round(euros * 100);
}

// photoUrl builds a cache-busted display URL for a hasPhoto-gated GET
// endpoint, or null when the entity has no photo. The session cookie is
// same-origin (or configured-CORS) and sent automatically by <img>/CSS
// background-image requests, so no auth token needs to be embedded here.
// The timestamp query param exists purely to bust the browser's HTTP cache
// on re-upload — it is recomputed only when the mapper re-runs (i.e. on a
// fresh API response), not on every render, so it doesn't defeat caching
// between unrelated re-renders of the same mapped object.
function photoUrl(hasPhoto: boolean | undefined, path: string): string | null {
  if (!hasPhoto) return null;
  return `${config.apiBaseUrl}/api/v1${path}?v=${Date.now()}`;
}

// The backend's attendance-quote fields (MemberStat.quote, EventStat.pct,
// StatsOverview.avg, MemberAttendanceStats.quote) are 0-1 fractions
// (internal/stats/service.go's quote() divides yes/counted); every frontend
// consumer (Stats.tsx, MemberSheets.tsx) renders them as a whole-number
// percentage (`+ '%'`) with 0-100 thresholds, matching the mock service
// layer's convention. Round to the nearest integer percentage here, once, so
// every caller of these mappers gets the scale the UI already assumes.
function fractionToPercent(fraction: number): number {
  return Math.round(fraction * 100);
}

export function mapUser(u: S['User']): User {
  return {
    id: u.id,
    name: u.name,
    email: u.email,
    phone: u.phone ?? '',
    avatarColor: u.avatarColor,
    photo: photoUrl(u.hasPhoto, '/auth/me/photo'),
    birthday: u.birthday ?? '',
    address: u.address ?? '',
  };
}

export function mapProvider(p: S['Provider']): Provider {
  return {
    id: p.id,
    name: p.name,
    sub: p.sub,
    glyph: p.glyph,
    bg: p.bg,
    fg: p.fg,
    border: p.border ? true : undefined,
  };
}

export function mapRole(r: S['Role']): Role {
  return {
    id: r.id,
    teamId: r.teamId,
    name: r.name,
    system: r.system,
    color: r.color ?? '#888888',
    permissions: r.permissions as Permissions,
  };
}

export function mapTeam(t: S['Team']): Team {
  return {
    id: t.id,
    name: t.name,
    short: t.short ?? t.name.charAt(0).toUpperCase(),
    icon: t.icon ?? '⭐',
    iconBg: t.iconBg ?? '#1565C0',
    iconFg: t.iconFg ?? '#FFFFFF',
    photo: photoUrl(t.hasPhoto, `/teams/${t.id}/photo`),
    logo: photoUrl(t.hasLogo, `/teams/${t.id}/logo`),
    description: t.description ?? '',
    reasonVisibilityRoles: t.reasonVisibilityRoleIds,
  };
}

export function mapTeamForUser(t: S['TeamForUser']): TeamForUser {
  return {
    ...mapTeam(t),
    myRoles: (t.myRoles ?? []).map(mapRole),
    myPerms: t.myPerms as Permissions,
    membershipId: t.membershipId,
    memberCount: t.memberCount,
  };
}

export function mapInvite(inv: S['Invite']): Invite {
  return {
    id: inv.id,
    teamId: inv.teamId,
    code: inv.code,
    link: inv.link,
    createdAt: inv.createdAt,
    expiresAt: inv.expiresAt,
  };
}

export function mapMember(m: S['Member']): Member {
  const dto: MemberDto = {
    membershipId: m.membershipId,
    userId: m.userId,
    name: m.name,
    email: m.email,
    phone: m.phone ?? '',
    birthday: m.birthday ?? '',
    address: m.address ?? '',
    avatarColor: m.avatarColor,
    photo: null,
    group: m.group ?? '',
    roles: (m.roles ?? []).map(mapRole),
    joinedAt: m.joinedAt,
  };
  return {
    ...dto,
    primaryRole: m.primaryRole ? mapRole(m.primaryRole) : null,
    perms: (m.perms ?? { events: 'none', members: 'none', finances: 'none', news: 'none', polls: 'none', settings: 'none' }) as Permissions,
  };
}

function mapEventSummary(s: S['EventSummary']): EventSummary {
  return {
    yes: s.yes,
    no: s.no,
    maybe: s.maybe,
    pending: s.pending,
    notNominated: s.notNominated,
    nominated: s.nominated,
    total: s.total,
  };
}

export function mapTeamEvent(e: S['TeamEvent']): TeamEvent {
  return {
    id: e.id,
    teamId: e.teamId,
    seriesId: e.seriesId ?? null,
    type: e.type,
    title: e.title,
    date: e.date,
    location: e.location ?? '',
    note: e.note ?? '',
    result: e.result,
    meetTime: e.meetTime ?? null,
    startTime: e.startTime ?? null,
    endTime: e.endTime ?? null,
    meetTimeMandatory: e.meetTimeMandatory ?? false,
    responseMode: (e.responseMode ?? 'opt_in') as 'opt_in' | 'opt_out',
    nominatedRoleIds: e.nominatedRoleIds,
    recurring: e.recurring,
    status: e.status,
    summary: mapEventSummary(e.summary),
    myStatus: e.myStatus ?? 'pending',
    myAuto: e.myAuto ?? false,
    myReason: '',
  };
}

export function mapAttendanceRow(r: S['AttendanceRow']): AttendanceRow {
  return {
    userId: r.userId,
    name: r.name,
    avatarColor: r.avatarColor,
    photo: null,
    group: r.group ?? '',
    primaryRole: r.primaryRole ? mapRole(r.primaryRole) : null,
    status: r.status,
    reason: r.reason ?? '',
    reasonId: r.reasonId ?? null,
    reasonVisibility: (r.reasonVisibility as 'trainers' | 'team' | null) ?? null,
    auto: r.auto ?? false,
    absent: r.absent ?? false,
  };
}

export function mapEventComment(c: S['EventComment']): EventComment {
  return {
    id: c.id,
    eventId: c.eventId,
    userId: c.userId,
    text: c.text,
    createdAt: c.createdAt,
    name: c.authorName,
    color: c.authorColor,
    photo: null,
  };
}

export function mapAbsence(a: S['Absence']): Absence {
  return {
    id: a.id,
    userId: a.userId,
    from: a.from,
    to: a.to,
    reason: a.reason ?? '',
    createdAt: a.createdAt,
    name: a.memberName,
    avatarColor: a.memberAvatarColor,
    photo: null,
    roleColor: a.roleColor,
    roleName: a.roleName,
  };
}

export function mapNewsItem(n: S['NewsItem']): NewsItem {
  return {
    id: n.id,
    teamId: n.teamId,
    authorId: n.authorId,
    title: n.title,
    body: n.body,
    pinned: n.pinned,
    createdAt: n.createdAt,
    authorName: n.authorName,
    authorColor: n.authorColor,
    authorPhoto: null,
  };
}

export function mapPoll(p: S['Poll']): Poll {
  return {
    id: p.id,
    question: p.question,
    multiple: p.multiple,
    anonymous: p.anonymous,
    createdAt: p.createdAt,
    totalVotes: p.totalVotes,
    myVote: p.myVote ?? [],
    options: (p.options ?? []).map((o) => ({
      id: o.id,
      text: o.text,
      count: o.count,
      pct: o.pct,
      voters: (o.voters ?? []).map((v) => ({
        name: v.name ?? '',
        color: v.color ?? '#888',
        photo: null,
      })),
    })),
  };
}

export function mapNotification(n: S['AppNotification']): AppNotification {
  return {
    id: n.id,
    teamId: n.teamId,
    type: n.type as AppNotification['type'],
    actorId: n.actorId,
    status: n.status as AppNotification['status'],
    title: n.title,
    eventId: n.eventId,
    eventTitle: n.eventTitle,
    eventDate: n.eventDate,
    note: n.note,
    createdAt: n.createdAt,
    actorName: n.actorName,
    actorColor: n.actorColor,
    actorPhoto: null,
    unread: n.unread ?? false,
  };
}

export function mapNotificationsResult(r: S['NotificationsResult']): NotificationsResult {
  return {
    items: r.items.map(mapNotification),
    unreadCount: r.unreadCount,
  };
}

export function mapTransaction(t: S['Transaction']): Transaction {
  return {
    id: t.id,
    teamId: t.teamId,
    type: t.type,
    title: t.title,
    amount: centsToEuros(t.amount),
    date: t.date,
    category: t.category ?? '',
  };
}

export function mapPenalty(p: S['Penalty']): Penalty {
  return {
    id: p.id,
    teamId: p.teamId,
    label: p.label,
    amount: centsToEuros(p.amount),
  };
}

export function mapPenaltyAssignment(a: S['PenaltyAssignment']): PenaltyAssignment {
  return {
    id: a.id,
    teamId: a.teamId,
    userId: a.userId,
    penaltyId: a.penaltyId,
    paid: a.paid,
    date: a.date,
    name: a.memberName,
    avatarColor: a.memberAvatarColor,
    photo: null,
    label: a.label,
    amount: a.amount == null ? a.amount : centsToEuros(a.amount),
  };
}

export function mapContribution(c: S['Contribution']): Contribution {
  return {
    id: c.id,
    teamId: c.teamId,
    userId: c.userId,
    month: c.month,
    label: c.label ?? '',
    amount: centsToEuros(c.amount),
    status: c.status,
    name: c.memberName,
    avatarColor: c.memberAvatarColor,
    photo: null,
  };
}

export function mapFinanceOverview(o: S['FinanceOverview']): FinanceOverview {
  return {
    balance: centsToEuros(o.balance),
    income: centsToEuros(o.income),
    expense: centsToEuros(o.expense),
    transactions: o.transactions.map(mapTransaction),
    penalties: o.penalties.map(mapPenalty),
    assignments: o.assignments.map(mapPenaltyAssignment),
    openPenalties: o.openPenalties.map((op) => ({
      userId: op.userId,
      name: op.name,
      avatarColor: op.avatarColor,
      photo: null,
      amount: centsToEuros(op.amount),
    })),
    openPenaltySum: centsToEuros(o.openPenaltySum),
    contributions: o.contributions.map(mapContribution),
    contribOpen: o.contribOpen,
  };
}

export function mapMemberStat(s: S['MemberStat']): MemberStat {
  return {
    userId: s.userId,
    name: s.name,
    avatarColor: s.avatarColor,
    photo: null,
    // counted === 0 means "no data yet", not "0% attendance" — Stats.tsx
    // renders these as distinct states (– / gray vs. 0% / red). The backend
    // always returns a number (0 when counted is 0), so map that case to
    // null here, matching the mock and stats.attendanceFor's convention.
    quote: s.counted > 0 ? fractionToPercent(s.quote) : null,
    counted: s.counted,
    yes: s.yes,
  };
}

export function mapEventStat(s: S['EventStat']): EventStat {
  return {
    id: s.id,
    title: s.title,
    type: s.type,
    date: s.date,
    yes: s.yes,
    nominated: s.nominated,
    pct: fractionToPercent(s.pct),
    enough: s.enough,
  };
}

export function mapStatsOverview(o: S['StatsOverview']): StatsOverview {
  return {
    avg: fractionToPercent(o.avg),
    members: o.members.map(mapMemberStat),
    events: o.events.map(mapEventStat),
    pastCount: o.pastCount,
    from: o.from,
    to: o.to,
  };
}
