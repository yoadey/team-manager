import { describe, it, expect, beforeEach } from 'vitest';
import { render, screen, waitFor, act } from '@testing-library/react';
import { AppProvider, useApp, useAppActions } from './AppContext';

beforeEach(() => localStorage.clear());

function Probe({ onActions }: { onActions: (ref: object) => void }) {
  const { state } = useApp();
  const actions = useAppActions();
  onActions(actions);
  return <div data-testid="phase">{state.phase}</div>;
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
    // Wait for the bootstrap state change (loading -> login) to force a re-render.
    await waitFor(() => expect(screen.getByTestId('phase').textContent).toBe('login'));
    // Allow any pending state updates to flush.
    await act(async () => {});
    expect(seen.length).toBeGreaterThan(1);
    // Every observed actions reference must be identical (stable identity).
    expect(seen.every((ref) => ref === seen[0])).toBe(true);
  });
});
