package plugininventory

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"whaleisle/internal/dsh/api"
	"whaleisle/internal/errcopy"
)

func fixtureLoad() fixtureCaller {
	return fixtureCaller(func(_ context.Context, _ string, _, out any) error {
		return json.Unmarshal([]byte(inventoryFixture), out)
	})
}

func TestInventorySnapshotSharedUntilWrite(t *testing.T) {
	var c Controller
	c.Attach(fixtureLoad())
	c.Refresh(context.Background())
	first := c.Snapshot()
	if first != c.Snapshot() {
		t.Fatal("unchanged state must hand out the same *Version")
	}
	if got := testing.AllocsPerRun(100, func() { c.Snapshot() }); got != 0 {
		t.Fatalf("Snapshot allocates %g per run", got)
	}
	if len(first.Value.Global) == 0 || &first.Value.Global[0] != &c.Snapshot().Value.Global[0] {
		t.Fatal("rows must be shared, not cloned per read")
	}
}

func TestInventoryWritePublishesAndSignals(t *testing.T) {
	var c Controller
	c.Attach(fixtureLoad())
	c.Refresh(context.Background())
	before := c.Snapshot()
	changed := c.Changed()
	c.SetQuery(before.Value.Host, "shared")
	after := c.Snapshot()
	if after == before || after.Rev != before.Rev+1 {
		t.Fatalf("SetQuery did not publish Rev+1: %d -> %d", before.Rev, after.Rev)
	}
	select {
	case <-changed:
	default:
		t.Fatal("Changed channel did not close on publish")
	}
	c.SetQuery(before.Value.Host, "shared")
	if c.Snapshot() != after {
		t.Fatal("a no-op write must not publish")
	}
}

func TestInventoryStaleRefreshDropped(t *testing.T) {
	var c Controller
	started, release, done := make(chan struct{}), make(chan struct{}), make(chan struct{})
	c.Attach(fixtureCaller(func(ctx context.Context, m string, a, out any) error {
		close(started)
		<-release
		return fixtureLoad().Call(ctx, m, a, out)
	}))
	oldHost := c.Snapshot().Value.Host
	go func() { c.Refresh(context.Background()); close(done) }()
	<-started
	// A blocked RPC must not block readers.
	read := make(chan struct{})
	go func() { _ = c.Snapshot().Value.Busy; close(read) }()
	select {
	case <-read:
	case <-time.After(2 * time.Second):
		t.Fatal("Snapshot blocked behind the inventory RPC")
	}
	if !c.Snapshot().Value.Busy {
		t.Fatal("in-flight refresh not published as Busy")
	}
	c.Attach(fixtureLoad())
	newHost := c.Snapshot()
	if newHost.Value.Host == oldHost || newHost.Value.Host != c.hostL.Generation() {
		t.Fatalf("Attach did not renew the Host lease: %d -> %d", oldHost, newHost.Value.Host)
	}
	close(release)
	<-done
	if got := c.Snapshot(); got != newHost || got.Value.Ready {
		t.Fatal("stale Host completion was published")
	}
	c.Accept(oldHost, 1, api.PluginInventorySnapshot{})
	if c.Snapshot() != newHost {
		t.Fatal("Accept from a retired Host was published")
	}
}

func TestInventoryFaultMapping(t *testing.T) {
	var c Controller
	c.Attach(fixtureCaller(func(context.Context, string, any, any) error {
		return errors.New("dial tcp 127.0.0.1:1: connectex: connection refused")
	}))
	c.Refresh(context.Background())
	f := c.Snapshot().Value.Err
	if f == nil || f.Cause != errcopy.CauseNetwork {
		t.Fatalf("Err = %+v", f)
	}
	if text := f.SentenceText(); text == "" || text == "dial tcp 127.0.0.1:1: connectex: connection refused" {
		t.Fatalf("fault renders raw text: %q", text)
	}
}
