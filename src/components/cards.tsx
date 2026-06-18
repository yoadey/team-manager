import { memo } from 'react';
import Box from '@mui/material/Box';
import ButtonBase from '@mui/material/ButtonBase';
import { useAppActions } from '@/context/AppContext';
import { useCompact } from '@/layouts/AppShell';
import { buildTokens, fmtDate, hhmm, NEUTRAL, statusMeta, typeMeta } from '@/styles/tokens';
import { parseDateOnlyLocal, todayLocalDate } from '@/utils/date';
import { getIntlLocale } from '@/i18n';
import type { TeamEvent } from '@/features/events';
import type { NewsItem } from '@/features/news';
import { Av, Chip, Sym, metaItem } from './ui';

/** Event list card (used on Home and Events). Mirrors prototype eventCard().
 *  Memoised + actions-only so it skips re-renders when unrelated state changes
 *  (e.g. typing in a form, incoming notifications) — only `e`/layout changes
 *  re-render a row. */
export const EventCard = memo(function EventCard({ e }: { e: TeamEvent }) {
  const { openEventDetail } = useAppActions();
  const compact = useCompact();
  const today = todayLocalDate();
  const tm = typeMeta(e.type);
  const sm = statusMeta(e.myStatus);
  const isPast = e.date < today;
  const cancelled = e.status === 'cancelled';
  const day = parseDateOnlyLocal(e.date);

  return (
    <ButtonBase
      onClick={() => openEventDetail(e.id)}
      sx={{
        display: 'flex',
        gap: '13px',
        width: '100%',
        textAlign: 'left',
        background: '#fff',
        border: `1px solid ${NEUTRAL.line}`,
        borderRadius: '18px',
        p: '13px 15px',
        alignItems: 'stretch',
        opacity: cancelled ? 0.62 : isPast ? 0.92 : 1,
        justifyContent: 'flex-start',
      }}
    >
      <Box
        sx={{
          flex: '0 0 50px',
          display: 'flex',
          flexDirection: 'column',
          alignItems: 'center',
          justifyContent: 'center',
          background: tm.bg,
          color: tm.on,
          borderRadius: '12px',
          p: '8px 0',
        }}
      >
        <Box sx={{ fontSize: '20px', fontWeight: 800, lineHeight: 1 }}>{day.getDate()}</Box>
        <Box sx={{ fontSize: '10px', fontWeight: 700, textTransform: 'uppercase' }}>
          {new Intl.DateTimeFormat(getIntlLocale(), { month: 'short' }).format(day).replace('.', '')}
        </Box>
      </Box>
      <Box
        sx={{ flex: 1, minWidth: 0, display: 'flex', flexDirection: 'column', gap: '5px', justifyContent: 'center' }}
      >
        <Box sx={{ display: 'flex', alignItems: 'center', gap: '7px' }}>
          <Chip label={tm.label} color={tm.color} bg={tm.bg} icon={tm.icon} />
          {e.recurring ? <Sym name="repeat" size={15} color={NEUTRAL.faint} /> : null}
          {cancelled ? <Chip label="Abgesagt" color="#BA1A1A" bg="#FFDAD6" icon="event_busy" /> : null}
        </Box>
        <Box
          sx={{
            fontSize: '15px',
            fontWeight: 600,
            whiteSpace: 'nowrap',
            overflow: 'hidden',
            textOverflow: 'ellipsis',
            textDecoration: cancelled ? 'line-through' : 'none',
          }}
        >
          {e.title}
        </Box>
        <Box
          sx={{
            display: 'flex',
            alignItems: 'center',
            gap: '12px',
            fontSize: '12px',
            color: NEUTRAL.secondary,
            flexWrap: 'wrap',
          }}
        >
          {metaItem('schedule', hhmm(e.startTime) + '–' + hhmm(e.endTime), 'time')}
          {e.meetTime ? metaItem('login', 'Treff ' + hhmm(e.meetTime), 'meet') : null}
          {e.location && !compact ? metaItem('place', e.location, 'loc') : null}
        </Box>
      </Box>
      <Box
        sx={{
          display: 'flex',
          flexDirection: 'column',
          alignItems: 'flex-end',
          justifyContent: 'space-between',
          gap: '6px',
        }}
      >
        {isPast || cancelled ? null : <Chip label={sm.label} color={sm.color} bg={sm.bg} icon={sm.icon} />}
        <Box sx={{ display: 'flex', alignItems: 'center', gap: '7px', fontSize: '12px', fontWeight: 600 }}>
          <Box component="span" aria-label={`${e.summary.yes} zugesagt`} sx={{ color: '#2E7D32' }}>
            {e.summary.yes}✓
          </Box>
          <Box component="span" aria-label={`${e.summary.no} abgesagt`} sx={{ color: '#BA1A1A' }}>
            {e.summary.no}✕
          </Box>
          {e.summary.maybe ? (
            <Box component="span" aria-label={`${e.summary.maybe} vielleicht`} sx={{ color: '#9A5B00' }}>
              {e.summary.maybe}?
            </Box>
          ) : null}
        </Box>
      </Box>
    </ButtonBase>
  );
});

/** News card. compact=true clamps body to 2 lines. Mirrors prototype newsCard().
 *  Memoised + state-free so it skips re-renders when unrelated state changes. */
export const NewsCard = memo(function NewsCard({
  n,
  compact = false,
  primaryColor,
}: {
  n: NewsItem;
  compact?: boolean;
  primaryColor: string;
}) {
  const t = buildTokens(primaryColor);
  return (
    <Box sx={{ background: '#fff', border: `1px solid ${NEUTRAL.line}`, borderRadius: '16px', p: '15px 16px' }}>
      <Box sx={{ display: 'flex', alignItems: 'center', gap: '9px', mb: '7px' }}>
        <Av name={n.authorName} photo={n.authorPhoto} color={n.authorColor} size={28} font={11} />
        <Box sx={{ flex: 1, minWidth: 0 }}>
          <Box sx={{ fontSize: '12px', fontWeight: 600, color: NEUTRAL.onSurfaceVariant }}>{n.authorName}</Box>
          <Box sx={{ fontSize: '11px', color: NEUTRAL.faint }}>{fmtDate(n.createdAt.slice(0, 10))}</Box>
        </Box>
        {n.pinned ? <Sym name="push_pin" size={17} color={t.primary} /> : null}
      </Box>
      <Box sx={{ fontSize: '15px', fontWeight: 700, mb: '4px' }}>{n.title}</Box>
      <Box
        sx={{
          fontSize: '13px',
          color: NEUTRAL.onSurfaceVariant,
          lineHeight: 1.5,
          display: '-webkit-box',
          WebkitLineClamp: compact ? 2 : 99,
          WebkitBoxOrient: 'vertical',
          overflow: 'hidden',
        }}
      >
        {n.body}
      </Box>
    </Box>
  );
});
