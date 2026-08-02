import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';

const setUserMock = vi.fn();
const initMock = vi.fn();

vi.mock('@sentry/react', () => ({
  init: (...args: unknown[]) => initMock(...args),
  setUser: (...args: unknown[]) => setUserMock(...args),
  captureException: vi.fn(),
  browserTracingIntegration: vi.fn(() => ({})),
}));

// initMonitoring/setSentryUser both read config.sentryDsn once via the
// already-imported `config` singleton, so each scenario needs a fresh
// module instance (vi.resetModules + dynamic import) to pick up a
// different runtime DSN — same pattern as config.test.ts.
async function loadMonitoringWithDsn(dsn: string) {
  vi.resetModules();
  window.__RUNTIME_CONFIG__ = { SENTRY_DSN: dsn };
  return import('./monitoring');
}

beforeEach(() => {
  setUserMock.mockClear();
  initMock.mockClear();
});

afterEach(() => {
  delete (window as { __RUNTIME_CONFIG__?: unknown }).__RUNTIME_CONFIG__;
});

describe('setSentryUser', () => {
  it('sends only the opaque user id, never the display name', async () => {
    const { setSentryUser } = await loadMonitoringWithDsn('https://example.ingest.sentry.io/1');
    setSentryUser({ id: 'user-123', name: 'Jane Doe' });
    expect(setUserMock).toHaveBeenCalledWith({ id: 'user-123' });
    const sentUser = setUserMock.mock.calls[0]?.[0];
    expect(sentUser).not.toHaveProperty('username');
    expect(JSON.stringify(sentUser)).not.toContain('Jane Doe');
  });

  it('clears the user when passed null', async () => {
    const { setSentryUser } = await loadMonitoringWithDsn('https://example.ingest.sentry.io/1');
    setSentryUser(null);
    expect(setUserMock).toHaveBeenCalledWith(null);
  });

  it('is a no-op when no Sentry DSN is configured', async () => {
    const { setSentryUser } = await loadMonitoringWithDsn('');
    setSentryUser({ id: 'user-123', name: 'Jane Doe' });
    expect(setUserMock).not.toHaveBeenCalled();
  });
});

describe('initMonitoring beforeSend', () => {
  it('strips email, IP address, and username from outgoing events', async () => {
    const { initMonitoring } = await loadMonitoringWithDsn('https://example.ingest.sentry.io/1');
    initMonitoring();
    expect(initMock).toHaveBeenCalledTimes(1);
    const { beforeSend } = initMock.mock.calls[0]?.[0] as {
      beforeSend: (event: Record<string, unknown>) => Record<string, unknown>;
    };
    const event = {
      user: { id: 'user-123', email: 'jane@example.com', ip_address: '1.2.3.4', username: 'Jane Doe' },
    };
    const result = beforeSend(event);
    expect(result.user).toEqual({ id: 'user-123' });
  });
});
