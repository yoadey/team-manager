import Box from '@mui/material/Box';
import ButtonBase from '@mui/material/ButtonBase';
import { useForm, type UseFormRegister, type FieldErrors } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { buildTokens, typeMeta, NEUTRAL } from '@/styles/tokens';
import { Field, labelSx, PrimaryButton, Sym, TextArea, TextInput } from '@/components/ui';
import type { Role } from '@/types';
import type { SheetProps } from '@/sheets/types';
import { eventFormSchema, type EventFormValues } from './eventFormSchema';
import { t } from '@/i18n';

type Tokens = ReturnType<typeof buildTokens>;
type EventTypeValue = EventFormValues['type'];

const TYPE_LABEL_KEYS: Record<EventTypeValue, string> = {
  training: 'events.typeTraining',
  auftritt: 'events.typeAuftritt',
  event: 'events.typeEvent',
};

function EventTypeSelector({ type, onSelect }: { type: EventTypeValue; onSelect: (tp: EventTypeValue) => void }) {
  return (
    <Box sx={{ display: 'flex', gap: '8px' }}>
      {(['training', 'auftritt', 'event'] as const).map((tp) => {
        const meta = typeMeta(tp);
        const sel = type === tp;
        return (
          <ButtonBase
            key={tp}
            type="button"
            onClick={() => onSelect(tp)}
            aria-pressed={sel}
            sx={{
              flex: 1,
              display: 'flex',
              flexDirection: 'column',
              alignItems: 'center',
              gap: '4px',
              p: '11px 6px',
              borderRadius: '13px',
              cursor: 'pointer',
              fontSize: '12px',
              fontWeight: 600,
              border: '1.5px solid ' + (sel ? meta.color : NEUTRAL.line3),
              background: sel ? meta.bg : NEUTRAL.card,
              color: sel ? meta.color : NEUTRAL.secondary,
            }}
          >
            <Sym name={meta.icon} size={18} color={sel ? meta.color : NEUTRAL.secondary} />
            {t(TYPE_LABEL_KEYS[tp])}
          </ButtonBase>
        );
      })}
    </Box>
  );
}

const RESPONSE_MODE_DEFS: readonly { value: 'opt_in' | 'opt_out'; icon: string }[] = [
  { value: 'opt_in', icon: 'login' },
  { value: 'opt_out', icon: 'logout' },
];

function ResponseModeSelector({
  responseMode,
  tk,
  onSelect,
}: {
  responseMode: string | undefined;
  tk: Tokens;
  onSelect: (mode: 'opt_in' | 'opt_out') => void;
}) {
  return (
    <Box sx={{ display: 'flex', gap: '8px' }}>
      {RESPONSE_MODE_DEFS.map(({ value, icon }) => {
        const sel = responseMode === value;
        const label = value === 'opt_in' ? t('events.modeOptIn') : t('events.modeOptOut');
        const desc = value === 'opt_in' ? t('events.modeOptInDesc') : t('events.modeOptOutDesc');
        return (
          <ButtonBase
            key={value}
            type="button"
            onClick={() => onSelect(value)}
            aria-pressed={sel}
            sx={{
              flex: 1,
              display: 'flex',
              flexDirection: 'column',
              gap: '4px',
              p: '12px',
              borderRadius: '13px',
              cursor: 'pointer',
              textAlign: 'left',
              alignItems: 'stretch',
              justifyContent: 'flex-start',
              border: '1.5px solid ' + (sel ? tk.primary : NEUTRAL.line3),
              background: sel ? tk.primaryContainer : NEUTRAL.card,
            }}
          >
            <Box
              sx={{
                display: 'flex',
                alignItems: 'center',
                gap: '6px',
                fontSize: '13px',
                fontWeight: 700,
                color: sel ? tk.onPrimaryContainer : NEUTRAL.onSurfaceVariant,
              }}
            >
              <Sym name={icon} size={17} color={sel ? tk.onPrimaryContainer : NEUTRAL.onSurfaceVariant} />
              {label}
            </Box>
            <Box sx={{ fontSize: '11px', color: sel ? tk.onPrimaryContainer : NEUTRAL.faint, lineHeight: 1.4 }}>
              {desc}
            </Box>
          </ButtonBase>
        );
      })}
    </Box>
  );
}

function MeetTimeToggle({ checked, tk, onToggle }: { checked: boolean; tk: Tokens; onToggle: () => void }) {
  return (
    <ButtonBase
      role="checkbox"
      aria-checked={checked}
      aria-label={t('events.meetTimeMandatory')}
      onClick={onToggle}
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
        justifyContent: 'flex-start',
      }}
    >
      <Box
        component="span"
        sx={{
          width: '22px',
          height: '22px',
          borderRadius: '7px',
          background: checked ? tk.primary : NEUTRAL.card,
          border: '2px solid ' + (checked ? tk.primary : NEUTRAL.faint),
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          flex: '0 0 auto',
        }}
      >
        {checked ? <Sym name="check" size={16} color="#fff" /> : null}
      </Box>
      <Box component="span" sx={{ flex: 1, textAlign: 'left', fontSize: '14px', fontWeight: 500 }}>
        {t('events.meetTimeMandatory')}
      </Box>
    </ButtonBase>
  );
}

function RecurringSection({
  show,
  recurring,
  tk,
  register,
  errors,
  onToggle,
}: {
  show: boolean;
  recurring: boolean;
  tk: Tokens;
  register: UseFormRegister<EventFormValues>;
  errors: FieldErrors<EventFormValues>;
  onToggle: () => void;
}) {
  if (!show) return null;
  return (
    <Box sx={{ borderTop: `1px solid ${NEUTRAL.line2}`, pt: '14px' }}>
      <ButtonBase
        role="switch"
        aria-checked={recurring}
        aria-label={t('events.recurWeekly')}
        onClick={onToggle}
        sx={{
          display: 'flex',
          alignItems: 'center',
          gap: '12px',
          width: '100%',
          p: '4px 2px',
          cursor: 'pointer',
          background: 'transparent',
          border: 'none',
          justifyContent: 'flex-start',
        }}
      >
        <Sym name="repeat" size={20} color={NEUTRAL.secondary} />
        <Box component="span" sx={{ flex: 1, textAlign: 'left', fontSize: '14px', fontWeight: 500 }}>
          {t('events.recurWeekly')}
        </Box>
        <Box
          component="span"
          sx={{
            width: '44px',
            height: '26px',
            borderRadius: '999px',
            background: recurring ? tk.primary : NEUTRAL.inputBorder,
            position: 'relative',
            flex: '0 0 auto',
          }}
        >
          <Box
            component="span"
            sx={{
              position: 'absolute',
              top: '3px',
              left: recurring ? '21px' : '3px',
              width: '20px',
              height: '20px',
              borderRadius: '50%',
              background: NEUTRAL.card,
              transition: 'left .2s',
            }}
          />
        </Box>
      </ButtonBase>
      {recurring ? (
        <Box sx={{ mt: '10px' }}>
          <Field label={t('events.recurWeeks')} error={!!errors.repeatWeeks} errorText={errors.repeatWeeks?.message}>
            <TextInput type="number" min="2" max="26" {...register('repeatWeeks')} />
          </Field>
        </Box>
      ) : null}
    </Box>
  );
}

function NominatedRolesSelector({
  roles,
  selectedIds,
  onToggle,
}: {
  roles: Role[];
  selectedIds: string[];
  onToggle: (roleId: string) => void;
}) {
  return (
    <Box sx={{ borderTop: `1px solid ${NEUTRAL.line2}`, pt: '14px' }}>
      <Box sx={labelSx}>{t('events.nominatedRoles')}</Box>
      <Box sx={{ fontSize: '12px', color: NEUTRAL.faint, m: '-2px 0 9px', lineHeight: 1.45 }}>
        {t('events.nominatedRolesHint')}
      </Box>
      <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: '8px' }}>
        {roles.map((r) => {
          const sel = selectedIds.includes(r.id);
          return (
            <ButtonBase
              key={r.id}
              role="checkbox"
              aria-checked={sel}
              aria-label={r.name}
              onClick={() => onToggle(r.id)}
              sx={{
                display: 'inline-flex',
                alignItems: 'center',
                gap: '7px',
                p: '8px 13px',
                borderRadius: '999px',
                cursor: 'pointer',
                fontSize: '13px',
                fontWeight: 600,
                border: '1.5px solid ' + (sel ? r.color : NEUTRAL.inputBorder),
                background: sel ? r.color + '1A' : NEUTRAL.card,
                color: sel ? r.color : NEUTRAL.faint,
              }}
            >
              <Box
                component="span"
                sx={{ width: '9px', height: '9px', borderRadius: '50%', background: sel ? r.color : NEUTRAL.faint }}
              />
              {r.name}
              {sel ? <Sym name="check" size={16} color={r.color} /> : null}
            </ButtonBase>
          );
        })}
      </Box>
    </Box>
  );
}

function SeriesEditSubmit({
  tk,
  disabled,
  onSubmitSingle,
  onSubmitSeries,
}: {
  tk: Tokens;
  disabled: boolean;
  onSubmitSingle: () => void;
  onSubmitSeries: () => void;
}) {
  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: '9px', mt: '4px' }}>
      <Box sx={{ display: 'flex', alignItems: 'center', gap: '7px', fontSize: '12px', color: NEUTRAL.secondary, fontWeight: 600 }}>
        <Sym name="repeat" size={16} color={NEUTRAL.faint} />
        {t('events.seriesHint')}
      </Box>
      <Box sx={{ display: 'flex', gap: '10px' }}>
        <ButtonBase
          type="button"
          onClick={onSubmitSingle}
          disabled={disabled}
          sx={{
            flex: 1,
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            gap: '8px',
            p: '13px',
            borderRadius: '13px',
            border: '1px solid ' + tk.primary,
            background: NEUTRAL.card,
            color: tk.primary,
            fontWeight: 700,
            fontSize: '14px',
            cursor: 'pointer',
          }}
        >
          <Sym name="event" size={18} color={tk.primary} />
          {t('events.seriesSingle')}
        </ButtonBase>
        <ButtonBase
          type="button"
          onClick={onSubmitSeries}
          disabled={disabled}
          sx={{
            flex: 1,
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            gap: '8px',
            p: '13px',
            borderRadius: '13px',
            border: 'none',
            background: tk.primary,
            color: tk.onPrimary,
            fontWeight: 700,
            fontSize: '14px',
            cursor: 'pointer',
          }}
        >
          <Sym name="repeat" size={18} color={tk.onPrimary} />
          {t('events.seriesAll')}
        </ButtonBase>
      </Box>
    </Box>
  );
}

export function EventFormSheet({ app, sheet }: SheetProps) {
  const { state } = app;
  const tk = buildTokens(state.primaryColor);

  const {
    register,
    handleSubmit,
    setValue,
    watch,
    formState: { errors, isSubmitting },
  } = useForm<EventFormValues>({
    resolver: zodResolver(eventFormSchema),
    mode: 'onBlur',
    defaultValues: sheet.formInitial as EventFormValues,
  });

  const type = watch('type');
  const responseMode = watch('responseMode');
  const meetTimeMandatory = watch('meetTimeMandatory');
  const recurring = watch('recurring');
  const nominatedRoleIds = watch('nominatedRoleIds') || [];
  const seriesId = watch('seriesId');
  const titleVal = watch('title');
  const dateVal = watch('date');
  const canSubmit = !!titleVal?.trim() && !!dateVal;

  const toggleRole = (roleId: string) => {
    const next = nominatedRoleIds.includes(roleId)
      ? nominatedRoleIds.filter((id) => id !== roleId)
      : [...nominatedRoleIds, roleId];
    setValue('nominatedRoleIds', next, { shouldValidate: true });
  };

  const onSubmit = async (values: EventFormValues, scope: 'single' | 'series' = 'single') => {
    try {
      await app.saveEvent(values, scope);
    } catch {
      // Ignored
    }
  };

  const showSeriesButtons = sheet.mode === 'edit' && !!seriesId;
  const submitting = isSubmitting || app.state.savingEvent;

  return (
    <Box
      component="form"
      onSubmit={handleSubmit((v) => onSubmit(v, 'single'))}
      sx={{ display: 'flex', flexDirection: 'column', gap: '16px' }}
    >
      <Box>
        <Box sx={labelSx}>{t('events.eventType')}</Box>
        <EventTypeSelector type={type} onSelect={(tp) => setValue('type', tp, { shouldValidate: true })} />
      </Box>
      <Field label={t('events.fieldTitle')} required error={!!errors.title} errorText={errors.title?.message}>
        <TextInput placeholder={t('events.fieldTitlePlaceholder')} maxLength={255} {...register('title')} />
      </Field>
      <Field label={t('events.fieldDate')} required error={!!errors.date} errorText={errors.date?.message}>
        <TextInput type="date" {...register('date')} />
      </Field>
      <Box sx={{ display: 'flex', gap: '10px' }}>
        <Field label={t('events.fieldMeetTime')} error={!!errors.meetT} errorText={errors.meetT?.message}>
          <TextInput type="time" {...register('meetT')} />
        </Field>
        <Field label={t('events.fieldStartTime')} error={!!errors.startT} errorText={errors.startT?.message}>
          <TextInput type="time" {...register('startT')} />
        </Field>
        <Field label={t('events.fieldEndTime')} error={!!errors.endT} errorText={errors.endT?.message}>
          <TextInput type="time" {...register('endT')} />
        </Field>
      </Box>
      <MeetTimeToggle
        checked={!!meetTimeMandatory}
        tk={tk}
        onToggle={() => setValue('meetTimeMandatory', !meetTimeMandatory, { shouldValidate: true })}
      />
      <Box>
        <Box sx={labelSx}>{t('events.responseMode')}</Box>
        <ResponseModeSelector
          responseMode={responseMode}
          tk={tk}
          onSelect={(mode) => setValue('responseMode', mode, { shouldValidate: true })}
        />
      </Box>
      <NominatedRolesSelector roles={state.roles} selectedIds={nominatedRoleIds} onToggle={toggleRole} />
      <Field label={t('events.fieldLocation')} error={!!errors.location} errorText={errors.location?.message}>
        <TextInput placeholder={t('events.fieldLocationPlaceholder')} maxLength={255} {...register('location')} />
      </Field>
      <Field label={t('events.fieldNote')} error={!!errors.note} errorText={errors.note?.message}>
        <TextArea
          placeholder={t('events.fieldNotePlaceholder')}
          minHeight={64}
          maxLength={10000}
          {...register('note')}
        />
      </Field>
      <RecurringSection
        show={sheet.mode === 'create'}
        recurring={!!recurring}
        tk={tk}
        register={register}
        errors={errors}
        onToggle={() => setValue('recurring', !recurring, { shouldValidate: true })}
      />
      {showSeriesButtons ? (
        <SeriesEditSubmit
          tk={tk}
          disabled={submitting}
          onSubmitSingle={handleSubmit((v) => onSubmit(v, 'single'))}
          onSubmitSeries={handleSubmit((v) => onSubmit(v, 'series'))}
        />
      ) : (
        <PrimaryButton
          label={sheet.mode === 'edit' ? t('events.saveChanges') : t('events.createEvent')}
          onClick={handleSubmit((v) => onSubmit(v, 'single'))}
          busy={submitting}
          disabled={!canSubmit}
        />
      )}
    </Box>
  );
}
