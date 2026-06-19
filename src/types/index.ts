// =============================================================================
// Shared domain types — cross-cutting concerns used by multiple features,
// shared infrastructure (styles, services, context), or both.
// Feature-specific types live in src/features/<feature>/types.ts.
// =============================================================================

export type PermLevel = 'none' | 'read' | 'write';
export type ModuleKey = 'events' | 'members' | 'finances' | 'news' | 'polls' | 'settings';
export type Permissions = Record<ModuleKey, PermLevel>;

export type EventType = 'training' | 'auftritt' | 'event';
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

/** Raw role payload as it should be returned by roles.* API endpoints. */
export interface RoleDto {
  id: string;
  teamId: string;
  name: string;
  system: boolean;
  color: string;
  permissions: Permissions;
}

export type Role = RoleDto;

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

export interface MemberAttendanceStats {
  quote: number | null;
  counted: number;
  yes: number;
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
