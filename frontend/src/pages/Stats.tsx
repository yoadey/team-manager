import Box from '@mui/material/Box';
import ButtonBase from '@mui/material/ButtonBase';
import { useApp } from '@/context/AppContext';
import { buildTokens, fmtDate, NEUTRAL, todayStr, typeMeta } from '@/styles/tokens';
import { ALL_TIME_FROM_DATE, monthsAgoLocal } from '@/utils/date';
import { Av, Chip, EmptyState, SectionTitle, SpinnerBox, Sym, inputSx } from '@/components/ui';
import { t as tr } from '@/i18n';
import type { DateRange } from '@/types';

export function Stats() {
  const app = useApp();
  const { state } = app;
  const t = buildTokens(state.primaryColor);
  const st = state.stats;

  const today = todayStr();
  const ago = (months: number) => monthsAgoLocal(today, months);
  const R: DateRange = state.statsRange || { from: null, to: null };
  // 'all' passes an explicit far-past `from` rather than null: the service
  // layers treat a null/omitted `from` as "no range selected yet" and apply
  // their 3-month default (see ALL_TIME_FROM_DATE's doc comment), so an
  // unselected range and a genuine "show everything" request must be
  // distinguishable, not both collapse to the same null value.
  const presets: [string, string, DateRange | null][] = [
    ['all', tr('stats.presetAll'), { from: ALL_TIME_FROM_DATE, to: today }],
    ['3m', tr('stats.presetMonths', { n: 3 }), { from: ago(3), to: today }],
    ['6m', tr('stats.presetMonths', { n: 6 }), { from: ago(6), to: today }],
    ['12m', tr('stats.presetMonths', { n: 12 }), { from: ago(12), to: today }],
  ];
  // No explicit selection yet (R.from/to both null) means the service layers
  // are actually applying their 3-month default, so highlight '3 Monate'
  // rather than 'Gesamt' — matches what's genuinely being displayed.
  const activeKey =
    !R || (!R.from && !R.to)
      ? '3m'
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
              border: '1.5px solid ' + (sel ? t.primary : NEUTRAL.inputBorder),
              background: sel ? t.primaryContainer : NEUTRAL.card,
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
        background: NEUTRAL.card,
        border: `1px solid ${NEUTRAL.line}`,
        borderRadius: '20px',
        p: '20px',
        mb: '20px',
      }}
    >
      <Box
        role="img"
        aria-label={tr('stats.ringAria', { avg: st.avg })}
        sx={{
          width: '96px',
          height: '96px',
          borderRadius: '50%',
          flex: '0 0 auto',
          background: 'conic-gradient(' + t.primary + ' ' + st.avg * 3.6 + 'deg, ' + NEUTRAL.line2 + ' 0deg)',
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
            background: NEUTRAL.card,
            display: 'flex',
            flexDirection: 'column',
            alignItems: 'center',
            justifyContent: 'center',
          }}
        >
          <Box sx={{ fontSize: '22px', fontWeight: 800, color: t.primary }}>{st.avg + '%'}</Box>
          <Box sx={{ fontSize: '10px', color: NEUTRAL.faint }}>{tr('stats.avgQuote')}</Box>
        </Box>
      </Box>
      <Box sx={{ flex: 1 }}>
        <Box sx={{ fontSize: '16px', fontWeight: 700 }}>{tr('stats.teamAttendance')}</Box>
        <Box sx={{ fontSize: '13px', color: NEUTRAL.secondary, mt: '4px', lineHeight: 1.5 }}>
          {tr('stats.teamAttendanceDesc', { n: st.pastCount, count: st.pastCount })}
        </Box>
      </Box>
    </Box>
  );

  const memberBars = (
    <Box key="mb" sx={{ mb: '22px' }}>
      <SectionTitle>{tr('stats.quotePerPerson')}</SectionTitle>
      <Box sx={{ display: 'flex', flexDirection: 'column', gap: '10px', maxHeight: '60vh', overflowY: 'auto' }}>
        {st.members.map((m) => {
          const q = m.quote === null ? 0 : m.quote;
          const col =
            m.quote === null ? NEUTRAL.faint : q >= 80 ? NEUTRAL.success : q >= 50 ? NEUTRAL.warn : NEUTRAL.error;
          return (
            <Box
              key={m.userId}
              sx={{
                display: 'flex',
                alignItems: 'center',
                gap: '12px',
                background: NEUTRAL.card,
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
                  aria-label={tr('stats.memberAria', {
                    name: m.name,
                    value: m.quote === null ? tr('stats.noData') : tr('stats.attendancePct', { q }),
                  })}
                  sx={{ height: '8px', borderRadius: '5px', background: NEUTRAL.line2, overflow: 'hidden' }}
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
      <SectionTitle>{tr('stats.lineupPerEvent')}</SectionTitle>
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
                  background: NEUTRAL.card,
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
                    {fmtDate(e.date) + ' · ' + tr('stats.present', { yes: e.yes, nominated: e.nominated })}
                  </Box>
                </Box>
                <Chip
                  label={e.enough ? tr('stats.complete') : tr('stats.tooFew')}
                  color={e.enough ? NEUTRAL.success : NEUTRAL.error}
                  bg={e.enough ? NEUTRAL.successBg : NEUTRAL.errorBg}
                  icon={e.enough ? 'check_circle' : 'warning'}
                />
              </Box>
            );
          })
        ) : (
          <EmptyState icon="insights" text={tr('stats.emptyPast')} />
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
