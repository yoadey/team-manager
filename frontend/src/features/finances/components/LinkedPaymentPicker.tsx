import { useState } from 'react';
import Box from '@mui/material/Box';
import ButtonBase from '@mui/material/ButtonBase';
import { buildTokens, fmtMoney, NEUTRAL } from '@/styles/tokens';
import { Sym } from '@/components/ui';
import type { Contribution, PenaltyAssignment } from '../types';
import { ContribLinkMatrixDialog } from './ContribLinkMatrixDialog';
import { PenaltyLinkDialog } from './PenaltyLinkDialog';
import { t } from '@/i18n';

type Tk = ReturnType<typeof buildTokens>;

interface LinkedPaymentPickerProps {
  tk: Tk;
  /** Already filtered to fees that aren't fully paid yet. */
  contributions: Contribution[];
  /** Already filtered to fines that aren't fully paid yet. */
  assignments: PenaltyAssignment[];
  contributionId: string | undefined;
  penaltyAssignmentId: string | undefined;
  onSelectContribution: (id: string) => void;
  onSelectPenalty: (id: string) => void;
  onClear: () => void;
}

/**
 * Single-step linking control for the transaction form's "Verknüpfen mit"
 * section -- a heading plus two direct buttons (Beiträge / Strafen), each
 * opening its own popup, mirroring the always-visible "Betrag" field above
 * it instead of hiding behind a collapsed toggle (see
 * openspec/changes/finance-matrix-transactions).
 */
export function LinkedPaymentPicker({
  tk,
  contributions,
  assignments,
  contributionId,
  penaltyAssignmentId,
  onSelectContribution,
  onSelectPenalty,
  onClear,
}: LinkedPaymentPickerProps) {
  const [matrixOpen, setMatrixOpen] = useState(false);
  const [penaltyOpen, setPenaltyOpen] = useState(false);

  if (!contributions.length && !assignments.length) return null;

  const selectedContrib = contributionId ? contributions.find((c) => c.id === contributionId) : undefined;
  const selectedAssignment = penaltyAssignmentId ? assignments.find((a) => a.id === penaltyAssignmentId) : undefined;
  const selected = selectedContrib || selectedAssignment;

  const dialogs = (
    <>
      <ContribLinkMatrixDialog
        tk={tk}
        open={matrixOpen}
        onClose={() => setMatrixOpen(false)}
        contributions={contributions}
        selectedId={contributionId}
        onSelect={(id) => onSelectContribution(id)}
      />
      <PenaltyLinkDialog
        tk={tk}
        open={penaltyOpen}
        onClose={() => setPenaltyOpen(false)}
        assignments={assignments}
        selectedId={penaltyAssignmentId}
        onSelect={(id) => onSelectPenalty(id)}
      />
    </>
  );

  if (selected) {
    const name = selectedContrib ? selectedContrib.name : selectedAssignment?.name;
    const label = selectedContrib ? selectedContrib.label : selectedAssignment?.label;
    const owed = selectedContrib
      ? selectedContrib.amount - selectedContrib.paidAmount
      : (selectedAssignment?.amount ?? 0) - (selectedAssignment?.paidAmount ?? 0);
    return (
      <Box>
        <Box sx={{ fontSize: '13px', fontWeight: 700, mb: '8px' }}>{t('finances.linkedPickerTitle')}</Box>
        <Box
          sx={{
            display: 'flex',
            alignItems: 'center',
            gap: '10px',
            p: '10px 12px',
            borderRadius: '12px',
            border: `1.5px solid ${tk.primary}`,
            background: tk.primaryContainer,
          }}
        >
          <Sym name="link" size={18} color={tk.onPrimaryContainer} />
          <Box sx={{ flex: 1, minWidth: 0 }}>
            <Box
              sx={{
                fontSize: '13px',
                fontWeight: 700,
                color: tk.onPrimaryContainer,
                whiteSpace: 'nowrap',
                overflow: 'hidden',
                textOverflow: 'ellipsis',
              }}
            >
              {name} · {label}
            </Box>
            <Box sx={{ fontSize: '11px', color: tk.onPrimaryContainer }}>{fmtMoney(owed)}</Box>
          </Box>
          <ButtonBase
            type="button"
            onClick={() => (selectedContrib ? setMatrixOpen(true) : setPenaltyOpen(true))}
            sx={{ fontSize: '12px', fontWeight: 600, color: tk.onPrimaryContainer, p: '5px 8px', borderRadius: '8px' }}
          >
            {t('finances.linkedPickerChange')}
          </ButtonBase>
          <ButtonBase
            type="button"
            aria-label={t('finances.linkedPickerRemove')}
            onClick={onClear}
            sx={{ width: '26px', height: '26px', borderRadius: '8px', color: tk.onPrimaryContainer }}
          >
            <Sym name="close" size={16} color={tk.onPrimaryContainer} />
          </ButtonBase>
        </Box>
        {dialogs}
      </Box>
    );
  }

  return (
    <Box>
      <Box sx={{ fontSize: '13px', fontWeight: 700, mb: '8px' }}>{t('finances.linkedPickerTitle')}</Box>
      <Box sx={{ display: 'flex', gap: '8px' }}>
        <ButtonBase
          type="button"
          onClick={() => setMatrixOpen(true)}
          sx={{
            flex: 1,
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            gap: '8px',
            p: '11px',
            borderRadius: '11px',
            border: `1.5px solid ${NEUTRAL.line3}`,
            color: NEUTRAL.onSurfaceVariant,
            fontWeight: 700,
            fontSize: '13px',
            cursor: 'pointer',
          }}
        >
          <Sym name="payments" size={17} color={NEUTRAL.onSurfaceVariant} />
          {t('finances.linkedPickerKindContrib')}
        </ButtonBase>
        <ButtonBase
          type="button"
          onClick={() => setPenaltyOpen(true)}
          sx={{
            flex: 1,
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            gap: '8px',
            p: '11px',
            borderRadius: '11px',
            border: `1.5px solid ${NEUTRAL.line3}`,
            color: NEUTRAL.onSurfaceVariant,
            fontWeight: 700,
            fontSize: '13px',
            cursor: 'pointer',
          }}
        >
          <Sym name="gavel" size={17} color={NEUTRAL.onSurfaceVariant} />
          {t('finances.linkedPickerKindPenalty')}
        </ButtonBase>
      </Box>
      {dialogs}
    </Box>
  );
}
