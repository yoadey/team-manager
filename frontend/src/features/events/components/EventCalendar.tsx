import React from 'react';
import Box from '@mui/material/Box';
import ButtonBase from '@mui/material/ButtonBase';
import { useApp } from '@/context/AppContext';
import { useCompact } from '@/layouts/useCompact';
import { buildTokens, fmtDateLong, hhmm, typeMeta, NEUTRAL } from '@/styles/tokens';
import { formatDateOnly, parseDateOnlyLocal, todayLocalDate } from '@/utils/date';
import { getIntlLocale, t } from '@/i18n';
import { Sym, Card } from '@/components/ui';
import { useMembersQuery } from '@/features/members';
import { useEventsQuery } from '../hooks/useEventQueries';
import { useAbsencesQuery } from '../hooks/useAbsenceQueries';
import type { Absence, TeamEvent } from '../types';
import { synthesizeBirthdayEvents, groupBirthdaysByDate, type BirthdayEntry } from './synthesizeBirthdayEvents';

// Fixed color pair for the birthday chip, deliberately not one of typeMeta's
// event-type colors (nor NEUTRAL) so a birthday pseudo-event reads as
// visually distinct from both real events and absences at a glance.
const BIRTHDAY_COLOR = { bg: '#FCE4EC', on: '#7A1750' };

/** One calendar day's occurrence of an event, plus its position within a multi-day span (both 1 for a single-day event). */
interface EventOccurrence {
  event: TeamEvent;
  dayIndex: number;
  totalDays: number;
}

// Expands each event across every calendar day from `date` through
// `multiDayEndDate` inclusive (single-day events being the degenerate
// one-day case) -- mirrors groupAbsencesByDate's identical from/to
// expansion below, including its DST-safe local-day increment.
function groupEventsByDate(events: TeamEvent[] | undefined): Record<string, EventOccurrence[]> {
  const byDate: Record<string, EventOccurrence[]> = {};
  (events ?? []).forEach((e) => {
    if (!e.multiDayEndDate) {
      (byDate[e.date] = byDate[e.date] || []).push({ event: e, dayIndex: 1, totalDays: 1 });
      return;
    }
    // Collect every spanned day first (DST-safe local-day increment) so
    // totalDays reflects the actual number of calendar days walked, not a
    // fixed-ms-duration division that DST would throw off.
    const spanDates: string[] = [];
    let d = parseDateOnlyLocal(e.date);
    const end = parseDateOnlyLocal(e.multiDayEndDate);
    while (d <= end) {
      spanDates.push(formatDateOnly(d));
      d = new Date(d.getFullYear(), d.getMonth(), d.getDate() + 1);
    }
    spanDates.forEach((ds, idx) => {
      (byDate[ds] = byDate[ds] || []).push({ event: e, dayIndex: idx + 1, totalDays: spanDates.length });
    });
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

function EventChip({ occurrence, mobile, onOpen }: { occurrence: EventOccurrence; mobile: boolean; onOpen: () => void }) {
  const { event, dayIndex, totalDays } = occurrence;
  const tm = typeMeta(event.type);
  const isMultiDay = totalDays > 1;
  const label = isMultiDay
    ? event.title + ' ' + t('events.multiDayChipIndicator', { day: dayIndex, total: totalDays })
    : mobile
      ? event.title
      : hhmm(event.startTime) + ' ' + event.title;
  return (
    <ButtonBase
      onClick={onOpen}
      onDoubleClick={(e) => e.stopPropagation()}
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
      {label}
    </ButtonBase>
  );
}

function AbsenceChip({ absence, mobile }: { absence: Absence; mobile: boolean }) {
  return (
    <Box
      onDoubleClick={(e) => e.stopPropagation()}
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

// Synthetic pseudo-event -- styled like AbsenceChip's pill shape (not
// EventChip's block button, since it isn't clickable/openable) but with a
// dedicated icon + color pair so it can't be mistaken for a real event or an
// absence.
function BirthdayChip({ entry, mobile }: { entry: BirthdayEntry; mobile: boolean }) {
  return (
    <Box
      onDoubleClick={(e) => e.stopPropagation()}
      sx={{
        display: 'flex',
        alignItems: 'center',
        gap: '3px',
        background: BIRTHDAY_COLOR.bg,
        borderRadius: '5px',
        p: '1px 4px',
        fontSize: mobile ? '8px' : '9px',
        fontWeight: 600,
        color: BIRTHDAY_COLOR.on,
        whiteSpace: 'nowrap',
        overflow: 'hidden',
        textOverflow: 'ellipsis',
      }}
    >
      <Sym name="cake" size={mobile ? 9 : 10} color={BIRTHDAY_COLOR.on} />
      {entry.name}
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
  events: EventOccurrence[];
  absences: Absence[];
  birthdays: BirthdayEntry[];
  onOpenEvent: (id: string) => void;
  // Explicit `| undefined` (exactOptionalPropertyTypes) -- the caller passes
  // `undefined` for out-of-month/no-permission cells to mean "not creatable
  // here", distinct from simply omitting the prop.
  onCreateEvent?: (() => void) | undefined;
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
  birthdays,
  onOpenEvent,
  onCreateEvent,
}: CalendarDayCellProps) {
  const eventLimit = mobile ? 2 : 3;
  const absenceLimit = mobile ? 1 : 2;
  const visibleEvents = events.slice(0, eventLimit);
  const visibleAbsences = absences.slice(0, absenceLimit);
  // Double-click is the primary gesture, but a div with an onDoubleClick has
  // no keyboard equivalent by default -- expose the same action as a
  // focusable "button" (Enter/Space) so the day isn't a mouse-only affordance.
  const createLabel = onCreateEvent ? t('events.calCreateEventOnDay', { date: fmtDateLong(formatDateOnly(date)) }) : undefined;
  const onKeyDown = onCreateEvent
    ? (e: React.KeyboardEvent<HTMLDivElement>) => {
        if (e.key === 'Enter' || e.key === ' ') {
          e.preventDefault();
          onCreateEvent();
        }
      }
    : undefined;

  return (
    <Box
      onDoubleClick={onCreateEvent}
      onKeyDown={onKeyDown}
      role={onCreateEvent ? 'button' : undefined}
      tabIndex={onCreateEvent ? 0 : undefined}
      aria-label={createLabel}
      sx={{
        minHeight: mobile ? '58px' : '76px',
        border: `1px solid ${NEUTRAL.line2}`,
        borderRadius: mobile ? 0 : '9px',
        p: mobile ? '3px' : '5px',
        background: inMonth ? NEUTRAL.card : NEUTRAL.sidebar,
        opacity: inMonth ? 1 : 0.55,
        display: 'flex',
        flexDirection: 'column',
        gap: '2px',
        overflow: 'hidden',
        ...(onCreateEvent
          ? {
              cursor: 'pointer',
              userSelect: 'none',
              '&:hover': { background: NEUTRAL.sidebar },
              '&:focus-visible': { outline: `2px solid ${primary}`, outlineOffset: '-2px' },
            }
          : {}),
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
      {visibleEvents.map((occ) => (
        <EventChip
          key={occ.event.id + '-' + occ.dayIndex}
          occurrence={occ}
          mobile={mobile}
          onOpen={() => onOpenEvent(occ.event.id)}
        />
      ))}
      {events.length > eventLimit ? (
        <Box sx={{ fontSize: '9px', color: NEUTRAL.faint, pl: '3px' }}>{'+' + (events.length - eventLimit)}</Box>
      ) : null}
      {birthdays.map((b) => (
        <BirthdayChip key={'b' + b.membershipId} entry={b} mobile={mobile} />
      ))}
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
  // Birthdays are gated by the same rule as the member profile's birthday
  // field (members:write -- the "Trainerteam"/coaching-team roles; see
  // members.contactNote). A member without that permission gets an empty
  // birthday list rather than a filtered one, so no birthday entry ever
  // reaches the DOM for them.
  const canSeeBirthdays = app.can('members', 'write');
  const { data: members } = useMembersQuery(app.api, canSeeBirthdays ? state.activeTeamId : null);
  const canCreateEvent = app.can('events', 'write');

  const cur = state.calMonth || new Date(new Date().getFullYear(), new Date().getMonth(), 1);
  const year = cur.getFullYear();
  const month = cur.getMonth();
  const first = new Date(year, month, 1);
  const startDow = (first.getDay() + 6) % 7;
  const today = todayLocalDate();

  const evByDate = groupEventsByDate(events);
  const absByDate = groupAbsencesByDate(absences, state.calShowAbsences);
  // Grid range: the 42 rendered cells run from `1 - startDow` to `1 -
  // startDow + 41` days of the visible month (see the cell loop below), so
  // the synthesis range must match exactly or a birthday landing in a
  // leading/trailing cell from an adjacent month (possibly a different
  // calendar year, e.g. a December grid's early-January trailing cells)
  // would be silently dropped.
  const gridStart = new Date(year, month, 1 - startDow);
  const gridEnd = new Date(year, month, 1 - startDow + 41);
  const birthdaysByDate = canSeeBirthdays
    ? groupBirthdaysByDate(synthesizeBirthdayEvents(members, gridStart, gridEnd))
    : {};

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
    const inMonth = d.getMonth() === month;
    cells.push(
      <CalendarDayCell
        key={'c' + i}
        date={d}
        inMonth={inMonth}
        isToday={ds === today}
        mobile={mobile}
        primary={tk.primary}
        primaryContainer={tk.primaryContainer}
        events={evByDate[ds] || []}
        absences={absByDate[ds] || []}
        birthdays={birthdaysByDate[ds] || []}
        onOpenEvent={app.openEventDetail}
        onCreateEvent={inMonth && canCreateEvent ? () => app.openEventForm(null, ds) : undefined}
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
          sx={{ display: 'grid', gridTemplateColumns: 'repeat(7,1fr)', gap: mobile ? 0 : '6px' }}
        >
          {cells}
        </Box>
      </Card>
    </Box>
  );
}
