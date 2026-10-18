// Exact e9 native dispatcher, enabled only for verified top-product frame input.
// This changes build COPY types/startup wiring, never an iframe's source/sandbox.
import fs from 'node:fs';
import path from 'node:path';
import crypto from 'node:crypto';
export const frameGuardID='mygo-shortcut-frames-v1';
export const indexPath='packages/client/shortcuts/src/client/index.ts';
export const nativePath='packages/client/shortcuts/src/client/native.ts';
export const indexBaseSHA256='58d9fa5ab8e1e7a36d9721263ae34e1d1dcbbf00ff7ebbb8f80d1953c88f31db';
export const nativeBaseSHA256='0e70c5d0cc265d1adbd4d39c404aabe2fcc617c9208bd8f2f6d6e95d7a18ec60';
const hash=x=>crypto.createHash('sha256').update(x).digest('hex');
const subscription=`    // mygo-shortcut-frames-v1: only accepted native iframe gestures reach this subscriber.
    const guestKeyboard = provider?.guestKeyboard
    if (typeof guestKeyboard?.subscribe === 'function') ctx.effect(() => installNativeKeyboard(window,
      guestKeyboard, this.registry, () => this.config.getSnapshot(), () => { this.fixedInput({ type: 'reset' }) }),
      'shortcuts: verified embedded native input')
`;
export function patchFrameIndex(source){
 if(hash(source)!==indexBaseSHA256)throw Error('Frame input adaptation requires the exact e9 provider-adapted index.ts');
 return source.replace('version: number; platform:',"guestKeyboard?: Pick<DesktopKeyboardApi, 'subscribe'>; version: number; platform:")
  .replace('    this.syncDefinitions()',subscription+'    this.syncDefinitions()');
}
export function patchFrameNative(source){
 if(hash(source)!==nativeBaseSHA256)throw Error('Frame input adaptation requires exact e9 native.ts');
 return source.replace('keyboard: DesktopKeyboardApi, registry:',"keyboard: Pick<DesktopKeyboardApi, 'subscribe'> & Partial<Omit<DesktopKeyboardApi, 'subscribe'>>, registry:");
}
export const framePatchSHA256=hash(patchFrameIndex.toString()+patchFrameNative.toString()+subscription);
export function applyShortcutFrames(harness){
 const indexFile=path.join(harness,indexPath),nativeFile=path.join(harness,nativePath),recordFile=path.join(harness,'mygo-shortcut-frames.json');
 const index=fs.readFileSync(indexFile,'utf8'),native=fs.readFileSync(nativeFile,'utf8');
 if(fs.existsSync(recordFile)){
  const r=JSON.parse(fs.readFileSync(recordFile,'utf8'));
  if(r.id===frameGuardID&&r.patchSHA256===framePatchSHA256&&r.indexSHA256===hash(index)&&r.nativeSHA256===hash(native))return r;
  throw Error('Frame input adaptation provenance mismatch');
 }
 const indexOut=patchFrameIndex(index),nativeOut=patchFrameNative(native);
 const record={id:frameGuardID,sourceCommit:'e9f584be6d3b491ce777bb6823f30bc2bbc6f195',indexBaseSHA256,nativeBaseSHA256,patchSHA256:framePatchSHA256,indexSHA256:hash(indexOut),nativeSHA256:hash(nativeOut)};
 fs.writeFileSync(indexFile,indexOut);fs.writeFileSync(nativeFile,nativeOut);fs.writeFileSync(recordFile,JSON.stringify(record,null,2));return record;
}
