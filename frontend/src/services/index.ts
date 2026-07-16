import type { ModuleKey } from '@/types';
import type { ApiContract } from './apiContract';
import { realApi } from './serviceLayerReal';

// `realApi` (the generated openapi-fetch client) is the only implementation,
// for production, dev-demo and tests alike. A backend-less demo is provided
// by intercepting its HTTP requests with MSW (see src/mocks/), not by a
// second in-code business-logic implementation — see openspec/changes/
// replace-mock-with-msw.
export const api: ApiContract = realApi;
export type Api = ApiContract;

export const MODULE_LABELS: Record<ModuleKey, string> = {
  events: 'Termine',
  members: 'Mitglieder',
  finances: 'Finanzen',
  news: 'Neuigkeiten',
  polls: 'Umfragen',
  settings: 'Einstellungen',
};

// AppContext's "reset demo data" action calls this then reloads the page.
// Against a real backend there is nothing to reset (no-op). In dev-demo mode
// there is also nothing to do here: src/mocks/db.ts's in-memory DB is a
// module-level singleton that is re-seeded from scratch the moment the page
// reload re-executes it, so the reload alone accomplishes the reset.
export function resetDemoData(): void {
  /* no-op — see comment above */
}
