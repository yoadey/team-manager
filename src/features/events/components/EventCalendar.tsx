import React from 'react';
import Box from '@mui/material/Box';
import ButtonBase from '@mui/material/ButtonBase';
import { useApp } from '../../../store/AppContext';
import { useCompact } from '../../../layouts/AppShell';
import { buildTokens, hhmm, typeMeta } from '../../../styles/tokens';
import { formatDateOnly, parseDateOnlyLocal, todayLocalDate } from '../../../utils/date';
import { Sym, Card } from '../../../components/ui';

export function EventCalendar() {
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
