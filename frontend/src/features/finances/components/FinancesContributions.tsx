import Box from '@mui/material/Box';
import ButtonBase from '@mui/material/ButtonBase';
import { useApp } from '@/context/AppContext';
import { buildTokens, fmtDate, fmtMoney, NEUTRAL } from '@/styles/tokens';
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

  const header = (
    <Box key="hd" sx={{ display: 'flex', justifyContent: 'flex-end', mb: '14px' }}>
      {canFin ? (
        <ButtonBase
          onClick={() => app.openContribCreate()}
          sx={{
            display: 'flex',
            alignItems: 'center',
            gap: '6px',
            border: 'none',
            background: tk.primary,
            color: tk.onPrimary,
            borderRadius: '12px',
            p: '10px 14px',
            fontSize: '13px',
            fontWeight: 700,
            cursor: 'pointer',
          }}
        >
          <Sym name="add" size={17} color={tk.onPrimary} />
          {t('finances.contribCreateBtn')}
        </ButtonBase>
      ) : null}
    </Box>
  );

  if (!contribs.length) {
    return (
      <Box key="bei">
        {header}
        <EmptyState icon="payments" text={t('finances.contribEmpty')} />
      </Box>
    );
  }

  // Groups by fee name (the batch a create action fanned out into) --
  // soonest due date first, groups without a due date sort last, ties
  // broken alphabetically.
  const groupDueDate: Record<string, string | null> = {};
  contribs.forEach((c) => {
    const cur = groupDueDate[c.label];
    if (cur === undefined || (c.dueDate && (!cur || c.dueDate < cur))) groupDueDate[c.label] = c.dueDate || cur || null;
  });
  const groups = [...new Set(contribs.map((c) => c.label))].sort((a, b) => {
    const da = groupDueDate[a];
    const db = groupDueDate[b];
    if (da && db) return da.localeCompare(db);
    if (da) return -1;
    if (db) return 1;
    return a.localeCompare(b, getIntlLocale());
  });
  // groups.length > 0 is guaranteed by the `!contribs.length` early return above.
  const sel = state.contribGroup && groups.includes(state.contribGroup) ? state.contribGroup : groups[0]!;
  const rows = contribs
    .filter((c) => c.label === sel)
    .sort((a, b) => (a.name ?? '').localeCompare(b.name ?? '', getIntlLocale()));
  const paidRows = rows.filter((c) => c.status === 'paid');
  const sum = paidRows.reduce((s, c) => s + c.paidAmount, 0);
  const total = rows.reduce((s, c) => s + c.amount, 0);
  const pct = rows.length ? Math.round((paidRows.length / rows.length) * 100) : 0;

  const groupChips = (
    <Box key="mc" sx={{ display: 'flex', gap: '8px', overflowX: 'auto', pb: '4px', mb: '14px' }}>
      {groups.map((g) => {
        const s = g === sel;
        const open = contribs.filter((c) => c.label === g && c.status !== 'paid').length;
        return (
          <ButtonBase
            key={g}
            onClick={() => app.setState({ contribGroup: g })}
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
              maxWidth: '220px',
            }}
          >
            <Box
              component="span"
              sx={{
                fontSize: '13px',
                fontWeight: 700,
                whiteSpace: 'nowrap',
                overflow: 'hidden',
                textOverflow: 'ellipsis',
                maxWidth: '100%',
              }}
            >
              {g}
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
  const dueDate = groupDueDate[sel];
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
        <Box sx={{ fontSize: '15px', fontWeight: 700, overflow: 'hidden', textOverflow: 'ellipsis' }}>{sel}</Box>
        <Box sx={{ fontSize: '13px', color: NEUTRAL.secondary, mt: '3px' }}>
          {t('finances.contribSummary', {
            paid: paidRows.length,
            total: rows.length,
            paidAmt: fmtMoney(sum),
            totalAmt: fmtMoney(total),
          })}
        </Box>
        {dueDate ? (
          <Box sx={{ fontSize: '12px', color: NEUTRAL.faint, mt: '3px' }}>
            {t('finances.contribDueDate')} {fmtDate(dueDate)}
          </Box>
        ) : null}
      </Box>
      <Box
        sx={{ fontSize: '24px', fontWeight: 800, color: pct === 100 ? NEUTRAL.success : tk.primary, flex: '0 0 auto' }}
      >
        {pct + '%'}
      </Box>
    </Box>
  );
  const statusMeta = (c: Contribution): { label: string; icon: string; color: string; bg: string } => {
    if (c.status === 'paid') return { label: t('finances.contribPaid'), icon: 'check_circle', color: NEUTRAL.success, bg: NEUTRAL.successBg };
    if (c.status === 'partial') return { label: t('finances.contribPartial'), icon: 'incomplete_circle', color: tk.primary, bg: tk.primaryContainer };
    return { label: t('finances.contribOpen'), icon: 'schedule', color: NEUTRAL.warn, bg: '#FFE5B8' };
  };
  const list = (
    <Box key="l" sx={{ display: 'flex', flexDirection: 'column', gap: '8px' }}>
      {rows.map((c: Contribution) => {
        const meta = statusMeta(c);
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
              <Box sx={{ fontSize: '12px', color: NEUTRAL.faint }}>
                {c.status === 'partial' ? fmtMoney(c.paidAmount) + ' / ' + fmtMoney(c.amount) : fmtMoney(c.amount)}
              </Box>
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
            <Chip label={meta.label} color={meta.color} bg={meta.bg} icon={meta.icon} />
          </Box>
        );
      })}
    </Box>
  );
  return (
    <Box key="bei">
      {header}
      {groupChips}
      {summary}
      {list}
    </Box>
  );
}
