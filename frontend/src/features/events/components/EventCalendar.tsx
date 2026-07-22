import React from 'react';
import Box from '@mui/material/Box';
import ButtonBase from '@mui/material/ButtonBase';
import { useApp } from '@/context/AppContext';
import { useCompact } from '@/layouts/useCompact';
import { buildTokens, hhmm, typeMeta, NEUTRAL } from '@/styles/tokens';
import { formatDateOnly, parseDateOnlyLocal, todayLocalDate } from '@/utils/date';
import { getIntlLocale, t } from '@/i18n';
import { Sym, Card } from '@/components/ui';
import { useEventsQuery } from '../hooks/useEventQueries';
import { useAbsencesQuery } from '../hooks/useAbsenceQueries';
import type { Absence, TeamEvent } from '../types';

function groupEventsByDate(events: TeamEvent[] | undefined): Record<string, TeamEvent[]> {
  const byDate: Record<string, TeamEvent[]> = {};
  (events ?? []).forEach((e) => {
    (byDate[e.date] = byDate[e.date] || []).push(e);
  });
  return byDate;
}

function groupAbsencesByDate(absences: Absence[] | undefined, show: boolean): Record<string, Absence[]> {
  const byDate: Record<string, Absence[]> = {};
  if (!show || !absences) return byDate;
  absences.forEach((a) => {
    let d = parseDateOnlyLocal(a.from);
    const end = parseDateOnlyLocal(a.to);
    while (d <= end) {
      const ds = formatDateOnly(d);
      (byDate[ds] = byDate[ds] || []).push(a);
      // Increment by calendar day, not a fixed 24h in ms -- across a DST
      // transition the local day is 23 or 25 hours, so +86400000 either
      // lands on the same date twice or skips a day entirely.
      d = new Date(d.getFullYear(), d.getMonth(), d.getDate() + 1);
    }
  });
  return byDate;
}

function EventChip({ event, mobile, onOpen }: { event: TeamEvent; mobile: boolean; onOpen: () => void }) {
  const tm = typeMeta(event.type);
  return (
    <ButtonBase
      onClick={onOpen}
      sx={{
        display: 'block',
        width: '100%',
        textAlign: 'left',
        justifyContent: 'flex-start',
        border: 'none',
        background: tm.bg,
        color: tm.on,
        borderRadius: '5px',
        p: '2px 5px',
        fontSize: mobile ? '9px' : '10px',
        fontWeight: 600,
        whiteSpace: 'nowrap',
        overflow: 'hidden',
        textOverflow: 'ellipsis',
      }}
    >
      {mobile ? event.title : hhmm(event.startTime) + ' ' + event.title}
    </ButtonBase>
  );
}

function AbsenceChip({ absence, mobile }: { absence: Absence; mobile: boolean }) {
  return (
    <Box
      sx={{
        display: 'flex',
        alignItems: 'center',
        gap: '3px',
        background: NEUTRAL.line2,
        borderRadius: '5px',
        p: '1px 4px',
        fontSize: mobile ? '8px' : '9px',
        color: NEUTRAL.secondary,
        whiteSpace: 'nowrap',
        overflow: 'hidden',
        textOverflow: 'ellipsis',
      }}
    >
      <Box
        component="span"
        sx={{ width: '6px', height: '6px', borderRadius: '50%', background: absence.roleColor, flex: '0 0 auto' }}
      />
      {(absence.name || '').split(' ')[0]}
    </Box>
  );
}

interface CalendarDayCellProps {
  date: Date;
  inMonth: boolean;
  isToday: boolean;
  mobile: boolean;
  primary: string;
  primaryContainer: string;
  events: TeamEvent[];
  absences: Absence[];
  onOpenEvent: (id: string) => void;
}

function CalendarDayCell({
  date,
  inMonth,
  isToday,
  mobile,
  primary,
  primaryContainer,
  events,
  absences,
  onOpenEvent,
}: CalendarDayCellProps) {
  const eventLimit = mobile ? 2 : 3;
  const absenceLimit = mobile ? 1 : 2;
  const visibleEvents = events.slice(0, eventLimit);
  const visibleAbsences = absences.slice(0, absenceLimit);

  return (
    <Box
      sx={{
        minHeight: mobile ? '58px' : '76px',
        border: `1px solid ${NEUTRAL.line2}`,
        borderRadius: '9px',
        p: mobile ? '3px' : '5px',
        background: inMonth ? NEUTRAL.card : NEUTRAL.sidebar,
        opacity: inMonth ? 1 : 0.55,
        display: 'flex',
        flexDirection: 'column',
        gap: '2px',
        overflow: 'hidden',
      }}
    >
      <Box
        sx={{
          fontSize: mobile ? '11px' : '12px',
          fontWeight: isToday ? 800 : 500,
          color: isToday ? primary : NEUTRAL.onSurfaceVariant,
          display: 'flex',
          justifyContent: 'center',
          alignItems: 'center',
          width: '20px',
          height: '20px',
          borderRadius: '50%',
          background: isToday ? primaryContainer : 'transparent',
          alignSelf: 'flex-start',
        }}
      >
        {date.getDate()}
      </Box>
      {visibleEvents.map((e) => (
        <EventChip key={e.id} event={e} mobile={mobile} onOpen={() => onOpenEvent(e.id)} />
      ))}
      {events.length > eventLimit ? (
        <Box sx={{ fontSize: '9px', color: NEUTRAL.faint, pl: '3px' }}>{'+' + (events.length - eventLimit)}</Box>
      ) : null}
      {visibleAbsences.map((a, idx) => (
        <AbsenceChip key={'a' + idx} absence={a} mobile={mobile} />
      ))}
      {absences.length > absenceLimit ? (
        <Box sx={{ fontSize: '9px', color: NEUTRAL.faint, pl: '3px' }}>
          {'+' + (absences.length - absenceLimit) + ' ' + t('events.absentShort')}
        </Box>
      ) : null}
    </Box>
  );
}

export function EventCalendar() {
  const app = useApp();
  const { state } = app;
  const tk = buildTokens(state.primaryColor);
  const compact = useCompact();
  const mobile = compact;
  const { data: events } = useEventsQuery(app.api, state.activeTeamId);
  const { data: absences } = useAbsencesQuery(app.api, state.activeTeamId, state.calShowAbsences);

  const cur = state.calMonth || new Date(new Date().getFullYear(), new Date().getMonth(), 1);
  const year = cur.getFullYear();
  const month = cur.getMonth();
  const first = new Date(year, month, 1);
  const startDow = (first.getDay() + 6) % 7;
  const today = todayLocalDate();

  const evByDate = groupEventsByDate(events);
  const absByDate = groupAbsencesByDate(absences, state.calShowAbsences);

  const cells: React.ReactNode[] = [];
  const dtf = new Intl.DateTimeFormat(getIntlLocale(), { weekday: mobile ? 'narrow' : 'short' });
  // Build Mon–Sun headers using a known Monday (2024-01-01 was a Monday)
  const dows = Array.from({ length: 7 }, (_, i) => dtf.format(new Date(2024, 0, 1 + i)));
  dows.forEach((d) =>
    cells.push(
      <Box
        key={'h' + d}
        sx={{ textAlign: 'center', fontSize: '11px', fontWeight: 700, color: NEUTRAL.faint, p: '4px 0' }}
      >
        {d}
      </Box>,
    ),
  );

  for (let i = 0; i < 42; i++) {
    const d = new Date(year, month, 1 - startDow + i);
    const ds = formatDateOnly(d);
    cells.push(
      <CalendarDayCell
        key={'c' + i}
        date={d}
        inMonth={d.getMonth() === month}
        isToday={ds === today}
        mobile={mobile}
        primary={tk.primary}
        primaryContainer={tk.primaryContainer}
        events={evByDate[ds] || []}
        absences={absByDate[ds] || []}
        onOpenEvent={app.openEventDetail}
      />,
    );
  }

  const monthLabel = new Intl.DateTimeFormat(getIntlLocale(), { month: 'long', year: 'numeric' }).format(cur);
  const nav = (delta: number) => () => {
    app.setState({ calMonth: new Date(year, month + delta, 1) });
  };

  return (
    <Box sx={{ maxWidth: '820px' }}>
      <Box
        component="label"
        sx={{
          display: 'flex',
          alignItems: 'center',
          gap: '9px',
          mb: '12px',
          cursor: 'pointer',
          fontSize: '13px',
          color: NEUTRAL.onSurfaceVariant,
          fontWeight: 500,
        }}
      >
        <input
          type="checkbox"
          checked={state.calShowAbsences}
          onChange={() => app.toggleCalAbsences()}
          style={{ width: '18px', height: '18px', accentColor: tk.primary }}
        />
        <Sym name="beach_access" size={18} color={NEUTRAL.secondary} />
        {t('events.showAbsences')}
      </Box>
      <Card>
        <Box sx={{ display: 'flex', alignItems: 'center', gap: '10px', mb: '12px' }}>
          <Box sx={{ flex: 1, fontSize: '16px', fontWeight: 700, textTransform: 'capitalize' }}>{monthLabel}</Box>
          <ButtonBase
            onClick={nav(-1)}
            aria-label={t('events.calPrevMonth')}
            sx={{
              width: '34px',
              height: '34px',
              borderRadius: '50%',
              border: `1px solid ${NEUTRAL.line3}`,
              background: NEUTRAL.card,
              color: NEUTRAL.onSurfaceVariant,
            }}
          >
            <Sym name="chevron_left" size={22} color={NEUTRAL.onSurfaceVariant} />
          </ButtonBase>
          <ButtonBase
            onClick={nav(1)}
            aria-label={t('events.calNextMonth')}
            sx={{
              width: '34px',
              height: '34px',
              borderRadius: '50%',
              border: `1px solid ${NEUTRAL.line3}`,
              background: NEUTRAL.card,
              color: NEUTRAL.onSurfaceVariant,
            }}
          >
            <Sym name="chevron_right" size={22} color={NEUTRAL.onSurfaceVariant} />
          </ButtonBase>
        </Box>
        <Box
          data-testid="calendar-grid"
          sx={{ display: 'grid', gridTemplateColumns: 'repeat(7,1fr)', gap: mobile ? '4px' : '6px' }}
        >
          {cells}
        </Box>
      </Card>
    </Box>
  );
}
