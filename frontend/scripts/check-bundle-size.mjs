/**
 * Bundle size budget check. Run after `npm run build`.
 *
 * Budgets (gzipped):
 *   - Any single JS chunk : 250 KB
 *   - Total JS             : 600 KB
 */

import { readdirSync, readFileSync } from 'fs';
import { join } from 'path';
import { gzipSync } from 'zlib';

const DIST_ASSETS = join(process.cwd(), 'dist', 'assets');

const CHUNK_LIMIT_KB = 250;
const TOTAL_LIMIT_KB = 600;

function gzippedKB(filePath) {
  const content = readFileSync(filePath);
  return gzipSync(content, { level: 9 }).length / 1024;
}

let files;
try {
  files = readdirSync(DIST_ASSETS).filter((f) => f.endsWith('.js'));
} catch {
  console.error('No dist/assets directory found. Run `npm run build` first.');
  process.exit(1);
}

let totalKB = 0;
let failed = false;
const rows = [];

for (const file of files.sort()) {
  const kb = gzippedKB(join(DIST_ASSETS, file));
  totalKB += kb;
  const over = kb > CHUNK_LIMIT_KB;
  if (over) failed = true;
  rows.push({ file, kb: kb.toFixed(1), over });
}

console.log('\nBundle size report (gzipped):\n');
for (const { file, kb, over } of rows) {
  const flag = over ? '  ❌ OVER BUDGET' : '';
  console.log(`  ${kb.padStart(7)} KB  ${file}${flag}`);
}
console.log(`${'─'.repeat(50)}`);
const totalOver = totalKB > TOTAL_LIMIT_KB;
if (totalOver) failed = true;
console.log(`  ${totalKB.toFixed(1).padStart(7)} KB  TOTAL${totalOver ? '  ❌ OVER BUDGET' : ''}`);

console.log(`\nBudgets: chunk ≤ ${CHUNK_LIMIT_KB} KB · total ≤ ${TOTAL_LIMIT_KB} KB (gzipped)\n`);

if (failed) {
  console.error('Bundle size budget exceeded. Reduce chunk sizes before merging.\n');
  process.exit(1);
} else {
  console.log('All chunks within budget.\n');
}
