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
      case 'home':     return <Home />;
      case 'events':   return <EventsPage />;
      case 'members':  return <MembersPage />;
      case 'finances': return app.can('finances', 'read') ? <FinancesPage /> : <Home />;
      case 'stats':    return <Stats />;
      case 'news':     return <NewsPage />;
      case 'polls':    return <PollsPage />;
      case 'team':     return <TeamPage />;
      default:         return <Home />;
    }
  })();

  return <Suspense fallback={<SpinnerBox />}>{page}</Suspense>;
}
