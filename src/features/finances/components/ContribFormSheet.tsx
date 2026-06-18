import Box from '@mui/material/Box';
import { buildTokens } from '@/styles/tokens';
import { Field, PrimaryButton, TextInput } from '@/components/ui';
import type { SheetProps } from '@/sheets/types';

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
