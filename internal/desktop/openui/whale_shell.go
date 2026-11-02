package openui

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"html"
)

// The new child protocol is adapted into the existing bounded helper transport.
// The child has no native bindings and only this shell knows packaged assets.
func whaleShell(viewer string, r Render, width int, appearance string) (string, error) {
	if width < 240 || width > 1600 || (appearance != "light" && appearance != "dark") {
		return "", errors.New("invalid Whale UI dimensions or appearance")
	}
	if r.Language == "" {
		r.Language = "whale-ui"
	}
	if r.Locale == "" {
		r.Locale = "en"
	}
	variables := r.Variables
	if variables == nil {
		variables = map[string]string{}
	}
	initial, _ := json.Marshal(map[string]any{"program": r.Program, "language": r.Language, "streaming": r.Streaming, "state": r.State, "locale": r.Locale, "panel": r.Panel, "revision": r.Revision, "media": r.Media, "htmlView": false, "theme": map[string]any{"appearance": appearance, "variables": variables}})
	// srcdoc inherits the outer policy as an additional restriction. Keep its
	// offline data resources allowed at both levels; the historical parentPolicy
	// otherwise blocks even validated bitmaps and packaged engine/font bytes.
	whaleParentPolicy := "default-src 'none'; script-src 'unsafe-inline' data: blob:; style-src 'unsafe-inline'; img-src data: blob:; font-src data:; frame-src 'self' about: data: blob:; connect-src 'none'; worker-src 'none'; object-src 'none'; media-src 'none'; base-uri 'none'; form-action 'none'"
	childPolicy := whaleParentPolicy
	child, _ := json.Marshal(`<meta http-equiv="Content-Security-Policy" content="` + html.EscapeString(childPolicy) + `">` + isolatedWhaleViewer(viewer))
	id, _ := json.Marshal(r.InstanceID)
	bundles := map[string]string{}
	for name := range bundleNames {
		b, err := Bundle(name)
		if err != nil {
			return "", err
		}
		bundles[name] = base64.StdEncoding.EncodeToString(b)
	}
	assets, _ := json.Marshal(bundles)
	enabled, _ := json.Marshal(r.HostActions)
	return `<!doctype html><meta http-equiv="Content-Security-Policy" content="` + html.EscapeString(whaleParentPolicy) + `"><style>html,body{margin:0;height:100%;overflow:hidden}iframe{border:0;width:100%;height:100vh}</style><script>
'use strict';const instance=` + string(id) + `,render=` + string(initial) + `,assets=` + string(assets) + `,hostActions=` + string(enabled) + `;
let queue=[],downloads=[],seq=0,ready=false,closed=false;const pending=new Map(),seen=new Set(),nativePending=new Set(),cancelled=new Set();
window.__previewState={ready:false,height:320,error:'',consoleMessages:[],omitted:0};
const frame=document.createElement('iframe');frame.name='whale-ui:'+instance;frame.setAttribute('sandbox','allow-scripts allow-forms');frame.referrerPolicy='no-referrer';
const post=(type,payload={})=>{if(!closed)frame.contentWindow.postMessage({channel:'whale-ui',version:2,instanceId:instance,type,...payload},'*')};
const bytes=x=>new TextEncoder().encode(x).length;
const record=x=>x!==null&&typeof x==='object'&&!Array.isArray(x);
const event=(type,payload={})=>({channel:'whale-openui',version:1,instanceId:instance,revision:render.revision,type,...payload});
const enqueue=e=>{if(bytes(JSON.stringify(e))>65536)throw Error('Whale UI event exceeds transport limit');if(queue.length>=16)throw Error('Whale UI event queue is full');queue.push(e)};
const reject=(requestId,error)=>post('action-response',{requestId,result:{ok:false,error}});
window.__whaleOpenUIDrain=()=>{let result=queue.splice(0,16);while(result.length<16&&downloads.length)result.push(downloads.shift());return result};
window.__whaleOpenUIHtmlViewSupport=enabled=>{if(typeof enabled!=='boolean')throw Error('Invalid HtmlView capability');render.htmlView=enabled;if(ready)post('render',render);return true;};
window.__whaleOpenUIRender=next=>{
 if(!next||next.instanceId!==instance||typeof next.program!=='string'||typeof next.streaming!=='boolean'||bytes(next.program)>262144||!Number.isSafeInteger(next.revision)||next.revision<render.revision)throw Error('Invalid Whale UI update');
 if(next.program!==render.program||next.streaming!==render.streaming){for(const [key,p]of pending){cancelled.add(key);if(p.kind==='action')reject(p.original,'The interface changed before the action completed.')}while(cancelled.size>256)cancelled.delete(cancelled.values().next().value);pending.clear();queue=queue.filter(e=>e.type==='state');downloads=[];seen.clear();}
 if(next.language!==undefined){if(!['whale-ui','dsh-ui','openui','openui-lang'].includes(next.language))throw Error('Invalid Whale UI language');render.language=next.language;}if(next.locale!==undefined){if(!['zh','en'].includes(next.locale))throw Error('Invalid Whale UI locale');render.locale=next.locale;}if(next.variables!==undefined){if(!record(next.variables))throw Error('Invalid Whale UI theme');render.theme.variables=next.variables;}if(next.panel!==undefined)render.panel=next.panel;
 render.program=next.program;render.streaming=next.streaming;render.revision=next.revision;post('render',render);return true;
};
window.__previewTheme=data=>{if(!data||!['light','dark'].includes(data.appearance))throw Error('Invalid Whale UI theme');const vars={...(data.variables||{})};if(['zh','en'].includes(vars['--whale-locale']))render.locale=vars['--whale-locale'];delete vars['--whale-locale'];render.theme={appearance:data.appearance,variables:{...render.theme.variables,...vars}};if(ready)post('render',render)};
window.__whaleOpenUIAPIResponse=r=>{
 if(r&&r.instanceId===instance&&cancelled.delete(r.requestId))return true;
 if(!r||r.instanceId!==instance||typeof r.requestId!=='string'||!pending.has(r.requestId)||!Number.isInteger(r.status)||r.status<200||r.status>599)throw Error('Invalid Whale UI reply');
 const p=pending.get(r.requestId);pending.delete(r.requestId);if(p.kind==='native-resource'){post('api-response',{requestId:p.original,status:r.status,body:r.body});return true;}if(p.kind==='media'){post('media-response',{requestId:p.original,...r.body});return true;}if(p.kind==='resource')post('resource-response',{requestId:p.original,status:r.status,body:r.body});else {const ok=r.status<300&&record(r.body)&&r.body.ok===true;post('action-response',{requestId:p.original,result:ok?{ok:true}:{ok:false,error:typeof r.body?.error==='string'?r.body.error:'The Host action failed.'}})}return true;
};
addEventListener('message',e=>{
 const d=e.data;if(closed||e.source!==frame.contentWindow||e.origin!=='null'||!record(d)||d.channel!=='whale-ui'||d.version!==2||d.instanceId!==instance)return;
 try{
 if(d.type==='ready'){ready=true;window.__previewState.ready=true;post('render',render);return;}
 if(d.type==='native-adapter'){
  if(typeof d.requestId!=='string'||nativePending.has(d.requestId))return;
  if(d.nativeType==='api-request'){const allowed={'/api/map-tile':['z','x','y','style'],'/api/street-route':['coords'],'/api/image':['src'],'/api/wiki':['title'],'/api/image-search':['q'],'/api/html-script':['src','integrity']};if(!Object.hasOwn(allowed,d.path)||!record(d.query)||Object.entries(d.query).some(([k,v])=>!allowed[d.path].includes(k)||typeof v!=='string'))return;nativePending.add(d.requestId);const key='resource-'+(++seq);pending.set(key,{kind:'native-resource',original:d.requestId});enqueue(event('api-request',{requestId:key,path:d.path,query:d.query}));return;}
  if(d.nativeType==='media'&&render.media===true&&record(d.media)&&d.media.revision===render.revision&&typeof d.sequence==='number'&&d.requestId==='media-'+d.sequence){nativePending.add(d.requestId);pending.set(d.requestId,{kind:'media',original:d.requestId});enqueue(event('media',{requestId:d.requestId,sequence:d.sequence,userGesture:d.userGesture===true,media:d.media}));}return;
 }
 if(d.type==='height'){if(Number.isFinite(d.height))window.__previewState.height=Math.max(80,Math.min(2000,Math.ceil(d.height)));return;}
 if(d.type==='state'){if(!record(d.state)||bytes(JSON.stringify(d.state))>49152)throw Error('Whale UI state exceeds limit');render.state=d.state;queue=queue.filter(x=>x.type!=='state');enqueue(event('state',{state:d.state}));return;}
 if(d.type==='title'){if(typeof d.title==='string')enqueue(event('title',{text:d.title.trim().slice(0,120)}));return;}
 if(d.type==='diagnostics'){if(d.program!==render.program||!Array.isArray(d.diagnostics))return;const errors=d.diagnostics.filter(x=>record(x)&&typeof x.message==='string').map(x=>(typeof x.path==='string'?x.path+': ':'')+x.message).join('\n');if(errors)enqueue(event('error',{message:errors.slice(0,1000),code:'whale-ui-diagnostics'}));else if(d.complete===true)enqueue(event('resolved',{program:bytes(render.program)<=32768?render.program:''}));return;}
 if(d.type==='panel'){if(render.panel||render.streaming||!record(d.operation)||!record(d.operation.spec)||!Array.isArray(d.operation.spec.items)||typeof d.operation.append!=='boolean')throw Error('Invalid Whale UI panel operation');enqueue(event('panel',{operation:{append:d.operation.append,spec:d.operation.spec}}));return;}
 if(!['resource','bundle','action'].includes(d.type))return;
 if(typeof d.requestId!=='string'||!d.requestId||d.requestId.length>160||seen.has(d.requestId))return;seen.add(d.requestId);
 if(d.type==='bundle'){if(typeof d.name!=='string'||!Object.hasOwn(assets,d.name)){post('bundle-response',{requestId:d.requestId,name:String(d.name),error:'Unsupported packaged rendering engine'});return;}post('bundle-response',{requestId:d.requestId,name:d.name,base64:assets[d.name]});return;}
 if(d.type==='resource'){const keys={'/api/wiki':['title'],'/api/image-search':['q'],'/api/street-route':['coords']};if(!Object.hasOwn(keys,d.path)||!record(d.query)||Object.entries(d.query).some(([k,v])=>!keys[d.path].includes(k)||typeof v!=='string')){post('resource-response',{requestId:d.requestId,status:400,body:{error:'Unsupported Whale UI resource'}});return;}const key='resource-'+(++seq);pending.set(key,{kind:'resource',original:d.requestId});enqueue(event('api-request',{requestId:key,path:d.path,query:d.query}));return;}
 if(!hostActions||render.streaming||d.userGesture!==true){reject(d.requestId,'This action requires an active interface and a user activation.');return;}
 if(downloads.length){reject(d.requestId,'Wait for the current export to finish transferring.');return;}
 const sequence=++seq,key='native-action-'+sequence,base={requestId:key,sequence,userGesture:true};
 if(d.action==='assistant'){if(typeof d.message!=='string'||!d.message.trim()||bytes(d.message)>16384)throw Error('Invalid Whale UI assistant message');pending.set(key,{kind:'action',original:d.requestId});enqueue(event('assistant',{...base,message:d.message,...(d.formState===undefined?{}:{formState:d.formState}),...(d.context===undefined?{}:{context:d.context})}));return;}
 if(d.action==='clipboard'){if(typeof d.text!=='string'||d.text.includes('\u0000'))throw Error('Invalid clipboard text');pending.set(key,{kind:'action',original:d.requestId});enqueue(event('clipboard',{...base,text:d.text}));return;}
 if(d.action==='open-url'){let url=new URL(d.url);if(typeof d.url!=='string'||/[\u0000-\u0020\u007f-\u009f]/.test(d.url)||!['http:','https:'].includes(url.protocol)||!url.hostname||url.username||url.password)throw Error('Only absolute HTTP(S) links without credentials are supported');pending.set(key,{kind:'action',original:d.requestId});enqueue(event('open-url',{...base,url:d.url}));return;}
 if(d.action==='download'){
  if(typeof d.filename!=='string'||!d.filename||d.filename.length>240||/[<>:"/\\|?*\u0000-\u001f\u007f]/.test(d.filename)||/[. ]$/.test(d.filename)||d.filename==='.'||d.filename==='..'||typeof d.mime!=='string'||!/^[-\w.+]+\/[-\w.+]+(?:;[-\w=.+ ]+)*$/.test(d.mime)||typeof d.content!=='string'||bytes(d.content)>25165824||downloads.length)throw Error('Invalid download or another export is pending');
  const parts=[];for(let i=0;i<d.content.length;){let end=Math.min(i+12000,d.content.length);if(end<d.content.length&&/[\uD800-\uDBFF]/.test(d.content[end-1]))end--;parts.push(d.content.slice(i,end));i=end;}if(!parts.length)parts.push('');
  pending.set(key,{kind:'action',original:d.requestId});downloads=parts.map((content,part)=>event('download-part',{...base,filename:d.filename,mime:d.mime,content,part,parts:parts.length}));downloads.push(event('download',{...base,filename:d.filename,mime:d.mime,parts:parts.length}));return;
 }
 throw Error('Unsupported Whale UI action');
 }catch(error){if(d.type==='action'&&typeof d.requestId==='string')reject(d.requestId,String(error.message||error));else if(d.type==='resource'&&typeof d.requestId==='string')post('resource-response',{requestId:d.requestId,status:503,body:{error:String(error.message||error)}});else window.__previewState.error=String(error.message||error);}
});
addEventListener('pagehide',()=>{closed=true;queue=[];downloads=[];pending.clear()},{once:true});
frame.srcdoc=` + string(child) + `;addEventListener('DOMContentLoaded',()=>document.body.append(frame),{once:true});</script><body></body>`, nil
}
