package main

import (
	"strings"
	"testing"

	"github.com/egoist/mygo"
	"whaleisle/internal/shortcuts"
	"whaleisle/internal/ui/i18n"
)

// menuRows flattens a submenu into id rows, "-" for a separator.
func menuRows(items []*mygo.MenuItem) []string {
	out := []string{}
	for _, item := range items {
		switch {
		case item.Type == mygo.MenuItemSeparator:
			out = append(out, "-")
		case len(item.Submenu) > 0:
			out = append(out, "["+item.ID+"]")
		default:
			out = append(out, item.ID)
		}
	}
	return out
}

func buildTestMenu(t *testing.T, spec applicationMenuSpec) *mygo.Menu {
	t.Helper()
	if spec.Platform == "" {
		spec.Platform = "windows"
	}
	if spec.Fallbacks == nil {
		spec.Fallbacks = shortcuts.Resolve(shortcuts.SharedDefaults())
	}
	return buildApplicationMenu(spec)
}

// keycapOf returns the part after the label/keycap gap, empty when there is none.
func keycapOf(t *testing.T, menu *mygo.Menu, id string) string {
	t.Helper()
	row := menu.ItemByID(id)
	if row == nil {
		t.Fatalf("missing %s", id)
	}
	_, keycap, _ := strings.Cut(row.Label, applicationMenuKeycapGap)
	return keycap
}

// TestApplicationMenuTemplateMatchesNative pins the native template row by row
// (reference/electron/src/main/menu.js:22-121): File / Run / View / Help in that
// order with the same separators, and no invented rows.
func TestApplicationMenuTemplateMatchesNative(t *testing.T) {
	menu := buildTestMenu(t, applicationMenuSpec{})
	if got, want := strings.Join(menuRows(menu.Items()), ","), "[file],[run],[view],[help]"; got != want {
		t.Fatalf("top level = %s, want %s", got, want)
	}
	// The native template has no Edit menu and, off macOS, no application menu.
	if menu.ItemByID("edit") != nil || menu.ItemByID("app") != nil {
		t.Fatalf("non-native top-level rows: %v", menuRows(menu.Items()))
	}
	wantFile := "workspace.add,launcher.open,workspace.reveal,-,settings.open,market.open,settings.mcp,settings.skills,-,app.quit"
	if got := strings.Join(menuRows(menu.ItemByID("file").Submenu), ","); got != wantFile {
		t.Fatalf("File = %s, want %s", got, wantFile)
	}
	wantRun := "harness.restart,window.refresh"
	if got := strings.Join(menuRows(menu.ItemByID("run").Submenu), ","); got != wantRun {
		t.Fatalf("Run = %s, want %s", got, wantRun)
	}
	wantView := "window.actualSize,window.zoomIn,window.zoomOut,-,window.fullscreen,window.inspector"
	if got := strings.Join(menuRows(menu.ItemByID("view").Submenu), ","); got != wantView {
		t.Fatalf("View = %s, want %s", got, wantView)
	}
	wantHelp := "help.web,help.sdk,-,settings.about"
	if got := strings.Join(menuRows(menu.ItemByID("help").Submenu), ","); got != wantHelp {
		t.Fatalf("Help = %s, want %s", got, wantHelp)
	}
	if menu.ItemByID("window.inspector").Role != mygo.RoleToggleDevTools {
		t.Fatal("developer tools must keep the native role")
	}
	// The rows the native template does not have stay out of the menu; their
	// commands remain reachable through their own bindings.
	for _, id := range []string{"workspace.openLocal", "shortcuts.open", "edit"} {
		if menu.ItemByID(id) != nil {
			t.Fatalf("%s is not in the native template", id)
		}
	}
}

// TestApplicationMenuLabelsMatchNative checks the copy in both catalogs against
// the native template's Chinese labels.
func TestApplicationMenuLabelsMatchNative(t *testing.T) {
	cases := []struct{ id, zh, en string }{
		{"workspace.add", "打开工作区…", "Open workspace…"},
		{"launcher.open", "打开启动器", "Open launcher"},
		{"workspace.reveal", "在资源管理器中打开工作区", "Open workspace in explorer"},
		{"settings.open", "设置…", "Settings…"},
		{"market.open", "插件市场…", "Marketplace…"},
		{"settings.mcp", "MCP…", "MCP…"},
		{"settings.skills", "技能…", "Skills…"},
		{"harness.restart", "重启 Harness", "Restart Harness"},
		{"window.refresh", "重新加载界面", "Reload interface"},
		{"window.actualSize", "实际大小", "Actual size"},
		{"window.zoomIn", "放大", "Zoom in"},
		{"window.zoomOut", "缩小", "Zoom out"},
		{"window.fullscreen", "全屏", "Fullscreen"},
		{"window.inspector", "开发者工具", "Developer tools"},
		{"help.web", "Web UI 指南", "Web UI guide"},
		{"help.sdk", "Python SDK", "Python SDK"},
		{"settings.about", "关于", "About"},
		{"app.quit", "退出", "Quit"},
	}
	defer i18n.SetLocale(i18n.DefaultLocale)
	for _, locale := range []string{i18n.LocaleZH, i18n.LocaleEN} {
		i18n.SetLocale(locale)
		menu := buildTestMenu(t, applicationMenuSpec{})
		for _, c := range cases {
			want := c.zh
			if locale == i18n.LocaleEN {
				want = c.en
			}
			row := menu.ItemByID(c.id)
			if row == nil {
				t.Fatalf("%s: missing %s", locale, c.id)
			}
			if !strings.HasPrefix(row.Label, want) {
				t.Fatalf("%s %s label = %q, want prefix %q", locale, c.id, row.Label, want)
			}
		}
	}
}

// TestApplicationMenuKeycapsFollowTheRevision covers the native commandItem rule
// (menu.js:9-19): the accepted binding is the only keycap once a revision
// exists, an unbound command shows none, and before the first snapshot the
// shipped default keeps the menu readable.
func TestApplicationMenuKeycapsFollowTheRevision(t *testing.T) {
	// Start from the shipped shared defaults so the compatibility row
	// workspace.open (also Mod+O there) stays unbound and cannot shadow the
	// command under test.
	values := shortcuts.SharedDefaults()
	values["window.zoomIn"] = "Mod+Alt+9"
	values["window.zoomOut"] = ""
	values["workspace.add"] = "Mod+O"
	values["settings.open"] = "Mod+,"
	values["launcher.open"] = "Mod+Shift+L"
	values["settings.mcp"] = "Mod+Alt+1"
	values["market.open"] = "Mod+Shift+M"
	values["harness.restart"] = "Mod+Alt+F5"
	values["window.refresh"] = "Mod+F5"
	values["window.actualSize"] = "Mod+0"
	values["window.fullscreen"] = "Mod+Shift+I"
	bindings := shortcuts.Resolve(values)
	menu := buildTestMenu(t, applicationMenuSpec{Bindings: bindings, HasRevision: true})
	if got := keycapOf(t, menu, "workspace.add"); got != "Mod+O" {
		t.Fatalf("workspace.add keycap = %q, want the accepted Mod+O", got)
	}
	if got := keycapOf(t, menu, "settings.open"); got != "Mod+," {
		t.Fatalf("settings.open keycap = %q", got)
	}
	if got := keycapOf(t, menu, "window.zoomIn"); got != "Mod+Alt+9" {
		t.Fatalf("zoomIn keycap = %q, want the accepted chord", got)
	}
	if got := keycapOf(t, menu, "window.zoomOut"); got != "" {
		t.Fatalf("unbound zoomOut shows %q, want no keycap", got)
	}
	// Rows the native template gives no accelerator never show one, even when a
	// page binding exists: the menu is not the accelerator owner.
	for _, id := range []string{"launcher.open", "workspace.reveal", "settings.mcp", "settings.skills", "help.web", "help.sdk", "settings.about", "app.quit"} {
		if got := keycapOf(t, menu, id); got != "" {
			t.Fatalf("%s shows %q, want no keycap", id, got)
		}
	}
	// Before the first accepted revision the shipped defaults stand in for the
	// native fallbackAccelerator.
	before := buildTestMenu(t, applicationMenuSpec{HasRevision: false})
	if got := keycapOf(t, before, "workspace.add"); got != "Mod+O" {
		t.Fatalf("before the first revision workspace.add shows %q, want the shipped Mod+O", got)
	}
	if got := keycapOf(t, before, "settings.open"); got != "Mod+," {
		t.Fatalf("before the first revision settings.open shows %q", got)
	}
	if got := keycapOf(t, before, "settings.mcp"); got != "" {
		t.Fatalf("settings.mcp shows %q before any revision, want no keycap", got)
	}
}

// TestApplicationMenuDispatchRoutesEachRow checks every row's click, and that no
// row installs an accelerator: MyGo's own shortcut owner must stay the only one.
func TestApplicationMenuDispatchRoutesEachRow(t *testing.T) {
	for _, platform := range []string{"windows", "linux", "darwin"} {
		var called []string
		menu := buildTestMenu(t, applicationMenuSpec{
			Bindings:    shortcuts.Resolve(shortcuts.SharedDefaults()),
			HasRevision: true,
			Platform:    platform,
			Dispatch:    func(id string) { called = append(called, id) },
		})
		rows := 0
		var walk func(items []*mygo.MenuItem)
		walk = func(items []*mygo.MenuItem) {
			for _, item := range items {
				if len(item.Submenu) > 0 {
					walk(item.Submenu)
					continue
				}
				if item.Type == mygo.MenuItemSeparator || item.Role != "" {
					continue
				}
				rows++
				if item.Accelerator != "" {
					t.Fatalf("%s leaves an accelerator on %s: two keyboard owners", platform, item.ID)
				}
				before := len(called)
				item.Click(item, nil)
				if len(called) != before+1 || called[len(called)-1] != item.ID {
					t.Fatalf("%s dispatched %q for %s", platform, called[len(called)-1], item.ID)
				}
			}
		}
		walk(menu.Items())
		if rows == 0 {
			t.Fatal("no actionable rows")
		}
		if platform == "darwin" {
			app := menu.ItemByID("app")
			if app == nil || len(app.Submenu) != 3 || app.Submenu[1].Type != mygo.MenuItemSeparator {
				t.Fatalf("macOS app menu = %+v", app)
			}
			file := menuRows(menu.ItemByID("file").Submenu)
			if len(file) == 0 || file[len(file)-1] != "window.close" {
				t.Fatalf("macOS File must end with Close window: %v", file)
			}
			if menu.ItemByID("app.quit") == nil {
				t.Fatal("macOS app menu keeps Quit")
			}
			continue
		}
		file := menuRows(menu.ItemByID("file").Submenu)
		if len(file) == 0 || file[len(file)-1] != "app.quit" || menu.ItemByID("window.close") != nil {
			t.Fatalf("%s must end File with Quit and no Close window row: %v", platform, file)
		}
	}
}

// TestApplicationMenuEveryRowHasAnAction keeps the menu from painting a row whose
// gesture the shell cannot run.
func TestApplicationMenuEveryRowHasAnAction(t *testing.T) {
	state := &mainState{}
	actions := state.applicationMenuActions(nil)
	menu := buildTestMenu(t, applicationMenuSpec{})
	missing := []string{}
	var walk func(items []*mygo.MenuItem)
	walk = func(items []*mygo.MenuItem) {
		for _, item := range items {
			if len(item.Submenu) > 0 {
				walk(item.Submenu)
				continue
			}
			if item.Type == mygo.MenuItemSeparator || item.Role != "" {
				continue
			}
			if actions[item.ID] == nil {
				missing = append(missing, item.ID)
			}
		}
	}
	walk(menu.Items())
	if len(missing) > 0 {
		t.Fatalf("menu rows without an action: %v", missing)
	}
	// Reveal is wired to the configured workspace, like native menu.js:44-52,
	// and does nothing when none is configured.
	actions["workspace.reveal"]()
}
