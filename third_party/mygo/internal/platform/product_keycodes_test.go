package platform

import "testing"

func TestProductPhysicalKeyMappings(t *testing.T) {
	for _, row := range []struct {
		mac, gdk uint16
		vk       uint32
	}{{0, 38, 'A'}, {12, 24, 'Q'}, {18, 10, '1'}, {29, 19, '0'}, {24, 21, 0xbb}, {39, 48, 0xde}, {48, 23, 9}, {49, 65, 32}, {123, 113, 37}, {122, 67, 112}, {111, 96, 123}, {90, 198, 131}} {
		if got := ProductMacVirtualKey(row.mac); got != row.vk {
			t.Fatalf("mac %d = %d want %d", row.mac, got, row.vk)
		}
		if got := ProductGDKVirtualKey(row.gdk); got != row.vk {
			t.Fatalf("GDK %d = %d want %d", row.gdk, got, row.vk)
		}
	}
	for _, code := range []uint16{54, 55, 56, 57, 58, 59, 60, 61, 62, 63, 65535} {
		if ProductMacVirtualKey(code) != 0 {
			t.Fatalf("mac modifier/unknown %d accepted", code)
		}
	}
	for _, code := range []uint16{37, 50, 62, 64, 108, 133, 134, 65535} {
		if ProductGDKVirtualKey(code) != 0 {
			t.Fatalf("GDK modifier/unknown %d accepted", code)
		}
	}
}
