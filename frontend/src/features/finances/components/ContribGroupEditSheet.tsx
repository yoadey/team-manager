import Box from '@mui/material/Box';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { Field, PrimaryButton, TextArea, TextInput } from '@/components/ui';
import type { SheetProps } from '@/sheets/types';
import { contribGroupEditFormSchema, type ContribGroupEditFormValues } from './contribGroupEditFormSchema';
import { MAX_MONEY_AMOUNT_EUROS, validateMoneyAmount } from '@/utils/validation';
import { t } from '@/i18n';

/**
 * Edits a whole fee period (every row sharing the group's label+dueDate) in
 * one action -- the only place label/amount/description/dueDate can be
 * changed, mirroring `archiveContribGroup`'s fan-out. See
 * openspec/changes/contribution-detail-readonly-parent-edit.
 */
export function ContribGroupEditSheet({ app, sheet }: SheetProps) {
  const rows = sheet.contribGroupRows || [];
  const {
    register,
    handleSubmit,
    watch,
    formState: { errors, isSubmitting },
  } = useForm<ContribGroupEditFormValues>({
    resolver: zodResolver(contribGroupEditFormSchema),
    defaultValues: sheet.formInitial as ContribGroupEditFormValues,
    mode: 'onBlur',
  });

  const label = watch('label');
  const amount = watch('amount');
  const canSubmit = !!label?.trim() && validateMoneyAmount(amount, { positive: true, max: MAX_MONEY_AMOUNT_EUROS }).ok;

  const onSubmit = async (values: ContribGroupEditFormValues) => {
    try {
      await app.editContribGroup(rows, values);
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
    </Box>
  );
}
