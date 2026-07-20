import { describe, it, expect, vi, beforeEach } from 'vitest';
import { pageMeta } from './pageMeta';

vi.mock('@/context/AppContext', () => ({
  useApp: vi.fn(),
}));

// pageMeta calls t() from @/i18n which reads the German catalog — let it run
// so we can assert against the real translated strings. No mock needed.

// ────────────────────────────────────────────────────────────────────────────
// App mock factory
// ────────────────────────────────────────────────────────────────────────────

function makeApp(routeOverride = 'home', extras: Record<string, unknown> = {}) {
  return {
    state: {
      route: routeOverride,
      primaryColor: '#4285F4',
      ...((extras.state as Record<string, unknown>) ?? {}),
    },
    can: vi.fn().mockReturnValue(false),
    activePageSheet: vi.fn().mockReturnValue(null),
    activeTeam: vi.fn().mockReturnValue(null),
    openEventForm: vi.fn(),
    openInvite: vi.fn(),
    openTxForm: vi.fn(),
    openNewsForm: vi.fn(),
    openPollForm: vi.fn(),
    ...extras,
  };
}

// ────────────────────────────────────────────────────────────────────────────
// Tests
// ────────────────────────────────────────────────────────────────────────────

describe('pageMeta()', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  // 1. Home page title / subtitle
  it('returns home page title and subtitle for route "home"', () => {
    const app = makeApp('home');
    const meta = pageMeta(app as never);
    expect(meta.title).toBe('Willkommen zurück');
    expect(meta.subtitle).toBe('Dein Überblick');
  });

  // 2. Events page title
  it('returns events page title for route "events"', () => {
    const app = makeApp('events');
    const meta = pageMeta(app as never);
    expect(meta.title).toBe('Termine');
  });

  // 3. showPrimaryAction is false for home
  it('showPrimaryAction is false for home route', () => {
    const app = makeApp('home');
    const meta = pageMeta(app as never);
    expect(meta.showPrimaryAction).toBe(false);
  });

  // 4. showPrimaryAction is true for events when can('events','write') = true
  it('showPrimaryAction is true for events when user can write events', () => {
    const app = makeApp('events');
    (app.can as ReturnType<typeof vi.fn>).mockReturnValue(true);
    const meta = pageMeta(app as never);
    expect(meta.showPrimaryAction).toBe(true);
  });

  // 5. showPrimaryAction is false for events when not authorized
  it('showPrimaryAction is false for events when user cannot write', () => {
    const app = makeApp('events');
    (app.can as ReturnType<typeof vi.fn>).mockReturnValue(false);
    const meta = pageMeta(app as never);
    expect(meta.showPrimaryAction).toBe(false);
  });

  // 6. primaryAction calls openEventForm(null) for events route
  it('primaryAction calls openEventForm(null) for events route', () => {
    const app = makeApp('events');
    (app.can as ReturnType<typeof vi.fn>).mockReturnValue(true);
    const meta = pageMeta(app as never);
    meta.primaryAction();
    expect(app.openEventForm).toHaveBeenCalledWith(null);
  });

  // 7. primaryAction calls openTxForm() for finances route
  it('primaryAction calls openTxForm() for finances route', () => {
    const app = makeApp('finances');
    (app.can as ReturnType<typeof vi.fn>).mockReturnValue(true);
    const meta = pageMeta(app as never);
    meta.primaryAction();
    expect(app.openTxForm).toHaveBeenCalled();
  });

  // 8. Route 'team' returns no primary action (showPrimaryAction false)
  it('returns no primary action for team route', () => {
    const app = makeApp('team');
    const meta = pageMeta(app as never);
    expect(meta.showPrimaryAction).toBe(false);
  });

  // 9. Returns subtitle with member count for members route
  it('returns subtitle containing member count for members route', () => {
    const app = makeApp('members', {
      state: { route: 'members', primaryColor: '#4285F4' },
    });
    const meta = pageMeta(app as never, undefined, 3);
    expect(meta.subtitle).toContain('3');
  });

  // Regression test: page.membersSubtitle had no _one/_other plural forms
  // and was called with n but not count, so a team with exactly one member
  // showed the grammatically wrong "1 Personen" ("1 people") instead of
  // "1 Person".
  it('uses the singular member-count form in the members subtitle for a one-member team', () => {
    const app = makeApp('members', {
      state: { route: 'members', primaryColor: '#4285F4' },
    });
    const meta = pageMeta(app as never, undefined, 1);
    expect(meta.subtitle).toContain('1 Person ');
    expect(meta.subtitle).not.toContain('1 Personen');
  });

  // 10. pageSheetMeta returns correct title for 'eventDetail' with event
  it('returns event title for eventDetail sheet with event', () => {
    const mockEvent = { id: 'e1', title: 'Sommerball', date: '2026-07-04' };
    const app = makeApp('events');
    (app.activePageSheet as ReturnType<typeof vi.fn>).mockReturnValue({
      type: 'eventDetail',
      eventId: 'e1',
    });
    // The eventDetail sheet no longer carries its event -- the caller (AppShell)
    // fetches it via useEventDetailQuery and passes it through explicitly.
    const meta = pageMeta(app as never, mockEvent as never);
    expect(meta.title).toBe('Sommerball');
  });

  // 11. pageSheetMeta returns correct title for 'eventForm' create mode
  it('returns "Neuer Termin" title for eventForm in create mode', () => {
    const app = makeApp('events');
    (app.activePageSheet as ReturnType<typeof vi.fn>).mockReturnValue({
      type: 'eventForm',
      mode: 'create',
    });
    const meta = pageMeta(app as never);
    expect(meta.title).toBe('Neuer Termin');
  });

  // 12. pageSheetMeta returns correct title for 'memberDetail' with member
  it('returns member name as title for memberDetail sheet', () => {
    const mockMember = { id: 'm1', name: 'Julia Schneider', roles: [] };
    const app = makeApp('members');
    (app.activePageSheet as ReturnType<typeof vi.fn>).mockReturnValue({
      type: 'memberDetail',
      member: mockMember,
    });
    const meta = pageMeta(app as never);
    expect(meta.title).toBe('Julia Schneider');
  });

  // 13. members primaryAction calls openInvite
  it('primaryAction calls openInvite for members route', () => {
    const app = makeApp('members', {
      state: { route: 'members', members: [], primaryColor: '#4285F4' },
    });
    (app.can as ReturnType<typeof vi.fn>).mockReturnValue(true);
    const meta = pageMeta(app as never);
    meta.primaryAction();
    expect(app.openInvite).toHaveBeenCalled();
  });

  // 14. news route title
  it('returns news page title for route "news"', () => {
    const app = makeApp('news');
    const meta = pageMeta(app as never);
    expect(meta.title).toBe('Neuigkeiten');
  });

  // 15. news primaryAction calls openNewsForm
  it('primaryAction calls openNewsForm for news route', () => {
    const app = makeApp('news');
    (app.can as ReturnType<typeof vi.fn>).mockReturnValue(true);
    const meta = pageMeta(app as never);
    meta.primaryAction();
    expect(app.openNewsForm).toHaveBeenCalled();
  });

  // 16. polls route title
  it('returns polls page title for route "polls"', () => {
    const app = makeApp('polls');
    const meta = pageMeta(app as never);
    expect(meta.title).toBe('Umfragen');
  });

  // 17. polls primaryAction calls openPollForm
  it('primaryAction calls openPollForm for polls route', () => {
    const app = makeApp('polls');
    (app.can as ReturnType<typeof vi.fn>).mockReturnValue(true);
    const meta = pageMeta(app as never);
    meta.primaryAction();
    expect(app.openPollForm).toHaveBeenCalled();
  });

  // 18. pageSheetMeta memberForm (self mode)
  it('returns self-label for memberForm sheet in self mode', () => {
    const app = makeApp('members');
    (app.activePageSheet as ReturnType<typeof vi.fn>).mockReturnValue({
      type: 'memberForm',
      self: true,
    });
    const meta = pageMeta(app as never);
    expect(meta.title).toBeTruthy();
    expect(meta.showPrimaryAction).toBe(false);
  });

  // 19. pageSheetMeta teamSettings
  it('returns correct title for teamSettings sheet', () => {
    const app = makeApp('team');
    (app.activeTeam as ReturnType<typeof vi.fn>).mockReturnValue({ name: 'SG Muster Heim' });
    (app.activePageSheet as ReturnType<typeof vi.fn>).mockReturnValue({ type: 'teamSettings' });
    const meta = pageMeta(app as never);
    expect(meta.title).toBeTruthy();
    expect(meta.showPrimaryAction).toBe(false);
  });

  // Regression test: the teamSettings subtitle used to run the team name
  // through shortName(), truncating it to its first word even though this
  // subtitle isn't width-constrained (SheetHost renders it in a plain,
  // wrapping Box, not an ellipsis-truncated one) -- an admin managing e.g.
  // "SG Musterstadt Handball" would see just "SG" instead of the full club
  // name at exactly the moment ("confirm you're editing the right team")
  // that context matters most.
  it('shows the full team name (not just its first word) as the teamSettings subtitle', () => {
    const app = makeApp('team');
    (app.activeTeam as ReturnType<typeof vi.fn>).mockReturnValue({ name: 'SG Musterstadt Handball' });
    (app.activePageSheet as ReturnType<typeof vi.fn>).mockReturnValue({ type: 'teamSettings' });
    const meta = pageMeta(app as never);
    expect(meta.subtitle).toBe('SG Musterstadt Handball');
  });

  // 20. pageSheetMeta roles
  it('returns correct title for roles sheet', () => {
    const app = makeApp('team');
    (app.activePageSheet as ReturnType<typeof vi.fn>).mockReturnValue({ type: 'roles' });
    const meta = pageMeta(app as never);
    expect(meta.title).toBeTruthy();
  });

  // 21. pageSheetMeta roleForm
  it('returns correct title for roleForm sheet', () => {
    const app = makeApp('team');
    (app.activePageSheet as ReturnType<typeof vi.fn>).mockReturnValue({ type: 'roleForm' });
    const meta = pageMeta(app as never);
    expect(meta.title).toBeTruthy();
  });

  // 22. pageSheetMeta unknown type returns empty
  it('returns empty title for unknown sheet type', () => {
    const app = makeApp('home');
    (app.activePageSheet as ReturnType<typeof vi.fn>).mockReturnValue({ type: 'unknownSheetType' });
    const meta = pageMeta(app as never);
    expect(meta.title).toBe('');
  });

  // 23. pageSheetMeta eventDetail without event
  it('returns fallback title for eventDetail sheet with no event', () => {
    const app = makeApp('events');
    (app.activePageSheet as ReturnType<typeof vi.fn>).mockReturnValue({
      type: 'eventDetail',
      event: null,
    });
    const meta = pageMeta(app as never);
    expect(meta.showPrimaryAction).toBe(false);
  });

  // 24. pageSheetMeta memberDetail without member
  it('returns fallback for memberDetail sheet with no member', () => {
    const app = makeApp('members');
    (app.activePageSheet as ReturnType<typeof vi.fn>).mockReturnValue({
      type: 'memberDetail',
      member: null,
    });
    const meta = pageMeta(app as never);
    expect(meta.showPrimaryAction).toBe(false);
  });

  // 25. eventForm edit mode
  it('returns edit title for eventForm in edit mode', () => {
    const app = makeApp('events');
    (app.activePageSheet as ReturnType<typeof vi.fn>).mockReturnValue({
      type: 'eventForm',
      mode: 'edit',
    });
    const meta = pageMeta(app as never);
    expect(meta.title).toBeTruthy();
  });

  // 26. stats route
  it('returns stats title with no primary action', () => {
    const app = makeApp('stats');
    const meta = pageMeta(app as never);
    expect(meta.showPrimaryAction).toBe(false);
  });

  // 27. unknown route falls back to home
  it('falls back to home for unknown route', () => {
    const app = makeApp('unknown_route_xyz');
    const meta = pageMeta(app as never);
    expect(meta.title).toBe('Willkommen zurück');
  });

  // 28. teamSettings with no activeTeam
  it('returns empty subtitle for teamSettings when no active team', () => {
    const app = makeApp('team');
    (app.activeTeam as ReturnType<typeof vi.fn>).mockReturnValue(null);
    (app.activePageSheet as ReturnType<typeof vi.fn>).mockReturnValue({ type: 'teamSettings' });
    const meta = pageMeta(app as never);
    expect(meta.subtitle).toBe('');
  });

  // 29. memberDetail with roles displays joined roles
  it('returns joined role names as subtitle for memberDetail with roles', () => {
    const mockMember = {
      id: 'm1',
      name: 'Max Mustermann',
      roles: [{ name: 'Trainer' }, { name: 'Vorstand' }],
    };
    const app = makeApp('members');
    (app.activePageSheet as ReturnType<typeof vi.fn>).mockReturnValue({
      type: 'memberDetail',
      member: mockMember,
    });
    const meta = pageMeta(app as never);
    expect(meta.subtitle).toContain('Trainer');
    expect(meta.subtitle).toContain('Vorstand');
  });
});
