package session

import (
	"encoding/json"
	"fmt"
	"reflect"
	"sync"
	"testing"

	"whaleisle/internal/dsh"
	"whaleisle/internal/dsh/api"
	"whaleisle/internal/dsh/events"
	"whaleisle/internal/workflowrun"
)

// snapshotCacheFixture builds a store that owns every dimension the earlier
// cache attempts broke: workspace membership rows, collapsed grouping state,
// an open navigation dialog and parsed trajectory requests.
func snapshotCacheFixture(t *testing.T) *Store {
	t.Helper()
	s := benchmarkStore(30)
	s.Attach(&dsh.Client{Endpoint: dsh.Endpoint{Origin: "http://127.0.0.1:1"}}, nil)
	s.applyWorkspaceFollowFrame(json.RawMessage(`{"type":"baseline","value":{"items":[{"workspaceId":"one","path":"/one","title":"One","sessionIds":["a","b"]}],"pinnedSessionIds":[],"archivedSessionIds":[],"scratchCwd":"/scratch"}}`))
	s.mu.Lock()
	s.workspaceLayout.view.Collapsed = map[string]bool{"one": true}
	s.workspaceLayout.view.SessionOrders = map[string][]string{"one": {"a", "b"}}
	s.openWorkspaceNavigationLocked("pick", "")
	s.mu.Unlock()
	return s
}

// TestSnapshotCacheDoesNotAlias pins the package contract for the cached path:
// a caller may mutate anything it receives, and the next Snapshot must still
// report the stored values. Workspaces, collapse state, the navigation view,
// messages and trajectory requests are each mutated in place.
func TestSnapshotCacheDoesNotAlias(t *testing.T) {
	s := snapshotCacheFixture(t)
	first := s.Snapshot()
	if len(first.Workspaces) != 1 || len(first.Workspaces[0].SessionIDs) != 2 {
		t.Fatalf("fixture lost workspace rows: %+v", first.Workspaces)
	}
	if first.WorkspaceNavigation == nil {
		t.Fatal("fixture lost the navigation dialog")
	}
	if len(first.TrajectoryRequests) == 0 || len(first.TrajectoryRequests[0].Events) == 0 {
		t.Fatal("fixture lost trajectory requests")
	}
	if len(first.Messages) == 0 {
		t.Fatal("fixture lost messages")
	}

	first.Workspaces[0].SessionIDs[0] = "mutated"
	first.WorkspaceLayout.Collapsed["one"] = false
	first.WorkspaceLayout.SessionOrders["one"][0] = "mutated"
	first.WorkspaceNavigation.Draft = "mutated"
	first.WorkspaceNavigation.Mode = "mutated"
	first.Messages[0].Body = "mutated"
	first.TrajectoryRequests[0].Status = "mutated"
	first.TrajectoryRequests[0].Events[0] = json.RawMessage(`{"type":"mutated"}`)

	buildsBefore, hitsBefore := s.SnapshotStats()
	second := s.Snapshot()
	buildsAfter, hitsAfter := s.SnapshotStats()
	if buildsAfter != buildsBefore {
		t.Fatalf("caller mutation forced a rebuild: builds %d -> %d", buildsBefore, buildsAfter)
	}
	if hitsAfter != hitsBefore+1 {
		t.Fatalf("second snapshot was not a cache hit: hits %d -> %d", hitsBefore, hitsAfter)
	}
	if got := second.Workspaces[0].SessionIDs[0]; got != "a" {
		t.Fatalf("cached membership aliased the caller: %q", got)
	}
	if !second.WorkspaceLayout.Collapsed["one"] {
		t.Fatal("cached collapse state aliased the caller")
	}
	if got := second.WorkspaceLayout.SessionOrders["one"][0]; got != "a" {
		t.Fatalf("cached session order aliased the caller: %q", got)
	}
	if second.WorkspaceNavigation.Draft != "" || second.WorkspaceNavigation.Mode != "pick" {
		t.Fatalf("cached navigation view aliased the caller: %+v", second.WorkspaceNavigation)
	}
	if second.Messages[0].Body == "mutated" {
		t.Fatal("cached messages aliased the caller")
	}
	if second.TrajectoryRequests[0].Status == "mutated" {
		t.Fatal("cached trajectory request aliased the caller")
	}
	if string(second.TrajectoryRequests[0].Events[0]) == `{"type":"mutated"}` {
		t.Fatal("cached trajectory events aliased the caller")
	}
}

// TestSnapshotCacheInvalidatesOnFire proves the other half of the contract: an
// observable change advances the revision, so the next Snapshot rebuilds with
// the new content and only then becomes cacheable again.
func TestSnapshotCacheInvalidatesOnFire(t *testing.T) {
	s := snapshotCacheFixture(t)
	first := s.Snapshot()
	builds1, hits1 := s.SnapshotStats()
	if builds1 != 1 || hits1 != 0 {
		t.Fatalf("unexpected first snapshot stats: builds=%d hits=%d", builds1, hits1)
	}
	second := s.Snapshot()
	builds2, hits2 := s.SnapshotStats()
	if builds2 != builds1 || hits2 != hits1+1 {
		t.Fatalf("repeat snapshot was not a hit: builds %d->%d hits %d->%d", builds1, builds2, hits1, hits2)
	}
	if len(second.Messages) != len(first.Messages) {
		t.Fatalf("cached messages changed without a fire: %d -> %d", len(first.Messages), len(second.Messages))
	}

	s.mu.Lock()
	s.messages = append(s.messages, Message{ID: "cache-extra", Role: "assistant", Seq: 31, Raw: json.RawMessage(`{"type":"step/start","seq":31,"data":{"turn":3,"step":1}}`), Body: "extra"})
	s.lastAction = "cache invalidated"
	s.mu.Unlock()
	s.fire()

	third := s.Snapshot()
	builds3, hits3 := s.SnapshotStats()
	if builds3 != builds2+1 {
		t.Fatalf("fire did not invalidate the snapshot cache: builds %d -> %d", builds2, builds3)
	}
	if hits3 != hits2 {
		t.Fatalf("stale cache was served after fire: hits %d -> %d", hits2, hits3)
	}
	if third.LastAction != "cache invalidated" {
		t.Fatalf("snapshot did not observe the change: %q", third.LastAction)
	}
	if len(third.Messages) <= len(second.Messages) {
		t.Fatalf("appended message missing from the rebuilt snapshot: %d -> %d", len(second.Messages), len(third.Messages))
	}

	fourth := s.Snapshot()
	builds4, hits4 := s.SnapshotStats()
	if builds4 != builds3 || hits4 != hits3+1 {
		t.Fatalf("rebuilt snapshot was not cached: builds %d->%d hits %d->%d", builds3, builds4, hits3, hits4)
	}
	if fourth.LastAction != third.LastAction || len(fourth.Messages) != len(third.Messages) {
		t.Fatal("cached snapshot does not match the rebuilt one")
	}
}

// TestSnapshotCacheObservesDraftEdits pins the reason the mutation funnels bump
// the revision themselves: a dialog draft is observable in ViewState but is
// deliberately published without a notification.
func TestSnapshotCacheObservesDraftEdits(t *testing.T) {
	s := snapshotCacheFixture(t)
	navigation := s.Snapshot().WorkspaceNavigation
	if navigation == nil || navigation.Token == 0 {
		t.Fatalf("fixture lost the navigation dialog: %+v", navigation)
	}
	s.SetWorkspaceNavigationDraft(navigation.Token, "typed draft")
	got := s.Snapshot().WorkspaceNavigation
	if got == nil || got.Draft != "typed draft" {
		t.Fatalf("snapshot served a stale draft: %+v", got)
	}
}

func TestSnapshotBytesIndependentAcrossFramesAndSource(t *testing.T) {
	s := benchmarkStore(30)
	frame := s.BeginFrame()
	defer frame.EndFrame()
	first := s.Snapshot() // A cache miss while a frame is open.
	before := s.Snapshot()
	view := frame.Snapshot()
	if len(first.Messages) == 0 || len(first.Messages[0].Raw) == 0 || len(first.TrajectoryRequests) == 0 || len(first.TrajectoryRequests[0].Events) == 0 {
		t.Fatal("fixture must contain message and trajectory event bytes")
	}
	originalRaw := string(before.Messages[0].Raw)
	originalEvent := string(before.TrajectoryRequests[0].Events[0])
	first.Messages[0].Raw[0] ^= 1
	first.TrajectoryRequests[0].Events[0][0] ^= 1
	after := s.Snapshot()
	if string(before.Messages[0].Raw) != originalRaw || string(after.Messages[0].Raw) != originalRaw || string(view.Messages[0].Raw) != originalRaw {
		t.Fatal("in-place message mutation contaminated another snapshot or frame view")
	}
	if string(before.TrajectoryRequests[0].Events[0]) != originalEvent || string(after.TrajectoryRequests[0].Events[0]) != originalEvent || string(view.TrajectoryRequests[0].Events[0]) != originalEvent {
		t.Fatal("in-place trajectory mutation contaminated another snapshot or frame view")
	}
	s.mu.Lock()
	sourceRaw := string(s.messages[0].Raw)
	sourceBoundary := string(s.trajectoryBoundaries[1])
	s.mu.Unlock()
	if sourceRaw != originalRaw || sourceBoundary != originalEvent {
		t.Fatalf("snapshot mutation contaminated source bytes: message=%q boundary=%q", sourceRaw, sourceBoundary)
	}
}

func TestSnapshotCloneNestedJSONBytes(t *testing.T) {
	raw := json.RawMessage(`{"raw":true}`)
	bytes := []byte("payload")
	source := ViewState{TrajectoryRequestHeaders: []TrajectoryRequestHeader{{Header: RequestInspectionHeader{
		Tools: []events.RequestHeaderDataHeaderToolsItem{{Parameters: map[string]any{"nested": []any{raw, bytes}}}},
	}}}}
	first := cloneViewState(source)
	before := cloneViewState(source)
	values := first.TrajectoryRequestHeaders[0].Header.Tools[0].Parameters["nested"].([]any)
	values[0].(json.RawMessage)[0] ^= 1
	values[1].([]byte)[0] ^= 1
	after := cloneViewState(source)
	for _, v := range []ViewState{source, before, after} {
		values := v.TrajectoryRequestHeaders[0].Header.Tools[0].Parameters["nested"].([]any)
		if string(values[0].(json.RawMessage)) != `{"raw":true}` || string(values[1].([]byte)) != "payload" {
			t.Fatal("nested byte slices alias another view or source")
		}
	}
}

func TestSnapshotCloneViewMatchesIndependentDeepCopy(t *testing.T) {
	s := snapshotCacheFixture(t)
	in := s.Snapshot()
	usage := 3.0
	effort := "high"
	in.TurnUsage = map[float64]TurnUsage{1: {CacheRead: &usage, Routes: []string{"a", "b"}}}
	in.ModelSelection = &api.ModelCatalogDefault{ReasoningEffort: &effort}
	in.Messages = append(in.Messages, Message{
		Raw: json.RawMessage(`{"type":"test"}`), ContentBlock: json.RawMessage(`{"type":"text"}`),
		Files: []string{"first"}, Tool: &ToolCallView{Children: []*ToolCallView{{Events: []Message{{Raw: json.RawMessage(`{"type":"nested"}`)}}}}},
	})
	in.WorkflowRuns = []workflowrun.Run{{Events: []json.RawMessage{json.RawMessage(`{"type":"workflow"}`)}}}
	clone := cloneViewState(in)
	if !reflect.DeepEqual(clone, in) {
		t.Fatal("clone did not preserve the view")
	}
	assertDeepIndependent(t, reflect.ValueOf(in), reflect.ValueOf(clone), "view")
}

// Walk exported and private value fields. Immutable strings and revision
// tokens (identities with no writable content) may be shared, but no mutable
// pointer, map or slice in a snapshot may be.
func assertDeepIndependent(t *testing.T, a, b reflect.Value, path string) {
	t.Helper()
	if !a.IsValid() || a.Type() == reflect.TypeOf((*projectionToken)(nil)) {
		return
	}
	switch a.Kind() {
	case reflect.Interface:
		if !a.IsNil() {
			assertDeepIndependent(t, a.Elem(), b.Elem(), path)
		}
	case reflect.Pointer:
		if !a.IsNil() {
			if a.Pointer() == b.Pointer() {
				t.Fatalf("shared pointer at %s", path)
			}
			assertDeepIndependent(t, a.Elem(), b.Elem(), path+".*")
		}
	case reflect.Slice:
		if a.Len() > 0 && a.Pointer() == b.Pointer() {
			t.Fatalf("shared slice at %s", path)
		}
		for i := 0; i < a.Len(); i++ {
			assertDeepIndependent(t, a.Index(i), b.Index(i), fmt.Sprintf("%s[%d]", path, i))
		}
	case reflect.Map:
		if !a.IsNil() {
			if a.Pointer() == b.Pointer() {
				t.Fatalf("shared map at %s", path)
			}
			iter := a.MapRange()
			for iter.Next() {
				key := iter.Key()
				assertDeepIndependent(t, iter.Value(), b.MapIndex(key), path+"[key]")
			}
		}
	case reflect.Struct:
		for i := 0; i < a.NumField(); i++ {
			assertDeepIndependent(t, a.Field(i), b.Field(i), path+"."+a.Type().Field(i).Name)
		}
	case reflect.Array:
		for i := 0; i < a.Len(); i++ {
			assertDeepIndependent(t, a.Index(i), b.Index(i), fmt.Sprintf("%s[%d]", path, i))
		}
	}
}

func TestSnapshotSetLastActionInvalidates(t *testing.T) {
	s := New()
	frame := s.BeginFrame()
	defer frame.EndFrame()
	_ = s.Snapshot()
	_ = frame.Snapshot()
	called := false
	s.notify = func() { called = true }
	s.SetLastAction("  saved  ")
	if !called {
		t.Fatal("SetLastAction did not notify")
	}
	if got := s.Snapshot().LastAction; got != "saved" {
		t.Fatalf("Snapshot LastAction = %q", got)
	}
	if got := frame.Snapshot().LastAction; got != "saved" {
		t.Fatalf("frame LastAction = %q", got)
	}
}

func TestFrameSnapshotTracksRevisionAndConcurrentSnapshots(t *testing.T) {
	s := benchmarkStore(30)
	frame := s.BeginFrame()
	defer frame.EndFrame()
	first := frame.Snapshot()
	originalRaw := string(first.Messages[0].Raw)
	again := frame.Snapshot()
	if &first.Messages[0] != &again.Messages[0] {
		t.Fatal("unchanged frame did not reuse its read-only view")
	}
	independent := s.Snapshot()
	independent.Messages[0].Raw[0] ^= 1
	if string(first.Messages[0].Raw) != originalRaw || string(again.Messages[0].Raw) != originalRaw {
		t.Fatal("ordinary Snapshot aliases a frame view")
	}

	const readers = 8
	start := make(chan struct{})
	changed := make(chan struct{})
	errs := make(chan error, readers)
	var ready sync.WaitGroup
	ready.Add(readers)
	for i := 0; i < readers; i++ {
		go func() {
			ready.Done()
			<-start
			for j := 0; j < 10; j++ {
				_ = s.Snapshot()
			}
			<-changed
			got := s.Snapshot()
			if got.SelectedID != "next" || len(got.Messages) != 0 {
				errs <- fmt.Errorf("concurrent Snapshot selected=%q messages=%d", got.SelectedID, len(got.Messages))
				return
			}
			errs <- nil
		}()
	}
	ready.Wait()
	close(start)
	done := make(chan struct{})
	go func() {
		s.Select("next")
		close(changed)
		close(done)
	}()
	<-done
	for i := 0; i < readers; i++ {
		if err := <-errs; err != nil {
			t.Error(err)
		}
	}
	current := frame.Snapshot()
	if current.SelectedID != "next" || len(current.Messages) != 0 {
		t.Fatalf("frame reused an earlier revision: selected=%q messages=%d", current.SelectedID, len(current.Messages))
	}
	if first.SelectedID != "bench" || len(first.Messages) == 0 {
		t.Fatal("updating the frame mutated an earlier view")
	}

	s.SetLastAction("in-frame")
	if got := frame.Snapshot().LastAction; got != "in-frame" {
		t.Fatalf("frame missed same-pass change: %q", got)
	}
}

// Replacing a retained boundary without changing the journal length used to
// collide with the history fingerprint and return the previous event bytes.
func TestSnapshotBoundaryReplacementInvalidatesParsedHistory(t *testing.T) {
	s := New()
	start := json.RawMessage(`{"type":"step/start","seq":1,"data":{"turn":1,"step":1}}`)
	first := json.RawMessage(`{"type":"step/end","seq":2,"data":{"turn":1,"step":1,"marker":"a"}}`)
	second := json.RawMessage(`{"type":"step/end","seq":2,"data":{"turn":1,"step":1,"marker":"b"}}`)
	s.mu.Lock()
	s.captureTrajectoryBoundaryLocked(start)
	s.captureTrajectoryBoundaryLocked(first)
	s.mu.Unlock()
	old := s.Snapshot()
	if len(old.TrajectoryRequests) != 1 || len(old.TrajectoryRequests[0].Events) != 2 {
		t.Fatalf("unexpected trajectory fixture: %+v", old.TrajectoryRequests)
	}
	s.mu.Lock()
	s.captureTrajectoryBoundaryLocked(second)
	s.mu.Unlock()
	current := s.Snapshot()
	if len(current.TrajectoryRequests) != 1 || len(current.TrajectoryRequests[0].Events) != 2 {
		t.Fatalf("replacement lost trajectory events: %+v", current.TrajectoryRequests)
	}
	if string(current.TrajectoryRequests[0].Events[1]) != string(second) {
		t.Fatalf("stale boundary after replacement: %s", current.TrajectoryRequests[0].Events[1])
	}
	if string(old.TrajectoryRequests[0].Events[1]) != string(first) {
		t.Fatal("updating a boundary changed an earlier snapshot")
	}
}

func TestOverlappingFramesRemainIndependent(t *testing.T) {
	s := benchmarkStore(30)
	a, b := s.BeginFrame(), s.BeginFrame()
	defer b.EndFrame()
	av, bv := a.Snapshot(), b.Snapshot()
	if &av.Messages[0] == &bv.Messages[0] {
		t.Fatal("separate frames share mutable backing data")
	}
	a.EndFrame()
	if again := b.Snapshot(); &again.Messages[0] != &bv.Messages[0] {
		t.Fatal("ending another frame invalidated this frame")
	}
}
