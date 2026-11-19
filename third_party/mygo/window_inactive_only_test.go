package mygo

import "testing"

// A launch nobody asked for may put a window on screen; it may not take the
// foreground. The flag is on the window, so every raising call reaches
// ShowInactive instead of the activating Show, including calls added later.
func TestInactiveOnlyWindowAppearsWithoutActivating(t *testing.T) {
	w, _ := testWindow(t, WindowOptions{Title: "quiet", Width: 300, Height: 200, Hidden: true})
	w.SetInactiveOnly(true)

	w.Show()
	if !w.IsVisible() {
		t.Fatal("Show left an inactive-only window hidden")
	}
	if w.IsFocused() {
		t.Fatal("Show focused an inactive-only window")
	}

	w.Hide()
	w.Restore()
	if !w.IsVisible() {
		t.Fatal("Restore left an inactive-only window hidden")
	}
	if w.IsFocused() {
		t.Fatal("Restore focused an inactive-only window")
	}

	w.SetInactiveOnly(false)
	w.Show()
	if !w.IsFocused() {
		t.Fatal("Show did not focus the window once it was allowed to activate")
	}
}
