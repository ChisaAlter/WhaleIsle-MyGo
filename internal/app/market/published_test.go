package marketctl

import (
	"context"
	"reflect"
	"testing"

	"whaleisle/internal/dsh"
	marketstore "whaleisle/internal/store/market"
)

type envDeps struct {
	HostDeps
	env *HostEnvSnapshot
}

func (d envDeps) Env() HostEnvSnapshot              { return *d.env }
func (d envDeps) LifecycleContext() context.Context { return context.Background() }
func (d envDeps) InvalidateWindow()                 {}
func (d envDeps) UpdateWindow(func())               {}

func TestCoordinatorViewSharedUntilWrite(t *testing.T) {
	c := NewCoordinator(nil)
	c.SetStore(marketstore.NewWithPersonalPath(""))
	first := c.ViewSnapshot()
	if first != c.ViewSnapshot() {
		t.Fatal("unchanged state must hand out the same *Version")
	}
	if got := testing.AllocsPerRun(100, func() { c.ViewSnapshot() }); got != 0 {
		t.Fatalf("ViewSnapshot allocates %g per run", got)
	}
	// Read-only sections mint no revision.
	c.IsBusy()
	c.Open()
	c.SetOpen(false)
	if c.ViewSnapshot() != first {
		t.Fatal("a section that changed nothing published a new view")
	}
}

func TestCoordinatorWritePublishesAndSignals(t *testing.T) {
	c := NewCoordinator(nil)
	before := c.ViewSnapshot()
	changed := c.ViewChanged()
	c.SetInstallSpec("owner/plugin")
	after := c.ViewSnapshot()
	if after == before || after.Rev != before.Rev+1 || after.Value.InstallSpec != "owner/plugin" {
		t.Fatalf("SetInstallSpec did not publish Rev+1: %d -> %d", before.Rev, after.Rev)
	}
	select {
	case <-changed:
	default:
		t.Fatal("ViewChanged channel did not close on publish")
	}
}

// The shell environment and the store's runtime token are folded into the
// view: a change in either republishes on the next read.
func TestCoordinatorViewFoldsHostEnvAndStoreRuntime(t *testing.T) {
	env := HostEnvSnapshot{}
	c := NewCoordinator(envDeps{env: &env})
	st := marketstore.NewWithPersonalPath("")
	c.SetStore(st)
	v := c.ViewSnapshot()
	if v.Value.CanRestart {
		t.Fatal("CanRestart without a supervisor")
	}
	env.HasSupervisor = true
	next := c.ViewSnapshot()
	if next == v || !next.Value.CanRestart {
		t.Fatal("a supervisor appearing did not republish CanRestart")
	}
	if c.ViewSnapshot() != next {
		t.Fatal("unchanged environment republished")
	}
	st.Attach(&dsh.Client{}, nil)
	reattached := c.ViewSnapshot()
	if reattached == next || reattached.Value.Owner.Generation != st.RuntimeToken() {
		t.Fatalf("store reattachment not folded into Owner: %d vs %d", reattached.Value.Owner.Generation, st.RuntimeToken())
	}
}

// viewEqual must cover every ViewSnapshot field; a new field has to be
// added there too, or a change in it would never publish.
func TestViewEqualCoversEveryField(t *testing.T) {
	if n := reflect.TypeFor[ViewSnapshot]().NumField(); n != 17 {
		t.Fatalf("ViewSnapshot has %d fields; update viewEqual and this count", n)
	}
}
