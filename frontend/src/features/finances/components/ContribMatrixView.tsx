import Box from '@mui/material/Box';
import ButtonBase from '@mui/material/ButtonBase';
import { useApp } from '@/context/AppContext';
import { fmtDate, fmtMoney, NEUTRAL } from '@/styles/tokens';
import { Av, EmptyState, Sym } from '@/components/ui';
import type { Contribution } from '../types';
import { contributionAmountStatus, type ContributionAmountStatus } from '../contributionStatus';
import { groupKey } from './FinancesContributions';
import { getIntlLocale, t } from '@/i18n';

type App = ReturnType<typeof useApp>;

interface Props {
  app: App;
  /** Already filtered to non-archived contributions -- the matrix never shows archived fee periods. */
  contributions: Contribution[];
}

// Glyph + colour for each cell state -- "overpaid" reuses the same green as
// "paid" (it IS paid, plus a credit) but a distinct glyph so the two don't
// look identical at a glance.
const CELL_META: Record<ContributionAmountStatus, { icon: string; color: string }> = {
  open: { icon: 'schedule', color: NEUTRAL.warn },
  partial: { icon: 'incomplete_circle', color: NEUTRAL.faint },
  paid: { icon: 'check', color: NEUTRAL.success },
  overpaid: { icon: 'savings', color: NEUTRAL.success },
};

/** Member x fee-group grid, mirroring Stats.tsx's attendance MatrixView. */
export function ContribMatrixView({ app, contributions }: Props) {
  if (!contributions.length) return <EmptyState icon="grid_on" text={t('finances.contribMatrixEmpty')} />;

  const groupLabel: Record<string, string> = {};
  const groupDueDate: Record<string, string | null> = {};
  contributions.forEach((c) => {
    const key = groupKey(c);
    groupLabel[key] = c.label;
    groupDueDate[key] = c.dueDate || null;
  });
  const groups = [...new Set(contributions.map(groupKey))].sort((a, b) => {
    const da = groupDueDate[a];
    const db = groupDueDate[b];
    if (da && db) return da.localeCompare(db) || groupLabel[a]!.localeCompare(groupLabel[b]!, getIntlLocale());
    if (da) return -1;
    if (db) return 1;
    return groupLabel[a]!.localeCompare(groupLabel[b]!, getIntlLocale());
  });

  const memberIds = [...new Set(contributions.map((c) => c.userId))];
  const memberMeta: Record<string, { name: string; avatarColor: string | undefined; photo: string | null | undefined }> = {};
  contributions.forEach((c) => {
    if (!memberMeta[c.userId]) memberMeta[c.userId] = { name: c.name || '', avatarColor: c.avatarColor, photo: c.photo };
  });
  const members = memberIds
    .map((id) => ({ userId: id, ...memberMeta[id]! }))
    .sort((a, b) => a.name.localeCompare(b.name, getIntlLocale()));

  // cells[userId][group] -- assumes at most one contribution per member per
  // fee-group, the normal case (a treasurer wouldn't create the same named
  // fee with the same due date for the same member twice).
  const cells: Record<string, Record<string, Contribution>> = {};
  contributions.forEach((c) => {
    const key = groupKey(c);
    cells[c.userId] ??= {};
    cells[c.userId]![key] = c;
  });

  const thBase = {
    borderBottom: `1px solid ${NEUTRAL.line}`,
    padding: '8px 6px',
    fontSize: '12px',
    fontWeight: 700,
    color: NEUTRAL.secondary,
    background: NEUTRAL.card,
  } as const;
  const nameColSx = {
    position: 'sticky',
    left: 0,
    zIndex: 1,
    background: NEUTRAL.card,
    textAlign: 'left',
    minWidth: '150px',
    boxShadow: `1px 0 0 ${NEUTRAL.line}`,
  } as const;

  return (
    <Box sx={{ overflowX: 'auto', border: `1px solid ${NEUTRAL.line}`, borderRadius: '14px' }}>
      <Box component="table" sx={{ borderCollapse: 'collapse', width: '100%', tableLayout: 'auto' }}>
        <Box component="thead">
          <Box component="tr">
            <Box component="th" scope="col" sx={{ ...thBase, ...nameColSx }}>
              {t('finances.contribMatrixMemberHeader')}
            </Box>
            {groups.map((g) => {
              const due = groupDueDate[g];
              return (
                <Box
                  key={g}
                  component="th"
                  scope="col"
                  title={due ? groupLabel[g] + ' · ' + fmtDate(due) : groupLabel[g]}
                  sx={{ ...thBase, textAlign: 'center', minWidth: '56px', maxWidth: '90px' }}
                >
                  <Box
                    sx={{
                      whiteSpace: 'nowrap',
                      overflow: 'hidden',
                      textOverflow: 'ellipsis',
                    }}
                  >
                    {groupLabel[g]}
                  </Box>
                  {due ? <Box sx={{ fontSize: '10px', color: NEUTRAL.faint, fontWeight: 500 }}>{fmtDate(due)}</Box> : null}
                </Box>
              );
            })}
          </Box>
        </Box>
        <Box component="tbody">
          {members.map((m) => (
            <Box component="tr" key={m.userId}>
              <Box
                component="th"
                scope="row"
                sx={{
                  ...nameColSx,
                  borderBottom: `1px solid ${NEUTRAL.line}`,
                  padding: '6px 8px',
                  fontWeight: 600,
                  fontSize: '13px',
                }}
              >
                <Box sx={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
                  <Av name={m.name} photo={m.photo} color={m.avatarColor} size={26} />
                  <Box sx={{ whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis' }}>{m.name}</Box>
                </Box>
              </Box>
              {groups.map((g) => {
                const c = cells[m.userId]?.[g];
                if (!c) {
                  return (
                    <Box key={g} component="td" sx={{ borderBottom: `1px solid ${NEUTRAL.line}`, textAlign: 'center', padding: '6px' }}>
                      <Sym name="remove" size={16} color={NEUTRAL.line3} label={t('finances.contribMatrixCellAria', { name: m.name, group: groupLabel[g]!, status: '' })} />
                    </Box>
                  );
                }
                const info = contributionAmountStatus(c.amount, c.paidAmount);
                const cm = CELL_META[info.status];
                const amountLabel =
                  info.status === 'overpaid'
                    ? t('finances.contribOverpaidAmount', { amount: fmtMoney(info.excess) })
                    : fmtMoney(info.displayAmount) + ' / ' + fmtMoney(c.amount);
                return (
                  <Box
                    key={g}
                    component="td"
                    title={amountLabel}
                    sx={{ borderBottom: `1px solid ${NEUTRAL.line}`, textAlign: 'center', padding: '6px' }}
                  >
                    <ButtonBase
                      type="button"
                      onClick={() => app.openContribDetail(c)}
                      sx={{ borderRadius: '8px', p: '2px' }}
                    >
                      <Sym
                        name={cm.icon}
                        size={18}
                        color={cm.color}
                        label={t('finances.contribMatrixCellAria', { name: m.name, group: groupLabel[g]!, status: amountLabel })}
                      />
                    </ButtonBase>
                  </Box>
                );
              })}
            </Box>
          ))}
        </Box>
      </Box>
    </Box>
  );
}
