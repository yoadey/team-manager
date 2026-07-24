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
import { AuthError, ForbiddenError, NetworkError, ValidationError } from '@/utils/errors';

// ── Mock the HTTP client ─────────────────────────────────────────────────────
vi.mock('@/api/client', () => ({
  apiClient: {
    GET: vi.fn(),
    POST: vi.fn(),
    PUT: vi.fn(),
    PATCH: vi.fn(),
    DELETE: vi.fn(),
  },
  apiOrigin: '',
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
    mapAcceptInviteResponse: tag('acceptInviteResponse'),
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
    // Real (not tagged) — serviceLayerReal calls these directly on request
    // bodies, not just on responses, so tests exercising request payloads
    // need the actual conversion, not an identity stub.
    centsToEuros: (cents: number) => cents / 100,
    eurosToCents: (euros: number) => Math.round(euros * 100),
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
  it('throws AuthError on 401 and ForbiddenError on 403', async () => {
    client.GET.mockResolvedValueOnce(fail(401, { detail: 'expired' }));
    await expect(realApi.auth.providers()).rejects.toBeInstanceOf(AuthError);
    client.GET.mockResolvedValueOnce(fail(403));
    await expect(realApi.auth.providers()).rejects.toBeInstanceOf(ForbiddenError);
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

  it('deleteAccount surfaces the server-provided detail message instead of a generic HTTP status', async () => {
    client.DELETE.mockResolvedValueOnce(fail(400, { detail: 'Confirmation email does not match' }));
    await expect(realApi.auth.deleteAccount('wrong@example.com')).rejects.toThrow('Confirmation email does not match');
  });

  it('deleteAccount throws AuthError with the extracted detail on 401', async () => {
    client.DELETE.mockResolvedValueOnce(fail(401, { detail: 'Session expired' }));
    const err = await realApi.auth.deleteAccount('me@example.com').catch((e: unknown) => e);
    expect(err).toBeInstanceOf(AuthError);
    expect((err as Error).message).toBe('Session expired');
  });

  it('setPhoto uploads multipart via PUT then refetches the user', async () => {
    const fetchMock = vi.fn().mockResolvedValue({ ok: true, status: 200 });
    vi.stubGlobal('fetch', fetchMock);
    client.GET.mockResolvedValueOnce(ok({ id: 'u1' }));
    const dataUrl = 'data:image/jpeg;base64,' + btoa('hello');
    const res = await realApi.auth.setPhoto(dataUrl);
    expect(fetchMock).toHaveBeenCalledTimes(1);
    const [, init] = fetchMock.mock.calls[0]!;
    // The backend registers PUT (not POST) for /auth/me/photo.
    expect(init.method).toBe('PUT');
    expect(init.credentials).toBe('include');
    expect(res).toMatchObject({ __mapped: 'user' });
    vi.unstubAllGlobals();
  });

  it('setPhoto rejects an invalid data URL', async () => {
    await expect(realApi.auth.setPhoto('not-a-data-url')).rejects.toThrow();
  });

  it('setPhoto throws AuthError when the session expired (401)', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue({ ok: false, status: 401, clone: () => ({ json: () => Promise.resolve({}) }) }),
    );
    const dataUrl = 'data:image/jpeg;base64,' + btoa('x');
    await expect(realApi.auth.setPhoto(dataUrl)).rejects.toThrow(AuthError);
    vi.unstubAllGlobals();
  });

  it('setPhoto throws when the upload fails', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue({ ok: false, status: 500, clone: () => ({ json: () => Promise.resolve({}) }) }),
    );
    const dataUrl = 'data:image/jpeg;base64,' + btoa('x');
    await expect(realApi.auth.setPhoto(dataUrl)).rejects.toThrow('HTTP 500');
    vi.unstubAllGlobals();
  });

  it('setPhoto (uploadImage) surfaces the server-provided detail message, e.g. a file-size limit', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue({
        ok: false,
        status: 413,
        clone: () => ({ json: () => Promise.resolve({ detail: 'File too large (max 5 MB)' }) }),
      }),
    );
    const dataUrl = 'data:image/jpeg;base64,' + btoa('x');
    await expect(realApi.auth.setPhoto(dataUrl)).rejects.toThrow('File too large (max 5 MB)');
    vi.unstubAllGlobals();
  });

  it('setPhoto (uploadImage) falls back to a generic HTTP message when the error body is not JSON', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue({
        ok: false,
        status: 500,
        clone: () => ({ json: () => Promise.reject(new Error('not json')) }),
      }),
    );
    const dataUrl = 'data:image/jpeg;base64,' + btoa('x');
    await expect(realApi.auth.setPhoto(dataUrl)).rejects.toThrow('HTTP 500');
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
    expect(client.POST).toHaveBeenCalledWith('/teams', {
      body: { name: 'A', icon: 'i', iconBg: undefined, iconFg: undefined },
    });
  });

  it('create does not attempt a photo upload when no photo was picked', async () => {
    const fetchMock = vi.fn();
    vi.stubGlobal('fetch', fetchMock);
    client.POST.mockResolvedValueOnce(ok({ id: 't1' }));
    await realApi.teams.create({ name: 'A' });
    expect(fetchMock).not.toHaveBeenCalled();
    vi.unstubAllGlobals();
  });

  // CreateTeamRequest (openapi.yaml) has no photo field, so a photo picked in
  // CreateTeamSheet before the first save must be uploaded as a second step
  // via the per-team PUT .../photo endpoint once the team id exists —
  // otherwise it's silently dropped against the real backend while the mock
  // (which accepts `photo` directly in its create() call) persists it.
  it('create uploads a picked photo via multipart PUT after the team is created', async () => {
    const fetchMock = vi.fn().mockResolvedValue({ ok: true, status: 200 });
    vi.stubGlobal('fetch', fetchMock);
    client.POST.mockResolvedValueOnce(ok({ id: 't1' }));
    client.GET.mockResolvedValueOnce(ok({ id: 't1' }));
    const dataUrl = 'data:image/png;base64,' + btoa('img');
    await realApi.teams.create({ name: 'A', photo: dataUrl });
    expect(fetchMock).toHaveBeenCalledTimes(1);
    const [url, init] = fetchMock.mock.calls[0]!;
    expect(url).toContain('/api/v1/teams/t1/photo');
    expect(init.method).toBe('PUT');
    expect(client.GET).toHaveBeenCalledWith('/teams/{teamId}', { params: { path: { teamId: 't1' } } });
    vi.unstubAllGlobals();
  });

  it('updateSettings maps reasonVisibilityRoles onto reasonVisibilityRoleIds', async () => {
    client.PATCH.mockResolvedValueOnce(ok({ id: 't1' }));
    client.GET.mockResolvedValueOnce(ok({ id: 't1' }));
    await realApi.teams.updateSettings('t1', { name: 'New', reasonVisibilityRoles: ['r1'] });
    expect(client.PATCH).toHaveBeenCalledWith(
      '/teams/{teamId}',
      expect.objectContaining({ body: expect.objectContaining({ name: 'New', reasonVisibilityRoleIds: ['r1'] }) }),
    );
  });

  it('updateSettings uploads photo/logo via multipart PUT and skips the JSON PATCH when no other field changed', async () => {
    const fetchMock = vi.fn().mockResolvedValue({ ok: true, status: 200 });
    vi.stubGlobal('fetch', fetchMock);
    client.GET.mockResolvedValueOnce(ok({ id: 't1' }));
    const dataUrl = 'data:image/png;base64,' + btoa('img');
    await realApi.teams.updateSettings('t1', { photo: dataUrl });
    expect(client.PATCH).not.toHaveBeenCalled();
    expect(fetchMock).toHaveBeenCalledTimes(1);
    const [url, init] = fetchMock.mock.calls[0]!;
    expect(url).toContain('/api/v1/teams/t1/photo');
    expect(init.method).toBe('PUT');
    vi.unstubAllGlobals();
  });

  it('updateSettings uploads both photo and logo alongside JSON fields', async () => {
    const fetchMock = vi.fn().mockResolvedValue({ ok: true, status: 200 });
    vi.stubGlobal('fetch', fetchMock);
    client.PATCH.mockResolvedValueOnce(ok({ id: 't1' }));
    client.GET.mockResolvedValueOnce(ok({ id: 't1' }));
    const photoUrl = 'data:image/png;base64,' + btoa('photo');
    const logoUrl = 'data:image/png;base64,' + btoa('logo');
    await realApi.teams.updateSettings('t1', { name: 'New', photo: photoUrl, logo: logoUrl });
    expect(client.PATCH).toHaveBeenCalledTimes(1);
    expect(fetchMock).toHaveBeenCalledTimes(2);
    expect(fetchMock.mock.calls[0]![0]).toContain('/photo');
    expect(fetchMock.mock.calls[1]![0]).toContain('/logo');
    vi.unstubAllGlobals();
  });

  it('updateSettings calls DELETE when logo is explicitly null (revert to icon)', async () => {
    client.PATCH.mockResolvedValueOnce(ok({ id: 't1' }));
    client.DELETE.mockResolvedValueOnce({ response: { ok: true, status: 204 } });
    client.GET.mockResolvedValueOnce(ok({ id: 't1' }));
    await realApi.teams.updateSettings('t1', { icon: '🏆', logo: null });
    expect(client.DELETE).toHaveBeenCalledWith('/teams/{teamId}/logo', { params: { path: { teamId: 't1' } } });
  });

  it('updateSettings calls DELETE when photo is explicitly null', async () => {
    client.DELETE.mockResolvedValueOnce({ response: { ok: true, status: 204 } });
    client.GET.mockResolvedValueOnce(ok({ id: 't1' }));
    await realApi.teams.updateSettings('t1', { photo: null });
    expect(client.DELETE).toHaveBeenCalledWith('/teams/{teamId}/photo', { params: { path: { teamId: 't1' } } });
  });

  it('updateSettings does not call DELETE when photo/logo are simply omitted', async () => {
    client.PATCH.mockResolvedValueOnce(ok({ id: 't1' }));
    client.GET.mockResolvedValueOnce(ok({ id: 't1' }));
    await realApi.teams.updateSettings('t1', { name: 'New' });
    expect(client.DELETE).not.toHaveBeenCalled();
  });

  it('createInvite maps the invite', async () => {
    client.POST.mockResolvedValueOnce(ok({ code: 'abc' }));
    const res = await realApi.teams.createInvite('t1');
    expect(res).toMatchObject({ __mapped: 'invite' });
  });

  it('acceptInvite posts the code and maps the resulting team', async () => {
    client.POST.mockResolvedValueOnce(ok({ id: 't1' }));
    const res = await realApi.teams.acceptInvite('abc123');
    expect(client.POST).toHaveBeenCalledWith('/invites/{code}/accept', { params: { path: { code: 'abc123' } } });
    expect(res).toMatchObject({ __mapped: 'acceptInviteResponse' });
  });
});

// ─── members ──────────────────────────────────────────────────────────────────

describe('members', () => {
  it('list maps members', async () => {
    client.GET.mockResolvedValueOnce(ok({ items: [{ id: 'm1' }], nextCursor: null }));
    const res = await realApi.members.list('t1');
    expect(res[0]).toMatchObject({ __mapped: 'member' });
  });

  it('list walks every keyset page and forwards the cursor', async () => {
    client.GET.mockResolvedValueOnce(ok({ items: [{ id: 'm1' }], nextCursor: 'c1' })).mockResolvedValueOnce(
      ok({ items: [{ id: 'm2' }], nextCursor: null }),
    );
    const res = await realApi.members.list('t1');
    expect(res).toHaveLength(2);
    expect(client.GET).toHaveBeenCalledTimes(2);
    // second request must carry the cursor returned by the first page
    expect(client.GET).toHaveBeenLastCalledWith(
      '/teams/{teamId}/members',
      expect.objectContaining({
        params: expect.objectContaining({ query: expect.objectContaining({ cursor: 'c1' }) }),
      }),
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

  it('remove surfaces the server-provided detail message on failure instead of a generic HTTP status', async () => {
    client.DELETE.mockResolvedValueOnce(fail(409, { detail: 'Cannot remove the last admin' }));
    await expect(realApi.members.remove('m1', 't1')).rejects.toThrow('Cannot remove the last admin');
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
    client.GET.mockResolvedValueOnce(ok({ items: [{ id: 'e1' }], nextCursor: null }));
    await realApi.events.list('t1', 'upcoming');
    expect(client.GET).toHaveBeenCalledWith(
      '/teams/{teamId}/events',
      expect.objectContaining({
        params: expect.objectContaining({ query: expect.objectContaining({ scope: 'upcoming' }) }),
      }),
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

  it('create sends meetT/startT/endT under the meetTime/startTime/endTime keys the backend expects', async () => {
    client.POST.mockResolvedValueOnce(ok({ id: 'e1' }));
    await realApi.events.create('t1', {
      type: 'training',
      title: 'T',
      date: '2026-01-01',
      meetT: '19:15',
      startT: '19:30',
      endT: '21:30',
    });
    expect(client.POST).toHaveBeenCalledWith(
      '/teams/{teamId}/events',
      expect.objectContaining({
        body: expect.objectContaining({ meetTime: '19:15', startTime: '19:30', endTime: '21:30' }),
      }),
    );
  });

  it('update sends meetT/startT/endT under the meetTime/startTime/endTime keys the backend expects', async () => {
    client.PATCH.mockResolvedValueOnce(ok({ id: 'e1' }));
    await realApi.events.update('e1', { title: 'X', meetT: '20:00', startT: '20:15', endT: '22:00' }, 'single', 't1');
    expect(client.PATCH).toHaveBeenCalledWith(
      '/teams/{teamId}/events/{eventId}',
      expect.objectContaining({
        body: expect.objectContaining({ meetTime: '20:00', startTime: '20:15', endTime: '22:00' }),
      }),
    );
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
    client.GET.mockResolvedValueOnce(ok({ items: [{ id: 'c1' }], nextCursor: null }));
    expect((await realApi.events.listComments('e1', 't1'))[0]).toMatchObject({ __mapped: 'eventComment' });
    expect(client.GET).toHaveBeenCalledWith(
      '/teams/{teamId}/events/{eventId}/comments',
      expect.objectContaining({ params: { path: { teamId: 't1', eventId: 'e1' }, query: { limit: 500, cursor: undefined } } }),
    );

    client.POST.mockResolvedValueOnce(ok({ id: 'c1' }));
    expect(await realApi.events.addComment('e1', 'hi', 't1')).toMatchObject({ __mapped: 'eventComment' });

    client.DELETE.mockResolvedValueOnce(ok(undefined, 204));
    await expect(realApi.events.removeComment('c1', 'e1', 't1')).resolves.toBeUndefined();
  });

  // listEventComments is keyset paginated with a { items, nextCursor } envelope
  // (oldest-first). Without walking every page to completion an event with a
  // full page of comments would silently lose everything after it against the
  // real backend, while the mock returns every comment unconditionally.
  it('listComments walks every keyset page and forwards the cursor', async () => {
    const fullPage = Array.from({ length: 500 }, (_, i) => ({ id: `c${i}` }));
    client.GET
      .mockResolvedValueOnce(ok({ items: fullPage, nextCursor: 'c1' }))
      .mockResolvedValueOnce(ok({ items: [{ id: 'c500' }], nextCursor: null }));
    const res = await realApi.events.listComments('e1', 't1');
    expect(res).toHaveLength(501);
    expect(client.GET).toHaveBeenCalledTimes(2);
    expect(client.GET).toHaveBeenNthCalledWith(
      1,
      '/teams/{teamId}/events/{eventId}/comments',
      expect.objectContaining({ params: expect.objectContaining({ query: { limit: 500, cursor: undefined } }) }),
    );
    // The second request must carry the cursor the first page returned.
    expect(client.GET).toHaveBeenNthCalledWith(
      2,
      '/teams/{teamId}/events/{eventId}/comments',
      expect.objectContaining({ params: expect.objectContaining({ query: { limit: 500, cursor: 'c1' } }) }),
    );
  });

  it('issueCalendarFeedToken posts and returns the url', async () => {
    client.POST.mockResolvedValueOnce(ok({ url: 'https://app.example.com/api/v1/calendar-feed/abc.ics' }));
    const url = await realApi.events.issueCalendarFeedToken('t1');
    expect(client.POST).toHaveBeenCalledWith(
      '/teams/{teamId}/calendar-feed/token',
      expect.objectContaining({ params: { path: { teamId: 't1' } } }),
    );
    expect(url).toBe('https://app.example.com/api/v1/calendar-feed/abc.ics');
  });

  it('revokeCalendarFeedToken deletes', async () => {
    client.DELETE.mockResolvedValueOnce(ok(undefined, 204));
    await realApi.events.revokeCalendarFeedToken('t1');
    expect(client.DELETE).toHaveBeenCalledWith(
      '/teams/{teamId}/calendar-feed/token',
      expect.objectContaining({ params: { path: { teamId: 't1' } } }),
    );
  });
});

// ─── attendance ───────────────────────────────────────────────────────────────

describe('attendance', () => {
  it('listForEvent maps rows', async () => {
    client.GET.mockResolvedValueOnce(ok([{ id: 'a1' }]));
    expect((await realApi.attendance.listForEvent('e1', 't1'))[0]).toMatchObject({ __mapped: 'attendanceRow' });
  });

  it('listForEvent groups by status (yes, maybe, pending, no, not_nominated) then name, matching the mock', async () => {
    // The backend orders attendance rows alphabetically only (ORDER BY
    // u.name ASC); EventDetailSheet.tsx renders the rows in the order it
    // receives them, so the service layer must re-impose the mock's status
    // grouping itself.
    client.GET.mockResolvedValueOnce(
      ok([
        { status: 'no', name: 'Bob' },
        { status: 'yes', name: 'Zoe' },
        { status: 'yes', name: 'Anna' },
        { status: 'not_nominated', name: 'Cy' },
        { status: 'pending', name: 'Mo' },
      ]),
    );
    const rows = (await realApi.attendance.listForEvent('e1', 't1')) as unknown as { input: { name: string } }[];
    expect(rows.map((r) => r.input.name)).toEqual(['Anna', 'Zoe', 'Mo', 'Bob', 'Cy']);
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
    client.GET.mockResolvedValueOnce(ok({ items: [{ id: 'ab1' }], nextCursor: null }));
    expect((await realApi.absences.listForTeam('t1'))[0]).toMatchObject({ __mapped: 'absence' });

    client.GET.mockResolvedValueOnce(ok({ items: [{ id: 'ab1' }], nextCursor: null }));
    expect((await realApi.absences.listMine('t1'))[0]).toMatchObject({ __mapped: 'absence' });

    client.POST.mockResolvedValueOnce(ok({ id: 'ab1' }));
    expect(await realApi.absences.create({ teamId: 't1', userId: 'u1', from: 'a', to: 'b' })).toMatchObject({
      __mapped: 'absence',
    });

    client.PATCH.mockResolvedValueOnce(ok({ id: 'ab1' }));
    expect(await realApi.absences.update('ab1', { reason: 'r' }, 't1')).toMatchObject({ __mapped: 'absence' });

    client.DELETE.mockResolvedValueOnce(ok(undefined, 204));
    await expect(realApi.absences.remove('ab1', 't1')).resolves.toBeUndefined();
  });

  it('listForTeam and listMine sort ascending by `from`, matching the mock (backend returns newest-first)', async () => {
    // absences.Repository orders `from_date DESC, id DESC`; the mock sorts
    // ascending so the soonest-upcoming absence is first, and
    // EventAbsences.tsx renders the list in the order it receives with no
    // client-side sort.
    client.GET.mockResolvedValueOnce(
      ok({
        items: [
          { id: 'ab-aug', from: '2026-08-01' },
          { id: 'ab-jul', from: '2026-07-01' },
        ],
        nextCursor: null,
      }),
    );
    const forTeam = (await realApi.absences.listForTeam('t1')) as unknown as { input: { id: string } }[];
    expect(forTeam.map((a) => a.input.id)).toEqual(['ab-jul', 'ab-aug']);

    client.GET.mockResolvedValueOnce(
      ok({
        items: [
          { id: 'ab-aug', from: '2026-08-01' },
          { id: 'ab-jul', from: '2026-07-01' },
        ],
        nextCursor: null,
      }),
    );
    const mine = (await realApi.absences.listMine('t1')) as unknown as { input: { id: string } }[];
    expect(mine.map((a) => a.input.id)).toEqual(['ab-jul', 'ab-aug']);
  });
});

// ─── news ──────────────────────────────────────────────────────────────────

describe('news', () => {
  it('list, create, update, remove', async () => {
    client.GET.mockResolvedValueOnce(ok({ items: [{ id: 'n1' }], nextCursor: null }));
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
    client.GET.mockResolvedValueOnce(ok({ items: [{ id: 'p1' }], nextCursor: null }));
    expect((await realApi.polls.list('t1'))[0]).toMatchObject({ __mapped: 'poll' });

    client.POST.mockResolvedValueOnce(ok(undefined, 204));
    await expect(realApi.polls.vote('p1', ['o1'], 't1')).resolves.toBeUndefined();

    client.POST.mockResolvedValueOnce(ok({ id: 'p1' }));
    expect(await realApi.polls.create('t1', { question: 'Q?', options: ['a', 'b'] })).toMatchObject({
      __mapped: 'poll',
    });

    client.DELETE.mockResolvedValueOnce(ok(undefined, 204));
    await expect(realApi.polls.remove('p1', 't1')).resolves.toBeUndefined();
  });
});

// ─── finances ─────────────────────────────────────────────────────────────────

describe('finances', () => {
  it('overview maps the finance overview', async () => {
    client.GET.mockResolvedValueOnce(ok({ balance: 0, assignments: [] }));
    expect(await realApi.finances.overview('t1')).toMatchObject({ __mapped: 'finance' });
  });

  it('overview reverses assignments to ascending order, matching the mock (backend returns newest-first)', async () => {
    // finances.Repository.ListAssignments orders `pa.date DESC`; the mock's
    // array is in ascending insertion order. FinancesPenalties.tsx renders
    // `f.assignments.slice().reverse()`, which only yields newest-first if
    // fed an ascending array — passing the backend's descending order straight
    // through would get reversed a second time and show the oldest
    // assignment on top.
    client.GET.mockResolvedValueOnce(ok({ balance: 0, assignments: [{ id: 'a-newest' }, { id: 'a-oldest' }] }));
    const overview = (await realApi.finances.overview('t1')) as unknown as {
      input: { assignments: { id: string }[] };
    };
    expect(overview.input.assignments.map((a) => a.id)).toEqual(['a-oldest', 'a-newest']);
  });

  it('transaction lifecycle', async () => {
    client.POST.mockResolvedValueOnce(ok({ id: 'tx1' }));
    expect(await realApi.finances.addTransaction('t1', { type: 'income', title: 'T', amount: 5 })).toMatchObject({
      __mapped: 'transaction',
    });

    client.PATCH.mockResolvedValueOnce(ok({ id: 'tx1' }));
    expect(await realApi.finances.updateTransaction('tx1', { amount: 9 }, 't1')).toMatchObject({
      __mapped: 'transaction',
    });

    client.DELETE.mockResolvedValueOnce(ok(undefined, 204));
    await expect(realApi.finances.deleteTransaction('tx1', 't1')).resolves.toBeUndefined();
  });

  it('deleteTransaction surfaces the server-provided detail message on failure', async () => {
    client.DELETE.mockResolvedValueOnce(fail(422, { detail: 'Transaction already reconciled' }));
    await expect(realApi.finances.deleteTransaction('tx1', 't1')).rejects.toThrow('Transaction already reconciled');
  });

  it('penalty lifecycle', async () => {
    client.POST.mockResolvedValueOnce(ok({ id: 'pe1' }));
    expect(await realApi.finances.createPenalty('t1', { label: 'Late', amount: 2 })).toMatchObject({
      __mapped: 'penalty',
    });

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

    client.PUT.mockResolvedValueOnce(ok({ id: 'pa1' }));
    expect(await realApi.finances.setPenaltyPaid('pa1', 't1', true)).toMatchObject({ __mapped: 'penaltyAssignment' });
  });

  it('contributions', async () => {
    client.PATCH.mockResolvedValueOnce(ok({ id: 'co1' }));
    expect(await realApi.finances.updateContribution('co1', { amount: 1 }, 't1')).toMatchObject({
      __mapped: 'contribution',
    });

    client.PUT.mockResolvedValueOnce(ok({ id: 'co1' }));
    expect(await realApi.finances.setContributionPaid('co1', 't1', true)).toMatchObject({ __mapped: 'contribution' });
  });
});

// ─── stats ──────────────────────────────────────────────────────────────────

describe('stats', () => {
  it('attendanceFor scales the backend 0-1 quote fraction to a 0-100 percentage', async () => {
    client.GET.mockResolvedValueOnce(ok({ quote: 0.5, counted: 4, yes: 2 }));
    expect(await realApi.stats.attendanceFor('t1', 'u1')).toEqual({ quote: 50, counted: 4, yes: 2 });
  });

  it('attendanceFor maps counted:0 to quote:null ("no data"), matching the mock', async () => {
    client.GET.mockResolvedValueOnce(ok({ quote: 0, counted: 0, yes: 0 }));
    expect(await realApi.stats.attendanceFor('t1', 'u1')).toEqual({ quote: null, counted: 0, yes: 0 });
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

describe('push', () => {
  const validSubscription: PushSubscriptionJSON = {
    endpoint: 'https://push.example/abc',
    keys: { p256dh: 'p256dh-value', auth: 'auth-value' },
  };

  it('subscribe posts the endpoint and keys', async () => {
    client.POST.mockResolvedValueOnce(ok(undefined, 204));
    await realApi.push.subscribe(validSubscription);
    expect(client.POST).toHaveBeenCalledWith(
      '/users/me/push-subscriptions',
      expect.objectContaining({
        body: { endpoint: 'https://push.example/abc', keys: { p256dh: 'p256dh-value', auth: 'auth-value' } },
      }),
    );
  });

  it('subscribe rejects a subscription missing endpoint or keys without calling the API', async () => {
    await expect(realApi.push.subscribe({ endpoint: '', keys: undefined } as never)).rejects.toThrow();
    expect(client.POST).not.toHaveBeenCalled();
  });

  it('unsubscribe deletes with the endpoint as a query param', async () => {
    client.DELETE.mockResolvedValueOnce(ok(undefined, 204));
    await realApi.push.unsubscribe('https://push.example/abc');
    expect(client.DELETE).toHaveBeenCalledWith(
      '/users/me/push-subscriptions',
      expect.objectContaining({ params: { query: { endpoint: 'https://push.example/abc' } } }),
    );
  });
});

afterEach(() => {
  vi.unstubAllGlobals();
});
