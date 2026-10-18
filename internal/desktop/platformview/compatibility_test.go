package platformview

import (
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"os/exec"
	"strings"
	"testing"
)

func TestPlatformDeploymentCompatibility(t *testing.T) {
	policy := "default-src 'self'; script-src 'self' https://cdn.example; frame-src https://pay.example"
	upstream := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Deployment") != "fixture-header" {
			t.Error("missing scoped header")
		}
		w.Header().Set("Content-Security-Policy", policy)
		if r.URL.Path == "/redirect" {
			w.Header().Set("Location", "https://pay.example/checkout")
			w.WriteHeader(302)
			return
		}
		w.Write([]byte("fixture"))
	}))
	defer upstream.Close()
	g, err := New(Session{Origin: upstream.URL, Token: "fixture-token", RequestHeaders: map[string]string{"X-Deployment": "fixture-header"}}, ClientMetadata{})
	if err != nil {
		t.Fatal(err)
	}
	defer g.Close()
	g.transport.TLSClientConfig = upstream.Client().Transport.(*http.Transport).TLSClientConfig.Clone()
	jar, _ := cookiejar.New(nil)
	client := &http.Client{Jar: jar}
	entry, _ := g.Entry("usage")
	r, err := client.Get(entry)
	if err != nil {
		t.Fatal(err)
	}
	r.Body.Close()
	if got := r.Header.Values("Content-Security-Policy"); len(got) != 1 || got[0] != policy {
		t.Fatal("trusted deployment CSP changed", got)
	}
	r, err = client.Get(g.origin + "/redirect")
	if err != nil {
		t.Fatal(err)
	}
	r.Body.Close()
	if r.StatusCode != 502 {
		t.Fatal("cross-origin same-tab redirect admitted")
	}
	if raw, ok := g.ExternalURL("https://pay.example/checkout?order=synthetic"); !ok || raw != "https://pay.example/checkout?order=synthetic" {
		t.Fatal("clean popup rejected")
	}
	for _, raw := range []string{"http://pay.example/", "https://u:p@pay.example/", entry, "https://pay.example/?token=fixture%2Dtoken", "javascript:alert(1)"} {
		if _, ok := g.ExternalURL(raw); ok {
			t.Fatal("unsafe popup admitted")
		}
	}
	if raw, ok := g.ExternalURL(g.origin + "/top_up"); !ok || raw != upstream.URL+"/top_up" {
		t.Fatal("account popup was not deproxied")
	}
	if g.Allows("https://pay.example/") {
		t.Fatal("external navigation admitted")
	}
}

func TestPlatformPreloadExternalHeaders(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node unavailable for JavaScript fixture")
	}
	g := &Gateway{origin: "http://127.0.0.1:12345", session: Session{Origin: "https://account.example", Token: "fixture-token", RequestHeaders: map[string]string{"X-Deployment": "fixture"}}}
	script := `global.window=global;window.top=window;global.location={origin:'http://127.0.0.1:12345',href:'http://127.0.0.1:12345/usage'};global.addEventListener=()=>{};let sent;window.fetch=async r=>{sent=r;return {}};global.XMLHttpRequest=class {open(m,u){this.url=u;}setRequestHeader(n,v){this.headers??={};this.headers[n]=v;}};` + g.Preload("en") + `
(async()=>{await fetch('https://cdn.example/data',{headers:{Authorization:'Bearer fixture-token','X-Deployment':'fixture',Accept:'application/json'}});if(sent.headers.has('Authorization')||sent.headers.has('X-Deployment')||sent.credentials!=='omit'||sent.headers.get('Accept')!=='application/json')throw Error('external header leak');await fetch('https://account.example/api',{headers:{Authorization:'Bearer fixture-token'}});if(sent.url!=='http://127.0.0.1:12345/api'||!sent.headers.has('Authorization'))throw Error('account rewrite failed');const x=new XMLHttpRequest();x.open('GET','https://cdn.example/a');x.setRequestHeader('Authorization','fixture-token');x.setRequestHeader('X-Deployment','fixture');if(x.headers)throw Error('XHR leaked');})().catch(e=>{console.error(e.message);process.exitCode=1});`
	if output, err := exec.Command(node, "-e", script).CombinedOutput(); err != nil {
		t.Fatalf("preload fixture: %v %s", err, strings.TrimSpace(string(output)))
	}
}
