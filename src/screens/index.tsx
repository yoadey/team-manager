import { useApp } from '../store/AppContext';
import { Home } from './Home';
import { Events } from './Events';
import { Members } from './Members';
import { Finances } from './Finances';
import { Stats } from './Stats';
import { News } from './News';
import { Polls } from './Polls';
import { Team } from './Team';

export function RouteScreen() {
  const { state } = useApp();
  switch (state.route) {
    case 'home': return <Home />;
    case 'events': return <Events />;
    case 'members': return <Members />;
    case 'finances': return <Finances />;
    case 'stats': return <Stats />;
    case 'news': return <News />;
    case 'polls': return <Polls />;
    case 'team': return <Team />;
    default: return <Home />;
  }
}
