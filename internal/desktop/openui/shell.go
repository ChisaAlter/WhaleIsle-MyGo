package openui

import (
	"encoding/json"
	"errors"
	"html"
	"strings"
	"unicode/utf8"
)

const childPolicy = "default-src 'none'; script-src 'unsafe-inline' data:; style-src 'unsafe-inline'; img-src data:; font-src data:; connect-src 'none'; frame-src 'none'; worker-src 'none'; object-src 'none'; media-src 'none'; base-uri 'none'; form-action 'none'"
const parentPolicy = "default-src 'none'; script-src 'unsafe-inline'; style-src 'unsafe-inline'; frame-src 'self' about:; connect-src 'none'; worker-src 'none'; object-src 'none'; base-uri 'none'; form-action 'none'"

func Shell(viewerHTML string, r Render, width int, appearance string) (string, error) {
	if err := ValidateRender(r); err != nil {
		return "", err
	}
	if len(viewerHTML) > MaxViewerBytes || !utf8.ValidString(viewerHTML) || strings.TrimSpace(viewerHTML) == "" || width < 240 || width > 1600 || (appearance != "light" && appearance != "dark") {
		return "", errors.New("invalid OpenUI viewer parameters")
	}
	child, _ := json.Marshal(`<meta http-equiv="Content-Security-Policy" content="` + html.EscapeString(childPolicy) + `">` + viewerHTML)
	initial, _ := json.Marshal(map[string]any{"channel": "whale-openui", "version": 1, "instanceId": r.InstanceID, "type": "render", "program": r.Program, "streaming": r.Streaming, "state": r.State, "htmlView": false, "hostActions": r.HostActions && IsOwnedViewer(viewerHTML), "theme": map[string]any{"appearance": appearance, "variables": map[string]string{}}})
	id, _ := json.Marshal(r.InstanceID)
	return `<!doctype html><meta http-equiv="Content-Security-Policy" content="` + html.EscapeString(parentPolicy) + `"><style>html,body{margin:0;height:100%;overflow:hidden}iframe{border:0;width:100%;height:100vh}</style><script>
'use strict';const instance=` + string(id) + `,render=` + string(initial) + `;let queue=[];const apiPending=new Set(),actionPending=new Map();let lastActionSequence=0;
window.__previewState={ready:false,height:360,error:'',consoleMessages:[],omitted:0};
window.__whaleOpenUIDrain=()=>queue.splice(0,16);
const frame=document.createElement('iframe');frame.name='whale-openui:'+instance;frame.setAttribute('sandbox','allow-scripts allow-forms');frame.referrerPolicy='no-referrer';
const post=(type,data={})=>frame.contentWindow.postMessage({channel:'whale-openui',version:1,instanceId:instance,type,...data},'*');
// Only trusted native Eval calls this setter after actual frame guards exist.
// No descendant message is accepted as a capability grant.
window.__whaleOpenUIHtmlViewSupport=enabled=>{if(typeof enabled!=='boolean')throw Error('Invalid HtmlView capability');render.htmlView=enabled;post('render',render);return true};
window.__whaleOpenUIRender=next=>{if(!next||next.instanceId!==instance||typeof next.program!=='string'||typeof next.streaming!=='boolean'||new TextEncoder().encode(next.program).length>262144)throw Error('Invalid OpenUI update');render.program=next.program;render.streaming=next.streaming;post('render',render);return true};
window.__whaleOpenUIAPIResponse=r=>{
 if(!r||r.instanceId!==instance||typeof r.requestId!=='string'||(!apiPending.has(r.requestId)&&!actionPending.has(r.requestId))||!Number.isInteger(r.status)||r.status<200||r.status>599)throw Error('Invalid OpenUI API response');
 const action=actionPending.get(r.requestId);
 if(action){actionPending.delete(r.requestId);const body=r.body,ok=r.status<300&&body&&typeof body==='object'&&!Array.isArray(body)&&body.ok===true;let error=body&&typeof body.error==='string'?body.error:'The Host action was not completed.';if(new TextEncoder().encode(error).length>4096)error='The Host action returned an oversized error.';post('action-response',{requestId:r.requestId,action,ok:!!ok,...(ok?{}:{error})});return true;}
 apiPending.delete(r.requestId);post('api-response',{requestId:r.requestId,status:r.status,body:r.body});return true
};
window.__previewTheme=data=>{render.theme={appearance:data.appearance,variables:data.variables||{}};post('render',render)};
addEventListener('message',event=>{const d=event.data;if(event.source!==frame.contentWindow||event.origin!=='null'||!d||d.channel!=='whale-openui'||d.version!==1||d.instanceId!==instance)return;
 if(d.type==='ready'){window.__previewState.ready=true;post('render',render);return;}
 if(d.type==='height'){if(Number.isFinite(d.height))window.__previewState.height=Math.max(80,Math.min(2000,Math.ceil(d.height)));return;}
 if(d.type==='api-request'){const keys={'/api/wiki':['title'],'/api/image-search':['q'],'/api/street-route':['coords'],'/api/map-tile':['z','x','y','style']};if(typeof d.requestId!=='string'||!d.requestId||d.requestId.startsWith('native-action-')||!keys[d.path]||!d.query||Array.isArray(d.query)||typeof d.query!=='object'||Object.entries(d.query).some(([k,v])=>!keys[d.path].includes(k)||typeof v!=='string')){post('api-response',{requestId:d.requestId,status:400,body:{error:{message:'Unsupported OpenUI API request'}}});return;}if(apiPending.has(d.requestId))return;}
 if(d.type==='clipboard'||d.type==='open-url'){
  const reject=error=>post('action-response',{requestId:d.requestId,action:d.type,ok:false,error});
  if(!render.hostActions){reject('Host actions are disabled for this viewer.');return;}
  if(!Number.isSafeInteger(d.sequence)||d.sequence<1||d.requestId!=='native-action-'+d.sequence||d.userGesture!==true){reject('Invalid Host action request.');return;}
  if(actionPending.has(d.requestId))return;
  if(d.sequence<=lastActionSequence){reject('This Host action was already consumed.');return;}
  if(d.type==='clipboard'&&(typeof d.text!=='string'||d.text.includes('\u0000'))){reject('Invalid clipboard text.');return;}
  if(d.type==='open-url'){let url;try{url=new URL(d.url)}catch{reject('Only absolute HTTP(S) links are supported.');return;}if(typeof d.url!=='string'||/[\u0000-\u0020\u007f-\u009f]/.test(d.url)||!['http:','https:'].includes(url.protocol)||!url.hostname||url.username||url.password){reject('Only absolute HTTP(S) links without embedded credentials are supported.');return;}}
 }
 if(!['state','assistant','error','resolved','api-request','clipboard','open-url'].includes(d.type))return;
 let raw;try{raw=JSON.stringify(d)}catch{return}if(new TextEncoder().encode(raw).length>65536){window.__previewState.error='OpenUI event exceeds limit';return;}
 if(d.type==='state'){render.state=d.state;queue=queue.filter(x=>x.type!=='state');}
 if(queue.length>=16){if(d.type==='clipboard'||d.type==='open-url')post('action-response',{requestId:d.requestId,action:d.type,ok:false,error:'The existing Host event queue is full; nothing was queued.'});else window.__previewState.error='OpenUI event queue overflow';return;}if(d.type==='api-request')apiPending.add(d.requestId);if(d.type==='clipboard'||d.type==='open-url'){lastActionSequence=d.sequence;actionPending.set(d.requestId,d.type);}queue.push(d);
});frame.srcdoc=` + string(child) + `;addEventListener('DOMContentLoaded',()=>document.body.append(frame),{once:true});</script><body></body>`, nil
}
