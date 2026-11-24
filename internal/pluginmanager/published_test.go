package pluginmanager

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestManagerSnapshotSharedUntilWrite(t *testing.T) {
	c := New()
	c.Attach(basicFixture(nil), nil)
	defer c.Detach()
	if err := c.Load(context.Background()); err != nil {
		t.Fatal(err)
	}
	first := c.Snapshot()
	if first != c.Snapshot() {
		t.Fatal("unchanged state must hand out the same *Version")
	}
	if got := testing.AllocsPerRun(100, func() { c.Snapshot() }); got != 0 {
		t.Fatalf("Snapshot allocates %g per run", got)
	}
	// Read-only critical sections must not mint a revision.
	c.InstalledPlugin("fixture-plugin")
	c.fire()
	c.Ensure()
	if c.Snapshot() != first {
		t.Fatal("a section that changed nothing published a new version")
	}
}

func TestManagerWritePublishesAndSignals(t *testing.T) {
	c := New()
	c.Attach(basicFixture(nil), nil)
	defer c.Detach()
	before := c.Snapshot()
	changed := c.Changed()
	c.Actions(before.Value.Owner).FilterRows("fixture")
	after := c.Snapshot()
	if after == before || after.Rev != before.Rev+1 || after.Value.RowFilter != "fixture" {
		t.Fatalf("FilterRows did not publish Rev+1: %d -> %d", before.Rev, after.Rev)
	}
	select {
	case <-changed:
	default:
		t.Fatal("Changed channel did not close on publish")
	}
}

// In-place edits of the working state (BusyKeys, install runs) must never
// reach a version a reader already holds.
func TestManagerPublishedValueIsolated(t *testing.T) {
	c := New()
	c.Attach(basicFixture(nil), nil)
	defer c.Detach()
	if err := c.Load(context.Background()); err != nil {
		t.Fatal(err)
	}
	c.mu.Lock()
	c.state.BusyKeys = map[string]bool{"one": true}
	c.state.Install.Runs = []InstallRun{{JobID: "a", Chunks: []InstallOutput{{Text: "x"}}}}
	c.unlockPublish()
	held := c.Snapshot()
	c.mu.Lock()
	c.state.BusyKeys["two"] = true
	c.state.Install.Runs[0].Chunks[0].Text = "y"
	c.state.Packages[0].Parts[0].Phase = "failed"
	c.unlockPublish()
	if held.Value.BusyKeys["two"] || held.Value.Install.Runs[0].Chunks[0].Text != "x" || held.Value.Packages[0].Parts[0].Phase == "failed" {
		t.Fatal("in-place working-state edit leaked into a published version")
	}
	if next := c.Snapshot(); next.Rev != held.Rev+1 || !next.Value.BusyKeys["two"] {
		t.Fatal("edit was not published")
	}
}

func TestManagerStaleOwnerDropped(t *testing.T) {
	c := New()
	c.Attach(basicFixture(nil), nil)
	old := c.Snapshot().Value.Owner
	life := c.life
	c.Attach(basicFixture(nil), nil)
	defer c.Detach()
	if life.Err() == nil {
		t.Fatal("Attach did not cancel the retired owner's lifecycle")
	}
	current := c.Snapshot()
	if current.Value.Owner == old || current.Value.Owner != c.ownerL.Generation() {
		t.Fatalf("owner lease not renewed: %d -> %d", old, current.Value.Owner)
	}
	if c.Actions(old).edit(func(s *State) { s.RowFilter = "stale" }) {
		t.Fatal("an action from the retired owner was applied")
	}
	if c.Snapshot() != current {
		t.Fatal("stale owner edit published")
	}
}

func TestManagerBlockedLoadDoesNotBlockSnapshot(t *testing.T) {
	c := New()
	started, release := make(chan struct{}), make(chan struct{})
	var once sync.Once
	base := basicFixture(nil)
	c.Attach(fixtureCall(func(ctx context.Context, m string, a, out any) error {
		once.Do(func() { close(started) })
		<-release
		return base(ctx, m, a, out)
	}), nil)
	defer c.Detach()
	done := make(chan error, 1)
	go func() { done <- c.Load(context.Background()) }()
	<-started
	read := make(chan bool, 1)
	go func() { read <- c.Snapshot().Value.Loading }()
	select {
	case loading := <-read:
		if !loading {
			t.Fatal("in-flight load not published as Loading")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Snapshot blocked behind the manager RPC")
	}
	close(release)
	if err := <-done; err != nil {
		t.Fatal(err)
	}
}
