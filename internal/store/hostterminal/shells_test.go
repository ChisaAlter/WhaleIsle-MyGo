package hostterminal

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
	"whaleisle/internal/config"
)

type shellReply struct {
	items []Shell
	err   error
}
type shellFixtureRemote struct {
	*fixtureRemote
	requests chan map[string]any
	replies  chan shellReply
}

func (r *shellFixtureRemote) Call(ctx context.Context, method string, args, out any) error {
	if method != "terminal/shells" {
		return r.fixtureRemote.Call(ctx, method, args, out)
	}
	r.requests <- object(args)
	reply := <-r.replies // Deliberately allow a canceled Host's late response.
	if reply.err != nil {
		return reply.err
	}
	b, _ := json.Marshal(reply.items)
	return json.Unmarshal(b, out)
}
func shellFixture() (*Store, *shellFixtureRemote, uint64) {
	r := &shellFixtureRemote{fixture(), make(chan map[string]any, 8), make(chan shellReply, 8)}
	s := new(Store)
	s.bind(r, r, true, nil)
	s.Show("session", true)
	return s, r, s.Snapshot("session").Owner
}
func replyShells(t *testing.T, r *shellFixtureRemote, items []Shell, err error) {
	t.Helper()
	select {
	case req := <-r.requests:
		if len(req) != 1 || req["agentId"] != "session" {
			t.Fatal(req)
		}
	case <-time.After(time.Second):
		t.Fatal("shell discovery missing")
	}
	r.replies <- shellReply{items, err}
}
func waitShells(t *testing.T, s *Store, o uint64, pred func(ShellView) bool) ShellView {
	t.Helper()
	deadline := time.After(time.Second)
	for {
		v := s.ShellSnapshot(o, "session")
		if pred(v) {
			return v
		}
		select {
		case <-deadline:
			t.Fatalf("shell state %+v", v)
		case <-time.After(time.Millisecond):
		}
	}
}

var shellChoices = []Shell{{Path: "/host/default", Name: "Default", Args: []string{"-l"}}, {Path: "/host/preferred", Name: "Preferred"}}

func discoverShells(t *testing.T, s *Store, r *shellFixtureRemote, o uint64) ShellView {
	t.Helper()
	s.DiscoverShells(o, "session")
	replyShells(t, r, shellChoices, nil)
	return waitShells(t, s, o, func(v ShellView) bool { return v.Loaded && !v.Loading })
}
func lastCreate(t *testing.T, r *shellFixtureRemote) map[string]any {
	t.Helper()
	r.mu.Lock()
	defer r.mu.Unlock()
	for i := len(r.calls) - 1; i >= 0; i-- {
		if r.calls[i].method == "terminal/create" {
			a := r.calls[i].args
			if a["agentId"] != "session" {
				t.Fatal(a)
			}
			return a["request"].(map[string]any)
		}
	}
	t.Fatal("create missing")
	return nil
}
func TestHostTerminalShellsDiscoveryAndSelectionNeverAllocate(t *testing.T) {
	s, r, o := shellFixture()
	defer s.bind(nil, nil, false, nil)
	saved := ""
	s.ConfigureShellPreference("/host/preferred", func(p string) error { saved = p; return nil })
	v := discoverShells(t, s, r, o)
	if v.SelectedPath != "/host/preferred" || saved != "" {
		t.Fatal(v, saved)
	}
	v.Items[0].Args[0] = "mutated"
	if s.ShellSnapshot(o, "session").Items[0].Args[0] != "-l" {
		t.Fatal("aliased shell snapshot")
	}
	s.SelectShell(o, "session", "/invented")
	if saved != "" {
		t.Fatal("accepted arbitrary path")
	}
	s.SelectShell(o, "session", "/host/default")
	if saved != "/host/default" || s.ShellSnapshot(o, "session").SelectedPath != saved {
		t.Fatal(saved)
	}
	if r.count("terminal/create") != 0 || r.count("terminal/environment") != 0 || len(r.opens) != 0 {
		t.Fatal("discovery or selection allocated/attached")
	}
}
func TestHostTerminalShellsRememberedFallbackAndCreateWire(t *testing.T) {
	s, r, o := shellFixture()
	defer s.bind(nil, nil, false, nil)
	s.ConfigureShellPreference("/another-host/missing", nil)
	s.Create(o, "session")
	replyShells(t, r, shellChoices, nil)
	v := wait(t, s, "session", func(v View) bool { return !v.Busy && v.ActiveID != "" })
	req := lastCreate(t, r)
	if len(req) != 4 || req["shellPath"] != "/host/default" || req["id"] != v.ActiveID || req["args"] != nil {
		t.Fatal(req)
	}
	if r.count("terminal/create") != 1 {
		t.Fatal("duplicate creation")
	}
}
func TestHostTerminalShellsUnavailableExplicitSelectionPreserved(t *testing.T) {
	s, r, o := shellFixture()
	defer s.bind(nil, nil, false, nil)
	discoverShells(t, s, r, o)
	s.SelectShell(o, "session", "/host/preferred")
	s.Create(o, "session")
	replyShells(t, r, shellChoices[:1], nil)
	v := wait(t, s, "session", func(v View) bool { return !v.Busy && v.Error != "" })
	if v.ActiveID != "" || r.count("terminal/create") != 0 || s.ShellSnapshot(o, "session").SelectedPath != "/host/preferred" {
		t.Fatal(v)
	}
}
func TestHostTerminalShellsPersistenceFailureDoesNotBlockExplicitCreate(t *testing.T) {
	s, r, o := shellFixture()
	defer s.bind(nil, nil, false, nil)
	s.ConfigureShellPreference("", func(string) error { return errors.New("read-only preferences") })
	discoverShells(t, s, r, o)
	s.SelectShell(o, "session", "/host/preferred")
	if s.ShellSnapshot(o, "session").PreferenceError == "" {
		t.Fatal("missing save warning")
	}
	s.Create(o, "session")
	replyShells(t, r, shellChoices, nil)
	wait(t, s, "session", func(v View) bool { return !v.Busy && v.ActiveID != "" })
	if lastCreate(t, r)["shellPath"] != "/host/preferred" {
		t.Fatal("lost choice")
	}
}
func TestHostTerminalShellsHostReplacementRejectsStaleDiscoveryAndSelection(t *testing.T) {
	s, r, o := shellFixture()
	s.DiscoverShells(o, "session")
	s.bind(nil, nil, false, nil)
	r2 := fixture()
	s.bind(r2, r2, true, nil)
	defer s.bind(nil, nil, false, nil)
	s.Show("session", true)
	replyShells(t, r, shellChoices, nil)
	s.SelectShell(o, "session", "/host/preferred")
	s.Create(o, "session")
	if s.ShellSnapshot(o, "session").Loaded || s.ShellSnapshot(s.Snapshot("session").Owner, "session").Loaded || r2.count("terminal/create") != 0 {
		t.Fatal("crossed Host")
	}
}
func TestHostTerminalShellsHideCancelsDiscoveryAndPreAllocation(t *testing.T) {
	s, r, o := shellFixture()
	defer s.bind(nil, nil, false, nil)
	s.DiscoverShells(o, "session")
	s.Show("", false)
	replyShells(t, r, shellChoices, nil)
	if s.ShellSnapshot(o, "session").Loading || s.ShellSnapshot(o, "session").Loaded {
		t.Fatal("hidden discovery survived")
	}
	s.Show("session", true)
	s.Create(o, "session")
	s.Show("", false)
	replyShells(t, r, shellChoices, nil)
	v := wait(t, s, "session", func(v View) bool { return !v.Busy && v.Error != "" })
	if r.count("terminal/create") != 0 || v.ActiveID != "" {
		t.Fatal("allocated after hide", v)
	}
}
func TestHostTerminalShellsDiscoveryFailurePreservesSelection(t *testing.T) {
	s, r, o := shellFixture()
	defer s.bind(nil, nil, false, nil)
	discoverShells(t, s, r, o)
	s.SelectShell(o, "session", "/host/preferred")
	s.DiscoverShells(o, "session")
	replyShells(t, r, nil, errors.New("discovery unavailable"))
	v := waitShells(t, s, o, func(v ShellView) bool { return !v.Loading && v.Error != "" })
	if v.SelectedPath != "/host/preferred" || !v.Loaded || r.count("terminal/create") != 0 {
		t.Fatal(v)
	}
	select {
	case req := <-r.requests:
		t.Fatal("automatic discovery retry", req)
	default:
	}
}
func TestHostTerminalShellsEmptyDiscoveryUsesHostDefault(t *testing.T) {
	s, r, o := shellFixture()
	defer s.bind(nil, nil, false, nil)
	s.Create(o, "session")
	replyShells(t, r, nil, nil)
	wait(t, s, "session", func(v View) bool { return !v.Busy && v.ActiveID != "" })
	if req := lastCreate(t, r); len(req) != 3 || req["shellPath"] != nil {
		t.Fatal(req)
	}
}
func TestHostTerminalShellsLocalPreferenceRoundTrip(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, config.FileName), []byte(`{"untouched":{"value":7}}`), 0600); err != nil {
		t.Fatal(err)
	}
	cfg, err := config.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	s, r, o := shellFixture()
	defer s.bind(nil, nil, false, nil)
	s.ConfigureShellPreference(cfg.Get().HostTerminalShell, func(p string) error { return cfg.Update(func(c *config.Config) { c.HostTerminalShell = p }) })
	discoverShells(t, s, r, o)
	s.SelectShell(o, "session", "/host/preferred")
	reopened, err := config.Open(dir)
	if err != nil || reopened.Get().HostTerminalShell != "/host/preferred" {
		t.Fatal(err, reopened)
	}
	b, _ := os.ReadFile(filepath.Join(dir, config.FileName))
	var m map[string]any
	_ = json.Unmarshal(b, &m)
	if m["untouched"] == nil || m["goHostTerminalShell"] != "/host/preferred" {
		t.Fatal(string(b))
	}
	s2, r2, o2 := shellFixture()
	defer s2.bind(nil, nil, false, nil)
	s2.ConfigureShellPreference(reopened.Get().HostTerminalShell, nil)
	if v := discoverShells(t, s2, r2, o2); v.SelectedPath != "/host/preferred" {
		t.Fatal(v)
	}
}
