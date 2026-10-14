package session

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"whaleisle/internal/dsh"
)

func TestNormalizePromptMode(t *testing.T) {
	if NormalizePromptMode("steer") != PromptModeSteer {
		t.Fatal("steer")
	}
	if NormalizePromptMode("interrupt") != PromptModeSteer {
		t.Fatal("interrupt alias")
	}
	if NormalizePromptMode("") != PromptModeQueue {
		t.Fatal("default queue")
	}
}

func TestSetPromptModeUpdatesSnapshot(t *testing.T) {
	st := New()
	st.SetPromptMode(PromptModeSteer)
	if got := st.Snapshot().PromptMode; got != PromptModeSteer {
		t.Fatalf("mode = %q", got)
	}
}

func TestSendPromptUsesSteerMode(t *testing.T) {
	var mode atomic.Value
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		if strings.HasSuffix(r.URL.Path, "/session/prompt") {
			payload, _ := body["payload"].(map[string]any)
			args, _ := payload["args"].(map[string]any)
			req, _ := args["request"].(map[string]any)
			if req != nil {
				mode.Store(req["mode"])
			}
		}
		rpcOK(w, body["rpcId"], map[string]any{"accepted": true})
	}))
	defer srv.Close()

	client := &dsh.Client{Endpoint: dsh.Endpoint{Origin: srv.URL}}
	st := New()
	st.Attach(client, nil)
	st.mu.Lock()
	st.selectedID = "s1"
	st.promptMode = PromptModeSteer
	st.mu.Unlock()

	st.SendPrompt(context.Background(), "hello")

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if v := mode.Load(); v != nil {
			if v.(string) != "steer" {
				t.Fatalf("mode = %v", v)
			}
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("no session/prompt observed")
}
