package main

import (
	"testing"
	"time"
)

// A withdrawn selected view gets one grace window while the bridge stays
// alive: an HMR rebuild re-registers inside it, a real disable does not.
func TestPluginViewMissingDecisionGraceThenClose(t *testing.T) {
	p := &nativePluginNavigationState{}
	t0 := time.Now()

	notify, closeView := pluginViewMissingDecision(true, p, t0)
	if notify || closeView {
		t.Fatal("first miss during a live bridge must stay quiet and open")
	}
	if p.missingSince.IsZero() {
		t.Fatal("first miss did not start the grace clock")
	}

	notify, closeView = pluginViewMissingDecision(true, p, t0.Add(2*time.Second))
	if notify || closeView {
		t.Fatal("miss inside the grace window must stay quiet and open")
	}

	notify, closeView = pluginViewMissingDecision(true, p, t0.Add(3*time.Second))
	if !notify || !closeView || !p.missingNotified {
		t.Fatal("miss past the grace window must notify once and close")
	}

	notify, closeView = pluginViewMissingDecision(true, p, t0.Add(4*time.Second))
	if notify || !closeView {
		t.Fatal("a second evaluation must close without a duplicate notice")
	}
}

func TestPluginViewMissingDecisionDeadBridgeClosesQuietly(t *testing.T) {
	p := &nativePluginNavigationState{}
	notify, closeView := pluginViewMissingDecision(false, p, time.Now())
	if notify || !closeView {
		t.Fatal("a dead bridge must close immediately without the HMR notice")
	}
	if !p.missingSince.IsZero() {
		t.Fatal("a dead bridge must not start the grace clock")
	}
}
