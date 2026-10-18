// One offline fixture: no ORT implementation, model, Host or user assets run.
import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import vm from 'node:vm';
const scope = { Float32Array, Uint8ClampedArray, Set, Math };
vm.runInNewContext(readFileSync(new URL('../internal/desktop/pet/assets/tha4.js', import.meta.url), 'utf8'), scope);
const { createEngine, imageCHW } = scope.WhaleTHA4;
const n=512*512, rgba=new Uint8ClampedArray(n*4); rgba.set([255,128,0,128]);
const image=imageCHW(rgba);
assert.equal(image.length,4*n); assert.equal(image[2*n],-1); assert.ok(Math.abs(image[0]-(128/255*2-1))<1e-6);
let released=0,disposed=0, mode='ok', resolveRun;
class Tensor { constructor(type,data,dims){Object.assign(this,{type,data,dims});} dispose(){disposed++;} }
const raw=new Float32Array(n*4); raw[0]=200;raw[n]=100;raw[2*n]=50;raw[3*n]=255;
const out={dims:[1,4,512,512],data:raw,dispose(){released++;}};
const session={async run(feeds){assert.deepEqual(Array.from(feeds.image.dims),[1,4,512,512]);assert.deepEqual(Array.from(feeds.pose.dims),[1,45]); if(mode==='wait')await new Promise(r=>resolveRun=r); if(mode==='bad')return {rgba_f:{...out,dims:[1,3,1,1]},alias:out};return {rgba_f:out,alias:out};},async release(){released+=10;}};
const engine=createEngine({Tensor},session,image);
const pixels=await engine.frame(new Float32Array(45));assert.deepEqual(Array.from(pixels.slice(0,4)),[200,100,50,255]);assert.equal(released,1);
mode='bad';await assert.rejects(engine.frame(new Float32Array(45)),/shape/);assert.equal(released,3);
mode='wait';const pending=engine.frame(new Float32Array(45));assert.equal(await engine.frame(new Float32Array(45)),null);const closing=engine.close();resolveRun();assert.equal(await pending,null);await closing;assert.equal(released,14);assert.equal(disposed,2);await engine.close();assert.equal(released,14);
console.log('THA4 offline protocol/lifecycle fixture passed');
