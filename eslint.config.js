import js from '@eslint/js';
import tsEslint from 'typescript-eslint';
import reactHooks from 'eslint-plugin-react-hooks';

export default tsEslint.config(
  js.configs.recommended,
  ...tsEslint.configs.recommended,
  {
    plugins: { 'react-hooks': reactHooks },
    rules: {
      'react-hooks/rules-of-hooks': 'error',
      // Promoted to error: stale-closure bugs are a real correctness risk and
      // the codebase is currently clean, so this stays enforceable.
      'react-hooks/exhaustive-deps': 'error',
      '@typescript-eslint/no-explicit-any': 'warn',
      '@typescript-eslint/no-unused-vars': ['warn', { argsIgnorePattern: '^_', varsIgnorePattern: '^_' }],
      'no-console': 'warn',
    },
  },
  {
    // The service worker runs in a worker scope with its own globals.
    files: ['public/sw.js'],
    languageOptions: {
      globals: {
        self: 'readonly',
        caches: 'readonly',
        fetch: 'readonly',
        Response: 'readonly',
        Promise: 'readonly',
      },
    },
  },
  {
    ignores: ['dist/', 'node_modules/', '*.config.*', 'eslint.config.js'],
  },
);
