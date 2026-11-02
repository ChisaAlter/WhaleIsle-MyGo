package sidebar

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
)

func TestBatchKnownBlankIsInertButMissingArchivedSummarySelectable(t *testing.T) {
	i18n.SetLocale("en")
	t.Cleanup(func() { i18n.SetLocale(i18n.DefaultLocale) })
	blank := session.Item{SessionID: "blank", Title: "do not select", Blank: true}
	expanded := true
	calls, navigated := 0, 0
	m := Model{Batch: session.SessionBatch{Enabled: true}, archiveExpanded: &expanded, Snapshot: session.ViewState{WorkspaceReady: true, SelectedID: "blank", Items: []session.Item{blank}, ArchivedSessionIDs: []string{"missing"}, Workspaces: []session.WorkspaceRow{{WorkspaceID: "w", SessionIDs: []string{"blank"}}}}, OnBatchEdit: func(string, []session.SessionBatchTarget) { calls++ }, OnSelect: func(string) { navigated++ }, OnOpenSessionPlugins: func(string) { navigated++ }}
	if targets := VisibleBatchTargets(m); len(targets) != 1 || targets[0].ID != "missing" {
		t.Fatalf("known blank/missing archived summary eligibility: %+v", targets)
	}
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Width(280).Children(func() { rowSession(c, blank, 0, m, c.Theme().Text, c.Theme().TextMuted, c.Theme().Border) })
	}, 400, 200)
	r, ok := tt.Find("New session")
	if !ok || r.H != 32 {
		t.Fatalf("inert new-session row missing or wrong height: %+v/%t", r, ok)
	}
	if err := tt.Click("New session"); err != nil {
		t.Fatal(err)
	}
	tt.Key(0, ui.KeyEnter)
	if calls != 0 || navigated != 0 || tt.HasText("Select checkbox blank") || tt.HasText(sessionActionsLabel("blank")) {
		t.Fatalf("blank batch row remained interactive: calls=%d nav=%d texts=%v", calls, navigated, tt.Texts())
	}
}

func TestBatchRealCheckboxAndRowDispatchOnceAndArchivedBodyStaysInert(t *testing.T) {
	for _, archived := range []bool{false, true} {
		t.Run(map[bool]string{false: "ordinary", true: "archived-summary"}[archived], func(t *testing.T) {
			calls := 0
			m := Model{Batch: session.SessionBatch{Enabled: true, Selected: map[string]string{}}, OnBatchEdit: func(action string, targets []session.SessionBatchTarget) {
				if action != "select" || len(targets) != 1 || targets[0].ID != "id" {
					t.Fatalf("wrong batch callback %s/%v", action, targets)
				}
				calls++
			}}
			if archived {
				m.Snapshot.ArchivedSessionIDs = []string{"id"}
			}
			tt := ui.NewTester(func(c *ui.Context) {
				ui.Column(c).Width(280).Children(func() {
					if archived {
						rowArchivedSession(c, "id", "Summary", 0, m, c.Theme().TextMuted)
					} else {
						rowSession(c, session.Item{SessionID: "id", Title: "Summary"}, 0, m, c.Theme().Text, c.Theme().TextMuted, c.Theme().Border)
					}
				})
			}, 400, 180)
			check, ok := tt.Find("Select checkbox id")
			if !ok || check.W != 16 || check.H != 16 {
				t.Fatalf("real checkbox bounds=%+v present=%t", check, ok)
			}
			if err := tt.Click("Select checkbox id"); err != nil {
				t.Fatal(err)
			}
			if calls != 1 {
				t.Fatalf("checkbox toggled %d times", calls)
			}
			row, ok := tt.Find("Select session id")
			if !ok || row.H != 32 {
				t.Fatalf("batch row geometry %+v/%t", row, ok)
			}
			tt.ClickAt(row.X+80, row.Y+row.H/2)
			want := 2
			if archived {
				want = 1
			}
			if calls != want {
				t.Fatalf("body calls=%d want=%d archived=%t", calls, want, archived)
			}
			m.Batch.Busy = true
			tt.Frame()
			_ = tt.Click("Select checkbox id")
			if calls != want {
				t.Fatal("busy row changed selection")
			}
		})
	}
}

func TestBatchRootDialogCancelFocusFrozenTargetsAndBusyIntent(t *testing.T) {
	i18n.SetLocale("en")
	t.Cleanup(func() { i18n.SetLocale(i18n.DefaultLocale) })
	confirms, cancels := 0, 0
	m := Model{Collapsed: true, Batch: session.SessionBatch{Enabled: true, Confirming: true, Kind: "delete", Token: 31, Selected: map[string]string{"current": "Current choices"}, Targets: []session.SessionBatchTarget{{ID: "frozen", Title: "Frozen target"}}}, OnBatchCancel: func(token uint64) {
		if token != 31 {
			t.Fatal(token)
		}
		cancels++
	}, OnBatchConfirm: func(token uint64) {
		if token != 31 {
			t.Fatal(token)
		}
		confirms++
	}}
	tt := ui.NewTester(func(c *ui.Context) { View(c, m) }, 800, 600)
	tt.SetPreferences(ui.Preferences{ReduceMotion: true})
	tt.Frame()
	if !tt.Focused("Cancel batch") || !tt.HasText("Frozen target · frozen") || tt.HasText("Current choices") {
		t.Fatalf("root modal lost focus or frozen target contract: %v", tt.Texts())
	}
	tt.Key(0, ui.KeyEnter)
	if cancels != 1 || confirms != 0 {
		t.Fatalf("initial Enter committed destruction: cancel=%d confirm=%d", cancels, confirms)
	}
	m.Batch.Busy = true
	m.Batch.Confirming = false
	tt.Frame()
	tt.Key(0, ui.KeyEscape)
	_ = tt.Click("Cancel batch")
	tt.Key(0, ui.KeyEnter)
	if cancels != 1 || confirms != 0 {
		t.Fatal("busy modal accepted cancellation or confirm")
	}
	if !tt.HasText("Frozen target · frozen") {
		t.Fatal("busy modal lost frozen targets")
	}
	m.Batch.Busy = false
	m.Batch.Confirming = true
	tt.Frame()
	if err := tt.Click("Confirm batch"); err != nil {
		t.Fatal(err)
	}
	if confirms != 1 {
		t.Fatalf("explicit confirm callbacks=%d", confirms)
	}
}
