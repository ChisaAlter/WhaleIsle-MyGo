package composer

import (
	"github.com/egoist/mygo/ui"
	"math"
	"strings"
	"testing"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

func closeMetric(t *testing.T, got, want float32, name string) {
	t.Helper()
	if math.Abs(float64(got-want)) > .1 {
		t.Errorf("%s=%.2f want %.2f", name, got, want)
	}
}

func TestComposerCurrentCardGeometry(t *testing.T) {
	for _, hero := range []bool{false, true} {
		draft := "hello"
		tt := ui.NewTester(func(c *ui.Context) { View(c, Model{Draft: &draft, Hero: hero}) }, 800, 200)
		card, ok := tt.Find("Composer card")
		if !ok {
			t.Fatal("composer card missing")
		}
		input, ok := tt.Find("Composer draft")
		if !ok {
			t.Fatal("composer input missing")
		}
		closeMetric(t, input.Y-card.Y, 8, "card top padding")
		closeMetric(t, input.X-card.X, 0, "input owns horizontal inset")
		closeMetric(t, input.W, card.W, "draft scrollport width")
		send, _ := tt.Find(i18n.T(i18n.KeyComposerSend))
		closeMetric(t, send.W, 34, "primary width")
		closeMetric(t, send.H, 34, "primary height")
		closeMetric(t, card.X+card.W-(send.X+send.W), 8, "primary right inset")
		closeMetric(t, card.Y+card.H-(send.Y+send.H), 8, "primary bottom inset including upward shift")
		closeMetric(t, card.X, 16, "shared side clearance")
	}
}

func TestComposerSendAndStopUseInfoFill(t *testing.T) {
	for _, busy := range []bool{false, true} {
		draft := "hello"
		if busy {
			draft = ""
		}
		sent, stopped := "", false
		tt := ui.NewTester(func(c *ui.Context) {
			View(c, Model{Draft: &draft, Snapshot: session.ViewState{PromptBusy: busy}, OnSend: func(s string) { sent = s }, OnStop: func() { stopped = true }})
		}, 800, 200)
		label := i18n.T(i18n.KeyComposerSend)
		if busy {
			label = i18n.T(i18n.KeyComposerStop)
		}
		button, ok := tt.Find(label)
		if !ok {
			t.Fatalf("missing %s", label)
		}
		p := tt.Image().RGBAAt(int(button.X+6), int(button.Y+button.H/2))
		if p.R != 65 || p.G != 118 || p.B != 230 {
			t.Errorf("%s fill=%v want info fill 65,118,230", label, p)
		}
		if err := tt.Click(label); err != nil {
			t.Fatal(err)
		}
		if busy && !stopped {
			t.Fatal("stop callback lost")
		}
		if !busy && sent != "hello" {
			t.Fatalf("send callback=%q", sent)
		}
	}
}

func TestComposerResolvedAndCustomPalette(t *testing.T) {
	for _, colors := range []theme.Colors{theme.Resolve("light").Active(false), theme.Resolve("dark").Active(false), {FG: "#202020", Muted: "#606060", Border: "#888888", InputMajor: "#d7dbdf", SelectorFill: "#adbdbb", ButtonInfoFill: "#ffffff", ButtonFG: "#111111", Accent: "#606060", BG: "#cccccc"}} {
		for _, busy := range []bool{false, true} {
			draft := "hello"
			if busy {
				draft = ""
			}
			tt := ui.NewTester(func(c *ui.Context) {
				viewWithColors(c, Model{Draft: &draft, WorkspaceRoot: "fixture", Snapshot: session.ViewState{PromptBusy: busy}, OnStop: func() {}}, colors)
			}, 800, 200)
			label := i18n.T(i18n.KeyComposerSend)
			if busy {
				label = i18n.T(i18n.KeyComposerStop)
			}
			r, ok := tt.Find(label)
			if !ok {
				t.Fatalf("missing primary %s", label)
			}
			want := chrome.Color(colors.ButtonInfoFill)
			got := tt.Image().RGBAAt(int(r.X+6), int(r.Y+r.H/2))
			if got.R != want.R || got.G != want.G || got.B != want.B {
				t.Errorf("primary ignores palette %q: %v", colors.ButtonInfoFill, got)
			}
			ink := chrome.Color(colors.ButtonFG)
			inkPixels := 0
			rendered := tt.Image()
			for y := int(r.Y + 6); y < int(r.Y+r.H-6); y++ {
				for x := int(r.X + 9); x < int(r.X+r.W-9); x++ {
					p := rendered.RGBAAt(x, y)
					if absByte(p.R, ink.R) < 20 && absByte(p.G, ink.G) < 20 && absByte(p.B, ink.B) < 20 {
						inkPixels++
					}
				}
			}
			if inkPixels < 1 {
				t.Errorf("primary on-accent glyph did not use %s (%d pixels)", colors.ButtonFG, inkPixels)
			}
			card, _ := tt.Find("Composer card")
			fill := chrome.Color(colors.InputMajor)
			p := tt.Image().RGBAAt(int(card.X+40), int(card.Y+4))
			// The software renderer composites the semantic translucent fill over white.
			for i, pair := range [][2]uint8{{p.R, fill.R}, {p.G, fill.G}, {p.B, fill.B}} {
				expected := uint8((int(pair[1])*int(fill.A) + 255*(255-int(fill.A)) + 127) / 255)
				if absByte(pair[0], expected) > 1 {
					t.Errorf("card channel %d=%d want %d", i, pair[0], expected)
				}
			}
		}
	}
}
func absByte(a, b uint8) int { return int(math.Abs(float64(a) - float64(b))) }

func TestComposerNarrowToolbarKeepsPrimaryAndActionsVisible(t *testing.T) {
	for _, width := range []int{320, 400, 598} {
		d := NewDraftState("hello")
		selected := 0
		picked, speech, cycled, skills, modes := 0, 0, 0, 0, 0
		model := Model{State: d, Draft: &d.Text, TextInput: d.Client, UseTextInputClient: true, SuggestList: &ui.ListState{Selected: &selected}, WorkspaceRoot: "fixture", OnPickImage: func() { picked++ }, OnPickDirectory: func() {}, OnPickSpeech: func() { speech++ }, OnCycleSpeech: func() { cycled++ }, OnPromptModeChange: func(session.PromptMode) { modes++ }, OnToggleSkills: func(bool) { skills++ }}
		tt := ui.NewTester(func(c *ui.Context) { View(c, model) }, width, 700)
		card, _ := tt.Find("Composer card")
		send, ok := tt.Find(i18n.T(i18n.KeyComposerSend))
		if !ok {
			t.Fatalf("width %d lost primary", width)
		}
		closeMetric(t, send.W, 34, "narrow primary width")
		closeMetric(t, send.H, 34, "narrow primary height")
		closeMetric(t, card.X+card.W-(send.X+send.W), 8, "narrow primary right inset")
		if card.H > 120 {
			t.Fatalf("idle tools expanded card at width %d: %v", width, card)
		}
		for _, label := range []string{composerText("添加文件", "Add files"), composerText("转写语音文件", "Transcribe audio file"), composerText("切换语音服务", "Change speech provider"), composerText("忙碌时排队", "Queue while busy"), composerText("技能", "Skills")} {
			if tt.HasText(label) {
				t.Fatalf("idle tool %q is permanently visible", label)
			}
			if err := tt.Click("+"); err != nil {
				t.Fatal(err)
			}
			if !tt.HasText(label) {
				t.Fatalf("width %d tool %q missing from +", width, label)
			}
			// Select through the same list and Enter path as the native keyboard.
			items := buildSuggestItems(model, SuggestSlash, "")
			for i, item := range items {
				if item.label == label {
					selected = i
					break
				}
			}
			tt.Key(0, ui.KeyEnter)
			if d.Text != "hello" {
				t.Fatalf("tool %q replaced draft: %q", label, d.Text)
			}
		}
		if picked != 1 || speech != 1 || cycled != 1 || skills != 1 || modes != 1 {
			t.Fatalf("width %d callbacks pick=%d speech=%d cycle=%d skills=%d modes=%d", width, picked, speech, cycled, skills, modes)
		}
	}
}
func TestComposerLongModelLabelsKeepPrimaryVisible(t *testing.T) {
	for _, width := range []int{320, 400, 598, 800, 1200} {
		for _, busy := range []bool{false, true} {
			draft := "hello"
			if busy {
				draft = ""
			}
			model := strings.Repeat("DeepSeek-Reasoner-Extended-", 3)
			permission := strings.Repeat("Workspace write ", 3)
			preset := strings.Repeat("Coding Agent ", 3)
			tt := ui.NewTester(func(c *ui.Context) {
				View(c, Model{Draft: &draft, WorkspaceRoot: "fixture", ModelPick: &model, PermissionPick: &permission, AgentPresetPick: &preset, Snapshot: session.ViewState{ModelLabels: []string{model, "small"}, PermissionLabels: []string{permission, "read"}, AgentPresetLabels: []string{preset, "default"}, PromptBusy: busy}, OnStop: func() {}, OnPickImage: func() {}, OnPickDirectory: func() {}, OnPickSpeech: func() {}, OnCycleSpeech: func() {}, OnPromptModeChange: func(session.PromptMode) {}, OnToggleSkills: func(bool) {}})
			}, width, 1000)
			label := i18n.T(i18n.KeyComposerSend)
			if busy {
				label = i18n.T(i18n.KeyComposerStop)
			}
			card, _ := tt.Find("Composer card")
			r, ok := tt.Find(label)
			t.Logf("width=%d busy=%v card=%v primary=%v", width, busy, card, r)
			if !ok || r.W < 33.9 || r.H < 33.9 || r.X < card.X || r.X+r.W > card.X+card.W+.1 {
				t.Errorf("width=%d busy=%v clipped primary: %v card=%v", width, busy, r, card)
			}
			tt.SetSize(320, 1000)
			r, _ = tt.Find(label)
			card, _ = tt.Find("Composer card")
			if r.W < 33.9 || r.X < card.X || r.X+r.W > card.X+card.W+.1 {
				t.Errorf("shrink clipped primary: %v card=%v", r, card)
			}
			tt.SetSize(1200, 1000)
			r, _ = tt.Find(label)
			card, _ = tt.Find("Composer card")
			if r.W < 33.9 || r.X < card.X || r.X+r.W > card.X+card.W+.1 {
				t.Errorf("expand clipped primary: %v card=%v", r, card)
			}
		}
	}
}

func TestReviewOrdinaryModelStatusKeepsPrimary(t *testing.T) {
	for _, label := range []string{"deepseek-reasoner (thinking)", "DeepSeek-V3.2-Exp (reasoning enabled)", "connection reset by peer while fetching models"} {
		for _, errorMode := range []bool{false, true} {
			draft := "x"
			model := label
			ss := session.ViewState{ModelLabels: []string{label}}
			if errorMode {
				ss.ModelLabels = nil
				ss.ModelsErr = label
			}
			tt := ui.NewTester(func(c *ui.Context) {
				View(c, Model{Draft: &draft, WorkspaceRoot: "fixture", ModelPick: &model, Snapshot: ss})
			}, 320, 500)
			r, _ := tt.Find(i18n.T(i18n.KeyComposerSend))
			t.Logf("label=%q error=%v primary=%v", label, errorMode, r)
			if r.W < 33.9 || r.X+r.W > 304.1 {
				t.Errorf("primary collapsed or offcard for %q error=%v: %v", label, errorMode, r)
			}
		}
	}
}

func TestComposerTruncatedModelKeepsRealValueAndSelection(t *testing.T) {
	draft := "hello"
	original := "DeepSeek-V3.2-Exp (reasoning enabled)"
	model := original
	changed := ""
	changes := 0
	tt := ui.NewTester(func(c *ui.Context) {
		View(c, Model{Draft: &draft, WorkspaceRoot: "fixture", ModelPick: &model, Snapshot: session.ViewState{ModelLabels: []string{original, "small"}}, OnModelChange: func(s string) { changed = s; changes++ }})
	}, 320, 500)
	trigger, ok := tt.Find(original)
	if !ok {
		t.Fatal("complete model value not available through trigger label")
	}
	if trigger.W > 220.1 {
		t.Errorf("model trigger exceeds source cap: %v", trigger)
	}
	if model != original || changes != 0 {
		t.Fatal("rendering changed real model selection")
	}
	if err := tt.Click(original); err != nil {
		t.Fatal(err)
	}
	if !tt.HasText(original) {
		t.Fatal("full model choice missing from popup")
	}
	if err := tt.Click("small"); err != nil {
		t.Fatal(err)
	}
	if model != "small" || changed != "small" || changes != 1 {
		t.Fatalf("selection/callback: model=%q changed=%q count=%d", model, changed, changes)
	}
	if err := tt.Click("small"); err != nil {
		t.Fatal(err)
	}
	tt.Key(0, ui.KeyEscape)
	if model != "small" || changes != 1 {
		t.Fatal("dismissal changed selection or repeated callback")
	}
}
