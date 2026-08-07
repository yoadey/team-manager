import Box from '@mui/material/Box';
import ButtonBase from '@mui/material/ButtonBase';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { NEUTRAL } from '@/styles/tokens';
import { Field, PrimaryButton, Sym, TextArea, TextInput } from '@/components/ui';
import type { SheetProps } from '@/sheets/types';
import { contribFormSchema, type ContribFormValues } from './contribFormSchema';
import { MAX_MONEY_AMOUNT_EUROS, validateMoneyAmount } from '@/utils/validation';
import { t } from '@/i18n';

export function ContribFormSheet({ app, sheet }: SheetProps) {
  const {
    register,
    handleSubmit,
    watch,
    setValue,
    formState: { errors, isSubmitting },
  } = useForm<ContribFormValues>({
    resolver: zodResolver(contribFormSchema),
    defaultValues: sheet.formInitial as ContribFormValues,
    mode: 'onBlur',
  });

  const label = watch('label');
  const amount = watch('amount');
  const archived = watch('archived');
  const canSubmit = !!label?.trim() && validateMoneyAmount(amount, { positive: true, max: MAX_MONEY_AMOUNT_EUROS }).ok;

  const onSubmit = async (values: ContribFormValues) => {
    try {
      await app.saveContrib(values);
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
      {archived ? (
        <Box
          sx={{
            display: 'flex',
            alignItems: 'center',
            gap: '8px',
            p: '10px 12px',
            borderRadius: '12px',
            background: NEUTRAL.sidebar,
            color: NEUTRAL.secondary,
            fontSize: '12px',
            fontWeight: 600,
          }}
        >
          <Sym name="archive" size={16} color={NEUTRAL.secondary} />
          {t('finances.contribArchivedNotice')}
        </Box>
      ) : null}
      <Field label={t('finances.contribFieldLabel')} required error={!!errors.label} errorText={errors.label?.message}>
        <TextInput placeholder={t('finances.contribFieldLabelPlaceholder')} maxLength={255} {...register('label')} />
      </Field>
      <Field
        label={t('finances.contribFieldAmount')}
        required
        error={!!errors.amount}
        errorText={errors.amount?.message}
      >
        <TextInput type="number" max={MAX_MONEY_AMOUNT_EUROS} {...register('amount')} />
      </Field>
      <Field
        label={t('finances.contribFieldDescription')}
        error={!!errors.description}
        errorText={errors.description?.message}
      >
        <TextArea maxLength={2000} placeholder={t('finances.contribFieldDescriptionPlaceholder')} {...register('description')} />
      </Field>
      <Field label={t('finances.contribFieldDueDate')} error={!!errors.dueDate} errorText={errors.dueDate?.message}>
        <TextInput type="date" {...register('dueDate')} />
      </Field>
      <PrimaryButton
        label={t('finances.contribSave')}
        onClick={handleSubmit(onSubmit)}
        busy={isSubmitting || app.state.savingContrib}
        disabled={!canSubmit}
      />
      <ButtonBase
        type="button"
        onClick={() => setValue('archived', !archived, { shouldValidate: true })}
        sx={{
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          gap: '8px',
          p: '12px',
          borderRadius: '13px',
          border: `1px solid ${NEUTRAL.line}`,
          background: NEUTRAL.sidebar,
          color: NEUTRAL.onSurfaceVariant,
          fontWeight: 600,
          cursor: 'pointer',
        }}
      >
        <Sym name={archived ? 'unarchive' : 'archive'} size={19} color={NEUTRAL.onSurfaceVariant} />
        {archived ? t('finances.contribUnarchive') : t('finances.contribArchive')}
      </ButtonBase>
      <ButtonBase
        type="button"
        onClick={() => app.deleteContrib((sheet.formInitial as ContribFormValues).id)}
        sx={{
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          gap: '8px',
          p: '12px',
          borderRadius: '13px',
          border: '1px solid #F0C4C0',
          background: NEUTRAL.errorBg,
          color: NEUTRAL.error,
          fontWeight: 600,
          cursor: 'pointer',
        }}
      >
        <Sym name="delete" size={19} color={NEUTRAL.error} />
        {t('common.delete')}
      </ButtonBase>
    </Box>
  );
}
