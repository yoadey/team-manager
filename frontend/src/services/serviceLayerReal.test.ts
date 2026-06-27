// Unit tests for the real (HTTP) service layer. The openapi-fetch client and the
// DTO mappers are mocked so these tests isolate serviceLayerReal's own
// responsibilities: the correct endpoint/method, forwarding the right path
// params / query / body, threading responses through the mappers, and — via the
// shared check() helper — translating HTTP error codes into typed error classes.
//
// This is the code path that actually ships (api resolves to realApi whenever
// VITE_API_BASE_URL is set), so it is exercised here directly rather than only
// through docker-compose integration tests.
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { AuthError, NetworkError, ValidationError } from '@/utils/errors';

// ── Mock the HTTP client ─────────────────────────────────────────────────────
vi.mock('@/api/client', () => ({
  apiClient: {
    GET: vi.fn(),
    POST: vi.fn(),
    PUT: vi.fn(),
    PATCH: vi.fn(),
    DELETE: vi.fn(),
  },
}));

// ── Mock the mappers with tagged identity stubs ──────────────────────────────
// Each mapper becomes a function that wraps its input so tests can assert the
// service layer ran the response through the right mapper without depending on
// the mappers' internal shape (those are covered by mappers.test.ts).
vi.mock('@/api/map', () => {
  const tag = (name: string) => vi.fn((input: unknown) => ({ __mapped: name, input }));
  return {
    mapUser: tag('user'),
    mapProvider: tag('provider'),
    mapRole: tag('role'),
    mapTeam: tag('team'),
    mapTeamForUser: tag('teamForUser'),
    mapInvite: tag('invite'),
    mapMember: tag('member'),
    mapTeamEvent: tag('teamEvent'),
    mapAttendanceRow: tag('attendanceRow'),
    mapEventComment: tag('eventComment'),
    mapAbsence: tag('absence'),
    mapNewsItem: tag('newsItem'),
    mapPoll: tag('poll'),
    mapNotificationsResult: tag('notifications'),
    mapFinanceOverview: tag('finance'),
    mapTransaction: tag('transaction'),
    mapPenalty: tag('penalty'),
    mapPenaltyAssignment: tag('penaltyAssignment'),
    mapContribution: tag('contribution'),
    mapStatsOverview: tag('statsOverview'),
  };
});

import { apiClient } from '@/api/client';
import { realApi } from './serviceLayerReal';

type Verb = 'GET' | 'POST' | 'PUT' | 'PATCH' | 'DELETE';
const client = apiClient as unknown as Record<Verb, ReturnType<typeof vi.fn>>;

/** A successful openapi-fetch result. */
function ok<T>(data: T, status = 200) {
  return { data, error: undefined, response: { ok: true, status } as Response };
}
/** A failed openapi-fetch result (no data, error body, !ok). */
function fail(status: number, body?: { detail?: string; title?: string }) {
  return { data: undefined, error: body ?? {}, response: { ok: false, status } as Response };
}

beforeEach(() => {
  for (const verb of ['GET', 'POST', 'PUT', 'PATCH', 'DELETE'] as Verb[]) {
    client[verb].mockReset();
  }
});

// ─── check(): HTTP status → typed error mapping ───────────────────────────────

describe('error translation (check)', () => {
  it('throws AuthError on 401 and 403', async () => {
    client.GET.mockResolvedValueOnce(fail(401, { detail: 'expired' }));
    await expect(realApi.auth.providers()).rejects.toBeInstanceOf(AuthError);
    client.GET.mockResolvedValueOnce(fail(403));
    await expect(realApi.auth.providers()).rejects.toBeInstanceOf(AuthError);
  });

  it('throws ValidationError on 400 and 422', async () => {
    client.GET.mockResolvedValueOnce(fail(400, { detail: 'bad' }));
    await expect(realApi.auth.providers()).rejects.toBeInstanceOf(ValidationError);
    client.GET.mockResolvedValueOnce(fail(422));
    await expect(realApi.auth.providers()).rejects.toBeInstanceOf(ValidationError);
  });

  it('throws NetworkError on 5xx', async () => {
    client.GET.mockResolvedValueOnce(fail(503));
    await expect(realApi.auth.providers()).rejects.toBeInstanceOf(NetworkError);
  });

  it('throws a generic Error on other failures and prefers detail then title', async () => {
    client.GET.mockResolvedValueOnce(fail(418, { title: 'Teapot' }));
    await expect(realApi.auth.providers()).rejects.toThrow('Teapot');
  });
});

// ─── auth ─────────────────────────────────────────────────────────────────────

describe('auth', () => {
  it('providers maps each provider', async () => {
    client.GET.mockResolvedValueOnce(ok([{ id: 'password' }, { id: 'oidc' }]));
    const res = await realApi.auth.providers();
    expect(client.GET).toHaveBeenCalledWith('/auth/providers');
    expect(res).toHaveLength(2);
    expect(res[0]).toMatchObject({ __mapped: 'provider' });
  });

  it('login posts credentials and returns token + mapped user', async () => {
    client.POST.mockResolvedValueOnce(ok({ token: 'jwt', user: { id: 'u1' } }));
    const res = await realApi.auth.login('a@b.c', 'pw');
    expect(client.POST).toHaveBeenCalledWith('/auth/login', { body: { email: 'a@b.c', password: 'pw' } });
    expect(res).toMatchObject({ token: 'jwt', provider: 'password', user: { __mapped: 'user' } });
  });

  it('currentUser returns null on 401 and a mapped user otherwise', async () => {
    client.GET.mockResolvedValueOnce(fail(401));
    expect(await realApi.auth.currentUser()).toBeNull();
    client.GET.mockResolvedValueOnce(ok({ id: 'u1' }));
    expect(await realApi.auth.currentUser()).toMatchObject({ __mapped: 'user' });
  });

  it('logout posts to the logout endpoint', async () => {
    client.POST.mockResolvedValueOnce(ok(undefined, 204));
    await realApi.auth.logout();
    expect(client.POST).toHaveBeenCalledWith('/auth/logout', {});
  });

  it('exportData fetches and returns the export document', async () => {
    const doc = { exportedAt: '2026-06-27', profile: { email: 'me@example.com' } };
    client.GET.mockResolvedValueOnce(ok(doc));
    await expect(realApi.auth.exportData()).resolves.toEqual(doc);
    expect(client.GET).toHaveBeenCalledWith('/auth/me/data-export');
  });

  it('deleteAccount sends the confirmation email and resolves on success', async () => {
    client.DELETE.mockResolvedValueOnce(ok(undefined, 204));
    await expect(realApi.auth.deleteAccount('me@example.com')).resolves.toBeUndefined();
    expect(client.DELETE).toHaveBeenCalledWith('/auth/me', { body: { confirmEmail: 'me@example.com' } });
  });

  it('deleteAccount throws on failure', async () => {
    client.DELETE.mockResolvedValueOnce(fail(401));
    await expect(realApi.auth.deleteAccount('wrong@example.com')).rejects.toThrow('HTTP 401');
  });

  it('setPhoto uploads multipart then refetches the user', async () => {
    const fetchMock = vi.fn().mockResolvedValue({ ok: true });
    vi.stubGlobal('fetch', fetchMock);
    client.GET.mockResolvedValueOnce(ok({ id: 'u1' }));
    const dataUrl = 'data:image/jpeg;base64,' + btoa('hello');
    const res = await realApi.auth.setPhoto(dataUrl);
    expect(fetchMock).toHaveBeenCalledTimes(1);
    const [, init] = fetchMock.mock.calls[0];
    expect(init.method).toBe('POST');
    expect(init.credentials).toBe('include');
    expect(res).toMatchObject({ __mapped: 'user' });
    vi.unstubAllGlobals();
  });

  it('setPhoto rejects an invalid data URL', async () => {
    await expect(realApi.auth.setPhoto('not-a-data-url')).rejects.toThrow();
  });

  it('setPhoto throws when the upload fails', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ ok: false, status: 500 }));
    const dataUrl = 'data:image/jpeg;base64,' + btoa('x');
    await expect(realApi.auth.setPhoto(dataUrl)).rejects.toThrow('Photo upload failed');
    vi.unstubAllGlobals();
  });
});

// ─── teams ──────────────────────────────────────────────────────────────────

describe('teams', () => {
  it('listForCurrentUser maps each team', async () => {
    client.GET.mockResolvedValueOnce(ok([{ id: 't1' }]));
    const res = await realApi.teams.listForCurrentUser();
    expect(client.GET).toHaveBeenCalledWith('/teams');
    expect(res[0]).toMatchObject({ __mapped: 'teamForUser' });
  });

  it('get passes the teamId path param', async () => {
    client.GET.mockResolvedValueOnce(ok({ id: 't1' }));
    await realApi.teams.get('t1');
    expect(client.GET).toHaveBeenCalledWith('/teams/{teamId}', { params: { path: { teamId: 't1' } } });
  });

  it('create posts the team body', async () => {
    client.POST.mockResolvedValueOnce(ok({ id: 't1' }));
    await realApi.teams.create({ name: 'A', icon: 'i' });
    expect(client.POST).toHaveBeenCalledWith('/teams', { body: { name: 'A', icon: 'i', iconBg: undefined, iconFg: undefined } });
  });

  it('updateSettings maps reasonVisibilityRoles onto reasonVisibilityRoleIds', async () => {
    client.PATCH.mockResolvedValueOnce(ok({ id: 't1' }));
    await realApi.teams.updateSettings('t1', { name: 'New', reasonVisibilityRoles: ['r1'] });
    expect(client.PATCH).toHaveBeenCalledWith(
      '/teams/{teamId}',
      expect.objectContaining({ body: expect.objectContaining({ name: 'New', reasonVisibilityRoleIds: ['r1'] }) }),
    );
  });

  it('createInvite maps the invite', async () => {
    client.POST.mockResolvedValueOnce(ok({ code: 'abc' }));
    const res = await realApi.teams.createInvite('t1');
    expect(res).toMatchObject({ __mapped: 'invite' });
  });
});

// ─── members ──────────────────────────────────────────────────────────────────

describe('members', () => {
  it('list maps members', async () => {
    client.GET.mockResolvedValueOnce(ok([{ id: 'm1' }]));
    const res = await realApi.members.list('t1');
    expect(res[0]).toMatchObject({ __mapped: 'member' });
  });

  it('add forwards roleIDs as roleIds', async () => {
    client.POST.mockResolvedValueOnce(ok({ id: 'm1' }));
    await realApi.members.add('t1', { name: 'N', email: 'e@x.c', roleIDs: ['r1'] });
    expect(client.POST).toHaveBeenCalledWith(
      '/teams/{teamId}/members',
      expect.objectContaining({ body: expect.objectContaining({ roleIds: ['r1'] }) }),
    );
  });

  it('update patches a membership', async () => {
    client.PATCH.mockResolvedValueOnce(ok({ id: 'm1' }));
    await realApi.members.update('m1', { name: 'N' }, 't1');
    expect(client.PATCH).toHaveBeenCalledWith(
      '/teams/{teamId}/members/{membershipId}',
      expect.objectContaining({ params: { path: { teamId: 't1', membershipId: 'm1' } } }),
    );
  });

  it('setRoles puts the role ids', async () => {
    client.PUT.mockResolvedValueOnce(ok({ id: 'm1' }));
    await realApi.members.setRoles('m1', ['r1', 'r2'], 't1');
    expect(client.PUT).toHaveBeenCalledWith(
      '/teams/{teamId}/members/{membershipId}/roles',
      expect.objectContaining({ body: { roleIds: ['r1', 'r2'] } }),
    );
  });

  it('remove resolves on ok and throws on failure', async () => {
    client.DELETE.mockResolvedValueOnce(ok(undefined, 204));
    await expect(realApi.members.remove('m1', 't1')).resolves.toBeUndefined();
    client.DELETE.mockResolvedValueOnce(fail(409));
    await expect(realApi.members.remove('m1', 't1')).rejects.toThrow('HTTP 409');
  });
});

// ─── roles ──────────────────────────────────────────────────────────────────

describe('roles', () => {
  it('list, create, update, remove hit the role endpoints', async () => {
    client.GET.mockResolvedValueOnce(ok([{ id: 'r1' }]));
    expect((await realApi.roles.list('t1'))[0]).toMatchObject({ __mapped: 'role' });

    client.POST.mockResolvedValueOnce(ok({ id: 'r1' }));
    await realApi.roles.create('t1', { name: 'Trainer', permissions: {} as never });
    expect(client.POST).toHaveBeenCalledWith('/teams/{teamId}/roles', expect.anything());

    client.PATCH.mockResolvedValueOnce(ok({ id: 'r1' }));
    await realApi.roles.update('r1', { name: 'X' }, 't1');
    expect(client.PATCH).toHaveBeenCalledWith('/teams/{teamId}/roles/{roleId}', expect.anything());

    client.DELETE.mockResolvedValueOnce(ok(undefined, 204));
    await expect(realApi.roles.remove('r1', 't1')).resolves.toBeUndefined();
  });
});

// ─── events ──────────────────────────────────────────────────────────────────

describe('events', () => {
  it('list passes the scope query', async () => {
    client.GET.mockResolvedValueOnce(ok([{ id: 'e1' }]));
    await realApi.events.list('t1', 'upcoming');
    expect(client.GET).toHaveBeenCalledWith(
      '/teams/{teamId}/events',
      expect.objectContaining({ params: { path: { teamId: 't1' }, query: { scope: 'upcoming' } } }),
    );
  });

  it('get returns null on 404', async () => {
    client.GET.mockResolvedValueOnce(fail(404));
    expect(await realApi.events.get('e1', 't1')).toBeNull();
  });

  it('create returns the first element when the backend returns a series array', async () => {
    client.POST.mockResolvedValueOnce(ok([{ id: 'e1' }, { id: 'e2' }]));
    const res = await realApi.events.create('t1', { type: 'training', title: 'T', date: '2026-01-01' });
    expect(res).toMatchObject({ __mapped: 'teamEvent', input: { id: 'e1' } });
  });

  it('create handles a single-object response', async () => {
    client.POST.mockResolvedValueOnce(ok({ id: 'e1' }));
    const res = await realApi.events.create('t1', { type: 'event', title: 'T', date: '2026-01-01' });
    expect(res).toMatchObject({ __mapped: 'teamEvent', input: { id: 'e1' } });
  });

  it('create throws when no data is returned', async () => {
    client.POST.mockResolvedValueOnce({ data: undefined, response: { ok: false, status: 500 } as Response });
    await expect(realApi.events.create('t1', { type: 'event', title: 'T', date: 'd' })).rejects.toThrow('HTTP 500');
  });

  it('update, setStatus and remove pass the scope query', async () => {
    client.PATCH.mockResolvedValueOnce(ok({ id: 'e1' }));
    await realApi.events.update('e1', { title: 'X' }, 'series', 't1');
    expect(client.PATCH).toHaveBeenCalledWith(
      '/teams/{teamId}/events/{eventId}',
      expect.objectContaining({ params: { path: { teamId: 't1', eventId: 'e1' }, query: { scope: 'series' } } }),
    );

    client.POST.mockResolvedValueOnce(ok({ id: 'e1' }));
    await realApi.events.setStatus('e1', 'cancelled', 'single', 't1');
    expect(client.POST).toHaveBeenCalledWith('/teams/{teamId}/events/{eventId}/status', expect.anything());

    client.DELETE.mockResolvedValueOnce(ok(undefined, 204));
    await expect(realApi.events.remove('e1', 'single', 't1')).resolves.toBeUndefined();
  });

  it('comments: list, add and remove', async () => {
    client.GET.mockResolvedValueOnce(ok([{ id: 'c1' }]));
    expect((await realApi.events.listComments('e1', 't1'))[0]).toMatchObject({ __mapped: 'eventComment' });

    client.POST.mockResolvedValueOnce(ok({ id: 'c1' }));
    expect(await realApi.events.addComment('e1', 'hi', 't1')).toMatchObject({ __mapped: 'eventComment' });

    client.DELETE.mockResolvedValueOnce(ok(undefined, 204));
    await expect(realApi.events.removeComment('c1', 'e1', 't1')).resolves.toBeUndefined();
  });
});

// ─── attendance ───────────────────────────────────────────────────────────────

describe('attendance', () => {
  it('listForEvent maps rows', async () => {
    client.GET.mockResolvedValueOnce(ok([{ id: 'a1' }]));
    expect((await realApi.attendance.listForEvent('e1', 't1'))[0]).toMatchObject({ __mapped: 'attendanceRow' });
  });

  it('set posts the attendance body', async () => {
    client.POST.mockResolvedValueOnce(ok({ ok: true }));
    await realApi.attendance.set('e1', 'u1', { status: 'yes' }, 't1');
    expect(client.POST).toHaveBeenCalledWith(
      '/teams/{teamId}/events/{eventId}/attendance',
      expect.objectContaining({ body: expect.objectContaining({ userId: 'u1', status: 'yes' }) }),
    );
  });

  it('setNomination puts and returns true', async () => {
    client.PUT.mockResolvedValueOnce(ok(undefined, 200));
    expect(await realApi.attendance.setNomination('e1', 'u1', true, 't1')).toBe(true);
  });
});

// ─── absences ─────────────────────────────────────────────────────────────────

describe('absences', () => {
  it('listForTeam, listMine, create, update, remove', async () => {
    client.GET.mockResolvedValueOnce(ok([{ id: 'ab1' }]));
    expect((await realApi.absences.listForTeam('t1'))[0]).toMatchObject({ __mapped: 'absence' });

    client.GET.mockResolvedValueOnce(ok([{ id: 'ab1' }]));
    expect((await realApi.absences.listMine('t1'))[0]).toMatchObject({ __mapped: 'absence' });

    client.POST.mockResolvedValueOnce(ok({ id: 'ab1' }));
    expect(await realApi.absences.create({ teamId: 't1', userId: 'u1', from: 'a', to: 'b' })).toMatchObject({ __mapped: 'absence' });

    client.PATCH.mockResolvedValueOnce(ok({ id: 'ab1' }));
    expect(await realApi.absences.update('ab1', { reason: 'r' }, 't1')).toMatchObject({ __mapped: 'absence' });

    client.DELETE.mockResolvedValueOnce(ok(undefined, 204));
    await expect(realApi.absences.remove('ab1', 't1')).resolves.toBeUndefined();
  });
});

// ─── news ──────────────────────────────────────────────────────────────────

describe('news', () => {
  it('list, create, update, remove', async () => {
    client.GET.mockResolvedValueOnce(ok([{ id: 'n1' }]));
    expect((await realApi.news.list('t1'))[0]).toMatchObject({ __mapped: 'newsItem' });

    client.POST.mockResolvedValueOnce(ok({ id: 'n1' }));
    expect(await realApi.news.create('t1', { title: 'T', body: 'B' })).toMatchObject({ __mapped: 'newsItem' });

    client.PATCH.mockResolvedValueOnce(ok({ id: 'n1' }));
    expect(await realApi.news.update('n1', { pinned: true }, 't1')).toMatchObject({ __mapped: 'newsItem' });

    client.DELETE.mockResolvedValueOnce(ok(undefined, 204));
    await expect(realApi.news.remove('n1', 't1')).resolves.toBeUndefined();
  });
});

// ─── polls ──────────────────────────────────────────────────────────────────

describe('polls', () => {
  it('list, vote, create, remove', async () => {
    client.GET.mockResolvedValueOnce(ok([{ id: 'p1' }]));
    expect((await realApi.polls.list('t1'))[0]).toMatchObject({ __mapped: 'poll' });

    client.POST.mockResolvedValueOnce(ok(undefined, 204));
    await expect(realApi.polls.vote('p1', ['o1'], 't1')).resolves.toBeUndefined();

    client.POST.mockResolvedValueOnce(ok({ id: 'p1' }));
    expect(await realApi.polls.create('t1', { question: 'Q?', options: ['a', 'b'] })).toMatchObject({ __mapped: 'poll' });

    client.DELETE.mockResolvedValueOnce(ok(undefined, 204));
    await expect(realApi.polls.remove('p1', 't1')).resolves.toBeUndefined();
  });
});

// ─── finances ─────────────────────────────────────────────────────────────────

describe('finances', () => {
  it('overview maps the finance overview', async () => {
    client.GET.mockResolvedValueOnce(ok({ balance: 0 }));
    expect(await realApi.finances.overview('t1')).toMatchObject({ __mapped: 'finance' });
  });

  it('transaction lifecycle', async () => {
    client.POST.mockResolvedValueOnce(ok({ id: 'tx1' }));
    expect(await realApi.finances.addTransaction('t1', { type: 'income', title: 'T', amount: 5 })).toMatchObject({ __mapped: 'transaction' });

    client.PATCH.mockResolvedValueOnce(ok({ id: 'tx1' }));
    expect(await realApi.finances.updateTransaction('tx1', { amount: 9 }, 't1')).toMatchObject({ __mapped: 'transaction' });

    client.DELETE.mockResolvedValueOnce(ok(undefined, 204));
    await expect(realApi.finances.deleteTransaction('tx1', 't1')).resolves.toBeUndefined();
  });

  it('penalty lifecycle', async () => {
    client.POST.mockResolvedValueOnce(ok({ id: 'pe1' }));
    expect(await realApi.finances.createPenalty('t1', { label: 'Late', amount: 2 })).toMatchObject({ __mapped: 'penalty' });

    client.PATCH.mockResolvedValueOnce(ok({ id: 'pe1' }));
    expect(await realApi.finances.updatePenalty('pe1', { amount: 3 }, 't1')).toMatchObject({ __mapped: 'penalty' });

    client.DELETE.mockResolvedValueOnce(ok(undefined, 204));
    await expect(realApi.finances.deletePenalty('pe1', 't1')).resolves.toBeUndefined();
  });

  it('penalty assignments', async () => {
    client.POST.mockResolvedValueOnce(ok({ id: 'pa1' }));
    expect(await realApi.finances.assignPenalty('t1', { userId: 'u1', penaltyId: 'pe1' })).toMatchObject({
      __mapped: 'penaltyAssignment',
    });

    client.DELETE.mockResolvedValueOnce(ok(undefined, 204));
    await expect(realApi.finances.deleteAssignment('pa1', 't1')).resolves.toBeUndefined();

    client.POST.mockResolvedValueOnce(ok({ id: 'pa1' }));
    expect(await realApi.finances.togglePenaltyPaid('pa1', 't1')).toMatchObject({ __mapped: 'penaltyAssignment' });
  });

  it('contributions', async () => {
    client.PATCH.mockResolvedValueOnce(ok({ id: 'co1' }));
    expect(await realApi.finances.updateContribution('co1', { amount: 1 }, 't1')).toMatchObject({ __mapped: 'contribution' });

    client.POST.mockResolvedValueOnce(ok({ id: 'co1' }));
    expect(await realApi.finances.toggleContribution('co1', 't1')).toMatchObject({ __mapped: 'contribution' });
  });
});

// ─── stats ──────────────────────────────────────────────────────────────────

describe('stats', () => {
  it('attendanceFor returns the raw quote/counted/yes', async () => {
    client.GET.mockResolvedValueOnce(ok({ quote: 0.5, counted: 4, yes: 2 }));
    expect(await realApi.stats.attendanceFor('t1', 'u1')).toEqual({ quote: 0.5, counted: 4, yes: 2 });
  });

  it('teamOverview forwards the date range and maps the overview', async () => {
    client.GET.mockResolvedValueOnce(ok({ totals: {} }));
    await realApi.stats.teamOverview('t1', { from: '2026-01-01', to: '2026-02-01' });
    expect(client.GET).toHaveBeenCalledWith(
      '/teams/{teamId}/stats',
      expect.objectContaining({ params: { path: { teamId: 't1' }, query: { from: '2026-01-01', to: '2026-02-01' } } }),
    );
  });
});

// ─── notifications ────────────────────────────────────────────────────────────

describe('notifications', () => {
  it('list maps the notifications result', async () => {
    client.GET.mockResolvedValueOnce(ok({ items: [] }));
    expect(await realApi.notifications.list('t1')).toMatchObject({ __mapped: 'notifications' });
  });

  it('markSeen posts and returns true', async () => {
    client.POST.mockResolvedValueOnce(ok(undefined, 204));
    expect(await realApi.notifications.markSeen('t1')).toBe(true);
  });
});

afterEach(() => {
  vi.unstubAllGlobals();
});
