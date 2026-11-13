import { lstatSync, realpathSync, rmSync } from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

// Callers validate their owned stage/workspace before removal.
// fs.rm unlinks descendant junctions/symlinks;
// it does not descend into their external targets.
function removeOwnedDirectory(target) {
  const same = (a, b) => process.platform === 'win32' ? a.toLowerCase() === b.toLowerCase() : a === b;
  for (let current = target; ; current = path.dirname(current)) {
    const stat = lstatSync(current);
    if (!stat.isDirectory() || stat.isSymbolicLink() || !same(realpathSync.native(current), current)) {
      throw new Error(`Refusing redirected package cleanup path: ${current}`);
    }
    if (path.dirname(current) === current) break;
  }
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

if (process.argv[1] && path.resolve(process.argv[1]) === fileURLToPath(import.meta.url)) {
  try {
    if (process.argv.length !== 5) throw new Error('Usage: node build-storage.mjs package-stage|workspace TARGET PARENT');
    if (process.argv[2] === 'package-stage') removePackageStage(process.argv[3], process.argv[4]);
    else if (process.argv[2] === 'workspace') removeBuildWorkspace(process.argv[3], process.argv[4]);
    else throw new Error('Unknown cleanup operation');
  } catch (error) {
    console.error(error.message);
    process.exitCode = 1;
  }
}
