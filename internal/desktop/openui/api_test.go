package openui

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"os/exec"
	"strings"
	"testing"
)

type resourceTransport func(*http.Request) (*http.Response, error)

func (f resourceTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func TestOpenUIResourceAdapters(t *testing.T) {
	calls := []string{}
	client := &http.Client{Transport: resourceTransport(func(r *http.Request) (*http.Response, error) {
		if r.Header.Get("Authorization") != "" || r.Header.Get("Cookie") != "" {
			t.Fatal("credentials forwarded")
		}
		calls = append(calls, r.URL.String())
		body := ""
		var rawBody []byte
		switch {
		case r.URL.Host == "upload.wikimedia.org":
			rawBody, _ = base64.StdEncoding.DecodeString("iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mNk+A8AAQUBAScY42YAAAAASUVORK5CYII=")
		case r.URL.Host == "en.wikipedia.org" && strings.Contains(r.URL.Path, "summary/"):
			body = `{"thumbnail":{"source":"https://upload.wikimedia.org/photo.jpg"},"coordinates":{"lat":35,"lon":139}}`
		case r.URL.Host == "en.wikipedia.org" && strings.Contains(r.URL.Path, "media-list/"):
			body = `{"items":[{"type":"image","srcset":[{"src":"//upload.wikimedia.org/second.jpg"}]}]}`
		case r.URL.Host == "en.wikipedia.org" && r.URL.Path == "/w/api.php":
			body = `{"query":{"pages":[{"title":"Kyoto temple","index":1,"thumbnail":{"source":"https://upload.wikimedia.org/temple.jpg"}}]}}`
		case r.URL.Host == "commons.wikimedia.org":
			body = `{"query":{"pages":[{"imageinfo":[{"mime":"image/jpeg","thumburl":"https://upload.wikimedia.org/commons.jpg"}]}]}}`
		case r.URL.Host == "router.project-osrm.org":
			body = `{"code":"Ok","routes":[{"geometry":{"coordinates":[[139,35],[140,36]]}}]}`
		default:
			t.Fatalf("unexpected destination %s", r.URL)
		}
		if rawBody == nil {
			rawBody = []byte(body)
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(bytes.NewReader(rawBody)), Header: make(http.Header)}, nil
	})}
	api := &ResourceAPI{client: client}
	id := Identity("h", "s", "m", 0)
	cases := []struct{ path, key, value, want string }{{"/api/wiki", "title", "Kyoto temple", "\"lat\":35"}, {"/api/image-search", "q", "Kyoto temple", "temple.jpg"}, {"/api/street-route", "coords", "139,35;140,36", "[[35,139],[36,140]]"}}
	for _, tc := range cases {
		e := Event{InstanceID: id, RequestID: tc.key, Path: tc.path, Query: map[string]string{tc.key: tc.value}}
		got := api.Handle(context.Background(), e)
		if got.Status != 200 || !strings.Contains(string(got.Body), tc.want) {
			t.Fatalf("%s: %+v %s", tc.path, got, string(got.Body))
		}
		if err := ValidateAPIResponse(got, id); err != nil {
			t.Fatal(err)
		}
	}
	before := len(calls)
	got := api.Handle(context.Background(), Event{InstanceID: id, RequestID: "bad", Path: "https://attacker.invalid", Query: map[string]string{}})
	if got.Status != 400 || len(calls) != before {
		t.Fatal("arbitrary path requested")
	}
}
func TestOpenUIResourceProtocolAndCancel(t *testing.T) {
	id := Identity("h", "s", "m", 0)
	e := Event{Channel: "whale-openui", Version: 1, InstanceID: id, Type: "api-request", RequestID: "1", Path: "/api/wiki", Query: map[string]string{"title": "Synthetic"}}
	raw, _ := json.Marshal(e)
	if _, err := ValidateEvent(raw, id); err != nil {
		t.Fatal(err)
	}
	e.Query["url"] = "https://attacker.invalid"
	raw, _ = json.Marshal(e)
	if _, err := ValidateEvent(raw, id); err == nil {
		t.Fatal("unknown parameter accepted")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	api := &ResourceAPI{client: &http.Client{Transport: resourceTransport(func(r *http.Request) (*http.Response, error) {
		if r.Context().Err() == nil {
			t.Fatal("owner cancellation not propagated")
		}
		return nil, r.Context().Err()
	})}}
	got := api.Handle(ctx, Event{InstanceID: id, RequestID: "cancel", Path: "/api/wiki", Query: map[string]string{"title": "Synthetic"}})
	if got.Status != 502 {
		t.Fatal("canceled lookup reported success")
	}
	if ValidateAPIResponse(APIResponse{InstanceID: "other", RequestID: "1", Status: 200, Body: json.RawMessage(`{}`)}, id) == nil {
		t.Fatal("foreign reply accepted")
	}
}

func TestOpenUIResourceShellOwnership(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("Node unavailable")
	}
	shell, err := Shell("<html>fixture</html>", Render{InstanceID: Identity("h", "s", "m", 0), Program: "fixture", State: json.RawMessage(`{}`)}, 900, "light")
	if err != nil {
		t.Fatal(err)
	}
	script := strings.Split(strings.Split(shell, "<script>")[1], "</script>")[0]
	fixture := `global.window=global;const listeners={};global.addEventListener=(n,f)=>listeners[n]=f;const sent=[];const fakeFrame={contentWindow:{postMessage:m=>sent.push(m)},setAttribute:()=>{}};global.document={createElement:()=>fakeFrame,body:{append:()=>{}}};` + script + `
 const emit=d=>listeners.message({source:frame.contentWindow,origin:'null',data:{channel:'whale-openui',version:1,instanceId:instance,...d}});
 emit({type:'api-request',requestId:'wiki-1',path:'/api/wiki',query:{title:'Synthetic'}});
 const q=window.__whaleOpenUIDrain();if(q.length!==1||q[0].requestId!=='wiki-1')throw Error('request not relayed');
 if(window.__whaleOpenUIAPIResponse({instanceId:instance,requestId:'wiki-1',status:200,body:{photos:[]}})!==true)throw Error('reply failed');
 if(sent.at(-1).type!=='api-response'||sent.at(-1).requestId!=='wiki-1')throw Error('wrong reply');
 for(const r of [{instanceId:instance,requestId:'wiki-1'},{instanceId:'foreign',requestId:'wiki-2'}]){let rejected=false;try{window.__whaleOpenUIAPIResponse({...r,status:200,body:{}})}catch{rejected=true}if(!rejected)throw Error('unknown/duplicate reply accepted')}
 emit({type:'api-request',requestId:'evil',path:'https://attacker.invalid',query:{}});if(window.__whaleOpenUIDrain().length!==0||sent.at(-1).status!==400)throw Error('arbitrary route forwarded');`
	if out, err := exec.Command(node, "-e", fixture).CombinedOutput(); err != nil {
		t.Fatalf("shell ownership: %v %s", err, out)
	}
}
