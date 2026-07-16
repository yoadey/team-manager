import Box from '@mui/material/Box';
import ButtonBase from '@mui/material/ButtonBase';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { buildTokens, NEUTRAL } from '@/styles/tokens';
import { Field, PrimaryButton, Sym, TextInput, TextArea, inputSx } from '@/components/ui';
import type { SheetProps } from '@/sheets/types';
import { txFormSchema, type TxFormValues } from './txFormSchema';
import { MAX_MONEY_AMOUNT_EUROS, validateMoneyAmount } from '@/utils/validation';
import { getIntlLocale, t } from '@/i18n';

export function TxFormSheet({ app, sheet }: SheetProps) {
  const { state } = app;
  const errs = state.formErrors || {};
  const tk = buildTokens(state.primaryColor);
  const edit = sheet.mode === 'edit';

  const {
    register,
    handleSubmit,
    setValue,
    watch,
    formState: { errors, isSubmitting },
  } = useForm<TxFormValues>({
    resolver: zodResolver(txFormSchema),
    defaultValues: state.form as TxFormValues,
    mode: 'onBlur',
  });

  const type = watch('type');
  const category = watch('category');
  const title = watch('title');
  const amount = watch('amount');
  const canSubmit = !!title?.trim() && validateMoneyAmount(amount, { positive: true, max: MAX_MONEY_AMOUNT_EUROS }).ok;

  const onSubmit = async (values: TxFormValues) => {
    try {
      await app.saveTx(values);
    } catch {
      // Ignored
    }
  };

  const typeDefs: [string, string, string, string, string][] = [
    ['income', t('finances.txIncome'), 'south_west', NEUTRAL.success, NEUTRAL.successBg],
    ['expense', t('finances.txExpense'), 'north_east', NEUTRAL.error, NEUTRAL.errorBg],
  ];
  const typeBtns = typeDefs.map(([v, l, ic, c, bg]) => {
    const sel = type === v;
    return (
      <ButtonBase
        key={v}
        type="button"
        onClick={() => setValue('type', v as 'income' | 'expense', { shouldValidate: true })}
        aria-pressed={sel}
        sx={{
          flex: 1,
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          gap: '7px',
          p: '12px',
          borderRadius: '13px',
          cursor: 'pointer',
          fontSize: '14px',
          fontWeight: 700,
          border: '1.5px solid ' + (sel ? c : NEUTRAL.line3),
          background: sel ? bg : NEUTRAL.card,
          color: sel ? c : NEUTRAL.secondary,
        }}
      >
        <Sym name={ic} size={18} color={sel ? c : NEUTRAL.secondary} />
        {l}
      </ButtonBase>
    );
  });

  const cats = [
    ...new Set(((app.state.finances && app.state.finances.transactions) || []).map((x) => x.category).filter(Boolean)),
  ].sort((a, b) => a.localeCompare(b, getIntlLocale()));

  const catField = (
    <Box>
      <Field label={t('finances.txFieldCategory')} error={!!errors.category || !!errs.category} errorText={errors.category?.message || errs.category}>
        <input
          key="i"
          list="tvCatList"
          autoComplete="off"
          maxLength={255}
          placeholder={t('finances.txCategoryPlaceholder')}
          style={inputSx}
          {...register('category')}
        />
      </Field>
      <datalist key="dl" id="tvCatList">
        {cats.map((c) => (
          <option key={c} value={c} />
        ))}
      </datalist>
      {cats.length ? (
        <Box key="qp" sx={{ display: 'flex', flexWrap: 'wrap', gap: '6px', mt: '8px' }}>
          {cats.map((c) => {
            const sel = category === c;
            return (
              <ButtonBase
                key={c}
                type="button"
                onClick={() => setValue('category', c, { shouldValidate: true })}
                sx={{
                  p: '5px 11px',
                  borderRadius: '999px',
                  fontSize: '12px',
                  fontWeight: 600,
                  cursor: 'pointer',
                  border: '1px solid ' + (sel ? tk.primary : NEUTRAL.inputBorder),
                  background: sel ? tk.primaryContainer : NEUTRAL.card,
                  color: sel ? tk.onPrimaryContainer : NEUTRAL.onSurfaceVariant,
                }}
              >
                {c}
              </ButtonBase>
            );
          })}
        </Box>
      ) : null}
      <Box key="hint" sx={{ fontSize: '11px', color: NEUTRAL.faint, mt: '8px', lineHeight: 1.5 }}>
        {t('finances.txCategoryHint')}
      </Box>
    </Box>
  );

  const del = edit ? (
    <ButtonBase
      key="del"
      type="button"
      onClick={() =>
        app.askConfirm({
          title: t('finances.txDeleteConfirmTitle'),
          message: t('finances.txDeleteConfirmMsg', { title: String((state.form as TxFormValues).title || t('finances.txDelete')) }),
          confirmLabel: t('common.delete'),
          danger: true,
          onConfirm: async () => {
            await app.deleteTx((state.form as TxFormValues).id!);
          },
        })
      }
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
      {t('finances.txDelete')}
    </ButtonBase>
  ) : null;

  return (
    <Box component="form" onSubmit={handleSubmit(onSubmit)} sx={{ display: 'flex', flexDirection: 'column', gap: '16px' }}>
      <Box sx={{ display: 'flex', gap: '8px' }}>{typeBtns}</Box>
      <Field label={t('finances.txFieldTitle')} required error={!!errors.title || !!errs.title} errorText={errors.title?.message || errs.title}>
        <TextInput placeholder={t('finances.txFieldTitlePlaceholder')} maxLength={255} {...register('title')} />
      </Field>
      <Field label={t('finances.txFieldAmount')} required error={!!errors.amount || !!errs.amount} errorText={errors.amount?.message || errs.amount}>
        <TextInput type="number" max={MAX_MONEY_AMOUNT_EUROS} {...register('amount')} />
      </Field>
      {catField}
      <PrimaryButton
        label={edit ? t('finances.txSaveEdit') : t('finances.txSave')}
        onClick={handleSubmit(onSubmit)}
        busy={isSubmitting}
        disabled={!canSubmit}
      />
      {del}
    </Box>
  );
}
