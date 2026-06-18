import Box from '@mui/material/Box';
import ButtonBase from '@mui/material/ButtonBase';
import { buildTokens } from '@/styles/tokens';
import { Av, Field, labelSx, PrimaryButton, SectionTitle, Sym, TextArea, TextInput } from '@/components/ui';
import { shortName } from '@/layouts/AppShell';
import type { Invite } from '@/types';
import type { SheetProps } from '@/sheets/types';

const TEAM_ICONS = ['🏆', '⭐', '💃', '🕺', '🎭', '🔥', '👑', '🎯', '💎', '🦅', '⚡', '🌟'];

export function CreateTeamSheet({ app, sheet }: SheetProps) {
  void sheet;
  const { state } = app;
  const t = buildTokens(state.primaryColor);
  const team = app.activeTeam()!;
  void team;
  const F = app.state.form;

  const icons = TEAM_ICONS.map((em) => (
    <ButtonBase key={em} onClick={() => app.setFormVal({ icon: em })} sx={{ width: '48px', height: '48px', borderRadius: '13px', border: '2px solid ' + (F.icon === em ? t.primary : '#E0E2EA'), background: F.icon === em ? t.primaryContainer : '#fff', cursor: 'pointer', fontSize: '22px' }}>
      {em}
    </ButtonBase>
  ));

  const photoRow = (
    <Box key="ph" component="label" sx={{ display: 'flex', alignItems: 'center', gap: '12px', p: '12px 14px', borderRadius: '13px', border: '1px dashed #C8CAD2', background: '#F4F4FA', cursor: 'pointer' }}>
      {F.photo
        ? <Av key="a" name="" photo={F.photo} color="#ccc" size={40} />
        : <Sym name="add_photo_alternate" size={24} color="#6A6D76" />}
      <Box key="l" component="span" sx={{ flex: 1, fontSize: '13px', fontWeight: 600, color: '#44474E' }}>{F.photo ? 'Teamfoto ausgewählt' : 'Teamfoto hochladen (optional)'}</Box>
      <input key="f" type="file" accept="image/*" onChange={(e) => app.onFile(e, (d) => app.setFormVal({ photo: d }))} hidden />
    </Box>
  );

  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: '16px' }}>
      <Box key="i" sx={{ display: 'flex', alignItems: 'center', gap: '12px', fontSize: '13px', color: '#6A6D76', lineHeight: 1.5, background: '#F4F4FA', p: '12px 14px', borderRadius: '13px' }}>
        <Sym name="shield_person" size={24} color={t.primary} />
        Du wirst automatisch Administrator des neuen Teams. Standard-Rollen werden angelegt.
      </Box>
      <Field label="Team-Name"><TextInput name="name" placeholder="z. B. C-Team TSC Schwarz-Gelb" /></Field>
      <Box key="ic">
        <Box key="l" sx={labelSx}>Icon</Box>
        <Box key="b" sx={{ display: 'flex', flexWrap: 'wrap', gap: '8px' }}>{icons}</Box>
      </Box>
      {photoRow}
      <PrimaryButton label="Team anlegen" onClick={() => app.createTeam()} busy={app.state.busy === 'save'} />
    </Box>
  );
}

export function InviteSheet({ app, sheet }: SheetProps) {
  const { state } = app;
  const t = buildTokens(state.primaryColor);
  const team = app.activeTeam()!;
  const inv: Invite | null = sheet.invite;

  return (
    <Box>
      <Box key="hero" sx={{ textAlign: 'center', p: '6px 2px 18px' }}>
        <Box key="i" sx={{ width: '64px', height: '64px', borderRadius: '18px', background: t.primaryContainer, color: t.primary, display: 'inline-flex', alignItems: 'center', justifyContent: 'center', fontFamily: "'Material Symbols Outlined'", fontSize: '34px' }}>link</Box>
        <Box key="s" sx={{ fontSize: '14px', color: '#6A6D76', mt: '12px', lineHeight: 1.5 }}>
          Teile diesen Link. Neue Mitglieder treten <b key="b">{shortName(team.name)}</b> bei. Gültig 7 Tage.
        </Box>
      </Box>
      <Box key="box" sx={{ display: 'flex', alignItems: 'center', gap: '8px', background: '#F4F4FA', border: '1px solid #E0E2EA', borderRadius: '13px', p: '12px 14px' }}>
        <Box key="l" component="span" sx={{ flex: 1, fontSize: '13px', fontFamily: 'monospace', color: '#1A1C20', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>{inv ? inv.link : 'Erzeuge Link…'}</Box>
        {inv ? (
          <ButtonBase key="c" onClick={() => app.copyInvite()} sx={{ display: 'flex', alignItems: 'center', gap: '6px', background: t.primary, color: t.onPrimary, border: 'none', borderRadius: '9px', p: '8px 12px', fontSize: '13px', fontWeight: 600, cursor: 'pointer' }}>
            <Sym name="content_copy" size={16} color={t.onPrimary} />{sheet.copied ? 'Kopiert' : 'Kopieren'}
          </ButtonBase>
        ) : null}
      </Box>
      {inv ? (
        <Box key="code" sx={{ textAlign: 'center', mt: '14px', fontSize: '13px', color: '#6A6D76' }}>
          Beitritts-Code: <Box key="b" component="b" sx={{ fontFamily: 'monospace', fontSize: '15px', letterSpacing: '1px', color: '#1A1C20' }}>{inv.code}</Box>
        </Box>
      ) : null}
    </Box>
  );
}

export function TeamSettingsSheet({ app, sheet }: SheetProps) {
  void sheet;
  const { state } = app;
  const t = buildTokens(state.primaryColor);
  const team = app.activeTeam()!;
  const F = app.state.form;
  const roles = app.state.roles;

  const upLabel = (icon: string, label: string, cb: (d: string) => void) => (
    <Box key="u" component="label" sx={{ display: 'inline-flex', alignItems: 'center', gap: '8px', p: '9px 14px', borderRadius: '12px', border: '1px solid #C8CAD2', background: '#fff', cursor: 'pointer', fontSize: '13px', fontWeight: 600, color: '#44474E' }}>
      <Sym name={icon} size={18} />
      {label}
      <input key="f" type="file" accept="image/*" onChange={(e) => app.onFile(e, cb)} hidden />
    </Box>
  );

  const logoPreview = (
    <Box
      key="lp"
      component="span"
      sx={{
        width: '58px', height: '58px', borderRadius: '15px', display: 'flex', alignItems: 'center', justifyContent: 'center', fontSize: '28px', flex: '0 0 auto', overflow: 'hidden',
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
      <SectionTitle>Logo</SectionTitle>
      <Box key="r" sx={{ display: 'flex', alignItems: 'center', gap: '14px', mb: '10px' }}>
        {logoPreview}
        {upLabel('upload', F.logo ? 'Logo ändern' : 'Bild-Logo hochladen', (d) => app.saveTeamLogo(d))}
      </Box>
      <Box key="em" sx={{ display: 'flex', flexWrap: 'wrap', gap: '8px' }}>
        {TEAM_ICONS.map((em) => (
          <ButtonBase key={em} onClick={() => app.setTeamIcon(em)} sx={{ width: '44px', height: '44px', borderRadius: '12px', border: '2px solid ' + ((!F.logo && F.icon === em) ? t.primary : '#E0E2EA'), background: (!F.logo && F.icon === em) ? t.primaryContainer : '#fff', cursor: 'pointer', fontSize: '20px' }}>
            {em}
          </ButtonBase>
        ))}
      </Box>
    </Box>
  );

  const photoSec = (
    <Box key="photo">
      <SectionTitle>Gruppenbild</SectionTitle>
      <Box key="r" sx={{ display: 'flex', alignItems: 'center', gap: '14px' }}>
        {team.photo
          ? <Av key="a" name={team.name} photo={team.photo} color="#ccc" size={58} />
          : <Box key="i" component="span" sx={{ width: '58px', height: '58px', borderRadius: '15px', background: '#ECEDF3', color: '#9A9DA6', display: 'flex', alignItems: 'center', justifyContent: 'center', fontFamily: "'Material Symbols Outlined'", fontSize: '26px', flex: '0 0 auto' }}>image</Box>}
        {upLabel('photo_camera', team.photo ? 'Bild ändern' : 'Gruppenbild hochladen', (d) => app.saveTeamPhoto(d))}
      </Box>
      <Box key="h" sx={{ fontSize: '12px', color: '#9A9DA6', mt: '8px', lineHeight: 1.5 }}>Wird als Titelbild auf der Startseite und der Team-Seite angezeigt.</Box>
    </Box>
  );

  const visSec = (
    <Box key="vis">
      <SectionTitle>Sichtbarkeit von Absage-Kommentaren</SectionTitle>
      <Box key="h" sx={{ fontSize: '12px', color: '#9A9DA6', m: '-2px 0 10px', lineHeight: 1.5 }}>Welche Rollen dürfen die Kommentare bei Absagen sehen? Den eigenen Kommentar sieht jede Person selbst.</Box>
      <Box key="b" sx={{ display: 'flex', flexWrap: 'wrap', gap: '8px' }}>
        {roles.map((r) => {
          const sel = (F.reasonRoles || []).includes(r.id);
          return (
            <ButtonBase key={r.id} onClick={() => app.toggleReasonRole(r.id)} sx={{ display: 'inline-flex', alignItems: 'center', gap: '7px', p: '8px 13px', borderRadius: '999px', cursor: 'pointer', fontSize: '13px', fontWeight: 600, border: '1.5px solid ' + (sel ? r.color : '#D0D2DA'), background: sel ? r.color + '1A' : '#fff', color: sel ? r.color : '#44474E' }}>
              <Box key="d" component="span" sx={{ width: '9px', height: '9px', borderRadius: '50%', background: r.color }} />
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
      <Field label="Team-Name"><TextInput name="name" placeholder="Team-Name" /></Field>
      <Field label="Teambeschreibung"><TextArea name="description" placeholder="Kurze Beschreibung des Teams…" minHeight={80} /></Field>
      {logoSec}
      {photoSec}
      {visSec}
      <PrimaryButton label="Einstellungen speichern" onClick={() => app.saveTeamSettings()} busy={app.state.busy === 'save'} />
    </Box>
  );
}
