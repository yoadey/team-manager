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
      ],
      // Minimum coverage floors. Keep these at or above measured baselines so
      // regressions fail CI; raise them as the test suite grows toward the
      // enterprise target of 80 % statements, 70 % branches, 75 % functions.
      thresholds: { statements: 62, branches: 45, functions: 58, lines: 62 },
    },
  },
});
