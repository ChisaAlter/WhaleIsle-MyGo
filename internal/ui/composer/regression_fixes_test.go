package composer

import (
	"fmt"
	"math"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/dsh"
	"whaleisle/internal/dsh/api"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
)

func TestWorkspaceChipReflectsOpenStateAndTogglesSecondClick(t *testing.T) {
	prev := i18n.Locale()
	i18n.SetLocale(i18n.LocaleEN)
	t.Cleanup(func() { i18n.SetLocale(prev) })

	d := NewDraftState("")
	var anchor ui.Handle
	var nav *session.WorkspaceNavigationState
	tt := ui.NewTester(func(c *ui.Context) {
		m := Model{
			State:                 d,
			Draft:                 &d.Text,
			Hero:                  true,
			WorkspaceRoot:         `C:\work\demo`,
			WorkspacePickerAnchor: &anchor,
			Snapshot:              session.ViewState{WorkspaceNavigation: nav},
			OnPickHeroWorkspace: func() {
				if nav != nil && nav.Mode == "pick" {
					nav = nil
					return
				}
				nav = &session.WorkspaceNavigationState{Token: 1, Mode: "pick"}
			},
		}
		View(c, m)
		if nav != nil && nav.Mode == "pick" {
			open := true
			ui.PopoverBase(c, c.Resolve(anchor), &open, func(panel ui.Element) {
				panel.Width(200).Children(func() {
					ui.Text(c, "No working directory")
					ui.Text(c, "Add workspace…")
				})
			})
			if !open {
				nav = nil
			}
		}
	}, 800, 600)

	if err := tt.Click("Composer workspace"); err != nil {
		t.Fatal(err)
	}
	if nav == nil || !tt.HasText("Add workspace…") {
		t.Fatalf("first click did not open workspace popover: nav=%+v texts=%v", nav, tt.Texts())
	}
	if err := tt.Click("Composer workspace"); err != nil {
		t.Fatal(err)
	}
	if nav != nil || tt.HasText("Add workspace…") {
		t.Fatalf("second click on workspace trigger did not close popover: nav=%+v texts=%v", nav, tt.Texts())
	}
}

func TestSkillsPopupArrowsAndInlineButtonsDoNotSubmitDraft(t *testing.T) {
	prev := i18n.Locale()
	i18n.SetLocale(i18n.LocaleEN)
	t.Cleanup(func() { i18n.SetLocale(prev) })

	d := NewDraftState("")
	sel := 0
	filter := ""
	skillsOpen := true
	toggled := ""
	deleted := ""
	skills := []session.SkillOption{
		{Name: "alpha", Description: "First skill", UserOnly: true},
		{Name: "beta", Description: "Second skill", UserOnly: true},
	}
	tt := ui.NewTester(func(c *ui.Context) {
		View(c, Model{
			State:        d,
			Draft:        &d.Text,
			SkillsOpen:   skillsOpen,
			Skills:       skills,
			SkillsList:   &ui.ListState{Selected: &sel},
			SkillsFilter: &filter,
			OnToggleSkills: func(open bool) {
				skillsOpen = open
			},
			OnToggleSkillModel: func(name string, _ bool) {
				toggled = name
			},
			OnDeleteSkill: func(name string) {
				deleted = name
			},
		})
	}, 800, 600)

	// Single-clicking the second row selects it without inserting or closing.
	if err := tt.Click("/beta"); err != nil {
		t.Fatal(err)
	}
	if d.Text != "" || !skillsOpen || sel != 1 {
		t.Fatalf("selecting skill row mutated draft or closed popup: draft=%q open=%v sel=%d", d.Text, skillsOpen, sel)
	}
	// Arrow key moves selection without inserting or closing.
	tt.Key(0, ui.KeyUp)
	if d.Text != "" || !skillsOpen || sel != 0 {
		t.Fatalf("arrow key mutated draft or closed popup: draft=%q open=%v sel=%d", d.Text, skillsOpen, sel)
	}
	// Inline action buttons must not insert or close.
	if err := tt.Click(i18n.T(i18n.KeyComposerSkillsToggleModel)); err != nil {
		t.Fatal(err)
	}
	if toggled != "alpha" || d.Text != "" || !skillsOpen {
		t.Fatalf("inline toggle button submitted skill: toggled=%q draft=%q open=%v", toggled, d.Text, skillsOpen)
	}
	if err := tt.Click(i18n.T(i18n.KeyComposerSkillsDelete)); err != nil {
		t.Fatal(err)
	}
	if deleted != "alpha" || d.Text != "" || !skillsOpen {
		t.Fatalf("inline delete button submitted skill: deleted=%q draft=%q open=%v", deleted, d.Text, skillsOpen)
	}
	// Explicit Enter submits the selected skill and closes the popup.
	if err := tt.Click("/beta"); err != nil {
		t.Fatal(err)
	}
	tt.Key(0, ui.KeyEnter)
	if d.Text != "/beta " || skillsOpen {
		t.Fatalf("Enter did not submit selected skill: draft=%q open=%v", d.Text, skillsOpen)
	}
}

func TestSuggestPopupDrillRowZeroShiftOnHoverAndSelection(t *testing.T) {
	d := NewDraftState("")
	d.Client.input.ReplaceText(nil, "@")
	sel := 0
	m := Model{
		State:              d,
		Draft:              &d.Text,
		TextInput:          d.Client,
		UseTextInputClient: true,
		SuggestList:        &ui.ListState{Selected: &sel},
		OnSearchReferences: func(string, bool) []session.ReferenceCandidate {
			return []session.ReferenceCandidate{
				{Kind: "directory", Path: "alpha-dir"},
				{Kind: "directory", Path: "beta-dir"},
			}
		},
	}
	tt := ui.NewTester(func(c *ui.Context) { View(c, m) }, 800, 600)
	alpha0, ok1 := tt.Find("alpha-dir")
	beta0, ok2 := tt.Find("beta-dir")
	if !ok1 || !ok2 {
		t.Fatalf("missing drillable rows: %v", tt.Texts())
	}
	// Hover the second directory row to move selection from row 0 to row 1.
	tt.Move(beta0.X+beta0.W/2, beta0.Y+beta0.H/2)
	tt.Frame()
	if sel != 1 {
		t.Fatalf("hover did not select second row: sel=%d", sel)
	}
	alpha1, _ := tt.Find("alpha-dir")
	beta1, _ := tt.Find("beta-dir")
	checkSame := func(name string, a, b ui.Rect) {
		t.Helper()
		if math.Abs(float64(a.X-b.X)) > 0.01 || math.Abs(float64(a.Y-b.Y)) > 0.01 ||
			math.Abs(float64(a.W-b.W)) > 0.01 || math.Abs(float64(a.H-b.H)) > 0.01 {
			t.Fatalf("%s shifted on hover/selection: %+v -> %+v", name, a, b)
		}
	}
	checkSame("alpha row", alpha0, alpha1)
	checkSame("beta row", beta0, beta1)
}

func TestCommandLauncherNoDuplicateHeadersAndLongListScrollable(t *testing.T) {
	prev := i18n.Locale()
	i18n.SetLocale(i18n.LocaleZH)
	t.Cleanup(func() { i18n.SetLocale(prev) })

	d := NewDraftState("")
	sel := 0
	cmds := make([]api.ListResultItem2, 0, 14)
	for i := 0; i < 14; i++ {
		cmds = append(cmds, api.ListResultItem2{
			Name:        fmt.Sprintf("cmd-%02d", i),
			Description: fmt.Sprintf("Command %02d", i),
		})
	}
	m := Model{
		State:              d,
		Draft:              &d.Text,
		TextInput:          d.Client,
		UseTextInputClient: true,
		SuggestList:        &ui.ListState{Selected: &sel},
		Snapshot: session.ViewState{
			ClientReady:        true,
			SelectedID:         "s1",
			SlashCommandsKnown: true,
			SlashCommands:      cmds,
		},
		OnPickImage:           func() {},
		OnPickDirectory:       func() {},
		OnPickSpeech:          func() {},
		OnPromptModeChange:    func(session.PromptMode) {},
		OnToggleSkills:        func(bool) {},
		OnUseBusyEnterSetting: func() {},
	}
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Size(800, 600).Justify(ui.End).Children(func() { View(c, m) })
	}, 800, 600)
	if err := tt.Click(i18n.T(i18n.KeyComposerCommands)); err != nil {
		t.Fatal(err)
	}
	addCount, cmdCount := 0, 0
	for _, txt := range tt.Texts() {
		switch txt {
		case "添加":
			addCount++
		case "指令":
			cmdCount++
		}
	}
	if addCount != 1 || cmdCount != 1 {
		t.Fatalf("duplicate or missing launcher section headers: 添加=%d 指令=%d texts=%v", addCount, cmdCount, tt.Texts())
	}
	addRect, _ := tt.Find("添加")
	cmdRect, _ := tt.Find("指令")
	if addRect.Y >= cmdRect.Y {
		t.Fatalf("添加 section (%v) should precede 指令 section (%v)", addRect.Y, cmdRect.Y)
	}
	footer, ok := tt.Find(i18n.T(i18n.KeyComposerMenuFooter))
	if !ok || footer.Y+footer.H > 600 {
		t.Fatalf("footer hint not pinned inside viewport: ok=%v rect=%+v", ok, footer)
	}
}

func TestComposerCollapseDoesNotSpinIdleAnimation(t *testing.T) {
	d := NewDraftState("")
	collapsed := false
	start := time.Unix(2000, 0)
	tt := ui.NewTester(func(c *ui.Context) {
		View(c, Model{
			State:     d,
			Draft:     &d.Text,
			Collapsed: collapsed,
		})
	}, 800, 400)
	tt.SetTime(start)
	collapsed = true
	tt.SetTime(start.Add(time.Millisecond))
	// Toolbar is collapsed immediately without leaving a dead 200ms animation spinning.
	if _, ok := tt.Find(i18n.T(i18n.KeyComposerCommands)); ok {
		t.Fatal("collapsed pill still rendered expanded toolbar")
	}
	collapsed = false
	tt.SetTime(start.Add(50 * time.Millisecond))
	if _, ok := tt.Find(i18n.T(i18n.KeyComposerCommands)); !ok {
		t.Fatal("expanding composer did not restore toolbar")
	}
}

func TestModelSeatWithoutCatalogShowsUnconfiguredGuidanceLoadingAndSettings(t *testing.T) {
	prev := i18n.Locale()
	i18n.SetLocale(i18n.LocaleEN)
	t.Cleanup(func() { i18n.SetLocale(prev) })

	snap := session.ViewState{
		SelectedID:          "s1",
		SelectedModel:       "deepseek-official/deepseek-flash",
		SelectedModelEffort: "high",
		NativeContributions: dsh.NativeContributions{Known: true, ModelCommand: true},
	}
	reloaded, openedSettings := 0, 0
	tt := ui.NewTester(func(c *ui.Context) {
		paintModelPicker(c, Model{
			Snapshot:            snap,
			OnRetryModels:       func() { reloaded++ },
			OnOpenModelSettings: func() { openedSettings++ },
		}, 28)
	}, 700, 400)
	if err := tt.Click("deepseek-official/deepseek-flash"); err != nil {
		t.Fatal(err)
	}
	if !tt.HasText("Default (not configured)") {
		t.Fatalf("unconfigured model popover omitted unconfigured default header: %v", tt.Texts())
	}
	if tt.HasText("Current model") {
		t.Fatalf("unconfigured model popover should not label placeholder as Current model: %v", tt.Texts())
	}
	if !tt.HasText(i18n.T(i18n.KeyComposerModelUnavailable)) {
		t.Fatalf("catalog-less model popover omitted unavailable notice: %v", tt.Texts())
	}
	if !tt.HasText("Not signed in to a DeepSeek account or missing a model API key.") {
		t.Fatalf("unconfigured model popover omitted login/API-key guidance: %v", tt.Texts())
	}
	if err := tt.Click(i18n.T(i18n.KeyComposerModelReload)); err != nil || reloaded != 1 {
		t.Fatalf("reload action failed: err=%v reloaded=%d", err, reloaded)
	}

	// While ModelCatalogLoading is true, show loading state and block repeat clicks.
	snap.ModelCatalogLoading = true
	tt.Frame()
	if !tt.HasText(ModelSelectionLocaleText("trigger.loading")) || !tt.HasText(ModelSelectionLocaleText("status.loading")) {
		t.Fatalf("loading model popover omitted loading copy: %v", tt.Texts())
	}
	if err := tt.Click(ModelSelectionLocaleText("trigger.loading")); err != nil {
		t.Fatalf("expected disabled loading reload button in popover: %v", err)
	}
	if reloaded != 1 {
		t.Fatalf("disabled loading reload button fired again: reloaded=%d", reloaded)
	}

	// Once loading settles back to unconfigured, clicking Account & Model Settings opens settings and closes popover.
	snap.ModelCatalogLoading = false
	tt.Frame()
	if err := tt.Click("Account & Model Settings"); err != nil || openedSettings != 1 {
		t.Fatalf("settings shortcut action failed: err=%v openedSettings=%d", err, openedSettings)
	}
	if tt.HasText("Account & Model Settings") {
		t.Fatal("model seat popover remained open after navigating to settings")
	}

	// Chinese locale verifies localized guidance & header copy.
	i18n.SetLocale(i18n.LocaleZH)
	tt.Frame()
	if err := tt.Click("deepseek-official/deepseek-flash"); err != nil {
		t.Fatal(err)
	}
	if !tt.HasText("默认模型（未配置）") || !tt.HasText("尚未登录 DeepSeek 账号或配置模型 API Key") || !tt.HasText("前往账号/模型设置") {
		t.Fatalf("zh unconfigured guidance missing: %v", tt.Texts())
	}
}
