export const config = {
  appName: import.meta.env.VITE_APP_NAME ?? 'Teamverwaltung',
  apiBaseUrl: import.meta.env.VITE_API_BASE_URL ?? '',
  storageKeyPrefix: import.meta.env.VITE_STORAGE_KEY_PREFIX ?? 'tv_db_',
  mockDelayMin: Number(import.meta.env.VITE_MOCK_DELAY_MIN ?? 120),
  mockDelayMax: Number(import.meta.env.VITE_MOCK_DELAY_MAX ?? 320),
  sentryDsn: import.meta.env.VITE_SENTRY_DSN ?? '',
} as const;
