package boot

import (
	"errors"
	"github.com/egoist/mygo/ui"
	"testing"
	"whaleisle/internal/harness"
	"whaleisle/internal/ui/i18n"
)

func TestBootSceneFailureAndLogDrawer(t *testing.T) {
	i18n.SetLocale("en")
	defer i18n.SetLocale(i18n.DefaultLocale)
	st := harness.Status{State: harness.Starting}
	retries := 0
	tt := ui.NewTester(func(c *ui.Context) {
		RecoveryControl(c, Model{Status: st, Logs: []string{"first evidence", "latest evidence"}, OnRetry: func() { retries++ }})
	}, 900, 600)
	if !tt.HasText(i18n.T(KeyStarting)) {
		t.Fatal("startup scene absent")
	}
	tt.Click("All startup logs")
	if !tt.HasText("first evidence") {
		t.Fatal("drawer lost earlier evidence")
	}
	tt.Key(0, ui.KeyEscape)
	if tt.HasText("first evidence") {
		t.Fatal("drawer did not dismiss")
	}
	st = harness.Status{State: harness.Failed, Err: errors.New("actual host failure")}
	tt.Frame()
	if !tt.HasText(i18n.T(KeyFailed) + ": actual host failure") {
		t.Fatal("real error hidden")
	}
	tt.Click(i18n.T(KeyRetry))
	if retries != 1 {
		t.Fatal("retry not dispatched")
	}
	st = harness.Status{State: harness.Starting}
	tt.Frame()
	tt.Click(i18n.T(KeyRetry))
	if retries != 1 {
		t.Fatal("retry active while starting")
	}
}
