package harness

import (
	"errors"
	"testing"
)

// A process that fails after Ready is the native snapshot's phase="runtime";
// one that never got there is a start failure.
func TestFailRecordsRuntimeFailurePhase(t *testing.T) {
	crash := New(Options{})
	crash.gen = 1
	crash.setLocked(Status{State: Ready})
	crash.fail(1, errors.New("dsh web exited: exit status 1"), "fixture")
	if st := crash.Status(); st.State != Failed || !st.RuntimeFailure {
		t.Fatalf("ready exit: %+v", st)
	}

	start := New(Options{})
	start.gen = 1
	start.setLocked(Status{State: Starting})
	start.fail(1, errors.New("dsh web exited: exit status 1"), "fixture")
	if st := start.Status(); st.State != Failed || st.RuntimeFailure {
		t.Fatalf("start failure: %+v", st)
	}
}
