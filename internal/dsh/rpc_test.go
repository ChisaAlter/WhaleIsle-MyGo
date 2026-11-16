package dsh

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"strings"
	"testing"
	"time"
)

func TestCallSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/workspace/create" {
			t.Fatalf("path %s", r.URL.Path)
		}
		if r.Header.Get("Cookie") != "dsh-auth-x=ok" {
			t.Fatalf("cookie %q", r.Header.Get("Cookie"))
		}
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"type":  "server-response",
			"rpcId": body["rpcId"],
			"result": map[string]any{
				"ok":    true,
				"value": map[string]any{"workspace": map[string]any{"path": "/tmp/ws"}},
			},
		})
	}))
	defer srv.Close()

	c := &Client{Endpoint: Endpoint{Origin: srv.URL, Cookie: "dsh-auth-x=ok"}}
	var out struct {
		Workspace struct {
			Path string `json:"path"`
		} `json:"workspace"`
	}
	err := c.Call(context.Background(), "workspace/create", map[string]any{
		"request": map[string]any{"path": "/tmp/ws"},
	}, &out)
	if err != nil {
		t.Fatal(err)
	}
	if out.Workspace.Path != "/tmp/ws" {
		t.Fatalf("path = %q", out.Workspace.Path)
	}
}

func TestCallRPCError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"type":  "server-response",
			"rpcId": body["rpcId"],
			"result": map[string]any{
				"ok": false,
				"error": map[string]any{
					"code":    "NOT_FOUND",
					"message": "missing",
				},
			},
		})
	}))
	defer srv.Close()
	c := &Client{Endpoint: Endpoint{Origin: srv.URL}}
	err := c.Call(context.Background(), "session/get", map[string]any{}, nil)
	var rpcErr *Error
	if !errors.As(err, &rpcErr) || rpcErr.Code != "NOT_FOUND" || rpcErr.Message != "missing" {
		t.Fatalf("got %v", err)
	}
}

func TestCallRejectsBadMethod(t *testing.T) {
	c := &Client{Endpoint: Endpoint{Origin: "http://127.0.0.1"}}
	if err := c.Call(context.Background(), "nopath", nil, nil); err == nil {
		t.Fatal("expected error")
	}
}

func TestCallMultipartSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		rpcID, _ := body["rpcId"].(string)
		var buf bytes.Buffer
		mw := multipart.NewWriter(&buf)
		h := make(textproto.MIMEHeader)
		h.Set("Content-Disposition", `form-data; name="response"`)
		h.Set("Content-Type", "application/json")
		p, err := mw.CreatePart(h)
		if err != nil {
			t.Fatal(err)
		}
		enc, _ := json.Marshal(map[string]any{
			"type":  "server-response",
			"rpcId": rpcID,
			"result": map[string]any{
				"ok":    true,
				"value": map[string]any{"bytes": 3},
			},
		})
		_, _ = p.Write(enc)
		_ = mw.Close()
		w.Header().Set("Content-Type", mw.FormDataContentType())
		_, _ = w.Write(buf.Bytes())
	}))
	defer srv.Close()

	c := &Client{Endpoint: Endpoint{Origin: srv.URL}}
	var out struct {
		Bytes int `json:"bytes"`
	}
	if err := c.Call(context.Background(), "file/get", map[string]any{}, &out); err != nil {
		t.Fatal(err)
	}
	if out.Bytes != 3 {
		t.Fatalf("got %#v", out)
	}
}

func TestDefaultHTTPClientTransportAndCallTimeout(t *testing.T) {
	if defaultHTTPClient.Timeout != 0 {
		t.Fatalf("defaultHTTPClient.Timeout = %v, want 0 to keep SSE/export streams alive", defaultHTTPClient.Timeout)
	}
	tr, ok := defaultHTTPClient.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("defaultHTTPClient.Transport = %T, want *http.Transport", defaultHTTPClient.Transport)
	}
	if tr.DialContext == nil || tr.TLSHandshakeTimeout <= 0 || tr.ResponseHeaderTimeout <= 0 {
		t.Fatalf("missing transport timeouts: dial=%v tls=%v header=%v", tr.DialContext != nil, tr.TLSHandshakeTimeout, tr.ResponseHeaderTimeout)
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/slow/rpc":
			var body map[string]any
			_ = json.NewDecoder(r.Body).Decode(&body)
			select {
			case <-time.After(80 * time.Millisecond):
				_ = json.NewEncoder(w).Encode(map[string]any{
					"type":   "server-response",
					"rpcId":  body["rpcId"],
					"result": map[string]any{"ok": true, "value": map[string]any{"done": true}},
				})
			case <-r.Context().Done():
			}
		case "/plugins/events":
			w.Header().Set("Content-Type", "text/event-stream")
			w.WriteHeader(http.StatusOK)
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
			time.Sleep(60 * time.Millisecond)
			_, _ = w.Write([]byte("event: ping\ndata: {}\n\n"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	prevTimeout := defaultCallTimeout
	defaultCallTimeout = 25 * time.Millisecond
	t.Cleanup(func() { defaultCallTimeout = prevTimeout })

	c := &Client{Endpoint: Endpoint{Origin: srv.URL}}
	err := c.Call(context.Background(), "slow/rpc", nil, nil)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("unbounded ctx Call err = %v, want DeadlineExceeded", err)
	}

	explicitCtx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	var out struct {
		Done bool `json:"done"`
	}
	if err := c.Call(explicitCtx, "slow/rpc", nil, &out); err != nil || !out.Done {
		t.Fatalf("explicit deadline Call err=%v out=%#v", err, out)
	}

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, srv.URL+"/plugins/events", nil)
	if err != nil {
		t.Fatal(err)
	}
	res, err := (*c.http()).Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	buf := make([]byte, 64)
	n, err := res.Body.Read(buf)
	if !strings.Contains(string(buf[:n]), "event: ping") {
		t.Fatalf("SSE read n=%d err=%v body=%q", n, err, string(buf[:n]))
	}
}
