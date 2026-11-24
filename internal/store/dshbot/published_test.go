package dshbot

import (
	"errors"
	"reflect"
	"strings"
	"sync"
	"testing"

	"whaleisle/internal/desktop/pluginrpc"
	"whaleisle/internal/errcopy"
	"whaleisle/internal/state"
	"whaleisle/internal/store/storelocale"
)

func describeResult() pluginrpc.DshbotDescribeResult {
	return pluginrpc.DshbotDescribeResult{
		View: pluginrpc.DshbotSettingsView{
			Value: pluginrpc.DshbotCatalogValue{
				Items: []pluginrpc.DshbotCatalogItem{
					{ID: "b1", Kind: "bot", Name: "Alpha", SessionID: "s1"},
				},
			},
		},
	}
}

func TestRosterSnapshotSharedWhenUnchanged(t *testing.T) {
	s, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	a := s.Snapshot()
	if b := s.Snapshot(); a != b {
		t.Fatal("unchanged Snapshot must return the same published version")
	}
	allocs := testing.AllocsPerRun(100, func() { _ = s.Snapshot() })
	if allocs != 0 {
		t.Fatalf("unchanged Snapshot allocs = %v", allocs)
	}
}

func TestRosterWritePublishesNewRevision(t *testing.T) {
	s, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	before := s.Snapshot()
	s.SetFromDescribe(describeResult(), false)
	after := s.Snapshot()
	if after == before || after.Rev != before.Rev+1 {
		t.Fatalf("rev %d -> %d", before.Rev, after.Rev)
	}
	if len(after.Value.Rows) != 1 {
		t.Fatalf("rows = %d", len(after.Value.Rows))
	}
	// Mutating the earlier version must not corrupt stored state.
	before.Value.Rows = append(before.Value.Rows, Row{ID: "x"})
	if len(s.Snapshot().Value.Rows) != 1 {
		t.Fatal("mutating a published version leaked into the store")
	}
}

func TestRosterConcurrentReadWrite(t *testing.T) {
	s, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				s.SetFromDescribe(describeResult(), false)
			}
		}()
	}
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 200; j++ {
				_ = s.Snapshot().Value.Rows
			}
		}()
	}
	wg.Wait()
	if len(s.Snapshot().Value.Rows) != 1 {
		t.Fatal("roster corrupted under concurrency")
	}
}

// Every migrated controller must share one published version for unchanged
// reads (zero allocations) and publish a new version after a write.
func TestPublishedControllersShareVersions(t *testing.T) {
	rs := RoutineScope{HostKey: "host", BotID: "bot"}
	ts := TaskScope{HostKey: "host", BotID: "bot"}
	revOf := func(v any) uint64 {
		return reflect.ValueOf(v).Elem().FieldByName("Rev").Uint()
	}
	type pubCase struct {
		name  string
		snap  func() any
		noop  func()
		write func()
	}
	cases := []pubCase{}
	add := func(name string, snap func() any, write func()) {
		cases = append(cases, pubCase{name: name, snap: snap, write: write})
	}
	addWithNoop := func(name string, snap func() any, noop, write func()) {
		cases = append(cases, pubCase{name: name, snap: snap, noop: noop, write: write})
	}
	{
		var c AvatarControls
		c.Bind(rs)
		add("AvatarControls", func() any { return c.Snapshot(rs) }, func() { c.Dismiss(rs, nil) })
	}
	{
		var c BotCreator
		c.Bind("account", "client")
		add("BotCreator", func() any { return c.Snapshot("account") }, func() { c.Bind("account", "client2") })
	}
	{
		var c BotDeleter
		c.Bind("account", "client")
		add("BotDeleter", func() any { return c.Snapshot("account") }, func() { c.Bind("account", "client2") })
	}
	{
		var c BotDuplicator
		c.Bind("account", "client")
		add("BotDuplicator", func() any { return c.Snapshot("account") }, func() { c.Bind("account", "client2") })
	}
	{
		var c BotVisibilityManager
		c.Bind("account", "client")
		add("BotVisibilityManager", func() any { return c.Snapshot("account") }, func() { c.Bind("account", "client2") })
	}
	{
		var c CapabilityControls
		c.Bind(rs)
		addWithNoop("CapabilityControls", func() any { return c.Snapshot(rs) }, func() { c.Bind(rs) }, func() { c.BeginLoad(rs) })
	}
	{
		var c ContactControls
		c.Bind("account", "client")
		add("ContactControls", func() any { return c.Snapshot("account") }, func() { c.Bind("account", "client2") })
	}
	{
		var c MemoryControls
		c.Bind(rs)
		addWithNoop("MemoryControls", func() any { return c.Snapshot(rs) }, func() { c.Bind(rs) }, func() { c.BeginLoad(rs) })
	}
	{
		var c MemoryReviewControls
		c.Bind(rs)
		addWithNoop("MemoryReviewControls", func() any { return c.Snapshot(rs) }, func() { c.Bind(rs) }, func() { c.BeginLoad(rs) })
	}
	{
		var c NotifyControls
		c.Bind(rs)
		add("NotifyControls", func() any { return c.Snapshot(rs) }, func() { c.Dismiss(rs, nil) })
	}
	{
		var c ObjectiveControls
		c.Bind(rs)
		addWithNoop("ObjectiveControls", func() any { return c.Snapshot(rs) }, func() { c.Bind(rs) }, func() { c.BeginLoad(rs) })
	}
	{
		var c ObjectiveConfigControls
		c.Bind(rs)
		addWithNoop("ObjectiveConfigControls", func() any { return c.Snapshot(rs) }, func() { c.Bind(rs) }, func() { c.BeginLoad(rs) })
	}
	{
		var c RoomCreator
		c.Bind("account", "client")
		add("RoomCreator", func() any { return c.Snapshot("account") }, func() { c.Bind("account", "client2") })
	}
	{
		var c RoomEditor
		c.Bind("account", "client")
		add("RoomEditor", func() any { return c.Snapshot("account") }, func() { c.Bind("account", "client2") })
	}
	{
		var c RoomRuntimeReader
		c.Bind("scope")
		addWithNoop("RoomRuntimeReader", func() any { return c.Snapshot() }, func() { c.Bind("scope") }, func() { c.BeginRead("scope", 1) })
	}
	{
		var c RouteControls
		c.Bind(rs)
		add("RouteControls", func() any { return c.Snapshot(rs) }, func() { c.Dismiss(rs, nil) })
	}
	{
		var c RoutineControls
		c.Bind(rs)
		addWithNoop("RoutineControls", func() any { return c.Snapshot(rs) }, func() { c.Bind(rs) }, func() { c.BeginLoad(rs) })
	}
	{
		var c RoutineCreator
		c.Sync(rs, 1, RoutineSnapshot{Scope: rs})
		add("RoutineCreator", func() any { return c.Snapshot(rs) }, func() { c.Dismiss(rs) })
	}
	{
		var c SenderPolicyControls
		c.Bind(rs)
		add("SenderPolicyControls", func() any { return c.Snapshot(rs) }, func() { c.Dismiss(rs, nil) })
	}
	{
		var c TaskControls
		c.Bind(ts)
		addWithNoop("TaskControls", func() any { return c.Snapshot(ts) }, func() { c.Bind(ts) }, func() { c.BeginLoad(ts) })
	}
	{
		var c TextProfileControls
		c.Bind(rs)
		add("TextProfileControls", func() any { return c.Snapshot(rs) }, func() { c.Dismiss(rs, nil) })
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			before := tc.snap()
			if again := tc.snap(); again != before {
				t.Fatal("unchanged Snapshot must return the shared published version")
			}
			if tc.noop != nil {
				tc.noop()
				if afterNoop := tc.snap(); afterNoop != before {
					t.Fatal("no-op Bind must not invalidate published version")
				}
			}
			if allocs := testing.AllocsPerRun(50, func() { _ = tc.snap() }); allocs != 0 {
				t.Fatalf("unchanged Snapshot allocs = %v", allocs)
			}
			tc.write()
			after := tc.snap()
			if after == before {
				t.Fatal("write must publish a new version")
			}
			if revOf(after) != revOf(before)+1 {
				t.Fatalf("rev %d -> %d", revOf(before), revOf(after))
			}
		})
	}
}

func TestForeignZeroScopeSnapshotNeverReturnsNil(t *testing.T) {
	rs := RoutineScope{HostKey: "host", BotID: "bot"}
	ts := TaskScope{HostKey: "host", BotID: "bot"}
	assertZeroRoutine := func(t *testing.T, name string, snap func(RoutineScope) (any, RoutineScope)) {
		t.Helper()
		t.Run(name, func(t *testing.T) {
			v1, s1 := snap(RoutineScope{})
			if v1 == nil {
				t.Fatal("zero-scope Snapshot returned nil")
			}
			if s1 != (RoutineScope{}) {
				t.Fatalf("Scope = %+v, want zero", s1)
			}
			v2, _ := snap(RoutineScope{})
			if v2 != v1 {
				t.Fatal("repeated zero-scope Snapshot must share published version")
			}
			if allocs := testing.AllocsPerRun(50, func() { _, _ = snap(RoutineScope{}) }); allocs != 0 {
				t.Fatalf("repeated zero-scope Snapshot allocs = %v", allocs)
			}
		})
	}
	{
		var c CapabilityControls
		c.Bind(rs)
		assertZeroRoutine(t, "CapabilityControls", func(k RoutineScope) (any, RoutineScope) {
			v := c.Snapshot(k)
			if v == nil {
				return nil, RoutineScope{}
			}
			return v, v.Value.Scope
		})
	}
	{
		var c MemoryControls
		c.Bind(rs)
		assertZeroRoutine(t, "MemoryControls", func(k RoutineScope) (any, RoutineScope) {
			v := c.Snapshot(k)
			if v == nil {
				return nil, RoutineScope{}
			}
			return v, v.Value.Scope
		})
	}
	{
		var c MemoryReviewControls
		c.Bind(rs)
		assertZeroRoutine(t, "MemoryReviewControls", func(k RoutineScope) (any, RoutineScope) {
			v := c.Snapshot(k)
			if v == nil {
				return nil, RoutineScope{}
			}
			return v, v.Value.Scope
		})
	}
	{
		var c ObjectiveControls
		c.Bind(rs)
		assertZeroRoutine(t, "ObjectiveControls", func(k RoutineScope) (any, RoutineScope) {
			v := c.Snapshot(k)
			if v == nil {
				return nil, RoutineScope{}
			}
			return v, v.Value.Scope
		})
	}
	{
		var c ObjectiveConfigControls
		c.Bind(rs)
		assertZeroRoutine(t, "ObjectiveConfigControls", func(k RoutineScope) (any, RoutineScope) {
			v := c.Snapshot(k)
			if v == nil {
				return nil, RoutineScope{}
			}
			return v, v.Value.Scope
		})
	}
	{
		var c RoutineControls
		c.Bind(rs)
		assertZeroRoutine(t, "RoutineControls", func(k RoutineScope) (any, RoutineScope) {
			v := c.Snapshot(k)
			if v == nil {
				return nil, RoutineScope{}
			}
			return v, v.Value.Scope
		})
	}
	t.Run("TaskControls", func(t *testing.T) {
		var c TaskControls
		c.Bind(ts)
		v1 := c.Snapshot(TaskScope{})
		if v1 == nil {
			t.Fatal("zero-scope TaskControls Snapshot returned nil")
		}
		if v1.Value.Scope != (TaskScope{}) {
			t.Fatalf("Scope = %+v, want zero", v1.Value.Scope)
		}
		if v2 := c.Snapshot(TaskScope{}); v2 != v1 {
			t.Fatal("repeated zero-scope TaskControls Snapshot must share published version")
		}
		if allocs := testing.AllocsPerRun(50, func() { _ = c.Snapshot(TaskScope{}) }); allocs != 0 {
			t.Fatalf("repeated zero-scope TaskControls Snapshot allocs = %v", allocs)
		}
	})
}

func TestAuthoredValidationCopyReachesView(t *testing.T) {
	checkAuthored := func(t *testing.T, f *state.Fault, want error) {
		t.Helper()
		if want == nil {
			t.Fatal("fixture did not produce a validation error")
		}
		if f == nil {
			t.Fatal("snapshot carries no fault")
		}
		if got := f.SentenceText(); got != want.Error() {
			t.Fatalf("view sentence = %q (Cause=%v Text=%q), want %q", got, f.Cause, f.Text, want.Error())
		}
	}

	t.Run("TextProfilePrepare", func(t *testing.T) {
		var parent RoutineControls
		k := RoutineScope{HostKey: "fixture-account", BotID: "b"}
		parent.Bind(k)
		c := &parent.TextProfile
		load, _ := c.BeginLoad(k)
		c.FinishLoad(load, textProfileStoreFixture(), true, nil)
		d := pluginrpc.DshbotTextProfileValues{Name: "   ", Title: "t", Description: "d"}
		_, want := pluginrpc.ParseDshbotTextProfile(d)
		c.SetDraft(k, d)
		c.Prepare(k)
		checkAuthored(t, c.Snapshot(k).Value.Error, want)
	})

	t.Run("ObjectiveConfigPrepare", func(t *testing.T) {
		c, k := objectiveConfigLoaded(t)
		d := pluginrpc.DshbotObjectiveConfigDraft{Objective: "Changed objective", MaxRounds: "0"}
		_, want := pluginrpc.ParseDshbotObjectiveConfigDraft(d)
		c.SetDraft(k, d)
		c.Prepare(k)
		checkAuthored(t, c.Snapshot(k).Value.Error, want)
	})

	t.Run("RoutineAdvancedContextEdit", func(t *testing.T) {
		c, k := advancedReady(t, false)
		c.Edit(k, "contextToggle", "foreign")
		checkAuthored(t, c.Snapshot(k).Value.Error, storelocale.Error(storelocale.KeyStoreDshbotContextSourceIsNotOwnedByThisBot))
	})

	t.Run("RoutineControlsPrepare", func(t *testing.T) {
		c, k := advancedReady(t, false)
		c.Edit(k, "schedule", "")
		c.Prepare(k, "save")
		snap := c.Snapshot(k).Value
		want := pluginrpc.ValidateDshbotRoutineEdit(snap.Draft.Edit)
		checkAuthored(t, snap.Error, want)
	})

	t.Run("RoutineCreateSchedulePrepare", func(t *testing.T) {
		var c RoutineCreator
		k := RoutineScope{HostKey: "account", BotID: "b"}
		c.Sync(k, 1, createSource(4))
		c.Start(k, "Asia/Shanghai")
		c.Change(k, "prompt", "Fixture prompt")
		c.Change(k, "frequency", "weekly")
		c.Change(k, "weekday", "7")
		_, want := routineCreateSchedule(*c.entries[k].state.Draft)
		c.Prepare(k)
		checkAuthored(t, c.Snapshot(k).Value.Error, want)
	})

	t.Run("RoomEditorPrepare", func(t *testing.T) {
		c := &RoomEditor{}
		k := "account-room"
		c.Bind(k, "client-1")
		load, _ := c.BeginLoad(k)
		c.FinishLoad(load, roomEditStoreStatus(), true, nil)
		c.Start(k)
		c.Change(k, "name", "   ")
		_, want := pluginrpc.ParseDshbotRoomText(pluginrpc.DshbotRoomTextValues{Name: "   ", Description: "preserve full description"})
		c.Prepare(k)
		checkAuthored(t, c.Snapshot(k).Value.Error, want)
	})

	t.Run("RoomCreatorPrepare", func(t *testing.T) {
		c, k := roomStoreReady(t)
		c.ToggleMember(k, "a")
		c.Prepare(k)
		snap := c.Snapshot(k).Value
		cmd := pluginrpc.DshbotRoomCreateCommand{
			ID: snap.Draft.ID, SessionID: snap.Draft.SessionID,
			Expected: snap.Catalog,
			Draft: pluginrpc.DshbotRoomDraft{
				Name: strings.TrimSpace(snap.Draft.Name), Description: strings.TrimSpace(snap.Draft.Description),
				WorkspaceID: strings.TrimSpace(snap.Draft.WorkspaceID), ScratchCwd: strings.TrimSpace(snap.Draft.ScratchCwd),
				MemberBotIDs: snap.Draft.MemberBotIDs, MaxRounds: 3, MaxSpeaks: 10,
			},
		}
		checkAuthored(t, snap.Error, pluginrpc.ValidateDshbotRoomCreate(cmd))
	})

	t.Run("MemoryControlsPrepare", func(t *testing.T) {
		c, k := memoryLoaded(t)
		c.StartEdit(k, 0)
		huge := strings.Repeat("a", 8001)
		c.Change(k, huge)
		c.Prepare(k)
		snap := c.Snapshot(k).Value
		cmd := pluginrpc.DshbotMemoryCommand{
			BotID: k.BotID, SessionID: snap.SessionID, Track: snap.Draft.Track,
			MemoryRevision: snap.Draft.BaseRevision, Text: pluginrpc.DshbotMemoryText([]string{huge}),
			Limit: snap.Server.Limit(snap.Draft.Track),
		}
		checkAuthored(t, snap.Error, pluginrpc.ValidateDshbotMemoryCommand(cmd))
	})

	t.Run("UnknownRawErrorDoesNotLeak", func(t *testing.T) {
		var c RoutineControls
		k := RoutineScope{HostKey: "fixture-host", BotID: "b"}
		c.Bind(k)
		ticket, _ := c.BeginLoad(k)
		rawErr := errors.New("internal backend stack trace detail")
		c.FinishLoad(ticket, nil, nil, nil, rawErr)
		f := c.Snapshot(k).Value.Error
		if f == nil || f.Text != "" {
			t.Fatalf("raw backend error unexpectedly populated Fault.Text: %#v", f)
		}
		got := f.SentenceText()
		if got != errcopy.Sentence(rawErr) || strings.Contains(got, "stack trace") {
			t.Fatalf("unexpected sentence %q", got)
		}
	})
}

func TestRoutineCreatorUnchangedSyncKeepsVersion(t *testing.T) {
	t.Run("with active draft", func(t *testing.T) {
		var c RoutineCreator
		k := RoutineScope{HostKey: "account", BotID: "b"}
		src := createSource(4)
		c.Sync(k, 1, src)
		c.Start(k, "Asia/Shanghai")
		before := c.Snapshot(k)
		for range 5 {
			c.Sync(k, 1, src)
			if got := c.Snapshot(k); got != before {
				t.Fatalf("unchanged Sync republished: %p rev %d -> %p rev %d", before, before.Rev, got, got.Rev)
			}
		}
		if allocs := testing.AllocsPerRun(100, func() {
			c.Sync(k, 1, src)
			_ = c.Snapshot(k)
		}); allocs != 0 {
			t.Fatalf("unchanged draft Sync+Snapshot allocs = %v", allocs)
		}
	})

	t.Run("with pending attempt candidate and retry-verified steady state", func(t *testing.T) {
		var c RoutineCreator
		k := RoutineScope{HostKey: "account", BotID: "b"}
		createReady(t, &c, k)
		ticket, cmd, ok := c.Begin(k, c.Snapshot(k).Value.Review)
		if !ok || !c.MarkSent(ticket) {
			t.Fatal("begin failed")
		}
		c.Finish(ticket, pluginrpc.DshbotRoutineCreateResult{}, errors.New("lost"))

		srcCand := createSource(5)
		srcCand.Routines = []pluginrpc.DshbotRoutine{{
			ID: "new", BotID: "b", Name: cmd.Edit.Name, Prompt: cmd.Edit.Prompt,
			Schedule: cmd.Edit.Schedule, Timezone: cmd.Edit.Timezone,
		}}
		c.Sync(k, 1, srcCand)
		before := c.Snapshot(k)
		if len(before.Value.Candidates) != 1 {
			t.Fatalf("candidates = %d, want 1", len(before.Value.Candidates))
		}
		for range 3 {
			c.Sync(k, 1, srcCand)
			if got := c.Snapshot(k); got != before {
				t.Fatalf("unchanged candidate Sync republished: rev %d -> %d", before.Rev, got.Rev)
			}
		}
		if allocs := testing.AllocsPerRun(100, func() {
			c.Sync(k, 1, srcCand)
			_ = c.Snapshot(k)
		}); allocs != 0 {
			t.Fatalf("unchanged candidate Sync+Snapshot allocs = %v", allocs)
		}

		srcRetry := createSource(6)
		c.Sync(k, 1, srcRetry)
		beforeRetry := c.Snapshot(k)
		if !beforeRetry.Value.RetryVerified {
			t.Fatal("expected RetryVerified")
		}
		if allocs := testing.AllocsPerRun(100, func() {
			c.Sync(k, 1, srcRetry)
			_ = c.Snapshot(k)
		}); allocs != 0 {
			t.Fatalf("unchanged retry-verified Sync+Snapshot allocs = %v", allocs)
		}
	})
}
