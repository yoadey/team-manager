import React from 'react';
import Box from '@mui/material/Box';
import ButtonBase from '@mui/material/ButtonBase';
import type { SheetProps } from '@/sheets/types';
import type { AppNotification } from '../types';
import { buildTokens, statusMeta, fmtDate, relTime } from '@/styles/tokens';
import { getIntlLocale, t } from '@/i18n';
import { Av, EmptyState, SpinnerBox } from '@/components/ui';

interface NotifMeta {
  col: string;
  bg: string;
  icon: string;
  line1: string;
  line2: string;
  onClick: (() => void) | null;
  group: string;
  avatar?: boolean;
}

function notifDayLabel(isoStr: string) {
  const d = new Date(isoStr);
  const a = new Date();
  a.setHours(0, 0, 0, 0);
  const b = new Date(d);
  b.setHours(0, 0, 0, 0);
  const diff = Math.round((a.getTime() - b.getTime()) / 86400000);
  if (diff <= 0) return t('notifications.today');
  if (diff === 1) return t('notifications.yesterday');
  if (diff < 7) return t('notifications.thisWeek');
  if (diff < 14) return t('notifications.lastWeek');
  return new Intl.DateTimeFormat(getIntlLocale(), { month: 'long', year: 'numeric' }).format(d);
}

export function NotificationsSheet({ app }: SheetProps) {
  const { state } = app;
  const tk = buildTokens(state.primaryColor);
  const S = state;

  const notifMeta = (n: AppNotification): NotifMeta => {
    if (n.type === 'attendance') {
      const sm = statusMeta(n.status!);
      const verb =
        n.status === 'yes'
          ? t('notifications.attendanceYes')
          : n.status === 'no'
            ? t('notifications.attendanceNo')
            : t('notifications.attendanceMaybe');
      return {
        col: sm.color,
        bg: sm.bg,
        icon: sm.icon,
        line1: n.actorName + ' ' + verb,
        line2: n.eventTitle + ' · ' + fmtDate(n.eventDate!),
        onClick: n.eventId ? () => app.openEventDetail(n.eventId!) : null,
        group: 'attendance',
        avatar: true,
      };
    }
    if (n.type && n.type.indexOf('event_') === 0) {
      const map: Record<string, [string, string, string, string]> = {
        event_created: ['event_available', '#1565C0', '#D7E3FF', t('notifications.eventCreated')],
        event_updated: ['edit_calendar', '#9A5B00', '#FFE5B8', t('notifications.eventUpdated')],
        event_cancelled: ['event_busy', '#BA1A1A', '#FFDAD6', t('notifications.eventCancelled')],
        event_reactivated: ['event_available', '#2E7D32', '#D7F0D8', t('notifications.eventReactivated')],
        event_deleted: ['delete', '#BA1A1A', '#FFDAD6', t('notifications.eventDeleted')],
      };
      const m = map[n.type] || ['event', '#6A6D76', '#ECEDF3', 'Event'];
      return {
        icon: m[0],
        col: m[1],
        bg: m[2],
        line1: m[3],
        line2: n.title + (n.note ? ' · ' + n.note : '') + (n.actorName ? ' · ' + n.actorName : ''),
        onClick: n.eventId && n.type !== 'event_deleted' ? () => app.openEventDetail(n.eventId!) : null,
        group: 'events',
      };
    }
    if (n.type === 'news')
      return {
        icon: 'campaign',
        col: '#6750A4',
        bg: '#EADDFF',
        line1: t('notifications.newsNew'),
        line2: n.title + ' · ' + n.actorName,
        onClick: () => app.go('news'),
        group: 'other',
      };
    if (n.type === 'poll')
      return {
        icon: 'how_to_vote',
        col: '#00796B',
        bg: '#9DF1E2',
        line1: t('notifications.pollNew'),
        line2: n.title + ' · ' + n.actorName,
        onClick: () => app.go('polls'),
        group: 'other',
      };
    if (n.type === 'absence')
      return {
        icon: 'beach_access',
        col: '#8A6100',
        bg: '#FFE5B8',
        line1: n.actorName + ' ' + t('notifications.absenceLogged'),
        line2: n.title || '',
        onClick: () => {
          app.setState({ route: 'events', sheet: null, eventsView: 'absences' });
          app.loadAbsences();
        },
        group: 'other',
        avatar: true,
      };
    return {
      icon: 'notifications',
      col: '#6A6D76',
      bg: '#ECEDF3',
      line1: n.title || '',
      line2: '',
      onClick: null,
      group: 'other',
    };
  };

  if (!S.notifications) return <SpinnerBox />;

  const filt = S.notifFilter || 'all';
  const chips: Array<[string, string]> = [
    ['all', t('notifications.filterAll')],
    ['attendance', t('notifications.filterAttendance')],
    ['events', t('notifications.filterEvents')],
    ['other', t('notifications.filterOther')],
  ];
  const chipBar = (
    <Box key="cb" sx={{ display: 'flex', gap: '7px', flexWrap: 'wrap', mb: '14px' }}>
      {chips.map(([k, l]) => {
        const sel = filt === k;
        return (
          <ButtonBase
            key={k}
            onClick={() => app.setNotifFilter(k as typeof S.notifFilter)}
            sx={{
              p: '7px 13px',
              borderRadius: '999px',
              fontSize: '12px',
              fontWeight: 700,
              cursor: 'pointer',
              border: '1.5px solid ' + (sel ? tk.primary : '#D0D2DA'),
              background: sel ? tk.primaryContainer : '#fff',
              color: sel ? tk.onPrimaryContainer : '#6A6D76',
            }}
          >
            {l}
          </ButtonBase>
        );
      })}
    </Box>
  );

  const list = (S.notifications || []).filter((n) => filt === 'all' || notifMeta(n).group === filt);
  if (!list.length) {
    return (
      <Box>
        {chipBar}
        <EmptyState icon="notifications_off" text={t('notifications.empty')} />
      </Box>
    );
  }

  const out: React.ReactNode[] = [chipBar];
  let lastDay: string | null = null;
  let dayCount = 0;
  list.forEach((n) => {
    const m = notifMeta(n);
    const day = notifDayLabel(n.createdAt);
    if (day !== lastDay) {
      lastDay = day;
      out.push(
        <Box
          key={'d' + n.id}
          sx={{
            fontSize: '11px',
            fontWeight: 700,
            color: '#9A9DA6',
            letterSpacing: '.4px',
            textTransform: 'uppercase',
            m: (dayCount > 0 ? '14px' : '2px') + ' 2px 8px',
          }}
        >
          {day}
        </Box>,
      );
      dayCount++;
    }
    const lead =
      m.avatar && n.actorName ? (
        <Av key="a" name={n.actorName} photo={n.actorPhoto} color={n.actorColor} size={38} font={14} />
      ) : (
        <Box
          key="i"
          component="span"
          sx={{
            width: '38px',
            height: '38px',
            borderRadius: '11px',
            background: m.bg,
            color: m.col,
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            fontFamily: "'Material Symbols Outlined'",
            fontSize: '20px',
            flex: '0 0 auto',
          }}
        >
          {m.icon}
        </Box>
      );
    const dot = n.unread ? (
      <Box
        key="u"
        component="span"
        sx={{ width: '9px', height: '9px', borderRadius: '50%', background: tk.primary, flex: '0 0 auto' }}
      />
    ) : null;
    const body = (
      <>
        {lead}
        <Box key="m" sx={{ flex: 1, minWidth: 0 }}>
          <Box key="l1" sx={{ fontSize: '14px', fontWeight: n.unread ? 700 : 600, color: '#1A1C20', lineHeight: 1.3 }}>
            {m.line1}
          </Box>
          <Box
            key="l2"
            sx={{
              fontSize: '12px',
              color: '#6A6D76',
              mt: '2px',
              whiteSpace: 'nowrap',
              overflow: 'hidden',
              textOverflow: 'ellipsis',
            }}
          >
            {m.line2}
          </Box>
        </Box>
        <Box key="r" sx={{ display: 'flex', alignItems: 'center', gap: '8px', flex: '0 0 auto' }}>
          <Box key="t" component="span" sx={{ fontSize: '11px', color: '#9A9DA6', whiteSpace: 'nowrap' }}>
            {relTime(n.createdAt)}
          </Box>
          {dot}
        </Box>
      </>
    );
    const st = {
      display: 'flex',
      alignItems: 'center',
      gap: '12px',
      width: '100%',
      textAlign: 'left',
      p: '10px 12px',
      borderRadius: '14px',
      border: '1px solid ' + (n.unread ? tk.primaryContainer : '#ECEDF3'),
      background: n.unread ? '#FBFAFF' : '#fff',
      mb: '7px',
    } as const;
    out.push(
      m.onClick ? (
        <ButtonBase key={n.id} onClick={m.onClick} sx={{ ...st, cursor: 'pointer', justifyContent: 'flex-start' }}>
          {body}
        </ButtonBase>
      ) : (
        <Box key={n.id} sx={st}>
          {body}
        </Box>
      ),
    );
  });
  out.push(
    <Box key="foot" sx={{ fontSize: '12px', color: '#9A9DA6', textAlign: 'center', p: '14px 0 4px', lineHeight: 1.5 }}>
      {t('notifications.footer')}
    </Box>,
  );
  return <Box>{out}</Box>;
}
