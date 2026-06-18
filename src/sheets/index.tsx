import type { AppContextValue, SheetState } from '@/context/AppContext';
export type { SheetProps } from './types';
import { eventSheetMap } from '@/features/events';
import { financeSheetMap } from '@/features/finances';
import { memberSheetMap } from '@/features/members';
import { newsSheetMap } from '@/features/news';
import { notificationsSheetMap } from '@/features/notifications';
import { pollSheetMap } from '@/features/polls';
import { teamSheetMap } from '@/features/team';
import { ConfirmSheet, SeriesActionSheet, CommentSheet } from './DialogSheets';
import type { SheetProps } from './types';

type SheetComponent = React.ComponentType<SheetProps>;

const sheetRegistry: Record<string, SheetComponent> = {
  ...teamSheetMap,
  ...notificationsSheetMap,
  ...eventSheetMap,
  ...memberSheetMap,
  ...newsSheetMap,
  ...pollSheetMap,
  ...financeSheetMap,
  confirm: ConfirmSheet,
  seriesAction: SeriesActionSheet,
  comment: CommentSheet,
};

export function renderSheet(app: AppContextValue, sheet: SheetState) {
  const Comp = sheetRegistry[sheet.type];
  return Comp ? <Comp app={app} sheet={sheet} /> : null;
}

export function sheetMeta(
  app: AppContextValue,
  sheet: SheetState,
): { title: string; hasBack: boolean; onBack?: () => void; subtitle?: string } {
  const s = sheet;
  const titles: Record<string, string> = {
    teams: 'Team wechseln',
    profile: 'Konto & Rollen',
    more: 'Mehr',
    notifications: 'Benachrichtigungen',
    calExport: 'Kalender-Export',
    eventDetail: 'Termin',
    comment: 'Kommentar',
    confirm: 'Bestätigen',
    seriesAction: 'Serientermin',
    eventForm: s.mode === 'edit' ? 'Termin bearbeiten' : 'Neuer Termin',
    memberDetail: 'Mitglied',
    memberForm: 'Profil bearbeiten',
    roles: 'Rollen & Rechte',
    roleForm: 'Eigene Rolle',
    createTeam: 'Neues Team',
    invite: 'Einladungslink',
    teamSettings: 'Team-Einstellungen',
    absenceForm: s.mode === 'edit' ? 'Abwesenheit bearbeiten' : 'Abwesenheit eintragen',
    newsForm: 'Neuigkeit verfassen',
    txForm: s.mode === 'edit' ? 'Buchung bearbeiten' : 'Buchung erfassen',
    pollForm: 'Neue Umfrage',
    penaltyForm: s.mode === 'create' ? 'Strafe hinzufügen' : 'Strafe bearbeiten',
    penaltyCatalog: 'Strafenkatalog',
    penaltyAssign: 'Strafe erfassen',
    contribForm: 'Beitrag bearbeiten',
  };
  const meta: { title: string; hasBack: boolean; onBack?: () => void; subtitle?: string } = {
    title: titles[s.type] || '',
    hasBack: false,
  };
  if (s.type === 'eventDetail' && s.event) meta.title = s.event.title;
  if (s.type === 'roleForm') {
    meta.hasBack = true;
    meta.onBack = () => app.openRoles();
  }
  if (s.type === 'comment' && s.eventId) {
    meta.hasBack = true;
    meta.onBack = () => app.openEventDetail(s.eventId!);
  }
  if (s.type === 'penaltyForm' && s.back && s.back.type === 'penaltyCatalog') {
    meta.hasBack = true;
    meta.onBack = () => app.openPenaltyCatalog();
  }
  if (s.type === 'seriesAction')
    meta.title =
      s.action === 'delete' ? 'Termin löschen' : s.action === 'reactivate' ? 'Termin aktivieren' : 'Termin absagen';
  return meta;
}
