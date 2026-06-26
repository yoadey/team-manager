import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { todayLocalDate } from '@/utils/date';

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

  it('adds a member and grows the roster', async () => {
    const before = (await settle(api.members.list('t_a'))).length;
    const created = await settle(api.members.add('t_a', { name: 'Neue Tänzerin' }));
    expect(created.membershipId).toBeTruthy();
    const after = await settle(api.members.list('t_a'));
    expect(after).toHaveLength(before + 1);
    expect(after.some((m) => m.name === 'Neue Tänzerin')).toBe(true);
  });

  it('updates a member profile', async () => {
    const members = await settle(api.members.list('t_a'));
    const target = members[0];
    await settle(api.members.update(target.membershipId, { phone: '+49 111 222333' }));
    const reloaded = await settle(api.members.list('t_a'));
    expect(reloaded.find((m) => m.membershipId === target.membershipId)?.phone).toBe('+49 111 222333');
  });

  it('removes a member from the team', async () => {
    const members = await settle(api.members.list('t_a'));
    const target = members[0];
    await settle(api.members.remove(target.membershipId));
    const reloaded = await settle(api.members.list('t_a'));
    expect(reloaded.some((m) => m.membershipId === target.membershipId)).toBe(false);
  });

  it('rejects when updating an unknown membership (error handling)', async () => {
    await expectRejection(api.members.update('unknown', { name: 'X' }));
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
    expect(await settle(api.events.get('nope'))).toBeNull();
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
    await settle(api.attendance.set(event.id, 'u4', { status: 'no', reason: 'Krank' }));

    const rows = await settle(api.attendance.listForEvent(event.id));
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
    const fresh = await settle(api.events.get(created.id));
    expect(fresh?.responseMode).toBe('opt_out');
    expect(fresh?.summary.yes).toBeGreaterThan(0);
  });

  it('adds and lists event comments enriched with author data', async () => {
    const events = await settle(api.events.list('t_a', 'all'));
    const event = events[0];
    await settle(api.events.addComment(event.id, 'Bitte pünktlich sein'));
    const comments = await settle(api.events.listComments(event.id));
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
    await settle(api.finances.togglePenaltyPaid(assignment.id));
    const reloaded = await settle(api.finances.overview('t_a'));
    expect(reloaded.assignments.find((a) => a.id === assignment.id)?.paid).toBe(!wasPaid);
  });

  it('deletes a transaction', async () => {
    const overview = await settle(api.finances.overview('t_a'));
    const tx = overview.transactions[0];
    await settle(api.finances.deleteTransaction(tx.id));
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

  it('computes a single member attendance quote', async () => {
    const stat = await settle(api.stats.attendanceFor('t_a', 'u1'));
    expect(stat.counted).toBeGreaterThanOrEqual(0);
    if (stat.quote !== null) {
      expect(stat.quote).toBeGreaterThanOrEqual(0);
      expect(stat.quote).toBeLessThanOrEqual(100);
    }
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
    await settle(api.polls.vote(poll.id, [optionId]));

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
    const created = await settle(api.absences.create({ from: '2099-05-01', to: '2099-05-05', reason: 'Urlaub' }));
    expect(created.reason).toBe('Urlaub');
    const mine = await settle(api.absences.listMine());
    expect(mine.some((a) => a.id === created.id)).toBe(true);
  });

  it('lists team absences enriched with member display data', async () => {
    const list = await settle(api.absences.listForTeam('t_a'));
    expect(list.length).toBeGreaterThan(0);
    expect(list[0].name).toBeTruthy();
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
});
