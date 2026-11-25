package session

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"whaleisle/internal/config"
	"whaleisle/internal/dsh"
	"whaleisle/internal/dsh/api"
)

func stubOKClient(origin string) *dsh.Client {
	return &dsh.Client{
		Endpoint: dsh.Endpoint{Origin: origin},
		HTTP: &http.Client{Transport: lifecycleAdmissionTransport(func(r *http.Request) (*http.Response, error) {
			var body struct {
				RPCID string `json:"rpcId"`
			}
			_ = json.NewDecoder(r.Body).Decode(&body)
			raw, _ := json.Marshal(map[string]any{"type": "server-response", "rpcId": body.RPCID, "result": map[string]any{"ok": true, "value": map[string]any{}}})
			return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(string(raw)))}, nil
		})},
	}
}

func TestSessionLRUOptimisticSwitchEvictionAndSameWorkspaceReuse(t *testing.T) {
	s := New()
	client := stubOKClient("http://lru-fixture.invalid")
	s.Attach(client, nil)
	s.mu.Lock()
	s.cwd = "/ws-a"
	s.items = []Item{
		{SessionID: "s1", Cwd: "/ws-a"},
		{SessionID: "s2", Cwd: "/ws-a"},
		{SessionID: "s3", Cwd: "/ws-b"},
	}
	s.mu.Unlock()

	// Load s1 with authoritative history + workspace skills & commands.
	s.Select("s1")
	s.applyFollowFrame(json.RawMessage(`{"type":"snapshot","cursor":2,"hasMore":false,"records":[{"type":"event","event":{"type":"user/message","seq":1,"data":{"content":[{"type":"text","text":"hello s1"}]}}},{"type":"event","event":{"type":"turn/end","seq":2,"data":{"turn":1}}}]}`))
	s.mu.Lock()
	s.commandCatalog = commandCatalog{client: client, sessionID: "s1", loaded: true, items: []api.ListResultItem2{{Name: "review"}}}
	s.skillOptions = []SkillOption{{Name: "go-test"}}
	s.skillOptionsOwner = s.captureSkillOptionsScopeLocked()
	s.skillCatalog = SkillCatalog{Skills: []api.SkillInventorySnapshotSkillsItem{{Name: "go-test"}}}
	s.skillCatalogClient, s.skillCatalogSession, s.skillCatalogCwd = client, "s1", "/ws-a"
	s.mu.Unlock()

	// Switch to s2 in the SAME workspace: skills & commands stay warm (no empty flash).
	s.Select("s2")
	snapS2 := s.Snapshot()
	if snapS2.HistoryReady || !snapS2.HistoryLoading {
		t.Fatalf("cold session s2 should be loading: ready=%v loading=%v", snapS2.HistoryReady, snapS2.HistoryLoading)
	}
	if len(snapS2.SlashCommands) != 1 || snapS2.SlashCommands[0].Name != "review" {
		t.Fatalf("same-workspace switch cleared SlashCommands: %+v", snapS2.SlashCommands)
	}
	if len(snapS2.SkillOptions) != 1 || snapS2.SkillOptions[0].Name != "go-test" {
		t.Fatalf("same-workspace switch cleared SkillOptions: %+v", snapS2.SkillOptions)
	}

	// Populate s2 then switch back to s1 -> optimistic instant switch (HistoryReady=true, zero blank flash).
	s.applyFollowFrame(json.RawMessage(`{"type":"snapshot","cursor":1,"hasMore":false,"records":[{"type":"event","event":{"type":"user/message","seq":1,"data":{"content":[{"type":"text","text":"hello s2"}]}}}]}`))
	s.Select("s1")
	snapS1 := s.Snapshot()
	if !snapS1.HistoryReady || snapS1.HistoryLoading || len(snapS1.Messages) == 0 || snapS1.Messages[0].Body != "hello s1" {
		t.Fatalf("LRU switch back to s1 did not restore immediately: ready=%v loading=%v msgs=%+v", snapS1.HistoryReady, snapS1.HistoryLoading, snapS1.Messages)
	}

	// Switch to s3 in a DIFFERENT workspace -> skills & commands must be cleared.
	s.Select("s3")
	snapS3 := s.Snapshot()
	if len(snapS3.SlashCommands) != 0 || len(snapS3.SkillOptions) != 0 {
		t.Fatalf("cross-workspace switch leaked commands/skills: cmds=%d skills=%d", len(snapS3.SlashCommands), len(snapS3.SkillOptions))
	}

	// Bounded LRU eviction: insert defaultSessionCacheLimit+4 sessions and confirm oldest is evicted.
	for i := 0; i < defaultSessionCacheLimit+4; i++ {
		id := fmt.Sprintf("evict-%d", i)
		s.Select(id)
		s.applyFollowFrame(json.RawMessage(fmt.Sprintf(`{"type":"snapshot","cursor":1,"records":[{"type":"event","event":{"type":"user/message","seq":1,"data":{"content":[{"type":"text","text":"msg %s"}]}}}]}`, id)))
	}
	s.Select("evict-0")
	if snap := s.Snapshot(); snap.HistoryReady || len(snap.Messages) != 0 {
		t.Fatalf("oldest evicted session unexpectedly hit LRU: ready=%v msgs=%d", snap.HistoryReady, len(snap.Messages))
	}

	// Host replacement clears LRU so stale sessions cannot leak across hosts.
	s.Attach(stubOKClient("http://new-host.invalid"), nil)
	s.Select("evict-25")
	if snap := s.Snapshot(); snap.HistoryReady || len(snap.Messages) != 0 {
		t.Fatal("LRU leaked across Host replacement")
	}
}

func TestTrajectoryLazyProjectionAndOnDemandSnapshot(t *testing.T) {
	s := benchmarkStore(60)
	s.SetTrajectoryEnabled(false)
	if s.TrajectoryEnabled() {
		t.Fatal("expected TrajectoryEnabled() == false")
	}

	// Default hot Snapshot/Published skips all Trajectory fields.
	hot := s.Snapshot()
	if hot.TrajectoryMessages != nil || hot.TrajectoryRequests != nil || hot.TrajectoryRequestHeaders != nil {
		t.Fatalf("lazy mode populated Trajectory fields in ViewState: msgs=%d reqs=%d hdrs=%d",
			len(hot.TrajectoryMessages), len(hot.TrajectoryRequests), len(hot.TrajectoryRequestHeaders))
	}
	if len(hot.Messages) == 0 {
		t.Fatal("lazy mode dropped chat Messages")
	}

	// On-demand TrajectorySnapshot returns full independent copy even while disabled.
	tv := s.TrajectorySnapshot()
	if len(tv.Messages) == 0 || len(tv.Requests) == 0 {
		t.Fatalf("on-demand TrajectorySnapshot empty: msgs=%d reqs=%d", len(tv.Messages), len(tv.Requests))
	}
	tv.Requests[0].Status = "mutated"
	tv2 := s.TrajectoryPublished()
	if tv2.Requests[0].Status == "mutated" {
		t.Fatal("TrajectorySnapshot mutation leaked into cached TrajectoryPublished")
	}

	// Re-enabling populates ViewState accurately.
	s.SetTrajectoryEnabled(true)
	full := s.Snapshot()
	if len(full.TrajectoryMessages) != len(tv2.Messages) || len(full.TrajectoryRequests) != len(tv2.Requests) {
		t.Fatalf("re-enabled Trajectory mismatch: snap=(%d,%d) lazy=(%d,%d)",
			len(full.TrajectoryMessages), len(full.TrajectoryRequests), len(tv2.Messages), len(tv2.Requests))
	}
}

func TestAssistantStream16msCoalescingAndImmediateTerminalFlush(t *testing.T) {
	s := New()
	var notifies atomic.Int32
	notifyCh := make(chan struct{}, 64)
	s.Attach(nil, func() {
		notifies.Add(1)
		select {
		case notifyCh <- struct{}{}:
		default:
		}
	})
	s.Select("stream-session")
	s.applyFollowFrame(json.RawMessage(`{"type":"snapshot","cursor":1,"records":[{"type":"event","event":{"type":"step/start","seq":1,"data":{"turn":1,"step":1}}}]}`))
	s.applyFollowFrame(json.RawMessage(`{"type":"assistant-stream","frame":{"type":"start","attemptId":"a1","revision":1,"startedAfterSeq":1,"turn":1,"step":1}}`))

	// Reset count before rapid burst.
	notifies.Store(0)
	for len(notifyCh) > 0 {
		<-notifyCh
	}

	const burst = 20
	for i := 0; i < burst; i++ {
		raw := fmt.Sprintf(`{"type":"assistant-stream","frame":{"type":"chunk","attemptId":"a1","revision":%d,"index":%d,"time":%d,"chunk":{"type":"text-delta","text":"w"}}}`, i+2, i, 1000+i)
		s.applyFollowFrame(json.RawMessage(raw))
	}
	// State is immediately up-to-date synchronously, while notify count is coalesced.
	snap := s.Snapshot()
	if len(snap.Messages) == 0 || len(snap.Messages[len(snap.Messages)-1].Body) != burst {
		t.Fatalf("streaming state lagged behind burst: msgs=%+v", snap.Messages)
	}
	immediate := notifies.Load()
	if immediate >= burst {
		t.Fatalf("expected 16ms coalescing to reduce notifies below %d, got %d", burst, immediate)
	}

	// Wait for trailing 16ms flush to fire.
	time.Sleep(40 * time.Millisecond)
	afterTrailing := notifies.Load()
	if afterTrailing <= immediate {
		t.Fatalf("trailing 16ms notify did not fire: immediate=%d after=%d", immediate, afterTrailing)
	}

	// Terminal abandoned end must flush immediately without waiting 16ms.
	beforeEnd := notifies.Load()
	endRaw := fmt.Sprintf(`{"type":"assistant-stream","frame":{"type":"end","attemptId":"a1","revision":%d,"index":%d,"outcome":{"kind":"abandoned"}}}`, burst+2, burst)
	s.applyFollowFrame(json.RawMessage(endRaw))
	if notifies.Load() <= beforeEnd {
		t.Fatal("terminal assistant-stream end did not flush notify immediately")
	}
}

func TestCreateSessionBlankReuseNonBlocking(t *testing.T) {
	entered := make(chan struct{})
	release := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		if strings.HasSuffix(r.URL.Path, "/session/blankReuse") {
			close(entered)
			<-release
			rpcOK(w, body["rpcId"], map[string]any{"reusable": true})
			return
		}
		rpcOK(w, body["rpcId"], map[string]any{})
	}))
	defer srv.Close()

	st := New()
	st.mu.Lock()
	st.cwd = "/ws"
	st.items = []Item{{SessionID: "blank-slow", Blank: true, Cwd: "/ws"}}
	st.mu.Unlock()
	done := make(chan struct{}, 1)
	st.Attach(&dsh.Client{Endpoint: dsh.Endpoint{Origin: srv.URL}}, func() {
		if st.Snapshot().SelectedID == "blank-slow" {
			select {
			case done <- struct{}{}:
			default:
			}
		}
	})

	start := time.Now()
	st.CreateSession(context.Background())
	if elapsed := time.Since(start); elapsed > 100*time.Millisecond {
		close(release)
		t.Fatalf("CreateSession blocked caller for %v while blankReuse RPC was pending", elapsed)
	}

	select {
	case <-entered:
	case <-time.After(2 * time.Second):
		close(release)
		t.Fatal("background blankReuse RPC never started")
	}
	close(release)
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("async blankReuse did not complete selection")
	}
}

type lockProbePreferenceStore struct {
	store  *Store
	writes atomic.Int32
	mu     sync.Mutex
	values map[string]config.WorkspaceNavigationPreferences
}

func (p *lockProbePreferenceStore) WorkspaceNavigationPreferences(key string) (config.WorkspaceNavigationPreferences, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return cloneWorkspacePreferences(p.values[key]), nil
}

func (p *lockProbePreferenceStore) SaveWorkspaceNavigationPreferences(key string, prefs config.WorkspaceNavigationPreferences) error {
	// Calling p.store.Snapshot() here would immediately deadlock if Store.mu were
	// still held across persistence I/O.
	_ = p.store.Snapshot()
	p.writes.Add(1)
	p.mu.Lock()
	p.values[key] = cloneWorkspacePreferences(prefs)
	p.mu.Unlock()
	return nil
}

func TestWorkspacePreferencesNeverHoldStoreMutexAndAsyncDebounceFlush(t *testing.T) {
	s := New()
	probe := &lockProbePreferenceStore{store: s, values: map[string]config.WorkspaceNavigationPreferences{}}
	client := &dsh.Client{Endpoint: dsh.Endpoint{Origin: "http://pref-probe.invalid"}}
	s.Attach(client, nil)
	s.BindWorkspacePreferences(client, "host-probe", probe)
	s.applyWorkspaceFollowFrame(json.RawMessage(`{"type":"baseline","value":{"scratchCwd":"/scratch","items":[{"workspaceId":"w1","path":"/w1","sessionIds":[]}],"pinnedSessionIds":[],"archivedSessionIds":[]}}`))

	// Synchronous fallback path must not hold Store.mu across Save.
	s.ToggleWorkspaceGroup("w1")
	if probe.writes.Load() != 1 {
		t.Fatalf("expected 1 unlocked save, got %d", probe.writes.Load())
	}

	// Async debounce mode coalesces burst toggles and flushes on demand.
	probe.writes.Store(0)
	s.SetWorkspacePreferencesAsync(true)
	s.SetWorkspacePreferencesDebounce(40 * time.Millisecond)
	for i := 0; i < 8; i++ {
		s.ToggleWorkspaceGroup("w1")
	}
	if err := s.FlushWorkspacePreferences(); err != nil {
		t.Fatalf("FlushWorkspacePreferences: %v", err)
	}
	if n := probe.writes.Load(); n != 1 {
		t.Fatalf("expected debounced burst to coalesce into 1 write, got %d", n)
	}

	// Real *config.Store integration: non-blocking async save + durable Flush.
	dir := t.TempDir()
	cfgStore, err := config.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	cfgStore.SetAsyncDebounce(30 * time.Millisecond)
	s2 := New()
	s2.Attach(client, nil)
	s2.BindWorkspacePreferences(client, "host-real", cfgStore)
	s2.applyWorkspaceFollowFrame(json.RawMessage(`{"type":"baseline","value":{"scratchCwd":"/scratch","items":[{"workspaceId":"w1","path":"/w1","sessionIds":[]}],"pinnedSessionIds":[],"archivedSessionIds":[]}}`))
	s2.SetWorkspaceGrouping("workspace-tree")
	s2.ToggleWorkspaceGroup("w1")
	if err := s2.FlushWorkspacePreferences(); err != nil {
		t.Fatalf("FlushWorkspacePreferences real config: %v", err)
	}
	reopened, err := config.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	scope := preferenceHash("host", "host-real", "/scratch")
	persisted, err := reopened.WorkspaceNavigationPreferences(scope)
	if err != nil {
		t.Fatal(err)
	}
	if persisted.GroupBy != "workspace-tree" {
		t.Fatalf("persisted GroupBy = %q, want workspace-tree", persisted.GroupBy)
	}
}

func TestHistoryDerivedMemoStoreO1FastPathDuringStreaming(t *testing.T) {
	s := benchmarkStore(200)
	s.messages = append(s.messages, streamingRow())
	s.assistantStream.active = &assistantStreamAttempt{AttemptID: "fast-path", Turn: 99, Step: 1}
	v1 := s.Published().Value
	key1 := v1.DurableHistoryProjectionKey()
	if !key1.Valid() {
		t.Fatal("expected valid DurableHistoryProjectionKey from Store")
	}
	var memo HistoryDerivedMemo
	tok1, ms1, steps1 := memo.RecordedPerformance(v1)
	if !memo.keyFastPath {
		t.Fatal("expected HistoryDerivedMemo to take O(1) key fast path")
	}

	// Simulate streaming chunks mutating only the volatile tail row.
	s.mu.Lock()
	s.messages[len(s.messages)-1].Body += " more streamed text"
	s.markStreamDirtyLocked()
	s.mu.Unlock()
	v2 := s.Published().Value
	if v2.DurableHistoryProjectionKey() != key1 {
		t.Fatal("volatile streaming tick invalidated DurableHistoryProjectionKey")
	}
	tok2, ms2, steps2 := memo.RecordedPerformance(v2)
	if tok1 != tok2 || ms1 != ms2 || steps1 != steps2 {
		t.Fatalf("O(1) fast path changed result: (%v,%v,%d) vs (%v,%v,%d)", tok1, ms1, steps1, tok2, ms2, steps2)
	}

	// Appending a durable assistant message must advance DurableHistoryProjectionKey.
	s.mu.Lock()
	s.messages = append(s.messages[:len(s.messages)-1], Message{
		ID: "durable-extra", Role: "assistant", Seq: 999,
		Raw: json.RawMessage(`{"type":"assistant/message","seq":999,"data":{"turn":99,"step":1,"usage":{"outputTokens":77}}}`),
	})
	s.markHistoryDirtyLocked()
	s.mu.Unlock()
	v3 := s.Published().Value
	if v3.DurableHistoryProjectionKey() == key1 {
		t.Fatal("durable append did not advance DurableHistoryProjectionKey")
	}
	_, _, steps3 := memo.RecordedPerformance(v3)
	if steps3 != steps1+1 {
		t.Fatalf("steps after durable append = %d, want %d", steps3, steps1+1)
	}
}

func TestSessionStoreConcurrentReadWriteSafety(t *testing.T) {
	s := benchmarkStore(80)
	s.messages = append(s.messages, streamingRow())
	s.assistantStream.active = &assistantStreamAttempt{AttemptID: "race-1", Turn: 10, Step: 1}

	var wg sync.WaitGroup
	for r := 0; r < 4; r++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			frame := s.BeginFrame()
			defer frame.EndFrame()
			for i := 0; i < 60; i++ {
				_ = s.Published()
				_ = frame.Snapshot()
				_ = s.Snapshot()
				_ = s.TrajectorySnapshot()
			}
		}()
	}
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 60; i++ {
			s.SetTrajectoryEnabled(i%2 == 0)
			s.mu.Lock()
			if n := len(s.messages); n > 0 {
				s.messages[n-1].Body += "z"
			}
			s.markStreamDirtyLocked()
			s.mu.Unlock()
			s.fireStream()
		}
		s.fire()
	}()
	wg.Wait()
	s.Detach()
}

func TestStreamingWithQueuedComposerPendingDoesNotDuplicateOrMutatePublished(t *testing.T) {
	s := New()
	client := stubOKClient("http://queued-stream.invalid")
	s.Attach(client, nil)
	s.Select("q-session")
	s.applyFollowFrame(json.RawMessage(`{"type":"snapshot","cursor":1,"records":[{"type":"event","event":{"type":"step/start","seq":1,"data":{"turn":1,"step":1}}}]}`))
	s.applyFollowFrame(json.RawMessage(`{"type":"assistant-stream","frame":{"type":"start","attemptId":"a1","revision":1,"startedAfterSeq":1,"turn":1,"step":1}}`))

	// Install a queued pending submission and an existing PlanBoard.Queue entry.
	target, err := s.CaptureComposerTarget("q-session")
	if err != nil {
		t.Fatalf("CaptureComposerTarget: %v", err)
	}
	s.mu.Lock()
	s.composerPending = map[string]*composerPending{
		"req-queued": {
			PendingSubmission: PendingSubmission{
				RequestID: "req-queued",
				SessionID: "q-session",
				Text:      "queued follow-up",
				Placement: "queued",
				Order:     1,
			},
			target: target,
		},
	}
	s.planBoard.Queue = []PlanQueueItem{
		{ID: "q1", RequestID: "req-hide", Preview: "hidden item"},
		{ID: "q2", RequestID: "req-keep", Preview: "visible item"},
	}
	s.invalidateViewLocked()
	s.mu.Unlock()

	first := s.Published().Value
	if len(first.PlanBoard.Pending) != 1 || first.PlanBoard.Pending[0].RequestID != "req-queued" {
		t.Fatalf("initial queued pending mismatch: %+v", first.PlanBoard.Pending)
	}
	if len(first.PlanBoard.Queue) != 2 || first.PlanBoard.Queue[0].ID != "q1" || first.PlanBoard.Queue[1].ID != "q2" {
		t.Fatalf("initial queue mismatch: %+v", first.PlanBoard.Queue)
	}
	firstQueueCopy := append([]PlanQueueItem(nil), first.PlanBoard.Queue...)
	firstPendingCopy := append([]PendingSubmission(nil), first.PlanBoard.Pending...)

	// Hide q1 via composerObserved (non-queued placement) and send 10 streaming chunks.
	s.mu.Lock()
	s.composerObserved = map[string]composerObserved{
		"req-hide": {target: target, placement: "transcript"},
	}
	s.mu.Unlock()

	for i := 0; i < 10; i++ {
		raw := fmt.Sprintf(`{"type":"assistant-stream","frame":{"type":"chunk","attemptId":"a1","revision":%d,"index":%d,"time":%d,"chunk":{"type":"text-delta","text":"c"}}}`, i+2, i, 2000+i)
		s.applyFollowFrame(json.RawMessage(raw))
		cur := s.Published().Value
		if len(cur.PlanBoard.Pending) != 1 {
			t.Fatalf("chunk %d duplicated PlanBoard.Pending: len=%d (%+v)", i, len(cur.PlanBoard.Pending), cur.PlanBoard.Pending)
		}
		if len(cur.PlanBoard.Queue) != 1 || cur.PlanBoard.Queue[0].ID != "q2" {
			t.Fatalf("chunk %d queue filter mismatch: %+v", i, cur.PlanBoard.Queue)
		}
	}

	// Previously published snapshot slices must remain completely unmutated.
	if len(first.PlanBoard.Pending) != len(firstPendingCopy) || first.PlanBoard.Pending[0].RequestID != firstPendingCopy[0].RequestID {
		t.Fatalf("streaming mutated prior published PlanBoard.Pending: %+v", first.PlanBoard.Pending)
	}
	if len(first.PlanBoard.Queue) != len(firstQueueCopy) || first.PlanBoard.Queue[0].ID != firstQueueCopy[0].ID || first.PlanBoard.Queue[1].ID != firstQueueCopy[1].ID {
		t.Fatalf("streaming mutated prior published PlanBoard.Queue backing array: %+v", first.PlanBoard.Queue)
	}
}

type blockingPreferenceStore struct {
	entered   chan struct{}
	release   chan struct{}
	firstOnce sync.Once
	mu        sync.Mutex
	writes    int
	saved     map[string]config.WorkspaceNavigationPreferences
}

func (b *blockingPreferenceStore) WorkspaceNavigationPreferences(key string) (config.WorkspaceNavigationPreferences, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return cloneWorkspacePreferences(b.saved[key]), nil
}

func (b *blockingPreferenceStore) SaveWorkspaceNavigationPreferences(key string, prefs config.WorkspaceNavigationPreferences) error {
	b.firstOnce.Do(func() {
		close(b.entered)
		<-b.release
	})
	b.mu.Lock()
	b.writes++
	b.saved[key] = cloneWorkspacePreferences(prefs)
	b.mu.Unlock()
	return nil
}

func TestFlushWorkspacePreferencesWaitsForInFlightTimerWriteAndLatestSeq(t *testing.T) {
	s := New()
	repo := &blockingPreferenceStore{
		entered: make(chan struct{}),
		release: make(chan struct{}),
		saved:   map[string]config.WorkspaceNavigationPreferences{},
	}
	client := stubOKClient("http://flush-wait.invalid")
	s.Attach(client, nil)
	s.SetWorkspacePreferencesAsync(true)
	s.SetWorkspacePreferencesDebounce(15 * time.Millisecond)
	s.BindWorkspacePreferences(client, "host-flush", repo)
	s.applyWorkspaceFollowFrame(json.RawMessage(`{"type":"baseline","value":{"scratchCwd":"/scratch","items":[{"workspaceId":"w1","path":"/w1","sessionIds":[]}],"pinnedSessionIds":[],"archivedSessionIds":[]}}`))

	// First edit triggers the debounced timer, which blocks inside Save.
	s.SetWorkspaceGrouping("flat")
	select {
	case <-repo.entered:
	case <-time.After(2 * time.Second):
		t.Fatal("debounced timer write did not start")
	}

	// While the first write is in flight, apply a newer edit and call FlushWorkspacePreferences.
	s.SetWorkspaceGrouping("workspace-tree")
	flushDone := make(chan error, 1)
	go func() {
		flushDone <- s.FlushWorkspacePreferences()
	}()

	// Flush must NOT return while the in-flight timer write is still blocked.
	select {
	case err := <-flushDone:
		t.Fatalf("FlushWorkspacePreferences returned prematurely while write was in flight: err=%v", err)
	case <-time.After(50 * time.Millisecond):
	}

	// Unblock the in-flight write; Flush must finish and persist the latest seq ("workspace-tree").
	close(repo.release)
	select {
	case err := <-flushDone:
		if err != nil {
			t.Fatalf("FlushWorkspacePreferences failed: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("FlushWorkspacePreferences did not complete after unblocking write")
	}

	scope := preferenceHash("host", "host-flush", "/scratch")
	repo.mu.Lock()
	got := repo.saved[scope]
	writes := repo.writes
	repo.mu.Unlock()
	if got.GroupBy != "workspace-tree" || writes < 2 {
		t.Fatalf("FlushWorkspacePreferences lost latest edit or skipped in-flight write: groupBy=%q writes=%d", got.GroupBy, writes)
	}
}

func BenchmarkSnapshotWhileStreamingLazyTrajectory(b *testing.B) {
	for _, size := range []int{500, 2000, 8000} {
		b.Run(fmt.Sprintf("messages_%d_frame_stream_only_lazy", size), func(b *testing.B) {
			s := benchmarkStore(size)
			s.SetTrajectoryEnabled(false)
			s.messages = append(s.messages, streamingRow())
			s.assistantStream.active = &assistantStreamAttempt{AttemptID: "bench-lazy", Turn: 999, Step: 1}
			frame := s.BeginFrame()
			defer frame.EndFrame()
			_ = frame.Snapshot()
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				s.mu.Lock()
				s.messages[len(s.messages)-1].Body += "x"
				s.markStreamDirtyLocked()
				s.mu.Unlock()
				_ = frame.Snapshot()
			}
		})
	}
}

func BenchmarkHistoryDerivedStoreFastPath(b *testing.B) {
	for _, rows := range []int{400, 2000} {
		s := benchmarkStore(rows)
		s.messages = append(s.messages, streamingRow())
		v := s.Published().Value
		b.Run(fmt.Sprintf("rows_%d/performance_store_o1", rows), func(b *testing.B) {
			var memo HistoryDerivedMemo
			memo.RecordedPerformance(v)
			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				historyDerivedPerfSink.tokens, historyDerivedPerfSink.millis, historyDerivedPerfSink.steps = memo.RecordedPerformance(v)
			}
		})
		b.Run(fmt.Sprintf("rows_%d/combined_store_o1", rows), func(b *testing.B) {
			var memo HistoryDerivedMemo
			memo.RecordedPerformance(v)
			memo.ScheduleCards(v)
			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				memo.RecordedPerformance(v)
				historyDerivedCardsSink = memo.ScheduleCards(v)
			}
		})
	}
}
