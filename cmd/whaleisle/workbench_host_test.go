package main

import (
	"errors"
	"testing"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/harness"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/frame"
	"whaleisle/internal/ui/i18n"
)

func TestHostFailureKeepsWorkbenchMounted(t *testing.T) {
	i18n.SetLocale("en")
	defer i18n.SetLocale(i18n.DefaultLocale)
	s := &mainState{}
	store, sizes := session.New(), frame.DefaultLayout()
	st := harness.Status{State: harness.Failed, Err: errors.New("workspace unavailable")}
	tt := ui.NewTester(func(c *ui.Context) {
		frame.View(c, frame.Model{
			Store: store, Sizes: &sizes,
			HostStatus: func(c *ui.Context) { s.workbenchHostStatus(c, nil, st, nil, nil) },
		})
	}, 1440, 900)
	for _, label := range []string{"New session", "workspace unavailable", "Export log"} {
		if !tt.HasText(label) {
			t.Fatalf("workbench lost %q with Host unavailable", label)
		}
	}
	if err := tt.Click("Startup diagnostics"); err != nil {
		t.Fatal(err)
	}
	if s.mainRoute != routeStartupDiagnostics {
		t.Fatalf("startup diagnostics did not select the in-window route: %v", s.mainRoute)
	}
	st = harness.Status{State: harness.Ready}
	tt.Frame()
	if tt.HasText("workspace unavailable") || tt.HasText("Retry") || !tt.HasText("New session") {
		t.Fatal("healthy workbench retained startup controls or lost navigation")
	}
}
