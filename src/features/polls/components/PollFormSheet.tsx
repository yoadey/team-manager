import Box from '@mui/material/Box';
import ButtonBase from '@mui/material/ButtonBase';
import { buildTokens } from '@/styles/tokens';
import { Field, PrimaryButton, Sym, TextInput, labelSx } from '@/components/ui';
import type { SheetProps } from '@/sheets/types';

export function PollFormSheet({ app, sheet }: SheetProps) {
  const { state } = app;
  const t = buildTokens(state.primaryColor);
  void sheet;
  const F = app.state.form;
  const errs = state.formErrors;

  const toggle = (key: string, label: string, icon: string) => (
    <ButtonBase
      key={key}
      onClick={() => app.setFormVal({ [key]: !F[key] })}
      sx={{
        flex: 1,
        display: 'flex',
        alignItems: 'center',
        gap: '9px',
        p: '12px',
        borderRadius: '13px',
        cursor: 'pointer',
        border: '1px solid ' + (F[key] ? t.primary : '#E0E2EA'),
        background: F[key] ? t.primaryContainer : '#fff',
      }}
    >
      <Sym name={icon} size={19} color={F[key] ? t.primary : '#6A6D76'} />
      <Box
        component="span"
        sx={{ fontSize: '13px', fontWeight: 600, color: F[key] ? t.onPrimaryContainer : '#44474E' }}
      >
        {label}
      </Box>
    </ButtonBase>
  );

  const validateQuestion = () => app.setFormErrors({ question: String(F.question ?? '').trim() ? '' : 'Frage fehlt.' });

  const opts = [F.opt0, F.opt1, F.opt2, F.opt3].map((o) => String(o ?? '').trim()).filter(Boolean);
  const canSubmit = !!(F.question as string | undefined)?.trim() && opts.length >= 2;

  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: '16px' }}>
      <Field label="Frage" required error={!!errs.question} errorText={errs.question}>
        <TextInput name="question" placeholder="Worüber soll abgestimmt werden?" onBlur={validateQuestion} />
      </Field>
      <Box>
        <Box sx={labelSx}>Antwortoptionen</Box>
        {opts.length < 2 && errs.options ? (
          <Box sx={{ fontSize: '12px', color: '#BA1A1A', mb: '6px' }} role="alert">
            {errs.options}
          </Box>
        ) : null}
        <Box sx={{ display: 'flex', flexDirection: 'column', gap: '8px' }}>
          <TextInput
            name="opt0"
            placeholder="Option 1"
            onBlur={() => app.setFormErrors({ options: opts.length < 2 ? 'Mindestens zwei Optionen angeben.' : '' })}
          />
          <TextInput
            name="opt1"
            placeholder="Option 2"
            onBlur={() => app.setFormErrors({ options: opts.length < 2 ? 'Mindestens zwei Optionen angeben.' : '' })}
          />
          <TextInput name="opt2" placeholder="Option 3 (optional)" />
          <TextInput name="opt3" placeholder="Option 4 (optional)" />
        </Box>
      </Box>
      <Box sx={{ display: 'flex', gap: '8px' }}>
        {toggle('multiple', 'Mehrfachauswahl', 'checklist')}
        {toggle('anonymous', 'Anonym', 'visibility_off')}
      </Box>
      <PrimaryButton
        label="Umfrage erstellen"
        onClick={() => app.savePoll()}
        busy={app.state.busy === 'save'}
        disabled={!canSubmit}
      />
    </Box>
  );
}
