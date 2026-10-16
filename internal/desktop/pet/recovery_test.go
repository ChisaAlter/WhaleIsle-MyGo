package pet

import "testing"

func TestPetRecoveryWindowOwnership(t *testing.T) {
	for mask := 0; mask < 16; mask++ {
		same, ready, enabled, destroyed := mask&1 != 0, mask&2 != 0, mask&4 != 0, mask&8 != 0
		if got := petRecoveryCurrent(same, ready, enabled, destroyed); got != (mask == 7) {
			t.Fatalf("state %04b: recovery=%v", mask, got)
		}
	}
}
