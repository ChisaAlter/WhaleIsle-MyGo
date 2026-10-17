package composer

import (
	"github.com/egoist/mygo/ui"
	"testing"
	"whaleisle/internal/dsh/api"
)

func TestLeadingInputSpaceClaimsDeclaredHostCommand(t *testing.T) {
	d, m, _ := suggestionFixture("/goal")
	m.Snapshot.ClientReady = true
	m.Snapshot.SelectedID = "a"
	m.Snapshot.SlashCommands = []api.ListResultItem2{{Name: "goal", Input: &api.ListResultItem2Input{Hint: "Describe goal"}}}
	if !draftInputEvent(d.Client, m, ui.InputEvent{Kind: ui.InputKeyDown, Key: ui.KeySpace}) || d.Text != "/goal " {
		t.Fatalf("Space did not claim: %q", d.Text)
	}
}

func TestLeadingClaimHintLifecycleAndSessionOwnership(t *testing.T) {
	d, m, _ := suggestionFixture("/goal")
	m.Snapshot.ClientReady = true
	m.Snapshot.SelectedID = "a"
	yes := true
	m.Snapshot.SlashCommands = []api.ListResultItem2{{Name: "goal", Input: &api.ListResultItem2Input{Hint: "Describe goal", Attachments: &yes}}}
	draftInputEvent(d.Client, m, ui.InputEvent{Kind: ui.InputKeyDown, Key: ui.KeySpace})
	claim := currentLeadingClaim(m)
	if claim == nil || claim.hint != "Describe goal" || !claim.attachments {
		t.Fatalf("claim %+v", claim)
	}
	d.Client.ReplaceText(nil, "work")
	if currentLeadingClaim(m) == nil {
		t.Fatal("args released claim")
	}
	m.Snapshot.SelectedID = "other"
	if currentLeadingClaim(m) != nil {
		t.Fatal("claim leaked to other session")
	}
	m.Snapshot.SelectedID = "a"
	if currentLeadingClaim(m) != nil {
		t.Fatal("stale claim restored")
	}
}

func TestLeadingSpacePassesUnknownPopupAndIME(t *testing.T) {
	for _, text := range []string{"/unknown", "/model", "/permission", "/plain", "prefix /goal"} {
		d, m, _ := suggestionFixture(text)
		m.Snapshot.ClientReady = true
		m.Snapshot.SlashCommands = []api.ListResultItem2{{Name: "goal", Input: &api.ListResultItem2Input{Hint: "Goal"}}, {Name: "plain"}, {Name: "model", Input: &api.ListResultItem2Input{Hint: "Model"}}}
		if claimLeadingSpace(d.Client, m, ui.InputEvent{Kind: ui.InputKeyDown, Key: ui.KeySpace}) || d.Text != text {
			t.Fatalf("unexpected claim %q", text)
		}
	}
	d, m, _ := suggestionFixture("/goal")
	m.Snapshot.ClientReady = true
	m.Snapshot.SlashCommands = []api.ListResultItem2{{Name: "goal", Input: &api.ListResultItem2Input{}}}
	d.Client.marked = &ui.TextInputRange{Start: 0, End: 5}
	if claimLeadingSpace(d.Client, m, ui.InputEvent{Kind: ui.InputKeyDown, Key: ui.KeySpace}) {
		t.Fatal("IME stolen")
	}
	d.Client.marked = nil
	d.Client.SetSelectionUTF16(0, 5)
	if claimLeadingSpace(d.Client, m, ui.InputEvent{Kind: ui.InputKeyDown, Key: ui.KeySpace}) {
		t.Fatal("selection replaced")
	}
}
