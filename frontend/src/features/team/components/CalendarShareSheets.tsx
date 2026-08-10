import { useState } from 'react';
import Box from '@mui/material/Box';
import ButtonBase from '@mui/material/ButtonBase';
import { buildTokens, NEUTRAL } from '@/styles/tokens';
import { EmptyState, Field, PrimaryButton, SkeletonList, Sym, TextInput } from '@/components/ui';
import type { SheetProps } from '@/sheets/types';
import {
  useCalendarSharesQuery,
  useSharedCalendarEventsQuery,
  useSharedCalendarSourcesQuery,
} from '../hooks/useCalendarShareQueries';
import { t, getIntlLocale } from '@/i18n';

const fmtDate = (iso: string) => new Intl.DateTimeFormat(getIntlLocale(), { day: 'numeric', month: 'long', year: 'numeric' }).format(new Date(iso));
const fmtEventDate = (dateOnly: string) =>
  new Intl.DateTimeFormat(getIntlLocale(), { day: 'numeric', month: 'short' }).format(new Date(dateOnly + 'T00:00:00'));

// A v4 UUID has 32 hex digits arranged 8-4-4-4-12 -- enough of a shape check
// to catch an obviously wrong paste (e.g. an invite code or a stray word)
// client-side, before the request round-trip; the server remains the source
// of truth for whether the id actually resolves to a team.
const UUID_RE = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i;

/** Settings-gated: manage which other teams may read this team's (redacted) calendar. */
export function CalendarSharesSheet({ app }: SheetProps) {
  const { state } = app;
  const tk = buildTokens(state.primaryColor);
  const team = app.activeTeam()!;
  const canManage = app.can('settings', 'write');
  const { data: shares, isLoading } = useCalendarSharesQuery(app.api, team.id);

  const [viewerTeamId, setViewerTeamId] = useState('');
  const [copied, setCopied] = useState(false);
  const [submitting, setSubmitting] = useState(false);

  const validId = UUID_RE.test(viewerTeamId.trim());

  const submit = async () => {
    if (!validId) return;
    setSubmitting(true);
    try {
      await app.grantCalendarShare(viewerTeamId.trim());
      setViewerTeamId('');
    } catch {
      // Ignored -- reportActionError already surfaced a toast.
    } finally {
      setSubmitting(false);
    }
  };

  const copyOwnId = async () => {
    try {
      await navigator.clipboard.writeText(team.id);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch {
      // Clipboard access denied -- the id is still visible to copy by hand.
    }
  };

  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: '16px' }}>
      <Box sx={{ fontSize: '13px', color: NEUTRAL.secondary, lineHeight: 1.5 }}>{t('team.calendarSharesDesc')}</Box>

      <Box
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
        <Box sx={{ flex: 1, minWidth: 0 }}>
          <Box sx={{ fontSize: '11px', color: NEUTRAL.secondary, mb: '2px' }}>{t('team.yourTeamId')}</Box>
          <Box sx={{ fontSize: '13px', fontFamily: 'monospace', overflow: 'hidden', textOverflow: 'ellipsis' }}>
            {team.id}
          </Box>
        </Box>
        <ButtonBase
          onClick={copyOwnId}
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
          {copied ? t('team.inviteCopied') : t('team.inviteCopy')}
        </ButtonBase>
      </Box>

      {canManage ? (
        <Box sx={{ display: 'flex', flexDirection: 'column', gap: '10px' }}>
          <Field label={t('team.grantCalendarShareLabel')} helperText={t('team.grantCalendarShareHint')}>
            <TextInput
              name="viewerTeamId"
              placeholder="00000000-0000-0000-0000-000000000000"
              value={viewerTeamId}
              onChange={(e) => setViewerTeamId(e.target.value)}
            />
          </Field>
          <PrimaryButton
            label={t('team.grantCalendarShareSubmit')}
            onClick={submit}
            disabled={!validId}
            busy={submitting}
          />
        </Box>
      ) : null}

      {isLoading ? (
        <SkeletonList rows={2} rowHeight={56} />
      ) : shares && shares.length > 0 ? (
        <Box sx={{ display: 'flex', flexDirection: 'column', gap: '8px' }}>
          {shares.map((s) => (
            <Box
              key={s.viewerTeamId}
              sx={{
                display: 'flex',
                alignItems: 'center',
                gap: '10px',
                border: `1px solid ${NEUTRAL.line}`,
                borderRadius: '13px',
                p: '12px 14px',
                background: NEUTRAL.card,
              }}
            >
              <Box sx={{ flex: 1, minWidth: 0 }}>
                <Box sx={{ fontSize: '14px', fontWeight: 600 }}>{s.viewerTeamName}</Box>
                <Box sx={{ fontSize: '12px', color: NEUTRAL.secondary, mt: '2px' }}>
                  {t('team.calendarShareGrantedOn', { date: fmtDate(s.createdAt) })}
                </Box>
              </Box>
              {canManage ? (
                <ButtonBase
                  onClick={() => app.revokeCalendarShare(s.viewerTeamId, s.viewerTeamName)}
                  aria-label={t('team.revokeCalendarShareLabel')}
                  sx={{
                    width: '30px',
                    height: '30px',
                    borderRadius: '50%',
                    background: NEUTRAL.errorBg,
                    color: NEUTRAL.error,
                    cursor: 'pointer',
                  }}
                >
                  <Sym name="delete" size={16} color={NEUTRAL.error} />
                </ButtonBase>
              ) : null}
            </Box>
          ))}
        </Box>
      ) : (
        <EmptyState icon="calendar_month" text={t('team.noCalendarShares')} />
      )}
    </Box>
  );
}

/** Membership-gated: browse the redacted schedule of a team that shared its calendar with this one. */
export function SharedCalendarsSheet({ app }: SheetProps) {
  const team = app.activeTeam()!;
  const { data: sources, isLoading } = useSharedCalendarSourcesQuery(app.api, team.id);
  const [expandedOwnerId, setExpandedOwnerId] = useState<string | null>(null);

  if (isLoading) return <SkeletonList rows={2} rowHeight={56} />;
  if (!sources || sources.length === 0) return <EmptyState icon="calendar_month" text={t('team.noSharedCalendars')} />;

  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: '10px' }}>
      {sources.map((src) => (
        <SharedCalendarSourceRow
          key={src.ownerTeamId}
          app={app}
          teamId={team.id}
          ownerTeamId={src.ownerTeamId}
          ownerTeamName={src.ownerTeamName}
          expanded={expandedOwnerId === src.ownerTeamId}
          onToggle={() => setExpandedOwnerId((cur) => (cur === src.ownerTeamId ? null : src.ownerTeamId))}
        />
      ))}
    </Box>
  );
}

function SharedCalendarSourceRow({
  app,
  teamId,
  ownerTeamId,
  ownerTeamName,
  expanded,
  onToggle,
}: {
  app: SheetProps['app'];
  teamId: string;
  ownerTeamId: string;
  ownerTeamName: string;
  expanded: boolean;
  onToggle: () => void;
}) {
  const { data: events, isLoading } = useSharedCalendarEventsQuery(app.api, teamId, expanded ? ownerTeamId : null);

  return (
    <Box sx={{ border: `1px solid ${NEUTRAL.line}`, borderRadius: '13px', background: NEUTRAL.card, overflow: 'hidden' }}>
      <ButtonBase
        onClick={onToggle}
        sx={{ display: 'flex', alignItems: 'center', gap: '10px', width: '100%', p: '12px 14px', textAlign: 'left' }}
      >
        <Box sx={{ flex: 1, fontSize: '14px', fontWeight: 600 }}>{ownerTeamName}</Box>
        <Sym name={expanded ? 'expand_less' : 'expand_more'} size={20} color={NEUTRAL.faint} />
      </ButtonBase>
      {expanded ? (
        <Box sx={{ borderTop: `1px solid ${NEUTRAL.line}`, p: '10px 14px' }}>
          {isLoading ? (
            <SkeletonList rows={2} rowHeight={40} />
          ) : events && events.length > 0 ? (
            <Box sx={{ display: 'flex', flexDirection: 'column', gap: '8px' }}>
              {events.map((e) => (
                <Box key={e.id} sx={{ display: 'flex', gap: '10px', fontSize: '13px' }}>
                  <Box sx={{ color: NEUTRAL.secondary, flex: '0 0 auto' }}>
                    {e.multiDayEndDate ? `${fmtEventDate(e.date)} – ${fmtEventDate(e.multiDayEndDate)}` : fmtEventDate(e.date)}
                  </Box>
                  <Box sx={{ flex: 1, fontWeight: 600 }}>{e.title}</Box>
                  {e.startTime ? <Box sx={{ color: NEUTRAL.secondary }}>{e.startTime}</Box> : null}
                  {e.location ? <Box sx={{ color: NEUTRAL.secondary }}>{e.location}</Box> : null}
                </Box>
              ))}
            </Box>
          ) : (
            <Box sx={{ fontSize: '13px', color: NEUTRAL.secondary, p: '8px 0' }}>{t('team.noSharedCalendarEvents')}</Box>
          )}
        </Box>
      ) : null}
    </Box>
  );
}
