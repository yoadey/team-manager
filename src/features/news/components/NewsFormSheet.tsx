import Box from '@mui/material/Box';
import ButtonBase from '@mui/material/ButtonBase';
import { buildTokens, NEUTRAL } from '@/styles/tokens';
import { Field, PrimaryButton, Sym, TextArea, TextInput } from '@/components/ui';
import type { SheetProps } from '@/sheets/types';
import { formValues } from '@/utils/forms';
import type { NewsFormValues } from '../types';
import { t } from '@/i18n';

export function NewsFormSheet({ app, sheet }: SheetProps) {
  const { state } = app;
  const tk = buildTokens(state.primaryColor);
  const F = formValues<NewsFormValues>(app.state);
  const errs = state.formErrors;

  const pin = (
    <ButtonBase
      key="pin"
      onClick={() => app.setFormVal({ pinned: !F.pinned })}
      sx={{
        display: 'flex',
        alignItems: 'center',
        gap: '12px',
        width: '100%',
        p: '12px 14px',
        borderRadius: '13px',
        cursor: 'pointer',
        border: `1px solid ${NEUTRAL.line}`,
        background: NEUTRAL.sidebar,
      }}
    >
      <Sym name="push_pin" size={20} color={F.pinned ? tk.primary : NEUTRAL.faint} />
      <Box component="span" sx={{ flex: 1, textAlign: 'left', fontSize: '14px', fontWeight: 500 }}>
        {t('news.pinned')}
      </Box>
      <Box
        component="span"
        sx={{
          width: '44px',
          height: '26px',
          borderRadius: '999px',
          background: F.pinned ? tk.primary : NEUTRAL.inputBorder,
          position: 'relative',
        }}
      >
        <Box
          component="span"
          sx={{
            position: 'absolute',
            top: '3px',
            left: F.pinned ? '21px' : '3px',
            width: '20px',
            height: '20px',
            borderRadius: '50%',
            background: NEUTRAL.card,
            transition: 'left .2s',
          }}
        />
      </Box>
    </ButtonBase>
  );

  const validateTitle = () =>
    app.setFormErrors({ title: String(F.title ?? '').trim() ? '' : t('news.fieldTitleError') });
  const validateBody = () => app.setFormErrors({ body: String(F.body ?? '').trim() ? '' : t('news.fieldBodyError') });

  const canSubmit = !!F.title?.trim() && !!F.body?.trim();

  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: '16px' }}>
      <Field label={t('news.fieldTitle')} required error={!!errs.title} errorText={errs.title}>
        <TextInput name="title" placeholder={t('news.fieldTitlePlaceholder')} onBlur={validateTitle} maxLength={120} />
      </Field>
      <Field label={t('news.fieldBody')} required error={!!errs.body} errorText={errs.body}>
        <TextArea
          name="body"
          placeholder={t('news.fieldBodyPlaceholder')}
          minHeight={120}
          onBlur={validateBody}
          maxLength={5000}
        />
      </Field>
      {pin}
      <PrimaryButton
        label={sheet.mode === 'edit' ? t('news.saveChanges') : t('news.publish')}
        onClick={() => app.saveNews()}
        busy={app.state.busy === 'save'}
        disabled={!canSubmit}
      />
    </Box>
  );
}
