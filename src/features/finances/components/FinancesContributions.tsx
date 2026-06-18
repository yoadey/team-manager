import Box from '@mui/material/Box';
import ButtonBase from '@mui/material/ButtonBase';
import { useApp } from '@/context/AppContext';
import { buildTokens, fmtMoney, monthName, NEUTRAL } from '@/styles/tokens';
import { Av, Chip, EmptyState, Sym } from '@/components/ui';
import type { Contribution, FinanceOverview } from '../types';

type App = ReturnType<typeof useApp>;
type Tk = ReturnType<typeof buildTokens>;

interface Props { app: App; t: Tk; f: FinanceOverview; canFin: boolean; }

export function FinancesContributions({ app, t, f, canFin }: Props) {
  const { state } = app;
  const contribs = f.contributions || [];
  const months = [...new Set(contribs.map((c) => c.month).filter(Boolean))].sort().reverse();
  if (!months.length) return <EmptyState icon="payments" text="Noch keine Beiträge erfasst" />;
  const sel = state.contribMonth && months.includes(state.contribMonth) ? state.contribMonth : months[0];
  const rows = contribs.filter((c) => c.month === sel).sort((a, b) => a.name!.localeCompare(b.name!, 'de'));
  const paidRows = rows.filter((c) => c.status === 'paid');
  const sum = paidRows.reduce((s, c) => s + c.amount, 0);
  const total = rows.reduce((s, c) => s + c.amount, 0);
  const pct = rows.length ? Math.round((paidRows.length / rows.length) * 100) : 0;

  const monthChips = (
    <Box key="mc" sx={{ display: 'flex', gap: '8px', overflowX: 'auto', pb: '4px', mb: '14px' }}>
      {months.map((m) => {
        const s = m === sel;
        const open = contribs.filter((c) => c.month === m && c.status === 'open').length;
        return (
          <ButtonBase key={m} onClick={() => app.setState({ contribMonth: m })} sx={{ flex: '0 0 auto', display: 'flex', flexDirection: 'column', alignItems: 'flex-start', gap: '2px', p: '9px 14px', borderRadius: '12px', cursor: 'pointer', border: '1.5px solid ' + (s ? t.primary : '#D0D2DA'), background: s ? t.primaryContainer : '#fff', color: s ? t.onPrimaryContainer : NEUTRAL.onSurfaceVariant }}>
            <Box component="span" sx={{ fontSize: '13px', fontWeight: 700, whiteSpace: 'nowrap', textTransform: 'capitalize' }}>{monthName(m)}</Box>
            <Box component="span" sx={{ fontSize: '11px', color: s ? t.onPrimaryContainer : NEUTRAL.faint, whiteSpace: 'nowrap' }}>{open ? open + ' offen' : 'vollständig'}</Box>
          </ButtonBase>
        );
      })}
    </Box>
  );
  const summary = (
    <Box key="sum" sx={{ display: 'flex', alignItems: 'center', gap: '14px', background: '#fff', border: `1px solid ${NEUTRAL.line}`, borderRadius: '16px', p: '15px 16px', mb: '14px' }}>
      <Box sx={{ flex: 1, minWidth: 0 }}>
        <Box sx={{ fontSize: '15px', fontWeight: 700, textTransform: 'capitalize' }}>{monthName(sel)}</Box>
        <Box sx={{ fontSize: '13px', color: NEUTRAL.secondary, mt: '3px' }}>{paidRows.length + ' von ' + rows.length + ' bezahlt · ' + fmtMoney(sum) + ' von ' + fmtMoney(total)}</Box>
      </Box>
      <Box sx={{ fontSize: '24px', fontWeight: 800, color: pct === 100 ? '#2E7D32' : t.primary, flex: '0 0 auto' }}>{pct + '%'}</Box>
    </Box>
  );
  const list = (
    <Box key="l" sx={{ display: 'flex', flexDirection: 'column', gap: '8px' }}>
      {rows.map((c: Contribution) => {
        const paid = c.status === 'paid';
        return (
          <Box key={c.id} sx={{ display: 'flex', alignItems: 'center', gap: '11px', background: '#fff', border: `1px solid ${NEUTRAL.line}`, borderRadius: '14px', p: '10px 13px' }}>
            <Av name={c.name} photo={c.photo} color={c.avatarColor} size={36} />
            <Box sx={{ flex: 1, minWidth: 0 }}>
              <Box sx={{ fontSize: '14px', fontWeight: 600, whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis' }}>{c.name}</Box>
              <Box sx={{ fontSize: '12px', color: NEUTRAL.faint }}>{fmtMoney(c.amount)}</Box>
            </Box>
            {canFin ? (
              <ButtonBase onClick={() => app.toggleContribution(c.id)} sx={{ border: 'none', cursor: 'pointer', borderRadius: '9px', p: '7px 12px', fontSize: '12px', fontWeight: 600, background: paid ? '#D7F0D8' : '#FFE5B8', color: paid ? '#235C26' : '#8A6100', display: 'flex', alignItems: 'center', gap: '5px' }}>
                <Sym name={paid ? 'check_circle' : 'schedule'} size={15} color={paid ? '#235C26' : '#8A6100'} />{paid ? 'bezahlt' : 'offen'}
              </ButtonBase>
            ) : (
              <Chip label={paid ? 'bezahlt' : 'offen'} color={paid ? '#2E7D32' : '#9A5B00'} bg={paid ? '#D7F0D8' : '#FFE5B8'} icon={paid ? 'check_circle' : 'schedule'} />
            )}
          </Box>
        );
      })}
    </Box>
  );
  return (
    <Box key="bei">
      {monthChips}
      {summary}
      {list}
    </Box>
  );
}
