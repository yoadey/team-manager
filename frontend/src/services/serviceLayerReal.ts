// Real backend service layer — replaces localStorage mock with HTTP API calls.
// Only activated when VITE_API_BASE_URL is set.

import { apiClient } from '@/api/client';
import {
  mapUser,
  mapProvider,
  mapRole,
  mapTeam,
  mapTeamForUser,
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
} from '@/api/map';
import type { User, Team, TeamForUser, Role, Invite, Provider, DateRange, StatsOverview } from '@/types';
import type { TeamEvent, AttendanceRow, EventComment, Absence } from '@/features/events';
import type { Member } from '@/features/members';
import type { NewsItem } from '@/features/news';
import type { Poll } from '@/features/polls';
import type { NotificationsResult } from '@/features/notifications';
import type { FinanceOverview, Transaction, Penalty, PenaltyAssignment, Contribution } from '@/features/finances';
import { AuthError, NetworkError, ValidationError } from '@/utils/errors';

// Throws a typed error when the API returns an error response, so callers can
// react to the failure class — notably AuthError (401/403), which the app's
// reportActionError/onAuthError wiring turns into a logout + redirect to the
// login screen when a session expires mid-use.
async function check<T>(
  result: { data?: T; error?: unknown; response: Response },
): Promise<T> {
  if (result.error || !result.data) {
    const status = result.response.status;
    const err = result.error as { detail?: string; title?: string } | undefined;
    const msg = err?.detail ?? err?.title ?? `HTTP ${status}`;
    if (status === 401 || status === 403) throw new AuthError(msg);
    if (status === 400 || status === 422) throw new ValidationError(msg);
    if (status >= 500) throw new NetworkError(msg);
    throw new Error(msg);
  }
  return result.data;
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
      await apiClient.POST('/auth/logout', {});
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
      if (!res.response.ok) throw new Error(`HTTP ${res.response.status}`);
    },

    async setPhoto(dataUrl: string): Promise<User> {
      // Convert data URL to Blob for multipart upload.
      const arr = dataUrl.split(',');
      const mimeMatch = arr[0].match(/:(.*?);/);
      if (!mimeMatch || arr.length < 2) throw new Error('Invalid data URL format');
      const mime = mimeMatch[1];
      const bstr = atob(arr[1]);
      const bytes = new Uint8Array(bstr.length);
      for (let i = 0; i < bstr.length; i++) bytes[i] = bstr.charCodeAt(i);
      const blob = new Blob([bytes], { type: mime });
      const formData = new FormData();
      formData.append('photo', blob, 'photo.jpg');

      const resp = await fetch(
        (import.meta.env.VITE_API_BASE_URL ?? '') + '/api/v1/auth/me/photo',
        { method: 'POST', credentials: 'include', body: formData },
      );
      if (!resp.ok) throw new Error('Photo upload failed');
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

    async create(opts: { name: string; icon?: string; iconBg?: string; iconFg?: string; photo?: string | null }): Promise<Team> {
      const res = await apiClient.POST('/teams', {
        body: { name: opts.name, icon: opts.icon, iconBg: opts.iconBg, iconFg: opts.iconFg },
      });
      const t = await check(res);
      return mapTeam(t);
    },

    async updateSettings(teamId: string, patch: Partial<Team>): Promise<Team> {
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
      const t = await check(res);
      return mapTeam(t);
    },

    async createInvite(teamId: string): Promise<Invite> {
      const res = await apiClient.POST('/teams/{teamId}/invite', { params: { path: { teamId } } });
      const inv = await check(res);
      return mapInvite(inv);
    },
  },

  members: {
    async list(teamId: string): Promise<Member[]> {
      const res = await apiClient.GET('/teams/{teamId}/members', { params: { path: { teamId } } });
      const members = await check(res);
      return (members as unknown[]).map((m) => mapMember(m as Parameters<typeof mapMember>[0]));
    },

    async add(teamId: string, params: {
      name: string; email: string; phone?: string | null; group?: string | null; roleIDs?: string[];
    }): Promise<Member> {
      const res = await apiClient.POST('/teams/{teamId}/members', {
        params: { path: { teamId } },
        body: {
          name: params.name,
          email: params.email as string & { format: 'email' },
          phone: params.phone ?? undefined,
          group: params.group ?? undefined,
          roleIds: params.roleIDs,
        },
      });
      const m = await check(res);
      return mapMember(m);
    },

    async update(membershipId: string, patch: {
      name?: string; email?: string; phone?: string | null; birthday?: string | null;
      address?: string | null; group?: string | null; roleIds?: string[];
    }, teamId: string): Promise<Member> {
      const res = await apiClient.PATCH('/teams/{teamId}/members/{membershipId}', {
        params: { path: { teamId, membershipId } },
        body: {
          name: patch.name,
          email: patch.email as (string & { format: 'email' }) | undefined,
          phone: patch.phone ?? undefined,
          birthday: patch.birthday ?? undefined,
          address: patch.address ?? undefined,
          group: patch.group ?? undefined,
          roleIds: patch.roleIds,
        },
      });
      const m = await check(res);
      return mapMember(m);
    },

    async setRoles(membershipId: string, roleIds: string[], teamId: string): Promise<Member> {
      const res = await apiClient.PUT('/teams/{teamId}/members/{membershipId}/roles', {
        params: { path: { teamId, membershipId } },
        body: { roleIds },
      });
      const m = await check(res);
      return mapMember(m);
    },

    async remove(membershipId: string, teamId: string): Promise<void> {
      const res = await apiClient.DELETE('/teams/{teamId}/members/{membershipId}', {
        params: { path: { teamId, membershipId } },
      });
      if (!res.response.ok) throw new Error(`HTTP ${res.response.status}`);
    },
  },

  roles: {
    async list(teamId: string): Promise<Role[]> {
      const res = await apiClient.GET('/teams/{teamId}/roles', { params: { path: { teamId } } });
      const roles = await check(res);
      return roles.map(mapRole);
    },

    async create(teamId: string, payload: { name: string; color?: string; permissions: Role['permissions'] }): Promise<Role> {
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
      if (!res.response.ok) throw new Error(`HTTP ${res.response.status}`);
    },
  },

  events: {
    async list(teamId: string, scope: 'all' | 'upcoming' | 'past' = 'all'): Promise<TeamEvent[]> {
      const res = await apiClient.GET('/teams/{teamId}/events', {
        params: { path: { teamId }, query: { scope } },
      });
      const events = await check(res);
      return events.map(mapTeamEvent);
    },

    async get(eventId: string, teamId: string): Promise<TeamEvent | null> {
      const res = await apiClient.GET('/teams/{teamId}/events/{eventId}', {
        params: { path: { teamId, eventId } },
      });
      if (res.response.status === 404) return null;
      const e = await check(res);
      return mapTeamEvent(e);
    },

    async create(teamId: string, payload: {
      type: string; title: string; date: string; location?: string; note?: string;
      meetTime?: string | null; startTime?: string | null; endTime?: string | null;
      meetTimeMandatory?: boolean; responseMode?: string; nominatedRoleIds?: string[];
      recurring?: boolean; repeatWeeks?: number;
    }): Promise<TeamEvent> {
      const res = await apiClient.POST('/teams/{teamId}/events', {
        params: { path: { teamId } },
        body: {
          type: payload.type as 'training' | 'auftritt' | 'event',
          title: payload.title,
          date: payload.date,
          location: payload.location ?? undefined,
          note: payload.note ?? undefined,
          meetTime: payload.meetTime ?? undefined,
          startTime: payload.startTime ?? undefined,
          endTime: payload.endTime ?? undefined,
          meetTimeMandatory: payload.meetTimeMandatory,
          responseMode: payload.responseMode as 'opt_in' | 'opt_out' | undefined,
          nominatedRoleIds: payload.nominatedRoleIds,
          recurring: payload.recurring,
          repeatWeeks: payload.repeatWeeks,
        },
      });
      // Backend may return an array for series
      const data = res.data;
      if (!data) throw new Error(`HTTP ${res.response.status}`);
      const first = Array.isArray(data) ? data[0] : data;
      return mapTeamEvent(first);
    },

    async update(eventId: string, patch: Record<string, unknown>, scope: 'single' | 'series', teamId: string): Promise<TeamEvent> {
      const res = await apiClient.PATCH('/teams/{teamId}/events/{eventId}', {
        params: { path: { teamId, eventId }, query: { scope } },
        body: patch as Parameters<typeof apiClient.PATCH>[1]['body'],
      });
      const e = await check(res);
      return mapTeamEvent(e);
    },

    async setStatus(eventId: string, status: 'active' | 'cancelled', scope: 'single' | 'series', teamId: string): Promise<TeamEvent> {
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
      if (!res.response.ok) throw new Error(`HTTP ${res.response.status}`);
    },

    async listComments(eventId: string, teamId: string): Promise<EventComment[]> {
      const res = await apiClient.GET('/teams/{teamId}/events/{eventId}/comments', {
        params: { path: { teamId, eventId } },
      });
      const comments = await check(res);
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
      if (!res.response.ok) throw new Error(`HTTP ${res.response.status}`);
    },
  },

  attendance: {
    async listForEvent(eventId: string, teamId: string): Promise<AttendanceRow[]> {
      const res = await apiClient.GET('/teams/{teamId}/events/{eventId}/attendance', {
        params: { path: { teamId, eventId } },
      });
      const rows = await check(res);
      return rows.map(mapAttendanceRow);
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
      if (!res.response.ok) throw new Error(`HTTP ${res.response.status}`);
      return true;
    },
  },

  absences: {
    async listForTeam(teamId: string): Promise<Absence[]> {
      const res = await apiClient.GET('/teams/{teamId}/absences', { params: { path: { teamId } } });
      const absences = await check(res);
      return absences.map(mapAbsence);
    },

    async listMine(teamId: string): Promise<Absence[]> {
      const res = await apiClient.GET('/teams/{teamId}/absences/mine', { params: { path: { teamId } } });
      const absences = await check(res);
      return absences.map(mapAbsence);
    },

    async create(payload: { teamId: string; userId: string; from: string; to: string; reason?: string }): Promise<Absence> {
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

    async update(absenceId: string, patch: { from?: string; to?: string; reason?: string }, teamId: string): Promise<Absence> {
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
      if (!res.response.ok) throw new Error(`HTTP ${res.response.status}`);
    },
  },

  news: {
    async list(teamId: string): Promise<NewsItem[]> {
      const res = await apiClient.GET('/teams/{teamId}/news', { params: { path: { teamId } } });
      const news = await check(res);
      return news.map(mapNewsItem);
    },

    async create(teamId: string, payload: { title: string; body: string; pinned?: boolean }): Promise<NewsItem> {
      const res = await apiClient.POST('/teams/{teamId}/news', {
        params: { path: { teamId } },
        body: { title: payload.title, body: payload.body, pinned: payload.pinned ?? false },
      });
      const n = await check(res);
      return mapNewsItem(n);
    },

    async update(id: string, patch: { title?: string; body?: string; pinned?: boolean }, teamId: string): Promise<NewsItem> {
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
      if (!res.response.ok) throw new Error(`HTTP ${res.response.status}`);
    },
  },

  polls: {
    async list(teamId: string): Promise<Poll[]> {
      const res = await apiClient.GET('/teams/{teamId}/polls', { params: { path: { teamId } } });
      const polls = await check(res);
      return polls.map(mapPoll);
    },

    async vote(pollId: string, optionIds: string[], teamId: string): Promise<void> {
      const res = await apiClient.POST('/teams/{teamId}/polls/{pollId}/vote', {
        params: { path: { teamId, pollId } },
        body: { optionIds },
      });
      if (!res.response.ok) throw new Error(`HTTP ${res.response.status}`);
    },

    async create(teamId: string, payload: {
      question: string; options: string[]; multiple?: boolean; anonymous?: boolean;
    }): Promise<Poll> {
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
      if (!res.response.ok) throw new Error(`HTTP ${res.response.status}`);
    },
  },

  finances: {
    async overview(teamId: string): Promise<FinanceOverview> {
      const res = await apiClient.GET('/teams/{teamId}/finances', { params: { path: { teamId } } });
      const o = await check(res);
      return mapFinanceOverview(o);
    },

    async addTransaction(teamId: string, payload: {
      type: 'income' | 'expense'; title: string; amount: number; category?: string; date?: string;
    }): Promise<Transaction> {
      const res = await apiClient.POST('/teams/{teamId}/finances/transactions', {
        params: { path: { teamId } },
        body: { type: payload.type, title: payload.title, amount: payload.amount, category: payload.category },
      });
      const t = await check(res);
      return mapTransaction(t);
    },

    async updateTransaction(id: string, patch: Partial<Transaction>, teamId: string): Promise<Transaction> {
      const res = await apiClient.PATCH('/teams/{teamId}/finances/transactions/{transactionId}', {
        params: { path: { teamId, transactionId: id } },
        body: { type: patch.type, title: patch.title, amount: patch.amount, category: patch.category },
      });
      const t = await check(res);
      return mapTransaction(t);
    },

    async deleteTransaction(id: string, teamId: string): Promise<void> {
      const res = await apiClient.DELETE('/teams/{teamId}/finances/transactions/{transactionId}', {
        params: { path: { teamId, transactionId: id } },
      });
      if (!res.response.ok) throw new Error(`HTTP ${res.response.status}`);
    },

    async createPenalty(teamId: string, payload: { label: string; amount: number }): Promise<Penalty> {
      const res = await apiClient.POST('/teams/{teamId}/finances/penalties', {
        params: { path: { teamId } },
        body: { label: payload.label, amount: payload.amount },
      });
      const p = await check(res);
      return mapPenalty(p);
    },

    async updatePenalty(id: string, patch: { label?: string; amount?: number }, teamId: string): Promise<Penalty> {
      const res = await apiClient.PATCH('/teams/{teamId}/finances/penalties/{penaltyId}', {
        params: { path: { teamId, penaltyId: id } },
        body: patch,
      });
      const p = await check(res);
      return mapPenalty(p);
    },

    async deletePenalty(id: string, teamId: string): Promise<void> {
      const res = await apiClient.DELETE('/teams/{teamId}/finances/penalties/{penaltyId}', {
        params: { path: { teamId, penaltyId: id } },
      });
      if (!res.response.ok) throw new Error(`HTTP ${res.response.status}`);
    },

    async assignPenalty(teamId: string, userId: string, penaltyId: string): Promise<PenaltyAssignment> {
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
      if (!res.response.ok) throw new Error(`HTTP ${res.response.status}`);
    },

    async togglePenaltyPaid(id: string, teamId: string): Promise<PenaltyAssignment> {
      const res = await apiClient.POST('/teams/{teamId}/finances/penalty-assignments/{assignmentId}/toggle-paid', {
        params: { path: { teamId, assignmentId: id } },
      });
      const a = await check(res);
      return mapPenaltyAssignment(a);
    },

    async updateContribution(id: string, patch: { label?: string; amount?: number }, teamId: string): Promise<Contribution> {
      const res = await apiClient.PATCH('/teams/{teamId}/finances/contributions/{contributionId}', {
        params: { path: { teamId, contributionId: id } },
        body: patch,
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
    async attendanceFor(teamId: string, userId: string): Promise<{ quote: number | null; counted: number; yes: number }> {
      const res = await apiClient.GET('/teams/{teamId}/stats/members/{userId}', {
        params: { path: { teamId, userId } },
      });
      const s = await check(res);
      return { quote: s.quote, counted: s.counted, yes: s.yes };
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
      if (!res.response.ok) throw new Error(`HTTP ${res.response.status}`);
      return true;
    },
  },

  MODULES: ['events', 'members', 'finances', 'news', 'polls', 'settings'] as const,
};

export type RealApi = typeof realApi;
