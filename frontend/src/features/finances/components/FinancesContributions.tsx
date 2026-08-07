import { useState } from 'react';
import Box from '@mui/material/Box';
import ButtonBase from '@mui/material/ButtonBase';
import { useApp } from '@/context/AppContext';
import { buildTokens, fmtDate, fmtMoney, NEUTRAL } from '@/styles/tokens';
import { Av, Chip, EmptyState, Sym } from '@/components/ui';
import type { Contribution, FinanceOverview } from '../types';
import { contributionAmountStatus } from '../contributionStatus';
import { ContribMatrixView } from './ContribMatrixView';
import { getIntlLocale, t } from '@/i18n';

type App = ReturnType<typeof useApp>;
type Tk = ReturnType<typeof buildTokens>;

interface Props {
  app: App;
  t: Tk;
  f: FinanceOverview;
  canFin: boolean;
}

// Groups by (fee name, due date) -- not name alone, since a treasurer
// creating the same-named recurring fee period after period (there's no
// reusable catalog forcing period-differentiated names, see design.md)
// would otherwise merge unrelated batches into one group, blending their
// paid/total progress and losing each row's individual due date. Two
// batches only ever collapse into the same group when they share both
// name AND due date, which is the actual "same fee, re-touched" case.
export function groupKey(c: Contribution): string {
  return c.dueDate ? c.label + ' ' + c.dueDate : c.label;
}

export function FinancesContributions({ app, t: tk, f, canFin }: Props) {
  const { state } = app;
  const [view, setView] = useState<'list' | 'matrix'>('list');
  const [showArchived, setShowArchived] = useState(false);
  const allContribs = f.contributions || [];
  const contribs = showArchived ? allContribs : allContribs.filter((c) => !c.archived);

  const header = (
    <Box key="hd" sx={{ display: 'flex', justifyContent: 'flex-end', gap: '10px', mb: '14px' }}>
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

  const viewTabs = (
    <Box
      key="vt"
      role="tablist"
      aria-label={t('finances.contribViewLabel')}
      sx={{ display: 'flex', gap: '8px', mb: '14px' }}
    >
      {(['list', 'matrix'] as const).map((v) => {
        const s = view === v;
        return (
          <ButtonBase
            key={v}
            role="tab"
            aria-selected={s}
            onClick={() => setView(v)}
            sx={{
              p: '8px 14px',
              borderRadius: '10px',
              fontSize: '13px',
              fontWeight: 600,
              cursor: 'pointer',
              border: '1.5px solid ' + (s ? tk.primary : NEUTRAL.inputBorder),
              background: s ? tk.primaryContainer : NEUTRAL.card,
              color: s ? tk.onPrimaryContainer : NEUTRAL.onSurfaceVariant,
            }}
          >
            {v === 'list' ? t('finances.contribViewList') : t('finances.contribViewMatrix')}
          </ButtonBase>
        );
      })}
    </Box>
  );

  const archivedToggle = (
    <ButtonBase
      key="arc"
      role="checkbox"
      aria-checked={showArchived}
      onClick={() => setShowArchived((v) => !v)}
      sx={{
        display: 'flex',
        alignItems: 'center',
        gap: '6px',
        p: '6px 10px',
        borderRadius: '10px',
        fontSize: '12px',
        fontWeight: 600,
        color: NEUTRAL.secondary,
        cursor: 'pointer',
        mb: '14px',
      }}
    >
      <Sym
        name={showArchived ? 'check_box' : 'check_box_outline_blank'}
        size={17}
        color={showArchived ? tk.primary : NEUTRAL.faint}
      />
      {t('finances.contribShowArchived')}
    </ButtonBase>
  );

  if (!allContribs.length) {
    return (
      <Box key="bei">
        {header}
        <EmptyState icon="payments" text={t('finances.contribEmpty')} />
      </Box>
    );
  }

  if (view === 'matrix') {
    return (
      <Box key="bei">
        {header}
        {viewTabs}
        <ContribMatrixView contributions={allContribs.filter((c) => !c.archived)} />
      </Box>
    );
  }

  if (!contribs.length) {
    return (
      <Box key="bei">
        {header}
        {viewTabs}
        {archivedToggle}
        <EmptyState icon="payments" text={t('finances.contribEmpty')} />
      </Box>
    );
  }

  // Groups sort soonest-due-first; groups without a due date sort last;
  // ties (including the "no due date" bucket) break alphabetically by name.
  const groupLabel: Record<string, string> = {};
  const groupDueDate: Record<string, string | null> = {};
  contribs.forEach((c) => {
    const key = groupKey(c);
    groupLabel[key] = c.label;
    groupDueDate[key] = c.dueDate || null;
  });
  const groups = [...new Set(contribs.map(groupKey))].sort((a, b) => {
    const da = groupDueDate[a];
    const db = groupDueDate[b];
    if (da && db) return da.localeCompare(db) || groupLabel[a]!.localeCompare(groupLabel[b]!, getIntlLocale());
    if (da) return -1;
    if (db) return 1;
    return groupLabel[a]!.localeCompare(groupLabel[b]!, getIntlLocale());
  });
  // groups.length > 0 is guaranteed by the `!contribs.length` early return above.
  const sel = state.contribGroup && groups.includes(state.contribGroup) ? state.contribGroup : groups[0]!;
  const rows = contribs
    .filter((c) => groupKey(c) === sel)
    .sort((a, b) => (a.name ?? '').localeCompare(b.name ?? '', getIntlLocale()));
  // All rows sharing sel's group key, including archived ones -- used for
  // the bulk archive/un-archive action, which must act on the whole group
  // regardless of the "show archived" toggle's current filtering.
  const allGroupRows = allContribs.filter((c) => groupKey(c) === sel);
  const groupFullyArchived = allGroupRows.length > 0 && allGroupRows.every((c) => c.archived);
  const paidRows = rows.filter((c) => c.status === 'paid');
  const sum = paidRows.reduce((s, c) => s + c.paidAmount, 0);
  const total = rows.reduce((s, c) => s + c.amount, 0);
  const pct = rows.length ? Math.round((paidRows.length / rows.length) * 100) : 0;

  const groupChips = (
    <Box key="mc" sx={{ display: 'flex', gap: '8px', overflowX: 'auto', pb: '4px', mb: '14px' }}>
      {groups.map((g) => {
        const s = g === sel;
        const open = contribs.filter((c) => groupKey(c) === g && c.status !== 'paid').length;
        const statusText = open ? open + ' ' + t('finances.contribOpen') : t('finances.contribPaid');
        const due = groupDueDate[g];
        // Only reachable with the "show archived" toggle on -- a fully
        // archived group is otherwise filtered out of `groups` entirely.
        const archivedGroup = contribs.filter((c) => groupKey(c) === g).every((c) => c.archived);
        // Two groups can share a name (a recurring fee re-created for a new
        // period) and differ only by due date -- show it so the chips stay
        // distinguishable instead of rendering as identical-looking twins.
        const secondaryText =
          (archivedGroup ? t('finances.contribArchivedChip') + ' · ' : '') +
          (due ? statusText + ' · ' + fmtDate(due) : statusText);
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
              {groupLabel[g]}
            </Box>
            <Box
              component="span"
              sx={{ fontSize: '11px', color: s ? tk.onPrimaryContainer : NEUTRAL.faint, whiteSpace: 'nowrap' }}
            >
              {secondaryText}
            </Box>
          </ButtonBase>
        );
      })}
    </Box>
  );
  const dueDate = groupDueDate[sel];
  const archiveGroupAction = canFin ? (
    <ButtonBase
      key="arcg"
      onClick={() =>
        app.askConfirm({
          title: t(groupFullyArchived ? 'finances.contribUnarchiveGroupConfirmTitle' : 'finances.contribArchiveGroupConfirmTitle'),
          message: t(groupFullyArchived ? 'finances.contribUnarchiveGroupConfirmMsg' : 'finances.contribArchiveGroupConfirmMsg'),
          confirmLabel: t(groupFullyArchived ? 'finances.contribUnarchiveGroupBtn' : 'finances.contribArchiveGroupBtn'),
          onConfirm: () => app.archiveContribGroup(allGroupRows, !groupFullyArchived),
        })
      }
      sx={{
        display: 'flex',
        alignItems: 'center',
        gap: '6px',
        p: '8px 10px',
        borderRadius: '10px',
        border: `1px solid ${NEUTRAL.line}`,
        color: NEUTRAL.secondary,
        fontSize: '12px',
        fontWeight: 600,
        cursor: 'pointer',
        flex: '0 0 auto',
      }}
    >
      <Sym name={groupFullyArchived ? 'unarchive' : 'archive'} size={16} color={NEUTRAL.secondary} />
      {t(groupFullyArchived ? 'finances.contribUnarchiveGroupBtn' : 'finances.contribArchiveGroupBtn')}
    </ButtonBase>
  ) : null;
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
        <Box sx={{ fontSize: '15px', fontWeight: 700, overflow: 'hidden', textOverflow: 'ellipsis' }}>{groupLabel[sel]}</Box>
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
      {archiveGroupAction}
    </Box>
  );
  const statusMeta = (c: Contribution): { label: string; icon: string; color: string; bg: string } => {
    const info = contributionAmountStatus(c.amount, c.paidAmount);
    if (info.status === 'overpaid') return { label: t('finances.contribOverpaid'), icon: 'savings', color: NEUTRAL.success, bg: NEUTRAL.successBg };
    if (info.status === 'paid') return { label: t('finances.contribPaid'), icon: 'check_circle', color: NEUTRAL.success, bg: NEUTRAL.successBg };
    if (info.status === 'partial') return { label: t('finances.contribPartial'), icon: 'incomplete_circle', color: tk.primary, bg: tk.primaryContainer };
    return { label: t('finances.contribOpen'), icon: 'schedule', color: NEUTRAL.warn, bg: '#FFE5B8' };
  };
  const list = (
    <Box key="l" sx={{ display: 'flex', flexDirection: 'column', gap: '8px' }}>
      {rows.map((c: Contribution) => {
        const meta = statusMeta(c);
        const info = contributionAmountStatus(c.amount, c.paidAmount);
        const amountText =
          info.status === 'overpaid'
            ? fmtMoney(info.displayAmount) + ' (' + t('finances.contribOverpaidAmount', { amount: fmtMoney(info.excess) }) + ')'
            : info.status === 'partial'
              ? fmtMoney(c.paidAmount) + ' / ' + fmtMoney(c.amount)
              : fmtMoney(c.amount);
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
                title={c.description || undefined}
              >
                {c.name}
              </Box>
              <Box sx={{ fontSize: '12px', color: NEUTRAL.faint }}>{amountText}</Box>
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
      {viewTabs}
      {archivedToggle}
      {groupChips}
      {summary}
      {list}
    </Box>
  );
}
