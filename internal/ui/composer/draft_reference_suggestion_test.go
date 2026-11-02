package composer

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

func TestInlineCommittedReferenceIsOpaqueToSuggestionTriggers(t *testing.T) {
	for _, token := range []string{"@src/file.go", "/review", `@"my file.txt"`, "@[Team chat](dsh-session:ImEi)"} {
		d := NewDraftState(token)
		m := Model{Draft: &d.Text, State: d, TextInput: d.Client, UseTextInputClient: true}
		if _, _, _, _, open := activeSuggestion(m, d.Client); open {
			t.Fatalf("committed reference reopened a query: %q", token)
		}
		d.Client.ReplaceText(&ui.TextInputRange{Start: ui.UTF16Len(token), End: ui.UTF16Len(token)}, " @editable")
		// Explicit insertion creates objects. A typed trigger after the object
		// remains editable and must still open its own completion menu.
		d.Client.SetSelectionUTF16(ui.UTF16Len(d.Text), ui.UTF16Len(d.Text))
		d.Client.input.ReplaceText(nil, " @new")
		kind, query, _, _, open := activeSuggestion(m, d.Client)
		if !open || kind != SuggestMention || query != "new" {
			t.Fatalf("prose trigger after chip suppressed: kind=%v query=%q open=%v", kind, query, open)
		}
	}
}
