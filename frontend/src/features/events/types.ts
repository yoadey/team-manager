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
  /** RSVP deadline as a `<input type="datetime-local">` value (YYYY-MM-DDTHH:mm), or '' for none. */
  rsvpDeadline: string;
  /** Cancellation/RSVP-change lead time before start, split for the hours+minutes inputs; both 0 means no cutoff. */
  cancelLeadHours: number;
  cancelLeadMinutes: number;
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
  /** ISO 8601 timestamp; null when the event has no RSVP deadline. */
  rsvpDeadline: string | null;
  /** Cutoff, in minutes before the event's start; null when the event has no cancellation lead time. */
  cancelLeadMinutes: number | null;
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
