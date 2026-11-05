package main

import "testing"

func TestNativePluginHostIsOnUnlessSwitchedOff(t *testing.T) {
	t.Setenv("WHALEISLE_NATIVE_PLUGIN_HOST", "")
	if !nativePluginHostEnabled() {
		t.Fatal("native plugin host must be on by default")
	}
	for _, on := range []string{"1", "true", "yes"} {
		t.Setenv("WHALEISLE_NATIVE_PLUGIN_HOST", on)
		if !nativePluginHostEnabled() {
			t.Fatalf("WHALEISLE_NATIVE_PLUGIN_HOST=%q must keep the host on", on)
		}
	}
	t.Setenv("WHALEISLE_NATIVE_PLUGIN_HOST", " 0 ")
	if nativePluginHostEnabled() {
		t.Fatal("WHALEISLE_NATIVE_PLUGIN_HOST=0 must switch the host off")
	}
}
