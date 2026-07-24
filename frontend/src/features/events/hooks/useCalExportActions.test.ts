import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { renderHook, act } from '@testing-library/react';
import { useCalExportActions } from './useCalExportActions';
import { createQueryWrapper } from '@/test/queryTestUtils';
import { setLocale } from '@/i18n';
import type { AppState } from '@/context/AppContext';
import type { TeamEvent } from '../types';

function makeEvent(overrides: Partial<TeamEvent> = {}): TeamEvent {
  return {
    id: 'ev1',
    teamId: 'team1',
    type: 'training',
    title: 'Weekly Training',
    date: '2026-01-05',
    location: '',
    note: '',
    meetTime: '17:30',
    startTime: '18:00',
    endTime: null,
    meetTimeMandatory: false,
    responseMode: 'opt_in',
    recurring: false,
    seriesId: null,
    status: 'active',
    summary: { yes: 0, no: 0, maybe: 0, pending: 0 } as never,
    myStatus: 'pending',
    myAuto: false,
    myReason: '',
    ...overrides,
  };
}

function makeState(overrides: Partial<AppState> = {}): AppState {
  return {
    activeTeamId: 'team1',
    ...overrides,
  } as unknown as AppState;
}

describe('useCalExportActions', () => {
  let stateRef: AppState;
  let setState: ReturnType<typeof vi.fn>;
  let toastMsg: ReturnType<typeof vi.fn>;
  let activeTeam: ReturnType<typeof vi.fn>;
  let api: { events: { list: ReturnType<typeof vi.fn> } };
  let events: TeamEvent[];
  let capturedIcsText: string;

  beforeEach(() => {
    events = [makeEvent()];
    stateRef = makeState();
    setState = vi.fn();
    toastMsg = vi.fn();
    activeTeam = vi.fn(() => ({ id: 'team1', name: 'Test Team', short: 'tt' }) as never);
    api = { events: { list: vi.fn(() => Promise.resolve(events)) } };
    capturedIcsText = '';
    vi.stubGlobal(
      'Blob',
      vi.fn(function (this: unknown, parts: string[]) {
        capturedIcsText = parts.join('');
        return {};
      }),
    );
    vi.stubGlobal('URL', { createObjectURL: vi.fn(() => 'blob:mock'), revokeObjectURL: vi.fn() });
  });

  afterEach(() => {
    setLocale('de');
    vi.unstubAllGlobals();
    vi.useRealTimers();
  });

  function renderActions() {
    return renderHook(
      () =>
        useCalExportActions({
          api: api as never,
          S: () => stateRef,
          setState: setState as never,
          activeTeam: activeTeam as never,
          teamId: stateRef.activeTeamId,
          toastMsg: toastMsg as never,
        }),
      { wrapper: createQueryWrapper() },
    );
  }

  // Regression test: buildIcs used to hardcode German event-type/field
  // labels ('Training', 'Auftritt / Turnier', 'Team-Event', 'Treffen: ',
  // 'Typ: ') independent of the active UI locale, unlike the rest of the app
  // (e.g. src/styles/tokens.ts already uses t('eventType.*') for the same
  // event types). An English-locale user downloading their calendar got
  // German words baked permanently into every exported event description.
  it('uses the active locale for event-type and field labels in the exported ICS', async () => {
    setLocale('en');
    events = [makeEvent({ type: 'auftritt', meetTime: '17:30' })];
    const { result, rerender } = renderActions();
    await act(async () => {
      await Promise.resolve();
    });
    rerender();
    act(() => {
      result.current.downloadIcs();
    });

    expect(capturedIcsText).toContain('Performance / Tournament');
    expect(capturedIcsText).not.toContain('Auftritt / Turnier');
    expect(capturedIcsText).not.toContain('Treffen');
    expect(capturedIcsText).not.toContain('Typ:');
  });

  it('uses German labels when the locale is de', async () => {
    setLocale('de');
    events = [makeEvent({ type: 'auftritt', meetTime: '17:30' })];
    const { result, rerender } = renderActions();
    await act(async () => {
      await Promise.resolve();
    });
    rerender();
    act(() => {
      result.current.downloadIcs();
    });

    expect(capturedIcsText).toContain('Auftritt / Turnier');
  });
});

// Separate describe block (own local state/helpers) covering openCalExport,
// downloadIcs's cancelled-event filtering, and copyCalUrl -- the locale-label
// formatting above and this behavior were originally split across two files
// (the latter alongside useAbsenceActions' own tests, back when this hook's
// module was shared with the absences vertical); merged into one file once
// the absences migration moved its tests out to their own file, since nothing
// about splitting calExport coverage across two files was ever intentional.
describe('useCalExportActions (sheet/filter/copy)', () => {
  function makeFilterCopyState(overrides: Partial<AppState> = {}): AppState {
    return {
      phase: 'app',
      user: { id: 'u1', name: 'Test User', email: 'test@test.com', avatarColor: '#000', photo: null },
      activeTeamId: 'team1',
      sheet: null,
      form: {},
      formErrors: {},
      busy: null,
      toast: null,
      route: 'home',
      members: [],
      finances: null,
      stats: null,
      statsRange: null,
      news: [],
      polls: [],
      teams: [],
      roles: [],
      notifUnread: 0,
      notifications: [],
      primaryColor: '#000',
      ...overrides,
    } as unknown as AppState;
  }

  let setState: ReturnType<typeof vi.fn>;
  let toastMsg: ReturnType<typeof vi.fn>;
  let stateRef: AppState;
  let api: { events: { list: ReturnType<typeof vi.fn> } };

  const events = [
    {
      id: 'ev1',
      title: 'Training',
      date: '2026-03-01',
      type: 'training',
      status: 'active',
      startTime: null,
      endTime: null,
      meetTime: null,
      location: 'Halle',
      note: 'Bring boots',
    },
    {
      id: 'ev2',
      title: 'Cancelled',
      date: '2026-03-05',
      type: 'event',
      status: 'cancelled',
      startTime: null,
      endTime: null,
      meetTime: null,
      location: null,
      note: null,
    },
  ] as never[];

  beforeEach(() => {
    stateRef = makeFilterCopyState();
    setState = vi.fn((patch) => {
      if (typeof patch === 'function') {
        const result = patch(stateRef);
        stateRef = { ...stateRef, ...result };
      } else {
        stateRef = { ...stateRef, ...patch };
      }
    });
    toastMsg = vi.fn();
    api = { events: { list: vi.fn(() => Promise.resolve(events)) } };
  });

  function renderActions() {
    return renderHook(
      () =>
        useCalExportActions({
          api: api as never,
          S: () => stateRef,
          setState: setState as never,
          activeTeam: () => ({ id: 'team1', name: 'Test Team', short: 'TT' }) as never,
          teamId: stateRef.activeTeamId,
          toastMsg: toastMsg as never,
        }),
      { wrapper: createQueryWrapper() },
    );
  }

  it('openCalExport sets calExport sheet', () => {
    const { result } = renderActions();
    act(() => {
      result.current.openCalExport();
    });
    expect(setState).toHaveBeenCalledWith({ sheet: { type: 'calExport' } });
  });

  it('downloadIcs filters cancelled events and shows toast', async () => {
    URL.createObjectURL = vi.fn().mockReturnValue('blob:test');
    URL.revokeObjectURL = vi.fn();
    const { result, rerender } = renderActions();
    // Let useEventsQuery's fetch resolve and re-render before downloadIcs
    // closes over the fetched events.
    await act(async () => {
      await Promise.resolve();
    });
    rerender();
    act(() => {
      result.current.downloadIcs();
    });
    expect(toastMsg).toHaveBeenCalledWith('1 Termine als .ics exportiert');
  });

  const testFeedUrl = 'https://app.example.com/api/v1/calendar-feed/abc123.ics';

  it('copyCalUrl sets copied and shows toast', async () => {
    Object.assign(navigator, { clipboard: { writeText: vi.fn().mockResolvedValue(undefined) } });
    stateRef = makeFilterCopyState({ sheet: { type: 'calExport' } as never });
    const { result } = renderActions();
    await act(async () => {
      await result.current.copyCalUrl(testFeedUrl);
    });
    expect(toastMsg).toHaveBeenCalledWith('Abo-Link kopiert');
  });

  it('copyCalUrl shows an error toast when the clipboard write fails', async () => {
    Object.assign(navigator, { clipboard: { writeText: vi.fn().mockRejectedValue(new Error('denied')) } });
    stateRef = makeFilterCopyState({ sheet: { type: 'calExport' } as never });
    const { result } = renderActions();
    await act(async () => {
      await result.current.copyCalUrl(testFeedUrl);
    });
    expect(toastMsg).toHaveBeenCalledWith('Kopieren fehlgeschlagen', undefined, 'error');
  });

  // Regression test: the sheet update used to check only sheet.type ===
  // 'calExport', never the team. If the user switched teams and reopened
  // the calExport sheet (also type 'calExport') for the new team before a
  // slow clipboard write for the old team resolved, the stale resolution
  // would show "Copied!" on the new team's sheet even though nothing was
  // copied for it.
  it("does not mark a different team's calExport sheet as copied after a slow clipboard write resolves", async () => {
    let resolveWrite!: () => void;
    Object.assign(navigator, {
      clipboard: { writeText: vi.fn(() => new Promise<void>((resolve) => (resolveWrite = resolve))) },
    });
    stateRef = makeFilterCopyState({ sheet: { type: 'calExport' } as never });
    const { result } = renderActions();

    let copyPromise!: Promise<void>;
    act(() => {
      copyPromise = result.current.copyCalUrl(testFeedUrl);
    });

    // User switches teams and opens THAT team's own (also empty) calExport sheet.
    stateRef = { ...stateRef, activeTeamId: 'team2', sheet: { type: 'calExport' } as never };

    await act(async () => {
      resolveWrite();
      await copyPromise;
    });

    expect(stateRef.sheet).toEqual({ type: 'calExport' });
  });
});
