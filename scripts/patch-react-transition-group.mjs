/**
 * react-transition-group v4 ships no "exports" field in package.json, so
 * Node.js ESM cannot resolve subpath directory imports like
 * "react-transition-group/TransitionGroupContext". MUI v9 relies on those
 * imports in its internal/Transition.mjs, which breaks the Vitest/jsdom
 * environment. This script injects the missing "exports" map into the
 * installed package so both the native ESM loader and Vite's resolver work.
 *
 * Run automatically via the "postinstall" npm hook.
 */

import { readFileSync, writeFileSync } from 'fs';
import { createRequire } from 'module';

const require = createRequire(import.meta.url);
const pkgPath = require.resolve('react-transition-group/package.json');
const pkg = JSON.parse(readFileSync(pkgPath, 'utf8'));

if (pkg.exports) {
  console.log('react-transition-group already has exports field, skipping patch.');
  process.exit(0);
}

pkg.exports = {
  '.': { import: './esm/index.js', require: './cjs/index.js', default: './cjs/index.js' },
  './TransitionGroupContext': {
    import: './esm/TransitionGroupContext.js',
    require: './cjs/TransitionGroupContext.js',
    default: './cjs/TransitionGroupContext.js',
  },
  './Transition': {
    import: './esm/Transition.js',
    require: './cjs/Transition.js',
    default: './cjs/Transition.js',
  },
  './CSSTransition': {
    import: './esm/CSSTransition.js',
    require: './cjs/CSSTransition.js',
    default: './cjs/CSSTransition.js',
  },
  './TransitionGroup': {
    import: './esm/TransitionGroup.js',
    require: './cjs/TransitionGroup.js',
    default: './cjs/TransitionGroup.js',
  },
  './SwitchTransition': {
    import: './esm/SwitchTransition.js',
    require: './cjs/SwitchTransition.js',
    default: './cjs/SwitchTransition.js',
  },
  './config': {
    import: './esm/config.js',
    require: './cjs/config.js',
    default: './cjs/config.js',
  },
  './*': './*',
};

writeFileSync(pkgPath, JSON.stringify(pkg, null, 2));
console.log('Patched react-transition-group exports map.');
