package dsh

import "testing"

func TestNativeModelLabelRevisionTracksRendererClosure(t *testing.T) {
	makeGraph := func() nativeTestGraph {
		g := nativeTestFullGraph()
		g.entry(nativeLabel1017).Inject = []string{"test-inject/client"}
		g.entry(nativeLabel1017).External = []string{"test-external/client", "react"}
		return g
	}
	snapshot := func(g nativeTestGraph) NativeContributions {
		t.Helper()
		var value NativeContributions
		w := nativeContributionWatch{publish: func(n NativeContributions) { value = n }}
		if err := w.receive([]byte(nativeTestJSON(map[string]any{"type": "graph", "graph": g}))); err != nil {
			t.Fatal(err)
		}
		if !value.Known || !value.ModelLabel1017 || value.ModelLabelRevision() == "" {
			t.Fatal("valid renderer graph has no retained owner identity")
		}
		return value
	}
	baseline := snapshot(makeGraph())
	for _, id := range []string{nativeLabel1017, "test-leaf", "test-external"} {
		t.Run(id, func(t *testing.T) {
			g := makeGraph()
			g.entry(id).Rev = "new-renderer-code"
			if snapshot(g).ModelLabelRevision() == baseline.ModelLabelRevision() {
				t.Fatal("renderer or dependency reload reused the old presentation owner")
			}
		})
	}
	g := makeGraph()
	g.Rev = "delivery-only"
	g.entry(nativeFeedback).Rev = "unrelated-code"
	for left, right := 0, len(g.Entries)-1; left < right; left, right = left+1, right-1 {
		g.Entries[left], g.Entries[right] = g.Entries[right], g.Entries[left]
	}
	g.rebatch()
	if snapshot(g).ModelLabelRevision() != baseline.ModelLabelRevision() {
		t.Fatal("unrelated code or delivery reorder retired the label owner")
	}
	baseline.ModelLabel1017 = false
	if baseline.ModelLabelRevision() != "" {
		t.Fatal("withdrawn renderer retained its identity")
	}
}
