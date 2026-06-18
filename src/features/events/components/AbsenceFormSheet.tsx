import Box from '@mui/material/Box';
import { buildTokens } from '@/styles/tokens';
import { Field, PrimaryButton, Sym, TextInput } from '@/components/ui';
import type { SheetProps } from '@/sheets/types';

export function AbsenceFormSheet({ app, sheet }: SheetProps) {
  const { state } = app;
  const t = buildTokens(state.primaryColor);
  void t;
  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: '16px' }}>
      <Box sx={{ display: 'flex', gap: '12px', fontSize: '13px', color: '#6A6D76', lineHeight: 1.5, background: '#FFF7E6', border: '1px solid #F0DBA8', p: '12px 14px', borderRadius: '13px' }}>
        <Sym name="info" size={20} color="#8A6100" />
        In diesem Zeitraum wirst du für alle Termine automatisch abgesagt. Du kannst einzelne Termine manuell überschreiben.
      </Box>
      <Box sx={{ display: 'flex', gap: '10px' }}>
        <Field label="Von"><TextInput name="from" type="date" /></Field>
        <Field label="Bis"><TextInput name="to" type="date" /></Field>
      </Box>
      <Field label="Grund"><TextInput name="reason" placeholder="z. B. Urlaub" /></Field>
      <PrimaryButton label={sheet.mode === 'edit' ? 'Änderungen speichern' : 'Abwesenheit eintragen'} onClick={() => app.saveAbsence()} busy={app.state.busy === 'save'} />
    </Box>
  );
}
