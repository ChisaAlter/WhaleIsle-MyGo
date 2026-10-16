// Exact e9 DOM adapter patch in the isolated build copy. Independent from the
// provider record, so an already adapted provider can acquire this capability.
import fs from 'node:fs';
import path from 'node:path';
import crypto from 'node:crypto';
import {fileURLToPath} from 'node:url';
export const id='mygo-shortcut-held-keys-v1';
export const relativePath='packages/client/shortcuts/src/client/dom.ts';
export const baseSHA256='ea6f57ac2f7d98d3a62b4dca29e0584cbc982566860c7ec4d47a58a505e1cb59';
const hash=x=>crypto.createHash('sha256').update(x).digest('hex');
const before='const gesture = { code: event.code, control: event.ctrlKey';
const after=`// mygo-shortcut-held-keys-v1: only the product capability annotates
    // accepted physical pairs; ordinary webpages keep the original DOM grammar.
    const second = (window as Window & { whaleisleShortcutProvider?: { version: number } })
      .whaleisleShortcutProvider?.version === 1
      ? (event as KeyboardEvent & { whaleisleSecondCode?: string }).whaleisleSecondCode : undefined
    const gesture = { code: event.code, ...(second === undefined ? {} : { secondCode: second }), control: event.ctrlKey`;
export function patchSource(source){
 if(hash(source)!==baseSHA256||source.split(before).length!==2)throw new Error('Held-key adapter requires exact e9 DOM source');
 return source.replace(before,after);
}
export const patchSHA256=hash(patchSource.toString()+after);
export function applyShortcutChords(harness){
 const file=path.join(harness,relativePath),recordFile=path.join(harness,'mygo-shortcut-held-keys.json');
 const source=fs.readFileSync(file,'utf8');
 if(fs.existsSync(recordFile)){
  const record=JSON.parse(fs.readFileSync(recordFile,'utf8'));
  if(record.id===id&&record.patchSHA256===patchSHA256&&record.sourceSHA256===hash(source))return record;
  throw new Error('Held-key adapter adaptation record mismatch');
 }
 const output=patchSource(source);
 const record={id,sourceCommit:'e9f584be6d3b491ce777bb6823f30bc2bbc6f195',baseSHA256,patchSHA256,sourceSHA256:hash(output)};
 fs.writeFileSync(file,output);fs.writeFileSync(recordFile,JSON.stringify(record,null,2));return record;
}
if(process.argv[1]&&path.resolve(process.argv[1])===fileURLToPath(import.meta.url)){
 try{console.log(JSON.stringify(applyShortcutChords(process.argv[2])))}catch(e){console.error(e.message);process.exitCode=1}
}
