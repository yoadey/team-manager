import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';

// config.apiBaseUrl is computed once at module load time, so each scenario
// needs a fresh module instance (vi.resetModules + dynamic import) rather
// than mutating the already-evaluated export.
beforeEach(() => {
  vi.resetModules();
  delete (window as { __RUNTIME_CONFIG__?: unknown }).__RUNTIME_CONFIG__;
});

afterEach(() => {
  delete (window as { __RUNTIME_CONFIG__?: unknown }).__RUNTIME_CONFIG__;
});

describe('config.apiBaseUrl', () => {
  it('falls back to the mock backend (empty string) when no runtime config is injected', async () => {
    const { config } = await import('./config');
    expect(config.apiBaseUrl).toBe('');
  });

  it('prefers a non-empty window.__RUNTIME_CONFIG__.API_BASE_URL over the build-time env var', async () => {
    window.__RUNTIME_CONFIG__ = { API_BASE_URL: 'https://api.example.com' };
    const { config } = await import('./config');
    expect(config.apiBaseUrl).toBe('https://api.example.com');
  });

  it('trims whitespace from the runtime value', async () => {
    window.__RUNTIME_CONFIG__ = { API_BASE_URL: '  https://api.example.com  ' };
    const { config } = await import('./config');
    expect(config.apiBaseUrl).toBe('https://api.example.com');
  });

  it('treats an empty or whitespace-only runtime value as unset (falls back to mock)', async () => {
    window.__RUNTIME_CONFIG__ = { API_BASE_URL: '   ' };
    const { config } = await import('./config');
    expect(config.apiBaseUrl).toBe('');
  });
});

describe('config.sentryDsn', () => {
  it('falls back to empty (disabled) when no runtime config is injected', async () => {
    const { config } = await import('./config');
    expect(config.sentryDsn).toBe('');
  });

  it('prefers a non-empty window.__RUNTIME_CONFIG__.SENTRY_DSN over the build-time env var', async () => {
    window.__RUNTIME_CONFIG__ = { SENTRY_DSN: 'https://key@o0.ingest.sentry.io/1' };
    const { config } = await import('./config');
    expect(config.sentryDsn).toBe('https://key@o0.ingest.sentry.io/1');
  });

  it('treats an empty or whitespace-only runtime value as unset (falls back to build-time env)', async () => {
    window.__RUNTIME_CONFIG__ = { SENTRY_DSN: '   ' };
    const { config } = await import('./config');
    expect(config.sentryDsn).toBe('');
  });
});

describe('config.operator', () => {
  it('is all-undefined when no runtime config is injected', async () => {
    const { config } = await import('./config');
    expect(config.operator.name).toBeUndefined();
    expect(config.operator.email).toBeUndefined();
    expect(config.operator.s3Provider).toBeUndefined();
  });

  it('reads each OPERATOR_* runtime var into its corresponding field', async () => {
    window.__RUNTIME_CONFIG__ = {
      OPERATOR_NAME: 'Stefan May',
      OPERATOR_STREET: 'Robensstraße 56',
      OPERATOR_POSTAL_CODE: '52070',
      OPERATOR_CITY: 'Aachen',
      OPERATOR_EMAIL: 'info@yoadey.de',
      OPERATOR_S3_PROVIDER: 'self-hosted',
    };
    const { config } = await import('./config');
    expect(config.operator.name).toBe('Stefan May');
    expect(config.operator.street).toBe('Robensstraße 56');
    expect(config.operator.postalCode).toBe('52070');
    expect(config.operator.city).toBe('Aachen');
    expect(config.operator.email).toBe('info@yoadey.de');
    expect(config.operator.s3Provider).toBe('self-hosted');
    expect(config.operator.legalForm).toBeUndefined();
  });

  it('treats a blank OPERATOR_* value as unset', async () => {
    window.__RUNTIME_CONFIG__ = { OPERATOR_NAME: '   ' };
    const { config } = await import('./config');
    expect(config.operator.name).toBeUndefined();
  });
});
