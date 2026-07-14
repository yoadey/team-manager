import js from '@eslint/js';
import tsEslint from 'typescript-eslint';
import reactHooks from 'eslint-plugin-react-hooks';
import jsxA11y from 'eslint-plugin-jsx-a11y';

export default tsEslint.config(
  js.configs.recommended,
  ...tsEslint.configs.recommended,
  jsxA11y.flatConfigs.recommended,
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
      // Prevent dangerouslySetInnerHTML — all user content must be sanitised via DOMPurify first
      'no-restricted-syntax': [
        'error',
        {
          selector: 'JSXAttribute[name.name="dangerouslySetInnerHTML"]',
          message: 'Use DOMPurify.sanitize() before setting HTML via dangerouslySetInnerHTML.',
        },
      ],
      // Complexity guards
      complexity: ['warn', 20],
      'max-params': ['warn', 5],
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
    // Loaded via a plain <script> tag in index.html — runs in a browser
    // window scope, not a module.
    files: ['public/config.js'],
    languageOptions: {
      globals: {
        window: 'readonly',
      },
    },
  },
  {
    // Node.js scripts (bundle checker, etc.) use Node globals
    files: ['scripts/**/*.mjs', 'scripts/**/*.js'],
    languageOptions: {
      globals: {
        process: 'readonly',
        console: 'readonly',
      },
    },
  },
  {
    ignores: ['dist/', 'node_modules/', '*.config.*', 'eslint.config.js'],
  },
);
