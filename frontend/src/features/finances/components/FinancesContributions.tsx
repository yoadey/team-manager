import Box from '@mui/material/Box';
import ButtonBase from '@mui/material/ButtonBase';
import { useApp } from '@/context/AppContext';
import { buildTokens, fmtMoney, monthName, NEUTRAL } from '@/styles/tokens';
import { Av, Chip, EmptyState, Sym } from '@/components/ui';
import type { Contribution, FinanceOverview } from '../types';
import { getIntlLocale, t } from '@/i18n';

type App = ReturnType<typeof useApp>;
type Tk = ReturnType<typeof buildTokens>;

interface Props {
  app: App;
  t: Tk;
  f: FinanceOverview;
  canFin: boolean;
}

export function FinancesContributions({ app, t: tk, f, canFin }: Props) {
  const { state } = app;
  const contribs = f.contributions || [];
  const months = [...new Set(contribs.map((c) => c.month).filter(Boolean))].sort().reverse();
  if (!months.length) return <EmptyState icon="payments" text={t('finances.contribEmpty')} />;
  // months[0]! is safe: the `!months.length` branch above already returned.
  const sel = state.contribMonth && months.includes(state.contribMonth) ? state.contribMonth : months[0]!;
  const rows = contribs
    .filter((c) => c.month === sel)
    .sort((a, b) => (a.name ?? '').localeCompare(b.name ?? '', getIntlLocale()));
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
          <ButtonBase
            key={m}
            onClick={() => app.setState({ contribMonth: m })}
            sx={{
              flex: '0 0 auto',
              display: 'flex',
              flexDirection: 'column',
              alignItems: 'flex-start',
              gap: '2px',
              p: '9px 14px',
              borderRadius: '12px',
              cursor: 'pointer',
              border: '1.5px solid ' + (s ? tk.primary : NEUTRAL.inputBorder),
              background: s ? tk.primaryContainer : NEUTRAL.card,
              color: s ? tk.onPrimaryContainer : NEUTRAL.onSurfaceVariant,
            }}
          >
            <Box
              component="span"
              sx={{ fontSize: '13px', fontWeight: 700, whiteSpace: 'nowrap', textTransform: 'capitalize' }}
            >
              {monthName(m)}
            </Box>
            <Box
              component="span"
              sx={{ fontSize: '11px', color: s ? tk.onPrimaryContainer : NEUTRAL.faint, whiteSpace: 'nowrap' }}
            >
              {open ? open + ' ' + t('finances.contribOpen') : t('finances.contribPaid')}
            </Box>
          </ButtonBase>
        );
      })}
    </Box>
  );
  const summary = (
    <Box
      key="sum"
      sx={{
        display: 'flex',
        alignItems: 'center',
        gap: '14px',
        background: NEUTRAL.card,
        border: `1px solid ${NEUTRAL.line}`,
        borderRadius: '16px',
        p: '15px 16px',
        mb: '14px',
      }}
    >
      <Box sx={{ flex: 1, minWidth: 0 }}>
        <Box sx={{ fontSize: '15px', fontWeight: 700, textTransform: 'capitalize' }}>{monthName(sel)}</Box>
        <Box sx={{ fontSize: '13px', color: NEUTRAL.secondary, mt: '3px' }}>
          {t('finances.contribSummary', {
            paid: paidRows.length,
            total: rows.length,
            paidAmt: fmtMoney(sum),
            totalAmt: fmtMoney(total),
          })}
        </Box>
      </Box>
      <Box
        sx={{ fontSize: '24px', fontWeight: 800, color: pct === 100 ? NEUTRAL.success : tk.primary, flex: '0 0 auto' }}
      >
        {pct + '%'}
      </Box>
    </Box>
  );
  const list = (
    <Box key="l" sx={{ display: 'flex', flexDirection: 'column', gap: '8px' }}>
      {rows.map((c: Contribution) => {
        const paid = c.status === 'paid';
        return (
          <Box
            key={c.id}
            sx={{
              display: 'flex',
              alignItems: 'center',
              gap: '11px',
              background: NEUTRAL.card,
              border: `1px solid ${NEUTRAL.line}`,
              borderRadius: '14px',
              p: '10px 13px',
            }}
          >
            <Av name={c.name} photo={c.photo} color={c.avatarColor} size={36} />
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
                {c.name}
              </Box>
              <Box sx={{ fontSize: '12px', color: NEUTRAL.faint }}>{fmtMoney(c.amount)}</Box>
            </Box>
            {canFin ? (
              <ButtonBase
                onClick={() => app.openContribForm(c)}
                aria-label={t('finances.editContribLabel')}
                sx={{
                  width: '30px',
                  height: '30px',
                  borderRadius: '50%',
                  background: NEUTRAL.sidebar,
                  color: NEUTRAL.faint,
                  cursor: 'pointer',
                  flex: '0 0 auto',
                }}
              >
                <Sym name="edit" size={16} color={NEUTRAL.faint} />
              </ButtonBase>
            ) : null}
            {canFin ? (
              <ButtonBase
                onClick={() => app.toggleContribution(c.id)}
                sx={{
                  border: 'none',
                  cursor: 'pointer',
                  borderRadius: '9px',
                  p: '7px 12px',
                  fontSize: '12px',
                  fontWeight: 600,
                  background: paid ? NEUTRAL.successBg : '#FFE5B8',
                  color: paid ? NEUTRAL.success : NEUTRAL.warn,
                  display: 'flex',
                  alignItems: 'center',
                  gap: '5px',
                }}
              >
                <Sym
                  name={paid ? 'check_circle' : 'schedule'}
                  size={15}
                  color={paid ? NEUTRAL.success : NEUTRAL.warn}
                />
                {paid ? t('finances.contribPaid') : t('finances.contribOpen')}
              </ButtonBase>
            ) : (
              <Chip
                label={paid ? t('finances.contribPaid') : t('finances.contribOpen')}
                color={paid ? NEUTRAL.success : NEUTRAL.warn}
                bg={paid ? NEUTRAL.successBg : '#FFE5B8'}
                icon={paid ? 'check_circle' : 'schedule'}
              />
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
