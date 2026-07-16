// Centralised, validated runtime configuration. All variables are optional —
// the app boots with sensible defaults — but values are coerced and sanity
// checked here so a malformed `.env` fails loudly/predictably instead of
// producing `NaN` delays or silently-empty settings deep in the app.

declare global {
  interface Window {
    // Populated by /config.js, loaded before this module runs (see
    // index.html). frontend/public/config.js checks in defaults for local
    // dev/tests/preview; the production Docker image regenerates it from the
    // container's API_BASE_URL/SENTRY_DSN env vars at startup (see
    // frontend/docker/) so one built image can point at any backend/Sentry
    // project without rebuilding.
    __RUNTIME_CONFIG__?: { API_BASE_URL?: string; SENTRY_DSN?: string };
  }
}

/** Reads a runtime-injected __RUNTIME_CONFIG__ value, treating blank as unset. */
function runtimeConfig(key: 'API_BASE_URL' | 'SENTRY_DSN'): string | undefined {
  const v = typeof window !== 'undefined' ? window.__RUNTIME_CONFIG__?.[key] : undefined;
  return v && v.trim() !== '' ? v.trim() : undefined;
}

/**
 * The API base URL, preferring the runtime-injected value over the
 * build-time VITE_API_BASE_URL Vite env var (used when no config.js is
 * loaded, e.g. Vitest's jsdom environment).
 */
function resolveApiBaseUrl(): string {
  return runtimeConfig('API_BASE_URL') ?? stringEnv(import.meta.env.VITE_API_BASE_URL, '');
}

/**
 * The Sentry DSN, preferring the runtime-injected value over the build-time
 * VITE_SENTRY_DSN Vite env var. The runtime path is the only way to enable
 * Sentry in a released Docker image at all — the Dockerfile/release.yml
 * build pipeline never passes VITE_SENTRY_DSN as a build arg, so without
 * this it would be permanently baked in as empty regardless of environment.
 */
function resolveSentryDsn(): string {
  return runtimeConfig('SENTRY_DSN') ?? stringEnv(import.meta.env.VITE_SENTRY_DSN, '');
}

/** Parse a non-negative integer env var, falling back when missing/invalid. */
function numberEnv(raw: string | undefined, fallback: number): number {
  if (raw == null || raw === '') return fallback;
  const n = Number(raw);
  if (!Number.isFinite(n) || n < 0) {
    // eslint-disable-next-line no-console
    if (import.meta.env.DEV) console.warn(`[config] invalid numeric env value "${raw}", using fallback ${fallback}`);
    return fallback;
  }
  return n;
}

function stringEnv(raw: string | undefined, fallback: string): string {
  const v = (raw ?? '').trim();
  return v || fallback;
}

const mockDelayMin = numberEnv(import.meta.env.VITE_MOCK_DELAY_MIN, 120);
const mockDelayMaxRaw = numberEnv(import.meta.env.VITE_MOCK_DELAY_MAX, 320);
// Guarantee min <= max so the mock delay range is always valid.
const mockDelayMax = Math.max(mockDelayMin, mockDelayMaxRaw);

export const config = {
  appName: stringEnv(import.meta.env.VITE_APP_NAME, 'Teamverwaltung'),
  apiBaseUrl: resolveApiBaseUrl(),
  storageKeyPrefix: stringEnv(import.meta.env.VITE_STORAGE_KEY_PREFIX, 'tv_db_'),
  mockDelayMin,
  mockDelayMax,
  sentryDsn: resolveSentryDsn(),
} as const;

// NOTE: `VITE_ALLOW_MOCK` (production fail-safe opt-in for the MSW demo
// backend) is deliberately NOT surfaced here as a `config.*` value. It must
// be read as the literal `import.meta.env.VITE_ALLOW_MOCK` expression at
// its call site (see main.tsx) so Vite/Rollup can statically prove it false
// and dead-code-eliminate the demo backend's dynamic import — and every
// mock/seed module it pulls in — out of a genuine production build. Reading
// it through this module first would erase that static-analysis guarantee.
