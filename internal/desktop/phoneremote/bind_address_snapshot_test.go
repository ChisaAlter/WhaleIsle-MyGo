package phoneremote

import "testing"

func TestRuntimeBindAddressSnapshotKeepsUnknownAndObservedEmptyDistinct(t *testing.T) {
	r := NewRuntime()
	child := &fixtureChild{}
	r.child = child // No process, pipes, Start or Stop: only the existing decoder owner.
	r.consume(child, []byte(`{"kind":"snapshot","listening":false}`))
	if r.Snapshot().BindAddresses != nil {
		t.Fatal("omitted address scan became an observed empty scan")
	}
	r.consume(child, []byte(`{"kind":"snapshot","listening":false,"addresses":[]}`))
	empty := r.Snapshot().BindAddresses
	if empty == nil || len(empty) != 0 {
		t.Fatal("observed empty scan was lost while cloning", empty)
	}
	r.consume(child, []byte(`{"kind":"snapshot","listening":false,"addresses":["10.0.0.2"]}`))
	first := r.Snapshot()
	first.BindAddresses[0] = "mutated outside runtime"
	if got := r.Snapshot().BindAddresses; len(got) != 1 || got[0] != "10.0.0.2" {
		t.Fatal("snapshot exposed mutable runtime address state", got)
	}
	r.consume(&fixtureChild{}, []byte(`{"kind":"snapshot","addresses":[]}`))
	if got := r.Snapshot().BindAddresses; len(got) != 1 || got[0] != "10.0.0.2" {
		t.Fatal("retired runtime child replaced current NIC state", got)
	}
}
