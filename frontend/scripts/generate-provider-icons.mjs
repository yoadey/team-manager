// Generates the static login-provider icons under public/provider-icons/.
//
// Why generate rather than hand-write: the backend advertises a provider's
// icon as a URL (OIDC_PROVIDER_ICON -> Provider.icon), and a URL needs a real
// file. @mui/icons-material already ships the marks we want, but as React
// components -- unusable as an <img src>. So we lift the path data straight
// out of those modules and emit plain SVG files.
//
// The fill is baked in rather than left as currentColor: an <img> has no
// inheritable colour, so currentColor would render as black regardless of the
// button it sits on. #1e293b matches the provider button's own `fg`.
//
// Output is committed, so a build never depends on this script. Re-run with:
//   npm run icons:providers
import { mkdirSync, readFileSync, writeFileSync } from 'fs';
import { fileURLToPath } from 'url';
import { dirname, join } from 'path';

const ROOT = join(dirname(fileURLToPath(import.meta.url)), '..');
const OUT_DIR = join(ROOT, 'public', 'provider-icons');
const ICON_PKG = join(ROOT, 'node_modules', '@mui', 'icons-material');

const FILL = '#1e293b';

// Source MUI icon -> emitted file name. `openid` is the neutral fallback for a
// provider with no recognisable brand (a company's own Keycloak, say).
const ICONS = {
  google: 'Google',
  apple: 'Apple',
  microsoft: 'Microsoft',
  github: 'GitHub',
  facebook: 'Facebook',
  openid: 'LockOutlined',
};

/** Extracts every `d="..."` path from a @mui/icons-material CommonJS module. */
function pathsFor(iconName) {
  const source = readFileSync(join(ICON_PKG, `${iconName}.js`), 'utf8');
  const paths = [...source.matchAll(/\bd:\s*"([^"]+)"/g)].map((m) => m[1]);
  if (paths.length === 0) {
    throw new Error(
      `No path data found in @mui/icons-material/${iconName}.js — the package layout may have changed.`,
    );
  }
  return paths;
}

function svgFor(iconName) {
  const body = pathsFor(iconName)
    .map((d) => `<path d="${d}"/>`)
    .join('');
  // role/aria-hidden are deliberately omitted: the file is referenced via
  // <img alt="">, which already makes it decorative to assistive tech.
  return (
    `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" ` +
    `width="24" height="24" fill="${FILL}">${body}</svg>\n`
  );
}

mkdirSync(OUT_DIR, { recursive: true });

for (const [fileName, iconName] of Object.entries(ICONS)) {
  const out = join(OUT_DIR, `${fileName}.svg`);
  writeFileSync(out, svgFor(iconName));
  console.log(`wrote public/provider-icons/${fileName}.svg (from ${iconName})`);
}
