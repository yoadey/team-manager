import { lazy, Suspense } from 'react';
import { useApp } from '@/context/AppContext';
import { ROUTE_MODULE } from '@/context/urlState';
import { Home } from './Home';
import { EventsPage } from '@/features/events';
import { MembersPage } from '@/features/members';
import { SpinnerBox } from '@/components/ui';
import { ErrorBoundary } from '@/components/ErrorBoundary';
import { captureError } from '@/monitoring';

// EventsPage and MembersPage cannot be code-split here because useFeatureActions
// and sheets/index already import the feature modules statically. True lazy
// loading requires moving feature-action hooks into per-feature modules.
const Stats = lazy(() => import('./Stats').then((m) => ({ default: m.Stats })));
const FinancesPage = lazy(() => import('@/features/finances').then((m) => ({ default: m.FinancesPage })));
const NewsPage = lazy(() => import('@/features/news').then((m) => ({ default: m.NewsPage })));
const PollsPage = lazy(() => import('@/features/polls').then((m) => ({ default: m.PollsPage })));
const TeamPage = lazy(() => import('@/features/team').then((m) => ({ default: m.TeamPage })));

export function RouteScreen() {
  const app = useApp();

  const module = ROUTE_MODULE[app.state.route];

  const page = (() => {
    if (module && !app.can(module, 'read')) return <Home />;
    switch (app.state.route) {
      case 'events':
        return <EventsPage />;
      case 'members':
        return <MembersPage />;
      case 'finances':
        return <FinancesPage />;
      case 'stats':
        return <Stats />;
      case 'news':
        return <NewsPage />;
      case 'polls':
        return <PollsPage />;
      case 'team':
        return <TeamPage />;
      case 'home':
      default:
        return <Home />;
    }
  })();

  // A per-route boundary keeps a crash in one feature page from taking down the
  // whole app (navigation chrome / shell stay alive). Keying on the route resets
  // the boundary when the user navigates away, so a previously failed page can be
  // re-entered. Errors still propagate to Sentry via captureError; if the route
  // boundary itself somehow fails, the app-level boundary in main.tsx catches it.
  return (
    <ErrorBoundary key={app.state.route} onError={captureError}>
      <Suspense fallback={<SpinnerBox />}>{page}</Suspense>
    </ErrorBoundary>
  );
}
