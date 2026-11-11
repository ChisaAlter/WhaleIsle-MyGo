// Preserve MyGo's optional dshbot 0.2.0 when the newer desktop no longer bundles it.
import fs from 'node:fs';
import path from 'node:path';
import {fileURLToPath} from 'node:url';
import {cloneGraph} from './session-runtime-graph.mjs';
import {verifyStagedSessionRuntime} from './session-runtime.mjs';

export async function carryDshbot(sourceVendor, targetStage) {
  const targetVendor = path.join(fs.realpathSync(targetStage), 'vendor');
  const host = await verifyStagedSessionRuntime(targetVendor);
  if (host.cliVersion !== '0.2.1-rc.4') throw Error('dshbot carry-forward requires the rc.4 target');
  const source = fs.realpathSync(path.join(sourceVendor, 'dshbot'));
  const manifest = JSON.parse(fs.readFileSync(path.join(source, 'package.json'), 'utf8'));
  if (manifest.name !== 'dshbot' || manifest.version !== '0.2.0') throw Error('Unsupported source dshbot package');
  const peers = ['dsh-agent', 'dsh-llm', 'dsh-scope', 'dsh-session', 'dsh-settings', 'dsh-subagent', 'dsh-system-prompt', 'dsh-tools'];
  const before = {};
  for (const name of peers) {
    const key = '@deepseek-ai/' + name;
    const value = manifest.peerDependencies?.[key];
    if (typeof value !== 'string' || !value.split('||').some(part => part.trim() === '0.2.1-rc.2')) throw Error(`Unsupported dshbot peer: ${key}`);
    const installed = JSON.parse(fs.readFileSync(path.join(targetVendor, 'deepseek-harness/node_modules', key, 'package.json'), 'utf8'));
    if (installed.version !== '0.2.1-rc.4') throw Error(`Target peer version mismatch: ${key}`);
    before[key] = value;
    manifest.peerDependencies[key] = value + ' || 0.2.1-rc.4';
  }
  const target = path.join(targetVendor, 'dshbot');
  cloneGraph(source, target);
  fs.writeFileSync(path.join(target, 'package.json'), JSON.stringify(manifest, null, 2) + '\n');
  fs.writeFileSync(path.join(target, 'mygo-compatibility.json'), JSON.stringify({schemaVersion: 1, source, version: manifest.version, targetHost: host.cliVersion, originalPeerDependencies: before, adapter: 'internal/harness/dshbot-room-adapter.mjs'}, null, 2) + '\n', {flag: 'wx'});
  return {package: manifest.name, version: manifest.version, targetHost: host.cliVersion, target};
}

if (process.argv[1] && path.resolve(process.argv[1]) === fileURLToPath(import.meta.url)) {
  const [sourceVendor, targetStage, extra] = process.argv.slice(2);
  if (!sourceVendor || !targetStage || extra) throw Error('Usage: session-runtime-dshbot.mjs <old vendor> <staged rc.4 runtime>');
  console.log(JSON.stringify(await carryDshbot(sourceVendor, targetStage), null, 2));
}
