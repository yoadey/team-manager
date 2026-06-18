import Box from '@mui/material/Box';
import ButtonBase from '@mui/material/ButtonBase';
import { useApp } from '@/context/AppContext';
import { buildTokens, NEUTRAL } from '@/styles/tokens';
import { Card, Chip, EmptyState, SpinnerBox, Sym } from '@/components/ui';
import type { Poll } from './types';

export function PollsPage() {
  const app = useApp();
  const { state } = app;
  const t = buildTokens(state.primaryColor);
  if (!state.polls) return <SpinnerBox />;
  if (!state.polls.length) return <EmptyState icon="how_to_vote" text="Noch keine Umfragen" />;

  return (
    <Box sx={{ maxWidth: '680px', display: 'flex', flexDirection: 'column', gap: '14px' }}>
      {state.polls.map((p: Poll) => {
        const voted = !!p.myVote;
        const opts = p.options.map((o) => {
          const sel = voted && p.myVote!.includes(o.id);
          return (
            <ButtonBase key={o.id} onClick={() => app.togglePollOption(p, o.id)} sx={{ position: 'relative', display: 'block', width: '100%', textAlign: 'left', border: '1.5px solid ' + (sel ? t.primary : '#E0E2EA'), background: '#fff', borderRadius: '12px', p: '11px 14px', cursor: 'pointer', overflow: 'hidden', justifyContent: 'flex-start' }}>
              <Box sx={{ position: 'absolute', left: 0, top: 0, bottom: 0, width: (voted ? o.pct : 0) + '%', background: sel ? t.primaryContainer : '#F0F1F5', transition: 'width .4s', zIndex: 0 }} />
              <Box sx={{ position: 'relative', zIndex: 1, display: 'flex', alignItems: 'center', gap: '10px' }}>
                <Box component="span" sx={{ width: '20px', height: '20px', borderRadius: p.multiple ? '5px' : '50%', border: '2px solid ' + (sel ? t.primary : '#B0B3BC'), display: 'flex', alignItems: 'center', justifyContent: 'center', flex: '0 0 auto' }}>
                  {sel ? <Sym name="check" size={14} color={t.primary} /> : null}
                </Box>
                <Box component="span" sx={{ flex: 1, fontSize: '14px', fontWeight: sel ? 700 : 500 }}>{o.text}</Box>
                {voted ? <Box component="span" sx={{ fontSize: '13px', fontWeight: 700, color: NEUTRAL.secondary }}>{o.pct + '%'}</Box> : null}
              </Box>
            </ButtonBase>
          );
        });
        return (
          <Card key={p.id}>
            <Box sx={{ display: 'flex', alignItems: 'flex-start', gap: '8px', mb: '12px' }}>
              <Box sx={{ flex: 1, fontSize: '16px', fontWeight: 700 }}>{p.question}</Box>
              <Chip label={p.anonymous ? 'Anonym' : p.multiple ? 'Mehrfach' : 'Einfach'} color={NEUTRAL.secondary} bg="#ECEDF3" icon={p.anonymous ? 'visibility_off' : p.multiple ? 'checklist' : 'radio_button_checked'} />
            </Box>
            <Box sx={{ display: 'flex', flexDirection: 'column', gap: '8px' }}>{opts}</Box>
            <Box sx={{ mt: '10px', fontSize: '12px', color: NEUTRAL.faint }}>{p.totalVotes + ' Stimme(n)' + (p.anonymous ? ' · anonym' : '')}</Box>
          </Card>
        );
      })}
    </Box>
  );
}
