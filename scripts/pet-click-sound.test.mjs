import assert from 'node:assert/strict';
import {readFileSync} from 'node:fs';
import vm from 'node:vm';

// Real runtime + interaction handlers; only browser/audio devices are simulated.
const nodes = new Map(), timers = new Map();
let id = 0, blips = 0, closedAudio = 0, clickSound = true;
function node(name) {
 if (nodes.has(name)) return nodes.get(name);
 const handlers = new Map();
 const n = {hidden:false, style:{setProperty(){}}, handlers,
  addEventListener(type, fn){handlers.set(type, [...(handlers.get(type)||[]), fn]);},
  removeEventListener(type, fn){handlers.set(type, (handlers.get(type)||[]).filter(f=>f!==fn));},
  emit(type, extra={}){for(const fn of handlers.get(type)||[])fn({target:n, button:0, buttons:1, pointerId:1, screenX:10, screenY:10, clientX:10, clientY:10, preventDefault(){},stopPropagation(){}, ...extra});},
  setPointerCapture(){n.captured=true;}, hasPointerCapture(){return n.captured;},
  releasePointerCapture(){n.captured=false;n.emit('lostpointercapture');},
  getBoundingClientRect:()=>({left:0,top:0,right:100,bottom:100,width:100,height:100}),
  getContext:()=>({}), closest:()=>null, removeAttribute(){}, setAttribute(){}, replaceChildren(){}, remove(){},
 };
 nodes.set(name,n);return n;
}
const window=node('window'), document=node('document');
Object.assign(document,{hidden:false, body:{dataset:{behavior:'{"clickSound":true}'},appendChild(){}},
 documentElement:node('html'),getElementById:node,querySelector:()=>node('task'),createElement:()=>node('new')});
class AudioContext {
 state='running';currentTime=0;
 createOscillator(){return {type:'',frequency:{setValueAtTime(){},linearRampToValueAtTime(){}},connect(){},disconnect(){},start(){blips++;},stop(){}};}
 createGain(){return {gain:{setValueAtTime(){},exponentialRampToValueAtTime(){}},connect(){},disconnect(){}};}
 close(){closedAudio++;return Promise.resolve();}
}
window.AudioContext=AudioContext;
window.WhalePetCareAnimation={create:()=>({close(){},update(){}})};
const context=vm.createContext({window, document, location:{href:''}, performance:{now:()=>1000},
 fetch:url=>url==='task.json'?Promise.resolve({ok:true,json:async()=>({Behavior:{clickSound}})}):new Promise(()=>{}), Image:class {set src(_){}},
 setTimeout:(fn,ms)=>{timers.set(++id,{fn,ms});return id;},clearTimeout:i=>timers.delete(i),
 setInterval:()=>++id,clearInterval(){},requestAnimationFrame:()=>++id,cancelAnimationFrame(){}});
for(const file of ['pet-interaction.js','pet-runtime.js'])vm.runInContext(readFileSync(new URL('../internal/desktop/pet/assets/'+file,import.meta.url),'utf8'),context);
const pet=node('sprite'), fallback=node('fallback');
function tap(target=pet) {target.emit('pointerdown');target.emit('pointerup');}
tap();assert.equal(blips,1,'a completed character tap sounds once despite capture loss');
pet.emit('click');document.emit('click',{target:pet});pet.emit('dblclick');
assert.equal(blips,1,'synthetic click/dblclick must not replay a completed gesture');
tap();assert.equal(blips,2,'a second genuine tap retains upstream rapid-tap semantics');
pet.emit('pointerdown');pet.emit('pointermove',{screenX:50});pet.emit('pointerup');
assert.equal(blips,2,'drag release is silent');
pet.emit('pointerdown');pet.emit('pointercancel');pet.emit('pointerup');
assert.equal(blips,2,'cancelled gestures are silent');
pet.emit('pointerdown');window.emit('blur');pet.emit('pointerup');
assert.equal(blips,2,'blur cancellation is silent');
tap(fallback);assert.equal(blips,3,'fallback character uses the same tap path');
const link={closest:()=>({})};document.emit('click',{target:link});
assert.equal(blips,4,'existing link sound remains available');
clickSound=false;
await [...timers.values()].find(t=>t.ms===1500).fn();
tap();document.emit('click',{target:link});assert.equal(blips,4,'live disabled setting silences characters and links');
clickSound=true;
await [...timers.values()].find(t=>t.ms===1500).fn();
tap();assert.equal(blips,5,'live enabled setting restores character sound');
window.emit('pagehide');assert.equal(closedAudio,1,'the shared context is closed on unload');
console.log('PET32 character tap, rapid taps, drag/cancel/blur, fallback and audio cleanup passed');
