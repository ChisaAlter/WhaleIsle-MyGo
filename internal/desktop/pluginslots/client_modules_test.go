package pluginslots

import (
	"context"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

func TestClientModulesPageBridge(t *testing.T) {
	host := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/__whale_client_modules") {
			t.Error("client bridge was forwarded to Host")
		}
		_, _ = io.WriteString(w, "page")
	}))
	defer host.Close()
	gateway, err := NewGateway(host.URL, "host-session=secret")
	if err != nil {
		t.Fatal(err)
	}
	defer gateway.Close()
	entry, _ := gateway.EntryURL(host.URL + "/")
	jar, _ := cookiejar.New(nil)
	browser := &http.Client{Jar: jar}
	response, err := browser.Get(entry)
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	socketURL := *response.Request.URL
	socketURL.Path = "/__whale_client_modules"
	socketURL.Scheme = "ws"
	headers := http.Header{"Origin": {gateway.browserOrigin}}
	for _, cookie := range jar.Cookies(response.Request.URL) {
		headers.Add("Cookie", cookie.Name+"="+cookie.Value)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if conn, resp, err := websocket.Dial(ctx, socketURL.String(), nil); err == nil {
		conn.CloseNow()
		t.Fatal("unauthenticated bridge accepted")
	} else if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("unauthenticated bridge: %v", err)
	}
	foreign := headers.Clone()
	foreign.Set("Origin", "https://attacker.invalid")
	if conn, _, err := websocket.Dial(ctx, socketURL.String(), &websocket.DialOptions{HTTPHeader: foreign}); err == nil {
		conn.CloseNow()
		t.Fatal("foreign bridge accepted")
	}
	connect := func() *websocket.Conn {
		t.Helper()
		c, _, err := websocket.Dial(ctx, socketURL.String(), &websocket.DialOptions{HTTPHeader: headers, Subprotocols: []string{"whaleisle.client-modules.v1"}})
		if err != nil {
			t.Fatal(err)
		}
		return c
	}
	wait := func(check func(ClientModulesSnapshot) bool) ClientModulesSnapshot {
		t.Helper()
		for {
			s := gateway.ClientModules()
			if check(s) {
				return s
			}
			select {
			case <-ctx.Done():
				t.Fatalf("bridge state: %+v", s)
				return s
			case <-time.After(time.Millisecond):
			}
		}
	}
	if gateway.ClientModules().Connected || gateway.RetryClientModules(0) == nil {
		t.Fatal("no page presented as connected/retryable")
	}
	first := connect()
	defer first.CloseNow()
	if err := wsjson.Write(ctx, first, map[string]any{"type": "state", "syncing": false, "failures": []map[string]string{{"id": "broken", "message": "apply failed"}}}); err != nil {
		t.Fatal(err)
	}
	snap := wait(func(s ClientModulesSnapshot) bool { return s.Connected && len(s.Failures) == 1 })
	snap.Failures[0].Message = "mutated"
	if gateway.ClientModules().Failures[0].Message != "apply failed" {
		t.Fatal("mutable snapshot alias")
	}
	if err := gateway.RetryClientModules(snap.Page); err != nil {
		t.Fatal(err)
	}
	var command struct {
		Type string `json:"type"`
		ID   uint64 `json:"id"`
	}
	if err := wsjson.Read(ctx, first, &command); err != nil || command.Type != "retry" || command.ID == 0 {
		t.Fatalf("retry command: %+v %v", command, err)
	}
	if gateway.RetryClientModules(snap.Page) == nil {
		t.Fatal("duplicate retry accepted")
	}
	_ = wsjson.Write(ctx, first, map[string]any{"type": "state", "syncing": true, "failures": []any{}})
	wait(func(s ClientModulesSnapshot) bool { return s.Syncing && s.Retrying })
	_ = wsjson.Write(ctx, first, map[string]any{"type": "retryResult", "id": command.ID, "error": "retry rejected"})
	wait(func(s ClientModulesSnapshot) bool { return !s.Retrying && s.Error == "retry rejected" })
	second := connect()
	defer second.CloseNow()
	_ = wsjson.Write(ctx, second, map[string]any{"type": "state", "syncing": false, "failures": []any{}})
	current := wait(func(s ClientModulesSnapshot) bool { return s.Connected && s.Page != snap.Page })
	if len(current.Failures) != 0 || current.Error != "" || current.Retrying {
		t.Fatalf("old page leaked: %+v", current)
	}
	if gateway.RetryClientModules(snap.Page) == nil {
		t.Fatal("stale native callback retried replacement page")
	}
	_ = wsjson.Write(ctx, first, map[string]any{"type": "state", "syncing": false, "failures": []map[string]string{{"id": "stale", "message": "stale"}}})
	if gateway.ClientModules().Page != current.Page || len(gateway.ClientModules().Failures) != 0 {
		t.Fatal("old socket changed active page")
	}
	gateway.DisconnectClientModules()
	wait(func(s ClientModulesSnapshot) bool { return !s.Connected && len(s.Failures) == 0 && !s.Retrying })
	if gateway.RetryClientModules(current.Page) == nil {
		t.Fatal("disposed page retry accepted")
	}
	third := connect()
	defer third.CloseNow()
	_ = wsjson.Write(ctx, third, map[string]any{"type": "state", "syncing": false, "failures": []any{}})
	wait(func(s ClientModulesSnapshot) bool { return s.Connected })
	gateway.Close()
	if gateway.ClientModules().Connected {
		t.Fatal("closed Host retained page state")
	}
}

func TestClientModulesAdapterLifecycle(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, "client.js")
	fixture := filepath.Join(dir, "test.cjs")
	if err := os.WriteFile(script, clientSource, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(fixture, []byte(clientModulesNodeFixture), 0600); err != nil {
		t.Fatal(err)
	}
	node, err := exec.LookPath("node")
	if err != nil {
		t.Fatal(err)
	}
	if output, err := exec.Command(node, fixture, script).CombinedOutput(); err != nil {
		t.Fatalf("page adapter: %s\n%v", output, err)
	}
}

// The real adapter runs against the public entries contract, without a
// desktopPlugin query. Removing its store subscription or calling Host refresh
// instead of entries.retry makes these observable assertions fail.
const clientModulesNodeFixture = `
const assert=require('node:assert/strict'),fs=require('node:fs'),vm=require('node:vm');
(async()=>{
let exported,retries=0,disposed=0,state={syncing:false,failures:[{id:'bad',message:'import failed'}]},listener,release;
const effects=[],sockets=[],events=new Map();
class Socket {
 static OPEN=1; static CLOSED=3;
 constructor(url,protocol){this.url=url;this.protocol=protocol;this.readyState=0;this.sent=[];this.events={};sockets.push(this)}
 addEventListener(name,fn){this.events[name]=fn}
 send(data){this.sent.push(JSON.parse(data))}
 open(){this.readyState=1;this.events.open()}
 message(data){return this.events.message({data:JSON.stringify(data)})}
 close(){this.readyState=3;this.events.close?.()}
}
const context={URL,URLSearchParams,location:{search:'',href:'http://127.0.0.1:5555/',origin:'http://127.0.0.1:5555'},WebSocket:Socket,
 window:{addEventListener(n,f){events.set(n,f)},removeEventListener(n){events.delete(n)},__ModuleLoader__:{load(row){exported=row.factory(()=>({createElement(){}}))}}}};
vm.runInNewContext(fs.readFileSync(process.argv[2],'utf8'),context);
const ctx={modules:{entries:{state:{getSnapshot(){return state},subscribe(fn){listener=fn;return ()=>{disposed++;listener=null}}},retry(){retries++;state={syncing:true,failures:state.failures};listener();return new Promise(resolve=>{release=()=>{state={syncing:false,failures:[]};listener?.();resolve()}})}}},effect(fn){effects.push(fn())}};
exported.apply(ctx);assert.equal(sockets.length,1,'ordinary active Host page must bind before desktopPlugin guard');
const socket=sockets[0];assert.equal(socket.url,'ws://127.0.0.1:5555/__whale_client_modules');assert.equal(socket.protocol,'whaleisle.client-modules.v1');
socket.open();assert.deepEqual(socket.sent.at(-1),{type:'state',syncing:false,failures:[{id:'bad',message:'import failed'}]});
state={syncing:false,failures:[{id:'new-graph',message:'latest failure'}]};listener();assert.equal(socket.sent.at(-1).failures[0].id,'new-graph');
const pending=socket.message({type:'retry',id:1});assert.equal(retries,1);assert.equal(socket.sent.at(-1).syncing,true);
await socket.message({type:'retry',id:2});assert.equal(retries,1,'duplicate retry must not run');
release();await pending;assert.deepEqual(socket.sent.at(-1),{type:'retryResult',id:1,error:''});assert.equal(socket.sent.at(-2).failures.length,0);
await socket.message({type:'hostRpc',method:'plugins/toggle'});assert.equal(retries,1,'only page-local retry is accepted');
const pending2=socket.message({type:'retry',id:3});effects.forEach(dispose=>dispose());const count=socket.sent.length;release();await pending2;
assert.equal(disposed,1);assert.equal(socket.readyState,Socket.CLOSED);assert.equal(socket.sent.length,count,'disposed retry must not report');assert.equal(events.has('pagehide'),false);
console.log('page-local state, latest retry, duplicate control, disposal and guard passed');
})().catch(err=>{console.error(err);process.exitCode=1});
`
