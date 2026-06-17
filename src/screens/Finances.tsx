import Box from '@mui/material/Box';
import ButtonBase from '@mui/material/ButtonBase';
import { useApp } from '../store/AppContext';
import { buildTokens, fmtDate, fmtMoney, monthName, NEUTRAL } from '../theme/tokens';
import { Av, Chip, EmptyState, SectionTitle, SpinnerBox, Sym } from '../components/ui';
import type { Contribution, FinanceOverview, Transaction } from '../services/types';

export function Finances() {
  const app = useApp();
  const { state } = app;
  const t = buildTokens(state.primaryColor);
  const f = state.finances;
  if (!f) return <SpinnerBox />;
  const canFin = app.can('finances', 'write');

  const stat = (label: string, val: number, col: string) => (
    <Box key={label} sx={{ flex: 1, minWidth: '120px', background: '#fff', border: `1px solid ${NEUTRAL.line}`, borderRadius: '16px', p: '15px' }}>
      <Box sx={{ fontSize: '22px', fontWeight: 800, color: col }}>{fmtMoney(val)}</Box>
      <Box sx={{ fontSize: '12px', color: NEUTRAL.secondary, mt: '3px' }}>{label}</Box>
    </Box>
  );

  const tab = state.finTab || 'umsaetze';
  const tabDef: [typeof state.finTab, string, string][] = [
    ['umsaetze', 'Umsätze', 'receipt_long'],
    ['strafen', 'Strafen', 'gavel'],
    ['beitraege', 'Beiträge', 'payments'],
  ];

  const body = tab === 'strafen' ? finStrafen(app, t, f, canFin) : tab === 'beitraege' ? finBeitraege(app, t, f, canFin) : finUmsaetze(app, t, f, canFin);

  return (
    <Box sx={{ maxWidth: '760px' }}>
      <Box sx={{ background: `linear-gradient(120deg,${t.primary},${t.onPrimaryContainer})`, color: '#fff', borderRadius: '20px', p: '20px', mb: '16px' }}>
        <Box sx={{ fontSize: '13px', opacity: 0.85 }}>Aktueller Kassenstand</Box>
        <Box sx={{ fontSize: '34px', fontWeight: 800, mt: '4px' }}>{fmtMoney(f.balance)}</Box>
      </Box>
      <Box sx={{ display: 'flex', gap: '10px', flexWrap: 'wrap', mb: '18px' }}>
        {stat('Einnahmen', f.income, '#2E7D32')}
        {stat('Ausgaben', f.expense, '#BA1A1A')}
      </Box>
      <Box sx={{ display: 'flex', gap: '4px', background: '#ECEDF3', borderRadius: '14px', p: '4px', mb: '18px' }}>
        {tabDef.map(([k, l, ic]) => {
          const sel = tab === k;
          return (
            <ButtonBase
              key={k}
              onClick={() => app.setState({ finTab: k })}
              sx={{ flex: 1, display: 'flex', alignItems: 'center', justifyContent: 'center', gap: '7px', p: '10px 6px', borderRadius: '11px', border: 'none', cursor: 'pointer', fontSize: '13px', fontWeight: 700, background: sel ? '#fff' : 'transparent', color: sel ? t.primary : '#5A5D66', boxShadow: sel ? '0 1px 3px rgba(0,0,0,.12)' : 'none' }}
            >
              <Sym name={ic} size={18} color={sel ? t.primary : '#7A7D86'} />
              {l}
            </ButtonBase>
          );
        })}
      </Box>
      {body}
    </Box>
  );
}

type App = ReturnType<typeof useApp>;
type Tk = ReturnType<typeof buildTokens>;

// ---- Finanzen: Umsätze (alle Buchungen) ----
function finUmsaetze(app: App, t: Tk, f: FinanceOverview, canFin: boolean) {
  const txRow = (tx: Transaction) => {
    const inner = (
      <>
        <Box component="span" sx={{ width: '38px', height: '38px', borderRadius: '11px', background: tx.type === 'income' ? '#D7F0D8' : '#FFDAD6', color: tx.type === 'income' ? '#2E7D32' : '#BA1A1A', display: 'flex', alignItems: 'center', justifyContent: 'center', fontFamily: "'Material Symbols Outlined'", fontSize: '20px', flex: '0 0 auto' }}>
          {tx.type === 'income' ? 'south_west' : 'north_east'}
        </Box>
        <Box sx={{ flex: 1, minWidth: 0 }}>
          <Box sx={{ fontSize: '14px', fontWeight: 600, whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis' }}>{tx.title}</Box>
          <Box sx={{ fontSize: '12px', color: NEUTRAL.faint }}>{tx.category + ' · ' + fmtDate(tx.date)}</Box>
        </Box>
        <Box sx={{ fontSize: '15px', fontWeight: 700, color: tx.type === 'income' ? '#2E7D32' : '#BA1A1A' }}>
          {(tx.type === 'income' ? '+' : '−') + fmtMoney(tx.amount).replace('-', '')}
        </Box>
        {canFin ? <Sym name="chevron_right" size={20} color="#C0C2CA" /> : null}
      </>
    );
    const st = { display: 'flex', alignItems: 'center', gap: '12px', p: '12px 14px', background: '#fff', width: '100%', textAlign: 'left', border: 'none' } as const;
    return canFin
      ? <ButtonBase key={tx.id} onClick={() => app.openTxForm(tx)} sx={{ ...st, cursor: 'pointer', justifyContent: 'flex-start' }}>{inner}</ButtonBase>
      : <Box key={tx.id} sx={st}>{inner}</Box>;
  };
  return (
    <Box key="um">
      <SectionTitle right={canFin ? (
        <ButtonBase onClick={() => app.openTxForm(null)} sx={{ display: 'flex', alignItems: 'center', gap: '5px', border: 'none', background: 'transparent', color: t.primary, fontWeight: 700, fontSize: '13px', cursor: 'pointer' }}>
          <Sym name="add" size={17} color={t.primary} />Buchung
        </ButtonBase>
      ) : null}>{'Alle Umsätze · ' + f.transactions.length}</SectionTitle>
      {f.transactions.length ? (
        <Box sx={{ display: 'flex', flexDirection: 'column', gap: '1px', border: `1px solid ${NEUTRAL.line}`, borderRadius: '16px', overflow: 'hidden' }}>
          {f.transactions.map(txRow)}
        </Box>
      ) : <EmptyState icon="receipt_long" text="Noch keine Umsätze gebucht" />}
    </Box>
  );
}

// ---- Finanzen: Strafen (Katalog hinter Button) ----
function finStrafen(app: App, t: Tk, f: FinanceOverview, canFin: boolean) {
  const header = (
    <Box key="hd" sx={{ display: 'flex', alignItems: 'center', gap: '10px', mb: '16px', flexWrap: 'wrap' }}>
      <Box sx={{ flex: 1, minWidth: '120px' }}>
        <Box sx={{ fontSize: '22px', fontWeight: 800, color: '#BA1A1A' }}>{fmtMoney(f.openPenaltySum)}</Box>
        <Box sx={{ fontSize: '12px', color: NEUTRAL.secondary, mt: '2px' }}>offene Strafen gesamt</Box>
      </Box>
      <ButtonBase onClick={() => app.openPenaltyCatalog()} sx={{ display: 'inline-flex', alignItems: 'center', gap: '7px', p: '10px 14px', borderRadius: '12px', border: '1px solid #D0D2DA', background: '#fff', cursor: 'pointer', fontSize: '13px', fontWeight: 600, color: NEUTRAL.onSurfaceVariant }}>
        <Sym name="menu_book" size={18} color={NEUTRAL.secondary} />Strafenkatalog
      </ButtonBase>
      {canFin ? (
        <ButtonBase onClick={() => app.openPenaltyAssign()} sx={{ display: 'flex', alignItems: 'center', gap: '6px', border: 'none', background: t.primary, color: t.onPrimary, borderRadius: '12px', p: '10px 14px', fontSize: '13px', fontWeight: 700, cursor: 'pointer' }}>
          <Sym name="gavel" size={17} color={t.onPrimary} />Strafe erfassen
        </ButtonBase>
      ) : null}
    </Box>
  );
  const list = (
    <Box key="op" sx={{ display: 'flex', flexDirection: 'column', gap: '8px' }}>
      {f.assignments.length ? f.assignments.slice().reverse().map((a) => (
        <Box key={a.id} sx={{ display: 'flex', alignItems: 'center', gap: '11px', background: '#fff', border: `1px solid ${NEUTRAL.line}`, borderRadius: '14px', p: '10px 13px' }}>
          <Av name={a.name} photo={a.photo} color={a.avatarColor} size={36} />
          <Box sx={{ flex: 1, minWidth: 0 }}>
            <Box sx={{ fontSize: '14px', fontWeight: 600 }}>{a.name}</Box>
            <Box sx={{ fontSize: '12px', color: NEUTRAL.faint }}>{a.label + ' · ' + fmtDate(a.date)}</Box>
          </Box>
          <Box component="span" sx={{ fontSize: '14px', fontWeight: 700, color: a.paid ? NEUTRAL.faint : '#BA1A1A', textDecoration: a.paid ? 'line-through' : 'none' }}>{fmtMoney(a.amount || 0)}</Box>
          {canFin ? (
            <ButtonBase onClick={() => app.togglePenalty(a.id)} sx={{ border: 'none', cursor: 'pointer', borderRadius: '9px', p: '7px 11px', fontSize: '12px', fontWeight: 600, background: a.paid ? '#ECEDF3' : '#D7F0D8', color: a.paid ? NEUTRAL.secondary : '#235C26' }}>{a.paid ? 'offen' : 'bezahlt'}</ButtonBase>
          ) : null}
          {canFin ? (
            <ButtonBase onClick={() => app.deleteAssignment(a.id)} title="Löschen" sx={{ width: '30px', height: '30px', borderRadius: '8px', border: 'none', cursor: 'pointer', background: '#FFF4F3', color: '#BA1A1A', flex: '0 0 auto' }}>
              <Sym name="delete" size={18} color="#BA1A1A" />
            </ButtonBase>
          ) : null}
        </Box>
      )) : <EmptyState icon="savings" text="Noch keine Strafen erfasst" />}
    </Box>
  );
  return (
    <Box key="str">
      {header}
      <SectionTitle>Erfasste Strafen</SectionTitle>
      {list}
    </Box>
  );
}

// ---- Finanzen: Beiträge (nach Monat, filterbar) ----
function finBeitraege(app: App, t: Tk, f: FinanceOverview, canFin: boolean) {
  const { state } = app;
  const contribs = f.contributions || [];
  const months = [...new Set(contribs.map((c) => c.month).filter(Boolean))].sort().reverse();
  if (!months.length) return <EmptyState icon="payments" text="Noch keine Beiträge erfasst" />;
  const sel = state.contribMonth && months.includes(state.contribMonth) ? state.contribMonth : months[0];
  const rows = contribs.filter((c) => c.month === sel).sort((a, b) => a.name!.localeCompare(b.name!, 'de'));
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
          <ButtonBase key={m} onClick={() => app.setState({ contribMonth: m })} sx={{ flex: '0 0 auto', display: 'flex', flexDirection: 'column', alignItems: 'flex-start', gap: '2px', p: '9px 14px', borderRadius: '12px', cursor: 'pointer', border: '1.5px solid ' + (s ? t.primary : '#D0D2DA'), background: s ? t.primaryContainer : '#fff', color: s ? t.onPrimaryContainer : NEUTRAL.onSurfaceVariant }}>
            <Box component="span" sx={{ fontSize: '13px', fontWeight: 700, whiteSpace: 'nowrap', textTransform: 'capitalize' }}>{monthName(m)}</Box>
            <Box component="span" sx={{ fontSize: '11px', color: s ? t.onPrimaryContainer : NEUTRAL.faint, whiteSpace: 'nowrap' }}>{open ? open + ' offen' : 'vollständig'}</Box>
          </ButtonBase>
        );
      })}
    </Box>
  );
  const summary = (
    <Box key="sum" sx={{ display: 'flex', alignItems: 'center', gap: '14px', background: '#fff', border: `1px solid ${NEUTRAL.line}`, borderRadius: '16px', p: '15px 16px', mb: '14px' }}>
      <Box sx={{ flex: 1, minWidth: 0 }}>
        <Box sx={{ fontSize: '15px', fontWeight: 700, textTransform: 'capitalize' }}>{monthName(sel)}</Box>
        <Box sx={{ fontSize: '13px', color: NEUTRAL.secondary, mt: '3px' }}>{paidRows.length + ' von ' + rows.length + ' bezahlt · ' + fmtMoney(sum) + ' von ' + fmtMoney(total)}</Box>
      </Box>
      <Box sx={{ fontSize: '24px', fontWeight: 800, color: pct === 100 ? '#2E7D32' : t.primary, flex: '0 0 auto' }}>{pct + '%'}</Box>
    </Box>
  );
  const list = (
    <Box key="l" sx={{ display: 'flex', flexDirection: 'column', gap: '8px' }}>
      {rows.map((c: Contribution) => {
        const paid = c.status === 'paid';
        return (
          <Box key={c.id} sx={{ display: 'flex', alignItems: 'center', gap: '11px', background: '#fff', border: `1px solid ${NEUTRAL.line}`, borderRadius: '14px', p: '10px 13px' }}>
            <Av name={c.name} photo={c.photo} color={c.avatarColor} size={36} />
            <Box sx={{ flex: 1, minWidth: 0 }}>
              <Box sx={{ fontSize: '14px', fontWeight: 600, whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis' }}>{c.name}</Box>
              <Box sx={{ fontSize: '12px', color: NEUTRAL.faint }}>{fmtMoney(c.amount)}</Box>
            </Box>
            {canFin ? (
              <ButtonBase onClick={() => app.toggleContribution(c.id)} sx={{ border: 'none', cursor: 'pointer', borderRadius: '9px', p: '7px 12px', fontSize: '12px', fontWeight: 600, background: paid ? '#D7F0D8' : '#FFE5B8', color: paid ? '#235C26' : '#8A6100', display: 'flex', alignItems: 'center', gap: '5px' }}>
                <Sym name={paid ? 'check_circle' : 'schedule'} size={15} color={paid ? '#235C26' : '#8A6100'} />{paid ? 'bezahlt' : 'offen'}
              </ButtonBase>
            ) : (
              <Chip label={paid ? 'bezahlt' : 'offen'} color={paid ? '#2E7D32' : '#9A5B00'} bg={paid ? '#D7F0D8' : '#FFE5B8'} icon={paid ? 'check_circle' : 'schedule'} />
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
