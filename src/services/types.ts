// =============================================================================
// Domain types — shared contract between the (mock) service layer and the UI.
// These mirror the API surface described in the Lastenheft / design handoff.
// =============================================================================

export type PermLevel = 'none' | 'read' | 'write';
export type ModuleKey = 'events' | 'members' | 'finances' | 'news' | 'polls' | 'settings';
export type Permissions = Record<ModuleKey, PermLevel>;

export type EventType = 'training' | 'auftritt' | 'event';
export type ResponseMode = 'opt_in' | 'opt_out';
export type EventStatus = 'active' | 'cancelled';
export type AttendanceStatus = 'yes' | 'maybe' | 'no' | 'pending' | 'not_nominated';
export type ReasonVisibility = 'trainers' | 'team' | null;

export interface User {
  id: string;
  name: string;
  email: string;
  phone: string;
  avatarColor: string;
  photo: string | null;
  birthday: string;
  address: string;
}

export interface Role {
  id: string;
  teamId: string;
  name: string;
  system: boolean;
  color: string;
  permissions: Permissions;
}

export interface Team {
  id: string;
  name: string;
  short: string;
  icon: string;
  iconBg: string;
  iconFg: string;
  photo: string | null;
  logo: string | null;
  description: string;
  reasonVisibilityRoles?: string[];
}

/** Team enriched for the current user (returned by teams.listForCurrentUser). */
export interface TeamForUser extends Team {
  myRoles: Role[];
  myPerms: Permissions;
  membershipId: string;
  memberCount: number;
}

export interface Membership {
  id: string;
  teamId: string;
  userId: string;
  roleIds: string[];
  group: string;
  joinedAt: string;
}

export interface Member {
  membershipId: string;
  userId: string;
  name: string;
  email: string;
  phone: string;
  birthday: string;
  address: string;
  avatarColor: string;
  photo: string | null;
  group: string;
  roles: Role[];
  primaryRole: Role | null;
  perms: Permissions;
  joinedAt: string;
}

export interface EventSummary {
  yes: number;
  no: number;
  maybe: number;
  pending: number;
  notNominated: number;
  nominated: number;
  total: number;
}

export interface TeamEvent {
  id: string;
  teamId: string;
  type: EventType;
  title: string;
  /** Calendar date in local team/user context, formatted as YYYY-MM-DD. */
  date: string;
  location: string;
  note: string;
  result?: string;
  /** Local wall-clock times for date, formatted as HH:mm. */
  meetTime: string | null;
  startTime: string | null;
  endTime: string | null;
  meetTimeMandatory: boolean;
  responseMode: ResponseMode;
  recurring: boolean;
  seriesId: string | null;
  status: EventStatus;
  // attached by _withSummary
  summary: EventSummary;
  myStatus: AttendanceStatus;
  myAuto: boolean;
  myReason: string;
}

export interface AttendanceRow {
  userId: string;
  name: string;
  avatarColor: string;
  photo: string | null;
  group: string;
  primaryRole: Role | null;
  status: AttendanceStatus;
  reason: string;
  reasonId: string | null;
  reasonVisibility: ReasonVisibility;
  auto: boolean;
  absent: boolean;
}

export interface EventComment {
  id: string;
  eventId: string;
  userId: string;
  text: string;
  createdAt: string;
  name?: string;
  color?: string;
  photo?: string | null;
}

export interface Absence {
  id: string;
  userId: string;
  from: string;
  to: string;
  reason: string;
  createdAt: string;
  // enriched in listForTeam
  name?: string;
  avatarColor?: string;
  photo?: string | null;
  roleColor?: string;
  roleName?: string;
}

export interface NewsItem {
  id: string;
  teamId: string;
  title: string;
  body: string;
  authorId: string;
  pinned: boolean;
  createdAt: string;
  authorName?: string;
  authorColor?: string;
  authorPhoto?: string | null;
}

export interface Transaction {
  id: string;
  teamId: string;
  type: 'income' | 'expense';
  title: string;
  amount: number;
  date: string;
  category: string;
}

export interface Penalty {
  id: string;
  teamId: string;
  label: string;
  amount: number;
}

export interface PenaltyAssignment {
  id: string;
  teamId: string;
  userId: string;
  penaltyId: string;
  paid: boolean;
  date: string;
  name?: string;
  avatarColor?: string;
  photo?: string | null;
  label?: string;
  amount?: number;
}

export interface OpenPenalty {
  userId: string;
  name: string;
  avatarColor: string;
  photo: string | null;
  amount: number;
}

export interface Contribution {
  id: string;
  teamId: string;
  userId: string;
  month: string;
  label: string;
  amount: number;
  status: 'paid' | 'open';
  name?: string;
  avatarColor?: string;
  photo?: string | null;
}

export interface FinanceOverview {
  balance: number;
  income: number;
  expense: number;
  transactions: Transaction[];
  penalties: Penalty[];
  assignments: PenaltyAssignment[];
  openPenalties: OpenPenalty[];
  openPenaltySum: number;
  contributions: Contribution[];
  contribOpen: number;
}

export interface MemberStat {
  userId: string;
  name: string;
  avatarColor: string;
  photo: string | null;
  quote: number | null;
  counted: number;
  yes: number;
}

export interface EventStat {
  id: string;
  title: string;
  type: EventType;
  date: string;
  yes: number;
  nominated: number;
  pct: number;
  enough?: boolean;
}

export interface StatsOverview {
  avg: number;
  members: MemberStat[];
  events: EventStat[];
  pastCount: number;
  from: string | null;
  to: string | null;
}

export interface PollOption {
  id: string;
  text: string;
  count: number;
  pct: number;
  voters: { name: string; color: string; photo: string | null }[];
}

export interface Poll {
  id: string;
  question: string;
  multiple: boolean;
  anonymous: boolean;
  createdAt: string;
  totalVotes: number;
  myVote: string[] | null;
  options: PollOption[];
}

export type NotificationType =
  | 'attendance'
  | 'event_created'
  | 'event_updated'
  | 'event_cancelled'
  | 'event_reactivated'
  | 'event_deleted'
  | 'news'
  | 'poll'
  | 'absence';

export interface AppNotification {
  id: string;
  teamId: string;
  type: NotificationType;
  actorId?: string;
  status?: AttendanceStatus;
  title?: string;
  eventId?: string | null;
  eventTitle?: string;
  eventDate?: string;
  note?: string;
  createdAt: string;
  actorName?: string;
  actorColor?: string;
  actorPhoto?: string | null;
  unread?: boolean;
}

export interface NotificationsResult {
  items: AppNotification[];
  unreadCount: number;
}

export interface Provider {
  id: string;
  name: string;
  sub: string;
  glyph: string;
  bg: string;
  fg: string;
  border?: boolean;
}

export interface Invite {
  id: string;
  teamId: string;
  code: string;
  link: string;
  createdAt: string;
  expiresAt: string;
}

export interface DateRange {
  from: string | null;
  to: string | null;
}
