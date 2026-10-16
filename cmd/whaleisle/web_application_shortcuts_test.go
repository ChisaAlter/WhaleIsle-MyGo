package main

import (
	"testing"
	"whaleisle/internal/shortcuts"
)

func TestWebApplicationFallbacks(t *testing.T) {
	bindings := shortcuts.Resolve(nil)
	var calls []string
	rows := webApplicationFallbacks(bindings, func(id string) { calls = append(calls, id) })
	for _, row := range rows {
		row.Run()
	}
	seen := map[string]bool{}
	for _, id := range calls {
		seen[id] = true
	}
	if !seen["settings.open"] || !seen["shortcuts.open"] || !seen["launcher.open"] || !seen["market.open"] {
		t.Fatalf("missing page fallback: %v", calls)
	}
	if seen["harness.restart"] || seen["session.new"] {
		t.Fatalf("browser/reserved/mutating command forwarded: %v", calls)
	}
	for _, c := range []shortcuts.Chord{{Key: "P"}, {Key: "P", Shift: true}, {Key: "F"}, {Key: "O", Shift: true}} {
		if !browserOwnsChord(c) {
			t.Fatal("browser chord stolen", c)
		}
	}
}
