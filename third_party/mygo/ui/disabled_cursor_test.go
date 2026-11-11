package ui

import "testing"

func TestDisabledCursorIsNotAllowed(t *testing.T) {
	disabled := false
	tt := coreNewTester(func(c *context) {
		coreColumn(c).Padding(20).Children(func() {
			coreButton(c, "Save").Disabled(disabled)
		})
	}, 200, 100)
	b, _ := tt.Find("Save")
	tt.Move(b.X+b.W/2, b.Y+b.H/2)
	if tt.Cursor() != CursorPointer {
		t.Fatalf("enabled control cursor = %v", tt.Cursor())
	}
	disabled = true
	tt.Frame()
	tt.Move(b.X+b.W/2, b.Y+b.H/2)
	if tt.Cursor() != CursorNotAllowed {
		t.Fatalf("disabled control cursor = %v, want not-allowed", tt.Cursor())
	}
}
