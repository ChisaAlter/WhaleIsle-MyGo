package openui

import (
	"encoding/json"
	"errors"
	"html"
	"strings"
	"unicode/utf8"
)

const childPolicy = "default-src 'none'; script-src 'unsafe-inline' data:; style-src 'unsafe-inline'; img-src data:; font-src data:; connect-src 'none'; frame-src 'none'; worker-src 'none'; object-src 'none'; media-src data: blob:; base-uri 'none'; form-action 'none'"
const parentPolicy = "default-src 'none'; script-src 'unsafe-inline'; style-src 'unsafe-inline'; frame-src 'self' about:; connect-src 'none'; worker-src 'none'; object-src 'none'; base-uri 'none'; form-action 'none'"

func Shell(viewerHTML string, r Render, width int, appearance string) (string, error) {
	if err := ValidateRender(r); err != nil {
		return "", err
	}
	if IsWhaleViewer(viewerHTML) {
		return whaleShell(viewerHTML, r, width, appearance)
	}
	if len(viewerHTML) > MaxViewerBytes || !utf8.ValidString(viewerHTML) || strings.TrimSpace(viewerHTML) == "" || width < 240 || width > 1600 || (appearance != "light" && appearance != "dark") {
		return "", errors.New("invalid OpenUI viewer parameters")
	}
	child, _ := json.Marshal(`<meta http-equiv="Content-Security-Policy" content="` + html.EscapeString(childPolicy) + `">` + viewerHTML)
	initial, _ := json.Marshal(map[string]any{"channel": "whale-openui", "version": 1, "instanceId": r.InstanceID, "type": "render", "program": r.Program, "streaming": r.Streaming, "state": r.State, "htmlView": false, "hostActions": r.HostActions && IsOwnedViewer(viewerHTML), "media": r.Media && r.Revision > 0 && IsOwnedViewer(viewerHTML), "revision": r.Revision, "theme": map[string]any{"appearance": appearance, "variables": map[string]string{}}})
	id, _ := json.Marshal(r.InstanceID)
	return `<!doctype html><meta http-equiv="Content-Security-Policy" content="` + html.EscapeString(parentPolicy) + `"><style>html,body{margin:0;height:100%;overflow:hidden}iframe{border:0;width:100%;height:100vh}</style><script>
'use strict';const instance=` + string(id) + `,render=` + string(initial) + `;let queue=[];const apiPending=new Set(),actionPending=new Map(),mediaPending=new Map();let lastActionSequence=0,lastMediaSequence=0;
window.__previewState={ready:false,height:360,error:'',consoleMessages:[],omitted:0};
window.__whaleOpenUIDrain=()=>queue.splice(0,16);
const frame=document.createElement('iframe');frame.name='whale-openui:'+instance;frame.setAttribute('sandbox','allow-scripts allow-forms');frame.referrerPolicy='no-referrer';
const post=(type,data={})=>frame.contentWindow.postMessage({channel:'whale-openui',version:1,instanceId:instance,type,...data},'*');
// Only trusted native Eval calls this setter after actual frame guards exist.
// No descendant message is accepted as a capability grant.
window.__whaleOpenUIHtmlViewSupport=enabled=>{if(typeof enabled!=='boolean')throw Error('Invalid HtmlView capability');render.htmlView=enabled;post('render',render);return true};
window.__whaleOpenUIRender=next=>{if(!next||next.instanceId!==instance||typeof next.program!=='string'||typeof next.streaming!=='boolean'||new TextEncoder().encode(next.program).length>262144)throw Error('Invalid OpenUI update');if(render.media&&(!Number.isSafeInteger(next.revision)||next.revision<render.revision||((next.program!==render.program||next.streaming!==render.streaming)&&next.revision<=render.revision)))throw Error('Invalid media source revision');if(next.revision!==undefined){if(!Number.isSafeInteger(next.revision)||next.revision<0)throw Error('Invalid source revision');render.revision=next.revision;}render.program=next.program;render.streaming=next.streaming;post('render',render);return true};
window.__whaleOpenUIAPIResponse=r=>{
 if(!r||r.instanceId!==instance||typeof r.requestId!=='string'||(!apiPending.has(r.requestId)&&!actionPending.has(r.requestId)&&!mediaPending.has(r.requestId))||!Number.isInteger(r.status)||r.status<200||r.status>599)throw Error('Invalid OpenUI API response');
 const media=mediaPending.get(r.requestId);
 if(media){mediaPending.delete(r.requestId);const body=r.body,valid=body&&typeof body==='object'&&!Array.isArray(body),state=valid?body.state:undefined,ok=r.status<300&&valid&&body.ok===true&&['native-view-created','hidden','closed'].includes(state);let error=valid&&typeof body.error==='string'?body.error:'The Host could not create the isolated player.';if(new TextEncoder().encode(error).length>4096)error='The Host returned an oversized media error.';post('media-response',{requestId:r.requestId,ok:!!ok,...(ok?{state}:{error})});return true;}
 const action=actionPending.get(r.requestId);
 if(action){actionPending.delete(r.requestId);const body=r.body,ok=r.status<300&&body&&typeof body==='object'&&!Array.isArray(body)&&body.ok===true;let error=body&&typeof body.error==='string'?body.error:'The Host action was not completed.';if(new TextEncoder().encode(error).length>4096)error='The Host action returned an oversized error.';post('action-response',{requestId:r.requestId,action,ok:!!ok,...(ok?{}:{error})});return true;}
 apiPending.delete(r.requestId);post('api-response',{requestId:r.requestId,status:r.status,body:r.body});return true
};
window.__previewTheme=data=>{render.theme={appearance:data.appearance,variables:data.variables||{}};post('render',render)};
addEventListener('message',event=>{const d=event.data;if(event.source!==frame.contentWindow||event.origin!=='null'||!d||d.channel!=='whale-openui'||d.version!==1||d.instanceId!==instance)return;
 if(d.type==='ready'){window.__previewState.ready=true;post('render',render);return;}
 if(d.type==='height'){if(Number.isFinite(d.height))window.__previewState.height=Math.max(80,Math.min(2000,Math.ceil(d.height)));return;}
 if(d.type==='api-request'){const keys={'/api/wiki':['title'],'/api/image-search':['q'],'/api/image':['src'],'/api/media':['src'],'/api/html-script':['src','integrity'],'/api/street-route':['coords'],'/api/map-tile':['z','x','y','style']};if(typeof d.requestId!=='string'||!d.requestId||(d.requestId.startsWith('native-action-')||d.requestId.startsWith('media-'))||!keys[d.path]||!d.query||Array.isArray(d.query)||typeof d.query!=='object'||Object.entries(d.query).some(([k,v])=>!keys[d.path].includes(k)||typeof v!=='string')){post('api-response',{requestId:d.requestId,status:400,body:{error:{message:'Unsupported OpenUI API request'}}});return;}if(apiPending.has(d.requestId))return;}
 if(d.type==='clipboard'||d.type==='open-url'){
  const reject=error=>post('action-response',{requestId:d.requestId,action:d.type,ok:false,error});
  if(!render.hostActions){reject('Host actions are disabled for this viewer.');return;}
  if(!Number.isSafeInteger(d.sequence)||d.sequence<1||d.requestId!=='native-action-'+d.sequence||d.userGesture!==true){reject('Invalid Host action request.');return;}
  if(actionPending.has(d.requestId))return;
  if(d.sequence<=lastActionSequence){reject('This Host action was already consumed.');return;}
  if(d.type==='clipboard'&&(typeof d.text!=='string'||d.text.includes('\u0000'))){reject('Invalid clipboard text.');return;}
  if(d.type==='open-url'){let url;try{url=new URL(d.url)}catch{reject('Only absolute HTTP(S) links are supported.');return;}if(typeof d.url!=='string'||/[\u0000-\u0020\u007f-\u009f]/.test(d.url)||!['http:','https:'].includes(url.protocol)||!url.hostname||url.username||url.password){reject('Only absolute HTTP(S) links without embedded credentials are supported.');return;}}
 }
 if(d.type==='media'){
  const reject=error=>post('media-response',{requestId:d.requestId,ok:false,error});
  if(!render.media||render.streaming){reject('Isolated media is unavailable for this viewer.');return;}
  if(!Number.isSafeInteger(d.sequence)||d.sequence<1||d.requestId!=='media-'+d.sequence||typeof d.userGesture!=='boolean'){reject('Invalid media request.');return;}
  if(mediaPending.has(d.requestId))return;
  if(d.sequence<=lastMediaSequence){reject('This media request was already consumed.');return;}
  const m=d.media,keys=['operation','componentId','revision','videoId','title','rect','clip','viewport','visible'];
  if(!m||typeof m!=='object'||Array.isArray(m)||Object.keys(m).some(k=>!keys.includes(k))||!['upsert','remove'].includes(m.operation)||typeof m.componentId!=='string'||m.componentId.length>80||!/^youtube-[1-9][0-9]*$/.test(m.componentId)||m.revision!==render.revision||typeof m.videoId!=='string'||!/^[A-Za-z0-9_-]{11}$/.test(m.videoId)||typeof m.title!=='string'||new TextEncoder().encode(m.title).length>512||typeof m.visible!=='boolean'){reject('Invalid fixed-video media payload.');return;}
  const r=m.rect,c=m.clip,v=m.viewport,box=b=>b&&typeof b==='object'&&!Array.isArray(b)&&['x','y','width','height'].every(k=>Number.isFinite(b[k]));
  if(!box(r)||!box(c)||!v||!Number.isFinite(v.width)||!Number.isFinite(v.height)||v.width<1||v.height<1||v.width>4096||v.height>4096||Math.abs(r.x)>1e6||Math.abs(r.y)>1e6||r.width<0||r.height<0||r.width>4096||r.height>4096||c.x<0||c.y<0||c.width<0||c.height<0||c.x+c.width>v.width+0.5||c.y+c.height>v.height+0.5||(c.width>0&&c.height>0&&(c.x<r.x-0.5||c.y<r.y-0.5||c.x+c.width>r.x+r.width+0.5||c.y+c.height>r.y+r.height+0.5))||(m.visible&&(c.width<=0||c.height<=0))){reject('Invalid player geometry.');return;}
 }
 if(!['state','assistant','error','resolved','api-request','clipboard','open-url','media'].includes(d.type))return;
 let raw;try{raw=JSON.stringify(d)}catch{return}if(new TextEncoder().encode(raw).length>65536){window.__previewState.error='OpenUI event exceeds limit';return;}
 if(d.type==='state'){render.state=d.state;queue=queue.filter(x=>x.type!=='state');}
 if(queue.length>=16){if(d.type==='media')post('media-response',{requestId:d.requestId,ok:false,error:'The existing Host event queue is full; no player request was queued.'});else if(d.type==='clipboard'||d.type==='open-url')post('action-response',{requestId:d.requestId,action:d.type,ok:false,error:'The existing Host event queue is full; nothing was queued.'});else window.__previewState.error='OpenUI event queue overflow';return;}if(d.type==='api-request')apiPending.add(d.requestId);if(d.type==='clipboard'||d.type==='open-url'){lastActionSequence=d.sequence;actionPending.set(d.requestId,d.type);}if(d.type==='media'){lastMediaSequence=d.sequence;mediaPending.set(d.requestId,d.media.operation);}queue.push(d);
});frame.srcdoc=` + string(child) + `;addEventListener('DOMContentLoaded',()=>document.body.append(frame),{once:true});</script><body></body>`, nil
}
