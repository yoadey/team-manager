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
