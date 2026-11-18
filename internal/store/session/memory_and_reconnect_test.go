package session

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	"whaleisle/internal/dsh"
)

func TestInvalidateFollowClearsDecodeCachesAndBoundsPerSessionSize(t *testing.T) {
	s := New()
	s.Select("A")
	s.mu.Lock()
	s.usageEventsCache = map[float64]usageCacheEntry{}
	s.trajEventsCache = map[float64]trajectoryRequestCacheEntry{}
	s.trajTimingCache = map[float64]trajectoryTimingCacheEntry{}
	s.assistantMsgCache = map[float64]assistantMsgCacheEntry{}
	for i := 1; i <= maxSessionDecodeCacheEntries+120; i++ {
		seq := float64(i)
		raw := json.RawMessage(fmt.Sprintf(`{"type":"user/message","seq":%d,"data":{"content":[{"type":"text","text":"m-%d"}]}}`, i, i))
		s.usageEventsCache[seq] = usageCacheEntry{raw: string(raw)}
		s.trajEventsCache[seq] = trajectoryRequestCacheEntry{raw: string(raw)}
		s.trajTimingCache[seq] = trajectoryTimingCacheEntry{raw: string(raw)}
		s.assistantMsgCache[seq] = assistantMsgCacheEntry{raw: string(raw)}
	}
	pruneSeqCache(s.usageEventsCache, maxSessionDecodeCacheEntries, nil)
	pruneSeqCache(s.trajEventsCache, maxSessionDecodeCacheEntries, nil)
	pruneSeqCache(s.trajTimingCache, maxSessionDecodeCacheEntries, nil)
	pruneSeqCache(s.assistantMsgCache, maxSessionDecodeCacheEntries, nil)
	if len(s.usageEventsCache) != maxSessionDecodeCacheEntries ||
		len(s.trajEventsCache) != maxSessionDecodeCacheEntries ||
		len(s.trajTimingCache) != maxSessionDecodeCacheEntries ||
		len(s.assistantMsgCache) != maxSessionDecodeCacheEntries {
		s.mu.Unlock()
		t.Fatalf("expected caches bounded to %d", maxSessionDecodeCacheEntries)
	}
	if _, ok := s.usageEventsCache[1]; ok {
		s.mu.Unlock()
		t.Fatal("oldest seq was not evicted from usageEventsCache")
	}
	s.invalidateFollowLocked()
	if s.usageEventsCache != nil || s.trajEventsCache != nil || s.trajTimingCache != nil || s.assistantMsgCache != nil {
		s.mu.Unlock()
		t.Fatal("invalidateFollowLocked did not release per-session decode caches")
	}
	s.mu.Unlock()
}

func TestSessionInspectorSlidingWindowAndOpenCloseResize(t *testing.T) {
	s := New()
	s.Select("insp")
	s.mu.Lock()
	for i := 1; i <= inspectorClosedMaxEvents+200; i++ {
		raw := json.RawMessage(fmt.Sprintf(`{"type":"unrecognized/private","seq":%d,"data":{"n":%d}}`, i, i))
		s.captureInspectorEventWithHeadLocked(raw, "unrecognized/private", float64(i))
		if s.inspector.settledRoots == nil {
			s.inspector.settledRoots = map[float64]string{}
		}
		s.inspector.settledRoots[float64(i)] = fmt.Sprintf("root:%d", i)
	}
	s.pruneInspectorEventsLocked()
	if got := len(s.inspector.events); got > inspectorClosedMaxEvents {
		s.mu.Unlock()
		t.Fatalf("closed inspector exceeded cap: got=%d cap=%d", got, inspectorClosedMaxEvents)
	}
	if got := len(s.inspector.settledRoots); got > inspectorClosedMaxEvents {
		s.mu.Unlock()
		t.Fatalf("settledRoots leaked past evicted events: got=%d", got)
	}
	if _, ok := s.inspector.events[1]; ok {
		s.mu.Unlock()
		t.Fatal("expected oldest inspector event evicted")
	}
	if *s.inspectorCursorLocked() != float64(inspectorClosedMaxEvents+200) {
		s.mu.Unlock()
		t.Fatalf("cursor changed after oldest eviction: %v", *s.inspectorCursorLocked())
	}
	s.mu.Unlock()

	s.SetInspectorOpen(true)
	s.mu.Lock()
	for i := inspectorClosedMaxEvents + 201; i <= inspectorClosedMaxEvents+900; i++ {
		raw := json.RawMessage(fmt.Sprintf(`{"type":"unrecognized/private","seq":%d,"data":{"n":%d}}`, i, i))
		s.captureInspectorEventWithHeadLocked(raw, "unrecognized/private", float64(i))
	}
	openCount := len(s.inspector.events)
	s.mu.Unlock()
	if openCount <= inspectorClosedMaxEvents {
		t.Fatalf("open inspector should retain more than closed cap (%d <= %d)", openCount, inspectorClosedMaxEvents)
	}

	s.SetInspectorOpen(false)
	s.mu.Lock()
	defer s.mu.Unlock()
	if got := len(s.inspector.events); got > inspectorClosedMaxEvents {
		t.Fatalf("closing inspector did not shrink window back to closed cap: got=%d cap=%d", got, inspectorClosedMaxEvents)
	}
}

func TestAttachmentCacheBoundedEviction(t *testing.T) {
	t.Cleanup(func() {
		attachmentCache.mu.Lock()
		attachmentCache.paths = map[string]string{}
		attachmentCache.order = nil
		attachmentCache.mu.Unlock()
	})
	attachmentCache.mu.Lock()
	attachmentCache.paths = map[string]string{}
	attachmentCache.order = nil
	for i := 0; i < maxAttachmentCacheEntries+25; i++ {
		putAttachmentCacheLocked(fmt.Sprintf("att-%d", i), fmt.Sprintf("/tmp/%d.png", i))
	}
	gotLen := len(attachmentCache.paths)
	_, hasOldest := attachmentCache.paths["att-0"]
	_, hasNewest := attachmentCache.paths[fmt.Sprintf("att-%d", maxAttachmentCacheEntries+24)]
	attachmentCache.mu.Unlock()
	if gotLen != maxAttachmentCacheEntries || hasOldest || !hasNewest {
		t.Fatalf("attachmentCache eviction invalid: len=%d hasOldest=%v hasNewest=%v", gotLen, hasOldest, hasNewest)
	}
}

func TestFollowCarrierLostReconnectsNonDestructivelyAndPreservesEarlierPages(t *testing.T) {
	var attempt atomic.Int32
	reconnectingObserved := make(chan struct{})
	releaseSecond := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := websocket.Accept(w, r, nil)
		if err != nil {
			return
		}
		defer c.CloseNow()
		for {
			var open dsh.ClientFrame
			if wsjson.Read(r.Context(), c, &open) != nil {
				return
			}
			if open.Type != dsh.FrameOpen || open.Endpoint != "session/follow" {
				continue
			}
			n := attempt.Add(1)
			if n == 1 {
				_ = wsjson.Write(r.Context(), c, dsh.ServerFrame{
					Type:     dsh.FrameItem,
					StreamID: open.StreamID,
					Value:    json.RawMessage(`{"type":"snapshot","cursor":11,"hasMore":true,"records":[{"type":"event","event":{"type":"user/message","seq":10,"data":{"content":[{"type":"text","text":"m10"}]}}},{"type":"event","event":{"type":"user/message","seq":11,"data":{"content":[{"type":"text","text":"m11"}]}}}]}`),
				})
				// Abruptly close socket without FrameEnd -> ErrCarrierLost in stream.Recv.
				c.CloseNow()
				return
			}
			<-releaseSecond
			_ = wsjson.Write(r.Context(), c, dsh.ServerFrame{
				Type:     dsh.FrameItem,
				StreamID: open.StreamID,
				Value:    json.RawMessage(`{"type":"snapshot","cursor":12,"hasMore":true,"records":[{"type":"event","event":{"type":"user/message","seq":11,"data":{"content":[{"type":"text","text":"m11"}]}}},{"type":"event","event":{"type":"user/message","seq":12,"data":{"content":[{"type":"text","text":"m12"}]}}}]}`),
			})
		}
	}))
	defer srv.Close()

	mux := &dsh.Mux{Endpoint: dsh.Endpoint{Origin: srv.URL}}
	defer mux.Close()
	s := New()
	s.Attach(&dsh.Client{}, nil)
	s.AttachMux(mux)
	s.selectedID = "sess-reconnect"
	s.historyLoading = true
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	go s.runFollow(ctx, mux, "sess-reconnect", s.followGeneration)

	// Wait for first snapshot (seq 10..11) and carrier-lost transition.
	deadline := time.Now().Add(3 * time.Second)
	for {
		st := s.Snapshot()
		if st.HistoryReady && st.HistoryReconnecting {
			close(reconnectingObserved)
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("did not enter HistoryReconnecting state cleanly: %+v", st)
		}
		time.Sleep(5 * time.Millisecond)
	}
	<-reconnectingObserved

	mid := s.Snapshot()
	if mid.HistoryError != "" || len(mid.Messages) != 2 || !mid.HistoryReady {
		t.Fatalf("carrier loss flashed error or cleared history: err=%q msgs=%d ready=%v", mid.HistoryError, len(mid.Messages), mid.HistoryReady)
	}

	// Simulate an older page (seq 8..9) already loaded to the bottom (hasMore=false) before reconnect snapshot arrives.
	s.mu.Lock()
	s.messages = append([]Message{
		{ID: "p8", Kind: MessageKindText, Role: "user", Body: "m8", Seq: 8},
		{ID: "p9", Kind: MessageKindText, Role: "user", Body: "m9", Seq: 9},
	}, s.messages...)
	s.historyHasMore = false
	s.invalidateViewLocked()
	s.mu.Unlock()

	close(releaseSecond)
	deadline = time.Now().Add(3 * time.Second)
	for {
		st := s.Snapshot()
		if !st.HistoryReconnecting && len(st.Messages) == 5 && st.Messages[4].Seq == 12 {
			if st.HistoryError != "" {
				t.Fatalf("unexpected historyError after reconnect: %q", st.HistoryError)
			}
			if st.HistoryHasMore {
				t.Fatal("reconnect snapshot clobbered pre-existing HistoryHasMore=false paging depth")
			}
			if st.Messages[0].Seq != 8 || st.Messages[1].Seq != 9 || st.Messages[2].Seq != 10 || st.Messages[3].Seq != 11 {
				t.Fatalf("unexpected merged transcript order: %+v", st.Messages)
			}
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("reconnect snapshot did not merge cleanly: reconnecting=%v msgs=%+v", st.HistoryReconnecting, st.Messages)
		}
		time.Sleep(5 * time.Millisecond)
	}
}
