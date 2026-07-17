import Box from '@mui/material/Box';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { Field, PrimaryButton, TextInput } from '@/components/ui';
import type { SheetProps } from '@/sheets/types';
import { contribFormSchema, type ContribFormValues } from './contribFormSchema';
import { MAX_MONEY_AMOUNT_EUROS, validateMoneyAmount } from '@/utils/validation';
import { t } from '@/i18n';

export function ContribFormSheet({ app, sheet }: SheetProps) {
  const {
    register,
    handleSubmit,
    watch,
    formState: { errors, isSubmitting },
  } = useForm<ContribFormValues>({
    resolver: zodResolver(contribFormSchema),
    defaultValues: sheet.formInitial as ContribFormValues,
    mode: 'onBlur',
  });

  const label = watch('label');
  const amount = watch('amount');
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
      <PrimaryButton
        label={t('finances.contribSave')}
        onClick={handleSubmit(onSubmit)}
        busy={isSubmitting || app.state.savingContrib}
        disabled={!canSubmit}
      />
    </Box>
  );
}
