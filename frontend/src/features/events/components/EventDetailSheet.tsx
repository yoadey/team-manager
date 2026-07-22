import { useEffect, useState } from 'react';
import type { ReactNode } from 'react';
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

type AppApi = SheetProps['app'];
type Tokens = ReturnType<typeof buildTokens>;

function CancelledBanner({ event, canEdit, onReactivate }: { event: TeamEvent; canEdit: boolean; onReactivate: () => void }) {
  if (event.status !== 'cancelled') return null;
  return (
    <Box
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
      <Box component="span" sx={{ flex: 1, fontSize: '13px', fontWeight: 600 }}>
        {t('events.cancelledBanner')}
      </Box>
      {canEdit ? (
        <ButtonBase
          onClick={onReactivate}
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
  );
}

function EventInfoCard({ event }: { event: TeamEvent }) {
  return (
    <Box
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
      {event.meetTime ? (
        <Box sx={{ display: 'flex', alignItems: 'center', gap: '12px', p: '13px 14px', background: NEUTRAL.warnBg }}>
          <Sym name="login" size={20} color={NEUTRAL.warn} />
          <Box component="span" sx={{ flex: 1 }}>
            <b>{t('events.meetTime', { time: hhmm(event.meetTime) })}</b>
          </Box>
          {event.meetTimeMandatory ? (
            <Chip label={t('events.mandatory')} color={NEUTRAL.warn} bg={NEUTRAL.warnBg} />
          ) : null}
        </Box>
      ) : null}
      <Box sx={{ display: 'flex', alignItems: 'center', gap: '12px', p: '13px 14px', background: NEUTRAL.card }}>
        <Sym name="schedule" size={20} color={NEUTRAL.secondary} />
        <Box component="span" sx={{ flex: 1 }}>
          {t('events.startEnd')} <b>{hhmm(event.startTime) + '–' + hhmm(event.endTime)}</b>
        </Box>
      </Box>
      {event.location ? (
        <Box sx={{ display: 'flex', alignItems: 'center', gap: '12px', p: '13px 14px', background: NEUTRAL.card }}>
          <Sym name="place" size={20} color={NEUTRAL.secondary} />
          <Box component="span" sx={{ flex: 1 }}>
            {event.location}
          </Box>
        </Box>
      ) : null}
    </Box>
  );
}

interface RsvpButtonProps {
  label: string;
  icon: string;
  status: AttendanceStatus;
  active: boolean;
  activeColor: string;
  activeBg: string;
  passiveBg: string;
  passiveColor: string;
  onClick: () => void;
}

function RsvpButton({ label, icon, active, activeColor, activeBg, passiveBg, passiveColor, onClick }: RsvpButtonProps) {
  return (
    <ButtonBase
      onClick={onClick}
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
        background: active ? activeBg : passiveBg,
        color: active ? activeColor : passiveColor,
      }}
    >
      <Sym name={icon} size={19} color={active ? activeColor : passiveColor} />
      {label}
    </ButtonBase>
  );
}

interface MyResponseSectionProps {
  app: AppApi;
  event: TeamEvent;
  tk: Tokens;
  me: string;
}

function MyResponseSection({ app, event, tk, me }: MyResponseSectionProps) {
  const today = todayLocalDate();
  const isPast = event.date < today;
  const cancelled = event.status === 'cancelled';
  if (isPast || cancelled) return null;

  const myStatus = event.myStatus;
  if (myStatus === 'not_nominated') {
    return (
      <Box
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
  }

  const myComment = event.myReason || '';
  const autoMessage = event.myAuto
    ? myStatus === 'no'
      ? t('events.autoAbsent')
      : event.responseMode === 'opt_out'
        ? t('events.autoOptOut')
        : ''
    : null;

  return (
    <Box>
      {autoMessage ? (
        <Box
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
          {autoMessage}
        </Box>
      ) : null}
      <Box sx={{ display: 'flex', gap: '8px', mb: '10px' }}>
        <RsvpButton
          label={t('events.rsvpYes')}
          icon="check_circle"
          status="yes"
          active={myStatus === 'yes'}
          activeColor="#fff"
          activeBg={NEUTRAL.success}
          passiveBg={NEUTRAL.successBg}
          passiveColor={NEUTRAL.success}
          onClick={() => app.setMyStatus(event.id, 'yes', event.myReason)}
        />
        <RsvpButton
          label={t('events.rsvpMaybe')}
          icon="help"
          status="maybe"
          active={myStatus === 'maybe'}
          activeColor="#fff"
          activeBg={NEUTRAL.warn}
          passiveBg={NEUTRAL.warnBg}
          passiveColor={NEUTRAL.warn}
          onClick={() => app.setMyStatus(event.id, 'maybe', event.myReason)}
        />
        <RsvpButton
          label={t('events.rsvpNo')}
          icon="cancel"
          status="no"
          active={myStatus === 'no'}
          activeColor="#fff"
          activeBg={NEUTRAL.error}
          passiveBg={NEUTRAL.errorBg}
          passiveColor={NEUTRAL.error}
          onClick={() => app.setMyStatus(event.id, 'no', event.myReason)}
        />
      </Box>
      <ButtonBase
        onClick={() => app.openComment(event, { userId: me, name: t('events.meLabel'), status: myStatus, reason: myComment })}
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
        <Sym name="chat_bubble" size={18} color={myComment ? tk.primary : NEUTRAL.faint} />
        <Box
          component="span"
          sx={{
            flex: 1,
            fontSize: '13px',
            color: myComment ? NEUTRAL.onSurfaceVariant : NEUTRAL.faint,
            whiteSpace: 'nowrap',
            overflow: 'hidden',
            textOverflow: 'ellipsis',
          }}
        >
          {myComment || t('events.commentPlaceholder')}
        </Box>
        <Sym name={myComment ? 'edit' : 'add'} size={18} color={NEUTRAL.faint} />
      </ButtonBase>
    </Box>
  );
}

function AttendanceSummary({ event }: { event: TeamEvent }) {
  const total = event.summary.total || 1;
  return (
    <>
      <SectionTitle
        right={
          <Box sx={{ display: 'flex', gap: '9px', fontSize: '12px', fontWeight: 700 }}>
            <Box component="span" sx={{ color: NEUTRAL.success }}>
              {event.summary.yes + ' ' + t('events.summaryYes', { n: '' }).replace('{n} ', '')}
            </Box>
            <Box component="span" sx={{ color: NEUTRAL.warn }}>
              {event.summary.maybe + ' ' + t('events.summaryMaybe', { n: '' }).replace('{n} ', '')}
            </Box>
            <Box component="span" sx={{ color: NEUTRAL.error }}>
              {event.summary.no + ' ' + t('events.summaryNo', { n: '' }).replace('{n} ', '')}
            </Box>
            <Box component="span" sx={{ color: NEUTRAL.secondary }}>
              {event.summary.pending + ' ' + t('events.summaryPending', { n: '' }).replace('{n} ', '')}
            </Box>
          </Box>
        }
      >
        {t('events.participants')}
      </SectionTitle>
      <Box
        sx={{
          height: '8px',
          borderRadius: '6px',
          overflow: 'hidden',
          display: 'flex',
          background: NEUTRAL.line2,
          m: '2px 0 12px',
        }}
      >
        <Box sx={{ width: (event.summary.yes / total) * 100 + '%', background: NEUTRAL.success }} />
        <Box sx={{ width: (event.summary.maybe / total) * 100 + '%', background: '#E8910C' }} />
        <Box sx={{ width: (event.summary.no / total) * 100 + '%', background: NEUTRAL.error }} />
      </Box>
    </>
  );
}

function AttendanceStatusButton({
  row,
  status,
  icon,
  color,
  onClick,
}: {
  row: AttendanceRow;
  status: AttendanceStatus;
  icon: string;
  color: string;
  onClick: () => void;
}) {
  const selected = row.status === status;
  return (
    <ButtonBase
      title={statusMeta(status).label}
      aria-label={statusMeta(status).label}
      aria-pressed={selected}
      onClick={onClick}
      sx={{
        width: '30px',
        height: '30px',
        borderRadius: '8px',
        border: 'none',
        cursor: 'pointer',
        background: selected ? color : '#F1F2F6',
        color: selected ? '#fff' : NEUTRAL.faint,
        fontFamily: "'Material Symbols Outlined'",
        fontSize: '17px',
        flex: '0 0 auto',
      }}
    >
      {icon}
    </ButtonBase>
  );
}

function attendanceRowControls(params: {
  row: AttendanceRow;
  event: TeamEvent;
  app: AppApi;
  canEdit: boolean;
  isPast: boolean;
  mine: boolean;
  tk: Tokens;
}): ReactNode[] {
  const { row, event, app, canEdit, isPast, mine, tk } = params;
  const notNominated = row.status === 'not_nominated';
  const editable = (canEdit || mine) && !isPast;

  if (notNominated) {
    return [
      <Chip key="c" label={t('events.notNominatedLabel')} color={NEUTRAL.secondary} bg={NEUTRAL.line2} icon="block" />,
      canEdit && !isPast ? (
        <IconBtn
          key="nom"
          icon="person_add"
          onClick={() => app.toggleNomination(event.id, row.userId, false)}
          color={tk.primary}
          bg="#E7F0FF"
          title={t('events.nominate')}
        />
      ) : null,
      canEdit || mine ? (
        <IconBtn
          key="cm"
          icon="chat_bubble"
          onClick={() => app.openComment(event, row)}
          color={NEUTRAL.secondary}
          bg={NEUTRAL.sidebar}
          title={t('events.comment')}
        />
      ) : null,
    ];
  }

  if (editable) {
    return [
      <AttendanceStatusButton
        key="yes"
        row={row}
        status="yes"
        icon="check"
        color={NEUTRAL.success}
        onClick={() => app.setStatusFor(event, row, 'yes')}
      />,
      <AttendanceStatusButton
        key="maybe"
        row={row}
        status="maybe"
        icon="help"
        color={NEUTRAL.warn}
        onClick={() => app.setStatusFor(event, row, 'maybe')}
      />,
      <AttendanceStatusButton
        key="no"
        row={row}
        status="no"
        icon="close"
        color={NEUTRAL.error}
        onClick={() => app.setStatusFor(event, row, 'no')}
      />,
      <IconBtn
        key="cm"
        icon="chat_bubble"
        onClick={() => app.openComment(event, row)}
        color={NEUTRAL.secondary}
        bg={NEUTRAL.sidebar}
        title={t('events.comment')}
      />,
      canEdit && !isPast ? (
        <IconBtn
          key="rm"
          icon="person_remove"
          onClick={() => app.toggleNomination(event.id, row.userId, true)}
          color={NEUTRAL.faint}
          bg={NEUTRAL.sidebar}
          title={t('events.deNominate')}
        />
      ) : null,
    ];
  }

  const rsm = statusMeta(row.status);
  return [<Chip key="c" label={rsm.label} color={rsm.color} bg={rsm.bg} icon={rsm.icon} />];
}

function AttendanceRowItem({
  row,
  event,
  app,
  canEdit,
  isPast,
  me,
  canSeeComment,
  tk,
}: {
  row: AttendanceRow;
  event: TeamEvent;
  app: AppApi;
  canEdit: boolean;
  isPast: boolean;
  me: string;
  canSeeComment: boolean;
  tk: Tokens;
}) {
  const mine = row.userId === me;
  const notNominated = row.status === 'not_nominated';
  const controls = attendanceRowControls({ row, event, app, canEdit, isPast, mine, tk });

  return (
    <Box
      sx={{
        display: 'flex',
        alignItems: 'center',
        gap: '10px',
        p: '8px',
        borderRadius: '12px',
        background: NEUTRAL.card,
        opacity: notNominated ? 0.72 : 1,
      }}
    >
      <Av name={row.name} photo={row.photo} color={row.avatarColor} size={34} font={12} />
      <Box sx={{ flex: 1, minWidth: 0 }}>
        <Box sx={{ fontSize: '14px', fontWeight: 500, whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis' }}>
          {row.name + (mine ? ' · ' + t('events.meLabel') : '')}
        </Box>
        {canSeeComment && row.reason ? (
          <Box sx={{ fontSize: '11px', color: NEUTRAL.warn, whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis' }}>
            {row.reason}
          </Box>
        ) : (
          <Box sx={{ fontSize: '11px', color: NEUTRAL.faint }}>
            {row.group + (row.absent ? ' · ' + t('events.absent') : '')}
          </Box>
        )}
      </Box>
      <Box sx={{ display: 'flex', alignItems: 'center', gap: '5px', flex: '0 0 auto' }}>{controls}</Box>
    </Box>
  );
}

function EventEditActions({
  event,
  canEdit,
  onEdit,
  onCancel,
  onDelete,
}: {
  event: TeamEvent;
  canEdit: boolean;
  onEdit: () => void;
  onCancel: () => void;
  onDelete: () => void;
}) {
  if (!canEdit) return null;
  const cancelled = event.status === 'cancelled';
  return (
    <Box sx={{ display: 'flex', gap: '10px', mt: '18px', flexWrap: 'wrap' }}>
      <ButtonBase
        onClick={onEdit}
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
          onClick={onCancel}
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
        onClick={onDelete}
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
  );
}

function CommentThread({
  comments,
  me,
  canEdit,
  tk,
  onPost,
  onRemove,
}: {
  comments: EventComment[];
  me: string;
  canEdit: boolean;
  tk: Tokens;
  onPost: (text: string) => Promise<boolean>;
  onRemove: (commentId: string) => void;
}) {
  const [newComment, setNewComment] = useState('');

  const submit = () => {
    const txt = newComment;
    setNewComment('');
    onPost(txt).then((ok) => {
      if (!ok) setNewComment(txt);
    });
  };

  return (
    <Box sx={{ mt: '22px' }}>
      <SectionTitle>{t('events.comments') + (comments.length ? ' (' + comments.length + ')' : '')}</SectionTitle>
      <Box sx={{ display: 'flex', flexDirection: 'column', gap: '10px', mb: '12px' }}>
        {comments.length ? (
          comments.map((c) => (
            <Box key={c.id} sx={{ display: 'flex', gap: '10px', alignItems: 'flex-start' }}>
              <Av name={c.name} photo={c.photo} color={c.color} size={32} font={12} />
              <Box sx={{ flex: 1, minWidth: 0, background: NEUTRAL.sidebar, borderRadius: '12px', p: '9px 12px' }}>
                <Box sx={{ display: 'flex', alignItems: 'center', gap: '7px', mb: '2px' }}>
                  <Box component="span" sx={{ fontSize: '12px', fontWeight: 700 }}>
                    {c.name}
                  </Box>
                  <Box component="span" sx={{ fontSize: '11px', color: NEUTRAL.faint }}>
                    {fmtDateTime(c.createdAt)}
                  </Box>
                </Box>
                <Box sx={{ fontSize: '13px', color: NEUTRAL.onSurfaceVariant, lineHeight: 1.45, wordBreak: 'break-word' }}>
                  {c.text}
                </Box>
              </Box>
              {c.userId === me || canEdit ? (
                <IconBtn
                  icon="delete"
                  onClick={() => onRemove(c.id)}
                  color={NEUTRAL.error}
                  bg={NEUTRAL.errorBg}
                  title={t('events.delete')}
                />
              ) : null}
            </Box>
          ))
        ) : (
          <Box sx={{ fontSize: '13px', color: NEUTRAL.faint, p: '4px 2px' }}>{t('events.noComments')}</Box>
        )}
      </Box>
      <Box sx={{ display: 'flex', gap: '8px' }}>
        <input
          name="newEventComment"
          value={newComment}
          onChange={(ev) => setNewComment(ev.target.value)}
          onKeyDown={(ev) => {
            if (ev.key === 'Enter') submit();
          }}
          placeholder={t('events.commentWrite')}
          maxLength={10000}
          style={{ ...inputSx, flex: 1 }}
        />
        <ButtonBase
          onClick={submit}
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
}

export function EventDetailSheet({ app, sheet }: SheetProps) {
  const { state } = app;
  const tk = buildTokens(state.primaryColor);
  const eventId = sheet.eventId ?? null;
  const detail = useEventDetailQuery(app.api, state.activeTeamId, eventId);

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
  const canEdit = app.can('events', 'write');
  const me = state.user!.id;
  const canSeeCommentFn = app.canSeeComment;
  const rows = detail.data?.rows || [];
  const comments = detail.data?.comments || [];

  return (
    <Box>
      <Box sx={{ display: 'flex', alignItems: 'center', gap: '10px', flexWrap: 'wrap', mb: '10px' }}>
        <Chip label={tm.label} color={tm.color} bg={tm.bg} icon={tm.icon} fs={12} />
        {e.recurring ? (
          <Chip label={t('events.weekly')} color={NEUTRAL.secondary} bg={NEUTRAL.line2} icon="repeat" fs={12} />
        ) : null}
      </Box>
      <Box sx={{ fontSize: '13px', color: NEUTRAL.secondary, fontWeight: 500, m: '0 2px 12px' }}>
        {fmtDateLong(e.date)}
      </Box>
      <CancelledBanner event={e} canEdit={canEdit} onReactivate={() => app.askEventAction('reactivate', e)} />
      <EventInfoCard event={e} />
      {e.note ? (
        <Box sx={{ display: 'flex', gap: '10px', p: '12px 14px', background: NEUTRAL.sidebar, borderRadius: '14px', mb: '14px' }}>
          <Sym name="sticky_note_2" size={18} color={NEUTRAL.secondary} />
          <Box component="span" sx={{ fontSize: '13px', color: NEUTRAL.onSurfaceVariant, lineHeight: 1.5 }}>
            {e.note}
          </Box>
        </Box>
      ) : null}
      {e.result ? (
        <Box sx={{ display: 'flex', gap: '10px', p: '12px 14px', background: NEUTRAL.successBg, borderRadius: '14px', mb: '14px' }}>
          <Sym name="emoji_events" size={18} color={NEUTRAL.success} />
          <Box component="span" sx={{ fontSize: '13px', color: NEUTRAL.success, fontWeight: 600, lineHeight: 1.5 }}>
            {t('events.result', { val: e.result })}
          </Box>
        </Box>
      ) : null}
      <MyResponseSection app={app} event={e} tk={tk} me={me} />
      <AttendanceSummary event={e} />
      <Box sx={{ display: 'flex', flexDirection: 'column', gap: '2px' }}>
        {rows.map((r: AttendanceRow) => (
          <AttendanceRowItem
            key={r.userId}
            row={r}
            event={e}
            app={app}
            canEdit={canEdit}
            isPast={isPast}
            me={me}
            canSeeComment={canSeeCommentFn(r)}
            tk={tk}
          />
        ))}
      </Box>
      <CommentThread
        comments={comments}
        me={me}
        canEdit={canEdit}
        tk={tk}
        onPost={(text) => app.postEventComment(e.id, text)}
        onRemove={(commentId) => app.removeEventComment(e.id, commentId)}
      />
      <EventEditActions
        event={e}
        canEdit={canEdit}
        onEdit={() => app.openEventForm(e)}
        onCancel={() => app.askEventAction('cancel', e)}
        onDelete={() => app.askEventAction('delete', e)}
      />
    </Box>
  );
}
