//go:build windows && (amd64 || arm64)

package previewprocess

import "testing"

func TestProfileDeletionCompletesOnlySharingViolation(t *testing.T) {
	for _, test := range []struct {
		name              string
		first, next, want uintptr
		calls             int
	}{
		{"completed", 0, 0x80070005, 0, 0},
		{"denied", 0x80070005, 0, 0x80070005, 0},
		{"partial-delete", 0x80070020, 0, 0, 1},
		{"still-in-use", 0x80070020, 0x80070020, 0x80070020, 1},
	} {
		t.Run(test.name, func(t *testing.T) {
			calls := 0
			got := finishProfileDeletion(test.first, func() uintptr { calls++; return test.next })
			if got != test.want || calls != test.calls {
				t.Fatalf("delete result=%08x calls=%d, want %08x/%d", got, calls, test.want, test.calls)
			}
		})
	}
}
