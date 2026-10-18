package pluginrpc

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

type whaleSettingsTransport func(*http.Request) (*http.Response, error)

func (f whaleSettingsTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func TestWhaleSettingsRPCExactContractAndNoRetry(t *testing.T) {
	methods := []string{}
	fail := false
	c := &Client{BaseURL: "http://127.0.0.1", Cookie: "synthetic", HTTPClient: &http.Client{Transport: whaleSettingsTransport(func(r *http.Request) (*http.Response, error) {
		var req clientRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatal(err)
		}
		methods = append(methods, req.Method)
		if r.URL.Path != "/dsh-whale/"+req.Method || r.Header.Get("Cookie") != "synthetic" {
			t.Fatalf("wrong endpoint %s", r.URL)
		}
		p := req.Payload.(map[string]any)
		var value any
		switch req.Method {
		case "catalog":
			if len(p) != 0 {
				t.Fatal(p)
			}
			value = WhaleSettings{HomeDir: "/fixture", Memory: "memory", Name: "Whale"}
		case "memory/replace":
			if len(p) != 1 || p["text"] != "hello 🐳" {
				t.Fatal(p)
			}
			value = map[string]any{"ok": true, "chars": 8}
		case "memory/clear":
			if len(p) != 0 {
				t.Fatal(p)
			}
			value = map[string]any{"ok": true}
		case "settings/update":
			if len(p) != 1 || p["personaText"] != "persona" {
				t.Fatal(p)
			}
			value = WhaleSettings{PersonaText: "persona"}
		default:
			t.Fatal(req.Method)
		}
		if fail {
			value = map[string]any{"ok": false, "error": "synthetic refusal"}
		}
		raw, _ := json.Marshal(map[string]any{"result": map[string]any{"ok": true, "value": value}})
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(string(raw)))}, nil
	})}}
	ctx := context.Background()
	if _, e := c.WhaleSettings(ctx); e != nil {
		t.Fatal(e)
	}
	if e := c.WhaleReplaceMemory(ctx, "hello 🐳"); e != nil {
		t.Fatal(e)
	}
	if e := c.WhaleClearMemory(ctx); e != nil {
		t.Fatal(e)
	}
	if _, e := c.WhaleUpdateSettings(ctx, WhaleSettingsPatch{PersonaText: ptrWhale("persona")}); e != nil {
		t.Fatal(e)
	}
	fail = true
	if e := c.WhaleClearMemory(ctx); e == nil {
		t.Fatal("nested refusal accepted")
	}
	if len(methods) != 5 {
		t.Fatalf("retried request: %v", methods)
	}
	before := len(methods)
	if e := c.WhaleReplaceMemory(ctx, strings.Repeat("🐳", 20001)); e == nil {
		t.Fatal("silent truncation allowed")
	}
	if len(methods) != before {
		t.Fatal("invalid content transmitted")
	}
}
func ptrWhale(v string) *string { return &v }
