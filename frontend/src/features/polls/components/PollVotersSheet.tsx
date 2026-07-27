import { useMemo, useState } from 'react';
import Box from '@mui/material/Box';
import ButtonBase from '@mui/material/ButtonBase';
import type { SheetProps } from '@/sheets';
import { buildTokens, NEUTRAL } from '@/styles/tokens';
import { Av, Sym, EmptyState } from '@/components/ui';
import { t } from '@/i18n';
import type { Poll, PollVoter } from '../types';

type View = 'byOption' | 'matrix';

/** One row of the matrix: a distinct voter plus the set of option ids they picked. */
interface VoterRow {
  voter: PollVoter;
  picked: Set<string>;
}

/** Collapse the per-option voter lists into one row per distinct user (keyed by
 *  userId), remembering which options each user selected. */
function buildRows(poll: Poll): VoterRow[] {
  const byUser = new Map<string, VoterRow>();
  for (const opt of poll.options) {
    for (const v of opt.voters) {
      let row = byUser.get(v.userId);
      if (!row) {
        row = { voter: v, picked: new Set() };
        byUser.set(v.userId, row);
      }
      row.picked.add(opt.id);
    }
  }
  return [...byUser.values()].sort((a, b) => a.voter.name.localeCompare(b.voter.name));
}

export function PollVotersSheet({ app, sheet }: SheetProps) {
  const { state } = app;
  const tk = buildTokens(state.primaryColor);
  const poll = sheet.poll;
  const [view, setView] = useState<View>('byOption');

  const rows = useMemo(() => (poll ? buildRows(poll) : []), [poll]);

  // Anonymous polls (and polls with no votes yet) expose no identities.
  if (!poll || poll.anonymous) {
    return <EmptyState icon="visibility_off" text={t('polls.votersAnonymous')} />;
  }
  if (!rows.length) {
    return <EmptyState icon="how_to_vote" text={t('polls.votersEmpty')} />;
  }

  const toggle = (v: View, label: string, icon: string) => {
    const on = view === v;
    return (
      <ButtonBase
        key={v}
        aria-pressed={on}
        onClick={() => setView(v)}
        sx={{
          flex: 1,
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          gap: '6px',
          p: '9px',
          borderRadius: '10px',
          fontSize: '13px',
          fontWeight: on ? 700 : 600,
          cursor: 'pointer',
          background: on ? tk.primary : NEUTRAL.card,
          color: on ? NEUTRAL.card : NEUTRAL.onSurfaceVariant,
          border: `1px solid ${on ? tk.primary : NEUTRAL.line3}`,
        }}
      >
        <Sym name={icon} size={17} color={on ? NEUTRAL.card : NEUTRAL.faint} />
        {label}
      </ButtonBase>
    );
  };

  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: '14px' }}>
      <Box sx={{ display: 'flex', gap: '8px' }}>
        {toggle('byOption', t('polls.votersByOption'), 'format_list_bulleted')}
        {toggle('matrix', t('polls.votersMatrix'), 'grid_on')}
      </Box>
      {view === 'byOption' ? <ByOption poll={poll} /> : <Matrix poll={poll} rows={rows} primary={tk.primary} />}
    </Box>
  );
}

function VoterChip({ voter }: { voter: PollVoter }) {
  return (
    <Box sx={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
      <Av name={voter.name} photo={voter.photo} color={voter.color} size={26} />
      <Box component="span" sx={{ fontSize: '13px', fontWeight: 600 }}>
        {voter.name}
      </Box>
    </Box>
  );
}

function ByOption({ poll }: { poll: Poll }) {
  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: '14px' }}>
      {poll.options.map((o) => (
        <Box key={o.id}>
          <Box
            sx={{
              display: 'flex',
              alignItems: 'baseline',
              justifyContent: 'space-between',
              gap: '8px',
              mb: '8px',
            }}
          >
            <Box sx={{ fontSize: '14px', fontWeight: 700 }}>{o.text}</Box>
            <Box sx={{ fontSize: '12px', fontWeight: 700, color: NEUTRAL.secondary }}>{o.count}</Box>
          </Box>
          {o.voters.length ? (
            <Box sx={{ display: 'flex', flexDirection: 'column', gap: '8px', pl: '2px' }}>
              {o.voters.map((v) => (
                <VoterChip key={v.userId} voter={v} />
              ))}
            </Box>
          ) : (
            <Box sx={{ fontSize: '12px', color: NEUTRAL.faint }}>{t('polls.votersNone')}</Box>
          )}
        </Box>
      ))}
    </Box>
  );
}

function Matrix({ poll, rows, primary }: { poll: Poll; rows: VoterRow[]; primary: string }) {
  const cellSx = {
    textAlign: 'center' as const,
    fontSize: '12px',
    p: '6px',
    borderBottom: `1px solid ${NEUTRAL.line2}`,
  };
  return (
    <Box>
      <Box sx={{ overflowX: 'auto' }}>
        <Box component="table" sx={{ borderCollapse: 'collapse', width: '100%', minWidth: 0 }}>
          <Box component="thead">
            <Box component="tr">
              <Box component="th" sx={{ ...cellSx, textAlign: 'left', fontWeight: 700 }}>
                {t('polls.votersMemberCol')}
              </Box>
              {poll.options.map((_, i) => (
                <Box component="th" key={i} sx={{ ...cellSx, fontWeight: 700, minWidth: '28px' }}>
                  {i + 1}
                </Box>
              ))}
            </Box>
          </Box>
          <Box component="tbody">
            {rows.map(({ voter, picked }) => (
              <Box component="tr" key={voter.userId}>
                <Box component="td" sx={{ ...cellSx, textAlign: 'left' }}>
                  <VoterChip voter={voter} />
                </Box>
                {poll.options.map((o) => (
                  <Box component="td" key={o.id} sx={cellSx}>
                    {picked.has(o.id) ? (
                      <Sym name="check" size={16} color={primary} />
                    ) : (
                      <Box component="span" sx={{ color: NEUTRAL.faint }}>
                        ·
                      </Box>
                    )}
                  </Box>
                ))}
              </Box>
            ))}
          </Box>
        </Box>
      </Box>
      {/* Legend maps the numbered columns back to option text. */}
      <Box sx={{ mt: '12px', display: 'flex', flexDirection: 'column', gap: '4px' }}>
        {poll.options.map((o, i) => (
          <Box key={o.id} sx={{ fontSize: '12px', color: NEUTRAL.secondary }}>
            <Box component="span" sx={{ fontWeight: 700 }}>
              {i + 1}.
            </Box>{' '}
            {o.text}
          </Box>
        ))}
      </Box>
    </Box>
  );
}
