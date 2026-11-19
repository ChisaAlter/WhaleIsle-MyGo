package sidebar

import (
	"testing"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
)

// TestEmptySessionListNamesTheHostWhileItIsDown is the sidebar half of the dead
// page: an empty list answered with 「连接后显示会话」 when the local service was
// not running, which tells a person to connect to something that does not exist
// yet. The list has to name the state that is holding it.
func TestEmptySessionListNamesTheHostWhileItIsDown(t *testing.T) {
	i18n.SetLocale(i18n.LocaleZH)
	defer i18n.SetLocale(i18n.DefaultLocale)
	ink, muted := ui.RGB(90, 90, 90), ui.RGB(150, 150, 150)
	render := func(note string) *ui.Tester {
		return ui.NewTester(func(c *ui.Context) {
			workspaceListFeedback(c, Model{Snapshot: session.ViewState{}, HostNote: note}, nil, ink, muted)
		}, 320, 160)
	}

	note := "本地服务启动后，会话会显示在这里"
	tt := render(note)
	if !tt.HasText(note) {
		t.Fatalf("the sidebar hid the service note: %q", tt.Texts())
	}
	if tt.HasText("连接后显示会话") {
		t.Fatalf("an empty list still blamed the workspace: %q", tt.Texts())
	}

	// A running service keeps the workspace copy: there the workspace really is
	// what the person has yet to choose.
	plain := render("")
	if !plain.HasText("连接后显示会话") || plain.HasText(note) {
		t.Fatalf("a ready service lost its workspace copy: %q", plain.Texts())
	}
}
