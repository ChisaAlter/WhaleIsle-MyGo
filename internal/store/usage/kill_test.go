package usage

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	"whaleisle/internal/dsh"
)

func TestKillJobCallsRPC(t *testing.T) {
	var gotSession, gotJob string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		if strings.HasSuffix(r.URL.Path, "/job/kill") {
			payload, _ := body["payload"].(map[string]any)
			args, _ := payload["args"].(map[string]any)
			req, _ := args["request"].(map[string]any)
			if req != nil {
				gotSession, _ = req["sessionId"].(string)
				gotJob, _ = req["jobId"].(string)
			}
			writeRPC(w, body["rpcId"], map[string]any{"outcome": "killed"})
			return
		}
		if strings.HasSuffix(r.URL.Path, "/job/list") {
			writeRPC(w, body["rpcId"], map[string]any{"type": "snapshot", "jobs": []any{}})
			return
		}
		writeRPC(w, body["rpcId"], map[string]any{})
	}))
	defer srv.Close()

	st := New()
	st.Attach(&dsh.Client{Endpoint: dsh.Endpoint{Origin: srv.URL}}, nil)
	st.KillJob(context.Background(), "s1", "j9")
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if gotSession == "s1" && gotJob == "j9" {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("session/job = %q/%q", gotSession, gotJob)
}

func TestKillJobAndFollowJobRespectHostReplacement(t *testing.T) {
	killArrived, releaseKill := make(chan struct{}), make(chan struct{})
	followFrameSent, followCancelled := make(chan struct{}), make(chan struct{})
	oldSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == dsh.MuxPath {
			c, err := websocket.Accept(w, r, nil)
			if err != nil {
				return
			}
			defer c.CloseNow()
			var followStreamID string
			opened := map[string]bool{}
			notified := false
			for {
				var frame dsh.ClientFrame
				if err := wsjson.Read(r.Context(), c, &frame); err != nil {
					return
				}
				if frame.Type == dsh.FrameOpen {
					opened[frame.Endpoint] = true
					if frame.Endpoint == "job/follow" {
						followStreamID = frame.StreamID
						_ = wsjson.Write(r.Context(), c, dsh.ServerFrame{
							Type:     dsh.FrameItem,
							StreamID: frame.StreamID,
							Value:    json.RawMessage(`"stale-preview"`),
						})
					}
				}
				if !notified && opened["account/watch"] && opened["account/watchExpiry"] && opened["job/follow"] {
					notified = true
					close(followFrameSent)
				}
				if frame.Type == dsh.FrameCancel && frame.StreamID == followStreamID && followStreamID != "" {
					close(followCancelled)
					return
				}
			}
		}
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		if strings.HasSuffix(r.URL.Path, "/job/kill") {
			close(killArrived)
			<-releaseKill
			writeRPC(w, body["rpcId"], map[string]any{"outcome": "killed"})
			return
		}
		writeRPC(w, body["rpcId"], map[string]any{})
	}))
	defer oldSrv.Close()

	var newHostCalls atomic.Int32
	newSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		newHostCalls.Add(1)
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		writeRPC(w, body["rpcId"], map[string]any{})
	}))
	defer newSrv.Close()

	st := New()
	st.Attach(&dsh.Client{Endpoint: dsh.Endpoint{Origin: oldSrv.URL}}, nil)
	mux := &dsh.Mux{Endpoint: dsh.Endpoint{Origin: oldSrv.URL}}
	defer mux.Close()
	st.AttachMux(mux)

	st.FollowJob(context.Background(), "old-session", "job-1")
	st.KillJob(context.Background(), "old-session", "job-1")
	<-followFrameSent
	<-killArrived

	// Switching Host cancels the active job/follow stream and fences the
	// in-flight job/kill completion so the replacement Host is not refreshed
	// with the old session ID or polluted with partial job previews.
	st.Attach(&dsh.Client{Endpoint: dsh.Endpoint{Origin: newSrv.URL}}, nil)
	close(releaseKill)

	select {
	case <-followCancelled:
	case <-time.After(3 * time.Second):
		t.Fatal("Attach did not cancel active FollowJob stream")
	}
	time.Sleep(30 * time.Millisecond)
	snap := st.Snapshot()
	if len(snap.Recent) != 0 || snap.Err != "" || newHostCalls.Load() != 0 {
		t.Fatalf("stale job actions leaked into replacement Host: recent=%+v err=%q newHostCalls=%d", snap.Recent, snap.Err, newHostCalls.Load())
	}
}

func writeRPC(w http.ResponseWriter, rpcID any, value any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"type":  "server-response",
		"rpcId": rpcID,
		"result": map[string]any{
			"ok":    true,
			"value": value,
		},
	})
}
