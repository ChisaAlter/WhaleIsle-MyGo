package pluginslots

import (
	"os/exec"
	"strconv"
	"testing"
)

func TestB25OriginalDocumentOwnerCarrier(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node unavailable")
	}
	script := `const vm=require('node:vm'),assert=require('node:assert/strict');
 const cleanup=[],subscriptions=[];let registryChanged,extensions=[],mounted,opened=[],selected=[],observer,readyToAdopt=false;
 const seat={dataset:{sidebarRightSession:'s#1'},marked:false,setAttribute(){this.marked=true},removeAttribute(){this.marked=false},querySelector(){return {}}};
 const nodes=[];function element(){return {dataset:{},hidden:false,setAttribute(){},remove(){},textContent:''}}
 const document={head:{appendChild(){}},body:{appendChild:n=>nodes.push(n)},createElement:element,querySelectorAll:s=>s==='[data-sidebar-right-session]'?[seat]:seat.marked?[seat]:[]};
 const right={mounted:{getSnapshot:()=>mounted,subscribe:f=>{subscriptions.push(f);return()=>{}}},openTabs:{subscribe:f=>{subscriptions.push(f);return()=>{}}},openResourceIn:(s,a,o)=>{if(!readyToAdopt)return false;opened.push([s,a,o]);return true},tabsIn:()=>opened.length?[{id:'tab',kind:'text',contentId:opened[0][1]}]:[],focus:()=>{}};
 const ctx={effect:f=>{cleanup.push(f())},inject:(deps,f)=>{assert.deepEqual(Array.from(deps),['uiWorkspace','sidebarRight','documentPreviews']);f({...ctx,sidebarRight:right,documentPreviews:{candidates:()=>extensions,subscribe:f=>{registryChanged=f;return()=>{}}},uiWorkspace:{openSession:s=>selected.push(s)}})}};
 const window={dispatchEvent(){}};const context={window,Event:class{},ctx,URLSearchParams,encodeURIComponent,location:{search:'?desktopDocumentSession=s%231&desktopDocumentPath=dir%2Fa%20%23.md'},document,CSS:{escape:s=>s},MutationObserver:class{constructor(f){observer=f}observe(){}disconnect(){}}};vm.createContext(context);
 vm.runInContext(` + strconv.Quote(string(documentViewerSource)) + `,context);vm.runInContext('whaleBindDocumentViewer(ctx)',context);
 assert.deepEqual(selected,['s#1']);assert.equal(window.__whaleDocumentRecommendation.extension,false);extensions=[{id:'custom'}];registryChanged();assert.equal(window.__whaleDocumentRecommendation.extension,true);extensions=[{priority:'builtin'}];registryChanged();assert.equal(window.__whaleDocumentRecommendation.extension,false);assert.equal(opened.length,0);assert.equal(seat.marked,false);
 extensions=[{id:'@deepseek-ai/dsh-client-ui-sidebar-documentpreview/excel',priority:'builtin'}];registryChanged();assert.equal(window.__whaleDocumentRecommendation.spreadsheet,false);
 mounted='s#1';subscriptions[0]();assert.equal(opened.length,0);
 readyToAdopt=true;observer();assert.equal(opened.length,1);assert.equal(opened[0][1],'dsh-resource://file/session/s%231/dir/a%20%23.md');assert.equal(opened[0][2].kind,'text');assert.equal(seat.marked,true);
 observer();assert.equal(opened.length,1);assert.equal(window.__whaleDocumentRecommendation.spreadsheet,true);
 extensions=[{id:'other',priority:'builtin'}];registryChanged();assert.equal(window.__whaleDocumentRecommendation.spreadsheet,false);
 extensions=[{id:'@deepseek-ai/dsh-client-ui-sidebar-documentpreview/excel',priority:'builtin'}];registryChanged();assert.equal(window.__whaleDocumentRecommendation.spreadsheet,true);
 mounted='other' ;subscriptions[0]();assert.equal(seat.marked,false);assert.equal(nodes[0].hidden,false);assert.equal(window.__whaleDocumentRecommendation.spreadsheet,false);
 cleanup.reverse().forEach(f=>f());assert.equal(seat.marked,false);`
	if out, err := exec.Command(node, "-e", script).CombinedOutput(); err != nil {
		t.Fatalf("document carrier: %v\n%s", err, out)
	}
}
