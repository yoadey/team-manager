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

/** ErrorBoundary hook: receives a React error + component stack. */
export function captureError(error: Error, info: ErrorInfo): void {
  if (import.meta.env.DEV) {
    // eslint-disable-next-line no-console
    console.error('[monitoring]', error, info.componentStack);
    return;
  }
  Sentry.captureException(error, { contexts: { react: { componentStack: info.componentStack ?? '' } } });
}

/** Generic capture for caught async/runtime errors (action hooks, global handlers). */
export function captureException(error: unknown, context?: Record<string, unknown>): void {
  if (import.meta.env.DEV) {
    // eslint-disable-next-line no-console
    console.error('[monitoring]', error, context ?? '');
    return;
  }
  Sentry.captureException(error, context ? { extra: context } : undefined);
}

/** Sets or clears the authenticated user on Sentry scope for error attribution. */
export function setSentryUser(user: { id: string; name: string; email?: string } | null): void {
  if (!config.sentryDsn) return;
  Sentry.setUser(user ? { id: user.id, username: user.name, email: user.email } : null);
}

/** Registers global handlers for otherwise-unhandled promise rejections and errors. */
export function installGlobalErrorHandlers(): void {
  window.addEventListener('unhandledrejection', (event) => {
    captureException(event.reason, { kind: 'unhandledrejection' });
  });
  window.addEventListener('error', (event) => {
    captureException(event.error ?? event.message, { kind: 'window.error' });
  });
}
