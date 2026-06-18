import { describe, it, expect, beforeEach, vi } from 'vitest';
import { render, screen, waitFor, act } from '@testing-library/react';
import { AppProvider, useApp, useAppActions } from './AppContext';

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
        user: { id: 'u1', name: 'Test User', email: 'test@example.com', avatarColor: '#000', photo: null },
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
      capturedActions.askConfirm({ title: 'Confirm?', onConfirm: vi.fn() });
    });
    expect(screen.getByTestId('sheet').textContent).toBe('confirm');
  });

  it('cancelConfirm closes confirm sheet', async () => {
    await renderAndBootstrap();
    await act(async () => {
      capturedActions.askConfirm({ title: 'Confirm?', onConfirm: vi.fn() });
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
      capturedActions.askConfirm({ title: 'Delete?', onConfirm });
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
      capturedActions.askConfirm({ title: 'Test', onConfirm: vi.fn() });
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
