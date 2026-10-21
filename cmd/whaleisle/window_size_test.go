package main

import "testing"

func TestFitWindowSizeClampsToSmallWorkArea(t *testing.T) {
	// 1707x960 screen at 150%: work area in DIP.
	w, h, minW, minH := fitWindowSize(1440, 920, 960, 640, 1138, 640)
	if w != 1090 || h != 592 || minW != 960 || minH != 592 {
		t.Fatalf("small work area = %dx%d min %dx%d", w, h, minW, minH)
	}
}

func TestFitWindowSizeLeavesLargeDisplaysAlone(t *testing.T) {
	w, h, minW, minH := fitWindowSize(1440, 920, 960, 640, 2560, 1440)
	if w != 1440 || h != 920 || minW != 960 || minH != 640 {
		t.Fatalf("large work area = %dx%d min %dx%d", w, h, minW, minH)
	}
}
