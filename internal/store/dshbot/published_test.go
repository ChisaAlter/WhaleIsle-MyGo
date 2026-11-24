package dshbot

import (
	"reflect"
	"sync"
	"testing"

	"whaleisle/internal/desktop/pluginrpc"
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
		write func()
	}
	cases := []pubCase{}
	add := func(name string, snap func() any, write func()) {
		cases = append(cases, pubCase{name, snap, write})
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
		add("CapabilityControls", func() any { return c.Snapshot(rs) }, func() { c.Bind(rs) })
	}
	{
		var c ContactControls
		c.Bind("account", "client")
		add("ContactControls", func() any { return c.Snapshot("account") }, func() { c.Bind("account", "client2") })
	}
	{
		var c MemoryControls
		c.Bind(rs)
		add("MemoryControls", func() any { return c.Snapshot(rs) }, func() { c.Bind(rs) })
	}
	{
		var c MemoryReviewControls
		c.Bind(rs)
		add("MemoryReviewControls", func() any { return c.Snapshot(rs) }, func() { c.Bind(rs) })
	}
	{
		var c NotifyControls
		c.Bind(rs)
		add("NotifyControls", func() any { return c.Snapshot(rs) }, func() { c.Dismiss(rs, nil) })
	}
	{
		var c ObjectiveControls
		c.Bind(rs)
		add("ObjectiveControls", func() any { return c.Snapshot(rs) }, func() { c.Bind(rs) })
	}
	{
		var c ObjectiveConfigControls
		c.Bind(rs)
		add("ObjectiveConfigControls", func() any { return c.Snapshot(rs) }, func() { c.Bind(rs) })
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
		add("RoomRuntimeReader", func() any { return c.Snapshot() }, func() { c.Bind("scope") })
	}
	{
		var c RouteControls
		c.Bind(rs)
		add("RouteControls", func() any { return c.Snapshot(rs) }, func() { c.Dismiss(rs, nil) })
	}
	{
		var c RoutineControls
		c.Bind(rs)
		add("RoutineControls", func() any { return c.Snapshot(rs) }, func() { c.Bind(rs) })
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
		add("TaskControls", func() any { return c.Snapshot(ts) }, func() { c.Bind(ts) })
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
