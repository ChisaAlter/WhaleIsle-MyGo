package api

import "testing"

func TestHasListPluginsRPCInCodegen(t *testing.T) {
	if !HasListPluginsRPC() {
		t.Fatal("expected pluginManager/listPlugins in registry")
	}
	if got := ListPluginsMethod(); got != "pluginManager/listPlugins" {
		t.Fatalf("ListPluginsMethod = %q", got)
	}
	methods := PluginManagerMethods()
	if len(methods) == 0 {
		t.Fatal("expected pluginManager/* methods")
	}
	found := false
	for _, m := range methods {
		if m == "pluginManager/listPlugins" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("methods = %v", methods)
	}
}
