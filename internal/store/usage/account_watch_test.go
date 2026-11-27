package usage

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"

	"whaleisle/internal/accountdetails"
	"whaleisle/internal/dsh"
)

func TestAccountWatchDecodesAccountUpdatePreservesDetailsAndNotifiesCredentials(t *testing.T) {
	frames := make(chan json.RawMessage, 8)
	var profileCalls, balanceCalls atomic.Int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == dsh.MuxPath {
			c, err := websocket.Accept(w, r, nil)
			if err != nil {
				return
			}
			defer c.CloseNow()
			for {
				var frame dsh.ClientFrame
				if err := wsjson.Read(r.Context(), c, &frame); err != nil {
					return
				}
				if frame.Type == dsh.FrameOpen && frame.Endpoint == "account/watch" {
					streamID := frame.StreamID
					go func() {
						for raw := range frames {
							if err := wsjson.Write(r.Context(), c, dsh.ServerFrame{
								Type:     dsh.FrameItem,
								StreamID: streamID,
								Value:    raw,
							}); err != nil {
								return
							}
						}
					}()
				}
			}
		}
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		switch {
		case strings.HasSuffix(r.URL.Path, "/account/getProfile"):
			profileCalls.Add(1)
			writeRPC(w, body["rpcId"], map[string]any{
				"status": "ready",
				"value": map[string]any{
					"id":      "u-1",
					"name":    "Ada Lovelace",
					"contact": "ada@deepseek.com",
				},
			})
		case strings.HasSuffix(r.URL.Path, "/account/getBalance"):
			balanceCalls.Add(1)
			writeRPC(w, body["rpcId"], map[string]any{
				"status": "ready",
				"value": []map[string]any{
					{"currency": "CNY", "balance": "42.50"},
				},
				"bonusWallets": []any{},
			})
		default:
			writeRPC(w, body["rpcId"], map[string]any{})
		}
	}))
	defer srv.Close()
	defer close(frames)

	var credMu sync.Mutex
	var credEvents []string

	st := New()
	st.SetOnCredentialsChanged(func(status string) {
		credMu.Lock()
		credEvents = append(credEvents, status)
		credMu.Unlock()
	})
	st.Attach(&dsh.Client{Endpoint: dsh.Endpoint{Origin: srv.URL}}, nil)
	mux := &dsh.Mux{Endpoint: dsh.Endpoint{Origin: srv.URL}}
	defer mux.Close()
	st.AttachMux(mux)

	waitSnap := func(desc string, pred func(ViewState) bool) ViewState {
		t.Helper()
		deadline := time.Now().Add(3 * time.Second)
		for time.Now().Before(deadline) {
			v := st.Snapshot().Value
			if pred(v) {
				return v
			}
			time.Sleep(5 * time.Millisecond)
		}
		t.Fatalf("%s did not settle; last=%+v", desc, st.Snapshot().Value)
		return ViewState{}
	}

	// 1. Real Host 0.2.2 wire envelope: AccountUpdate{type:"credentials-changed", state:AccountView}.
	frames <- json.RawMessage(`{
		"type": "credentials-changed",
		"state": {
			"status": "credential-stored",
			"links": {"usageUrl": "https://platform.deepseek.com/usage", "topUpUrl": "https://platform.deepseek.com/top_up"},
			"attempt": {"id": "att-1", "phase": "succeeded"}
		}
	}`)

	v := waitSnap("credential-stored details", func(v ViewState) bool {
		return v.AccountStatus == "credential-stored" &&
			v.Details.Profile.Status == accountdetails.Ready &&
			v.Details.Balance.Status == accountdetails.Ready
	})
	if v.Name != "Ada Lovelace" || v.Email != "ada@deepseek.com" {
		t.Fatalf("profile fields = name:%q email:%q", v.Name, v.Email)
	}
	if v.UsageURL != "https://platform.deepseek.com/usage" || v.TopUpURL != "https://platform.deepseek.com/top_up" {
		t.Fatalf("links = usage:%q topUp:%q", v.UsageURL, v.TopUpURL)
	}
	if v.AttemptPhase != "succeeded" || v.AttemptID != "att-1" {
		t.Fatalf("attempt = phase:%q id:%q", v.AttemptPhase, v.AttemptID)
	}
	credMu.Lock()
	if len(credEvents) != 1 || credEvents[0] != "credential-stored" {
		t.Fatalf("credEvents = %v, want [credential-stored]", credEvents)
	}
	credMu.Unlock()

	// 2. Subsequent identical snapshot must NOT wipe profile/email or re-fire credentials callback.
	prevProfileCalls := profileCalls.Load()
	frames <- json.RawMessage(`{
		"type": "snapshot",
		"state": {
			"status": "credential-stored",
			"links": {"usageUrl": "https://platform.deepseek.com/usage", "topUpUrl": "https://platform.deepseek.com/top_up"},
			"attempt": {"id": "att-1", "phase": "succeeded"}
		}
	}`)
	time.Sleep(40 * time.Millisecond)
	v = st.Snapshot().Value
	if v.Name != "Ada Lovelace" || v.Email != "ada@deepseek.com" || v.AttemptPhase != "succeeded" {
		t.Fatalf("snapshot wiped account details: %+v", v)
	}
	if profileCalls.Load() != prevProfileCalls {
		t.Fatalf("redundant snapshot re-queried getProfile: %d -> %d", prevProfileCalls, profileCalls.Load())
	}
	credMu.Lock()
	if len(credEvents) != 1 {
		t.Fatalf("redundant snapshot re-fired credentials callback: %v", credEvents)
	}
	credMu.Unlock()

	// 3. Sign-out credentials-changed clears details and notifies model catalog refresh.
	frames <- json.RawMessage(`{
		"type": "credentials-changed",
		"state": {
			"status": "signed-out",
			"links": {"usageUrl": "", "topUpUrl": ""},
			"attempt": {"id": "", "phase": "idle"}
		}
	}`)
	v = waitSnap("signed-out state", func(v ViewState) bool {
		return v.AccountStatus == "signed-out" && v.Name == "" && v.Email == ""
	})
	if v.AttemptPhase != "idle" {
		t.Fatalf("signed-out attemptPhase = %q, want idle", v.AttemptPhase)
	}
	credMu.Lock()
	if len(credEvents) != 2 || credEvents[1] != "signed-out" {
		t.Fatalf("credEvents after sign-out = %v, want [credential-stored signed-out]", credEvents)
	}
	credMu.Unlock()

	// 4. Legacy flat AccountView fixture fallback stays compatible.
	frames <- json.RawMessage(`{
		"status": "credential-stored",
		"links": {"usageUrl": "https://legacy/usage", "topUpUrl": "https://legacy/topup"},
		"attempt": {"id": "legacy-2", "phase": "succeeded"}
	}`)
	v = waitSnap("legacy flat AccountView", func(v ViewState) bool {
		return v.AccountStatus == "credential-stored" && v.AttemptID == "legacy-2" && v.Details.Profile.Status == accountdetails.Ready
	})
	if v.AttemptPhase != "succeeded" || v.UsageURL != "https://legacy/usage" {
		t.Fatalf("legacy flat decode failed: %+v", v)
	}
	credMu.Lock()
	if len(credEvents) != 3 || credEvents[2] != "credential-stored" {
		t.Fatalf("credEvents after legacy transition = %v", credEvents)
	}
	credMu.Unlock()
}
