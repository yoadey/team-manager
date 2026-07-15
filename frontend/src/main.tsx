import React from 'react';
import ReactDOM from 'react-dom/client';
import '@fontsource/roboto/300.css';
import '@fontsource/roboto/400.css';
import '@fontsource/roboto/500.css';
import '@fontsource/roboto/700.css';
import 'material-symbols/outlined.css';
import './index.css';
import { App } from './App';
import { ErrorBoundary, AppErrorFallback } from './components/ErrorBoundary';
import { LocaleProvider } from './i18n/LocaleProvider';
import { initMonitoring, captureError, installGlobalErrorHandlers } from './monitoring';
import { config } from './config';

// Apply the configurable app name (VITE_APP_NAME) to the browser tab so the
// static fallback in index.html can be overridden per deployment.
document.title = config.appName;

initMonitoring();
installGlobalErrorHandlers();

// Boots the MSW demo backend when no real API is configured, then renders.
//
// The `import('./mocks/browser')` call is guarded directly by
// `import.meta.env.DEV`/`import.meta.env.VITE_ALLOW_MOCK` (build-time
// constants Vite statically replaces), not by `config.allowMock` (a runtime
// value computed from them) — that distinction matters: Rollup can only
// dead-code-eliminate the dynamic import (and every mock/seed chunk it pulls
// in) from a genuine production build when the guarding condition is
// something it can *prove* false at build time, which a runtime value never
// is. A plain `npm run build` (DEV=false, VITE_ALLOW_MOCK unset) therefore
// ships with zero trace of `mocks/browser.ts`, its handlers, or the `msw`
// package in any emitted chunk — see openspec/changes/replace-mock-with-msw/
// specs/demo-mode/spec.md's "Demo artifacts excluded from production
// bundle" requirement. `VITE_ALLOW_MOCK=true` opts a *non-dev* build back in
// (e.g. the Playwright E2E job building against no backend).
async function bootstrapAndRender() {
  if (!config.apiBaseUrl) {
    // Production fail-safe: a production build with no configured backend
    // and no explicit opt-in must refuse to silently boot the demo backend
    // (which would otherwise serve seed data through a demo-only login).
    if (import.meta.env.PROD && !import.meta.env.VITE_ALLOW_MOCK) {
      throw new Error(
        'Teamverwaltung is misconfigured: no API_BASE_URL is set and VITE_ALLOW_MOCK is not enabled. ' +
          'Refusing to boot the demo backend in a production build.',
      );
    }
    if (import.meta.env.DEV || import.meta.env.VITE_ALLOW_MOCK) {
      const { worker } = await import('./mocks/browser');
      await worker.start({ onUnhandledRequest: 'bypass' });
    }
  }

  ReactDOM.createRoot(document.getElementById('root')!).render(
    <React.StrictMode>
      <ErrorBoundary onError={captureError} renderFallback={(error) => <AppErrorFallback error={error} />}>
        <LocaleProvider>
          <App />
        </LocaleProvider>
      </ErrorBoundary>
    </React.StrictMode>,
  );
}

bootstrapAndRender().catch((error) => {
  captureError(error instanceof Error ? error : new Error(String(error)), { componentStack: '' });
  document.body.innerHTML =
    '<pre style="padding:2rem;font:14px monospace;color:#b00020;white-space:pre-wrap">' +
    String(error instanceof Error ? error.message : error) +
    '</pre>';
});

// Register the offline shell service worker in production only (avoids dev HMR
// interference). Failures are non-fatal — the app works without it.
if (import.meta.env.PROD && 'serviceWorker' in navigator) {
  window.addEventListener('load', () => {
    navigator.serviceWorker.register('/sw.js').catch(() => {
      /* offline support is best-effort */
    });
  });
}
