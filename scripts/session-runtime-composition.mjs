// Assert a staged Host carries the composition the 本体 desktop ships for its
// harness version. 本体 vendors its own composition rows (mcp-servers-file,
// llm-vision-fallback, the tools plugin) on top of upstream, and every supported
// staging path copies an installed 本体 graph verbatim, so a stage from any other
// source silently produces a Host with different rows. This check is the boundary
// that refuses such a runtime instead of shipping it.
import fs from 'node:fs/promises';
import path from 'node:path';
import crypto from 'node:crypto';
import {fileURLToPath} from 'node:url';

const directory = path.dirname(fileURLToPath(import.meta.url));
const contractFile = path.join(directory, 'runtime-composition', 'contract.json');
const hash = data => crypto.createHash('sha256').update(data).digest('hex');
const read = async filename => fs.readFile(filename, 'utf8');

export async function loadContract(file = contractFile) {
  return JSON.parse(await read(file));
}

async function harnessRoot(target) {
  const resolved = path.resolve(target);
  const nested = path.join(resolved, 'vendor', 'deepseek-harness', 'package.json');
  const vendor = path.join(resolved, 'deepseek-harness', 'package.json');
  const direct = path.join(resolved, 'package.json');
  for (const candidate of [nested, vendor, direct]) {
    try {
      const manifest = JSON.parse(await read(candidate));
      if (manifest.name !== '@deepseek-ai/dsh-root') continue;
      return {root: path.dirname(candidate), version: manifest.version, runtimeVendor: candidate === nested ? path.join(resolved, 'vendor') : candidate === vendor ? resolved : null};
    } catch (error) {
      if (error.code !== 'ENOENT') throw error;
    }
  }
  throw new Error(`No DeepSeek Harness root under ${resolved} (expected a runtime root holding vendor/deepseek-harness, that vendor directory, or the harness root itself)`);
}

/** Row id to its whitespace-insensitive block, so two trees compare by content. */
function rows(text) {
  const found = new Map();
  let id = null;
  let block = [];
  const flush = () => {
    if (id === null) return;
    const body = block.map(line => line.trim()).filter(Boolean).join('\n');
    found.set(id, found.has(id) ? `${found.get(id)}\n---\n${body}` : body);
  };
  for (const line of text.split('\n')) {
    const match = /^\s*- id: (\S+)\s*$/.exec(line);
    if (match) {
      flush();
      id = match[1];
      block = [];
    } else if (id !== null) block.push(line);
  }
  flush();
  return found;
}

async function upstreamText(entry) {
  const filename = path.isAbsolute(entry.upstream.file) ? entry.upstream.file : path.join(directory, 'runtime-composition', entry.upstream.file);
  const text = await read(filename);
  const actual = hash(text);
  if (actual !== entry.upstream.sha256) {
    throw new Error(`${entry.upstream.file} is not the recorded pin: sha256 ${actual}, contract says ${entry.upstream.sha256}. Restore the tracked pin copy; a mismatch here means the tracked source drifted, not the staged runtime.`);
  }
  return text;
}

function compositionOf(root, contract) {
  return path.join(root, ...contract.file.split('/'));
}

function counts(text) {
  return {bytes: Buffer.byteLength(text), lines: text.split('\n').length - (text.endsWith('\n') ? 1 : 0)};
}

export async function verifyComposition(target, options = {}) {
  const contract = options.contract ?? await loadContract();
  const {root, version, runtimeVendor} = await harnessRoot(target);
  const entry = contract.versions[version];
  if (!entry) {
    throw new Error(`Harness ${version} has no composition contract. Stage it, confirm its ${contract.file} matches the 本体 desktop release that vendors this harness, then record that sha256 in scripts/runtime-composition/contract.json. Known versions: ${Object.keys(contract.versions).join(', ')}.`);
  }
  if (runtimeVendor) {
    let recorded = null;
    try { recorded = JSON.parse(await read(path.join(runtimeVendor, 'harness-upstream.json'))); } catch (error) { if (error.code !== 'ENOENT') throw error; }
    if (recorded && (recorded.ref !== entry.harnessPin.ref || recorded.sha !== entry.harnessPin.sha)) {
      throw new Error(`Stage records harness ${recorded.ref} @ ${String(recorded.sha).slice(0, 12)}, but version ${version} belongs to ${entry.harnessPin.ref} @ ${entry.harnessPin.sha.slice(0, 12)}. The stage mixes a harness version with another pin's record; restage it.`);
    }
  }
  const filename = compositionOf(root, contract);
  const text = await read(filename);
  const observed = hash(text);
  const size = counts(text);
  const pin = await upstreamText(entry);
  const acceptable = [{sha256: entry.sha256, why: entry.desktop.reference}, ...(entry.alternates ?? [])];
  const matched = acceptable.find(candidate => candidate.sha256 === observed);
  if (matched) {
    await assertDesktopPackages(root, entry);
    return {root, filename, version, sha256: observed, ...size, expected: matched.why, alternate: matched !== acceptable[0]};
  }

  const stagedRows = rows(text);
  const pinRows = rows(pin);
  const expectedRows = entry.desktop.rows ?? [];
  const renamed = entry.desktop.renamedRows ?? {};
  const missingRows = expectedRows.filter(id => !stagedRows.has(id));
  const desktopIds = [...expectedRows, ...Object.values(renamed)];
  const unpinned = [...stagedRows.keys()].filter(id => !pinRows.has(id) && !desktopIds.includes(id));
  const staleIds = Object.entries(renamed).filter(([from, to]) => stagedRows.has(from) && !stagedRows.has(to)).map(([, to]) => to);
  const missingConfig = (entry.desktop.configs ?? []).filter(snippet => !text.includes(snippet));
  const details = [
    missingRows.length ? `missing rows the desktop composes: ${missingRows.join(', ')}` : '',
    staleIds.length ? `rows still under their upstream id: ${staleIds.join(', ')}` : '',
    unpinned.length ? `rows this contract does not account for: ${unpinned.join(', ')}` : '',
    missingConfig.length ? `missing desktop config: ${missingConfig.join(', ')}` : '',
    text.includes('\r\n') ? 'the staged file uses CRLF line endings; every recorded expectation is LF' : '',
    `rows added over the ${entry.harnessPin.ref} pin: ${[...stagedRows.keys()].filter(id => !pinRows.has(id)).join(', ') || 'none'}`,
  ].filter(Boolean).join('\n  ');
  throw new Error(`Host composition does not match the tracked contract.\n  file      : ${filename}\n  harness   : ${version}\n  observed  : ${observed.slice(0, 16)} (${size.bytes} bytes, ${size.lines} lines)\n  expected  : ${entry.sha256.slice(0, 16)} — ${entry.desktop.reference}${(entry.alternates ?? []).map(alt => `\n  also ok   : ${alt.sha256.slice(0, 16)} — ${alt.why}`).join('')}\n  ${details}\nA staged tree this wrong came from a source that is not the vendored 本体 graph: restage with scripts/build-harness.ps1 -RuntimeSourceResources <installed 本体> (or -SourceMode current from the pinned 本体 source) and rerun this check. Never edit ${contract.file} inside a stage.`);
}

/**
 * The desktop rows mount by package name, so a stage that carries the composed
 * file but not the packages it names boots and silently lacks them. 本体's
 * vendored harness is a fork: entries recorded ahead of a 本体 release declare
 * the packages only that fork has, and the upstream tag cannot supply them.
 */
async function assertDesktopPackages(root, entry) {
  const required = entry.staging?.desktopPackages ?? [];
  const missing = [];
  for (const relative of required) {
    const stat = await fs.stat(path.join(root, relative)).catch(() => null);
    if (!stat?.isDirectory()) missing.push(relative);
  }
  if (!missing.length) return;
  throw new Error([
    'Host composition names a package this stage does not carry.',
    `  version : ${entry.harnessPin.ref}`,
    `  missing : ${missing.join(', ')}`,
    '  本体 vendors these on top of the upstream tag, so a stage composed from the tag alone',
    '  mounts rows it cannot load. Stage from the 本体 vendor that carries them',
    '  (scripts/build-harness.ps1 -RuntimeSourceResources <installed 本体>).',
    '  scripts/runtime-fork-audit.mjs names every such package for a desktop vendor and a tag.',
  ].join('\n'));
}

export async function diffComposition(left, right, options = {}) {
  const contract = options.contract ?? await loadContract();
  const describe = async target => {
    const {root, version} = await harnessRoot(target);
    const text = await read(compositionOf(root, contract));
    return {root, version, sha256: hash(text), ...counts(text), rows: rows(text)};
  };
  const [one, two] = [await describe(left), await describe(right)];
  const onlyIn = (source, other) => [...source.rows.keys()].filter(id => !other.rows.has(id));
  return {
    left: {root: one.root, version: one.version, sha256: one.sha256, bytes: one.bytes, lines: one.lines},
    right: {root: two.root, version: two.version, sha256: two.sha256, bytes: two.bytes, lines: two.lines},
    rowsOnlyLeft: onlyIn(one, two),
    rowsOnlyRight: onlyIn(two, one),
    rowsWithDifferentConfig: [...one.rows.keys()].filter(id => two.rows.has(id) && one.rows.get(id) !== two.rows.get(id)),
  };
}

async function main(args) {
  const [command, ...rest] = args;
  if (command === 'verify' && rest.length === 1) {
    const result = await verifyComposition(rest[0]);
    console.log(`Host composition OK: harness ${result.version}, ${result.sha256} (${result.bytes} bytes, ${result.lines} lines)${result.alternate ? ` — ${result.expected}` : ''}`);
    return;
  }
  if (command === 'diff' && rest.length === 2) {
    const result = await diffComposition(rest[0], rest[1]);
    console.log(JSON.stringify(result, null, 2));
    return;
  }
  throw new Error('Usage: session-runtime-composition.mjs verify <runtime-or-harness-root> | diff <runtime-or-harness-root> <runtime-or-harness-root>');
}

if (process.argv[1] && path.resolve(process.argv[1]) === fileURLToPath(import.meta.url)) {
  try {
    await main(process.argv.slice(2));
  } catch (error) {
    console.error(error.message);
    process.exitCode = 1;
  }
}
