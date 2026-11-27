package coss

import "testing"

func TestIconsParse(t *testing.T) {
	for i := Icon(1); i < iconCount; i++ {
		if i.SVG() == nil {
			t.Fatalf("icon %d did not parse", i)
		}
		wantStroke := "1.5"
		if i == IconCheckboxCheck || i == IconCheckboxMinus {
			wantStroke = "2.2"
		}
		if got := i.stroke(); got != wantStroke {
			t.Fatalf("icon %d stroke = %q, want %q", i, got, wantStroke)
		}
	}
}
