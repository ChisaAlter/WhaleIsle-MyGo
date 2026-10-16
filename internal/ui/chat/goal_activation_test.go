package chat

import (
	"github.com/egoist/mygo/ui"
	"testing"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
)

func TestGoalActivationControls(t *testing.T) {
	for _, tc := range []struct{ phase, activation, action string }{
		{"active", "disarmed", "resume"}, {"active", "armed", "pause"}, {"paused", "disarmed", "resume"}, {"blocked", "disarmed", ""}, {"active", "", ""}, {"complete", "disarmed", ""},
	} {
		t.Run(tc.phase+"_"+tc.activation, func(t *testing.T) {
			open := true
			called := ""
			goal := &session.PlanGoal{ID: "g", Objective: "Ship", Phase: tc.phase, Activation: tc.activation, BlockedReason: session.PlanGoalBlockedReason{Code: "limit", Message: "Round limit"}}
			tt := ui.NewTester(func(c *ui.Context) {
				PlanBoardView(c, PlanBoardModel{Board: session.PlanBoard{Goal: goal}, Open: &open, OnPauseGoal: func() { called = "pause" }, OnResumeGoal: func() { called = "resume" }})
			}, 640, 420)
			pause, resume := i18n.T(i18n.KeyPlanBoardGoalPause), i18n.T(i18n.KeyPlanBoardGoalResume)
			if tt.HasText(pause) != (tc.action == "pause") || tt.HasText(resume) != (tc.action == "resume") {
				t.Fatalf("wrong controls: %q", tt.Texts())
			}
			if !tt.HasText("[limit] Round limit") {
				t.Fatalf("missing blocker: %q", tt.Texts())
			}
			if tc.action != "" {
				label := resume
				if tc.action == "pause" {
					label = pause
				}
				if err := tt.Click(label); err != nil {
					t.Fatal(err)
				}
				if called != tc.action {
					t.Fatalf("callback=%q", called)
				}
			}
		})
	}
}
