import Box from '@mui/material/Box';
import ButtonBase from '@mui/material/ButtonBase';
import { useApp } from '@/context/AppContext';
import { buildTokens, fmtDate, NEUTRAL, todayStr, typeMeta } from '@/styles/tokens';
import { formatDateOnly, parseDateOnlyLocal } from '@/utils/date';
import { Av, Chip, EmptyState, SectionTitle, SpinnerBox, Sym, inputSx } from '@/components/ui';
import type { DateRange } from '@/types';

export function Stats() {
  const app = useApp();
  const { state } = app;
  const t = buildTokens(state.primaryColor);
  const st = state.stats;

  const today = todayStr();
  const d = parseDateOnlyLocal(today);
  const ago = (months: number) => {
    const x = new Date(d);
    x.setMonth(x.getMonth() - months);
    return formatDateOnly(x);
  };
  const R: DateRange = state.statsRange || { from: null, to: null };
  const presets: [string, string, DateRange | null][] = [
    ['all', 'Gesamt', null],
    ['3m', '3 Monate', { from: ago(3), to: today }],
    ['6m', '6 Monate', { from: ago(6), to: today }],
    ['12m', '12 Monate', { from: ago(12), to: today }],
  ];
  const activeKey =
    !R || (!R.from && !R.to)
      ? 'all'
      : (presets.find((p) => p[2] && p[2].from === R.from && p[2].to === R.to) || ['custom'])[0];

  const dateInput: React.CSSProperties = { ...inputSx, padding: '7px 9px', fontSize: '12px', width: 'auto' };

  const filterBar = (
    <Box key="flt" sx={{ display: 'flex', flexWrap: 'wrap', gap: '8px', alignItems: 'center', mb: '16px' }}>
      {presets.map(([k, l, rng]) => {
        const sel = activeKey === k;
        return (
          <ButtonBase
            key={k}
            onClick={() => app.setStatsRange(rng)}
            sx={{
              p: '8px 14px',
              borderRadius: '999px',
              fontSize: '13px',
              fontWeight: 600,
              cursor: 'pointer',
              border: '1.5px solid ' + (sel ? t.primary : '#D0D2DA'),
              background: sel ? t.primaryContainer : '#fff',
              color: sel ? t.onPrimaryContainer : NEUTRAL.onSurfaceVariant,
            }}
          >
            {l}
          </ButtonBase>
        );
      })}
      <Box key="cust" sx={{ display: 'flex', alignItems: 'center', gap: '6px', ml: 'auto' }}>
        <input
          key="f"
          type="date"
          value={R.from || ''}
          max={R.to || today}
          onChange={(e) => app.setStatsRange({ ...R, from: e.target.value || null })}
          style={dateInput}
        />
        <Box component="span" sx={{ color: NEUTRAL.faint, fontSize: '13px' }}>
          –
        </Box>
        <input
          key="t"
          type="date"
          value={R.to || ''}
          min={R.from || ''}
          onChange={(e) => app.setStatsRange({ ...R, to: e.target.value || null })}
          style={dateInput}
        />
      </Box>
    </Box>
  );

  if (!st)
    return (
      <Box sx={{ maxWidth: '760px' }}>
        {filterBar}
        <SpinnerBox />
      </Box>
    );

  const ring = (
    <Box
      key="ring"
      sx={{
        display: 'flex',
        alignItems: 'center',
        gap: '20px',
        background: '#fff',
        border: `1px solid ${NEUTRAL.line}`,
        borderRadius: '20px',
        p: '20px',
        mb: '20px',
      }}
    >
      <Box
        role="img"
        aria-label={`Durchschnittliche Anwesenheitsquote: ${st.avg}%`}
        sx={{
          width: '96px',
          height: '96px',
          borderRadius: '50%',
          flex: '0 0 auto',
          background: 'conic-gradient(' + t.primary + ' ' + st.avg * 3.6 + 'deg, #ECEDF3 0deg)',
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
        }}
      >
        <Box
          sx={{
            width: '72px',
            height: '72px',
            borderRadius: '50%',
            background: '#fff',
            display: 'flex',
            flexDirection: 'column',
            alignItems: 'center',
            justifyContent: 'center',
          }}
        >
          <Box sx={{ fontSize: '22px', fontWeight: 800, color: t.primary }}>{st.avg + '%'}</Box>
          <Box sx={{ fontSize: '10px', color: NEUTRAL.faint }}>Ø Quote</Box>
        </Box>
      </Box>
      <Box sx={{ flex: 1 }}>
        <Box sx={{ fontSize: '16px', fontWeight: 700 }}>Team-Anwesenheit</Box>
        <Box sx={{ fontSize: '13px', color: NEUTRAL.secondary, mt: '4px', lineHeight: 1.5 }}>
          {'Durchschnittliche Zusagequote über ' +
            st.pastCount +
            ' vergangene Termine. Nicht nominierte Termine zählen nicht.'}
        </Box>
      </Box>
    </Box>
  );

  const memberBars = (
    <Box key="mb" sx={{ mb: '22px' }}>
      <SectionTitle>Quote pro Person</SectionTitle>
      <Box sx={{ display: 'flex', flexDirection: 'column', gap: '10px', maxHeight: '60vh', overflowY: 'auto' }}>
        {st.members.map((m) => {
          const q = m.quote === null ? 0 : m.quote;
          const col = m.quote === null ? '#C0C2CA' : q >= 80 ? NEUTRAL.success : q >= 50 ? '#9A5B00' : NEUTRAL.error;
          return (
            <Box
              key={m.userId}
              sx={{
                display: 'flex',
                alignItems: 'center',
                gap: '12px',
                background: '#fff',
                border: `1px solid ${NEUTRAL.line}`,
                borderRadius: '14px',
                p: '11px 14px',
              }}
            >
              <Av name={m.name} photo={m.photo} color={m.avatarColor} size={36} />
              <Box sx={{ flex: 1, minWidth: 0 }}>
                <Box sx={{ fontSize: '14px', fontWeight: 600, mb: '5px' }}>{m.name}</Box>
                <Box
                  role="progressbar"
                  aria-valuenow={q}
                  aria-valuemin={0}
                  aria-valuemax={100}
                  aria-label={`${m.name}: ${m.quote === null ? 'keine Daten' : q + '% Anwesenheit'}`}
                  sx={{ height: '8px', borderRadius: '5px', background: '#ECEDF3', overflow: 'hidden' }}
                >
                  <Box sx={{ height: '100%', width: q + '%', background: col, borderRadius: '5px' }} />
                </Box>
              </Box>
              <Box sx={{ fontSize: '15px', fontWeight: 800, color: col, width: '52px', textAlign: 'right' }}>
                {m.quote === null ? '–' : q + '%'}
              </Box>
            </Box>
          );
        })}
      </Box>
    </Box>
  );

  const eventsSec = (
    <Box key="ev">
      <SectionTitle>Aufstellung je Termin</SectionTitle>
      <Box sx={{ display: 'flex', flexDirection: 'column', gap: '8px' }}>
        {st.events.length ? (
          st.events.map((e) => {
            const tm = typeMeta(e.type);
            return (
              <Box
                key={e.id}
                sx={{
                  display: 'flex',
                  alignItems: 'center',
                  gap: '12px',
                  background: '#fff',
                  border: `1px solid ${NEUTRAL.line}`,
                  borderRadius: '14px',
                  p: '11px 14px',
                }}
              >
                <Sym name={tm.icon} size={20} color={tm.color} />
                <Box sx={{ flex: 1, minWidth: 0 }}>
                  <Box
                    sx={{
                      fontSize: '14px',
                      fontWeight: 600,
                      whiteSpace: 'nowrap',
                      overflow: 'hidden',
                      textOverflow: 'ellipsis',
                    }}
                  >
                    {e.title}
                  </Box>
                  <Box sx={{ fontSize: '12px', color: NEUTRAL.faint }}>
                    {fmtDate(e.date) + ' · ' + e.yes + '/' + e.nominated + ' anwesend'}
                  </Box>
                </Box>
                <Chip
                  label={e.enough ? 'Vollständig' : 'Zu wenig'}
                  color={e.enough ? NEUTRAL.success : NEUTRAL.error}
                  bg={e.enough ? NEUTRAL.successBg : NEUTRAL.errorBg}
                  icon={e.enough ? 'check_circle' : 'warning'}
                />
              </Box>
            );
          })
        ) : (
          <EmptyState icon="insights" text="Noch keine vergangenen Termine" />
        )}
      </Box>
    </Box>
  );

  return (
    <Box sx={{ maxWidth: '760px' }}>
      {filterBar}
      {ring}
      {memberBars}
      {eventsSec}
    </Box>
  );
}
