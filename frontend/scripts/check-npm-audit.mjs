#!/usr/bin/env node
// npm has no native per-advisory ignore mechanism (unlike Trivy's
// `trivyignores` file, see .github/trivyignore-*.txt), so this reimplements
// that same pattern for `npm audit`: a plain text allowlist of GHSA IDs with
// a written justification, consumed here instead of a CLI flag.
//
// Unlike the previous `npm audit --omit=dev --audit-level=high`, this
// audits ALL dependencies including devDependencies -- `npm ci` installs and
// runs devDependency install/build scripts in nearly every CI job, so a
// HIGH/CRITICAL CVE there is exactly as reachable at CI-runner-privilege
// as one in a production dependency, and skipping devDependencies here
// meant it was never caught by any gate, PR-time or scheduled.
//
// Any HIGH or CRITICAL advisory not present in the allowlist fails the
// build. Allowlisted advisories still print a warning summary so they stay
// visible, not silently swallowed.
import { execFileSync } from 'node:child_process';
import { readFileSync } from 'node:fs';
import { fileURLToPath } from 'node:url';
import path from 'node:path';

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const allowlistPath = path.join(__dirname, '..', 'npm-audit-allowlist.txt');
const FAIL_SEVERITIES = new Set(['high', 'critical']);

function loadAllowlist(filePath) {
  const ids = new Set();
  for (const line of readFileSync(filePath, 'utf8').split('\n')) {
    const trimmed = line.trim();
    if (!trimmed || trimmed.startsWith('#')) continue;
    ids.add(trimmed);
  }
  return ids;
}

function ghsaIdFromUrl(url) {
  const match = /advisories\/(GHSA-[\w-]+)/.exec(url ?? '');
  return match ? match[1] : null;
}

function runAudit() {
  try {
    return execFileSync('npm', ['audit', '--json'], { encoding: 'utf8', maxBuffer: 32 * 1024 * 1024 });
  } catch (err) {
    // npm audit exits non-zero when it finds anything at all; the JSON
    // report is still on stdout in that case.
    if (err.stdout) return err.stdout;
    throw err;
  }
}

const allowlist = loadAllowlist(allowlistPath);
const report = JSON.parse(runAudit());

// `npm audit --json` reports a tool-level failure (corrupted lockfile,
// registry outage, throttling, etc.) as `{"error": {...}}` with no
// `vulnerabilities` key at all -- a shape that must never be treated the
// same as a genuinely clean `{"vulnerabilities": {}, ...}` report. Failing
// loudly here (rather than defaulting to `{}` and reporting "no advisories
// found") matters most for the unattended weekly run in
// scheduled-security-scan.yml, where nobody is watching for a "too quiet"
// green build.
if (report.error || !('vulnerabilities' in report)) {
  console.error('npm audit itself failed to produce a vulnerability report:');
  console.error(JSON.stringify(report.error ?? report, null, 2));
  process.exit(1);
}

const vulnerabilities = report.vulnerabilities;

const failing = [];
const allowed = [];

for (const [pkg, vuln] of Object.entries(vulnerabilities)) {
  for (const via of vuln.via ?? []) {
    if (typeof via === 'string') continue; // a bare dependency name, not an advisory
    if (!FAIL_SEVERITIES.has(via.severity)) continue;
    const id = ghsaIdFromUrl(via.url);
    const entry = { pkg, id: id ?? '(unknown)', title: via.title, severity: via.severity, url: via.url };
    if (id && allowlist.has(id)) {
      allowed.push(entry);
    } else {
      failing.push(entry);
    }
  }
}

if (allowed.length > 0) {
  console.log('Allowlisted advisories (see npm-audit-allowlist.txt for justification):');
  for (const a of allowed) console.log(`  - ${a.pkg}: ${a.id} (${a.severity}) ${a.title}`);
}

if (failing.length > 0) {
  console.error('\nFound HIGH/CRITICAL npm advisories not covered by npm-audit-allowlist.txt:');
  for (const f of failing) console.error(`  - ${f.pkg}: ${f.id} (${f.severity}) ${f.title} — ${f.url}`);
  console.error(
    '\nFix the dependency, or add the GHSA ID to npm-audit-allowlist.txt with a written justification if it is genuinely unreachable.',
  );
  process.exit(1);
}

console.log('\nNo un-allowlisted HIGH/CRITICAL npm advisories found (including devDependencies).');
