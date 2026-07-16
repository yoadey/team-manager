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
