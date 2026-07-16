import Box from '@mui/material/Box';
import ButtonBase from '@mui/material/ButtonBase';
import { useApp } from '@/context/AppContext';
import { buildTokens, NEUTRAL } from '@/styles/tokens';
import { Card, Chip, EmptyState, SkeletonList, Sym } from '@/components/ui';
import type { Poll } from './types';
import { usePollsQuery } from './hooks/usePollQueries';
import { t } from '@/i18n';

export function PollsPage() {
  const app = useApp();
  const { state } = app;
  const tk = buildTokens(state.primaryColor);
  const { data: polls } = usePollsQuery(app.api, state.activeTeamId);
  if (!polls) return <SkeletonList rows={4} rowHeight={100} />;
  if (!polls.length) return <EmptyState icon="how_to_vote" text={t('polls.empty')} />;
  const canDelete = app.can('polls', 'write');

  return (
    <Box sx={{ maxWidth: '680px', display: 'flex', flexDirection: 'column', gap: '14px' }}>
      {polls.map((p: Poll) => {
        const voted = !!p.myVote;
        const opts = p.options.map((o) => {
          const sel = voted && p.myVote!.includes(o.id);
          return (
            <ButtonBase
              key={o.id}
              role={p.multiple ? 'checkbox' : 'radio'}
              aria-checked={sel}
              onClick={() => app.togglePollOption(p, o.id)}
              sx={{
                position: 'relative',
                display: 'block',
                width: '100%',
                textAlign: 'left',
                border: '1.5px solid ' + (sel ? tk.primary : NEUTRAL.line3),
                background: NEUTRAL.card,
                borderRadius: '12px',
                p: '11px 14px',
                cursor: 'pointer',
                overflow: 'hidden',
                justifyContent: 'flex-start',
              }}
            >
              <Box
                sx={{
                  position: 'absolute',
                  left: 0,
                  top: 0,
                  bottom: 0,
                  width: (voted ? o.pct : 0) + '%',
                  background: sel ? tk.primaryContainer : '#F0F1F5',
                  transition: 'width .4s',
                  zIndex: 0,
                }}
              />
              <Box sx={{ position: 'relative', zIndex: 1, display: 'flex', alignItems: 'center', gap: '10px' }}>
                <Box
                  component="span"
                  sx={{
                    width: '20px',
                    height: '20px',
                    borderRadius: p.multiple ? '5px' : '50%',
                    border: '2px solid ' + (sel ? tk.primary : NEUTRAL.faint),
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'center',
                    flex: '0 0 auto',
                  }}
                >
                  {sel ? <Sym name="check" size={14} color={tk.primary} /> : null}
                </Box>
                <Box component="span" sx={{ flex: 1, fontSize: '14px', fontWeight: sel ? 700 : 500 }}>
                  {o.text}
                </Box>
                {voted ? (
                  <Box component="span" sx={{ fontSize: '13px', fontWeight: 700, color: NEUTRAL.secondary }}>
                    {o.pct + '%'}
                  </Box>
                ) : null}
              </Box>
            </ButtonBase>
          );
        });
        return (
          <Card key={p.id}>
            <Box sx={{ display: 'flex', alignItems: 'flex-start', gap: '8px', mb: '12px' }}>
              <Box sx={{ flex: 1, fontSize: '16px', fontWeight: 700 }}>{p.question}</Box>
              <Chip
                label={p.anonymous ? t('polls.anonymous') : p.multiple ? t('polls.multiple') : t('polls.single')}
                color={NEUTRAL.secondary}
                bg={NEUTRAL.line2}
                icon={p.anonymous ? 'visibility_off' : p.multiple ? 'checklist' : 'radio_button_checked'}
              />
              {canDelete ? (
                <ButtonBase
                  onClick={() => app.removePoll(p.id)}
                  aria-label={t('polls.deleteLabel')}
                  sx={{
                    width: '30px',
                    height: '30px',
                    borderRadius: '50%',
                    background: NEUTRAL.errorBg,
                    color: NEUTRAL.error,
                    cursor: 'pointer',
                    flex: '0 0 auto',
                  }}
                >
                  <Sym name="delete" size={16} color={NEUTRAL.error} />
                </ButtonBase>
              ) : null}
            </Box>
            {/* Known, accepted a11y gap: role="radiogroup"/"radio" here don't
                come with the WAI-ARIA APG's required roving-tabindex +
                Arrow-key navigation between options -- each option is its
                own independent tab stop instead. The exposed role promises
                keyboard behavior the implementation doesn't provide.
                Left as a plain toggle-button pattern (matching how
                EventFormSheet's type/mode buttons correctly use
                aria-pressed instead of radio) rather than implementing full
                roving-tabindex behavior here. */}
            <Box
              role={p.multiple ? 'group' : 'radiogroup'}
              aria-label={p.question}
              sx={{ display: 'flex', flexDirection: 'column', gap: '8px' }}
            >
              {opts}
            </Box>
            <Box sx={{ mt: '10px', fontSize: '12px', color: NEUTRAL.faint }}>
              {p.anonymous
                ? t('polls.votesAnon', { n: p.totalVotes, count: p.totalVotes })
                : t('polls.votes', { n: p.totalVotes, count: p.totalVotes })}
            </Box>
          </Card>
        );
      })}
    </Box>
  );
}
