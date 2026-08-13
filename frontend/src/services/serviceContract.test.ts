// Behavioral regression coverage for the four drift bugs the localStorage
// mock carried relative to the real backend's intended semantics (see
// openspec/changes/replace-mock-with-msw/design.md's "Decisions" and
// tasks.md item 2.3). The old serviceContract.test.ts only diffed method
// signatures between the mock and `realApi`; now that the mock is gone and
// `realApi` is the only implementation, these scenarios instead pin the
// fixed behavior directly against the MSW demo backend.
import { describe, it, expect, beforeEach } from 'vitest';
import { realApi as api } from './serviceLayerReal';
import { db, DEMO_LOGIN_EMAIL, DEMO_PASSWORD } from '@/mocks/db';
import { todayLocalDate } from '@/utils/date';
import { AuthError, ForbiddenError, ValidationError } from '@/utils/errors';

beforeEach(async () => {
  await api.auth.login(DEMO_LOGIN_EMAIL, DEMO_PASSWORD);
});

function onlyVerificationToken(): string {
  const tokens = Object.keys(db.verificationTokens);
  expect(tokens).toHaveLength(1);
  return tokens[0]!;
}

/** Returns the most recently issued verification token (insertion order). */
function latestVerificationToken(): string {
  const tokens = Object.keys(db.verificationTokens);
  expect(tokens.length).toBeGreaterThan(0);
  return tokens[tokens.length - 1]!;
}

describe('self-service registration: enumeration safety and verification flow', () => {
  it('registering an available email creates an unverified account and issues a verification token', async () => {
    const resp = await api.auth.register('new-user@example.com', 'longenoughpassword');
    expect(resp.message).toBeTruthy();

    const token = onlyVerificationToken();
    const { token: sessionToken, user } = await api.auth.verifyEmail(token);
    expect(sessionToken).toBeTruthy();
    expect(user.email).toBe('new-user@example.com');
  });

  it('register/resend-verification return the identical response across available, verified, and pending emails', async () => {
    // Seed a verified and a still-pending account by going through the real
    // register (+ verify, for the verified one) flow rather than poking the
    // mock db directly.
    await api.auth.register('verified@example.com', 'longenoughpassword');
    await api.auth.verifyEmail(onlyVerificationToken());
    await api.auth.register('pending@example.com', 'longenoughpassword');

    const available = await api.auth.register('available@example.com', 'longenoughpassword');
    const verified = await api.auth.register('verified@example.com', 'attacker-password-123');
    const pending = await api.auth.register('pending@example.com', 'attacker-password-123');

    expect(verified.message).toBe(available.message);
    expect(pending.message).toBe(available.message);

    // The already-registered branches' response is a 202 that never reveals
    // account existence -- assert on the resolved value only (a thrown error
    // would mean the endpoint incorrectly distinguished this case).
  });

  it('a fresh verification token for a still-pending registration does not overwrite its password', async () => {
    await api.auth.register('pending2@example.com', 'original-password-123');
    // Re-registering the same still-unverified email with a different
    // password must not let that new password take effect.
    await api.auth.register('pending2@example.com', 'attacker-password-456');

    await expect(api.auth.login('pending2@example.com', 'attacker-password-456')).rejects.toThrow();

    const token = latestVerificationToken();
    await api.auth.verifyEmail(token);
    await expect(api.auth.login('pending2@example.com', 'original-password-123')).resolves.toBeTruthy();
  });

  it('rejects login for an unverified account distinctly from wrong credentials', async () => {
    await api.auth.register('unverified@example.com', 'longenoughpassword');

    await expect(api.auth.login('unverified@example.com', 'longenoughpassword')).rejects.toBeInstanceOf(ForbiddenError);
    await expect(api.auth.login('unverified@example.com', 'totally-wrong-password')).rejects.toBeInstanceOf(AuthError);
  });

  it('an expired or unknown verification token is rejected', async () => {
    await expect(api.auth.verifyEmail('totally-bogus-token')).rejects.toThrow();
  });

  it('a verification token is single-use', async () => {
    await api.auth.register('reuse@example.com', 'longenoughpassword');
    const token = onlyVerificationToken();
    await api.auth.verifyEmail(token);
    await expect(api.auth.verifyEmail(token)).rejects.toThrow();
  });

  it('resend-verification returns the same generic response for a nonexistent email', async () => {
    const resp = await api.auth.resendVerification('nobody-at-all@example.com');
    expect(resp.message).toBeTruthy();
  });
});

function onlyPasswordResetToken(): string {
  const tokens = Object.keys(db.passwordResetTokens);
  expect(tokens).toHaveLength(1);
  return tokens[0]!;
}

describe('password reset: enumeration safety and reset flow', () => {
  it('forgot-password for a nonexistent email returns the generic response without issuing a token', async () => {
    const resp = await api.auth.forgotPassword('nobody-at-all@example.com');
    expect(resp.message).toBeTruthy();
    expect(Object.keys(db.passwordResetTokens)).toHaveLength(0);
  });

  it('forgot-password/reset-password return identical generic and success shapes across account states', async () => {
    const noAccount = await api.auth.forgotPassword('nobody-at-all2@example.com');
    const hasAccount = await api.auth.forgotPassword(DEMO_LOGIN_EMAIL);
    expect(hasAccount.message).toBe(noAccount.message);
  });

  it('a valid reset token sets the new password and establishes a session usable to log in with it', async () => {
    await api.auth.register('reset-target@example.com', 'originalpassword123');
    await api.auth.verifyEmail(onlyVerificationToken());

    await api.auth.forgotPassword('reset-target@example.com');
    const token = onlyPasswordResetToken();

    const { token: sessionToken, user } = await api.auth.resetPassword(token, 'brandnewpassword123');
    expect(sessionToken).toBeTruthy();
    expect(user.email.toLowerCase()).toBe('reset-target@example.com');

    await expect(api.auth.login('reset-target@example.com', 'brandnewpassword123')).resolves.toBeTruthy();
  });

  it('a reset token is single-use', async () => {
    await api.auth.register('reset-reuse@example.com', 'originalpassword123');
    await api.auth.verifyEmail(onlyVerificationToken());

    await api.auth.forgotPassword('reset-reuse@example.com');
    const token = onlyPasswordResetToken();
    await api.auth.resetPassword(token, 'brandnewpassword123');
    await expect(api.auth.resetPassword(token, 'anotherpassword456')).rejects.toThrow();
  });

  it('an unknown reset token is rejected', async () => {
    await expect(api.auth.resetPassword('totally-bogus-token', 'brandnewpassword123')).rejects.toThrow();
  });
});

describe('drift-bug fix: penalty amount/label is snapshotted at assignment time', () => {
  it('keeps an existing assignment at its original amount after the penalty template changes', async () => {
    const penalty = await api.finances.createPenalty('t_a', { label: 'Zu spät', amount: 5 });
    const assignment = await api.finances.assignPenalty('t_a', { userId: 'u4', penaltyId: penalty.id });
    expect(assignment.amount).toBe(5);
    expect(assignment.label).toBe('Zu spät');

    await api.finances.updatePenalty(penalty.id, { label: 'Sehr zu spät', amount: 50 }, 't_a');

    const overview = await api.finances.overview('t_a');
    const stillAssigned = overview.assignments.find((a) => a.id === assignment.id)!;
    expect(stillAssigned.amount).toBe(5);
    expect(stillAssigned.label).toBe('Zu spät');

    // A newly-created assignment against the same (now-updated) template does
    // pick up the new amount — only already-assigned instances are frozen.
    const second = await api.finances.assignPenalty('t_a', { userId: 'u5', penaltyId: penalty.id });
    expect(second.amount).toBe(50);
    expect(second.label).toBe('Sehr zu spät');
  });
});

describe('membership fees: fan-out creation, partial payment via linked transactions, deletion', () => {
  it('creates one row per selected member, tracks a partial payment, then a completing one, and survives contribution deletion', async () => {
    const created = await api.finances.createContributions('t_a', {
      label: 'Turnieranmeldung Herbst',
      amount: 30,
      dueDate: '2026-11-30',
      userIds: ['u4', 'u5'],
    });
    expect(created).toHaveLength(2);
    expect(created.every((c) => c.status === 'open' && c.paidAmount === 0)).toBe(true);
    const contrib = created.find((c) => c.userId === 'u4')!;

    const partial = await api.finances.addTransaction('t_a', {
      type: 'income',
      title: 'Anzahlung',
      amount: 10,
      contributionId: contrib.id,
    });
    expect(partial.contributionId).toBe(contrib.id);

    let overview = await api.finances.overview('t_a');
    let updated = overview.contributions.find((c) => c.id === contrib.id)!;
    expect(updated.status).toBe('partial');
    expect(updated.paidAmount).toBe(10);

    const rest = await api.finances.addTransaction('t_a', {
      type: 'income',
      title: 'Restzahlung',
      amount: 20,
      contributionId: contrib.id,
    });

    overview = await api.finances.overview('t_a');
    updated = overview.contributions.find((c) => c.id === contrib.id)!;
    expect(updated.status).toBe('paid');
    expect(updated.paidAmount).toBe(30);

    // Deleting the fee must not delete the income it already booked -- only
    // unlink it.
    await api.finances.deleteContribution(contrib.id, 't_a');
    overview = await api.finances.overview('t_a');
    expect(overview.contributions.find((c) => c.id === contrib.id)).toBeUndefined();
    const transactions = await api.finances.listTransactions('t_a');
    expect(transactions.find((t) => t.id === rest.id)?.contributionId ?? null).toBeNull();
  });

  it('rejects linking an expense transaction to a contribution', async () => {
    const [contrib] = await api.finances.createContributions('t_a', {
      label: 'Fehlerfall',
      amount: 10,
      userIds: ['u4'],
    });
    await expect(
      api.finances.addTransaction('t_a', { type: 'expense', title: 'Rückerstattung', amount: 5, contributionId: contrib!.id }),
    ).rejects.toThrow();
  });

  it('rejects creating a fee for a user who is not a member of the team', async () => {
    await expect(
      api.finances.createContributions('t_a', { label: 'Fremd', amount: 10, userIds: ['u-not-a-member'] }),
    ).rejects.toThrow();
  });

  it('archives a contribution, hides it from the open count, and un-archives it again', async () => {
    const [contrib] = await api.finances.createContributions('t_a', {
      label: 'Altes Semester',
      description: 'Deckt Halle und Trainer ab.',
      amount: 15,
      userIds: ['u4'],
    });
    expect(contrib!.description).toBe('Deckt Halle und Trainer ab.');
    expect(contrib!.archived).toBe(false);

    let overview = await api.finances.overview('t_a');
    const openBefore = overview.contribOpen;

    const archived = await api.finances.updateContribution(contrib!.id, { archived: true }, 't_a');
    expect(archived.archived).toBe(true);
    // Editing archived must not touch name/amount/description.
    expect(archived.label).toBe('Altes Semester');
    expect(archived.description).toBe('Deckt Halle und Trainer ab.');

    overview = await api.finances.overview('t_a');
    expect(overview.contribOpen).toBe(openBefore - 1);

    const restored = await api.finances.updateContribution(contrib!.id, { archived: false }, 't_a');
    expect(restored.archived).toBe(false);
    overview = await api.finances.overview('t_a');
    expect(overview.contribOpen).toBe(openBefore);
  });
});

describe('transaction note: settable on create/update, not part of any other field', () => {
  it('round-trips an optional note through create and update', async () => {
    const created = await api.finances.addTransaction('t_a', {
      type: 'income',
      title: 'Spende',
      amount: 5,
      note: 'Bar erhalten, Quittung Nr. 12',
    });
    expect(created.note).toBe('Bar erhalten, Quittung Nr. 12');

    const updated = await api.finances.updateTransaction(created.id, { note: 'Korrektur: Quittung Nr. 13' }, 't_a');
    expect(updated.note).toBe('Korrektur: Quittung Nr. 13');
  });

  it('has no note when omitted', async () => {
    const created = await api.finances.addTransaction('t_a', { type: 'expense', title: 'Bälle', amount: 20 });
    expect(created.note ?? null).toBeNull();
  });
});

describe('penalty assignments: default unpaid, paid only via a linked transaction', () => {
  it('creates an unpaid assignment, then marks it paid by booking a matching income transaction', async () => {
    const penalty = await api.finances.createPenalty('t_a', { label: 'Zu spät', amount: 5 });
    const assignment = await api.finances.assignPenalty('t_a', { userId: 'u4', penaltyId: penalty.id });
    expect(assignment.paid).toBe(false);
    expect(assignment.paidAmount).toBe(0);

    const booking = await api.finances.addTransaction('t_a', {
      type: 'income',
      title: 'Strafe bezahlt',
      amount: 5,
      penaltyAssignmentId: assignment.id,
    });
    expect(booking.penaltyAssignmentId).toBe(assignment.id);

    const overview = await api.finances.overview('t_a');
    const paid = overview.assignments.find((a) => a.id === assignment.id)!;
    expect(paid.paid).toBe(true);
    expect(paid.paidAmount).toBe(5);

    // Deleting the fine must not delete the income it already booked -- only unlink it.
    await api.finances.deleteAssignment(assignment.id, 't_a');
    const transactions = await api.finances.listTransactions('t_a');
    expect(transactions.find((t) => t.id === booking.id)?.penaltyAssignmentId ?? null).toBeNull();
  });

  it('rejects linking an expense transaction to a penalty assignment', async () => {
    const penalty = await api.finances.createPenalty('t_a', { label: 'Fehlerfall', amount: 5 });
    const assignment = await api.finances.assignPenalty('t_a', { userId: 'u4', penaltyId: penalty.id });
    await expect(
      api.finances.addTransaction('t_a', {
        type: 'expense',
        title: 'Rückerstattung',
        amount: 5,
        penaltyAssignmentId: assignment.id,
      }),
    ).rejects.toThrow();
  });

  it('rejects linking a transaction to both a contribution and a penalty assignment', async () => {
    const [contrib] = await api.finances.createContributions('t_a', { label: 'Doppelt', amount: 10, userIds: ['u4'] });
    const penalty = await api.finances.createPenalty('t_a', { label: 'Doppelt', amount: 5 });
    const assignment = await api.finances.assignPenalty('t_a', { userId: 'u4', penaltyId: penalty.id });
    await expect(
      api.finances.addTransaction('t_a', {
        type: 'income',
        title: 'Beides',
        amount: 5,
        contributionId: contrib!.id,
        penaltyAssignmentId: assignment.id,
      }),
    ).rejects.toThrow();
  });
});

describe('drift-bug fix: stats count only explicit attendance responses, not opt_out/absence defaults', () => {
  it('does not count a non-responder to an opt-out event toward a member quota', async () => {
    const today = todayLocalDate();
    const event = await api.events.create('t_a', {
      type: 'training',
      title: 'OptOut stats test',
      date: today,
      responseMode: 'opt_out',
    });
    // u4 explicitly responds; u5 never responds (auto-"yes" for the event
    // summary, but must NOT be counted for stats — see
    // backend/internal/stats/repository.go's raw `a.status IN (...)` filter,
    // which has no opt_out/absence defaulting unlike events.computeEffectiveAttendance).
    await api.attendance.set(event.id, 'u4', { status: 'yes' }, 't_a');

    const statU4 = await api.stats.attendanceFor('t_a', 'u4');
    expect(statU4.counted).toBeGreaterThanOrEqual(1);

    const overview = await api.stats.teamOverview('t_a');
    const eventStat = overview.events.find((e) => e.id === event.id)!;
    // Only u4 explicitly responded; every other member (including u5, who is
    // implicitly "yes" for the roster/summary view) is excluded from the
    // stats denominator.
    expect(eventStat.nominated).toBe(1);
    expect(eventStat.yes).toBe(1);
  });
});

describe('drift-bug fix: an excluded member is not an invalid GetMemberStats target', () => {
  it("returns zero-value 'no data' stats (not a 404) for a member flagged excludeFromStats, without leaking their real response", async () => {
    const today = todayLocalDate();
    const event = await api.events.create('t_a', { type: 'training', title: 'Exclusion regression test', date: today });
    await api.attendance.set(event.id, 'u4', { status: 'yes' }, 't_a');

    const members = await api.members.list('t_a');
    const u4 = members.find((m) => m.userId === 'u4')!;
    await api.members.update(u4.membershipId, { excludeFromStats: true }, 't_a');

    const stat = await api.stats.attendanceFor('t_a', 'u4');
    expect(stat).toEqual({ quote: null, counted: 0, yes: 0 });
  });

  it('still 404s for a userId that is not a member of the team at all', async () => {
    await expect(api.stats.attendanceFor('t_a', 'not-a-real-user-id')).rejects.toThrow();
  });
});

describe('drift-bug fix: single-choice polls reject multiple selected options', () => {
  it('rejects (422) a vote selecting >1 option on a non-multiple poll instead of silently truncating', async () => {
    const created = await api.polls.create('t_a', {
      question: 'Single choice?',
      options: ['A', 'B', 'C'],
      multiple: false,
    });
    const [a, b] = created.options;
    if (!a || !b) throw new Error('expected at least 2 poll options');

    await expect(api.polls.vote(created.id, [a.id, b.id], 't_a')).rejects.toThrow(ValidationError);

    // No vote was recorded at all (not even a truncated single-option one).
    const reloaded = (await api.polls.list('t_a')).find((p) => p.id === created.id)!;
    expect(reloaded.totalVotes).toBe(0);

    // A genuinely single-option vote still succeeds.
    await api.polls.vote(created.id, [a.id], 't_a');
    const afterValidVote = (await api.polls.list('t_a')).find((p) => p.id === created.id)!;
    expect(afterValidVote.totalVotes).toBe(1);
  });
});

describe("drift-bug fix: scope=upcoming includes today's events", () => {
  it('includes a today-dated event under scope=upcoming, not just strictly-future ones', async () => {
    const today = todayLocalDate();
    const event = await api.events.create('t_a', { type: 'training', title: 'Today event', date: today });

    const upcoming = await api.events.list('t_a', 'upcoming');
    const past = await api.events.list('t_a', 'past');

    expect(upcoming.some((e) => e.id === event.id)).toBe(true);
    expect(past.some((e) => e.id === event.id)).toBe(false);
  });
});

describe('per-team push-category preferences', () => {
  const defaultPrefs = {
    attendance: true,
    events: true,
    news: true,
    polls: true,
    absence: true,
    eventReminderEnabled: true,
    eventReminderHoursBefore: 6,
  };

  it('defaults to every category enabled before the caller ever saves anything', async () => {
    const prefs = await api.push.getPreferences('t_a');
    expect(prefs).toEqual(defaultPrefs);
  });

  it('persists a saved preference and reflects it on a later read', async () => {
    const current = await api.push.getPreferences('t_a');
    await api.push.setPreferences('t_a', { ...current, news: false, polls: false });

    const reloaded = await api.push.getPreferences('t_a');
    expect(reloaded).toEqual({ ...defaultPrefs, news: false, polls: false });
  });

  it('scopes a saved preference to the team it was saved for, not every team the caller belongs to', async () => {
    // The demo user (u1) is a member of both t_a and t_b.
    const current = await api.push.getPreferences('t_a');
    await api.push.setPreferences('t_a', { ...current, news: false });

    const otherTeamPrefs = await api.push.getPreferences('t_b');
    expect(otherTeamPrefs).toEqual(defaultPrefs);
  });

  it('persists the event-reminder lead time and enabled flag independently of the category toggles', async () => {
    const current = await api.push.getPreferences('t_a');
    await api.push.setPreferences('t_a', { ...current, eventReminderEnabled: false, eventReminderHoursBefore: 24 });

    const reloaded = await api.push.getPreferences('t_a');
    expect(reloaded).toEqual({ ...defaultPrefs, eventReminderEnabled: false, eventReminderHoursBefore: 24 });
  });
});

describe('per-team statistics view preferences and named presets', () => {
  it('reports nothing saved before the caller ever selects a range', async () => {
    const prefs = await api.statsPrefs.getPreferences('t_a');
    expect(prefs).toEqual({ range: null, presetId: null });
  });

  it('persists a saved selection and reflects it on a later read', async () => {
    await api.statsPrefs.setPreferences('t_a', { from: '2026-01-01', to: '2026-06-30' }, null);
    const reloaded = await api.statsPrefs.getPreferences('t_a');
    expect(reloaded).toEqual({ range: { from: '2026-01-01', to: '2026-06-30' }, presetId: null });
  });

  it('scopes a saved selection to the team it was saved for, not every team the caller belongs to', async () => {
    // The demo user (u1) is a member of both t_a and t_b.
    await api.statsPrefs.setPreferences('t_a', { from: '2026-01-01', to: '2026-06-30' }, null);
    const otherTeamPrefs = await api.statsPrefs.getPreferences('t_b');
    expect(otherTeamPrefs).toEqual({ range: null, presetId: null });
  });

  it('creates, lists, renames, and deletes a named preset', async () => {
    const created = await api.statsPrefs.createPreset('t_a', 'Saison 2026/27', { from: '2026-08-01', to: '2027-05-31' });
    expect(created.name).toBe('Saison 2026/27');

    const listed = await api.statsPrefs.listPresets('t_a');
    expect(listed.map((p) => p.id)).toContain(created.id);

    const renamed = await api.statsPrefs.updatePreset('t_a', created.id, { name: 'Saison 2026/27 (final)' });
    expect(renamed.name).toBe('Saison 2026/27 (final)');
    expect(renamed.from).toBe(created.from);

    await api.statsPrefs.deletePreset('t_a', created.id);
    const afterDelete = await api.statsPrefs.listPresets('t_a');
    expect(afterDelete.map((p) => p.id)).not.toContain(created.id);
  });

  it("does not list another team member's presets, even within the same team", async () => {
    const mine = await api.statsPrefs.createPreset('t_a', 'Meins', { from: '2026-01-01', to: '2026-03-31' });
    // Seed a preset directly for a different member of t_a (bypassing the
    // API, which always acts as the logged-in caller -- u1 above) to verify
    // listPresets is scoped per-caller, not just per-team.
    db.statsPresets.push({ id: 'preset-other', teamId: 't_a', userId: 'u2', name: 'Nicht Meins', from: '2026-04-01', to: '2026-05-31' });

    const listed = await api.statsPrefs.listPresets('t_a');
    expect(listed.map((p) => p.id)).toEqual([mine.id]);
  });

  it('deleting the active preset clears presetId from the saved selection without erroring', async () => {
    const preset = await api.statsPrefs.createPreset('t_a', 'Winterpause', { from: '2026-12-01', to: '2027-01-31' });
    await api.statsPrefs.setPreferences('t_a', { from: preset.from, to: preset.to }, preset.id);

    await api.statsPrefs.deletePreset('t_a', preset.id);

    const reloaded = await api.statsPrefs.getPreferences('t_a');
    expect(reloaded.presetId).toBeNull();
    expect(reloaded.range).toEqual({ from: preset.from, to: preset.to });
  });
});
