import type { ErrorInfo } from 'react';
import * as Sentry from '@sentry/react';
import { config } from './config';

export function initMonitoring(): void {
  if (!config.sentryDsn) return;
  Sentry.init({
    dsn: config.sentryDsn,
    integrations: [Sentry.browserTracingIntegration()],
    tracesSampleRate: 0.1,
    environment: import.meta.env.MODE,
  });
}

export function captureError(error: Error, info: ErrorInfo): void {
  if (import.meta.env.DEV) {
    console.error('[monitoring]', error, info.componentStack);
    return;
  }
  Sentry.captureException(error, { contexts: { react: { componentStack: info.componentStack ?? '' } } });
}
