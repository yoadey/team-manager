// Global test setup executed once per test file before the suite runs.
// Registers jest-dom matchers (toBeInTheDocument, etc.) for React Testing
// Library assertions, starts the MSW node server that intercepts every
// `realApi` (openapi-fetch) request, and resets all mutable test state
// between tests.
import '@testing-library/jest-dom/vitest';
import { afterAll, afterEach, beforeEach } from 'vitest';
import { cleanup } from '@testing-library/react';
import { server } from '@/mocks/server';
import { resetDb } from '@/mocks/seedControls';

// Started synchronously at setup-module-evaluation time (not inside
// `beforeAll`), because openapi-fetch's `createClient()` captures
// `globalThis.fetch` as a default parameter *once, at call time*, and
// `@/api/client.ts`'s `apiClient` singleton is created the moment any test
// file imports it — which happens while Vitest is still loading that file's
// own top-level imports, strictly before its (or this file's) `beforeAll`
// hooks run. Patching fetch here instead, during setupFiles evaluation
// (which Vitest guarantees completes before a test file's own module graph
// is evaluate), ensures `apiClient` always captures the MSW-patched fetch.
// `onUnhandledRequest: 'error'` fails a test loudly if it hits a route with
// no matching handler, instead of letting the request silently fall through
// to a real network call (or hang) — see openspec/changes/
// replace-mock-with-msw/tasks.md item 4.1.
server.listen({ onUnhandledRequest: 'error' });

beforeEach(() => {
  // Some UI state (e.g. color-scheme preference) still persists to
  // localStorage independent of the API layer; keep starting each test from
  // a clean slate.
  localStorage.clear();
});

afterEach(() => {
  cleanup();
  server.resetHandlers();
  resetDb();
});

afterAll(() => server.close());
