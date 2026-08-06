import type { ErrorInfo } from 'react';
import * as Sentry from '@sentry/react';
import { config } from './config';

// § 25 TDDDG (formerly TTDSG, the German ePrivacy transposition) requires
// consent before storing/accessing non-essential information on the end
// device. Determination (openspec/changes/webapp-legal-compliance task 3):
// with only browserTracingIntegration() enabled (no replayIntegration() /
// session-tracking, which is what Sentry's own docs tie sessionStorage use
// to), Sentry.init + captureException/setUser wrote no cookies and no
// localStorage/sessionStorage keys in a jsdom probe exercising this exact
// config. No consent gate is required -- disclosure in the privacy policy
// (see features/legal/content.ts's "Cookies und lokale Speicherung" section)
// is sufficient. Re-verify this if replayIntegration or any session-tracking
// integration is ever added here.
export function initMonitoring(): void {
  if (!config.sentryDsn) return;
  Sentry.init({
    dsn: config.sentryDsn,
    integrations: [Sentry.browserTracingIntegration()],
    tracesSampleRate: 0.1,
    environment: import.meta.env.MODE,
    release: (import.meta.env.VITE_BUILD_VERSION as string | undefined) || undefined,
    beforeSend(event) {
      // Strip PII from outgoing events — email, IP, and display name must never leave the browser.
      if (event.user) {
        delete event.user.email;
        delete event.user.ip_address;
        delete event.user.username;
      }
      return event;
    },
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

/**
 * Sets or clears the authenticated user on Sentry scope for error attribution.
 * Only the opaque id is sent — the display name is PII and must never leave the browser.
 */
export function setSentryUser(user: { id: string; name: string } | null): void {
  if (!config.sentryDsn) return;
  Sentry.setUser(user ? { id: user.id } : null);
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
