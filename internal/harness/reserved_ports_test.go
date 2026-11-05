package harness

import (
	"net"
	"testing"
)

func TestFreePortLeavesReservedCompanionAvailable(t *testing.T) {
	probe, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	start := probe.Addr().(*net.TCPAddr).Port
	probe.Close()
	legacy, err := FreePort("127.0.0.1", start)
	if err != nil || legacy != start {
		t.Fatalf("ordinary first-free contract: port=%d err=%v", legacy, err)
	}
	selected, err := freePort("127.0.0.1", start, []int{start})
	if err != nil || selected <= start {
		t.Fatalf("reserved companion was selected: port=%d err=%v", selected, err)
	}
	companion, err := net.Listen("tcp", probe.Addr().String())
	if err != nil {
		t.Fatalf("companion cannot bind its reserved port: %v", err)
	}
	defer companion.Close()
}

func TestFreePortKeepsScanBoundWithReservedPorts(t *testing.T) {
	reserved := make([]int, portScanRange)
	for i := range reserved {
		reserved[i] = 40000 + i
	}
	if port, err := freePort("127.0.0.1", 40000, reserved); err == nil || port != 0 {
		t.Fatalf("reserved scan escaped its original range: port=%d err=%v", port, err)
	}
}
