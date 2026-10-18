package main

import (
	"github.com/egoist/mygo"
	"strings"
	"testing"
	"whaleisle/internal/shortcuts"
)

func TestApplicationMenuRoutesAndBindings(t *testing.T) {
	for _, platform := range []string{"windows", "darwin", "linux"} {
		var called []string
		bindings := shortcuts.Resolve(map[string]string{"window.zoomIn": "Mod+Alt+9", "window.zoomOut": ""})
		menu := buildApplicationMenu(bindings, platform, func(id string) { called = append(called, id) })
		for _, id := range []string{"workspace.add", "workspace.openLocal", "launcher.open", "settings.open", "market.open", "settings.mcp", "settings.skills", "harness.restart", "window.refresh", "window.actualSize", "window.zoomIn", "window.zoomOut", "window.fullscreen", "shortcuts.open", "help.web", "help.sdk", "settings.about", "app.quit"} {
			row := menu.ItemByID(id)
			if row == nil || row.Click == nil {
				t.Fatalf("%s missing %s", platform, id)
			}
			if row.Accelerator != "" {
				t.Fatalf("duplicate keyboard owner: %s", id)
			}
			row.Click(row, nil)
			if called[len(called)-1] != id {
				t.Fatalf("wrong route %s", id)
			}
		}
		if !strings.Contains(menu.ItemByID("window.zoomIn").Label, "Mod+Alt+9") {
			t.Fatal("saved binding missing")
		}
		if strings.Contains(menu.ItemByID("window.zoomOut").Label, "Mod+") {
			t.Fatal("unbound command shows default")
		}
		if menu.ItemByID("window.inspector").Role != mygo.RoleToggleDevTools {
			t.Fatal("native inspector missing")
		}
		if platform == "darwin" && menu.ItemByID("window.close") == nil {
			t.Fatal("Mac close route missing")
		}
	}
}
