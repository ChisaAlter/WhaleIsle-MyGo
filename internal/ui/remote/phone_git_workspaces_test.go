package remote

import (
	"github.com/egoist/mygo/ui"
	"testing"
)

func TestPhoneGitWorkspaceExplicitSelection(t *testing.T) {
	calls := 0
	id := ""
	selected := false
	tt := ui.NewTester(func(c *ui.Context) {
		phoneGitWorkspaces(c, Model{RemoteEnabled: true, PhoneGitWorkspaces: []PhoneGitWorkspace{{ID: "host:w", Title: "Work", Root: "/fixture"}}, OnTogglePhoneGitWorkspace: func(key string, on bool) { calls++; id = key; selected = on }})
	}, 900, 600)
	if calls != 0 {
		t.Fatal("render granted workspace")
	}
	tt.Click("Work · /fixture")
	if calls != 1 || id != "host:w" || !selected {
		t.Fatal("explicit selection missing")
	}
}
