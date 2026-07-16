// Behavioral regression coverage for the four drift bugs the localStorage
// mock carried relative to the real backend's intended semantics (see
// openspec/changes/replace-mock-with-msw/design.md's "Decisions" and
// tasks.md item 2.3). The old serviceContract.test.ts only diffed method
// signatures between the mock and `realApi`; now that the mock is gone and
// `realApi` is the only implementation, these scenarios instead pin the
// fixed behavior directly against the MSW demo backend.
import { describe, it, expect, beforeEach } from 'vitest';
import { realApi as api } from './serviceLayerReal';
import { DEMO_LOGIN_EMAIL, DEMO_PASSWORD } from '@/mocks/db';
import { todayLocalDate } from '@/utils/date';
import { ValidationError } from '@/utils/errors';

beforeEach(async () => {
  await api.auth.login(DEMO_LOGIN_EMAIL, DEMO_PASSWORD);
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
