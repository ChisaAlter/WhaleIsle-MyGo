package pluginrpc

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCallWhaleEnsureAndPetChat(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/dsh-whale/assistant/ensure", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method %s", r.Method)
		}
		if r.Header.Get("Cookie") != "dsh-auth=1" {
			t.Fatalf("cookie = %q", r.Header.Get("Cookie"))
		}
		var req clientRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatal(err)
		}
		if req.Type != "client-request" || req.Method != "assistant/ensure" {
			t.Fatalf("req = %+v", req)
		}
		_ = json.NewEncoder(w).Encode(serverResponse{
			Type:  "server-response",
			RPCID: req.RPCID,
			Result: mustRaw(t, resultEnvelope{
				OK:    true,
				Value: mustRaw(t, WhaleEnsureResult{OK: true, SessionID: "session-whale-1", Created: false}),
			}),
		})
	})
	mux.HandleFunc("/dsh-whale/pet/chat", func(w http.ResponseWriter, r *http.Request) {
		var req clientRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatal(err)
		}
		payload, _ := json.Marshal(req.Payload)
		if !strings.Contains(string(payload), "hello") {
			t.Fatalf("payload = %s", payload)
		}
		_ = json.NewEncoder(w).Encode(serverResponse{
			Type:  "server-response",
			RPCID: req.RPCID,
			Result: mustRaw(t, resultEnvelope{
				OK:    true,
				Value: mustRaw(t, WhalePetChatResult{OK: true, Reply: "hi there"}),
			}),
		})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	c := &Client{BaseURL: srv.URL, Cookie: "dsh-auth=1"}
	ens, err := c.WhaleEnsureTyped(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if ens.SessionID != "session-whale-1" {
		t.Fatalf("session = %q", ens.SessionID)
	}
	chat, err := c.WhalePetChatTyped(context.Background(), "hello")
	if err != nil {
		t.Fatal(err)
	}
	if chat.Reply != "hi there" {
		t.Fatalf("reply = %q", chat.Reply)
	}
}

func TestWhaleEnsureNestedFailure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req clientRequest
		_ = json.NewDecoder(r.Body).Decode(&req)
		_ = json.NewEncoder(w).Encode(serverResponse{
			Type:  "server-response",
			RPCID: req.RPCID,
			Result: mustRaw(t, resultEnvelope{
				OK:    true,
				Value: mustRaw(t, WhaleEnsureResult{OK: false, Error: "missing-home"}),
			}),
		})
	}))
	defer srv.Close()
	c := &Client{BaseURL: srv.URL}
	_, err := c.WhaleEnsureTyped(context.Background())
	if err == nil || !strings.Contains(err.Error(), "missing-home") {
		t.Fatalf("err = %v", err)
	}
}

func TestDshbotDescribe(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/dshbot/describe") {
			http.NotFound(w, r)
			return
		}
		var req clientRequest
		_ = json.NewDecoder(r.Body).Decode(&req)
		_ = json.NewEncoder(w).Encode(serverResponse{
			Type:  "server-response",
			RPCID: req.RPCID,
			Result: mustRaw(t, resultEnvelope{
				OK:    true,
				Value: mustRaw(t, map[string]any{"revision": 1, "ok": true}),
			}),
		})
	}))
	defer srv.Close()
	c := &Client{BaseURL: srv.URL}
	raw, err := c.DshbotDescribe(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), "revision") {
		t.Fatalf("raw = %s", raw)
	}
	cat, err := c.DshbotCatalog(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(cat), "revision") {
		t.Fatalf("catalog alias = %s", cat)
	}
}

func TestWhalePetStateAndIMChannelStatus(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/dsh-whale/pet/state", func(w http.ResponseWriter, r *http.Request) {
		var req clientRequest
		_ = json.NewDecoder(r.Body).Decode(&req)
		if req.Method != "pet/state" {
			t.Fatalf("method = %q", req.Method)
		}
		_ = json.NewEncoder(w).Encode(serverResponse{
			Type:  "server-response",
			RPCID: req.RPCID,
			Result: mustRaw(t, resultEnvelope{
				OK:    true,
				Value: mustRaw(t, map[string]any{"ok": true, "history": []any{}}),
			}),
		})
	})
	mux.HandleFunc("/feishu/connection.status", func(w http.ResponseWriter, r *http.Request) {
		var req clientRequest
		_ = json.NewDecoder(r.Body).Decode(&req)
		if req.Method != "connection.status" {
			t.Fatalf("method = %q", req.Method)
		}
		_ = json.NewEncoder(w).Encode(serverResponse{
			Type:  "server-response",
			RPCID: req.RPCID,
			Result: mustRaw(t, resultEnvelope{
				OK:    true,
				Value: mustRaw(t, map[string]any{"snapshot": map[string]any{"bots": []any{}}}),
			}),
		})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()
	c := &Client{BaseURL: srv.URL}
	raw, err := c.WhalePetState(context.Background(), true)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), "history") {
		t.Fatalf("pet/state = %s", raw)
	}
	st, err := c.IMChannelStatus(context.Background(), "feishu")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(st), "snapshot") {
		t.Fatalf("status = %s", st)
	}
}

func TestIMBotReconnectOmitsUnrequestedSendTest(t *testing.T) {
	var calls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		var req clientRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatal(err)
		}
		payload, ok := req.Payload.(map[string]any)
		if !ok {
			t.Fatalf("payload type = %T", req.Payload)
		}
		if payload["botId"] != "bot-1" {
			t.Fatalf("payload = %#v", payload)
		}
		if calls == 1 {
			if _, present := payload["sendTest"]; present {
				t.Fatalf("sendTest should be omitted when false: %#v", payload)
			}
		} else if value, present := payload["sendTest"]; !present || value != true {
			t.Fatalf("sendTest should be true when requested: %#v", payload)
		}
		_ = json.NewEncoder(w).Encode(serverResponse{
			Type:   "server-response",
			RPCID:  req.RPCID,
			Result: mustRaw(t, resultEnvelope{OK: true, Value: mustRaw(t, map[string]any{"ok": true})}),
		})
	}))
	defer srv.Close()

	c := &Client{BaseURL: srv.URL}
	if _, err := c.IMBotReconnect(context.Background(), "dingtalk", "bot-1", false); err != nil {
		t.Fatal(err)
	}
	if _, err := c.IMBotReconnect(context.Background(), "dingtalk", "bot-1", true); err != nil {
		t.Fatal(err)
	}
}

func TestWhalePetLook(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/dsh-whale/pet/look", func(w http.ResponseWriter, r *http.Request) {
		var req clientRequest
		_ = json.NewDecoder(r.Body).Decode(&req)
		payload, _ := json.Marshal(req.Payload)
		if !strings.Contains(string(payload), `"provider":"openai"`) || !strings.Contains(string(payload), "abc") {
			t.Fatalf("payload = %s", payload)
		}
		_ = json.NewEncoder(w).Encode(serverResponse{
			Type:  "server-response",
			RPCID: req.RPCID,
			Result: mustRaw(t, resultEnvelope{
				OK:    true,
				Value: mustRaw(t, map[string]any{"ok": true, "reply": "cute"}),
			}),
		})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()
	c := &Client{BaseURL: srv.URL}
	raw, err := c.WhalePetLook(context.Background(), "openai", "gpt-4o", "abc")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), "cute") {
		t.Fatalf("raw = %s", raw)
	}
}

func mustRaw(t *testing.T, v any) json.RawMessage {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestCallRejectsNonLoopbackBaseURL(t *testing.T) {
	for _, base := range []string{
		"https://example.com",
		"http://user:pass@127.0.0.1:8787",
		"file:///tmp/plugin",
	} {
		c := &Client{BaseURL: base, Cookie: "dsh-auth=secret"}
		if err := c.Call(context.Background(), "dsh-whale", "assistant/ensure", map[string]any{}, nil); err == nil {
			t.Fatalf("expected base URL rejection for %q", base)
		}
	}
}
