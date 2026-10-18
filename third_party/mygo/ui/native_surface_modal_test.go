package ui

import "testing"

func TestNativeSurfaceForegroundModalTracksBuiltOverlay(t *testing.T) {
	rt := &engine{inFrame: true}
	c := &Context{rt: rt}
	if c.HasForegroundModal() {
		t.Fatal("empty frame suspended guest")
	}
	rt.c.overlay = &node{first: &node{first: &node{flags: flagModal}}}
	if !c.HasForegroundModal() {
		t.Fatal("newly built nested modal did not suspend guest")
	}
	rt.c.overlay = &node{first: &node{}}
	if c.HasForegroundModal() {
		t.Fatal("ordinary overlay suspended guest")
	}
	rt.inFrame = false
	if !c.HasForegroundModal() {
		t.Fatal("expired frame presented native guest")
	}
}
