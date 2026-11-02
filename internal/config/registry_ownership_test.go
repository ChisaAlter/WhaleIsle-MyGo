package config

import (
	"errors"
	"reflect"
	"testing"
)

// Pointer-shaped preferences must obey the same ownership contract as slices:
// neither a returned snapshot nor a retained callback may mutate a committed
// value, and a rejected candidate must leave memory and disk unchanged.
func TestRegistryPreferencesAreDetachedAcrossOwnershipBoundaries(t *testing.T) {
	dir := t.TempDir()
	writeConfig(t, dir, `{"goPluginInstallRegistry":{"kind":"custom","registry":"https://registry.example.test","url":"https://catalog.example.test"},"unknown":{"keep":true}}`)
	store, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	baseline := store.Get()
	bytes := readConfig(t, dir)
	returned := store.Get()
	returned.GoPluginInstallRegistry.Kind = "mutated snapshot"
	*returned.GoPluginInstallRegistry.Registry = "mutated address"
	if got := store.Get(); !reflect.DeepEqual(got, baseline) {
		t.Fatal("returned registry pointers still alias the committed configuration")
	}
	refused := errors.New("candidate refused")
	var rejected *PluginInstallRegistry
	if err := store.UpdateChecked(func(c *Config) error {
		rejected = c.GoPluginInstallRegistry
		rejected.Kind = "rejected kind"
		*rejected.Registry = "rejected address"
		return refused
	}); !errors.Is(err, refused) {
		t.Fatalf("candidate error was lost: %v", err)
	}
	if got := store.Get(); !reflect.DeepEqual(got, baseline) || readConfig(t, dir) != bytes {
		t.Fatal("rejected registry candidate changed committed memory or disk")
	}
	rejected.URL = "changed after rejection"
	if got := store.Get(); !reflect.DeepEqual(got, baseline) {
		t.Fatal("retained rejected candidate changed committed memory")
	}
	address := "https://new-registry.example.test"
	caller := &PluginInstallRegistry{Kind: "custom", Registry: &address}
	var retained *PluginInstallRegistry
	if err := store.UpdateChecked(func(c *Config) error {
		c.GoPluginInstallRegistry = caller
		retained = c.GoPluginInstallRegistry
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	committed := store.Get()
	retained.Kind = "changed retained"
	*caller.Registry = "changed caller"
	if got := store.Get(); !reflect.DeepEqual(got, committed) {
		t.Fatal("successful callback retained an alias to committed memory")
	}
	reopened, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(reopened.Get(), committed) {
		t.Fatal("registry memory diverged from saved configuration")
	}
}

func TestCloneConfigPreservesRegistryNullDistinctions(t *testing.T) {
	if got := cloneConfig(Config{}); got.GoPluginInstallRegistry != nil {
		t.Fatal("absent registry became present")
	}
	input := Config{GoPluginInstallRegistry: &PluginInstallRegistry{Kind: "system"}}
	got := cloneConfig(input)
	if got.GoPluginInstallRegistry == nil || got.GoPluginInstallRegistry == input.GoPluginInstallRegistry || got.GoPluginInstallRegistry.Registry != nil {
		t.Fatal("present registry with null address did not remain detached and null")
	}
	input.GoPluginInstallRegistry.Kind = "changed"
	if got.GoPluginInstallRegistry.Kind != "system" {
		t.Fatal("registry structure was not copied")
	}
}
