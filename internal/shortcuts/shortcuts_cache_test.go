package shortcuts

import (
	"reflect"
	"testing"
)

func TestCachedDefaultsAndSharedDefinitionsRemainIsolated(t *testing.T) {
	d1 := Defaults()
	d1["sidebar.toggle"] = "MUTATED"
	d1["injected.id"] = "Mod+X"
	d2 := Defaults()
	if d2["sidebar.toggle"] == "MUTATED" {
		t.Fatal("mutating Defaults() leaked into cache")
	}
	if _, ok := d2["injected.id"]; ok {
		t.Fatal("inserted key leaked into Defaults() cache")
	}
	for id, want := range d2 {
		if got := DefaultFor(id); got != want {
			t.Fatalf("DefaultFor(%q)=%q want %q", id, got, want)
		}
	}

	sd1 := SharedDefaults()
	sd1["session.search"] = "MUTATED"
	if got := SharedDefaults()["session.search"]; got != "Mod+K" {
		t.Fatalf("mutating SharedDefaults() leaked into cache: %q", got)
	}

	cmds := Commands()
	cmds[0].ID = "MUTATED"
	if Commands()[0].ID == "MUTATED" {
		t.Fatal("mutating Commands() leaked into cache")
	}

	defs1 := SharedDefinitions("desktop:windows")
	defs1[0].ID = "MUTATED"
	if len(defs1[0].Defaults) > 0 {
		for k := range defs1[0].Defaults {
			delete(defs1[0].Defaults, k)
		}
	}
	defs2 := SharedDefinitions("desktop:windows")
	if defs2[0].ID == "MUTATED" {
		t.Fatal("mutating SharedDefinitions() leaked into cache")
	}
	if !reflect.DeepEqual(defs2, SharedDefinitions("desktop:windows")) {
		t.Fatal("SharedDefinitions cache returned inconsistent values")
	}
}
