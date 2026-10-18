package externallauncher

import (
	"context"
	"errors"
	"testing"
	"whaleisle/internal/desktop/update"
)

func TestInstallWaitsThenAdopts(t *testing.T) {
	current := Installation{Root: "old", Version: "1"}
	waited := false
	released := false
	c := New(Dependencies{Discover: func() (Installation, error) { return current, nil }, Running: func(Installation) (bool, error) { return false, nil }, Maintenance: func() (func(), error) { return func() { released = true }, nil }, Install: func(context.Context, update.ApplyResult) error {
		waited = true
		current = Installation{Root: "new", Version: "2"}
		return nil
	}})
	got, err := c.Install(context.Background(), update.ApplyResult{}, "2")
	if err != nil || got.Root != "new" || !waited || !released {
		t.Fatalf("%+v %v", got, err)
	}
}
func TestInstallRejectsRunningAndWrongVersion(t *testing.T) {
	for _, running := range []bool{true, false} {
		calls := 0
		c := New(Dependencies{Discover: func() (Installation, error) { return Installation{Root: "old", Version: "1"}, nil }, Running: func(Installation) (bool, error) { return running, nil }, Maintenance: func() (func(), error) { return func() {}, nil }, Install: func(context.Context, update.ApplyResult) error { calls++; return nil }})
		_, err := c.Install(context.Background(), update.ApplyResult{}, "2")
		if err == nil || running && calls != 0 {
			t.Fatal("unsafe installation admitted")
		}
	}
}
func TestMaintenanceFailureBlocksStart(t *testing.T) {
	called := false
	c := New(Dependencies{Maintenance: func() (func(), error) { return nil, errors.New("import active") }, Start: func(Installation) error { called = true; return nil }})
	if c.Start(context.Background()) == nil || called {
		t.Fatal("import admission bypassed")
	}
}

func TestProtectedStopWaitsForExternalExit(t *testing.T) {
	running := true
	requested := false
	i := Installation{Root: "desktop", Executable: "desktop/whaleisle.exe", Version: "1"}
	c := New(Dependencies{Discover: func() (Installation, error) { return i, nil }, Running: func(got Installation) (bool, error) {
		if got != i {
			t.Fatal("changed target")
		}
		return running, nil
	}, Maintenance: func() (func(), error) { return func() {}, nil }, RequestStop: func(got Installation) error { requested = true; running = false; return nil }})
	if err := c.Stop(context.Background()); err != nil || !requested {
		t.Fatal(err)
	}
}
func TestInstallerFailureNeverAdopts(t *testing.T) {
	c := New(Dependencies{Discover: func() (Installation, error) { return Installation{Version: "2"}, nil }, Running: func(Installation) (bool, error) { return false, nil }, Maintenance: func() (func(), error) { return func() {}, nil }, Install: func(context.Context, update.ApplyResult) error { return errors.New("installer cancelled") }})
	if _, err := c.Install(context.Background(), update.ApplyResult{}, "2"); err == nil {
		t.Fatal("registration must not override failed installer")
	}
}
