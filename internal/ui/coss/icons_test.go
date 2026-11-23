package coss

import "testing"

func TestIconsParse(t *testing.T) {
	for i := Icon(1); i < iconCount; i++ {
		if i.SVG() == nil {
			t.Fatalf("icon %d did not parse", i)
		}
	}
}
