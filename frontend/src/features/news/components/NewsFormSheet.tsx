import Box from '@mui/material/Box';
import ButtonBase from '@mui/material/ButtonBase';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { buildTokens, NEUTRAL } from '@/styles/tokens';
import { Field, PrimaryButton, Sym, TextArea, TextInput } from '@/components/ui';
import type { SheetProps } from '@/sheets/types';
import { newsFormSchema, type NewsFormValues } from './newsFormSchema';
import { t } from '@/i18n';

export function NewsFormSheet({ app, sheet }: SheetProps) {
  const { state } = app;
  const tk = buildTokens(state.primaryColor);

  const {
    register,
    handleSubmit,
    setValue,
    watch,
    formState: { errors, isSubmitting },
  } = useForm<NewsFormValues>({
    resolver: zodResolver(newsFormSchema),
    defaultValues: state.form as NewsFormValues,
    mode: 'onBlur',
  });

  const title = watch('title');
  const body = watch('body');
  const pinned = watch('pinned');

  const canSubmit = !!title?.trim() && !!body?.trim();

  const onSubmit = async (values: NewsFormValues) => {
    try {
      await app.saveNews(values);
    } catch {
      // Ignored
    }
  };

  const pin = (
    <ButtonBase
      key="pin"
      type="button"
      role="switch"
      aria-checked={!!pinned}
      aria-label={t('news.pinned')}
      onClick={() => setValue('pinned', !pinned, { shouldValidate: true })}
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
      <Sym name="push_pin" size={20} color={pinned ? tk.primary : NEUTRAL.faint} />
      <Box component="span" sx={{ flex: 1, textAlign: 'left', fontSize: '14px', fontWeight: 500 }}>
        {t('news.pinned')}
      </Box>
      <Box
        component="span"
        sx={{
          width: '44px',
          height: '26px',
          borderRadius: '999px',
          background: pinned ? tk.primary : NEUTRAL.inputBorder,
          position: 'relative',
        }}
      >
        <Box
          component="span"
          sx={{
            position: 'absolute',
            top: '3px',
            left: pinned ? '21px' : '3px',
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

  const errs = state.formErrors || {};

  return (
    <Box
      component="form"
      onSubmit={handleSubmit(onSubmit)}
      sx={{ display: 'flex', flexDirection: 'column', gap: '16px' }}
    >
      <Field
        label={t('news.fieldTitle')}
        required
        error={!!errors.title || !!errs.title}
        errorText={errors.title?.message || errs.title}
      >
        <TextInput placeholder={t('news.fieldTitlePlaceholder')} maxLength={255} {...register('title')} />
      </Field>
      <Field
        label={t('news.fieldBody')}
        required
        error={!!errors.body || !!errs.body}
        errorText={errors.body?.message || errs.body}
      >
        <TextArea
          placeholder={t('news.fieldBodyPlaceholder')}
          minHeight={120}
          maxLength={10000}
          {...register('body')}
        />
      </Field>
      {pin}
      <PrimaryButton
        label={sheet.mode === 'edit' ? t('news.saveChanges') : t('news.publish')}
        onClick={handleSubmit(onSubmit)}
        busy={isSubmitting}
        disabled={!canSubmit}
      />
    </Box>
  );
}
