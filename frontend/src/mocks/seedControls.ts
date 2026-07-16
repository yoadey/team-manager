// Resets the in-memory demo DB (src/mocks/db.ts) back to a fresh seed.
// Used by src/test/setup.ts between tests and, in principle, by a demo-mode
// "reset demo data" action — though in the browser a page reload already
// re-executes this module from scratch, so AppContext's resetDemo() relies
// on that instead of calling this directly (see src/services/index.ts).
export { resetDb } from './db';
