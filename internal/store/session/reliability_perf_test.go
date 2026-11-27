package session

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"whaleisle/internal/dsh"
	"whaleisle/internal/ui/i18n"
)

func TestControlBaselineAppliesSelectedSessionProjections(t *testing.T) {
	s := New()
	s.mu.Lock()
	s.selectedID = "sess-1"
	s.items = []Item{{SessionID: "sess-1", Title: "Demo"}}
	s.permissionOptions = []PermissionOption{{Value: "ask", Label: "Ask"}, {Value: "auto", Label: "Auto"}}
	raw := json.RawMessage(`{
		"type": "baseline",
		"value": {
			"projections": {
				"sess-1": {
					"asOfSeq": 42,
					"values": {
						"inbox": {
							"next-turn": [{"id":"q1","content":[{"type":"text","text":"queued task"}],"source":{"kind":"user","rpcId":"rpc-q1"}}]
						},
						"permissions": {"currentValue": "auto"},
						"modelSelection": {"next": {"provider": "p1", "model": "m1"}},
						"contextPressure": {"pressureTokens": 1200, "contextWindow": 8000},
						"plan": {"active": true, "pending": false},
						"imageLimits": {"maxImages": 4, "maxBytes": 1048576},
						"turnOutline": [{"turn": 1, "seq": 1, "prompt": "hello", "response": "world"}]
					}
				}
			}
		}
	}`)
	var b struct {
		Value struct {
			Projections map[string]agentNavigationProjection `json:"projections"`
		} `json:"value"`
	}
	if err := json.Unmarshal(raw, &b); err != nil {
		s.mu.Unlock()
		t.Fatal(err)
	}
	s.applyControlBaselineMapLocked(sidebarCatalogOwner{}, raw, b.Value.Projections, true)
	s.mu.Unlock()

	snap := s.Snapshot()
	if snap.SelectedPermissionValue != "auto" {
		t.Fatalf("expected permission auto from control baseline, got %q", snap.SelectedPermissionValue)
	}
	if snap.ModelSelection == nil || snap.ModelSelection.Model != "m1" {
		t.Fatalf("expected model m1 from control baseline, got %+v", snap.ModelSelection)
	}
	if !snap.PlanModeControl.Active {
		t.Fatalf("expected plan mode active from control baseline, got %+v", snap.PlanModeControl)
	}
	if len(snap.PlanBoard.Queue) != 1 || snap.PlanBoard.Queue[0].RequestID != "rpc-q1" {
		t.Fatalf("expected queue item rpc-q1 from control baseline, got %+v", snap.PlanBoard.Queue)
	}
	if snap.ContextOccupancy == nil || snap.ContextOccupancy.UsedTokens != 1200 {
		t.Fatalf("expected context occupancy from control baseline, got %+v", snap.ContextOccupancy)
	}
	if len(snap.HistoryNavigation.Turns) != 1 {
		t.Fatalf("expected turn outline from control baseline, got %+v", snap.HistoryNavigation.Turns)
	}
}

func TestSendPromptBusyResetsOnSelectAndDetach(t *testing.T) {
	entered := make(chan struct{})
	release := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		select {
		case <-entered:
		default:
			close(entered)
		}
		<-release
		_ = json.NewEncoder(w).Encode(map[string]any{
			"type":   "server-response",
			"rpcId":  body["rpcId"],
			"result": map[string]any{"ok": true, "value": map[string]any{}},
		})
	}))
	defer srv.Close()

	s := New()
	client := &dsh.Client{Endpoint: dsh.Endpoint{Origin: srv.URL}}
	s.Attach(client, nil)
	s.mu.Lock()
	s.selectedID = "s1"
	s.mu.Unlock()

	s.SendPrompt(context.Background(), "hello")
	<-entered
	if !s.Snapshot().PromptBusy {
		t.Fatal("expected PromptBusy=true while send is in flight")
	}
	// Switching sessions must immediately clear PromptBusy rather than wedging it.
	s.Select("s2")
	if s.Snapshot().PromptBusy {
		t.Fatal("expected PromptBusy=false after switching sessions")
	}
	close(release)
	time.Sleep(25 * time.Millisecond)
	if s.Snapshot().PromptBusy {
		t.Fatal("expected stale SendPrompt completion not to re-wedge PromptBusy")
	}

	// Verify Detach also clears PromptBusy immediately.
	entered2 := make(chan struct{})
	release2 := make(chan struct{})
	srv2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		close(entered2)
		<-release2
		_ = json.NewEncoder(w).Encode(map[string]any{
			"type":   "server-response",
			"rpcId":  body["rpcId"],
			"result": map[string]any{"ok": true, "value": map[string]any{}},
		})
	}))
	defer srv2.Close()

	s.Attach(&dsh.Client{Endpoint: dsh.Endpoint{Origin: srv2.URL}}, nil)
	s.mu.Lock()
	s.selectedID = "s2"
	s.mu.Unlock()
	s.SendPrompt(context.Background(), "second")
	<-entered2
	if !s.Snapshot().PromptBusy {
		t.Fatal("expected PromptBusy=true before Detach")
	}
	s.Detach()
	if s.Snapshot().PromptBusy {
		t.Fatal("expected PromptBusy=false after Detach")
	}
	close(release2)
}

func TestOpenWorkspaceAndChildActionsGuardStaleHostGeneration(t *testing.T) {
	entered := make(chan struct{}, 3)
	release := make(chan struct{})
	oldSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		entered <- struct{}{}
		<-release
		_ = json.NewEncoder(w).Encode(map[string]any{
			"type":   "server-response",
			"rpcId":  body["rpcId"],
			"result": map[string]any{"ok": false, "error": map[string]any{"message": "stale-host-error"}},
		})
	}))
	defer oldSrv.Close()

	s := New()
	oldClient := &dsh.Client{Endpoint: dsh.Endpoint{Origin: oldSrv.URL}}
	s.Attach(oldClient, nil)

	s.OpenWorkspacePath(context.Background(), "/tmp/a", "", "")
	s.InterruptChild(context.Background(), "p1", "c1", "cancel")
	s.PromptChild(context.Background(), "p1", "c1", "hi")
	for i := 0; i < 3; i++ {
		<-entered
	}

	newClient := &dsh.Client{Endpoint: dsh.Endpoint{Origin: "http://127.0.0.1:1"}}
	s.Detach()
	s.Attach(newClient, nil)
	s.SetLastAction("fresh-host")
	close(release)
	time.Sleep(40 * time.Millisecond)

	if got := s.Snapshot().LastAction; got != "fresh-host" {
		t.Fatalf("stale host RPC clobbered LastAction: got %q, want %q", got, "fresh-host")
	}
}

func TestAgentPresetSelectionLocalizedAndSanitizedErrors(t *testing.T) {
	prev := i18n.Locale()
	defer i18n.SetLocale(prev)

	for _, lang := range []string{"zh", "en"} {
		i18n.SetLocale(lang)
		s := New()
		s.client = &dsh.Client{}
		s.selectedID = "blank"
		s.items = []Item{{SessionID: "blank", Blank: true}}
		s.agentPresetOptions = []AgentPresetOption{
			{ID: "bad", Label: "BrokenPreset", Broken: "raw yaml parse error at C:\\secret\\preset.yaml:12"},
		}
		s.SelectAgentPresetIDForSession(context.Background(), "blank", "bad")
		got := s.AgentPresetSelection()
		if got.Error == "" {
			t.Fatalf("lang=%s expected non-empty preset error", lang)
		}
		if strings.Contains(got.Error, "yaml") || strings.Contains(got.Error, "secret") {
			t.Fatalf("lang=%s leaked raw broken preset text: %q", lang, got.Error)
		}
		if !strings.Contains(got.Error, "BrokenPreset") {
			t.Fatalf("lang=%s missing preset label in error: %q", lang, got.Error)
		}

		// Non-blank session should produce localized error without hardcoded English in zh mode.
		s.items = []Item{{SessionID: "used", Blank: false}}
		s.selectedID = "used"
		s.agentPresetOptions = []AgentPresetOption{{ID: "ok", Label: "Coder"}}
		s.SelectAgentPresetIDForSession(context.Background(), "used", "ok")
		gotUsed := s.AgentPresetSelection()
		if lang == "zh" && strings.Contains(gotUsed.Error, "presets can only be applied") {
			t.Fatalf("zh mode leaked hardcoded English message: %q", gotUsed.Error)
		}
	}
}

func TestStreamingSplitZeroCopyMatchesFlatMessages(t *testing.T) {
	s := benchmarkStore(200)
	s.messages = append(s.messages, streamingRow())
	s.assistantStream.active = &assistantStreamAttempt{AttemptID: "split-test", Turn: 1, Step: 1}

	flat := s.Snapshot()
	s.SetStreamingSplitEnabled(true)
	frame := s.BeginFrame()
	defer frame.EndFrame()
	split1 := frame.Snapshot()
	if split1.MessageCount() != len(flat.Messages) {
		t.Fatalf("split MessageCount=%d want %d", split1.MessageCount(), len(flat.Messages))
	}
	if len(split1.DurableMessages) != len(flat.Messages)-1 || len(split1.StreamingTail) != 1 {
		t.Fatalf("unexpected split sizes: durable=%d tail=%d", len(split1.DurableMessages), len(split1.StreamingTail))
	}
	for i := 0; i < split1.MessageCount(); i++ {
		if split1.MessageAt(i).ID != flat.Messages[i].ID || split1.MessageAt(i).Body != flat.Messages[i].Body {
			t.Fatalf("mismatch at %d: got (%q,%q) want (%q,%q)", i, split1.MessageAt(i).ID, split1.MessageAt(i).Body, flat.Messages[i].ID, flat.Messages[i].Body)
		}
	}
	durablePtr := &split1.DurableMessages[0]
	streamTick(s)
	split2 := frame.Snapshot()
	if &split2.DurableMessages[0] != durablePtr {
		t.Fatal("streaming tick copied DurableMessages instead of sharing the immutable prefix")
	}
	if split2.MessageAt(split2.MessageCount()-1).Body == split1.MessageAt(split1.MessageCount()-1).Body {
		t.Fatal("streaming tick did not update StreamingTail")
	}
}

func TestSessionLRUCacheByteBudgetAndDerivedCacheRelease(t *testing.T) {
	s := New()
	s.Attach(&dsh.Client{}, nil)
	s.mu.Lock()
	s.sessionLRU.maxBytes = 2 << 20 // 2 MiB budget
	s.mu.Unlock()

	payload := strings.Repeat("a", 512)
	for i := 0; i < 24; i++ {
		id := fmt.Sprintf("sess-%02d", i)
		s.Select(id)
		s.mu.Lock()
		msgs := make([]Message, 400)
		for j := range msgs {
			msgs[j] = Message{
				ID:        fmt.Sprintf("%s-m-%d", id, j),
				Role:      "assistant",
				Kind:      MessageKindText,
				Seq:       float64(j + 1),
				Body:      payload,
				EventType: "assistant/message",
				Raw:       json.RawMessage(`{"type":"assistant/message","seq":1,"data":{"text":"hello"}}`),
			}
		}
		s.messages = msgs
		s.trajectoryBoundaries = map[float64]json.RawMessage{
			1: json.RawMessage(`{"type":"request/header","seq":1,"data":{"header":{"tools":[{"name":"big","parameters":{"blob":"` + strings.Repeat("x", 4096) + `"}}]}}}`),
			2: json.RawMessage(`{"type":"step/start","seq":2,"data":{"turn":1,"step":1}}`),
		}
		s.historyReady = true
		s.markHistoryDirtyLocked()
		s.mu.Unlock()
		_ = s.Snapshot()
	}
	s.Select("sess-final")

	s.mu.Lock()
	totalBytes := s.sessionLRU.totalBytes
	entryCount := len(s.sessionLRU.entries)
	var headerLeaked bool
	for _, e := range s.sessionLRU.entries {
		for _, raw := range e.trajectoryBoundaries {
			if typ, ok := peekEventType(raw); ok && typ == "request/header" {
				headerLeaked = true
			}
		}
	}
	s.mu.Unlock()

	if totalBytes > 2<<20 {
		t.Fatalf("sessionLRU exceeded byte budget: got %d > %d", totalBytes, 2<<20)
	}
	if entryCount == 0 || entryCount >= 24 {
		t.Fatalf("expected byte-budget eviction between 1 and 23 entries, got %d (totalBytes=%d)", entryCount, totalBytes)
	}
	if headerLeaked {
		t.Fatal("cached session entry retained bulky request/header boundary")
	}

	// Switching back to the most recent cached session restores immediately and projects cleanly.
	s.Select("sess-23")
	snap := s.Snapshot()
	if !snap.HistoryReady || len(snap.Messages) != 400 {
		t.Fatalf("expected cached session sess-23 restored with 400 messages, ready=%v len=%d", snap.HistoryReady, len(snap.Messages))
	}
}

func TestAssistantStreamBoundedWhenTrajectoryDisabled(t *testing.T) {
	s := New()
	s.SetTrajectoryEnabled(false)
	s.mu.Lock()
	s.selectedID = "s1"
	s.mu.Unlock()

	start := json.RawMessage(`{"type":"assistant-stream","frame":{"type":"start","attemptId":"a1","revision":1,"startedAfterSeq":0,"turn":1,"step":1}}`)
	s.applyFollowFrame(start)
	for i := 0; i < 200; i++ {
		frame := json.RawMessage(fmt.Sprintf(
			`{"type":"assistant-stream","frame":{"type":"chunk","attemptId":"a1","revision":%d,"index":%d,"time":%d,"chunk":{"type":"text-delta","index":0,"text":"x"}}}`,
			i+2, i, 1000+i,
		))
		s.applyFollowFrame(frame)
	}
	usageFrame := json.RawMessage(fmt.Sprintf(
		`{"type":"assistant-stream","frame":{"type":"chunk","attemptId":"a1","revision":%d,"index":%d,"time":%d,"chunk":{"type":"usage","usage":{"inputTokens":10,"outputTokens":25}}}}`,
		202, 200, 1300,
	))
	s.applyFollowFrame(usageFrame)

	s.mu.Lock()
	a := s.assistantStream.active
	streamLen := len(a.Stream)
	usage := a.currentLiveUsage()
	s.mu.Unlock()

	if streamLen > maxInactiveAssistantStreamRecords {
		t.Fatalf("a.Stream grew unbounded when trajectory disabled: got %d > %d", streamLen, maxInactiveAssistantStreamRecords)
	}
	if usage["outputTokens"] != 25 || usage["inputTokens"] != 10 {
		t.Fatalf("expected incremental liveUsage preserved when stream capped, got %+v", usage)
	}
	snap := s.Snapshot()
	if len(snap.Messages) != 1 || len(snap.Messages[0].Body) != 200 {
		t.Fatalf("expected all 200 streamed chunks folded into message body, got %+v", snap.Messages)
	}
}

func BenchmarkSnapshotWhileStreamingSplit(b *testing.B) {
	for _, size := range []int{2000, 8000} {
		b.Run(fmt.Sprintf("messages_%d_frame_live_attempt_split", size), func(b *testing.B) {
			s := benchmarkStore(size)
			s.SetStreamingSplitEnabled(true)
			s.messages = append(s.messages, streamingRow())
			s.assistantStream.active = &assistantStreamAttempt{AttemptID: "bench-split", Turn: 1, Step: 1}
			frame := s.BeginFrame()
			defer frame.EndFrame()
			_ = frame.Snapshot()
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				streamTick(s)
				_ = frame.Snapshot()
			}
		})
	}
}

func BenchmarkFollowAssistantChunkParse(b *testing.B) {
	s := New()
	s.SetTrajectoryEnabled(false)
	s.mu.Lock()
	s.selectedID = "bench"
	s.mu.Unlock()
	s.applyFollowFrame(json.RawMessage(`{"type":"assistant-stream","frame":{"type":"start","attemptId":"b1","revision":1,"startedAfterSeq":0,"turn":1,"step":1}}`))
	chunks := make([]json.RawMessage, 1024)
	for i := range chunks {
		chunks[i] = json.RawMessage(fmt.Sprintf(
			`{"type":"assistant-stream","frame":{"type":"chunk","attemptId":"b1","revision":%d,"index":%d,"time":%d,"chunk":{"type":"text-delta","index":0,"text":"tok"}}}`,
			i+2, i, 1000+i,
		))
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		idx := i % len(chunks)
		if idx == 0 && i > 0 {
			s.mu.Lock()
			s.assistantStream.revision = 1
			s.assistantStream.active.NextIndex = 0
			s.assistantStream.active.blocks = nil
			s.assistantStream.active.rows = nil
			s.messages = s.messages[:0]
			s.mu.Unlock()
		}
		s.applyFollowFrame(chunks[idx])
	}
}
