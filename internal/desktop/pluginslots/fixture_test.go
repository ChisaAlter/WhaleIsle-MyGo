package pluginslots

import (
	"context"
	"github.com/coder/websocket"
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
)

// One temporary fixture covers the packaged consumer and its exact-Host transport.
func TestPluginSlotFixture(t *testing.T) {
	host := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Cookie") != "dsh-auth-fixture=secret" {
			t.Errorf("Host auth missing")
		}
		if r.URL.Path == "/socket" {
			connection, err := websocket.Accept(w, r, &websocket.AcceptOptions{InsecureSkipVerify: true})
			if err != nil {
				return
			}
			defer connection.CloseNow()
			ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
			defer cancel()
			for {
				kind, data, err := connection.Read(ctx)
				if err != nil {
					return
				}
				if err = connection.Write(ctx, kind, data); err != nil {
					return
				}
			}
		}
		if r.URL.Path == "/escape" {
			http.Redirect(w, r, "https://example.invalid/", http.StatusFound)
			return
		}
		if r.Header.Get("Origin") != "" && r.Header.Get("Origin") != "http://"+r.Host {
			t.Errorf("wrong Host Origin")
		}
		_, _ = io.WriteString(w, r.URL.RequestURI())
	}))
	defer host.Close()
	gateway, err := NewGateway(host.URL, "dsh-auth-fixture=secret")
	if err != nil {
		t.Fatal(err)
	}
	defer gateway.Close()
	entry, err := gateway.EntryURL(host.URL + "/?desktopPlugin=fixture")
	if err != nil {
		t.Fatal(err)
	}
	jar, _ := cookiejar.New(nil)
	browser := &http.Client{Jar: jar}
	response, err := browser.Get(entry)
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(response.Body)
	response.Body.Close()
	if string(body) != "/?desktopPlugin=fixture" || response.StatusCode != 200 {
		t.Fatalf("slot URL/auth flow: %d %s", response.StatusCode, body)
	}
	if strings.Contains(entry, "secret") {
		t.Fatal("Host credential in URL")
	}
	logical := gateway.LogicalURL(response.Request.URL.String())
	if logical != host.URL+"/?desktopPlugin=fixture" {
		t.Fatalf("logical URL: %s", logical)
	}
	direct, _ := http.Get(response.Request.URL.String())
	direct.Body.Close()
	if direct.StatusCode != http.StatusForbidden {
		t.Fatalf("unauthenticated proxy %d", direct.StatusCode)
	}
	req, _ := http.NewRequest("GET", response.Request.URL.String(), nil)
	req.Header.Set("Origin", "https://attacker.invalid")
	denied, _ := browser.Do(req)
	denied.Body.Close()
	if denied.StatusCode != http.StatusForbidden {
		t.Fatalf("cross origin %d", denied.StatusCode)
	}
	if _, err = gateway.EntryURL("https://attacker.invalid/"); err == nil {
		t.Fatal("foreign target accepted")
	}
	escaped, _ := browser.Get(strings.Split(response.Request.URL.String(), "?")[0] + "escape")
	escaped.Body.Close()
	if escaped.StatusCode != http.StatusBadGateway {
		t.Fatalf("external redirect not blocked: %d", escaped.StatusCode)
	}
	if !gateway.Matches(host.URL, "dsh-auth-fixture=secret") || gateway.Matches(host.URL, "changed") || gateway.Matches(host.URL+"0", "dsh-auth-fixture=secret") {
		t.Fatal("stale Host session reused")
	}
	socketURL := *response.Request.URL
	socketURL.Path = "/socket"
	socketURL.RawQuery = ""
	socketURL.Scheme = "ws"
	headers := http.Header{}
	for _, cookie := range jar.Cookies(response.Request.URL) {
		headers.Add("Cookie", cookie.Name+"="+cookie.Value)
	}
	socket, _, err := websocket.Dial(context.Background(), socketURL.String(), &websocket.DialOptions{HTTPHeader: headers})
	if err != nil {
		t.Fatal(err)
	}
	if err = socket.Write(context.Background(), websocket.MessageText, []byte("fixture")); err != nil {
		t.Fatal(err)
	}
	_, echoed, err := socket.Read(context.Background())
	if err != nil || string(echoed) != "fixture" {
		t.Fatalf("socket forwarding: %s %v", echoed, err)
	}
	gateway.Close()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if _, _, err = socket.Read(ctx); err == nil {
		t.Fatal("upgraded connection survived close")
	}
	if _, err = browser.Get(entry); err == nil {
		t.Fatal("browser capability survived close")
	}
	profile := t.TempDir()
	patch, err := Prepare(profile)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = os.Stat(patch); err != nil {
		t.Fatal(err)
	}
	script := filepath.Join(profile, "node_modules", PackageName, "client.js")
	fixture := filepath.Join(profile, "fixture.cjs")
	if err = os.WriteFile(fixture, []byte(nodeFixture), 0600); err != nil {
		t.Fatal(err)
	}
	node, err := exec.LookPath("node")
	if err != nil {
		t.Fatal(err)
	}
	if output, err := exec.Command(node, fixture, script).CombinedOutput(); err != nil {
		t.Fatalf("consumer fixture: %s\n%v", output, err)
	}
}

const nodeFixture = `
const assert=require('node:assert/strict'),fs=require('node:fs'),vm=require('node:vm');
let exported,paint,changed,selected=[],settings=[],disposed=false;
const entries={main:[{options:{key:'custom-view',registrant:'fixture',label:'Custom view'}}], 'settings.section':[], 'sidebar.page':[], 'sidebar.nav.tab':[], 'settings.remote.tab':[]};
const react={createElement(){return null},useState(v){return [v,()=>{}]},useEffect(){},useSyncExternalStore(){return 0}};
let buttons=[],cards=[];const document={documentElement:{},querySelectorAll(selector){return selector==='[data-plugin-item]'?cards:buttons}};
const context={URL,URLSearchParams,location:{search:'?desktopPlugin=fixture'},document,console,queueMicrotask,MutationObserver:class{observe(){} disconnect(){}},setTimeout,clearTimeout,window:{__ModuleLoader__:{load(row){exported=row.factory(id=>react)}}}};
vm.runInNewContext(fs.readFileSync(process.argv[2],'utf8'),context);
const ctx={slots:{entriesOfSlot(k){return entries[k]||[]},entries(k){return entries[k]||[]},inject(k,fn){return fn()},register(o,c){paint=c;return ()=>{disposed=true}},snapshot(){return []}},layout:{selectPanel(id){selected.push(id)}},settingsNavigation:{open(id){settings.push(id)}},effect(fn){this.dispose=fn()},on(k,fn){changed=fn;return ()=>{}},get(k){return this[k]}};
exported.apply(ctx);
assert.deepEqual(selected,['custom-view']);assert.equal(typeof paint,'function');
changed('main');assert.deepEqual(selected,['custom-view']);
entries.main=[];changed('main');
ctx.dispose();assert.equal(disposed,true);
context.location.search='?desktopPlugin=fixture';entries['settings.section']=[{options:{id:'plugin-settings',registrant:'fixture'}}];
exported.apply(ctx);assert.deepEqual(settings,['plugin-settings']);ctx.dispose();
entries['settings.section']=[];entries['settings.remote.tab']=[{options:{id:'channels',registrant:'fixture'}}];
let clicks=0;buttons=[{getAttribute(name){return name==='data-dsh-settings-remote-tab'?'channels':null},click(){clicks++}}];
exported.apply(ctx);assert.equal(settings.at(-1),'remote');assert.equal(clicks,1);ctx.dispose();
entries['settings.remote.tab']=[];entries['sidebar.page']=[{options:{key:'bots',registrant:'fixture'}}];entries['sidebar.nav.tab']=[{options:{id:'bots',label:()=> 'Bots'}}];
buttons=[{getAttribute(name){return name==='aria-label'?'Bots':null},hasAttribute(){return false},click(){clicks++}}];
exported.apply(ctx);assert.equal(clicks,2);ctx.dispose();
entries['sidebar.page']=[];entries['settings.section']=[{options:{id:'general',registrant:'host',children:{'settings.general.item':{}}}}];entries['settings.general.item']=[{options:{id:'custom',registrant:'fixture'}}];
exported.apply(ctx);assert.equal(settings.at(-1),'general');ctx.dispose();
entries['settings.section']=[];entries['settings.general.item']=[];
entries.main=[{options:{key:'plugins',registrant:'manager',children:{'plugins.bundle.config':{},'plugins.row.config':{},'plugins.item':{}}}}];
entries['plugins.bundle.config']=[{options:{key:'fixture-bundle',registrant:'fixture'}}];
const openedBundles=[];ctx.pluginNavigation={openBundle(name){openedBundles.push(name)}};
exported.apply(ctx);assert.deepEqual(openedBundles,['fixture-bundle']);changed('plugins.bundle.config');assert.equal(openedBundles.length,1);ctx.dispose();
entries['plugins.bundle.config']=[];entries['plugins.row.config']=[{options:{key:'fixture-bundle#worker',registrant:'fixture'}}];
exported.apply(ctx);assert.equal(openedBundles.at(-1),'fixture-bundle');ctx.dispose();
entries['plugins.row.config']=[];entries['plugins.item']=[{options:{id:'official-settings',registrant:'fixture',label:'Official settings'}}];
cards=[{getAttribute(){return 'official-settings'},querySelectorAll(){return [{click(){clicks++}}]}}];
exported.apply(ctx);assert.equal(selected.at(-1),'plugins');assert.equal(clicks,3);
entries['plugins.item']=[];changed('plugins.item');assert.equal(clicks,3);ctx.dispose();
entries.main=[{options:{key:'custom-owner',registrant:'host',children:{'custom.settings':{}}}}];
entries['custom.settings']=[{options:{id:'contribution',registrant:'fixture'}}];
exported.apply(ctx);assert.equal(selected.at(-1),'custom-owner');ctx.dispose();
console.log('main/settings/plugin configuration owners, single activation and disposal passed');
`
