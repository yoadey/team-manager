import Box from '@mui/material/Box';
import ButtonBase from '@mui/material/ButtonBase';
import { useApp } from '../context/AppContext';
import { buildTokens } from '../styles/tokens';
import { EmptyState, SpinnerBox, Sym } from '../components/ui';
import { NewsCard } from '../components/cards';

export function News() {
  const app = useApp();
  const { state } = app;
  const t = buildTokens(state.primaryColor);
  void t;
  if (!state.news) return <SpinnerBox />;
  if (!state.news.length) return <EmptyState icon="campaign" text="Noch keine Neuigkeiten" />;
  const canDel = app.can('news', 'write');

  return (
    <Box sx={{ maxWidth: '720px', display: 'flex', flexDirection: 'column', gap: '12px' }}>
      {state.news.map((n) => {
        const cardEl = <NewsCard n={n} compact={false} />;
        if (!canDel) return <Box key={n.id}>{cardEl}</Box>;
        return (
          <Box key={n.id} sx={{ position: 'relative' }}>
            {cardEl}
            <ButtonBase onClick={() => app.removeNews(n.id)} sx={{ position: 'absolute', top: '12px', right: '12px', width: '30px', height: '30px', borderRadius: '50%', border: 'none', background: '#F4F4FA', color: '#9A9DA6', cursor: 'pointer' }}>
              <Sym name="delete" size={17} color="#9A9DA6" />
            </ButtonBase>
          </Box>
        );
      })}
    </Box>
  );
}
