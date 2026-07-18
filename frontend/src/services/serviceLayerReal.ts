// Real backend service layer — the sole API-contract implementation, for
// production, dev-demo, and tests alike (see src/services/index.ts). In
// dev-demo (no config.apiBaseUrl) its HTTP calls are intercepted by MSW
// (src/mocks/) rather than a second in-code implementation.

import { apiClient, apiOrigin } from '@/api/client';
import {
  mapUser,
  mapProvider,
  mapRole,
  mapTeam,
  mapTeamForUser,
  mapAcceptInviteResponse,
  mapInvite,
  mapMember,
  mapTeamEvent,
  mapAttendanceRow,
  mapEventComment,
  mapAbsence,
  mapNewsItem,
  mapPoll,
  mapNotificationsResult,
  mapFinanceOverview,
  mapTransaction,
  mapPenalty,
  mapPenaltyAssignment,
  mapContribution,
  mapStatsOverview,
  eurosToCents,
} from '@/api/map';
import type { User, Team, TeamForUser, Role, Invite, Provider, DateRange, StatsOverview } from '@/types';
import type { TeamEvent, AttendanceRow, EventComment, Absence } from '@/features/events';
import type { Member } from '@/features/members';
import type { NewsItem } from '@/features/news';
import type { Poll } from '@/features/polls';
import type { NotificationsResult } from '@/features/notifications';
import type { FinanceOverview, Transaction, Penalty, PenaltyAssignment, Contribution } from '@/features/finances';
import { AuthError, ForbiddenError, NetworkError, ValidationError } from '@/utils/errors';

// Builds the typed error for a given HTTP status + optional RFC 9457
// problem+json body, preferring `detail` then `title` over a generic
// "HTTP {status}" message so callers (and the toasts they feed) can surface
// the server's actual explanation (e.g. "Email already in use"). 401 (no/
// expired session) and 403 (valid session, insufficient permission) map to
// distinct error types — conflating them would log a fully-authenticated
// user out just for lacking write access to a module.
function errorFor(status: number, body?: { detail?: string; title?: string } | null): Error {
  const msg = body?.detail ?? body?.title ?? `HTTP ${status}`;
  if (status === 401) return new AuthError(msg);
  if (status === 403) return new ForbiddenError(msg);
  if (status === 400 || status === 422) return new ValidationError(msg);
  if (status >= 500) return new NetworkError(msg);
  return new Error(msg);
}

// Throws a typed error when the API returns an error response, so callers can
// react to the failure class — notably AuthError (401), which the app's
// reportActionError/onAuthError wiring turns into a logout + redirect to the
// login screen when a session expires mid-use.
async function check<T>(result: { data?: T; error?: unknown; response: Response }): Promise<T> {
  if (result.error || !result.data) {
    const err = result.error as { detail?: string; title?: string } | undefined;
    throw errorFor(result.response.status, err);
  }
  return result.data;
}

// Throws the same typed errors as check() when an openapi-fetch result failed,
// without requiring a successful `data` payload (for endpoints whose success
// response is empty/void, e.g. DELETE). Resolves (returns void) otherwise.
async function checkOk(result: { error?: unknown; response: Response }): Promise<void> {
  if (!result.response.ok) {
    const err = result.error as { detail?: string; title?: string } | undefined;
    throw errorFor(result.response.status, err);
  }
}

// Uploads a data: URL as a multipart image field via PUT, throwing the same
// typed errors as check() (notably AuthError on 401, so a session expiring
// mid-upload still triggers the app's logout redirect). On failure, the
// response body is parsed as RFC 9457 problem+json (best-effort) so the
// thrown error carries the server's actual detail (e.g. "File too large")
// instead of a generic "HTTP {status}".
async function uploadImage(path: string, fieldName: string, dataUrl: string): Promise<Response> {
  const arr = dataUrl.split(',');
  const mimeMatch = arr[0].match(/:(.*?);/);
  if (!mimeMatch || arr.length < 2) throw new Error('Invalid data URL format');
  const mime = mimeMatch[1];
  const bstr = atob(arr[1]);
  const bytes = new Uint8Array(bstr.length);
  for (let i = 0; i < bstr.length; i++) bytes[i] = bstr.charCodeAt(i);
  const blob = new Blob([bytes], { type: mime });
  const formData = new FormData();
  formData.append(fieldName, blob, fieldName + '.jpg');

  const resp = await fetch(apiOrigin + path, {
    method: 'PUT',
    credentials: 'include',
    body: formData,
  });
  if (!resp.ok) {
    const body = (await resp
      .clone()
      .json()
      .catch(() => undefined)) as { detail?: string; title?: string } | undefined;
    throw errorFor(resp.status, body);
  }
  return resp;
}

// Per-page size when walking a keyset list to completion (the backend caps at 500).
const PAGE_LIMIT = 500;

// Mirrors serviceLayer.ts's STATUS_ORDER — the display grouping the mock uses
// for attendance rows (see attendance.listForEvent below).
const ATTENDANCE_STATUS_ORDER: Record<string, number> = { yes: 0, maybe: 1, pending: 2, no: 3, not_nominated: 4 };

// fetchAllPages walks the keyset { items, nextCursor } envelope to the end and
// returns every row. The app has no paging UI yet and consumers expect full
// arrays, so without this the real backend would silently truncate lists to the
// first page (the mock returns everything). Pages are fetched sequentially
// because each request needs the previous page's cursor.
async function fetchAllPages<T>(
  fetchPage: (cursor: string | undefined) => Promise<{ items: T[]; nextCursor?: string | null }>,
): Promise<T[]> {
  const all: T[] = [];
  let cursor: string | undefined;
  let more = true;
  while (more) {
    const page = await fetchPage(cursor);
    all.push(...page.items);
    if (all.length > 10_000) throw new Error('fetchAllPages: too many pages');
    cursor = page.nextCursor ?? undefined;
    more = cursor !== undefined;
  }
  return all;
}

// fetchAllOffsetPages walks a plain-array endpoint paginated via limit/offset
// (no { items, nextCursor } envelope, e.g. listEventComments) to completion,
// stopping once a page comes back shorter than PAGE_LIMIT. Without this, the
// real backend's default limit=50 would silently truncate any event with
// more than 50 comments to its oldest 50 (ORDER BY created_at ASC), while the
// mock returns every comment unconditionally.
async function fetchAllOffsetPages<T>(fetchPage: (limit: number, offset: number) => Promise<T[]>): Promise<T[]> {
  const all: T[] = [];
  let offset = 0;
  for (;;) {
    const page = await fetchPage(PAGE_LIMIT, offset);
    all.push(...page);
    if (all.length > 10_000) throw new Error('fetchAllOffsetPages: too many pages');
    if (page.length < PAGE_LIMIT) break;
    offset += PAGE_LIMIT;
  }
  return all;
}

export const realApi = {
  auth: {
    async providers(): Promise<Provider[]> {
      const res = await apiClient.GET('/auth/providers');
      const providers = await check(res);
      return providers.map(mapProvider);
    },

    // login accepts either (email, password) for real backend OR (providerId) for compatibility.
    // When called from tests without VITE_API_BASE_URL, the mock path is used instead.
    async login(email: string, password?: string): Promise<{ token: string; provider: string; user: User }> {
      const res = await apiClient.POST('/auth/login', {
        body: { email: email as string & { format: 'email' }, password: password ?? '' },
      });
      const data = await check(res);
      // The session cookie is set by the server; the body token is unused.
      return { token: data.token, provider: 'password', user: mapUser(data.user) };
    },

    async currentUser(): Promise<User | null> {
      // The session cookie travels automatically; a 401 means no active session.
      const res = await apiClient.GET('/auth/me');
      if (res.response.status === 401) return null;
      const data = await check(res);
      return mapUser(data);
    },

    async logout() {
      const res = await apiClient.POST('/auth/logout', {});
      if (!res.response.ok) await check(res);
    },

    // GDPR Art. 15: returns the personal-data export document for the current
    // user. The caller turns it into a downloadable file.
    async exportData(): Promise<unknown> {
      const res = await apiClient.GET('/auth/me/data-export');
      return check(res);
    },

    // GDPR Art. 17 erasure by anonymization. Authorized by the active session;
    // the caller echoes the account email to confirm intent (works for OIDC
    // accounts that have no password). The server clears the cookie on success.
    async deleteAccount(confirmEmail: string): Promise<void> {
      const res = await apiClient.DELETE('/auth/me', {
        body: { confirmEmail: confirmEmail as string & { format: 'email' } },
      });
      await checkOk(res);
    },

    async setPhoto(dataUrl: string): Promise<User> {
      // The backend registers PUT for this endpoint (not POST).
      await uploadImage('/api/v1/auth/me/photo', 'photo', dataUrl);
      const meRes = await apiClient.GET('/auth/me');
      const data = await check(meRes);
      return mapUser(data);
    },
  },

  teams: {
    async listForCurrentUser(): Promise<TeamForUser[]> {
      const res = await apiClient.GET('/teams');
      const teams = await check(res);
      return teams.map(mapTeamForUser);
    },

    async get(teamId: string): Promise<Team> {
      const res = await apiClient.GET('/teams/{teamId}', { params: { path: { teamId } } });
      const t = await check(res);
      return mapTeam(t);
    },

    async create(opts: {
      name: string;
      icon?: string;
      iconBg?: string;
      iconFg?: string;
      photo?: string | null;
    }): Promise<Team> {
      const res = await apiClient.POST('/teams', {
        body: { name: opts.name, icon: opts.icon, iconBg: opts.iconBg, iconFg: opts.iconFg },
      });
      const t = await check(res);
      // CreateTeamRequest has no photo field (see openapi.yaml) — the mock
      // accepts `opts.photo` directly on the created team, but the real
      // backend only exposes a photo upload via the per-team PUT
      // .../{teamId}/photo endpoint, so it must be applied as a second step
      // once the team (and its id) exists, matching updateSettings()'s
      // photo-upload path below. Without this, a photo picked in
      // CreateTeamSheet (useTeamActions.ts's createTeam()) was silently
      // dropped against the real backend while the mock persisted it.
      if (opts.photo) {
        await uploadImage(`/api/v1/teams/${t.id}/photo`, 'photo', opts.photo);
        const refreshed = await check(await apiClient.GET('/teams/{teamId}', { params: { path: { teamId: t.id } } }));
        return mapTeam(refreshed);
      }
      return mapTeam(t);
    },

    async updateSettings(teamId: string, patch: Partial<Team>): Promise<Team> {
      // photo/logo are uploaded as multipart images via their own PUT
      // endpoints; the JSON PATCH body only carries the other fields (the
      // backend rejects unknown JSON properties for these, and base64 data
      // URLs are not part of the Team JSON schema).
      const hasJsonFields =
        patch.name !== undefined ||
        patch.icon !== undefined ||
        patch.iconBg !== undefined ||
        patch.iconFg !== undefined ||
        patch.description !== undefined ||
        patch.reasonVisibilityRoles !== undefined;

      if (hasJsonFields) {
        const res = await apiClient.PATCH('/teams/{teamId}', {
          params: { path: { teamId } },
          body: {
            name: patch.name,
            icon: patch.icon,
            iconBg: patch.iconBg,
            iconFg: patch.iconFg,
            description: patch.description,
            reasonVisibilityRoleIds: patch.reasonVisibilityRoles,
          },
        });
        await check(res);
      }

      if (patch.photo) {
        await uploadImage(`/api/v1/teams/${teamId}/photo`, 'photo', patch.photo);
      } else if (patch.photo === null) {
        await checkOk(await apiClient.DELETE('/teams/{teamId}/photo', { params: { path: { teamId } } }));
      }
      if (patch.logo) {
        await uploadImage(`/api/v1/teams/${teamId}/logo`, 'logo', patch.logo);
      } else if (patch.logo === null) {
        await checkOk(await apiClient.DELETE('/teams/{teamId}/logo', { params: { path: { teamId } } }));
      }

      const res = await apiClient.GET('/teams/{teamId}', { params: { path: { teamId } } });
      const t = await check(res);
      return mapTeam(t);
    },

    async createInvite(teamId: string): Promise<Invite> {
      const res = await apiClient.POST('/teams/{teamId}/invite', { params: { path: { teamId } } });
      const inv = await check(res);
      return mapInvite(inv);
    },

    async acceptInvite(code: string): Promise<TeamForUser & { alreadyMember: boolean }> {
      const res = await apiClient.POST('/invites/{code}/accept', { params: { path: { code } } });
      const body = await check(res);
      return mapAcceptInviteResponse(body);
    },
  },

  members: {
    async list(teamId: string): Promise<Member[]> {
      // Keyset { items, nextCursor } envelope; walked to completion so the full
      // roster is returned (no paging UI yet).
      const items = await fetchAllPages(async (cursor) => {
        const res = await apiClient.GET('/teams/{teamId}/members', {
          params: { path: { teamId }, query: { limit: PAGE_LIMIT, cursor } },
        });
        return check(res);
      });
      return (items as unknown[]).map((m) => mapMember(m as Parameters<typeof mapMember>[0], teamId));
    },

    async update(
      membershipId: string,
      patch: {
        name?: string;
        email?: string;
        phone?: string | null;
        birthday?: string | null;
        address?: string | null;
        group?: string | null;
      },
      teamId: string,
    ): Promise<Member> {
      const res = await apiClient.PATCH('/teams/{teamId}/members/{membershipId}', {
        params: { path: { teamId, membershipId } },
        body: {
          name: patch.name,
          email: patch.email as (string & { format: 'email' }) | undefined,
          phone: patch.phone ?? undefined,
          birthday: patch.birthday ?? undefined,
          address: patch.address ?? undefined,
          group: patch.group ?? undefined,
        },
      });
      const m = await check(res);
      return mapMember(m, teamId);
    },

    async setRoles(membershipId: string, roleIds: string[], teamId: string): Promise<Member> {
      const res = await apiClient.PUT('/teams/{teamId}/members/{membershipId}/roles', {
        params: { path: { teamId, membershipId } },
        body: { roleIds },
      });
      const m = await check(res);
      return mapMember(m, teamId);
    },

    async remove(membershipId: string, teamId: string): Promise<void> {
      const res = await apiClient.DELETE('/teams/{teamId}/members/{membershipId}', {
        params: { path: { teamId, membershipId } },
      });
      await checkOk(res);
    },
  },

  roles: {
    async list(teamId: string): Promise<Role[]> {
      const res = await apiClient.GET('/teams/{teamId}/roles', { params: { path: { teamId } } });
      const roles = await check(res);
      return roles.map(mapRole);
    },

    async create(
      teamId: string,
      payload: { name: string; color?: string; permissions: Role['permissions'] },
    ): Promise<Role> {
      const res = await apiClient.POST('/teams/{teamId}/roles', {
        params: { path: { teamId } },
        body: { name: payload.name, color: payload.color, permissions: payload.permissions },
      });
      const r = await check(res);
      return mapRole(r);
    },

    async update(roleId: string, patch: Partial<Role>, teamId: string): Promise<Role> {
      const res = await apiClient.PATCH('/teams/{teamId}/roles/{roleId}', {
        params: { path: { teamId, roleId } },
        body: { name: patch.name, color: patch.color, permissions: patch.permissions },
      });
      const r = await check(res);
      return mapRole(r);
    },

    async remove(roleId: string, teamId: string): Promise<void> {
      const res = await apiClient.DELETE('/teams/{teamId}/roles/{roleId}', {
        params: { path: { teamId, roleId } },
      });
      await checkOk(res);
    },
  },

  events: {
    async list(teamId: string, scope: 'all' | 'upcoming' | 'past' = 'all'): Promise<TeamEvent[]> {
      // Keyset { items, nextCursor } envelope; walked to completion.
      const items = await fetchAllPages(async (cursor) => {
        const res = await apiClient.GET('/teams/{teamId}/events', {
          params: { path: { teamId }, query: { scope, limit: PAGE_LIMIT, cursor } },
        });
        return check(res);
      });
      return items.map(mapTeamEvent);
    },

    async get(eventId: string, teamId: string): Promise<TeamEvent | null> {
      const res = await apiClient.GET('/teams/{teamId}/events/{eventId}', {
        params: { path: { teamId, eventId } },
      });
      if (res.response.status === 404) return null;
      const e = await check(res);
      return mapTeamEvent(e);
    },

    // `meetT`/`startT`/`endT` (not `meetTime`/`startTime`/`endTime`) are the
    // names the event form (useEventFormActions.ts) actually sends — they
    // hold "HH:mm" strings, which is also exactly the wire format the
    // backend's meetTime/startTime/endTime fields expect (see openapi.yaml),
    // so no conversion is needed, only the rename below. Reading
    // `payload.meetTime` here (as this used to) is always undefined given
    // the caller's actual payload shape, silently dropping every meet/start/
    // end time on event creation against the real backend.
    async create(
      teamId: string,
      payload: {
        type: string;
        title: string;
        date: string;
        location?: string;
        note?: string;
        meetT?: string | null;
        startT?: string | null;
        endT?: string | null;
        meetTimeMandatory?: boolean;
        responseMode?: string;
        nominatedRoleIds?: string[];
        recurring?: boolean;
        repeatWeeks?: number;
      },
    ): Promise<TeamEvent> {
      const res = await apiClient.POST('/teams/{teamId}/events', {
        params: { path: { teamId } },
        body: {
          type: payload.type as 'training' | 'auftritt' | 'event',
          title: payload.title,
          date: payload.date,
          location: payload.location ?? undefined,
          note: payload.note ?? undefined,
          meetTime: payload.meetT ?? undefined,
          startTime: payload.startT ?? undefined,
          endTime: payload.endT ?? undefined,
          meetTimeMandatory: payload.meetTimeMandatory,
          responseMode: payload.responseMode as 'opt_in' | 'opt_out' | undefined,
          nominatedRoleIds: payload.nominatedRoleIds,
          recurring: payload.recurring,
          repeatWeeks: payload.repeatWeeks,
        },
      });
      // Backend may return an array for series
      const data = res.data;
      if (!data) {
        const err = res.error as { detail?: string; title?: string } | undefined;
        throw errorFor(res.response.status, err);
      }
      const first = Array.isArray(data) ? data[0] : data;
      return mapTeamEvent(first);
    },

    // Same `meetT`/`startT`/`endT` -> `meetTime`/`startTime`/`endTime` rename
    // as create() above. This used to forward the raw `patch` object as the
    // request body (cast past the generated client's body type), so it sent
    // unrecognised `meetT`/`startT`/`endT` JSON keys that the backend's
    // plain (non-DisallowUnknownFields) json.Decode silently ignores — the
    // PATCH still returns 200 and the UI still shows a success toast, but
    // edited meet/start/end times never persisted.
    async update(
      eventId: string,
      patch: {
        type?: string;
        title?: string;
        date?: string;
        location?: string;
        note?: string;
        meetT?: string | null;
        startT?: string | null;
        endT?: string | null;
        meetTimeMandatory?: boolean;
        responseMode?: string;
        nominatedRoleIds?: string[];
      },
      scope: 'single' | 'series',
      teamId: string,
    ): Promise<TeamEvent> {
      const res = await apiClient.PATCH('/teams/{teamId}/events/{eventId}', {
        params: { path: { teamId, eventId }, query: { scope } },
        body: {
          type: patch.type as 'training' | 'auftritt' | 'event' | undefined,
          title: patch.title,
          date: patch.date,
          location: patch.location,
          note: patch.note,
          meetTime: patch.meetT ?? undefined,
          startTime: patch.startT ?? undefined,
          endTime: patch.endT ?? undefined,
          meetTimeMandatory: patch.meetTimeMandatory,
          responseMode: patch.responseMode as 'opt_in' | 'opt_out' | undefined,
          nominatedRoleIds: patch.nominatedRoleIds,
        },
      });
      const e = await check(res);
      return mapTeamEvent(e);
    },

    async setStatus(
      eventId: string,
      status: 'active' | 'cancelled',
      scope: 'single' | 'series',
      teamId: string,
    ): Promise<TeamEvent> {
      const res = await apiClient.POST('/teams/{teamId}/events/{eventId}/status', {
        params: { path: { teamId, eventId }, query: { scope } },
        body: { status },
      });
      const e = await check(res);
      return mapTeamEvent(e);
    },

    async remove(eventId: string, scope: 'single' | 'series', teamId: string): Promise<void> {
      const res = await apiClient.DELETE('/teams/{teamId}/events/{eventId}', {
        params: { path: { teamId, eventId }, query: { scope } },
      });
      await checkOk(res);
    },

    async listComments(eventId: string, teamId: string): Promise<EventComment[]> {
      // limit/offset paginated (see fetchAllOffsetPages doc comment above) —
      // walked to completion so events with more than one page of comments
      // (default limit 50) don't silently lose their oldest ones.
      const comments = await fetchAllOffsetPages((limit, offset) =>
        apiClient
          .GET('/teams/{teamId}/events/{eventId}/comments', {
            params: { path: { teamId, eventId }, query: { limit, offset } },
          })
          .then(check),
      );
      return comments.map(mapEventComment);
    },

    async addComment(eventId: string, text: string, teamId: string): Promise<EventComment> {
      const res = await apiClient.POST('/teams/{teamId}/events/{eventId}/comments', {
        params: { path: { teamId, eventId } },
        body: { text },
      });
      const c = await check(res);
      return mapEventComment(c);
    },

    async removeComment(commentId: string, eventId: string, teamId: string): Promise<void> {
      const res = await apiClient.DELETE('/teams/{teamId}/events/{eventId}/comments/{commentId}', {
        params: { path: { teamId, eventId, commentId } },
      });
      await checkOk(res);
    },
  },

  attendance: {
    async listForEvent(eventId: string, teamId: string): Promise<AttendanceRow[]> {
      const res = await apiClient.GET('/teams/{teamId}/events/{eventId}/attendance', {
        params: { path: { teamId, eventId } },
      });
      const rows = await check(res);
      // events.Repository.ListAttendance orders `ORDER BY u.name ASC` only;
      // the mock additionally groups rows by response status first (yes,
      // maybe, pending, no, not_nominated — see serviceLayer.ts's
      // STATUS_ORDER) before falling back to name. EventDetailSheet.tsx
      // renders `sheet.rows` in the order it receives them (no client-side
      // re-sort), so without this the participant list order silently
      // differed depending on which backend served the request.
      const sorted = [...rows].sort(
        (a, b) =>
          ATTENDANCE_STATUS_ORDER[a.status] - ATTENDANCE_STATUS_ORDER[b.status] || a.name.localeCompare(b.name, 'de'),
      );
      return sorted.map(mapAttendanceRow);
    },

    async set(
      eventId: string,
      userId: string,
      payload: { status: string; reason?: string; reasonId?: string | null; reasonVisibility?: string | null },
      teamId: string,
    ) {
      const res = await apiClient.POST('/teams/{teamId}/events/{eventId}/attendance', {
        params: { path: { teamId, eventId } },
        body: {
          userId,
          status: payload.status as 'yes' | 'no' | 'maybe' | 'pending' | 'not_nominated',
          reason: payload.reason,
          reasonId: payload.reasonId ?? undefined,
          reasonVisibility: payload.reasonVisibility as 'trainers' | 'team' | undefined,
        },
      });
      return check(res);
    },

    async setNomination(eventId: string, userId: string, nominated: boolean, teamId: string): Promise<boolean> {
      const res = await apiClient.PUT('/teams/{teamId}/events/{eventId}/attendance/nominations', {
        params: { path: { teamId, eventId } },
        body: { userId, nominated },
      });
      await checkOk(res);
      return true;
    },
  },

  absences: {
    async listForTeam(teamId: string): Promise<Absence[]> {
      const items = await fetchAllPages(async (cursor) => {
        const res = await apiClient.GET('/teams/{teamId}/absences', {
          params: { path: { teamId }, query: { limit: PAGE_LIMIT, cursor } },
        });
        return check(res);
      });
      // absences.Repository orders `from_date DESC, id DESC` (newest-starting
      // first); the mock sorts ascending by `from` so the soonest-upcoming
      // absence appears first. EventAbsences.tsx renders the list in the
      // order it receives (no client-side sort), so re-sort ascending here to
      // match the mock's convention.
      return [...items].sort((a, b) => a.from.localeCompare(b.from)).map(mapAbsence);
    },

    async listMine(teamId: string): Promise<Absence[]> {
      const items = await fetchAllPages(async (cursor) => {
        const res = await apiClient.GET('/teams/{teamId}/absences/mine', {
          params: { path: { teamId }, query: { limit: PAGE_LIMIT, cursor } },
        });
        return check(res);
      });
      return [...items].sort((a, b) => a.from.localeCompare(b.from)).map(mapAbsence);
    },

    async create(payload: {
      teamId: string;
      userId: string;
      from: string;
      to: string;
      reason?: string;
    }): Promise<Absence> {
      const res = await apiClient.POST('/teams/{teamId}/absences', {
        params: { path: { teamId: payload.teamId } },
        body: {
          userId: payload.userId,
          from: payload.from,
          to: payload.to,
          reason: payload.reason,
        },
      });
      const a = await check(res);
      return mapAbsence(a);
    },

    async update(
      absenceId: string,
      patch: { from?: string; to?: string; reason?: string },
      teamId: string,
    ): Promise<Absence> {
      const res = await apiClient.PATCH('/teams/{teamId}/absences/{absenceId}', {
        params: { path: { teamId, absenceId } },
        body: patch,
      });
      const a = await check(res);
      return mapAbsence(a);
    },

    async remove(absenceId: string, teamId: string): Promise<void> {
      const res = await apiClient.DELETE('/teams/{teamId}/absences/{absenceId}', {
        params: { path: { teamId, absenceId } },
      });
      await checkOk(res);
    },
  },

  news: {
    async list(teamId: string): Promise<NewsItem[]> {
      // Keyset { items, nextCursor } envelope; walked to completion.
      const items = await fetchAllPages(async (cursor) => {
        const res = await apiClient.GET('/teams/{teamId}/news', {
          params: { path: { teamId }, query: { limit: PAGE_LIMIT, cursor } },
        });
        return check(res);
      });
      return items.map(mapNewsItem);
    },

    async create(teamId: string, payload: { title: string; body: string; pinned?: boolean }): Promise<NewsItem> {
      const res = await apiClient.POST('/teams/{teamId}/news', {
        params: { path: { teamId } },
        body: { title: payload.title, body: payload.body, pinned: payload.pinned ?? false },
      });
      const n = await check(res);
      return mapNewsItem(n);
    },

    async update(
      id: string,
      patch: { title?: string; body?: string; pinned?: boolean },
      teamId: string,
    ): Promise<NewsItem> {
      const res = await apiClient.PATCH('/teams/{teamId}/news/{newsId}', {
        params: { path: { teamId, newsId: id } },
        body: patch,
      });
      const n = await check(res);
      return mapNewsItem(n);
    },

    async remove(id: string, teamId: string): Promise<void> {
      const res = await apiClient.DELETE('/teams/{teamId}/news/{newsId}', {
        params: { path: { teamId, newsId: id } },
      });
      await checkOk(res);
    },
  },

  polls: {
    async list(teamId: string): Promise<Poll[]> {
      // Keyset { items, nextCursor } envelope; walked to completion.
      const items = await fetchAllPages(async (cursor) => {
        const res = await apiClient.GET('/teams/{teamId}/polls', {
          params: { path: { teamId }, query: { limit: PAGE_LIMIT, cursor } },
        });
        return check(res);
      });
      return items.map(mapPoll);
    },

    async vote(pollId: string, optionIds: string[], teamId: string): Promise<void> {
      const res = await apiClient.POST('/teams/{teamId}/polls/{pollId}/vote', {
        params: { path: { teamId, pollId } },
        body: { optionIds },
      });
      await checkOk(res);
    },

    async create(
      teamId: string,
      payload: {
        question: string;
        options: string[];
        multiple?: boolean;
        anonymous?: boolean;
      },
    ): Promise<Poll> {
      const res = await apiClient.POST('/teams/{teamId}/polls', {
        params: { path: { teamId } },
        body: {
          question: payload.question,
          options: payload.options,
          multiple: payload.multiple ?? false,
          anonymous: payload.anonymous ?? false,
        },
      });
      const p = await check(res);
      return mapPoll(p);
    },

    async remove(pollId: string, teamId: string): Promise<void> {
      const res = await apiClient.DELETE('/teams/{teamId}/polls/{pollId}', {
        params: { path: { teamId, pollId } },
      });
      await checkOk(res);
    },
  },

  finances: {
    async overview(teamId: string): Promise<FinanceOverview> {
      const res = await apiClient.GET('/teams/{teamId}/finances', { params: { path: { teamId } } });
      const o = await check(res);
      // finances.Repository.ListAssignments orders `pa.date DESC` (newest
      // first); the mock's array is in ascending insertion order (oldest
      // first). FinancesPenalties.tsx renders `f.assignments.slice().reverse()`
      // — which only produces newest-first if the input was ascending — so
      // passing the backend's already-descending order through unchanged
      // gets reversed a second time and shows the oldest assignment on top.
      // Reverse here so both service layers hand the UI the same (ascending)
      // convention.
      return mapFinanceOverview({ ...o, assignments: [...o.assignments].reverse() });
    },

    async addTransaction(
      teamId: string,
      payload: {
        type: 'income' | 'expense';
        title: string;
        amount: number;
        category?: string;
        date?: string;
      },
    ): Promise<Transaction> {
      const res = await apiClient.POST('/teams/{teamId}/finances/transactions', {
        params: { path: { teamId } },
        body: {
          type: payload.type,
          title: payload.title,
          amount: eurosToCents(payload.amount),
          category: payload.category,
        },
      });
      const t = await check(res);
      return mapTransaction(t);
    },

    async updateTransaction(id: string, patch: Partial<Transaction>, teamId: string): Promise<Transaction> {
      const res = await apiClient.PATCH('/teams/{teamId}/finances/transactions/{transactionId}', {
        params: { path: { teamId, transactionId: id } },
        body: {
          type: patch.type,
          title: patch.title,
          amount: patch.amount == null ? patch.amount : eurosToCents(patch.amount),
          category: patch.category,
        },
      });
      const t = await check(res);
      return mapTransaction(t);
    },

    async deleteTransaction(id: string, teamId: string): Promise<void> {
      const res = await apiClient.DELETE('/teams/{teamId}/finances/transactions/{transactionId}', {
        params: { path: { teamId, transactionId: id } },
      });
      await checkOk(res);
    },

    async createPenalty(teamId: string, payload: { label: string; amount: number }): Promise<Penalty> {
      const res = await apiClient.POST('/teams/{teamId}/finances/penalties', {
        params: { path: { teamId } },
        body: { label: payload.label, amount: eurosToCents(payload.amount) },
      });
      const p = await check(res);
      return mapPenalty(p);
    },

    async updatePenalty(id: string, patch: { label?: string; amount?: number }, teamId: string): Promise<Penalty> {
      const res = await apiClient.PATCH('/teams/{teamId}/finances/penalties/{penaltyId}', {
        params: { path: { teamId, penaltyId: id } },
        body: {
          label: patch.label,
          amount: patch.amount == null ? patch.amount : eurosToCents(patch.amount),
        },
      });
      const p = await check(res);
      return mapPenalty(p);
    },

    async deletePenalty(id: string, teamId: string): Promise<void> {
      const res = await apiClient.DELETE('/teams/{teamId}/finances/penalties/{penaltyId}', {
        params: { path: { teamId, penaltyId: id } },
      });
      await checkOk(res);
    },

    async assignPenalty(
      teamId: string,
      { userId, penaltyId }: { userId: string; penaltyId: string },
    ): Promise<PenaltyAssignment> {
      const res = await apiClient.POST('/teams/{teamId}/finances/penalty-assignments', {
        params: { path: { teamId } },
        body: { userId, penaltyId },
      });
      const a = await check(res);
      return mapPenaltyAssignment(a);
    },

    // Named to match the mock + the app's call site (api.finances.deleteAssignment).
    async deleteAssignment(id: string, teamId: string): Promise<void> {
      const res = await apiClient.DELETE('/teams/{teamId}/finances/penalty-assignments/{assignmentId}', {
        params: { path: { teamId, assignmentId: id } },
      });
      await checkOk(res);
    },

    async togglePenaltyPaid(id: string, teamId: string): Promise<PenaltyAssignment> {
      const res = await apiClient.POST('/teams/{teamId}/finances/penalty-assignments/{assignmentId}/toggle-paid', {
        params: { path: { teamId, assignmentId: id } },
      });
      const a = await check(res);
      return mapPenaltyAssignment(a);
    },

    async updateContribution(
      id: string,
      patch: { label?: string; amount?: number },
      teamId: string,
    ): Promise<Contribution> {
      const res = await apiClient.PATCH('/teams/{teamId}/finances/contributions/{contributionId}', {
        params: { path: { teamId, contributionId: id } },
        body: {
          label: patch.label,
          amount: patch.amount == null ? patch.amount : eurosToCents(patch.amount),
        },
      });
      const c = await check(res);
      return mapContribution(c);
    },

    async toggleContribution(id: string, teamId: string): Promise<Contribution> {
      const res = await apiClient.POST('/teams/{teamId}/finances/contributions/{contributionId}/toggle', {
        params: { path: { teamId, contributionId: id } },
      });
      const c = await check(res);
      return mapContribution(c);
    },
  },

  stats: {
    async attendanceFor(
      teamId: string,
      userId: string,
    ): Promise<{ quote: number | null; counted: number; yes: number }> {
      const res = await apiClient.GET('/teams/{teamId}/stats/members/{userId}', {
        params: { path: { teamId, userId } },
      });
      const s = await check(res);
      // s.quote is a 0-1 fraction (see api/map.ts's fractionToPercent doc
      // comment); every UI consumer expects a 0-100 percentage. The backend
      // always returns a number (0 when counted is 0 — see
      // internal/stats/service.go's quote()), but MemberSheets.tsx renders
      // null as "no data" (–) vs. 0 as "0% attendance" (red) — genuinely
      // different states for a member with no counted events yet vs. one who
      // attended none of several. Map counted === 0 to null here so both
      // service layers agree on that distinction.
      return { quote: s.counted > 0 ? Math.round(s.quote * 100) : null, counted: s.counted, yes: s.yes };
    },

    async teamOverview(teamId: string, range?: DateRange | null): Promise<StatsOverview> {
      const res = await apiClient.GET('/teams/{teamId}/stats', {
        params: { path: { teamId }, query: { from: range?.from ?? undefined, to: range?.to ?? undefined } },
      });
      const o = await check(res);
      return mapStatsOverview(o);
    },
  },

  notifications: {
    async list(teamId: string): Promise<NotificationsResult> {
      const res = await apiClient.GET('/teams/{teamId}/notifications', { params: { path: { teamId } } });
      const r = await check(res);
      return mapNotificationsResult(r);
    },

    async markSeen(teamId: string): Promise<boolean> {
      const res = await apiClient.POST('/teams/{teamId}/notifications/seen', {
        params: { path: { teamId } },
      });
      await checkOk(res);
      return true;
    },
  },

  MODULES: ['events', 'members', 'finances', 'news', 'polls', 'settings'] as const,
};

export type RealApi = typeof realApi;
