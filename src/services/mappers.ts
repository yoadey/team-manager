import type {
  AttendanceDto,
  AttendanceRow,
  EventDto,
  EventSummary,
  Member,
  MemberDto,
  Role,
  TeamEvent,
} from './types';

/** Builds the event ViewModel expected by the UI from a raw event DTO plus client aggregates. */
export function mapEventDtoToTeamEvent(
  event: EventDto,
  summary: EventSummary,
  mine: Pick<TeamEvent, 'myStatus' | 'myAuto' | 'myReason'>,
): TeamEvent {
  return { ...event, summary, ...mine };
}

/** Builds the member ViewModel expected by the UI from a raw member DTO plus derived permissions. */
export function mapMemberDtoToMember(
  member: MemberDto,
  primaryRole: Role | null,
  perms: Member['perms'],
): Member {
  return { ...member, primaryRole, perms };
}

/** Builds the attendance row ViewModel expected by the UI from raw attendance plus member display data. */
export function mapAttendanceDtoToRow(
  attendance: AttendanceDto,
  display: Pick<AttendanceRow, 'userId' | 'name' | 'avatarColor' | 'photo' | 'group' | 'primaryRole' | 'auto' | 'absent'>,
): AttendanceRow {
  return { ...display, status: attendance.status, reason: attendance.reason, reasonId: attendance.reasonId, reasonVisibility: attendance.reasonVisibility };
}
