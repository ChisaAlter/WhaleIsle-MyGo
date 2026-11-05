package webview

import (
	"github.com/egoist/mygo"
	"testing"
)

func TestPageFailureCapabilityFollowsNavigationAndHostLifetime(t *testing.T) {
	var failures []PageFailure
	h := &Host{win: &mygo.Window{}, navigationEpoch: 7, failure: func(f PageFailure) { failures = append(failures, f) }}
	h.reportFailure("load", "engine code 1")
	h.reportFailure("process", "same failed navigation")
	if len(failures) != 1 || !failures[0].Current() {
		t.Fatal("failure was duplicated or recovery was not current")
	}
	h.navigationEpoch++
	if failures[0].Current() {
		t.Fatal("old recovery could control a new navigation")
	}
	h.reportFailure("process", "engine exit")
	if len(failures) != 2 || !failures[1].Current() {
		t.Fatal("new navigation failure was hidden")
	}
	h.win = nil
	if failures[1].Current() {
		t.Fatal("closed Host retained a recovery capability")
	}
}
