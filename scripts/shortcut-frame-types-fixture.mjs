// Compile the actual generated parameter type against both supported callers.
import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import {spawnSync} from 'node:child_process';
import assert from 'node:assert/strict';
import {patchFrameNative} from './current-host-shortcut-frames.mjs';
const native=patchFrameNative(fs.readFileSync(process.argv[2],'utf8'));
const parameter=native.match(/keyboard: (.*), registry: ShortcutRegistry/)[1];
const tmp=fs.mkdtempSync(path.join(os.tmpdir(),'mygo-frame-types-'));
try{
 const source=`interface DesktopKeyboardApi {subscribe(listener:(input:unknown)=>void):()=>void;closeWindow(revision:string):Promise<void>}
 declare function consume(keyboard: ${parameter}):void;
 consume({subscribe(){return ()=>{}}});
 consume({subscribe(){return ()=>{}},async closeWindow(revision){void revision}});
 // @ts-expect-error subscribe is required
 consume({async closeWindow(){}});
 // @ts-expect-error unrelated capabilities remain rejected
 consume({subscribe(){return ()=>{}}, arbitraryHostRPC(){}});
 `;
 const file=path.join(tmp,'fixture.ts');fs.writeFileSync(file,source);
 const result=spawnSync(process.execPath,[path.resolve(process.argv[3]),'--strict','--noEmit','--lib','es2022',file],{cwd:tmp,encoding:'utf8'});
 assert.equal(result.status,0,result.stdout+result.stderr);
 console.log('frame subscriber and original complete keyboard API type compatibility PASS');
}finally{fs.rmSync(tmp,{recursive:true,force:true})}
