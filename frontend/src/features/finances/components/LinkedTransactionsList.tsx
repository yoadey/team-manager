import Box from '@mui/material/Box';
import ButtonBase from '@mui/material/ButtonBase';
import { fmtDate, fmtMoney, NEUTRAL } from '@/styles/tokens';
import { Sym } from '@/components/ui';
import type { Transaction } from '../types';
import { t } from '@/i18n';

interface Props {
  transactions: Transaction[];
  onSelect: (tx: Transaction) => void;
}

/** Read-only list of transactions linked to a contribution or penalty
 * assignment, shown in that entry's detail view -- see
 * openspec/changes/finance-detail-linked-entries. */
export function LinkedTransactionsList({ transactions, onSelect }: Props) {
  return (
    <Box>
      <Box sx={{ fontSize: '12px', fontWeight: 700, color: NEUTRAL.secondary, mb: '8px' }}>
        {t('finances.linkedTxTitle')}
      </Box>
      {transactions.length ? (
        <Box sx={{ display: 'flex', flexDirection: 'column', gap: '6px' }}>
          {transactions.map((tx) => (
            <ButtonBase
              key={tx.id}
              type="button"
              onClick={() => onSelect(tx)}
              sx={{
                display: 'flex',
                alignItems: 'center',
                gap: '10px',
                p: '9px 11px',
                borderRadius: '11px',
                border: `1px solid ${NEUTRAL.line}`,
                background: NEUTRAL.card,
                cursor: 'pointer',
                textAlign: 'left',
                justifyContent: 'flex-start',
              }}
            >
              <Sym name="receipt_long" size={16} color={NEUTRAL.secondary} />
              <Box sx={{ flex: 1, minWidth: 0 }}>
                <Box sx={{ fontSize: '13px', fontWeight: 600, overflow: 'hidden', textOverflow: 'ellipsis' }}>
                  {tx.title}
                </Box>
                <Box sx={{ fontSize: '11px', color: NEUTRAL.faint }}>{fmtDate(tx.date)}</Box>
              </Box>
              <Box component="span" sx={{ fontSize: '13px', fontWeight: 700, color: NEUTRAL.onSurfaceVariant }}>
                {fmtMoney(tx.amount)}
              </Box>
            </ButtonBase>
          ))}
        </Box>
      ) : (
        <Box sx={{ fontSize: '12px', color: NEUTRAL.faint, fontStyle: 'italic' }}>{t('finances.linkedTxEmpty')}</Box>
      )}
    </Box>
  );
}
