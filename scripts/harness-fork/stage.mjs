// Stage a Host runtime from the fork's built source tree.
//
// The installed path copies 本体's finished graph, so a stage is whatever that
// desktop shipped. Here the graph is assembled from the fork build instead, and
// the same stager, readiness gate and composition contract decide whether the
// result is a stage at all. Desktop plugins (dsh-whale, dsh-im, the mobile
// bridge, dshbot) are not harness source and still come from the installed
// desktop.
import fs from 'node:fs';
import path from 'node:path';
import {spawn, spawnSync} from 'node:child_process';
import {fileURLToPath} from 'node:url';
import {loadPin, tagEntry, directory as forkDirectory} from './fork.mjs';
import {carryPluginCompatibility} from './plugin-compat.mjs';
import {assembleInstanceGraph, compositionSeeds, verifyInstanceGraph} from './graph.mjs';
import {stageSessionRuntime, verifyStagedSessionRuntime} from '../session-runtime.mjs';
import {verifyComposition, loadContract} from '../session-runtime-composition.mjs';
import {extractTo} from './archive.mjs';

const scripts = path.resolve(forkDirectory, '..');
const projectRoot = path.resolve(scripts, '..');
const forkWork = path.join(projectRoot, '.harness-fork');
const contract = await loadContract();
const log = message => console.log(message);

const desktopRecords = new Set(['deepseek-harness', 'harness-upstream.json', 'host-session-runtime.json', 'runtime-graph.json', 'session-runtime-readiness.json']);

export async function stageFork({fork, tag, dest, node, desktopResources, work = forkWork}) {
  const pin = loadPin();
  const ref = tag ?? pin.bumps.at(-1)?.ref ?? pin.upstream.tags.at(-1).ref;
  const entry = tagEntry(pin, ref);
  const version = entry.ref.replace(/^dsh-v/, '');
  const destination = path.resolve(dest ?? path.join(projectRoot, `runtime-${version.replace(/\./g, '')}`));
  if (fs.existsSync(destination)) throw new Error(`Runtime destination must be fresh: ${destination}`);
  const source = path.join(work, `stage-source-${version.replace(/[^\w.-]+/g, '-')}`);
  fs.rmSync(source, {recursive: true, force: true});
  const vendor = path.join(source, 'vendor');
  const harness = path.join(vendor, 'deepseek-harness');
  fs.mkdirSync(vendor, {recursive: true});

  const seeds = compositionSeeds(fork, contract.file);
  const graph = assembleInstanceGraph({harnessRoot: fork, targetRoot: harness, seeds});
  log(`instance graph: ${graph.packages} packages, ${graph.files} files, ${graph.links} links from ${seeds.length} seeds`);
  for (const name of ['package.json', 'LICENSE']) {
    if (fs.existsSync(path.join(fork, name))) fs.copyFileSync(path.join(fork, name), path.join(harness, name));
  }
  const verified = verifyInstanceGraph({harnessRoot: fork, targetRoot: harness, seeds});
  log(`instance graph verified: ${verified.packages} packages, ${verified.files} files, ${verified.links} links`);

  fs.writeFileSync(path.join(vendor, 'harness-upstream.json'), `${JSON.stringify({repo: pin.upstream.repo, ref: entry.ref, sha: entry.sha, npm: version}, null, 2)}\n`);

  if (desktopResources) {
    const resourcesVendor = path.join(desktopResources, 'vendor');
    const carried = fs.readdirSync(resourcesVendor).filter(name => !desktopRecords.has(name));
    for (const name of carried) fs.cpSync(path.join(resourcesVendor, name), path.join(vendor, name), {recursive: true, verbatimSymlinks: true});
    log(`desktop plugins carried from ${resourcesVendor}: ${carried.length} entries`);
    const widened = carryPluginCompatibility(vendor, carried, version);
    log(`plugin compatibility for ${version}: ${widened.length === 0 ? 'every carried plugin already names it' : widened.join('; ')}`);
  }

  const nodeTarget = path.join(source, 'runtime', 'primary-runtime', 'dependencies', 'node', 'bin', 'node.exe');
  fs.mkdirSync(path.dirname(nodeTarget), {recursive: true});
  fs.copyFileSync(node ?? process.execPath, nodeTarget);

  const staged = await stageSessionRuntime(source, destination);
  log(`staged and verified: harness ${staged.cliVersion}, ${staged.relocatedLinks} relocated links`);
  return {dest: destination, version, cliVersion: staged.cliVersion, staged};
}

export async function gates({dest, fork = path.join(forkWork, 'tree')}) {
  const pin = loadPin();
  const composition = await verifyComposition(dest);
  // Audit against the tag the stage ships, not the carry's base: a tag the fork
  // carries plus a package upstream added since is not an unrecorded carry, and
  // counting it as one would make every bump fail on upstream's own new package.
  const shipped = JSON.parse(fs.readFileSync(path.join(dest, 'vendor', 'harness-upstream.json'), 'utf8'));
  const baseRef = pin.upstream.tags.some(tag => tag.ref === shipped.ref) ? shipped.ref : pin.carry.base;
  const base = path.join(forkWork, 'audit', baseRef);
  if (!fs.existsSync(path.join(base, 'packages'))) {
    fs.mkdirSync(base, {recursive: true});
    const archive = path.join(forkWork, 'audit', `${baseRef}.tar`);
    const archived = spawnSync('git', ['-C', path.join(forkWork, 'tree'), 'archive', '--format=tar', `--output=${archive}`, `refs/harness/upstream/${baseRef}`, 'packages'], {windowsHide: true});
    if (archived.status !== 0) throw new Error(`Cannot read the ${baseRef} package tree to audit the carry: ${(archived.error?.message ?? archived.stderr?.toString() ?? '').trim()}`);
    await extractTo({archive, destination: base});
    fs.rmSync(archive, {force: true});
  }
  const audit = spawnSync(process.execPath, [
    path.join(scripts, 'runtime-fork-audit.mjs'), base, fork, `--expect=${path.join(forkDirectory, 'pin.json')}`,
  ], {encoding: 'utf8', windowsHide: true});
  const auditOutput = `${audit.stdout}${audit.stderr}`.trim();
  if (audit.status !== 0) throw new Error(`Fork audit failed:\n${auditOutput}`);
  const staged = await verifyStagedSessionRuntime(path.join(dest, 'vendor'));
  return {
    composition: {version: composition.version, sha256: composition.sha256, bytes: composition.bytes, lines: composition.lines},
    forkAudit: auditOutput,
    runtime: {cliVersion: staged.cliVersion, ref: staged.harnessPin?.ref, sha: staged.harnessPin?.sha, links: staged.relocatedLinks},
  };
}

/** Boot the staged Host against a scratch home and prove it serves its web app. */
export async function probe({dest, timeout = 120000}) {
  const harness = path.join(dest, 'vendor', 'deepseek-harness');
  const home = path.join(path.dirname(dest), `.harness-probe-${process.pid}`);
  fs.rmSync(home, {recursive: true, force: true});
  fs.mkdirSync(home, {recursive: true});
  const child = spawn(process.execPath, [path.join(harness, 'apps', 'cli', 'lib', 'bin.js'), 'web', '--port', '0', '--no-open'], {
    cwd: harness,
    env: {...process.env, DSH_HOME: home},
    stdio: ['ignore', 'pipe', 'pipe'],
    windowsHide: true,
  });
  let output = '';
  try {
    const url = await new Promise((resolve, reject) => {
      const timer = setTimeout(() => reject(new Error(`The staged Host printed no URL within ${timeout / 1000}s:\n${output.slice(-2000)}`)), timeout);
      const scan = () => {
        const match = /http:\/\/(?:127\.0\.0\.1|localhost):\d+[^\s"']*/.exec(output);
        if (!match) return;
        clearTimeout(timer);
        resolve(match[0].replace(/[.,;]$/, ''));
      };
      child.stdout.on('data', chunk => { output += chunk.toString(); scan(); });
      child.stderr.on('data', chunk => { output += chunk.toString(); scan(); });
      child.on('exit', code => { clearTimeout(timer); reject(new Error(`The staged Host exited ${code} before serving:\n${output.slice(-2000)}`)); });
      scan();
    });
    // The launch URL's token mints a browser cookie and redirects; a fetch that
    // follows the redirect without the cookie is answered 401, so the proof is
    // the minted cookie and the index the cookie then serves.
    const minted = await fetch(url, {redirect: 'manual'});
    const cookie = (minted.headers.getSetCookie?.() ?? [minted.headers.get('set-cookie') ?? ''])
      .map(value => value.split(';')[0]).filter(Boolean).join('; ');
    if (minted.status < 300 || minted.status >= 400 || !cookie) {
      throw new Error(`The staged Host did not accept its launch token at ${url}: answered ${minted.status}${cookie ? '' : ' without a cookie'}`);
    }
    const location = new URL(minted.headers.get('location') ?? './', url).href;
    const response = await fetch(location, {headers: {cookie}});
    const body = await response.text();
    if (!response.ok || !/<html/i.test(body)) throw new Error(`The staged Host answered ${response.status} at ${location} (${body.length} bytes)`);
    return {url, status: response.status, bytes: body.length, boot: output.trim().split('\n').slice(-6)};
  } finally {
    child.kill();
    fs.rmSync(home, {recursive: true, force: true});
  }
}

/** Finish a stage with the parts that are the installed desktop's, not harness source. */
export async function complete({dest, desktopResources, legacyVendor}) {
  const {completeInstalledMobile} = await import('../installed-mobile-runtime.mjs');
  const {carryDshbot} = await import('../session-runtime-dshbot.mjs');
  if (!desktopResources) throw new Error('complete needs --desktop-resources: the mobile bridge and dshbot come from the installed desktop, not from the harness fork.');
  const mobile = await completeInstalledMobile(desktopResources, dest);
  const dshbot = legacyVendor ? await carryDshbot(legacyVendor, dest) : null;
  return {mobile, dshbot};
}

function flag(name) {
  const index = process.argv.indexOf(`--${name}`);
  return index === -1 ? null : process.argv[index + 1];
}

/** Positional arguments in order, with every `--flag value` pair removed. */
function positional(argv) {
  const values = [];
  for (let index = 0; index < argv.length; index++) {
    if (argv[index].startsWith('--')) { index++; continue; }
    values.push(argv[index]);
  }
  return values;
}

async function main() {
  const [command, ...args] = process.argv.slice(2);
  const rest = positional(args);
  try {
    if (command === 'stage' && rest.length === 1) {
      const result = await stageFork({
        fork: path.resolve(rest[0]),
        tag: flag('tag'),
        dest: flag('dest'),
        node: flag('node'),
        desktopResources: flag('desktop-resources'),
        work: flag('work') ? path.resolve(flag('work')) : undefined,
      });
      const report = await gates({dest: result.dest, fork: path.resolve(rest[0])});
      console.log(`Host composition OK: harness ${report.composition.version}, ${report.composition.sha256} (${report.composition.bytes} bytes, ${report.composition.lines} lines)`);
      console.log(report.forkAudit);
      console.log(`stage: ${result.dest}`);
      console.log(JSON.stringify(report.runtime, null, 2));
      return;
    }
    if (command === 'complete' && rest.length === 1) {
      const result = await complete({dest: path.resolve(rest[0]), desktopResources: flag('desktop-resources'), legacyVendor: flag('legacy-vendor')});
      console.log(JSON.stringify({mobile: result.mobile, dshbot: result.dshbot?.package ?? null}, null, 2));
      return;
    }
    if (command === 'gates' && rest.length === 1) {
      console.log(JSON.stringify(await gates({dest: path.resolve(rest[0])}), null, 2));
      return;
    }
    if (command === 'probe' && rest.length === 1) {
      const result = await probe({dest: path.resolve(rest[0]), timeout: Number(flag('timeout') ?? 120000)});
      console.log(`probe OK: ${result.url} answered ${result.status} with ${result.bytes} bytes`);
      console.log(result.boot.join('\n'));
      return;
    }
    throw new Error('Usage: stage.mjs stage <fork tree> [--tag <ref>] [--dest <fresh dir>] [--node <bin>] [--desktop-resources <installed desktop>] | gates <stage> | probe <stage> [--timeout <ms>] | complete <stage> --desktop-resources <installed desktop> [--legacy-vendor <vendor>]');
  } catch (error) {
    console.error(`harness stage: ${error.message}`);
    process.exitCode = 1;
  }
}

if (process.argv[1] && path.resolve(process.argv[1]) === fileURLToPath(import.meta.url)) {
  main().catch(error => {
    console.error(`harness stage: ${error.message}`);
    process.exitCode = 1;
  });
}
