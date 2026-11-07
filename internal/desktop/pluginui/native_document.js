/* A contribution keeps its actual DOM, CSS and React owner. Native only lends
 * that exact owner a clipped seat; it never serializes or remounts a component. */
/* Measured while a node is not seated: once the layout lends it a seat it fills the seat, so its seated geometry would read as a full-screen fragment and every layout would publish a new revision. */ const whaleNaturalFragments = new WeakMap(); function whaleNativeDocument(root, identify, actions, owner, budget) {
 if(!root?.isConnected||whaleRichSurface.hidden(root))return [];
 const rect=whaleRichSurface.rect(root), anchor=identify(root);
 const fragment=(node,popup)=>{
  if(budget.left--<=0||!node.isConnected||whaleRichSurface.hidden(node))return null;
  const handle=identify(node),pinned=!popup&&whaleRichSurface.applied(node)?whaleNaturalFragments.get(node):null,r=pinned||whaleRichSurface.rect(node),style=getComputedStyle(node);
  if(r.width<=0||r.height<=0)return null;
  actions.set(handle,{node,owner});
  const children=[...node.children].filter(child=>!whaleRichSurface.hidden(child));
  const childStyle=children.length===1?getComputedStyle(children[0]):null;
  const modal=node.getAttribute('aria-modal')==='true'||children.length===1&&children[0].getAttribute('aria-modal')==='true'||node.tagName==='DIALOG'&&node.matches(':modal');
  const full=pinned?pinned.full:(style.position==='fixed'||childStyle?.position==='fixed')&&r.width>=innerWidth-2&&r.height>=innerHeight-2;
  const inline=pinned?pinned.inline:(node.hasAttribute('data-whale-slot-entry')||style.display==='contents')&&children.length===1&&(/^(inline|inline-block|inline-flex|inline-grid)$/.test(childStyle.display)||['BUTTON','IMG','SVG','INPUT'].includes(children[0].tagName));
  if(!popup&&!pinned)whaleNaturalFragments.set(node,{width:r.width,height:r.height,full,inline});return {kind:'rich',richType:'fragment',handle,label:node.getAttribute('aria-label')||'',height:Math.min(1200,Math.max(1,r.height)),width:inline||popup||full?Math.min(8192,Math.max(1,r.width)):0,inline,popup:!!popup,modal,fullScreen:full,anchor:popup?anchor:'',x:popup?r.left-rect.left:0,y:popup?r.top-rect.top:0};
 };
 const result=[],main=fragment(root,false);if(main)result.push(main);
 const portals=new Set(whaleRichSurface.portals(owner));
 for(const trigger of root.querySelectorAll('[aria-controls][aria-expanded="true"]'))for(const id of String(trigger.getAttribute('aria-controls')||'').split(/\s+/)){
  const portal=document.getElementById(id);if(portal&&!root.contains(portal)&&!whaleRichSurface.hidden(portal))portals.add(portal);
 }
 for(const node of portals){if([...portals].some(parent=>parent!==node&&parent.contains(node)))continue;const value=fragment(node,true);if(value)result.push(value)}
 return result;
}
/* Ambient shell entries keep their authored coordinates: every occupied box
 * becomes an absolute fragment, so fixed descendants and owned portals land
 * where the contribution placed them. Once the page is resized to the native
 * content bounds, DOM and native coordinates agree; vw/vh carry the
 * measure-time viewport so the first layout can edge-anchor stale readings. */
function whaleAmbientDocument(root, identify, actions, owner) {
 if(!root?.isConnected||whaleRichSurface.hidden(root))return [];
 const vw=Math.max(1,window.innerWidth||0),vh=Math.max(1,window.innerHeight||0),out=[],seen=new Set(),ambientOwner={...owner,ambient:true};
 const seat=(node,modal)=>{
  if(out.length>=24||!node?.isConnected||seen.has(node)||whaleRichSurface.hidden(node))return;
  seen.add(node);
  const r=whaleRichSurface.rect(node);if(r.width<1||r.height<1)return;
  const style=getComputedStyle(node),handle=identify(node);actions.set(handle,{node,owner:ambientOwner});
  const dialog=node.tagName==='DIALOG'&&!!node.matches?.(':modal');
  modal=modal||node.getAttribute?.('aria-modal')==='true'||dialog;
  out.push({kind:'rich',richType:'fragment',handle,label:String(node.getAttribute?.('aria-label')||'').slice(0,256),height:Math.min(1200,Math.max(1,r.height)),width:Math.min(8192,Math.max(1,r.width)),popup:true,modal,fullScreen:modal||dialog||(style.position==='fixed'&&r.width>=vw-2&&r.height>=vh-2),ambient:true,x:r.left,y:r.top,vw,vh});
 };
 for(const child of root.children||[]){if(out.length>=24)break;const r=whaleRichSurface.rect(child);if(r.width>=1&&r.height>=1)seat(child)}
 for(const node of root.querySelectorAll('*')){if(out.length>=24)break;if(getComputedStyle(node).position==='fixed'&&![...seen].some(parent=>parent!==node&&parent.contains?.(node)))seat(node)}
 for(const portal of whaleRichSurface.portals(owner))seat(portal,true);
 return out;
}
function whaleNativeInteraction(command, record) {
 const node=record?.node;
 if(!node?.isConnected||node.disabled||node.getAttribute('aria-disabled')==='true')return false;
 if(command.action==='escape'){node.dispatchEvent(new KeyboardEvent('keydown',{key:'Escape',code:'Escape',bubbles:true,cancelable:true}));return true}
 if(command.action==='outside'){for(const type of ['pointerdown','pointerup','click'])document.body.dispatchEvent(type.startsWith('pointer')?new PointerEvent(type,{bubbles:true,cancelable:true,pointerId:1,pointerType:'mouse',button:0}):new MouseEvent(type,{bubbles:true,cancelable:true,button:0}));return true}
 if(command.action==='click'){node.click();return true}
 if(command.action!=='input'||typeof command.value!=='string'||command.value.length>65536)return false;
 if(node.isContentEditable)node.textContent=command.value;
 else{const proto=node.tagName==='TEXTAREA'?HTMLTextAreaElement.prototype:node.tagName==='SELECT'?HTMLSelectElement.prototype:HTMLInputElement.prototype;const setter=Object.getOwnPropertyDescriptor(proto,'value')?.set;if(!setter)return false;setter.call(node,command.value)}
 node.dispatchEvent(new Event('input',{bubbles:true}));node.dispatchEvent(new Event('change',{bubbles:true}));return true;
}
function whaleNativeContains(owner,node){
 const root=owner?.root||owner?.marker;if(!root?.isConnected||!node?.isConnected)return false;if(root.contains(node))return true;
 for(const trigger of root.querySelectorAll('[aria-controls][aria-expanded="true"]'))for(const id of String(trigger.getAttribute('aria-controls')||'').split(/\s+/)){const portal=document.getElementById(id);if(portal?.isConnected&&!whaleRichSurface.hidden(portal)&&portal.contains(node))return true}
 for(const portal of whaleRichSurface.portals(owner))if(portal.contains(node))return true;return false;
}
