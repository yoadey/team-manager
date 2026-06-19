import Box from '@mui/material/Box';
import { buildTokens } from '@/styles/tokens';
import { Field, PrimaryButton, Sym, TextInput } from '@/components/ui';
import type { SheetProps } from '@/sheets/types';
import { t } from '@/i18n';

export function AbsenceFormSheet({ app, sheet }: SheetProps) {
  const { state } = app;
  const tk = buildTokens(state.primaryColor);
  void tk;
  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: '16px' }}>
      <Box
        sx={{
          display: 'flex',
          gap: '12px',
          fontSize: '13px',
          color: '#6A6D76',
          lineHeight: 1.5,
          background: '#FFF7E6',
          border: '1px solid #F0DBA8',
          p: '12px 14px',
          borderRadius: '13px',
        }}
      >
        <Sym name="info" size={20} color="#8A6100" />
        {t('events.absenceHint')}
      </Box>
      <Box sx={{ display: 'flex', gap: '10px' }}>
        <Field label={t('events.absenceFrom')}>
          <TextInput name="from" type="date" />
        </Field>
        <Field label={t('events.absenceTo')}>
          <TextInput name="to" type="date" />
        </Field>
      </Box>
      <Field label={t('events.absenceReason')}>
        <TextInput name="reason" placeholder={t('events.absenceReasonPlaceholder')} />
      </Field>
      <PrimaryButton
        label={sheet.mode === 'edit' ? t('events.absenceSaveEdit') : t('events.absenceSave')}
        onClick={() => app.saveAbsence()}
        busy={app.state.busy === 'save'}
      />
    </Box>
  );
}
