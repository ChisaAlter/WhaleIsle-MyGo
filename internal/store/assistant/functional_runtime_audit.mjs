// Read-only audit of this task's stopped, synthetic assistant profile. Imports
// only format validators; never starts Host, resolves providers or reads bodies
// into output. The frame scanner is extracted from the actual pinned backend.
import fs from 'node:fs/promises';
import path from 'node:path';
import {pathToFileURL} from 'node:url';
import {zstdDecompressSync} from 'node:zlib';
const [stage,ownedRoot]=process.argv.slice(2);
if(!stage||!ownedRoot||!path.basename(ownedRoot).startsWith('assistant-functional-check-'))throw Error('Explicit owned assistant run required');
const harness=path.join(stage,'vendor/deepseek-harness');
const backend=await fs.readFile(path.join(harness,'packages/session/session-persistence-jsonl/lib/index.js'),'utf8');
const begin=backend.indexOf('function scanZstdFrames('),end=backend.indexOf('\n}',begin)+2;
const magic=backend.match(/const ZSTD_MAGIC = (\d+);/);
if(begin<0||end<=begin||!magic)throw Error('Pinned backend frame scanner not found');
const scan=Function('const ZSTD_MAGIC='+magic[1]+';'+backend.slice(begin,end)+';return scanZstdFrames;')();
const {sessionFormatCatalog:catalog}=await import(pathToFileURL(path.join(harness,'packages/session/session-format-catalog/lib/index.js')));
const files=[];
async function walk(dir){for(const e of await fs.readdir(dir,{withFileTypes:true})){if(e.isDirectory())await walk(path.join(dir,e.name));else if(e.isFile()&&e.name==='session.v4.jsonl.zstd')files.push(path.join(dir,e.name));else if(e.isSymbolicLink())throw Error('Unexpected link in owned Session corpus');}}
await walk(path.join(ownedRoot,'user-data/dsh-home/sessions'));
const types={};let eventCount=0,frameCount=0;
for(const filename of files){
 const bytes=await fs.readFile(filename),{frames,tornStart}=scan(bytes);
 if(tornStart!==undefined||frames.length===0)throw Error('Owned Session has an incomplete frame');
 frameCount+=frames.length;
 const plaintext=Buffer.concat(frames.map(({start,end})=>zstdDecompressSync(bytes.subarray(start,end)))).toString('utf8');
 const records=plaintext.trim().split('\n').map(line=>JSON.parse(line));
 if(records[0].type!=='session')throw Error('Owned Session header missing');
 const restore=catalog.createRestore(records[0],{recovery:'strict',validation:'current'});
 for(const record of records.slice(1))restore.decodeRow(record);
 const artifact=restore.finish();
 for(const event of artifact.events){types[event.type]=(types[event.type]??0)+1;eventCount++;}
}
// Accept only the inert metadata actually required by these empty Sessions;
// unknown event names cannot silently evade the no-execution assertion.
const metadataTypes=new Set(['permission/preset','sandbox/mode','approval/policy','session/presentation','session/end-seed']);
const executionTypes=Object.fromEntries(Object.entries(types).filter(([type])=>!metadataTypes.has(type)));
const result={schemaVersion:1,scope:'all five stopped owned synthetic Session snapshots, complete zstd frames + actual strict current-format decoder',sessions:files.length,frames:frameCount,events:eventCount,eventTypes:types,executionTypes,
 noPersistedPromptTurnToolOrModelRequestEvidence:Object.keys(executionTypes).length===0,
 networkPacketCapture:false,note:'RPC calls were loopback-only; this audit does not claim capture of all process-level network traffic.'};
if(files.length!==5||eventCount===0||Object.keys(executionTypes).length)throw Error('Owned Session execution audit did not meet the no-execution condition');
console.log(JSON.stringify(result,null,2));
