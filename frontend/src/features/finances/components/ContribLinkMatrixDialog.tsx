import { useMemo, useState } from 'react';
import Box from '@mui/material/Box';
import ButtonBase from '@mui/material/ButtonBase';
import Dialog from '@mui/material/Dialog';
import { buildTokens, fmtDate, fmtMoney, NEUTRAL } from '@/styles/tokens';
import { EmptyState, Sym, TextInput } from '@/components/ui';
import type { Contribution } from '../types';
import { groupKey } from './FinancesContributions';
import { getIntlLocale, t } from '@/i18n';

type Tk = ReturnType<typeof buildTokens>;

interface Props {
  tk: Tk;
  open: boolean;
  onClose: () => void;
  /** Already filtered to non-archived, not-yet-fully-paid contributions. */
  contributions: Contribution[];
  selectedId: string | undefined;
  onSelect: (id: string) => void;
}

/**
 * Popup member x fee-group grid for linking a new income transaction to the
 * contribution it pays -- a checkbox-styled single-select over the same
 * grid shape as FinancesContributions' matrix view, in its own dialog
 * because the transaction sheet is too narrow for a grid alongside its
 * other fields (see design.md's "Linking-picker matrix lives in a Dialog"
 * decision).
 */
export function ContribLinkMatrixDialog({ tk, open, onClose, contributions, selectedId, onSelect }: Props) {
  const [query, setQuery] = useState('');

  // A row matches if either its member name or its fee label matches --
  // mirroring the flat list's per-row filter this replaced -- and members
  // and fee-group columns are then derived from that filtered set, so a
  // fee-name search (e.g. "Turnier") still shows every member who has a
  // row in that fee, not just members whose own name happens to match too.
  const q = query.trim().toLowerCase();
  const filtered = q
    ? contributions.filter((c) => (c.name ?? '').toLowerCase().includes(q) || c.label.toLowerCase().includes(q))
    : contributions;

  const groupLabel: Record<string, string> = {};
  const groupDueDate: Record<string, string | null> = {};
  filtered.forEach((c) => {
    const key = groupKey(c);
    groupLabel[key] = c.label;
    groupDueDate[key] = c.dueDate || null;
  });
  const groups = [...new Set(filtered.map(groupKey))].sort((a, b) => {
    const da = groupDueDate[a];
    const db = groupDueDate[b];
    if (da && db) return da.localeCompare(db) || groupLabel[a]!.localeCompare(groupLabel[b]!, getIntlLocale());
    if (da) return -1;
    if (db) return 1;
    return groupLabel[a]!.localeCompare(groupLabel[b]!, getIntlLocale());
  });

  const memberIds = [...new Set(filtered.map((c) => c.userId))];
  const memberMeta: Record<string, { name: string; avatarColor: string | undefined; photo: string | null | undefined }> = {};
  filtered.forEach((c) => {
    if (!memberMeta[c.userId]) memberMeta[c.userId] = { name: c.name || '', avatarColor: c.avatarColor, photo: c.photo };
  });
  const members = memberIds
    .map((id) => ({ userId: id, ...memberMeta[id]! }))
    .sort((a, b) => a.name.localeCompare(b.name, getIntlLocale()));

  const cells: Record<string, Record<string, Contribution>> = useMemo(() => {
    const out: Record<string, Record<string, Contribution>> = {};
    contributions.forEach((c) => {
      const key = groupKey(c);
      out[c.userId] ??= {};
      out[c.userId]![key] = c;
    });
    return out;
  }, [contributions]);

  const thBase = {
    borderBottom: `1px solid ${NEUTRAL.line}`,
    padding: '5px 3px',
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
    width: '1px',
    maxWidth: '140px',
    boxShadow: `1px 0 0 ${NEUTRAL.line}`,
  } as const;

  return (
    <Dialog open={open} onClose={onClose} maxWidth="lg" fullWidth>
      <Box sx={{ p: '18px', display: 'flex', flexDirection: 'column', gap: '12px' }}>
        <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
          <Box sx={{ fontSize: '15px', fontWeight: 700 }}>{t('finances.linkMatrixTitle')}</Box>
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
          name="contribLinkMatrixSearch"
          placeholder={t('finances.linkMatrixSearchPlaceholder')}
          value={query}
          onChange={(e) => setQuery(e.target.value)}
        />
        {!members.length || !groups.length ? (
          <EmptyState icon="grid_on" text={t('finances.linkMatrixEmpty')} />
        ) : (
          <Box sx={{ overflow: 'auto', maxHeight: '60vh', border: `1px solid ${NEUTRAL.line}`, borderRadius: '14px' }}>
            <Box component="table" sx={{ borderCollapse: 'collapse', width: '100%', tableLayout: 'auto' }}>
              <Box component="thead">
                <Box component="tr">
                  <Box component="th" scope="col" sx={{ ...thBase, ...nameColSx }}>
                    {t('finances.linkMatrixMemberHeader')}
                  </Box>
                  {groups.map((g) => {
                    const due = groupDueDate[g];
                    return (
                      <Box
                        key={g}
                        component="th"
                        scope="col"
                        title={due ? groupLabel[g] + ' · ' + fmtDate(due) : groupLabel[g]}
                        sx={{ ...thBase, textAlign: 'center', minWidth: '56px', maxWidth: '84px' }}
                      >
                        <Box sx={{ whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis' }}>{groupLabel[g]}</Box>
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
                      sx={{ ...nameColSx, borderBottom: `1px solid ${NEUTRAL.line}`, padding: '5px 6px', fontWeight: 600, fontSize: '13px' }}
                    >
                      <Box sx={{ whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis' }}>{m.name}</Box>
                    </Box>
                    {groups.map((g) => {
                      const c = cells[m.userId]?.[g];
                      if (!c) {
                        return (
                          <Box key={g} component="td" sx={{ borderBottom: `1px solid ${NEUTRAL.line}`, textAlign: 'center', padding: '3px 2px' }}>
                            <Sym name="remove" size={16} color={NEUTRAL.line3} />
                          </Box>
                        );
                      }
                      const sel = c.id === selectedId;
                      const owed = c.amount - c.paidAmount;
                      return (
                        <Box key={g} component="td" sx={{ borderBottom: `1px solid ${NEUTRAL.line}`, textAlign: 'center', padding: '2px' }}>
                          <ButtonBase
                            type="button"
                            role="checkbox"
                            aria-checked={sel}
                            aria-label={t('finances.linkMatrixCellAria', { name: m.name, group: groupLabel[g]!, amount: fmtMoney(owed) })}
                            onClick={() => {
                              onSelect(c.id);
                              onClose();
                            }}
                            sx={{
                              width: '100%',
                              p: '5px 6px',
                              borderRadius: '100px',
                              cursor: 'pointer',
                              fontSize: '12px',
                              fontWeight: 700,
                              border: '1.5px solid ' + (sel ? tk.primary : NEUTRAL.line3),
                              background: sel ? tk.primaryContainer : NEUTRAL.card,
                              color: sel ? tk.onPrimaryContainer : NEUTRAL.onSurfaceVariant,
                              transition: 'border-color .12s ease, background .12s ease',
                              '&:hover': { borderColor: tk.primary },
                            }}
                          >
                            {fmtMoney(owed)}
                          </ButtonBase>
                        </Box>
                      );
                    })}
                  </Box>
                ))}
              </Box>
            </Box>
          </Box>
        )}
      </Box>
    </Dialog>
  );
}
