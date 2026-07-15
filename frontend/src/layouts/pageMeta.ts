import type { useApp, SheetState } from '@/context/AppContext';
import type { TeamEvent } from '@/features/events';
import { fmtDateLong } from '@/styles/tokens';
import { t as tl } from '@/i18n';
import { shortName } from './useCompact';

export interface PM {
  title: string;
  subtitle: string;
  showPrimaryAction: boolean;
  primaryActionLabel: string;
  primaryActionIcon: string;
  primaryAction: () => void;
}

/**
 * `eventDetailEvent` is the currently-open `eventDetail` page sheet's event
 * (fetched by the caller via `useEventDetailQuery`, since the sheet itself no
 * longer carries it) -- undefined/null while that query is still loading.
 */
export function pageMeta(app: ReturnType<typeof useApp>, eventDetailEvent?: TeamEvent | null): PM {
  const { state } = app;
  const pageSheet = app.activePageSheet();
  if (pageSheet) return pageSheetMeta(app, pageSheet, eventDetailEvent);
  const noop = () => {};
  type M = [string, string, boolean, string?, string?, (() => void)?];
  const defs: Record<string, M> = {
    home: [tl('page.homeTitle'), tl('page.homeSubtitle'), false],
    events: [
      tl('nav.events'),
      tl('page.eventsSubtitle'),
      app.can('events', 'write'),
      tl('page.eventsAction'),
      'add',
      () => app.openEventForm(null),
    ],
    members: [
      tl('nav.members'),
      tl('page.membersSubtitle', {
        n: state.members?.length ?? 0,
        count: state.members?.length ?? 0,
      }),
      app.can('settings', 'write'),
      tl('page.membersAction'),
      'person_add',
      () => app.openInvite(),
    ],
    finances: [
      tl('nav.finances'),
      tl('page.financesSubtitle'),
      app.can('finances', 'write'),
      tl('page.financesAction'),
      'add',
      () => app.openTxForm(),
    ],
    stats: [tl('nav.stats'), tl('page.statsSubtitle'), false],
    news: [
      tl('nav.news'),
      tl('page.newsSubtitle'),
      app.can('news', 'write'),
      tl('page.newsAction'),
      'add',
      () => app.openNewsForm(),
    ],
    polls: [
      tl('nav.polls'),
      tl('page.pollsSubtitle'),
      app.can('polls', 'write'),
      tl('page.pollsAction'),
      'add',
      () => app.openPollForm(),
    ],
    team: [tl('nav.team'), tl('page.teamSubtitle'), false],
  };
  const d = defs[state.route] || defs['home'];
  return {
    title: d[0],
    subtitle: d[1],
    showPrimaryAction: !!d[2],
    primaryActionLabel: d[3] || '',
    primaryActionIcon: d[4] || 'add',
    primaryAction: d[5] || noop,
  };
}

function pageSheetMeta(app: ReturnType<typeof useApp>, s: SheetState, eventDetailEvent?: TeamEvent | null): PM {
  const team = app.activeTeam();
  const base = (title: string, subtitle: string): PM => ({
    title,
    subtitle,
    showPrimaryAction: false,
    primaryActionLabel: '',
    primaryActionIcon: 'add',
    primaryAction: () => {},
  });
  if (s.type === 'eventDetail') {
    const e = eventDetailEvent;
    return base(e ? e.title : tl('sheet.eventDetail'), e ? fmtDateLong(e.date) : tl('sheet.eventDetailSubtitle'));
  }
  if (s.type === 'eventForm')
    return base(
      s.mode === 'edit' ? tl('sheet.eventFormEdit') : tl('sheet.eventFormCreate'),
      s.mode === 'edit' ? tl('sheet.eventFormEditSubtitle') : tl('sheet.eventFormCreateSubtitle'),
    );
  if (s.type === 'memberDetail') {
    const m = s.member;
    return base(
      m ? m.name : tl('sheet.memberDetail'),
      m ? m.roles.map((r: { name: string }) => r.name).join(' · ') : tl('sheet.memberDetailSubtitle'),
    );
  }
  if (s.type === 'memberForm')
    return base(s.self ? tl('sheet.memberFormSelf') : tl('sheet.memberForm'), tl('sheet.memberFormSubtitle'));
  if (s.type === 'teamSettings') return base(tl('sheet.teamSettings'), team ? shortName(team.name) : '');
  if (s.type === 'roles') return base(tl('sheet.roles'), tl('sheet.rolesSubtitle'));
  if (s.type === 'roleForm')
    return base(s.mode === 'edit' ? tl('sheet.roleFormEdit') : tl('sheet.roleFormCreate'), tl('sheet.roleFormSubtitle'));
  return base('', '');
}
