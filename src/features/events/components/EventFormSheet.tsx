import Box from '@mui/material/Box';
import ButtonBase from '@mui/material/ButtonBase';
import { buildTokens, typeMeta } from '@/styles/tokens';
import { Field, labelSx, PrimaryButton, Sym, TextArea, TextInput } from '@/components/ui';
import type { Role } from '@/types';
import type { SheetProps } from '@/sheets/types';

export function EventFormSheet({ app, sheet }: SheetProps) {
  const { state } = app;
  const t = buildTokens(state.primaryColor);
  const F = app.state.form;

  const typeBtns = (['training', 'auftritt', 'event'] as const).map((tp) => {
    const meta = typeMeta(tp);
    const sel = F.type === tp;
    return (
      <ButtonBase key={tp} onClick={() => app.setFormVal({ type: tp })} sx={{ flex: 1, display: 'flex', flexDirection: 'column', alignItems: 'center', gap: '4px', p: '11px 6px', borderRadius: '13px', cursor: 'pointer', fontSize: '12px', fontWeight: 600, border: '1.5px solid ' + (sel ? meta.color : '#E0E2EA'), background: sel ? meta.bg : '#fff', color: sel ? meta.color : '#6A6D76' }}>
        <Sym name={meta.icon} size={18} color={sel ? meta.color : '#6A6D76'} />
        {tp === 'training' ? 'Training' : (tp === 'auftritt' ? 'Auftritt' : 'Event')}
      </ButtonBase>
    );
  });

  const modeDefs: [string, string, string, string][] = [
    ['opt_in', 'Aktiv zusagen', 'Standard offen – jeder sagt aktiv zu', 'login'],
    ['opt_out', 'Aktiv absagen', 'Alle gelten als zugesagt', 'logout'],
  ];
  const modeBtns = modeDefs.map(([v, l, d, ic]) => {
    const sel = F.responseMode === v;
    return (
      <ButtonBase key={v} onClick={() => app.setFormVal({ responseMode: v })} sx={{ flex: 1, display: 'flex', flexDirection: 'column', gap: '4px', p: '12px', borderRadius: '13px', cursor: 'pointer', textAlign: 'left', alignItems: 'stretch', justifyContent: 'flex-start', border: '1.5px solid ' + (sel ? t.primary : '#E0E2EA'), background: sel ? t.primaryContainer : '#fff' }}>
        <Box key="h" sx={{ display: 'flex', alignItems: 'center', gap: '6px', fontSize: '13px', fontWeight: 700, color: sel ? t.onPrimaryContainer : '#44474E' }}>
          <Sym name={ic} size={17} color={sel ? t.onPrimaryContainer : '#44474E'} />{l}
        </Box>
        <Box key="d" sx={{ fontSize: '11px', color: sel ? t.onPrimaryContainer : '#9A9DA6', lineHeight: 1.4 }}>{d}</Box>
      </ButtonBase>
    );
  });

  const meetToggle = (
    <ButtonBase key="mm" onClick={() => app.setFormVal({ meetTimeMandatory: !F.meetTimeMandatory })} sx={{ display: 'flex', alignItems: 'center', gap: '12px', width: '100%', p: '12px 14px', borderRadius: '13px', cursor: 'pointer', border: '1px solid #E6E7EE', background: '#F4F4FA', justifyContent: 'flex-start' }}>
      <Box key="c" component="span" sx={{ width: '22px', height: '22px', borderRadius: '7px', background: F.meetTimeMandatory ? t.primary : '#fff', border: '2px solid ' + (F.meetTimeMandatory ? t.primary : '#B0B3BC'), display: 'flex', alignItems: 'center', justifyContent: 'center', flex: '0 0 auto' }}>
        {F.meetTimeMandatory ? <Sym name="check" size={16} color="#fff" /> : null}
      </Box>
      <Box key="l" component="span" sx={{ flex: 1, textAlign: 'left', fontSize: '14px', fontWeight: 500 }}>Treffzeit ist Pflichtfeld</Box>
    </ButtonBase>
  );

  const recur = sheet.mode === 'create' ? (
    <Box key="rec" sx={{ borderTop: '1px solid #ECEDF3', pt: '14px' }}>
      <ButtonBase key="tg" onClick={() => app.setFormVal({ recurring: !F.recurring })} sx={{ display: 'flex', alignItems: 'center', gap: '12px', width: '100%', p: '4px 2px', cursor: 'pointer', background: 'transparent', border: 'none', justifyContent: 'flex-start' }}>
        <Sym name="repeat" size={20} color="#6A6D76" />
        <Box key="l" component="span" sx={{ flex: 1, textAlign: 'left', fontSize: '14px', fontWeight: 500 }}>Wöchentlich wiederholen</Box>
        <Box key="sw" component="span" sx={{ width: '44px', height: '26px', borderRadius: '999px', background: F.recurring ? t.primary : '#C8CAD2', position: 'relative', flex: '0 0 auto' }}>
          <Box component="span" sx={{ position: 'absolute', top: '3px', left: F.recurring ? '21px' : '3px', width: '20px', height: '20px', borderRadius: '50%', background: '#fff', transition: 'left .2s' }} />
        </Box>
      </ButtonBase>
      {F.recurring ? (
        <Box key="w" sx={{ mt: '10px' }}>
          <Field label="Anzahl Wochen"><TextInput name="repeatWeeks" type="number" min="2" max="26" /></Field>
        </Box>
      ) : null}
    </Box>
  ) : null;

  const nomSel = (
    <Box key="nomsel" sx={{ borderTop: '1px solid #ECEDF3', pt: '14px' }}>
      <Box key="l" sx={labelSx}>Nominierte Rollen</Box>
      <Box key="h" sx={{ fontSize: '12px', color: '#9A9DA6', m: '-2px 0 9px', lineHeight: 1.45 }}>Nur Mitglieder mit einer gewählten Rolle werden nominiert. Abgewählte Rollen können nicht zu-/absagen.</Box>
      <Box key="b" sx={{ display: 'flex', flexWrap: 'wrap', gap: '8px' }}>
        {state.roles.map((r: Role) => {
          const sel = (F.nominatedRoleIds || []).includes(r.id);
          return (
            <ButtonBase key={r.id} onClick={() => app.toggleFormNomRole(r.id)} sx={{ display: 'inline-flex', alignItems: 'center', gap: '7px', p: '8px 13px', borderRadius: '999px', cursor: 'pointer', fontSize: '13px', fontWeight: 600, border: '1.5px solid ' + (sel ? r.color : '#D0D2DA'), background: sel ? r.color + '1A' : '#fff', color: sel ? r.color : '#9A9DA6' }}>
              <Box key="d" component="span" sx={{ width: '9px', height: '9px', borderRadius: '50%', background: sel ? r.color : '#C0C2CA' }} />
              {r.name}
              {sel ? <Sym name="check" size={16} color={r.color} /> : null}
            </ButtonBase>
          );
        })}
      </Box>
    </Box>
  );

  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: '16px' }}>
      <Box key="type">
        <Box key="l" sx={labelSx}>Termintyp</Box>
        <Box key="b" sx={{ display: 'flex', gap: '8px' }}>{typeBtns}</Box>
      </Box>
      <Field label="Titel"><TextInput name="title" placeholder="z. B. Lateinformation – Training" /></Field>
      <Field label="Datum"><TextInput name="date" type="date" /></Field>
      <Box key="times" sx={{ display: 'flex', gap: '10px' }}>
        <Field label="Treffzeit"><TextInput name="meetT" type="time" /></Field>
        <Field label="Beginn"><TextInput name="startT" type="time" /></Field>
        <Field label="Ende"><TextInput name="endT" type="time" /></Field>
      </Box>
      {meetToggle}
      <Box key="mode">
        <Box key="l" sx={labelSx}>Rückmeldung</Box>
        <Box key="b" sx={{ display: 'flex', gap: '8px' }}>{modeBtns}</Box>
      </Box>
      {nomSel}
      <Field label="Ort"><TextInput name="location" placeholder="Halle / Adresse" /></Field>
      <Field label="Notiz"><TextArea name="note" placeholder="Hinweise für das Team…" minHeight={64} /></Field>
      {recur}
      {(sheet.mode === 'edit' && F.seriesId) ? (
        <Box key="serbtn" sx={{ display: 'flex', flexDirection: 'column', gap: '9px', mt: '4px' }}>
          <Box key="h" sx={{ display: 'flex', alignItems: 'center', gap: '7px', fontSize: '12px', color: '#6A6D76', fontWeight: 600 }}>
            <Sym name="repeat" size={16} color="#9A9DA6" />Teil einer Serie – was soll gespeichert werden?
          </Box>
          <Box key="b" sx={{ display: 'flex', gap: '10px' }}>
            <ButtonBase key="one" onClick={() => app.saveEvent('single')} disabled={app.state.busy === 'save'} sx={{ flex: 1, display: 'flex', alignItems: 'center', justifyContent: 'center', gap: '8px', p: '13px', borderRadius: '13px', border: '1px solid ' + t.primary, background: '#fff', color: t.primary, fontWeight: 700, fontSize: '14px', cursor: 'pointer' }}>
              <Sym name="event" size={18} color={t.primary} />Nur dieser Termin
            </ButtonBase>
            <ButtonBase key="all" onClick={() => app.saveEvent('series')} disabled={app.state.busy === 'save'} sx={{ flex: 1, display: 'flex', alignItems: 'center', justifyContent: 'center', gap: '8px', p: '13px', borderRadius: '13px', border: 'none', background: t.primary, color: t.onPrimary, fontWeight: 700, fontSize: '14px', cursor: 'pointer' }}>
              <Sym name="repeat" size={18} color={t.onPrimary} />Ganze Serie
            </ButtonBase>
          </Box>
        </Box>
      ) : (
        <PrimaryButton label={sheet.mode === 'edit' ? 'Änderungen speichern' : 'Termin anlegen'} onClick={() => app.saveEvent('single')} busy={app.state.busy === 'save'} />
      )}
    </Box>
  );
}
