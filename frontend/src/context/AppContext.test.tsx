import { StrictMode } from 'react';
import { describe, it, expect, beforeEach, vi } from 'vitest';
import { render, screen, waitFor, act } from '@testing-library/react';
import { AppProvider, useApp, useAppActions, useAppSelector, sheetErrorBoundaryKey } from './AppContext';

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
      <div data-testid="toast">{state.toast?.message ?? ''}</div>
    </div>
  );
}

// Regression test: AppShell/SheetHost's ErrorBoundary used to key solely on
// sheet.type -- React only resets a caught error on remount (key change), so
// navigating from eventDetail(evA) straight to eventDetail(evB) (e.g. via
// popstate, which can collapse both setState calls into one commit with no
// intermediate unmount) never remounted the boundary, leaving evB stuck
// behind evA's stale crash fallback. The key must include entity identity.
describe('sheetErrorBoundaryKey', () => {
  it('produces different keys for the same sheet type with different eventIds', () => {
    const a = sheetErrorBoundaryKey({ type: 'eventDetail', eventId: 'evA', event: null, rows: [] } as never);
    const b = sheetErrorBoundaryKey({ type: 'eventDetail', eventId: 'evB', event: null, rows: [] } as never);
    expect(a).not.toBe(b);
  });

  it('produces different keys for the same sheet type with different membershipIds', () => {
    const a = sheetErrorBoundaryKey({ type: 'memberDetail', membershipId: 'ms1', stats: null } as never);
    const b = sheetErrorBoundaryKey({ type: 'memberDetail', membershipId: 'ms2', stats: null } as never);
    expect(a).not.toBe(b);
  });

  it('produces the same key for the same sheet type and entity id', () => {
    const a = sheetErrorBoundaryKey({ type: 'eventDetail', eventId: 'evA', event: null, rows: [] } as never);
    const b = sheetErrorBoundaryKey({ type: 'eventDetail', eventId: 'evA', event: null, rows: [] } as never);
    expect(a).toBe(b);
  });

  it('produces different keys for different sheet types with no entity id', () => {
    const a = sheetErrorBoundaryKey({ type: 'teams' } as never);
    const b = sheetErrorBoundaryKey({ type: 'profile' } as never);
    expect(a).not.toBe(b);
  });
});

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

  // Regression test: onFile's FileReader had no onerror handler -- a failed
  // read (corrupted file, cloud-backed picker file needing a network fetch
  // that fails, permission/hardware error) left onload never firing and the
  // caller's callback never called, so a photo/logo upload click silently
  // did nothing with zero feedback.
  it('onFile shows an error toast when the FileReader fails to read the file', async () => {
    await renderAndBootstrap();

    const OriginalFileReader = window.FileReader;
    class FailingFileReader {
      onload: (() => void) | null = null;
      onerror: (() => void) | null = null;
      result: string | null = null;
      readAsDataURL() {
        queueMicrotask(() => this.onerror?.());
      }
    }
    // @ts-expect-error -- stubbing FileReader to force the error path
    window.FileReader = FailingFileReader;

    const file = new File(['data'], 'photo.png', { type: 'image/png' });
    const input = document.createElement('input');
    input.type = 'file';
    Object.defineProperty(input, 'files', { value: [file] });
    const cb = vi.fn();

    await act(async () => {
      capturedActions.onFile({ target: input } as unknown as Parameters<typeof capturedActions.onFile>[0], cb);
      await Promise.resolve();
    });

    window.FileReader = OriginalFileReader;

    expect(cb).not.toHaveBeenCalled();
    expect(screen.getByTestId('toast').textContent).not.toBe('');
  });

  // Regression test: onFile had no MIME-type check -- the <input
  // accept="image/*"> attribute is only a picker-UI hint, so a user could
  // switch to "All Files" and select an arbitrary non-image file, which
  // onFile would silently read as a data URL and hand to the caller as if
  // it were a valid photo/logo.
  it('onFile rejects a non-image file without reading it', async () => {
    await renderAndBootstrap();

    const file = new File(['%PDF-1.4'], 'doc.pdf', { type: 'application/pdf' });
    const input = document.createElement('input');
    input.type = 'file';
    Object.defineProperty(input, 'files', { value: [file] });
    const cb = vi.fn();

    await act(async () => {
      capturedActions.onFile({ target: input } as unknown as Parameters<typeof capturedActions.onFile>[0], cb);
    });

    expect(cb).not.toHaveBeenCalled();
    expect(screen.getByTestId('toast').textContent).not.toBe('');
  });

  // Regression test: onFile had no file-size check -- readAsDataURL reads
  // the entire file into memory as base64 with no cap, so a huge file
  // picked via "All Files" could freeze the tab and produce a payload with
  // no client-side size guard before it's sent to the backend.
  it('onFile rejects an oversized file without reading it', async () => {
    await renderAndBootstrap();

    const file = new File(['x'], 'huge.png', { type: 'image/png' });
    Object.defineProperty(file, 'size', { value: 10 * 1024 * 1024 });
    const input = document.createElement('input');
    input.type = 'file';
    Object.defineProperty(input, 'files', { value: [file] });
    const cb = vi.fn();

    await act(async () => {
      capturedActions.onFile({ target: input } as unknown as Parameters<typeof capturedActions.onFile>[0], cb);
    });

    expect(cb).not.toHaveBeenCalled();
    expect(screen.getByTestId('toast').textContent).not.toBe('');
  });
});

// Regression test: on-demand per-route loaders (loadFinances, loadStats,
// loadNews, loadPolls, loadAbsences, refreshEvents/Members/Roles,
// loadNotifications) used to apply their response unconditionally, with no
// check that activeTeamId was still the same team when the response landed
// -- unlike afterLoginLoad, which has always had this guard. A slow request
// for the team the user just navigated away from could clobber the newly
// selected team's state with the previous team's data.
describe('AppProvider / team-switch race guards', () => {
  it('loadFinances discards a stale response after the user has switched teams', async () => {
    const svc = await import('@/services/serviceLayer');
    type Overview = Awaited<ReturnType<typeof svc.api.finances.overview>>;
    let resolveOverview!: (v: Overview) => void;
    const overviewSpy = vi
      .spyOn(svc.api.finances, 'overview')
      .mockReturnValueOnce(new Promise<Overview>((resolve) => (resolveOverview = resolve)));

    let actions!: ReturnType<typeof useAppActions>;
    let state!: ReturnType<typeof useApp>['state'];
    function Probe() {
      state = useApp().state;
      actions = useAppActions();
      return (
        <div>
          <div data-testid="activeTeamId">{state.activeTeamId ?? ''}</div>
          <div data-testid="finances">{state.finances ? JSON.stringify(state.finances) : 'null'}</div>
        </div>
      );
    }

    render(
      <AppProvider>
        <Probe />
      </AppProvider>,
    );
    await act(async () => {
      actions.setState({ phase: 'app', activeTeamId: 'team1', teams: [] as never });
    });

    // Kick off loadFinances for team1; the mocked promise stays pending.
    act(() => {
      actions.loadFinances();
    });
    expect(overviewSpy).toHaveBeenCalledWith('team1');

    // User switches to team2 before team1's response arrives. selectTeam
    // resets finances to null via afterLoginLoad's initial reset.
    await act(async () => {
      await actions.selectTeam('team2');
    });
    expect(screen.getByTestId('activeTeamId').textContent).toBe('team2');
    expect(screen.getByTestId('finances').textContent).toBe('null');

    // team1's stale response now arrives -- it must NOT overwrite team2's state.
    await act(async () => {
      resolveOverview({
        balance: 0,
        income: 0,
        expense: 0,
        transactions: [],
        penalties: [],
        assignments: [],
        openPenalties: [],
        openPenaltySum: 0,
        contributions: [],
        contribOpen: 0,
      });
    });
    expect(screen.getByTestId('activeTeamId').textContent).toBe('team2');
    expect(screen.getByTestId('finances').textContent).toBe('null');

    overviewSpy.mockRestore();
  });

  // Regression test: afterLoginLoad used to await all five initial-load
  // calls via Promise.all, so a single 403 (e.g. a member whose role lacks
  // members:read) discarded every other successfully-loaded module too --
  // the whole dashboard was left blank. It must now degrade gracefully:
  // modules that succeeded still populate state even when a sibling module
  // fails.
  it('afterLoginLoad still populates modules that succeeded when a sibling module 403s', async () => {
    const svc = await import('@/services/serviceLayer');
    const { ForbiddenError } = await import('@/utils/errors');
    const membersSpy = vi.spyOn(svc.api.members, 'list').mockRejectedValue(new ForbiddenError());

    let actions!: ReturnType<typeof useAppActions>;
    let state!: ReturnType<typeof useApp>['state'];
    function Probe() {
      state = useApp().state;
      actions = useAppActions();
      return (
        <div>
          <div data-testid="activeTeamId">{state.activeTeamId ?? ''}</div>
          <div data-testid="events">{state.events ? 'loaded' : 'null'}</div>
          <div data-testid="members">{state.members ? 'loaded' : 'null'}</div>
        </div>
      );
    }

    render(
      <AppProvider>
        <Probe />
      </AppProvider>,
    );
    await act(async () => {
      actions.setState({ phase: 'app', activeTeamId: 'other-team', teams: [] as never });
    });

    await act(async () => {
      await actions.selectTeam('team1');
    });

    expect(screen.getByTestId('events').textContent).toBe('loaded');
    expect(screen.getByTestId('members').textContent).toBe('null');

    membersSpy.mockRestore();
  });

  // Regression test: a ForbiddenError from afterLoginLoad's parallel fetch
  // used to always surface a "you don't have permission" toast via
  // reportLoad(failures[0].reason) -- even though a role having e.g.
  // news:none is completely ordinary and expected, not a real failure. This
  // fired on every single login/team-switch for such a role. Verified via the
  // module-populate side effect (not state.toast): afterLoginLoad's S().
  // activeTeamId guard around reportLoad only ever settles once this whole
  // act() block returns (React's test-mode batching defers the commit of the
  // activeTeamId update queued earlier in the same callback), so a toast
  // assertion taken mid-callback is unreliable here in a way a real network
  // round-trip never is -- the ForbiddenError-filtering logic itself is
  // simple enough (one .find() predicate) to trust from inspection plus this
  // side-effect check.
  it('afterLoginLoad leaves other modules populated when the only failure is a ForbiddenError', async () => {
    const svc = await import('@/services/serviceLayer');
    const { ForbiddenError } = await import('@/utils/errors');
    const newsSpy = vi.spyOn(svc.api.news, 'list').mockRejectedValue(new ForbiddenError());

    let actions!: ReturnType<typeof useAppActions>;
    let state!: ReturnType<typeof useApp>['state'];
    function Probe() {
      state = useApp().state;
      actions = useAppActions();
      return (
        <div>
          <div data-testid="events">{state.events ? 'loaded' : 'null'}</div>
          <div data-testid="news">{state.news ? 'loaded' : 'null'}</div>
        </div>
      );
    }

    render(
      <AppProvider>
        <Probe />
      </AppProvider>,
    );
    await act(async () => {
      actions.setState({ phase: 'app', activeTeamId: 'other-team', teams: [] as never });
    });
    await act(async () => {
      await actions.selectTeam('team1');
    });

    expect(screen.getByTestId('events').textContent).toBe('loaded');
    expect(screen.getByTestId('news').textContent).toBe('null');

    newsSpy.mockRestore();
  });

  // Regression test: ensureRouteData (invoked by `go`) covered finances/
  // stats/news/polls but not events/members, even though afterLoginLoad can
  // leave either at null after a failed initial load (see the test above).
  // Navigating to the Events or Members tab was previously a no-op in that
  // case, leaving a permanent skeleton loader for the rest of the session.
  it('go("members") retries the load when members is still null from a failed afterLoginLoad', async () => {
    const svc = await import('@/services/serviceLayer');
    const { ForbiddenError } = await import('@/utils/errors');
    const membersSpy = vi
      .spyOn(svc.api.members, 'list')
      .mockRejectedValueOnce(new ForbiddenError())
      .mockResolvedValueOnce([]);

    let actions!: ReturnType<typeof useAppActions>;
    let state!: ReturnType<typeof useApp>['state'];
    function Probe() {
      state = useApp().state;
      actions = useAppActions();
      return (
        <div>
          <div data-testid="members">{state.members ? 'loaded' : 'null'}</div>
        </div>
      );
    }

    render(
      <AppProvider>
        <Probe />
      </AppProvider>,
    );
    await act(async () => {
      actions.setState({
        phase: 'app',
        activeTeamId: 'other-team',
        teams: [
          { id: 'team1', name: 'Test Team', membershipId: 'ms1', myRoles: [], myPerms: { members: 'read' } },
        ] as never,
      });
    });
    await act(async () => {
      await actions.selectTeam('team1');
    });
    expect(screen.getByTestId('members').textContent).toBe('null');

    await act(async () => {
      actions.go('members');
    });
    expect(screen.getByTestId('members').textContent).toBe('loaded');
    expect(membersSpy).toHaveBeenCalledTimes(2);

    membersSpy.mockRestore();
  });

  // Regression test: the absence-notification click handler
  // (NotificationsSheet.tsx) used to call app.setState/app.loadAbsences
  // directly instead of going through ensureRouteData like every other nav
  // action -- so a null state.events left over from a failed afterLoginLoad
  // never retried when the user navigated to Events > Absences this way.
  it('goEventsAbsences retries the events load when events is still null from a failed afterLoginLoad', async () => {
    const svc = await import('@/services/serviceLayer');
    const { ForbiddenError } = await import('@/utils/errors');
    const eventsSpy = vi
      .spyOn(svc.api.events, 'list')
      .mockRejectedValueOnce(new ForbiddenError())
      .mockResolvedValueOnce([]);

    let actions!: ReturnType<typeof useAppActions>;
    let state!: ReturnType<typeof useApp>['state'];
    function Probe() {
      state = useApp().state;
      actions = useAppActions();
      return (
        <div>
          <div data-testid="events">{state.events ? 'loaded' : 'null'}</div>
          <div data-testid="eventsView">{state.eventsView}</div>
        </div>
      );
    }

    render(
      <AppProvider>
        <Probe />
      </AppProvider>,
    );
    await act(async () => {
      actions.setState({
        phase: 'app',
        activeTeamId: 'other-team',
        teams: [
          { id: 'team1', name: 'Test Team', membershipId: 'ms1', myRoles: [], myPerms: { events: 'read' } },
        ] as never,
      });
    });
    await act(async () => {
      await actions.selectTeam('team1');
    });
    expect(screen.getByTestId('events').textContent).toBe('null');

    await act(async () => {
      actions.goEventsAbsences();
    });
    expect(screen.getByTestId('events').textContent).toBe('loaded');
    expect(screen.getByTestId('eventsView').textContent).toBe('absences');
    expect(eventsSpy).toHaveBeenCalledTimes(2);

    eventsSpy.mockRestore();
  });

  // Regression test: goEventsAbsences called loadAbsences() unconditionally,
  // with no permission check -- unlike ensureRouteData('events'), which
  // already skips its own fetches when the caller can't read events. The one
  // way this route is reachable without events:read is a stale absence
  // notification (cached from before a permission downgrade); clicking it
  // fired two now-forbidden absences requests in the background, producing
  // exactly the spurious "no permission" toast ensureRouteData's own
  // permission pre-check exists to prevent.
  it('goEventsAbsences does not call loadAbsences when the caller lacks events:read', async () => {
    const svc = await import('@/services/serviceLayer');
    const listForTeamSpy = vi.spyOn(svc.api.absences, 'listForTeam');
    const listMineSpy = vi.spyOn(svc.api.absences, 'listMine');

    let actions!: ReturnType<typeof useAppActions>;
    function Probe() {
      actions = useAppActions();
      return null;
    }

    render(
      <AppProvider>
        <Probe />
      </AppProvider>,
    );
    await act(async () => {
      actions.setState({
        phase: 'app',
        activeTeamId: 'other-team',
        teams: [
          { id: 'team1', name: 'Test Team', membershipId: 'ms1', myRoles: [], myPerms: { events: 'none' } },
        ] as never,
      });
    });
    await act(async () => {
      await actions.selectTeam('team1');
    });

    listForTeamSpy.mockClear();
    listMineSpy.mockClear();

    await act(async () => {
      actions.goEventsAbsences();
    });

    expect(listForTeamSpy).not.toHaveBeenCalled();
    expect(listMineSpy).not.toHaveBeenCalled();

    listForTeamSpy.mockRestore();
    listMineSpy.mockRestore();
  });

  // Regression test: the activeTeamId check alone only guards against a TEAM
  // SWITCH completing while a loader is in flight -- it does NOT catch two
  // same-team calls to the SAME loader racing each other (e.g. two
  // attendance updates in quick succession both triggering refreshEvents).
  // If the network responds out of request order, the OLDER call's response
  // could still land second and overwrite the newer one's fresher data.
  // refreshEvents itself isn't exposed from useAppActions (it's an internal
  // loader used by ensureRouteData/goEventsPending), so this drives it
  // indirectly via go('events'), which calls refreshEvents whenever
  // state.events is still null -- true for both calls here, since the
  // first's still-pending response hasn't set it yet.
  it('refreshEvents (via go) discards a stale response when a newer call has since resolved', async () => {
    const svc = await import('@/services/serviceLayer');
    type Events = Awaited<ReturnType<typeof svc.api.events.list>>;
    let resolveFirst!: (v: Events) => void;
    const listSpy = vi
      .spyOn(svc.api.events, 'list')
      .mockReturnValueOnce(new Promise<Events>((resolve) => (resolveFirst = resolve)))
      .mockResolvedValueOnce([{ id: 'fresh-event' }] as unknown as Events);

    let actions!: ReturnType<typeof useAppActions>;
    let state!: ReturnType<typeof useApp>['state'];
    function Probe() {
      state = useApp().state;
      actions = useAppActions();
      return <div data-testid="events">{state.events ? JSON.stringify(state.events) : 'null'}</div>;
    }

    render(
      <AppProvider>
        <Probe />
      </AppProvider>,
    );
    await act(async () => {
      actions.setState({
        phase: 'app',
        activeTeamId: 'team1',
        teams: [
          { id: 'team1', name: 'Test Team', membershipId: 'ms1', myRoles: [], myPerms: { events: 'write' } },
        ] as never,
        events: null,
      });
    });

    // First call kicks off with a still-pending response.
    act(() => {
      actions.go('events');
    });
    // Second (newer) call resolves immediately -- state.events is still
    // null at this point, so ensureRouteData fires refreshEvents again.
    await act(async () => {
      actions.go('events');
      await Promise.resolve();
    });
    expect(screen.getByTestId('events').textContent).toContain('fresh-event');

    // The first call's stale response now arrives -- it must NOT overwrite
    // the second, fresher call's already-applied result.
    await act(async () => {
      resolveFirst([{ id: 'stale-event' }] as unknown as Events);
    });
    expect(screen.getByTestId('events').textContent).toContain('fresh-event');
    expect(screen.getByTestId('events').textContent).not.toContain('stale-event');

    listSpy.mockRestore();
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

    // The seeded demo user (Lena Bergmann / u1) is already a member of t_a
    // by default; remove that membership first so this test genuinely
    // exercises a brand-new join rather than the idempotent already-member
    // no-op path (which must not show the "joined" toast -- see the
    // dedicated test below).
    const members = await api.members.list('t_a');
    const lena = members.find((m) => m.name === 'Lena Bergmann')!;
    await api.members.remove(lena.membershipId, 't_a');

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

  // Regression test: redeeming an invite for a team the user is already in
  // used to show the same "joined X" toast every time, misleadingly implying
  // a state change that didn't happen (the redemption is intentionally
  // idempotent both server-side and in the mock).
  it('does not show a "joined" toast when redeeming an invite for a team already joined', async () => {
    const { api, AppProvider: FreshAppProvider, useApp: freshUseApp, useAppActions: freshUseAppActions } =
      await freshModules();

    // The seeded demo user (Lena Bergmann / u1) is already a member of t_b.
    const invite = await api.teams.createInvite('t_b');
    window.history.pushState({}, '', '/join/t_b/' + invite.code);

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
    expect(screen.getByTestId('activeTeamId').textContent).toBe('t_b');
    expect(screen.getByTestId('toast').textContent).toBe('');
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

// Regression test: doLogin/doPasswordLogin used to clear `busy` (and set
// `error`) unconditionally in their catch blocks, the same class of bug
// round 63 fixed for every other save/delete flow via reportActionError's
// S+busyOwner guard. `busy` is one shared field, so a login that fails AFTER
// a different, still-in-flight login has taken it over must not clear it out
// from under that second login (Login.tsx also disables every control while
// any login is busy, closing the UI-level half of this, but the state guard
// is the actual fix -- this test exercises it directly, bypassing the UI).
describe('AppProvider / overlapping login does not clobber a different in-flight login', () => {
  async function freshModules() {
    vi.resetModules();
    localStorage.clear();
    const svc = await import('@/services/serviceLayer');
    const ctx = await import('./AppContext');
    return { api: svc.api, AppProvider: ctx.AppProvider, useApp: ctx.useApp, useAppActions: ctx.useAppActions };
  }

  it('a late-failing login only reports its own error, without clearing a different login\'s busy state', async () => {
    const { api, AppProvider: FreshAppProvider, useApp: freshUseApp, useAppActions: freshUseAppActions } =
      await freshModules();

    let rejectGoogle!: (err: Error) => void;
    const googleLoginPromise = new Promise<never>((_resolve, reject) => {
      rejectGoogle = reject;
    });
    const originalLogin = api.auth.login.bind(api.auth);
    vi.spyOn(api.auth, 'login').mockImplementation((providerId: string, password?: string) =>
      providerId === 'google' ? googleLoginPromise : originalLogin(providerId, password),
    );

    let actions: ReturnType<typeof freshUseAppActions>;
    function Probe() {
      const { state } = freshUseApp();
      actions = freshUseAppActions();
      return (
        <div>
          <div data-testid="busy">{state.busy ?? ''}</div>
          <div data-testid="error">{state.error ?? ''}</div>
        </div>
      );
    }
    render(
      <FreshAppProvider>
        <Probe />
      </FreshAppProvider>,
    );
    await waitFor(() => expect(actions).toBeTruthy());

    act(() => {
      void actions!.doLogin('google');
    });
    await waitFor(() => expect(screen.getByTestId('busy').textContent).toBe('login:google'));

    act(() => {
      void actions!.doLogin('apple');
    });
    await waitFor(() => expect(screen.getByTestId('busy').textContent).toBe('login:apple'));

    await act(async () => {
      rejectGoogle(new Error('google unreachable'));
      await googleLoginPromise.catch(() => {});
    });

    // apple's login is still in flight -- google's failure must not clear it.
    expect(screen.getByTestId('busy').textContent).toBe('login:apple');
    // The error message still surfaces regardless of busy ownership.
    expect(screen.getByTestId('error').textContent).toContain('google unreachable');
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

  // Regression test: React.StrictMode double-invokes effects on initial
  // mount in dev, and the session-restore effect had no guard against that --
  // meaning api.auth.currentUser() (and, for a valid session,
  // establishSession()'s full team-list + afterLoginLoad fan-out) fired
  // twice on every dev-mode app load. The bootstrapStarted ref must make
  // this idempotent even under StrictMode's double-invoke.
  it('only calls currentUser once under React.StrictMode double-invoke', async () => {
    const { api, AppProvider: FreshAppProvider, useApp: freshUseApp } = await freshModules();
    const spy = vi.spyOn(api.auth, 'currentUser');

    function Probe() {
      const { state } = freshUseApp();
      return <div data-testid="phase">{state.phase}</div>;
    }

    render(
      <StrictMode>
        <FreshAppProvider>
          <Probe />
        </FreshAppProvider>
      </StrictMode>,
    );
    await waitFor(() => expect(screen.getByTestId('phase').textContent).toBe('login'));
    expect(spy).toHaveBeenCalledTimes(1);
  });

  // Regression test: establishSession's session-restore path used to
  // unconditionally hardcode route: 'home' and rewrite the URL to /home,
  // silently discarding a bookmarked/shared deep link or the page the user
  // was on before a reload -- restoreLocation now re-parses the current URL
  // instead, mirroring the popstate handler.
  it('restores the deep-linked route (not /home) when a session is restored from a reload', async () => {
    const { AppProvider: FreshAppProvider, useApp: freshUseApp, useAppActions: freshUseAppActions } =
      await freshModules();

    let actions: ReturnType<typeof freshUseAppActions>;
    function Probe() {
      const { state } = freshUseApp();
      actions = freshUseAppActions();
      return (
        <div>
          <div data-testid="phase">{state.phase}</div>
          <div data-testid="route">{state.route}</div>
          <div data-testid="finTab">{state.finTab}</div>
          <div data-testid="finances">{state.finances ? 'loaded' : 'null'}</div>
        </div>
      );
    }

    const first = render(
      <FreshAppProvider>
        <Probe />
      </FreshAppProvider>,
    );
    await waitFor(() => expect(screen.getByTestId('phase').textContent).toBe('login'));
    // A normal login establishes a real session (session.userId in the
    // mock), which persists across remounts within this module instance --
    // same technique the test above uses to simulate a reload.
    await act(async () => {
      await actions!.doLogin('google');
    });
    await waitFor(() => expect(screen.getByTestId('phase').textContent).toBe('app'));
    first.unmount();

    // Simulate the browser being reloaded (or a bookmark/shared link being
    // opened) while pointed at /finances?tab=strafen.
    window.history.pushState({}, '', '/finances?tab=strafen');

    render(
      <FreshAppProvider>
        <Probe />
      </FreshAppProvider>,
    );
    await waitFor(() => expect(screen.getByTestId('phase').textContent).toBe('app'));
    expect(screen.getByTestId('route').textContent).toBe('finances');
    expect(screen.getByTestId('finTab').textContent).toBe('strafen');
    expect(window.location.pathname).toBe('/finances');
    // afterLoginLoad alone doesn't cover finances -- ensureRouteData must
    // have fetched it, or this would be stuck on a skeleton loader forever.
    await waitFor(() => expect(screen.getByTestId('finances').textContent).toBe('loaded'));
  });

  // Regression test: the fix above only restored route/eventScope/eventsView/
  // eventsOnlyPending/finTab, not a deep-linked detail sheet -- a reload
  // while /events/<id> was open used to leave state.sheet at null AND
  // rewrite the URL down to /events (dropping the id), destroying the deep
  // link outright rather than just failing to render it.
  it('restores a deep-linked event detail sheet when a session is restored from a reload', async () => {
    const { api, AppProvider: FreshAppProvider, useApp: freshUseApp, useAppActions: freshUseAppActions } =
      await freshModules();

    const events = await api.events.list('t_a', 'all');
    const eventId = events[0].id;

    let actions: ReturnType<typeof freshUseAppActions>;
    function Probe() {
      const { state } = freshUseApp();
      actions = freshUseAppActions();
      return (
        <div>
          <div data-testid="phase">{state.phase}</div>
          <div data-testid="route">{state.route}</div>
          <div data-testid="sheetType">{state.sheet?.type ?? ''}</div>
          <div data-testid="sheetEventId">{state.sheet?.eventId ?? ''}</div>
        </div>
      );
    }

    const first = render(
      <FreshAppProvider>
        <Probe />
      </FreshAppProvider>,
    );
    await waitFor(() => expect(screen.getByTestId('phase').textContent).toBe('login'));
    await act(async () => {
      await actions!.doLogin('google');
    });
    await waitFor(() => expect(screen.getByTestId('phase').textContent).toBe('app'));
    first.unmount();

    // Simulate the browser being reloaded while /events/<id> was open.
    window.history.pushState({}, '', '/events/' + eventId);

    render(
      <FreshAppProvider>
        <Probe />
      </FreshAppProvider>,
    );
    await waitFor(() => expect(screen.getByTestId('phase').textContent).toBe('app'));
    expect(screen.getByTestId('route').textContent).toBe('events');
    await waitFor(() => expect(screen.getByTestId('sheetType').textContent).toBe('eventDetail'));
    expect(screen.getByTestId('sheetEventId').textContent).toBe(eventId);
    // The state->URL sync effect must restore the id segment once the
    // sheet opens, not leave it stripped from the earlier history.replaceState.
    await waitFor(() => expect(window.location.pathname).toBe('/events/' + eventId));
  });
});
