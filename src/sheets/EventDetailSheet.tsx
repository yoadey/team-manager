import Box from '@mui/material/Box';
import ButtonBase from '@mui/material/ButtonBase';
import { buildTokens, fmtDateLong, fmtDateTime, hhmm, statusMeta, typeMeta } from '../theme/tokens';
import { Av, Chip, IconBtn, inputSx, SectionTitle, SpinnerBox, Sym } from '../components/ui';
import type { AttendanceRow, AttendanceStatus, EventComment, TeamEvent } from '../services/types';
import type { SheetProps } from './types';

export function EventDetailSheet({ app, sheet }: SheetProps) {
  const { state } = app;
  const t = buildTokens(state.primaryColor);

  const e: TeamEvent | null = sheet.event;
  if (!e) return <SpinnerBox />;

  const tm = typeMeta(e.type);
  const today = new Date().toISOString().slice(0, 10);
  const isPast = e.date < today;
  const myStatus = e.myStatus;
  const canEdit = app.can('events', 'write');
  const notNom = myStatus === 'not_nominated';
  const me = state.user!.id;
  const cancelled = e.status === 'cancelled';

  const banner = (
    <Box key="b" sx={{ display: 'flex', alignItems: 'center', gap: '10px', flexWrap: 'wrap', mb: '10px' }}>
      <Chip key="tm" label={tm.label} color={tm.color} bg={tm.bg} icon={tm.icon} fs={12} />
      {e.recurring ? <Chip key="r" label="wöchentlich" color="#6A6D76" bg="#ECEDF3" icon="repeat" fs={12} /> : null}
    </Box>
  );

  const dateLine = (
    <Box key="dl" sx={{ fontSize: '13px', color: '#6A6D76', fontWeight: 500, m: '0 2px 12px' }}>{fmtDateLong(e.date)}</Box>
  );

  const cancelBanner = cancelled ? (
    <Box key="cb" sx={{ display: 'flex', alignItems: 'center', gap: '10px', p: '13px 14px', background: '#FFDAD6', borderRadius: '14px', mb: '14px', color: '#8C1410' }}>
      <Sym name="event_busy" size={20} color="#BA1A1A" />
      <Box key="t" component="span" sx={{ flex: 1, fontSize: '13px', fontWeight: 600 }}>Dieser Termin wurde abgesagt – er bleibt sichtbar, Rückmeldungen sind nicht mehr nötig.</Box>
      {canEdit ? (
        <ButtonBase key="r" onClick={() => app.askEventAction('reactivate', e)} sx={{ border: 'none', background: '#fff', color: '#BA1A1A', borderRadius: '9px', p: '7px 12px', fontSize: '12px', fontWeight: 700, cursor: 'pointer', flex: '0 0 auto' }}>Aktivieren</ButtonBase>
      ) : null}
    </Box>
  ) : null;

  const info = (
    <Box key="info" sx={{ display: 'flex', flexDirection: 'column', gap: '1px', borderRadius: '16px', overflow: 'hidden', border: '1px solid #E6E7EE', mb: '14px' }}>
      {e.meetTime ? (
        <Box key="meet" sx={{ display: 'flex', alignItems: 'center', gap: '12px', p: '13px 14px', background: '#FFF7E6' }}>
          <Sym name="login" size={20} color="#8A6100" />
          <Box key="t" component="span" sx={{ flex: 1 }}><b>Treffen {hhmm(e.meetTime)}</b></Box>
          {e.meetTimeMandatory ? <Chip key="p" label="Pflicht" color="#8A6100" bg="#FCE2B8" /> : null}
        </Box>
      ) : null}
      <Box key="time" sx={{ display: 'flex', alignItems: 'center', gap: '12px', p: '13px 14px', background: '#fff' }}>
        <Sym name="schedule" size={20} color="#6A6D76" />
        <Box key="t" component="span" sx={{ flex: 1 }}>Beginn–Ende <b>{hhmm(e.startTime) + '–' + hhmm(e.endTime)}</b></Box>
      </Box>
      {e.location ? (
        <Box key="loc" sx={{ display: 'flex', alignItems: 'center', gap: '12px', p: '13px 14px', background: '#fff' }}>
          <Sym name="place" size={20} color="#6A6D76" />
          <Box key="t" component="span" sx={{ flex: 1 }}>{e.location}</Box>
        </Box>
      ) : null}
    </Box>
  );

  const note = e.note ? (
    <Box key="note" sx={{ display: 'flex', gap: '10px', p: '12px 14px', background: '#F4F4FA', borderRadius: '14px', mb: '14px' }}>
      <Sym name="sticky_note_2" size={18} color="#6A6D76" />
      <Box key="t" component="span" sx={{ fontSize: '13px', color: '#44474E', lineHeight: 1.5 }}>{e.note}</Box>
    </Box>
  ) : null;

  const result = e.result ? (
    <Box key="res" sx={{ display: 'flex', gap: '10px', p: '12px 14px', background: '#EAF6EA', borderRadius: '14px', mb: '14px' }}>
      <Sym name="emoji_events" size={18} color="#2E7D32" />
      <Box key="t" component="span" sx={{ fontSize: '13px', color: '#235C26', fontWeight: 600, lineHeight: 1.5 }}>{'Ergebnis: ' + e.result}</Box>
    </Box>
  ) : null;

  // my response
  let respond: React.ReactNode = null;
  if (!isPast && !cancelled) {
    if (notNom) {
      respond = (
        <Box key="nn" sx={{ display: 'flex', alignItems: 'center', gap: '10px', p: '13px 14px', background: '#F0F0F4', borderRadius: '14px', mb: '16px', color: '#6A6D76', fontSize: '13px' }}>
          <Sym name="block" size={20} color="#9A9DA6" />
          Du bist für diesen Termin nicht nominiert und kannst nicht zu-/absagen.
        </Box>
      );
    } else {
      const rb = (label: string, icon: string, st: AttendanceStatus, active: boolean, actCol: string, actBg: string, passBg: string, passCol: string) => (
        <ButtonBase key={st} onClick={() => app.setMyStatus(e.id, st)} sx={{ flex: 1, display: 'flex', alignItems: 'center', justifyContent: 'center', gap: '7px', p: '12px 6px', borderRadius: '14px', cursor: 'pointer', fontWeight: 700, fontSize: '13px', border: 'none', background: active ? actBg : passBg, color: active ? actCol : passCol }}>
          <Sym name={icon} size={19} color={active ? actCol : passCol} />
          {label}
        </ButtonBase>
      );
      const myC = e.myReason || '';
      const commentRow = (
        <ButtonBase key="mc" onClick={() => app.openComment(e, { userId: me, name: 'Du', status: myStatus, reason: myC })} sx={{ display: 'flex', alignItems: 'center', gap: '10px', width: '100%', mb: '16px', p: '11px 13px', borderRadius: '13px', border: '1px solid #E6E7EE', background: '#fff', cursor: 'pointer', textAlign: 'left', justifyContent: 'flex-start' }}>
          <Sym name="chat_bubble" size={18} color={myC ? t.primary : '#9A9DA6'} />
          <Box key="t" component="span" sx={{ flex: 1, fontSize: '13px', color: myC ? '#44474E' : '#9A9DA6', whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis' }}>{myC || 'Kurzen Kommentar hinzufügen (optional)'}</Box>
          <Sym name={myC ? 'edit' : 'add'} size={18} color="#9A9DA6" />
        </ButtonBase>
      );
      respond = (
        <Box key="rp">
          {e.myAuto ? (
            <Box key="auto" sx={{ fontSize: '12px', color: '#9A5B00', background: '#FFF1D6', borderRadius: '10px', p: '8px 12px', mb: '10px', display: 'flex', alignItems: 'center', gap: '7px' }}>
              <Sym name="info" size={16} color="#9A5B00" />
              {e.responseMode === 'opt_out' ? 'Ohne Reaktion giltst du als zugesagt.' : (myStatus === 'no' ? 'Automatisch abgesagt (geplante Abwesenheit) – du kannst überschreiben.' : '')}
            </Box>
          ) : null}
          <Box key="btns" sx={{ display: 'flex', gap: '8px', mb: '10px' }}>
            {rb('Zusagen', 'check_circle', 'yes', myStatus === 'yes', '#fff', '#2E7D32', '#D7F0D8', '#235C26')}
            {rb('Unsicher', 'help', 'maybe', myStatus === 'maybe', '#fff', '#9A5B00', '#FFE5B8', '#8A6100')}
            {rb('Absagen', 'cancel', 'no', myStatus === 'no', '#fff', '#BA1A1A', '#FFDAD6', '#8C1410')}
          </Box>
          {commentRow}
        </Box>
      );
    }
  }

  // summary + list
  const total = e.summary.total || 1;
  const sumHead = (
    <SectionTitle right={
      <Box key="s" sx={{ display: 'flex', gap: '9px', fontSize: '12px', fontWeight: 700 }}>
        <Box key="y" component="span" sx={{ color: '#2E7D32' }}>{e.summary.yes + ' zu'}</Box>
        <Box key="mb" component="span" sx={{ color: '#9A5B00' }}>{e.summary.maybe + ' uns.'}</Box>
        <Box key="n" component="span" sx={{ color: '#BA1A1A' }}>{e.summary.no + ' ab'}</Box>
        <Box key="p" component="span" sx={{ color: '#6A6D76' }}>{e.summary.pending + ' offen'}</Box>
      </Box>
    }>Teilnehmer</SectionTitle>
  );
  const bar = (
    <Box key="bar" sx={{ height: '8px', borderRadius: '6px', overflow: 'hidden', display: 'flex', background: '#ECEDF3', m: '2px 0 12px' }}>
      <Box key="y" sx={{ width: (e.summary.yes / total * 100) + '%', background: '#2E7D32' }} />
      <Box key="mb" sx={{ width: (e.summary.maybe / total * 100) + '%', background: '#E8910C' }} />
      <Box key="n" sx={{ width: (e.summary.no / total * 100) + '%', background: '#BA1A1A' }} />
    </Box>
  );

  const sbtn = (r: AttendanceRow, st: AttendanceStatus, icon: string, col: string) => {
    const sel = r.status === st;
    return (
      <ButtonBase key={st} title={statusMeta(st).label} onClick={() => app.setStatusFor(e, r, st)} sx={{ width: '30px', height: '30px', borderRadius: '8px', border: 'none', cursor: 'pointer', background: sel ? col : '#F1F2F6', color: sel ? '#fff' : '#A2A7B0', fontFamily: "'Material Symbols Outlined'", fontSize: '17px', flex: '0 0 auto' }}>{icon}</ButtonBase>
    );
  };

  const rows = (sheet.rows || []).map((r: AttendanceRow) => {
    const rsm = statusMeta(r.status);
    const mine = r.userId === me;
    const editable = (canEdit || mine) && !isPast;
    const seeC = app.canSeeComment(r);
    const notN = r.status === 'not_nominated';
    let controls: React.ReactNode[];
    if (notN) {
      controls = [
        <Chip key="c" label="Nicht nominiert" color="#6A6D76" bg="#ECEDF3" icon="block" />,
        (canEdit && !isPast) ? <IconBtn key="nom" icon="person_add" onClick={() => app.toggleNomination(e.id, r.userId, false)} color={t.primary} bg="#E7F0FF" title="Nominieren" /> : null,
        (canEdit || mine) ? <IconBtn key="cm" icon="chat_bubble" onClick={() => app.openComment(e, r)} color="#6A6D76" bg="#F4F4FA" title="Kommentar" /> : null,
      ];
    } else if (editable) {
      controls = [
        sbtn(r, 'yes', 'check', '#2E7D32'),
        sbtn(r, 'maybe', 'help', '#9A5B00'),
        sbtn(r, 'no', 'close', '#BA1A1A'),
        <IconBtn key="cm" icon="chat_bubble" onClick={() => app.openComment(e, r)} color="#6A6D76" bg="#F4F4FA" title="Kommentar" />,
        (canEdit && !isPast) ? <IconBtn key="rm" icon="person_remove" onClick={() => app.toggleNomination(e.id, r.userId, true)} color="#9A9DA6" bg="#F4F4FA" title="Nicht nominieren" /> : null,
      ];
    } else {
      controls = [<Chip key="c" label={rsm.label} color={rsm.color} bg={rsm.bg} icon={rsm.icon} />];
    }
    return (
      <Box key={r.userId} sx={{ display: 'flex', alignItems: 'center', gap: '10px', p: '8px', borderRadius: '12px', background: '#fff', opacity: notN ? 0.72 : 1 }}>
        <Av name={r.name} photo={r.photo} color={r.avatarColor} size={34} font={12} />
        <Box key="m" sx={{ flex: 1, minWidth: 0 }}>
          <Box key="n" sx={{ fontSize: '14px', fontWeight: 500, whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis' }}>{r.name + (mine ? ' · Du' : '')}</Box>
          {(seeC && r.reason)
            ? <Box key="r" sx={{ fontSize: '11px', color: '#9A5B00', whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis' }}>{r.reason}</Box>
            : <Box key="g" sx={{ fontSize: '11px', color: '#9A9DA6' }}>{r.group + (r.absent ? ' · abwesend' : '')}</Box>}
        </Box>
        <Box key="ctl" sx={{ display: 'flex', alignItems: 'center', gap: '5px', flex: '0 0 auto' }}>{controls}</Box>
      </Box>
    );
  });

  const edit = canEdit ? (
    <Box key="ed" sx={{ display: 'flex', gap: '10px', mt: '18px', flexWrap: 'wrap' }}>
      <ButtonBase key="e" onClick={() => app.openEventForm(e)} sx={{ flex: '1 1 130px', display: 'flex', alignItems: 'center', justifyContent: 'center', gap: '8px', p: '12px', borderRadius: '13px', border: '1px solid #C8CAD2', background: '#fff', color: '#44474E', fontWeight: 600, fontSize: '14px', cursor: 'pointer' }}>
        <Sym name="edit" size={19} color="#44474E" />Bearbeiten
      </ButtonBase>
      {!cancelled ? (
        <ButtonBase key="c" onClick={() => app.askEventAction('cancel', e)} sx={{ flex: '1 1 130px', display: 'flex', alignItems: 'center', justifyContent: 'center', gap: '8px', p: '12px', borderRadius: '13px', border: '1px solid #F0D9A8', background: '#FFF7E6', color: '#8A6100', fontWeight: 600, fontSize: '14px', cursor: 'pointer' }}>
          <Sym name="event_busy" size={19} color="#8A6100" />Absagen
        </ButtonBase>
      ) : null}
      <ButtonBase key="d" onClick={() => app.askEventAction('delete', e)} sx={{ display: 'flex', alignItems: 'center', justifyContent: 'center', gap: '8px', p: '12px 16px', borderRadius: '13px', border: '1px solid #F0C4C0', background: '#FFF4F3', color: '#BA1A1A', fontWeight: 600, cursor: 'pointer' }}>
        <Sym name="delete" size={19} color="#BA1A1A" />Löschen
      </ButtonBase>
    </Box>
  ) : null;

  // Kommentar-Thread
  const cms: EventComment[] = sheet.comments || [];
  const thread = (
    <Box key="th" sx={{ mt: '22px' }}>
      <SectionTitle>{'Kommentare' + (cms.length ? ' (' + cms.length + ')' : '')}</SectionTitle>
      <Box key="l" sx={{ display: 'flex', flexDirection: 'column', gap: '10px', mb: '12px' }}>
        {cms.length ? cms.map((c) => (
          <Box key={c.id} sx={{ display: 'flex', gap: '10px', alignItems: 'flex-start' }}>
            <Av name={c.name} photo={c.photo} color={c.color} size={32} font={12} />
            <Box key="m" sx={{ flex: 1, minWidth: 0, background: '#F4F4FA', borderRadius: '12px', p: '9px 12px' }}>
              <Box key="h" sx={{ display: 'flex', alignItems: 'center', gap: '7px', mb: '2px' }}>
                <Box key="n" component="span" sx={{ fontSize: '12px', fontWeight: 700 }}>{c.name}</Box>
                <Box key="t" component="span" sx={{ fontSize: '11px', color: '#9A9DA6' }}>{fmtDateTime(c.createdAt)}</Box>
              </Box>
              <Box key="b" sx={{ fontSize: '13px', color: '#44474E', lineHeight: 1.45, wordBreak: 'break-word' }}>{c.text}</Box>
            </Box>
            {(c.userId === me || canEdit) ? <IconBtn key="del" icon="delete" onClick={() => app.removeEventComment(e.id, c.id)} color="#BA1A1A" bg="#FFF4F3" title="Löschen" /> : null}
          </Box>
        )) : <Box key="e" sx={{ fontSize: '13px', color: '#9A9DA6', p: '4px 2px' }}>Noch keine Kommentare.</Box>}
      </Box>
      <Box key="add" sx={{ display: 'flex', gap: '8px' }}>
        <input
          key="i"
          name="newEventComment"
          value={app.state.form.newEventComment || ''}
          onChange={(ev) => app.onFormInput(ev)}
          onKeyDown={(ev) => { if (ev.key === 'Enter') app.postEventComment(e.id); }}
          placeholder="Kommentar schreiben…"
          style={{ ...inputSx, flex: 1 }}
        />
        <ButtonBase key="b" onClick={() => app.postEventComment(e.id)} sx={{ background: t.primary, color: t.onPrimary, border: 'none', borderRadius: '12px', p: '0 16px', fontWeight: 600, cursor: 'pointer', display: 'flex', alignItems: 'center' }}>
          <Sym name="send" size={18} color={t.onPrimary} />
        </ButtonBase>
      </Box>
    </Box>
  );

  return (
    <Box>
      {banner}
      {dateLine}
      {cancelBanner}
      {info}
      {note}
      {result}
      {respond}
      {sumHead}
      {bar}
      <Box key="rows" sx={{ display: 'flex', flexDirection: 'column', gap: '2px' }}>{rows}</Box>
      {thread}
      {edit}
    </Box>
  );
}
