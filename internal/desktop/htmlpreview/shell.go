package htmlpreview

import (
	"encoding/json"
	"html"
)

// A stricter CSP is inserted before supplied markup. The sandbox never grants
// same-origin, navigation, popups, downloads, or native bridge access. Form
// submission stays blocked by CSP even in the interactive form-handling mode.
const childCSP = "default-src 'none'; script-src 'unsafe-inline' data:; style-src 'unsafe-inline'; img-src data:; font-src data:; connect-src 'none'; frame-src 'none'; worker-src 'none'; object-src 'none'; media-src 'none'; base-uri 'none'; form-action 'none'"
const parentCSP = "default-src 'none'; script-src 'unsafe-inline'; style-src 'unsafe-inline'; frame-src 'self' about:; connect-src 'none'; worker-src 'none'; object-src 'none'; base-uri 'none'; form-action 'none'"

type previewPalette struct{ Background, Surface, Text, Secondary, Border, Accent string }

func buildShell(input request, palette previewPalette) string {
	return buildShellMode(input, palette, false)
}

// Forms are enabled only inside the out-of-process helper. They may run local
// submit handlers, but form-action none and native frame/top-level navigation
// denial remain in force. The screenshot path keeps its existing sandbox.
func buildShellMode(input request, palette previewPalette, interactive bool) string {
	sandbox := "allow-scripts"
	if interactive {
		sandbox += " allow-forms"
	}
	variables, _ := json.Marshal(map[string]string{
		"--bg": palette.Background, "--background": palette.Background, "--surface": palette.Surface,
		"--text": palette.Text, "--text-secondary": palette.Secondary, "--border": palette.Border, "--accent": palette.Accent,
	})
	appearance, _ := json.Marshal(input.Appearance)
	imageReady, imageState := "", ""
	if input.inertImage {
		imageReady = `const image=document.images[0];if(document.images.length!==1||!image?.complete)throw new Error('Image did not decode');let iw=image.naturalWidth,ih=image.naturalHeight;if(iw===0||ih===0){const style=image.getAttribute('style');image.style.setProperty('width','auto','important');image.style.setProperty('height','auto','important');const box=image.getBoundingClientRect();if(style===null)image.removeAttribute('style');else image.setAttribute('style',style);iw=iw||box.width;ih=ih||box.height;}if(!Number.isFinite(iw)||!Number.isFinite(ih)||iw<=0||ih<=0||iw>32768||ih>32768||iw*ih>32000000)throw new Error('Image intrinsic dimensions exceed the preview budget');const intrinsic={imageWidth:iw,imageHeight:ih};`
		imageState = `if(data.type==='preview-ready'){if(!Number.isFinite(data.imageWidth)||!Number.isFinite(data.imageHeight)||data.imageWidth<=0||data.imageHeight<=0||data.imageWidth>32768||data.imageHeight>32768||data.imageWidth*data.imageHeight>32000000){state.error='Invalid image intrinsic dimensions.';return;}state.imageWidth=data.imageWidth;state.imageHeight=data.imageHeight;}`
	}
	readyMessage := `send({type:'preview-ready',height});`
	if input.inertImage {
		readyMessage = `send({type:'preview-ready',height,...intrinsic});`
	}
	child := `<meta http-equiv="Content-Security-Policy" content="` + html.EscapeString(childCSP) + `"><style>html,body{margin:0;background:transparent}*{box-sizing:border-box}</style><script>(()=>{
const send=data=>parent.postMessage(data,'*');
addEventListener('message',event=>{
 if(event.source!==parent||event.data?.type!=='whale-visual-reply-theme')return;
 const data=event.data;if(data.appearance!=='light'&&data.appearance!=='dark')return;
 document.documentElement.style.colorScheme=data.appearance;
 document.documentElement.toggleAttribute('data-ds-dark-theme',data.appearance==='dark');
 for(const [key,value] of Object.entries(data.variables||{}))if(key.startsWith('--')&&typeof value==='string')document.documentElement.style.setProperty(key,value);
});
for(const level of ['log','info','warn','error']){const original=console[level].bind(console);console[level]=(...args)=>{let text;try{text=args.map(x=>String(x)).join(' ')}catch{text='[unprintable console message]'}send({type:'preview-console',level:level==='warn'?'warning':level,text:text.slice(0,500)});original(...args)}}
addEventListener('error',e=>send({type:'preview-console',level:'error',text:String(e.message).slice(0,500)}));
addEventListener('unhandledrejection',e=>send({type:'preview-console',level:'error',text:String(e.reason).slice(0,500)}));
addEventListener('load',async()=>{try{
 document.documentElement.style.colorScheme=` + string(appearance) + `;
 document.documentElement.toggleAttribute('data-ds-dark-theme',` + string(appearance) + `==='dark');
 for(const [key,value] of Object.entries(` + string(variables) + `))document.documentElement.style.setProperty(key,value);
 await document.fonts.ready;
 await Promise.all([...document.images].map(image=>image.decode().catch(()=>{})));
 await new Promise(resolve=>requestAnimationFrame(()=>requestAnimationFrame(resolve)));
 for(const tag of new Set([...document.querySelectorAll('svg *')].filter(el=>el.constructor===SVGElement).map(el=>el.localName)))send({type:'preview-console',level:'warning',text:('HTML preview: Chromium does not recognize SVG tag <'+tag+'>.').slice(0,500)});
 const height=window.__whaleVisualReplyHeight?window.__whaleVisualReplyHeight():Math.max(document.body.scrollHeight,document.documentElement.scrollHeight);
 ` + imageReady + readyMessage + `
}catch(e){send({type:'preview-error',error:String(e).slice(0,500)})}},{once:true});
})();</script>` + input.HTML
	encoded, _ := json.Marshal(child) // escapes '<', so supplied markup cannot close our script
	background, _ := json.Marshal(palette.Background)
	themeBridge := ""
	if interactive {
		themeBridge = `window.__previewTheme=data=>{document.documentElement.style.backgroundColor=data.variables['--background'];frame.contentWindow?.postMessage(data,'*')};frame.addEventListener('load',()=>frame.contentWindow?.postMessage({type:'whale-visual-reply-theme',appearance:` + string(appearance) + `,variables:` + string(variables) + `},'*'));`
	}
	return `<!doctype html><meta http-equiv="Content-Security-Policy" content="` + html.EscapeString(parentCSP) + `"><style>html,body{margin:0;overflow:hidden}iframe{display:block;border:0;width:100%;height:100vh}</style><script>
window.__previewState={ready:false,height:0,error:'',consoleMessages:[],omitted:0};
const frame=document.createElement('iframe');frame.setAttribute('sandbox','` + sandbox + `');frame.setAttribute('referrerpolicy','no-referrer');
addEventListener('message',event=>{
 if(event.source!==frame.contentWindow||event.origin!=='null'||!event.data||typeof event.data!=='object')return;
 const state=window.__previewState,data=event.data;
 if(data.type==='preview-console'&&typeof data.text==='string'){
  if(state.consoleMessages.length<20)state.consoleMessages.push({level:['log','info','warning','error'].includes(data.level)?data.level:'log',text:data.text.slice(0,500)});else state.omitted++;
 }else if(data.type==='preview-ready'||data.type==='whale-visual-reply-size'){
  ` + imageState + `
  if(Number.isFinite(data.height)&&data.height>=1&&data.height<=Number.MAX_SAFE_INTEGER){state.height=Math.ceil(data.height);state.ready=true}else state.error='Page returned an invalid content height.';
 }else if(data.type==='preview-error'&&typeof data.error==='string')state.error=data.error.slice(0,500);
});
document.documentElement.style.backgroundColor=` + string(background) + `;
` + themeBridge + `
addEventListener('DOMContentLoaded',()=>{frame.srcdoc=` + string(encoded) + `;document.body.append(frame)},{once:true});
</script><body></body>`
}
