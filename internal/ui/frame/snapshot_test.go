package frame

import (
	"encoding/json"
	"fmt"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	desktopgit "whaleisle/internal/desktop/git"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

// Software-rendered shell evidence, with real text/footer bounds. No model or
// terminal process is started; a nil terminal draws the existing unavailable UI.
func TestShellSoftwareSnapshots(t *testing.T) {
	dir := os.Getenv("SHELL_SNAPSHOT_DIR")
	if dir == "" {
		t.Skip("set SHELL_SNAPSHOT_DIR to export software renders")
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	i18n.SetLocale("en")
	defer i18n.SetLocale(i18n.DefaultLocale)
	preferences := theme.CurrentPreferences()
	theme.SetPreferences(theme.Defaults())
	defer theme.SetPreferences(preferences)
	trailing := func(terminal bool) func(*ui.Context, string) {
		return func(c *ui.Context, density string) {
			chrome.WorkspaceTrailing(c, chrome.WorkspaceTrailingModel{
				Status:        &desktopgit.RepositoryStatus{IsRepo: true, Branch: "main", HasOrigin: true, HasUpstream: true, IsDefault: true},
				WorkspaceRoot: "C:/isolated-fixture", Density: density, GitLoaded: true,
				ShowTerminal: true, TerminalAvailable: true, TerminalOpen: terminal, ShowSurfaces: true, SurfacesOpen: true,
				OnToggleTerminal: func() {}, OnToggleSurfaces: func() {}, OnGitAction: func(string) {},
				RenderBranches: func(*ui.Context, ui.Element, *bool) {},
			})
		}
	}
	type record struct {
		Name                                        string `json:"name"`
		Width, Height                               int
		Center, Right, RememberedRight              float32
		TerminalTitle, SidebarFooter, RightFilesTab ui.Rect
		ComposerCard, UserBubble                    ui.Rect
	}
	var records []record
	save := func(name string, tt *ui.Tester, sizes *LayoutSizes, right float32, w, h int) {
		t.Helper()
		f, err := os.Create(filepath.Join(dir, name+".png"))
		if err != nil {
			t.Fatal(err)
		}
		if err = png.Encode(f, tt.Image()); err != nil {
			f.Close()
			t.Fatal(err)
		}
		if err = f.Close(); err != nil {
			t.Fatal(err)
		}
		term, _ := tt.Find(i18n.T(i18n.KeyTerminalTitle))
		footer, _ := tt.Find(i18n.T(i18n.KeySettingsOpen))
		files, _ := tt.Find(i18n.T(i18n.KeyRightPaneFiles))
		card, _ := tt.Find("Composer card")
		bubble, _ := tt.Find("User message bubble")
		records = append(records, record{ComposerCard: card, UserBubble: bubble, Name: name, Width: w, Height: h, Center: sizes.Center, Right: right, RememberedRight: sizes.Right, TerminalTitle: term, SidebarFooter: footer, RightFilesTab: files})
	}
	for _, width := range []int{900, 1200, 1600} {
		for _, dark := range []bool{false, true} {
			for _, terminal := range []bool{false, true} {
				sizes := DefaultLayout()
				sizes.Right = 540
				if !terminal {
					sizes.Bottom = 0
				}
				right := float32(0)
				tt := ui.NewTester(func(c *ui.Context) {
					theme.ApplyInterface(c)
					originalScrollbarTheme(c)
					View(c, Model{Store: session.New(), Sizes: &sizes, RightPaneOpen: true, FilePaneWidth: &right, SystemDark: dark, OnOpenSettings: func() {}, WorkspaceTrailing: trailing(terminal)})
				}, width, 800)
				save(fmt.Sprintf("shell-%d-dark-%v-terminal-%v", width, dark, terminal), tt, &sizes, right, width, 800)
			}
		}
	}
	for _, width := range []int{900, 1200, 1600, 2200} {
		for _, dark := range []bool{false, true} {
			sizes := DefaultLayout()
			sizes.Right = 540
			sizes.Bottom = 0
			right := float32(0)
			draft := "A composer fixture"
			snap := session.ViewState{SelectedID: "current", HistoryReady: true, Items: []session.Item{{SessionID: "current", Title: "Current conversation"}}, Messages: []session.Message{{ID: "snapshot-user", Seq: 1, Kind: session.MessageKindText, Role: "user", Body: strings.Repeat("A literal user message with a deliberately long line. ", 18)}, {ID: "snapshot-assistant", Seq: 2, Kind: session.MessageKindText, Role: "assistant", Body: "Assistant fixture with a short response."}}}
			tt := ui.NewTester(func(c *ui.Context) {
				theme.ApplyInterface(c)
				originalScrollbarTheme(c)
				cols := sizes.ResolveColumns(float32(width), true)
				right = cols.Details
				colors := theme.Resolve("system").Active(dark)
				m := Model{Store: session.New(), Sizes: &sizes, FilePaneWidth: &right, RightPaneOpen: true, Draft: &draft, SystemDark: dark, OnOpenSettings: func() {}, WorkspaceTrailing: trailing(false)}
				ui.Column(c).Fill().Background(chrome.Color(colors.BG)).Children(func() { mainRow(c, m, snap, colors, chrome.Color(colors.Border), chrome.Color(colors.Muted)) })
			}, width, 800)
			save(fmt.Sprintf("populated-%d-dark-%v", width, dark), tt, &sizes, right, width, 800)
		}
	}

	sizes := DefaultLayout()
	right := float32(0)
	open := true
	tt := ui.NewTester(func(c *ui.Context) {
		theme.ApplyInterface(c)
		originalScrollbarTheme(c)
		View(c, Model{Store: session.New(), Sizes: &sizes, RightPaneOpen: open, FilePaneWidth: &right, OnOpenSettings: func() {}, WorkspaceTrailing: trailing(true)})
	}, 1600, 800)
	save("first-open-1600", tt, &sizes, right, 1600, 800)
	tt.SetSize(900, 800)
	save("resize-900-remember720", tt, &sizes, right, 900, 800)
	tt.SetSize(1600, 800)
	save("resize-1600-restored720", tt, &sizes, right, 1600, 800)
	open = false
	tt.Frame()
	save("explicit-close-remember720", tt, &sizes, right, 1600, 800)
	open = true
	tt.Frame()
	save("explicit-reopen-restored720", tt, &sizes, right, 1600, 800)
	tt.SetSize(1200, 400)
	save("terminal-height-shrink", tt, &sizes, right, 1200, 400)
	data, err := json.MarshalIndent(records, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(dir, "rendered-bounds.json"), data, 0644); err != nil {
		t.Fatal(err)
	}
}

// Match the verified ui-theme scrollbar.css fixture; toolkit defaults to six.
func originalScrollbarTheme(c *ui.Context) {
	th := *c.Theme()
	th.ScrollbarWidth = conversationScrollbarWidth
	c.SetTheme(&th)
}
