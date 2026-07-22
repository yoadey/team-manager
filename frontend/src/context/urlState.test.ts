import { describe, it, expect } from 'vitest';
import { buildPath, parseLocation, parsePendingInvite, parseVerifyEmailToken, routeFromPath, type UrlState } from './urlState';

const base: UrlState = {
  route: 'home',
  eventScope: 'upcoming',
  eventsView: 'list',
  eventsOnlyPending: false,
  finTab: 'umsaetze',
  detail: null,
};

describe('buildPath', () => {
  it('encodes a bare route without query for defaults', () => {
    expect(buildPath({ ...base, route: 'events' })).toBe('/events');
    expect(buildPath({ ...base, route: 'home' })).toBe('/home');
  });

  it('encodes event list filters as query params, omitting defaults', () => {
    expect(buildPath({ ...base, route: 'events', eventScope: 'past' })).toBe('/events?scope=past');
    expect(buildPath({ ...base, route: 'events', eventsView: 'calendar', eventsOnlyPending: true })).toBe(
      '/events?view=calendar&pending=1',
    );
  });

  it('encodes the finance tab, omitting the default', () => {
    expect(buildPath({ ...base, route: 'finances' })).toBe('/finances');
    expect(buildPath({ ...base, route: 'finances', finTab: 'strafen' })).toBe('/finances?tab=strafen');
  });

  it('encodes a detail sheet as a path segment and drops list filters', () => {
    expect(buildPath({ ...base, route: 'events', eventScope: 'past', detail: { kind: 'event', id: 'ev1' } })).toBe(
      '/events/ev1',
    );
    expect(buildPath({ ...base, route: 'members', detail: { kind: 'member', id: 'm9' } })).toBe('/members/m9');
  });
});

describe('parseLocation', () => {
  it('parses route and detail id', () => {
    expect(parseLocation('/events', '')).toMatchObject({ route: 'events', detailId: null });
    expect(parseLocation('/events/ev1', '')).toMatchObject({ route: 'events', detailId: 'ev1' });
    expect(parseLocation('/members/m9', '')).toMatchObject({ route: 'members', detailId: 'm9' });
  });

  it('falls back to home for unknown routes', () => {
    expect(parseLocation('/nope', '').route).toBe('home');
  });

  it('parses filters with safe defaults', () => {
    expect(parseLocation('/events', '?scope=past&view=calendar&pending=1')).toMatchObject({
      eventScope: 'past',
      eventsView: 'calendar',
      eventsOnlyPending: true,
    });
    expect(parseLocation('/events', '?view=bogus')).toMatchObject({ eventsView: 'list' });
    expect(parseLocation('/finances', '?tab=beitraege')).toMatchObject({ finTab: 'beitraege' });
  });

  it('is the inverse of buildPath for representative states', () => {
    const cases: UrlState[] = [
      { ...base, route: 'events', eventScope: 'past', eventsView: 'calendar', eventsOnlyPending: true },
      { ...base, route: 'finances', finTab: 'beitraege' },
      { ...base, route: 'stats' },
    ];
    for (const s of cases) {
      const path = buildPath(s);
      const [pathname, search] = path.split('?');
      const parsed = parseLocation(pathname ?? '', search ? '?' + search : '');
      expect(parsed.route).toBe(s.route);
      expect(parsed.eventScope).toBe(s.eventScope);
      expect(parsed.eventsView).toBe(s.eventsView);
      expect(parsed.eventsOnlyPending).toBe(s.eventsOnlyPending);
      expect(parsed.finTab).toBe(s.finTab);
    }
  });
});

describe('parsePendingInvite', () => {
  it('parses a well-formed /join/<teamId>/<code> path', () => {
    expect(parsePendingInvite('/join/team-1/abc123')).toEqual({ teamId: 'team-1', code: 'abc123' });
  });

  it('decodes URI-encoded segments', () => {
    expect(parsePendingInvite('/join/team%201/ab%2Fc')).toEqual({ teamId: 'team 1', code: 'ab/c' });
  });

  it('returns null for paths that are not /join/... at all', () => {
    expect(parsePendingInvite('/home')).toBeNull();
    expect(parsePendingInvite('/')).toBeNull();
  });

  it('returns null for a malformed join path (missing segment or extra segment)', () => {
    expect(parsePendingInvite('/join/team-1')).toBeNull();
    expect(parsePendingInvite('/join/team-1/code/extra')).toBeNull();
    expect(parsePendingInvite('/join//code')).toBeNull();
    expect(parsePendingInvite('/join/team-1/')).toBeNull();
  });
});

describe('parseVerifyEmailToken', () => {
  it('parses a well-formed /verify-email/<token> path', () => {
    expect(parseVerifyEmailToken('/verify-email/abc123')).toBe('abc123');
  });

  it('decodes a URI-encoded token', () => {
    expect(parseVerifyEmailToken('/verify-email/ab%2Fc')).toBe('ab/c');
  });

  it('returns null for paths that are not /verify-email/... at all', () => {
    expect(parseVerifyEmailToken('/home')).toBeNull();
    expect(parseVerifyEmailToken('/')).toBeNull();
  });

  it('returns null for a malformed path (missing segment or extra segment)', () => {
    expect(parseVerifyEmailToken('/verify-email')).toBeNull();
    expect(parseVerifyEmailToken('/verify-email/')).toBeNull();
    expect(parseVerifyEmailToken('/verify-email/tok/extra')).toBeNull();
  });
});

describe('routeFromPath', () => {
  it('extracts the first segment as a known route', () => {
    expect(routeFromPath('/events/ev1')).toBe('events');
    expect(routeFromPath('/')).toBe('home');
    expect(routeFromPath('/unknown')).toBe('home');
  });
});
