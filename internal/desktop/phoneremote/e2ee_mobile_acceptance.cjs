'use strict';
// The original Worker and the shipped phone client run unchanged. This driver
// supplies only loopback hosting and an isolated in-memory browser store.
const fs = require('node:fs');
const path = require('node:path');
const readline = require('node:readline');
const {pathToFileURL} = require('node:url');
const {createRequire} = require('node:module');
const [work, component, toolRoot] = process.argv.slice(2);
const write = process.stdout.write.bind(process.stdout);
const emit = value => write(JSON.stringify(value) + '\n');
console.log = console.info = console.warn = console.error = () => {};
let relay, builder, mobile, api, active, endpoint, closing;
const clients = new Set();
const memory = new Map();
globalThis.self = globalThis;
globalThis.localStorage = {
  getItem: key => memory.get(key) ?? null,
  setItem: (key, value) => memory.set(key, String(value)),
  removeItem: key => memory.delete(key),
};
const assert = (condition, message) => { if (!condition) throw Error(message); };

async function close() {
  if (closing) return closing;
  closing = (async () => {
    await Promise.all([...clients].map(client => client.close().catch(() => {})));
    clients.clear();
    memory.clear();
    if (relay) await relay.dispose();
    if (builder) builder.stop();
  })();
  return closing;
}

async function connect(offer, sticky = false) {
  mobile.rotateClientId();
  const signal = AbortSignal.timeout(12000);
  const value = sticky
    ? await mobile.reconnectSticky(api, offer, {signal})
    : await mobile.pairFromOfferUrl(api, offer, {signal});
  value.client.setReconnectEnabled(false);
  clients.add(value.client);
  return value;
}

async function rejected(run, reason) {
  const auditPath = path.join(work, 'auth-audit.jsonl');
  const count = () => {
    try { return fs.readFileSync(auditPath, 'utf8').trim().split('\n').filter(line => JSON.parse(line).reason === reason).length; }
    catch { return 0; }
  };
  const before = count();
  let caught;
  try {
    const value = await run();
    await value.client.close();
    throw Error('authentication unexpectedly succeeded');
  } catch (error) {
    caught = error;
  }
  // The unchanged Worker deliberately translates daemon 4401 to relay 1012.
  // Assert the actual daemon's structured rejection, not a timeout or a
  // localized phone error that the relay does not preserve.
  assert(/Server disconnected/.test(caught?.message || ''), 'expected relay close after rejected hello');
  for (let i=0; i<20 && count()===before; i++) await new Promise(resolve => setTimeout(resolve,25));
  assert(count()>before, 'daemon did not record '+reason);
}

async function command(message) {
  if (message.action === 'pair') {
    const offer = api.parseConnectionOfferFromUrl(message.offer);
    assert(offer.v === 2 && offer.relay.endpoint === endpoint && offer.relay.useTls === false, 'offer did not target owned relay');
    active = await connect(message.offer);
    await active.client.ping({timeoutMs: 5000});
    const stored = mobile.loadSecrets()[active.serverId];
    assert(stored?.deviceSecret && stored?.deviceId, 'phone did not retain formal device identity');
    assert(active.client.getLastServerInfoMessage(), 'application hello did not complete');
    emit({kind:'paired', deviceId:stored.deviceId, hello:true, encryptedPing:true, secretPersisted:true});
    return;
  }
  if (message.action === 'sticky') {
    const id = active.serverId;
    await active.client.close();
    active = await connect(id, true);
    await active.client.ping({timeoutMs: 5000});
    emit({kind:'sticky', encryptedPing:true, tokenRequired:false});
    return;
  }
  if (message.action === 'replay') {
    await rejected(() => connect(message.offer), 'relay_pairing_token_invalid');
    // Keep the already authenticated device alive through the rejected replay.
    await active.client.ping({timeoutMs: 5000});
    emit({kind:'replay-rejected', activeDeviceUnaffected:true});
    return;
  }
  if (message.action === 'revoked') {
    let oldConnectionRejected = false;
    try { await active.client.ping({timeoutMs:1000}); } catch { oldConnectionRejected = true; }
    assert(oldConnectionRejected, 'revoked live device retained application access');
    await rejected(() => connect(active.serverId, true), 'relay_device_auth_invalid');
    emit({kind:'revoked-rejected', liveConnectionClosed:true, newProofRejected:true});
    return;
  }
  if (message.action === 'stop') {
    await close();
    emit({kind:'closed'});
    process.exit(0);
  }
  throw Error('unknown acceptance command');
}

async function main() {
  for (const input of [work, component, toolRoot]) assert(path.isAbsolute(input), 'absolute owned input required');
  const tools = createRequire(path.join(toolRoot, 'package.json'));
  builder = tools('esbuild');
  const worker = path.join(work, 'relay-worker.mjs');
  await builder.build({entryPoints:[path.join(component, 'node_modules/@chisacode/relay/dist/cloudflare-adapter.js')], outfile:worker, bundle:true, format:'esm', platform:'browser', target:'es2022', logLevel:'silent'});
  const {Miniflare, Log, LogLevel} = tools('miniflare');
  relay = new Miniflare({rootPath:work, modulesRoot:work, scriptPath:worker, modules:true, compatibilityDate:'2024-12-01', host:'127.0.0.1', port:0, durableObjects:{RELAY:{className:'RelayDurableObject',useSQLite:true}}, durableObjectsPersist:path.join(work,'relay-db'), log:new Log(LogLevel.NONE)});
  const url = await relay.ready;
  assert(url.hostname === '127.0.0.1', 'relay escaped loopback');
  endpoint = url.host;
  api = await import(pathToFileURL(path.join(component,'mobile/web/chisacode/daemon-client.bundle.js')));
  mobile = await import(pathToFileURL(path.join(component,'mobile/web/chisacode/session.js')));
  emit({kind:'ready', endpoint});
}

let queue = main();
const input = readline.createInterface({input:process.stdin});
input.on('line', line => {
  queue = queue.then(() => command(JSON.parse(line))).catch(async error => {
    // Known assertion messages contain no offers, tokens, or stored secrets.
    emit({kind:'error', message:String(error.message).replace(/https?:\/\/\S+/g,'[endpoint]')});
    await close();
    process.exit(1);
  });
});
input.on('close', () => { queue.finally(close).finally(() => process.exit(0)); });
queue.catch(async error => { emit({kind:'error', message:error.code || error.name || 'relay startup failed'}); await close(); process.exit(1); });
