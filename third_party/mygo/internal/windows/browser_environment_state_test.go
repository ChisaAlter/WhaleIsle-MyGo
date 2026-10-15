package windows

import "testing"

func TestBrowserEnvironmentOwners(t *testing.T) {
	var current browserEnvironmentOwners[int]
	if !current.add(1) || !current.add(2) || len(current.members) != 2 {
		t.Fatal("two Browser windows must share one live generation")
	}
	if current.remove(99) || current.remove(1) || current.closed {
		t.Fatal("missing or non-final owner must not release the environment")
	}
	if !current.remove(2) || !current.closed || len(current.members) != 0 {
		t.Fatal("last window must retire the environment even before creation completes")
	}
	if current.remove(2) || current.add(3) {
		t.Fatal("retired generation must not release twice or accept stale/new owners")
	}
	var reopened browserEnvironmentOwners[int]
	if !reopened.add(3) || reopened.closed || !current.closed {
		t.Fatal("reopen must use a separate generation without reviving an old callback")
	}
}
