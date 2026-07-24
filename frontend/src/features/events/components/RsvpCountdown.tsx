import { useEffect, useMemo, useState } from 'react';
import Box from '@mui/material/Box';
import { NEUTRAL } from '@/styles/tokens';
import { Sym } from '@/components/ui';
import { t } from '@/i18n';

const ONE_DAY_MS = 24 * 60 * 60 * 1000;
// 30s is plenty for a countdown that only ever displays minute granularity --
// re-rendering every second would just burn cycles re-computing the same
// displayed value most of the time.
const TICK_MS = 30_000;

/** "3h 12m" / "45m", localized -- minute granularity only (no seconds), since that's all formatRemaining ever displays. */
function formatRemaining(ms: number): string {
  const totalMinutes = Math.max(0, Math.floor(ms / 60_000));
  const hours = Math.floor(totalMinutes / 60);
  const minutes = totalMinutes % 60;
  if (hours > 0) return t('events.rsvpCountdownHoursMinutes', { hours, minutes });
  return t('events.rsvpCountdownMinutes', { minutes });
}

/**
 * Shows a live countdown once less than 24h remain until `deadline` (an ISO
 * 8601 timestamp). Renders nothing before that window opens, and nothing
 * once the deadline has passed (a passed deadline is communicated elsewhere
 * -- by the RSVP buttons/response path rejecting the change -- not by this
 * component switching to a "closed" message).
 */
export function RsvpCountdown({ deadline }: { deadline: string }) {
  const target = useMemo(() => new Date(deadline).getTime(), [deadline]);
  const [now, setNow] = useState(() => Date.now());

  useEffect(() => {
    const id = setInterval(() => setNow(Date.now()), TICK_MS);
    return () => clearInterval(id);
  }, []);

  const remaining = target - now;
  if (remaining <= 0 || remaining > ONE_DAY_MS) return null;

  return (
    <Box
      sx={{
        display: 'flex',
        alignItems: 'center',
        gap: '7px',
        fontSize: '12px',
        fontWeight: 600,
        color: NEUTRAL.warn,
        background: NEUTRAL.warnBg,
        borderRadius: '10px',
        p: '8px 12px',
        mb: '10px',
      }}
    >
      <Sym name="hourglass_top" size={16} color={NEUTRAL.warn} />
      {t('events.rsvpCountdown', { time: formatRemaining(remaining) })}
    </Box>
  );
}
