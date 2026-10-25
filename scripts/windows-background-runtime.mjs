// Reproducible Windows console fix for the selected desktop remote plugin.
// Changes only copied runtime files; the core Host and readiness pins stay intact.
import fs from 'node:fs/promises';
import path from 'node:path';
import crypto from 'node:crypto';
import {fileURLToPath} from 'node:url';

export const patchID = 'mygo-windows-background-process-v1';
export const patches = Object.freeze([
  {
    file: 'dsh-remote/lib/credential.js',
    baseSHA256: '2bdd220dac68da8391692fb03c671915a01cc8a50189b6e77f4efab9f35a6887',
    outputSHA256: 'c802cb5f03e1e0900cfe00ab8da65e5e201d6cbfc01f1a886d613b5ae10fa510',
    before: 'execFile(bin, args, { timeout: TIMEOUT, maxBuffer: 1 << 20,',
    after: 'execFile(bin, args, { windowsHide: true, timeout: TIMEOUT, maxBuffer: 1 << 20,',
  },
  {
    file: 'dsh-remote/lib/index.js',
    baseSHA256: '817b6d0823e7e340ec85c2408d1f2aa54b3f96209ae0ea05ebd9e530f21404e9',
    outputSHA256: '1fa59b08118575785221d9a36b19a2fb7325b6d21b9bdfdb5928b9f0d3859c13',
    before: 'execFile(bin, args, { timeout: PICK_TIMEOUT_MS, maxBuffer: 1 << 20 },',
    after: 'execFile(bin, args, { windowsHide: true, timeout: PICK_TIMEOUT_MS, maxBuffer: 1 << 20 },',
  },
]);
const hash = value => crypto.createHash('sha256').update(value).digest('hex');

export function patchBackgroundSource(patch, source) {
  const digest = hash(source);
  if (digest === patch.outputSHA256) return source;
  if (digest !== patch.baseSHA256 || source.split(patch.before).length !== 2) {
    throw new Error(`Windows background patch requires the pinned original or patched bytes: ${patch.file}`);
  }
  const output = source.replace(patch.before, patch.after);
  if (hash(output) !== patch.outputSHA256) throw new Error(`Windows background patch output differs: ${patch.file}`);
  return output;
}
function within(root, filename) {
  const relative = path.relative(root, filename);
  return relative !== '' && !path.isAbsolute(relative) && relative !== '..' && !relative.startsWith('..' + path.sep);
}

/** Patch both known files only after preflighting the entire pair. Unknown
 * versions fail closed, and repeat packaging accepts the same fixed bytes. */
export async function applyWindowsBackgroundRuntime(vendorDir) {
  const root = await fs.realpath(path.resolve(vendorDir));
  try { await fs.stat(path.join(root, 'dsh-remote/package.json')); }
  catch (error) { if (error.code === 'ENOENT') return {id: patchID, artifacts: {}}; throw error; }
  const prepared = [];
  for (const patch of patches) {
    const filename = path.join(root, patch.file);
    if (!within(root, await fs.realpath(filename)) || (await fs.lstat(filename)).isSymbolicLink()) {
      throw new Error(`Windows background patch target escapes the copied runtime: ${patch.file}`);
    }
    const source = await fs.readFile(filename, 'utf8');
    prepared.push({patch, filename, source, output: patchBackgroundSource(patch, source)});
  }
  const artifacts = {};
  for (const {patch, filename, source, output} of prepared) {
    if (source !== output) await fs.writeFile(filename, output);
    artifacts[patch.file] = patch.outputSHA256;
  }
  // Extend this copy's provenance without rewriting any core artifact identity.
  const manifestPath = path.join(root, 'host-session-runtime.json');
  try {
    if (!within(root, await fs.realpath(manifestPath)) || (await fs.lstat(manifestPath)).isSymbolicLink()) {
      throw new Error('Windows background patch manifest escapes the copied runtime');
    }
    const manifest = JSON.parse(await fs.readFile(manifestPath, 'utf8'));
    manifest.artifacts = {...manifest.artifacts, ...artifacts};
    manifest.windowsBackgroundPatch = {id: patchID, artifacts};
    await fs.writeFile(manifestPath, JSON.stringify(manifest, null, 2) + '\n');
  } catch (error) { if (error.code !== 'ENOENT') throw error; }
  return {id: patchID, artifacts};
}

if (process.argv[1] && path.resolve(process.argv[1]) === fileURLToPath(import.meta.url)) {
  try {
    const [vendor, ...extra] = process.argv.slice(2);
    if (!vendor || extra.length) throw new Error('Usage: windows-background-runtime.mjs <copied vendor directory>');
    console.log(JSON.stringify(await applyWindowsBackgroundRuntime(vendor), null, 2));
  } catch (error) { console.error(error.message); process.exitCode = 1; }
}
