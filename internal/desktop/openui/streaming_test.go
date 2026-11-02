package openui

import (
	"encoding/json"
	"os/exec"
	"strings"
	"testing"
	"whaleisle/internal/processutil"
)

func TestOpenUIStreamingShellPreservesState(t *testing.T) {
	node, e := exec.LookPath("node")
	if e != nil {
		t.Skip("Node unavailable")
	}
	id := Identity("host", "session", "message", 0)
	shell, e := Shell("<html>fixture</html>", Render{InstanceID: id, Program: "first", Streaming: true, State: json.RawMessage(`{}`)}, 900, "light")
	if e != nil {
		t.Fatal(e)
	}
	script := strings.Split(strings.Split(shell, "<script>")[1], "</script>")[0]
	fixture := `global.window=global;const listeners={};global.addEventListener=(n,f)=>listeners[n]=f;let frames=0;const sent=[];const fakeFrame={contentWindow:{postMessage:m=>sent.push(JSON.parse(JSON.stringify(m)))},setAttribute:()=>{}};global.document={createElement:()=>{frames++;return fakeFrame},body:{append:()=>{}}};` + script + `
listeners.message({source:frame.contentWindow,origin:'null',data:{channel:'whale-openui',version:1,instanceId:instance,type:'ready'}});
listeners.message({source:frame.contentWindow,origin:'null',data:{channel:'whale-openui',version:1,instanceId:instance,type:'state',state:{typed:'retained'}}});
window.__whaleOpenUIRender({instanceId:instance,program:'second',streaming:false,state:{typed:'wrong'}});
const last=sent.at(-1);if(frames!==1||last.program!=='second'||last.streaming!==false||last.state.typed!=='retained')throw Error('renderer reloaded or state reset');
let rejected=false;try{window.__whaleOpenUIRender({instanceId:'foreign',program:'bad',streaming:false})}catch{rejected=true}if(!rejected)throw Error('foreign update accepted');`
	cmd := exec.Command(node, "-e", fixture)
	processutil.Background(cmd)
	if out, e := cmd.CombinedOutput(); e != nil {
		t.Fatalf("streaming fixture: %v %s", e, out)
	}
}
