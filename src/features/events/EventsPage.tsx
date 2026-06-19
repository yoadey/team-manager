import { useMemo } from 'react';
import Box from '@mui/material/Box';
import ButtonBase from '@mui/material/ButtonBase';
import { useApp } from '@/context/AppContext';
import { buildTokens } from '@/styles/tokens';
import { todayLocalDate } from '@/utils/date';
import { Sym, EmptyState } from '@/components/ui';
import { EventCard } from '@/components/cards';
import { EventCalendar, EventAbsences } from '@/features/events';
import { t } from '@/i18n';

export function EventsPage() {
  const app = useApp();
  const { state } = app;
  const tk = buildTokens(state.primaryColor);
  const today = todayLocalDate();

  const scoped = useMemo(() => {
    let list = state.events.filter((e) => (state.eventScope === 'upcoming' ? e.date >= today : e.date < today));
    if (state.eventsOnlyPending && state.eventScope === 'upcoming')
      list = list.filter((e) => e.myStatus === 'pending' && e.status !== 'cancelled');
    return list
      .slice()
      .sort((a, b) => (state.eventScope === 'upcoming' ? a.date.localeCompare(b.date) : b.date.localeCompare(a.date)));
  }, [state.events, state.eventScope, state.eventsOnlyPending, today]);

  const seg = <T extends string>(label: string, val: T, cur: string, fn: (v: T) => void, flex?: string) => (
    <ButtonBase
      key={label}
      onClick={() => fn(val)}
      sx={{
        flex: flex || '0 0 auto',
        p: '9px 16px',
        border: 'none',
        borderRadius: '10px',
        fontSize: '13px',
        fontWeight: 600,
        background: cur === val ? '#fff' : 'transparent',
        color: cur === val ? tk.primary : '#5A5D66',
        boxShadow: cur === val ? '0 1px 3px rgba(0,0,0,.12)' : 'none',
      }}
    >
      {label}
    </ButtonBase>
  );

  const toolbar = (
    <Box sx={{ display: 'flex', gap: '10px', mb: '18px', flexWrap: 'wrap', alignItems: 'center' }}>
      <Box sx={{ display: 'flex', background: '#ECEDF3', borderRadius: '12px', p: '4px' }}>
        {seg(t('events.tabs.list'), 'list', state.eventsView, (v) => app.setEventsView(v))}
        {seg(t('events.tabs.calendar'), 'calendar', state.eventsView, (v) => app.setEventsView(v))}
        {seg(t('events.tabs.absences'), 'absences', state.eventsView, (v) => app.setEventsView(v))}
      </Box>
      <Box sx={{ flex: 1 }} />
      {state.eventsView === 'list' ? (
        <Box sx={{ display: 'flex', background: '#ECEDF3', borderRadius: '12px', p: '4px' }}>
          {seg(t('events.tabs.upcoming'), 'upcoming', state.eventScope, (v) => app.setState({ eventScope: v }))}
          {seg(t('events.tabs.past'), 'past', state.eventScope, (v) => app.setState({ eventScope: v }))}
        </Box>
      ) : null}
      <ButtonBase
        onClick={() => app.openCalExport()}
        title={t('events.exportTitle')}
        sx={{
          display: 'inline-flex',
          alignItems: 'center',
          gap: '7px',
          p: '9px 14px',
          borderRadius: '12px',
          border: '1px solid #D0D2DA',
          background: '#fff',
          fontSize: '13px',
          fontWeight: 600,
          color: '#44474E',
        }}
      >
        <Sym name="ios_share" size={18} color="#6A6D76" />
        {t('events.export')}
      </ButtonBase>
    </Box>
  );

  if (state.eventsView === 'calendar')
    return (
      <Box>
        {toolbar}
        <EventCalendar />
      </Box>
    );
  if (state.eventsView === 'absences')
    return (
      <Box>
        {toolbar}
        <EventAbsences />
      </Box>
    );

  const pendingFilter = state.eventsOnlyPending && state.eventScope === 'upcoming';

  const filterChip = pendingFilter ? (
    <ButtonBase
      onClick={() => app.setState({ eventsOnlyPending: false })}
      sx={{
        display: 'inline-flex',
        alignItems: 'center',
        gap: '8px',
        mb: '14px',
        p: '8px 12px 8px 14px',
        borderRadius: '999px',
        border: 'none',
        background: '#FFE5B8',
        color: '#8A6100',
        fontSize: '13px',
        fontWeight: 700,
      }}
    >
      <Sym name="pending_actions" size={17} color="#8A6100" />
      {t('events.filterPending')}
      <Sym name="close" size={17} color="#8A6100" />
    </ButtonBase>
  ) : null;

  if (!scoped.length) {
    return (
      <Box>
        {toolbar}
        {filterChip}
        <EmptyState
          icon={pendingFilter ? 'task_alt' : 'event_busy'}
          text={
            pendingFilter
              ? t('events.emptyPending')
              : state.eventScope === 'upcoming'
                ? t('events.emptyUpcoming')
                : t('events.emptyPast')
          }
        />
      </Box>
    );
  }

  return (
    <Box sx={{ maxWidth: '820px' }}>
      {toolbar}
      {filterChip}
      <Box sx={{ display: 'flex', flexDirection: 'column', gap: '10px' }}>
        {scoped.map((e) => (
          <EventCard key={e.id} e={e} />
        ))}
      </Box>
    </Box>
  );
}
