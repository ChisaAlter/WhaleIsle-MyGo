package pluginui

import (
	"encoding/json"
	"os/exec"
	"testing"

	"whaleisle/internal/processutil"
)

// Go leaves an empty Subject name out of the JSON it sends, so the page used to
// receive a main-navigation subject without a name and then died on
// requested.name.startsWith(...): every later snapshot was lost and a selected
// plugin page stayed on "Loading plugin view…" for good.
func TestNavigationSubjectFromGoAlwaysHasAName(t *testing.T) {
	var sent map[string]any
	raw, err := json.Marshal(map[string]any{"type": "select", "subject": Subject{Kind: "navigation"}})
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(raw, &sent); err != nil {
		t.Fatal(err)
	}
	if _, has := sent["subject"].(map[string]any)["name"]; has {
		t.Fatal("this test documents that Go omits the empty name; the page must cope")
	}

	node, err := exec.LookPath("node")
	if err != nil {
		t.Fatal(err)
	}
	source, err := json.Marshal(string(navigationSource))
	if err != nil {
		t.Fatal(err)
	}
	subject, err := json.Marshal(sent["subject"])
	if err != nil {
		t.Fatal(err)
	}
	script := `const vm=require('node:vm'),assert=require('node:assert/strict');
const context=vm.createContext({assert,sent:` + string(subject) + `});
vm.runInContext(` + string(source) + `,context);
vm.runInContext('(' + function(){
 const subject=whaleNavigationSubject(sent);
 assert.equal(subject.kind,'navigation');
 assert.equal(subject.name,'');
 assert.equal(subject.name.startsWith('settings:'),false);
 assert.equal(whaleNavigationSubject({kind:'view',name:'main:a:b'}).name,'main:a:b');
}.toString() + ')()',context);`
	cmd := exec.Command(node, "-e", script)
	processutil.Background(cmd)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("navigation subject: %v\n%s", err, out)
	}
}

func TestUnnamedNavigationDispatchesNativeInputAndPublicActions(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Fatal(err)
	}
	source, err := json.Marshal(string(navigationSource))
	if err != nil {
		t.Fatal(err)
	}
	command, err := json.Marshal(map[string]any{"type": "nativeInput", "revision": 1, "subject": Subject{Kind: "navigation"}, "handle": "action"})
	if err != nil {
		t.Fatal(err)
	}
	script := `const vm=require('node:vm'),assert=require('node:assert/strict');
const tasks=[],sent=[];let socket,inputs=0,menus=0;
const entry={registrant:'fixture-plugin',options:{id:'overlay'}},root={hidden:false,closest:()=>null};
const subscribe=()=>()=>{},entries=slot=>slot==='shell.overlay'?[entry]:[];
const context={URL,URLSearchParams,location:{href:'http://127.0.0.1:1234/?desktopNativeNavigation=1',search:'?desktopNativeNavigation=1'},queueMicrotask:fn=>tasks.push(fn),
 window:{__whaleSessionEntryOf:()=>entry,addEventListener(){},removeEventListener(){}},
 document:{querySelectorAll:()=>[root],addEventListener(){},removeEventListener(){},documentElement:{}},
 WebSocket:class{static OPEN=1;constructor(){socket=this;this.readyState=1;this.handlers={}}addEventListener(name,fn){this.handlers[name]=fn}send(raw){sent.push(JSON.parse(raw))}close(){}},
 MutationObserver:class{observe(){}disconnect(){}},
 whaleWorkspaceBridge:()=>({snapshot:()=>({}),dispose(){}}),
 whaleCreatePublicSlots:()=>({snapshot:()=>({}),dispose(){},invoke(){menus++;return true}}),
 whaleBundledRegistrant:()=>'',whaleStockReady:()=>true,whaleReportedRegistrant:entry=>entry.registrant,
 whaleRichSurface:{bind:()=>()=>{},clear(){},hidden:()=>false},
 whaleAmbientDocument(node,handle,actions,owner){actions.set('action',{node,owner});return [{kind:'rich',richType:'fragment',handle:'action',height:40,width:40,popup:true,ambient:true,vw:800,vh:600}]},
 whaleNativeDocument(node,handle,actions,owner){actions.set('action',{node,owner});return [{kind:'button',handle:'action',label:'Fixture'}]},
 whaleNativeContains:()=>true,whaleNativeInteraction(){inputs++;return true}};
context.ctx={effect:fn=>fn(),slots:{entriesOfSlot:entries,entries},get:()=>null,on:subscribe,locale:{getSnapshot:()=>({active:'en'}),subscribe},layout:{panelInfo:{subscribe}}};
vm.createContext(context);vm.runInContext(` + string(source) + `,context);vm.runInContext('whaleBindNativeNavigation(ctx)',context);
while(tasks.length)tasks.shift()();assert.equal(sent.at(-1).revision,1);
const native=` + string(command) + `;assert.equal(Object.hasOwn(native.subject,'name'),false);
socket.handlers.message({data:JSON.stringify(native)});assert.equal(inputs,1,'native global input rejected omitted empty name');
socket.handlers.message({data:JSON.stringify({...native,type:'publicAction'})});assert.equal(menus,1,'public menu rejected omitted empty name');
for(const subject of [undefined,{kind:'view'},{kind:'navigation',name:'settings:models'}])for(const type of ['nativeInput','publicAction'])socket.handlers.message({data:JSON.stringify({...native,type,subject})});
socket.handlers.message({data:JSON.stringify({...native,revision:0})});
assert.equal(inputs,1,'foreign subject or revision executed input');assert.equal(menus,1,'foreign subject executed menu');`
	cmd := exec.Command(node, "-e", script)
	processutil.Background(cmd)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("navigation action dispatch: %v\n%s", err, out)
	}
}

// Plugins compiled into the host bundle report the shared bundled registrant,
// identical to shipped shell overlays — registrant cannot tell them apart. The
// ambient loop therefore excludes only the known shipped overlay ids; bundled
// entries with any other id (dsh-pad, or a future bundled plugin) project.
func TestAmbientLoopExcludesShippedOverlayIdsOnly(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Fatal(err)
	}
	source, err := json.Marshal(string(navigationSource))
	if err != nil {
		t.Fatal(err)
	}
	script := `const vm=require('node:vm'),assert=require('node:assert/strict');
const tasks=[],sent=[];let socket;const ambientTargets=[];
const padEntry={registrant:'KC',options:{id:'dsh-pad'}},stockEntry={registrant:'KC',options:{id:'chat.quota-notice'}},futureEntry={registrant:'KC',options:{id:'future-bundled-overlay'}};
const padRoot={hidden:false,closest:()=>null},stockRoot={hidden:false,closest:()=>null},futureRoot={hidden:false,closest:()=>null};
const byRoot=new Map([[padRoot,padEntry],[stockRoot,stockEntry],[futureRoot,futureEntry]]);
const subscribe=()=>()=>{},entries=slot=>slot==='shell.overlay'?[padEntry,stockEntry,futureEntry]:[];
const context={URL,URLSearchParams,location:{href:'http://127.0.0.1:1234/?desktopNativeNavigation=1&whalePadOriginal=1',search:'?desktopNativeNavigation=1&whalePadOriginal=1'},queueMicrotask:fn=>tasks.push(fn),
 window:{__whaleSessionEntryOf:(node,slot)=>slot==='shell.overlay'?(byRoot.get(node)||null):null,addEventListener(){},removeEventListener(){}},
 document:{querySelectorAll:()=>[padRoot,stockRoot,futureRoot],addEventListener(){},removeEventListener(){},documentElement:{}},
 WebSocket:class{static OPEN=1;constructor(){socket=this;this.readyState=1;this.handlers={}}addEventListener(name,fn){this.handlers[name]=fn}send(raw){sent.push(JSON.parse(raw))}close(){}},
 MutationObserver:class{observe(){}disconnect(){}},
 whaleWorkspaceBridge:()=>({snapshot:()=>({}),dispose(){}}),
 whaleCreatePublicSlots:()=>({snapshot:()=>({}),dispose(){},invoke(){return true}}),
 whaleBundledRegistrant:()=>'KC',whaleStockReady:()=>true,whaleReportedRegistrant:entry=>entry.registrant==='KC'?'whaleisle:bundled':entry.registrant,
 whaleRichSurface:{bind:()=>()=>{},clear(){},hidden:()=>false},
 whaleAmbientDocument(node,handle,actions,owner){ambientTargets.push(owner.globalEntry);assert.notEqual(owner.globalEntry,stockEntry,'shipped overlay entry reached ambient measurement');return [{kind:'rich',richType:'fragment',handle:'a'+ambientTargets.length,height:40,width:40,popup:true,ambient:true,vw:800,vh:600}]},
 whaleNativeDocument(){return []},
 whaleNativeContains:()=>true,whaleNativeInteraction(){return true}};
context.ctx={effect:fn=>fn(),slots:{entriesOfSlot:entries,entries},get:()=>null,on:subscribe,locale:{getSnapshot:()=>({active:'en'}),subscribe},layout:{panelInfo:{subscribe}}};
vm.createContext(context);vm.runInContext(` + string(source) + `,context);vm.runInContext('whaleBindNativeNavigation(ctx)',context);
while(tasks.length)tasks.shift()();
const snap=sent.filter(m=>m.type==='state').at(-1);
assert.ok(snap,'no snapshot published');
assert.equal(snap.globalDocuments.length,2,'expected dsh-pad and the unknown bundled overlay');
assert.deepEqual(ambientTargets,[padEntry,futureEntry],'shipped overlay ids must be the only bundled exclusions');`
	cmd := exec.Command(node, "-e", script)
	processutil.Background(cmd)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("ambient bundled dsh-pad exemption: %v\n%s", err, out)
	}
}
