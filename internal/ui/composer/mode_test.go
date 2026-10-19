package composer

import (
	"testing"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
)

func TestViewPromptModeLabels(t *testing.T) {
	i18n.SetLocale("zh")
	defer i18n.SetLocale(i18n.DefaultLocale)

	draft := ""
	want := i18n.T(i18n.KeyComposerPermissionDefault)
	snap := nativeCommandSnapshot(t)
	snap.PromptMode = session.PromptModeQueue
	snap.SelectedPermission = want
	pick := want
	tt := ui.NewTester(func(c *ui.Context) {
		View(c, Model{
			Snapshot:           snap,
			PermissionOptions:  []session.PermissionOption{{Label: want, Value: "workspace-write"}},
			PermissionPick:     &pick,
			Draft:              &draft,
			OnPromptModeChange: func(session.PromptMode) {},
		})
	}, 800, 200)

	if !tt.HasText(want) {
		t.Fatalf("missing %q; texts = %q", want, tt.Texts())
	}
	// Withdrawing the contribution removes this seat even though the Host's
	// selected permission and process catalog are still retained.
	snap.NativeContributions.Known = false
	tt.Frame()
	if tt.HasText(want) {
		t.Fatal("withdrawn permission contribution retained its toolbar seat")
	}
}
