// Centralised, validated runtime configuration. All variables are optional —
// the app boots with sensible defaults — but values are coerced and sanity
// checked here so a malformed `.env` fails loudly/predictably instead of
// producing `NaN` delays or silently-empty settings deep in the app.

/** Parse a non-negative integer env var, falling back when missing/invalid. */
function numberEnv(raw: string | undefined, fallback: number): number {
  if (raw == null || raw === '') return fallback;
  const n = Number(raw);
  if (!Number.isFinite(n) || n < 0) {
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
  apiBaseUrl: stringEnv(import.meta.env.VITE_API_BASE_URL, ''),
  storageKeyPrefix: stringEnv(import.meta.env.VITE_STORAGE_KEY_PREFIX, 'tv_db_'),
  mockDelayMin,
  mockDelayMax,
  sentryDsn: stringEnv(import.meta.env.VITE_SENTRY_DSN, ''),
} as const;
