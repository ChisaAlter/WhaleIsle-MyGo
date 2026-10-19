package workspacepicker

import (
	"fmt"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

func TestPickerAnchorsAndUsesWorkspaceIdentity(t *testing.T) {
	i18n.SetLocale(i18n.LocaleEN)
	defer i18n.SetLocale(i18n.DefaultLocale)
	store := session.New()
	request := &session.WorkspaceNavigationState{Token: 42, Mode: "pick"}
	snap := session.ViewState{WorkspaceReady: true, Workspaces: []session.WorkspaceRow{{WorkspaceID: "one", Title: "First", Path: "/host/one"}, {WorkspaceID: "two", Title: strings.Repeat("Second workspace ", 12), Path: "/host/two"}}}
	var chosen string
	var token uint64
	var anchor ui.Element
	browse := 0
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Padding(40).Children(func() {
			anchor = ui.Button(c, "").Label("Workspace chip").Width(160).Height(32)
			anchor.Children(func() { ui.Text(c, "Workspace") })
		})
		pickMenu(c, Model{Store: store, Anchor: anchor, OnChoose: func(value uint64, id string) { token, chosen = value, id }, OnBrowse: func() { browse++ }}, snap, request, theme.Resolve("system").Light)
	}, 640, 480)
	chip, _ := tt.Find("Workspace chip")
	first, _ := tt.Find("First")
	if first.Y < chip.Y+chip.H || first.X < chip.X || first.W > 280 || first.H != 34 {
		t.Fatalf("unanchored or expanded row: chip=%+v first=%+v", chip, first)
	}
	if !tt.Focused("First") {
		t.Fatal("first choice lacks initial focus")
	}
	tt.Key(0, ui.KeyDown)
	if !tt.Focused(snap.Workspaces[1].Title) {
		t.Fatal("down arrow did not focus next workspace")
	}
	tt.Key(0, ui.KeyEnter)
	if token != 42 || chosen != "two" {
		t.Fatalf("selection token=%d workspace=%q", token, chosen)
	}
	if err := tt.Click("Add workspace…"); err != nil {
		t.Fatal(err)
	}
	if browse != 1 {
		t.Fatalf("browse count=%d", browse)
	}
}

func TestPickerLongListKeepsActionsPinned(t *testing.T) {
	i18n.SetLocale(i18n.LocaleEN)
	defer i18n.SetLocale(i18n.DefaultLocale)
	snap := session.ViewState{WorkspaceReady: true}
	for index := 0; index < 30; index++ {
		snap.Workspaces = append(snap.Workspaces, session.WorkspaceRow{WorkspaceID: fmt.Sprint(index), Title: fmt.Sprintf("Workspace %02d", index)})
	}
	store := session.New()
	tt := ui.NewTester(func(c *ui.Context) {
		anchor := ui.Button(c, "Chip").Height(32)
		pickMenu(c, Model{Store: store, Anchor: anchor, OnBrowse: func() {}, OnNoDirectory: func() {}}, snap, &session.WorkspaceNavigationState{Mode: "pick"}, theme.Resolve("system").Dark)
	}, 360, 480)
	footer, ok := tt.Find("Add workspace…")
	if !ok || footer.Y+footer.H > 480 {
		t.Fatalf("pinned add action outside viewport: %+v", footer)
	}
	first, _ := tt.Find("Workspace 00")
	tt.Scroll(first.X+first.W/2, first.Y+first.H/2, 0, 2000)
	last, ok := tt.Find("Workspace 29")
	if !ok || last.Y+last.H > footer.Y {
		t.Fatalf("last workspace unreachable: last=%+v footer=%+v", last, footer)
	}
	after, _ := tt.Find("Add workspace…")
	if after != footer {
		t.Fatalf("add action moved: %+v -> %+v", footer, after)
	}
	saveSecondarySnapshot(t, tt, "workspace-picker-long-list.png")
}

func TestRenameFocusAndBusyDismissal(t *testing.T) {
	i18n.SetLocale(i18n.LocaleEN)
	defer i18n.SetLocale(i18n.DefaultLocale)
	store := session.New()
	request := &session.WorkspaceNavigationState{Token: 1, Mode: "rename", TargetID: "one", Draft: "Workspace name"}
	tt := ui.NewTester(func(c *ui.Context) {
		managementDialog(c, Model{Store: store}, session.ViewState{Workspaces: []session.WorkspaceRow{{WorkspaceID: "one", Title: "Workspace name"}}}, request, theme.Resolve("system").Light)
	}, 500, 360)
	if !tt.Focused("Workspace name") {
		t.Fatal("rename field was not focused")
	}
	input, _ := tt.Find("Workspace name")
	if input.H != 44 {
		t.Fatalf("rename input=%+v", input)
	}
	saveSecondarySnapshot(t, tt, "workspace-rename.png")
	request.Busy = true
	tt.Frame()
	tt.Key(0, ui.KeyEscape)
	if !tt.HasText("Waiting for the Host…") {
		t.Fatal("busy rename disappeared after Escape")
	}
}

func saveSecondarySnapshot(t *testing.T, tt *ui.Tester, name string) {
	t.Helper()
	dir := os.Getenv("SECONDARY_SNAPSHOT_DIR")
	if dir == "" {
		return
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	file, err := os.Create(filepath.Join(dir, name))
	if err != nil {
		t.Fatal(err)
	}
	encodeErr := png.Encode(file, tt.Image())
	closeErr := file.Close()
	if encodeErr != nil {
		t.Fatal(encodeErr)
	}
	if closeErr != nil {
		t.Fatal(closeErr)
	}
}
