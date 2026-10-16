package mygo

import (
	"os/exec"
	"testing"
)

func TestWebShortcutDOMOwnership(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node unavailable for DOM contract test")
	}
	script := `const vm=require('node:vm');const handlers={},queue=[],sent=[];
 const el=(tag,extra={})=>Object.assign({nodeType:1,tagName:tag,matches:()=>false},extra);
 const body=el('BODY');const document={activeElement:body,querySelector:()=>null};
 const window={addEventListener:(n,f)=>{handlers[n]=f},chrome:{webview:{postMessage:m=>sent.push(m)}}};
 const context={window,document,setTimeout:f=>queue.push(f)};vm.createContext(context);
 vm.runInContext(` + "`" + webShortcutScript + "`" + `,context);
 function key(extra={}){const e=Object.assign({ctrlKey:true,metaKey:false,keyCode:188,isTrusted:true,repeat:false,isComposing:false,defaultPrevented:false,composedPath:()=>[body]},extra);handlers.keydown(e);return e}
 function flush(){while(queue.length)queue.shift()()}
 const consumed=key();consumed.defaultPrevented=true;flush();if(sent.length)throw Error('late plugin registry bypassed');
 for(const target of [el('INPUT'),el('TEXTAREA'),el('SELECT'),el('IFRAME'),el('DIV',{isContentEditable:true}),el('DIV',{matches:()=>true})]){document.activeElement=target;key({composedPath:()=>[target]});flush()}
 document.activeElement=body;
 for(const extra of [{isTrusted:false},{isComposing:true},{repeat:true},{keyCode:229}]){key(extra);flush()}
 handlers.compositionstart();key();flush();handlers.compositionend();
 if(sent.length)throw Error('editor/terminal/composition ownership lost');
 key();flush();if(sent.length!==1)throw Error('plain page key not relayed');
 const ack=JSON.parse(sent[0].slice('mygo:web-shortcut:'.length));if(ack.VK!==188||ack.Mods!==1)throw Error('wrong key metadata');`
	if out, err := exec.Command(node, "-e", script).CombinedOutput(); err != nil {
		t.Fatalf("DOM ownership: %v\n%s", err, out)
	}
}
