//go:build windows

package main

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/config"
	"whaleisle/internal/shortcuts"
)

// This fixture drives production menu models, state routes, preference storage,
// and the native UI input engine. It never installs an application/global menu,
// starts a Host, sends OS keys, or calls an external/destructive menu action.
func TestFunctionalOwnedAppCommands(t *testing.T) {
	checks := map[string]bool{}
	defer func() {
		if out := os.Getenv("WHALEISLE_APP_COMMANDS_EVIDENCE"); out != "" {
			report := struct {
				Passed bool
				Checks map[string]bool
				Scope  string
				NotRun []string
			}{!t.Failed(), checks, "Production mainState commands/menu model and native ui.Tester with fresh config only", []string{
				"OS menu mouse/keyboard gestures and global hotkeys", "native App.SetMenu installation", "real OS IME candidate window",
				"workspace picker/reveal, restart, quit, external help links, fullscreen OS transition", "original live-window pixel comparison",
			}}
			data, err := json.MarshalIndent(report, "", "  ")
			if err == nil {
				err = os.WriteFile(out, append(data, '\n'), 0600)
			}
			if err != nil {
				t.Errorf("save owned acceptance evidence: %v", err)
			}
		}
	}()

	t.Run("menu-and-navigation", func(t *testing.T) {
		cfg, err := config.Open(t.TempDir())
		if err != nil {
			t.Fatal(err)
		}
		s := &mainState{cfgStore: cfg}
		bindings := shortcuts.Resolve(s.pageShortcutBindings())
		encoded, _ := json.Marshal(bindings)
		// Only the platform's menu installation is excluded. Real production
		// pending-command consumption below still runs on each native frame.
		s.applicationMenu = &desktopApplicationMenu{signature: string(encoded) + resumeText("zh", "en")}
		menu := buildApplicationMenu(bindings, "windows", func(id string) {
			s.applicationMenu.pending = append(s.applicationMenu.pending, id)
		})
		tt := ui.NewTester(func(c *ui.Context) {
			s.syncApplicationMenu(c)
			ui.Button(c, "Owned main command scope").Width(220).Height(32)
		}, 600, 300)
		for _, id := range []string{"file", "edit", "run", "view", "help", "workspace.add", "workspace.openLocal", "harness.restart", "window.refresh", "window.actualSize", "window.zoomIn", "window.zoomOut", "window.fullscreen", "window.inspector", "help.web", "help.sdk", "app.quit"} {
			item := menu.ItemByID(id)
			if item == nil {
				t.Fatalf("missing menu identity %s", id)
			}
			if item.Role == "" && item.Accelerator != "" {
				t.Fatalf("menu creates second keyboard owner: %s", id)
			}
		}
		if menu.ItemByID("edit").Role != mygo.RoleEditMenu || menu.ItemByID("window.inspector").Role != mygo.RoleToggleDevTools {
			t.Fatal("native role identity changed")
		}
		for _, id := range []string{"launcher.open", "settings.open", "market.open", "settings.mcp", "settings.skills", "shortcuts.open", "settings.about"} {
			s.imOpen, s.remoteOpen, s.webviewOpen, s.browserPreviewOpen, s.assistantOpen = true, true, true, true, true
			item := menu.ItemByID(id)
			if item == nil || item.Disabled || item.Click == nil {
				t.Fatalf("safe navigation unavailable: %s", id)
			}
			item.Click(item, nil)
			tt.Frame()
			if len(s.applicationMenu.pending) != 0 {
				t.Fatalf("command not consumed once: %s", id)
			}
			if s.webviewOpen || s.browserPreviewOpen || s.assistantOpen {
				t.Fatalf("hidden route masks %s", id)
			}
			switch id {
			case "launcher.open":
				if !s.launcherOpen || s.settingsOpen || s.marketOpen || s.imOpen || s.remoteOpen {
					t.Fatal("launcher route")
				}
			case "market.open":
				if !s.marketOpen || s.settingsOpen || s.launcherOpen || s.imOpen || s.remoteOpen {
					t.Fatal("market route")
				}
			default:
				if !s.settingsOpen || s.launcherOpen || s.marketOpen {
					t.Fatalf("settings route %s", id)
				}
				if id == "settings.mcp" && !s.settingsMCPActive || id == "settings.skills" && !s.settingsSkillsActive || id == "shortcuts.open" && !s.settingsInterfaceActive || id == "settings.about" && !s.settingsAboutActive {
					t.Fatalf("wrong settings section %s", id)
				}
			}
		}
		checks["safe-production-menu-dispatch-and-section-navigation"] = true
		checks["menu-identities-native-roles-and-routed-keyboard-owner"] = true
	})

	t.Run("preferences-persistence-conflicts", func(t *testing.T) {
		root := t.TempDir()
		cfg, err := config.Open(root)
		if err != nil {
			t.Fatal(err)
		}
		s := &mainState{cfgStore: cfg}
		e := s.pageShortcutEditor()
		*e.Draft("settings.mcp"), *e.Draft("settings.skills") = "Mod+Alt+F8", "Mod+Alt+F9"
		*e.Draft("launcher.open") = ""
		if s.pageShortcutBindings()["settings.mcp"] != "" {
			t.Fatal("unsaved draft became active")
		}
		if !e.Save() {
			t.Fatal(e.Err)
		}
		freshCfg, err := config.Open(root)
		if err != nil {
			t.Fatal(err)
		}
		fresh := &mainState{cfgStore: freshCfg}
		if got := fresh.pageShortcutBindings(); got["settings.mcp"] != "Mod+Alt+F8" || got["settings.skills"] != "Mod+Alt+F9" || got["launcher.open"] != "" {
			t.Fatalf("reopen changed accepted binding: %v", got)
		}
		menu := buildApplicationMenu(shortcuts.Resolve(fresh.pageShortcutBindings()), "windows", nil)
		if !strings.Contains(menu.ItemByID("settings.mcp").Label, "Mod+Alt+F8") || strings.Contains(menu.ItemByID("launcher.open").Label, "Mod+") {
			t.Fatal("menu keycaps did not reflect binding/unbind")
		}
		before, _ := cfg.ShortcutDocument()
		*e.Draft("settings.mcp") = "Mod+Alt+F9"
		if e.Save() || !strings.Contains(e.Err, "conflict") {
			t.Fatal("colliding draft saved", e.Err)
		}
		after, _ := cfg.ShortcutDocument()
		if !bytes.Equal(before, after) || s.pageShortcutBindings()["settings.mcp"] != "Mod+Alt+F8" {
			t.Fatal("conflict changed accepted bytes")
		}
		e.Discard()
		*e.Draft("settings.mcp") = "Mod+S"
		if e.Save() {
			t.Fatal("reserved file-save chord accepted")
		}
		e.Discard()
		e.BeginRecording("settings.mcp")
		e.CancelRecording()
		if e.Recording != "" || *e.Draft("settings.mcp") != "Mod+Alt+F8" {
			t.Fatal("cancel recording changed binding")
		}
		e.BeginRecording("settings.mcp")
		if !e.Record("Mod+Alt+F10") || !e.Save() {
			t.Fatal("record/save failed", e.Err)
		}
		e.ResetDraft("launcher.open")
		if !e.Save() || s.pageShortcutBindings()["launcher.open"] != e.Default("launcher.open") {
			t.Fatal("reset did not restore default", e.Err)
		}
		checks["fresh-config-save-reopen-unbind-and-menu-keycaps"] = true
		checks["conflict-and-reserved-chord-retain-active-document"] = true
		checks["record-cancel-save-and-reset"] = true
		// Simulate an accepted external document with two conflicting rows:
		// both must become inactive without suppressing explicit menu clicks.
		raw, _ := cfg.ShortcutDocument()
		var doc map[string]any
		if json.Unmarshal(raw, &doc) != nil {
			t.Fatal("document")
		}
		profiles := doc["profiles"].(map[string]any)
		profile := profiles["desktop:"+shortcutPlatform()].(map[string]any)
		binding := map[string]any{"code": "F8", "modifiers": []string{"primary", "alt"}}
		profile[shortcuts.CanonicalID("settings.mcp")], profile[shortcuts.CanonicalID("settings.skills")] = binding, binding
		raw, _ = json.Marshal(doc)
		if err := cfg.SaveShortcutDocument(raw); err != nil {
			t.Fatal(err)
		}
		got := s.pageShortcutBindings()
		if got["settings.mcp"] != "" || got["settings.skills"] != "" {
			t.Fatal("conflicted bindings still executable", got)
		}
		checks["external-conflicts-disable-both-shortcuts"] = true
	})

	t.Run("native-input-priority", func(t *testing.T) {
		cfg, err := config.Open(t.TempDir())
		if err != nil {
			t.Fatal(err)
		}
		s := &mainState{cfgStore: cfg}
		e := s.pageShortcutEditor()
		*e.Draft("settings.mcp") = "Mod+Alt+F8"
		if !e.Save() {
			t.Fatal(e.Err)
		}
		text, modal := "preserve draft", false
		customKeys := 0
		tt := ui.NewTester(func(c *ui.Context) {
			s.registerApplicationCommands(c)
			ui.Column(c).Fill().Gap(8).Children(func() {
				ui.Button(c, "Neutral focus").Width(180).Height(30)
				ui.TextInput(c, &text).Label("Owned editor").Width(250).Height(30)
				ui.Text(c, "Owned selectable").Label("Owned selectable").Selectable().Focusable().Width(250).Height(30)
				ui.Box(c).Label("Owned custom input").Focusable().Width(250).Height(30).HandleInput(func(ev ui.InputEvent) bool {
					if ev.Kind == ui.InputKeyDown {
						customKeys++
					}
					return true
				})
			})
			ui.DialogBase(c, &modal, func(_, panel ui.Element) {
				panel.Width(220).Height(80).Children(func() { ui.Button(c, "Owned modal focus").Width(180).Height(30) })
			})
		}, 600, 400)
		tt.SetFocused(true)
		press := func() { tt.Key(ui.Cmd|ui.Alt, ui.KeyF8) }
		clearRoute := func() { s.settingsOpen, s.settingsMCPActive = false, false }
		if err := tt.Click("Neutral focus"); err != nil {
			t.Fatal(err)
		}
		press()
		if !s.settingsOpen || !s.settingsMCPActive {
			t.Fatal("accepted native binding did not navigate")
		}
		clearRoute()
		for _, focus := range []string{"Owned editor", "Owned selectable", "Owned custom input"} {
			if err := tt.Click(focus); err != nil {
				t.Fatal(err)
			}
			press()
			if s.settingsOpen {
				t.Fatalf("application shortcut stole %s input", focus)
			}
		}
		if customKeys == 0 {
			t.Fatal("custom input did not receive key")
		}
		if err := tt.Click("Owned editor"); err != nil {
			t.Fatal(err)
		}
		tt.Compose("ni", 2)
		press()
		if s.settingsOpen || text != "preserve draft" {
			t.Fatal("shortcut disturbed IME composition")
		}
		tt.Type("你")
		if text != "preserve draft你" {
			t.Fatalf("native IME commit lost text: %q", text)
		}
		if err := tt.Click("Neutral focus"); err != nil {
			t.Fatal(err)
		}
		tt.SetFocused(false)
		press()
		if s.settingsOpen {
			t.Fatal("unfocused window executed app shortcut")
		}
		tt.SetFocused(true)
		modal = true
		tt.Frame()
		press()
		if s.settingsOpen {
			t.Fatal("modal allowed app shortcut")
		}
		modal = false
		tt.Frame()
		if err := tt.Click("Neutral focus"); err != nil {
			t.Fatal(err)
		}
		press()
		if !s.settingsOpen || !s.settingsMCPActive {
			t.Fatal("dismissal did not restore app shortcut")
		}
		checks["production-app-shortcut-navigates"] = true
		checks["editor-selectable-custom-input-and-ime-priority"] = true
		checks["unfocused-window-and-modal-block-app-shortcut"] = true
	})

	t.Run("menu-role-binding-owner", func(t *testing.T) {
		cfg, err := config.Open(t.TempDir())
		if err != nil {
			t.Fatal(err)
		}
		s := &mainState{cfgStore: cfg}
		bindings := shortcuts.Resolve(s.pageShortcutBindings())
		menu := buildApplicationMenu(bindings, "windows", nil)
		inspector := menu.ItemByID("window.inspector")
		if inspector == nil {
			t.Fatal("inspector menu item missing")
		}
		if inspector.Accelerator != "" {
			// MyGo fills this native role accelerator even though the template
			// did not specify one. Compare the normalized production menu model
			// with accepted app-only bindings, without emitting any OS gesture.
			raw := strings.ReplaceAll(inspector.Accelerator, "Ctrl", "Mod")
			role, err := shortcuts.Parse(raw)
			if err == nil {
				for id, chord := range bindings {
					if chord == role {
						t.Fatalf("native inspector accelerator %s duplicates accepted application command %s (%s)", inspector.Accelerator, id, chord.String())
					}
				}
			}
		}
		checks["native-menu-role-does-not-duplicate-accepted-application-binding"] = true
	})
}
