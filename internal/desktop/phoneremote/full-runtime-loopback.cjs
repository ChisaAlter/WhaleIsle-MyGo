'use strict';
// Opt-in acceptance harness. Original install and verified core runtime are
// read-only inputs. This never creates a replacement relay protocol or disables auth.
const fs=require('node:fs'),path=require('node:path'),crypto=require('node:crypto'),readline=require('node:readline'),cp=require('node:child_process');
const {pathToFileURL}=require('node:url');
const [root,resources,toolRoot,coreRoot,mode]=process.argv.slice(2);
const emit=x=>process.stdout.write(JSON.stringify(x)+'\n');
const hash=b=>crypto.createHash('sha256').update(b).digest('hex');
const inside=(base,file)=>{const r=path.relative(base,file);return r===''||(!r.startsWith('..'+path.sep)&&r!=='..'&&!path.isAbsolute(r));};
const spawned=[];
const originalSpawn=cp.spawn;
cp.spawn=function(...args){const child=originalSpawn.apply(this,args);spawned.push({child,file:path.resolve(String(args[0])),pid:child.pid});return child;};
let mf,channel,socket,closing=false;
async function cleanup(){
 if(closing)return;closing=true;
 if(channel)channel.close(1000,'owned acceptance complete');
 if(socket)socket.terminate();
 if(mf)await mf.dispose();
 for(const {child} of spawned){if(child.exitCode===null&&child.signalCode===null)child.kill();}
 await Promise.all(spawned.map(({child})=>child.exitCode!==null||child.signalCode!==null?null:new Promise(resolve=>{child.once('exit',resolve);setTimeout(resolve,2000).unref();})));
 emit({kind:'closed',children:spawned.map(({child,file,pid})=>({file,pid,exitCode:child.exitCode,signalCode:child.signalCode,exited:child.exitCode!==null||child.signalCode!==null}))});
}
function copyGraph(source,target,label){
 const base=fs.realpathSync(source),pending=[],inventory=[];
 function copy(from,to){
  const st=fs.lstatSync(from),rel=path.relative(base,from);
  if(st.isSymbolicLink()){
   const real=fs.realpathSync(from);
   if(!inside(base,real))throw Error(label+': source dependency link escapes its complete root: '+rel);
   const targetPath=path.join(target,path.relative(base,real));
   if(fs.statSync(real).isDirectory()){pending.push({to,targetPath});inventory.push({path:rel,link:path.relative(base,real)});}
   else{fs.copyFileSync(real,to);inventory.push({path:rel,sha256:hash(fs.readFileSync(to))});}
  }else if(st.isDirectory()){
   fs.mkdirSync(to,{recursive:true});for(const name of fs.readdirSync(from))copy(path.join(from,name),path.join(to,name));
  }else if(st.isFile()){
   fs.copyFileSync(from,to);inventory.push({path:rel,sha256:hash(fs.readFileSync(to))});
   if(inventory.length%1000===0)emit({kind:'stage',section:label,entries:inventory.length});
  }else throw Error(label+': unsupported dependency entry');
 }
 copy(base,target);
 for(const {to,targetPath} of pending){if(!inside(target,targetPath))throw Error('target link escape');fs.symlinkSync(targetPath,to,'junction');}
 return {label,entries:inventory.length,links:pending.length,manifestSHA256:hash(JSON.stringify(inventory.sort((a,b)=>a.path.localeCompare(b.path))))};
}
function verifyGraph(base,expected){
 const inventory=[];
 function visit(file){const st=fs.lstatSync(file),rel=path.relative(base,file);if(st.isSymbolicLink()){const target=fs.realpathSync(file);if(!inside(base,target))throw Error('staged dependency link escape');inventory.push({path:rel,link:path.relative(base,target)});}else if(st.isDirectory()){for(const name of fs.readdirSync(file))visit(path.join(file,name));}else{inventory.push({path:rel,sha256:hash(fs.readFileSync(file))});}}
 visit(base);if(hash(JSON.stringify(inventory.sort((a,b)=>a.path.localeCompare(b.path))))!==expected.manifestSHA256)throw Error('staged complete dependency graph changed: '+expected.label);
}
async function start(){
 if(mode==='serve'){
  const manifest=JSON.parse(fs.readFileSync(path.join(root,'stage-manifest.json'),'utf8'));
  if(manifest.archiveSHA256!=='b1a6783d00c9db943a52bfdb6d698d346105badc868c22c034538da3f3901b45')throw Error('unknown staged desktop');
  const component=path.join(root,'vendor/dshd-remote');
  for(const {file,sha256} of manifest.assets)if(hash(fs.readFileSync(path.join(component,file)))!==sha256)throw Error('staged source changed');
  if(hash(fs.readFileSync(path.join(component,'node.exe')))!==manifest.nodeSHA256)throw Error('staged Node changed');
  verifyGraph(path.join(component,'vendor/chisacode-remote'),manifest.dependencies);
  verifyGraph(path.join(component,'vendor/deepseek-harness'),manifest.harness);
  await serve(manifest,component);return;
 }
 if(mode!=='stage-only')throw Error('explicit stage-only or serve required');
 if(!path.isAbsolute(root)||fs.existsSync(root))throw Error('fresh absolute component root required');
 fs.mkdirSync(root,{recursive:true});
 const archive=path.join(resources,'app.asar');
 if(hash(fs.readFileSync(archive))!=='b1a6783d00c9db943a52bfdb6d698d346105badc868c22c034538da3f3901b45')throw Error('unknown desktop archive');
 const asar=require(path.join(toolRoot,'../../node_modules/@electron/asar'));
 const read=p=>asar.extractFile(archive,path.normalize(p).replace(/^[/\\]+/,''));
 const vendor=path.join(root,'vendor'),component=path.join(vendor,'dshd-remote');fs.mkdirSync(component,{recursive:true});
 const files=['src/main/dshd-remote.js','src/main/dshd-daemon-runner.mjs','src/main/dshd-daemon-hooks.mjs','src/main/mobile-web-server.js','src/main/mobile-web.js','src/main/dshd-git-tunnel.js','src/main/dshd-git-dispatch.js','src/shared/official-deepseek-env.js','src/shared/lan.js','src/shared/offer.js','src/shared/dshd-host-tunnel.js','src/shared/dshd-mux-sse.js'];
 files.push(...asar.listPackage(archive).filter(p=>p.startsWith('\\mobile\\web\\')&&path.extname(p)).map(p=>p.replaceAll('\\','/').slice(1)));
 const assets=[];
 for(const file of files){const b=read(file),dest=path.join(component,file);fs.mkdirSync(path.dirname(dest),{recursive:true});fs.writeFileSync(dest,b);assets.push({file,sha256:hash(b)});}
 const node=path.join(resources,'runtime/primary-runtime/dependencies/node/bin/node.exe'),nodeSHA=hash(fs.readFileSync(node));
 if(nodeSHA!=='ba4e6d110e8c1592a1ecd390f6b05f3da124b13871a5be62b341a07a853c6c32')throw Error('unknown bundled Node');
 fs.copyFileSync(node,path.join(component,'node.exe'));
 emit({kind:'stage',section:'original-source-and-mobile',files:assets.length});
 const dependencies=copyGraph(path.join(resources,'vendor/dshd-remote'),path.join(component,'vendor/chisacode-remote'),'remote-dependencies');
 emit({kind:'stage',section:dependencies.label,entries:dependencies.entries});
 const harness=copyGraph(path.join(coreRoot,'vendor/deepseek-harness'),path.join(component,'vendor/deepseek-harness'),'private-verified-harness');
 emit({kind:'stage',section:harness.label,entries:harness.entries,links:harness.links});
 const manifest={archiveSHA256:hash(fs.readFileSync(archive)),nodeSHA256:nodeSHA,assets,dependencies,harness};
 fs.writeFileSync(path.join(root,'stage-manifest.json'),JSON.stringify(manifest,null,2)+'\n');
 emit({kind:'staged',root,manifest});process.exit(0);
}
async function serve(manifest,component){
 const entry=path.join(component,'vendor/chisacode-remote/node_modules/@chisacode/relay/dist/cloudflare-adapter.js');
 const esbuild=require(path.join(toolRoot,'node_modules/wrangler/node_modules/esbuild'));
 const worker=path.join(root,'relay-worker.mjs');
 await esbuild.build({entryPoints:[entry],outfile:worker,bundle:true,format:'esm',platform:'browser',target:'es2022',logLevel:'silent'});
 const {Miniflare,Log,LogLevel}=require(path.join(toolRoot,'node_modules/miniflare'));
 mf=new Miniflare({scriptPath:worker,modules:true,compatibilityDate:'2024-12-01',host:'127.0.0.1',port:0,durableObjects:{RELAY:{className:'RelayDurableObject',useSQLite:true}},durableObjectsPersist:path.join(process.env.HOME,'relay-db'),log:new Log(LogLevel.NONE)});
 const url=await mf.ready;
 emit({kind:'ready',endpoint:url.host,vendor:path.join(root,'vendor'),component,manifest,workerSHA256:hash(fs.readFileSync(worker)),children:spawned.map(({file,pid})=>({file,pid}))});
}
async function handshake(offerURL){
 const offer=JSON.parse(Buffer.from(new URL(offerURL).hash.slice(7),'base64url').toString('utf8'));
 if(!offer.relay.endpoint.startsWith('127.0.0.1:'))throw Error('offer endpoint is not own loopback');
 const {createRequire}=require('node:module');const localRequire=createRequire(path.join(root,'vendor/dshd-remote/vendor/chisacode-remote/node_modules/@chisacode/relay/package.json'));
 const {WebSocket}=localRequire('ws');
 const relayPath=localRequire.resolve('@chisacode/relay/e2ee');
 const {createClientChannel}=await import(pathToFileURL(relayPath).href);
 const {buildRelayWebSocketUrl}=await import(pathToFileURL(path.join(root,'vendor/dshd-remote/vendor/chisacode-remote/node_modules/@chisacode/protocol/dist/daemon-endpoints.js')).href);
 const wsURL=buildRelayWebSocketUrl({endpoint:offer.relay.endpoint,useTls:false,serverId:offer.serverId,role:'client',connectionId:'owned-acceptance-01'});
 socket=new WebSocket(wsURL,{handshakeTimeout:4000});
 await new Promise((resolve,reject)=>{socket.once('open',resolve);socket.once('error',reject);});
 const transport={send:data=>socket.send(data),close:(code,reason)=>socket.close(code,reason)};
 socket.on('message',data=>transport.onmessage?.(data.toString()));socket.on('close',(code,reason)=>transport.onclose?.(code,reason.toString()));socket.on('error',e=>transport.onerror?.(e));
 let resolveOpen,rejectOpen;const opened=new Promise((resolve,reject)=>{resolveOpen=resolve;rejectOpen=reject;});
 channel=await createClientChannel(transport,offer.daemonPublicKeyB64,{onopen:resolveOpen,onerror:rejectOpen});
 const timeout=setTimeout(()=>rejectOpen(Error('own relay E2EE handshake timeout')),6000);
 try{await opened;}finally{clearTimeout(timeout);}
 const security=channel.getSecurityContext();
 if(!security?.clientPublicKeyB64||!security?.authChallenge)throw Error('E2EE channel binding missing');
 // Never send application hello: provider/session initialization is outside this acceptance.
 channel.close(1000,'handshake-only acceptance');
 emit({kind:'handshake',e2ee:true,channelBoundChallenge:true,applicationHelloSent:false,pairingTokenConsumed:false});
}
const input=readline.createInterface({input:process.stdin});let queue=Promise.resolve();
input.on('line',line=>{queue=queue.then(async()=>{const message=JSON.parse(line);if(message.action==='handshake')await handshake(message.offer);if(message.action==='stop'){await cleanup();process.exit(0);}}).catch(async error=>{emit({kind:'error',message:error.message});await cleanup();process.exit(1);});});
input.on('close',()=>{queue=queue.then(cleanup).finally(()=>process.exit(0));});
start().catch(async error=>{emit({kind:'error',message:error.message});await cleanup();process.exit(1);});
