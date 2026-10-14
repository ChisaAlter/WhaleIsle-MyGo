'use strict';
// Adapter for the known original DshdRemote runtime. Its daemon runner retains
// JSON stdout and stop/stdin-close lifecycle. No protocol or credential changes.
// Only structured UI snapshots leave this process; raw logs/errors never do.
const write = process.stdout.write.bind(process.stdout);
const emit = (data) => write(JSON.stringify(data) + '\n');
console.log = console.info = console.warn = console.error = () => {};
const readline = require('node:readline');
let remote, config = {}, stopping = false, starting = false;
try {
  const { DshdRemote } = require(process.argv[1]);
  remote = new DshdRemote({
    getConfig: () => config,
    getHomeDir: () => process.argv[2],
    execPath: process.execPath,
    log: () => {},
    getHarnessOrigin: () => config.harnessOrigin || '',
    getSessionCookie: () => config.harnessCookie || '',
  });
} catch { emit({kind:'error',code:'runtime-load'}); process.exit(1); }
function snapshot() {
  if (stopping || starting) return;
  const s = remote.snapshot();
  if (s.error) { emit({kind:'error',code:'operation'}); return; }
  emit({kind:'snapshot', listening:!!s.listening, relayConnected:!!s.relayConnected,
    url:s.listening ? (s.urls?.[0]?.pairingUrl || '') : ''});
}
let queue = Promise.resolve();
async function stop() {
  if (stopping) return;
  stopping = true;
  clearInterval(timer);
  await remote.stopDaemon();
  emit({kind:'snapshot',listening:false,relayConnected:false,url:''});
  process.exit(0);
}
const timer = setInterval(() => { try { snapshot(); } catch { emit({kind:'error',code:'snapshot'}); } },1000);
remote.on('listening',snapshot);
const input=readline.createInterface({input:process.stdin});
input.on('line',line=>{
  if(line.length>128*1024)return;
  queue=queue.then(async()=>{
    const command=JSON.parse(line);
    if(command.action==='stop')return stop();
    if(stopping)return;
    if(command.action==='start') {config=command.config||{};starting=true;try { await remote.sync(); } finally { starting=false; }}
    else if(command.action==='refresh' && remote.daemon) {config=command.config||config;await remote.sync();}
    snapshot();
  }).catch(()=>emit({kind:'error',code:'operation'}));
});
input.on('close',()=>{queue=queue.then(stop).catch(()=>process.exit(1));});
process.on('SIGTERM',()=>{queue=queue.then(stop).catch(()=>process.exit(1));});
