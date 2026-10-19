package pluginmanager

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func registryPtr(s string) *string { return &s }

func TestInspectAndInstallPreserveExplicitNullRegistryAtRPCBoundary(t *testing.T) {
	var calls []string
	client := Client{Caller: fixtureCall(func(_ context.Context, method string, args, out any) error {
		raw, err := json.Marshal(args)
		if err != nil {
			return err
		}
		var request struct {
			Options map[string]json.RawMessage `json:"options"`
		}
		if err := json.Unmarshal(raw, &request); err != nil {
			return err
		}
		if registry, exists := request.Options["registry"]; !exists || string(registry) != "null" {
			t.Fatalf("%s dropped pnpm's explicit default-registry choice: %s", method, raw)
		}
		calls = append(calls, method)
		return nil
	})}
	if _, err := client.Inspect(context.Background(), "fixture", nil); err != nil {
		t.Fatal(err)
	}
	if _, err := client.Install(context.Background(), "fixture", InstallOptions{}); err != nil {
		t.Fatal(err)
	}
	if len(calls) != 2 || calls[0] != "pluginManager/inspect" || calls[1] != "pluginManager/installBundle" {
		t.Fatal(calls)
	}
}

func TestDiscoverRegistryOverrideDoesNotReplaceRememberedInstallChoice(t *testing.T) {
	remembered := &RegistryChoice{Kind: "custom", URL: "https://remembered.invalid/"}
	var saved atomic.Int32
	c := New()
	c.SetRegistryPersistence(func() (*RegistryChoice, error) { return remembered, nil }, func(*RegistryChoice) error { saved.Add(1); return nil })
	c.Attach(basicFixture(nil), nil)
	defer c.Detach()
	owner := c.Snapshot().Owner
	if !c.openInstall(owner, "source-fixture", "https://source.invalid/", nil, false) {
		t.Fatal("source install input was rejected")
	}
	s := waitState(t, c, func(s State) bool { return s.Install.RegistryReadDone })
	if !s.Install.TransientRegistry || s.Install.Registry != "https://source.invalid/" || saved.Load() != 0 {
		t.Fatal("source-specific registry changed the stored preference", s.Install)
	}
	c.Actions(owner).CloseInstall()
	c.Actions(owner).OpenInstall()
	s = waitState(t, c, func(s State) bool { return s.Install.RegistryReadDone })
	if s.Install.TransientRegistry || s.Install.Registry != remembered.URL || saved.Load() != 0 {
		t.Fatal("ordinary Install inherited the previous source override", s.Install)
	}
}
func TestRegistryChoicesReconcileNormalizedURLsAndKeepPnpmNull(t *testing.T) {
	info := &RegistryInfo{Registry: registryPtr("https://REGISTRY.NPMJS.ORG:443"), Resolved: registryPtr(OfficialNPMRegistry), FallbackRegistries: []string{OfficialNPMRegistry, NPMMirrorRegistry, "https://registry.npmmirror.com"}}
	offered := OfferedRegistries(info)
	if len(offered) != 2 || RegistryKey(offered[0], info.Resolved) != OfficialNPMRegistry || RegistryKey(offered[1], info.Resolved) != NPMMirrorRegistry {
		t.Fatalf("duplicated aliases: %+v", offered)
	}
	remembered := &RegistryChoice{Kind: "offered", Registry: registryPtr("https://REGISTRY.NPMMIRROR.COM:443")}
	if got := reconciledRegistry(remembered, info); got.Kind != "offered" || got.Registry == nil || *got.Registry != NPMMirrorRegistry {
		t.Fatalf("remembered alias did not bind offered choice: %+v", got)
	}
	own := reconciledRegistry(&RegistryChoice{Kind: "offered"}, info)
	if own.Kind != "offered" || own.Registry != nil || RegistryValue(own) != nil {
		t.Fatal("pnpm null became a configured URL")
	}
	private := reconciledRegistry(&RegistryChoice{Kind: "offered", Registry: registryPtr("https://private.invalid/npm")}, info)
	if private.Kind != "custom" || private.URL != "https://private.invalid/npm" {
		t.Fatal(private)
	}
}

func TestRegistryPersistenceFailuresRemainVisibleAcrossOwnerAndCanDismiss(t *testing.T) {
	c := New()
	c.SetRegistryPersistence(func() (*RegistryChoice, error) { return nil, errors.New("read fixture") }, func(*RegistryChoice) error { return errors.New("write fixture") })
	c.Attach(basicFixture(nil), nil)
	t.Cleanup(c.Detach)
	if c.Snapshot().PreferenceError == "" {
		t.Fatal("replacement owner hid a failed preference read")
	}
	owner := c.Snapshot().Owner
	choice := RegistryChoice{Kind: "custom", URL: "https://private.invalid/npm/"}
	c.rememberRegistry(owner, choice)
	if c.Snapshot().PreferenceError != "Could not remember the installation registry: write fixture" {
		t.Fatal(c.Snapshot().PreferenceError)
	}
	c.Actions(owner).DismissPreferenceError()
	if c.Snapshot().PreferenceError != "" {
		t.Fatal("receipt did not dismiss")
	}
	c.Actions(owner).OpenInstall()
	waitState(t, c, func(s State) bool { return s.Install.RegistryReadDone })
	if c.Snapshot().Install.RegistrySelection.URL != choice.URL {
		t.Fatal("failed disk write discarded the active choice")
	}
}

func TestLatePublicMirrorProbeCannotOverwriteManualRegistry(t *testing.T) {
	started, release := make(chan struct{}), make(chan struct{})
	var once sync.Once
	unblock := func() { once.Do(func() { close(release) }) }
	defer unblock()
	base := basicFixture(nil)
	c := New()
	c.Attach(fixtureCall(func(ctx context.Context, method string, args, out any) error {
		switch method {
		case "pluginManager/registries":
			return reply(out, RegistryInfo{Resolved: registryPtr(OfficialNPMRegistry), FallbackRegistries: []string{NPMMirrorRegistry}})
		case "pluginRegistryProbe/fastest":
			close(started)
			<-release
			return reply(out, NPMMirrorRegistry)
		default:
			return base(ctx, method, args, out)
		}
	}), nil)
	defer c.Detach()
	a := c.Actions(c.Snapshot().Owner)
	a.OpenInstall()
	select {
	case <-started:
	case <-time.After(3 * time.Second):
		t.Fatal("eligible probe did not start")
	}
	a.ChooseRegistry(RegistryChoice{Kind: "custom", URL: "https://chosen.invalid/"})
	unblock()
	s := waitState(t, c, func(s State) bool { return s.Install.RegistryReadDone })
	if s.Install.Registry != "https://chosen.invalid/" || s.Install.RegistrySelection.Kind != "custom" {
		t.Fatalf("late probe replaced explicit input: %+v", s.Install)
	}
}

func TestGitHubRecoveryIsOfferedOnlyForTheOriginalFailureClass(t *testing.T) {
	initial := InstallState{Phase: "failed", Inspection: &Inspection{Host: "github.com"}, RegistryInfo: &RegistryInfo{FallbackRegistries: []string{NPMMirrorRegistry}}, Result: &Change{FailedAt: "spec-host", PackageResult: &PackageResult{Kind: "network"}}}
	if value, ok := GitHubRecoveryRegistry(initial); !ok || value == nil || *value != NPMMirrorRegistry {
		t.Fatal("eligible GitHub failure lost recovery")
	}
	for _, tc := range []struct {
		name   string
		change func(*InstallState)
	}{
		{"confirmed", func(s *InstallState) { s.Confirmed = true }},
		{"registry failure", func(s *InstallState) { r := *s.Result; r.FailedAt = "registry"; s.Result = &r }},
		{"permission", func(s *InstallState) {
			r := *s.Result
			r.PackageResult = &PackageResult{Kind: "permission"}
			s.Result = &r
		}},
		{"lookalike host", func(s *InstallState) { s.Inspection = &Inspection{Host: "github.com.attacker.invalid"} }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := initial
			tc.change(&s)
			if _, ok := GitHubRecoveryRegistry(s); ok {
				t.Fatal("unrelated failure offered a mirror")
			}
		})
	}
}

func TestRetiredInstallReplyCannotClearReplacementRequestBusy(t *testing.T) {
	firstRelease, secondRelease := make(chan struct{}), make(chan struct{})
	var firstOnce, secondOnce sync.Once
	freeFirst := func() { firstOnce.Do(func() { close(firstRelease) }) }
	freeSecond := func() { secondOnce.Do(func() { close(secondRelease) }) }
	defer freeFirst()
	defer freeSecond()
	var calls atomic.Int32
	base := basicFixture(nil)
	c := New()
	c.Attach(fixtureCall(func(ctx context.Context, method string, args, out any) error {
		if method == "pluginManager/installBundle" {
			if calls.Add(1) == 1 {
				<-firstRelease
			} else {
				<-secondRelease
			}
			return reply(out, Change{Application: "applied"})
		}
		if method == "pluginManager/cancelInstall" {
			return reply(out, map[string]string{"status": "cancelled"})
		}
		return base(ctx, method, args, out)
	}), nil)
	defer c.Detach()
	a := c.Actions(c.Snapshot().Owner)
	start := func(spec string) {
		a.edit(func(s *State) {
			s.Install = InstallState{Open: true, Spec: spec, Phase: "idle", RegistryReadDone: true, Inspection: &Inspection{Status: "accepted", Name: spec}}
		})
		a.RequestInstall()
	}
	start("first")
	waitState(t, c, func(s State) bool { return calls.Load() == 1 && s.Busy })
	a.CancelInstall()
	waitState(t, c, func(s State) bool { return s.Install.Phase == "idle" && !s.Busy })
	start("second")
	waitState(t, c, func(s State) bool { return calls.Load() == 2 && s.Busy })
	changed := make(chan struct{}, 1)
	c.SetNotify(func() {
		select {
		case changed <- struct{}{}:
		default:
		}
	})
	freeFirst()
	select {
	case <-changed:
	case <-time.After(3 * time.Second):
		t.Fatal("old reply did not settle")
	}
	if s := c.Snapshot(); !s.Busy || s.Install.Spec != "second" || s.Install.Result != nil {
		t.Fatalf("old reply changed a new request: %+v", s.Install)
	}
	freeSecond()
	waitState(t, c, func(s State) bool { return !s.Busy && s.Install.Result != nil })
}
