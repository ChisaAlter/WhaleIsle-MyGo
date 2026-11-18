import { lstatSync, readdirSync, realpathSync, rmSync } from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

// Callers validate their owned stage/workspace before removal.
// Pre-unlink descendant junctions/symlinks without descending into targets,
// then remove remaining real directories and files.
function unlinkNestedReparsePoints(dir) {
  for (const entry of readdirSync(dir, { withFileTypes: true })) {
    const child = path.join(dir, entry.name);
    if (entry.isSymbolicLink()) {
      rmSync(child, { force: true });
    } else if (entry.isDirectory()) {
      unlinkNestedReparsePoints(child);
    }
  }
}

function assertRealAncestors(target) {
  const same = (a, b) => process.platform === 'win32' ? a.toLowerCase() === b.toLowerCase() : a === b;
  for (let current = target; ; current = path.dirname(current)) {
    const stat = lstatSync(current);
    if (!stat.isDirectory() || stat.isSymbolicLink() || !same(realpathSync.native(current), current)) {
      throw new Error(`Refusing redirected package cleanup path: ${current}`);
    }
    if (path.dirname(current) === current) break;
  }
}

function removeOwnedDirectory(target) {
  assertRealAncestors(target);
  unlinkNestedReparsePoints(target);
  rmSync(target, { recursive: true });
}

export function removePackageStage(stage, outputRoot) {
  const target = path.resolve(stage);
  if (target !== path.join(path.resolve(outputRoot), 'whaleisle')) throw new Error('Cleanup target must be this output directory\'s whaleisle stage');
  removeOwnedDirectory(target);
}

export function removeBuildWorkspace(workspace, tempRoot) {
  const target = path.resolve(workspace);
  if (path.dirname(target) !== path.resolve(tempRoot) || !/^whaleisle-build-[a-f0-9]{32}$/.test(path.basename(target))) {
    throw new Error('Cleanup requires an owned whaleisle-build workspace under the selected TEMP');
  }
  removeOwnedDirectory(target);
}

export function removeOwnedDir(targetDir, parentDir, prefix) {
  const target = path.resolve(targetDir);
  const parent = path.resolve(parentDir);
  if (!prefix || path.dirname(target) !== parent || !path.basename(target).startsWith(prefix)) {
    throw new Error(`Cleanup requires an owned directory prefixed with "${prefix}" under ${parent}`);
  }
  try {
    lstatSync(target);
  } catch (error) {
    if (error && error.code === 'ENOENT') return;
    throw error;
  }
  removeOwnedDirectory(target);
}

if (process.argv[1] && path.resolve(process.argv[1]) === fileURLToPath(import.meta.url)) {
  try {
    if (process.argv[2] === 'package-stage' && process.argv.length === 5) removePackageStage(process.argv[3], process.argv[4]);
    else if (process.argv[2] === 'workspace' && process.argv.length === 5) removeBuildWorkspace(process.argv[3], process.argv[4]);
    else if (process.argv[2] === 'owned-dir' && process.argv.length === 6) removeOwnedDir(process.argv[3], process.argv[4], process.argv[5]);
    else throw new Error('Usage: node build-storage.mjs package-stage|workspace TARGET PARENT | owned-dir TARGET PARENT PREFIX');
  } catch (error) {
    console.error(error.message);
    process.exitCode = 1;
  }
}
