import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import vm from 'node:vm';
import assert from 'node:assert/strict';
import {applyShortcutProvider} from './current-host-shortcuts.mjs';
import {patchFrameNative,nativePath,indexPath} from './current-host-shortcut-frames.mjs';
const harness=process.argv[2],tmp=fs.mkdtempSync(path.join(os.tmpdir(),'mygo-frames-'));
try{
 for(const rel of [indexPath,nativePath,'packages/client/shortcuts/src/client/dom.ts']){const dest=path.join(tmp,rel);fs.mkdirSync(path.dirname(dest),{recursive:true});fs.copyFileSync(path.join(harness,rel),dest)}
 const first=applyShortcutProvider(tmp);assert.deepEqual(applyShortcutProvider(tmp),first);
 assert(fs.readFileSync(path.join(tmp,indexPath),'utf8').includes('verified embedded native input'));
 assert.throws(()=>patchFrameNative(fs.readFileSync(path.join(harness,nativePath),'utf8')+'\n'));
}finally{fs.rmSync(tmp,{recursive:true,force:true})}
const handlers=new Map();const on=(name,fn)=>{if(!handlers.has(name))handlers.set(name,new Set());handlers.get(name).add(fn)},off=(name,fn)=>handlers.get(name)?.delete(fn);
const win={addEventListener:on,removeEventListener:off,dispatchEvent(event){for(const fn of handlers.get(event.type)||[])fn(event)},top:null};win.top=win;
class Frame {constructor(name){this.name=name;this.isConnected=true}matches(selector){return selector.includes('iframe[data-html-preview]')}closest(selector){return selector.includes('iframe')?this:null}getAttribute(name){return name==='name'?this.name:null}get contentWindow(){throw Error('must not access guest document')}}
const frame=new Frame('dsh-sidebar-html-fixture');const doc={activeElement:frame,documentElement:{},hasFocus:()=>true,querySelector:()=>null,addEventListener:on,removeEventListener:off};win.document=doc;
let socket,dispatches=[];class Socket{static OPEN=1;constructor(){socket=this;this.readyState=1;this.events={};this.sent=[]}addEventListener(n,f){this.events[n]=f}send(data){this.sent.push(JSON.parse(data))}close(){}}
const snapshot={status:'ready',revision:'config-r'};const row={binding:{code:'KeyJ',modifiers:['control','alt']},issue:null,conflicts:[]};
const owner={catalog:{getSnapshot:()=>[row],subscribe:()=>()=>{}},config:{getSnapshot:()=>snapshot,subscribe:()=>()=>{}},recording:async()=>{}};
const context={window:win,document:doc,location:{origin:'http://127.0.0.1:23',href:'http://127.0.0.1:23/'},URL,WebSocket:Socket,HTMLIFrameElement:Frame,CustomEvent:class{constructor(type,init={}){this.type=type;this.detail=init.detail}},MutationObserver:class{observe(){}disconnect(){}},foregroundModalSurfaces:()=>[],setInterval:()=>1,clearInterval(){},setTimeout,clearTimeout,KeyboardEvent:class{constructor(type,init){this.type=type;Object.assign(this,init)}},fetch:async()=>{throw Error('no preference requests expected')}};
vm.createContext(context);
let provider=fs.readFileSync(new URL('../internal/desktop/pluginslots/shortcut_provider.js',import.meta.url),'utf8').replaceAll('__MYGO_ORIGIN__','"http://127.0.0.1:23"').replaceAll('__MYGO_PLATFORM__','"windows"');vm.runInContext(provider,context);
// Run the actual original native dispatcher, removing only its TypeScript annotations/imports.
let native=patchFrameNative(fs.readFileSync(path.join(harness,nativePath),'utf8')).replace(/^import .*\n/gm,'').replace(/export function installNativeKeyboard\([\s\S]*?\): \(\) => void \{/,'function installNativeKeyboard(window, keyboard, registry, snapshot, reset) {').replaceAll(' as const','');
vm.runInContext(native,context);
const unsubscribe=context.installNativeKeyboard(win,win.whaleisleShortcutProvider.guestKeyboard,{dispatch:(input,scope)=>dispatches.push({input,scope}),invoke:()=>{throw Error('menu not allowed')}},()=>snapshot);
context.ctx={shortcuts:owner,effect:fn=>fn()};vm.runInContext(fs.readFileSync(new URL('../internal/desktop/pluginslots/keyboard.js',import.meta.url),'utf8'),context);vm.runInContext('whaleBindProductKeyboard(ctx)',context);socket.events.open();
const input=()=>({type:'key',revision:socket.sent.at(-1).revision,code:'KeyJ',mods:3});
assert.equal(socket.sent.at(-1).active,true);socket.events.message({data:JSON.stringify(input())});assert.equal(dispatches.length,1);assert.equal(dispatches[0].scope.source,'iframe');assert.equal(dispatches[0].input.frameName,frame.name);
frame.name='replaced-name';socket.events.message({data:JSON.stringify(input())});assert.equal(dispatches.length,1,'changed name must not borrow pending physical key');frame.name='dsh-sidebar-html-fixture';
frame.isConnected=false;socket.events.message({data:JSON.stringify(input())});assert.equal(dispatches.length,1);frame.isConnected=true;
win.dispatchEvent(new context.CustomEvent('whaleisle-native-frame-key',{detail:{kind:'menu',commandId:'settings.open',revision:'config-r'}}));assert.equal(dispatches.length,1);
unsubscribe();assert.equal(win.whaleisleShortcutProvider.guestKeyboard.ready,false);assert.equal(socket.sent.at(-1).active,false);
console.log('verified iframe physical-key route through original native dispatcher PASS');
