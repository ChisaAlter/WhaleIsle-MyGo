// Seed the files-surface workspace fixture into <wsDir>: a small tree plus a
// git repo whose working tree carries one uncommitted change, identical on
// both sides because both Hosts point at the same directory. No network.
// Usage: node seed-files.mjs <wsDir>
import { execFileSync } from 'node:child_process';
import { mkdirSync, readdirSync, readFileSync, statSync, writeFileSync } from 'node:fs';
import { dirname, join, relative } from 'node:path';

const wsDir = process.argv[2];
if (!wsDir) throw new Error('usage: node seed-files.mjs <wsDir>');

const readme = `# Fixture Notes

First paragraph of the fixture README. It stays short so the markdown
preview's paragraph rhythm is measurable on both products.

Second paragraph, also plain prose, written to sit one block below the
first so the gap between blocks shows up in the comparison.

\`\`\`ts
export const fixture = 'code block'
\`\`\`
`;

// ~40 lines of TypeScript so the code preview's line gutter and line
// rhythm cover more than one screen row of content.
const appLines = [];
appLines.push(`import { padStart, trimEnd } from './util/strings'`);
appLines.push(``);
appLines.push(`export interface AppConfig {`);
appLines.push(`  readonly name: string`);
appLines.push(`  readonly retries: number`);
appLines.push(`  readonly verbose: boolean`);
appLines.push(`}`);
appLines.push(``);
appLines.push(`const defaults: AppConfig = {`);
appLines.push(`  name: 'fixture',`);
appLines.push(`  retries: 3,`);
appLines.push(`  verbose: false,`);
appLines.push(`}`);
appLines.push(``);
appLines.push(`export function renderBanner(config: AppConfig): string {`);
appLines.push(`  const title = padStart(config.name, 12, '*')`);
appLines.push(`  const suffix = config.verbose ? ' (verbose)' : ''`);
appLines.push(`  return trimEnd(title + suffix)`);
appLines.push(`}`);
appLines.push(``);
appLines.push(`export function shouldRetry(config: AppConfig, attempt: number): boolean {`);
appLines.push(`  if (attempt < 0) return false`);
appLines.push(`  return attempt < config.retries`);
appLines.push(`}`);
appLines.push(``);
appLines.push(`export async function main(): Promise<void> {`);
appLines.push(`  const banner = renderBanner(defaults)`);
appLines.push(`  for (let attempt = 0; shouldRetry(defaults, attempt); attempt++) {`);
appLines.push(`    console.log(banner, 'attempt', attempt)`);
appLines.push(`  }`);
appLines.push(`}`);
appLines.push(``);
appLines.push(`void main()`);

const strings = `export function padStart(value: string, width: number, fill: string): string {
  let out = value
  while (out.length < width) out = fill + out
  return out
}

export function trimEnd(value: string): string {
  return value.replace(/\\s+$/, '')
}
`;

const notes = `fixture notes
second line
third line
`;

mkdirSync(join(wsDir, 'src', 'util'), { recursive: true });
mkdirSync(join(wsDir, 'docs'), { recursive: true });
writeFileSync(join(wsDir, 'README.md'), readme);
writeFileSync(join(wsDir, 'src', 'app.ts'), appLines.join('\n') + '\n');
writeFileSync(join(wsDir, 'src', 'util', 'strings.ts'), strings);
writeFileSync(join(wsDir, 'docs', 'notes.txt'), notes);

// One commit, then an uncommitted change to src/app.ts so a working-tree
// diff exists: a few changed lines, one added line, one removed line.
const git = (args) => execFileSync('git', args, { cwd: wsDir, encoding: 'utf8' });
git(['init', '-b', 'main']);
// Keep the fixture's line endings literal on every host: a global
// core.autocrlf would make `git diff` emit "in the working copy" warnings
// that leak into the native workbench's parsed patch as phantom file rows.
git(['config', 'core.autocrlf', 'false']);
git(['-c', 'user.name=ui-compare', '-c', 'user.email=ui-compare@local', 'add', '-A']);
git(['-c', 'user.name=ui-compare', '-c', 'user.email=ui-compare@local', 'commit', '-m', 'fixture baseline', '-q']);

const dirty = appLines.slice();
dirty[3] = `  readonly name: string // display name`;
dirty.splice(10, 0, `  channel: 'stable',`); // one added line
dirty.splice(25, 1); // drop the blank line before main: one removed line
dirty[28] = `    console.log(banner, 'try', attempt)`;
writeFileSync(join(wsDir, 'src', 'app.ts'), dirty.join('\n') + '\n');

// ---- window.shell payload --------------------------------------------------
// The served web client's files/diff pages read the desktop preload's
// window.shell IPC, absent in a plain browser. Like the product's own
// browser-lane e2e, the runner injects a shim — but backed by this real
// listing/content/git data, never fabricated rows.

const tree = {};
const texts = {};
// The product's file-tree order: directories first, then locale name
// (FileTree orderOf); the desktop preload lists in that same order.
const byName = new Intl.Collator('zh-Hans', { sensitivity: 'base' });
const walk = (dir) => {
  const rel = relative(wsDir, dir).split('\\').join('/');
  tree[rel] = readdirSync(dir)
    .filter(name => name !== '.git')
    .map(name => ({ name, kind: statSync(join(dir, name)).isDirectory() ? 'directory' : 'file' }))
    .sort((l, r) => Number(r.kind === 'directory') - Number(l.kind === 'directory') || byName.compare(l.name, r.name));
  for (const e of tree[rel]) {
    const full = join(dir, e.name);
    const key = rel === '' ? e.name : `${rel}/${e.name}`;
    if (e.kind === 'directory') walk(full);
    else texts[key] = readFileSync(full, 'utf8');
  }
};
walk(wsDir);

// Parse `git diff` into the DiffPanel's DiffFile[] shape.
const parseDiff = (patch) => {
  const files = [];
  let file = null, hunk = null;
  for (const line of patch.split('\n')) {
    if (line.startsWith('diff --git ')) {
      file = { path: '', status: 'modified', hunks: [] };
      files.push(file);
      hunk = null;
    } else if (line.startsWith('+++ b/') && file) {
      file.path = line.slice(6);
    } else if (line.startsWith('new file mode') && file) {
      file.status = 'added';
    } else if (line.startsWith('deleted file mode') && file) {
      file.status = 'deleted';
    } else if (line.startsWith('@@ ') && file) {
      hunk = { header: line, lines: [] };
      file.hunks.push(hunk);
    } else if (hunk) {
      if (line.startsWith('+')) hunk.lines.push({ kind: 'add', text: line.slice(1) });
      else if (line.startsWith('-')) hunk.lines.push({ kind: 'del', text: line.slice(1) });
      else if (line.startsWith('\\')) hunk.lines.push({ kind: 'eof', text: line });
      else hunk.lines.push({ kind: 'context', text: line.startsWith(' ') ? line.slice(1) : line });
    }
  }
  return { files };
};

const diff = parseDiff(git(['diff']));
const entries = git(['status', '--porcelain=v1', '-uall'])
  .split('\n').filter(Boolean)
  .map(line => ({ path: line.slice(3), xy: line.slice(0, 2) }));
const branches = git(['branch', '--format=%(refname:short)%09%(HEAD)'])
  .split('\n').filter(Boolean)
  .map(line => { const [name, head] = line.split('\t'); return { name, isCurrent: head === '*' }; });
const payload = { tree, texts, diff, entries, branches, defaultRef: 'main' };
writeFileSync(join(dirname(wsDir), 'files-shell.json'), JSON.stringify(payload));
console.log(`[seed-files] workspace fixture -> ${wsDir}`);
