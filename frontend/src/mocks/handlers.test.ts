// Exercises the MSW demo/test backend (src/mocks/handlers.ts) end-to-end
// through `realApi` — the generated openapi-fetch client hitting the
// intercepted network layer, exactly as the app does in dev-demo and in
// every other test. Replaces the old serviceLayer.test.ts, which drove the
// deleted localStorage mock directly.
import { describe, it, expect, beforeEach } from 'vitest';
import { realApi as api } from '@/services/serviceLayerReal';
import { DEMO_LOGIN_EMAIL, DEMO_PASSWORD, DEMO_LOGIN_USER_ID } from './db';
import { AuthError, ValidationError } from '@/utils/errors';

async function login(): Promise<void> {
  await api.auth.login(DEMO_LOGIN_EMAIL, DEMO_PASSWORD);
}

describe('auth', () => {
  it('exposes the password provider', async () => {
    const providers = await api.auth.providers();
    expect(providers.map((p) => p.id)).toContain('password');
  });

  it('returns no current user before login', async () => {
    expect(await api.auth.currentUser()).toBeNull();
  });

  it('rejects an unknown email', async () => {
    await expect(api.auth.login('nope@example.de', DEMO_PASSWORD)).rejects.toThrow(AuthError);
  });

  it('rejects the correct email with a wrong password (not password-less)', async () => {
    await expect(api.auth.login(DEMO_LOGIN_EMAIL, 'wrong')).rejects.toThrow(AuthError);
    expect(await api.auth.currentUser()).toBeNull();
  });

  it('establishes a session on login with the correct demo credentials', async () => {
    const session = await api.auth.login(DEMO_LOGIN_EMAIL, DEMO_PASSWORD);
    expect(session.user.id).toBe(DEMO_LOGIN_USER_ID);
    const user = await api.auth.currentUser();
    expect(user?.id).toBe(DEMO_LOGIN_USER_ID);
  });

  it('clears the session on logout', async () => {
    await login();
    await api.auth.logout();
    expect(await api.auth.currentUser()).toBeNull();
  });
});

describe('teams', () => {
  beforeEach(login);

  it('lists the current user teams enriched with merged permissions', async () => {
    const teams = await api.teams.listForCurrentUser();
    const teamA = teams.find((t) => t.id === 't_a');
    expect(teamA?.myPerms.finances).toBe('write');
    expect(teamA?.memberCount).toBe(12);
  });

  it('creates a team and makes the creator an admin member', async () => {
    const team = await api.teams.create({ name: 'C-Team' });
    const teams = await api.teams.listForCurrentUser();
    const created = teams.find((t) => t.id === team.id);
    expect(created?.myPerms.settings).toBe('write');
  });

  it('creates an invite with a shareable link', async () => {
    const invite = await api.teams.createInvite('t_a');
    expect(invite.link).toContain('/join/t_a/' + invite.code);
  });

  it('acceptInvite is idempotent for an already-existing member', async () => {
    const invite = await api.teams.createInvite('t_a');
    const before = await api.members.list('t_a');
    const joined = await api.teams.acceptInvite(invite.code);
    expect(joined.id).toBe('t_a');
    const after = await api.members.list('t_a');
    expect(after).toHaveLength(before.length);
  });

  it('rejects fetching a non-existent team', async () => {
    await expect(api.teams.get('does-not-exist')).rejects.toThrow();
  });

  it('rejects accepting an unknown invite code', async () => {
    await expect(api.teams.acceptInvite('DOES-NOT-EXIST')).rejects.toThrow();
  });

  it('assigns the seeded default member role, not a privileged one, on accept', async () => {
    const invite = await api.teams.createInvite('t_a');
    const before = await api.members.list('t_a');
    const self = before.find((m) => m.userId === DEMO_LOGIN_USER_ID)!;
    await api.members.remove(self.membershipId, 't_a');
    await api.teams.acceptInvite(invite.code);
    const after = await api.members.list('t_a');
    const rejoined = after.find((m) => m.userId === DEMO_LOGIN_USER_ID)!;
    expect(rejoined.roles.every((r) => r.permissions.finances !== 'write' && r.permissions.settings !== 'write')).toBe(true);
  });
});

describe('members & roles', () => {
  beforeEach(login);

  it('lists members alphabetically', async () => {
    const members = await api.members.list('t_a');
    expect(members).toHaveLength(12);
    const names = members.map((m) => m.name);
    expect(names).toEqual([...names].sort((a, b) => a.localeCompare(b, 'de')));
  });

  it('updates a member profile', async () => {
    const members = await api.members.list('t_a');
    const target = members[0]!;
    await api.members.update(target.membershipId, { phone: '+49 111 222333' }, 't_a');
    const reloaded = await api.members.list('t_a');
    expect(reloaded.find((m) => m.membershipId === target.membershipId)?.phone).toBe('+49 111 222333');
  });

  it('removes a member from the team', async () => {
    const members = await api.members.list('t_a');
    const target = members[0]!;
    await api.members.remove(target.membershipId, 't_a');
    const reloaded = await api.members.list('t_a');
    expect(reloaded.some((m) => m.membershipId === target.membershipId)).toBe(false);
  });

  it('rejects updating an unknown member', async () => {
    await expect(api.members.update('does-not-exist', { phone: '+49 1' }, 't_a')).rejects.toThrow();
  });

  it('rejects removing an already-removed member', async () => {
    const members = await api.members.list('t_a');
    const target = members[0]!;
    await api.members.remove(target.membershipId, 't_a');
    await expect(api.members.remove(target.membershipId, 't_a')).rejects.toThrow();
  });

  it('clears all of a member roles when roleIds is set to an empty array', async () => {
    const members = await api.members.list('t_a');
    const target = members.find((m) => m.roles.length > 0)!;
    const updated = await api.members.setRoles(target.membershipId, [], 't_a');
    expect(updated.roles).toEqual([]);
  });

  it('creates, updates and removes a custom role', async () => {
    const role = await api.roles.create('t_a', {
      name: 'Testrolle',
      permissions: { events: 'read', members: 'none', finances: 'none', news: 'none', polls: 'none', settings: 'none' },
    });
    expect(role.name).toBe('Testrolle');
    const updated = await api.roles.update(role.id, { name: 'Umbenannt' }, 't_a');
    expect(updated.name).toBe('Umbenannt');
    await api.roles.remove(role.id, 't_a');
    const roles = await api.roles.list('t_a');
    expect(roles.some((r) => r.id === role.id)).toBe(false);
  });
});

describe('events & attendance', () => {
  beforeEach(login);

  it('creates a single event and lists it', async () => {
    const before = (await api.events.list('t_a', 'all')).length;
    const created = await api.events.create('t_a', { type: 'event', title: 'Sommerfest', date: '2099-08-01' });
    expect(created.title).toBe('Sommerfest');
    expect((await api.events.list('t_a', 'all')).length).toBe(before + 1);
  });

  it('creates a recurring series with one event per week', async () => {
    await api.events.create('t_a', { type: 'training', title: 'Serie', date: '2099-01-05', recurring: true, repeatWeeks: 4 });
    const after = await api.events.list('t_a', 'all');
    expect(after.filter((e) => e.title === 'Serie')).toHaveLength(4);
  });

  it('round-trips an attendance response', async () => {
    const events = await api.events.list('t_a', 'all');
    const event = events[0]!;
    await api.attendance.set(event.id, 'u4', { status: 'no', reason: 'Krank' }, 't_a');
    const rows = await api.attendance.listForEvent(event.id, 't_a');
    const row = rows.find((r) => r.userId === 'u4');
    expect(row?.status).toBe('no');
    expect(row?.reason).toBe('Krank');
  });

  it('returns null for an unknown event id', async () => {
    expect(await api.events.get('nope', 't_a')).toBeNull();
  });

  it('adds and lists event comments', async () => {
    const events = await api.events.list('t_a', 'all');
    const event = events[0]!;
    await api.events.addComment(event.id, 'Bitte pünktlich sein', 't_a');
    const comments = await api.events.listComments(event.id, 't_a');
    expect(comments.some((c) => c.text === 'Bitte pünktlich sein')).toBe(true);
  });
});

describe('absences', () => {
  beforeEach(login);

  it('creates an absence and lists it among personal absences', async () => {
    const me = await api.auth.currentUser();
    const created = await api.absences.create({ teamId: 't_a', userId: me!.id, from: '2099-05-01', to: '2099-05-05', reason: 'Urlaub' });
    expect(created.reason).toBe('Urlaub');
    const mine = await api.absences.listMine('t_a');
    expect(mine.some((a) => a.id === created.id)).toBe(true);
  });
});

describe('news', () => {
  beforeEach(login);

  it('creates and lists news, pinned first', async () => {
    await api.news.create('t_a', { title: 'Wichtig', body: 'x', pinned: true });
    const list = await api.news.list('t_a');
    expect(list[0]!.pinned).toBe(true);
  });
});

describe('finances', () => {
  beforeEach(login);

  it('aggregates balance from the seeded ledger', async () => {
    const overview = await api.finances.overview('t_a');
    expect(overview.balance).toBe(overview.income - overview.expense);
    expect(overview.penalties).toHaveLength(4);
  });

  it('reflects a new income transaction in the balance', async () => {
    const before = await api.finances.overview('t_a');
    await api.finances.addTransaction('t_a', { type: 'income', title: 'Spende', amount: 100 });
    const after = await api.finances.overview('t_a');
    expect(after.income).toBe(before.income + 100);
  });

  it('lists the full transaction history via the paginated endpoint', async () => {
    const overview = await api.finances.overview('t_a');
    const listed = await api.finances.listTransactions('t_a');
    // The paginated list surfaces at least everything the (capped) overview does.
    expect(listed.length).toBeGreaterThanOrEqual(overview.transactions.length);
    expect(listed.some((t) => t.id === overview.transactions[0].id)).toBe(true);
  });

  it('back-dates a transaction with a client-provided date', async () => {
    const created = await api.finances.addTransaction('t_a', {
      type: 'expense',
      title: 'Nachtrag',
      amount: 50,
      date: '2023-02-01',
    });
    expect(created.date).toBe('2023-02-01');
    const listed = await api.finances.listTransactions('t_a');
    expect(listed.find((t) => t.id === created.id)?.date).toBe('2023-02-01');
  });

  it('sets a penalty assignment paid state (idempotent)', async () => {
    const overview = await api.finances.overview('t_a');
    const assignment = overview.assignments[0]!;
    const target = !assignment.paid;
    await api.finances.setPenaltyPaid(assignment.id, 't_a', target);
    let reloaded = await api.finances.overview('t_a');
    expect(reloaded.assignments.find((a) => a.id === assignment.id)?.paid).toBe(target);
    // Idempotent: applying the same value again keeps it, not flips it.
    await api.finances.setPenaltyPaid(assignment.id, 't_a', target);
    reloaded = await api.finances.overview('t_a');
    expect(reloaded.assignments.find((a) => a.id === assignment.id)?.paid).toBe(target);
  });

  it('deletes a transaction', async () => {
    const overview = await api.finances.overview('t_a');
    const tx = overview.transactions[0]!;
    await api.finances.deleteTransaction(tx.id, 't_a');
    const reloaded = await api.finances.overview('t_a');
    expect(reloaded.transactions.some((t) => t.id === tx.id)).toBe(false);
  });
});

describe('polls', () => {
  beforeEach(login);

  it('lists polls with computed vote tallies', async () => {
    const polls = await api.polls.list('t_a');
    expect(polls.length).toBeGreaterThan(0);
    expect(polls[0]!.options.every((o) => typeof o.pct === 'number')).toBe(true);
  });

  it('records the current user vote and updates the tally', async () => {
    const polls = await api.polls.list('t_a');
    const poll = polls.find((p) => p.multiple)!;
    const before = poll.totalVotes;
    await api.polls.vote(poll.id, [poll.options[0]!.id], 't_a');
    const reloaded = (await api.polls.list('t_a')).find((p) => p.id === poll.id);
    expect(reloaded?.totalVotes).toBe(before + 1);
  });

  it('creates a poll', async () => {
    const created = await api.polls.create('t_a', { question: 'Trikotfarbe?', options: ['Rot', 'Blau'] });
    const polls = await api.polls.list('t_a');
    expect(polls.some((p) => p.id === created.id)).toBe(true);
  });
});

describe('notifications', () => {
  beforeEach(login);

  it('lists notifications with an unread count', async () => {
    const result = await api.notifications.list('t_a');
    expect(result.items.length).toBeGreaterThan(0);
  });

  it('marks notifications as seen so the unread count drops to zero', async () => {
    await api.notifications.markSeen('t_a');
    const result = await api.notifications.list('t_a');
    expect(result.unreadCount).toBe(0);
  });
});

describe('poll voting validation', () => {
  beforeEach(login);

  it('maps a single-choice poll rejecting multiple options to ValidationError (422)', async () => {
    const polls = await api.polls.list('t_a');
    const single = polls.find((p) => !p.multiple)!;
    await expect(api.polls.vote(single.id, [single.options[0]!.id, single.options[1]!.id], 't_a')).rejects.toThrow(
      ValidationError,
    );
  });
});
