//go:build windows && inspectorfixture

package frame

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
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
	p := SessionInspectorPaneState{collapsed: map[string]bool{}}
	i18n.SetLocale(i18n.LocaleEN)
	theme.SetPreferences(theme.Defaults())
	colors := theme.Default().Light
	view := ui.View(func(c *ui.Context) {
		theme.ApplyInterface(c)
		ui.Column(c).Fill().Background(chrome.Color(colors.BG)).Children(func() { inspectorPresentation(c, &p, source, session.ViewState{}, colors, nil) })
	})
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{Title: "Round102 inspector fixture", Width: 780, Height: 600, UseContentSize: true, Content: view, Frameless: true, BackgroundColor: "#ffffff"})
	})
	if err = mygo.App.Run(); err != nil {
		panic(err)
	}
}
