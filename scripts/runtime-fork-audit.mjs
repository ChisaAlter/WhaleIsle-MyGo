// Name the packages a 本体 vendor carries that an upstream tag does not.
// A harness version can only be staged from that vendor when this list is not
// empty: the desktop rows mount those packages by name, and a stage assembled
// from the tag alone boots a Host that silently lacks them.
import fs from 'node:fs';
import path from 'node:path';
import {fileURLToPath} from 'node:url';

/** Package directory names under packages/<group>/<name>, one level each. */
function packageNames(root) {
  const base = path.join(root, 'packages');
  const found = new Set();
  for (const group of fs.readdirSync(base, {withFileTypes: true})) {
    if (!group.isDirectory()) continue;
    for (const entry of fs.readdirSync(path.join(base, group.name), {withFileTypes: true})) {
      if (entry.isDirectory()) found.add(`${group.name}/${entry.name}`);
    }
  }
  return found;
}

export function auditFork(upstreamRoot, desktopRoot) {
  const upstream = packageNames(upstreamRoot);
  const desktop = packageNames(desktopRoot);
  return {
    upstreamPackages: upstream.size,
    desktopPackages: desktop.size,
    forkOnly: [...desktop].filter(name => !upstream.has(name)).sort(),
    upstreamOnly: [...upstream].filter(name => !desktop.has(name)).sort(),
  };
}

const usage = 'Usage: runtime-fork-audit.mjs <upstream-tree> <desktop-harness-root>';

if (process.argv[1] && path.resolve(process.argv[1]) === fileURLToPath(import.meta.url)) {
  const [upstreamRoot, desktopRoot] = process.argv.slice(2);
  if (!upstreamRoot || !desktopRoot) {
    console.error(usage);
    process.exitCode = 2;
  } else {
    try {
      const report = auditFork(upstreamRoot, desktopRoot);
      if (!report.forkOnly.length) {
        console.log(`Fork audit: the desktop vendor adds no package over the upstream tree (${report.upstreamPackages} packages each); a stage from the tag is equivalent.`);
      } else {
        console.error(`Fork audit: the desktop vendor carries ${report.forkOnly.length} package(s) the upstream tree does not:`);
        for (const name of report.forkOnly) console.error(`  packages/${name}`);
        console.error('A stage for this version must copy that vendor; the upstream tag alone cannot carry the rows those packages mount.');
        process.exitCode = 1;
      }
    } catch (error) {
      console.error(`Fork audit: ${error.message}`);
      process.exitCode = 2;
    }
  }
}
