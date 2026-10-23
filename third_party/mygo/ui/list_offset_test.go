package ui

import "testing"

func TestListOffsetRequestOverridesKeyAnchorAtSameOffset(t *testing.T) {
	ids := make([]int, 100)
	for i := range ids {
		ids[i] = i
	}
	var state ListState
	var scroll ScrollState
	state.Key = func(i int) any { return ids[i] }
	tt := coreNewTester(func(c *context) {
		coreList(c, &state, len(ids), func(i int) { coreBox(c).Height(30) }).TrackScroll(&scroll).Grow(1).Padding(0, 0, 240, 0)
	}, 300, 300)
	state.ScrollToOffset(600)
	tt.Frame()
	if scroll.Y != 600 {
		t.Fatalf("offset=%v", scroll.Y)
	}
	ids = append([]int{1000, 1001}, ids...)
	state.ScrollToOffset(600)
	tt.Frame()
	if scroll.Y != 600 {
		t.Fatalf("key anchoring replaced equal offset: %v", scroll.Y)
	}
	state.ScrollToOffset(-20)
	tt.Frame()
	if scroll.Y != 0 {
		t.Fatalf("negative offset=%v", scroll.Y)
	}
	state.ScrollToOffset(1e7)
	tt.Frame()
	if scroll.Y != scroll.MaxY {
		t.Fatalf("offset did not clamp: %+v", scroll)
	}
}
