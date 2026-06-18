import Box from '@mui/material/Box';
import ButtonBase from '@mui/material/ButtonBase';
import { useApp } from '@/context/AppContext';
import { buildTokens } from '@/styles/tokens';
import { EmptyState, SpinnerBox, Sym } from '@/components/ui';
import { NewsCard } from '@/components/cards';

export function NewsPage() {
  const app = useApp();
  const { state } = app;
  const t = buildTokens(state.primaryColor);
  void t;
  if (!state.news) return <SpinnerBox />;
  if (!state.news.length) return <EmptyState icon="campaign" text="Noch keine Neuigkeiten" />;
  const canEdit = app.can('news', 'write');

  return (
    <Box sx={{ maxWidth: '720px', display: 'flex', flexDirection: 'column', gap: '12px' }}>
      {state.news.map((n) => {
        const cardEl = <NewsCard n={n} compact={false} primaryColor={state.primaryColor} />;
        if (!canEdit) return <Box key={n.id}>{cardEl}</Box>;
        return (
          <Box key={n.id} sx={{ position: 'relative' }}>
            {cardEl}
            <Box sx={{ position: 'absolute', top: '12px', right: '12px', display: 'flex', gap: '6px' }}>
              <ButtonBase
                onClick={() => app.openNewsForm(n)}
                title="Bearbeiten"
                aria-label="News bearbeiten"
                sx={{
                  width: '30px',
                  height: '30px',
                  borderRadius: '50%',
                  background: '#F4F4FA',
                  color: '#9A9DA6',
                  cursor: 'pointer',
                }}
              >
                <Sym name="edit" size={16} color="#9A9DA6" />
              </ButtonBase>
              <ButtonBase
                onClick={() => app.removeNews(n.id)}
                title="Löschen"
                aria-label="News löschen"
                sx={{
                  width: '30px',
                  height: '30px',
                  borderRadius: '50%',
                  background: '#FFF4F3',
                  color: '#BA1A1A',
                  cursor: 'pointer',
                }}
              >
                <Sym name="delete" size={16} color="#BA1A1A" />
              </ButtonBase>
            </Box>
          </Box>
        );
      })}
    </Box>
  );
}
