// Centralised, validated runtime configuration. All variables are optional —
// the app boots with sensible defaults — but values are coerced and sanity
// checked here so a malformed `.env` fails loudly/predictably instead of
// producing `NaN` delays or silently-empty settings deep in the app.

/** Operator identity/contact + processor-disclosure fields for the legal-notice
 * and privacy-policy pages (see features/legal/content.ts). Runtime-only —
 * these have no VITE_* build-time equivalent, so an unset field is simply
 * absent rather than falling back to a build-time env var. */
export type OperatorConfigField =
  | 'name'
  | 'legalForm'
  | 'street'
  | 'postalCode'
  | 'city'
  | 'representedBy'
  | 'phone'
  | 'email'
  | 'registerCourt'
  | 'registerNumber'
  | 'vatId'
  | 'dataProtectionEmail'
  | 's3Provider'
  | 'smtpProvider'
  | 'sentryProvider'
  | 'otelProvider';

const OPERATOR_RUNTIME_KEYS: Record<OperatorConfigField, string> = {
  name: 'OPERATOR_NAME',
  legalForm: 'OPERATOR_LEGAL_FORM',
  street: 'OPERATOR_STREET',
  postalCode: 'OPERATOR_POSTAL_CODE',
  city: 'OPERATOR_CITY',
  representedBy: 'OPERATOR_REPRESENTED_BY',
  phone: 'OPERATOR_PHONE',
  email: 'OPERATOR_EMAIL',
  registerCourt: 'OPERATOR_REGISTER_COURT',
  registerNumber: 'OPERATOR_REGISTER_NUMBER',
  vatId: 'OPERATOR_VAT_ID',
  dataProtectionEmail: 'OPERATOR_DATA_PROTECTION_EMAIL',
  s3Provider: 'OPERATOR_S3_PROVIDER',
  smtpProvider: 'OPERATOR_SMTP_PROVIDER',
  sentryProvider: 'OPERATOR_SENTRY_PROVIDER',
  otelProvider: 'OPERATOR_OTEL_PROVIDER',
};

type RuntimeConfigKey = 'API_BASE_URL' | 'SENTRY_DSN' | 'VAPID_PUBLIC_KEY' | (typeof OPERATOR_RUNTIME_KEYS)[OperatorConfigField];

declare global {
  interface Window {
    // Populated by /config.js, loaded before this module runs (see
    // index.html). frontend/public/config.js checks in defaults for local
    // dev/tests/preview; the production Docker image regenerates it from the
    // container's API_BASE_URL/SENTRY_DSN/OPERATOR_* env vars at startup (see
    // frontend/docker/) so one built image can point at any backend/Sentry
    // project, and carry any operator's own legal-notice/privacy-policy
    // identity data, without rebuilding.
    __RUNTIME_CONFIG__?: Partial<Record<RuntimeConfigKey, string>>;
  }
}

/** Reads a runtime-injected __RUNTIME_CONFIG__ value, treating blank as unset. */
function runtimeConfig(key: RuntimeConfigKey): string | undefined {
  const v = typeof window !== 'undefined' ? window.__RUNTIME_CONFIG__?.[key] : undefined;
  return v && v.trim() !== '' ? v.trim() : undefined;
}

/**
 * Operator identity/contact/processor-disclosure fields, read straight from
 * __RUNTIME_CONFIG__ (no build-time VITE_* fallback — see
 * features/legal/content.ts for how an unset field renders: a placeholder
 * marker for the always-required fields, an omitted section for the
 * optional ones).
 */
function resolveOperatorConfig(): Record<OperatorConfigField, string | undefined> {
  const entries = Object.entries(OPERATOR_RUNTIME_KEYS) as [OperatorConfigField, string][];
  return Object.fromEntries(entries.map(([field, runtimeKey]) => [field, runtimeConfig(runtimeKey)])) as Record<
    OperatorConfigField,
    string | undefined
  >;
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

/**
 * The VAPID public key passed to PushManager.subscribe() as
 * applicationServerKey, preferring the runtime-injected value over the
 * build-time VITE_VAPID_PUBLIC_KEY Vite env var -- same rationale as
 * resolveSentryDsn: a released image must be able to pick up a rotated
 * backend VAPID keypair without a rebuild, and the release pipeline never
 * passes this as a build arg.
 */
function resolveVapidPublicKey(): string {
  return runtimeConfig('VAPID_PUBLIC_KEY') ?? stringEnv(import.meta.env.VITE_VAPID_PUBLIC_KEY, '');
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
  vapidPublicKey: resolveVapidPublicKey(),
  operator: resolveOperatorConfig(),
} as const;

// NOTE: `VITE_ALLOW_MOCK` (production fail-safe opt-in for the MSW demo
// backend) is deliberately NOT surfaced here as a `config.*` value. It must
// be read as the literal `import.meta.env.VITE_ALLOW_MOCK` expression at
// its call site (see main.tsx) so Vite/Rollup can statically prove it false
// and dead-code-eliminate the demo backend's dynamic import — and every
// mock/seed module it pulls in — out of a genuine production build. Reading
// it through this module first would erase that static-analysis guarantee.
