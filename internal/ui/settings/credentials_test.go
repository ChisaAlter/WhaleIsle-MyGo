package settings

import "testing"

func TestMaskSecret(t *testing.T) {
	if maskSecret("") != "(empty)" {
		t.Fatal("empty")
	}
	if maskSecret("ab") != "••••" {
		t.Fatal("short")
	}
	if got := maskSecret("abcdefgh"); got != "ab••••gh" {
		t.Fatalf("got %q", got)
	}
}
