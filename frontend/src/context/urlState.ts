// =============================================================================
// URL <-> AppState mapping for shallow, dependency-free routing.
//
// The app keeps navigation in React state (state.route + sheets + filters).
// To make views bookmarkable / shareable and to make the browser Back button
// behave (close a detail sheet instead of leaving the whole feature), we mirror
// the bookmark-relevant slice of state into the URL:
//
//   /events?scope=past&view=calendar&pending=1   (list filters)
//   /events/<eventId>                            (event detail sheet)
//   /members/<membershipId>                      (member detail sheet)
//   /finances?tab=strafen                        (finance tab)
//
// Only defaults-omitted, human-meaningful state is encoded. Everything here is
// pure so it can be unit-tested without a DOM.
// =============================================================================

export type Route = 'home' | 'events' | 'members' | 'finances' | 'stats' | 'news' | 'polls' | 'team';

export const ALL_ROUTES: Route[] = ['home', 'events', 'members', 'finances', 'stats', 'news', 'polls', 'team'];

export function routeFromPath(path: string): Route {
  const seg = path.replace(/^\//, '').split('/')[0] as Route;
  return ALL_ROUTES.includes(seg) ? seg : 'home';
}

/** The subset of AppState that is reflected in the URL. */
export interface UrlState {
  route: Route;
  eventScope: 'upcoming' | 'past';
  eventsView: 'list' | 'calendar' | 'absences';
  eventsOnlyPending: boolean;
  finTab: 'umsaetze' | 'strafen' | 'beitraege';
  /** Active page-level detail sheet, if any (drives the path segment). */
  detail: { kind: 'event' | 'member'; id: string } | null;
}

/** Build the path+query string (e.g. "/events?scope=past") for the given state. */
export function buildPath(s: UrlState): string {
  let path = '/' + s.route;
  const params = new URLSearchParams();

  if (s.detail && (s.route === 'events' || s.route === 'members')) {
    path += '/' + encodeURIComponent(s.detail.id);
  } else if (s.route === 'events') {
    if (s.eventScope === 'past') params.set('scope', 'past');
    if (s.eventsView !== 'list') params.set('view', s.eventsView);
    if (s.eventsOnlyPending) params.set('pending', '1');
  } else if (s.route === 'finances') {
    if (s.finTab !== 'umsaetze') params.set('tab', s.finTab);
  }

  const q = params.toString();
  return q ? `${path}?${q}` : path;
}

export interface ParsedLocation {
  route: Route;
  /** Detail id from the second path segment (only for events/members routes). */
  detailId: string | null;
  eventScope: 'upcoming' | 'past';
  eventsView: 'list' | 'calendar' | 'absences';
  eventsOnlyPending: boolean;
  finTab: 'umsaetze' | 'strafen' | 'beitraege';
}

/** Parse a pathname + search string back into the URL-reflected state. */
export function parseLocation(pathname: string, search: string): ParsedLocation {
  const segs = pathname.replace(/^\//, '').split('/');
  const route = ALL_ROUTES.includes(segs[0] as Route) ? (segs[0] as Route) : 'home';
  const rawId = (route === 'events' || route === 'members') && segs[1] ? decodeURIComponent(segs[1]) : null;

  const p = new URLSearchParams(search);
  const view = p.get('view');
  const tab = p.get('tab');

  return {
    route,
    detailId: rawId,
    eventScope: p.get('scope') === 'past' ? 'past' : 'upcoming',
    eventsView: view === 'calendar' || view === 'absences' ? view : 'list',
    eventsOnlyPending: p.get('pending') === '1',
    finTab: tab === 'strafen' || tab === 'beitraege' ? tab : 'umsaetze',
  };
}

/** Current location as a path+query string, for comparing against buildPath. */
export function currentPath(): string {
  return window.location.pathname + window.location.search;
}
