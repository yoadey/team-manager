import Box from '@mui/material/Box';
import ButtonBase from '@mui/material/ButtonBase';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { buildTokens, NEUTRAL } from '@/styles/tokens';
import { Field, PrimaryButton, Sym, TextInput } from '@/components/ui';
import type { SheetProps } from '@/sheets/types';
import { penaltyFormSchema, type PenaltyFormValues } from './penaltyFormSchema';
import { MAX_MONEY_AMOUNT_EUROS, validateMoneyAmount } from '@/utils/validation';
import { t } from '@/i18n';

export function PenaltyFormSheet({ app, sheet }: SheetProps) {
  const { state } = app;
  const tk = buildTokens(state.primaryColor);
  const create = sheet.mode === 'create';

  const {
    register,
    handleSubmit,
    watch,
    formState: { errors, isSubmitting },
  } = useForm<PenaltyFormValues>({
    resolver: zodResolver(penaltyFormSchema),
    defaultValues: state.form as PenaltyFormValues,
    mode: 'onBlur',
  });

  const label = watch('label');
  const amount = watch('amount');
  const canSubmit = !!label?.trim() && validateMoneyAmount(amount, { positive: true, max: MAX_MONEY_AMOUNT_EUROS }).ok;

  const onSubmit = async (values: PenaltyFormValues) => {
    try {
      await app.savePenalty(values);
    } catch {
      // Ignored
    }
  };

  const errs = state.formErrors || {};

  return (
    <Box component="form" onSubmit={handleSubmit(onSubmit)} sx={{ display: 'flex', flexDirection: 'column', gap: '16px' }}>
      <Box
        sx={{
          display: 'flex',
          gap: '12px',
          fontSize: '13px',
          color: NEUTRAL.secondary,
          lineHeight: 1.5,
          background: NEUTRAL.sidebar,
          border: `1px solid ${NEUTRAL.line}`,
          p: '12px 14px',
          borderRadius: '13px',
        }}
      >
        <Sym name="gavel" size={20} color={tk.primary} />
        {create ? t('finances.penaltyFormHintCreate') : t('finances.penaltyFormHintEdit')}
      </Box>
      <Field label={t('finances.penaltyFieldLabel')} required error={!!errors.label || !!errs.label} errorText={errors.label?.message || errs.label}>
        <TextInput placeholder={t('finances.penaltyFieldLabelPlaceholder')} maxLength={255} {...register('label')} />
      </Field>
      <Field label={t('finances.penaltyFieldAmount')} required error={!!errors.amount || !!errs.amount} errorText={errors.amount?.message || errs.amount}>
        <TextInput type="number" max={MAX_MONEY_AMOUNT_EUROS} {...register('amount')} />
      </Field>
      <PrimaryButton
        label={create ? t('finances.penaltySaveCreate') : t('finances.penaltySaveEdit')}
        onClick={handleSubmit(onSubmit)}
        busy={isSubmitting}
        disabled={!canSubmit}
      />
      {create ? null : (
        <ButtonBase
          key="del"
          type="button"
          onClick={() => app.deletePenaltyDef((state.form as PenaltyFormValues).id!)}
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
          {t('finances.penaltyRemove')}
        </ButtonBase>
      )}
    </Box>
  );
}
