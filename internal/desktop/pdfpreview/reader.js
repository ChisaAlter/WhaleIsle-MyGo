// Product-owned, offline PDF.js 6.3.289 reader. No PDF scripting/annotations,
// native bridge, arbitrary URL, file access, or network fallback is installed.
import { getDocument, PDFWorker } from './pdf.mjs';
import { TextLayerBuilder } from './pdf_viewer.mjs';

const port = document.querySelector('#scrollport'), pages = document.querySelector('#pages');
const pageInput = document.querySelector('#page'), zoomInput = document.querySelector('#zoom');
const status = document.querySelector('#status'), retry = document.querySelector('#retry');
let pdf, worker, bridge, loading, observer, disposed = false, restoring = false, generation = 0;
const number = (x, lo, hi, fallback) => Number.isInteger(+x) && +x >= lo && +x <= hi ? +x : fallback;
let records = [], state = readState(), publishing = false, scrollTimer, renderQueue = Promise.resolve(), navigationPosition = null;
function readState() {
  const p = new URLSearchParams(location.hash.slice(1));
  const z = (p.get('zoom') || '100,0,0').split(',');
  const s = { page:number(p.get('page'),1,5000,1), zoom:number(z[0],25,500,100), left:number(z[1],0,1000000000,0), top:number(z[2],0,1000000000,0), fit:p.get('fit') !== '0' };
  const sel = p.get('selection');
  if (sel && sel.length <= 160) {
    const v = sel.split(',').map(Number);
    if (v.length === 4 && v.every(Number.isSafeInteger) && v[0] > 0 && v[0] <= 5000 && v[2] > 0 && v[2] <= 5000 && v[1] >= 0 && v[1] <= 1000000000 && v[3] >= 0 && v[3] <= 1000000000) s.selection = v;
  }
  return s;
}
function publish() {
  if (!pdf || restoring || disposed) return;
  state.left = Math.round(port.scrollLeft); state.top = Math.round(port.scrollTop);
  // A short last page cannot always align with the viewport's top. Keep an
  // explicit page choice at its clamped scroll position, until actual scrolling
  // changes that position; otherwise Next immediately snaps its page number back.
  if (!navigationPosition || state.left !== navigationPosition.left || state.top !== navigationPosition.top) {
    navigationPosition = null;
    let nearest = records[0];
    for (const r of records) { if (r.node.offsetTop <= port.scrollTop + 16) nearest = r; else break; }
    state.page = nearest?.index || 1;
  }
  const p = new URLSearchParams({page:String(state.page),zoom:[state.zoom,state.left,state.top].join(','),fit:state.fit?'1':'0'});
  if (state.selection) p.set('selection',state.selection.join(','));
  publishing = true;
  history.replaceState(null,'',location.pathname+'#'+p.toString());
  publishing = false;
  pageInput.value = String(state.page); zoomInput.value = state.fit ? 'fit' : String(state.zoom);
}
function failure(error) {
  if (disposed) return;
  status.textContent = error?.name === 'PasswordException' ? 'PDF 已加密，请使用系统打开。' : 'PDF 无法载入：'+String(error?.message || error);
  retry.hidden = false;
}
function cancelRender(r) {
  r.task?.cancel(); r.task = undefined;
  r.layer?.cancel(); r.layer = undefined;
  r.version = -1;
}
function layout() {
  const width = records[0]?.size.width || 600;
  const scale = state.fit ? Math.max(.1, (port.clientWidth - 24) / width) : state.zoom / 100;
  generation++;
  for (const r of records) {
    cancelRender(r);
    r.scale = scale; r.node.style.width = r.size.width*scale+'px'; r.node.style.height = r.size.height*scale+'px';
    r.node.style.setProperty('--total-scale-factor',String(scale * r.page.userUnit));
    r.node.style.setProperty('--scale-round-x','1px');r.node.style.setProperty('--scale-round-y','1px');
    r.text.replaceChildren();
  }
}
function requestRender(r) {
  const wanted = generation;
  if (r.version === wanted || r.queued === wanted || disposed) return;
  r.queued = wanted;
  renderQueue = renderQueue.then(async () => {
    if (disposed || wanted !== generation) return;
    r.node.dataset.loading = '';
    r.node.querySelector('.failure')?.remove();
    const viewport = r.page.getViewport({scale:r.scale});
    const dpr = Math.min(devicePixelRatio || 1,2);
    const pixels = viewport.width * viewport.height * dpr*dpr;
    if (pixels > 32000000) throw new Error('页面缩放超过绘制上限，请降低缩放后重试。');
    const canvas = document.createElement('canvas');
    canvas.width = Math.ceil(viewport.width*dpr);canvas.height = Math.ceil(viewport.height*dpr);
    r.task = r.page.render({canvas,canvasContext:canvas.getContext('2d'),viewport,transform:dpr===1?undefined:[dpr,0,0,dpr,0,0],annotationMode:0});
    await r.task.promise;
    if (disposed || wanted !== generation) return;
    r.canvas.replaceWith(canvas);r.canvas=canvas;r.task=undefined;
    const layer = new TextLayerBuilder({pdfPage:r.page});
    r.layer=layer;layer.div.style.setProperty('--total-scale-factor',String(viewport.scale*viewport.userUnit));
    layer.div.style.setProperty('--scale-round-x','1px');layer.div.style.setProperty('--scale-round-y','1px');
    r.text.replaceChildren(layer.div);
    await layer.render({viewport});
    if (disposed || wanted !== generation) return;
    r.version=wanted;delete r.node.dataset.loading;
    restoreSelection();
  }).catch(error => {
    if (disposed || wanted !== generation || error?.name==='RenderingCancelledException') return;
    delete r.node.dataset.loading;
    const msg=document.createElement('p');msg.className='failure';msg.textContent=String(error?.message||error);r.node.append(msg);
    r.queued=-1;
  });
}
function renderVisible() {
  const top=port.scrollTop, bottom=top+port.clientHeight;
  for (const r of records) if (r.node.offsetTop+r.node.offsetHeight >= top-port.clientHeight && r.node.offsetTop <= bottom+port.clientHeight) requestRender(r);
  if (state.selection) for (const n of [state.selection[0],state.selection[2]]) {const r=records[n-1];if(r)requestRender(r);}
}
function selectionPoint(node,offset) {
  const el=node?.nodeType===Node.ELEMENT_NODE?node:node?.parentElement;
  const page=el?.closest('.page'),text=el?.closest('.textLayer');
  if(!page||!text)return null;
  const range=document.createRange();range.selectNodeContents(text);
  try{range.setEnd(node,offset);}catch{return null;}
  return [+page.dataset.page,range.toString().length];
}
function textPoint(r,offset) {
  const root=r.text.querySelector('.textLayer');if(!root)return null;
  const walk=document.createTreeWalker(root,NodeFilter.SHOW_TEXT);let node;
  while((node=walk.nextNode())){if(offset<=node.length)return [node,offset];offset-=node.length;}
  return null;
}
function restoreSelection() {
  const v=state.selection;if(!v)return;
  const a=records[v[0]-1],b=records[v[2]-1];if(!a||!b||a.version!==generation||b.version!==generation)return;
  const start=textPoint(a,v[1]),end=textPoint(b,v[3]);if(!start||!end)return;
  const savedLeft=port.scrollLeft,savedTop=port.scrollTop;
  restoring=true;getSelection().setBaseAndExtent(...start,...end);port.scrollTo(savedLeft,savedTop);restoring=false;
}
async function apply(next,jump=false) {
  state=next;if(!pdf)return;
  restoring=true;
  const changed=state.fit!==lastLayout.fit||state.zoom!==lastLayout.zoom;
  if(changed){layout();lastLayout={fit:state.fit,zoom:state.zoom};}
  state.page=Math.min(state.page,pdf.numPages);
  port.scrollTo(state.left,jump?records[state.page-1].node.offsetTop-12:state.top);
  navigationPosition={left:Math.round(port.scrollLeft),top:Math.round(port.scrollTop)};
  restoring=false;renderVisible();restoreSelection();publish();
}
let lastLayout={};
let userSelecting=false;
port.addEventListener("pointerdown",()=>{userSelecting=true;});
port.addEventListener("pointerup",()=>setTimeout(()=>{userSelecting=false;},0));
port.addEventListener("keydown",()=>{userSelecting=true;});
port.addEventListener("keyup",()=>setTimeout(()=>{userSelecting=false;},0));
pageInput.addEventListener('change',()=>apply({...state,page:number(pageInput.value,1,pdf?.numPages||1,1),selection:undefined},true));
document.querySelector('#previous').addEventListener('click',()=>apply({...state,page:Math.max(1,state.page-1),selection:undefined},true));
document.querySelector('#next').addEventListener('click',()=>apply({...state,page:Math.min(pdf?.numPages||1,state.page+1),selection:undefined},true));
zoomInput.addEventListener('change',()=>{
  const next={...state,fit:zoomInput.value==='fit',zoom:zoomInput.value==='fit'?state.zoom:number(zoomInput.value,25,500,100)};
  apply(next,true);
});
port.addEventListener('scroll',()=>{if(restoring)return;renderVisible();clearTimeout(scrollTimer);scrollTimer=setTimeout(publish,60);},{passive:true});
port.addEventListener('pointerup',publish);
document.addEventListener('selectionchange',()=>{
  if(restoring||!pdf||!userSelecting)return;
  const sel=getSelection();const a=selectionPoint(sel.anchorNode,sel.anchorOffset),b=selectionPoint(sel.focusNode,sel.focusOffset);
  if(a&&b&&!sel.isCollapsed)state.selection=[...a,...b];else if(sel.isCollapsed)delete state.selection;
  publish();
});
window.addEventListener('hashchange',()=>{if(!publishing)apply(readState());});
new ResizeObserver(()=>{if(pdf&&state.fit){lastLayout={};apply({...state},true);}}).observe(port);
retry.addEventListener('click',()=>location.reload());
window.addEventListener('pagehide',()=>{publish();dispose();});
async function dispose(){if(disposed)return;disposed=true;clearTimeout(scrollTimer);observer?.disconnect();records.forEach(cancelRender);try{await Promise.race([loading?.destroy(),new Promise(resolve=>setTimeout(resolve,1000))]);}finally{bridge?.destroy();worker?.terminate();}}

try {
  const [response,assetResponse]=await Promise.all([fetch('document.pdf',{credentials:'omit'}),fetch('assets.json',{credentials:'omit'})]);
  if(!response.ok||!assetResponse.ok)throw new Error('文档已过期，请重新打开。');
  const bytes=new Uint8Array(await response.arrayBuffer()),assets=await assetResponse.json();
  class BinaryDataFactory {async fetch({kind,filename}){const files=assets[kind];if(!files||!Object.hasOwn(files,filename))throw new Error('缺少离线 PDF 资源');return Uint8Array.from(atob(files[filename]),c=>c.charCodeAt(0));}}
  worker=new Worker(new URL('./pdf.worker.mjs',import.meta.url),{type:'module',name:'whaleisle-pdf'});
  let rejectWorker;const broken=new Promise((_,reject)=>{rejectWorker=reject;});
  const workerFailed=()=>{const error=new Error('PDF 工作线程已停止，请重试。');rejectWorker(error);failure(error);void dispose();};
  worker.addEventListener('error',workerFailed);worker.addEventListener('messageerror',workerFailed);
  bridge=PDFWorker.create({port:worker});
  loading=getDocument({data:bytes,worker:bridge,BinaryDataFactory,cMapPacked:true,useWorkerFetch:false,enableXfa:false,stopAtErrors:true,isEvalSupported:false});
  let startupTimer;
  try { pdf=await Promise.race([loading.promise,broken,new Promise((_,reject)=>{startupTimer=setTimeout(()=>reject(new Error("PDF 载入超时，请重试。")),30000);})]); } finally { clearTimeout(startupTimer); }
  if(disposed)throw new Error('Reader closed');
  if(pdf.numPages>5000)throw new Error('PDF 页数超过预览上限，请使用系统打开。');
  // Establish every page's geometry before restoring physical scroll offsets.
  // Canvas/text work remains lazy and bounded to the viewport.
  for(let index=1;index<=pdf.numPages;index++){
    const page=await pdf.getPage(index),size=page.getViewport({scale:1});
    const node=document.createElement('section');node.className='page';node.dataset.page=String(index);node.setAttribute('aria-label','第 '+index+' 页');
    const canvas=document.createElement('canvas'),text=document.createElement('div');canvas.width=canvas.height=0;node.append(canvas,text);pages.append(node);
    records.push({index,page,size,node,canvas,text,version:-1,queued:-1});
  }
  document.querySelector('#count').textContent='/ '+pdf.numPages;pageInput.max=String(pdf.numPages);status.textContent='';
  observer=new IntersectionObserver(entries=>{for(const e of entries)if(e.isIntersecting)requestRender(records[+e.target.dataset.page-1]);},{root:port,rootMargin:'100% 0px'});
  records.forEach(r=>observer.observe(r.node));
  await apply(state,state.top===0&&state.page>1);
}catch(error){failure(error);void dispose();}
