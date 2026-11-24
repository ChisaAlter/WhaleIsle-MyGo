package session

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"testing"

	"whaleisle/internal/dsh"
	"whaleisle/internal/projectflow"
	"whaleisle/internal/state"
	"whaleisle/internal/store/storelocale"
)

func TestPublishedReusesSharedVersionWithoutAlloc(t *testing.T) {
	s := snapshotCacheFixture(t)
	first := s.Published()
	if first == nil {
		t.Fatal("Published returned nil")
	}
	if first.Rev == 0 {
		t.Fatalf("Published Rev = 0, want >= 1")
	}
	second := s.Published()
	if second != first {
		t.Fatalf("unchanged Published returned a new pointer: %p -> %p", first, second)
	}
	if allocs := testing.AllocsPerRun(100, func() {
		v := s.Published()
		if v != first {
			t.Fatalf("Published pointer changed during unchanged reads")
		}
	}); allocs != 0 {
		t.Fatalf("unchanged Published allocs = %v, want 0", allocs)
	}
}

func TestPublishedAdvancesRevAndProtectsPriorVersions(t *testing.T) {
	s := snapshotCacheFixture(t)
	v1 := s.Published()
	if len(v1.Value.Items) == 0 && len(v1.Value.Workspaces) == 0 {
		t.Fatal("fixture missing workspaces")
	}
	origSelected := v1.Value.SelectedID
	origMessagesLen := len(v1.Value.Messages)
	origFirstBody := ""
	if origMessagesLen > 0 {
		origFirstBody = v1.Value.Messages[0].Body
	}
	origCollapsed := v1.Value.WorkspaceLayout.Collapsed["one"]

	// A state-only mutation advances Rev and keeps prior versions untouched.
	s.SetLastAction("updated action")
	v2 := s.Published()
	if v2 == v1 || v2.Rev != v1.Rev+1 {
		t.Fatalf("SetLastAction rev %d -> %d, ptr %p -> %p", v1.Rev, v2.Rev, v1, v2)
	}
	if v2.Value.LastAction != "updated action" || v1.Value.LastAction == "updated action" {
		t.Fatalf("SetLastAction leaked or missed: v1=%q v2=%q", v1.Value.LastAction, v2.Value.LastAction)
	}

	// Mutating selection, messages, and workspace collapse state advances Rev without mutating v1/v2.
	s.Select("next-session")
	v3 := s.Published()
	if v3 == v2 || v3.Rev != v2.Rev+1 {
		t.Fatalf("Select rev %d -> %d", v2.Rev, v3.Rev)
	}
	if v3.Value.SelectedID != "next-session" || v1.Value.SelectedID != origSelected {
		t.Fatalf("Select leaked into prior version: v1=%q v3=%q", v1.Value.SelectedID, v3.Value.SelectedID)
	}

	s.AppendStreamDelta("assistant", "streamed-chunk")
	v4 := s.Published()
	if v4 == v3 || v4.Rev != v3.Rev+1 {
		t.Fatalf("AppendStreamDelta rev %d -> %d", v3.Rev, v4.Rev)
	}
	if len(v1.Value.Messages) != origMessagesLen {
		t.Fatalf("AppendStreamDelta mutated v1 messages len: %d -> %d", origMessagesLen, len(v1.Value.Messages))
	}
	if origMessagesLen > 0 && v1.Value.Messages[0].Body != origFirstBody {
		t.Fatal("AppendStreamDelta mutated v1 message body")
	}

	s.mu.Lock()
	s.workspaceLayout.view.Collapsed["one"] = false
	s.invalidateViewLocked()
	s.mu.Unlock()
	v5 := s.Published()
	if v5 == v4 || v5.Rev != v4.Rev+1 {
		t.Fatalf("workspaceLayout update rev %d -> %d", v4.Rev, v5.Rev)
	}
	if v1.Value.WorkspaceLayout.Collapsed["one"] != origCollapsed || v5.Value.WorkspaceLayout.Collapsed["one"] {
		t.Fatalf("workspace collapse mutated prior version: v1=%v v5=%v", v1.Value.WorkspaceLayout.Collapsed["one"], v5.Value.WorkspaceLayout.Collapsed["one"])
	}

	// Mutable Snapshot() copy must never alias Published().
	snap := s.Snapshot()
	snap.LastAction = "caller-edited"
	if len(snap.Messages) > 0 {
		snap.Messages[0].Body = "caller-edited"
	}
	if s.Published().Value.LastAction == "caller-edited" {
		t.Fatal("Snapshot mutation leaked into Published")
	}
	if len(s.Published().Value.Messages) > 0 && s.Published().Value.Messages[0].Body == "caller-edited" {
		t.Fatal("Snapshot message mutation leaked into Published")
	}
}

func TestPublishedConcurrentReadWriteRaceSafe(t *testing.T) {
	s := snapshotCacheFixture(t)
	_ = s.Published()
	s.mu.Lock()
	gen := s.followGeneration
	s.mu.Unlock()

	var wg sync.WaitGroup
	for w := 0; w < 4; w++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for i := 0; i < 40; i++ {
				switch i % 4 {
				case 0:
					s.SetLastAction("action")
				case 1:
					s.AppendStreamDelta("assistant", "x")
				case 2:
					s.applyWorkspaceFollowFrame(json.RawMessage(`{"type":"baseline","value":{"items":[{"workspaceId":"one","path":"/one","title":"One","sessionIds":["a","b"]}],"pinnedSessionIds":[],"archivedSessionIds":[],"scratchCwd":"/scratch"}}`))
				case 3:
					s.reportFollowError(context.Background(), "bench", gen, "follow: ", errors.New("dial tcp 127.0.0.1:1: connection refused"))
				}
			}
		}(w)
	}
	for r := 0; r < 8; r++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			var lastRev uint64
			for i := 0; i < 150; i++ {
				v := s.Published()
				if v == nil || v.Rev == 0 {
					t.Errorf("invalid published version: %+v", v)
					return
				}
				if v.Rev < lastRev {
					t.Errorf("published rev regressed: %d -> %d", lastRev, v.Rev)
					return
				}
				lastRev = v.Rev
				_ = v.Value.SelectedID
				_ = len(v.Value.Messages)
				_ = len(v.Value.Items)
				if i%10 == 0 {
					snap := s.Snapshot()
					snap.LastAction = "mutated"
				}
			}
		}()
	}
	wg.Wait()
}

func TestViewStateFaultsDoNotLeakRawErrorText(t *testing.T) {
	rawErr := errors.New("dial tcp 10.99.88.77:9999: connectex: connection refused")
	s := New()
	s.Attach(&dsh.Client{}, nil)
	s.mu.Lock()
	s.selectedID = "s1"
	s.commandCatalog = commandCatalog{client: s.client, sessionID: "s1", loaded: true, err: rawErr, fault: state.FaultOf(rawErr)}
	s.setHistoryErrLocked("follow: ", rawErr)
	s.setModelsErrLocked(rawErr)
	s.setPermissionsErrLocked(rawErr)
	s.setAgentPresetsErrLocked(rawErr)
	s.setSkillsErrLocked(rawErr)
	s.setSearchErrLocked(rawErr)
	s.goalReadError = goalReadErrorState{client: s.client, sessionID: "s1", generation: s.followGeneration, fault: state.FaultOf(rawErr)}
	s.setHistoryPageErrLocked(rawErr)
	s.invalidateViewLocked()
	s.mu.Unlock()

	v := s.Published().Value
	faults := []*state.Fault{
		v.SlashCommandsErr,
		v.HistoryError,
		v.ModelsErr,
		v.PermissionsErr,
		v.AgentPresetsErr,
		v.SkillsErr,
		v.SearchErr,
		v.GoalReadError,
		v.HistoryPageErr,
	}
	for i, f := range faults {
		if f == nil {
			t.Fatalf("fault[%d] is nil", i)
		}
		text := f.SentenceText()
		if text == "" || strings.Contains(text, "10.99.88.77") || strings.Contains(text, "connectex") {
			t.Fatalf("fault[%d] leaked raw error: %q", i, text)
		}
	}
}

func TestBoundaryFaultClassificationAndNoRawErrorLeak(t *testing.T) {
	rawErr := errors.New("dial tcp 10.99.88.77:9999: connectex: connection refused (secret-token)")
	assertSafe := func(label, got string) {
		t.Helper()
		if got == "" {
			t.Fatalf("%s is empty", label)
		}
		for _, needle := range []string{"10.99.88.77", "connectex", "secret-token"} {
			if strings.Contains(got, needle) {
				t.Fatalf("%s leaked raw error token %q: %q", label, needle, got)
			}
		}
	}

	s := snapshotCacheFixture(t)
	v0 := s.Published()

	// 1. setLastActionFaultLocked advances Published().Rev and strips raw error text.
	s.mu.Lock()
	s.setLastActionFaultLocked("test.action", "", rawErr)
	s.mu.Unlock()
	v1 := s.Published()
	if v1.Rev <= v0.Rev {
		t.Fatalf("LastAction fault did not advance Published Rev: %d -> %d", v0.Rev, v1.Rev)
	}
	assertSafe("LastAction", v1.Value.LastAction)

	// Prefixed contract ("queue: ") preserves prefix without leaking raw error text.
	s.mu.Lock()
	s.setLastActionFaultLocked("test.queue", "queue: ", rawErr)
	s.mu.Unlock()
	v2 := s.Published()
	if !strings.HasPrefix(v2.Value.LastAction, "queue: ") {
		t.Fatalf("prefixed LastAction lost prefix: %q", v2.Value.LastAction)
	}
	assertSafe("prefixed LastAction", v2.Value.LastAction)

	// storelocale.AuthoredError copy is preserved verbatim across faultText.
	authored := storelocale.Authored("authored validation message")
	if got := faultText("test.authored", authored); got != "authored validation message" {
		t.Fatalf("AuthoredError was not preserved: %q", got)
	}

	// 2. workspaceNavigation.view.Error updates Published() without raw text.
	s.OpenWorkspacePicker()
	nav := s.Published().Value.WorkspaceNavigation
	if nav == nil {
		t.Fatal("WorkspaceNavigation is nil after OpenWorkspacePicker")
	}
	beforeNavRev := s.Published().Rev
	s.SetWorkspaceNavigationError(nav.Token, rawErr)
	vNav := s.Published()
	if vNav.Rev <= beforeNavRev {
		t.Fatalf("SetWorkspaceNavigationError did not advance Published Rev: %d -> %d", beforeNavRev, vNav.Rev)
	}
	if vNav.Value.WorkspaceNavigation == nil {
		t.Fatal("WorkspaceNavigation became nil")
	}
	assertSafe("WorkspaceNavigation.Error", vNav.Value.WorkspaceNavigation.Error)

	// 3. agentNavigation view.Error, ActionStatus, and Resources[path].Error strip raw text.
	s.mu.Lock()
	s.selectedID = "parent"
	s.planHistoryBinding = "binding"
	s.items = append(s.items, Item{SessionID: "parent", AgentAvailable: true})
	addr := AgentSessionAddress{Kind: "subagent", ParentSessionID: "parent", ChildSessionID: "child", Mode: "continuable"}
	s.agentNavigation = agentNavigationState{
		client:     s.client,
		mux:        s.mux,
		generation: s.followGeneration,
		binding:    "binding",
		root:       "parent",
		view: AgentNavigationView{
			Token:     11,
			SessionID: "child",
			Address:   addr,
			Process:   projectflow.ProcessSnapshot{Live: true},
		},
	}
	s.mu.Unlock()

	s.SetAgentResourceStatus(11, addr, "report.xlsx", false, rawErr)
	agentView := s.AgentNavigationSnapshot()
	assertSafe("AgentNavigation.Resources.Error", agentView.Resources["report.xlsx"].Error)

	s.AddAgentNavigationImages(11, addr, nil, rawErr)
	agentView = s.AgentNavigationSnapshot()
	assertSafe("AgentNavigation.ActionStatus", agentView.ActionStatus)

	s.mu.Lock()
	s.agentNavigation.view.Error = faultText("test.agentLive", rawErr)
	s.mu.Unlock()
	assertSafe("AgentNavigation.Error", s.AgentNavigationSnapshot().Error)
}

func TestPublishedConcurrentFaultWritesAndReadOnlyReads(t *testing.T) {
	s := snapshotCacheFixture(t)
	s.OpenWorkspacePicker()
	nav := s.Published().Value.WorkspaceNavigation
	if nav == nil {
		t.Fatal("expected open workspace picker")
	}
	rawErr := errors.New("dial tcp 10.99.88.77:9999: connectex: connection refused")

	var wg sync.WaitGroup
	for w := 0; w < 4; w++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for i := 0; i < 40; i++ {
				switch (i + id) % 3 {
				case 0:
					s.mu.Lock()
					s.setLastActionFaultLocked("bench.fault", "", rawErr)
					s.mu.Unlock()
				case 1:
					s.SetWorkspaceNavigationError(nav.Token, rawErr)
				case 2:
					s.SetLastAction("ok")
				}
			}
		}(w)
	}
	for r := 0; r < 6; r++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			var lastRev uint64
			for i := 0; i < 150; i++ {
				v := s.Published()
				if v == nil || v.Rev == 0 {
					t.Errorf("invalid Published version: %+v", v)
					return
				}
				if v.Rev < lastRev {
					t.Errorf("Published rev regressed: %d -> %d", lastRev, v.Rev)
					return
				}
				lastRev = v.Rev
				if strings.Contains(v.Value.LastAction, "10.99.88.77") {
					t.Errorf("LastAction leaked raw error: %q", v.Value.LastAction)
					return
				}
				if v.Value.WorkspaceNavigation != nil && strings.Contains(v.Value.WorkspaceNavigation.Error, "10.99.88.77") {
					t.Errorf("WorkspaceNavigation.Error leaked raw error: %q", v.Value.WorkspaceNavigation.Error)
					return
				}
			}
		}()
	}
	wg.Wait()
}
