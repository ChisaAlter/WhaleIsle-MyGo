package frame

import (
	"testing"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/ui/i18n"
)

func TestJumpToLatest(t *testing.T) {
	oldLocale := i18n.Locale()
	i18n.SetLocale(i18n.LocaleEN)
	defer i18n.SetLocale(oldLocale)

	store := seedLongConversationStore(t, 40)
	layout := DefaultLayout()
	draft := ""
	chatList := &ui.ListState{FollowEnd: true}
	paneTab := "chat"
	m := Model{
		Store:       store,
		Sizes:       &layout,
		ChatList:    chatList,
		Draft:       &draft,
		ChatPaneTab: &paneTab,
		ThemeID:     "deepseek.light",
	}
	tt := ui.NewTester(func(c *ui.Context) {
		View(c, m)
	}, 1100, 720)
	tt.Frame()

	// At bottom: Jump to latest button must be hidden.
	if _, found := tt.Find("Jump to latest"); found {
		t.Fatal("Jump to latest button visible while already at bottom")
	}

	// Scroll up away from bottom: Jump to latest button must appear.
	body, ok := tt.Find("Conversation body")
	if !ok {
		t.Fatal("Conversation body not found")
	}
	cx, cy := body.X+body.W/2, body.Y+body.H/3
	for i := 0; i < 6; i++ {
		tt.Scroll(cx, cy, 0, -120)
	}
	tt.Frame()
	if chatList.AtEnd() {
		t.Fatal("expected ChatList not to be at end after scrolling up")
	}
	if _, found := tt.Find("Jump to latest"); !found {
		t.Fatalf("Jump to latest button missing after scrolling up; texts=%v", tt.Texts())
	}

	// Trajectory tab hides Jump to latest.
	paneTab = "trajectory"
	tt.Frame()
	if _, found := tt.Find("Jump to latest"); found {
		t.Fatal("Jump to latest button should stay hidden on trajectory tab")
	}
	paneTab = "chat"
	tt.Frame()

	// Clicking Jump to latest scrolls to end and hides the button.
	if err := tt.Click("Jump to latest"); err != nil {
		t.Fatalf("clicking Jump to latest failed: %v", err)
	}
	tt.Frame()
	if !chatList.AtEnd() {
		t.Fatal("expected ChatList.AtEnd() == true after clicking Jump to latest")
	}
	if _, found := tt.Find("Jump to latest"); found {
		t.Fatal("Jump to latest button remained visible after returning to bottom")
	}
}
