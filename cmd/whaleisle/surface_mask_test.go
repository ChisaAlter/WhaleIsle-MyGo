package main

import "testing"

func TestExclusiveSurfaceMaskHidesOverlappingSurfaces(t *testing.T) {
	s := &mainState{settingsOpen: true}
	mask := s.exclusiveSurfaceMaskLocked()
	if !mask.modalMasked() || !mask.shellOverlayMasked() || !mask.nativeOverlayMasked() {
		t.Fatalf("settingsOpen did not mask overlapping surfaces: %+v", mask)
	}

	s = &mainState{webviewOpen: true}
	mask = s.exclusiveSurfaceMaskLocked()
	if !mask.shellOverlayMasked() {
		t.Fatalf("webviewOpen did not mask shell overlays: %+v", mask)
	}
	if mask.nativeOverlayMasked() {
		t.Fatalf("webviewOpen alone must not trigger native overlay hiding: %+v", mask)
	}

	s = &mainState{confirmAsk: &confirmAskRequest{}}
	mask = s.exclusiveSurfaceMaskLocked()
	if !mask.nativeOverlayMasked() || !mask.shellOverlayMasked() {
		t.Fatalf("confirmAsk did not mask native and shell surfaces: %+v", mask)
	}

	s = &mainState{marketOpen: true}
	mask = s.exclusiveSurfaceMaskLocked()
	if !mask.nativeOverlayMasked() {
		t.Fatalf("marketOpen did not mask native surfaces: %+v", mask)
	}
	if mask.shellOverlayMasked() {
		t.Fatalf("marketOpen alone must not mask shell overlays: %+v", mask)
	}
}
