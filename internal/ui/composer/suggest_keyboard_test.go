package composer

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

func suggestionFixture(text string) (*DraftState, Model, *int) {
	d := NewDraftState(text)
	selected := 0
	sent := 0
	m := Model{Draft: &d.Text, State: d, TextInput: d.Client, UseTextInputClient: true,
		SuggestList: &ui.ListState{Selected: &selected}, SlashCatalog: StubSlashCommands(),
		FilePaths: []string{"readme.md", "report.md"}, OnSubmit: func() { sent++ }}
	return d, m, &sent
}

func TestSuggestionEnterAcceptsHighlightWithoutSending(t *testing.T) {
	for _, text := range []string{"/", "@r"} {
		d, m, sent := suggestionFixture(text)
		if text == "/" {
			m.SlashCatalog = []SlashCommand{{Line: "/remote", Description: "Remote command"}}
		}
		if !draftInputEvent(d.Client, m, ui.InputEvent{Kind: ui.InputKeyDown, Key: ui.KeyDown}) {
			t.Fatal("down not consumed")
		}
		kind, query, tok, quoted, _ := activeSuggestion(m, d.Client)
		items := buildSuggestItemsWithQuote(m, kind, query, quoted)
		want := ReplaceToken(text, tok, items[*m.SuggestList.Selected].insert)
		if !draftInputEvent(d.Client, m, ui.InputEvent{Kind: ui.InputKeyDown, Key: ui.KeyEnter}) || d.Text != want || *sent != 0 {
			t.Fatalf("accept %q: text=%q want=%q sent=%d", text, d.Text, want, *sent)
		}
		if _, _, _, _, open := activeSuggestion(m, d.Client); open {
			t.Fatal("accepted menu reopened")
		}
		draftInputEvent(d.Client, m, ui.InputEvent{Kind: ui.InputKeyDown, Key: ui.KeyEnter})
		if *sent != 1 {
			t.Fatal("next Enter did not submit accepted draft")
		}
	}
}

func TestSuggestionDismissalPersistsUntilTokenChanges(t *testing.T) {
	for _, text := range []string{"/", "@r"} {
		for _, event := range []ui.InputEvent{
			{Kind: ui.InputKeyDown, Key: ui.KeyEscape},
			{Kind: ui.InputKeyDown, Key: ui.KeyTab, Mods: ui.Shift},
		} {
			d, m, sent := suggestionFixture(text)
			beforeSelection := d.Client.Selection()
			if !draftInputEvent(d.Client, m, event) || d.Text != text || d.Client.Selection() != beforeSelection || *sent != 0 {
				t.Fatalf("dismiss changed draft: %q", d.Text)
			}
			for i := 0; i < 3; i++ {
				if _, _, _, _, open := activeSuggestion(m, d.Client); open {
					t.Fatal("repaint reopened dismissed token")
				}
			}
			d.Client.ReplaceText(nil, "e")
			if _, _, _, _, open := activeSuggestion(m, d.Client); !open {
				t.Fatal("query change did not re-arm completion")
			}
		}
	}
}

func TestSuggestionIMEAndModifiedEnterKeepPriority(t *testing.T) {
	d, m, sent := suggestionFixture("@r")
	d.Client.marked = &ui.TextInputRange{End: 2}
	for _, key := range []ui.Key{ui.KeyEnter, ui.KeyEscape, ui.KeyTab, ui.KeyDown} {
		if handleSuggestKey(d.Client, m, ui.InputEvent{Kind: ui.InputKeyDown, Key: key}) {
			t.Fatalf("IME key %v consumed", key)
		}
	}
	if d.Text != "@r" || *sent != 0 {
		t.Fatal("IME mutated draft")
	}
	d.Client.marked = nil
	if handleSuggestKey(d.Client, m, ui.InputEvent{Kind: ui.InputKeyDown, Key: ui.KeyEnter, Mods: ui.Shift}) {
		t.Fatal("shift Enter selected candidate")
	}
	draftInputEvent(d.Client, m, ui.InputEvent{Kind: ui.InputKeyDown, Key: ui.KeyEnter, Mods: ui.Shift})
	if d.Text != "@r\n" || *sent != 0 {
		t.Fatal("shift Enter lost newline")
	}
}

func TestSuggestionAbsentCandidatesPreserveNormalSubmit(t *testing.T) {
	d, m, sent := suggestionFixture("@no-matching-path")
	draftInputEvent(d.Client, m, ui.InputEvent{Kind: ui.InputKeyDown, Key: ui.KeyEnter})
	if *sent != 1 {
		t.Fatal("empty menu blocked normal submit")
	}
}

func TestSuggestionEscapePrecedesEditCancel(t *testing.T) {
	d, m, _ := suggestionFixture("@r")
	cancelled := 0
	m.Editing = true
	m.OnCancelEdit = func() { cancelled++ }
	draftInputEvent(d.Client, m, ui.InputEvent{Kind: ui.InputKeyDown, Key: ui.KeyEscape})
	if cancelled != 0 || d.Text != "@r" {
		t.Fatal("first Escape cancelled edit instead of menu")
	}
	draftInputEvent(d.Client, m, ui.InputEvent{Kind: ui.InputKeyDown, Key: ui.KeyEscape})
	if cancelled != 1 {
		t.Fatal("second Escape did not cancel edit")
	}
}
