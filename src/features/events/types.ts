import type { EventType, EventStatus, AttendanceStatus, ReasonVisibility, Role } from '@/types';

export type ResponseMode = 'opt_in' | 'opt_out';

// --- Editing buffer shapes for the event sheets ---

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
}

/** Plan-an-absence sheet. */
export interface AbsenceFormValues extends Record<string, unknown> {
  id?: string;
  from: string;
  to: string;
  reason: string;
}

/** Attendance-reason comment sheet (per member/event). */
export interface AttendanceCommentFormValues extends Record<string, unknown> {
  commentText: string;
}

/** New-event-comment input on the event detail sheet. */
export interface EventCommentFormValues extends Record<string, unknown> {
  newEventComment: string;
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
