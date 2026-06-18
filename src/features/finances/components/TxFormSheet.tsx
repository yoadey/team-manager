import Box from '@mui/material/Box';
import ButtonBase from '@mui/material/ButtonBase';
import { buildTokens } from '@/styles/tokens';
import { Field, PrimaryButton, Sym, TextInput, inputSx } from '@/components/ui';
import type { SheetProps } from '@/sheets/types';

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
