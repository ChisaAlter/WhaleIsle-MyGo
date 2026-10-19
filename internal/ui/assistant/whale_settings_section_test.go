package assistant

import (
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/pluginrpc"
	assiststore "whaleisle/internal/store/assistant"
	"whaleisle/internal/ui/i18n"
)

func TestWhaleSettingsSectionSourceRowsAndHostDialogIsolation(t *testing.T) {
	old := i18n.Locale()
	i18n.SetLocale("en")
	defer i18n.SetLocale(old)
	var editor assiststore.WhaleSettingsEditor
	editor.Bind("fixture-host-a")
	ticket, _ := editor.BeginLoad("fixture-host-a")
	editor.FinishLoad(ticket, pluginrpc.WhaleSettings{Name: "Whale", UserTitle: "Captain", PersonaText: "Patient and concise", Memory: "Fixture memory"}, nil)
	m := Model{HarnessReady: true, WhaleEnabled: true, WhaleSettings: WhaleSettingsModel{State: editor.Snapshot(), OnRefresh: func() {}, Preferences: WhalePreferencesModel{State: assiststore.WhalePreferencesSnapshot{Host: "fixture-host-a", Loaded: true}, OnRefresh: func() {}}}}
	tt := ui.NewTester(func(c *ui.Context) { ui.Scroll(c).Fill().Padding(24).Children(func() { WhaleSettingsSection(c, m) }) }, 620, 1000)
	for _, label := range []string{"Name and title", "Model and reasoning effort", "Default IM conversation", "Extra persona", "Long-term memory", "Skills"} {
		if !tt.HasText(label) {
			t.Fatal("missing source contribution", label)
		}
	}
	snapshotWhaleSection(t, tt, "settings-pet-assistant-wide.png")
	tt.SetSize(340, 1000)
	snapshotWhaleSection(t, tt, "settings-pet-assistant-narrow.png")
	tt.SetSize(900, 1000)
	if err := tt.Click("manage-whale-memory"); err != nil {
		t.Fatal(err)
	}
	if !tt.HasText("Assistant settings") {
		t.Fatal("memory editor did not open")
	}
	m.WhaleSettings.State.HostKey = "fixture-host-b"
	tt.Frame()
	if tt.HasText("Assistant settings") {
		t.Fatal("old Host's settings dialog adopted a replacement Host")
	}
	m.WhaleSettings.State.Loaded = false
	m.WhaleSettings.State.Busy = true
	tt.Frame()
	snapshotWhaleSection(t, tt, "settings-pet-assistant-loading.png")
	m.WhaleSettings.State.Busy = false
	m.WhaleSettings.State.Error = "Synthetic assistant settings error"
	tt.Frame()
	if !tt.HasText(m.WhaleSettings.State.Error) {
		t.Fatal("Host error hidden")
	}
	snapshotWhaleSection(t, tt, "settings-pet-assistant-error.png")
}

func snapshotWhaleSection(t *testing.T, tt *ui.Tester, name string) {
	t.Helper()
	dir := os.Getenv("SECONDARY_SNAPSHOT_DIR")
	if dir == "" {
		return
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	f, err := os.Create(filepath.Join(dir, name))
	if err != nil {
		t.Fatal(err)
	}
	if err = png.Encode(f, tt.Image()); err != nil {
		_ = f.Close()
		t.Fatal(err)
	}
	if err = f.Close(); err != nil {
		t.Fatal(err)
	}
}
