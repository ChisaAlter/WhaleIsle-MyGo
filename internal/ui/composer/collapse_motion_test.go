package composer

import (
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

func testCollapseGuards() ComposerCollapseGuards {
	return ComposerCollapseGuards{
		ActivePhase: true,
	}
}

func TestReduceComposerScrollThresholdsAndPinning(t *testing.T) {
	guards := testCollapseGuards()
	s := ComposerCollapseState{Mode: ComposerCollapseExpanded}

	// Short viewport below collapse threshold stays expanded.
	s = ReduceComposerScroll(s, ComposerScrollSample{
		ScrollTop: 10, ScrollHeight: 540, ClientHeight: 500,
		PrevScrollTop: 20, PrevScrollHeight: 540,
	}, guards)
	if s.Mode != ComposerCollapseExpanded {
		t.Fatalf("short viewport collapsed: %+v", s)
	}

	// Scrolling up past 64 DIP from bottom collapses to capsule mode.
	s = ReduceComposerScroll(s, ComposerScrollSample{
		ScrollTop: 100, ScrollHeight: 1000, ClientHeight: 500, // maxScroll=500, dist=400 >= 64
		PrevScrollTop: 200, PrevScrollHeight: 1000,
	}, guards)
	if !IsComposerCollapsed(s, guards) {
		t.Fatalf("expected collapse when scrolled up >=64 DIP from bottom, got %+v", s)
	}

	// Explicit click/focus while scrolled up pins expanded at current ScrollTop.
	s = PinComposerExpanded(100)
	if IsComposerCollapsed(s, guards) || s.Mode != ComposerCollapsePinnedExpanded {
		t.Fatalf("expected pinned-expanded, got %+v", s)
	}

	// Small scroll (<32 DIP unpin delta) stays pinned expanded.
	s = ReduceComposerScroll(s, ComposerScrollSample{
		ScrollTop: 120, ScrollHeight: 1000, ClientHeight: 500,
		PrevScrollTop: 100, PrevScrollHeight: 1000,
	}, guards)
	if s.Mode != ComposerCollapsePinnedExpanded {
		t.Fatalf("small scroll <32 DIP unpinned unexpectedly: %+v", s)
	}

	// Scrolling >=32 DIP away while >=64 DIP from bottom unpins back to collapsed.
	s = ReduceComposerScroll(s, ComposerScrollSample{
		ScrollTop: 132, ScrollHeight: 1000, ClientHeight: 500,
		PrevScrollTop: 120, PrevScrollHeight: 1000,
	}, guards)
	if !IsComposerCollapsed(s, guards) {
		t.Fatalf("scroll >=32 DIP should re-collapse, got %+v", s)
	}

	// Returning within <=24 DIP of bottom expands again.
	s = ReduceComposerScroll(s, ComposerScrollSample{
		ScrollTop: 476, ScrollHeight: 1000, ClientHeight: 500, // dist=24 <= 24
		PrevScrollTop: 132, PrevScrollHeight: 1000,
	}, guards)
	if s.Mode != ComposerCollapseExpanded || IsComposerCollapsed(s, guards) {
		t.Fatalf("within 24 DIP of bottom should expand, got %+v", s)
	}
}

func TestComposerCollapseGuardsPreventCollapse(t *testing.T) {
	collapsed := ComposerCollapseState{Mode: ComposerCollapseCollapsed}
	base := testCollapseGuards()
	cases := []struct {
		name  string
		guard func(ComposerCollapseGuards) ComposerCollapseGuards
	}{
		{"inactivePhase", func(g ComposerCollapseGuards) ComposerCollapseGuards { g.ActivePhase = false; return g }},
		{"overlay", func(g ComposerCollapseGuards) ComposerCollapseGuards { g.HasOverlay = true; return g }},
		{"pendingInteraction", func(g ComposerCollapseGuards) ComposerCollapseGuards { g.HasPendingInteraction = true; return g }},
		{"multilineOrAttachment", func(g ComposerCollapseGuards) ComposerCollapseGuards { g.HasMultilineOrAttachment = true; return g }},
		{"editSession", func(g ComposerCollapseGuards) ComposerCollapseGuards { g.HasEditSession = true; return g }},
		{"openPopover", func(g ComposerCollapseGuards) ComposerCollapseGuards { g.HasOpenPopover = true; return g }},
	}
	for _, tc := range cases {
		g := tc.guard(base)
		if IsComposerCollapsed(collapsed, g) {
			t.Fatalf("guard %s allowed collapsed rendering", tc.name)
		}
		next := ReduceComposerScroll(collapsed, ComposerScrollSample{
			ScrollTop: 100, ScrollHeight: 1000, ClientHeight: 500,
			PrevScrollTop: 200, PrevScrollHeight: 1000,
		}, g)
		if next.Mode != ComposerCollapseExpanded {
			t.Fatalf("guard %s did not reset collapsed state to expanded: %+v", tc.name, next)
		}
	}
}

func TestCollapseProgressTimingAndReduceMotion(t *testing.T) {
	if ComposerCollapseDuration != 200*time.Millisecond {
		t.Fatalf("collapse duration = %v, want 200ms", ComposerCollapseDuration)
	}
	collapsed := false
	var got float32
	start := time.Unix(1000, 0)
	tt := ui.NewTester(func(c *ui.Context) {
		host := ui.Box(c).Key("host").Fill()
		got = CollapseProgress(c, host, collapsed)
	}, 100, 100)
	tt.SetTime(start)
	if got != 0 {
		t.Fatalf("initial progress = %v, want 0", got)
	}
	collapsed = true
	tt.SetTime(start.Add(time.Millisecond))
	tt.SetTime(start.Add(100 * time.Millisecond))
	if got <= 0 || got >= 1 {
		t.Fatalf("mid-transition progress = %v, want in (0,1)", got)
	}
	tt.SetTime(start.Add(ComposerCollapseDuration + 2*time.Millisecond))
	if got != 1 {
		t.Fatalf("settled progress = %v, want 1", got)
	}
	// ReduceMotion snaps immediately.
	tt.SetPreferences(ui.Preferences{ReduceMotion: true, TextScale: 1})
	collapsed = false
	tt.SetTime(start.Add(300 * time.Millisecond))
	if got != 0 {
		t.Fatalf("reduce-motion progress = %v, want 0 immediately", got)
	}
}
