import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';
import { visualizer } from 'rollup-plugin-visualizer';
import { fileURLToPath, URL } from 'url';

// https://vite.dev/config/
export default defineConfig({
  plugins: [react(), visualizer({ filename: 'dist/stats.html', gzipSize: true, brotliSize: true })],
  resolve: {
    alias: { '@': fileURLToPath(new URL('./src', import.meta.url)) },
  },
  server: {
    port: 5173,
    host: true,
    headers: {
      'Content-Security-Policy': "frame-ancestors 'none'",
      'X-Frame-Options': 'DENY',
    },
  },
  build: {
    // Emit source maps so Sentry can symbolicate production stack traces.
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
