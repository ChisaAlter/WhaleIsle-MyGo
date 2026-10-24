package composer

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/dsh/api"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

func TestStatefulEmptyAttachmentRailDoesNotDisplaceDraft(t *testing.T) {
	for _, hero := range []bool{false, true} {
		for _, native := range []bool{false, true} {
			d := NewDraftState("")
			resize := &ResizeState{Height: 220}
			m := Model{State: d, Draft: &d.Text, TextInput: d.Client, UseTextInputClient: native, Hero: hero, Resize: resize}
			tt := ui.NewTester(func(c *ui.Context) { View(c, m) }, 800, 1000)
			card, _ := tt.Find("Composer card")
			input, _ := tt.Find("Composer draft")
			wantTop, want, limit := float32(8), float32(36), float32(120)
			if hero {
				// HeroShell.card: 8px top padding around the writing well, a 12-DIP
				// gap and the 34-DIP control row; the tallest well (72 DIP without
				// the shipped draft client) makes the card 134 DIP.
				wantTop, limit = 8, 134
				want = 72
				if native {
					want = 52
				}
			}
			closeMetric(t, input.Y-card.Y, wantTop, "stateful zero-attachment draft top")
			closeMetric(t, input.H, want, "natural empty draft")
			if card.H > limit+8 {
				t.Fatalf("empty attachment rail still expands card: %+v", card)
			}
			d.Error = "Attachment rejected"
			tt.Frame()
			if !tt.HasText(d.Error) {
				t.Fatal("empty rail suppression hid the attachment error")
			}
			d.Error = ""
			m.ResizeEnabled = true
			tt.Frame()
			input, _ = tt.Find("Composer draft")
			closeMetric(t, input.H, 220, "explicit saved height")
			m.ResizeEnabled = false
			tt.Frame()
			input, _ = tt.Find("Composer draft")
			closeMetric(t, input.H, want, "natural height after disabling resize")
			if resize.Height != 220 {
				t.Fatal("natural layout mutated saved resize preference")
			}
		}
	}
}

func TestRunningComposerPrimaryPreservesSendAndStopRoles(t *testing.T) {
	for _, tc := range []struct {
		name, text, origin                      string
		attachment, blocked, wantSend, wantStop bool
	}{
		{name: "empty", wantStop: true},
		{name: "text", text: "queued message", wantSend: true},
		{name: "attachment", attachment: true, wantSend: true},
		{name: "blocked", text: "waiting", blocked: true, wantStop: true},
		{name: "child", text: "followup", origin: "subagent", wantSend: true, wantStop: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d := NewDraftState(tc.text)
			if tc.attachment {
				d.Attachments = []session.ComposerAttachment{{Name: "ready.txt", MediaType: "text/plain", Upload: api.FileUploadValue{ReceiptId: "fixture"}}}
			}
			sends, stops := 0, 0
			m := Model{State: d, Draft: &d.Text, StopBlocked: tc.blocked, Snapshot: session.ViewState{SelectedID: "A", PromptBusy: true, Items: []session.Item{{SessionID: "A", Running: true, Origin: tc.origin}}}, OnSubmitGesture: func(acc bool) {
				if acc {
					t.Error("click accelerated ordinary send")
				}
				sends++
			}, OnStop: func() { stops++ }}
			tt := ui.NewTester(func(c *ui.Context) { View(c, m) }, 800, 700)
			_, hasSend := tt.Find(i18n.T(i18n.KeyComposerSend))
			_, hasStop := tt.Find(i18n.T(i18n.KeyComposerStop))
			if hasSend != tc.wantSend || hasStop != tc.wantStop {
				t.Fatalf("send=%v stop=%v want %v/%v", hasSend, hasStop, tc.wantSend, tc.wantStop)
			}
			if hasSend {
				tt.Click(i18n.T(i18n.KeyComposerSend))
				if sends != 1 || stops != 0 {
					t.Fatalf("send click sent=%d stopped=%d", sends, stops)
				}
			}
			if hasStop {
				tt.Click(i18n.T(i18n.KeyComposerStop))
				if stops != 1 {
					t.Fatal("stop callback unavailable")
				}
			}
		})
	}
}

func TestComposerPhaseAndPanelsRetainMarkedIMEOwner(t *testing.T) {
	d := NewDraftState("A")
	selected := 0
	m := Model{State: d, Draft: &d.Text, TextInput: d.Client, UseTextInputClient: true, Hero: true, SuggestList: &ui.ListState{Selected: &selected}, SlashCatalog: StubSlashCommands()}
	tt := ui.NewTester(func(c *ui.Context) { View(c, m) }, 800, 1000)
	tt.Click("Composer draft")
	d.Client.SetSelectionUTF16(1, 1)
	d.Client.input.SetMarkedText(nil, "中文", ui.TextInputRange{Start: 2, End: 2})
	for step, change := range []func(){
		func() { m.Hero = false },
		func() { m.SkillsOpen = true },
		func() { m.SkillsOpen = false; toggleCommandLauncher(m) },
		func() { dismissSuggestion(m, d.Client); d.GroupReply = &GroupReplyDraft{ThreadID: "fixture"} },
		func() { d.GroupReply = nil; m.Hero = true },
	} {
		change()
		tt.Frame()
		marked, ok := d.Client.MarkedRange()
		if !ok || marked != (ui.TextInputRange{Start: 1, End: 3}) || d.Text != "A中文" || !tt.Focused("Composer draft") || d.Client.Draft != &d.Text {
			t.Fatalf("step%d lost focused marked owner: marked=%+v present=%v draft=%q focused=%v", step, marked, ok, d.Text, tt.Focused("Composer draft"))
		}
	}
}

// One 28px row above the card carries both entries: the workspace picker on
// the left and the agent preset on the right.
func TestHeroRowCarriesWorkspaceAndPreset(t *testing.T) {
	for _, width := range []int{432, 800, 1440} {
		for _, root := range []string{`C:\fixture`, "", `C:\a-very-long-workspace-name-that-must-truncate\inside-the-chip`} {
			d := NewDraftState("")
			preset := "Fixture agent"
			picked := 0
			tt := ui.NewTester(func(c *ui.Context) {
				View(c, Model{State: d, Draft: &d.Text, Hero: true, WorkspaceRoot: root,
					AgentPresetPick: &preset, OnPickHeroWorkspace: func() { picked++ }})
			}, width, 900)
			card, _ := tt.Find("Composer card")
			row, ok := tt.Find("Composer hero preset row")
			if !ok {
				t.Fatal("hero row missing")
			}
			workspace, ok := tt.Find("Composer workspace")
			if !ok {
				t.Fatalf("workspace entry missing for root=%q", root)
			}
			presetChip, ok := tt.Find("Composer hero preset")
			if !ok {
				t.Fatalf("preset chip missing for root=%q", root)
			}
			closeMetric(t, row.X, card.X, "hero row tracks card left")
			closeMetric(t, row.W, card.W, "hero row tracks card width")
			closeMetric(t, row.Y, 4, "hero row top margin")
			closeMetric(t, card.Y-row.Y-row.H, 8, "hero row-card gap")
			closeMetric(t, row.H, 28, "hero row height")
			// The row keeps Padding(0,16,0,20): 20 on the left, 16 on the right.
			closeMetric(t, workspace.X-row.X, 20, "workspace left inset")
			closeMetric(t, workspace.H, 28, "workspace chip height")
			closeMetric(t, presetChip.H, 28, "preset chip height")
			closeMetric(t, (workspace.Y+workspace.H/2)-(presetChip.Y+presetChip.H/2), 0, "both chips share the row")
			// The installed hero keeps the preset chip in the left flow right after
			// the workspace chip (probe: 标准模式 at x=638.3, ~18 DIP past the workspace
			// chip), not flush to the row's right edge.
			// Native rule (round205/round211): a fixed 150-DIP workspace chip, so
			// short and long names share the width and the preset chip after it
			// keeps the native x.
			closeMetric(t, workspace.W, 150, "workspace chip width")
			closeMetric(t, presetChip.X, workspace.X+workspace.W+18, "preset chip sits 18 DIP after the chip")
			t.Logf("width=%d workspace=%v preset=%v", width, workspace, presetChip)
			if gap := presetChip.X - (workspace.X + workspace.W); gap < 0 || gap > 24 {
				t.Fatalf("preset chip gap=%.2f, want it right after the workspace chip: %v %v", gap, workspace, presetChip)
			}
			if presetChip.X+presetChip.W > row.X+row.W-16 {
				t.Fatalf("preset chip overflows the row: %v row=%v", presetChip, row)
			}
			if len(root) > 30 && workspace.W != 150 {
				t.Fatalf("long workspace name changed the chip width: %v", workspace)
			}
			// The chip shows the workspace basename; the long case only pins the
			// width rule, so it skips the label assertion.
			if len(root) <= 30 {
				want := "fixture"
				if root == "" {
					want = i18n.T(i18n.KeyChatHeroChooseWorkspace)
				}
				if !tt.HasText(want) {
					t.Fatalf("workspace label = %v, want %q", tt.Texts(), want)
				}
			}
			if tt.HasText(i18n.T(i18n.KeyNoDirectory)) {
				t.Fatalf("hero row added a second no-directory button: %v", tt.Texts())
			}
			// The chip shares the permission/model capsule face: transparent at
			// rest, the theme's hover fill under the pointer.
			probe := func() ui.Color {
				pixel := tt.Image().RGBAAt(int(workspace.X+6), int(workspace.Y+workspace.H/2))
				return ui.RGBA(pixel.R, pixel.G, pixel.B, 255)
			}
			rest := probe()
			tt.Move(workspace.X+workspace.W/2, workspace.Y+workspace.H/2)
			tt.Frame()
			if hovered := probe(); hovered == rest {
				t.Fatalf("workspace chip painted no hover fill: %v", rest)
			}
			tt.Move(0, 0)
			tt.Frame()
			if err := tt.Click("Composer workspace"); err != nil {
				t.Fatalf("workspace entry is not clickable: %v", err)
			}
			if picked != 1 {
				t.Fatalf("workspace click calls=%d", picked)
			}
		}
	}
}

func TestComposerFont18UsesOriginal28LineHeight(t *testing.T) {
	previous := theme.CurrentPreferences()
	t.Cleanup(func() { theme.SetPreferences(previous) })
	preferences := previous
	preferences.Composer.Size = 18
	theme.SetPreferences(preferences)
	for _, native := range []bool{false, true} {
		d := NewDraftState("first\nsecond")
		tt := ui.NewTester(func(c *ui.Context) {
			View(c, Model{State: d, Draft: &d.Text, TextInput: d.Client, UseTextInputClient: native})
		}, 800, 700)
		input, _ := tt.Find("Composer draft")
		closeMetric(t, input.H, 60, "18px font two-line scrollport")
		if native {
			first, _, _ := d.Client.BoundsForRange(ui.TextInputRange{})
			second, _, _ := d.Client.BoundsForRange(ui.TextInputRange{Start: 6, End: 6})
			closeMetric(t, second.Y-first.Y, 28, "18px font native line advance")
		}
	}
}

func TestSubagentLauncherRetainsIntakeAndVoiceCallbacks(t *testing.T) {
	d := NewDraftState("keep draft")
	selected := 0
	pick, paste, voice := 0, 0, 0
	m := Model{State: d, Draft: &d.Text, TextInput: d.Client, UseTextInputClient: true, SuggestList: &ui.ListState{Selected: &selected}, Snapshot: session.ViewState{SelectedID: "child", Items: []session.Item{{SessionID: "child", Origin: "subagent"}}}, Skills: []session.SkillOption{{Name: "hidden-skill"}}, OnPickImage: func() { pick++ }, OnPasteImage: func(force bool) {
		if !force {
			t.Error("paste action was not explicit")
		}
		paste++
	}, OnVoiceSettings: func() { voice++ }}
	tt := ui.NewTester(func(c *ui.Context) { View(c, m) }, 800, 700)
	for _, label := range []string{composerText("添加文件", "Add files"), composerText("粘贴图片", "Paste image"), composerText("语音设置", "Voice settings")} {
		tt.Click("+")
		if tt.HasText("/hidden-skill") {
			t.Fatal("child exposed parent skill picker")
		}
		if err := tt.Click(label); err != nil {
			t.Fatal(err)
		}
	}
	if pick != 1 || paste != 1 || voice != 1 || d.Text != "keep draft" {
		t.Fatalf("pick=%d paste=%d voice=%d draft=%q", pick, paste, voice, d.Text)
	}
}

func TestAttachmentTransitionsRetainNativeIMEOwner(t *testing.T) {
	d := NewDraftState("A")
	m := Model{State: d, Draft: &d.Text, TextInput: d.Client, UseTextInputClient: true}
	tt := ui.NewTester(func(c *ui.Context) { View(c, m) }, 800, 1000)
	if err := tt.Click("Composer draft"); err != nil {
		t.Fatal(err)
	}
	d.Client.SetSelectionUTF16(1, 1)
	d.Client.input.SetMarkedText(nil, "中文", ui.TextInputRange{Start: 2, End: 2})
	for _, attached := range []bool{true, false} {
		d.Attachments = nil
		if attached {
			d.Attachments = []session.ComposerAttachment{{Name: "ready.txt", MediaType: "text/plain", Upload: api.FileUploadValue{ReceiptId: "fixture"}}}
		}
		tt.Frame()
		marked, ok := d.Client.MarkedRange()
		if !ok || marked != (ui.TextInputRange{Start: 1, End: 3}) || d.Text != "A中文" || !tt.Focused("Composer draft") {
			t.Fatalf("attached=%v lost focused IME owner: marked=%+v present=%v draft=%q focused=%v", attached, marked, ok, d.Text, tt.Focused("Composer draft"))
		}
	}
}

func TestComposerPrimaryEmptyFailedAndAttachmentOnlyAdmission(t *testing.T) {
	d := NewDraftState("")
	sent := 0
	m := Model{State: d, Draft: &d.Text, OnSubmit: func() { sent++ }}
	tt := ui.NewTester(func(c *ui.Context) { View(c, m) }, 800, 700)
	tt.Click(i18n.T(i18n.KeyComposerSend))
	if sent != 0 {
		t.Fatal("empty draft was submitted")
	}
	d.Attachments = []session.ComposerAttachment{{Name: "ready.txt", MediaType: "text/plain", Upload: api.FileUploadValue{ReceiptId: "fixture"}}}
	tt.Frame()
	tt.Click(i18n.T(i18n.KeyComposerSend))
	if sent != 1 {
		t.Fatal("valid attachment-only draft could not submit")
	}
	d.Attachments[0].PrepareError = "Upload failed"
	d.Error = "Upload failed"
	tt.Frame()
	tt.Click(i18n.T(i18n.KeyComposerSend))
	if sent != 1 || !tt.HasText("Upload failed") {
		t.Fatal("failed attachment admission or feedback regressed")
	}
}
