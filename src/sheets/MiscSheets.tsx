import Box from '@mui/material/Box';
import ButtonBase from '@mui/material/ButtonBase';
import { buildTokens } from '../styles/tokens';
import { Field, PrimaryButton, Sym, TextArea, TextInput, labelSx } from '../components/ui';
import type { SheetProps } from './types';

export function AbsenceFormSheet({ app, sheet }: SheetProps) {
  const { state } = app;
  const t = buildTokens(state.primaryColor);
  void t;
  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: '16px' }}>
      <Box sx={{ display: 'flex', gap: '12px', fontSize: '13px', color: '#6A6D76', lineHeight: 1.5, background: '#FFF7E6', border: '1px solid #F0DBA8', p: '12px 14px', borderRadius: '13px' }}>
        <Sym name="info" size={20} color="#8A6100" />
        In diesem Zeitraum wirst du für alle Termine automatisch abgesagt. Du kannst einzelne Termine manuell überschreiben.
      </Box>
      <Box sx={{ display: 'flex', gap: '10px' }}>
        <Field label="Von"><TextInput name="from" type="date" /></Field>
        <Field label="Bis"><TextInput name="to" type="date" /></Field>
      </Box>
      <Field label="Grund"><TextInput name="reason" placeholder="z. B. Urlaub" /></Field>
      <PrimaryButton label={sheet.mode === 'edit' ? 'Änderungen speichern' : 'Abwesenheit eintragen'} onClick={() => app.saveAbsence()} busy={app.state.busy === 'save'} />
    </Box>
  );
}

export function NewsFormSheet({ app, sheet }: SheetProps) {
  const { state } = app;
  const t = buildTokens(state.primaryColor);
  void sheet;
  const F = app.state.form;
  const pin = (
    <ButtonBase key="pin" onClick={() => app.setFormVal({ pinned: !F.pinned })} sx={{ display: 'flex', alignItems: 'center', gap: '12px', width: '100%', p: '12px 14px', borderRadius: '13px', cursor: 'pointer', border: '1px solid #E6E7EE', background: '#F4F4FA' }}>
      <Sym name="push_pin" size={20} color={F.pinned ? t.primary : '#9A9DA6'} />
      <Box component="span" sx={{ flex: 1, textAlign: 'left', fontSize: '14px', fontWeight: 500 }}>Oben anpinnen</Box>
      <Box component="span" sx={{ width: '44px', height: '26px', borderRadius: '999px', background: F.pinned ? t.primary : '#C8CAD2', position: 'relative' }}>
        <Box component="span" sx={{ position: 'absolute', top: '3px', left: F.pinned ? '21px' : '3px', width: '20px', height: '20px', borderRadius: '50%', background: '#fff', transition: 'left .2s' }} />
      </Box>
    </ButtonBase>
  );
  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: '16px' }}>
      <Field label="Titel"><TextInput name="title" placeholder="Überschrift" /></Field>
      <Field label="Text"><TextArea name="body" placeholder="Was gibt es Neues?" minHeight={120} /></Field>
      {pin}
      <PrimaryButton label="Veröffentlichen" onClick={() => app.saveNews()} busy={app.state.busy === 'save'} />
    </Box>
  );
}

export function PollFormSheet({ app, sheet }: SheetProps) {
  const { state } = app;
  const t = buildTokens(state.primaryColor);
  void sheet;
  const F = app.state.form;
  const toggle = (key: string, label: string, icon: string) => (
    <ButtonBase key={key} onClick={() => app.setFormVal({ [key]: !F[key] })} sx={{ flex: 1, display: 'flex', alignItems: 'center', gap: '9px', p: '12px', borderRadius: '13px', cursor: 'pointer', border: '1px solid ' + (F[key] ? t.primary : '#E0E2EA'), background: F[key] ? t.primaryContainer : '#fff' }}>
      <Sym name={icon} size={19} color={F[key] ? t.primary : '#6A6D76'} />
      <Box component="span" sx={{ fontSize: '13px', fontWeight: 600, color: F[key] ? t.onPrimaryContainer : '#44474E' }}>{label}</Box>
    </ButtonBase>
  );
  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: '16px' }}>
      <Field label="Frage"><TextInput name="question" placeholder="Worüber soll abgestimmt werden?" /></Field>
      <Box>
        <Box sx={labelSx}>Antwortoptionen</Box>
        <Box sx={{ display: 'flex', flexDirection: 'column', gap: '8px' }}>
          <TextInput name="opt0" placeholder="Option 1" />
          <TextInput name="opt1" placeholder="Option 2" />
          <TextInput name="opt2" placeholder="Option 3 (optional)" />
          <TextInput name="opt3" placeholder="Option 4 (optional)" />
        </Box>
      </Box>
      <Box sx={{ display: 'flex', gap: '8px' }}>
        {toggle('multiple', 'Mehrfachauswahl', 'checklist')}
        {toggle('anonymous', 'Anonym', 'visibility_off')}
      </Box>
      <PrimaryButton label="Umfrage erstellen" onClick={() => app.savePoll()} busy={app.state.busy === 'save'} />
    </Box>
  );
}
