package pluginmanager

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// Context binding occurs after capture releases the controller lock. Replace
// the owner there to make the reconnect/admission interleaving deterministic.
type reconnectDuringCapture struct {
	context.Context
	once    sync.Once
	replace func()
}

func (c *reconnectDuringCapture) Done() <-chan struct{} {
	c.once.Do(c.replace)
	return nil
}

func TestManagerRetiredLoadCannotMarkReplacementLoading(t *testing.T) {
	c := New()
	var retiredCalls atomic.Int32
	c.Attach(fixtureCall(func(context.Context, string, any, any) error { retiredCalls.Add(1); return errors.New("retired Host") }), nil)
	ctx := &reconnectDuringCapture{Context: context.Background(), replace: func() { c.Attach(basicFixture(nil), nil) }}
	if err := c.Load(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("retired load admitted: %v", err)
	}
	if state := c.Snapshot(); state.Loading || state.Status != "idle" || state.Error != "" || retiredCalls.Load() != 0 {
		t.Fatalf("retired read polluted replacement: %+v calls=%d", state, retiredCalls.Load())
	}
	if err := c.Load(context.Background()); err != nil {
		t.Fatal(err)
	}
	if state := c.Snapshot(); state.Loading || state.Status != "ready" {
		t.Fatalf("replacement cannot load: %+v", state)
	}
}

func TestManagerOldInstallRegistryReadCannotPolluteReopenedDialog(t *testing.T) {
	c := New()
	started, release := make(chan struct{}), make(chan struct{})
	var registryCalls atomic.Int32
	base := basicFixture(nil)
	c.Attach(fixtureCall(func(ctx context.Context, method string, args, out any) error {
		if method != "pluginManager/registries" {
			return base(ctx, method, args, out)
		}
		if registryCalls.Add(1) == 1 {
			close(started)
			<-release
			return errors.New("retired dialog registry error")
		}
		return reply(out, RegistryInfo{FallbackRegistries: []string{"https://fresh-registry.invalid/"}})
	}), nil)
	a := c.Actions(c.Snapshot().Owner)
	a.OpenInstall()
	select {
	case <-started:
	case <-time.After(3 * time.Second):
		t.Fatal("first registry read did not start")
	}
	a.CloseInstall()
	a.OpenInstall()
	waitState(t, c, func(s State) bool { return s.Install.RegistryInfo != nil })
	changed := make(chan struct{}, 1)
	c.SetNotify(func() {
		select {
		case changed <- struct{}{}:
		default:
		}
	})
	close(release)
	select {
	case <-changed:
	case <-time.After(3 * time.Second):
		t.Fatal("retired registry callback did not settle")
	}
	state := c.Snapshot()
	if state.Install.Error != "" || len(state.Install.RegistryInfo.FallbackRegistries) != 1 || state.Install.RegistryInfo.FallbackRegistries[0] != "https://fresh-registry.invalid/" {
		t.Fatalf("retired dialog read replaced the reopened dialog: %+v", state.Install)
	}
}
