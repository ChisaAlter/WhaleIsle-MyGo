package market

import (
	"context"
	"encoding/json"
	"errors"
	"path/filepath"
	"sync"
	"testing"
	"time"

	desktopmarket "whaleisle/internal/desktop/market"
	"whaleisle/internal/dsh"
	"whaleisle/internal/errcopy"
)

// republish publishes state a test wrote straight into the working fields.
func (s *Store) republish() {
	s.mu.Lock()
	s.publishLocked()
	s.mu.Unlock()
}

func waitFor(t *testing.T, ok func() bool) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for !ok() {
		if time.Now().After(deadline) {
			t.Fatal("condition not reached")
		}
		time.Sleep(5 * time.Millisecond)
	}
}

type inventoryFixture func(ctx context.Context, method string, out any) error

func (f inventoryFixture) Call(ctx context.Context, method string, _, out any) error {
	return f(ctx, method, out)
}

func bundlesReply(out any) error {
	return json.Unmarshal([]byte(`[{"name":"demo","official":false,"availability":"profile","enabled":true,"installed":true,"optional":true,"removable":true,"rows":[],"overrides":[]}]`), out)
}

func TestMarketSnapshotSharedUntilWrite(t *testing.T) {
	s := NewWithPersonalPath("")
	s.Attach(&dsh.Client{}, nil)
	s.SetInventoryCaller(inventoryFixture(func(_ context.Context, _ string, out any) error { return bundlesReply(out) }))
	s.Refresh(context.Background())
	waitFor(t, func() bool { return !s.Snapshot().Value.Loading })
	first := s.Snapshot()
	if first != s.Snapshot() {
		t.Fatal("unchanged state must hand out the same *Version")
	}
	if got := testing.AllocsPerRun(100, func() { s.Snapshot() }); got != 0 {
		t.Fatalf("Snapshot allocates %g per run", got)
	}
	if len(first.Value.Rows) != 1 || &first.Value.Rows[0] != &s.Snapshot().Value.Rows[0] {
		t.Fatal("rows must be shared, not copied per read")
	}
	if first.Value.Runtime != s.RuntimeToken() {
		t.Fatal("published Runtime differs from RuntimeToken")
	}
	d := s.DiscoveryVersion()
	if d != s.DiscoveryVersion() {
		t.Fatal("unchanged discovery must hand out the same *Version")
	}
	if got := testing.AllocsPerRun(100, func() { s.Discovery() }); got != 0 {
		t.Fatalf("Discovery allocates %g per run", got)
	}
}

func TestMarketWritePublishesAndSignals(t *testing.T) {
	s := NewWithPersonalPath("")
	before := s.Snapshot()
	changed := s.Changed()
	s.Attach(&dsh.Client{}, nil)
	after := s.Snapshot()
	if after == before || after.Rev != before.Rev+1 || !after.Value.ClientReady {
		t.Fatalf("Attach did not publish Rev+1: %d -> %d", before.Rev, after.Rev)
	}
	select {
	case <-changed:
	default:
		t.Fatal("Changed channel did not close on publish")
	}
	dBefore := s.DiscoveryVersion()
	s.SetQuery("demo")
	if d := s.DiscoveryVersion(); d == dBefore || d.Value.Query != "demo" {
		t.Fatal("SetQuery did not publish a discovery version")
	}
}

func TestMarketStaleInventoryReadDropped(t *testing.T) {
	s := NewWithPersonalPath("")
	client := &dsh.Client{}
	s.Attach(client, nil)
	started, cancelled := make(chan struct{}), make(chan struct{})
	var once sync.Once
	s.SetInventoryCaller(inventoryFixture(func(ctx context.Context, _ string, out any) error {
		first := false
		once.Do(func() { first = true })
		if first {
			close(started)
			<-ctx.Done()
			close(cancelled)
			return bundlesReply(out)
		}
		return errors.New("second read")
	}))
	s.RefreshShared(context.Background())
	<-started
	// A blocked inventory RPC must not block readers.
	read := make(chan bool, 1)
	go func() { read <- s.Snapshot().Value.Loading }()
	select {
	case loading := <-read:
		if !loading {
			t.Fatal("in-flight read not published as Loading")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Snapshot blocked behind the inventory RPC")
	}
	s.Attach(client, nil) // renews runtime and inventory leases
	select {
	case <-cancelled:
	case <-time.After(3 * time.Second):
		t.Fatal("superseded inventory read was not cancelled")
	}
	held := s.Snapshot()
	time.Sleep(50 * time.Millisecond)
	if got := s.Snapshot(); got != held || len(got.Value.Rows) != 0 {
		t.Fatal("stale inventory completion was published")
	}
}

func TestMarketFaultMapping(t *testing.T) {
	s := NewWithPersonalPath("")
	s.Attach(&dsh.Client{}, nil)
	s.SetInventoryCaller(inventoryFixture(func(context.Context, string, any) error {
		return errors.New("dial tcp 127.0.0.1:1: connectex: connection refused")
	}))
	s.Refresh(context.Background())
	waitFor(t, func() bool { return !s.Snapshot().Value.Loading })
	f := s.Snapshot().Value.Err
	if f == nil || f.Cause != errcopy.CauseNetwork || f.SentenceText() == "" {
		t.Fatalf("Err = %+v", f)
	}
	s.SetCatalogService(failingCatalog{})
	s.RefreshCatalog(false, "en")
	waitFor(t, func() bool { return !s.Discovery().Loading })
	if d := s.Discovery(); d.Err == nil || d.Err.Cause != errcopy.CauseNetwork {
		t.Fatalf("Discovery.Err = %+v", d.Err)
	}
}

type failingCatalog struct{}

func (failingCatalog) List(context.Context, bool, string) (desktopmarket.Catalog, error) {
	return desktopmarket.Catalog{}, errors.New("dial tcp: connection refused")
}
func (failingCatalog) Details(context.Context, desktopmarket.CatalogItem, bool) (desktopmarket.Details, error) {
	return desktopmarket.Details{}, errors.New("dial tcp: connection refused")
}

type blockingCatalog struct{ release chan struct{} }

func (c blockingCatalog) List(ctx context.Context, _ bool, _ string) (desktopmarket.Catalog, error) {
	select {
	case <-c.release:
	case <-ctx.Done():
		return desktopmarket.Catalog{}, ctx.Err()
	}
	return desktopmarket.Catalog{Items: []desktopmarket.CatalogItem{{ID: "late/item"}}}, nil
}
func (blockingCatalog) Details(context.Context, desktopmarket.CatalogItem, bool) (desktopmarket.Details, error) {
	return desktopmarket.Details{}, nil
}

// SetCatalogService retires the old service's read; its late result must
// not land in the replacement discovery state.
func TestMarketCatalogServiceSwapDropsStaleList(t *testing.T) {
	s := NewWithPersonalPath("")
	release := make(chan struct{})
	s.SetCatalogService(blockingCatalog{release: release})
	s.RefreshCatalog(false, "en")
	s.SetCatalogService(stubMarketCatalog{})
	close(release)
	time.Sleep(50 * time.Millisecond)
	if d := s.Discovery(); len(d.Catalog.Items) != 0 || d.Loading {
		t.Fatalf("stale catalog list landed after service swap: %+v", d)
	}
}

// Favorites persistence writes outside the state lock: a blocked disk
// write never blocks Snapshot or Discovery.
func TestMarketPersonalWriteOutsideLock(t *testing.T) {
	s := NewWithPersonalPath(filepath.Join(t.TempDir(), "market.json"))
	s.persistMu.Lock() // a writer holding the persistence slot
	done := make(chan struct{})
	go func() {
		defer close(done)
		_ = s.RecordInstall("owner/plugin", "applied", "")
	}()
	read := make(chan struct{})
	go func() { _ = s.Snapshot(); _ = s.Discovery(); _ = s.Personal(); close(read) }()
	select {
	case <-read:
	case <-time.After(2 * time.Second):
		t.Fatal("readers blocked behind personal persistence")
	}
	s.persistMu.Unlock()
	<-done
	if h := s.Personal().History; len(h) != 1 || h[0].Plugin != "owner/plugin" {
		t.Fatalf("history not adopted: %+v", h)
	}
}
