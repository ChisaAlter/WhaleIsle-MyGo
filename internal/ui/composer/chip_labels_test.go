package composer

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/dsh"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
)

func permissionModelFixture() Model {
	permission := "workspace-write"
	d := NewDraftState("")
	return Model{State: d, Draft: &d.Text, TextInput: d.Client, UseTextInputClient: true,
		SuggestList:    &ui.ListState{Selected: new(int)},
		PermissionPick: &permission,
		PermissionOptions: []session.PermissionOption{
			{Label: "read-only", Value: "read-only"},
			{Label: "workspace-write", Value: "workspace-write"},
			{Label: session.PermissionFullAccess, Value: session.PermissionFullAccess},
		},
		Snapshot: session.ViewState{
			SelectedID:         "s",
			ClientReady:        true,
			SelectedPermission: "workspace-write",
			NativeContributions: dsh.NativeContributions{
				Known: true, ModelCommand: true, PermissionCommand: true,
			},
			SelectedModel:       "DeepSeek V4.1 Flash",
			SelectedModelEffort: "Max",
			ModelLabels:         []string{"DeepSeek V4.1 Flash", "DeepSeek V4.1 Pro"},
		}}
}

func TestPermissionChipShowsProductLabels(t *testing.T) {
	picked := ""
	m := permissionModelFixture()
	m.OnPermissionChange = func(value string) { picked = value }
	tt := ui.NewTester(func(c *ui.Context) { View(c, m) }, 880, 420)
	if !tt.HasText(i18n.T(i18n.KeyComposerPermissionWorkspace)) {
		t.Fatal("permission chip does not show its product name")
	}
	for _, raw := range []string{"workspace-write", "read-only", session.PermissionFullAccess} {
		if tt.HasText(raw) {
			t.Fatalf("permission chip shows the Host value %q", raw)
		}
	}
	if err := tt.Click(i18n.T(i18n.KeyComposerPermissionWorkspace)); err != nil {
		t.Fatal(err)
	}
	if !tt.HasText(i18n.T(i18n.KeyComposerPermissionFullAccess)) || !tt.HasText(i18n.T(i18n.KeyComposerPermissionReadOnly)) {
		t.Fatal("permission popup does not list the product names")
	}
	if err := tt.Click(i18n.T(i18n.KeyComposerPermissionFullAccess)); err != nil {
		t.Fatal(err)
	}
	if picked != session.PermissionFullAccess {
		t.Fatalf("pick sent %q, want the opaque value", picked)
	}
}

func TestModelChipKeepsItsAffordanceWithoutCatalog(t *testing.T) {
	m := permissionModelFixture()
	m.Snapshot.ModelLabels = nil
	m.Snapshot.SelectedModel = "deepseek-official / deepseek-flash"
	reloads := 0
	m.OnRetryModels = func() { reloads++ }
	tt := ui.NewTester(func(c *ui.Context) { View(c, m) }, 880, 420)
	if !tt.HasText("deepseek-official / deepseek-flash") {
		t.Fatal("model seat lost the selected route")
	}
	if err := tt.Click("deepseek-official / deepseek-flash"); err != nil {
		t.Fatal("model seat is not a control:", err)
	}
	if !tt.HasText(i18n.T(i18n.KeyComposerModelUnavailable)) {
		t.Fatal("model seat did not open its empty state")
	}
	if err := tt.Click(i18n.T(i18n.KeyComposerModelReload)); err != nil {
		t.Fatal(err)
	}
	if reloads != 1 {
		t.Fatalf("reload ran %d times", reloads)
	}
}

func TestModelChipShowsNameAndEffort(t *testing.T) {
	m := permissionModelFixture()
	tt := ui.NewTester(func(c *ui.Context) { View(c, m) }, 880, 420)
	if !tt.HasText("DeepSeek V4.1 Flash") || !tt.HasText("Max") {
		t.Fatal("model chip lost its name or effort suffix")
	}
	if err := tt.Click("Model catalog"); err != nil {
		t.Fatal("model chip is not clickable:", err)
	}
	if !tt.HasText(i18n.T(i18n.KeyComposerModelMenuModel) + "  " + "DeepSeek V4.1 Flash" + "  ›") {
		t.Fatal("model menu did not open on the selected model")
	}
}

func TestPermissionPopupGeometryMatchesReference(t *testing.T) {
	m := permissionModelFixture()
	m.OnPermissionChange = func(string) {}
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Size(880, 420).Justify(ui.End).Children(func() {
			View(c, m)
		})
	}, 880, 420)
	triggerLabel := composerText("访问模式，当前：", "Access mode, current: ") + i18n.T(i18n.KeyComposerPermissionWorkspace)
	triggerBounds, ok := tt.Find(triggerLabel)
	if !ok {
		t.Fatal("missing Permission mode trigger")
	}
	if err := tt.Click(triggerLabel); err != nil {
		t.Fatal(err)
	}
	rowBounds, ok := tt.Find(i18n.T(i18n.KeyComposerPermissionReadOnly))
	if !ok {
		t.Fatal("missing permission row")
	}
	if rowBounds.H != 34 {
		t.Fatalf("permission row height = %v, want 34", rowBounds.H)
	}
	if rowBounds.W < 135 || rowBounds.W > 240 {
		t.Fatalf("permission row width = %v, want compact content-hugging width (135..240)", rowBounds.W)
	}
	// Panel has 4 DIP padding + 0.5 DIP border on each side, so row X is trigger.X + 4.5.
	if rowBounds.X < triggerBounds.X+4 || rowBounds.X > triggerBounds.X+5 {
		t.Fatalf("permission row X = %v, want trigger.X + 4.5 (%v)", rowBounds.X, triggerBounds.X+4.5)
	}
	if rowBounds.Y+rowBounds.H > triggerBounds.Y {
		t.Fatalf("permission popup did not open above trigger: row=%+v trigger=%+v", rowBounds, triggerBounds)
	}
}

