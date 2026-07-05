import Box from '@mui/material/Box';
import ButtonBase from '@mui/material/ButtonBase';
import { buildTokens, NEUTRAL } from '@/styles/tokens';
import { Av, Field, labelSx, PrimaryButton, SectionTitle, Sym, TextArea, TextInput } from '@/components/ui';
import { shortName } from '@/layouts/useCompact';
import type { Invite } from '@/types';
import type { SheetProps } from '@/sheets/types';
import type { CreateTeamFormValues, TeamSettingsFormValues } from '../types';
import { formValues } from '@/utils/forms';
import { t } from '@/i18n';

const TEAM_ICONS = ['🏆', '⭐', '💃', '🕺', '🎭', '🔥', '👑', '🎯', '💎', '🦅', '⚡', '🌟'];

export function CreateTeamSheet({ app, sheet }: SheetProps) {
  void sheet;
  const { state } = app;
  const tk = buildTokens(state.primaryColor);
  const team = app.activeTeam()!;
  void team;
  const F = formValues<CreateTeamFormValues>(app.state);

  const icons = TEAM_ICONS.map((em) => (
    <ButtonBase
      key={em}
      onClick={() => app.setFormVal({ icon: em })}
      sx={{
        width: '48px',
        height: '48px',
        borderRadius: '13px',
        border: '2px solid ' + (F.icon === em ? tk.primary : NEUTRAL.line3),
        background: F.icon === em ? tk.primaryContainer : NEUTRAL.card,
        cursor: 'pointer',
        fontSize: '22px',
      }}
    >
      {em}
    </ButtonBase>
  ));

  const photoRow = (
    <Box
      key="ph"
      component="label"
      sx={{
        display: 'flex',
        alignItems: 'center',
        gap: '12px',
        p: '12px 14px',
        borderRadius: '13px',
        border: `1px dashed ${NEUTRAL.inputBorder}`,
        background: NEUTRAL.sidebar,
        cursor: 'pointer',
      }}
    >
      {F.photo ? (
        <Av key="a" name={t('team.photoAlt')} photo={F.photo} color={NEUTRAL.inputBorder} size={40} />
      ) : (
        <Sym name="add_photo_alternate" size={24} color={NEUTRAL.secondary} />
      )}
      <Box
        key="l"
        component="span"
        sx={{ flex: 1, fontSize: '13px', fontWeight: 600, color: NEUTRAL.onSurfaceVariant }}
      >
        {F.photo ? t('team.photoSelected') : t('team.photoUpload')}
      </Box>
      <input
        key="f"
        type="file"
        accept="image/*"
        onChange={(e) => app.onFile(e, (d) => app.setFormVal({ photo: d }))}
        hidden
      />
    </Box>
  );

  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: '16px' }}>
      <Box
        key="i"
        sx={{
          display: 'flex',
          alignItems: 'center',
          gap: '12px',
          fontSize: '13px',
          color: NEUTRAL.secondary,
          lineHeight: 1.5,
          background: NEUTRAL.sidebar,
          p: '12px 14px',
          borderRadius: '13px',
        }}
      >
        <Sym name="shield_person" size={24} color={tk.primary} />
        {t('team.createTeamHint')}
      </Box>
      <Field label={t('team.teamNameField')}>
        <TextInput name="name" placeholder={t('team.teamNamePlaceholder')} maxLength={60} />
      </Field>
      <Box key="ic">
        <Box key="l" sx={labelSx}>
          {t('team.iconLabel')}
        </Box>
        <Box key="b" sx={{ display: 'flex', flexWrap: 'wrap', gap: '8px' }}>
          {icons}
        </Box>
      </Box>
      {photoRow}
      <PrimaryButton label={t('team.createBtn')} onClick={() => app.createTeam()} busy={app.state.busy === 'save'} />
    </Box>
  );
}

export function InviteSheet({ app, sheet }: SheetProps) {
  const { state } = app;
  const tk = buildTokens(state.primaryColor);
  const team = app.activeTeam()!;
  const inv: Invite | null = sheet.invite ?? null;

  return (
    <Box>
      <Box key="hero" sx={{ textAlign: 'center', p: '6px 2px 18px' }}>
        <Box
          key="i"
          sx={{
            width: '64px',
            height: '64px',
            borderRadius: '18px',
            background: tk.primaryContainer,
            color: tk.primary,
            display: 'inline-flex',
            alignItems: 'center',
            justifyContent: 'center',
            fontFamily: "'Material Symbols Outlined'",
            fontSize: '34px',
          }}
        >
          link
        </Box>
        <Box key="s" sx={{ fontSize: '14px', color: NEUTRAL.secondary, mt: '12px', lineHeight: 1.5 }}>
          {t('team.inviteDesc2', { teamName: shortName(team.name) })
            .split(shortName(team.name))
            .reduce<React.ReactNode[]>((acc, part, i, arr) => {
              if (i < arr.length - 1) return [...acc, part, <b key={i}>{shortName(team.name)}</b>];
              return [...acc, part];
            }, [])}
        </Box>
      </Box>
      <Box
        key="box"
        sx={{
          display: 'flex',
          alignItems: 'center',
          gap: '8px',
          background: NEUTRAL.sidebar,
          border: `1px solid ${NEUTRAL.line3}`,
          borderRadius: '13px',
          p: '12px 14px',
        }}
      >
        <Box
          key="l"
          component="span"
          sx={{
            flex: 1,
            fontSize: '13px',
            fontFamily: 'monospace',
            color: NEUTRAL.onSurface,
            overflow: 'hidden',
            textOverflow: 'ellipsis',
            whiteSpace: 'nowrap',
          }}
        >
          {inv ? inv.link : t('team.inviteGenerating')}
        </Box>
        {inv ? (
          <ButtonBase
            key="c"
            onClick={() => app.copyInvite()}
            sx={{
              display: 'flex',
              alignItems: 'center',
              gap: '6px',
              background: tk.primary,
              color: tk.onPrimary,
              border: 'none',
              borderRadius: '9px',
              p: '8px 12px',
              fontSize: '13px',
              fontWeight: 600,
              cursor: 'pointer',
            }}
          >
            <Sym name="content_copy" size={16} color={tk.onPrimary} />
            {sheet.copied ? t('team.inviteCopied') : t('team.inviteCopy')}
          </ButtonBase>
        ) : null}
      </Box>
      {inv ? (
        <Box key="code" sx={{ textAlign: 'center', mt: '14px', fontSize: '13px', color: NEUTRAL.secondary }}>
          {t('team.inviteCode')}{' '}
          <Box
            key="b"
            component="b"
            sx={{ fontFamily: 'monospace', fontSize: '15px', letterSpacing: '1px', color: NEUTRAL.onSurface }}
          >
            {inv.code}
          </Box>
        </Box>
      ) : null}
    </Box>
  );
}

export function TeamSettingsSheet({ app, sheet }: SheetProps) {
  void sheet;
  const { state } = app;
  const tk = buildTokens(state.primaryColor);
  const team = app.activeTeam()!;
  const F = formValues<TeamSettingsFormValues>(app.state);
  const roles = app.state.roles;

  const upLabel = (icon: string, label: string, cb: (d: string) => void) => (
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
      <Sym name={icon} size={18} />
      {label}
      <input key="f" type="file" accept="image/*" onChange={(e) => app.onFile(e, cb)} hidden />
    </Box>
  );

  const logoPreview = (
    <Box
      key="lp"
      component="span"
      role={F.logo ? 'img' : undefined}
      aria-label={F.logo ? t('team.logoAlt') : undefined}
      sx={{
        width: '58px',
        height: '58px',
        borderRadius: '15px',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        fontSize: '28px',
        flex: '0 0 auto',
        overflow: 'hidden',
        ...(F.logo
          ? { backgroundImage: 'url(' + F.logo + ')', backgroundSize: 'cover', backgroundPosition: 'center' }
          : { background: team.iconBg, color: team.iconFg }),
      }}
    >
      {F.logo ? '' : F.icon}
    </Box>
  );

  const logoSec = (
    <Box key="logo">
      <SectionTitle>{t('team.settingsLogoSection')}</SectionTitle>
      <Box key="r" sx={{ display: 'flex', alignItems: 'center', gap: '14px', mb: '10px' }}>
        {logoPreview}
        {upLabel('upload', F.logo ? t('team.settingsLogoChange') : t('team.settingsLogoUpload'), (d) =>
          app.saveTeamLogo(d),
        )}
      </Box>
      <Box key="em" sx={{ display: 'flex', flexWrap: 'wrap', gap: '8px' }}>
        {TEAM_ICONS.map((em) => (
          <ButtonBase
            key={em}
            onClick={() => app.setTeamIcon(em)}
            sx={{
              width: '44px',
              height: '44px',
              borderRadius: '12px',
              border: '2px solid ' + (!F.logo && F.icon === em ? tk.primary : NEUTRAL.line3),
              background: !F.logo && F.icon === em ? tk.primaryContainer : NEUTRAL.card,
              cursor: 'pointer',
              fontSize: '20px',
            }}
          >
            {em}
          </ButtonBase>
        ))}
      </Box>
    </Box>
  );

  const photoSec = (
    <Box key="photo">
      <SectionTitle>{t('team.settingsPhotoSection')}</SectionTitle>
      <Box key="r" sx={{ display: 'flex', alignItems: 'center', gap: '14px' }}>
        {team.photo ? (
          <Av key="a" name={team.name} photo={team.photo} color={NEUTRAL.inputBorder} size={58} />
        ) : (
          <Box
            key="i"
            component="span"
            sx={{
              width: '58px',
              height: '58px',
              borderRadius: '15px',
              background: NEUTRAL.line2,
              color: NEUTRAL.faint,
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              fontFamily: "'Material Symbols Outlined'",
              fontSize: '26px',
              flex: '0 0 auto',
            }}
          >
            image
          </Box>
        )}
        {upLabel('photo_camera', team.photo ? t('team.settingsPhotoChange') : t('team.settingsPhotoUpload'), (d) =>
          app.saveTeamPhoto(d),
        )}
        {team.photo ? (
          <ButtonBase
            key="rm"
            onClick={() => app.removeTeamPhoto()}
            aria-label={t('team.settingsPhotoRemove')}
            sx={{
              width: '36px',
              height: '36px',
              borderRadius: '50%',
              background: NEUTRAL.errorBg,
              color: NEUTRAL.error,
              cursor: 'pointer',
              flex: '0 0 auto',
            }}
          >
            <Sym name="delete" size={18} color={NEUTRAL.error} />
          </ButtonBase>
        ) : null}
      </Box>
      <Box key="h" sx={{ fontSize: '12px', color: NEUTRAL.faint, mt: '8px', lineHeight: 1.5 }}>
        {t('team.settingsPhotoHint')}
      </Box>
    </Box>
  );

  const visSec = (
    <Box key="vis">
      <SectionTitle>{t('team.settingsVisSection')}</SectionTitle>
      <Box key="h" sx={{ fontSize: '12px', color: NEUTRAL.faint, m: '-2px 0 10px', lineHeight: 1.5 }}>
        {t('team.settingsVisHint')}
      </Box>
      <Box key="b" sx={{ display: 'flex', flexWrap: 'wrap', gap: '8px' }}>
        {roles.map((r) => {
          const sel = (F.reasonRoles || []).includes(r.id);
          return (
            <ButtonBase
              key={r.id}
              role="checkbox"
              aria-checked={sel}
              aria-label={r.name}
              onClick={() => app.toggleReasonRole(r.id)}
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
  );

  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: '18px' }}>
      <Field label={t('team.settingsNameField')}>
        <TextInput name="name" placeholder={t('team.settingsNamePlaceholder')} maxLength={60} />
      </Field>
      <Field label={t('team.settingsDescField')}>
        <TextArea name="description" placeholder={t('team.settingsDescPlaceholder')} minHeight={80} />
      </Field>
      {logoSec}
      {photoSec}
      {visSec}
      <PrimaryButton
        label={t('team.settingsSave')}
        onClick={() => app.saveTeamSettings()}
        busy={app.state.busy === 'save'}
      />
    </Box>
  );
}
