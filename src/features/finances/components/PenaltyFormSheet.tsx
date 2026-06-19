import Box from '@mui/material/Box';
import ButtonBase from '@mui/material/ButtonBase';
import { buildTokens } from '@/styles/tokens';
import { Field, PrimaryButton, Sym, TextInput } from '@/components/ui';
import type { SheetProps } from '@/sheets/types';
import { t } from '@/i18n';

export function PenaltyFormSheet({ app, sheet }: SheetProps) {
  const { state } = app;
  const tk = buildTokens(state.primaryColor);
  const create = sheet.mode === 'create';
  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: '16px' }}>
      <Box
        sx={{
          display: 'flex',
          gap: '12px',
          fontSize: '13px',
          color: '#6A6D76',
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
      <Field label={t('finances.penaltyFieldLabel')}>
        <TextInput name="label" placeholder={t('finances.penaltyFieldLabelPlaceholder')} />
      </Field>
      <Field label={t('finances.penaltyFieldAmount')}>
        <TextInput name="amount" type="number" />
      </Field>
      <PrimaryButton
        label={create ? t('finances.penaltySaveCreate') : t('finances.penaltySaveEdit')}
        onClick={() => app.savePenalty()}
        busy={app.state.busy === 'save'}
      />
      {create ? null : (
        <ButtonBase
          key="del"
          onClick={() => app.deletePenaltyDef(app.state.form.id)}
          sx={{
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            gap: '8px',
            p: '12px',
            borderRadius: '13px',
            border: '1px solid #F0C4C0',
            background: '#FFF4F3',
            color: '#BA1A1A',
            fontWeight: 600,
            cursor: 'pointer',
          }}
        >
          <Sym name="delete" size={19} color="#BA1A1A" />
          {t('finances.penaltyRemove')}
        </ButtonBase>
      )}
    </Box>
  );
}
