import Box from '@mui/material/Box';
import ButtonBase from '@mui/material/ButtonBase';
import { useApp } from '../store/AppContext';
import { buildTokens, NEUTRAL } from '../theme/tokens';
import { todayLocalDate } from '../utils/date';
import { Av, EmptyState, SectionTitle, Sym } from '../components/ui';
import { EventCard, NewsCard } from '../components/cards';

export function Home() {
  const app = useApp();
  const { state } = app;
  const t = buildTokens(state.primaryColor);
  const team = app.activeTeam()!;
  const today = todayLocalDate();

  const next = state.events.filter((e) => e.date >= today).slice(0, 3);
  const news = (state.news || []).slice(0, 3);
  const myPending = state.events.filter((e) => e.date >= today && e.myStatus === 'pending' && e.status !== 'cancelled').length;

  const quickStat = (label: string, val: React.ReactNode, icon: string, col: string, onClick: () => void) => (
    <ButtonBase key={label} onClick={onClick} sx={{ flex: 1, minWidth: '120px', textAlign: 'left', background: '#fff', border: `1px solid ${NEUTRAL.line}`, borderRadius: '16px', p: '14px', flexDirection: 'column', alignItems: 'stretch' }}>
      <Box sx={{ display: 'flex', alignItems: 'center', gap: '7px', color: col, width: '100%' }}>
        <Sym name={icon} size={18} color={col} />
        <Box component="span" sx={{ fontSize: '22px', fontWeight: 800, color: NEUTRAL.onSurface, flex: 1 }}>{val}</Box>
        <Sym name="chevron_right" size={18} color="#C0C2CA" />
      </Box>
      <Box sx={{ fontSize: '12px', color: NEUTRAL.secondary, mt: '4px' }}>{label}</Box>
    </ButtonBase>
  );

  return (
    <Box sx={{ maxWidth: '760px' }}>
      <Box sx={{ borderRadius: '22px', p: '22px', mb: '18px', color: '#fff', display: 'flex', alignItems: 'center', gap: '18px', position: 'relative', overflow: 'hidden', ...(team.photo ? { backgroundImage: `linear-gradient(90deg, rgba(10,12,20,.78), rgba(10,12,20,.35)), url(${team.photo})`, backgroundSize: 'cover', backgroundPosition: 'center' } : { background: `linear-gradient(120deg, ${t.primary}, ${t.onPrimaryContainer})` }) }}>
        <Av name={team.name} color="rgba(255,255,255,.18)" size={54} font={24} />
        <Box sx={{ flex: 1, minWidth: 0 }}>
          <Box sx={{ fontSize: '19px', fontWeight: 700, lineHeight: 1.2 }}>{team.name}</Box>
          <Box sx={{ fontSize: '13px', opacity: 0.85, mt: '4px' }}>Hallo {state.user!.name.split(' ')[0]}! {myPending ? myPending + ' Termin(e) brauchen deine Rückmeldung.' : 'Alles beantwortet – stark.'}</Box>
        </Box>
      </Box>

      <Box sx={{ display: 'flex', gap: '10px', flexWrap: 'wrap', mb: '20px' }}>
        {quickStat('Anstehende Termine', state.events.filter((e) => e.date >= today && e.status !== 'cancelled').length, 'event', t.primary, () => app.go('events'))}
        {quickStat('Offene Rückmeldungen', myPending, 'pending_actions', '#9A5B00', () => app.goEventsPending())}
        {quickStat('Mitglieder', team.memberCount, 'group', '#2E7D32', () => app.go('members'))}
      </Box>

      <Box sx={{ mb: '22px' }}>
        <SectionTitle right={<ButtonBase onClick={() => app.go('events')} sx={{ color: t.primary, fontWeight: 600, fontSize: '13px' }}>Alle ansehen</ButtonBase>}>Nächste Termine</SectionTitle>
        <Box sx={{ display: 'flex', flexDirection: 'column', gap: '10px' }}>
          {next.length ? next.map((e) => <EventCard key={e.id} e={e} />) : <EmptyState icon="event_available" text="Keine anstehenden Termine" />}
        </Box>
      </Box>

      <Box>
        <SectionTitle right={<ButtonBase onClick={() => app.go('news')} sx={{ color: t.primary, fontWeight: 600, fontSize: '13px' }}>Alle ansehen</ButtonBase>}>Neuigkeiten</SectionTitle>
        <Box sx={{ display: 'flex', flexDirection: 'column', gap: '10px' }}>
          {news.length ? news.map((n) => <NewsCard key={n.id} n={n} compact />) : <EmptyState icon="campaign" text="Noch keine News" />}
        </Box>
      </Box>
    </Box>
  );
}
