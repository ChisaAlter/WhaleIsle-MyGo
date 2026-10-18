package taskguard

import (
	"errors"
	"testing"
)

func TestCloseAndActionDecisions(t *testing.T) {
	for _, tt := range []struct{ quitting, tray, want bool }{{false, true, true}, {false, false, false}, {true, true, false}, {true, false, false}} {
		if got := HideOnClose(tt.quitting, tt.tray); got != tt.want {
			t.Fatalf("HideOnClose(%v,%v)=%v", tt.quitting, tt.tray, got)
		}
	}
	for _, tt := range []struct {
		name    string
		running int
		choice  bool
		err     error
		want    bool
		calls   int
	}{
		{"idle does not prompt", 0, false, nil, true, 0},
		{"active cancel", 2, false, nil, false, 1},
		{"active explicit consent", 2, true, nil, true, 1},
		{"dialog failure blocks", 2, true, errors.New("dialog unavailable"), false, 1},
	} {
		t.Run(tt.name, func(t *testing.T) {
			calls := 0
			got, err := ConfirmRunning(tt.running, func() (bool, error) { calls++; return tt.choice, tt.err })
			if got != tt.want || calls != tt.calls || !errors.Is(err, tt.err) {
				t.Fatalf("got %v,%v calls=%d", got, err, calls)
			}
		})
	}
}
