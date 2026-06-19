import Box from '@mui/material/Box';
import ButtonBase from '@mui/material/ButtonBase';
import { buildTokens, NEUTRAL } from '@/styles/tokens';
import { Field, PrimaryButton, Sym, TextInput, inputSx } from '@/components/ui';
import type { SheetProps } from '@/sheets/types';
import { t } from '@/i18n';

export function TxFormSheet({ app, sheet }: SheetProps) {
  const { state } = app;
  const tk = buildTokens(state.primaryColor);
  const F = app.state.form;
  const errs = state.formErrors;
  const edit = sheet.mode === 'edit';

  const typeDefs: [string, string, string, string, string][] = [
    ['income', t('finances.txIncome'), 'south_west', NEUTRAL.success, NEUTRAL.successBg],
    ['expense', t('finances.txExpense'), 'north_east', NEUTRAL.error, NEUTRAL.errorBg],
  ];
  const typeBtns = typeDefs.map(([v, l, ic, c, bg]) => {
    const sel = F.type === v;
    return (
      <ButtonBase
        key={v}
        onClick={() => app.setFormVal({ type: v })}
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
          border: '1.5px solid ' + (sel ? c : '#E0E2EA'),
          background: sel ? bg : '#fff',
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
  ].sort((a, b) => a.localeCompare(b, 'de'));

  const catField = (
    <Field label={t('finances.txFieldCategory')}>
      <Box>
        <input
          key="i"
          name="category"
          list="tvCatList"
          autoComplete="off"
          value={F.category == null ? '' : F.category}
          onChange={app.onFormInput}
          placeholder={t('finances.txCategoryPlaceholder')}
          style={inputSx}
        />
        <datalist key="dl" id="tvCatList">
          {cats.map((c) => (
            <option key={c} value={c} />
          ))}
        </datalist>
        {cats.length ? (
          <Box key="qp" sx={{ display: 'flex', flexWrap: 'wrap', gap: '6px', mt: '8px' }}>
            {cats.map((c) => {
              const sel = F.category === c;
              return (
                <ButtonBase
                  key={c}
                  onClick={() => app.setFormVal({ category: c })}
                  sx={{
                    p: '5px 11px',
                    borderRadius: '999px',
                    fontSize: '12px',
                    fontWeight: 600,
                    cursor: 'pointer',
                    border: '1px solid ' + (sel ? tk.primary : '#D0D2DA'),
                    background: sel ? tk.primaryContainer : '#fff',
                    color: sel ? tk.onPrimaryContainer : '#44474E',
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
    </Field>
  );

  const del = edit ? (
    <ButtonBase
      key="del"
      onClick={() =>
        app.askConfirm({
          title: t('finances.txDeleteConfirmTitle'),
          message: t('finances.txDeleteConfirmMsg', { title: String(F.title || t('finances.txDelete')) }),
          confirmLabel: t('common.delete'),
          danger: true,
          onConfirm: async () => {
            await app.deleteTx(F.id);
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
        background: '#FFF4F3',
        color: NEUTRAL.error,
        fontWeight: 600,
        cursor: 'pointer',
      }}
    >
      <Sym name="delete" size={19} color={NEUTRAL.error} />
      {t('finances.txDelete')}
    </ButtonBase>
  ) : null;

  const validateTitle = () =>
    app.setFormErrors({ title: String(F.title ?? '').trim() ? '' : t('finances.txFieldTitleError') });
  const validateAmount = () => {
    const raw = String(F.amount ?? '')
      .trim()
      .replace(',', '.');
    const n = Number(raw);
    app.setFormErrors({
      amount: !raw
        ? t('finances.txFieldAmountError')
        : !Number.isFinite(n) || n <= 0
          ? t('finances.txFieldAmountErrorPositive')
          : '',
    });
  };

  const canSubmit =
    !!(F.title as string | undefined)?.trim() &&
    (() => {
      const raw = String(F.amount ?? '')
        .trim()
        .replace(',', '.');
      const n = Number(raw);
      return raw && Number.isFinite(n) && n > 0;
    })();

  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: '16px' }}>
      <Box sx={{ display: 'flex', gap: '8px' }}>{typeBtns}</Box>
      <Field label={t('finances.txFieldTitle')} required error={!!errs.title} errorText={errs.title}>
        <TextInput name="title" placeholder={t('finances.txFieldTitlePlaceholder')} onBlur={validateTitle} />
      </Field>
      <Field label={t('finances.txFieldAmount')} required error={!!errs.amount} errorText={errs.amount}>
        <TextInput name="amount" type="number" onBlur={validateAmount} />
      </Field>
      {catField}
      <PrimaryButton
        label={edit ? t('finances.txSaveEdit') : t('finances.txSave')}
        onClick={() => app.saveTx()}
        busy={app.state.busy === 'save'}
        disabled={!canSubmit}
      />
      {del}
    </Box>
  );
}
