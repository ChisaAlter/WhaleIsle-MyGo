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

const usage = 'Usage: runtime-fork-audit.mjs <upstream-tree> <desktop-harness-root> [--expect=<fork pin.json>]';

/** Compare against the packages the fork pin records, so a bump cannot drop one silently. */
function expectPackages(file) {
  const pin = JSON.parse(fs.readFileSync(file, 'utf8'));
  const recorded = pin.carry?.packages;
  if (!Array.isArray(recorded)) throw new Error(`${file} records no carry.packages list`);
  return recorded;
}

if (process.argv[1] && path.resolve(process.argv[1]) === fileURLToPath(import.meta.url)) {
  const [upstreamRoot, desktopRoot, ...flags] = process.argv.slice(2);
  const expectFlag = flags.find(flag => flag.startsWith('--expect='));
  if (!upstreamRoot || !desktopRoot || desktopRoot.startsWith('--') || flags.some(flag => !flag.startsWith('--expect='))) {
    console.error(usage);
    process.exitCode = 2;
  } else {
    try {
      const report = auditFork(upstreamRoot, desktopRoot);
      if (expectFlag) {
        const recorded = expectPackages(expectFlag.slice('--expect='.length));
        const missing = recorded.filter(name => !report.forkOnly.includes(name));
        const unexpected = report.forkOnly.filter(name => !recorded.includes(name));
        if (missing.length || unexpected.length) {
          console.error(`Fork audit: the carry no longer matches its record (${report.forkOnly.length} package(s) over the base tag, ${recorded.length} recorded).`);
          for (const name of missing) console.error(`  missing  packages/${name}`);
          for (const name of unexpected) console.error(`  unrecorded  packages/${name}`);
          console.error('A merge that adds or drops a fork-only package changes what the composition can mount; record it with the bump instead of letting the stage discover it.');
          process.exitCode = 1;
        } else {
          console.log(`Fork audit OK: ${report.forkOnly.length} package(s) over the base tag, exactly the recorded carry.`);
        }
      } else if (!report.forkOnly.length) {
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
