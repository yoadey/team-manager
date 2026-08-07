import { useMemo, useState } from 'react';
import Box from '@mui/material/Box';
import ButtonBase from '@mui/material/ButtonBase';
import { buildTokens, fmtMoney, NEUTRAL } from '@/styles/tokens';
import { Av, Sym, TextInput } from '@/components/ui';
import type { Contribution, PenaltyAssignment } from '../types';
import { ContribLinkMatrixDialog } from './ContribLinkMatrixDialog';
import { getIntlLocale, t } from '@/i18n';

type Tk = ReturnType<typeof buildTokens>;
type Kind = 'contribution' | 'penalty';

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
 * A collapsed-by-default, searchable picker for linking a new income
 * transaction to a membership fee or a penalty assignment. Filters the
 * already-fetched overview's open contributions/assignments client-side
 * (no second round-trip) so it stays usable at real-club scale (e.g. 40
 * members x 20 fees) instead of listing every member x fee combination.
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
  const [expanded, setExpanded] = useState(false);
  const [kind, setKind] = useState<Kind>('contribution');
  const [query, setQuery] = useState('');
  const [matrixOpen, setMatrixOpen] = useState(false);

  const selectedContrib = contributionId ? contributions.find((c) => c.id === contributionId) : undefined;
  const selectedAssignment = penaltyAssignmentId ? assignments.find((a) => a.id === penaltyAssignmentId) : undefined;

  const q = query.trim().toLowerCase();
  const filteredAssignments = useMemo(
    () =>
      assignments
        .filter((a) => !q || (a.name ?? '').toLowerCase().includes(q) || (a.label ?? '').toLowerCase().includes(q))
        .sort(
          (a, b) =>
            (a.name ?? '').localeCompare(b.name ?? '', getIntlLocale()) ||
            (a.label ?? '').localeCompare(b.label ?? '', getIntlLocale()),
        ),
    [assignments, q],
  );

  if (!contributions.length && !assignments.length) return null;

  const selected = selectedContrib || selectedAssignment;
  if (selected && !expanded) {
    const name = selectedContrib ? selectedContrib.name : selectedAssignment?.name;
    const label = selectedContrib ? selectedContrib.label : selectedAssignment?.label;
    const open = selectedContrib
      ? selectedContrib.amount - selectedContrib.paidAmount
      : (selectedAssignment?.amount ?? 0) - (selectedAssignment?.paidAmount ?? 0);
    return (
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
          <Box sx={{ fontSize: '11px', color: tk.onPrimaryContainer }}>{fmtMoney(open)}</Box>
        </Box>
        <ButtonBase
          type="button"
          onClick={() => setExpanded(true)}
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
    );
  }

  if (!expanded) {
    return (
      <ButtonBase
        type="button"
        onClick={() => setExpanded(true)}
        sx={{
          display: 'flex',
          alignItems: 'center',
          gap: '8px',
          p: '10px 12px',
          borderRadius: '12px',
          border: `1px dashed ${NEUTRAL.inputBorder}`,
          color: NEUTRAL.secondary,
          fontSize: '13px',
          fontWeight: 600,
          justifyContent: 'flex-start',
        }}
      >
        <Sym name="link" size={17} color={NEUTRAL.secondary} />
        {t('finances.linkedPickerOpen')}
      </ButtonBase>
    );
  }

  const kindDefs: [Kind, string, number][] = [
    ['contribution', t('finances.linkedPickerKindContrib'), contributions.length],
    ['penalty', t('finances.linkedPickerKindPenalty'), assignments.length],
  ];

  return (
    <Box
      sx={{
        display: 'flex',
        flexDirection: 'column',
        gap: '10px',
        p: '12px',
        borderRadius: '14px',
        border: `1.5px solid ${tk.primary}`,
        background: NEUTRAL.card,
      }}
    >
      <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
        <Box sx={{ fontSize: '13px', fontWeight: 700 }}>{t('finances.linkedPickerTitle')}</Box>
        <ButtonBase
          type="button"
          onClick={() => setExpanded(false)}
          aria-label={t('common.close')}
          sx={{ width: '26px', height: '26px', borderRadius: '8px', color: NEUTRAL.faint }}
        >
          <Sym name="close" size={16} color={NEUTRAL.faint} />
        </ButtonBase>
      </Box>
      <Box sx={{ display: 'flex', gap: '8px' }}>
        {kindDefs.map(([v, l, count]) => {
          const sel = kind === v;
          return (
            <ButtonBase
              key={v}
              type="button"
              onClick={() => setKind(v)}
              aria-pressed={sel}
              sx={{
                flex: 1,
                p: '8px',
                borderRadius: '10px',
                fontSize: '12px',
                fontWeight: 700,
                border: '1.5px solid ' + (sel ? tk.primary : NEUTRAL.line3),
                background: sel ? tk.primaryContainer : NEUTRAL.card,
                color: sel ? tk.onPrimaryContainer : NEUTRAL.secondary,
              }}
            >
              {l} ({count})
            </ButtonBase>
          );
        })}
      </Box>
      {kind === 'contribution' ? (
        <>
          <ButtonBase
            type="button"
            onClick={() => setMatrixOpen(true)}
            sx={{
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              gap: '8px',
              p: '11px',
              borderRadius: '11px',
              border: `1.5px solid ${tk.primary}`,
              color: tk.primary,
              fontWeight: 700,
              fontSize: '13px',
              cursor: 'pointer',
            }}
          >
            <Sym name="grid_on" size={17} color={tk.primary} />
            {t('finances.linkedPickerMatrixOpen')}
          </ButtonBase>
          <ContribLinkMatrixDialog
            tk={tk}
            open={matrixOpen}
            onClose={() => setMatrixOpen(false)}
            contributions={contributions}
            selectedId={contributionId}
            onSelect={(id) => {
              onSelectContribution(id);
              setExpanded(false);
            }}
          />
        </>
      ) : (
        <>
          <TextInput
            name="linkedPaymentSearch"
            placeholder={t('finances.linkedPickerSearchPlaceholder')}
            value={query}
            onChange={(e) => setQuery(e.target.value)}
          />
          <Box role="listbox" sx={{ display: 'flex', flexDirection: 'column', gap: '6px', maxHeight: '260px', overflowY: 'auto' }}>
            {filteredAssignments.length ? (
              filteredAssignments.map((a) => {
                const open = (a.amount ?? 0) - a.paidAmount;
                return (
                  <ButtonBase
                    key={a.id}
                    type="button"
                    role="option"
                    onClick={() => {
                      onSelectPenalty(a.id);
                      setExpanded(false);
                      setQuery('');
                    }}
                    sx={{
                      display: 'flex',
                      alignItems: 'center',
                      gap: '10px',
                      p: '8px 10px',
                      borderRadius: '10px',
                      textAlign: 'left',
                      justifyContent: 'flex-start',
                      border: `1px solid ${NEUTRAL.line3}`,
                    }}
                  >
                    <Av name={a.name} photo={a.photo} color={a.avatarColor} size={28} />
                    <Box sx={{ flex: 1, minWidth: 0 }}>
                      <Box sx={{ fontSize: '13px', fontWeight: 600, whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis' }}>
                        {a.name}
                      </Box>
                      <Box sx={{ fontSize: '11px', color: NEUTRAL.faint, whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis' }}>
                        {a.label}
                      </Box>
                    </Box>
                    <Box component="span" sx={{ fontSize: '12px', fontWeight: 700, color: NEUTRAL.secondary, flex: '0 0 auto' }}>
                      {fmtMoney(open)}
                    </Box>
                  </ButtonBase>
                );
              })
            ) : (
              <Box sx={{ fontSize: '12px', color: NEUTRAL.faint, textAlign: 'center', p: '12px' }}>
                {t('finances.linkedPickerEmpty')}
              </Box>
            )}
          </Box>
        </>
      )}
    </Box>
  );
}
