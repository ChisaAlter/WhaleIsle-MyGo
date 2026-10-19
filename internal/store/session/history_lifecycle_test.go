package session

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	"whaleisle/internal/dsh"
)

func TestHistoryLifecycleSelectionAndAuthoritativeSnapshot(t *testing.T) {
	s := New()
	s.items = []Item{{SessionID: "blank", Blank: true}, {SessionID: "old"}}
	s.Select("old")
	if st := s.Snapshot(); !st.HistoryLoading || st.HistoryReady || st.HistoryError != "" || st.SelectedSummaryBlank || len(st.Messages) != 0 {
		t.Fatalf("unopened selected history must remain unsettled: %+v", st)
	}
	// A malformed snapshot must never certify an empty conversation.
	s.applyFollowFrame(json.RawMessage(`{"type":"snapshot","records":null}`))
	if st := s.Snapshot(); st.HistoryReady || !st.HistoryLoading {
		t.Fatal("malformed snapshot completed history")
	}
	s.applyFollowFrame(json.RawMessage(`{"type":"snapshot","records":[]}`))
	if st := s.Snapshot(); !st.HistoryReady || st.HistoryLoading || len(st.Messages) != 0 {
		t.Fatalf("authoritative empty history not ready: %+v", st)
	}
	s.Select("blank")
	if st := s.Snapshot(); !st.SelectedSummaryBlank || st.HistoryReady || !st.HistoryLoading {
		t.Fatalf("Host blank summary must be separate from opening history: %+v", st)
	}
	s.reportFollowError(context.Background(), "blank", s.followGeneration, "follow: ", errors.New("offline"))
	if st := s.Snapshot(); !st.SelectedSummaryBlank || st.HistoryLoading || st.HistoryReady || st.HistoryError != "follow: offline" {
		t.Fatalf("known blank failure lost authority or feedback: %+v", st)
	}
	s.Select("")
	if st := s.Snapshot(); st.HistoryLoading || st.HistoryReady || st.HistoryError != "" || st.SelectedSummaryBlank {
		t.Fatalf("deselection retained prior lifecycle: %+v", st)
	}
}

func TestHistoryLifecycleFailureRecoveryAndStaleGeneration(t *testing.T) {
	s := New()
	s.Select("old")
	oldGeneration := s.followGeneration
	s.reportFollowError(context.Background(), "old", oldGeneration, "follow open: ", errors.New("unavailable"))
	if st := s.Snapshot(); st.HistoryLoading || st.HistoryReady || st.HistoryError == "" {
		t.Fatalf("unknown history failure masquerades as empty: %+v", st)
	}
	s.applyFollowFrameForTarget(context.Background(), "old", oldGeneration, json.RawMessage(`{"type":"snapshot","records":[{"type":"event","event":{"type":"user/message","seq":1,"data":{"content":[{"type":"text","text":"retained"}]}}}]}`))
	if st := s.Snapshot(); !st.HistoryReady || st.HistoryLoading || st.HistoryError != "" || st.LastAction != "" || len(st.Messages) != 1 {
		t.Fatalf("successful history did not recover: %+v", st)
	}
	s.mu.Lock()
	s.invalidateFollowLocked()
	s.historyLoading = true
	s.mu.Unlock()
	if st := s.Snapshot(); !st.HistoryReady || !st.HistoryLoading || len(st.Messages) != 1 {
		t.Fatalf("same-selection reconnect discarded baseline: %+v", st)
	}
	before := s.Snapshot()
	s.reportFollowError(context.Background(), "old", oldGeneration, "follow: ", errors.New("stale error"))
	if s.applyFollowFrameForTarget(context.Background(), "old", oldGeneration, json.RawMessage(`{"type":"snapshot","records":[]}`)) {
		t.Fatal("old follow generation accepted")
	}
	if got := s.Snapshot(); !reflect.DeepEqual(got, before) {
		t.Fatalf("stale follow changed lifecycle: %+v", got)
	}
	s.Select("next")
	before = s.Snapshot()
	s.reportFollowError(context.Background(), "old", s.followGeneration-1, "follow: ", errors.New("old selection"))
	if got := s.Snapshot(); !reflect.DeepEqual(got, before) {
		t.Fatal("old selection error changed new selection")
	}
}

func TestHistoryLifecycleRecoveryPreservesUnrelatedAction(t *testing.T) {
	for _, recovery := range []string{"snapshot", "invalidate"} {
		t.Run(recovery, func(t *testing.T) {
			s := New()
			s.Select("old")
			s.reportFollowError(context.Background(), "old", s.followGeneration, "follow: ", errors.New("offline"))
			s.mu.Lock()
			s.lastAction = "session renamed"
			if recovery == "invalidate" {
				s.invalidateFollowLocked()
			}
			s.mu.Unlock()
			if recovery == "snapshot" {
				s.applyFollowFrame(json.RawMessage(`{"type":"snapshot","records":[]}`))
			}
			if st := s.Snapshot(); st.HistoryError != "" || st.LastAction != "session renamed" {
				t.Fatalf("history recovery overwrote an unrelated action: %+v", st)
			}
		})
	}
}

func TestHistoryLifecycleStreamEndBeforeSnapshotIsVisibleFailure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := websocket.Accept(w, r, nil)
		if err != nil {
			return
		}
		defer c.CloseNow()
		var open dsh.ClientFrame
		if wsjson.Read(r.Context(), c, &open) != nil {
			return
		}
		_ = wsjson.Write(r.Context(), c, dsh.ServerFrame{Type: dsh.FrameEnd, StreamID: open.StreamID})
		var frame dsh.ClientFrame
		_ = wsjson.Read(r.Context(), c, &frame)
	}))
	defer srv.Close()
	mux := &dsh.Mux{Endpoint: dsh.Endpoint{Origin: srv.URL}}
	defer mux.Close()
	s := New()
	s.Select("old")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if restart := s.runFollowConnection(ctx, mux, "old", s.followGeneration); restart {
		t.Fatal("ended opening unexpectedly restarted")
	}
	if st := s.Snapshot(); st.HistoryLoading || st.HistoryReady || !strings.Contains(st.HistoryError, "before snapshot") {
		t.Fatalf("ended opening has no visible failure: %+v", st)
	}
}

func TestHistoryLifecycleFollowRetryRetainsBaseline(t *testing.T) {
	opened, release, failed := make(chan struct{}), make(chan struct{}), make(chan struct{})
	var releaseOnce, failedOnce sync.Once
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := websocket.Accept(w, r, nil)
		if err != nil {
			return
		}
		defer c.CloseNow()
		var open dsh.ClientFrame
		if wsjson.Read(r.Context(), c, &open) != nil {
			return
		}
		close(opened)
		<-release
		_ = wsjson.Write(r.Context(), c, dsh.ServerFrame{Type: dsh.FrameError, StreamID: open.StreamID, Error: &dsh.StreamFailure{Message: "retry unavailable"}})
		var frame dsh.ClientFrame
		_ = wsjson.Read(r.Context(), c, &frame)
	}))
	defer srv.Close()
	defer releaseOnce.Do(func() { close(release) })
	mux := &dsh.Mux{Endpoint: dsh.Endpoint{Origin: srv.URL}}
	defer mux.Close()
	s := New()
	s.Select("old")
	s.applyFollowFrame(json.RawMessage(`{"type":"snapshot","records":[{"type":"event","event":{"type":"user/message","seq":1,"data":{"content":[{"type":"text","text":"retained"}]}}}]}`))
	s.reportFollowError(context.Background(), "old", s.followGeneration, "follow: ", errors.New("previous failure"))
	s.AttachMux(mux)
	s.notify = func() {
		if s.Snapshot().HistoryError != "" {
			failedOnce.Do(func() { close(failed) })
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	s.FollowSelected(ctx)
	if st := s.Snapshot(); !st.HistoryLoading || !st.HistoryReady || st.HistoryError != "" || st.LastAction != "" || len(st.Messages) != 1 {
		t.Fatalf("retry did not expose loading with retained baseline: %+v", st)
	}
	wait(t, opened)
	releaseOnce.Do(func() { close(release) })
	wait(t, failed)
	if st := s.Snapshot(); st.HistoryLoading || !st.HistoryReady || !strings.Contains(st.HistoryError, "retry unavailable") || len(st.Messages) != 1 {
		t.Fatalf("retry failure discarded retained baseline: %+v", st)
	}
}
