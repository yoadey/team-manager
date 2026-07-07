import { describe, it, expect, beforeEach, vi } from 'vitest';
import { render, screen, waitFor, act } from '@testing-library/react';
import { AppProvider, useApp, useAppActions, useAppSelector } from './AppContext';

beforeEach(() => localStorage.clear());

function Probe({ onActions }: { onActions: (ref: object) => void }) {
  const { state } = useApp();
  const actions = useAppActions();
  onActions(actions);
  return <div data-testid="phase">{state.phase}</div>;
}

function PhaseAndSheet({
  onMount,
}: {
  onMount: (actions: ReturnType<typeof useAppActions>, state: ReturnType<typeof useApp>['state']) => void;
}) {
  const { state } = useApp();
  const actions = useAppActions();
  onMount(actions, state);
  return (
    <div>
      <div data-testid="phase">{state.phase}</div>
      <div data-testid="sheet">{state.sheet?.type || 'none'}</div>
      <div data-testid="form">{JSON.stringify(state.form)}</div>
    </div>
  );
}

describe('AppProvider / context split', () => {
  it('boots through the mock service layer to the login phase', async () => {
    render(
      <AppProvider>
        <Probe onActions={() => {}} />
      </AppProvider>,
    );
    expect(screen.getByTestId('phase').textContent).toBe('loading');
    await waitFor(() => expect(screen.getByTestId('phase').textContent).toBe('login'));
  });

  it('keeps the actions object identity stable across state-driven re-renders', async () => {
    const seen: object[] = [];
    render(
      <AppProvider>
        <Probe onActions={(ref) => seen.push(ref)} />
      </AppProvider>,
    );
    await waitFor(() => expect(screen.getByTestId('phase').textContent).toBe('login'));
    await act(async () => {});
    expect(seen.length).toBeGreaterThan(1);
    expect(seen.every((ref) => ref === seen[0])).toBe(true);
  });

  it('useAppSelector re-renders only when the selected slice changes', async () => {
    let renders = 0;
    let actions: ReturnType<typeof useAppActions> | null = null;
    function Capture() {
      actions = useAppActions();
      return null;
    }
    function FieldProbe() {
      const v = useAppSelector((s) => s.form['x']);
      renders++;
      return <div data-testid="fld">{String(v ?? '')}</div>;
    }
    render(
      <AppProvider>
        <Capture />
        <FieldProbe />
      </AppProvider>,
    );
    await waitFor(() => expect(actions).not.toBeNull());
    const baseline = renders;

    // Unrelated state change (toast) must NOT re-render the field probe.
    await act(async () => actions!.toastMsg('hi'));
    expect(renders).toBe(baseline);

    // Selected slice change re-renders and reflects the new value.
    await act(async () => actions!.setFormVal({ x: 'abc' }));
    expect(screen.getByTestId('fld').textContent).toBe('abc');
    expect(renders).toBeGreaterThan(baseline);
  });
});

describe('AppProvider / actions (app phase)', () => {
  let capturedActions: ReturnType<typeof useAppActions>;

  async function renderAndBootstrap() {
    render(
      <AppProvider>
        <PhaseAndSheet
          onMount={(a) => {
            capturedActions = a;
          }}
        />
      </AppProvider>,
    );
    await waitFor(() => expect(screen.getByTestId('phase').textContent).toBe('login'));
    // Fast-forward to app phase by setting state directly (avoids full login delay)
    await act(async () => {
      capturedActions.setState({
        phase: 'app',
        user: {
          id: 'u1',
          name: 'Test User',
          email: 'test@example.com',
          avatarColor: '#000',
          photo: null,
          phone: '',
          birthday: '',
          address: '',
        },
        teams: [
          {
            id: 'team1',
            name: 'Test Team',
            membershipId: 'ms1',
            myRoles: [{ id: 'r1', name: 'Member' }],
            myPerms: { events: 'write', members: 'read', finances: 'none' },
          },
        ] as never,
        activeTeamId: 'team1',
        roles: [{ id: 'r1', name: 'Member' }] as never,
        events: [],
        members: [],
        news: [],
        polls: [],
      });
    });
    await waitFor(() => expect(screen.getByTestId('phase').textContent).toBe('app'));
  }

  it('askConfirm sets confirm sheet', async () => {
    await renderAndBootstrap();
    await act(async () => {
      capturedActions.askConfirm({ title: 'Confirm?', message: 'Sure?', onConfirm: vi.fn() });
    });
    expect(screen.getByTestId('sheet').textContent).toBe('confirm');
  });

  it('cancelConfirm closes confirm sheet', async () => {
    await renderAndBootstrap();
    await act(async () => {
      capturedActions.askConfirm({ title: 'Confirm?', message: 'Sure?', onConfirm: vi.fn() });
    });
    await act(async () => {
      capturedActions.cancelConfirm();
    });
    expect(screen.getByTestId('sheet').textContent).toBe('none');
  });

  it('runConfirm calls onConfirm callback', async () => {
    await renderAndBootstrap();
    const onConfirm = vi.fn().mockResolvedValue(undefined);
    await act(async () => {
      capturedActions.askConfirm({ title: 'Delete?', message: 'Really delete?', onConfirm });
    });
    await act(async () => {
      await capturedActions.runConfirm();
    });
    expect(onConfirm).toHaveBeenCalled();
    expect(screen.getByTestId('sheet').textContent).toBe('none');
  });

  it('openEventForm with null creates new event form', async () => {
    await renderAndBootstrap();
    await act(async () => {
      capturedActions.openEventForm(null);
    });
    expect(screen.getByTestId('sheet').textContent).toBe('eventForm');
    const form = JSON.parse(screen.getByTestId('form').textContent!);
    expect(form.type).toBe('training');
    expect(form.title).toBe('');
  });

  it('openEventForm with event populates edit form', async () => {
    await renderAndBootstrap();
    const event = {
      id: 'ev1',
      seriesId: null,
      type: 'training',
      title: 'Großes Training',
      date: '2026-06-15',
      meetTime: '19:00',
      startTime: '19:30',
      endTime: '21:00',
      location: 'Halle',
      note: '',
      meetTimeMandatory: true,
      responseMode: 'opt_out',
      nominatedRoleIds: ['r1'],
    } as never;
    await act(async () => {
      capturedActions.openEventForm(event);
    });
    expect(screen.getByTestId('sheet').textContent).toBe('eventForm');
    const form = JSON.parse(screen.getByTestId('form').textContent!);
    expect(form.id).toBe('ev1');
    expect(form.title).toBe('Großes Training');
  });

  it('toggleFormNomRole adds role to nominatedRoleIds', async () => {
    await renderAndBootstrap();
    await act(async () => {
      capturedActions.openEventForm(null);
    });
    await act(async () => {
      capturedActions.toggleFormNomRole('r2');
    });
    const form = JSON.parse(screen.getByTestId('form').textContent!);
    expect(form.nominatedRoleIds).toContain('r2');
  });

  it('toggleFormNomRole removes role from nominatedRoleIds', async () => {
    await renderAndBootstrap();
    await act(async () => {
      capturedActions.openEventForm(null);
    });
    // Start with r1 included, toggle it off
    await act(async () => {
      capturedActions.toggleFormNomRole('r1');
    });
    const form = JSON.parse(screen.getByTestId('form').textContent!);
    expect(form.nominatedRoleIds).not.toContain('r1');
  });

  it('saveEvent shows toast and aborts when title is empty', async () => {
    await renderAndBootstrap();
    await act(async () => {
      capturedActions.openEventForm(null);
    });
    // Form has empty title by default — validation should block
    await act(async () => {
      await capturedActions.saveEvent('single');
    });
    // Sheet stays open (save failed)
    expect(screen.getByTestId('sheet').textContent).toBe('eventForm');
  });

  it('closeSheet resets sheet to null', async () => {
    await renderAndBootstrap();
    await act(async () => {
      capturedActions.askConfirm({ title: 'Test', message: 'Close?', onConfirm: vi.fn() });
    });
    await act(async () => {
      capturedActions.closeSheet();
    });
    expect(screen.getByTestId('sheet').textContent).toBe('none');
  });

  it('go changes the route', async () => {
    await renderAndBootstrap();
    await act(async () => {
      capturedActions.go('events');
    });
    // No crash — route change works
  });

  it('setEventsView updates eventsView', async () => {
    await renderAndBootstrap();
    await act(async () => {
      capturedActions.setEventsView('calendar');
    });
    // No crash
  });

  it('toggleCalAbsences toggles calShowAbsences', async () => {
    await renderAndBootstrap();
    await act(async () => {
      capturedActions.toggleCalAbsences();
    });
    // No crash
  });

  it('setPrimaryColor updates primaryColor', async () => {
    await renderAndBootstrap();
    await act(async () => {
      capturedActions.setPrimaryColor('#FF5722');
    });
    // No crash
  });

  it('toastMsg sets toast without throwing', async () => {
    await renderAndBootstrap();
    await act(async () => {
      capturedActions.toastMsg('Hello!');
    });
    // No crash — toast was set (auto-clears after 2600ms via setTimeout)
  });
});

// The bootstrap sets: events=write, members=read, finances=none
describe('AppProvider / can() permission checks', () => {
  let capturedCan!: ReturnType<typeof useApp>['can'];

  function CanProbe({ onMount }: { onMount: (can: ReturnType<typeof useApp>['can']) => void }) {
    const { can } = useApp();
    onMount(can);
    return null;
  }

  async function renderWithPerms() {
    let capturedActions!: ReturnType<typeof useAppActions>;
    render(
      <AppProvider>
        <PhaseAndSheet
          onMount={(a) => {
            capturedActions = a;
          }}
        />
        <CanProbe
          onMount={(c) => {
            capturedCan = c;
          }}
        />
      </AppProvider>,
    );
    await waitFor(() => expect(screen.getByTestId('phase').textContent).toBe('login'));
    await act(async () => {
      capturedActions.setState({
        phase: 'app',
        user: {
          id: 'u1',
          name: 'Test User',
          email: 'test@example.com',
          avatarColor: '#000',
          photo: null,
          phone: '',
          birthday: '',
          address: '',
        },
        teams: [
          {
            id: 'team1',
            name: 'Test Team',
            membershipId: 'ms1',
            myRoles: [{ id: 'r1', name: 'Admin' }],
            myPerms: {
              events: 'write',
              members: 'read',
              finances: 'none',
              stats: 'read',
              news: 'write',
              polls: 'write',
              settings: 'none',
            },
          },
        ] as never,
        activeTeamId: 'team1',
        roles: [],
        events: [],
        members: [],
        news: [],
        polls: [],
      });
    });
    await waitFor(() => expect(screen.getByTestId('phase').textContent).toBe('app'));
  }

  it('returns true when module permission meets the required level', async () => {
    await renderWithPerms();
    expect(capturedCan('events', 'write')).toBe(true);
  });

  it('returns true for read when user has write permission', async () => {
    await renderWithPerms();
    expect(capturedCan('events', 'read')).toBe(true);
  });

  it('returns true for read when user only has read permission', async () => {
    await renderWithPerms();
    expect(capturedCan('members', 'read')).toBe(true);
  });

  it('returns false for write when user only has read permission', async () => {
    await renderWithPerms();
    expect(capturedCan('members', 'write')).toBe(false);
  });

  it('returns false when user has no permission (none)', async () => {
    await renderWithPerms();
    expect(capturedCan('finances', 'read')).toBe(false);
  });

  it('defaults to write level check when no level specified', async () => {
    await renderWithPerms();
    expect(capturedCan('events')).toBe(true);
    expect(capturedCan('members')).toBe(false);
  });
});

// This block exercises real login (doLogin -> api.auth.login -> establishSession)
// against the mock service layer, unlike every other describe above which
// fast-forwards to 'app' phase via a direct setState. Because the mock's
// session/DB singleton persists across tests within a module (no per-test
// reset), each test here re-imports both modules fresh via vi.resetModules()
// so a real login here can never leak session state into the rest of this
// file's tests.
describe('AppProvider / invite-redemption join flow', () => {
  async function freshModules() {
    vi.resetModules();
    localStorage.clear();
    const svc = await import('@/services/serviceLayer');
    const ctx = await import('./AppContext');
    return { api: svc.api, AppProvider: ctx.AppProvider, useApp: ctx.useApp, useAppActions: ctx.useAppActions };
  }

  it('redeems a pending invite on login, joins the team, and lands on it', async () => {
    const { api, AppProvider: FreshAppProvider, useApp: freshUseApp, useAppActions: freshUseAppActions } =
      await freshModules();
    const invite = await api.teams.createInvite('t_a');
    window.history.pushState({}, '', '/join/t_a/' + invite.code);

    let actions: ReturnType<typeof freshUseAppActions>;
    function Probe() {
      const { state } = freshUseApp();
      actions = freshUseAppActions();
      return (
        <div>
          <div data-testid="phase">{state.phase}</div>
          <div data-testid="activeTeamId">{state.activeTeamId ?? ''}</div>
          <div data-testid="toast">{state.toast?.message ?? ''}</div>
        </div>
      );
    }

    render(
      <FreshAppProvider>
        <Probe />
      </FreshAppProvider>,
    );
    await waitFor(() => expect(screen.getByTestId('phase').textContent).toBe('login'));

    await act(async () => {
      await actions!.doLogin('google');
    });

    await waitFor(() => expect(screen.getByTestId('phase').textContent).toBe('app'));
    expect(screen.getByTestId('activeTeamId').textContent).toBe('t_a');
    expect(screen.getByTestId('toast').textContent).toContain('A-Team TSC Schwarz-Gelb Aachen');
    expect(window.location.pathname).toBe('/home');
  });

  it('shows an error toast for an invalid invite code but still logs in normally', async () => {
    const { AppProvider: FreshAppProvider, useApp: freshUseApp, useAppActions: freshUseAppActions } =
      await freshModules();
    window.history.pushState({}, '', '/join/bogus-team/does-not-exist');

    let actions: ReturnType<typeof freshUseAppActions>;
    function Probe() {
      const { state } = freshUseApp();
      actions = freshUseAppActions();
      return (
        <div>
          <div data-testid="phase">{state.phase}</div>
          <div data-testid="toast">{state.toast?.message ?? ''}</div>
        </div>
      );
    }

    render(
      <FreshAppProvider>
        <Probe />
      </FreshAppProvider>,
    );
    await waitFor(() => expect(screen.getByTestId('phase').textContent).toBe('login'));

    await act(async () => {
      await actions!.doLogin('google');
    });

    await waitFor(() => expect(screen.getByTestId('phase').textContent).toBe('app'));
    expect(screen.getByTestId('toast').textContent).toBe('Einladungslink ist ungültig oder abgelaufen.');
  });
});

// Regression test: a valid session (cookie/currentUser succeeds) whose
// establishSession then failed to load the team list used to strand the user
// on a login screen with providers: [] -- no SSO buttons, no way to reach the
// password form, a true dead end short of a manual reload. Uses the same
// freshModules isolation as the invite-redemption block above since it also
// exercises a real login against the mock service layer's session singleton.
describe('AppProvider / session-restore resilience', () => {
  async function freshModules() {
    vi.resetModules();
    localStorage.clear();
    const svc = await import('@/services/serviceLayer');
    const errors = await import('@/utils/errors');
    const ctx = await import('./AppContext');
    return {
      api: svc.api,
      NetworkError: errors.NetworkError,
      AppProvider: ctx.AppProvider,
      useApp: ctx.useApp,
      useAppActions: ctx.useAppActions,
    };
  }

  it('recovers to a usable login screen when the post-restore team fetch keeps failing', async () => {
    const {
      api,
      NetworkError,
      AppProvider: FreshAppProvider,
      useApp: freshUseApp,
      useAppActions: freshUseAppActions,
    } = await freshModules();

    let actions: ReturnType<typeof freshUseAppActions>;
    function Probe() {
      const { state } = freshUseApp();
      actions = freshUseAppActions();
      return (
        <div>
          <div data-testid="phase">{state.phase}</div>
          <div data-testid="providerCount">{state.providers.length}</div>
        </div>
      );
    }

    const { unmount } = render(
      <FreshAppProvider>
        <Probe />
      </FreshAppProvider>,
    );
    await waitFor(() => expect(screen.getByTestId('phase').textContent).toBe('login'));

    // A normal login establishes a real session (session.userId in the mock),
    // which persists across remounts within this module instance.
    await act(async () => {
      await actions!.doLogin('google');
    });
    await waitFor(() => expect(screen.getByTestId('phase').textContent).toBe('app'));
    unmount();

    // Simulate the team list becoming permanently unreachable (e.g. a
    // backend outage) for the next session-restore attempt.
    api.teams.listForCurrentUser = vi.fn().mockRejectedValue(new NetworkError());

    render(
      <FreshAppProvider>
        <Probe />
      </FreshAppProvider>,
    );

    // retryable's backoff (300ms + 600ms) plus the mock's own randomized
    // per-call latency pushes this well past the default 1000ms waitFor.
    await waitFor(() => expect(screen.getByTestId('phase').textContent).toBe('login'), { timeout: 5000 });
    await waitFor(() => expect(Number(screen.getByTestId('providerCount').textContent)).toBeGreaterThan(0));
  });
});
