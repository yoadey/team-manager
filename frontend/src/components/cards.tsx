import { memo, useSyncExternalStore } from 'react';
import Box from '@mui/material/Box';
import ButtonBase from '@mui/material/ButtonBase';
import { useAppActions } from '@/context/AppContext';
import { useCompact } from '@/layouts/AppShell';
import { buildTokens, fmtDate, hhmm, NEUTRAL, statusMeta, typeMeta } from '@/styles/tokens';
import { parseDateOnlyLocal, todayLocalDate } from '@/utils/date';
import { getIntlLocale, getLocale, subscribeLocale, t } from '@/i18n';
import type { TeamEvent } from '@/features/events';
import type { NewsItem } from '@/features/news';
import { Av, Chip, Sym, metaItem } from './ui';

/**
 * Subscribes to the module-level i18n store directly (rather than the
 * useLocale() context hook, which throws outside a LocaleProvider) so
 * EventCard/NewsCard re-render on a locale switch without depending on
 * being mounted inside any particular provider tree.
 */
function useLocaleSubscription(): void {
  useSyncExternalStore(subscribeLocale, getLocale);
}

/** Event list card (used on Home and Events). Mirrors prototype eventCard().
 *  Memoised + actions-only so it skips re-renders when unrelated state changes
 *  (e.g. typing in a form, incoming notifications) — only `e`/layout changes
 *  re-render a row. `t()`/`getIntlLocale()` read module-level i18n state, not
 *  props, so without useLocaleSubscription() a locale switch wouldn't
 *  re-render an already-mounted card until its `e` prop happened to change
 *  for an unrelated reason -- leaving cancelled/meet-time labels and the
 *  month abbreviation stuck in the old language. */
export const EventCard = memo(function EventCard({ e }: { e: TeamEvent }) {
  useLocaleSubscription();
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
        background: NEUTRAL.card,
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
          {cancelled ? (
            <Chip label={t('events.cancelledLabel')} color={NEUTRAL.error} bg={NEUTRAL.errorBg} icon="event_busy" />
          ) : null}
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
          {e.meetTime ? metaItem('login', t('events.meetTime', { time: hhmm(e.meetTime) }), 'meet') : null}
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
          <Box
            component="span"
            aria-label={t('events.summaryAriaYes', { n: e.summary.yes })}
            sx={{ color: NEUTRAL.success }}
          >
            {e.summary.yes}✓
          </Box>
          <Box
            component="span"
            aria-label={t('events.summaryAriaNo', { n: e.summary.no })}
            sx={{ color: NEUTRAL.error }}
          >
            {e.summary.no}✕
          </Box>
          {e.summary.maybe ? (
            <Box
              component="span"
              aria-label={t('events.summaryAriaMaybe', { n: e.summary.maybe })}
              sx={{ color: NEUTRAL.warn }}
            >
              {e.summary.maybe}?
            </Box>
          ) : null}
        </Box>
      </Box>
    </ButtonBase>
  );
});

/** News card. compact=true clamps body to 2 lines. Mirrors prototype newsCard().
 *  Memoised + state-free so it skips re-renders when unrelated state changes
 *  -- except locale, since fmtDate() reads module-level i18n state rather
 *  than a prop; see the identical useLocaleSubscription() note on EventCard
 *  above. */
export const NewsCard = memo(function NewsCard({
  n,
  compact = false,
  primaryColor,
}: {
  n: NewsItem;
  compact?: boolean;
  primaryColor: string;
}) {
  useLocaleSubscription();
  const t = buildTokens(primaryColor);
  return (
    <Box sx={{ background: NEUTRAL.card, border: `1px solid ${NEUTRAL.line}`, borderRadius: '16px', p: '15px 16px' }}>
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
