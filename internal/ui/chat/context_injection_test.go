package chat

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

func TestInjectedContextUsesReadOnlyDisclosure(t *testing.T) {
	old := i18n.Locale()
	i18n.SetLocale(i18n.LocaleEN)
	defer i18n.SetLocale(old)
	msg := session.Message{ID: "ctx", Seq: 1, Kind: session.MessageKindContext, Role: "context", Body: "model-facing instructions", Raw: json.RawMessage(`{"type":"user/message","data":{"source":{"kind":"agent-instructions","form":"instructions","changes":[{"path":"AGENTS.md","action":"replace"}]},"content":[{"type":"text","text":"model-facing instructions"}]}}`)}
	snap := session.ViewState{SelectedID: "one", Messages: []session.Message{msg}}
	tt := ui.NewTester(func(c *ui.Context) {
		ContextInjectionView(c, snap.Messages[0], snap.SelectedID, theme.Resolve(theme.IDDeepseekLight).Light)
	}, 800, 1000)
	if _, ok := tt.Find("User message bubble"); ok {
		t.Fatal("context uses human bubble")
	}
	if tt.HasText(msg.Body) {
		t.Fatal("context not initially collapsed")
	}
	if err := tt.Click("Context injection AGENTS.md"); err != nil {
		t.Fatal(err)
	}
	if !tt.HasText(msg.Body) || !tt.HasText("AGENTS.md") || !tt.HasText("updated") {
		t.Fatalf("resolved instructions absent: %q", tt.Texts())
	}
	for _, extra := range []string{"Model-facing content", "Copy context", "Context details", "changes", "instructions"} {
		if extra != "instructions" && tt.HasText(extra) {
			t.Fatalf("invented context chrome %q: %q", extra, tt.Texts())
		}
	}
	if showMessageCopy(msg) || showMessageEdit(msg) {
		t.Fatal("context carries human/assistant mutation actions")
	}
	snap.SelectedID = "two"
	tt.Frame()
	if tt.HasText(msg.Body) {
		t.Fatal("disclosure state crossed session")
	}
}

func TestContextDisplayBoundDoesNotChangeRetainedContent(t *testing.T) {
	old := i18n.Locale()
	i18n.SetLocale(i18n.LocaleEN)
	defer i18n.SetLocale(old)
	body := strings.Repeat("界", 20_000) + "END"
	msg := session.Message{ID: "long", Kind: session.MessageKindContext, Role: "context", Body: body}
	tt := ui.NewTester(func(c *ui.Context) {
		ContextInjectionView(c, msg, "host:session", theme.Resolve(theme.IDDeepseekLight).Light)
	}, 800, 1000)
	if err := tt.Click("Context injection"); err != nil {
		t.Fatal(err)
	}
	if tt.HasText("END") || !tt.HasText("20003") {
		t.Fatalf("UTF16 display bound/total absent: %q", tt.Texts())
	}
	if session.ContextMessageDetails(msg).Body != body {
		t.Fatal("raw body changed by display clipping")
	}
}

func TestContextBodyHorizontalWheelKeepsModelTextWrapped(t *testing.T) {
	contextBodyEnglish(t)
	beforePrefs := theme.CurrentPreferences()
	defer theme.SetPreferences(beforePrefs)
	theme.SetPreferences(theme.Defaults())
	key := strings.Repeat("fixed_key_", 12)
	paragraph := "ordinary words wrap across the available code well width while the fixed metadata key remains scrollable"
	source, _ := json.Marshal(map[string]any{"kind": "foreign", key: "value"})
	content, _ := json.Marshal([]map[string]any{{"type": "text", "text": paragraph}})
	msg := contextBodyFixture(string(source), string(content))
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).FillWidth().Padding(12).Children(func() { ContextInjectionView(c, msg, "overflow", theme.Default().Light) })
	}, 360, 400)
	if err := tt.Click("Context injection foreign"); err != nil {
		t.Fatal(err)
	}
	text, ok := tt.Find(paragraph)
	if !ok || text.H <= 16 || text.W != 286 {
		t.Fatalf("model text no longer wraps within the well: %+v %v", text, ok)
	}
	keyBefore, ok := tt.Find(key)
	if !ok {
		t.Fatal("fixed metadata missing")
	}
	tt.Move(200, text.Y+5)
	tt.Scroll(200, text.Y+5, 100, 0)
	keyAfter, ok := tt.Find(key)
	if !ok || keyAfter.X >= keyBefore.X || keyAfter.W <= keyBefore.W {
		t.Fatalf("horizontal wheel cannot reveal fixed metadata: before=%+v after=%+v found=%v", keyBefore, keyAfter, ok)
	}
}

func TestContextDisclosureGeometryKeyboardAndInertExit(t *testing.T) {
	oldLocale, oldPrefs := i18n.Locale(), theme.CurrentPreferences()
	i18n.SetLocale(i18n.LocaleEN)
	defer i18n.SetLocale(oldLocale)
	defer theme.SetPreferences(oldPrefs)
	for _, fontSize := range []float32{14, 18} {
		prefs := theme.Defaults()
		prefs.Chat.Size = fontSize
		theme.SetPreferences(prefs)
		msg := contextFixture("mixed", "tool-registry", `[{"type":"text","text":"first"},{"type":"text","text":"second"},{"type":"tool-addition","toolName":"new_tool"},{"type":"text","text":"last"}]`)
		tt := ui.NewTester(func(c *ui.Context) {
			ui.Column(c).FillWidth().Padding(12).Children(func() {
				ContextInjectionView(c, msg, "s", theme.Resolve(theme.IDDeepseekLight).Light)
				ui.Button(c, "after")
			})
		}, 420, 600)
		at := time.Unix(100, 0)
		tt.SetTime(at)
		name := "Context injection tool-registry"
		row, ok := tt.Find(name)
		if !ok || row.X != 12 || row.W != 396 || row.H != 24+fontSize-14 {
			t.Fatalf("flow row %+v found=%v", row, ok)
		}
		tt.Key(0, ui.KeyTab)
		if !tt.Focused(name) {
			t.Fatalf("Tab did not reach context disclosure: %q", tt.Texts())
		}
		tt.Key(0, ui.KeyEnter)
		tt.SetTime(at.Add(110 * time.Millisecond))
		text, ok := tt.Find("firstsecond")
		if !ok || text.X != 12+22+fontSize-14+12 || text.X+text.W != 420-12-16 {
			t.Fatalf("joined content/indent: %+v %v, %q", text, ok, tt.Texts())
		}
		if !tt.HasText("Unknown content block") || !tt.HasText("last") {
			t.Fatalf("mixed run order lost: %q", tt.Texts())
		}
		expanded, _ := tt.Find("after")
		tt.Key(0, ui.KeySpace)
		if tt.HasText("firstsecond") || tt.HasText("Unknown content block") {
			t.Fatal("retained exit body remains interactive/findable")
		}
		exit, _ := tt.Find("after")
		if exit.Y != expanded.Y {
			t.Fatalf("exit hold collapsed early: open=%v exit=%v", expanded.Y, exit.Y)
		}
		tt.SetTime(at.Add(311 * time.Millisecond))
		collapsed, _ := tt.Find("after")
		if collapsed.Y >= exit.Y {
			t.Fatal("exit hold did not release layout")
		}
	}
}
