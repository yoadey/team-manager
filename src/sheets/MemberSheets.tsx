import Box from '@mui/material/Box';
import ButtonBase from '@mui/material/ButtonBase';
import { buildTokens } from '../theme/tokens';
import { Av, Chip, Field, labelSx, PrimaryButton, Sym, TextInput } from '../components/ui';
import type { Member } from '../services/types';
import type { SheetProps } from './types';

export function MemberDetailSheet({ app, sheet }: SheetProps) {
  const { state } = app;
  const t = buildTokens(state.primaryColor);
  const team = app.activeTeam()!;
  const m: Member = sheet.member;
  const st: { quote: number | null; counted: number; yes: number } | null = sheet.stats;
  const qcol = st && st.quote !== null ? (st.quote >= 80 ? '#2E7D32' : (st.quote >= 50 ? '#9A5B00' : '#BA1A1A')) : '#9A9DA6';

  const head = (
    <Box key="hd" sx={{ display: 'flex', alignItems: 'center', gap: '14px', p: '4px 2px 18px' }}>
      <Av key="a" name={m.name} photo={m.photo} color={m.avatarColor} size={56} font={20} />
      <Box key="m" sx={{ minWidth: 0 }}>
        <Box key="n" sx={{ fontSize: '18px', fontWeight: 700 }}>{m.name}</Box>
        <Box key="r" sx={{ display: 'flex', flexWrap: 'wrap', gap: '5px', mt: '5px' }}>
          {m.roles.map((r) => <Chip key={r.id} label={r.name} color={r.color} bg={r.color + '1A'} icon="circle" fs={11} />)}
        </Box>
      </Box>
    </Box>
  );

  const stats = (
    <Box key="st" sx={{ display: 'flex', gap: '10px', mb: '16px' }}>
      <Box key="q" sx={{ flex: 1, background: '#F4F4FA', borderRadius: '14px', p: '14px', textAlign: 'center' }}>
        <Box key="v" sx={{ fontSize: '24px', fontWeight: 800, color: qcol }}>{st ? (st.quote === null ? '–' : st.quote + '%') : '…'}</Box>
        <Box key="l" sx={{ fontSize: '11px', color: '#6A6D76', mt: '2px' }}>Anwesenheitsquote</Box>
      </Box>
      <Box key="g" sx={{ flex: 1, background: '#F4F4FA', borderRadius: '14px', p: '14px', textAlign: 'center' }}>
        <Box key="v" sx={{ fontSize: '24px', fontWeight: 800 }}>{m.roles.length}</Box>
        <Box key="l" sx={{ fontSize: '11px', color: '#6A6D76', mt: '2px' }}>{m.roles.length === 1 ? 'Rolle' : 'Rollen'}</Box>
      </Box>
    </Box>
  );

  const fmtBd = (b: string) => b ? new Intl.DateTimeFormat('de-DE', { day: 'numeric', month: 'long', year: 'numeric' }).format(new Date(b + 'T00:00:00')) : '—';
  const cRow = (icon: string, val: string) => (
    <Box key={icon} sx={{ display: 'flex', alignItems: 'center', gap: '12px', p: '13px 14px', background: '#fff' }}>
      <Sym name={icon} size={19} color="#6A6D76" />
      <Box key="t" component="span" sx={{ flex: 1, fontSize: '14px' }}>{val || '—'}</Box>
    </Box>
  );

  const contact = (
    <Box key="c" sx={{ display: 'flex', flexDirection: 'column', gap: '1px', borderRadius: '14px', overflow: 'hidden', border: '1px solid #E6E7EE' }}>
      {cRow('mail', m.email)}
      {cRow('call', m.phone)}
      {cRow('cake', fmtBd(m.birthday))}
      {cRow('home', m.address)}
    </Box>
  );

  const isMe = m.userId === state.user!.id;
  const canWrite = app.can('members', 'write');
  const edit = (canWrite || isMe) ? (
    <Box key="ed" sx={{ display: 'flex', gap: '10px', mt: '18px' }}>
      <ButtonBase key="e" onClick={() => app.openMemberForm(m)} sx={{ flex: 1, display: 'flex', alignItems: 'center', justifyContent: 'center', gap: '8px', p: '12px', borderRadius: '13px', border: '1px solid #C8CAD2', background: '#fff', color: '#44474E', fontWeight: 600, fontSize: '14px', cursor: 'pointer' }}>
        <Sym name="edit" size={19} color="#44474E" />{isMe ? 'Profil bearbeiten' : 'Bearbeiten'}
      </ButtonBase>
      {(canWrite && !isMe) ? (
        <ButtonBase key="r" onClick={() => app.removeMember(m.membershipId)} sx={{ display: 'flex', alignItems: 'center', justifyContent: 'center', p: '12px 16px', borderRadius: '13px', border: '1px solid #F0C4C0', background: '#FFF4F3', color: '#BA1A1A', fontWeight: 600, cursor: 'pointer' }}>
          <Sym name="person_remove" size={19} color="#BA1A1A" />
        </ButtonBase>
      ) : null}
    </Box>
  ) : null;

  const note = app.can('members', 'write') ? (
    <Box key="nt" sx={{ display: 'flex', gap: '9px', mt: '14px', p: '11px 13px', background: '#F4F4FA', borderRadius: '13px', fontSize: '12px', color: '#6A6D76', lineHeight: 1.5 }}>
      <Sym name="info" size={17} color="#9A9DA6" />
      Die Team-Zugeh&ouml;rigkeit kann hier nicht ge&auml;ndert werden. Neue Mitglieder treten &uuml;ber einen Einladungslink bei &ndash; bestehende lassen sich nur aus dem Team entfernen.
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
  const t = buildTokens(state.primaryColor);
  const team = app.activeTeam()!;
  const F = app.state.form;
  const myIds: string[] = F.roleIds || [];
  const canRoles = app.can('members', 'write');

  const photoRow = (
    <Box key="ph" sx={{ display: 'flex', alignItems: 'center', gap: '14px', mb: '4px' }}>
      <Av key="a" name={F.name || '?'} photo={F.photo} color="#9A9DA6" size={56} font={20} />
      <Box key="u" component="label" sx={{ display: 'inline-flex', alignItems: 'center', gap: '8px', p: '9px 14px', borderRadius: '12px', border: '1px solid #C8CAD2', background: '#fff', cursor: 'pointer', fontSize: '13px', fontWeight: 600, color: '#44474E' }}>
        <Sym name="photo_camera" size={18} />
        {F.photo ? 'Foto ändern' : 'Foto hochladen'}
        <input key="f" type="file" accept="image/*" onChange={(e) => app.onFile(e, (d) => app.setFormVal({ photo: d }))} hidden />
      </Box>
    </Box>
  );

  const roleChips = canRoles ? (
    <Box key="rc">
      <Box key="l" sx={labelSx}>Rollen (Mehrfachauswahl)</Box>
      <Box key="b" sx={{ display: 'flex', flexWrap: 'wrap', gap: '8px' }}>
        {state.roles.map((r) => {
          const sel = myIds.includes(r.id);
          return (
            <ButtonBase key={r.id} onClick={() => app.toggleFormRole(r.id)} sx={{ display: 'inline-flex', alignItems: 'center', gap: '7px', p: '8px 13px', borderRadius: '999px', cursor: 'pointer', fontSize: '13px', fontWeight: 600, border: '1.5px solid ' + (sel ? r.color : '#D0D2DA'), background: sel ? r.color + '1A' : '#fff', color: sel ? r.color : '#44474E' }}>
              <Box key="d" component="span" sx={{ width: '9px', height: '9px', borderRadius: '50%', background: r.color }} />
              {r.name}
              {sel ? <Sym name="check" size={16} color={r.color} /> : null}
            </ButtonBase>
          );
        })}
      </Box>
    </Box>
  ) : null;

  const contactNote = (
    <Box key="cn" sx={{ fontSize: '12px', color: '#9A9DA6', lineHeight: 1.5, display: 'flex', gap: '8px' }}>
      <Sym name="lock" size={15} color="#C0C2CA" />
      Kontaktdaten sind optional. Geburtstag und Adresse sieht nur das Trainerteam.
    </Box>
  );

  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: '16px' }}>
      {photoRow}
      <Field label="Name"><TextInput name="name" placeholder="Vor- und Nachname" /></Field>
      <Field label="E-Mail"><TextInput name="email" type="email" placeholder="name@example.de" /></Field>
      <Field label="Telefon"><TextInput name="phone" placeholder="+49 …" /></Field>
      <Field label="Geburtstag"><TextInput name="birthday" type="date" /></Field>
      <Field label="Adresse"><TextInput name="address" placeholder="Straße, PLZ Ort" /></Field>
      {contactNote}
      {roleChips}
      <PrimaryButton label="Profil speichern" onClick={() => app.saveMember()} busy={app.state.busy === 'save'} />
    </Box>
  );
}
