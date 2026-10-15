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
let qrURL = '', qrPNG = '', qrGeneration = 0, offerBlocked = false;
function clearQR() { qrURL = ''; qrPNG = ''; ++qrGeneration; }
function usableURL(s) {
  if (offerBlocked || !s.listening || !s.relayConnected) return '';
  const raw = s.urls?.[0]?.pairingUrl || '';
  try {
    if (raw.length > 8192) return '';
    const url = new URL(raw);
    if (!['https:', 'http:'].includes(url.protocol) || !url.hash.startsWith('#offer=')) return '';
    const offer = JSON.parse(Buffer.from(url.hash.slice(7), 'base64url').toString('utf8'));
    return offer.v === 2 && offer.authBootstrap?.expiresAtMs > Date.now() ? raw : '';
  } catch { return ''; }
}

const originalGit=loadOriginalGit(__MYGO_GIT_SOURCES__);
const gitPending=new Map();let gitSequence=0;
const draftPending = new Map();
originalGit.git.setChangeRequestGenerator((input, root) => new Promise(resolve => {
 const id = String(++gitSequence);
 const timer = setTimeout(() => { draftPending.delete(id); resolve({error:'Generation timed out; retry or edit manually.'}); }, 65000);
 draftPending.set(id, result => { clearTimeout(timer); resolve(result); });
 const bounded = {commitSummary:input.commitSummary.slice(0,20000),diffSummary:input.diffSummary.slice(0,20000),diffPatch:input.diffPatch.slice(0,60000),fallbackTitle:input.fallbackTitle.slice(0,256),changeRequestTemplate:input.changeRequestTemplate.slice(0,8000)};
 emit({kind:'git-generate-draft', id, root, input:bounded});
}));
const gitGate=createGitGate({...originalGit,backend:originalGit.git,authorize:(root,action)=>new Promise(resolve=>{const id=String(++gitSequence);const timer=setTimeout(()=>{gitPending.delete(id);resolve(false);},2500);gitPending.set(id,allowed=>{clearTimeout(timer);resolve(allowed);});emit({kind:'git-authorize',id,root,action});})});
function revokeGit(){gitGate.revoke();for(const done of gitPending.values())done(false);gitPending.clear();for(const done of draftPending.values())done({error:'Mobile Git permission was revoked'});draftPending.clear();}
let remote, config = {}, stopping = false, starting = false;
try {
  installMobileDraftUI(process.argv[1], __MYGO_MOBILE_DRAFT_UI__);
  const { DshdRemote } = require(process.argv[1]);
  remote = new DshdRemote({
    getConfig: () => config,
    git: gitGate.git,
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
    url, qrPNG: url ? qrPNG : '', devices:publicPhoneDevices(s), devicesLoaded:Array.isArray(s.devices)});
}
async function deviceAction(command) {
  if (typeof command.id !== 'string' || !/^[0-9]{1,20}$/.test(command.id)) return;
  let applied = false, code = '';
  clearQR();
  starting = true;
  try {
    await managePhoneDevice(remote, command, {
      hideOffer() {
        offerBlocked = true;
        clearQR();
        emit({kind:'snapshot',listening:false,relayConnected:false,url:'',devices:publicPhoneDevices(remote.snapshot()),devicesLoaded:true});
      },
      revokeGit,
      applied() { applied = true; },
    });
    applied = true;
    if (command.operation === 'unbind') {
      // A workspace may disappear during reconnect. Keep Git fail-closed
      // without misreporting an already applied device revocation as failed.
      try { gitGate.configure(config); } catch { revokeGit(); }
    }
    if (command.operation !== 'rename') offerBlocked = false;
  } catch (error) {
    const known = ['invalid-action','runtime-offline','device-missing','invalid-name','rename-failed','unbind-failed'];
    code = known.includes(error?.message) ? error.message : 'operation';
  } finally {
    starting = false;
  }
  try { snapshot(); } catch { clearQR(); emit({kind:'error',code:'snapshot'}); }
  emit({kind:'device-action-result',id:command.id,operation:command.operation,code,applied});
}
let queue = Promise.resolve();
async function stop() {
  if (stopping) return;
  stopping = true;
  revokeGit();
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
  let immediate;try{immediate=JSON.parse(line);}catch{return;}
  if(immediate.action==='git-generation-result'){const done=draftPending.get(immediate.id);if(done){draftPending.delete(immediate.id);done(immediate.error?{error:immediate.error}:immediate.value);}return;}
  if(immediate.action==='git-authorize-result'){const done=gitPending.get(immediate.id);if(done){gitPending.delete(immediate.id);done(immediate.allowed===true);}return;}
  if(immediate.action==='git-revoke'||immediate.action==='stop'){revokeGit();if(immediate.action==='git-revoke')return;}
  queue=queue.then(async()=>{
    const command=JSON.parse(line);
    if(command.action==='stop')return stop();
    if(stopping)return;
    if(command.action==='device-action')return deviceAction(command);
    if(command.action==='start') {clearQR();offerBlocked=true;config=command.config||{};gitGate.configure(config);starting=true;try { await remote.sync();offerBlocked=false; } finally { starting=false; }}
    else if(command.action==='refresh' && remote.daemon) {clearQR();offerBlocked=true;emit({kind:'snapshot',listening:false,relayConnected:false,url:''});config=command.config||config;gitGate.configure(config);starting=true;try {await remote.sync();offerBlocked=false;} finally {starting=false;}}
    snapshot();
  }).catch(()=>emit({kind:'error',code:'operation'}));
});
input.on('close',()=>{revokeGit();queue=queue.then(stop).catch(()=>process.exit(1));});
process.on('SIGTERM',()=>{revokeGit();queue=queue.then(stop).catch(()=>process.exit(1));});
