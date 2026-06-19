import { lazy, Suspense } from 'react';
import { useApp } from '@/context/AppContext';
import { Home } from './Home';
import { EventsPage } from '@/features/events';
import { MembersPage } from '@/features/members';
import { SpinnerBox } from '@/components/ui';

const Stats = lazy(() => import('./Stats').then((m) => ({ default: m.Stats })));
const FinancesPage = lazy(() => import('@/features/finances').then((m) => ({ default: m.FinancesPage })));
const NewsPage = lazy(() => import('@/features/news').then((m) => ({ default: m.NewsPage })));
const PollsPage = lazy(() => import('@/features/polls').then((m) => ({ default: m.PollsPage })));
const TeamPage = lazy(() => import('@/features/team').then((m) => ({ default: m.TeamPage })));

export function RouteScreen() {
  const app = useApp();

  const page = (() => {
    switch (app.state.route) {
      case 'home':
        return <Home />;
      case 'events':
        return app.can('events', 'read') ? <EventsPage /> : <Home />;
      case 'members':
        return app.can('members', 'read') ? <MembersPage /> : <Home />;
      case 'finances':
        return app.can('finances', 'read') ? <FinancesPage /> : <Home />;
      case 'stats':
        return app.can('events', 'read') ? <Stats /> : <Home />;
      case 'news':
        return app.can('news', 'read') ? <NewsPage /> : <Home />;
      case 'polls':
        return app.can('polls', 'read') ? <PollsPage /> : <Home />;
      case 'team':
        return app.can('members', 'read') ? <TeamPage /> : <Home />;
      default:
        return <Home />;
    }
  })();

  return <Suspense fallback={<SpinnerBox />}>{page}</Suspense>;
}
