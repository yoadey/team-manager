export { TeamPage } from './TeamPage';
export { TeamsSheet, ProfileSheet, MoreSheet } from './components/NavSheets';
export { RolesSheet, RoleFormSheet } from './components/RoleSheets';
export { CreateTeamSheet, InviteSheet, TeamSettingsSheet } from './components/TeamSheets';
export { CalendarSharesSheet, SharedCalendarsSheet } from './components/CalendarShareSheets';
export { useTeamActions } from './hooks/useTeamActions';
export { useRoleActions } from './hooks/useRoleActions';
export { useCalendarShareActions } from './hooks/useCalendarShareActions';

import { TeamsSheet, ProfileSheet, MoreSheet } from './components/NavSheets';
import { RolesSheet, RoleFormSheet } from './components/RoleSheets';
import { CreateTeamSheet, InviteSheet, TeamSettingsSheet } from './components/TeamSheets';
import { CalendarSharesSheet, SharedCalendarsSheet } from './components/CalendarShareSheets';
export const teamSheetMap = {
  teams: TeamsSheet,
  profile: ProfileSheet,
  more: MoreSheet,
  roles: RolesSheet,
  roleForm: RoleFormSheet,
  createTeam: CreateTeamSheet,
  invite: InviteSheet,
  teamSettings: TeamSettingsSheet,
  calendarShares: CalendarSharesSheet,
  sharedCalendars: SharedCalendarsSheet,
} as const;
