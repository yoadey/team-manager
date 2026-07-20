import { describe, it, expect, afterEach } from 'vitest';
import { execFileSync } from 'node:child_process';
import { mkdtempSync, writeFileSync, rmSync } from 'node:fs';
import { tmpdir } from 'node:os';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

// scripts/check-npm-audit.mjs resolves its allowlist path relative to its
// own location (__dirname), not the process cwd, so it's safe to run it
// with cwd pointed at a throwaway fixture directory to control what `npm
// audit` itself sees, while still exercising the real allowlist file.
const scriptPath = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '../../scripts/check-npm-audit.mjs');

let fixtureDir: string | undefined;

afterEach(() => {
  if (fixtureDir) rmSync(fixtureDir, { recursive: true, force: true });
  fixtureDir = undefined;
});

describe('scripts/check-npm-audit.mjs', () => {
  // Regression test: `npm audit --json` reports a tool-level failure (e.g.
  // ENOLOCK for a missing lockfile) as `{"error": {...}}` with no
  // `vulnerabilities` key -- the script used to default that to `{}` via
  // `report.vulnerabilities ?? {}`, so a broken audit run printed "No
  // un-allowlisted HIGH/CRITICAL npm advisories found" and exited 0,
  // indistinguishable from a genuinely clean run.
  it('fails loudly instead of silently reporting clean when npm audit itself errors', () => {
    fixtureDir = mkdtempSync(path.join(tmpdir(), 'check-npm-audit-'));
    // No package-lock.json in this fixture -> `npm audit` fails with ENOLOCK.
    writeFileSync(path.join(fixtureDir, 'package.json'), JSON.stringify({ name: 'fixture', version: '1.0.0' }));

    expect(() => execFileSync('node', [scriptPath], { cwd: fixtureDir, encoding: 'utf8' })).toThrow();
  });
});
