package session

import "testing"

func TestFrameSnapshotOwnerMatchIsImmutableAndNilSafe(t *testing.T) {
	first, second := New(), New()
	var absent *FrameSnapshot
	if absent.BelongsTo(first) {
		t.Fatal("nil pass matched a Store")
	}
	pass := first.BeginFrame()
	if !pass.BelongsTo(first) || pass.BelongsTo(second) || pass.BelongsTo(nil) {
		t.Fatal("pass owner match was not exact")
	}
	first.Select("current")
	if !pass.BelongsTo(first) {
		t.Fatal("selection changed immutable pass owner")
	}
	pass.EndFrame()
	if !pass.BelongsTo(first) {
		t.Fatal("ending pass changed its Store identity")
	}
}
