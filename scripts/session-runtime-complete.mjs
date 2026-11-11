// One installed-rc.4 build path owns Host, phone, retained dshbot and main plugins.
import fs from 'node:fs';
import path from 'node:path';
import {fileURLToPath} from 'node:url';
import {stageSessionRuntime, verifyStagedSessionRuntime} from './session-runtime.mjs';
import {completeInstalledMobile} from './installed-mobile-runtime.mjs';
import {carryDshbot} from './session-runtime-dshbot.mjs';
import {stageDesktopPlugins} from './session-runtime-desktop-plugins.mjs';
import {verify as verifyMobile} from './mobile-runtime.mjs';
import {verifyAcp} from './session-runtime-acp.mjs';

const desktopPackages = ['dsh-whale', 'dsh-im', 'dsh-task-control', 'dsh-platform-session'];
const read = filename => JSON.parse(fs.readFileSync(filename, 'utf8'));

export async function verifyCompleteRuntime(runtimeRoot) {
  const vendor = path.join(runtimeRoot, 'vendor');
  const host = await verifyStagedSessionRuntime(vendor);
  if (host.cliVersion !== '0.2.1-rc.4') throw Error('Complete desktop assembly requires rc.4');
  const mobile = verifyMobile(path.join(vendor, 'dshd-remote'));
  const bot = read(path.join(vendor, 'dshbot/mygo-compatibility.json'));
  if (bot.version !== '0.2.0' || bot.targetHost !== host.cliVersion) throw Error('Retained dshbot is missing its rc.4 compatibility');
  let commit;
  for (const name of desktopPackages) {
    const root = path.join(vendor, name);
    const provenance = read(path.join(root, 'mygo-desktop-source.json'));
    const manifest = read(path.join(root, 'package.json'));
    if (!/^[a-f0-9]{40}$/.test(provenance.commit) || provenance.host !== host.cliVersion || provenance.package !== name || provenance.version !== manifest.version) throw Error(`Incomplete desktop plugin assembly: ${name}`);
    if (commit && commit !== provenance.commit) throw Error('Desktop plugins come from different source revisions');
    for (const entry of [manifest.main, manifest.dsh?.bundle?.patch].filter(Boolean)) {
      if (!fs.statSync(path.join(root, entry)).isFile()) throw Error(`Missing desktop entry: ${name}/${entry}`);
    }
    commit = provenance.commit;
  }
  const acpRoot=path.join(vendor,'dsh-acp');
  const acp=fs.existsSync(acpRoot)?verifyAcp(acpRoot):null;
  return {runtimeRoot: path.resolve(runtimeRoot), host: host.cliVersion, desktopCommit: commit, desktopPackages, dshbot: bot.version, mobile, acp};
}

export async function stageCompleteRuntime(resources, destination, desktopSource, revision, legacyVendor) {
  // Resolve required inputs before copying the substantial installed graph.
  for (const input of [resources, desktopSource, legacyVendor]) fs.realpathSync(input);
  const sourceHost = read(path.join(resources, 'vendor/deepseek-harness/package.json'));
  if (sourceHost.version !== '0.2.1-rc.4') throw Error('Complete desktop assembly expects installed rc.4 resources');
  await stageSessionRuntime(resources, destination);
  await completeInstalledMobile(resources, destination);
  await carryDshbot(legacyVendor, destination);
  await stageDesktopPlugins(desktopSource, revision, destination);
  return verifyCompleteRuntime(destination);
}

if (process.argv[1] && path.resolve(process.argv[1]) === fileURLToPath(import.meta.url)) {
  const [command, ...args] = process.argv.slice(2);
  if (command === 'stage' && args.length === 5) console.log(JSON.stringify(await stageCompleteRuntime(...args), null, 2));
  else if (command === 'verify' && args.length === 1) console.log(JSON.stringify(await verifyCompleteRuntime(args[0]), null, 2));
  else throw Error('Usage: session-runtime-complete.mjs stage <resources> <destination> <desktop source> <revision> <legacy vendor> | verify <runtime>');
}
