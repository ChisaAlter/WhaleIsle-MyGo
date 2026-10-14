'use strict';
// Adapter for the known original DshdRemote runtime. Its daemon runner retains
// JSON stdout and stop/stdin-close lifecycle. No protocol or credential changes.
// Only structured UI snapshots leave this process; raw logs/errors never do.
const write = process.stdout.write.bind(process.stdout);
const emit = (data) => write(JSON.stringify(data) + '\n');
console.log = console.info = console.warn = console.error = () => {};
const readline = require('node:readline');
const path = require('node:path');
const { createRequire } = require('node:module');
let qrEncoder;
try {
  // Resolve from the original server workspace, where qrcode is declared.
  const serverRequire = createRequire(path.resolve(path.dirname(process.argv[1]),
    '../../vendor/chisacode-remote/node_modules/@chisacode/server/package.json'));
  qrEncoder = serverRequire('qrcode');
} catch { /* Link pairing remains available when the optional QR encoder is missing. */ }
let qrURL = '', qrPNG = '', qrGeneration = 0;
function clearQR() { qrURL = ''; qrPNG = ''; ++qrGeneration; }
function usableURL(s) {
  if (!s.listening || !s.relayConnected) return '';
  const raw = s.urls?.[0]?.pairingUrl || '';
  try {
    if (raw.length > 8192) return '';
    const url = new URL(raw);
    if (!['https:', 'http:'].includes(url.protocol) || !url.hash.startsWith('#offer=')) return '';
    const offer = JSON.parse(Buffer.from(url.hash.slice(7), 'base64url').toString('utf8'));
    return offer.v === 2 && offer.authBootstrap?.expiresAtMs > Date.now() ? raw : '';
  } catch { return ''; }
}

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
  if (s.error) { clearQR(); emit({kind:'error',code:'operation'}); return; }
  const url = usableURL(s);
  if (url !== qrURL) {
    clearQR();
    qrURL = url;
    const generation = qrGeneration;
    if (url && qrEncoder) {
      Promise.resolve().then(() => qrEncoder.toBuffer(url, {type:'png', errorCorrectionLevel:'M', margin:4, scale:4}))
        .then(png => {
          if (stopping || starting || generation !== qrGeneration || qrURL !== url) return;
          const current = remote.snapshot();
          if (current.error || usableURL(current) !== url) { clearQR(); snapshot(); return; }
          if (!Buffer.isBuffer(png) || png.length > 96 * 1024) return;
          qrPNG = png.toString('base64');
          snapshot();
        }).catch(() => {});
    }
  }
  emit({kind:'snapshot', listening:!!s.listening, relayConnected:!!s.relayConnected,
    url, qrPNG: url ? qrPNG : ''});
}
let queue = Promise.resolve();
async function stop() {
  if (stopping) return;
  stopping = true;
  clearQR();
  clearInterval(timer);
  await remote.stopDaemon();
  emit({kind:'snapshot',listening:false,relayConnected:false,url:''});
  process.exit(0);
}
const timer = setInterval(() => { try { snapshot(); } catch { clearQR(); emit({kind:'error',code:'snapshot'}); } },1000);
remote.on('listening',snapshot);
const input=readline.createInterface({input:process.stdin});
input.on('line',line=>{
  if(line.length>128*1024)return;
  queue=queue.then(async()=>{
    const command=JSON.parse(line);
    if(command.action==='stop')return stop();
    if(stopping)return;
    if(command.action==='start') {clearQR();config=command.config||{};starting=true;try { await remote.sync(); } finally { starting=false; }}
    else if(command.action==='refresh' && remote.daemon) {clearQR();emit({kind:'snapshot',listening:false,relayConnected:false,url:''});config=command.config||config;starting=true;try {await remote.sync();} finally {starting=false;}}
    snapshot();
  }).catch(()=>emit({kind:'error',code:'operation'}));
});
input.on('close',()=>{queue=queue.then(stop).catch(()=>process.exit(1));});
process.on('SIGTERM',()=>{queue=queue.then(stop).catch(()=>process.exit(1));});
