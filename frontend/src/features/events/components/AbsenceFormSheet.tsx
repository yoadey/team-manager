import Box from '@mui/material/Box';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { buildTokens, NEUTRAL } from '@/styles/tokens';
import { Field, PrimaryButton, Sym, TextInput } from '@/components/ui';
import type { SheetProps } from '@/sheets/types';
import { absenceFormSchema, type AbsenceFormValues } from './absenceFormSchema';
import { t } from '@/i18n';

export function AbsenceFormSheet({ app, sheet }: SheetProps) {
  const { state } = app;
  const tk = buildTokens(state.primaryColor);
  void tk;

  const {
    register,
    handleSubmit,
    watch,
    formState: { errors, isSubmitting },
  } = useForm<AbsenceFormValues>({
    resolver: zodResolver(absenceFormSchema),
    defaultValues: state.form as AbsenceFormValues,
    mode: 'onTouched',
  });

  const from = watch('from');
  const to = watch('to');

  const onSubmit = async (values: AbsenceFormValues) => {
    try {
      await app.saveAbsence(values);
    } catch {
      // Ignored
    }
  };

  return (
    <Box
      component="form"
      onSubmit={handleSubmit(onSubmit)}
      sx={{ display: 'flex', flexDirection: 'column', gap: '16px' }}
    >
      <Box
        sx={{
          display: 'flex',
          gap: '12px',
          fontSize: '13px',
          color: NEUTRAL.secondary,
          lineHeight: 1.5,
          background: NEUTRAL.warnBg,
          border: '1px solid #F0DBA8',
          p: '12px 14px',
          borderRadius: '13px',
        }}
      >
        <Sym name="info" size={20} color={NEUTRAL.warn} />
        {t('events.absenceHint')}
      </Box>
      <Box sx={{ display: 'flex', gap: '10px' }}>
        <Field label={t('events.absenceFrom')} error={!!errors.from} errorText={errors.from?.message}>
          <TextInput type="date" max={to || undefined} {...register('from')} />
        </Field>
        <Field label={t('events.absenceTo')} error={!!errors.to} errorText={errors.to?.message}>
          <TextInput type="date" min={from || undefined} {...register('to')} />
        </Field>
      </Box>
      <Field label={t('events.absenceReason')} error={!!errors.reason} errorText={errors.reason?.message}>
        <TextInput placeholder={t('events.absenceReasonPlaceholder')} maxLength={500} {...register('reason')} />
      </Field>
      <PrimaryButton
        label={sheet.mode === 'edit' ? t('events.absenceSaveEdit') : t('events.absenceSave')}
        onClick={handleSubmit(onSubmit)}
        busy={isSubmitting || app.state.savingAbsence}
      />
    </Box>
  );
}
