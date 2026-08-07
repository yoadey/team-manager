import { useState } from 'react';
import Box from '@mui/material/Box';
import ButtonBase from '@mui/material/ButtonBase';
import { useApp } from '@/context/AppContext';
import { buildTokens, fmtDate, fmtMoney, NEUTRAL } from '@/styles/tokens';
import { SectionTitle, EmptyState, Sym } from '@/components/ui';
import type { FinanceOverview, Transaction } from '../types';
import { getIntlLocale, t } from '@/i18n';

type App = ReturnType<typeof useApp>;
type Tk = ReturnType<typeof buildTokens>;

interface Props {
  app: App;
  t: Tk;
  f: FinanceOverview;
  canFin: boolean;
}

export function FinancesTransactions({ app, t: tk, f, canFin }: Props) {
  const [category, setCategory] = useState<string | null>(null);
  const categories = [...new Set(f.transactions.map((tx) => tx.category).filter(Boolean))].sort((a, b) =>
    a.localeCompare(b, getIntlLocale()),
  );
  const shown = category ? f.transactions.filter((tx) => tx.category === category) : f.transactions;

  const txRow = (tx: Transaction) => {
    const inner = (
      <>
        <Box
          component="span"
          sx={{
            width: '38px',
            height: '38px',
            borderRadius: '11px',
            background: tx.type === 'income' ? NEUTRAL.successBg : NEUTRAL.errorBg,
            color: tx.type === 'income' ? NEUTRAL.success : NEUTRAL.error,
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            flex: '0 0 auto',
          }}
        >
          <Sym
            name={tx.type === 'income' ? 'south_west' : 'north_east'}
            size={20}
            color={tx.type === 'income' ? NEUTRAL.success : NEUTRAL.error}
          />
        </Box>
        <Box sx={{ flex: 1, minWidth: 0 }}>
          <Box
            sx={{
              fontSize: '14px',
              fontWeight: 600,
              whiteSpace: 'nowrap',
              overflow: 'hidden',
              textOverflow: 'ellipsis',
            }}
          >
            {tx.title}
          </Box>
          <Box sx={{ fontSize: '12px', color: NEUTRAL.faint }}>
            {[tx.category, fmtDate(tx.date)].filter(Boolean).join(' · ')}
          </Box>
        </Box>
        <Box sx={{ fontSize: '15px', fontWeight: 700, color: tx.type === 'income' ? NEUTRAL.success : NEUTRAL.error }}>
          {(tx.type === 'income' ? '+' : '−') + fmtMoney(tx.amount).replace('-', '')}
        </Box>
        {canFin ? <Sym name="chevron_right" size={20} color={NEUTRAL.faint} /> : null}
      </>
    );
    const st = {
      display: 'flex',
      alignItems: 'center',
      gap: '12px',
      p: '12px 14px',
      background: NEUTRAL.card,
      width: '100%',
      textAlign: 'left',
      border: 'none',
    } as const;
    return canFin ? (
      <ButtonBase
        key={tx.id}
        onClick={() => app.openTxForm(tx)}
        sx={{ ...st, cursor: 'pointer', justifyContent: 'flex-start' }}
      >
        {inner}
      </ButtonBase>
    ) : (
      <Box key={tx.id} sx={st}>
        {inner}
      </Box>
    );
  };
  return (
    <Box key="um">
      <SectionTitle
        right={
          canFin ? (
            <ButtonBase
              onClick={() => app.openTxForm()}
              sx={{
                display: 'flex',
                alignItems: 'center',
                gap: '5px',
                border: 'none',
                background: 'transparent',
                color: tk.primary,
                fontWeight: 700,
                fontSize: '13px',
                cursor: 'pointer',
              }}
            >
              <Sym name="add" size={17} color={tk.primary} />
              {t('finances.txAdd')}
            </ButtonBase>
          ) : null
        }
      >
        {t('finances.txListTitle', { n: shown.length })}
      </SectionTitle>
      {categories.length ? (
        <Box sx={{ display: 'flex', gap: '8px', overflowX: 'auto', pb: '4px', mb: '14px' }}>
          {[null, ...categories].map((c) => {
            const sel = category === c;
            return (
              <ButtonBase
                key={c ?? '__all'}
                onClick={() => setCategory(c)}
                sx={{
                  flex: '0 0 auto',
                  p: '7px 13px',
                  borderRadius: '999px',
                  fontSize: '12px',
                  fontWeight: 700,
                  cursor: 'pointer',
                  border: '1.5px solid ' + (sel ? tk.primary : NEUTRAL.inputBorder),
                  background: sel ? tk.primaryContainer : NEUTRAL.card,
                  color: sel ? tk.onPrimaryContainer : NEUTRAL.onSurfaceVariant,
                }}
              >
                {c ?? t('finances.txFilterAll')}
              </ButtonBase>
            );
          })}
        </Box>
      ) : null}
      {f.transactions.length ? (
        shown.length ? (
          <Box
            sx={{
              display: 'flex',
              flexDirection: 'column',
              gap: '1px',
              border: `1px solid ${NEUTRAL.line}`,
              borderRadius: '16px',
              overflow: 'hidden',
            }}
          >
            {shown.map(txRow)}
          </Box>
        ) : (
          <EmptyState icon="receipt_long" text={t('finances.txFilterEmpty')} />
        )
      ) : (
        <EmptyState icon="receipt_long" text={t('finances.txEmpty')} />
      )}
    </Box>
  );
}
