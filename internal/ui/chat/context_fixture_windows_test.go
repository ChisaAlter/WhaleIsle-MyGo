//go:build windows && contextfixture

package chat

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

// Manual, isolated native component fixture using the same JSON as the
// installed reference component probe. It never connects to a session Host.
// Build: go test -tags contextfixture -c -o .devdata/context-fixture.exe ./internal/ui/chat
// Run with CONTEXT_NATIVE_FIXTURE set to the absolute testdata/context-components.json path.
func TestMain(m *testing.M) {
	path := os.Getenv("CONTEXT_NATIVE_FIXTURE")
	if path == "" {
		os.Exit(m.Run())
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		panic(err)
	}
	var input []struct {
		ID      string          `json:"id"`
		Source  json.RawMessage `json:"source"`
		Content json.RawMessage `json:"content"`
		Time    float64         `json:"time"`
	}
	if err = json.Unmarshal(raw, &input); err != nil {
		panic(err)
	}
	var messages []session.Message
	for i, f := range input {
		event, _ := json.Marshal(map[string]any{"type": "user/message", "data": map[string]any{"source": f.Source, "content": f.Content}})
		messages = append(messages, session.Message{ID: f.ID, Seq: float64(i + 1), Kind: session.MessageKindContext, Role: "context", Raw: event, EventTime: f.Time, ContextWaking: f.ID == "trigger"})
	}
	i18n.SetLocale(i18n.LocaleEN)
	theme.SetPreferences(theme.Defaults())
	palette := theme.Default().Light
	view := ui.View(func(c *ui.Context) {
		theme.ApplyInterface(c)
		ui.Column(c).Fill().Background(chrome.Color(palette.BG)).Children(func() {
			ui.Column(c).Width(600).Padding(24).Gap(12).Children(func() {
				for _, msg := range messages {
					if msg.ContextWaking {
						TurnTriggerView(c, msg, "r101-fixture", palette)
					} else {
						ContextInjectionView(c, msg, "r101-fixture", palette)
					}
				}
			})
		})
	})
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{Title: "Round101 context fixture", Width: 900, Height: 800, Content: view, TitleBarStyle: mygo.TitleBarHidden, BackgroundColor: "#ffffff"})
	})
	if err = mygo.App.Run(); err != nil {
		panic(err)
	}
}
