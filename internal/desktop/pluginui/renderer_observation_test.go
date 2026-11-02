package pluginui

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strconv"

	"path/filepath"
	"strings"
	"testing"
	"whaleisle/internal/processutil"
)

// The installed renderer is an explicit read-only behavioral input for this
// opt-in. Normal package tests use the minimal exact fragment below.
func observationSource(t *testing.T) []byte {
	t.Helper()
	if root := os.Getenv("WHALEISLE_PLUGIN_UI_RENDERER"); root != "" {
		source, err := os.ReadFile(filepath.Clean(root))
		if err != nil {
			t.Fatal(err)
		}
		return source
	}
	return []byte(`window.__ModuleLoader__.load({ id: "@deepseek-ai/dsh-client-ui-renderer", factory() {` + "\n\t\tfunction SlotOutlet({ slotKey, ownerProps, opts, declaredInline }) {\n}\n" + `
		function renderOutletContent(host, slotKey, ownerProps, opts, scopeBinding, spec) {
			const guarded = (entry, key, owner = ownerProps) => {
				return spec.scope === "session" ? (0, react_jsx_runtime.jsx)(StrictSessionEntry, {entry},key) : (0, react_jsx_runtime.jsx)(SlotErrorBoundary, {
					children: entry.component
				}, key);
			};
		}
		/** Root outlet: untouched fallback */
}});`)
}

func TestSessionObservationPreservesUnknownVersionsAndOriginalDispatch(t *testing.T) {
	source := observationSource(t)
	observed, ok := observeRenderer(source)
	if !ok {
		t.Fatal("known renderer was not recognized")
	}
	if bytes.Count(observed, []byte("return whaleSessionObserveEntry(Anchor, slotKey, entry, key, owner,")) != 1 {
		t.Fatal("actual guarded entry is not observed exactly once")
	}
	if !bytes.Contains(observed, []byte(`{key, "data-whale-slot-entry"`)) {
		t.Fatal("entry marker lost original key")
	}
	if out, ok := observeRenderer(observed); ok || !bytes.Equal(out, observed) {
		t.Fatal("second observation changes lifecycle")
	}
	unknown := bytes.Replace(source, []byte("StrictSessionEntry"), []byte("NewSessionEntry"), -1)
	if out, ok := observeRenderer(unknown); ok || !bytes.Equal(out, unknown) {
		t.Fatal("unknown renderer modified")
	}
}

func TestSessionObservationUsesCanonicalTurnTailDataWithoutCallingEntry(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Fatal(err)
	}
	script := `const vm=require('node:vm'),assert=require('node:assert/strict');
const context={URLSearchParams,location:{search:'?desktopNativeSession=1'},window:{},react:{createElement:(tag,props,content)=>({tag,props,content})}};
vm.createContext(context);vm.runInContext(` + strconv.Quote(rendererPrelude) + `,context);
function wrap(owner,slot='conversation.chat.turnTail'){
 const entry={};const result=context.whaleSessionObserveEntry('div',slot,entry,'original-key',owner,'original-result');
 assert.equal(result.content,'original-result');assert.equal(result.props.key,'original-key');assert.equal(result.props.style.display,'contents');
 return result.props;
}
const turn={turn:0,data:new Map([['turn-tail',{turn:0,seq:19}]])};
let p=wrap({turn,seq:17});assert.equal(p['data-whale-turn-tail-turn'],0);assert.equal(p['data-whale-turn-tail-end-seq'],19);
for(const owner of [null,{}, {seq:17,turn:{turn:0,data:new Map()}}, {turn:{turn:1,data:new Map([['turn-tail',{turn:0,seq:19}]])}}, {turn:{turn:0,data:{get(){throw Error('missing')}}}}, {turn:{turn:0,data:new Map([['turn-tail',{turn:0,seq:1.5}]])}}]){
 p=wrap(owner);assert.equal(p['data-whale-turn-tail-end-seq'],undefined);
}
p=wrap({turn,seq:17},'conversation.chat.assistant-actions');assert.equal(p['data-whale-turn-tail-end-seq'],undefined);
console.log('Canonical original turn-tail turn/seq retained; owner.seq and other slots refused');`
	cmd := exec.Command(node, "-e", script)
	processutil.Background(cmd)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("canonical renderer metadata: %v\n%s", err, out)
	} else {
		t.Log(string(out))
	}
}

func TestSessionRendererResponseKeepsEncodedUnknownAndCSPPages(t *testing.T) {
	known := observationSource(t)
	for _, tc := range []struct {
		name, encoding, csp string
		known               bool
	}{{"plain", "", "", true}, {"gzip", "gzip", "", true}, {"csp", "", "script-src 'self'", true}, {"unsupported", "br", "", true}, {"unknown", "gzip", "", false}} {
		t.Run(tc.name, func(t *testing.T) {
			source := known
			if !tc.known {
				source = []byte("untouched future renderer")
			}
			raw := source
			if tc.encoding == "gzip" {
				var b bytes.Buffer
				z := gzip.NewWriter(&b)
				_, _ = z.Write(source)
				_ = z.Close()
				raw = b.Bytes()
			}
			r := &http.Response{StatusCode: 200, Request: &http.Request{URL: &url.URL{Path: "/plugins/"}}, Header: http.Header{"Content-Type": []string{"text/javascript"}}, Body: io.NopCloser(bytes.NewReader(raw)), ContentLength: int64(len(raw))}
			r.Header.Set("Content-Encoding", tc.encoding)
			r.Header.Set("Content-Security-Policy", tc.csp)
			if err := sessionRendererResponse(r); err != nil {
				t.Fatal(err)
			}
			out, _ := io.ReadAll(r.Body)
			_ = r.Body.Close()
			changed := tc.known && tc.csp == "" && (tc.encoding == "" || tc.encoding == "gzip")
			if changed {
				if !strings.Contains(string(out), "whaleSessionObserveEntry") || r.Header.Get("Content-Encoding") != "" {
					t.Fatal("observed body or encoding incorrect")
				}
			} else if !bytes.Equal(out, raw) || r.Header.Get("Content-Encoding") != tc.encoding {
				t.Fatal("incompatible response changed")
			}
		})
	}
}
