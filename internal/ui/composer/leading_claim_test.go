package composer

import (
	"github.com/egoist/mygo/ui"
	"testing"
	"whaleisle/internal/dsh/api"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
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
	graph := nativeCommandContributionFixture(t, nil)
	for _, text := range []string{"/unknown", "/model", "/permission", "/plain", "prefix /goal"} {
		d, m, _ := suggestionFixture(text)
		m.Snapshot.ClientReady = true
		m.Snapshot.NativeContributions = graph
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

func TestPermissionDecorationLeavesHostSpaceAndArguments(t *testing.T) {
	d, m, _ := suggestionFixture("/permission")
	m.Snapshot = nativeCommandSnapshot(t)
	m.Snapshot.SlashCommands = []api.ListResultItem2{{Name: "permission", Input: &api.ListResultItem2Input{Hint: "Permission name"}}}
	if !claimLeadingSpace(d.Client, m, ui.InputEvent{Kind: ui.InputKeyDown, Key: ui.KeySpace}) || d.Text != "/permission " {
		t.Fatal("decoration claimed Host Space path")
	}
	if currentLeadingClaim(m) == nil {
		t.Fatal("Host leading-input claim not retained")
	}
	d.Client.ReplaceText(nil, "workspace-write")
	if handleModelCommand(m) || d.PermissionCommand.Open {
		t.Fatal("argument-bearing Host permission opened native popup")
	}
}

func TestCommandAttachmentsUnsupportedRefusesClaimedAndBareHostCommands(t *testing.T) {
	defer i18n.SetLocale(i18n.DefaultLocale)
	for _, locale := range []string{i18n.LocaleZH, i18n.LocaleEN} {
		i18n.SetLocale(locale)
		d, m, _ := suggestionFixture("/goal")
		m.Snapshot.ClientReady = true
		m.Snapshot.SelectedID = "a"
		yes := true
		m.Snapshot.SlashCommands = []api.ListResultItem2{
			{Name: "goal", Input: &api.ListResultItem2Input{Hint: "Goal"}},
			{Name: "compact"},
			{Name: "attachable", Input: &api.ListResultItem2Input{Hint: "Attachable", Attachments: &yes}},
		}
		d.Attachments = []session.ComposerAttachment{{Path: "note.txt", Name: "note.txt"}}
		submitted := 0
		m.OnSubmitGesture = func(bool) { submitted++ }
		if !draftInputEvent(d.Client, m, ui.InputEvent{Kind: ui.InputKeyDown, Key: ui.KeySpace}) {
			t.Fatal("Space did not claim /goal")
		}
		d.Client.ReplaceText(nil, "ship v1")
		if !draftInputEvent(d.Client, m, ui.InputEvent{Kind: ui.InputKeyDown, Key: ui.KeyEnter}) {
			t.Fatal("Enter not handled")
		}
		wantGoalErr := CommandAttachmentsUnsupportedNotice("goal")
		if submitted != 0 || d.Error != wantGoalErr || d.Text != "/goal ship v1" || len(d.Attachments) != 1 {
			t.Fatalf("locale %s claimed command with attachment should refuse without clearing: submitted=%d err=%q want=%q text=%q", locale, submitted, d.Error, wantGoalErr, d.Text)
		}

		// Bare Host command with attachments also surfaces the refusal notice and keeps draft/attachments.
		d.Error = ""
		d.Client.ReplaceText(&ui.TextInputRange{Start: 0, End: ui.UTF16Len(d.Text)}, "/compact")
		if !draftInputEvent(d.Client, m, ui.InputEvent{Kind: ui.InputKeyDown, Key: ui.KeyEnter}) {
			t.Fatal("Enter not handled for /compact")
		}
		wantCompactErr := CommandAttachmentsUnsupportedNotice("compact")
		if submitted != 0 || d.Error != wantCompactErr || d.Text != "/compact" || len(d.Attachments) != 1 {
			t.Fatalf("locale %s bare command with attachment should refuse without clearing: submitted=%d err=%q want=%q text=%q", locale, submitted, d.Error, wantCompactErr, d.Text)
		}

		// A command declaring attachments=true proceeds to submission.
		d.Error = ""
		d.Client.ReplaceText(&ui.TextInputRange{Start: 0, End: ui.UTF16Len(d.Text)}, "/attachable")
		if !draftInputEvent(d.Client, m, ui.InputEvent{Kind: ui.InputKeyDown, Key: ui.KeySpace}) {
			t.Fatal("Space did not claim /attachable")
		}
		d.Client.ReplaceText(nil, "with file")
		if !draftInputEvent(d.Client, m, ui.InputEvent{Kind: ui.InputKeyDown, Key: ui.KeyEnter}) {
			t.Fatal("Enter not handled for /attachable")
		}
		if submitted != 1 || d.Error != "" {
			t.Fatalf("locale %s attachable command was blocked: submitted=%d err=%q", locale, submitted, d.Error)
		}
	}
}
