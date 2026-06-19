import type { AppContextValue, SheetState } from '@/context/AppContext';
export type { SheetProps } from './types';
import { t } from '@/i18n';
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
    teams: t('sheet.teams'),
    profile: t('sheet.profile'),
    more: t('sheet.more'),
    notifications: t('shell.notifications'),
    calExport: t('sheet.calExport'),
    eventDetail: t('sheet.eventDetail'),
    comment: t('sheet.comment'),
    confirm: t('common.confirm'),
    seriesAction: t('sheet.seriesAction'),
    eventForm: s.mode === 'edit' ? t('sheet.eventFormEdit') : t('sheet.eventFormCreate'),
    memberDetail: t('sheet.memberDetail'),
    memberForm: t('sheet.memberForm'),
    roles: t('sheet.roles'),
    roleForm: t('sheet.roleForm'),
    createTeam: t('sheet.createTeam'),
    invite: t('sheet.invite'),
    teamSettings: t('sheet.teamSettings'),
    absenceForm: s.mode === 'edit' ? t('sheet.absenceFormEdit') : t('sheet.absenceFormCreate'),
    newsForm: s.mode === 'edit' ? t('sheet.newsFormEdit') : t('sheet.newsFormCreate'),
    txForm: s.mode === 'edit' ? t('sheet.txFormEdit') : t('sheet.txFormCreate'),
    pollForm: t('sheet.pollForm'),
    penaltyForm: s.mode === 'create' ? t('sheet.penaltyFormCreate') : t('sheet.penaltyFormEdit'),
    penaltyCatalog: t('sheet.penaltyCatalog'),
    penaltyAssign: t('sheet.penaltyAssign'),
    contribForm: t('sheet.contribForm'),
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
      s.action === 'delete'
        ? t('sheet.seriesActionDelete')
        : s.action === 'reactivate'
          ? t('sheet.seriesActionReactivate')
          : t('sheet.seriesActionCancel');
  return meta;
}
