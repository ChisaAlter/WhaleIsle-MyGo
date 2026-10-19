package composer

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
)

func TestPresetIntroductionReducedMotionAcknowledgesExactCueOnce(t *testing.T) {
	type receipt struct {
		host    uint64
		session string
		cue     uint64
	}
	var got []receipt
	show := false
	family := "monospace"
	m := Model{Snapshot: session.ViewState{ComposerHostGeneration: 4, SelectedID: "creator-a"}, AgentPresetSelection: session.AgentPresetSelectionView{Introduction: 9}, OnAgentPresetIntroduced: func(host uint64, id string, cue uint64) { got = append(got, receipt{host, id, cue}) }}
	tt := ui.NewTester(func(c *ui.Context) {
		activeTheme := *c.Theme()
		activeTheme.Font = family
		c.SetTheme(&activeTheme)
		if show {
			paintPresetIntroduction(c, m, "创造模式", c.Theme().Text)
		}
	}, 600, 100)
	tt.SetPreferences(ui.Preferences{ReduceMotion: true})
	show = true
	tt.Frame()
	if len(got) != 1 || got[0] != (receipt{4, "creator-a", 9}) {
		t.Fatalf("wrong cue receipt: %+v", got)
	}
	if !tt.HasText("创造模式") {
		t.Fatal("motion reduction hid the preset caption")
	}
	family = "serif"
	tt.Frame()
	if len(got) != 1 || !tt.HasText("创造模式") {
		t.Fatal("font change replayed a settled cue or hid its caption")
	}
	m.Snapshot.SelectedID = "creator-b"
	m.AgentPresetSelection.Introduction = 10
	tt.Frame()
	if len(got) != 2 || got[1] != (receipt{4, "creator-b", 10}) {
		t.Fatal("new Session cue was swallowed", got)
	}
	m.Snapshot.ComposerHostGeneration = 5
	m.AgentPresetSelection.Introduction = 1
	tt.Frame()
	if len(got) != 3 || got[2] != (receipt{5, "creator-b", 1}) {
		t.Fatal("new Host cue reused the prior timeline", got)
	}
	m.AgentPresetSelection.Introduction = 0
	tt.Frame()
	if len(got) != 3 {
		t.Fatal("ordinary preset generated an introduction receipt")
	}
}
