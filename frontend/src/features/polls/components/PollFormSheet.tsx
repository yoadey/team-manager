import Box from '@mui/material/Box';
import ButtonBase from '@mui/material/ButtonBase';
import { buildTokens, NEUTRAL } from '@/styles/tokens';
import { Field, PrimaryButton, Sym, TextInput, labelSx } from '@/components/ui';
import type { SheetProps } from '@/sheets/types';
import { formValues } from '@/utils/forms';
import type { PollFormValues } from '../types';
import { t } from '@/i18n';

export function PollFormSheet({ app, sheet }: SheetProps) {
  const { state } = app;
  const tk = buildTokens(state.primaryColor);
  void sheet;
  const F = formValues<PollFormValues>(app.state);
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
        border: '1px solid ' + (F[key] ? tk.primary : NEUTRAL.line3),
        background: F[key] ? tk.primaryContainer : NEUTRAL.card,
      }}
    >
      <Sym name={icon} size={19} color={F[key] ? tk.primary : NEUTRAL.secondary} />
      <Box
        component="span"
        sx={{ fontSize: '13px', fontWeight: 600, color: F[key] ? tk.onPrimaryContainer : NEUTRAL.onSurfaceVariant }}
      >
        {label}
      </Box>
    </ButtonBase>
  );

  const validateQuestion = () =>
    app.setFormErrors({ question: String(F.question ?? '').trim() ? '' : t('polls.fieldQuestionError') });

  const opts = [F.opt0, F.opt1, F.opt2, F.opt3].map((o) => String(o ?? '').trim()).filter(Boolean);
  const canSubmit = !!F.question?.trim() && opts.length >= 2;

  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: '16px' }}>
      <Field label={t('polls.fieldQuestion')} required error={!!errs.question} errorText={errs.question}>
        <TextInput name="question" placeholder={t('polls.fieldQuestionPlaceholder')} onBlur={validateQuestion} />
      </Field>
      <Box>
        <Box sx={labelSx}>{t('polls.answerOptions')}</Box>
        {opts.length < 2 && errs.options ? (
          <Box sx={{ fontSize: '12px', color: NEUTRAL.error, mb: '6px' }} role="alert">
            {errs.options}
          </Box>
        ) : null}
        <Box sx={{ display: 'flex', flexDirection: 'column', gap: '8px' }}>
          <TextInput
            name="opt0"
            placeholder={t('polls.option1')}
            onBlur={() => app.setFormErrors({ options: opts.length < 2 ? t('polls.optionsError') : '' })}
          />
          <TextInput
            name="opt1"
            placeholder={t('polls.option2')}
            onBlur={() => app.setFormErrors({ options: opts.length < 2 ? t('polls.optionsError') : '' })}
          />
          <TextInput name="opt2" placeholder={t('polls.option3')} />
          <TextInput name="opt3" placeholder={t('polls.option4')} />
        </Box>
      </Box>
      <Box sx={{ display: 'flex', gap: '8px' }}>
        {toggle('multiple', t('polls.toggleMultiple'), 'checklist')}
        {toggle('anonymous', t('polls.toggleAnonymous'), 'visibility_off')}
      </Box>
      <PrimaryButton
        label={t('polls.create')}
        onClick={() => app.savePoll()}
        busy={app.state.busy === 'save'}
        disabled={!canSubmit}
      />
    </Box>
  );
}
