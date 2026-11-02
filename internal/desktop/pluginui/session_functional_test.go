//go:build functionalacceptance

package pluginui_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"whaleisle/internal/desktop/pluginslots"
	"whaleisle/internal/desktop/pluginui"
	"whaleisle/internal/dsh"
	"whaleisle/internal/harness"
	"whaleisle/internal/processutil"
)

// A Loader-loaded contribution uses the real renderer Session scope and real
// Session.rename. The test-only HTTP driver calls production Owner methods;
// neither the browser bridge nor Session object is replaced.
func TestRealHostSessionPluginSeat(t *testing.T) {
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
		root, err = os.MkdirTemp(out, "session-")
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("owned Session fixture output: %s", root)
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
	manifest := `{"name":"whaleisle-session-ui-fixture","version":"1.0.0","type":"module","main":"index.js","exports":{".":"./index.js","./client":"./client.js","./package.json":"./package.json"},"dsh":{"client":{"platform":"web","inject":["@deepseek-ai/dsh-client-ui-conversation"]}}}`
	for name, text := range map[string]string{"package.json": manifest, "index.js": "export function apply() {}\n", "client.js": sessionFixtureClient} {
		if err := os.WriteFile(filepath.Join(fixture, name), []byte(text), 0600); err != nil {
			t.Fatal(err)
		}
	}
	patch := filepath.Join(root, "fixture.patch.yml")
	if err := os.WriteFile(patch, []byte(fmt.Sprintf("- insert:\n    - id: session-plugin-fixture\n      name: %q\n", filepath.ToSlash(filepath.Join(fixture, "index.js")))), 0600); err != nil {
		t.Fatal(err)
	}
	sup := harness.New(harness.Options{Launch: harness.LaunchOptions{NodeBin: node, HarnessRoot: harnessRoot, VendorDir: vendor, Home: home, Host: "127.0.0.1", Port: 37136, PatchFiles: []string{patch}, ExtraEnv: []string{"npm_config_offline=true", "COREPACK_ENABLE_NETWORK=0"}}, Prepare: harness.PrepareOptions{VendorDir: vendor, HarnessRoot: harnessRoot, DshHome: home, ProfileDir: profile, RemoteSet: true, Remote: false}, ReadyTimeout: 75 * time.Second})
	sup.Start()
	defer sup.Stop()
	deadline := time.Now().Add(80 * time.Second)
	for sup.Status().State != harness.Ready {
		if sup.Status().State == harness.Failed || time.Now().After(deadline) {
			t.Fatalf("owned Host failed: %v\n%v", sup.Status().Err, sup.Logs())
		}
		time.Sleep(25 * time.Millisecond)
	}
	ep := sup.Status().Endpoint
	client := &dsh.Client{Endpoint: dsh.Endpoint{Origin: ep.Origin, Cookie: ep.Cookie}}
	for _, sid := range []string{"fixture-session-a", "fixture-session-b"} {
		if err := client.Call(context.Background(), "session/create", map[string]any{"request": map[string]any{"sessionId": sid, "presentation": map[string]any{"owner": "fixture", "title": "Fixture Session"}}}, nil); err != nil {
			t.Fatal(err)
		}
	}
	owner, bridge := pluginui.NewBridgeFixture()
	defer owner.Close()
	owner.Select(pluginui.Subject{Kind: "session", SessionID: "fixture-session-a", Epoch: "1:1"})
	var observing, unknownOnboarding atomic.Bool
	observing.Store(true)
	handler := func(w http.ResponseWriter, r *http.Request) bool {
		if r.URL.Path != "/__whale_plugin_ui/fixture" {
			return bridge(w, r)
		}
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodGet {
			s := owner.Snapshot()
			json.NewEncoder(w).Encode(map[string]any{"page": s.Page, "state": s})
			return true
		}
		var m struct {
			Type           string
			Page, Revision uint64
			Subject        pluginui.Subject
			Handle         string
		}
		if json.NewDecoder(http.MaxBytesReader(w, r.Body, 16384)).Decode(&m) != nil {
			http.Error(w, "bad request", 400)
			return true
		}
		var err error
		switch m.Type {
		case "select":
			owner.Select(m.Subject)
		case "show":
			err = owner.ShowSessionOutlet(m.Page, m.Revision, m.Subject, m.Handle)
		case "click":
			err = owner.Click(m.Page, m.Revision, m.Subject, m.Handle)
		case "sessionClick":
			err = owner.ClickSessionControl(m.Page, m.Revision, m.Subject, m.Handle)
		case "page":
			err = owner.ShowSessionPage(m.Page, m.Revision, m.Subject)
		case "collapse":
			owner.Show("")
		case "observationOff":
			observing.Store(false)
		case "unknownOnboarding":
			unknownOnboarding.Store(true)
		default:
			http.Error(w, "bad command", 400)
			return true
		}
		if err != nil {
			w.WriteHeader(http.StatusConflict)
			json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		} else {
			json.NewEncoder(w).Encode(map[string]bool{"ok": true})
		}
		return true
	}
	gateway, err := pluginslots.NewGatewayWithResponseFilter(ep.Origin, ep.Cookie, handler, func(r *http.Response) error {
		if !observing.Load() && r.Request != nil {
			r.Request.Header.Set("X-Whale-Test-No-Entry-Observation", "1")
		}
		if unknownOnboarding.Load() && r.Request != nil {
			r.Request.Header.Set("X-Whale-Test-Unknown-Onboarding", "1")
		}
		return pluginui.SessionRendererResponseFixture(owner, r)
	})
	if err != nil {
		t.Fatal(err)
	}
	defer gateway.Close()
	entry, err := gateway.EntryURL(ep.Origin + "/?desktopNativeSession=1")
	if err != nil {
		t.Fatal(err)
	}
	script := filepath.Join(root, "session-browser.mjs")
	if err := os.WriteFile(script, []byte(sessionBrowserScript), 0600); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(node, script, entry, root)
	processutil.Background(cmd)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("real Session browser: %v\n%s", err, out)
	}
	t.Log(string(out))
}

const sessionBrowserScript = `import{spawn}from'node:child_process';import{mkdirSync,writeFileSync}from'node:fs';import path from'node:path';import assert from'node:assert/strict';
const[entry,root]=process.argv.slice(2),profile=path.join(root,'chrome');mkdirSync(profile,{recursive:true});const port=19400+process.pid%600,child=spawn('C:/Program Files/Google/Chrome/Application/chrome.exe',['--headless=new','--disable-gpu','--no-first-run','--remote-debugging-address=127.0.0.1','--remote-debugging-port='+port,'--user-data-dir='+profile,'about:blank'],{stdio:'ignore',windowsHide:true});const delay=ms=>new Promise(r=>setTimeout(r,ms));let ws;
try{let target;for(let i=0;i<150;i++){try{target=(await(await fetch('http://127.0.0.1:'+port+'/json/list')).json()).find(t=>t.type==='page');if(target)break}catch{}await delay(100)}assert(target);ws=new WebSocket(target.webSocketDebuggerUrl);await new Promise((r,j)=>{ws.addEventListener('open',r);ws.addEventListener('error',j)});let id=0;const pending=new Map;ws.addEventListener('message',e=>{const m=JSON.parse(e.data);if(pending.has(m.id)){const{r,j}=pending.get(m.id);pending.delete(m.id);m.error?j(Error(JSON.stringify(m.error))):r(m.result)}});const send=(method,params={})=>new Promise((r,j)=>{const n=++id;pending.set(n,{r,j});ws.send(JSON.stringify({id:n,method,params}))});await send('Page.enable');const evaluate=async expression=>{const r=await send('Runtime.evaluate',{expression,returnByValue:true,awaitPromise:true});if(r.exceptionDetails)throw Error(r.exceptionDetails.exception?.description||r.exceptionDetails.text);return r.result.value};
await send('Page.navigate',{url:entry});for(let i=0;i<100;i++){try{if(await evaluate('location.protocol==="http:"&&document.readyState!=="loading"'))break}catch{}await delay(50)}const state=()=>evaluate('fetch("/__whale_plugin_ui/fixture").then(r=>r.json())');const command=m=>evaluate('fetch("/__whale_plugin_ui/fixture",{method:"POST",headers:{"Content-Type":"application/json"},body:'+JSON.stringify(JSON.stringify(m))+'}).then(async r=>({status:r.status,value:await r.json()}))');
const wait=async(test,why)=>{for(let i=0;i<250;i++){const value=await test();if(value)return value;await delay(100)}writeFileSync(path.join(root,'failure.json'),JSON.stringify({state:await state(),focus:await evaluate('({active:document.activeElement?.outerHTML,visible:document.visibilityState,ancestors:(()=>{let n=document.querySelector("[data-fixture-session]"),a=[];while(n){a.push({tag:n.tagName,cls:n.className,style:getComputedStyle(n).display,visible:getComputedStyle(n).visibility,rect:n.getBoundingClientRect().toJSON(),hidden:n.getAttribute("aria-hidden")});n=n.parentElement}return a})()})'),dom:await evaluate('document.documentElement.outerHTML')},null,2));throw Error(why)};
let first=await wait(async()=>{const v=await state();return v.state.subject?.sessionId==='fixture-session-a'&&v.state.observation&&await evaluate('typeof window.__setSessionFixture === "function"')?v:null},'real Session observation did not activate');
assert(!(first.state.outlets||[]).some(o=>o.registrant==='whaleisle-session-ui-fixture'));assert.equal(await evaluate('window.__whaleSessionNoOnboarding'),true);assert.equal(await evaluate('"dshDesktop" in globalThis'),false,'binding-free seat invented desktop permissions');assert.equal(await evaluate('[...document.querySelectorAll("[role=dialog]")].some(n=>/预览版说明|Preview Notice|添加一个 API Key|Add an API key/.test(n.textContent))'),false,'fresh profile mounted onboarding');
await evaluate('window.__setSessionFixture("declined")');await delay(200);first=await state();assert(!(first.state.outlets||[]).some(o=>o.registrant==='whaleisle-session-ui-fixture'),'chain/keyed fallback counted as a contribution');
await evaluate('window.__setSessionFixture("null")');await delay(200);assert(!((await state()).state.outlets||[]).some(o=>o.entryId==='fixture-null'),'null contribution counted as visible');
await evaluate('window.__setSessionFixture("composer")');let takeover=await wait(async()=>{const v=await state();return (v.state.outlets||[]).some(o=>o.slot==='conversation.composer'&&o.registrant==='whaleisle-session-ui-fixture')?v:null},'elected composer missing');
let composer=takeover.state.outlets.find(o=>o.slot==='conversation.composer'&&o.registrant==='whaleisle-session-ui-fixture');assert.equal((await command({type:'show',page:takeover.page,revision:takeover.state.revision,subject:takeover.state.subject,handle:composer.handle})).status,200);await wait(()=>evaluate('!!document.querySelector("[data-composer-seat][data-whale-session-surface]")'),'full composer seat not selected');
await evaluate('window.__setSessionFixture("header")');first=await wait(async()=>{const v=await state();return (v.state.outlets||[]).some(o=>o.entryId==='fixture-session-action'&&o.available)?v:null},'real Session fixture header missing');
const outlet=first.state.outlets.find(o=>o.entryId==='fixture-session-action'&&o.available);assert.equal((await command({type:'show',page:first.page,revision:first.state.revision,subject:first.state.subject,handle:outlet.handle})).status,200);
first=await wait(async()=>{const v=await state();return v.state.surface==='conversation.session.header.actions'&&v.state.actions.some(a=>a.label==='Rename fixture-session-a')?v:null},'local Session seat missing');
assert.equal(await evaluate('getComputedStyle(document.querySelector("[data-whale-session-surface]")).position'),'fixed');
await wait(()=>evaluate('document.activeElement?.getAttribute("data-fixture-session")==="fixture-session-a"'),'original control did not receive focus');
await evaluate('document.querySelector("[data-fixture-draft]").click()');await wait(()=>evaluate('JSON.parse(localStorage.getItem("dsh.conversation.fixture-session-a")||"{}").draft?.text==="Retained fixture draft"'),'original draft did not persist');
const mounts=await evaluate('window.__fixtureMounts'),unmounts=await evaluate('window.__fixtureUnmounts||0');
for(let round=0;round<2;round++){
  await command({type:'collapse'});await wait(async()=>!(await state()).state.surface,'seat did not collapse');
  assert.equal(await evaluate('document.activeElement?.hasAttribute("data-fixture-session")'),false,'hidden owner kept focus');
  first=await state();const current=first.state.outlets.find(o=>o.entryId==='fixture-session-action');assert(current);
  assert.equal((await command({type:'show',page:first.page,revision:first.state.revision,subject:first.state.subject,handle:current.handle})).status,200);
  first=await wait(async()=>{const v=await state();return v.state.surface==='conversation.session.header.actions'?v:null},'retained seat did not reopen');
}
assert.equal(await evaluate('JSON.parse(localStorage.getItem("dsh.conversation.fixture-session-a")).draft.text'),'Retained fixture draft','retained seat lost its original draft');
assert.equal(await evaluate('window.__fixtureMounts'),mounts,'reopening duplicated plugin effect');assert.equal(await evaluate('window.__fixtureUnmounts||0'),unmounts,'collapse unloaded the retained owner');
let action;await wait(async()=>{first=await state();action=first.state.sessionControls.find(a=>a.label==='Rename fixture-session-a');if(!action)return false;const r=await command({type:'sessionClick',page:first.page,revision:first.state.revision,subject:first.state.subject,handle:action.handle});if(r.status===409)return false;assert.equal(r.status,200);return true},'current original header action not admitted');await wait(()=>evaluate('document.documentElement.dataset.fixtureRenamed==="fixture-session-a"'),'Session rename not committed');
await wait(()=>evaluate('document.querySelector("[data-fixture-title]")?.textContent==="Fixture renamed fixture-session-a"'),'real Session projection did not echo committed title');
await command({type:'observationOff'});const previousPage=first.page;await send('Page.reload',{ignoreCache:true});first=await wait(async()=>{const v=await state();return v.page>previousPage&&v.state.subject?.sessionId==='fixture-session-a'&&!v.state.observation?v:null},'unknown renderer did not enter original-page fallback');
assert.equal((await command({type:'sessionClick',page:previousPage,revision:1,subject:first.state.subject,handle:action.handle})).status,409,'prior page retained authority');
assert.equal(await evaluate('[...document.querySelectorAll("[role=dialog]")].some(n=>/预览版说明|Preview Notice|添加一个 API Key|Add an API key/.test(n.textContent))'),false,'reloaded profile mounted onboarding');await wait(()=>evaluate('typeof window.__setSessionFixture==="function"'),'reloaded fixture unavailable');await evaluate('window.__setSessionFixture("header")');await wait(()=>evaluate('!!document.querySelector("[data-fixture-session]")'),'unobserved original renderer lost contribution');
first=await state();assert.equal((await command({type:'page',page:first.page,revision:first.state.revision,subject:first.state.subject})).status,200);await wait(()=>evaluate('document.querySelector("[data-slot=\\"main.conversation\\"]")?.hasAttribute("data-whale-session-surface")'),'original page fallback inaccessible');
await evaluate('document.querySelector("[data-fixture-session]").click()');await wait(()=>evaluate('document.querySelector("[data-fixture-title]")?.textContent==="Fixture renamed fixture-session-a"'),'fallback original control lost writeback');
assert.equal((await command({type:'select',subject:{kind:'session',sessionId:'fixture-session-b',epoch:'1:2'}})).status,200);await wait(async()=>{const v=await state();return v.state.subject?.sessionId==='fixture-session-b'},'second Session did not replace owner');
assert.equal((await command({type:'sessionClick',page:first.page,revision:first.state.revision,subject:first.state.subject,handle:action.handle})).status,409);assert.equal(await evaluate('document.documentElement.dataset.fixtureRenamed'),'fixture-session-a');
assert.equal(await evaluate('JSON.parse(localStorage.getItem("dsh.conversation.fixture-session-a")).draft.text'),'Retained fixture draft','reload or switch lost the original saved draft');
assert((await state()).state.entries.some(e=>e.id==='fixture-session-action'));await evaluate('window.__withdrawSessionFixture()');await wait(async()=>{const v=await state();return v.state.subject?.sessionId==='fixture-session-b'&&Array.isArray(v.state.entries)&&!v.state.entries.some(e=>e.id==='fixture-session-action')},'disposed registration retained metadata');
assert.equal((await command({type:'select',subject:{kind:'session',sessionId:'missing-session',epoch:'1:3'}})).status,200);await wait(async()=>{const v=await state();return v.state.subject===null&&v.state.unavailable==='session-owner-unavailable'&&v.state.surface===''},'missing owner did not fail closed');
await command({type:'unknownOnboarding'});await send('Page.reload',{ignoreCache:true});await wait(async()=>{const v=await state();return v.state.unavailable==='session-version-incompatible'},'unknown onboarding version was not blocked');assert.equal(await evaluate('[...document.querySelectorAll("[role=dialog]")].some(n=>/预览版说明|Preview Notice|添加一个 API Key|Add an API key/.test(n.textContent))'),false,'unknown version exposed onboarding');const blocked=await state();assert.equal(blocked.state.subject,null);assert.equal(blocked.state.surface,'');assert.equal((await command({type:'page',page:first.page,revision:first.state.revision,subject:first.state.subject})).status,409);
writeFileSync(path.join(root,'result.json'),JSON.stringify({passed:['real Loader registration','retained Session header owner','local original slot seat','real Session rename writeback','stale Session/page/revision rejected','missing Session owner fails closed','registration disposed','declined chain does not expose fallback','wrong keyed renderer does not expose fallback','null component has no visible outlet','elected complete composer seat','collapse releases focus without unmount','reopen does not repeat effects','original draft persists across retained seat','unknown observation preserves full original page','fresh and restarted pages have no welcome or credential onboarding','no invented desktop API or acknowledgement','unknown Models version blocks only new seat without welcome'],last:await state()},null,2));console.log('real Host Session plugin seat passed');
}finally{ws?.close();child.kill();await delay(300)}
`
