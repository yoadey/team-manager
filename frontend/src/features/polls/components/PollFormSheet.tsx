import Box from '@mui/material/Box';
import ButtonBase from '@mui/material/ButtonBase';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { buildTokens, NEUTRAL } from '@/styles/tokens';
import { Field, PrimaryButton, Sym, TextInput, labelSx } from '@/components/ui';
import type { SheetProps } from '@/sheets/types';
import { pollFormSchema, type PollFormValues } from './pollFormSchema';
import { t } from '@/i18n';

export function PollFormSheet({ app, sheet }: SheetProps) {
  const { state } = app;
  const tk = buildTokens(state.primaryColor);

  const {
    register,
    handleSubmit,
    setValue,
    watch,
    formState: { errors, isSubmitting, touchedFields },
  } = useForm<PollFormValues>({
    resolver: zodResolver(pollFormSchema),
    defaultValues: sheet.formInitial as PollFormValues,
    mode: 'onBlur',
  });

  const multiple = watch('multiple');
  const anonymous = watch('anonymous');
  const question = watch('question');
  const opt0 = watch('opt0');
  const opt1 = watch('opt1');
  const opt2 = watch('opt2');
  const opt3 = watch('opt3');

  const optsCount = [opt0, opt1, opt2, opt3].map((o) => String(o ?? '').trim()).filter(Boolean).length;
  const canSubmit = !!question?.trim() && optsCount >= 2;
  // The "at least 2 options" rule is a cross-field check (schema
  // `.superRefine()` on a virtual `options` path, still enforced at submit
  // time), but react-hook-form only reliably surfaces `formState.errors` for
  // registered field names -- so the inline hint is derived directly from
  // the watched option values instead, shown once the user has actually
  // interacted with either of the first two options (matching the
  // pre-RHF on-blur behavior).
  const optionsTouched = !!(touchedFields.opt0 || touchedFields.opt1);
  const optionsError = optionsTouched && optsCount < 2 ? t('polls.optionsError') : undefined;

  const onSubmit = async (values: PollFormValues) => {
    try {
      await app.savePoll(values);
    } catch {
      // Ignored
    }
  };

  const toggle = (key: 'multiple' | 'anonymous', value: boolean, label: string, icon: string) => (
    <ButtonBase
      key={key}
      type="button"
      onClick={() => setValue(key, !value, { shouldValidate: true })}
      sx={{
        flex: 1,
        display: 'flex',
        alignItems: 'center',
        gap: '9px',
        p: '12px',
        borderRadius: '13px',
        cursor: 'pointer',
        border: '1px solid ' + (value ? tk.primary : NEUTRAL.line3),
        background: value ? tk.primaryContainer : NEUTRAL.card,
      }}
    >
      <Sym name={icon} size={19} color={value ? tk.primary : NEUTRAL.secondary} />
      <Box
        component="span"
        sx={{ fontSize: '13px', fontWeight: 600, color: value ? tk.onPrimaryContainer : NEUTRAL.onSurfaceVariant }}
      >
        {label}
      </Box>
    </ButtonBase>
  );

  return (
    <Box
      component="form"
      onSubmit={handleSubmit(onSubmit)}
      sx={{ display: 'flex', flexDirection: 'column', gap: '16px' }}
    >
      <Field
        label={t('polls.fieldQuestion')}
        required
        error={!!errors.question}
        errorText={errors.question?.message}
      >
        <TextInput placeholder={t('polls.fieldQuestionPlaceholder')} maxLength={1000} {...register('question')} />
      </Field>
      <Box role="group" aria-labelledby="poll-options-label">
        <Box id="poll-options-label" sx={labelSx}>
          {t('polls.answerOptions')}
        </Box>
        {optionsError ? (
          <Box id="poll-options-error" sx={{ fontSize: '12px', color: NEUTRAL.error, mb: '6px' }} role="alert">
            {optionsError}
          </Box>
        ) : null}
        <Box sx={{ display: 'flex', flexDirection: 'column', gap: '8px' }}>
          <TextInput
            placeholder={t('polls.option1')}
            maxLength={500}
            aria-invalid={!!optionsError}
            aria-describedby={optionsError ? 'poll-options-error' : undefined}
            {...register('opt0')}
          />
          <TextInput
            placeholder={t('polls.option2')}
            maxLength={500}
            aria-invalid={!!optionsError}
            aria-describedby={optionsError ? 'poll-options-error' : undefined}
            {...register('opt1')}
          />
          <TextInput placeholder={t('polls.option3')} maxLength={500} {...register('opt2')} />
          <TextInput placeholder={t('polls.option4')} maxLength={500} {...register('opt3')} />
        </Box>
      </Box>
      <Box sx={{ display: 'flex', gap: '8px' }}>
        {toggle('multiple', !!multiple, t('polls.toggleMultiple'), 'checklist')}
        {toggle('anonymous', !!anonymous, t('polls.toggleAnonymous'), 'visibility_off')}
      </Box>
      <PrimaryButton
        label={t('polls.create')}
        onClick={handleSubmit(onSubmit)}
        busy={isSubmitting || app.state.savingPoll}
        disabled={!canSubmit}
      />
    </Box>
  );
}
