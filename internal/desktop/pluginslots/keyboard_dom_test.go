package pluginslots

import (
	"os/exec"
	"testing"
)

func TestProductKeyboardOwnerContract(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node unavailable")
	}
	script := `const vm=require('node:vm'),sent=[],delivered=[],handlers={},subscriptions=[];
 let focused=true,editing=false,revision='original',rows=[{binding:{code:'KeyP',modifiers:['control','shift']},issue:null,conflicts:[]}];
 class Socket {static OPEN=1;constructor(){this.readyState=1;this.handlers={};globalThis.sock=this}send(raw){sent.push(JSON.parse(raw))}addEventListener(n,f){this.handlers[n]=f}close(){}}
 const target={closest:()=>editing?{}:null};const document={hasFocus:()=>focused,activeElement:target,querySelector:()=>null,documentElement:{},addEventListener:(n,f)=>handlers[n]=f,removeEventListener(){}};
 const window={addEventListener:(n,f)=>handlers[n]=f,removeEventListener(){},dispatchEvent:e=>delivered.push(e)};window.top=window;
 const owner={catalog:{getSnapshot:()=>rows,subscribe:f=>{subscriptions.push(f);return()=>{}}},config:{getSnapshot:()=>({revision,status:'ready'}),subscribe:()=>()=>{}},recording:async()=>{}};
 const ctx={shortcuts:owner,effect:f=>f()};
 const context={window,document,WebSocket:Socket,location:{href:'http://127.0.0.1:1234/'},URL,KeyboardEvent:class{constructor(type,data){Object.assign(this,data);this.type=type}},ctx};vm.createContext(context);
 vm.runInContext(` + "`" + string(keyboardSource) + "`" + `,context);vm.runInContext('whaleBindProductKeyboard(ctx)',context);sock.handlers.open();
 let state=sent.at(-1);if(!state.active||state.bindings.length!==1)throw Error('accepted registry absent');
 const sendKey=(r=state.revision)=>sock.handlers.message({data:JSON.stringify({type:'key',revision:r,code:'KeyP',mods:5})});
 sendKey();if(delivered.length!==1)throw Error('native input did not reach existing DOM dispatcher');
 sendKey('stale');editing=true;sendKey();editing=false;handlers.compositionstart();sendKey();handlers.compositionend();
 if(delivered.length!==1)throw Error('stale/editor/composition command ran');
 rows=[{binding:{code:'KeyA',secondCode:'KeyB',modifiers:[]},issue:null,conflicts:[]}];subscriptions[0]();state=sent.at(-1);
 if(state.bindings[0].secondCode!=='KeyB')throw Error('held pair dropped');
 sock.handlers.message({data:JSON.stringify({type:'key',revision:state.revision,code:'KeyA',secondCode:'KeyB',mods:0})});
 if(delivered.at(-1).whaleisleSecondCode!=='KeyB')throw Error('native pair lost before DOM');
 const physical=(code,type='keydown',repeat=false)=>{const e={code,type,repeat,isTrusted:true,preventDefault(){this.prevented=true}};handlers[type](e);return e};
 if(physical('KeyB').whaleisleSecondCode)throw Error('first key consumed');
 if(physical('KeyA').whaleisleSecondCode!=='KeyB')throw Error('reversed held pair lost');
 if(!physical('KeyB','keydown',true).prevented)throw Error('pair repeated');
 physical('KeyA','keyup');physical('KeyB','keyup');physical('KeyA');physical('KeyA','keyup');
 if(physical('KeyB').whaleisleSecondCode)throw Error('sequence became held pair');
 subscriptions[0]();physical('KeyA');handlers.compositionstart();handlers.compositionend();
 if(physical('KeyB').whaleisleSecondCode)throw Error('composition kept pending pair');
 rows=[];subscriptions[0]();if(sent.at(-1).bindings.length)throw Error('unregistered command retained');`
	if out, err := runNodeFixture(node, "-e", script); err != nil {
		t.Fatalf("product contract: %v\n%s", err, out)
	}
}
