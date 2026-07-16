import Box from '@mui/material/Box';
import ButtonBase from '@mui/material/ButtonBase';
import { buildTokens, fmtMoney, NEUTRAL } from '@/styles/tokens';
import { EmptyState, Sym } from '@/components/ui';
import type { Penalty } from '../types';
import type { SheetProps } from '@/sheets/types';
import { useFinanceOverviewQuery } from '../hooks/useFinanceQueries';
import { t } from '@/i18n';

export function PenaltyCatalogSheet({ app, sheet }: SheetProps) {
  const { state } = app;
  const tk = buildTokens(state.primaryColor);
  void sheet;
  const { data: f } = useFinanceOverviewQuery(app.api, state.activeTeamId);
  const canFin = app.can('finances', 'write');
  const pens: Penalty[] = f ? f.penalties : [];

  const rows = pens.map((p) => {
    const inner = (
      <>
        <Box
          component="span"
          key="i"
          sx={{
            width: '36px',
            height: '36px',
            borderRadius: '10px',
            background: tk.primaryContainer,
            color: tk.primary,
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            fontFamily: "'Material Symbols Outlined'",
            fontSize: '19px',
            flex: '0 0 auto',
          }}
        >
          gavel
        </Box>
        <Box key="m" sx={{ flex: 1, minWidth: 0 }}>
          <Box
            key="l"
            sx={{
              fontSize: '14px',
              fontWeight: 600,
              whiteSpace: 'nowrap',
              overflow: 'hidden',
              textOverflow: 'ellipsis',
            }}
          >
            {p.label}
          </Box>
        </Box>
        <Box component="b" key="a" sx={{ fontSize: '14px', color: tk.primary, flex: '0 0 auto' }}>
          {fmtMoney(p.amount)}
        </Box>
        {canFin ? <Sym name="chevron_right" size={20} color={NEUTRAL.faint} /> : null}
      </>
    );
    const baseSx = {
      display: 'flex',
      alignItems: 'center',
      gap: '11px',
      background: NEUTRAL.card,
      border: `1px solid ${NEUTRAL.line}`,
      borderRadius: '14px',
      p: '11px 13px',
      width: '100%',
      textAlign: 'left' as const,
    };
    return canFin ? (
      <ButtonBase
        key={p.id}
        onClick={() => app.openPenaltyForm(p)}
        sx={{ ...baseSx, cursor: 'pointer', justifyContent: 'flex-start' }}
      >
        {inner}
      </ButtonBase>
    ) : (
      <Box key={p.id} sx={baseSx}>
        {inner}
      </Box>
    );
  });

  const add = canFin ? (
    <ButtonBase
      key="add"
      onClick={() => app.openPenaltyForm()}
      sx={{
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        gap: '9px',
        width: '100%',
        p: '13px',
        borderRadius: '14px',
        border: `1.5px dashed ${NEUTRAL.inputBorder}`,
        background: 'transparent',
        cursor: 'pointer',
        color: tk.primary,
        fontWeight: 600,
        fontSize: '14px',
      }}
    >
      <Sym name="add_circle" size={20} color={tk.primary} />
      {t('finances.catalogAdd')}
    </ButtonBase>
  ) : null;

  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: '10px' }}>
      <Box key="i" sx={{ fontSize: '13px', color: NEUTRAL.secondary, lineHeight: 1.5, mb: '2px' }}>
        {t('finances.catalogHint')}
      </Box>
      {rows.length ? rows : <EmptyState icon="gavel" text={t('finances.catalogEmpty')} />}
      {add}
    </Box>
  );
}
