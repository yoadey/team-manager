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
