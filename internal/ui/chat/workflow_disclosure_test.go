package chat

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
	"whaleisle/internal/workflowrun"
)

// Drive the production renderer and its actual focus tree. Merely testing the
// disclosure reducer would miss a completed member button losing native focus.
func TestWorkflowDisclosureCompletionWaitsForContentFocus(t *testing.T) {
	for _, departure := range []string{"outside", "header-toggle"} {
		t.Run(departure, func(t *testing.T) {
			phase := "Build phase"
			run := workflowrun.Run{ID: "run", Name: "Build", Status: "running", Phases: []workflowrun.Phase{{Key: "phase", Name: &phase, Members: []workflowrun.Member{{Seq: 1, ChildID: "child", Label: "focused live member", Status: "running"}}}}}
			model := Model{Snapshot: session.ViewState{SelectedID: "parent", PlanHistoryBinding: "owned-binding", Items: []session.Item{{SessionID: "child", ParentID: "parent", Running: true}}}, FeedbackStore: session.New()}
			// The transport-free Store has no binding/client, so the member
			// action can acquire focus but cannot start an observer or request.
			tt := ui.NewTester(func(c *ui.Context) {
				ui.Column(c).FillWidth().Children(func() {
					WorkflowRunView(c, run, model, ToolCardStyle{FG: c.Theme().Text, Muted: c.Theme().TextMuted})
					ui.Button(c, "Outside workflow")
				})
			}, 900, 600)
			if err := tt.Click("查看过程 / View process"); err != nil {
				t.Fatal(err)
			}
			if !tt.Focused("查看过程 / View process") {
				t.Fatal("member action did not hold native focus")
			}
			run.Status, run.Phases[0].Members[0].Status = "completed", "completed"
			model.Snapshot.Items[0].Running = false
			for frame := 0; frame < 3; frame++ {
				tt.Frame()
				if !tt.HasText("focused live member") || !tt.Focused("查看过程 / View process") {
					t.Fatal("clean transition unmounted or disabled the focused member before blur")
				}
			}
			label := "Outside workflow"
			if departure == "header-toggle" {
				label = "Workflow Build"
			}
			if err := tt.Click(label); err != nil {
				t.Fatal(err)
			}
			if tt.HasText("focused live member") {
				t.Fatal("completion did not collapse after actual departure or explicit header toggle")
			}
			if err := tt.Click("Workflow Build"); err != nil {
				t.Fatal(err)
			}
			if tt.HasText("focused live member") {
				t.Fatal("outer hiding retained an obsolete pending phase expansion")
			}
			if err := tt.Click("Workflow phase Build phase"); err != nil {
				t.Fatal(err)
			}
			if !tt.HasText("focused live member") {
				t.Fatal("completed member history could not be expanded explicitly")
			}
		})
	}
}

func TestWorkflowDisclosureNewPhaseCycleReopensButOngoingGrowthDoesNot(t *testing.T) {
	p, q := "Phase P", "Phase Q"
	run := workflowrun.Run{ID: "run", Name: "Build", Status: "running", Phases: []workflowrun.Phase{
		{Key: "p", Name: &p, Members: []workflowrun.Member{{Seq: 1, Label: "P original member", Status: "completed"}}},
		{Key: "q", Name: &q, Members: []workflowrun.Member{{Seq: 2, Label: "Q ongoing member", Status: "running"}}},
	}}
	model := Model{Snapshot: session.ViewState{PlanHistoryBinding: "binding"}}
	tt := ui.NewTester(func(c *ui.Context) {
		WorkflowRunView(c, run, model, ToolCardStyle{FG: c.Theme().Text, Muted: c.Theme().TextMuted})
	}, 900, 600)
	if tt.HasText("P original member") || !tt.HasText("Q ongoing member") {
		t.Fatal("initial clean/running phase disclosure did not match the source")
	}
	if err := tt.Click("Workflow Build"); err != nil {
		t.Fatal(err)
	}
	run.Phases[0].Members[0].Status = "running"
	tt.Frame()
	if !tt.HasText("P original member") || !tt.HasText("Q ongoing member") {
		t.Fatal("a new clean-to-running phase cycle did not reveal the manually collapsed run")
	}
	if err := tt.Click("Workflow phase Phase P"); err != nil {
		t.Fatal(err)
	}
	run.Phases[0].Members = append(run.Phases[0].Members, workflowrun.Member{Seq: 3, Label: "P later member", Status: "running"})
	tt.Frame()
	if tt.HasText("P original member") || tt.HasText("P later member") {
		t.Fatal("same-mode activity growth overrode an explicit collapse")
	}
	run.Phases[0].Members[0].Status = "failed"
	tt.Frame()
	if !tt.HasText("P original member") || !tt.HasText("P later member") {
		t.Fatal("new abnormal state failed to reopen its phase")
	}
	if err := tt.Click("Workflow Build"); err != nil {
		t.Fatal(err)
	}
	run.Phases[1].Members[0].Status = "completed"
	tt.Frame()
	if tt.HasText("Q ongoing member") {
		t.Fatal("unrelated phase completion reopened the abnormal run")
	}
	run.Phases[1].Members[0].Status = "running"
	tt.Frame()
	if !tt.HasText("Q ongoing member") {
		t.Fatal("new phase cycle was lost because aggregate run mode stayed abnormal")
	}
}
