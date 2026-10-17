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
	initial, _ := json.Marshal(map[string]any{"channel": "whale-openui", "version": 1, "instanceId": r.InstanceID, "type": "render", "program": r.Program, "streaming": r.Streaming, "state": r.State, "theme": map[string]any{"appearance": appearance, "variables": map[string]string{}}})
	id, _ := json.Marshal(r.InstanceID)
	return `<!doctype html><meta http-equiv="Content-Security-Policy" content="` + html.EscapeString(parentPolicy) + `"><style>html,body{margin:0;height:100%;overflow:hidden}iframe{border:0;width:100%;height:100vh}</style><script>
'use strict';const instance=` + string(id) + `,render=` + string(initial) + `;let queue=[];
window.__previewState={ready:false,height:360,error:'',consoleMessages:[],omitted:0};
window.__whaleOpenUIDrain=()=>queue.splice(0,16);
const frame=document.createElement('iframe');frame.name='whale-openui:'+instance;frame.setAttribute('sandbox','allow-scripts allow-forms');frame.referrerPolicy='no-referrer';
const post=(type,data={})=>frame.contentWindow.postMessage({channel:'whale-openui',version:1,instanceId:instance,type,...data},'*');
window.__previewTheme=data=>{render.theme={appearance:data.appearance,variables:data.variables||{}};post('render',render)};
addEventListener('message',event=>{const d=event.data;if(event.source!==frame.contentWindow||event.origin!=='null'||!d||d.channel!=='whale-openui'||d.version!==1||d.instanceId!==instance)return;
 if(d.type==='ready'){window.__previewState.ready=true;post('render',render);return;}
 if(d.type==='height'){if(Number.isFinite(d.height))window.__previewState.height=Math.max(80,Math.min(2000,Math.ceil(d.height)));return;}
 if(d.type==='api-request'){post('api-response',{requestId:d.requestId,status:503,body:{error:'External resources are disabled in this isolated OpenUI viewer.'}});return;}
 if(d.type==='clipboard'){post('clipboard-response',{requestId:d.requestId,ok:false,error:'Clipboard access is disabled in this isolated viewer.'});return;}
 if(d.type==='open-url'){return;}
 if(!['state','assistant','error','resolved'].includes(d.type))return;
 let raw;try{raw=JSON.stringify(d)}catch{return}if(new TextEncoder().encode(raw).length>65536){window.__previewState.error='OpenUI event exceeds limit';return;}
 if(d.type==='state'){render.state=d.state;queue=queue.filter(x=>x.type!=='state');}
 if(queue.length>=16){window.__previewState.error='OpenUI event queue overflow';return;}queue.push(d);
});frame.srcdoc=` + string(child) + `;addEventListener('DOMContentLoaded',()=>document.body.append(frame),{once:true});</script><body></body>`, nil
}
