package chrome

import "testing"

// TestFlatBackdropFill pins the masked flat canvas: the dark base token
// rgb(21,21,23) reads ~12,12,12 after the native rgba(0,0,0,0.5) mask, while
// the light half keeps its base colour.
func TestFlatBackdropFill(t *testing.T) {
	dark := Color("#151517")
	got := FlatBackdropFill(dark, true)
	// rgba(0,0,0,0.5) over rgb(21,21,23) = (10.5,10.5,11.5).
	if got.A != 255 {
		t.Fatalf("flat fill must be opaque, alpha = %v", got.A)
	}
	if got.R < 10 || got.R > 12 || got.B < 11 || got.B > 13 {
		t.Fatalf("dark flat fill = %v, want ~(11,11,12)", got)
	}
	light := Color("#ffffff")
	if Fill := FlatBackdropFill(light, false); Fill != light {
		t.Fatalf("light flat fill changed: %v", Fill)
	}
}
