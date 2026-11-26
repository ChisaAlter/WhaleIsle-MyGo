package frame

import (
	"fmt"
	"testing"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/composer"
	"whaleisle/internal/ui/theme"
)

func TestComposerTypingKeepsScrolledUpReadingPosition(t *testing.T) {
	store := session.New()
	store.Select("scroll-test")
	msgs := make([]session.Message, 0, 40)
	for i := 0; i < 40; i++ {
		msgs = append(msgs, session.Message{
			ID:   fmt.Sprintf("msg-%d", i),
			Role: "assistant",
			Kind: session.MessageKindText,
			Body: fmt.Sprintf("Line %d of a long conversation message body that makes the transcript scroll well past a single viewport height.", i),
		})
	}
	draft := "hello"
	draftState := composer.DraftState{}
	client := composer.NewDraftClient(&draft)
	list := ui.ListState{}
	sizes := DefaultLayout()
	sizes.Bottom = 0
	snap := session.ViewState{
		SelectedID:      "scroll-test",
		PromptAttempted: true,
		Messages:        msgs,
	}
	var tt *ui.Tester
	tt = ui.NewTester(func(c *ui.Context) {
		originalScrollbarTheme(c)
		sizes.ResolveColumns(1200, false)
		colors := theme.Resolve("deepseek.light").Light
		root := ui.Column(c).Fill()
		root.Children(func() {
			mainRow(c, Model{
				Store:                store,
				Sizes:                &sizes,
				Draft:                &draft,
				ComposerState:        &draftState,
				ChatList:             &list,
				UseComposerTextInput: true,
				ComposerTextInput:    client,
			}, snap, colors, theme.Color(colors.Border), theme.Color(colors.Muted))
		})
	}, 1200, 800)
	tt.Frame()

	// Scroll transcript up away from the tail (row 4).
	list.ScrollTo(4, ui.Start)
	tt.Frame()
	beforeFirst, beforeLast := list.Visible()
	if beforeFirst > 8 || list.AtEnd() {
		t.Fatalf("expected transcript scrolled up near row 4, got visible=[%d,%d] atEnd=%v", beforeFirst, beforeLast, list.AtEnd())
	}

	// Typing, newline, and backspace in the composer must not pull the scrolled-up transcript down.
	if err := tt.Click("Composer draft"); err != nil {
		t.Fatal(err)
	}
	client.ReplaceText(nil, "hello world\nsecond line")
	tt.Frame()
	afterFirst, afterLast := list.Visible()
	if afterFirst != beforeFirst || list.AtEnd() {
		t.Fatalf("typing shifted scrolled-up reading position: [%d,%d] -> [%d,%d]", beforeFirst, beforeLast, afterFirst, afterLast)
	}
	client.ReplaceText(nil, "hello")
	tt.Frame()
	afterDelFirst, _ := list.Visible()
	if afterDelFirst != beforeFirst || list.AtEnd() {
		t.Fatalf("deleting shifted scrolled-up reading position: %d -> %d", beforeFirst, afterDelFirst)
	}
}
