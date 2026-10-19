package assistant

import (
	"github.com/egoist/mygo/ui"
	"testing"
	"whaleisle/internal/desktop/pluginrpc"
	assiststore "whaleisle/internal/store/assistant"
)

func TestWhaleIdentityBlurAndEnterSubmitOnlyTheFocusedField(t *testing.T) {
	var e assiststore.WhaleSettingsEditor
	e.Bind("owner-a")
	q, _ := e.BeginLoad("owner-a")
	e.FinishLoad(q, pluginrpc.WhaleSettings{HomeDir: "/fixture", SessionID: "whale-a", Name: "Whale", UserTitle: "You", Memory: "server memory"}, nil)
	e.Edit("owner-a", "memory", "unsubmitted memory")
	saves := []string{}
	tt := ui.NewTester(func(c *ui.Context) {
		m := Model{Mode: assiststore.ModeWhale, HarnessReady: true, WhaleEnabled: true, WhaleSettings: WhaleSettingsModel{State: e.Snapshot(), OnRefresh: func() {}, OnEdit: func(field, value string) { e.Edit("owner-a", field, value) }, OnSave: func(action string) { saves = append(saves, action) }}}
		whaleSettingsBlock(c, m, ui.RGB(255, 255, 255), ui.RGB(150, 150, 150), ui.RGB(20, 20, 20), ui.RGB(90, 90, 90), ui.RGB(60, 130, 240))
	}, 1000, 1800)
	if err := tt.Click("Whale name"); err != nil {
		t.Fatal(err)
	}
	tt.Key(ui.Cmd, ui.KeyA)
	tt.Type("New Whale")
	if len(saves) != 0 {
		t.Fatal("identity typing saved before blur/Enter", saves)
	}
	if err := tt.Click("Whale user title"); err != nil {
		t.Fatal(err)
	}
	if len(saves) != 1 || saves[0] != "name" {
		t.Fatal("name blur submitted another field", saves)
	}
	tt.Key(ui.Cmd, ui.KeyA)
	tt.Type("Captain")
	tt.Key(0, ui.KeyEnter)
	if len(saves) != 2 || saves[1] != "title" {
		t.Fatal("title Enter did not submit narrowly", saves)
	}
	if e.Snapshot().Memory != "unsubmitted memory" {
		t.Fatal("identity gesture discarded another field")
	}
	// Host replacement while focused may not auto-submit the retired draft.
	e.Bind("owner-b")
	q, _ = e.BeginLoad("owner-b")
	e.FinishLoad(q, pluginrpc.WhaleSettings{HomeDir: "/fixture-b", SessionID: "whale-b", Name: "Replacement"}, nil)
	tt.Frame()
	if len(saves) != 2 {
		t.Fatal("Host replacement committed an old identity draft", saves)
	}
}
