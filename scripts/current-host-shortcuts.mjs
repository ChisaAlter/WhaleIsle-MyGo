// MyGo-only, exact-source adaptation in an isolated build COPY.
// Provenance: ChisaAlter/WhaleIsle e9f584be6d3b491ce777bb6823f30bc2bbc6f195
// vendor/deepseek-harness/packages/client/shortcuts/src/client/index.ts
import fs from 'node:fs';
import {applyShortcutChords} from './current-host-shortcut-chords.mjs';
import {applyShortcutFrames} from './current-host-shortcut-frames.mjs';
import path from 'node:path';
import crypto from 'node:crypto';
import {fileURLToPath} from 'node:url';
export const id='mygo-shortcut-provider-v1';
export const baseSHA256='50f36616c5e4af3c4844e2737068e78152e9cc0bb61be0cef9e9d7101409a457';
export const relativePath='packages/client/shortcuts/src/client/index.ts';
const hash=x=>crypto.createHash('sha256').update(x).digest('hex');
const original="    const environment = detectEnvironment(document, navigator)";
const replacement=`    // mygo-shortcut-provider-v1: capability scoped exclusively to this service.
    const candidate = (window as Window & { whaleisleShortcutProvider?: {
      version: number; platform: 'macos' | 'windows' | 'linux'; shortcuts: DesktopShortcutsApi
    } }).whaleisleShortcutProvider
    const provider = candidate?.version === 1 && typeof candidate.shortcuts?.get === 'function'
      && typeof candidate.shortcuts?.edit === 'function' && typeof candidate.shortcuts?.subscribe === 'function'
      && typeof candidate.shortcuts?.recording === 'function' ? candidate : undefined
    const environment = provider ? { runtime: 'desktop' as const, platform: provider.platform }
      : detectEnvironment(document, navigator)`;
export function patchSource(source){
 if(hash(source)!==baseSHA256)throw new Error('Shortcut provider requires exact e9 source; source checkout is unchanged');
 return source.replace(original,replacement)
  .replace("if (environment.runtime === 'desktop' && keyboard === undefined)","if (provider === undefined && environment.runtime === 'desktop' && keyboard === undefined)")
  .replace("const web = this.runtime === 'web' ?", "const web = provider === undefined && this.runtime === 'web' ?")
  .replace('this.adapter = web ?? desktopShortcutStorage(window)','this.adapter = provider?.shortcuts ?? web ?? desktopShortcutStorage(window)');
}
export const patchSHA256=hash(patchSource.toString()+replacement);
export function applyShortcutProvider(harness){
 const file=path.join(harness,relativePath),recordFile=path.join(harness,'mygo-shortcut-provider.json');
 const source=fs.readFileSync(file,'utf8');
 const compose=record=>{
  applyShortcutChords(harness);
  const frames=applyShortcutFrames(harness);
  const combined={...record,providerSourceSHA256:record.providerSourceSHA256??record.sourceSHA256,sourceSHA256:frames.indexSHA256,frames};
  fs.writeFileSync(recordFile,JSON.stringify(combined,null,2));return combined;
 };
 if(fs.existsSync(recordFile)){
  const r=JSON.parse(fs.readFileSync(recordFile,'utf8'));
  if(r.id===id&&r.patchSHA256===patchSHA256&&r.sourceSHA256===hash(source)){return compose(r);}
  throw new Error('Shortcut provider adaptation record mismatch');
 }
 const output=patchSource(source);
 const record={id,sourceCommit:'e9f584be6d3b491ce777bb6823f30bc2bbc6f195',baseSHA256,patchSHA256,sourceSHA256:hash(output)};
 applyShortcutChords(harness);
 fs.writeFileSync(file,output);fs.writeFileSync(recordFile,JSON.stringify(record,null,2));return compose(record);
}
if(process.argv[1]&&path.resolve(process.argv[1])===fileURLToPath(import.meta.url)){
 try{console.log(JSON.stringify(applyShortcutProvider(process.argv[2])))}catch(e){console.error(e.message);process.exitCode=1}
}
