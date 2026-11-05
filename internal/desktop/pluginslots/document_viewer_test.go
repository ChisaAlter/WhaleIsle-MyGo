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
 const cleanup=[],subscriptions=[];let registryChanged,extensions=[],mounted,opened=[],selected=[],observer,observerOptions,readyToAdopt=false,seatPresent=true,warnings=[];
 function portal(role,visibility=''){return {marked:false,style:{visibility},getAttribute:name=>name==='role'?role:null,setAttribute(name){assert.equal(name,'data-whale-document-overlay');this.marked=true},removeAttribute(name){assert.equal(name,'data-whale-document-overlay');this.marked=false}}}
 const font=portal('dialog','hidden'),unrelated=portal('dialog'),notDialog=portal('menu'),portals={font,unrelated,notDialog};
 const seat={dataset:{sidebarRightSession:'s#1'},marked:false,setAttribute(){this.marked=true},removeAttribute(){this.marked=false},querySelector(){return {}},querySelectorAll(selector){assert.equal(selector,'[data-office-font-warning] [aria-controls]');return warnings}};
 const nodes=[];function element(){return {dataset:{},hidden:false,setAttribute(){},remove(){},textContent:''}}
 const document={head:{appendChild(){}},body:{appendChild:n=>nodes.push(n)},createElement:element,getElementById:id=>portals[id]||null,querySelectorAll:s=>s==='[data-sidebar-right-session]'?(seatPresent?[seat]:[]):s==='[data-whale-document-seat]'?(seat.marked?[seat]:[]):s==='[data-whale-document-overlay]'?Object.values(portals).filter(p=>p.marked):[]};
 const right={mounted:{getSnapshot:()=>mounted,subscribe:f=>{subscriptions.push(f);return()=>{}}},openTabs:{subscribe:f=>{subscriptions.push(f);return()=>{}}},openResourceIn:(s,a,o)=>{if(!readyToAdopt)return false;opened.push([s,a,o]);return true},tabsIn:()=>opened.length?[{id:'tab',kind:'text',contentId:opened[0][1]}]:[],focus:()=>{}};
 const ctx={effect:f=>{cleanup.push(f())},inject:(deps,f)=>{assert.deepEqual(Array.from(deps),['uiWorkspace','sidebarRight','documentPreviews']);f({...ctx,sidebarRight:right,documentPreviews:{candidates:()=>extensions,subscribe:f=>{registryChanged=f;return()=>{}}},uiWorkspace:{openSession:s=>selected.push(s)}})}};
 const window={dispatchEvent(){}};const context={window,Event:class{},ctx,URLSearchParams,encodeURIComponent,location:{search:'?desktopDocumentSession=s%231&desktopDocumentPath=dir%2Fa%20%23.md'},document,CSS:{escape:s=>s},MutationObserver:class{constructor(f){observer=f}observe(node,options){observerOptions=options}disconnect(){}}};vm.createContext(context);
 vm.runInContext(` + strconv.Quote(string(documentViewerSource)) + `,context);vm.runInContext('whaleBindDocumentViewer(ctx)',context);
 assert.deepEqual(selected,['s#1']);assert.equal(window.__whaleDocumentRecommendation.extension,false);extensions=[{id:'custom'}];registryChanged();assert.equal(window.__whaleDocumentRecommendation.extension,true);extensions=[{priority:'builtin'}];registryChanged();assert.equal(window.__whaleDocumentRecommendation.extension,false);assert.equal(opened.length,0);assert.equal(seat.marked,false);
 extensions=[{id:'@deepseek-ai/dsh-client-ui-sidebar-documentpreview/excel',priority:'builtin'}];registryChanged();assert.equal(window.__whaleDocumentRecommendation.spreadsheet,false);
 mounted='s#1';subscriptions[0]();assert.equal(opened.length,0);
 readyToAdopt=true;observer();assert.equal(opened.length,1);assert.equal(opened[0][1],'dsh-resource://file/session/s%231/dir/a%20%23.md');assert.equal(opened[0][2].kind,'text');assert.equal(seat.marked,true);
 observer();assert.equal(opened.length,1);assert.equal(window.__whaleDocumentRecommendation.spreadsheet,true);
 // Original FontNotice portals to body and stays hidden until positioned.
 assert(observerOptions.attributeFilter.includes('style'));assert(observerOptions.attributeFilter.includes('aria-controls'));
 warnings=['font','notDialog','missing'].map(id=>({getAttribute:name=>name==='aria-controls'?id:null}));observer();assert.equal(font.marked,false);
 font.style.visibility='';observer();assert.equal(font.marked,true);assert.equal(unrelated.marked,false);assert.equal(notDialog.marked,false);
 warnings=[];observer();assert.equal(font.marked,false);
 warnings=[{getAttribute:()=> 'font'}];observer();assert.equal(font.marked,true);
 seatPresent=false;observer();assert.equal(font.marked,false);assert.equal(seat.marked,false);
 seatPresent=true;observer();assert.equal(font.marked,true);
 extensions=[{id:'other',priority:'builtin'}];registryChanged();assert.equal(window.__whaleDocumentRecommendation.spreadsheet,false);
 extensions=[{id:'@deepseek-ai/dsh-client-ui-sidebar-documentpreview/excel',priority:'builtin'}];registryChanged();assert.equal(window.__whaleDocumentRecommendation.spreadsheet,true);
 mounted='other' ;subscriptions[0]();assert.equal(seat.marked,false);assert.equal(font.marked,false);assert.equal(nodes[0].hidden,false);assert.equal(window.__whaleDocumentRecommendation.spreadsheet,false);
 mounted='s#1';subscriptions[0]();assert.equal(font.marked,true);
 cleanup.reverse().forEach(f=>f());assert.equal(seat.marked,false);assert.equal(font.marked,false);`
	if out, err := runNodeFixture(node, "-e", script); err != nil {
		t.Fatalf("document carrier: %v\n%s", err, out)
	}
}
