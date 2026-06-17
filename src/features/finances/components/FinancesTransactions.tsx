import Box from '@mui/material/Box';
import ButtonBase from '@mui/material/ButtonBase';
import { useApp } from '../../../context/AppContext';
import { buildTokens, fmtDate, fmtMoney, NEUTRAL } from '../../../styles/tokens';
import { SectionTitle, EmptyState, Sym } from '../../../components/ui';
import type { FinanceOverview, Transaction } from '../../../types';

type App = ReturnType<typeof useApp>;
type Tk = ReturnType<typeof buildTokens>;

interface Props { app: App; t: Tk; f: FinanceOverview; canFin: boolean; }

export function FinancesTransactions({ app, t, f, canFin }: Props) {
  const txRow = (tx: Transaction) => {
    const inner = (
      <>
        <Box component="span" sx={{ width: '38px', height: '38px', borderRadius: '11px', background: tx.type === 'income' ? '#D7F0D8' : '#FFDAD6', color: tx.type === 'income' ? '#2E7D32' : '#BA1A1A', display: 'flex', alignItems: 'center', justifyContent: 'center', fontFamily: "'Material Symbols Outlined'", fontSize: '20px', flex: '0 0 auto' }}>
          {tx.type === 'income' ? 'south_west' : 'north_east'}
        </Box>
        <Box sx={{ flex: 1, minWidth: 0 }}>
          <Box sx={{ fontSize: '14px', fontWeight: 600, whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis' }}>{tx.title}</Box>
          <Box sx={{ fontSize: '12px', color: NEUTRAL.faint }}>{tx.category + ' · ' + fmtDate(tx.date)}</Box>
        </Box>
        <Box sx={{ fontSize: '15px', fontWeight: 700, color: tx.type === 'income' ? '#2E7D32' : '#BA1A1A' }}>
          {(tx.type === 'income' ? '+' : '−') + fmtMoney(tx.amount).replace('-', '')}
        </Box>
        {canFin ? <Sym name="chevron_right" size={20} color="#C0C2CA" /> : null}
      </>
    );
    const st = { display: 'flex', alignItems: 'center', gap: '12px', p: '12px 14px', background: '#fff', width: '100%', textAlign: 'left', border: 'none' } as const;
    return canFin
      ? <ButtonBase key={tx.id} onClick={() => app.openTxForm(tx)} sx={{ ...st, cursor: 'pointer', justifyContent: 'flex-start' }}>{inner}</ButtonBase>
      : <Box key={tx.id} sx={st}>{inner}</Box>;
  };
  return (
    <Box key="um">
      <SectionTitle right={canFin ? (
        <ButtonBase onClick={() => app.openTxForm(null)} sx={{ display: 'flex', alignItems: 'center', gap: '5px', border: 'none', background: 'transparent', color: t.primary, fontWeight: 700, fontSize: '13px', cursor: 'pointer' }}>
          <Sym name="add" size={17} color={t.primary} />Buchung
        </ButtonBase>
      ) : null}>{'Alle Umsätze · ' + f.transactions.length}</SectionTitle>
      {f.transactions.length ? (
        <Box sx={{ display: 'flex', flexDirection: 'column', gap: '1px', border: `1px solid ${NEUTRAL.line}`, borderRadius: '16px', overflow: 'hidden' }}>
          {f.transactions.map(txRow)}
        </Box>
      ) : <EmptyState icon="receipt_long" text="Noch keine Umsätze gebucht" />}
    </Box>
  );
}
