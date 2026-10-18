// Child-only native load probe. All home/config paths are supplied by the parent.
import fs from 'node:fs';import path from 'node:path';import net from 'node:net';import tls from 'node:tls';import dgram from 'node:dgram';import cp from 'node:child_process';import {createRequire,syncBuiltinESMExports} from 'node:module';
if(process.platform!=='win32'||process.arch!=='x64'||process.version!=='v24.19.0'||process.versions.modules!=='137')throw Error('Expected actual Windows x64 Node24.19.0 ABI137');
const deny=()=>{throw Error('Network/process operation forbidden in offline native verification');};
net.Socket.prototype.connect=deny;net.Server.prototype.listen=deny;tls.connect=deny;tls.createServer=deny;dgram.createSocket=deny;globalThis.fetch=deny;for(const key of ['spawn','spawnSync','exec','execSync','execFile','execFileSync','fork'])cp[key]=deny;syncBuiltinESMExports();
const root=path.resolve(process.argv[2]);const mobile=path.join(root,'runtime/vendor/dshd-remote/vendor/chisacode-remote');const require=createRequire(path.join(mobile,'package.json'));
const Database=require('better-sqlite3'),db=new Database(':memory:');try{if(db.prepare('select 1 AS n').get().n!==1)throw Error('SQLite probe failed');}finally{db.close();}require('node-pty');
// Load every actual target .node exactly once, without calling any model/device API.
let loaded=0;const seen=new Set();function walk(dir){for(const e of fs.readdirSync(dir,{withFileTypes:true})){const p=path.join(dir,e.name);if(e.isSymbolicLink())continue;if(e.isDirectory())walk(p);else if(e.name.endsWith('.node')){const real=fs.realpathSync(p);if(seen.has(real))continue;seen.add(real);try{require(real);}catch(error){throw Error(`Native load failed at ${path.relative(root,p)}: ${error.message}`);}loaded++;}}}
walk(path.join(root,'runtime'));if(!loaded)throw Error('No native bindings found');
console.log(JSON.stringify({platform:process.platform,arch:process.arch,node:process.version,abi:process.versions.modules,nativeBindingsLoaded:loaded,sqlite:true,mobilePty:true,networkAndSubprocessAPIsBlocked:true}));process.exit(0);
