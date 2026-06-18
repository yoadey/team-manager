import { useApp } from '@/context/AppContext';
import { Home } from './Home';
import { Stats } from './Stats';
import { EventsPage } from '@/features/events';
import { MembersPage } from '@/features/members';
import { FinancesPage } from '@/features/finances';
import { NewsPage } from '@/features/news';
import { PollsPage } from '@/features/polls';
import { TeamPage } from '@/features/team';

export function RouteScreen() {
  const { state } = useApp();
  switch (state.route) {
    case 'home': return <Home />;
    case 'events': return <EventsPage />;
    case 'members': return <MembersPage />;
    case 'finances': return <FinancesPage />;
    case 'stats': return <Stats />;
    case 'news': return <NewsPage />;
    case 'polls': return <PollsPage />;
    case 'team': return <TeamPage />;
    default: return <Home />;
  }
}
