package mygo

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

// TestPageOnBeforeInputNeedsAPage: a window that shows native Content has no
// page, so the hook reports no support instead of silently doing nothing.
func TestPageOnBeforeInputNeedsAPage(t *testing.T) {
	w := NewWindow(WindowOptions{Title: "content", Width: 200, Height: 100, Content: ui.View(func(*ui.Context) {})})
	defer w.Destroy()
	if w.Page().OnBeforeInput(func(KeyInput) bool { return true }) {
		t.Fatal("a Content window reported before-input support")
	}
	if w.Page().OnBeforeInput(nil) {
		t.Fatal("a nil callback reported support")
	}
}
