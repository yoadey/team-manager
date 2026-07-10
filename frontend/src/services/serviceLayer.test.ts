import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { ALL_TIME_FROM_DATE, todayLocalDate } from '@/utils/date';
import { config } from '@/config';

/**
 * The service layer keeps an in-memory `DB` singleton that is seeded once at
 * module-evaluation time and persisted to localStorage. To guarantee test
 * isolation we reset the module registry and re-import a pristine instance
 * before every test (the global setup clears localStorage beforehand, forcing
 * a fresh seed). Artificial network latency is implemented with setTimeout, so
 * we fake only the timer functions and fast-forward them on demand — real Date
 * is kept intact so the seed's "today relative" event logic stays realistic.
 */

type ServiceModule = typeof import('./serviceLayer');

let mod: ServiceModule;
let api: ServiceModule['api'];

/** Fast-forwards the faked latency timer and resolves the pending service call. */
async function settle<T>(promise: Promise<T>): Promise<T> {
  await vi.runAllTimersAsync();
  return promise;
}

/**
 * Asserts that a latency-delayed service call rejects. The rejection handler is
 * attached before the timers are advanced so the eventual rejection is never
 * reported as an unhandled promise rejection.
 */
async function expectRejection(promise: Promise<unknown>): Promise<void> {
  const assertion = expect(promise).rejects.toThrow();
  await vi.runAllTimersAsync();
  await assertion;
}

beforeEach(async () => {
  vi.useFakeTimers({ toFake: ['setTimeout', 'clearTimeout', 'setInterval', 'clearInterval'] });
  vi.resetModules();
  mod = await import('./serviceLayer');
  api = mod.api;
});

afterEach(() => {
  vi.useRealTimers();
});

/** Logs in as the seeded demo user (u1) and returns once the session is active. */
async function login(): Promise<void> {
  await settle(api.auth.login('google'));
}

describe('auth', () => {
  it('exposes the configured identity providers', async () => {
    const providers = await settle(api.auth.providers());
    expect(providers.map((p) => p.id)).toContain('google');
    expect(providers.length).toBeGreaterThan(0);
  });

  it('returns no current user before login', async () => {
    expect(await settle(api.auth.currentUser())).toBeNull();
  });

  it('establishes a session on login and resolves the current user', async () => {
    const session = await settle(api.auth.login('vereins-sso'));
    expect(session.token).toMatch(/^mock\.jwt\./);
    expect(session.provider).toBe('vereins-sso');

    const user = await settle(api.auth.currentUser());
    expect(user?.id).toBe('u1');
  });

  it('clears the session on logout', async () => {
    await login();
    api.auth.logout();
    expect(await settle(api.auth.currentUser())).toBeNull();
  });

  it('persists an uploaded profile photo on the current user', async () => {
    await login();
    const updated = await settle(api.auth.setPhoto('data:image/png;base64,AAAA'));
    expect(updated.photo).toBe('data:image/png;base64,AAAA');
  });

  it('exportData returns the current user profile and memberships', async () => {
    await login();
    const data = (await settle(api.auth.exportData())) as {
      profile: { id: string } | null;
      memberships: unknown[];
    };
    expect(data.profile?.id).toBe('u1');
    expect(Array.isArray(data.memberships)).toBe(true);
  });

  it('anonymizes the account and ends the session on deleteAccount', async () => {
    await login();
    const before = await settle(api.auth.currentUser());
    const id = before!.id;

    await settle(api.auth.deleteAccount('any-password'));

    // Session ended.
    expect(await settle(api.auth.currentUser())).toBeNull();

    // Re-login resolves the same record, now anonymized (PII overwritten).
    await login();
    const after = await settle(api.auth.currentUser());
    expect(after?.id).toBe(id);
    expect(after?.name).toBe('Gelöschtes Mitglied');
    expect(after?.email).toBe(`deleted+${id}@invalid`);
    expect(after?.photo).toBeNull();
  });
});

describe('teams', () => {
  beforeEach(login);

  it('lists the current user teams enriched with merged permissions', async () => {
    const teams = await settle(api.teams.listForCurrentUser());
    const teamA = teams.find((t) => t.id === 't_a');
    expect(teamA).toBeDefined();
    // u1 is Admin/Trainer in team A, so the merged permission for every module is "write".
    expect(teamA?.myPerms.finances).toBe('write');
    expect(teamA?.memberCount).toBe(12);
  });

  it('creates a team and makes the creator an admin member', async () => {
    const team = await settle(api.teams.create({ name: 'C-Team' }));
    expect(team.name).toBe('C-Team');
    expect(team.short).toBe('C');

    const teams = await settle(api.teams.listForCurrentUser());
    const created = teams.find((t) => t.id === team.id);
    expect(created?.myPerms.settings).toBe('write');
  });

  it('updates team settings', async () => {
    const updated = await settle(api.teams.updateSettings('t_a', { description: 'Neu' }));
    expect(updated.description).toBe('Neu');
  });

  it('creates an invite with a shareable link', async () => {
    const invite = await settle(api.teams.createInvite('t_a'));
    expect(invite.code).toHaveLength(6);
    expect(invite.link).toContain(invite.code);
    expect(invite.link).toContain('/join/t_a/' + invite.code);
  });

  it('acceptInvite is idempotent for a code redeemed by an already-existing member', async () => {
    const invite = await settle(api.teams.createInvite('t_a'));
    const before = await settle(api.members.list('t_a'));

    const joined = await settle(api.teams.acceptInvite(invite.code));
    expect(joined.id).toBe('t_a');

    const after = await settle(api.members.list('t_a'));
    expect(after).toHaveLength(before.length);
  });

  it('acceptInvite rejects an unknown code', async () => {
    await expectRejection(api.teams.acceptInvite('does-not-exist'));
  });

  it('rejects when fetching a non-existent team (error handling)', async () => {
    await expectRejection(api.teams.get('does-not-exist'));
  });
});

describe('members', () => {
  beforeEach(login);

  it('lists members alphabetically with derived role and permissions', async () => {
    const members = await settle(api.members.list('t_a'));
    expect(members).toHaveLength(12);
    const names = members.map((m) => m.name);
    expect([...names]).toEqual([...names].sort((a, b) => a.localeCompare(b, 'de')));
    expect(members[0].perms).toBeDefined();
    expect(members.every((m) => 'primaryRole' in m)).toBe(true);
  });

  it('updates a member profile', async () => {
    const members = await settle(api.members.list('t_a'));
    const target = members[0];
    await settle(api.members.update(target.membershipId, { phone: '+49 111 222333' }, 't_a'));
    const reloaded = await settle(api.members.list('t_a'));
    expect(reloaded.find((m) => m.membershipId === target.membershipId)?.phone).toBe('+49 111 222333');
  });

  it('removes a member from the team', async () => {
    const members = await settle(api.members.list('t_a'));
    const target = members[0];
    await settle(api.members.remove(target.membershipId, 't_a'));
    const reloaded = await settle(api.members.list('t_a'));
    expect(reloaded.some((m) => m.membershipId === target.membershipId)).toBe(false);
  });

  it('rejects when updating an unknown membership (error handling)', async () => {
    await expectRejection(api.members.update('unknown', { name: 'X' }, 't_a'));
  });
});

describe('events & attendance', () => {
  beforeEach(login);

  it('lists all seeded team events and partitions them by scope', async () => {
    const all = await settle(api.events.list('t_a', 'all'));
    const upcoming = await settle(api.events.list('t_a', 'upcoming'));
    const past = await settle(api.events.list('t_a', 'past'));

    const today = todayLocalDate();
    expect(all.length).toBeGreaterThan(0);
    expect(upcoming.length + past.length).toBe(all.length);
    expect(upcoming.every((e) => e.date >= today)).toBe(true);
    expect(past.every((e) => e.date < today)).toBe(true);
  });

  it('attaches an attendance summary and personal status to each event', async () => {
    const events = await settle(api.events.list('t_a', 'all'));
    const event = events[0];
    expect(event.summary.total).toBe(12);
    expect(event.summary.yes + event.summary.no + event.summary.maybe + event.summary.pending + event.summary.notNominated).toBe(12);
    expect(event.myStatus).toBeTruthy();
  });

  it('returns null for an unknown event id (graceful, not thrown)', async () => {
    expect(await settle(api.events.get('nope', 't_a'))).toBeNull();
  });

  it('creates a single event', async () => {
    const before = (await settle(api.events.list('t_a', 'all'))).length;
    const created = await settle(
      api.events.create('t_a', { type: 'event', title: 'Sommerfest', date: '2099-08-01' }),
    );
    expect(created.title).toBe('Sommerfest');
    const after = await settle(api.events.list('t_a', 'all'));
    expect(after).toHaveLength(before + 1);
  });

  it('creates a recurring series with one event per week', async () => {
    const before = (await settle(api.events.list('t_a', 'all'))).length;
    await settle(
      api.events.create('t_a', { type: 'training', title: 'Serie', date: '2099-01-05', recurring: true, repeatWeeks: 4 }),
    );
    const after = await settle(api.events.list('t_a', 'all'));
    expect(after.filter((e) => e.title === 'Serie')).toHaveLength(4);
    expect(after).toHaveLength(before + 4);
  });

  it('round-trips an attendance response for a member', async () => {
    const events = await settle(api.events.list('t_a', 'all'));
    const event = events[0];
    await settle(api.attendance.set(event.id, 'u4', { status: 'no', reason: 'Krank' }, 't_a'));

    const rows = await settle(api.attendance.listForEvent(event.id, 't_a'));
    const row = rows.find((r) => r.userId === 'u4');
    expect(row?.status).toBe('no');
    expect(row?.reason).toBe('Krank');
    expect(rows).toHaveLength(12);
  });

  it('treats opt-out events as implicit yes for members without a record', async () => {
    // Create an opt-out event with nobody responding, then verify the auto-yes default.
    const created = await settle(
      api.events.create('t_a', { type: 'training', title: 'OptOut', date: '2099-03-03', responseMode: 'opt_out' }),
    );
    const fresh = await settle(api.events.get(created.id, 't_a'));
    expect(fresh?.responseMode).toBe('opt_out');
    expect(fresh?.summary.yes).toBeGreaterThan(0);
  });

  it('adds and lists event comments enriched with author data', async () => {
    const events = await settle(api.events.list('t_a', 'all'));
    const event = events[0];
    await settle(api.events.addComment(event.id, 'Bitte pünktlich sein', 't_a'));
    const comments = await settle(api.events.listComments(event.id, 't_a'));
    const mine = comments.find((c) => c.text === 'Bitte pünktlich sein');
    expect(mine).toBeDefined();
    expect(mine?.name).toBeTruthy();
  });
});

describe('finances', () => {
  beforeEach(login);

  it('aggregates balance, income and expense from the seeded ledger', async () => {
    const overview = await settle(api.finances.overview('t_a'));
    expect(overview.income).toBe(1147);
    expect(overview.expense).toBe(840);
    expect(overview.balance).toBe(307);
    expect(overview.penalties).toHaveLength(4);
  });

  it('reflects a new income transaction in the balance', async () => {
    await settle(api.finances.addTransaction('t_a', { type: 'income', title: 'Spende', amount: 100 }));
    const overview = await settle(api.finances.overview('t_a'));
    expect(overview.income).toBe(1247);
    expect(overview.balance).toBe(407);
  });

  it('coerces string amounts to numbers when adding a transaction', async () => {
    const tx = await settle(api.finances.addTransaction('t_a', { type: 'expense', title: 'Bus', amount: '49.5' }));
    expect(tx.amount).toBe(49.5);
  });

  it('toggles a penalty assignment paid state', async () => {
    const overview = await settle(api.finances.overview('t_a'));
    const assignment = overview.assignments[0];
    const wasPaid = assignment.paid;
    await settle(api.finances.togglePenaltyPaid(assignment.id, 't_a'));
    const reloaded = await settle(api.finances.overview('t_a'));
    expect(reloaded.assignments.find((a) => a.id === assignment.id)?.paid).toBe(!wasPaid);
  });

  it('deletes a transaction', async () => {
    const overview = await settle(api.finances.overview('t_a'));
    const tx = overview.transactions[0];
    await settle(api.finances.deleteTransaction(tx.id, 't_a'));
    const reloaded = await settle(api.finances.overview('t_a'));
    expect(reloaded.transactions.some((t) => t.id === tx.id)).toBe(false);
  });
});

describe('stats', () => {
  beforeEach(login);

  it('computes a team attendance overview', async () => {
    const stats = await settle(api.stats.teamOverview('t_a'));
    expect(stats.members).toHaveLength(12);
    expect(typeof stats.avg).toBe('number');
    expect(stats.avg).toBeGreaterThanOrEqual(0);
    expect(stats.avg).toBeLessThanOrEqual(100);
  });

  // Regression coverage for aligning the mock's stats semantics with the
  // real backend's (stats.Service.GetOverview / stats.Repository) after a
  // set of divergences surfaced: an 80% "enough" threshold instead of the
  // backend's 50%, a denominator that included 'pending' responses instead
  // of excluding them like the backend's yes/no/maybe-only COUNT FILTER, and
  // an events cap of 8 plus a strict "date < today" filter that don't exist
  // on the real backend.
  it('marks an event "enough" at 60% attendance, matching the backend\'s 50% threshold (not the old 80%)', async () => {
    const today = todayLocalDate();
    const event = await settle(api.events.create('t_a', { type: 'training', title: 'Threshold test', date: today }));
    // 3 yes, 2 no => 60% of 5 counted responses. Any other team members are
    // left with no explicit record (pending) and must not count toward the
    // denominator.
    await settle(api.attendance.set(event.id, 'u1', { status: 'yes' }, 't_a'));
    await settle(api.attendance.set(event.id, 'u2', { status: 'yes' }, 't_a'));
    await settle(api.attendance.set(event.id, 'u4', { status: 'yes' }, 't_a'));
    await settle(api.attendance.set(event.id, 'u5', { status: 'no' }, 't_a'));
    await settle(api.attendance.set(event.id, 'u6', { status: 'no' }, 't_a'));

    const stats = await settle(api.stats.teamOverview('t_a'));
    const stat = stats.events.find((e) => e.id === event.id)!;
    expect(stat.nominated).toBe(5);
    expect(stat.yes).toBe(3);
    expect(stat.pct).toBe(60);
    expect(stat.enough).toBe(true);
  });

  it('includes today-dated events in the default range, not just strictly-past ones', async () => {
    const today = todayLocalDate();
    const event = await settle(api.events.create('t_a', { type: 'training', title: 'Today event', date: today }));

    const stats = await settle(api.stats.teamOverview('t_a'));
    expect(stats.events.some((e) => e.id === event.id)).toBe(true);
  });

  it('does not cap events to 8 within the default range', async () => {
    const today = todayLocalDate();
    const created = await Promise.all(
      Array.from({ length: 9 }, (_, i) => settle(api.events.create('t_a', { type: 'training', title: `Cap test ${i}`, date: today }))),
    );

    const stats = await settle(api.stats.teamOverview('t_a'));
    for (const c of created) {
      expect(stats.events.some((e) => e.id === c.id)).toBe(true);
    }
  });

  // Regression coverage for the Stats page's "Gesamt" preset: it passes
  // ALL_TIME_FROM_DATE as an explicit `from` rather than omitting the range,
  // since an omitted `from` makes teamOverview apply its 3-month default —
  // the same value a genuinely all-time request must not collapse into.
  it('includes events far outside the 3-month default range when given an explicit all-time range', async () => {
    const old = await settle(
      api.events.create('t_a', { type: 'training', title: 'Ancient event', date: '2020-01-01' }),
    );

    const defaultRange = await settle(api.stats.teamOverview('t_a'));
    expect(defaultRange.events.some((e) => e.id === old.id)).toBe(false);

    const allTime = await settle(
      api.stats.teamOverview('t_a', { from: ALL_TIME_FROM_DATE, to: todayLocalDate() }),
    );
    expect(allTime.events.some((e) => e.id === old.id)).toBe(true);
  });

  it('computes a single member attendance quote', async () => {
    const stat = await settle(api.stats.attendanceFor('t_a', 'u1'));
    expect(stat.counted).toBeGreaterThanOrEqual(0);
    if (stat.quote !== null) {
      expect(stat.quote).toBeGreaterThanOrEqual(0);
      expect(stat.quote).toBeLessThanOrEqual(100);
    }
  });

  // Regression coverage for stats.Service.GetOverview's `avg`: it sums every
  // team member's quote (0, not skipped, when a member has 0 counted events)
  // and divides by the total member count. The mock previously excluded
  // no-data members from both the numerator and denominator, which inflates
  // the average whenever any member has no counted attendance in range (e.g.
  // a member with only opt-in events they haven't responded to yet).
  it('averages every member including those with zero counted events, not just members with data', async () => {
    // Team t_b has no events in the seed, so this is the only event in its
    // default 3-month range and every other member stays 'pending' (opt_in,
    // no explicit record) => counted 0 => quote null for them.
    const event = await settle(
      api.events.create('t_b', { type: 'training', title: 'Opt-in only test', date: todayLocalDate() }),
    );
    await settle(api.attendance.set(event.id, 'u20', { status: 'yes' }, 't_b'));

    const stats = await settle(api.stats.teamOverview('t_b'));
    expect(stats.members).toHaveLength(5);
    const u20Stat = stats.members.find((m) => m.userId === 'u20')!;
    expect(u20Stat.quote).toBe(100);
    const others = stats.members.filter((m) => m.userId !== 'u20');
    expect(others.every((m) => m.quote === null)).toBe(true);
    // (100 + 0 + 0 + 0 + 0) / 5 = 20 — not 100, which is what you'd get by
    // averaging only the one member with counted > 0.
    expect(stats.avg).toBe(20);
  });
});

describe('polls', () => {
  beforeEach(login);

  it('lists polls with computed vote tallies', async () => {
    const polls = await settle(api.polls.list('t_a'));
    expect(polls.length).toBeGreaterThan(0);
    expect(polls[0].options.every((o) => typeof o.pct === 'number')).toBe(true);
  });

  it('records the current user vote and updates the tally', async () => {
    const polls = await settle(api.polls.list('t_a'));
    const poll = polls.find((p) => p.myVote === null) ?? polls[0];
    const before = poll.totalVotes;
    const optionId = poll.options[0].id;
    await settle(api.polls.vote(poll.id, [optionId], 't_a'));

    const reloaded = (await settle(api.polls.list('t_a'))).find((p) => p.id === poll.id);
    expect(reloaded?.myVote).toContain(optionId);
    expect(reloaded?.totalVotes).toBe(before + 1);
  });

  it('creates a poll', async () => {
    const created = await settle(
      api.polls.create('t_a', { question: 'Trikotfarbe?', options: ['Rot', 'Blau'] }),
    );
    expect(created.question).toBe('Trikotfarbe?');
    const polls = await settle(api.polls.list('t_a'));
    expect(polls.some((p) => p.id === created.id)).toBe(true);
  });
});

describe('notifications', () => {
  beforeEach(login);

  it('lists notifications with an unread count', async () => {
    const result = await settle(api.notifications.list('t_a'));
    expect(result.items.length).toBeGreaterThan(0);
    expect(result.unreadCount).toBeGreaterThanOrEqual(0);
  });

  it('marks notifications as seen so the unread count drops to zero', async () => {
    await settle(api.notifications.markSeen('t_a'));
    const result = await settle(api.notifications.list('t_a'));
    expect(result.unreadCount).toBe(0);
  });
});

describe('absences', () => {
  beforeEach(login);

  it('creates an absence for the current user and lists it among personal absences', async () => {
    const me = await settle(api.auth.currentUser());
    const created = await settle(
      api.absences.create({ teamId: 't_a', userId: me!.id, from: '2099-05-01', to: '2099-05-05', reason: 'Urlaub' }),
    );
    expect(created.reason).toBe('Urlaub');
    const mine = await settle(api.absences.listMine('t_a'));
    expect(mine.some((a) => a.id === created.id)).toBe(true);
  });

  it('lists team absences enriched with member display data', async () => {
    const list = await settle(api.absences.listForTeam('t_a'));
    expect(list.length).toBeGreaterThan(0);
    expect(list[0].name).toBeTruthy();
  });
});

describe('cross-tab sync', () => {
  // Regression: the in-memory DB singleton was seeded once at module-load
  // time and every persist() unconditionally overwrote the whole localStorage
  // blob with that tab's own in-memory snapshot. With two tabs open, a stale
  // tab's next mutation -- however unrelated to what the other tab wrote --
  // would silently clobber/resurrect data, since the two never otherwise
  // communicated. A `storage` event listener now resyncs the in-memory DB
  // from a fresh cross-tab write before this tab's own next mutation, closing
  // that window.
  it('picks up a write made by another tab before its own next mutation', async () => {
    const key = config.storageKeyPrefix + 'v7_' + todayLocalDate();
    const raw = localStorage.getItem(key);
    expect(raw).toBeTruthy();
    const otherTabDb = JSON.parse(raw!);
    otherTabDb.news.push({
      id: 'news_from_other_tab',
      teamId: 't_a',
      title: 'Written by another tab',
      body: '',
      authorId: otherTabDb.users[0].id,
      pinned: false,
      createdAt: new Date().toISOString(),
    });
    const newValue = JSON.stringify(otherTabDb);
    localStorage.setItem(key, newValue);
    window.dispatchEvent(new StorageEvent('storage', { key, newValue }));

    const list = await settle(api.news.list('t_a'));
    expect(list.some((n) => n.id === 'news_from_other_tab')).toBe(true);
  });

  it('ignores storage events for unrelated keys', async () => {
    const before = await settle(api.news.list('t_a'));

    window.dispatchEvent(
      new StorageEvent('storage', { key: 'some_other_apps_key', newValue: JSON.stringify({ news: [] }) }),
    );

    const after = await settle(api.news.list('t_a'));
    expect(after.map((n) => n.id)).toEqual(before.map((n) => n.id));
  });
});

describe('resetDemoData', () => {
  it('removes all persisted demo databases from localStorage', async () => {
    await login();
    // Trigger a persist so a tv_db_* key definitely exists.
    await settle(api.teams.updateSettings('t_a', { description: 'x' }));
    expect(Object.keys(localStorage).some((k) => k.startsWith('tv_db_'))).toBe(true);

    mod.resetDemoData();
    expect(Object.keys(localStorage).some((k) => k.startsWith('tv_db_'))).toBe(false);
  });

  // Regression: config.storageKeyPrefix was defined and documented (.env,
  // CLAUDE.md) but never actually consumed -- the persistence layer hardcoded
  // its own 'tv_db_' literal, so setting VITE_STORAGE_KEY_PREFIX had no effect
  // on where the mock stored its data or what resetDemoData cleaned up.
  it('honors a custom VITE_STORAGE_KEY_PREFIX for both persistence and reset', async () => {
    vi.stubEnv('VITE_STORAGE_KEY_PREFIX', 'custom_prefix_');
    vi.resetModules();
    const customMod = await import('./serviceLayer');
    const customApi = customMod.api;

    await settle(customApi.auth.login('google'));
    await settle(customApi.teams.updateSettings('t_a', { description: 'x' }));

    expect(Object.keys(localStorage).some((k) => k.startsWith('custom_prefix_'))).toBe(true);

    customMod.resetDemoData();
    expect(Object.keys(localStorage).some((k) => k.startsWith('custom_prefix_'))).toBe(false);

    vi.unstubAllEnvs();
  });
});
