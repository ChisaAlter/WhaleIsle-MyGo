package windows

import "testing"

func TestProductPhysicalKeyScanCodes(t *testing.T) {
	for _, test := range []struct {
		scan     uint32
		extended bool
		want     uint32
	}{{0x10, false, 'Q'}, {0x15, false, 'Y'}, {0x2c, false, 'Z'}, {0x27, false, 186}, {0x02, false, '1'}, {0x0b, false, '0'}, {0x48, true, 38}, {0x48, false, 0}, {0x3b, false, 112}, {0x76, false, 135}, {0x70, false, 0}, {0, false, 0}} {
		param := test.scan << 16
		if test.extended {
			param |= 1 << 24
		}
		if got := productPhysicalVK(param); got != test.want {
			t.Fatalf("scan %x extended %v: got %d want %d", test.scan, test.extended, got, test.want)
		}
	}
}
