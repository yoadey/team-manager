import Box from '@mui/material/Box';
import ButtonBase from '@mui/material/ButtonBase';
import { useApp } from '@/context/AppContext';
import { buildTokens, NEUTRAL } from '@/styles/tokens';
import { EmptyState, SkeletonList, Sym } from '@/components/ui';
import { NewsCard } from '@/components/cards';
import { t } from '@/i18n';

export function NewsPage() {
  const app = useApp();
  const { state } = app;
  const tk = buildTokens(state.primaryColor);
  void tk;
  if (!state.news) return <SkeletonList rows={4} rowHeight={120} />;
  if (!state.news.length) return <EmptyState icon="campaign" text={t('news.empty')} />;
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
                aria-label={t('news.editLabel')}
                sx={{
                  width: '30px',
                  height: '30px',
                  borderRadius: '50%',
                  background: NEUTRAL.sidebar,
                  color: NEUTRAL.faint,
                  cursor: 'pointer',
                }}
              >
                <Sym name="edit" size={16} color={NEUTRAL.faint} />
              </ButtonBase>
              <ButtonBase
                onClick={() => app.removeNews(n.id)}
                aria-label={t('news.deleteLabel')}
                sx={{
                  width: '30px',
                  height: '30px',
                  borderRadius: '50%',
                  background: NEUTRAL.errorBg,
                  color: NEUTRAL.error,
                  cursor: 'pointer',
                }}
              >
                <Sym name="delete" size={16} color={NEUTRAL.error} />
              </ButtonBase>
            </Box>
          </Box>
        );
      })}
    </Box>
  );
}
