//go:build windows && inspectorfixture

package frame

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/chat"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

// Opt-in desktop fixture, using the same raw records as the Electron probe.
// Build: go test -tags inspectorfixture -c -o .devdata/inspector-fixture.exe ./internal/ui/frame
// Set INSPECTOR_NATIVE_FIXTURE to the absolute testdata/inspector-records.json path.
func TestMain(m *testing.M) {
	path := os.Getenv("INSPECTOR_NATIVE_FIXTURE")
	if path == "" {
		os.Exit(m.Run())
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		panic(err)
	}
	var records []struct {
		Key, Parent, Type string
		Seq, Time         float64
		Value             json.RawMessage
	}
	if err = json.Unmarshal(raw, &records); err != nil {
		panic(err)
	}
	source := session.InspectorSnapshot{Binding: "round102-fixture", SessionID: "fixture"}
	for _, r := range records {
		source.Records = append(source.Records, session.InspectorRecord{Key: r.Key, Parent: r.Parent, Type: r.Type, Seq: r.Seq, Time: r.Time, Raw: r.Value})
	}
	var phases []struct {
		Name   string
		Frames []json.RawMessage
	}
	phase := 0
	var advance func()
	if scenario := os.Getenv("INSPECTOR_NATIVE_FIXTURE_SCENARIO"); scenario == "stream" || scenario == "scroll" {
		if err = json.Unmarshal(raw, &phases); err != nil {
			panic(err)
		}
		store := session.New()
		store.Select("fixture")
		advance = func() {
			if phase >= len(phases) {
				return
			}
			for _, frame := range phases[phase].Frames {
				store.ApplyInspectorFixtureFrame(frame)
			}
			source = store.InspectorSnapshot()
			if prefix := os.Getenv("INSPECTOR_NATIVE_FIXTURE_OUTPUT"); prefix != "" {
				projection, _ := json.MarshalIndent(source.Records, "", "  ")
				if err := os.WriteFile(prefix+"-projection-"+phases[phase].Name+".json", projection, 0600); err != nil {
					panic(err)
				}
			}
			phase++
		}
		advance()
	}
	p := SessionInspectorPaneState{collapsed: map[string]inspectorDisclosure{}}
	i18n.SetLocale(i18n.LocaleEN)
	theme.SetPreferences(theme.Defaults())
	colors := theme.Default().Light
	view := ui.View(func(c *ui.Context) {
		theme.ApplyInterface(c)
		if advance != nil && c.Shortcut(0, ui.KeyF8) {
			advance()
		}
		if os.Getenv("INSPECTOR_NATIVE_FIXTURE_SCENARIO") == "scroll" {
			if c.Shortcut(0, ui.KeyF9) {
				prefix := os.Getenv("INSPECTOR_NATIVE_FIXTURE_OUTPUT")
				metrics, _ := json.MarshalIndent(map[string]any{"offset": p.tableY.Y, "max": p.tableY.MaxY, "padding": p.scroll.layout.bottomPadding, "mode": p.scroll.mode, "phase": phase, "viewportHeight": float32(len(p.scroll.layout.rows)+1)*inspectorRowHeight + p.scroll.layout.bottomPadding - p.tableY.MaxY}, "", "  ")
				if err := os.WriteFile(prefix+"-metrics.json", metrics, 0600); err != nil {
					panic(err)
				}
			}
		}
		if os.Getenv("INSPECTOR_NATIVE_FIXTURE_SCENARIO") == "interaction" {
			// Emulate the Chat-owned pick handoff without connecting a Host.
			if c.Shortcut(0, ui.KeyF6) {
				p.navigation.Picked = &chat.InspectorTarget{Seq: 4}
			}
			if c.Shortcut(0, ui.KeyF7) {
				p.navigation.Picked = &chat.InspectorTarget{Seq: 3}
			}
		}
		ui.Column(c).Fill().Background(chrome.Color(colors.BG)).Children(func() { inspectorPresentation(c, &p, source, session.ViewState{}, colors, nil) })
	})
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{Title: "Native inspector fixture", Width: 780, Height: 600, UseContentSize: true, Content: view, Frameless: true, BackgroundColor: "#ffffff"})
	})
	if err = mygo.App.Run(); err != nil {
		panic(err)
	}
}
