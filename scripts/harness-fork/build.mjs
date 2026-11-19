// Build the official Host profile from the fork source tree.
//
// The vendored artifact used to arrive already built, so nobody had to know how
// it was produced. This is the recipe 本体 itself runs: the pinned pnpm, the
// frozen lockfile, and build:official, with the tag's commit hash exported so
// client artifacts carry the same provenance a git checkout would.
import fs from 'node:fs';
import path from 'node:path';
import {spawnSync} from 'node:child_process';
import {fileURLToPath} from 'node:url';

export const artifacts = [
  'apps/cli/lib/bin.js',
  'apps/web/dist/index.html',
  'packages/client/ui-user-terminal/lib/assets/ghostty-vt.wasm',
  'packages/client/ui-user-terminal/lib/assets/ghostty-write-pty.wasm',
  'packages/client/ui-user-terminal/lib/assets/SymbolsNerdFontMono-Regular.woff2',
  'packages/bundle/base/cordis.patch.yml',
];

function run(command, args, options = {}) {
  const result = spawnSync(command, args, {
    cwd: options.cwd,
    env: {...process.env, ...options.env},
    stdio: options.quiet ? 'pipe' : 'inherit',
    encoding: 'utf8',
    shell: options.shell ?? false,
    windowsHide: true,
  });
  if (result.error) throw result.error;
  if (result.status !== 0) {
    const detail = options.quiet ? `${(result.stderr || result.stdout || '').trim().split('\n').slice(-8).join('\n')}` : '';
    throw new Error(`${command} ${args.join(' ')} failed in ${options.cwd} (exit ${result.status})${detail ? `:\n${detail}` : ''}`);
  }
  return result;
}

/**
 * npm is a shell shim on Windows, so spawning it without a shell is ENOENT there.
 * Running the CLI through the node that carries it works on every host.
 */
function runNpm(args, options) {
  const cli = path.join(path.dirname(process.execPath), 'node_modules', 'npm', 'bin', 'npm-cli.js');
  if (fs.existsSync(cli)) return run(process.execPath, [cli, ...args], options);
  return run(process.platform === 'win32' ? 'npm.cmd' : 'npm', args, {...options, shell: process.platform === 'win32'});
}

/** The exact pnpm the tree pins, installed under the fork's own directory so the user's toolchain is untouched. */
export function pinnedPnpm(tree, {work, log = () => {}}) {
  const manifest = JSON.parse(fs.readFileSync(path.join(tree, 'package.json'), 'utf8'));
  const wanted = String(manifest.packageManager ?? '').replace(/^pnpm@/, '');
  if (!/^\d+\.\d+\.\d+/.test(wanted)) throw new Error(`${tree}/package.json has no pnpm packageManager pin; the build cannot pick a package manager.`);
  const tools = path.join(work, 'tools', `pnpm-${wanted}`);
  const entry = ['pnpm.cjs', 'pnpm.mjs'].map(name => path.join(tools, 'node_modules', 'pnpm', 'bin', name)).find(candidate => fs.existsSync(candidate));
  if (entry) return entry;
  fs.mkdirSync(tools, {recursive: true});
  log(`installing pnpm ${wanted} for this tree`);
  runNpm(['install', '--prefix', tools, '--ignore-scripts', '--no-audit', '--no-fund', `pnpm@${wanted}`], {quiet: true});
  const installed = ['pnpm.cjs', 'pnpm.mjs'].map(name => path.join(tools, 'node_modules', 'pnpm', 'bin', name)).find(candidate => fs.existsSync(candidate));
  if (!installed) throw new Error(`pnpm@${wanted} installed without a CLI entry under ${tools}`);
  return installed;
}

/**
 * Install and build the tree in place.
 * @returns {{tree: string, cliVersion: string, artifacts: string[]}}
 */
export function buildFork({tree, commit, work, frozen = true, log = () => {}}) {
  const pnpm = pinnedPnpm(tree, {work, log});
  const env = commit ? {DSH_CLIENT_COMMIT_HASH: commit} : {};
  log(frozen ? 'pnpm install --frozen-lockfile' : 'pnpm install (relock: this tree is mid-bump)');
  run(process.execPath, [pnpm, 'install', ...(frozen ? ['--frozen-lockfile'] : [])], {cwd: tree, env});
  log('pnpm run build:official');
  run(process.execPath, [pnpm, 'run', 'build:official'], {cwd: tree, env});
  const missing = artifacts.filter(relative => !fs.existsSync(path.join(tree, ...relative.split('/'))));
  if (missing.length) throw new Error(`build:official exited 0 without producing: ${missing.join(', ')}. The tree is not a complete Host.`);
  const cliVersion = JSON.parse(fs.readFileSync(path.join(tree, 'apps', 'cli', 'package.json'), 'utf8')).version;
  return {tree, cliVersion, artifacts};
}

if (process.argv[1] && path.resolve(process.argv[1]) === fileURLToPath(import.meta.url)) {
  try {
    const tree = process.argv[2];
    if (!tree) throw new Error('Usage: build.mjs <fork tree> [--commit <sha>] [--work <dir>] [--relock]');
    const flag = name => {
      const index = process.argv.indexOf(`--${name}`);
      return index === -1 ? null : process.argv[index + 1];
    };
    const result = buildFork({
      tree: path.resolve(tree),
      commit: flag('commit'),
      work: path.resolve(flag('work') ?? path.join(tree, '..')),
      frozen: !process.argv.includes('--relock'),
      log: message => console.log(message),
    });
    console.log(JSON.stringify(result, null, 2));
  } catch (error) {
    console.error(`harness build: ${error.message}`);
    process.exitCode = 1;
  }
}
