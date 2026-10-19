package ui

import "testing"

func TestLayoutOffsetCurrentFrameMeasurementFlowAndHit(t *testing.T) {
	var card, viewport Rect
	var clicked int
	shift := float32(60)
	tt := NewTester(func(c *Context) {
		Column(c).Fill().ObserveLayout(func(r Rect) { viewport = r }).Children(func() {
			Column(c).Width(200).Height(50).OffsetAfterLayout(func(r Rect) Point {
				if card.Y != r.Y || viewport.H != 200 {
					t.Fatalf("offset ran before current-frame measurements: card=%+v parent=%+v viewport=%+v", card, r, viewport)
				}
				return Point{X: 25, Y: shift}
			}).Children(func() {
				b := Button(c, "Moved button").Label("Moved control").Width(100).Height(30).ObserveLayout(func(r Rect) { card = r })
				if b.Clicked() {
					clicked++
				}
			})
			Box(c).Label("Unmoved sibling").Height(30).Width(100)
		})
	}, 300, 200)
	defer tt.rt.close()
	moved, ok := tt.Find("Moved control")
	if !ok || moved.X != 25 || moved.Y != 60 || card.Y != 0 {
		t.Fatalf("first frame did not shift from current geometry: shown=%+v measured=%+v", moved, card)
	}
	sibling, _ := tt.Find("Unmoved sibling")
	if sibling.Y != 50 {
		t.Fatalf("relative displacement changed sibling flow: %+v", sibling)
	}
	if err := tt.Click("Moved control"); err != nil {
		t.Fatal(err)
	}
	// The later sibling occupies x0..100/y50..80 and covers the moved
	// button's center. Preserve ordinary paint-order hit occlusion, then
	// press its exposed right edge (x120), which only the moved control owns.
	if clicked != 0 {
		t.Fatal("displacement bypassed the later sibling's hit occlusion")
	}
	tt.ClickAt(moved.X+moved.W-5, moved.Y+moved.H/2)
	if clicked != 1 {
		t.Fatalf("displaced hit target did not follow paint: %d", clicked)
	}
	shift = 20
	tt.Frame()
	moved, _ = tt.Find("Moved control")
	if moved.Y != 20 || card.Y != 0 {
		t.Fatalf("offset accumulated across frames: shown=%+v measured=%+v", moved, card)
	}
}

func TestLayoutOffsetMovesNativeTextCandidateGeometry(t *testing.T) {
	text := "hello"
	shift := float32(0)
	tt := NewTester(func(c *Context) {
		Column(c).Fill().Children(func() {
			Column(c).Width(200).Height(60).OffsetAfterLayout(func(Rect) Point { return Point{X: shift, Y: shift} }).Children(func() {
				TextInput(c, &text).Label("Editor").Width(150).Height(30).AutoFocus()
			})
		})
	}, 400, 200)
	defer tt.rt.close()
	client := tt.NativeTextInputClient()
	if client == nil {
		t.Fatal("native text client absent")
	}
	before, _, ok := client.BoundsForRange(TextInputRange{})
	if !ok {
		t.Fatal("initial candidate geometry absent")
	}
	shift = 40
	tt.Frame()
	client = tt.NativeTextInputClient()
	after, _, ok := client.BoundsForRange(TextInputRange{})
	if !ok || after.X-before.X != 40 || after.Y-before.Y != 40 {
		t.Fatalf("native candidate geometry did not move with editor: before=%+v after=%+v ok=%v", before, after, ok)
	}
	if text != "hello" {
		t.Fatal("displacement changed resident draft")
	}
}
