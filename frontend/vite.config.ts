import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';
import { visualizer } from 'rollup-plugin-visualizer';
import { fileURLToPath, URL } from 'url';
import { execSync } from 'child_process';

function gitCommit(): string {
  try {
    return execSync('git rev-parse --short HEAD').toString().trim();
  } catch {
    return 'unknown';
  }
}

// https://vite.dev/config/
export default defineConfig({
  plugins: [react(), visualizer({ filename: 'dist/stats.html', gzipSize: true, brotliSize: true })],
  define: {
    // Injected at build time so Sentry can symbolicate stack traces to the
    // correct release and the app can surface the running version.
    'import.meta.env.VITE_BUILD_COMMIT': JSON.stringify(process.env.VITE_BUILD_COMMIT ?? gitCommit()),
    'import.meta.env.VITE_BUILD_VERSION': JSON.stringify(process.env.VITE_BUILD_VERSION ?? 'dev'),
  },
  resolve: {
    alias: { '@': fileURLToPath(new URL('./src', import.meta.url)) },
  },
  server: {
    port: 5173,
    host: true,
    // Dev-server security headers that mirror what the production web server
    // (nginx / Caddy / CDN) must also set.  The CSP frame-ancestors directive
    // must stay in an HTTP header — meta-tag CSP ignores it in all browsers.
    headers: {
      'Content-Security-Policy': "frame-ancestors 'none'",
      'X-Frame-Options': 'DENY',
      'X-Content-Type-Options': 'nosniff',
      'Referrer-Policy': 'strict-origin-when-cross-origin',
      'Permissions-Policy': 'camera=(), microphone=(), geolocation=()',
    },
    proxy: {
      '/api': {
        target: 'http://localhost:8080',
      },
    },
  },
  build: {
    // Emitted for local stack-trace symbolication and potential future Sentry
    // release uploads. The Dockerfile deletes *.map from dist/ before the
    // image is built, and nginx also denies *.map requests as a backstop —
    // never serve source maps from the public production site.
    sourcemap: true,
    rollupOptions: {
      output: {
        // Split large, rarely-changing vendor code into its own long-lived
        // chunks for better caching and a smaller main bundle.
        manualChunks: (id) => {
          if (id.includes('node_modules')) {
            if (id.includes('react') || id.includes('react-dom')) return 'react';
            if (id.includes('@mui') || id.includes('@emotion')) return 'mui';
            if (id.includes('@sentry')) return 'sentry';
          }
        },
      },
    },
  },
});
