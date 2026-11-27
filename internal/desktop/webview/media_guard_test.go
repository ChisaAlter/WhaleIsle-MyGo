package webview

import (
	"strings"
	"testing"
)

func TestHostLocalMediaURLNavigationGuarded(t *testing.T) {
	h := &Host{localMediaURL: "http://127.0.0.1:43210/media/preview.mp4"}
	err := h.Navigate("https://example.com/escape")
	if err == nil {
		t.Fatal("expected Navigate outside localMediaURL to be blocked")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "media") {
		t.Fatalf("expected media-specific block error, got %q", err.Error())
	}
	// Fragment navigation on the exact same media URL should pass the media guard
	// and fail later only because this unit-test Host has no native window.
	err = h.Navigate("http://127.0.0.1:43210/media/preview.mp4#t=5")
	if err == nil || strings.Contains(strings.ToLower(err.Error()), "blocked") {
		t.Fatalf("same localMediaURL with fragment was wrongly blocked: %v", err)
	}
}
