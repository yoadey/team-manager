import { useEffect, useState } from 'react';
import Box from '@mui/material/Box';
import ButtonBase from '@mui/material/ButtonBase';
import { todayLocalDate } from '@/utils/date';
import { buildTokens, fmtDateLong, fmtDateTime, hhmm, statusMeta, typeMeta, NEUTRAL } from '@/styles/tokens';
import { Av, Chip, EmptyState, IconBtn, inputSx, SectionTitle, SpinnerBox, Sym } from '@/components/ui';
import type { AttendanceRow, EventComment, TeamEvent } from '../types';
import type { AttendanceStatus } from '@/types';
import type { SheetProps } from '@/sheets/types';
import { reportActionError } from '@/utils/errors';
import { useEventDetailQuery } from '../hooks/useEventQueries';
import { t } from '@/i18n';

export function EventDetailSheet({ app, sheet }: SheetProps) {
  const { state } = app;
  const tk = buildTokens(state.primaryColor);
  const eventId = sheet.eventId ?? null;
  const detail = useEventDetailQuery(app.api, state.activeTeamId, eventId);
  const [newComment, setNewComment] = useState('');

  // A thrown INITIAL fetch (a genuine network failure, or events:none after a
  // permission downgrade) never resolves to a `{ event: null }` success --
  // without this, the sheet was stuck spinning forever; close it instead of
  // leaving a permanently-loading sheet open, after surfacing why. A failed
  // BACKGROUND refetch (e.g. the invalidation after an unrelated attendance
  // mutation hitting a transient blip) must not do this -- `detail.data` is
  // still the last good response, so the sheet keeps showing it rather than
  // discarding valid, already-rendered content over a momentary hiccup.
  useEffect(() => {
    if (!detail.isError || detail.data) return;
    reportActionError(
      { setState: app.setState, toastMsg: app.toastMsg, onAuthError: app.logout },
      detail.error,
      'error.load',
    );
    app.setState((s) =>
      s.sheet && s.sheet.type === 'eventDetail' && s.sheet.eventId === eventId ? { sheet: null } : {},
    );
  }, [detail.isError, detail.error, detail.data, eventId, app]);

  if (detail.isError && !detail.data) return null;
  if (detail.isLoading) return <SpinnerBox />;

  // event === null once the query has resolved is a confirmed-missing event
  // (deleted or inaccessible), distinct from still-loading above.
  const e: TeamEvent | null = detail.data?.event ?? null;
  if (!e) return <EmptyState icon="event_busy" text={t('events.detailNotFound')} />;

  const tm = typeMeta(e.type);
  const today = todayLocalDate();
  const isPast = e.date < today;
  const myStatus = e.myStatus;
  const canEdit = app.can('events', 'write');
  const notNom = myStatus === 'not_nominated';
  const me = state.user!.id;
  const cancelled = e.status === 'cancelled';

  const banner = (
    <Box key="b" sx={{ display: 'flex', alignItems: 'center', gap: '10px', flexWrap: 'wrap', mb: '10px' }}>
      <Chip key="tm" label={tm.label} color={tm.color} bg={tm.bg} icon={tm.icon} fs={12} />
      {e.recurring ? (
        <Chip key="r" label={t('events.weekly')} color={NEUTRAL.secondary} bg={NEUTRAL.line2} icon="repeat" fs={12} />
      ) : null}
    </Box>
  );

  const dateLine = (
    <Box key="dl" sx={{ fontSize: '13px', color: NEUTRAL.secondary, fontWeight: 500, m: '0 2px 12px' }}>
      {fmtDateLong(e.date)}
    </Box>
  );

  const cancelBanner = cancelled ? (
    <Box
      key="cb"
      sx={{
        display: 'flex',
        alignItems: 'center',
        gap: '10px',
        p: '13px 14px',
        background: NEUTRAL.errorBg,
        borderRadius: '14px',
        mb: '14px',
        color: NEUTRAL.error,
      }}
    >
      <Sym name="event_busy" size={20} color={NEUTRAL.error} />
      <Box key="t" component="span" sx={{ flex: 1, fontSize: '13px', fontWeight: 600 }}>
        {t('events.cancelledBanner')}
      </Box>
      {canEdit ? (
        <ButtonBase
          key="r"
          onClick={() => app.askEventAction('reactivate', e)}
          sx={{
            border: 'none',
            background: NEUTRAL.card,
            color: NEUTRAL.error,
            borderRadius: '9px',
            p: '7px 12px',
            fontSize: '12px',
            fontWeight: 700,
            cursor: 'pointer',
            flex: '0 0 auto',
          }}
        >
          {t('events.reactivate')}
        </ButtonBase>
      ) : null}
    </Box>
  ) : null;

  const info = (
    <Box
      key="info"
      sx={{
        display: 'flex',
        flexDirection: 'column',
        gap: '1px',
        borderRadius: '16px',
        overflow: 'hidden',
        border: `1px solid ${NEUTRAL.line}`,
        mb: '14px',
      }}
    >
      {e.meetTime ? (
        <Box
          key="meet"
          sx={{ display: 'flex', alignItems: 'center', gap: '12px', p: '13px 14px', background: NEUTRAL.warnBg }}
        >
          <Sym name="login" size={20} color={NEUTRAL.warn} />
          <Box key="t" component="span" sx={{ flex: 1 }}>
            <b>{t('events.meetTime', { time: hhmm(e.meetTime) })}</b>
          </Box>
          {e.meetTimeMandatory ? (
            <Chip key="p" label={t('events.mandatory')} color={NEUTRAL.warn} bg={NEUTRAL.warnBg} />
          ) : null}
        </Box>
      ) : null}
      <Box
        key="time"
        sx={{ display: 'flex', alignItems: 'center', gap: '12px', p: '13px 14px', background: NEUTRAL.card }}
      >
        <Sym name="schedule" size={20} color={NEUTRAL.secondary} />
        <Box key="t" component="span" sx={{ flex: 1 }}>
          {t('events.startEnd')} <b>{hhmm(e.startTime) + '–' + hhmm(e.endTime)}</b>
        </Box>
      </Box>
      {e.location ? (
        <Box
          key="loc"
          sx={{ display: 'flex', alignItems: 'center', gap: '12px', p: '13px 14px', background: NEUTRAL.card }}
        >
          <Sym name="place" size={20} color={NEUTRAL.secondary} />
          <Box key="t" component="span" sx={{ flex: 1 }}>
            {e.location}
          </Box>
        </Box>
      ) : null}
    </Box>
  );

  const note = e.note ? (
    <Box
      key="note"
      sx={{
        display: 'flex',
        gap: '10px',
        p: '12px 14px',
        background: NEUTRAL.sidebar,
        borderRadius: '14px',
        mb: '14px',
      }}
    >
      <Sym name="sticky_note_2" size={18} color={NEUTRAL.secondary} />
      <Box key="t" component="span" sx={{ fontSize: '13px', color: NEUTRAL.onSurfaceVariant, lineHeight: 1.5 }}>
        {e.note}
      </Box>
    </Box>
  ) : null;

  const result = e.result ? (
    <Box
      key="res"
      sx={{
        display: 'flex',
        gap: '10px',
        p: '12px 14px',
        background: NEUTRAL.successBg,
        borderRadius: '14px',
        mb: '14px',
      }}
    >
      <Sym name="emoji_events" size={18} color={NEUTRAL.success} />
      <Box key="t" component="span" sx={{ fontSize: '13px', color: NEUTRAL.success, fontWeight: 600, lineHeight: 1.5 }}>
        {t('events.result', { val: e.result })}
      </Box>
    </Box>
  ) : null;

  // my response
  let respond: React.ReactNode = null;
  if (!isPast && !cancelled) {
    if (notNom) {
      respond = (
        <Box
          key="nn"
          sx={{
            display: 'flex',
            alignItems: 'center',
            gap: '10px',
            p: '13px 14px',
            background: NEUTRAL.line2,
            borderRadius: '14px',
            mb: '16px',
            color: NEUTRAL.secondary,
            fontSize: '13px',
          }}
        >
          <Sym name="block" size={20} color={NEUTRAL.faint} />
          {t('events.notNominated')}
        </Box>
      );
    } else {
      const rb = (
        label: string,
        icon: string,
        st: AttendanceStatus,
        active: boolean,
        actCol: string,
        actBg: string,
        passBg: string,
        passCol: string,
      ) => (
        <ButtonBase
          key={st}
          onClick={() => app.setMyStatus(e.id, st, e.myReason)}
          sx={{
            flex: 1,
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            gap: '7px',
            p: '12px 6px',
            borderRadius: '14px',
            cursor: 'pointer',
            fontWeight: 700,
            fontSize: '13px',
            border: 'none',
            background: active ? actBg : passBg,
            color: active ? actCol : passCol,
          }}
        >
          <Sym name={icon} size={19} color={active ? actCol : passCol} />
          {label}
        </ButtonBase>
      );
      const myC = e.myReason || '';
      const commentRow = (
        <ButtonBase
          key="mc"
          onClick={() => app.openComment(e, { userId: me, name: t('events.meLabel'), status: myStatus, reason: myC })}
          sx={{
            display: 'flex',
            alignItems: 'center',
            gap: '10px',
            width: '100%',
            mb: '16px',
            p: '11px 13px',
            borderRadius: '13px',
            border: `1px solid ${NEUTRAL.line}`,
            background: NEUTRAL.card,
            cursor: 'pointer',
            textAlign: 'left',
            justifyContent: 'flex-start',
          }}
        >
          <Sym name="chat_bubble" size={18} color={myC ? tk.primary : NEUTRAL.faint} />
          <Box
            key="t"
            component="span"
            sx={{
              flex: 1,
              fontSize: '13px',
              color: myC ? NEUTRAL.onSurfaceVariant : NEUTRAL.faint,
              whiteSpace: 'nowrap',
              overflow: 'hidden',
              textOverflow: 'ellipsis',
            }}
          >
            {myC || t('events.commentPlaceholder')}
          </Box>
          <Sym name={myC ? 'edit' : 'add'} size={18} color={NEUTRAL.faint} />
        </ButtonBase>
      );
      respond = (
        <Box key="rp">
          {e.myAuto ? (
            <Box
              key="auto"
              sx={{
                fontSize: '12px',
                color: NEUTRAL.warn,
                background: NEUTRAL.warnBg,
                borderRadius: '10px',
                p: '8px 12px',
                mb: '10px',
                display: 'flex',
                alignItems: 'center',
                gap: '7px',
              }}
            >
              <Sym name="info" size={16} color={NEUTRAL.warn} />
              {myStatus === 'no' ? t('events.autoAbsent') : e.responseMode === 'opt_out' ? t('events.autoOptOut') : ''}
            </Box>
          ) : null}
          <Box key="btns" sx={{ display: 'flex', gap: '8px', mb: '10px' }}>
            {rb(
              t('events.rsvpYes'),
              'check_circle',
              'yes',
              myStatus === 'yes',
              '#fff',
              NEUTRAL.success,
              NEUTRAL.successBg,
              NEUTRAL.success,
            )}
            {rb(
              t('events.rsvpMaybe'),
              'help',
              'maybe',
              myStatus === 'maybe',
              '#fff',
              NEUTRAL.warn,
              NEUTRAL.warnBg,
              NEUTRAL.warn,
            )}
            {rb(
              t('events.rsvpNo'),
              'cancel',
              'no',
              myStatus === 'no',
              '#fff',
              NEUTRAL.error,
              NEUTRAL.errorBg,
              NEUTRAL.error,
            )}
          </Box>
          {commentRow}
        </Box>
      );
    }
  }

  // summary + list
  const total = e.summary.total || 1;
  const sumHead = (
    <SectionTitle
      right={
        <Box key="s" sx={{ display: 'flex', gap: '9px', fontSize: '12px', fontWeight: 700 }}>
          <Box key="y" component="span" sx={{ color: NEUTRAL.success }}>
            {e.summary.yes + ' ' + t('events.summaryYes', { n: '' }).replace('{n} ', '')}
          </Box>
          <Box key="mb" component="span" sx={{ color: NEUTRAL.warn }}>
            {e.summary.maybe + ' ' + t('events.summaryMaybe', { n: '' }).replace('{n} ', '')}
          </Box>
          <Box key="n" component="span" sx={{ color: NEUTRAL.error }}>
            {e.summary.no + ' ' + t('events.summaryNo', { n: '' }).replace('{n} ', '')}
          </Box>
          <Box key="p" component="span" sx={{ color: NEUTRAL.secondary }}>
            {e.summary.pending + ' ' + t('events.summaryPending', { n: '' }).replace('{n} ', '')}
          </Box>
        </Box>
      }
    >
      {t('events.participants')}
    </SectionTitle>
  );
  const bar = (
    <Box
      key="bar"
      sx={{
        height: '8px',
        borderRadius: '6px',
        overflow: 'hidden',
        display: 'flex',
        background: NEUTRAL.line2,
        m: '2px 0 12px',
      }}
    >
      <Box key="y" sx={{ width: (e.summary.yes / total) * 100 + '%', background: NEUTRAL.success }} />
      <Box key="mb" sx={{ width: (e.summary.maybe / total) * 100 + '%', background: '#E8910C' }} />
      <Box key="n" sx={{ width: (e.summary.no / total) * 100 + '%', background: NEUTRAL.error }} />
    </Box>
  );

  const sbtn = (r: AttendanceRow, st: AttendanceStatus, icon: string, col: string) => {
    const sel = r.status === st;
    return (
      <ButtonBase
        key={st}
        title={statusMeta(st).label}
        aria-label={statusMeta(st).label}
        aria-pressed={sel}
        onClick={() => app.setStatusFor(e, r, st)}
        sx={{
          width: '30px',
          height: '30px',
          borderRadius: '8px',
          border: 'none',
          cursor: 'pointer',
          background: sel ? col : '#F1F2F6',
          color: sel ? '#fff' : NEUTRAL.faint,
          fontFamily: "'Material Symbols Outlined'",
          fontSize: '17px',
          flex: '0 0 auto',
        }}
      >
        {icon}
      </ButtonBase>
    );
  };

  const rows = (detail.data?.rows || []).map((r: AttendanceRow) => {
    const rsm = statusMeta(r.status);
    const mine = r.userId === me;
    const editable = (canEdit || mine) && !isPast;
    const seeC = app.canSeeComment(r);
    const notN = r.status === 'not_nominated';
    let controls: React.ReactNode[];
    if (notN) {
      controls = [
        <Chip
          key="c"
          label={t('events.notNominatedLabel')}
          color={NEUTRAL.secondary}
          bg={NEUTRAL.line2}
          icon="block"
        />,
        canEdit && !isPast ? (
          <IconBtn
            key="nom"
            icon="person_add"
            onClick={() => app.toggleNomination(e.id, r.userId, false)}
            color={tk.primary}
            bg="#E7F0FF"
            title={t('events.nominate')}
          />
        ) : null,
        canEdit || mine ? (
          <IconBtn
            key="cm"
            icon="chat_bubble"
            onClick={() => app.openComment(e, r)}
            color={NEUTRAL.secondary}
            bg={NEUTRAL.sidebar}
            title={t('events.comment')}
          />
        ) : null,
      ];
    } else if (editable) {
      controls = [
        sbtn(r, 'yes', 'check', NEUTRAL.success),
        sbtn(r, 'maybe', 'help', NEUTRAL.warn),
        sbtn(r, 'no', 'close', NEUTRAL.error),
        <IconBtn
          key="cm"
          icon="chat_bubble"
          onClick={() => app.openComment(e, r)}
          color={NEUTRAL.secondary}
          bg={NEUTRAL.sidebar}
          title={t('events.comment')}
        />,
        canEdit && !isPast ? (
          <IconBtn
            key="rm"
            icon="person_remove"
            onClick={() => app.toggleNomination(e.id, r.userId, true)}
            color={NEUTRAL.faint}
            bg={NEUTRAL.sidebar}
            title={t('events.deNominate')}
          />
        ) : null,
      ];
    } else {
      controls = [<Chip key="c" label={rsm.label} color={rsm.color} bg={rsm.bg} icon={rsm.icon} />];
    }
    return (
      <Box
        key={r.userId}
        sx={{
          display: 'flex',
          alignItems: 'center',
          gap: '10px',
          p: '8px',
          borderRadius: '12px',
          background: NEUTRAL.card,
          opacity: notN ? 0.72 : 1,
        }}
      >
        <Av name={r.name} photo={r.photo} color={r.avatarColor} size={34} font={12} />
        <Box key="m" sx={{ flex: 1, minWidth: 0 }}>
          <Box
            key="n"
            sx={{
              fontSize: '14px',
              fontWeight: 500,
              whiteSpace: 'nowrap',
              overflow: 'hidden',
              textOverflow: 'ellipsis',
            }}
          >
            {r.name + (mine ? ' · ' + t('events.meLabel') : '')}
          </Box>
          {seeC && r.reason ? (
            <Box
              key="r"
              sx={{
                fontSize: '11px',
                color: NEUTRAL.warn,
                whiteSpace: 'nowrap',
                overflow: 'hidden',
                textOverflow: 'ellipsis',
              }}
            >
              {r.reason}
            </Box>
          ) : (
            <Box key="g" sx={{ fontSize: '11px', color: NEUTRAL.faint }}>
              {r.group + (r.absent ? ' · ' + t('events.absent') : '')}
            </Box>
          )}
        </Box>
        <Box key="ctl" sx={{ display: 'flex', alignItems: 'center', gap: '5px', flex: '0 0 auto' }}>
          {controls}
        </Box>
      </Box>
    );
  });

  const edit = canEdit ? (
    <Box key="ed" sx={{ display: 'flex', gap: '10px', mt: '18px', flexWrap: 'wrap' }}>
      <ButtonBase
        key="e"
        onClick={() => app.openEventForm(e)}
        sx={{
          flex: '1 1 130px',
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
        {t('events.edit')}
      </ButtonBase>
      {!cancelled ? (
        <ButtonBase
          key="c"
          onClick={() => app.askEventAction('cancel', e)}
          sx={{
            flex: '1 1 130px',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            gap: '8px',
            p: '12px',
            borderRadius: '13px',
            border: '1px solid #F0D9A8',
            background: NEUTRAL.warnBg,
            color: NEUTRAL.warn,
            fontWeight: 600,
            fontSize: '14px',
            cursor: 'pointer',
          }}
        >
          <Sym name="event_busy" size={19} color={NEUTRAL.warn} />
          {t('events.cancel')}
        </ButtonBase>
      ) : null}
      <ButtonBase
        key="d"
        onClick={() => app.askEventAction('delete', e)}
        sx={{
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          gap: '8px',
          p: '12px 16px',
          borderRadius: '13px',
          border: '1px solid #F0C4C0',
          background: NEUTRAL.errorBg,
          color: NEUTRAL.error,
          fontWeight: 600,
          cursor: 'pointer',
        }}
      >
        <Sym name="delete" size={19} color={NEUTRAL.error} />
        {t('events.delete')}
      </ButtonBase>
    </Box>
  ) : null;

  // Comment thread
  const cms: EventComment[] = detail.data?.comments || [];
  const thread = (
    <Box key="th" sx={{ mt: '22px' }}>
      <SectionTitle>{t('events.comments') + (cms.length ? ' (' + cms.length + ')' : '')}</SectionTitle>
      <Box key="l" sx={{ display: 'flex', flexDirection: 'column', gap: '10px', mb: '12px' }}>
        {cms.length ? (
          cms.map((c) => (
            <Box key={c.id} sx={{ display: 'flex', gap: '10px', alignItems: 'flex-start' }}>
              <Av name={c.name} photo={c.photo} color={c.color} size={32} font={12} />
              <Box
                key="m"
                sx={{ flex: 1, minWidth: 0, background: NEUTRAL.sidebar, borderRadius: '12px', p: '9px 12px' }}
              >
                <Box key="h" sx={{ display: 'flex', alignItems: 'center', gap: '7px', mb: '2px' }}>
                  <Box key="n" component="span" sx={{ fontSize: '12px', fontWeight: 700 }}>
                    {c.name}
                  </Box>
                  <Box key="t" component="span" sx={{ fontSize: '11px', color: NEUTRAL.faint }}>
                    {fmtDateTime(c.createdAt)}
                  </Box>
                </Box>
                <Box
                  key="b"
                  sx={{ fontSize: '13px', color: NEUTRAL.onSurfaceVariant, lineHeight: 1.45, wordBreak: 'break-word' }}
                >
                  {c.text}
                </Box>
              </Box>
              {c.userId === me || canEdit ? (
                <IconBtn
                  key="del"
                  icon="delete"
                  onClick={() => app.removeEventComment(e.id, c.id)}
                  color={NEUTRAL.error}
                  bg={NEUTRAL.errorBg}
                  title={t('events.delete')}
                />
              ) : null}
            </Box>
          ))
        ) : (
          <Box key="e" sx={{ fontSize: '13px', color: NEUTRAL.faint, p: '4px 2px' }}>
            {t('events.noComments')}
          </Box>
        )}
      </Box>
      <Box key="add" sx={{ display: 'flex', gap: '8px' }}>
        <input
          key="i"
          name="newEventComment"
          aria-label={t('events.commentWrite')}
          value={newComment}
          onChange={(ev) => setNewComment(ev.target.value)}
          onKeyDown={(ev) => {
            if (ev.key === 'Enter') {
              const txt = newComment;
              setNewComment('');
              app.postEventComment(e.id, txt).then((ok) => {
                if (!ok) setNewComment(txt);
              });
            }
          }}
          placeholder={t('events.commentWrite')}
          maxLength={10000}
          style={{ ...inputSx, flex: 1 }}
        />
        <ButtonBase
          key="b"
          aria-label={t('events.commentSend')}
          onClick={() => {
            const txt = newComment;
            setNewComment('');
            app.postEventComment(e.id, txt).then((ok) => {
              if (!ok) setNewComment(txt);
            });
          }}
          sx={{
            background: tk.primary,
            color: tk.onPrimary,
            border: 'none',
            borderRadius: '12px',
            p: '0 16px',
            fontWeight: 600,
            cursor: 'pointer',
            display: 'flex',
            alignItems: 'center',
          }}
        >
          <Sym name="send" size={18} color={tk.onPrimary} />
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
      <Box key="rows" sx={{ display: 'flex', flexDirection: 'column', gap: '2px' }}>
        {rows}
      </Box>
      {thread}
      {edit}
    </Box>
  );
}
