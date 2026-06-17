import type { AppContextValue, SheetState } from '../store/AppContext';
export type { SheetProps } from './types';
import {
  TeamsSheet, ProfileSheet, MoreSheet,
} from './NavSheets';
import { NotificationsSheet } from './NotificationsSheet';
import { CalExportSheet } from './CalExportSheet';
import { ConfirmSheet, SeriesActionSheet, CommentSheet } from './DialogSheets';
import { EventDetailSheet } from './EventDetailSheet';
import { EventFormSheet } from './EventFormSheet';
import { MemberDetailSheet, MemberFormSheet } from './MemberSheets';
import { RolesSheet, RoleFormSheet } from './RoleSheets';
import { CreateTeamSheet, InviteSheet, TeamSettingsSheet } from './TeamSheets';
import { AbsenceFormSheet, NewsFormSheet, PollFormSheet } from './MiscSheets';
import { TxFormSheet, PenaltyCatalogSheet, PenaltyFormSheet, PenaltyAssignSheet, ContribFormSheet } from './FinanceSheets';

export function renderSheet(app: AppContextValue, sheet: SheetState) {
  const p = { app, sheet };
  switch (sheet.type) {
    case 'teams': return <TeamsSheet {...p} />;
    case 'profile': return <ProfileSheet {...p} />;
    case 'more': return <MoreSheet {...p} />;
    case 'notifications': return <NotificationsSheet {...p} />;
    case 'calExport': return <CalExportSheet {...p} />;
    case 'confirm': return <ConfirmSheet {...p} />;
    case 'seriesAction': return <SeriesActionSheet {...p} />;
    case 'comment': return <CommentSheet {...p} />;
    case 'eventDetail': return <EventDetailSheet {...p} />;
    case 'eventForm': return <EventFormSheet {...p} />;
    case 'memberDetail': return <MemberDetailSheet {...p} />;
    case 'memberForm': return <MemberFormSheet {...p} />;
    case 'roles': return <RolesSheet {...p} />;
    case 'roleForm': return <RoleFormSheet {...p} />;
    case 'createTeam': return <CreateTeamSheet {...p} />;
    case 'invite': return <InviteSheet {...p} />;
    case 'teamSettings': return <TeamSettingsSheet {...p} />;
    case 'absenceForm': return <AbsenceFormSheet {...p} />;
    case 'newsForm': return <NewsFormSheet {...p} />;
    case 'pollForm': return <PollFormSheet {...p} />;
    case 'txForm': return <TxFormSheet {...p} />;
    case 'penaltyCatalog': return <PenaltyCatalogSheet {...p} />;
    case 'penaltyForm': return <PenaltyFormSheet {...p} />;
    case 'penaltyAssign': return <PenaltyAssignSheet {...p} />;
    case 'contribForm': return <ContribFormSheet {...p} />;
    default: return null;
  }
}

export function sheetMeta(app: AppContextValue, sheet: SheetState): { title: string; hasBack: boolean; onBack?: () => void; subtitle?: string } {
  const s = sheet;
  const titles: Record<string, string> = {
    teams: 'Team wechseln', profile: 'Konto & Rollen', more: 'Mehr', notifications: 'Benachrichtigungen',
    calExport: 'Kalender-Export', eventDetail: 'Termin', comment: 'Kommentar', confirm: 'Bestätigen',
    seriesAction: 'Serientermin', eventForm: s.mode === 'edit' ? 'Termin bearbeiten' : 'Neuer Termin',
    memberDetail: 'Mitglied', memberForm: 'Profil bearbeiten', roles: 'Rollen & Rechte', roleForm: 'Eigene Rolle',
    createTeam: 'Neues Team', invite: 'Einladungslink', teamSettings: 'Team-Einstellungen',
    absenceForm: s.mode === 'edit' ? 'Abwesenheit bearbeiten' : 'Abwesenheit eintragen',
    newsForm: 'Neuigkeit verfassen', txForm: s.mode === 'edit' ? 'Buchung bearbeiten' : 'Buchung erfassen',
    pollForm: 'Neue Umfrage', penaltyForm: s.mode === 'create' ? 'Strafe hinzufügen' : 'Strafe bearbeiten',
    penaltyCatalog: 'Strafenkatalog', penaltyAssign: 'Strafe erfassen', contribForm: 'Beitrag bearbeiten',
  };
  const meta: { title: string; hasBack: boolean; onBack?: () => void; subtitle?: string } = { title: titles[s.type] || '', hasBack: false };
  if (s.type === 'eventDetail' && s.event) meta.title = s.event.title;
  if (s.type === 'roleForm') { meta.hasBack = true; meta.onBack = () => app.openRoles(); }
  if (s.type === 'comment' && s.eventId) { meta.hasBack = true; meta.onBack = () => app.openEventDetail(s.eventId); }
  if (s.type === 'penaltyForm' && s.back && s.back.type === 'penaltyCatalog') { meta.hasBack = true; meta.onBack = () => app.openPenaltyCatalog(); }
  if (s.type === 'seriesAction') meta.title = s.action === 'delete' ? 'Termin löschen' : (s.action === 'reactivate' ? 'Termin aktivieren' : 'Termin absagen');
  return meta;
}
