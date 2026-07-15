import { defineConfig } from 'vitest/config';
import react from '@vitejs/plugin-react';
import { fileURLToPath, URL } from 'url';

// Vitest configuration kept separate from vite.config.ts so the production
// build pipeline stays free of test-only settings. The jsdom environment is
// required because the service layer persists to localStorage and the project
// targets React Testing Library for component/hook tests.
export default defineConfig({
  plugins: [react()],
  resolve: {
    alias: [{ find: '@', replacement: fileURLToPath(new URL('./src', import.meta.url)) }],
  },
  test: {
    environment: 'jsdom',
    globals: true,
    setupFiles: ['./src/test/setup.ts'],
    css: false,
    include: ['src/**/*.{test,spec}.{ts,tsx}'],
    coverage: {
      provider: 'v8',
      reporter: ['text', 'html'],
      // Measure the whole app (previously only services + utils were counted,
      // which made the thresholds misleading). Presentational-only files and
      // non-logic entry points are excluded; they are covered by component
      // tests, which are tracked as a separate, growing effort.
      include: ['src/**/*.{ts,tsx}'],
      exclude: [
        'src/**/*.{test,spec}.{ts,tsx}',
        'src/test/**',
        'src/types/**',
        'src/**/index.ts',
        'src/main.tsx',
        'src/monitoring.ts',
        'src/i18n/de.ts',
        'src/i18n/en.ts',
        // Thin openapi-fetch client factory — config only, no logic to unit-test.
        // (serviceLayerReal.ts is now unit-tested in serviceLayerReal.test.ts and
        // intentionally counted toward the coverage floors.)
        'src/api/client.ts',
        // Top-level app shell — layout-only, requires E2E tests for meaningful coverage
        'src/layouts/AppShell.tsx',
        // Demo/test backend infrastructure (MSW handlers + in-memory seed DB),
        // not application logic — exercised indirectly by every test that
        // calls `api`, but excluded from the floors so a large, straight-line
        // fixture file can't inflate the overall percentage and mask thin
        // coverage elsewhere. mocks/handlers.test.ts still exercises it directly.
        'src/mocks/**',
      ],
      // Enterprise-ready coverage floors.
      thresholds: { statements: 80, branches: 65, functions: 75, lines: 80 },
    },
  },
});
