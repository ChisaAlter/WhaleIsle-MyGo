package windows

import "testing"

func TestBrowserSiteDataExactKinds(t *testing.T) {
	// Fixed official SDK values, not aliases for inclusive "all" categories.
	if browserSiteDataKinds != 0x804e {
		t.Fatalf("unexpected five-kind mask: %#x", browserSiteDataKinds)
	}
	for _, kind := range []int{0x2, 0x4, 0x8, 0x40, 0x8000} {
		if browserSiteDataKinds&kind != kind {
			t.Fatalf("required site-data kind missing: %#x", kind)
		}
	}
	// File systems, CacheStorage, inclusive groups, HTTP disk cache, history,
	// autofill, passwords and settings must never enter this cleanup request.
	for _, excluded := range []int{0x1, 0x10, 0x20, 0x80, 0x100, 0x200, 0x400, 0x800, 0x1000, 0x2000, 0x4000} {
		if browserSiteDataKinds&excluded != 0 {
			t.Fatalf("cleanup exceeds its five-kind scope: %#x", excluded)
		}
	}
}
