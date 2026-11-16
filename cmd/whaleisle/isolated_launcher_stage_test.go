package main

import (
	"testing"

	launcherctl "whaleisle/internal/app/launcher"
)

// Automatic launcher sync must not reveal an explicitly hidden QA desktop.
// The fake stage records only requested transitions; it never creates a window.
func TestIsolatedLauncherStageHonorsHiddenPolicy(t *testing.T) {
	for _, mode := range []struct {
		name, isolated, visible string
		hidden                  bool
	}{
		{"hidden-default", "1", "", true},
		{"hidden-zero", "1", "0", true},
		{"visible-isolated", "1", "1", false},
		{"normal", "", "", false},
		{"normal-visible-ignored", "", "1", false},
	} {
		t.Run(mode.name, func(t *testing.T) {
			t.Setenv("WHALEISLE_ISOLATED", mode.isolated)
			t.Setenv("WHALEISLE_ISOLATED_VISIBLE", mode.visible)
			for _, transition := range []struct {
				name            string
				open, available bool
				want            launcherctl.StageResult
			}{
				{"closed-panel", false, false, launcherctl.StageResult{MainShown: true}},
				{"open-panel", true, false, launcherctl.StageResult{MainShown: true}},
				{"open-launcher", true, true, launcherctl.StageResult{LauncherShown: true}},
				{"finish-launcher", false, true, launcherctl.StageResult{LauncherClosed: true, MainShown: true}},
			} {
				t.Run(transition.name, func(t *testing.T) {
					stage := &fakeLauncherStage{
						windowAvailable: transition.available,
						haveLauncher:    transition.available && !transition.open,
					}
					originalLauncher := stage.haveLauncher
					got := launcherctl.ApplyStage(stage, transition.open)
					if mode.hidden {
						if got != (launcherctl.StageResult{}) || len(stage.events) != 0 || stage.haveLauncher != originalLauncher || stage.mainShown || stage.created != 0 || stage.closed != 0 || stage.shows != 0 {
							t.Fatalf("hidden isolation requested a window transition: result=%+v stage=%+v", got, stage)
						}
						return
					}
					if got != transition.want {
						t.Fatalf("visible launcher transition=%+v, want %+v", got, transition.want)
					}
				})
			}
		})
	}
}
