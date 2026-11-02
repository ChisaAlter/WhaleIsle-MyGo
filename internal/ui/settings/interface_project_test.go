package settings

import (
	"github.com/egoist/mygo/ui"
	"testing"
	"whaleisle/internal/desktop/projectpreference"
	"whaleisle/internal/ui/i18n"
)

func TestInterfaceProjectLifecyclePendingAndAcceptedRollback(t *testing.T) {
	i18n.SetLocale("en")
	defer i18n.SetLocale(i18n.DefaultLocale)
	calls := []bool{}
	access := &ProjectPreferenceAccess{}
	access.OnSet = func(next bool) {
		calls = append(calls, next)
		access.State = projectpreference.View{Enabled: next, Busy: true}
	}
	tt := ui.NewTester(func(c *ui.Context) { interfaceProjectPreference(c, access, ui.RGB(0, 0, 0), ui.RGB(110, 110, 110)) }, 600, 500)
	for range 4 {
		tt.Key(0, ui.KeyTab)
		if tt.Focused("Project") {
			break
		}
	}
	tt.Key(0, ui.KeySpace)
	if len(calls) != 1 || !calls[0] || !access.State.Busy {
		t.Fatal("Project did not issue its own immediate callback")
	}
	for range 4 {
		tt.Key(0, ui.KeyTab)
		if tt.Focused("Project") {
			break
		}
	}
	tt.Key(0, ui.KeySpace)
	if len(calls) != 1 {
		t.Fatal("Project accepted a second input while confirmation/restart pending")
	}
	access.State = projectpreference.View{Enabled: false, Error: "Harness restart was cancelled; Project setting was kept", RestartCancelled: true}
	tt.Frame()
	if _, ok := tt.Find(access.State.Error); !ok {
		t.Fatal("cancelled restart lost visible feedback")
	}
	for range 4 {
		tt.Key(0, ui.KeyTab)
		if tt.Focused("Project") {
			break
		}
	}
	tt.Key(0, ui.KeySpace)
	if len(calls) != 2 || !calls[1] {
		t.Fatal("cancelled optimistic value was not restored before retry")
	}
	access.State = projectpreference.View{Enabled: true}
	tt.Frame()
	for range 4 {
		tt.Key(0, ui.KeyTab)
		if tt.Focused("Project") {
			break
		}
	}
	tt.Key(0, ui.KeySpace)
	if len(calls) != 3 || calls[2] {
		t.Fatal("accepted enabled value could not be disabled")
	}
}
