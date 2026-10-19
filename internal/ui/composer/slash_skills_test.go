package composer

import (
	"context"
	"testing"
	"whaleisle/internal/dsh/api"
	"whaleisle/internal/store/session"
)

func TestSlashMenuIncludesRankedSkillReferences(t *testing.T) {
	m := Model{Skills: []session.SkillOption{{Name: "code-review", Description: "Review", UserOnly: true}, {Name: "unrelated"}}}
	items := buildSuggestItems(m, SuggestSlash, "cdrv")
	if len(items) != 1 || items[0].insert != "/code-review " || items[0].sub == "Review" {
		t.Fatalf("missing ranked user-only skill: %+v", items)
	}
}

func TestSkillSelectionIsLiteralAndCommandsOwnCollisions(t *testing.T) {
	d, m, _ := suggestionFixture("/cr")
	m.Skills = []session.SkillOption{{Name: "code-review"}}
	m.Snapshot.SlashCommands = nil
	calls := 0
	m.OnExecuteSlash = func(context.Context, string) { calls++ }
	items := buildSuggestItems(m, SuggestSlash, "cr")
	if len(items) != 1 {
		t.Fatal(items)
	}
	applySuggest(m, items[0], TokenRange{0, 3})
	if d.Text != "/code-review " || calls != 0 || handleSlashSend(m, d.Text) {
		t.Fatalf("skill sent as command: %q calls %d", d.Text, calls)
	}
	m.Snapshot.SlashCommands = []api.ListResultItem2{{Name: "code-review", Description: "command"}}
	items = buildSuggestItems(m, SuggestSlash, "")
	if len(items) != 1 || items[0].sub != "command" || isSkillOnlyInvocation(m, "/code-review") {
		t.Fatal("command collision priority lost")
	}
	m.Snapshot.SelectedID = "child"
	m.Snapshot.Items = []session.Item{{SessionID: "child", Origin: "subagent"}}
	m.Snapshot.SlashCommands = nil
	if len(buildSuggestItems(m, SuggestSlash, "")) != 0 {
		t.Fatal("subagent skill source unexpectedly enabled")
	}
}
func TestCommandLauncherPreservesDraftAndCurrentSelection(t *testing.T) {
	d, m, sent := suggestionFixture("A😀B")
	m.Snapshot.SelectedID = "a"
	m.Snapshot.SlashCommands = nil
	m.Skills = []session.SkillOption{{Name: "reference"}}
	d.Client.SetSelectionUTF16(1, 3)
	toggleCommandLauncher(m)
	kind, query, tok, _, open := activeSuggestion(m, d.Client)
	if !open || kind != SuggestSlash || query != "" || tok != (TokenRange{1, 5}) || d.Text != "A😀B" {
		t.Fatalf("launcher mutated draft/selection: %q %+v", d.Text, tok)
	}
	items := buildSuggestItems(m, kind, query)
	applySuggest(m, items[0], tok)
	if d.Text != "A"+items[0].insert+"B" || *sent != 0 {
		t.Fatal("launcher did not replace captured selection")
	}
	toggleCommandLauncher(m)
	toggleCommandLauncher(m)
	if _, _, _, _, open = activeSuggestion(m, d.Client); open {
		t.Fatal("second click did not close")
	}
	toggleCommandLauncher(m)
	m.Snapshot.SelectedID = "b"
	if _, _, _, _, open = activeSuggestion(m, d.Client); open {
		t.Fatal("launcher leaked to another session")
	}
}
