package openui

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"os/exec"
	"strings"
	"testing"

	"whaleisle/internal/processutil"
)

func TestWhaleUIProductsAndBridge(t *testing.T) {
	viewer, err := LoadViewer("")
	if err != nil || !IsWhaleViewer(viewer) {
		t.Fatal("default v2 runtime", err)
	}
	var provenance struct {
		Assets []struct {
			Name, SHA256 string
			Bytes        int
		}
	}
	raw, err := os.ReadFile("viewer/v2/provenance.json")
	if err != nil || json.Unmarshal(raw, &provenance) != nil {
		t.Fatal("provenance", err)
	}
	for _, a := range provenance.Assets {
		raw, err := os.ReadFile("viewer/v2/" + a.Name)
		sum := sha256.Sum256(raw)
		if err != nil || a.Bytes != len(raw) || a.SHA256 != hex.EncodeToString(sum[:]) {
			t.Fatal("asset mismatch", a.Name, err)
		}
	}
	for _, n := range []string{"mermaid.js", "three.js", "echarts-core.js", "echarts-full.js", "icons.js", "standalone-runtime.js"} {
		if b, err := Bundle(n); err != nil || len(b) == 0 {
			t.Fatal("missing bundle", n, err)
		}
	}
	if _, err := Bundle("../host.cjs"); err == nil {
		t.Fatal("bundle traversal")
	}
	id := Identity("host", "session", "tool:call", 0)
	r := Render{InstanceID: id, Revision: 1, Program: `root = Col([Title("Whale UI live fixture","md"), Input($quantity,"Quantity","number"), Text("Total " + $quantity*2), Button("Queue action",@ToAssistant("Computed " + $quantity*2)), Button("Copy result",@CopyToClipboard("" + $quantity*2))])` + "\n$quantity = 2", Language: "whale-ui", Locale: "en", HostActions: true, Media: true, State: json.RawMessage(`{}`)}
	shell, err := Shell(viewer, r, 900, "dark")
	if err != nil {
		t.Fatal(err)
	}
	if path := os.Getenv("WHALE_UI_BROWSER_SHELL"); path != "" {
		if err := os.WriteFile(path, []byte(shell), 0600); err != nil {
			t.Fatal(err)
		}
	}
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node unavailable")
	}
	script := strings.Split(strings.Split(shell, "<script>")[1], "</script>")[0]
	fixture := `global.window=global;const listeners={};global.addEventListener=(n,f)=>listeners[n]=f;const sent=[];let frames=0;const fakeFrame={contentWindow:{postMessage:m=>sent.push(structuredClone(m))},setAttribute:()=>{}};global.document={createElement:()=>{frames++;return fakeFrame},body:{append:()=>{}}};` + script + `
const emit=d=>listeners.message({source:frame.contentWindow,origin:'null',data:{channel:'whale-ui',version:2,instanceId:instance,...d}});
emit({type:'ready'});if(sent.at(-1).language!=='whale-ui'||sent.at(-1).locale!=='en')throw Error('render dialect missing');
emit({type:'state',state:{format:'whale-ui-state',version:1,values:{x:7}}});window.__whaleOpenUIDrain();
window.__whaleOpenUIRender({instanceId:instance,revision:2,program:'second',streaming:false,locale:'zh',variables:{'--accent':'#123456'}});
if(frames!==1||sent.at(-1).state.values.x!==7||sent.at(-1).locale!=='zh')throw Error('state/lifecycle reset');
emit({type:'bundle',requestId:'asset-1',name:'mermaid.js'});if(!sent.at(-1).base64||sent.at(-1).type!=='bundle-response')throw Error('engine missing');
emit({type:'bundle',requestId:'bad-asset',name:'../../host.cjs'});if(!sent.at(-1).error)throw Error('bundle traversal');
emit({type:'action',requestId:'copy-1',action:'clipboard',text:'ok',userGesture:true});let copy=window.__whaleOpenUIDrain()[0];if(copy.type!=='clipboard')throw Error('native action not normalized');
window.__whaleOpenUIAPIResponse({instanceId:instance,requestId:copy.requestId,status:500,body:{ok:false,error:'native failed'}});if(sent.at(-1).result.ok||sent.at(-1).result.error!=='native failed')throw Error('false action success');
let replay=false;try{window.__whaleOpenUIAPIResponse({instanceId:instance,requestId:copy.requestId,status:200,body:{ok:true}})}catch{replay=true}if(!replay)throw Error('replayed reply');
emit({type:'action',requestId:'bad-copy',action:'clipboard',text:'bad',userGesture:false});if(window.__whaleOpenUIDrain().length||sent.at(-1).result.ok)throw Error('unactivated action');
emit({type:'action',requestId:'export',action:'download',filename:'report.html',mime:'text/html',content:'a'.repeat(300000),userGesture:true});let parts=window.__whaleOpenUIDrain();emit({type:'action',requestId:'copy-during-export',action:'clipboard',text:'bad sequence',userGesture:true});if(sent.at(-1).result.ok||!sent.at(-1).result.error.includes('export'))throw Error('concurrent export action reordered');for(let i=0;i<5;i++)parts.push(...window.__whaleOpenUIDrain());if(parts.filter(x=>x.type==='download-part').map(x=>x.content).join('').length!==300000||parts.at(-1).type!=='download'||parts.some(x=>x.type==='clipboard'))throw Error('fragmented export lost data/order');
emit({type:'panel',operation:{append:true,spec:{items:[{type:'text',content:'panel'}]}}});if(window.__whaleOpenUIDrain()[0].operation.spec.items[0].content!=='panel')throw Error('panel not connected');
listeners.message({source:{},origin:'null',data:{channel:'whale-ui',version:2,instanceId:instance,type:'state',state:{evil:true}}});if(window.__whaleOpenUIDrain().length)throw Error('foreign frame admitted');
emit({type:'resource',requestId:'slow',path:'/api/wiki',query:{title:'Slow'}});const slow=window.__whaleOpenUIDrain()[0];window.__whaleOpenUIRender({instanceId:instance,revision:3,program:'new page',streaming:false});if(window.__whaleOpenUIAPIResponse({instanceId:instance,requestId:slow.requestId,status:200,body:{photos:[]}})!==true)throw Error('late reply killed new page');
emit({type:'action',requestId:'after-late',action:'clipboard',text:'still active',userGesture:true});if(window.__whaleOpenUIDrain()[0].text!=='still active')throw Error('new page unusable after late reply');
window.__whaleOpenUIHtmlViewSupport(false);if(sent.at(-1).htmlView!==false)throw Error('capability false ignored');window.__whaleOpenUIHtmlViewSupport(true);if(sent.at(-1).htmlView!==true)throw Error('capability true ignored');
emit({type:'native-adapter',nativeType:'api-request',requestId:'wiki-fallback',path:'/api/wiki',query:{title:'Bridge image'}});const wiki=window.__whaleOpenUIDrain()[0];if(!wiki||wiki.path!=='/api/wiki'||wiki.query.title!=='Bridge image')throw Error('Image wiki fallback did not reach typed host route');window.__whaleOpenUIAPIResponse({instanceId:instance,requestId:wiki.requestId,status:200,body:{photos:['data:image/png;base64,aGVsbG8=']}});if(sent.at(-1).type!=='api-response'||sent.at(-1).requestId!=='wiki-fallback'||sent.at(-1).status!==200||sent.at(-1).body.photos.length!==1)throw Error('Image wiki fallback reply missing');
console.log('PASS v2 shell state, resources, receipts, export, panel, late cancelled reply, HtmlView capability and image wiki fallback');`
	cmd := exec.Command(node)
	cmd.Stdin = strings.NewReader(fixture)
	processutil.Background(cmd)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("v2 bridge: %v %s", err, out)
	}
}
func TestWhaleUIDownloadValidation(t *testing.T) {
	e := Event{Filename: "report.html", MIME: "text/html", Content: "data:text/plain;base64,aGVsbG8="}
	if b, err := DownloadBytes(e); err != nil || string(b) != "hello" {
		t.Fatal("download content", err)
	}
	for _, name := range []string{"../evil", "x\\evil", "CON.txt", "report.", "a:b", ""} {
		e.Filename = name
		if ValidateDownload(e) == nil {
			t.Fatal("unsafe filename", name)
		}
	}
}
