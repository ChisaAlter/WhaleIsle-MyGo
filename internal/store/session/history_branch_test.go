package session

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
	"whaleisle/internal/dsh"
)

func branchFixture() []Message {
	return []Message{
		{Role: "user", Seq: 1, Body: "question"},
		{Role: "assistant", Kind: MessageKindText, Seq: 6, Body: "answer"},
		{Role: "status", EventType: "turn/end", Seq: 9, Raw: json.RawMessage(`{"data":{"reason":{"kind":"complete"}}}`)},
	}
}

func TestHistoryBranchFixture(t *testing.T) {
	t.Run("boundary guards", func(t *testing.T) {
		msgs := branchFixture()
		if !CanForkTurn(msgs, 9) || CanForkTurn(msgs, 6) || CanForkTurn(msgs, 0) {
			t.Fatal("boundary selection")
		}
		msgs[1].Streaming = true
		if CanForkTurn(msgs, 9) {
			t.Fatal("streaming branch")
		}
		msgs = branchFixture()
		msgs[2].Raw = json.RawMessage(`{"data":{"reason":{"kind":"error"}}}`)
		if CanForkTurn(msgs, 9) {
			t.Fatal("error branch")
		}
		msgs = branchFixture()
		msgs[1].Role = "tool"
		if CanForkTurn(msgs, 9) {
			t.Fatal("tool branch")
		}
		msg, ok := classifyPersistenceEvent(json.RawMessage(`{"type":"turn/end","seq":9,"data":{"reason":{"kind":"complete"}}}`))
		if !ok || msg.EventType != "turn/end" || len(msg.Raw) == 0 {
			t.Fatal("paged boundary lost")
		}
	})
	for _, mode := range []string{"success", "stale", "error", "empty"} {
		t.Run(mode, func(t *testing.T) {
			entered, release, finished := make(chan map[string]any, 1), make(chan struct{}), make(chan struct{}, 1)
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var body map[string]any
				_ = json.NewDecoder(r.Body).Decode(&body)
				if strings.HasSuffix(r.URL.Path, "/session/fork") {
					entered <- body
					<-release
					if mode == "error" {
						http.Error(w, "fixture failure", 500)
						return
					}
					id := "child"
					if mode == "empty" {
						id = ""
					}
					rpcOK(w, body["rpcId"], map[string]any{"sessionId": id})
					return
				}
				rpcOK(w, body["rpcId"], map[string]any{})
			}))
			defer srv.Close()
			st := New()
			st.Attach(&dsh.Client{Endpoint: dsh.Endpoint{Origin: srv.URL}}, func() {
				select {
				case finished <- struct{}{}:
				default:
				}
			})
			st.mu.Lock()
			st.selectSessionLocked("source")
			st.messages = branchFixture()
			st.mu.Unlock()
			st.ForkTurn(context.Background(), "source", 9)
			var body map[string]any
			select {
			case body = <-entered:
			case <-time.After(2 * time.Second):
				t.Fatal("no fork")
			}
			request := body["payload"].(map[string]any)["args"].(map[string]any)["request"].(map[string]any)
			if request["sessionId"] != "source" || request["atSeq"] != float64(9) {
				t.Fatalf("request %v", request)
			}
			if !st.Snapshot().HistoryBranchBusy {
				t.Fatal("missing pending state")
			}
			st.ForkTurn(context.Background(), "source", 9) // duplicate must be suppressed
			if mode == "stale" {
				st.Select("other")
			}
			close(release)
			deadline := time.After(2 * time.Second)
			for {
				snap := st.Snapshot()
				if !snap.HistoryBranchBusy && (mode == "stale" || snap.LastAction != "") {
					break
				}
				select {
				case <-finished:
				case <-deadline:
					t.Fatal("fork did not finish")
				}
			}
			snap := st.Snapshot()
			switch mode {
			case "success":
				if snap.SelectedID != "child" {
					t.Fatalf("selection %s", snap.SelectedID)
				}
			case "stale":
				if snap.SelectedID != "other" {
					t.Fatal("late fork changed selection")
				}
			default:
				if snap.SelectedID != "source" || !strings.Contains(snap.LastAction, "session/fork:") {
					t.Fatalf("failure state %+v", snap)
				}
			}
		})
	}
}
