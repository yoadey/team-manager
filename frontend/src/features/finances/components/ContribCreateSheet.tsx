import Box from '@mui/material/Box';
import ButtonBase from '@mui/material/ButtonBase';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { buildTokens, NEUTRAL } from '@/styles/tokens';
import { Av, Field, PrimaryButton, Sym, TextInput } from '@/components/ui';
import { useMembersQuery } from '@/features/members';
import type { SheetProps } from '@/sheets/types';
import { contribCreateFormSchema, type ContribCreateFormValues } from './contribCreateFormSchema';
import { MAX_MONEY_AMOUNT_EUROS, validateMoneyAmount } from '@/utils/validation';
import { t } from '@/i18n';

export function ContribCreateSheet({ app, sheet }: SheetProps) {
  const { state } = app;
  const tk = buildTokens(state.primaryColor);
  const { data: members } = useMembersQuery(app.api, state.activeTeamId);

  const {
    register,
    handleSubmit,
    setValue,
    watch,
    formState: { errors, isSubmitting },
  } = useForm<ContribCreateFormValues>({
    resolver: zodResolver(contribCreateFormSchema),
    defaultValues: sheet.formInitial as ContribCreateFormValues,
    mode: 'onBlur',
  });

  const label = watch('label');
  const amount = watch('amount');
  const userIds = watch('userIds') || [];
  const allMembers = members || [];
  const allSelected = allMembers.length > 0 && userIds.length === allMembers.length;
  const canSubmit =
    !!label?.trim() &&
    validateMoneyAmount(amount, { positive: true, max: MAX_MONEY_AMOUNT_EUROS }).ok &&
    userIds.length > 0;

  const onSubmit = async (values: ContribCreateFormValues) => {
    try {
      await app.saveContribCreate(values);
    } catch {
      // Ignored
    }
  };

  const toggleMember = (userId: string) => {
    const next = userIds.includes(userId) ? userIds.filter((id) => id !== userId) : [...userIds, userId];
    setValue('userIds', next, { shouldValidate: true });
  };

  const toggleAll = () => {
    setValue('userIds', allSelected ? [] : allMembers.map((m) => m.userId), { shouldValidate: true });
  };

  const memberList = (
    <Box key="ml">
      <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', mb: '8px' }}>
        <Box sx={labelFieldSx}>{t('finances.contribCreateMembers')}</Box>
        <ButtonBase
          type="button"
          onClick={toggleAll}
          sx={{ fontSize: '12px', fontWeight: 700, color: tk.primary, p: '4px 6px', borderRadius: '8px' }}
        >
          {allSelected ? t('finances.contribCreateDeselectAll') : t('finances.contribCreateSelectAll')}
        </ButtonBase>
      </Box>
      {errors.userIds ? (
        <Box sx={{ fontSize: '12px', color: NEUTRAL.error, mb: '6px' }}>{errors.userIds.message}</Box>
      ) : null}
      <Box
        key="list"
        role="group"
        aria-label={t('finances.contribCreateMembers')}
        sx={{ display: 'flex', flexDirection: 'column', gap: '6px', maxHeight: '320px', overflowY: 'auto' }}
      >
        {allMembers.map((m) => {
          const sel = userIds.includes(m.userId);
          return (
            <ButtonBase
              key={m.userId}
              type="button"
              role="checkbox"
              aria-checked={sel}
              onClick={() => toggleMember(m.userId)}
              sx={{
                display: 'flex',
                alignItems: 'center',
                gap: '11px',
                p: '9px 12px',
                borderRadius: '12px',
                cursor: 'pointer',
                textAlign: 'left',
                justifyContent: 'flex-start',
                border: '1.5px solid ' + (sel ? tk.primary : NEUTRAL.line3),
                background: sel ? tk.primaryContainer : NEUTRAL.card,
              }}
            >
              <Box
                component="span"
                sx={{
                  width: '18px',
                  height: '18px',
                  borderRadius: '5px',
                  flex: '0 0 auto',
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'center',
                  border: '2px solid ' + (sel ? tk.primary : NEUTRAL.faint),
                  background: sel ? tk.primary : NEUTRAL.card,
                }}
              >
                {sel ? <Sym name="check" size={13} color="#fff" /> : null}
              </Box>
              <Av name={m.name} photo={m.photo} color={m.avatarColor} size={28} />
              <Box component="span" sx={{ flex: 1, fontSize: '13px', fontWeight: 600, color: NEUTRAL.onSurfaceVariant }}>
                {m.name}
              </Box>
            </ButtonBase>
          );
        })}
      </Box>
    </Box>
  );

  return (
    <Box
      component="form"
      onSubmit={handleSubmit(onSubmit)}
      sx={{ display: 'flex', flexDirection: 'column', gap: '16px' }}
    >
      <Field label={t('finances.contribFieldLabel')} required error={!!errors.label} errorText={errors.label?.message}>
        <TextInput placeholder={t('finances.contribFieldLabelPlaceholder')} maxLength={255} {...register('label')} />
      </Field>
      <Field
        label={t('finances.contribFieldAmount')}
        required
        error={!!errors.amount}
        errorText={errors.amount?.message}
      >
        <TextInput type="number" max={MAX_MONEY_AMOUNT_EUROS} {...register('amount')} />
      </Field>
      <Field label={t('finances.contribFieldDueDate')} error={!!errors.dueDate} errorText={errors.dueDate?.message}>
        <TextInput type="date" {...register('dueDate')} />
      </Field>
      {memberList}
      <PrimaryButton
        label={t('finances.contribCreateSave')}
        onClick={handleSubmit(onSubmit)}
        busy={isSubmitting || app.state.savingContribCreate}
        disabled={!canSubmit}
      />
    </Box>
  );
}

const labelFieldSx = { fontSize: '13px', fontWeight: 700, color: NEUTRAL.onSurfaceVariant };
