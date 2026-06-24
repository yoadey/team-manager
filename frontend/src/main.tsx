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

ReactDOM.createRoot(document.getElementById('root')!).render(
  <React.StrictMode>
    <ErrorBoundary onError={captureError} renderFallback={(error) => <AppErrorFallback error={error} />}>
      <LocaleProvider>
        <App />
      </LocaleProvider>
    </ErrorBoundary>
  </React.StrictMode>,
);

// Register the offline shell service worker in production only (avoids dev HMR
// interference). Failures are non-fatal — the app works without it.
if (import.meta.env.PROD && 'serviceWorker' in navigator) {
  window.addEventListener('load', () => {
    navigator.serviceWorker.register('/sw.js').catch(() => {
      /* offline support is best-effort */
    });
  });
}
