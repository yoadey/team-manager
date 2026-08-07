import { useMemo, useState } from 'react';
import Box from '@mui/material/Box';
import ButtonBase from '@mui/material/ButtonBase';
import Dialog from '@mui/material/Dialog';
import { buildTokens, fmtMoney, NEUTRAL } from '@/styles/tokens';
import { Av, EmptyState, Sym, TextInput } from '@/components/ui';
import type { PenaltyAssignment } from '../types';
import { getIntlLocale, t } from '@/i18n';

type Tk = ReturnType<typeof buildTokens>;

interface Props {
  tk: Tk;
  open: boolean;
  onClose: () => void;
  /** Already filtered to fines that aren't fully paid yet. */
  assignments: PenaltyAssignment[];
  selectedId: string | undefined;
  onSelect: (id: string) => void;
}

/**
 * Popup search+list for linking a new income transaction to a penalty
 * assignment -- the "Strafen" counterpart of ContribLinkMatrixDialog, opened
 * directly from LinkedPaymentPicker's "Strafen" button (see
 * openspec/changes/finance-matrix-transactions).
 */
export function PenaltyLinkDialog({ tk, open, onClose, assignments, selectedId, onSelect }: Props) {
  const [query, setQuery] = useState('');

  const q = query.trim().toLowerCase();
  const filtered = useMemo(
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

  return (
    <Dialog open={open} onClose={onClose} maxWidth="sm" fullWidth>
      <Box sx={{ p: '18px', display: 'flex', flexDirection: 'column', gap: '12px' }}>
        <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
          <Box sx={{ fontSize: '15px', fontWeight: 700 }}>{t('finances.penaltyLinkTitle')}</Box>
          <ButtonBase
            type="button"
            onClick={onClose}
            aria-label={t('shell.close')}
            sx={{ width: '30px', height: '30px', borderRadius: '9px', color: NEUTRAL.faint }}
          >
            <Sym name="close" size={18} color={NEUTRAL.faint} />
          </ButtonBase>
        </Box>
        <TextInput
          name="penaltyLinkSearch"
          placeholder={t('finances.penaltyLinkSearchPlaceholder')}
          value={query}
          onChange={(e) => setQuery(e.target.value)}
        />
        <Box role="listbox" sx={{ display: 'flex', flexDirection: 'column', gap: '6px', maxHeight: '60vh', overflowY: 'auto' }}>
          {filtered.length ? (
            filtered.map((a) => {
              const open = (a.amount ?? 0) - a.paidAmount;
              const sel = a.id === selectedId;
              return (
                <ButtonBase
                  key={a.id}
                  type="button"
                  role="option"
                  aria-selected={sel}
                  onClick={() => {
                    onSelect(a.id);
                    onClose();
                  }}
                  sx={{
                    display: 'flex',
                    alignItems: 'center',
                    gap: '10px',
                    p: '8px 10px',
                    borderRadius: '10px',
                    textAlign: 'left',
                    justifyContent: 'flex-start',
                    border: '1.5px solid ' + (sel ? tk.primary : NEUTRAL.line3),
                    background: sel ? tk.primaryContainer : NEUTRAL.card,
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
            <EmptyState icon="gavel" text={t('finances.penaltyLinkEmpty')} />
          )}
        </Box>
      </Box>
    </Dialog>
  );
}
