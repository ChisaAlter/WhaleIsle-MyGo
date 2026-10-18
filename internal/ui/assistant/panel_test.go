package assistant

import (
	"testing"

	"github.com/egoist/mygo/ui"

	assiststore "whaleisle/internal/store/assistant"
	dshbotstore "whaleisle/internal/store/dshbot"
	"whaleisle/internal/ui/i18n"
)

func TestViewAssistantLabels(t *testing.T) {
	i18n.SetLocale("zh")
	defer i18n.SetLocale(i18n.DefaultLocale)

	draft := "hello"
	tt := ui.NewTester(func(c *ui.Context) {
		View(c, Model{
			ThemeID:             "deepseek",
			Mode:                assiststore.ModeWhale,
			WhaleEnabled:        true,
			DshbotEnabled:       false,
			WhaleVendorPresent:  true,
			DshbotVendorPresent: true,
			HarnessReady:        true,
			LastWebViewURL:      "http://127.0.0.1:3080/?desktopPlugin=dsh-whale",
			LastNativeAction:    "sent",
			SessionID:           "session-whale-1",
			History: []assiststore.Turn{{
				Mode:     assiststore.ModeWhale,
				Prompt:   "hi",
				Response: "hello back",
				Source:   "rpc",
			}},
			Draft:         &draft,
			OnSetMode:     func(assiststore.Mode) {},
			OnOpenSession: func(string) {},
			OnWhaleEnsure: func() {},
		})
	}, 720, 1100)

	want := []string{
		i18n.T(i18n.KeyAssistantTitle),
		i18n.T(i18n.KeyAssistantWhaleTitle),
		i18n.T(i18n.KeyAssistantDshbotTitle),
		i18n.T(i18n.KeyAssistantStatusTitle),
		i18n.T(i18n.KeyAssistantComposeTitle),
		i18n.T(i18n.KeyAssistantSend),
		i18n.T(i18n.KeyAssistantModeTitle),
		i18n.T(i18n.KeyAssistantOpenSession),
		i18n.T(i18n.KeyAssistantPluginRPCTitle),
		i18n.T(i18n.KeyAssistantLastWebURL),
		i18n.T(i18n.KeyAssistantLastAction),
		"http://127.0.0.1:3080/?desktopPlugin=dsh-whale",
		"sent",
		i18n.T(i18n.KeyAssistantStubNote),
	}
	for _, s := range want {
		if !tt.HasText(s) {
			t.Fatalf("missing %q; texts=%q", s, tt.Texts())
		}
	}
}

func TestModeToggle(t *testing.T) {
	i18n.SetLocale("en")
	defer i18n.SetLocale(i18n.DefaultLocale)

	var got assiststore.Mode
	tt := ui.NewTester(func(c *ui.Context) {
		View(c, Model{
			ThemeID:      "deepseek",
			Mode:         assiststore.ModeWhale,
			HarnessReady: true,
			OnSetMode: func(mode assiststore.Mode) {
				got = mode
			},
		})
	}, 640, 720)

	label := i18n.T(i18n.KeyAssistantModeDshbot)
	if err := tt.Click(label); err != nil {
		t.Fatalf("click mode: %v; texts=%q", err, tt.Texts())
	}
	if got != assiststore.ModeDshbot {
		t.Fatalf("mode = %q, want dshbot", got)
	}
}

func TestOpenSession(t *testing.T) {
	i18n.SetLocale("en")
	defer i18n.SetLocale(i18n.DefaultLocale)

	var got string
	tt := ui.NewTester(func(c *ui.Context) {
		View(c, Model{
			ThemeID:      "deepseek",
			HarnessReady: true,
			WhaleEnabled: true,
			SessionID:    "sess-42",
			OnOpenSession: func(id string) {
				got = id
			},
		})
	}, 640, 720)

	if err := tt.Click(i18n.T(i18n.KeyAssistantOpenSession)); err != nil {
		t.Fatalf("click open session: %v; texts=%q", err, tt.Texts())
	}
	if got != "sess-42" {
		t.Fatalf("session = %q", got)
	}
}

func TestNativePromptSend(t *testing.T) {
	i18n.SetLocale("en")
	defer i18n.SetLocale(i18n.DefaultLocale)

	draft := "native hello"
	var got string
	tt := ui.NewTester(func(c *ui.Context) {
		View(c, Model{
			ThemeID:      "deepseek",
			HarnessReady: true,
			WhaleEnabled: true,
			Draft:        &draft,
			OnNativePrompt: func(text string) {
				got = text
			},
		})
	}, 640, 720)

	if err := tt.Click(i18n.T(i18n.KeyAssistantSend)); err != nil {
		t.Fatalf("click send: %v; texts=%q", err, tt.Texts())
	}
	if got != "native hello" {
		t.Fatalf("OnNativePrompt got %q, want %q", got, "native hello")
	}
	if draft != "" {
		t.Fatalf("draft should clear after send, got %q", draft)
	}
}

func TestDshbotRosterBlock(t *testing.T) {
	i18n.SetLocale("en")
	defer i18n.SetLocale(i18n.DefaultLocale)

	tt := ui.NewTester(func(c *ui.Context) {
		View(c, Model{
			ThemeID:       "deepseek",
			Mode:          assiststore.ModeDshbot,
			DshbotEnabled: true,
			HarnessReady:  true,
			DshbotRoster: []dshbotstore.Row{{
				Kind: "bot", Name: "Alpha", DisplayLabel: "Alpha · helper", SessionID: "s1",
			}},
			DshbotRosterFetched:       "2026-01-01T00:00:00Z",
			OnDshbotRefreshRoster:     func() {},
			OnDshbotOpenRosterSession: func(string) {},
		})
	}, 720, 900)

	for _, s := range []string{
		i18n.T(i18n.KeyAssistantDshbotRosterTitle),
		i18n.T(i18n.KeyAssistantDshbotRosterRefresh),
		"Alpha · helper",
	} {
		if !tt.HasText(s) {
			t.Fatalf("missing %q; texts=%q", s, tt.Texts())
		}
	}
}

func TestNativePromptEmptyDisabled(t *testing.T) {
	i18n.SetLocale("en")
	defer i18n.SetLocale(i18n.DefaultLocale)

	draft := "   "
	called := false
	tt := ui.NewTester(func(c *ui.Context) {
		View(c, Model{
			ThemeID: "deepseek",
			Draft:   &draft,
			OnNativePrompt: func(text string) {
				called = true
			},
		})
	}, 640, 480)

	_ = tt.Click(i18n.T(i18n.KeyAssistantSend))
	if called {
		t.Fatal("empty draft must not invoke OnNativePrompt")
	}
}
