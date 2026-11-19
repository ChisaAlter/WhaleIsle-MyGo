package main

import "testing"

func TestOnlyAPersonInitiatedLaunchIsActivatable(t *testing.T) {
	cases := []struct {
		name                               string
		background, isolated, nonActivable bool
	}{
		{name: "direct launch", background: false, isolated: false, nonActivable: false},
		{name: "background launch", background: true, isolated: false, nonActivable: true},
		{name: "isolated desktop run", background: false, isolated: true, nonActivable: true},
		{name: "background and isolated", background: true, isolated: true, nonActivable: true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := windowNonActivatable(c.background, c.isolated); got != c.nonActivable {
				t.Fatalf("windowNonActivatable(%v, %v) = %v, want %v", c.background, c.isolated, got, c.nonActivable)
			}
		})
	}
}
