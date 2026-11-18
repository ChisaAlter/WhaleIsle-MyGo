package pluginmanager

import (
	"context"
	"errors"
	"testing"
)

func TestInstalledPluginRequiresCompletedAuthoritativeInventory(t *testing.T) {
	c := New()
	base := basicFixture(nil)
	mode := "present"
	var during InstalledPluginState
	c.Attach(fixtureCall(func(ctx context.Context, method string, args, out any) error {
		if method == "pluginManager/listBundles" {
			during = c.InstalledPlugin("fixture-plugin")
			switch mode {
			case "failed":
				return errors.New("offline")
			case "absent":
				return reply(out, []Bundle{})
			case "disabled":
				bundle := fixtureBundle()
				bundle.Enabled = false
				return reply(out, []Bundle{bundle})
			}
		}
		return base(ctx, method, args, out)
	}), nil)
	defer c.Detach()
	if c.InstalledPlugin("fixture-plugin").Known {
		t.Fatal("unloaded inventory was treated as authoritative absence")
	}
	if err := c.Load(context.Background()); err != nil {
		t.Fatal(err)
	}
	first := c.InstalledPlugin("fixture-plugin")
	if !first.Known || !first.Enabled || first.Revision == 0 || during.Known {
		t.Fatalf("invalid completed inventory: first=%+v loading=%+v", first, during)
	}
	mode = "failed"
	if c.Load(context.Background()) == nil {
		t.Fatal("expected failed refresh")
	}
	if failed := c.InstalledPlugin("fixture-plugin"); failed.Known || failed.Revision != first.Revision {
		t.Fatalf("failed refresh acquired authority: %+v", failed)
	}
	for _, next := range []string{"disabled", "absent"} {
		mode = next
		if err := c.Load(context.Background()); err != nil {
			t.Fatal(err)
		}
		state := c.InstalledPlugin("fixture-plugin")
		if !state.Known || state.Enabled || state.Revision <= first.Revision || state.Owner != first.Owner {
			t.Fatalf("%s package did not revoke its completed lease: %+v", next, state)
		}
	}
	c.Attach(base, nil)
	if replacement := c.InstalledPlugin("fixture-plugin"); replacement.Known || replacement.Owner == first.Owner || replacement.Revision != 0 {
		t.Fatalf("replacement Host inherited availability: %+v", replacement)
	}
}

func TestInstalledPluginOverlayUsesExactCompletedLoaderInventory(t *testing.T) {
	const name = "dsh-usage-panel"
	c := New()
	base := basicFixture(nil)
	var bundles []Bundle
	plugins := []Plugin{{ModuleName: name, Enabled: true, Phase: "active"}}
	fail := false
	var during InstalledPluginState
	caller := fixtureCall(func(ctx context.Context, method string, args, out any) error {
		switch method {
		case "pluginManager/listBundles":
			return reply(out, bundles)
		case "pluginManager/listPlugins":
			during = c.InstalledPlugin(name)
			if fail {
				return errors.New("Loader inventory unavailable")
			}
			return reply(out, plugins)
		default:
			return base(ctx, method, args, out)
		}
	})
	c.Attach(caller, nil)
	defer c.Detach()
	if err := c.Load(context.Background()); err != nil {
		t.Fatal(err)
	}
	first := c.InstalledPlugin(name)
	if !first.Known || !first.Enabled || first.Revision == 0 || during.Known {
		t.Fatalf("patch-only overlay was rejected or published before complete read: %+v, loading=%+v", first, during)
	}
	for _, tc := range []struct {
		name   string
		bundle Bundle
	}{
		{"disabled", Bundle{Name: name, Availability: "profile", Enabled: false}},
		{"missing", Bundle{Name: name, Availability: "missing", Enabled: true}},
		{"failed", Bundle{Name: name, Availability: "profile", Enabled: true, Error: &ManagementError{Code: "failed"}}},
	} {
		bundles = []Bundle{tc.bundle}
		if err := c.Load(context.Background()); err != nil {
			t.Fatal(err)
		}
		if state := c.InstalledPlugin(name); !state.Known || state.Enabled {
			t.Fatalf("Loader bypassed %s bundle state: %+v", tc.name, state)
		}
	}
	bundles = nil
	for _, rows := range [][]Plugin{
		{{ModuleName: name + "-client", Enabled: true, Phase: "active"}},
		{{ModuleName: name, Enabled: false, Phase: "active"}},
		{{ModuleName: name, Enabled: true, Phase: "failed"}},
		nil,
	} {
		plugins = rows
		if err := c.Load(context.Background()); err != nil {
			t.Fatal(err)
		}
		if state := c.InstalledPlugin(name); !state.Known || state.Enabled {
			t.Fatalf("non-live or inexact Loader row kept overlay available: %+v from %+v", state, rows)
		}
	}
	plugins = []Plugin{{ModuleName: name, Enabled: true, Phase: "active"}}
	if err := c.Load(context.Background()); err != nil {
		t.Fatal(err)
	}
	last := c.InstalledPlugin(name)
	fail = true
	if c.Load(context.Background()) == nil {
		t.Fatal("expected incomplete Loader read")
	}
	if state := c.InstalledPlugin(name); state.Known || state.Revision != last.Revision {
		t.Fatalf("failed refresh published Loader authority: %+v", state)
	}
	c.Attach(caller, nil)
	if state := c.InstalledPlugin(name); state.Known || state.Enabled || state.Owner == first.Owner || state.Revision != 0 {
		t.Fatalf("replacement Host retained overlay availability: %+v", state)
	}
}
