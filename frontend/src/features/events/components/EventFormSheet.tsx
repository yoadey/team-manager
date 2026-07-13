import Box from '@mui/material/Box';
import ButtonBase from '@mui/material/ButtonBase';
import { buildTokens, typeMeta, NEUTRAL } from '@/styles/tokens';
import { Field, labelSx, PrimaryButton, Sym, TextArea, TextInput } from '@/components/ui';
import type { Role } from '@/types';
import type { SheetProps } from '@/sheets/types';
import { formValues } from '@/utils/forms';
import type { EventFormValues } from '../types';
import { t } from '@/i18n';

export function EventFormSheet({ app, sheet }: SheetProps) {
  const { state } = app;
  const tk = buildTokens(state.primaryColor);
  const F = formValues<EventFormValues>(app.state);
  const errs = state.formErrors;

  const req = (field: string, msg: string) => () =>
    app.setFormErrors({ [field]: String(F[field] ?? '').trim() ? '' : msg });

  const typeBtns = (['training', 'auftritt', 'event'] as const).map((tp) => {
    const meta = typeMeta(tp);
    const sel = F.type === tp;
    return (
      <ButtonBase
        key={tp}
        onClick={() => app.setFormVal({ type: tp })}
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
        {tp === 'training'
          ? t('events.typeTraining')
          : tp === 'auftritt'
            ? t('events.typeAuftritt')
            : t('events.typeEvent')}
      </ButtonBase>
    );
  });

  const modeDefs: [string, string, string, string][] = [
    ['opt_in', t('events.modeOptIn'), t('events.modeOptInDesc'), 'login'],
    ['opt_out', t('events.modeOptOut'), t('events.modeOptOutDesc'), 'logout'],
  ];
  const modeBtns = modeDefs.map(([v, l, d, ic]) => {
    const sel = F.responseMode === v;
    return (
      <ButtonBase
        key={v}
        onClick={() => app.setFormVal({ responseMode: v })}
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
          key="h"
          sx={{
            display: 'flex',
            alignItems: 'center',
            gap: '6px',
            fontSize: '13px',
            fontWeight: 700,
            color: sel ? tk.onPrimaryContainer : NEUTRAL.onSurfaceVariant,
          }}
        >
          <Sym name={ic} size={17} color={sel ? tk.onPrimaryContainer : NEUTRAL.onSurfaceVariant} />
          {l}
        </Box>
        <Box key="d" sx={{ fontSize: '11px', color: sel ? tk.onPrimaryContainer : NEUTRAL.faint, lineHeight: 1.4 }}>
          {d}
        </Box>
      </ButtonBase>
    );
  });

  const meetToggle = (
    <ButtonBase
      key="mm"
      role="checkbox"
      aria-checked={!!F.meetTimeMandatory}
      aria-label={t('events.meetTimeMandatory')}
      onClick={() => app.setFormVal({ meetTimeMandatory: !F.meetTimeMandatory })}
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
        key="c"
        component="span"
        sx={{
          width: '22px',
          height: '22px',
          borderRadius: '7px',
          background: F.meetTimeMandatory ? tk.primary : NEUTRAL.card,
          border: '2px solid ' + (F.meetTimeMandatory ? tk.primary : NEUTRAL.faint),
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          flex: '0 0 auto',
        }}
      >
        {F.meetTimeMandatory ? <Sym name="check" size={16} color="#fff" /> : null}
      </Box>
      <Box key="l" component="span" sx={{ flex: 1, textAlign: 'left', fontSize: '14px', fontWeight: 500 }}>
        {t('events.meetTimeMandatory')}
      </Box>
    </ButtonBase>
  );

  const recur =
    sheet.mode === 'create' ? (
      <Box key="rec" sx={{ borderTop: `1px solid ${NEUTRAL.line2}`, pt: '14px' }}>
        <ButtonBase
          key="tg"
          role="switch"
          aria-checked={!!F.recurring}
          aria-label={t('events.recurWeekly')}
          onClick={() => app.setFormVal({ recurring: !F.recurring })}
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
          <Box key="l" component="span" sx={{ flex: 1, textAlign: 'left', fontSize: '14px', fontWeight: 500 }}>
            {t('events.recurWeekly')}
          </Box>
          <Box
            key="sw"
            component="span"
            sx={{
              width: '44px',
              height: '26px',
              borderRadius: '999px',
              background: F.recurring ? tk.primary : NEUTRAL.inputBorder,
              position: 'relative',
              flex: '0 0 auto',
            }}
          >
            <Box
              component="span"
              sx={{
                position: 'absolute',
                top: '3px',
                left: F.recurring ? '21px' : '3px',
                width: '20px',
                height: '20px',
                borderRadius: '50%',
                background: NEUTRAL.card,
                transition: 'left .2s',
              }}
            />
          </Box>
        </ButtonBase>
        {F.recurring ? (
          <Box key="w" sx={{ mt: '10px' }}>
            <Field label={t('events.recurWeeks')}>
              <TextInput name="repeatWeeks" type="number" min="2" max="26" />
            </Field>
          </Box>
        ) : null}
      </Box>
    ) : null;

  const nomSel = (
    <Box key="nomsel" sx={{ borderTop: `1px solid ${NEUTRAL.line2}`, pt: '14px' }}>
      <Box key="l" sx={labelSx}>
        {t('events.nominatedRoles')}
      </Box>
      <Box key="h" sx={{ fontSize: '12px', color: NEUTRAL.faint, m: '-2px 0 9px', lineHeight: 1.45 }}>
        {t('events.nominatedRolesHint')}
      </Box>
      <Box key="b" sx={{ display: 'flex', flexWrap: 'wrap', gap: '8px' }}>
        {state.roles.map((r: Role) => {
          const sel = (F.nominatedRoleIds || []).includes(r.id);
          return (
            <ButtonBase
              key={r.id}
              role="checkbox"
              aria-checked={sel}
              aria-label={r.name}
              onClick={() => app.toggleFormNomRole(r.id)}
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
                key="d"
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

  const canSubmit = !!F.title?.trim() && !!F.date;

  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: '16px' }}>
      <Box key="type">
        <Box key="l" sx={labelSx}>
          {t('events.eventType')}
        </Box>
        <Box key="b" sx={{ display: 'flex', gap: '8px' }}>
          {typeBtns}
        </Box>
      </Box>
      <Field label={t('events.fieldTitle')} required error={!!errs.title} errorText={errs.title}>
        <TextInput
          name="title"
          placeholder={t('events.fieldTitlePlaceholder')}
          onBlur={req('title', t('events.fieldTitleError'))}
          maxLength={255}
        />
      </Field>
      <Field label={t('events.fieldDate')} required error={!!errs.date} errorText={errs.date}>
        <TextInput name="date" type="date" onBlur={req('date', t('events.fieldDateError'))} />
      </Field>
      <Box key="times" sx={{ display: 'flex', gap: '10px' }}>
        <Field label={t('events.fieldMeetTime')}>
          <TextInput name="meetT" type="time" />
        </Field>
        <Field label={t('events.fieldStartTime')}>
          <TextInput name="startT" type="time" />
        </Field>
        <Field label={t('events.fieldEndTime')}>
          <TextInput name="endT" type="time" />
        </Field>
      </Box>
      {meetToggle}
      <Box key="mode">
        <Box key="l" sx={labelSx}>
          {t('events.responseMode')}
        </Box>
        <Box key="b" sx={{ display: 'flex', gap: '8px' }}>
          {modeBtns}
        </Box>
      </Box>
      {nomSel}
      <Field label={t('events.fieldLocation')}>
        <TextInput name="location" placeholder={t('events.fieldLocationPlaceholder')} maxLength={255} />
      </Field>
      <Field label={t('events.fieldNote')}>
        <TextArea name="note" placeholder={t('events.fieldNotePlaceholder')} minHeight={64} maxLength={10000} />
      </Field>
      {recur}
      {sheet.mode === 'edit' && F.seriesId ? (
        <Box key="serbtn" sx={{ display: 'flex', flexDirection: 'column', gap: '9px', mt: '4px' }}>
          <Box
            key="h"
            sx={{
              display: 'flex',
              alignItems: 'center',
              gap: '7px',
              fontSize: '12px',
              color: NEUTRAL.secondary,
              fontWeight: 600,
            }}
          >
            <Sym name="repeat" size={16} color={NEUTRAL.faint} />
            {t('events.seriesHint')}
          </Box>
          <Box key="b" sx={{ display: 'flex', gap: '10px' }}>
            <ButtonBase
              key="one"
              onClick={() => app.saveEvent('single')}
              disabled={app.state.busy === 'save' || !canSubmit}
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
              key="all"
              onClick={() => app.saveEvent('series')}
              disabled={app.state.busy === 'save' || !canSubmit}
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
      ) : (
        <PrimaryButton
          label={sheet.mode === 'edit' ? t('events.saveChanges') : t('events.createEvent')}
          onClick={() => app.saveEvent('single')}
          busy={app.state.busy === 'save'}
          disabled={!canSubmit}
        />
      )}
    </Box>
  );
}
