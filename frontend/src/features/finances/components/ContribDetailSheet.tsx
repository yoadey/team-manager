import Box from '@mui/material/Box';
import { fmtMoney, NEUTRAL } from '@/styles/tokens';
import { Av, EmptyState, PrimaryButton } from '@/components/ui';
import type { SheetProps } from '@/sheets/types';
import { LinkedTransactionsList } from './LinkedTransactionsList';
import { useFinanceOverviewQuery } from '../hooks/useFinanceQueries';
import { t } from '@/i18n';

/**
 * Read-only detail view for a single member's contribution row -- editing
 * label/amount/description/dueDate happens exclusively through the
 * group-level `ContribGroupEditSheet`, never here. See
 * openspec/changes/contribution-detail-readonly-parent-edit.
 */
export function ContribDetailSheet({ app, sheet }: SheetProps) {
  const { state } = app;
  const { data: finances } = useFinanceOverviewQuery(app.api, state.activeTeamId);
  const contribId = (sheet.formInitial as { id: string }).id;
  const contribution = (finances?.contributions || []).find((c) => c.id === contribId);
  const linkedTx = ((finances && finances.transactions) || []).filter((tx) => tx.contributionId === contribId);

  if (!contribution) return <EmptyState icon="payments" text={t('finances.contribEmpty')} />;

  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: '16px' }}>
      {contribution.archived ? (
        <Box
          sx={{
            p: '10px 12px',
            borderRadius: '12px',
            background: NEUTRAL.sidebar,
            color: NEUTRAL.secondary,
            fontSize: '12px',
            fontWeight: 600,
          }}
        >
          {t('finances.contribArchivedNotice')}
        </Box>
      ) : null}
      <Box sx={{ display: 'flex', alignItems: 'center', gap: '11px' }}>
        <Av name={contribution.name} photo={contribution.photo} color={contribution.avatarColor} size={40} />
        <Box sx={{ minWidth: 0 }}>
          <Box sx={{ fontSize: '15px', fontWeight: 700, overflow: 'hidden', textOverflow: 'ellipsis' }}>
            {contribution.name}
          </Box>
          <Box sx={{ fontSize: '12px', color: NEUTRAL.faint }}>{contribution.label}</Box>
        </Box>
      </Box>
      <Box
        sx={{
          display: 'flex',
          justifyContent: 'space-between',
          background: NEUTRAL.card,
          border: `1px solid ${NEUTRAL.line}`,
          borderRadius: '14px',
          p: '12px 14px',
        }}
      >
        <Box>
          <Box sx={{ fontSize: '11px', color: NEUTRAL.faint, fontWeight: 600 }}>{t('finances.contribDetailPaid')}</Box>
          <Box sx={{ fontSize: '16px', fontWeight: 700 }}>{fmtMoney(contribution.paidAmount)}</Box>
        </Box>
        <Box sx={{ textAlign: 'right' }}>
          <Box sx={{ fontSize: '11px', color: NEUTRAL.faint, fontWeight: 600 }}>
            {t('finances.contribDetailRequired')}
          </Box>
          <Box sx={{ fontSize: '16px', fontWeight: 700 }}>{fmtMoney(contribution.amount)}</Box>
        </Box>
      </Box>
      <LinkedTransactionsList transactions={linkedTx} onSelect={(tx) => app.openTxForm(tx)} />
      <PrimaryButton
        label={t('finances.contribRecordPayment')}
        onClick={() => app.openTxFormForContribution(contribution)}
      />
    </Box>
  );
}
