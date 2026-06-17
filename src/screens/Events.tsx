import Box from '@mui/material/Box';
import ButtonBase from '@mui/material/ButtonBase';
import { useApp } from '../store/AppContext';
import { useCompact } from '../components/Shell';
import { buildTokens, fmtRange, hhmm, NEUTRAL, typeMeta } from '../theme/tokens';
import { formatDateOnly, parseDateOnlyLocal, todayLocalDate } from '../utils/date';
import { Av, Card, Chip, EmptyState, SectionTitle, Sym, SpinnerBox } from '../components/ui';
import { EventCard } from '../components/cards';

export function Events() {
  const app = useApp();
  const { state } = app;
  const t = buildTokens(state.primaryColor);
  const today = todayLocalDate();

  const seg = (label: string, val: string, cur: string, fn: (v: any) => void, flex?: string) => (
    <ButtonBase
      key={label}
      onClick={() => fn(val)}
      sx={{ flex: flex || '0 0 auto', p: '9px 16px', border: 'none', borderRadius: '10px', fontSize: '13px', fontWeight: 600, background: cur === val ? '#fff' : 'transparent', color: cur === val ? t.primary : '#5A5D66', boxShadow: cur === val ? '0 1px 3px rgba(0,0,0,.12)' : 'none' }}
    >
      {label}
    </ButtonBase>
  );

  const toolbar = (
    <Box sx={{ display: 'flex', gap: '10px', mb: '18px', flexWrap: 'wrap', alignItems: 'center' }}>
      <Box sx={{ display: 'flex', background: '#ECEDF3', borderRadius: '12px', p: '4px' }}>
        {seg('Liste', 'list', state.eventsView, (v) => app.setEventsView(v))}
        {seg('Kalender', 'calendar', state.eventsView, (v) => app.setEventsView(v))}
        {seg('Abwesend', 'absences', state.eventsView, (v) => app.setEventsView(v))}
      </Box>
      <Box sx={{ flex: 1 }} />
      {state.eventsView === 'list' ? (
        <Box sx={{ display: 'flex', background: '#ECEDF3', borderRadius: '12px', p: '4px' }}>
          {seg('Anstehend', 'upcoming', state.eventScope, (v) => app.setState({ eventScope: v }))}
          {seg('Archiv', 'past', state.eventScope, (v) => app.setState({ eventScope: v }))}
        </Box>
      ) : null}
      <ButtonBase
        onClick={() => app.openCalExport()}
        title="In Google / Apple / Android Kalender einbinden"
        sx={{ display: 'inline-flex', alignItems: 'center', gap: '7px', p: '9px 14px', borderRadius: '12px', border: '1px solid #D0D2DA', background: '#fff', fontSize: '13px', fontWeight: 600, color: '#44474E' }}
      >
        <Sym name="ios_share" size={18} color="#6A6D76" />
        Exportieren
      </ButtonBase>
    </Box>
  );

  if (state.eventsView === 'calendar') return <Box>{toolbar}<Calendar /></Box>;
  if (state.eventsView === 'absences') return <Box>{toolbar}<Absences /></Box>;

  const pendingFilter = state.eventsOnlyPending && state.eventScope === 'upcoming';
  let scoped = state.events.filter((e) => (state.eventScope === 'upcoming' ? e.date >= today : e.date < today));
  if (pendingFilter) scoped = scoped.filter((e) => e.myStatus === 'pending' && e.status !== 'cancelled');
  scoped = scoped.slice().sort((a, b) => (state.eventScope === 'upcoming' ? a.date.localeCompare(b.date) : b.date.localeCompare(a.date)));

  const filterChip = pendingFilter ? (
    <ButtonBase
      onClick={() => app.setState({ eventsOnlyPending: false })}
      sx={{ display: 'inline-flex', alignItems: 'center', gap: '8px', mb: '14px', p: '8px 12px 8px 14px', borderRadius: '999px', border: 'none', background: '#FFE5B8', color: '#8A6100', fontSize: '13px', fontWeight: 700 }}
    >
      <Sym name="pending_actions" size={17} color="#8A6100" />
      Nur offene Rückmeldungen
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
          text={pendingFilter ? 'Alle Rückmeldungen erledigt – nichts offen' : (state.eventScope === 'upcoming' ? 'Keine anstehenden Termine' : 'Kein Termin im Archiv')}
        />
      </Box>
    );
  }

  return (
    <Box sx={{ maxWidth: '820px' }}>
      {toolbar}
      {filterChip}
      <Box sx={{ display: 'flex', flexDirection: 'column', gap: '10px' }}>
        {scoped.map((e) => <EventCard key={e.id} e={e} />)}
      </Box>
    </Box>
  );
}

function Calendar() {
  const app = useApp();
  const { state } = app;
  const t = buildTokens(state.primaryColor);
  const compact = useCompact();
  const mobile = compact;

  const cur = state.calMonth || new Date(new Date().getFullYear(), new Date().getMonth(), 1);
  const year = cur.getFullYear();
  const month = cur.getMonth();
  const first = new Date(year, month, 1);
  const startDow = (first.getDay() + 6) % 7;
  const today = todayLocalDate();

  const evByDate: Record<string, typeof state.events> = {};
  state.events.forEach((e) => { (evByDate[e.date] = evByDate[e.date] || []).push(e); });

  const absByDate: Record<string, NonNullable<typeof state.absences>> = {};
  if (state.calShowAbsences && state.absences) {
    state.absences.forEach((a) => {
      let d = parseDateOnlyLocal(a.from);
      const end = parseDateOnlyLocal(a.to);
      while (d <= end) {
        const ds = formatDateOnly(d);
        (absByDate[ds] = absByDate[ds] || []).push(a);
        d = new Date(d.getTime() + 86400000);
      }
    });
  }

  const cells: React.ReactNode[] = [];
  const dows = ['Mo', 'Di', 'Mi', 'Do', 'Fr', 'Sa', 'So'];
  dows.forEach((d) => cells.push(
    <Box key={'h' + d} sx={{ textAlign: 'center', fontSize: '11px', fontWeight: 700, color: '#9A9DA6', p: '4px 0' }}>{mobile ? d[0] : d}</Box>
  ));

  for (let i = 0; i < 42; i++) {
    const d = new Date(year, month, 1 - startDow + i);
    const ds = formatDateOnly(d);
    const inMonth = d.getMonth() === month;
    const evs = evByDate[ds] || [];
    const abs = absByDate[ds] || [];
    const isToday = ds === today;
    const chips = evs.slice(0, mobile ? 2 : 3).map((e) => {
      const tm = typeMeta(e.type);
      return (
        <ButtonBase
          key={e.id}
          onClick={() => app.openEventDetail(e.id)}
          sx={{ display: 'block', width: '100%', textAlign: 'left', justifyContent: 'flex-start', border: 'none', background: tm.bg, color: tm.on, borderRadius: '5px', p: '2px 5px', fontSize: mobile ? '9px' : '10px', fontWeight: 600, whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis' }}
        >
          {mobile ? e.title : (hhmm(e.startTime) + ' ' + e.title)}
        </ButtonBase>
      );
    });
    const absChips = abs.slice(0, mobile ? 1 : 2).map((a, idx) => (
      <Box key={'a' + idx} sx={{ display: 'flex', alignItems: 'center', gap: '3px', background: '#F0F0F4', borderRadius: '5px', p: '1px 4px', fontSize: mobile ? '8px' : '9px', color: '#6A6D76', whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis' }}>
        <Box component="span" sx={{ width: '6px', height: '6px', borderRadius: '50%', background: a.roleColor, flex: '0 0 auto' }} />
        {(a.name || '').split(' ')[0]}
      </Box>
    ));
    cells.push(
      <Box key={'c' + i} sx={{ minHeight: mobile ? '58px' : '76px', border: '1px solid #ECEDF3', borderRadius: '9px', p: mobile ? '3px' : '5px', background: inMonth ? '#fff' : '#F7F7FB', opacity: inMonth ? 1 : 0.55, display: 'flex', flexDirection: 'column', gap: '2px', overflow: 'hidden' }}>
        <Box sx={{ fontSize: mobile ? '11px' : '12px', fontWeight: isToday ? 800 : 500, color: isToday ? t.primary : '#44474E', display: 'flex', justifyContent: 'center', alignItems: 'center', width: '20px', height: '20px', borderRadius: '50%', background: isToday ? t.primaryContainer : 'transparent', alignSelf: 'flex-start' }}>{d.getDate()}</Box>
        {chips}
        {evs.length > (mobile ? 2 : 3) ? <Box sx={{ fontSize: '9px', color: '#9A9DA6', pl: '3px' }}>{'+' + (evs.length - (mobile ? 2 : 3))}</Box> : null}
        {absChips}
        {abs.length > (mobile ? 1 : 2) ? <Box sx={{ fontSize: '9px', color: '#9A9DA6', pl: '3px' }}>{'+' + (abs.length - (mobile ? 1 : 2)) + ' abw.'}</Box> : null}
      </Box>
    );
  }

  const monthLabel = new Intl.DateTimeFormat('de-DE', { month: 'long', year: 'numeric' }).format(cur);
  const nav = (delta: number) => () => { app.setState({ calMonth: new Date(year, month + delta, 1) }); };

  return (
    <Box sx={{ maxWidth: '820px' }}>
      <Box component="label" sx={{ display: 'flex', alignItems: 'center', gap: '9px', mb: '12px', cursor: 'pointer', fontSize: '13px', color: '#44474E', fontWeight: 500 }}>
        <input type="checkbox" checked={state.calShowAbsences} onChange={() => app.toggleCalAbsences()} style={{ width: '18px', height: '18px', accentColor: t.primary }} />
        <Sym name="beach_access" size={18} color="#6A6D76" />
        Geplante Abwesenheiten anzeigen
      </Box>
      <Card>
        <Box sx={{ display: 'flex', alignItems: 'center', gap: '10px', mb: '12px' }}>
          <Box sx={{ flex: 1, fontSize: '16px', fontWeight: 700, textTransform: 'capitalize' }}>{monthLabel}</Box>
          <ButtonBase onClick={nav(-1)} sx={{ width: '34px', height: '34px', borderRadius: '50%', border: '1px solid #E0E2EA', background: '#fff', color: '#44474E' }}>
            <Sym name="chevron_left" size={22} color="#44474E" />
          </ButtonBase>
          <ButtonBase onClick={nav(1)} sx={{ width: '34px', height: '34px', borderRadius: '50%', border: '1px solid #E0E2EA', background: '#fff', color: '#44474E' }}>
            <Sym name="chevron_right" size={22} color="#44474E" />
          </ButtonBase>
        </Box>
        <Box sx={{ display: 'grid', gridTemplateColumns: 'repeat(7,1fr)', gap: mobile ? '4px' : '6px' }}>{cells}</Box>
      </Card>
    </Box>
  );
}

function Absences() {
  const app = useApp();
  const { state } = app;
  const t = buildTokens(state.primaryColor);

  if (!state.absences) return <SpinnerBox />;

  const today = todayLocalDate();
  const list = state.absences.filter((a) => a.to >= today);

  const rows = list.map((a) => {
    const isMe = a.userId === state.user!.id;
    return (
      <Box key={a.id} sx={{ display: 'flex', alignItems: 'center', gap: '12px', background: '#fff', border: `1px solid ${NEUTRAL.line}`, borderRadius: '15px', p: '12px 14px' }}>
        <Av name={a.name} photo={a.photo} color={a.avatarColor} size={40} />
        <Box sx={{ flex: 1, minWidth: 0 }}>
          <Box sx={{ fontSize: '14px', fontWeight: 600, display: 'flex', alignItems: 'center', gap: '7px' }}>
            {a.name}
            {isMe ? <Chip label="Du" color={t.primary} bg={t.primaryContainer} /> : null}
          </Box>
          <Box sx={{ fontSize: '12px', color: '#6A6D76', mt: '2px' }}>{fmtRange(a.from, a.to) + ' · ' + a.reason}</Box>
        </Box>
        <Box component="span" sx={{ width: '10px', height: '10px', borderRadius: '50%', background: a.roleColor, flex: '0 0 auto' }} />
        {isMe ? (
          <ButtonBase onClick={() => app.openAbsenceForm(a)} sx={{ width: '34px', height: '34px', borderRadius: '50%', border: 'none', background: '#F4F4FA', color: '#44474E', flex: '0 0 auto' }}>
            <Sym name="edit" size={18} color="#44474E" />
          </ButtonBase>
        ) : null}
        {isMe ? (
          <ButtonBase onClick={() => app.removeAbsence(a.id)} sx={{ width: '34px', height: '34px', borderRadius: '50%', border: 'none', background: '#FFF4F3', color: '#BA1A1A', flex: '0 0 auto' }}>
            <Sym name="delete" size={19} color="#BA1A1A" />
          </ButtonBase>
        ) : null}
      </Box>
    );
  });

  return (
    <Box sx={{ maxWidth: '720px' }}>
      <ButtonBase
        onClick={() => app.openAbsenceForm()}
        sx={{ display: 'flex', alignItems: 'center', justifyContent: 'center', gap: '9px', width: '100%', p: '13px', borderRadius: '14px', border: '1.5px dashed #C8CAD2', background: 'transparent', color: t.primary, fontWeight: 600, fontSize: '14px', mb: '18px' }}
      >
        <Sym name="event_busy" size={20} color={t.primary} />
        Eigene Abwesenheit eintragen
      </ButtonBase>
      <SectionTitle>Geplante Abwesenheiten</SectionTitle>
      <Box sx={{ display: 'flex', flexDirection: 'column', gap: '9px' }}>
        {list.length ? rows : <EmptyState icon="beach_access" text="Keine geplanten Abwesenheiten" />}
      </Box>
    </Box>
  );
}
