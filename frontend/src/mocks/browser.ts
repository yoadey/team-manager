// Browser MSW worker for dev-demo mode. Only imported dynamically from
// main.tsx (never a static import) so Rollup drops the mock/seed bundle from
// production builds entirely — see src/main.tsx and openspec/changes/
// replace-mock-with-msw/specs/demo-mode/spec.md's "Demo artifacts excluded
// from production bundle" requirement.
import { setupWorker } from 'msw/browser';
import { handlers } from './handlers';

export const worker = setupWorker(...handlers);
