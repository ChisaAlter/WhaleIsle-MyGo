import {readFileSync} from 'node:fs';
import vm from 'node:vm';
import path from 'node:path';

export const asset = name => readFileSync(new URL('../assets/' + name, import.meta.url), 'utf8');
export const baseline = process.env.PET_BODY_BASELINE;
export const plain = value => JSON.parse(JSON.stringify(value));
export const flush = async () => { for (let i = 0; i < 80; i++) await Promise.resolve(); };
export const deferred = () => {let resolve; const promise = new Promise(r => {resolve = r;}); return {promise, resolve};};

// Browser/device boundary only: every consumer, rig program and SR drawing
// helper executes its production source. No test calls a real native bridge.
export function fixture({old = false, bodyScript = true, renderer = 'neural', failScripts = [], failImages = [], empty = false, sleepEnabled = false, openGate = null, initialSleeping = false} = {}) {
  const priorNames = ['pet-runtime.js','pet-interaction.js','pet-status.js','pet-drop.js','pet-behavior.js','pet-care-animation.js','pet-layout.js','pet.html'];
  const read = name => old && priorNames.includes(name) ? readFileSync(path.join(baseline, name), 'utf8') : asset(name);
  let now = 1000, serial = 0, frameFailure = false, frameGate = null, interactionGate = null;
  const nav = [], draw = [], animation = [], captures = [], focus = [], fetches = [], scriptLoads = [], nodes = new Map();
  const timers = new Map(), rafs = new Map(), observers = [], rectReads = {};
  function listeners(node) {
    node.handlers = new Map();
    node.addEventListener = (type, fn, options) => {
      const list = node.handlers.get(type) || []; list.push({fn, capture: options === true || options?.capture, once: options?.once}); node.handlers.set(type, list);
    };
    node.removeEventListener = (type, fn) => node.handlers.set(type, (node.handlers.get(type) || []).filter(h => h.fn !== fn));
    node.invoke = (event, capture) => {
      for (const h of [...(node.handlers.get(event.type) || [])]) {
        if (!!h.capture !== !!capture) continue;
        event.currentTarget = node; h.fn(event);
        if (h.once) node.removeEventListener(event.type, h.fn);
      }
    };
    node.dispatchEvent = event => {node.invoke(event, true); node.invoke(event, false); return !event.defaultPrevented;};
    return node;
  }
  function make(name, tag = 'div') {
    const element = listeners({id: name, name, tagName: tag.toUpperCase(), hidden: false, isConnected: true, children: [], parentElement: null,
      textContent: '', attrs: {}, width: 192, height: 208, style: {setProperty(key, value) {this[key] = value;}},
      box: {x: 120, y: 180, width: 240, height: 260}, captured: new Set(), anims: [],
      getBoundingClientRect() {
        rectReads[name] = (rectReads[name] || 0) + 1;
        const r = this.box;
        return {...r, left: r.x, top: r.y, right: r.x + r.width, bottom: r.y + r.height};
      },
      appendChild(child) {child.parentElement = this; this.children.push(child); return child;},
      prepend(child) {child.parentElement = this; this.children.unshift(child);},
      remove() {this.isConnected = false; if (this.parentElement) this.parentElement.children = this.parentElement.children.filter(c => c !== this);},
      replaceChildren(...children) {this.children = children;},
      setAttribute(key, value) {this.attrs[key] = String(value);}, removeAttribute(key) {delete this.attrs[key];},
      contains(child) {for (let n = child; n; n = n.parentElement) if (n === this) return true; return false;},
      matches(selector) {
        return selector.split(',').some(raw => {
          const value = raw.trim();
          if (value.endsWith(':hover')) return hovered === this && this.matches(value.slice(0, -6));
          if (value[0] === '#') return this.id === value.slice(1);
          if (value === '.task') return this.id === 'pet-status-card';
          if (value === 'a[href]') return this.tagName === 'A' && !!this.attrs.href;
          if (value === '[role="button"]') return this.attrs.role === 'button';
          if (value === '[contenteditable="true"]') return this.attrs.contenteditable === 'true';
          if (value === '[data-pet-action]') return 'data-pet-action' in this.attrs;
          return this.tagName === value.toUpperCase();
        });
      },
      closest(selector) {for (let n = this; n; n = n.parentElement) if (n.matches(selector)) return n; return null;},
      querySelector(selector) {return this.children.find(n => n.matches(selector)) || null;},
      focus() {document.activeElement = this; focus.push(this.id);},
      setPointerCapture(id) {this.captured.add(id); captures.push(['set', this.id, id]);},
      hasPointerCapture(id) {return this.captured.has(id);},
      releasePointerCapture(id) {this.captured.delete(id); captures.push(['release', this.id, id]); emit(this, 'lostpointercapture', {pointerId: id});},
      animate(frames, options) {const item = {frames: plain(frames), options: plain(options), cancelled: false, cancel() {this.cancelled = true;}}; this.anims.push(item); animation.push([this.id, item]); return item;},
    });
    let matrix = [1, 0, 0, 1, 0, 0]; const stack = [];
    element.ctx = {
      canvas: element, save() {stack.push(matrix.slice());}, restore() {matrix = stack.pop() || [1, 0, 0, 1, 0, 0];},
      translate(x, y) {matrix[4] += matrix[0] * x + matrix[2] * y; matrix[5] += matrix[1] * x + matrix[3] * y;},
      scale(x, y) {matrix[0] *= x; matrix[1] *= x; matrix[2] *= y; matrix[3] *= y;},
      rotate(r) {const [a, b, c, d, e, f] = matrix, cos = Math.cos(r), sin = Math.sin(r); matrix = [a*cos+c*sin, b*cos+d*sin, c*cos-a*sin, d*cos-b*sin, e, f];},
      clearRect(...args) {draw.push([name, 'clear', args, matrix.slice()]);},
      drawImage(image, ...args) {draw.push([name, 'draw', image.id || image.src || 'image', args, matrix.slice()]);},
      getImageData() {return {data: new Uint8ClampedArray(512*512*4)};},
      createImageData() {return {data: new Uint8ClampedArray(512*512*4)};}, putImageData() {},
    };
    element.getContext = () => element.ctx;
    return element;
  }
  function node(id, tag) {if (!nodes.has(id)) nodes.set(id, make(id, tag)); return nodes.get(id);}
  const window = listeners({}), document = listeners({hidden: false, activeElement: null});
  let hovered = null;
  document.body = node('body'); document.body.dataset = {behavior: '{"opacity":0.75,"powerSave":true,"selfTalk":true}',
    layout: '{"X":120,"Y":180,"Width":240,"Height":260,"Area":{"X":0,"Y":0,"Width":1200,"Height":800}}',
    sleeping: String(initialSleeping), neural: String(renderer === 'neural'), rig: String(renderer !== 'sprite')};
  document.documentElement = node('html');
  document.getElementById = id => (id==='pet-body-hit'&&!sleepEnabled) || (empty && ['sprite', 'fallback'].includes(id)) ? null : node(id);
  if(sleepEnabled)node('pet-body-hit').hidden=true;
  document.querySelector = selector => {
    if (selector === '.task') return node('pet-status-card');
    if (selector.includes(':hover')) return hovered && hovered.matches(selector) ? hovered : null;
    return null;
  };
  document.createElement = tag => make('created-' + tag + '-' + ++serial, tag);
  document.activeElement = document.body;
  const sprite = node('sprite', 'canvas'), fallback = node('fallback', 'img'); sprite.hidden = true;
  sprite.parentElement = document.body; fallback.parentElement = document.body;
  for (const id of ['pet-status-card', 'pet-bubble', 'pet-status-toggle', 'render-notice']) node(id).parentElement = document.body;
  node('pet-status-card').hidden = true; node('pet-bubble').hidden = true;
  node('pet-status-toggle').tagName = 'BUTTON'; node('pet-status-close').tagName = 'BUTTON';
  node('pet-status-card').appendChild(node('pet-status-close'));
  for (const id of ['feed-file','pat-pet','feed-tokens','refresh-growth','rest-pet','pet-chat','pet-quick-chat','pet-look']) node(id).tagName = 'A';
  node('pet-status-card').appendChild(node('feed-file'));
  const layout = {X:120,Y:180,Width:240,Height:260,Area:{X:0,Y:0,Width:1200,Height:800}};
  let task = {State:'idle',Title:'test',HitEpoch:7,Layout:layout,Behavior:{opacity:.75,powerSave:true,selfTalk:true},Care:{},Growth:{},Rest:{Sleeping:false}};
  let interaction = {Epoch:7,Mode:'',Facing:1,CursorX:0,CursorY:0,Layout:layout};
  const location = {get href() {return 'https://pet.local/index.html';}, set href(value) {nav.push(value);}};
  const randomMath = Object.create(Math); randomMath.random = () => .23;
  class ClockDate extends Date {constructor(...args) {super(...(args.length ? args : [1700000000000 + now]));} static now() {return 1700000000000 + now;}}
  class Observer {constructor(callback) {this.callback = callback; this.targets = []; this.closed = false; observers.push(this);} observe(target) {this.targets.push(target.id);} disconnect() {this.closed = true;}}
  const scope = {window, document, location, performance: {now: () => now}, Date: ClockDate, Math: randomMath,
    ResizeObserver:Observer,MutationObserver:Observer,innerWidth:1200,innerHeight:800,Float32Array,Uint8ClampedArray,URL,Set,Map,Number,Object,Promise,Error,AbortController,
    CustomEvent: class {constructor(type, options) {this.type = type; Object.assign(this, options);}},
    getComputedStyle: element => ({display:element.hidden?'none':element.style.display||'block',opacity:element.style.opacity||'1'}),
    setTimeout: (fn, ms) => {timers.set(++serial, {fn, ms, interval:false}); return serial;}, clearTimeout: id => timers.delete(id),
    setInterval: (fn, ms) => {timers.set(++serial, {fn, ms, interval:true}); return serial;}, clearInterval: id => timers.delete(id),
    requestAnimationFrame: fn => {rafs.set(++serial, fn); return serial;}, cancelAnimationFrame: id => rafs.delete(id),
  };
  scope.Image = class {set src(src) {this.srcValue = src; this.id = src; this.naturalWidth = src.includes('character')?512:src==='sprite.webp'?1536:502; this.naturalHeight = src.includes('character')?512:src==='sprite.webp'?208:695; queueMicrotask(() => failImages.includes(src) ? this.onerror?.() : this.onload?.());}};
  scope.fetch = async url => {
    fetches.push(String(url));
    if (url === 'interaction.json') {const gate = interactionGate; interactionGate = null; if (gate) await gate.promise; return {ok:true,json:async()=>plain(interaction)};}
    if (url === 'task.json') return {ok:true,json:async()=>plain(task)};
    if (String(url).includes('pet-dialogue.json')) return {ok:true,json:async()=>JSON.parse(asset('pet-dialogue.json'))};
    return {ok:true,json:async()=>JSON.parse(asset('rig/manifest.json'))};
  };
  let pixels = new Uint8ClampedArray(512*512*4);
  for(let y=40;y<=480;y++)for(let x=80;x<=430;x++)pixels[(y*512+x)*4+3]=255;
  const neuralInputs=[];
  const neural = {gpuInputs:true, async frame(values) {const input={value:values,before:Array.from(values)};neuralInputs.push(input);const gate = frameGate; frameGate = null; if (gate) await gate.promise;input.after=Array.from(values);if (frameFailure) throw Error('test inference failure'); return pixels;}, async close() {draw.push(['neural-close']);}};
  scope.WhaleTHA4 = {imageCHW: () => new Float32Array(512*512*4), async open() {if(openGate)await openGate.promise;return {engine:neural,provider:'webgpu',capture:true,firstPixels:pixels};}};
  scope.ort = {env:{wasm:{}}};
  window.location = location;
  let context;
  document.head = {appendChild(tag) {scriptLoads.push(tag.src); queueMicrotask(() => {
    if (failScripts.includes(tag.src)) {tag.onerror?.(); return;}
    if (tag.src === 'pet-rig.js') vm.runInContext(asset(tag.src), context);
    tag.onload?.();
  });}};
  context = vm.createContext(scope);
  function load(name) {return vm.runInContext(read(name), context, {filename:name});}
  if (!old && bodyScript) load('pet-body.js');
  function emit(target, type, extra = {}) {
    const event = {type,target,relatedTarget:null,button:0,buttons:1,pointerId:1,screenX:140,screenY:200,clientX:140,clientY:200,
      defaultPrevented:false,stopped:false,preventDefault() {this.defaultPrevented = true;},stopPropagation() {this.stopped = true;},...extra};
    if (target !== document && target !== window) {
      document.invoke(event, true); target.invoke(event, true); target.invoke(event, false); if (!event.stopped) document.invoke(event, false);
    } else target.dispatchEvent(event);
    return event;
  }
  async function fire(ms, occurrence = 0) {
    const item = [...timers].filter(([, value]) => value.ms === ms)[occurrence];
    if (!item) throw Error('No timer at ' + ms);
    if (!item[1].interval) timers.delete(item[0]);
    await item[1].fn(); await flush();
  }
  async function frame(at = now + 50) {
    now = at; const pending = [...rafs]; rafs.clear();
    await Promise.all(pending.map(([, fn]) => fn(at))); await flush();
  }
  function observations() {
    return plain({nav,draw,animation:animation.map(([id,item])=>[id,{frames:item.frames,options:item.options,cancelled:item.cancelled}]),captures,focus,fetches,scriptLoads,rectReads,
      dataset:document.body.dataset,style:Object.fromEntries([...nodes].map(([id,n])=>[id,Object.fromEntries(Object.entries(n.style).filter(([,v])=>typeof v!=='function'))])),
      hidden:Object.fromEntries([...nodes].map(([id,n])=>[id,n.hidden])),card:node('pet-status-toggle').attrs,
      timers:[...timers.values()].map(t=>[t.ms,t.interval]),rafs:rafs.size,observers:observers.map(o=>({targets:o.targets,closed:o.closed})),
      decorations:document.body.children.map(n=>({text:n.textContent,style:Object.fromEntries(Object.entries(n.style).filter(([,v])=>typeof v!=='function'))})),
    });
  }
  return {scope,context,node,window,document,nav,draw,rafs,timers,rectReads,captures,neuralInputs,load,emit,fire,frame,observations,
    get now() {return now;}, set now(value) {now = value;}, setHover(value) {hovered = value;},
    setTask(next) {task = {...task,...next};}, setInteraction(next) {interaction = {...interaction,...next};},
    body() {return window.WhalePetBody?.current;},
    setPixels(next) {pixels=next;}, failFrame() {frameFailure = true;}, gateFrame() {frameGate = deferred(); return frameGate;}, gateInteraction() {interactionGate = deferred(); return interactionGate;},
    async start() {
      const tags = [...read('pet.html').matchAll(/<script src="([^"]+)" defer><\/script>/g)].map(match => match[1]);
      let start;
      for (const file of tags) {
        if (file === 'pet-sleep.js' && !sleepEnabled) continue;
        if (file === 'pet-body.js') continue; // Loaded, or deliberately unavailable, at fixture creation.
        if (file === 'pet-runtime.js') start = load(file);
        else if (file !== 'anime4k.js') load(file); // Only the optional GPU driver is absent; real SR uses raw fallback.
      }
      await flush(); await start; return this;
    },
    close() {emit(window, 'pagehide');},
  };
}
