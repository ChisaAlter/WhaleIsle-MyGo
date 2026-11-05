package pluginui

import (
	"bytes"
	"github.com/egoist/mygo/ui"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"testing"
	"whaleisle/internal/processutil"
)

func TestNativeDocumentConsumesOnlyCurrentObservedOwner(t *testing.T) {
	subject := Subject{Kind: "session", SessionID: "s", Epoch: "1:2"}
	row := SessionOutlet{Handle: "entry", Slot: "conversation.chat.user.content", Registrant: "fixture-extension", Available: true, Seq: 9, HasSeq: true, Native: []NativeNode{{Kind: "text", Text: "Native extension content"}, {Kind: "button", Handle: "action", Text: "Native extension action"}}}
	o := &Owner{selected: subject, state: Snapshot{Connected: true, Subject: &subject, Outlets: []SessionOutlet{row}}}
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Children(func() {
			if !o.RenderNativeMessage(c, row.Slot, "", 9) {
				ui.Text(c, "Native fallback")
			}
		})
	}, 700, 500)
	if !tt.HasText("Native extension content") || !tt.HasText("Native extension action") || tt.HasText("Native fallback") {
		t.Fatal(tt.Texts())
	}
	o.Select(Subject{Kind: "session", SessionID: "other", Epoch: "1:3"})
	tt.Frame()
	if !tt.HasText("Native fallback") || tt.HasText("Native extension content") {
		t.Fatal("source survived selection change")
	}
	if validNativeNodes([]NativeNode{{Kind: "script", Text: "arbitrary"}}) || validNativeNodes(make([]NativeNode, 513)) {
		t.Fatal("unbounded or executable node kind")
	}
	if !validSnapshot(Snapshot{Revision: 1, Subject: &Subject{Kind: "view", Name: "main:fixture:extension"}, Views: []NativeView{{Handle: "main:fixture:extension", ID: "fixture", Slot: "main"}}, Documents: []NativeDocument{{Slot: "main", Nodes: row.Native}}}) {
		t.Fatal("native navigation document rejected")
	}
}
func TestComposerProviderUsesNativeRevisionAndNeverBrowserDraft(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Fatal(err)
	}
	script := `(async()=>{const vm=require('node:vm'),assert=require('node:assert/strict');const insertions=[];const context={URLSearchParams,XMLHttpRequest:class{open(method,path,sync){assert.equal(method,'POST');assert.equal(path,'/__whale_plugin_ui/composer/insert');assert.equal(sync,false)}setRequestHeader(){}send(body){const request=JSON.parse(body);insertions.push(request);this.status=200;this.responseText=JSON.stringify({ok:true,page:3,request:request.request})}},location:{search:'?desktopNativeSession=1'},window:{dispatchEvent(){}},CustomEvent:class{}};vm.createContext(context);vm.runInContext(` + strconv.Quote(composerSource) + `,context);
 const api=context.window.__whaleNativeComposer,sent=[],binding={sessionId:'a'};let native=0;
 const p=api.provide(binding,{state:{getSnapshot:()=>({queue:[]})},actions:{setDraft(){throw Error('browser draft touched')}}});
 assert.equal(p.state.getSnapshot().draft,'');assert.equal(await p.actions.setDraft('before binding'),false);
 const subject={kind:'session',sessionId:'a',epoch:'2:3'},owner={subject,binding:'opaque',revision:7,targetId:'a',editSeq:-1};
 api.connect(command=>{sent.push(command);queueMicrotask(()=>api.receive({page:3,request:command.composer.request,ok:true}))},subject);api.receive({page:3,draft:{owner,text:'Native draft',attachments:[{id:8,name:'file.png'}],selectionStart:1,selectionEnd:4}});
 assert.equal(p.state.getSnapshot().draft,'Native draft');const span=p.actions.captureInsertion();assert.equal(span.start,1);assert.equal(span.end,4);
 assert.equal(p.actions.insertText('replacement',span),true);assert.equal(insertions.at(-1).owner.binding,'opaque');assert.equal(insertions.at(-1).replace.text,'replacement');
 assert.equal(await p.actions.removeAttachment('8'),true);assert.equal(sent.at(-1).composer.edit.removeFileId,8);assert.equal(p.actions.addAttachments(['browser-registry-id']),false);
 assert.equal(await p.actions.submit(),true);assert.equal(sent.at(-1).composer.op,'submit');assert.equal(sent.at(-1).composer.gesture.owner.revision,7);
 api.receive({draft:{owner:{...owner,revision:8},text:'New native edit',attachments:[],selectionStart:2,selectionEnd:2}});assert.equal(p.actions.insertText('stale',span),false);
 api.connect(command=>sent.push(command),{kind:'session',sessionId:'b',epoch:'2:4'});assert.equal(p.state.getSnapshot().draft,'');assert.equal(await p.actions.setDraft('cross owner'),false);})().catch(e=>{console.error(e);process.exitCode=1});`
	cmd := exec.Command(node, "-e", script)
	processutil.Background(cmd)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("native composer provider: %v\n%s", err, out)
	}
}
func TestComposerResponseMatchesOriginalProviderWithoutDuplicateMount(t *testing.T) {
	source := []byte(`id: "@deepseek-ai/dsh-client-ui-conversation"` + "\n\t\t\t\t\tconst shell = inputHub.shellFor(binding);\n\t\t\t\t\t\t\tinput: shell.state\n\t\t\t\t\t\tprops: { inputActions: shell.actions }")
	if path := os.Getenv("WHALEISLE_PLUGIN_UI_CONVERSATION"); path != "" {
		var err error
		source, err = os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
	}
	changed, ok := bridgeConversationComposer(source)
	if !ok || !bytes.Contains(changed, []byte("whaleNativeInput?.actions")) {
		t.Fatal("known actual input provider not bridged")
	}
	if os.Getenv("WHALEISLE_PLUGIN_UI_CONVERSATION") != "" && !bytes.Contains(changed, []byte("selectors.set(sessionId, select)")) {
		t.Fatal("conversation registry selector not bound")
	}
	if changed2, ok := bridgeConversationComposer([]byte(strings.ReplaceAll(string(source), "inputActions: shell.actions", "inputActions: unknownActions"))); ok || !bytes.Contains(changed2, []byte("unknownActions")) {
		t.Fatal("unknown provider rewritten")
	}
}
func TestActivationNavigationHandbackIsOneShotAndListScoped(t *testing.T) {
	o := &Owner{selected: Subject{Kind: "list"}, state: Snapshot{Navigation: &Subject{Kind: "bundle", Name: "voice"}}}
	if got := o.TakeNavigation(); got == nil || got.Name != "voice" {
		t.Fatal("activation details handback absent")
	}
	if o.TakeNavigation() != nil {
		t.Fatal("handback consumed twice")
	}
	o.selected = Subject{Kind: "bundle", Name: "other"}
	o.state.Navigation = &Subject{Kind: "bundle", Name: "voice"}
	if o.TakeNavigation() != nil {
		t.Fatal("handback stole another detail owner")
	}
}
func TestNativeDelegateAliasesConsumeExactObservedOutlets(t *testing.T) {
	subject := Subject{Kind: "session", SessionID: "s", Epoch: "1:2"}
	for _, tc := range []struct{ alias, slot string }{{"user.content", "conversation.chat.user.content"}, {"reasoning.body", "conversation.chat.reasoning.body"}, {"assistant-actions", "conversation.chat.assistant-actions"}, {"turnTail", "conversation.chat.turnTail"}, {"toolview/body", "tool.call.toolview"}} {
		t.Run(tc.alias, func(t *testing.T) {
			o := &Owner{selected: subject, state: Snapshot{Connected: true, Subject: &subject, Outlets: []SessionOutlet{{Handle: "actual-occurrence", Slot: tc.slot, Registrant: "fixture-extension", Available: true, Seq: 9, HasSeq: true, Native: []NativeNode{{Kind: "text", Text: "Owned " + tc.slot}}}}}}
			tt := ui.NewTester(func(c *ui.Context) {
				ui.Column(c).Fill().Children(func() {
					if !o.RenderNativeMessage(c, CanonicalNativeSlot(tc.alias), "", 9) {
						ui.Text(c, "wrong native fallback")
					}
				})
			}, 700, 500)
			if tt.HasText("wrong native fallback") || !tt.HasText("Owned "+tc.slot) {
				t.Fatal(tt.Texts())
			}
		})
	}
}
func TestGlobalShellOverlayConsumesLiveRootScopedDocument(t *testing.T) {
	subject := Subject{Kind: "navigation"}
	snapshot := Snapshot{Revision: 1, Connected: true, Subject: &subject, GlobalDocuments: []NativeDocument{{Slot: "shell.overlay", Handle: "overlay", Nodes: []NativeNode{{Kind: "button", Handle: "hello", Text: "Hello from generated plugin"}}}}}
	if !validSnapshot(snapshot) {
		t.Fatal("actual additive root document rejected")
	}
	o := &Owner{selected: subject, state: snapshot}
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Children(func() { o.RenderGlobalNativeSlot(c, "shell.overlay") })
	}, 700, 500)
	if !tt.HasText("Hello from generated plugin") {
		t.Fatal(tt.Texts())
	}
	o.Close()
	tt.Frame()
	if tt.HasText("Hello from generated plugin") {
		t.Fatal("root entry survived owner teardown")
	}
}
func TestFactoryObservationPreservesSingleActualMountAndInheritedChatSeq(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Fatal(err)
	}
	script := `const vm=require('node:vm'),assert=require('node:assert/strict');const context={URLSearchParams,location:{search:'?desktopNativeSession=1'},window:{},react:{createElement:(tag,props,content)=>({tag,props,content}),cloneElement:(element,extra)=>({...element,props:{...element.props,...extra}})}};vm.createContext(context);vm.runInContext(` + strconv.Quote(rendererPrelude) + `,context);
 let live=true;const definition={name:'conversation.chat.user.content'},content={mountedOnce:true};
 const result=context.whaleSessionObserveFactory({isFactoryLive:entry=>live&&entry===definition},definition.name,definition,{},content);
 assert.equal(result.content,content);assert.equal(result.props['data-whale-factory-entry'],definition.name);
 const marker={isConnected:true,getAttribute:name=>result.props[name]??null,hasAttribute:name=>Object.hasOwn(result.props,name)};
 assert.equal(context.window.__whaleSessionEntryOf(marker,definition.name),definition);live=false;assert.equal(context.window.__whaleSessionEntryOf(marker,definition.name),null);
 const row=context.whaleSessionObserveEntry('div','conversation.chat.node',{},'row',{node:{anchorSeq:3,data:{finalNode:{seq:19}}}},content);assert.equal(row.props['data-whale-owner-seq'],19);
 const user=context.whaleSessionObserveEntry('div','conversation.chat.node',{},'user',{node:{anchorSeq:3,data:{seq:7}}},content);assert.equal(user.props['data-whale-owner-seq'],7);`
	cmd := exec.Command(node, "-e", script)
	processutil.Background(cmd)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("factory actual owner: %v\n%s", err, out)
	}
}
func TestThirdPartyRootRendererCapturesOwnerAndRejectsStaleRoot(t *testing.T) {
	subject := Subject{Kind: "navigation"}
	s := Snapshot{Connected: true, Page: 1, Revision: 2, Subject: &subject, GlobalDocuments: []NativeDocument{{Slot: "root", Handle: "third-party", Nodes: []NativeNode{{Kind: "text", Text: "Actual third-party root"}}}}}
	if !validSnapshot(s) {
		t.Fatal("root document rejected")
	}
	o := &Owner{selected: subject, state: s}
	render := o.GlobalNativeRenderer("root")
	if render == nil {
		t.Fatal("missing root renderer")
	}
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Children(func() {
			if !render(c) {
				ui.Text(c, "Native shell fallback")
			}
		})
	}, 700, 400)
	if !tt.HasText("Actual third-party root") {
		t.Fatal(tt.Texts())
	}
	o.mu.Lock()
	o.state.Revision++
	o.mu.Unlock()
	tt.Frame()
	if !tt.HasText("Native shell fallback") || tt.HasText("Actual third-party root") {
		t.Fatal("captured stale root painted")
	}
}
func TestInstalledRootOutletObservationRetainsSingleActualRootEntry(t *testing.T) {
	path := os.Getenv("WHALEISLE_PLUGIN_UI_RENDERER")
	if path == "" {
		t.Skip("installed renderer shape fixture unavailable")
	}
	source, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	observed, ok := observeRenderer(source)
	if !ok {
		t.Fatal("installed renderer not matched")
	}
	if !bytes.Contains(observed, []byte(`return whaleSessionObserveEntry("span", "root", entry, entryKeyOf(entry), ownerProps,`)) {
		t.Fatal("actual RootOutlet was not observed")
	}
	if bytes.Count(source, []byte(`jsx)(RootEntry,`)) != bytes.Count(observed, []byte(`jsx)(RootEntry,`)) {
		t.Fatal("original root mounted twice")
	}
}
func TestPluginTabDocumentsAreRealPageScopedSubjects(t *testing.T) {
	subject := Subject{Kind: "pluginTab", Name: "fixture-tab"}
	s := Snapshot{Connected: true, Revision: 1, Subject: &subject, Entries: []Entry{{Slot: "plugins.tab", ID: "fixture-tab", Label: "Fixture tab"}}, Documents: []NativeDocument{{Slot: "plugins.tab", Nodes: []NativeNode{{Kind: "button", Handle: "original-action", Text: "Original tab action"}}}}}
	if !validSnapshot(s) {
		t.Fatal("page tab rejected")
	}
	o := &Owner{selected: subject, state: s}
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Children(func() { o.RenderNativeSlot(c, "plugins.tab", "", -1) })
	}, 700, 400)
	if !tt.HasText("Original tab action") {
		t.Fatal(tt.Texts())
	}
	o.mu.Lock()
	o.state.Navigation = &Subject{Kind: "bundle", Name: "fixture-plugin"}
	o.mu.Unlock()
	if next := o.TakeNavigation(); next == nil || next.Name != "fixture-plugin" {
		t.Fatal("original tab openPackage was not handed back to native manager")
	}
}
func TestNativeOriginalPortalRetainsModalInputBoundary(t *testing.T) {
	subject := Subject{Kind: "pluginTab", Name: "fixture-tab"}
	o := &Owner{selected: subject, state: Snapshot{Connected: true, Subject: &subject, Documents: []NativeDocument{{Slot: "plugins.tab", Nodes: []NativeNode{{Kind: "text", Text: "Underlying original tab"}, {Kind: "group", Handle: "original-install-dialog", Modal: true, Children: []NativeNode{{Kind: "text", Text: "Original install approval"}, {Kind: "button", Handle: "source-cancel", Text: "Original cancel"}}}}}}}}
	behind := 0
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Children(func() {
			if ui.Button(c, "Native underlying action").Clicked() {
				behind++
			}
			o.RenderNativeSlot(c, "plugins.tab", "", -1)
		})
	}, 900, 700)
	if !tt.HasText("Original install approval") || !tt.HasText("Original cancel") {
		t.Fatal(tt.Texts())
	}
	_ = tt.Click("Native underlying action")
	tt.Frame()
	if behind != 0 {
		t.Fatal("original source modal did not block underlying native actions")
	}
}
func TestNativeSelectKeepsDistinctValuesForDuplicateLabels(t *testing.T) {
	subject := Subject{Kind: "pluginTab", Name: "fixture"}
	o := &Owner{selected: subject, state: Snapshot{Connected: true, Subject: &subject, Documents: []NativeDocument{{Slot: "plugins.tab", Nodes: []NativeNode{{Kind: "select", Handle: "original-select", Value: "second", Options: []NativeOption{{Value: "first", Label: "Same label"}, {Value: "second", Label: "Same label"}}}}}}}}
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Children(func() { o.RenderNativeSlot(c, "plugins.tab", "", -1) })
	}, 700, 400)
	if !tt.HasText("Same label [2]") {
		t.Fatal("selected option identity was collapsed", tt.Texts())
	}
}
func TestComposerBarPublicAttachmentPropsUseNativeOwner(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Fatal(err)
	}
	script := `const vm=require('node:vm'),assert=require('node:assert/strict'),{File}=require('node:buffer');const sent=[],transfers=[];let received;const c={URLSearchParams,URL,Blob,File,FormData,location:{search:'?desktopNativeSession=1'},window:{dispatchEvent(){}},CustomEvent:class{},fetch:async(url,options)=>{transfers.push(url);if(url.endsWith('/file')){const req=JSON.parse(options.body);assert.equal(req.page,4);assert.equal(req.owner.binding,'native-binding');return new Response(req.id===8?'png bytes':'note bytes',{headers:{'content-type':req.id===8?'image/png':'text/plain'}})}const body=options.body;assert.equal(JSON.parse(body.get('owner')).owner.binding,'native-binding');const metadata=JSON.parse(body.get('metadata'));received=body.get('file');assert.equal(metadata.name,'dropped.txt');assert.equal(metadata.mediaType,'application/x-original');assert.equal(metadata.size,700000);assert.equal(await received.text(),'x'.repeat(700000));return Response.json({page:4,draft:{owner:{...owner,binding:'new-binding'},text:'draft',attachments:[...attachments,{id:10,name:received.name,mediaType:received.type,size:received.size}],selectionStart:0,selectionEnd:0}})}};vm.createContext(c);vm.runInContext(` + strconv.Quote(composerSource) + `,c);const api=c.window.__whaleNativeComposer,subject={kind:'session',sessionId:'a',epoch:'3:4'},owner={subject,binding:'native-binding',revision:9,targetId:'a',editSeq:-1},attachments=[{id:8,name:'selected.png',image:true,mediaType:'image/png',size:9,error:'retry needed'},{id:9,name:'notes.txt',mediaType:'',size:10}];api.connect(message=>{sent.push(message);queueMicrotask(()=>api.receive({page:4,request:message.composer.request,ok:true}))},subject);api.receive({page:4,draft:{owner,text:'draft',attachments,selectionStart:0,selectionEnd:0}});
 const old=()=>{throw Error('original browser attachment registry touched')};const props=api.barProps('a',{addFiles:old,removeAttachment:old,resolveDraftAttachments:old,retryFileUpload:old,hooks:{fileUploads:{getSnapshot:old}}});
 (async()=>{assert.equal(props.resolveDraftAttachments(['8','9']).length,0);await new Promise(r=>setTimeout(r,15));const rows=props.resolveDraftAttachments(['8','9']);assert.equal(rows.length,2);assert.equal(rows[0].kind,'image');assert.equal(rows[0].file.name,'selected.png');assert.equal(await rows[0].file.text(),'png bytes');assert(rows[0].previewUrl.startsWith('blob:'));assert.equal(rows[0].path,undefined);assert.equal(rows[1].file.type,'','explicit empty File.type is not sniffed MIME');
 assert.equal(props.hooks.fileUploads.getSnapshot()['8'].status,'error');assert.equal(props.hooks.fileUploads.getSnapshot()['8'].message,'retry needed');assert.equal(props.hooks.fileUploads.getSnapshot()['9'].status,'ready');assert.equal(props.hooks.fileUploads.getSnapshot(),props.hooks.fileUploads.getSnapshot());
 assert.equal(await props.retryFileUpload('8'),true);assert.equal(sent.at(-1).composer.edit.retryFileId,8);assert.equal(sent.at(-1).composer.edit.owner.binding,'native-binding');assert.equal(props.retryFileUpload('unknown'),false);assert.equal(await props.removeAttachment('9'),true);assert.equal(sent.at(-1).composer.edit.removeFileId,9);
 const dropped=new File(['x'.repeat(700000)],'dropped.txt',{type:'application/x-original'}),before=sent.length;assert.equal(props.addFiles([dropped]),null);await new Promise(r=>setTimeout(r,25));assert.equal(sent.length,before,'supplied File bytes must not use socket or reopen picker');const added=props.resolveDraftAttachments(['10']);assert.equal(added.length,1);assert.equal(added[0].file,dropped);assert.equal(added[0].file.type,'application/x-original');assert.equal(await added[0].file.text(),'x'.repeat(700000));assert.equal(transfers.filter(p=>p.endsWith('/files')).length,1);assert.notEqual(props.addFiles([{}]),null);
 api.connect(()=>{throw Error('retired bar wrote to another session')},{kind:'session',sessionId:'b',epoch:'3:5'});assert.equal(props.removeAttachment('9'),false);assert.equal(props.resolveDraftAttachments(['8']).length,0);assert.notEqual(props.addFiles([dropped]),null);})().catch(e=>{console.error(e);process.exitCode=1});`
	cmd := exec.Command(node, "-e", script)
	processutil.Background(cmd)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("public bar native facade: %v\n%s", err, out)
	}
}
func TestComposerPublicBarAdapterMatchesInstalledInject(t *testing.T) {
	path := os.Getenv("WHALEISLE_PLUGIN_UI_CONVERSATION")
	if path == "" {
		t.Skip("installed source unavailable")
	}
	source, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	out, ok := bridgeConversationComposer(source)
	if !ok || !bytes.Contains(out, []byte(`return (window.__whaleNativeComposer?.barProps || ((id, props) => props))(sessionId, {`)) {
		t.Fatal("installed public bar props were not wrapped")
	}
	if bytes.Count(out, []byte(`retryFileUpload: (id) => {`)) != bytes.Count(source, []byte(`retryFileUpload: (id) => {`)) {
		t.Fatal("public original injection body was duplicated or lost")
	}
	node, err := exec.LookPath("node")
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(node, "--check")
	cmd.Stdin = bytes.NewReader(out)
	processutil.Background(cmd)
	if diagnostic, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("installed transformed composer syntax: %v\n%s", err, diagnostic)
	}
}
