package session

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"whaleisle/internal/dsh"
	"whaleisle/internal/dsh/api"
)

// Real journal pages include non-visible extension/configuration events too.
// Keep their raw sequence dense without inventing extra chat messages.
func denseHistoryFixture(records []api.SessionPageRecordsItem, before float64) []api.SessionPageRecordsItem {
	if len(records) == 0 {
		return nil
	}
	bySeq := make(map[float64]api.SessionPageRecordsItem, len(records))
	for _, record := range records {
		bySeq[record.Event.Seq] = record
	}
	var dense []api.SessionPageRecordsItem
	for seq := records[0].Event.Seq; seq < before; seq++ {
		record, ok := bySeq[seq]
		if !ok {
			record = api.SessionPageRecordsItem{Type: "event", Event: api.SessionPageRecordsItemEvent{Type: "fixture/metadata", Seq: seq, Data: json.RawMessage(`{}`)}}
		}
		dense = append(dense, record)
	}
	return dense
}

func TestLoadEarlierHistoryPrependsMerge(t *testing.T) {
	var pageCalls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		if strings.HasSuffix(r.URL.Path, "/session/page") {
			pageCalls.Add(1)
			payload, _ := body["payload"].(map[string]any)
			args, _ := payload["args"].(map[string]any)
			req, _ := args["request"].(map[string]any)
			if req["throughSeq"] != 4.0 {
				t.Fatalf("throughSeq = %v", req["throughSeq"])
			}
			if req["beforeSeq"] != 3.0 {
				t.Fatalf("beforeSeq = %v", req["beforeSeq"])
			}
			rpcOK(w, body["rpcId"], map[string]any{
				"hasMore": false,
				"records": []map[string]any{
					pageEventRecord(1, "user/message", "first"),
					pageEventRecord(2, "assistant/message", "second"),
				},
			})
			return
		}
		rpcOK(w, body["rpcId"], map[string]any{})
	}))
	defer srv.Close()

	client := &dsh.Client{Endpoint: dsh.Endpoint{Origin: srv.URL}}
	st := New()
	st.Attach(client, nil)
	st.mu.Lock()
	st.selectedID = "sess-1"
	st.messages = []Message{
		{ID: "m3", Kind: MessageKindText, Role: "user", Body: "three", Seq: 3},
		{ID: "m4", Kind: MessageKindText, Role: "assistant", Body: "four", Seq: 4},
	}
	st.historyHasMore = true
	st.historyThroughSeq = 4
	st.mu.Unlock()

	done := make(chan struct{})
	st.Attach(client, func() {
		snap := st.Snapshot()
		if snap.HistoryPageBusy || len(snap.Messages) != 4 {
			return
		}
		select {
		case <-done:
		default:
			close(done)
		}
	})

	st.LoadEarlierHistory(context.Background())
	wait(t, done)

	if pageCalls.Load() != 1 {
		t.Fatalf("page calls = %d", pageCalls.Load())
	}
	snap := st.Snapshot()
	if snap.HistoryHasMore {
		t.Fatal("expected hasMore false after page")
	}
	if len(snap.Messages) != 4 {
		t.Fatalf("len = %d", len(snap.Messages))
	}
	if snap.Messages[0].Body != "first" || snap.Messages[1].Body != "second" {
		t.Fatalf("order = %#v", snap.Messages)
	}
	if snap.Messages[2].Seq != 3 || snap.Messages[3].Seq != 4 {
		t.Fatalf("seqs = %v", []float64{snap.Messages[2].Seq, snap.Messages[3].Seq})
	}
}

func TestLoadEarlierHistoryDedupesOverlap(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		if strings.HasSuffix(r.URL.Path, "/session/page") {
			rpcOK(w, body["rpcId"], map[string]any{
				"hasMore": true,
				"records": []map[string]any{
					pageEventRecord(1, "user/message", "one"),
					pageEventRecord(2, "user/message", "dup"),
				},
			})
			return
		}
		rpcOK(w, body["rpcId"], map[string]any{})
	}))
	defer srv.Close()

	client := &dsh.Client{Endpoint: dsh.Endpoint{Origin: srv.URL}}
	st := New()
	st.Attach(client, nil)
	st.mu.Lock()
	st.selectedID = "s"
	st.messages = []Message{
		{ID: "m2", Kind: MessageKindText, Role: "user", Body: "two", Seq: 2},
		{ID: "m3", Kind: MessageKindText, Role: "assistant", Body: "three", Seq: 3},
	}
	st.historyHasMore = true
	st.historyThroughSeq = 3
	st.mu.Unlock()

	done := make(chan struct{})
	st.Attach(client, func() {
		snap := st.Snapshot()
		if snap.HistoryPageBusy || len(snap.Messages) != 3 {
			return
		}
		select {
		case <-done:
		default:
			close(done)
		}
	})
	st.LoadEarlierHistory(context.Background())
	wait(t, done)

	snap := st.Snapshot()
	if len(snap.Messages) != 3 || snap.Messages[0].Body != "one" || snap.Messages[1].Body != "two" {
		t.Fatalf("%#v", snap.Messages)
	}
	if !snap.HistoryHasMore {
		t.Fatal("expected hasMore true")
	}
}

func TestLoadEarlierHistoryRejectsDiscontinuousPagesWithoutChangingWindow(t *testing.T) {
	for _, test := range []struct {
		name string
		seqs []int
	}{
		{"descending", []int{3, 2}},
		{"internal hole", []int{1, 3}},
		{"tail does not join", []int{1, 2}},
		{"crossed durable cut", []int{2, 3, 4, 5, 6}},
	} {
		t.Run(test.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var in map[string]any
				_ = json.NewDecoder(r.Body).Decode(&in)
				var records []map[string]any
				for _, seq := range test.seqs {
					records = append(records, pageEventRecord(float64(seq), "user/message", "invalid older record"))
				}
				rpcOK(w, in["rpcId"], map[string]any{"records": records, "hasMore": false})
			}))
			defer srv.Close()
			s := New()
			client := &dsh.Client{Endpoint: dsh.Endpoint{Origin: srv.URL}}
			done := make(chan struct{})
			var once sync.Once
			s.Attach(client, func() {
				v := s.Snapshot()
				if !v.HistoryPageBusy && v.HistoryPageErr != "" {
					once.Do(func() { close(done) })
				}
			})
			s.selectedID = "s"
			s.historyHasMore = true
			s.historyThroughSeq = 5
			s.messages = []Message{{ID: "current", Seq: 4, Body: "retained", Role: "user", Kind: MessageKindText}}
			s.LoadEarlierHistory(context.Background())
			wait(t, done)
			v := s.Snapshot()
			if len(v.Messages) != 1 || v.Messages[0].Body != "retained" || v.HistoryPageErr == "" || !v.HistoryHasMore || s.historyThroughSeq != 5 {
				t.Fatalf("invalid page changed current window: %+v", v)
			}
		})
	}
}

func TestLoadEarlierHistoryPreservesStreaming(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		if strings.HasSuffix(r.URL.Path, "/session/page") {
			rpcOK(w, body["rpcId"], map[string]any{
				"hasMore": false,
				"records": []map[string]any{
					pageEventRecord(1, "user/message", "old"),
				},
			})
			return
		}
		rpcOK(w, body["rpcId"], map[string]any{})
	}))
	defer srv.Close()

	client := &dsh.Client{Endpoint: dsh.Endpoint{Origin: srv.URL}}
	st := New()
	st.Attach(client, nil)
	st.mu.Lock()
	st.selectedID = "s"
	st.messages = []Message{
		{ID: "m1", Kind: MessageKindText, Role: "user", Body: "hi", Seq: 2},
		{ID: "live", Kind: MessageKindText, Role: "assistant", Body: "partial", Streaming: true},
	}
	st.historyHasMore = true
	st.historyThroughSeq = 2
	st.mu.Unlock()

	done := make(chan struct{})
	st.Attach(client, func() {
		snap := st.Snapshot()
		if snap.HistoryPageBusy || len(snap.Messages) != 3 {
			return
		}
		select {
		case <-done:
		default:
			close(done)
		}
	})
	st.LoadEarlierHistory(context.Background())
	wait(t, done)

	snap := st.Snapshot()
	if snap.Messages[0].Body != "old" {
		t.Fatalf("prepend = %#v", snap.Messages[0])
	}
	last := snap.Messages[len(snap.Messages)-1]
	if !last.Streaming || last.Body != "partial" {
		t.Fatalf("streaming tail = %#v", last)
	}
}

func pageEventRecord(seq float64, typ, text string) map[string]any {
	data, _ := json.Marshal(map[string]any{"text": text})
	return map[string]any{
		"type": "event",
		"event": map[string]any{
			"type": typ,
			"seq":  seq,
			"time": 0,
			"data": json.RawMessage(data),
		},
	}
}
