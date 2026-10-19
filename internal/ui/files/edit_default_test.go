package files

import "testing"

func TestEditDefaultKeepsSingleEditorAndExplicitModes(t *testing.T) {
	// Startup's fallback is Edit; a remembered choice still supersedes it.
	previous := editViewModeFallback
	t.Cleanup(func() { editViewModeFallback = previous })
	editViewModeFallback = EditViewEdit
	if got := resolveEditViewMode(Model{}, true); got != EditViewEdit {
		t.Fatalf("new editor fallback = %v", got)
	}
	invalid := EditViewMode(99)
	if got := resolveEditViewMode(Model{EditViewMode: &invalid}, true); got != EditViewEdit {
		t.Fatalf("invalid mode = %v", got)
	}
	for _, explicit := range []EditViewMode{EditViewSplit, EditViewHighlight} {
		if got := resolveEditViewMode(Model{EditViewMode: &explicit}, true); got != explicit {
			t.Fatalf("explicit mode %v became %v", explicit, got)
		}
	}
	if EditViewSplit != 0 || EditViewEdit != 1 || EditViewHighlight != 2 {
		t.Fatal("persisted mode numbers changed")
	}
}
