import Box from '@mui/material/Box';
import ButtonBase from '@mui/material/ButtonBase';
import { useApp } from '@/context/AppContext';
import { buildTokens, fmtMoney, NEUTRAL } from '@/styles/tokens';
import { SkeletonList, Sym } from '@/components/ui';
import { FinancesTransactions, FinancesPenalties, FinancesContributions } from '@/features/finances';
import { t } from '@/i18n';

export function FinancesPage() {
  const app = useApp();
  const { state } = app;
  const tk = buildTokens(state.primaryColor);
  const f = state.finances;
  if (!f) return <SkeletonList rows={5} rowHeight={64} />;
  const canFin = app.can('finances', 'write');

  const stat = (label: string, val: number, col: string) => (
    <Box
      key={label}
      sx={{
        flex: 1,
        minWidth: '120px',
        background: '#fff',
        border: `1px solid ${NEUTRAL.line}`,
        borderRadius: '16px',
        p: '15px',
      }}
    >
      <Box sx={{ fontSize: '22px', fontWeight: 800, color: col }}>{fmtMoney(val)}</Box>
      <Box sx={{ fontSize: '12px', color: NEUTRAL.secondary, mt: '3px' }}>{label}</Box>
    </Box>
  );

  const tab = state.finTab || 'umsaetze';
  const tabDef: [typeof state.finTab, string, string][] = [
    ['umsaetze', t('finances.tabs.transactions'), 'receipt_long'],
    ['strafen', t('finances.tabs.penalties'), 'gavel'],
    ['beitraege', t('finances.tabs.contributions'), 'payments'],
  ];

  const body =
    tab === 'strafen' ? (
      <FinancesPenalties app={app} t={tk} f={f} canFin={canFin} />
    ) : tab === 'beitraege' ? (
      <FinancesContributions app={app} t={tk} f={f} canFin={canFin} />
    ) : (
      <FinancesTransactions app={app} t={tk} f={f} canFin={canFin} />
    );

  return (
    <Box sx={{ maxWidth: '760px' }}>
      <Box
        sx={{
          background: `linear-gradient(120deg,${tk.primary},${tk.onPrimaryContainer})`,
          color: '#fff',
          borderRadius: '20px',
          p: '20px',
          mb: '16px',
        }}
      >
        <Box sx={{ fontSize: '13px', opacity: 0.85 }}>{t('finances.balance')}</Box>
        <Box sx={{ fontSize: '34px', fontWeight: 800, mt: '4px' }}>{fmtMoney(f.balance)}</Box>
      </Box>
      <Box sx={{ display: 'flex', gap: '10px', flexWrap: 'wrap', mb: '18px' }}>
        {stat(t('finances.income'), f.income, NEUTRAL.success)}
        {stat(t('finances.expense'), f.expense, NEUTRAL.error)}
      </Box>
      <Box sx={{ display: 'flex', gap: '4px', background: '#ECEDF3', borderRadius: '14px', p: '4px', mb: '18px' }}>
        {tabDef.map(([k, l, ic]) => {
          const sel = tab === k;
          return (
            <ButtonBase
              key={k}
              onClick={() => app.setState({ finTab: k })}
              sx={{
                flex: 1,
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
                gap: '7px',
                p: '10px 6px',
                borderRadius: '11px',
                border: 'none',
                cursor: 'pointer',
                fontSize: '13px',
                fontWeight: 700,
                background: sel ? '#fff' : 'transparent',
                color: sel ? tk.primary : '#5A5D66',
                boxShadow: sel ? '0 1px 3px rgba(0,0,0,.12)' : 'none',
              }}
            >
              <Sym name={ic} size={18} color={sel ? tk.primary : '#7A7D86'} />
              {l}
            </ButtonBase>
          );
        })}
      </Box>
      {body}
    </Box>
  );
}
