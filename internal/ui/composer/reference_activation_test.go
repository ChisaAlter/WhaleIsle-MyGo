package composer

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
)

func TestComposerReferencePrimaryOpensWithoutMutation(t *testing.T) {
	for _, token := range []string{"@src/main.go", `@"my file.txt"`, "/review"} {
		t.Run(token, func(t *testing.T) {
			d := NewDraftState("read " + token + " now")
			before := d.Text
			opened := ""
			m := Model{Draft: &d.Text, State: d, TextInput: d.Client, UseTextInputClient: true,
				Skills:          []session.SkillOption{{Name: "review", Path: "/skills/review/SKILL.md"}},
				OnOpenReference: func(path string) { opened = path }}
			tt := ui.NewTester(func(c *ui.Context) {
				paintMentionChips(c, m, ParseDraftMentions(d.Text), ui.RGB(0, 0, 0), ui.RGB(0, 0, 0), true, nil, nil)
			}, 800, 100)
			if err := tt.Click(ChipLabel(ParseDraftMentions(d.Text)[0])); err != nil {
				t.Fatal(err)
			}
			want := map[string]string{"@src/main.go": "src/main.go", `@"my file.txt"`: "my file.txt", "/review": "/skills/review/SKILL.md"}[token]
			if opened != want || d.Text != before || d.Client.Undo() {
				t.Fatalf("open=%q want=%q draft=%q", opened, want, d.Text)
			}
		})
	}
}

func TestComposerReferenceExplicitRemovalRemainsUndoable(t *testing.T) {
	d := NewDraftState("read @src/main.go now")
	before := d.Text
	opened := false
	m := Model{Draft: &d.Text, State: d, TextInput: d.Client, UseTextInputClient: true,
		OnOpenReference: func(string) { opened = true }}
	tt := ui.NewTester(func(c *ui.Context) {
		paintMentionChips(c, m, ParseDraftMentions(d.Text), ui.RGB(0, 0, 0), ui.RGB(0, 0, 0), true, nil, nil)
	}, 800, 100)
	if err := tt.Click("Remove reference @src/main.go"); err != nil {
		t.Fatal(err)
	}
	if opened || d.Text != "read now" || !d.Client.Undo() || d.Text != before {
		t.Fatalf("remove changed routing/history: open=%v draft=%q", opened, d.Text)
	}
}

func TestComposerReferenceUnavailableAndSelectedRemainUnchanged(t *testing.T) {
	for _, token := range []string{"@folder/", "@\"unfinished", "/unknown", "@[chat](dsh-session:ImEi)"} {
		d := NewDraftState(token)
		opened := false
		m := Model{Draft: &d.Text, TextInput: d.Client, OnOpenReference: func(string) { opened = true }}
		if activateDraftReference(m, DraftMention{Kind: DraftMentionAt, Text: token, End: len(token)}) || opened || d.Text != token {
			t.Fatalf("unsupported reference opened: %q", token)
		}
	}
	d := NewDraftState("@src/main.go")
	m := Model{Draft: &d.Text, TextInput: d.Client, OnOpenReference: func(string) { t.Fatal("selected reference opened") }}
	d.Client.SetSelectionUTF16(0, 4)
	if activateDraftReference(m, ParseDraftMentions(d.Text)[0]) {
		t.Fatal("noncollapsed selection consumed")
	}
	d.Client.SetSelectionUTF16(0, 0)
	m.OnOpenReference = nil
	if activateDraftReference(m, ParseDraftMentions(d.Text)[0]) || d.Text != "@src/main.go" {
		t.Fatal("missing opener mutated the draft")
	}
}

func TestComposerReferenceRightClickStillEdits(t *testing.T) {
	d := NewDraftState("@src/main.go")
	ed := &ChipEdit{}
	m := Model{Draft: &d.Text, TextInput: d.Client, ChipEdit: ed, OnOpenReference: func(string) { t.Fatal("right click opened reference") }}
	tt := ui.NewTester(func(c *ui.Context) {
		paintMentionChips(c, m, ParseDraftMentions(d.Text), ui.RGB(0, 0, 0), ui.RGB(0, 0, 0), true, nil, nil)
	}, 800, 100)
	if err := tt.RightClick("file @src/main.go"); err != nil {
		t.Fatal(err)
	}
	if !ed.Open || d.Text != "@src/main.go" {
		t.Fatalf("edit=%+v draft=%q", ed, d.Text)
	}
}
