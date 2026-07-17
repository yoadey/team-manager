import Box from '@mui/material/Box';
import ButtonBase from '@mui/material/ButtonBase';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { buildTokens, fmtMoney, NEUTRAL } from '@/styles/tokens';
import { Field, PrimaryButton, inputSx, labelSx } from '@/components/ui';
import { useMembersQuery } from '@/features/members';
import type { SheetProps } from '@/sheets/types';
import { penaltyAssignFormSchema, type PenaltyAssignFormValues } from './penaltyAssignFormSchema';
import { useFinanceOverviewQuery } from '../hooks/useFinanceQueries';
import { t } from '@/i18n';

export function PenaltyAssignSheet({ app, sheet }: SheetProps) {
  const { state } = app;
  const tk = buildTokens(state.primaryColor);
  const { data: f } = useFinanceOverviewQuery(app.api, state.activeTeamId);
  const { data: members } = useMembersQuery(app.api, state.activeTeamId);

  const {
    register,
    handleSubmit,
    setValue,
    watch,
    formState: { errors, isSubmitting },
  } = useForm<PenaltyAssignFormValues>({
    resolver: zodResolver(penaltyAssignFormSchema),
    defaultValues: sheet.formInitial as PenaltyAssignFormValues,
    mode: 'onBlur',
  });

  const userId = watch('userId');
  const penaltyId = watch('penaltyId');
  const canSubmit = !!userId && !!penaltyId;

  const onSubmit = async (values: PenaltyAssignFormValues) => {
    try {
      await app.savePenaltyAssign(values);
    } catch {
      // Ignored
    }
  };

  const penOpts = (
    <Box key="po">
      <Box key="l" sx={labelSx}>
        {t('finances.assignPenalty')}
      </Box>
      {errors.penaltyId ? (
        <Box sx={{ fontSize: '12px', color: NEUTRAL.error, mb: '6px' }}>{errors.penaltyId.message}</Box>
      ) : null}
      <Box
        key="b"
        role="radiogroup"
        aria-label={t('finances.assignPenalty')}
        sx={{ display: 'flex', flexDirection: 'column', gap: '8px' }}
      >
        {(f ? f.penalties : []).map((p) => {
          const sel = penaltyId === p.id;
          return (
            <ButtonBase
              key={p.id}
              type="button"
              role="radio"
              aria-checked={sel}
              onClick={() => {
                setValue('penaltyId', p.id, { shouldValidate: true });
              }}
              sx={{
                display: 'flex',
                alignItems: 'center',
                gap: '11px',
                p: '12px 14px',
                borderRadius: '13px',
                cursor: 'pointer',
                textAlign: 'left',
                justifyContent: 'flex-start',
                border: '1.5px solid ' + (sel ? tk.primary : NEUTRAL.line3),
                background: sel ? tk.primaryContainer : NEUTRAL.card,
              }}
            >
              <Box
                component="span"
                key="r"
                sx={{
                  width: '18px',
                  height: '18px',
                  borderRadius: '50%',
                  flex: '0 0 auto',
                  border: '2px solid ' + (sel ? tk.primary : NEUTRAL.faint),
                  background: sel ? tk.primary : NEUTRAL.card,
                  boxShadow: sel ? 'inset 0 0 0 3px #fff' : 'none',
                }}
              />
              <Box
                component="span"
                key="lb"
                sx={{ flex: 1, fontSize: '14px', fontWeight: 600, color: NEUTRAL.onSurfaceVariant }}
              >
                {p.label}
              </Box>
              <Box component="b" key="a" sx={{ fontSize: '14px', color: tk.primary }}>
                {fmtMoney(p.amount)}
              </Box>
            </ButtonBase>
          );
        })}
      </Box>
    </Box>
  );

  const memSel = (
    <Field label={t('finances.assignPerson')} required error={!!errors.userId} errorText={errors.userId?.message}>
      <select style={inputSx} {...register('userId')}>
        <option key="_" value="">
          {t('finances.assignPersonPlaceholder')}
        </option>
        {(members || []).map((m) => (
          <option key={m.userId} value={m.userId}>
            {m.name}
          </option>
        ))}
      </select>
    </Field>
  );

  return (
    <Box
      component="form"
      onSubmit={handleSubmit(onSubmit)}
      sx={{ display: 'flex', flexDirection: 'column', gap: '16px' }}
    >
      {memSel}
      {penOpts}
      <PrimaryButton
        label={t('finances.assignSave')}
        onClick={handleSubmit(onSubmit)}
        busy={isSubmitting || app.state.savingPenaltyAssign}
        disabled={!canSubmit}
      />
    </Box>
  );
}
