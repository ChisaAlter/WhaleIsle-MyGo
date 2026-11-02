import assert from 'node:assert/strict';
import {readFileSync} from 'node:fs';
import vm from 'node:vm';
import test from 'node:test';
const asset = name => readFileSync(new URL('../assets/'+name,import.meta.url),'utf8');
const n=512*512;
function thaScope(extra={}) {
  const scope={Float32Array,Uint8ClampedArray,Set,Math,...extra};
  vm.runInNewContext(asset('tha4.js'),scope);return scope;
}
function tensors() {
  const trace=[];
  class Tensor {
    constructor(type,data,dims){Object.assign(this,{type,data,dims});trace.push(['cpu',dims]);}
    dispose(){trace.push(['dispose-input',this.location||'cpu']);}
    static fromGpuBuffer(buffer,options){trace.push(['gpu',options.dims]);return {location:'gpu-buffer',dims:options.dims,buffer,dispose(){trace.push(['dispose-input','gpu']);}};}
  }
  const device={createBuffer(options){const buffer={...options,destroy(){trace.push(['destroy-buffer',this.size]);}};trace.push(['buffer',options]);return buffer;},queue:{writeBuffer(buffer,offset,data){trace.push(['write',buffer.size,data.length]);}}};
  const raw=new Float32Array(n*4);raw[0]=200;raw[n]=100;raw[2*n]=50;raw.fill(255,n*3);
  const output={dims:[1,4,512,512],location:'gpu-buffer',async getData(){trace.push(['readback']);return raw;},dispose(){trace.push(['dispose-output']);}};
  const session={async run(feeds){trace.push(['run',feeds.image.location,feeds.pose.location]);return {rgba_f:output,alias:output};},async release(){trace.push(['release']);}};
  return {trace,Tensor,device,raw,output,session};
}
test('THA4 external GPU inputs are persistent and every output is disposed once',async()=>{
  const f=tensors(),s=thaScope({GPUBufferUsage:{STORAGE:128,COPY_DST:8}});
  const rgba=new Uint8ClampedArray(n*4);rgba.set([255,128,0,128]);
  const chw=s.WhaleTHA4.imageCHW(rgba);assert.equal(chw[2*n],-1);assert.ok(Math.abs(chw[0]-(128/255*2-1))<1e-6);
  const engine=s.WhaleTHA4.createEngine({Tensor:f.Tensor},f.session,chw,{provider:'webgpu',device:f.device,capture:true});
  assert.equal(engine.gpuInputs,true);
  assert.deepEqual(Array.from((await engine.frame(new Float32Array(45))).slice(0,4)),[200,100,50,255]);
  await engine.frame(new Float32Array(45));
  assert.equal(f.trace.filter(x=>x[0]==='gpu').length,2);
  assert.equal(f.trace.filter(x=>x[0]==='write'&&x[1]===n*16).length,1,'constant image uploaded only on construction');
  assert.equal(f.trace.filter(x=>x[0]==='write'&&x[1]===45*4).length,3,'pose rewritten each frame');
  assert.equal(f.trace.filter(x=>x[0]==='dispose-output').length,2);
  await engine.close();await engine.close();
  assert.equal(f.trace.filter(x=>x[0]==='release').length,1);assert.equal(f.trace.filter(x=>x[0]==='destroy-buffer').length,2);
});
test('WebNN cannot use stale WebGPU device; failed captured first run retries uncaptured then WASM',async()=>{
  const f=tensors(),s=thaScope({GPUBufferUsage:{STORAGE:128,COPY_DST:8}}),image=new Float32Array(n*4);
  const cpu=s.WhaleTHA4.createEngine({Tensor:f.Tensor},f.session,image,{provider:'webnn',device:f.device});
  assert.equal(cpu.gpuInputs,false);await cpu.close();assert.equal(f.trace.filter(x=>x[0]==='gpu').length,0);
  const attempts=[],releases=[];
  const ort={Tensor:f.Tensor,env:{webgpu:{device:f.device}},InferenceSession:{async create(url,options){
    const ep=options.executionProviders[0];attempts.push(options);
    if(ep==='webnn')throw Error('unsupported');
    return {async run(){if(ep==='webgpu')throw Error('GPU run not supported');return {rgba_f:{dims:[1,4,512,512],data:f.raw,dispose(){}}};},async release(){releases.push(ep);}};
  }}};
  const opened=await s.WhaleTHA4.open(ort,'model.onnx',image);
  assert.equal(opened.provider,'wasm');assert.equal(opened.capture,false);assert.equal(opened.engine.gpuInputs,false);
  assert.deepEqual(attempts.map(x=>[x.executionProviders[0],!!x.enableGraphCapture]),[['webnn',false],['webgpu',true],['webgpu',false],['wasm',false]]);
  assert.equal(attempts[1].preferredOutputLocation,'gpu-buffer');assert.ok(!('preferredOutputLocation' in attempts[0]));
  assert.deepEqual(releases,['webgpu','webgpu']);await opened.engine.close();assert.deepEqual(releases,['webgpu','webgpu','wasm']);
});
test('THA4 close waits for an in-flight run, rejects overlap and retires late output',async()=>{
  const f=tensors(),s=thaScope();let resolve;
  f.session.run=async()=>{await new Promise(r=>resolve=r);return {rgba_f:f.output,alias:f.output};};
  const engine=s.WhaleTHA4.createEngine({Tensor:f.Tensor},f.session,new Float32Array(n*4));
  const pending=engine.frame(new Float32Array(45));assert.equal(await engine.frame(new Float32Array(45)),null);
  const closing=engine.close();resolve();assert.equal(await pending,null);await closing;
  assert.equal(f.trace.filter(x=>x[0]==='dispose-output').length,1);assert.equal(f.trace.filter(x=>x[0]==='release').length,1);
});
function surfaces() {
  const trace=[],canvases=[];
  const document={createElement(){const c={width:0,height:0};c.ctx=new Proxy({canvas:c},{get(t,k){if(k in t)return t[k];return (...a)=>trace.push([canvases.indexOf(c),k,...a]);},set(t,k,v){trace.push([canvases.indexOf(c),'set',k,v]);t[k]=v;return true;}});c.getContext=()=>c.ctx;canvases.push(c);return c;}};
  const scope={document};vm.runInNewContext(asset('pet-super-resolution.js'),scope);
  return {scope,trace,canvases,document};
}
test('Anime4K paints twice-bleeded RGB and restores alpha; failure uses raw crop without losing THA4',()=>{
  const f=surfaces();let fail=false,detached=0;
  class Upscaler {static isSupported(){return true;}attachSource(source,out){out.width=source.width*2;out.height=source.height*2;}upscale(){f.trace.push(['upscale']);if(fail)throw Error('context lost');}detachSource(){detached++;}}
  const sr=f.scope.WhalePetSuperResolution.create({ImageUpscaler:Upscaler,ANIME4KJS_SIMPLE_M_2X:'original-preset'});
  const target=f.document.createElement('canvas');target.width=780;target.height=984;
  sr.paint({source:true},target.ctx);assert.equal(sr.ready,true);
  assert.equal(f.trace.filter(x=>x[1]==='set'&&x[2]==='globalCompositeOperation'&&x[3]==='destination-over').length,2);
  assert.equal(f.trace.filter(x=>x[1]==='set'&&x[2]==='globalCompositeOperation'&&x[3]==='destination-in').length,1);
  assert.equal(f.trace.filter(x=>x[0]==='upscale').length,1);
  fail=true;sr.paint({source:true},target.ctx);assert.equal(sr.ready,false);
  const last=f.trace.filter(x=>x[1]==='drawImage').at(-1);assert.deepEqual(last.slice(3),[60,20,390,492,0,0,780,984]);
  sr.close();sr.close();assert.equal(detached,1);
});
test('floating surfaces fit the active work area even when overlay contains monitor gaps',()=>{
  const scope={window:{}};vm.runInNewContext(asset('pet-layout.js'),scope);
  const anchor=scope.window.WhalePetLayout.anchor;
  const area={X:1920,Y:200,Width:1440,Height:900};
  assert.deepEqual({...anchor({X:3000,Y:700,Width:240,Height:260},{width:248,height:360},area)},{x:2996,y:332});
  assert.deepEqual({...anchor({X:3300,Y:220,Width:240,Height:260},{width:248,height:360},area)},{x:3108,y:488});
  assert.deepEqual({...anchor({X:1900,Y:220,Width:384,Height:416},{width:248,height:360},area)},{x:1968,y:644});
});
