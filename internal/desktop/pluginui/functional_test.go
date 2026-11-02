//go:build functionalacceptance

package pluginui_test

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"whaleisle/internal/desktop/pluginslots"
	"whaleisle/internal/desktop/pluginui"
	"whaleisle/internal/harness"
	"whaleisle/internal/processutil"
)

// Runs an owned dsh web composition and independent headless Chrome. Installation
// and reference files remain read-only; all plugin/profile writes are temporary.
func TestRealHostPluginContributions(t *testing.T) {
	stage := os.Getenv("WHALEISLE_PLUGIN_UI_RUNTIME")
	if !filepath.IsAbs(stage) {
		t.Fatal("explicit absolute runtime stage required")
	}
	root := t.TempDir()
	if out := os.Getenv("WHALEISLE_PLUGIN_UI_OUTPUT"); out != "" {
		if err := os.MkdirAll(out, 0700); err != nil {
			t.Fatal(err)
		}
		var err error
		root, err = os.MkdirTemp(out, "run-")
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("owned fixture output: %s", root)
	}
	userData := filepath.Join(root, "user-data")
	home := filepath.Join(userData, "dsh-home")
	profile := filepath.Join(home, "profiles", "web")
	t.Setenv("WHALEISLE_USER_DATA", userData)
	t.Setenv("DSH_HOME", home)
	node := filepath.Join(stage, "node", "node.exe")
	vendor := filepath.Join(stage, "vendor")
	harnessRoot := filepath.Join(vendor, "deepseek-harness")
	fixture := filepath.Join(root, "fixture")
	if err := os.MkdirAll(fixture, 0700); err != nil {
		t.Fatal(err)
	}
	manifest := `{"name":"whaleisle-plugin-ui-fixture","version":"1.0.0","type":"module","main":"index.js","exports":{".":"./index.js","./client":"./client.js","./package.json":"./package.json"},"dsh":{"client":{"platform":"web","inject":["@deepseek-ai/dsh-client-ui-plugin-manager"]}}}`
	for name, text := range map[string]string{"package.json": manifest, "index.js": "export function apply() {}\n", "client.js": realFixtureClient} {
		if err := os.WriteFile(filepath.Join(fixture, name), []byte(text), 0600); err != nil {
			t.Fatal(err)
		}
	}
	patch := filepath.Join(root, "fixture.patch.yml")
	if err := os.WriteFile(patch, []byte(fmt.Sprintf("- insert:\n    - id: native-plugin-ui-fixture\n      name: %q\n", filepath.ToSlash(filepath.Join(fixture, "index.js")))), 0600); err != nil {
		t.Fatal(err)
	}
	sup := harness.New(harness.Options{Launch: harness.LaunchOptions{NodeBin: node, HarnessRoot: harnessRoot, VendorDir: vendor, Home: home, Host: "127.0.0.1", Port: 37130, PatchFiles: []string{patch}, ExtraEnv: []string{"npm_config_offline=true", "COREPACK_ENABLE_NETWORK=0"}}, Prepare: harness.PrepareOptions{VendorDir: vendor, HarnessRoot: harnessRoot, DshHome: home, ProfileDir: profile, RemoteSet: true, Remote: false}, ReadyTimeout: 75 * time.Second})
	sup.Start()
	defer sup.Stop()
	deadline := time.Now().Add(80 * time.Second)
	for sup.Status().State != harness.Ready {
		if sup.Status().State == harness.Failed {
			t.Fatalf("owned host failed: %v\n%v", sup.Status().Err, sup.Logs())
		}
		if time.Now().After(deadline) {
			t.Fatalf("owned host timeout\n%v", sup.Logs())
		}
		time.Sleep(25 * time.Millisecond)
	}
	endpoint := sup.Status().Endpoint
	// The test drives the same authenticated extension route through its public
	// browser page, without creating a native application window.
	owner, handler := pluginui.NewBridgeFixture()
	gateway, err := pluginslots.NewGatewayWithHandler(endpoint.Origin, endpoint.Cookie, handler)
	if err != nil {
		t.Fatal(err)
	}
	defer gateway.Close()
	defer owner.Close()
	browserEntry, err := gateway.EntryURL(endpoint.Origin + "/?desktopNativePlugins=1")
	if err != nil {
		t.Fatal(err)
	}
	script := filepath.Join(root, "browser.mjs")
	if err := os.WriteFile(script, []byte(realBrowserScript), 0600); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(node, script, browserEntry, root)
	processutil.Background(cmd)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("real contribution browser: %v\n%s", err, output)
	}
	t.Log(string(output))
	if owner.Snapshot().Connected {
		t.Fatal("closed real browser retained bridge authority")
	}
}

const realFixtureClient = `window.__ModuleLoader__.load({id:'whaleisle-plugin-ui-fixture',factory(require){const R=require('react'),h=R.createElement;return {inject:['slots','locale'],apply(ctx){let dispose;ctx.effect(()=>{dispose=ctx.slots.inject('plugins.item',function*(){yield ctx.slots.register({name:'plugins.item',id:'fixture-official',order:3,label:()=>ctx.locale.getSnapshot().active==='zh'?'动态配置':'Dynamic configuration'},function(p){if(p.view==='summary')return 'Live fixture summary';return h('div',null,h('strong',null,'Owner form revision '+(p.form?.state.revision??'none')),h('button',{onClick:()=>{document.documentElement.dataset.fixtureConfigured='yes'}},'Fixture configure'))});yield ctx.slots.register({name:'plugins.detail.actions',id:'fixture-action'},function(p){return p.subject.kind==='item'&&p.subject.id==='fixture-official'?h('button',{onClick:()=>{document.documentElement.dataset.fixtureAction='yes'}},'Fixture action'):null});yield ctx.slots.register({name:'plugins.detail.badge',id:'fixture-badge'},function(p){return p.subject.kind==='item'&&p.subject.id==='fixture-official'?h('span',null,'Fixture badge'):null});yield ctx.slots.register({name:'plugins.detail.section',id:'fixture-section'},function(p){return p.subject.kind==='item'&&p.subject.id==='fixture-official'?h('section',null,'Fixture section',h('button',{onClick:()=>{document.documentElement.dataset.fixtureSection='yes'}},'Section operation')):null});;yield ctx.slots.register({name:'plugins.bundle.activation',key:'@deepseek-ai/dsh-experimental-voice-input-bundle'},function(p){return h('section',null,'Fixture activation guidance',h('button',{onClick:p.onDismiss},'Dismiss fixture guidance'),h('button',{onClick:p.onOpenDetails},'Open fixture details'))});});window.__fixtureWithdraw=()=>dispose();return()=>{dispose();delete window.__fixtureWithdraw}})}}}});`

const realBrowserScript = `import {spawn} from 'node:child_process';import{mkdirSync,writeFileSync}from'node:fs';import path from'node:path';import assert from'node:assert/strict';
const[entry,root]=process.argv.slice(2),profile=path.join(root,'chrome');mkdirSync(profile,{recursive:true});
const port=19300+process.pid%600,child=spawn('C:/Program Files/Google/Chrome/Application/chrome.exe',['--headless=new','--disable-gpu','--no-first-run','--remote-debugging-address=127.0.0.1','--remote-debugging-port='+port,'--user-data-dir='+profile,'about:blank'],{stdio:'ignore',windowsHide:true});
const delay=ms=>new Promise(r=>setTimeout(r,ms));let ws;try{let target;for(let i=0;i<150;i++){try{target=(await(await fetch('http://127.0.0.1:'+port+'/json/list')).json()).find(t=>t.type==='page');if(target)break}catch{}await delay(100)}assert(target,'Chrome CDP unavailable');ws=new WebSocket(target.webSocketDebuggerUrl);await new Promise((r,j)=>{ws.addEventListener('open',r);ws.addEventListener('error',j)});let id=0;const pending=new Map;ws.addEventListener('message',e=>{const m=JSON.parse(e.data);if(pending.has(m.id)){const{r,j}=pending.get(m.id);pending.delete(m.id);m.error?j(Error(JSON.stringify(m.error))):r(m.result)}});const send=(method,params={})=>new Promise((r,j)=>{const n=++id;pending.set(n,{r,j});ws.send(JSON.stringify({id:n,method,params}))});await send('Page.enable');
const evaluate=async expression=>{const r=await send('Runtime.evaluate',{expression,returnByValue:true,awaitPromise:true});if(r.exceptionDetails)throw Error(r.exceptionDetails.exception?.description||r.exceptionDetails.text);return r.result.value};
await send('Page.addScriptToEvaluateOnNewDocument',{source:"window.__pluginUISnapshots=[];const NativeSocket=WebSocket;window.WebSocket=new Proxy(NativeSocket,{construct(T,a){const socket=Reflect.construct(T,a);if(a[1]==='whaleisle.plugin-ui.v1'){window.__pluginUISocket=socket;socket.addEventListener('message',e=>{const m=JSON.parse(e.data);if(m.type==='select')window.__initialPluginSelection=true});const send=socket.send.bind(socket);socket.send=raw=>{window.__pluginUISnapshots.push(JSON.parse(raw));send(raw)}}return socket}});window.__emitPlugins=m=>window.__pluginUISocket.dispatchEvent(new MessageEvent('message',{data:JSON.stringify(m)}));"});
await send('Page.navigate',{url:entry});
const wait=async(expr,why)=>{for(let i=0;i<250;i++){if(await evaluate(expr))return;await delay(100)}writeFileSync(path.join(root,'failure.json'),JSON.stringify(await evaluate('({body:document.body.innerText,html:document.documentElement.outerHTML,snapshots:window.__pluginUISnapshots})'),null,2));throw Error(why)};
await wait('window.__initialPluginSelection&&window.__pluginUISnapshots?.some(s=>s.entries.some(e=>e.id==="fixture-official"))&&!!document.querySelector("[data-plugin-item=fixture-official]")','real fixture slot absent');
await evaluate('window.__emitPlugins({type:"select",subject:{kind:"item",itemId:"fixture-official"}})');
await wait('window.__pluginUISnapshots.at(-1)?.subject?.itemId==="fixture-official"&&window.__pluginUISnapshots.at(-1).actions.length===1','item owner/navigation/actions missing');
const state=await evaluate('window.__pluginUISnapshots.at(-1)');assert(state.entries.some(e=>e.label==='Dynamic configuration'||e.label==='动态配置'));assert.deepEqual(state.badges,['Fixture badge']);assert(state.section&&state.configuration);
await evaluate('window.__emitPlugins({type:"surface",slot:"plugins.item"})');await wait('window.__pluginUISnapshots.at(-1).surface==="plugins.item"','config surface missing');
assert.equal(await evaluate('getComputedStyle(document.querySelector("[data-whale-plugin-surface]")).position'),'fixed');
await evaluate('[...document.querySelectorAll("[data-whale-plugin-surface] button")].find(b=>b.textContent==="Fixture configure").click()');assert.equal(await evaluate('document.documentElement.dataset.fixtureConfigured'),'yes');
const current=await evaluate('window.__pluginUISnapshots.at(-1)');
await evaluate('window.__emitPlugins('+JSON.stringify({type:'click',revision:current.revision,handle:current.actions[0].handle})+')');assert.equal(await evaluate('document.documentElement.dataset.fixtureAction'),'yes');
await evaluate('delete document.documentElement.dataset.fixtureAction;window.__emitPlugins('+JSON.stringify({type:'click',revision:current.revision-1,handle:current.actions[0].handle})+')');assert.equal(await evaluate('document.documentElement.dataset.fixtureAction||null'),null);
await evaluate('window.__emitPlugins({type:"surface",slot:"plugins.detail.section"})');await wait('window.__pluginUISnapshots.at(-1).surface==="plugins.detail.section"','section surface missing');
await evaluate('[...document.querySelectorAll("[data-whale-plugin-surface] button")].find(b=>b.textContent==="Section operation").click()');assert.equal(await evaluate('document.documentElement.dataset.fixtureSection'),'yes');
await evaluate('window.__emitPlugins({type:"select",subject:{kind:"list"}})');await wait('!!document.querySelector("[data-plugin-package=\\"@deepseek-ai/dsh-experimental-voice-input-bundle\\"] [role=switch]")','activation bundle card missing');
const activationState=await evaluate('window.__pluginUISnapshots.at(-1)');await evaluate('window.__emitPlugins('+JSON.stringify({type:'enableActivation',revision:activationState.revision,name:'@deepseek-ai/dsh-experimental-voice-input-bundle'})+')');
await wait('window.__pluginUISnapshots.at(-1).surface==="plugins.bundle.activation"&&!!document.querySelector("[data-whale-plugin-surface]")','original post-enable activation owner missing');
assert(await evaluate('document.querySelector("[data-whale-plugin-surface]").textContent.includes("Fixture activation guidance")'));
await evaluate('[...document.querySelectorAll("[data-whale-plugin-surface] button")].find(b=>b.textContent==="Open fixture details").click()');
await wait('!!document.querySelector("[data-plugin-detail=\\"@deepseek-ai/dsh-experimental-voice-input-bundle\\"]")','activation onOpenDetails lost original callback');
await evaluate('window.__fixtureWithdraw()');await wait('!window.__pluginUISnapshots.at(-1).entries.some(e=>e.id==="fixture-official")&&!window.__pluginUISnapshots.at(-1).actions.length','withdrawal retained authority');
await evaluate('window.__emitPlugins({type:"select",subject:{kind:"item",itemId:"agent-loop"}})');await wait('window.__pluginUISnapshots.at(-1).subject?.itemId==="agent-loop"&&!!document.querySelector("#plugin-config-agent-loop-parallel")','original settings form missing');
await evaluate('window.__emitPlugins({type:"surface",slot:"plugins.item"})');await wait('window.__pluginUISnapshots.at(-1).surface==="plugins.item"','original settings surface missing');
const before=await evaluate('document.querySelector("#plugin-config-agent-loop-parallel").value');const after=before==='2'?'3':'2';
const setInput=async value=>evaluate('(()=>{const input=document.querySelector("#plugin-config-agent-loop-parallel");Object.getOwnPropertyDescriptor(HTMLInputElement.prototype,"value").set.call(input,'+JSON.stringify(value)+');input.dispatchEvent(new Event("input",{bubbles:true}));input.dispatchEvent(new Event("change",{bubbles:true}));})()');
const save='[...document.querySelectorAll("[data-whale-plugin-surface] button")].find(b=>["Save","保存"].includes(b.textContent.trim()))';
await setInput(after);await wait('!!('+save+')&&!('+save+').disabled','edited form not writable');await evaluate('('+save+').click()');await wait('!!('+save+')&&('+save+').disabled','original Host did not accept save');assert.equal(await evaluate('document.querySelector("#plugin-config-agent-loop-parallel").value'),after);
await setInput(before);await wait('!!('+save+')&&!('+save+').disabled','restore edit not writable');await evaluate('('+save+').click()');await wait('!!('+save+')&&('+save+').disabled','original Host did not accept restore');
writeFileSync(path.join(root,'result.json'),JSON.stringify({passed:['real Cordis registration','dynamic metadata','original item detail subject','detail actions and stale revision refusal','badge','local config DOM interaction','local section interaction','slot withdrawal','original list enable and activation onOpenDetails','original revision-aware Host settings save and restore'],last:await evaluate('window.__pluginUISnapshots.at(-1)')},null,2));console.log('real Host + Chrome contribution fixture passed');}finally{ws?.close();child.kill();await delay(300)}
`
