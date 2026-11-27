package assistant

import (
	"math"
	"testing"

	"whaleisle/internal/desktop/pluginrpc"
	assiststore "whaleisle/internal/store/assistant"
	"whaleisle/internal/ui/i18n"

	"github.com/egoist/mygo/ui"
)

// TestWhaleSettingsSectionLocalizedLabelsAndRightAlignedIMSwitch verifies that
// Chinese locale exposes localized UIA labels (not hardcoded English) and that
// the IM default switch aligns to the right edge of the control column.
func TestWhaleSettingsSectionLocalizedLabelsAndRightAlignedIMSwitch(t *testing.T) {
	prev := i18n.Locale()
	i18n.SetLocale(i18n.LocaleZH)
	t.Cleanup(func() { i18n.SetLocale(prev) })

	var editor assiststore.WhaleSettingsEditor
	editor.Bind("host-a")
	ticket, _ := editor.BeginLoad("host-a")
	editor.FinishLoad(ticket, pluginrpc.WhaleSettings{Name: "鲸鱼娘", UserTitle: "主人"}, nil)

	m := Model{
		HarnessReady: true,
		WhaleEnabled: true,
		WhaleSettings: WhaleSettingsModel{
			State:     editor.Snapshot().Value,
			OnRefresh: func() {},
			Preferences: WhalePreferencesModel{
				State:     assiststore.WhalePreferencesSnapshot{Host: "host-a", Loaded: true, IMDefault: true},
				OnRefresh: func() {},
				OnEdit:    func(pluginrpc.WhaleModelRoute, bool) {},
				OnSave:    func(string) {},
			},
		},
	}

	tt := ui.NewTester(func(c *ui.Context) {
		ui.Scroll(c).Fill().Padding(16).Children(func() {
			WhaleSettingsSection(c, m)
		})
	}, 760, 900)
	tt.Frame()

	for _, leaked := range []string{"Whale model", "Whale reasoning effort", "Whale default IM", "Whale name", "Whale user title"} {
		if _, ok := tt.Find(leaked); ok {
			t.Fatalf("hardcoded English accessibility label %q leaked in Chinese locale", leaked)
		}
	}
	modelRect, ok := tt.Find("鲸鱼娘模型")
	if !ok {
		t.Fatalf("localized model select label missing")
	}
	if _, ok := tt.Find("鲸鱼娘思考强度"); !ok {
		t.Fatalf("localized reasoning effort select label missing")
	}
	imLabel := i18n.T(i18n.KeyW4AssistantImDefaultChat)
	var switchRect ui.Rect
	for _, r := range tt.FindAll(imLabel) {
		if r.W <= 34 && r.H <= 22 && r.W >= 26 {
			switchRect = r
			break
		}
	}
	if switchRect.W == 0 {
		t.Fatalf("localized IM switch %q not found; rects=%v", imLabel, tt.FindAll(imLabel))
	}
	// Right edge of the IM switch must align with the right edge of the model
	// select column above it (not left-aligned inside the 180px control box).
	modelRight := modelRect.X + modelRect.W
	switchRight := switchRect.X + switchRect.W
	if math.Abs(float64(switchRight-modelRight)) > 1.5 {
		t.Fatalf("IM switch right edge %.2f not aligned with control column right edge %.2f (switch=%+v model=%+v)", switchRight, modelRight, switchRect, modelRect)
	}
}
