import type { EventType, EventStatus, AttendanceStatus, ReasonVisibility, Role } from '@/types';

export type ResponseMode = 'opt_in' | 'opt_out';

// --- Editing buffer shapes for the event sheets ---

/** How a recurring series' extent is specified: a fixed occurrence count, or an end date. */
export type RepeatMode = 'weeks' | 'until';

/** Event create/edit sheet (times held as HH:MM strings). `id`/`seriesId` set when editing. */
export interface EventFormValues extends Record<string, unknown> {
  id?: string;
  seriesId?: string | null;
  type: EventType;
  title: string;
  date: string;
  /** Optional last day of a multi-day span (YYYY-MM-DD); empty for a single-day event. Only meaningful when !recurring. */
  multiDayEndDate: string;
  meetT: string;
  startT: string;
  endT: string;
  location: string;
  note: string;
  meetTimeMandatory: boolean;
  responseMode: ResponseMode;
  nominatedRoleIds: string[];
  recurring: boolean;
  repeatWeeks: number;
  /** Which of repeatWeeks/repeatEndDate the "N weeks" vs. "until date" toggle currently drives. */
  repeatMode: RepeatMode;
  /** Recurrence end date (YYYY-MM-DD), used instead of repeatWeeks when repeatMode === 'until'. */
  repeatEndDate: string;
  /** Cancellation/RSVP-change lead time before start, split for the hours+minutes inputs; both 0 means no cutoff. */
  cancelLeadHours: number;
  cancelLeadMinutes: number;
  /** When true, this event is left out of every attendance-statistics computation. */
  excludeFromStats: boolean;
}

/** Plan-an-absence sheet. */
export interface AbsenceFormValues extends Record<string, unknown> {
  id?: string;
  from: string;
  to: string;
  reason: string;
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

/** Raw event payload as it should be returned by events.* API endpoints. */
export interface EventDto {
  id: string;
  teamId: string;
  type: EventType;
  title: string;
  /** Calendar date in local team/user context, formatted as YYYY-MM-DD. */
  date: string;
  /** Optional last day of a multi-day span (YYYY-MM-DD, inclusive); null for a single-day event. */
  multiDayEndDate: string | null;
  location: string;
  note: string;
  result?: string;
  /** Local wall-clock times for date, formatted as HH:mm. */
  meetTime: string | null;
  startTime: string | null;
  endTime: string | null;
  meetTimeMandatory: boolean;
  responseMode: ResponseMode;
  nominatedRoleIds?: string[];
  recurring: boolean;
  seriesId: string | null;
  status: EventStatus;
  /** Cutoff, in minutes before the event's start; null when the event has no cancellation lead time. */
  cancelLeadMinutes: number | null;
  /** When true, this event is left out of every attendance-statistics computation. */
  excludeFromStats: boolean;
  /** Other teams (besides teamId, the owning team) this event targets. Members of any of these teams see the event and its merged attendance. Absent/empty for a single-team event. */
  crossTeamIds?: string[];
}

/** UI ViewModel consumed by event screens; summary and my* are client-side enrichment. */
export interface TeamEvent extends EventDto {
  summary: EventSummary;
  myStatus: AttendanceStatus;
  myAuto: boolean;
  myReason: string;
}

/** Raw attendance payload as it should be returned by attendance.* API endpoints. */
export interface AttendanceDto {
  id: string;
  eventId: string;
  userId: string;
  status: AttendanceStatus;
  reason: string;
  reasonId: string | null;
  reasonVisibility: ReasonVisibility;
  at?: string;
}

/** UI ViewModel for event attendance lists. */
export interface AttendanceRow {
  userId: string;
  name: string;
  avatarColor: string;
  photo: string | null;
  group: string;
  title: string;
  primaryRole: Role | null;
  status: AttendanceStatus;
  reason: string;
  reasonId: string | null;
  reasonVisibility: ReasonVisibility;
  auto: boolean;
  absent: boolean;
  /** Cross-team event only: set when this attendee does not belong to the viewer's own (currently active) team -- the alphabetically-first (by team name) team, among the event's targeted teams, that the attendee belongs to. Absent for an attendee who shares the viewer's own team, and always absent on a single-team event. When set, membershipId/group/title/primaryRole/reason/reasonId/reasonVisibility are absent from the underlying row -- no profile navigation or reason/absence detail for this attendee. */
  teamName?: string;
  /** True when this row's identity fields (membershipId/group/title/primaryRole/reason*) were stripped server-side because the attendee is outside the viewer's own team -- derived from membershipId's absence, NOT from teamName. The backend's badge computation has an accepted fail-closed race window (see resolveCrossTeamBadgeContext) where a foreign attendee's identity gets redacted but no teamName badge is assigned; teamName alone would then read as "same team" and offer RSVP/comment controls that are guaranteed to be rejected server-side. Always false on a single-team event. */
  foreign: boolean;
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
  /** When true, the event dates this absence covers are excluded entirely from this member's attendance statistics. */
  notRelevantForStats: boolean;
  // enriched in listForTeam
  name?: string;
  avatarColor?: string;
  photo?: string | null;
  roleColor?: string;
  roleName?: string;
}
