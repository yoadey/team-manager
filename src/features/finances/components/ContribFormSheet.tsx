import Box from '@mui/material/Box';
import { Field, PrimaryButton, TextInput } from '@/components/ui';
import type { SheetProps } from '@/sheets/types';
import { formValues } from '@/utils/forms';
import type { ContribFormValues } from '../types';
import { t } from '@/i18n';

export function ContribFormSheet({ app }: SheetProps) {
  const { state } = app;
  const F = formValues<ContribFormValues>(state);
  const errs = state.formErrors;

  const validateLabel = () => {
    const v = String(F.label ?? '').trim();
    app.setFormErrors({ label: v ? '' : t('finances.contribFieldLabelError') });
  };

  const validateAmount = () => {
    const raw = String(F.amount ?? '')
      .trim()
      .replace(',', '.');
    const n = Number(raw);
    app.setFormErrors({
      amount: !raw
        ? t('finances.contribFieldAmountError')
        : !Number.isFinite(n) || n <= 0
          ? t('finances.contribFieldAmountErrorPositive')
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
      <Field label={t('finances.contribFieldLabel')} required error={!!errs.label} errorText={errs.label}>
        <TextInput name="label" placeholder={t('finances.contribFieldLabelPlaceholder')} onBlur={validateLabel} />
      </Field>
      <Field label={t('finances.contribFieldAmount')} required error={!!errs.amount} errorText={errs.amount}>
        <TextInput name="amount" type="number" onBlur={validateAmount} />
      </Field>
      <PrimaryButton
        label={t('finances.contribSave')}
        onClick={() => app.saveContrib()}
        busy={app.state.busy === 'save'}
        disabled={!canSubmit}
      />
    </Box>
  );
}
