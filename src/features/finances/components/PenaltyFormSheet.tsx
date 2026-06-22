import Box from '@mui/material/Box';
import ButtonBase from '@mui/material/ButtonBase';
import { buildTokens, NEUTRAL } from '@/styles/tokens';
import { Field, PrimaryButton, Sym, TextInput } from '@/components/ui';
import type { SheetProps } from '@/sheets/types';
import { formValues } from '@/utils/forms';
import type { PenaltyFormValues } from '../types';
import { t } from '@/i18n';

export function PenaltyFormSheet({ app, sheet }: SheetProps) {
  const { state } = app;
  const tk = buildTokens(state.primaryColor);
  const create = sheet.mode === 'create';
  const F = formValues<PenaltyFormValues>(state);
  const errs = state.formErrors;

  const validateLabel = () => {
    const v = String(F.label ?? '').trim();
    app.setFormErrors({ label: v ? '' : t('finances.penaltyFieldLabelError') });
  };

  const validateAmount = () => {
    const raw = String(F.amount ?? '')
      .trim()
      .replace(',', '.');
    const n = Number(raw);
    app.setFormErrors({
      amount: !raw
        ? t('finances.penaltyFieldAmountError')
        : !Number.isFinite(n) || n <= 0
          ? t('finances.penaltyFieldAmountErrorPositive')
          : '',
    });
  };

  const canSubmit =
    !!String(F.label ?? '').trim() &&
    (() => {
      const raw = String(F.amount ?? '')
        .trim()
        .replace(',', '.');
      const n = Number(raw);
      return raw && Number.isFinite(n) && n > 0;
    })();

  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: '16px' }}>
      <Box
        sx={{
          display: 'flex',
          gap: '12px',
          fontSize: '13px',
          color: NEUTRAL.secondary,
          lineHeight: 1.5,
          background: '#F4F4FA',
          border: '1px solid #E6E7EE',
          p: '12px 14px',
          borderRadius: '13px',
        }}
      >
        <Sym name="gavel" size={20} color={tk.primary} />
        {create ? t('finances.penaltyFormHintCreate') : t('finances.penaltyFormHintEdit')}
      </Box>
      <Field label={t('finances.penaltyFieldLabel')} required error={!!errs.label} errorText={errs.label}>
        <TextInput name="label" placeholder={t('finances.penaltyFieldLabelPlaceholder')} onBlur={validateLabel} />
      </Field>
      <Field label={t('finances.penaltyFieldAmount')} required error={!!errs.amount} errorText={errs.amount}>
        <TextInput name="amount" type="number" onBlur={validateAmount} />
      </Field>
      <PrimaryButton
        label={create ? t('finances.penaltySaveCreate') : t('finances.penaltySaveEdit')}
        onClick={() => app.savePenalty()}
        busy={app.state.busy === 'save'}
        disabled={!canSubmit}
      />
      {create ? null : (
        <ButtonBase
          key="del"
          onClick={() => app.deletePenaltyDef(F.id!)}
          sx={{
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            gap: '8px',
            p: '12px',
            borderRadius: '13px',
            border: '1px solid #F0C4C0',
            background: '#FFF4F3',
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
