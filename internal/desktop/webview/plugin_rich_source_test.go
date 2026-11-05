package webview

import (
	"testing"
	"whaleisle/internal/desktop/pluginslots"
)

func TestPluginRichSourceKeepsCapabilityAndLegacyDimensionGate(t *testing.T) {
	// Invalid capabilities are rejected before any native window or application
	// loop is needed. Real HWND clipping is covered by the existing native test.
	h := &Host{}
	g := &pluginslots.Gateway{}
	for _, size := range [][2]int{{0, 1}, {1, 0}, {8193, 1}, {1, 8193}} {
		if _, err := h.PluginRichSource(g, size[0], size[1]); err == nil {
			t.Fatal("rich dimension accepted", size)
		}
	}
	if _, err := h.PluginRichSource(nil, 10, 10); err == nil {
		t.Fatal("nil capability accepted")
	}
	if _, err := h.PluginMirrorSource(g, 2049, 100); err == nil {
		t.Fatal("legacy mirror width expanded")
	}
	if _, err := h.PluginMirrorSource(g, 100, 1537); err == nil {
		t.Fatal("legacy mirror height expanded")
	}
	// A non-matching exact gateway is rejected without touching the absent HWND.
	if _, err := h.PluginRichSource(g, 10, 10); err == nil {
		t.Fatal("unregistered capability accepted")
	}
}
