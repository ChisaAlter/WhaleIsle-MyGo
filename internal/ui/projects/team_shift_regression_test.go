package projects

import (
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/projectflow"
	"whaleisle/internal/ui/i18n"
)

func TestTeamMemberCardsZeroShiftOnHoverAndSelection(t *testing.T) {
	prev := i18n.Locale()
	i18n.SetLocale(i18n.LocaleEN)
	t.Cleanup(func() { i18n.SetLocale(prev) })

	members := []projectflow.TeamMember{
		{ID: "m1", Name: "Planner", Phase: "active", ModelLabel: "deepseek-v4.1-flash"},
		{ID: "m2", Name: "Builder", Phase: "active", Running: true, ModelLabel: "deepseek-v4.1-pro"},
	}

	// 1. OrdinaryTeamView: hover never injects "Model: ..." into card flow text,
	// and switching highlightCurrent keeps member label geometry identical.
	sessionID := "m1"
	ttOrd := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Width(420).Children(func() {
			OrdinaryTeamView(c, OrdinaryTeamModel{
				SessionID: sessionID,
				Team:      &projectflow.TeamProjection{Members: members},
				State:     &OrdinaryTeamViewState{},
				Columns:   2,
				OnOpenMember: func(string) {
				},
			})
		})
	}, 460, 320)

	m2Before, ok := ttOrd.Find("Builder")
	if !ok {
		t.Fatalf("ordinary team member missing: %v", ttOrd.Texts())
	}
	ttOrd.Move(m2Before.X+m2Before.W/2, m2Before.Y+m2Before.H/2)
	ttOrd.Frame()
	if strings.Contains(strings.Join(ttOrd.Texts(), "\n"), "Model: deepseek-v4.1-pro") {
		t.Fatalf("hover injected model name into ordinary team card flow: %v", ttOrd.Texts())
	}
	m2Hovered, _ := ttOrd.Find("Builder")
	if m2Hovered != m2Before {
		t.Fatalf("ordinary team card shifted on hover: %+v -> %+v", m2Before, m2Hovered)
	}
	sessionID = "m2"
	ttOrd.Frame()
	m2Selected, _ := ttOrd.Find("Builder")
	if m2Selected.Y != m2Before.Y || m2Selected.X != m2Before.X {
		t.Fatalf("highlightCurrent border changed ordinary team card inset: %+v -> %+v", m2Before, m2Selected)
	}

	// 2. Project team member card: hover never injects "Model: ..." into flow,
	// and highlightCurrent uses constant 0.5px border + DrawOver.
	projSession := "m1"
	ttProj := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Width(320).Children(func() {
			projectTeamMember(c, ProjectTeamModel{
				SessionID: projSession,
				State:     &TeamViewState{},
				Overview:  &projectflow.Overview{Team: &projectflow.TeamProjection{Members: members}},
				OnOpenMember: func(string) error {
					return nil
				},
			}, members[1], members[1].Name)
		})
	}, 360, 200)
	pBefore, ok := ttProj.Find("Builder")
	if !ok {
		t.Fatalf("project team member missing: %v", ttProj.Texts())
	}
	ttProj.Move(pBefore.X+pBefore.W/2, pBefore.Y+pBefore.H/2)
	ttProj.Frame()
	if strings.Contains(strings.Join(ttProj.Texts(), "\n"), "Model: deepseek-v4.1-pro") {
		t.Fatalf("hover injected model name into project team card flow: %v", ttProj.Texts())
	}
	pHovered, _ := ttProj.Find("Builder")
	if pHovered != pBefore {
		t.Fatalf("project team card shifted on hover: %+v -> %+v", pBefore, pHovered)
	}
	projSession = "m2"
	ttProj.Frame()
	pSelected, _ := ttProj.Find("Builder")
	if pSelected.Y != pBefore.Y || pSelected.X != pBefore.X {
		t.Fatalf("highlightCurrent border changed project team card inset: %+v -> %+v", pBefore, pSelected)
	}
}
