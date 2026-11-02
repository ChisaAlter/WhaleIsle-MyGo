package composer

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/dsh"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
)

func heroEmptyTester(t *testing.T, mutate func(*Model)) (*ui.Tester, Model) {
	t.Helper()
	d := NewDraftState("")
	m := Model{State: d, Draft: &d.Text, TextInput: d.Client, UseTextInputClient: true, Hero: true}
	if mutate != nil {
		mutate(&m)
	}
	tt := ui.NewTester(func(c *ui.Context) {
		// A dark canvas makes the card's rounded corner and its light face
		// distinguishable in the pixel probes below.
		c.SetTheme(ui.DarkTheme())
		View(c, m)
	}, 800, 400)
	return tt, m
}

// Installed 0.3.10 InputBar: 8 top + 52 min writing well + 12 gap +
// toolbar padding (2 + 6) and 34-DIP primary. Card size is independent of
// caption/window geometry; the toolbar owns its 8-DIP horizontal inset.
func TestHeroEmptyComposerMatchesReferenceGeometry(t *testing.T) {
	permission := "workspace-write"
	tt, _ := heroEmptyTester(t, func(m *Model) {
		m.PermissionPick = &permission
		m.PermissionOptions = []session.PermissionOption{
			{Label: "read-only", Value: "read-only"},
			{Label: "workspace-write", Value: "workspace-write"},
		}
		m.Snapshot = session.ViewState{
			NativeContributions: dsh.NativeContributions{Known: true, PermissionCommand: true, ModelCommand: true},
			SelectedPermission:  "workspace-write",
			SelectedModel:       "deepseek-official/deepseek-flash",
			ModelLabels:         []string{"deepseek-official/deepseek-flash"},
		}
	})
	card, ok := tt.Find("Composer card")
	if !ok {
		t.Fatal("composer card missing")
	}
	closeMetric(t, card.H, 114, "hero card height from InputBar source")
	input, _ := tt.Find("Composer draft")
	closeMetric(t, input.Y-card.Y, 8, "hero top padding")
	closeMetric(t, input.X-card.X, 0, "hero writing well spans card")
	closeMetric(t, input.H, 52, "hero writing well height")
	send, ok := tt.Find(i18n.T(i18n.KeyComposerSend))
	if !ok {
		t.Fatal("hero send button missing")
	}
	closeMetric(t, send.W, 34, "hero primary width")
	closeMetric(t, send.H, 34, "hero primary height")
	closeMetric(t, card.X+card.W-(send.X+send.W), 8, "hero primary right inset")
	if !tt.HasText("+") || !tt.HasText(i18n.T(i18n.KeyComposerPermissionWorkspace)) {
		t.Fatalf("hero control row lost its launcher or permission pill: %v", tt.Texts())
	}
	if !tt.HasText("deepseek-official/deepseek-flash") {
		t.Fatalf("hero control row lost its model pill: %v", tt.Texts())
	}
	// A 28 DIP rounded card: the diagonal corner is outside the face while a
	// point 10px in already sits on it.
	image := tt.Image()
	corner := image.RGBAAt(int(card.X+6), int(card.Y+6))
	inside := image.RGBAAt(int(card.X+10), int(card.Y+10))
	closeBackground := image.RGBAAt(int(card.X+2), int(card.Y+2))
	if corner == inside {
		t.Fatalf("card corner is not rounded: %v", corner)
	}
	if inside == closeBackground || inside.R < 200 {
		t.Fatalf("card face missing at the diagonal: %v", inside)
	}
}

func TestHeroEmptyComposerHidesUnconfiguredVoiceAndContext(t *testing.T) {
	tt, _ := heroEmptyTester(t, nil)
	if tt.HasText(i18n.T(i18n.KeyComposerVoiceInput)) {
		t.Fatalf("hero empty state shows the microphone seat: %v", tt.Texts())
	}
	if tt.HasText("—%") || tt.HasText("0%") {
		t.Fatalf("hero empty state shows a context percentage: %v", tt.Texts())
	}
	if _, ok := tt.Find("Context occupancy unavailable"); ok {
		t.Fatal("hero empty state shows the context placeholder")
	}
	// The row's workspace entry stays, but the empty state keeps exactly one
	// affordance: no second "no directory" control beside it.
	if !tt.HasText(i18n.T(i18n.KeyChatHeroChooseWorkspace)) {
		t.Fatalf("hero lost its workspace entry: %v", tt.Texts())
	}
	if tt.HasText(i18n.T(i18n.KeyNoDirectory)) {
		t.Fatalf("hero rendered a second no-directory control: %v", tt.Texts())
	}
}

func TestHeroEmptyComposerRestoresVoiceAndContextWhenConfigured(t *testing.T) {
	voice, _ := heroEmptyTester(t, func(m *Model) {
		m.OnStartVoice = func() {}
		m.Snapshot = session.ViewState{ClientReady: true, SpeechSelection: "provider/en",
			SpeechReadinessKnown: true, SpeechReady: true}
	})
	if !voice.HasText(i18n.T(i18n.KeyComposerVoiceInput)) && !voice.HasText("语音输入") {
		t.Fatalf("configured hero keeps hiding the microphone: %v", voice.Texts())
	}
	context, _ := heroEmptyTester(t, func(m *Model) {
		m.Snapshot = session.ViewState{ContextOccupancy: &session.ContextOccupancy{Percent: 42}}
	})
	if !context.HasText("42%") {
		t.Fatalf("hero did not restore the context meter: %v", context.Texts())
	}
}
