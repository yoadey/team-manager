import { NEUTRAL } from '@/styles/tokens';
import Box from '@mui/material/Box';
import ButtonBase from '@mui/material/ButtonBase';
import { Av, Chip, EmptyState, Field, labelSx, PrimaryButton, Sym, TextInput } from '@/components/ui';
import type { Member, MemberFormValues } from '../types';
import type { SheetProps } from '@/sheets/types';
import { formValues } from '@/utils/forms';
import { getIntlLocale, t } from '@/i18n';
import { validateBirthday, validateEmail, validatePhone, validateRequiredText } from '@/utils/validation';

export function MemberDetailSheet({ app, sheet }: SheetProps) {
  const { state } = app;
  // member is looked up from the already-loaded local member list (not an
  // async fetch), so it can genuinely be missing -- e.g. a stale bookmarked
  // or browser-back/forward URL for a member who has since been removed.
  // Render a graceful empty state instead of force-unwrapping into a
  // render-time crash.
  if (!sheet.member) return <EmptyState icon="person_off" text={t('members.detailNotFound')} />;
  const m: Member = sheet.member;
  const st: { quote: number | null; counted: number; yes: number } | null = sheet.stats ?? null;
  const qcol =
    st && st.quote !== null
      ? st.quote >= 80
        ? NEUTRAL.success
        : st.quote >= 50
          ? NEUTRAL.warn
          : NEUTRAL.error
      : NEUTRAL.faint;

  const head = (
    <Box key="hd" sx={{ display: 'flex', alignItems: 'center', gap: '14px', p: '4px 2px 18px' }}>
      <Av key="a" name={m.name} photo={m.photo} color={m.avatarColor} size={56} font={20} />
      <Box key="m" sx={{ minWidth: 0 }}>
        <Box key="n" sx={{ fontSize: '18px', fontWeight: 700 }}>
          {m.name}
        </Box>
        <Box key="r" sx={{ display: 'flex', flexWrap: 'wrap', gap: '5px', mt: '5px' }}>
          {m.roles.map((r) => (
            <Chip key={r.id} label={r.name} color={r.color} bg={r.color + '1A'} icon="circle" fs={11} />
          ))}
        </Box>
      </Box>
    </Box>
  );

  const stats = (
    <Box key="st" sx={{ display: 'flex', gap: '10px', mb: '16px' }}>
      <Box key="q" sx={{ flex: 1, background: NEUTRAL.sidebar, borderRadius: '14px', p: '14px', textAlign: 'center' }}>
        <Box key="v" sx={{ fontSize: '24px', fontWeight: 800, color: qcol }}>
          {st ? (st.quote === null ? '–' : st.quote + '%') : '…'}
        </Box>
        <Box key="l" sx={{ fontSize: '11px', color: NEUTRAL.secondary, mt: '2px' }}>
          {t('members.attendanceRate')}
        </Box>
      </Box>
      <Box key="g" sx={{ flex: 1, background: NEUTRAL.sidebar, borderRadius: '14px', p: '14px', textAlign: 'center' }}>
        <Box key="v" sx={{ fontSize: '24px', fontWeight: 800 }}>
          {m.roles.length}
        </Box>
        <Box key="l" sx={{ fontSize: '11px', color: NEUTRAL.secondary, mt: '2px' }}>
          {m.roles.length === 1 ? t('members.role') : t('members.roles')}
        </Box>
      </Box>
    </Box>
  );

  const fmtBd = (b: string) =>
    b
      ? new Intl.DateTimeFormat(getIntlLocale(), { day: 'numeric', month: 'long', year: 'numeric' }).format(
          new Date(b + 'T00:00:00'),
        )
      : '—';
  const cRow = (icon: string, val: string) => (
    <Box
      key={icon}
      sx={{ display: 'flex', alignItems: 'center', gap: '12px', p: '13px 14px', background: NEUTRAL.card }}
    >
      <Sym name={icon} size={19} color={NEUTRAL.secondary} />
      <Box key="t" component="span" sx={{ flex: 1, fontSize: '14px' }}>
        {val || '—'}
      </Box>
    </Box>
  );

  const contact = (
    <Box
      key="c"
      sx={{
        display: 'flex',
        flexDirection: 'column',
        gap: '1px',
        borderRadius: '14px',
        overflow: 'hidden',
        border: `1px solid ${NEUTRAL.line}`,
      }}
    >
      {cRow('mail', m.email)}
      {cRow('call', m.phone)}
      {cRow('cake', fmtBd(m.birthday))}
      {cRow('home', m.address)}
    </Box>
  );

  const isMe = m.userId === state.user!.id;
  const canWrite = app.can('members', 'write');
  const edit =
    canWrite || isMe ? (
      <Box key="ed" sx={{ display: 'flex', gap: '10px', mt: '18px' }}>
        <ButtonBase
          key="e"
          onClick={() => app.openMemberForm(m)}
          sx={{
            flex: 1,
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            gap: '8px',
            p: '12px',
            borderRadius: '13px',
            border: `1px solid ${NEUTRAL.inputBorder}`,
            background: NEUTRAL.card,
            color: NEUTRAL.onSurfaceVariant,
            fontWeight: 600,
            fontSize: '14px',
            cursor: 'pointer',
          }}
        >
          <Sym name="edit" size={19} color={NEUTRAL.onSurfaceVariant} />
          {isMe ? t('members.editProfile') : t('members.edit')}
        </ButtonBase>
        {canWrite && !isMe ? (
          <ButtonBase
            key="r"
            aria-label={t('members.removeTitle')}
            onClick={() => app.removeMember(m.membershipId)}
            sx={{
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              p: '12px 16px',
              borderRadius: '13px',
              border: '1px solid #F0C4C0',
              background: NEUTRAL.errorBg,
              color: NEUTRAL.error,
              fontWeight: 600,
              cursor: 'pointer',
            }}
          >
            <Sym name="person_remove" size={19} color={NEUTRAL.error} />
          </ButtonBase>
        ) : null}
      </Box>
    ) : null;

  const note = app.can('members', 'write') ? (
    <Box
      key="nt"
      sx={{
        display: 'flex',
        gap: '9px',
        mt: '14px',
        p: '11px 13px',
        background: NEUTRAL.sidebar,
        borderRadius: '13px',
        fontSize: '12px',
        color: NEUTRAL.secondary,
        lineHeight: 1.5,
      }}
    >
      <Sym name="info" size={17} color={NEUTRAL.faint} />
      {t('members.membershipNote')}
    </Box>
  ) : null;

  return (
    <Box>
      {head}
      {stats}
      {contact}
      {edit}
      {note}
    </Box>
  );
}

export function MemberFormSheet({ app, sheet }: SheetProps) {
  const { state } = app;
  const F = formValues<MemberFormValues>(app.state);
  const errs = state.formErrors;
  const myIds: string[] = F.roleIds || [];
  // Role assignment is a settings:write operation on the backend (assigning a
  // role can hand out settings:write itself), not members:write — gating on
  // members:write would show editable role chips to an admin whose actual
  // save would be rejected.
  const canRoles = app.can('settings', 'write');
  const canSubmit = !!F.name?.trim();
  // The backend has no endpoint to set another member's photo at all (only
  // PUT /auth/me/photo, self-only) — show the control only when editing your
  // own profile, not when an admin edits someone else's.
  const canEditPhoto = !!sheet.self;

  const validateName = () => {
    const r = validateRequiredText(F.name, t('members.fieldNameError'));
    app.setFormErrors({ name: r.ok ? '' : r.message! });
  };
  const validateEmail_ = () => {
    const r = validateEmail(F.email, t('members.fieldEmailError'));
    app.setFormErrors({ email: r.ok ? '' : r.message! });
  };
  const validatePhone_ = () => {
    const r = validatePhone(F.phone, t('members.fieldPhoneError'));
    app.setFormErrors({ phone: r.ok ? '' : r.message! });
  };
  const validateBirthday_ = () => {
    const r = validateBirthday(F.birthday, t('members.fieldBirthdayError'));
    app.setFormErrors({ birthday: r.ok ? '' : r.message! });
  };

  const photoRow = canEditPhoto ? (
    <Box key="ph" sx={{ display: 'flex', alignItems: 'center', gap: '14px', mb: '4px' }}>
      <Av key="a" name={F.name || '?'} photo={F.photo} color={NEUTRAL.faint} size={56} font={20} />
      <Box
        key="u"
        component="label"
        sx={{
          display: 'inline-flex',
          alignItems: 'center',
          gap: '8px',
          p: '9px 14px',
          borderRadius: '12px',
          border: `1px solid ${NEUTRAL.inputBorder}`,
          background: NEUTRAL.card,
          cursor: 'pointer',
          fontSize: '13px',
          fontWeight: 600,
          color: NEUTRAL.onSurfaceVariant,
        }}
      >
        <Sym name="photo_camera" size={18} />
        {F.photo ? t('members.photoChange') : t('members.photoUpload')}
        <input
          key="f"
          type="file"
          accept="image/*"
          onChange={(e) => {
            // setFormVal writes into the single shared, untyped form buffer
            // regardless of which sheet is open. Snapshot the sheet type
            // here, synchronously, before onFile's async FileReader read
            // starts, and re-check it via setState's functional-update form
            // once the read completes -- state/app.state are just this
            // render's snapshot (React context value, not a live ref), so
            // re-reading them here would just compare the closure to itself
            // and never catch anything; the functional updater's `s`
            // argument is guaranteed to be the actual live state at apply
            // time. Without this, if the user closes this member form (or
            // opens a different sheet that also reads form.photo, e.g. team
            // settings or create-team) before the read completes, the
            // resolved callback would overwrite that other sheet's
            // in-progress data.
            const sheetType = state.sheet?.type;
            app.onFile(e, (d) => {
              app.setState((s) => (s.sheet?.type === sheetType ? { form: { ...s.form, photo: d } } : {}));
            });
          }}
          hidden
        />
      </Box>
    </Box>
  ) : null;

  const roleChips = canRoles ? (
    <Box key="rc">
      <Box key="l" sx={labelSx}>
        {t('members.rolesMulti')}
      </Box>
      <Box key="b" sx={{ display: 'flex', flexWrap: 'wrap', gap: '8px' }}>
        {state.roles.map((r) => {
          const sel = myIds.includes(r.id);
          return (
            <ButtonBase
              key={r.id}
              role="checkbox"
              aria-checked={sel}
              aria-label={r.name}
              onClick={() => app.toggleFormRole(r.id)}
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
                color: sel ? r.color : NEUTRAL.onSurfaceVariant,
              }}
            >
              <Box
                key="d"
                component="span"
                sx={{ width: '9px', height: '9px', borderRadius: '50%', background: r.color }}
              />
              {r.name}
              {sel ? <Sym name="check" size={16} color={r.color} /> : null}
            </ButtonBase>
          );
        })}
      </Box>
    </Box>
  ) : null;

  const contactNote = (
    <Box key="cn" sx={{ fontSize: '12px', color: NEUTRAL.faint, lineHeight: 1.5, display: 'flex', gap: '8px' }}>
      <Sym name="lock" size={15} color={NEUTRAL.faint} />
      {t('members.contactNote')}
    </Box>
  );

  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: '16px' }}>
      {photoRow}
      <Field label={t('members.fieldName')} required error={!!errs.name} errorText={errs.name}>
        <TextInput name="name" placeholder={t('members.fieldNamePlaceholder')} onBlur={validateName} maxLength={255} />
      </Field>
      <Field label={t('members.fieldEmail')} error={!!errs.email} errorText={errs.email}>
        <TextInput
          name="email"
          type="email"
          placeholder={t('members.fieldEmailPlaceholder')}
          onBlur={validateEmail_}
          maxLength={254}
        />
      </Field>
      <Field label={t('members.fieldPhone')} error={!!errs.phone} errorText={errs.phone}>
        <TextInput name="phone" placeholder={t('members.fieldPhonePlaceholder')} onBlur={validatePhone_} maxLength={32} />
      </Field>
      <Field label={t('members.fieldBirthday')} error={!!errs.birthday} errorText={errs.birthday}>
        <TextInput name="birthday" type="date" min="1900-01-01" onBlur={validateBirthday_} />
      </Field>
      <Field label={t('members.fieldAddress')}>
        <TextInput name="address" placeholder={t('members.fieldAddressPlaceholder')} maxLength={500} />
      </Field>
      {contactNote}
      {roleChips}
      <PrimaryButton
        label={t('members.saveProfile')}
        onClick={() => app.saveMember()}
        busy={app.state.savingMember}
        disabled={!canSubmit}
      />
    </Box>
  );
}
