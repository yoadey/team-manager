import Box from '@mui/material/Box';
import ButtonBase from '@mui/material/ButtonBase';
import { buildTokens } from '../../../styles/tokens';
import { Field, PrimaryButton, Sym, TextInput, labelSx } from '../../../components/ui';
import type { SheetProps } from '../../../sheets/types';

export function PollFormSheet({ app, sheet }: SheetProps) {
  const { state } = app;
  const t = buildTokens(state.primaryColor);
  void sheet;
  const F = app.state.form;
  const toggle = (key: string, label: string, icon: string) => (
    <ButtonBase key={key} onClick={() => app.setFormVal({ [key]: !F[key] })} sx={{ flex: 1, display: 'flex', alignItems: 'center', gap: '9px', p: '12px', borderRadius: '13px', cursor: 'pointer', border: '1px solid ' + (F[key] ? t.primary : '#E0E2EA'), background: F[key] ? t.primaryContainer : '#fff' }}>
      <Sym name={icon} size={19} color={F[key] ? t.primary : '#6A6D76'} />
      <Box component="span" sx={{ fontSize: '13px', fontWeight: 600, color: F[key] ? t.onPrimaryContainer : '#44474E' }}>{label}</Box>
    </ButtonBase>
  );
  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: '16px' }}>
      <Field label="Frage"><TextInput name="question" placeholder="Worüber soll abgestimmt werden?" /></Field>
      <Box>
        <Box sx={labelSx}>Antwortoptionen</Box>
        <Box sx={{ display: 'flex', flexDirection: 'column', gap: '8px' }}>
          <TextInput name="opt0" placeholder="Option 1" />
          <TextInput name="opt1" placeholder="Option 2" />
          <TextInput name="opt2" placeholder="Option 3 (optional)" />
          <TextInput name="opt3" placeholder="Option 4 (optional)" />
        </Box>
      </Box>
      <Box sx={{ display: 'flex', gap: '8px' }}>
        {toggle('multiple', 'Mehrfachauswahl', 'checklist')}
        {toggle('anonymous', 'Anonym', 'visibility_off')}
      </Box>
      <PrimaryButton label="Umfrage erstellen" onClick={() => app.savePoll()} busy={app.state.busy === 'save'} />
    </Box>
  );
}
