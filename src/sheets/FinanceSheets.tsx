import Box from '@mui/material/Box';
import ButtonBase from '@mui/material/ButtonBase';
import { buildTokens, fmtMoney } from '../styles/tokens';
import { EmptyState, Field, PrimaryButton, Sym, TextInput, inputSx, labelSx } from '../components/ui';
import type { Member, Penalty } from '../types';
import type { SheetProps } from './types';

export function TxFormSheet({ app, sheet }: SheetProps) {
  const { state } = app;
  const t = buildTokens(state.primaryColor);
  const F = app.state.form;
  const edit = sheet.mode === 'edit';

  const typeDefs: [string, string, string, string, string][] = [
    ['income', 'Einnahme', 'south_west', '#2E7D32', '#D7F0D8'],
    ['expense', 'Ausgabe', 'north_east', '#BA1A1A', '#FFDAD6'],
  ];
  const typeBtns = typeDefs.map(([v, l, ic, c, bg]) => {
    const sel = F.type === v;
    return (
      <ButtonBase key={v} onClick={() => app.setFormVal({ type: v })} sx={{ flex: 1, display: 'flex', alignItems: 'center', justifyContent: 'center', gap: '7px', p: '12px', borderRadius: '13px', cursor: 'pointer', fontSize: '14px', fontWeight: 700, border: '1.5px solid ' + (sel ? c : '#E0E2EA'), background: sel ? bg : '#fff', color: sel ? c : '#6A6D76' }}>
        <Sym name={ic} size={18} color={sel ? c : '#6A6D76'} />
        {l}
      </ButtonBase>
    );
  });

  const cats = [...new Set(((app.state.finances && app.state.finances.transactions) || []).map((x) => x.category).filter(Boolean))].sort((a, b) => a.localeCompare(b, 'de'));

  const catField = (
    <Field label="Kategorie">
      <Box>
        <input
          key="i"
          name="category"
          list="tvCatList"
          autoComplete="off"
          value={F.category == null ? '' : F.category}
          onChange={app.onFormInput}
          placeholder="Kategorie wählen oder neu eingeben…"
          style={inputSx}
        />
        <datalist key="dl" id="tvCatList">
          {cats.map((c) => <option key={c} value={c} />)}
        </datalist>
        {cats.length ? (
          <Box key="qp" sx={{ display: 'flex', flexWrap: 'wrap', gap: '6px', mt: '8px' }}>
            {cats.map((c) => {
              const sel = F.category === c;
              return (
                <ButtonBase key={c} onClick={() => app.setFormVal({ category: c })} sx={{ p: '5px 11px', borderRadius: '999px', fontSize: '12px', fontWeight: 600, cursor: 'pointer', border: '1px solid ' + (sel ? t.primary : '#D0D2DA'), background: sel ? t.primaryContainer : '#fff', color: sel ? t.onPrimaryContainer : '#44474E' }}>{c}</ButtonBase>
              );
            })}
          </Box>
        ) : null}
        <Box key="hint" sx={{ fontSize: '11px', color: '#9A9DA6', mt: '8px', lineHeight: 1.5 }}>Vorhandene Kategorie wählen oder eine neue eintippen – neue Kategorien werden automatisch übernommen.</Box>
      </Box>
    </Field>
  );

  const del = edit ? (
    <ButtonBase key="del" onClick={() => app.askConfirm({ title: 'Buchung löschen?', message: '„' + (F.title || 'Diese Buchung') + '" wird dauerhaft aus der Kasse entfernt.', confirmLabel: 'Löschen', danger: true, onConfirm: async () => { await app.deleteTx(F.id); } })} sx={{ display: 'flex', alignItems: 'center', justifyContent: 'center', gap: '8px', p: '12px', borderRadius: '13px', border: '1px solid #F0C4C0', background: '#FFF4F3', color: '#BA1A1A', fontWeight: 600, cursor: 'pointer' }}>
      <Sym name="delete" size={19} color="#BA1A1A" />
      Buchung löschen
    </ButtonBase>
  ) : null;

  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: '16px' }}>
      <Box sx={{ display: 'flex', gap: '8px' }}>{typeBtns}</Box>
      <Field label="Bezeichnung"><TextInput name="title" placeholder="z. B. Mitgliedsbeiträge" /></Field>
      <Field label="Betrag (€)"><TextInput name="amount" type="number" /></Field>
      {catField}
      <PrimaryButton label={edit ? 'Änderungen speichern' : 'Buchung erfassen'} onClick={() => app.saveTx()} busy={app.state.busy === 'save'} />
      {del}
    </Box>
  );
}

export function PenaltyCatalogSheet({ app, sheet }: SheetProps) {
  const { state } = app;
  const t = buildTokens(state.primaryColor);
  void sheet;
  const f = app.state.finances;
  const canFin = app.can('finances', 'write');
  const pens: Penalty[] = f ? f.penalties : [];

  const rows = pens.map((p) => {
    const inner = (
      <>
        <Box component="span" key="i" sx={{ width: '36px', height: '36px', borderRadius: '10px', background: t.primaryContainer, color: t.primary, display: 'flex', alignItems: 'center', justifyContent: 'center', fontFamily: "'Material Symbols Outlined'", fontSize: '19px', flex: '0 0 auto' }}>gavel</Box>
        <Box key="m" sx={{ flex: 1, minWidth: 0 }}>
          <Box key="l" sx={{ fontSize: '14px', fontWeight: 600, whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis' }}>{p.label}</Box>
        </Box>
        <Box component="b" key="a" sx={{ fontSize: '14px', color: t.primary, flex: '0 0 auto' }}>{fmtMoney(p.amount)}</Box>
        {canFin ? <Sym name="chevron_right" size={20} color="#C0C2CA" /> : null}
      </>
    );
    const baseSx = { display: 'flex', alignItems: 'center', gap: '11px', background: '#fff', border: '1px solid #E6E7EE', borderRadius: '14px', p: '11px 13px', width: '100%', textAlign: 'left' as const };
    return canFin
      ? <ButtonBase key={p.id} onClick={() => app.openPenaltyForm(p)} sx={{ ...baseSx, cursor: 'pointer', justifyContent: 'flex-start' }}>{inner}</ButtonBase>
      : <Box key={p.id} sx={baseSx}>{inner}</Box>;
  });

  const add = canFin ? (
    <ButtonBase key="add" onClick={() => app.openPenaltyForm(null)} sx={{ display: 'flex', alignItems: 'center', justifyContent: 'center', gap: '9px', width: '100%', p: '13px', borderRadius: '14px', border: '1.5px dashed #C8CAD2', background: 'transparent', cursor: 'pointer', color: t.primary, fontWeight: 600, fontSize: '14px' }}>
      <Sym name="add_circle" size={20} color={t.primary} />
      Strafe zum Katalog hinzufügen
    </ButtonBase>
  ) : null;

  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: '10px' }}>
      <Box key="i" sx={{ fontSize: '13px', color: '#6A6D76', lineHeight: 1.5, mb: '2px' }}>Diese Strafen können erfasst werden. Tippe zum Bearbeiten von Bezeichnung und Betrag, oder lege neue Strafen an.</Box>
      {rows.length ? rows : <EmptyState icon="gavel" text="Noch keine Strafen im Katalog" />}
      {add}
    </Box>
  );
}

export function PenaltyFormSheet({ app, sheet }: SheetProps) {
  const { state } = app;
  const t = buildTokens(state.primaryColor);
  const create = sheet.mode === 'create';
  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: '16px' }}>
      <Box sx={{ display: 'flex', gap: '12px', fontSize: '13px', color: '#6A6D76', lineHeight: 1.5, background: '#F4F4FA', border: '1px solid #E6E7EE', p: '12px 14px', borderRadius: '13px' }}>
        <Sym name="gavel" size={20} color={t.primary} />
        {create ? 'Lege eine neue Strafe für den Katalog an. Anschließend kannst du sie einzelnen Mitgliedern zuweisen.' : 'Bearbeite Bezeichnung und Betrag dieser Strafe.'}
      </Box>
      <Field label="Bezeichnung"><TextInput name="label" placeholder="z. B. Zu spät zum Training" /></Field>
      <Field label="Betrag (€)"><TextInput name="amount" type="number" /></Field>
      <PrimaryButton label={create ? 'Strafe hinzufügen' : 'Änderungen speichern'} onClick={() => app.savePenalty()} busy={app.state.busy === 'save'} />
      {create ? null : (
        <ButtonBase key="del" onClick={() => app.deletePenaltyDef(app.state.form.id)} sx={{ display: 'flex', alignItems: 'center', justifyContent: 'center', gap: '8px', p: '12px', borderRadius: '13px', border: '1px solid #F0C4C0', background: '#FFF4F3', color: '#BA1A1A', fontWeight: 600, cursor: 'pointer' }}>
          <Sym name="delete" size={19} color="#BA1A1A" />
          Strafe aus Katalog entfernen
        </ButtonBase>
      )}
    </Box>
  );
}

export function PenaltyAssignSheet({ app, sheet }: SheetProps) {
  const { state } = app;
  const t = buildTokens(state.primaryColor);
  void sheet;
  const F = app.state.form;
  const f = app.state.finances;
  const members: Member[] = app.state.members || [];

  const penOpts = (
    <Box key="po">
      <Box key="l" sx={labelSx}>Strafe</Box>
      <Box key="b" sx={{ display: 'flex', flexDirection: 'column', gap: '8px' }}>
        {(f ? f.penalties : []).map((p) => {
          const sel = F.penaltyId === p.id;
          return (
            <ButtonBase key={p.id} onClick={() => app.setFormVal({ penaltyId: p.id })} sx={{ display: 'flex', alignItems: 'center', gap: '11px', p: '12px 14px', borderRadius: '13px', cursor: 'pointer', textAlign: 'left', justifyContent: 'flex-start', border: '1.5px solid ' + (sel ? t.primary : '#E0E2EA'), background: sel ? t.primaryContainer : '#fff' }}>
              <Box component="span" key="r" sx={{ width: '18px', height: '18px', borderRadius: '50%', flex: '0 0 auto', border: '2px solid ' + (sel ? t.primary : '#C0C2CA'), background: sel ? t.primary : '#fff', boxShadow: sel ? 'inset 0 0 0 3px #fff' : 'none' }} />
              <Box component="span" key="lb" sx={{ flex: 1, fontSize: '14px', fontWeight: 600, color: '#44474E' }}>{p.label}</Box>
              <Box component="b" key="a" sx={{ fontSize: '14px', color: t.primary }}>{fmtMoney(p.amount)}</Box>
            </ButtonBase>
          );
        })}
      </Box>
    </Box>
  );

  const memSel = (
    <Field label="Person">
      <select name="userId" value={F.userId || ''} onChange={app.onFormInput} style={inputSx}>
        <option key="_" value="">Bitte wählen…</option>
        {members.map((m) => <option key={m.userId} value={m.userId}>{m.name}</option>)}
      </select>
    </Field>
  );

  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: '16px' }}>
      {memSel}
      {penOpts}
      <PrimaryButton label="Strafe erfassen" onClick={() => app.savePenaltyAssign()} busy={app.state.busy === 'save'} />
    </Box>
  );
}

export function ContribFormSheet({ app, sheet }: SheetProps) {
  const { state } = app;
  const t = buildTokens(state.primaryColor);
  void t;
  void sheet;
  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: '16px' }}>
      <Field label="Bezeichnung"><TextInput name="label" placeholder="z. B. Monatsbeitrag" /></Field>
      <Field label="Betrag (€)"><TextInput name="amount" type="number" /></Field>
      <PrimaryButton label="Änderungen speichern" onClick={() => app.saveContrib()} busy={app.state.busy === 'save'} />
    </Box>
  );
}
