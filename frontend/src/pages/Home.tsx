import Box from '@mui/material/Box';
import ButtonBase from '@mui/material/ButtonBase';
import { useApp } from '@/context/AppContext';
import { buildTokens, NEUTRAL } from '@/styles/tokens';
import { todayLocalDate } from '@/utils/date';
import { Av, EmptyState, SectionTitle, Sym } from '@/components/ui';
import { EventCard, NewsCard } from '@/components/cards';
import { useEventsQuery } from '@/features/events';
import { useNewsQuery } from '@/features/news/hooks/useNewsQueries';
import { t as tr } from '@/i18n';

export function Home() {
  const app = useApp();
  const { state } = app;
  const t = buildTokens(state.primaryColor);
  const team = app.activeTeam()!;
  const today = todayLocalDate();
  const { data: events } = useEventsQuery(app.api, state.activeTeamId);
  const { data: newsItems } = useNewsQuery(app.api, state.activeTeamId);

  const next = (events ?? []).filter((e) => e.date >= today).slice(0, 3);
  const news = (newsItems || []).slice(0, 3);
  const myPending = (events ?? []).filter(
    (e) => e.date >= today && e.myStatus === 'pending' && e.status !== 'cancelled',
  ).length;

  // A role can have events/members/news set to 'none' -- cross-links and
  // stats for a module the caller can't read must not be shown, mirroring
  // RouteScreen's/AppShell's ROUTE_MODULE-driven gating (see urlState.ts).
  // Without this, tapping through would just bounce back to Home with a
  // spurious forbidden toast.
  const canSeeEvents = app.can('events', 'read');
  const canSeeMembers = app.can('members', 'read');
  const canSeeNews = app.can('news', 'read');

  const quickStat = (label: string, val: React.ReactNode, icon: string, col: string, onClick: () => void) => (
    <ButtonBase
      key={label}
      onClick={onClick}
      sx={{
        flex: 1,
        minWidth: '120px',
        textAlign: 'left',
        background: NEUTRAL.card,
        border: `1px solid ${NEUTRAL.line}`,
        borderRadius: '16px',
        p: '14px',
        flexDirection: 'column',
        alignItems: 'stretch',
      }}
    >
      <Box sx={{ display: 'flex', alignItems: 'center', gap: '7px', color: col, width: '100%' }}>
        <Sym name={icon} size={18} color={col} />
        <Box component="span" sx={{ fontSize: '22px', fontWeight: 800, color: NEUTRAL.onSurface, flex: 1 }}>
          {val}
        </Box>
        <Sym name="chevron_right" size={18} color={NEUTRAL.faint} />
      </Box>
      <Box sx={{ fontSize: '12px', color: NEUTRAL.secondary, mt: '4px' }}>{label}</Box>
    </ButtonBase>
  );

  return (
    <Box sx={{ maxWidth: '760px' }}>
      <Box
        sx={{
          borderRadius: '22px',
          p: '22px',
          mb: '18px',
          color: '#fff',
          display: 'flex',
          alignItems: 'center',
          gap: '18px',
          position: 'relative',
          overflow: 'hidden',
          ...(team.photo
            ? {
                backgroundImage: `linear-gradient(90deg, rgba(10,12,20,.78), rgba(10,12,20,.35)), url(${team.photo})`,
                backgroundSize: 'cover',
                backgroundPosition: 'center',
              }
            : { background: `linear-gradient(120deg, ${t.primary}, ${t.onPrimaryContainer})` }),
        }}
      >
        <Av name={team.name} color="rgba(255,255,255,.18)" size={54} font={24} />
        <Box sx={{ flex: 1, minWidth: 0 }}>
          <Box sx={{ fontSize: '19px', fontWeight: 700, lineHeight: 1.2 }}>{team.name}</Box>
          <Box sx={{ fontSize: '13px', opacity: 0.85, mt: '4px' }}>
            {tr('home.greeting', { name: state.user!.name.split(' ')[0] })}{' '}
            {myPending ? tr('home.pendingPrompt', { n: myPending, count: myPending }) : tr('home.allAnswered')}
          </Box>
        </Box>
      </Box>

      <Box sx={{ display: 'flex', gap: '10px', flexWrap: 'wrap', mb: '20px' }}>
        {canSeeEvents &&
          quickStat(
            tr('home.statUpcoming'),
            (events ?? []).filter((e) => e.date >= today && e.status !== 'cancelled').length,
            'event',
            t.primary,
            () => app.go('events'),
          )}
        {canSeeEvents &&
          quickStat(tr('home.statPending'), myPending, 'pending_actions', NEUTRAL.warn, () => app.goEventsPending())}
        {canSeeMembers &&
          quickStat(tr('home.statMembers'), team.memberCount, 'group', NEUTRAL.success, () => app.go('members'))}
      </Box>

      {canSeeEvents && (
        <Box sx={{ mb: '22px' }}>
          <SectionTitle
            right={
              <ButtonBase onClick={() => app.go('events')} sx={{ color: t.primary, fontWeight: 600, fontSize: '13px' }}>
                {tr('home.viewAll')}
              </ButtonBase>
            }
          >
            {tr('home.nextEvents')}
          </SectionTitle>
          <Box sx={{ display: 'flex', flexDirection: 'column', gap: '10px' }}>
            {next.length ? (
              next.map((e) => <EventCard key={e.id} e={e} />)
            ) : (
              <EmptyState icon="event_available" text={tr('home.emptyEvents')} />
            )}
          </Box>
        </Box>
      )}

      {canSeeNews && (
        <Box>
          <SectionTitle
            right={
              <ButtonBase onClick={() => app.go('news')} sx={{ color: t.primary, fontWeight: 600, fontSize: '13px' }}>
                {tr('home.viewAll')}
              </ButtonBase>
            }
          >
            {tr('home.news')}
          </SectionTitle>
          <Box sx={{ display: 'flex', flexDirection: 'column', gap: '10px' }}>
            {news.length ? (
              news.map((n) => <NewsCard key={n.id} n={n} compact primaryColor={state.primaryColor} />)
            ) : (
              <EmptyState icon="campaign" text={tr('home.emptyNews')} />
            )}
          </Box>
        </Box>
      )}
    </Box>
  );
}
