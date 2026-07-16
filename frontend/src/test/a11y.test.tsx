import { describe, it, expect, vi, beforeAll } from 'vitest';
import { render } from '@testing-library/react';
import { axe } from 'vitest-axe';

function assertNoViolations(results: Awaited<ReturnType<typeof axe>>) {
  const { violations } = results;
  expect(
    violations,
    violations.map((v) => `${v.id}: ${v.help} — ${v.nodes.map((n) => n.html).join(', ')}`).join('\n'),
  ).toHaveLength(0);
}

// ─── Mock shared dependencies ───────────────────────────────────────────────

vi.mock('@/context/AppContext', () => ({
  useApp: vi.fn(),
  isPageSheet: vi.fn().mockReturnValue(false),
  sheetErrorBoundaryKey: (sheet: { type: string; eventId?: string; membershipId?: string; userId?: string }) =>
    sheet.type + ':' + (sheet.eventId || sheet.membershipId || sheet.userId || ''),
}));

vi.mock('@/sheets', () => ({
  renderSheet: vi.fn().mockReturnValue(<div>Sheet Content</div>),
  sheetMeta: vi.fn().mockReturnValue({ title: 'Test Sheet', subtitle: null, hasBack: false, onBack: null }),
}));

vi.mock('@/features/events', async (importOriginal) => {
  const mod = await importOriginal<typeof import('@/features/events')>();
  return {
    ...mod,
    EventCalendar: () => <div>Calendar</div>,
    EventAbsences: () => <div>Absences</div>,
  };
});

// Mocked directly on the hooks module (not just the `@/features/events`
// barrel's re-export above) -- EventsPage.tsx imports `useEventsQuery` via
// the barrel, but mocking the underlying module here is what actually proved
// reliable in CI for the other event components' tests.
vi.mock('@/features/events/hooks/useEventQueries', () => ({
  useEventsQuery: vi.fn().mockReturnValue({ data: [] }),
}));

// Same rationale as useEventQueries above -- EventCalendar (rendered when
// eventsView === 'calendar') calls useAbsencesQuery directly; the plain
// `EventCalendar: () => <div>Calendar</div>` override on the barrel mock
// above is enough on its own in isolation, but this test only reliably picks
// that override up across the full suite run when the underlying hook module
// is ALSO mocked directly, matching useEventsQuery/useMembersQuery here.
vi.mock('@/features/events/hooks/useAbsenceQueries', () => ({
  useAbsencesQuery: vi.fn().mockReturnValue({ data: [] }),
}));

// Same rationale as the events mock above -- MembersPage.tsx imports
// useMembersQuery via a relative path to this module.
vi.mock('@/features/members/hooks/useMemberQueries', () => ({
  useMembersQuery: vi.fn().mockReturnValue({ data: [] }),
}));

import { useApp } from '@/context/AppContext';
import { useMembersQuery } from '@/features/members/hooks/useMemberQueries';
const mockUseApp = useApp as ReturnType<typeof vi.fn>;
const mockUseMembersQuery = useMembersQuery as ReturnType<typeof vi.fn>;

// ─── Shared app state builders ───────────────────────────────────────────────

function makeBaseApp(overrides: Record<string, unknown> = {}) {
  return {
    state: {
      primaryColor: '#4285F4',
      form: {},
      colorScheme: 'system' as const,
      ...overrides,
    },
    can: vi.fn().mockReturnValue(false),
    isStaff: vi.fn().mockReturnValue(false),
    closeSheet: vi.fn(),
    ...overrides,
  };
}

function makeEventsApp() {
  return {
    api: {},
    ...makeBaseApp({
      activeTeamId: 't1',
      eventsView: 'list',
      eventScope: 'upcoming',
      eventsOnlyPending: false,
      calShowAbsences: false,
      calMonth: null,
      absences: null,
      user: { id: 'u1' },
    }),
    setEventsView: vi.fn(),
    goEventsPending: vi.fn(),
    toggleCalAbsences: vi.fn(),
    openEventForm: vi.fn(),
    openAbsenceForm: vi.fn(),
    openCalExport: vi.fn(),
    setState: vi.fn(),
  };
}

function makeMembersApp(members: unknown[] = []) {
  mockUseMembersQuery.mockReturnValue({ data: members });
  return {
    api: {},
    ...makeBaseApp({
      activeTeamId: 't1',
      user: { id: 'u1', name: 'Test User' },
    }),
    openMemberDetail: vi.fn(),
    openMemberForm: vi.fn(),
    openRoles: vi.fn(),
  };
}

// ─── Tests ───────────────────────────────────────────────────────────────────

describe('Accessibility: SheetHost', () => {
  it('modal sheet has no axe violations', async () => {
    const { SheetHost } = await import('@/components/SheetHost');
    mockUseApp.mockReturnValue({
      ...makeBaseApp({ sheet: { type: 'teams' }, primaryColor: '#4285F4' }),
    });
    const { container } = render(<SheetHost />);
    const results = await axe(container);
    assertNoViolations(results);
  });

  it('closed sheet (no sheet) has no axe violations', async () => {
    const { SheetHost } = await import('@/components/SheetHost');
    mockUseApp.mockReturnValue(makeBaseApp({ sheet: null }));
    const { container } = render(<SheetHost />);
    const results = await axe(container);
    assertNoViolations(results);
  });
});

describe('Accessibility: EventsPage', () => {
  beforeAll(() => {
    vi.clearAllMocks();
  });

  it('tablist and tabs have no axe violations', async () => {
    const { EventsPage } = await import('@/features/events/EventsPage');
    mockUseApp.mockReturnValue(makeEventsApp());
    const { container } = render(<EventsPage />);
    const results = await axe(container);
    assertNoViolations(results);
  });

  it('calendar view has no axe violations', async () => {
    const { EventsPage } = await import('@/features/events/EventsPage');
    mockUseApp.mockReturnValue({ ...makeEventsApp(), state: { ...makeEventsApp().state, eventsView: 'calendar' } });
    const { container } = render(<EventsPage />);
    const results = await axe(container);
    assertNoViolations(results);
  });
});

describe('Accessibility: MembersPage', () => {
  beforeAll(() => {
    vi.clearAllMocks();
  });

  it('empty members list has no axe violations', async () => {
    const { MembersPage } = await import('@/features/members/MembersPage');
    mockUseApp.mockReturnValue(makeMembersApp());
    const { container } = render(<MembersPage />);
    const results = await axe(container);
    assertNoViolations(results);
  });

  it('members list with entries has no axe violations', async () => {
    const { MembersPage } = await import('@/features/members/MembersPage');
    mockUseApp.mockReturnValue(
      makeMembersApp([
        {
          membershipId: 'ms1',
          userId: 'u2',
          name: 'Anna Müller',
          email: 'anna@test.com',
          avatarColor: '#4285F4',
          photo: null,
          roles: [{ id: 'r1', name: 'Mitglied' }],
          primaryRole: null,
          joinedAt: '2025-01-01',
        },
      ]),
    );
    const { container } = render(<MembersPage />);
    const results = await axe(container);
    assertNoViolations(results);
  });
});

describe('Accessibility: Field', () => {
  it('labeled text input has no axe violations', async () => {
    const { Field } = await import('@/components/ui');
    mockUseApp.mockReturnValue(makeBaseApp({ form: { username: '' } }));
    const { container } = render(
      <Field label="Name" required>
        <input type="text" />
      </Field>,
    );
    const results = await axe(container);
    assertNoViolations(results);
  });

  it('field with error state has no axe violations', async () => {
    const { Field } = await import('@/components/ui');
    mockUseApp.mockReturnValue(makeBaseApp({ form: {} }));
    const { container } = render(
      <Field label="E-Mail" error errorText="Ungültige E-Mail-Adresse">
        <input type="email" aria-invalid="true" />
      </Field>,
    );
    const results = await axe(container);
    assertNoViolations(results);
  });
});
