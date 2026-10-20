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
