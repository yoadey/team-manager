import Box from '@mui/material/Box';
import ButtonBase from '@mui/material/ButtonBase';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { buildTokens, fmtMoney, NEUTRAL } from '@/styles/tokens';
import { Field, PrimaryButton, Sym, TextArea, TextInput, inputSx } from '@/components/ui';
import type { AppContextValue } from '@/context/AppContext';
import type { SheetProps } from '@/sheets/types';
import { txFormSchema, type TxFormValues } from './txFormSchema';
import { LinkedPaymentPicker } from './LinkedPaymentPicker';
import { useFinanceOverviewQuery } from '../hooks/useFinanceQueries';
import type { Contribution, PenaltyAssignment } from '../types';
import { MAX_MONEY_AMOUNT_EUROS, validateMoneyAmount } from '@/utils/validation';
import { getIntlLocale, t } from '@/i18n';

/** Read-only "linked to" info shown when reopening an already-linked
 * transaction for editing -- see openspec/changes/finance-detail-linked-entries. */
function TxLinkedInfo({
  app,
  tk,
  contribution,
  assignment,
}: {
  app: AppContextValue;
  tk: ReturnType<typeof buildTokens>;
  contribution?: Contribution | undefined;
  assignment?: PenaltyAssignment | undefined;
}) {
  if (!contribution && !assignment) return null;
  const target = contribution || assignment!;
  return (
    <ButtonBase
      type="button"
      onClick={() => (contribution ? app.openContribDetail(contribution) : app.openPenaltyAssign(assignment))}
      sx={{
        display: 'flex',
        alignItems: 'center',
        gap: '9px',
        p: '11px 13px',
        borderRadius: '13px',
        border: `1px solid ${NEUTRAL.line}`,
        background: NEUTRAL.card,
        cursor: 'pointer',
        textAlign: 'left',
        justifyContent: 'flex-start',
      }}
    >
      <Sym name="link" size={17} color={tk.primary} />
      <Box sx={{ flex: 1, fontSize: '13px', fontWeight: 600, color: NEUTRAL.onSurfaceVariant }}>
        {contribution
          ? t('finances.txLinkedContribution', { label: contribution.label })
          : t('finances.txLinkedPenalty', { label: assignment!.label || '' })}
      </Box>
      <Box component="span" sx={{ fontSize: '13px', fontWeight: 700, color: tk.primary }}>
        {fmtMoney(target.amount || 0)}
      </Box>
    </ButtonBase>
  );
}

function TxLinkedPicker({
  show,
  tk,
  finances,
  contributionId,
  penaltyAssignmentId,
  setValue,
}: {
  show: boolean;
  tk: ReturnType<typeof buildTokens>;
  finances: { contributions: Contribution[]; assignments: PenaltyAssignment[] } | undefined;
  contributionId: string | undefined;
  penaltyAssignmentId: string | undefined;
  setValue: ReturnType<typeof useForm<TxFormValues>>['setValue'];
}) {
  if (!show) return null;
  // Only offered when creating a new income transaction -- linking only
  // happens at creation time (see CreateTransactionRequest.contributionId's
  // doc comment), and only fees/fines not yet fully paid are worth picking.
  const openContribs = (finances?.contributions || []).filter((c) => c.status !== 'paid' && !c.archived);
  const openAssignments = (finances?.assignments || []).filter((a) => !a.paid);
  return (
    <LinkedPaymentPicker
      tk={tk}
      contributions={openContribs}
      assignments={openAssignments}
      contributionId={contributionId}
      penaltyAssignmentId={penaltyAssignmentId}
      onSelectContribution={(id) => {
        setValue('contributionId', id, { shouldValidate: true });
        setValue('penaltyAssignmentId', '', { shouldValidate: true });
      }}
      onSelectPenalty={(id) => {
        setValue('penaltyAssignmentId', id, { shouldValidate: true });
        setValue('contributionId', '', { shouldValidate: true });
      }}
      onClear={() => {
        setValue('contributionId', '', { shouldValidate: true });
        setValue('penaltyAssignmentId', '', { shouldValidate: true });
      }}
    />
  );
}

function TxCategoryField({
  cats,
  category,
  errorText,
  register,
  setValue,
  tk,
}: {
  cats: string[];
  category: string | undefined;
  errorText: string | undefined;
  register: ReturnType<typeof useForm<TxFormValues>>['register'];
  setValue: ReturnType<typeof useForm<TxFormValues>>['setValue'];
  tk: ReturnType<typeof buildTokens>;
}) {
  return (
    <Box>
      <Field label={t('finances.txFieldCategory')} error={!!errorText} errorText={errorText}>
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
}

function TxDeleteButton({ app, sheet, title }: { app: AppContextValue; sheet: SheetProps['sheet']; title: string | undefined }) {
  if (sheet.mode !== 'edit') return null;
  return (
    <ButtonBase
      key="del"
      type="button"
      onClick={() =>
        app.askConfirm({
          title: t('finances.txDeleteConfirmTitle'),
          message: t('finances.txDeleteConfirmMsg', {
            title: String(title || t('finances.txDelete')),
          }),
          confirmLabel: t('common.delete'),
          danger: true,
          onConfirm: async () => {
            await app.deleteTx((sheet.formInitial as TxFormValues).id!);
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
  );
}

export function TxFormSheet({ app, sheet }: SheetProps) {
  const { state } = app;
  const tk = buildTokens(state.primaryColor);
  const { data: finances } = useFinanceOverviewQuery(app.api, state.activeTeamId);
  const edit = sheet.mode === 'edit';

  const {
    register,
    handleSubmit,
    setValue,
    watch,
    formState: { errors, isSubmitting },
  } = useForm<TxFormValues>({
    resolver: zodResolver(txFormSchema),
    defaultValues: sheet.formInitial as TxFormValues,
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

  const contributionId = watch('contributionId');
  const penaltyAssignmentId = watch('penaltyAssignmentId');

  const linkedContribution = edit
    ? ((finances && finances.contributions) || []).find((c) => c.id === contributionId)
    : undefined;
  const linkedAssignment = edit
    ? ((finances && finances.assignments) || []).find((a) => a.id === penaltyAssignmentId)
    : undefined;

  const cats = [...new Set(((finances && finances.transactions) || []).map((x) => x.category).filter(Boolean))].sort(
    (a, b) => a.localeCompare(b, getIntlLocale()),
  );

  return (
    <Box
      component="form"
      onSubmit={handleSubmit(onSubmit)}
      sx={{ display: 'flex', flexDirection: 'column', gap: '16px' }}
    >
      <Box sx={{ display: 'flex', gap: '8px' }}>{typeBtns}</Box>
      <Field label={t('finances.txFieldTitle')} required error={!!errors.title} errorText={errors.title?.message}>
        <TextInput placeholder={t('finances.txFieldTitlePlaceholder')} maxLength={255} {...register('title')} />
      </Field>
      <Field label={t('finances.txFieldAmount')} required error={!!errors.amount} errorText={errors.amount?.message}>
        <TextInput type="number" max={MAX_MONEY_AMOUNT_EUROS} {...register('amount')} />
      </Field>
      <Field label={t('finances.txFieldDate')} required error={!!errors.date} errorText={errors.date?.message}>
        <TextInput type="date" {...register('date')} />
      </Field>
      <TxLinkedPicker
        show={!edit && type === 'income'}
        tk={tk}
        finances={finances}
        contributionId={contributionId}
        penaltyAssignmentId={penaltyAssignmentId}
        setValue={setValue}
      />
      <TxLinkedInfo app={app} tk={tk} contribution={linkedContribution} assignment={linkedAssignment} />
      <TxCategoryField
        cats={cats}
        category={category}
        errorText={errors.category?.message}
        register={register}
        setValue={setValue}
        tk={tk}
      />
      <Field label={t('finances.txFieldNote')} error={!!errors.note} errorText={errors.note?.message}>
        <TextArea maxLength={10000} placeholder={t('finances.txFieldNotePlaceholder')} {...register('note')} />
      </Field>
      <PrimaryButton
        label={edit ? t('finances.txSaveEdit') : t('finances.txSave')}
        onClick={handleSubmit(onSubmit)}
        busy={isSubmitting || app.state.savingTx}
        disabled={!canSubmit}
      />
      <TxDeleteButton app={app} sheet={sheet} title={title} />
    </Box>
  );
}
