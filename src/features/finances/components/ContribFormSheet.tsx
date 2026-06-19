import Box from '@mui/material/Box';
import { buildTokens } from '@/styles/tokens';
import { Field, PrimaryButton, TextInput } from '@/components/ui';
import type { SheetProps } from '@/sheets/types';
import { t } from '@/i18n';

export function ContribFormSheet({ app, sheet }: SheetProps) {
  const { state } = app;
  const tk = buildTokens(state.primaryColor);
  void tk;
  void sheet;
  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: '16px' }}>
      <Field label={t('finances.contribFieldLabel')}>
        <TextInput name="label" placeholder={t('finances.contribFieldLabelPlaceholder')} />
      </Field>
      <Field label={t('finances.contribFieldAmount')}>
        <TextInput name="amount" type="number" />
      </Field>
      <PrimaryButton
        label={t('finances.contribSave')}
        onClick={() => app.saveContrib()}
        busy={app.state.busy === 'save'}
      />
    </Box>
  );
}
