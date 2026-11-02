package composer

import (
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
)

func inlineTester(d *DraftState, width int, hero, dark bool, open func(string)) *ui.Tester {
	return ui.NewTester(func(c *ui.Context) {
		if dark {
			c.SetTheme(ui.DarkTheme())
		}
		View(c, Model{Draft: &d.Text, State: d, TextInput: d.Client, UseTextInputClient: true,
			Hero: hero, OnOpenReference: open, Skills: []session.SkillOption{{Name: "review", Path: "/skills/review/SKILL.md"}}})
	}, width, 700)
}

func requireInlineRange(t *testing.T, c *DraftClient, start, end int) {
	t.Helper()
	if got := c.Selection().Range; got != (ui.TextInputRange{Start: start, End: end}) {
		t.Fatalf("selection=%+v want=%d:%d", got, start, end)
	}
}

func TestInlineReferencesShareVisibleTextLayout(t *testing.T) {
	for _, hero := range []bool{false, true} {
		for _, dark := range []bool{false, true} {
			d := NewDraftState("Read @src/deep/path/report.go with /review then @[Team chat](dsh-session:ImEi) now")
			tt := inlineTester(d, 900, hero, dark, nil)
			input, _ := tt.Find("Composer draft")
			left, _, top := d.Client.ins()
			if len(d.Client.inlineChips) != 3 || strings.Count(d.Client.displayText, draftObject) != 3 {
				t.Fatalf("projection=%q chips=%+v", d.Client.displayText, d.Client.inlineChips)
			}
			if strings.Contains(d.Client.displayText, "src/deep") || strings.Contains(d.Client.displayText, "dsh-session:") {
				t.Fatal("raw reference remains in painted paragraph", d.Client.displayText)
			}
			first := d.Client.inlineChips[0]
			chip, found := tt.Find("file @src/deep/path/report.go")
			if !found {
				t.Fatal("inline chip absent")
			}
			closeMetric(t, chip.Y, input.Y+top+1, "chip shares first text line")
			prefix, _ := ui.ShapeText("Read ", d.Client.font, 0).Size()
			closeMetric(t, chip.X-input.X-left, prefix, "prose precedes chip in line")
			if chip.Y < input.Y || chip.Y+chip.H > input.Y+input.H || chip.X+chip.W > input.X+input.W {
				t.Fatalf("chip escaped draft: chip=%+v input=%+v", chip, input)
			}
			before, _, _ := d.Client.BoundsForRange(ui.TextInputRange{Start: first.reference.Start, End: first.reference.Start})
			after, _, _ := d.Client.BoundsForRange(ui.TextInputRange{Start: first.reference.End, End: first.reference.End})
			closeMetric(t, after.X-before.X, chip.W, "native caret spans visible chip width")
			closeMetric(t, before.Y, after.Y, "native chip edges share baseline")
			// Pixels, rather than just exported geometry: the chip's actual fill
			// must differ from the draft gutter on the same visible text line.
			image := tt.Image()
			inside := image.RGBAAt(int(chip.X+3), int(chip.Y+chip.H/2))
			outside := image.RGBAAt(int(input.X+1), int(chip.Y+chip.H/2))
			if inside == outside {
				t.Fatalf("inline chip has no visible material dark=%v hero=%v", dark, hero)
			}
		}
	}
}

func TestInlineReferenceWrapAndNativeHitTests(t *testing.T) {
	d := NewDraftState("Before @very/long/path/with/a/long/name.go after /review end")
	tt := inlineTester(d, 330, false, false, nil)
	for _, width := range []int{330, 800, 270} {
		tt.SetSize(width, 700)
		input, _ := tt.Find("Composer draft")
		for _, chip := range d.Client.inlineChips {
			b := d.Client.chipRect(chip)
			if b.X < -.1 || b.X+b.W > d.Client.wrapWidth+.1 {
				t.Fatalf("width%d chip breaks/clips: %+v wrap=%v", width, b, d.Client.wrapWidth)
			}
			left, _, top := d.Client.ins()
			for _, edge := range []struct {
				x    float32
				want int
			}{{b.X + .1, chip.reference.Start}, {b.X + b.W - .1, chip.reference.End}} {
				got, ok := d.Client.IndexForPoint(ui.Point{X: left + edge.x, Y: top + b.Y + b.H/2})
				if !ok || got != edge.want {
					t.Fatalf("width%d edge=%v maps=%d want=%d", width, edge.x, got, edge.want)
				}
			}
		}
		if input.H+1 < d.Client.visualHeight() {
			t.Fatal("inline wrap did not grow draft")
		}
	}
}

func TestInlineNativeKeysCrossDeleteAndUndoWholeObjects(t *testing.T) {
	for _, token := range []string{"@src/main.go", `@"my file.txt"`, "/review", "@[Team chat](dsh-session:ImEi)"} {
		t.Run(token, func(t *testing.T) {
			d := NewDraftState("A😀 " + token + " e\u0301Z")
			tt := inlineTester(d, 800, false, false, nil)
			ref := d.Client.references[0]
			if tt.NativeTextInputClient() == nil {
				t.Fatal("shipped input did not register native client")
			}
			d.Client.SetSelectionUTF16(ref.Start, ref.Start)
			tt.Key(0, ui.KeyRight)
			requireInlineRange(t, d.Client, ref.End, ref.End)
			tt.Key(0, ui.KeyLeft)
			requireInlineRange(t, d.Client, ref.Start, ref.Start)
			tt.Key(ui.Shift, ui.KeyRight)
			requireInlineRange(t, d.Client, ref.Start, ref.End)
			tt.Command("copy")
			if tt.Clipboard() != token {
				t.Fatalf("copy=%q want raw token %q", tt.Clipboard(), token)
			}
			tt.Key(0, ui.KeyRight)
			tt.Key(0, ui.KeyBackspace)
			if d.Text != "A😀  e\u0301Z" {
				t.Fatalf("backspace split object: %q", d.Text)
			}
			tt.Key(ui.Cmd, ui.KeyZ)
			if d.Text != "A😀 "+token+" e\u0301Z" || len(d.Client.references) != 1 {
				t.Fatalf("undo lost object: %q", d.Text)
			}
			requireInlineRange(t, d.Client, ref.End, ref.End)
			d.Client.SetSelectionUTF16(ref.Start, ref.Start)
			tt.Key(0, ui.KeyDelete)
			if strings.Contains(d.Text, token) {
				t.Fatal("Delete left object content")
			}
			tt.Command("undo")
			tt.Command("selectAll")
			tt.Command("copy")
			if tt.Clipboard() != d.Text {
				t.Fatal("whole copy uses display projection", tt.Clipboard())
			}
		})
	}
}

func TestInlineNativePartialRangesAndCompositionAreAtomic(t *testing.T) {
	d := NewDraftState("A😀 @src/main.go Z")
	original := d.Text
	tt := inlineTester(d, 700, true, false, nil)
	ref := d.Client.references[0]
	native := tt.NativeTextInputClient()
	if native == nil {
		t.Fatal("native text service unavailable")
	}
	s, actual := native.TextForRange(ui.TextInputRange{Start: ref.Start + 2, End: ref.End - 2})
	if s != ref.Text || actual != (ui.TextInputRange{Start: ref.Start, End: ref.End}) {
		t.Fatalf("partial native query=%q %+v", s, actual)
	}
	d.Client.SetSelectionUTF16(ref.Start+2, ref.End-2)
	requireInlineRange(t, d.Client, ref.Start, ref.End)
	native.SetMarkedText(nil, "zhong", ui.TextInputRange{Start: 5, End: 5})
	if d.Text != "A😀 zhong Z" {
		t.Fatal("composition split object", d.Text)
	}
	native.SetMarkedText(nil, "中文", ui.TextInputRange{Start: 2, End: 2})
	tt.Frame()
	if d.Client.navigate(ui.InputEvent{Key: ui.KeyLeft}) {
		t.Fatal("IME lost navigation ownership")
	}
	native.ReplaceText(nil, "中文")
	tt.Frame()
	if d.Text != "A😀 中文 Z" {
		t.Fatal("IME commit", d.Text)
	}
	tt.Command("undo")
	if d.Text != original || len(d.Client.references) != 1 {
		t.Fatal("one undo did not restore chip", d.Text)
	}
	if _, active := d.Client.MarkedRange(); active {
		t.Fatal("undo revived preedit")
	}
	if d.Client.Undo() {
		t.Fatal("candidate updates leaked into undo")
	}
	native.ReplaceText(&ui.TextInputRange{Start: ref.Start + 1, End: ref.End - 1}, "replacement")
	if d.Text != "A😀 replacement Z" {
		t.Fatalf("native partial replacement split reference: %q", d.Text)
	}
}

func TestInlineTypingKeepsTriggerEditableUntilCommitted(t *testing.T) {
	d := NewDraftState("")
	tt := inlineTester(d, 700, false, false, nil)
	tt.Type("@src/main.go")
	if len(d.Client.references) != 0 || d.Client.displayText != d.Text {
		t.Fatal("unfinished trigger became an object", d.Client.displayText)
	}
	tt.Key(0, ui.KeyLeft)
	if d.Client.Selection().Caret() != ui.UTF16Len(d.Text)-1 {
		t.Fatal("unfinished trigger is not editable")
	}
	tt.Key(0, ui.KeyEnd)
	tt.Type(" ")
	if len(d.Client.references) != 1 || d.Client.displayText != draftObject+" " {
		t.Fatal("separator did not commit inline chip", d.Client.displayText)
	}
	tt.Type("normal")
	if d.Text != "@src/main.go normal" {
		t.Fatal("prose after chip", d.Text)
	}
	raw := d.SaveDraft()
	restored, err := RestoreDraftChecked(raw)
	if err != nil || restored.Text != d.Text || len(restored.Client.references) != 1 {
		t.Fatal("draft persistence lost raw identity", string(raw), err)
	}
}

func TestInlinePointerSelectionActivationEditAndMove(t *testing.T) {
	d := NewDraftState("read @src/main.go now /review end")
	opened := ""
	ed := &ChipEdit{}
	m := Model{Draft: &d.Text, State: d, TextInput: d.Client, UseTextInputClient: true, ChipEdit: ed, OnOpenReference: func(path string) { opened = path }}
	tt := ui.NewTester(func(c *ui.Context) { View(c, m) }, 800, 700)
	if err := tt.Click("file @src/main.go"); err != nil {
		t.Fatal(err)
	}
	if opened != "src/main.go" || d.Text != "read @src/main.go now /review end" {
		t.Fatal("single activation mutated document", opened, d.Text)
	}
	if err := tt.RightClick("file @src/main.go"); err != nil {
		t.Fatal(err)
	}
	if !ed.Open {
		t.Fatal("inline right-click did not open editor")
	}
	ref := d.Client.references[0]
	requireInlineRange(t, d.Client, ref.Start, ref.End)
	closeChipEdit(m)
	// A drag that starts in prose selects the entire object when it crosses it.
	d.Client.SetSelectionUTF16(0, 0)
	tt.Frame()
	input, _ := tt.Find("Composer draft")
	start, _, _ := d.Client.BoundsForRange(ui.TextInputRange{Start: 1, End: 1})
	end, _, _ := d.Client.BoundsForRange(ui.TextInputRange{Start: ref.End + 1, End: ref.End + 1})
	tt.Press(input.X+start.X, input.Y+start.Y+start.H/2)
	tt.Move(input.X+end.X, input.Y+end.Y+end.H/2)
	tt.Release(input.X+end.X, input.Y+end.Y+end.H/2)
	requireInlineRange(t, d.Client, 1, ref.End+1)
	tt.Command("copy")
	if tt.Clipboard() != "ead @src/main.go " {
		t.Fatalf("drag copy=%q", tt.Clipboard())
	}
	// A drag starting on the chip moves that one object at a document caret.
	d.Client.SetSelectionUTF16(0, 0)
	tt.Frame()
	chip, _ := tt.Find("file @src/main.go")
	last, _, _ := d.Client.BoundsForRange(ui.TextInputRange{Start: ui.UTF16Len(d.Text), End: ui.UTF16Len(d.Text)})
	original := d.Text
	tt.Press(chip.X+chip.W/2, chip.Y+chip.H/2)
	tt.Move(input.X+last.X, input.Y+last.Y+last.H/2)
	tt.Release(input.X+last.X, input.Y+last.Y+last.H/2)
	if d.Text != "read  now /review end @src/main.go" || len(d.Client.references) != 2 {
		t.Fatalf("chip move=%q refs=%+v", d.Text, d.Client.references)
	}
	restored, err := RestoreDraftChecked(d.SaveDraft())
	if err != nil || restored.Text != d.Text || len(restored.Client.references) != 2 {
		t.Fatal("moved chip persistence", err, d.Text)
	}
	tt.Command("undo")
	if d.Text != original || len(d.Client.references) != 2 {
		t.Fatal("move was not one undo", d.Text)
	}
}

func TestInlineRemovalEditAndChildEditorUseNativeDocument(t *testing.T) {
	for _, child := range []bool{false, true} {
		d := NewDraftState("read @src/main.go now")
		ed := &ChipEdit{}
		m := Model{Draft: &d.Text, State: d, TextInput: d.Client, UseTextInputClient: true, ChipEdit: ed}
		tt := ui.NewTester(func(c *ui.Context) {
			if child {
				ChildEditor(c, ChildEditorModel{Draft: d, Chip: ed})
			} else {
				View(c, m)
			}
		}, 600, 700)
		ref := d.Client.references[0]
		original := d.Text
		if err := tt.Click("Remove reference @src/main.go"); err != nil {
			t.Fatal(err)
		}
		if d.Text != "read now" {
			t.Fatal("explicit inline removal", d.Text)
		}
		tt.Command("undo")
		if d.Text != original {
			t.Fatal("explicit removal bypassed undo", d.Text)
		}
		mention, _ := d.Client.mention(ref)
		if !replaceDraftReference(m, mention, `@"my file.txt"`) {
			t.Fatal("inline editor did not replace native reference")
		}
		tt.Frame()
		if d.Text != `read @"my file.txt" now` || len(d.Client.references) != 1 {
			t.Fatal("inline editor split token", d.Text)
		}
		tt.Command("undo")
		if d.Text != original {
			t.Fatal("inline editor bypassed undo", d.Text)
		}
	}
}

func TestInlinePersistenceValidatesOccurrencesAndKeepsEditableText(t *testing.T) {
	d := NewDraftState("")
	tt := inlineTester(d, 600, false, false, nil)
	tt.Type("@still-editing")
	restored, err := RestoreDraftChecked(d.SaveDraft())
	if err != nil || restored.Text != d.Text || len(restored.Client.references) != 0 {
		t.Fatal("uncommitted trigger changed on restore", err, string(d.SaveDraft()))
	}
	legacy, err := RestoreDraftChecked([]byte(`{"Text":"@old/file.go "}`))
	if err != nil || len(legacy.Client.references) != 1 {
		t.Fatal("legacy plain-text draft no longer imports references", err)
	}
	for _, raw := range []string{
		`{"Text":"@a.go","References":[{"Start":1,"End":5,"Kind":0,"Text":"@a.go"}]}`,
		`{"Text":"@a.go","References":[{"Start":0,"End":5,"Kind":1,"Text":"@a.go"}]}`,
		`{"Text":"@a.go","References":[{"Start":0,"End":99,"Kind":0,"Text":"@a.go"}]}`,
		`{"Text":"@a.go","References":[{"Start":0,"End":5,"Kind":0,"Text":"@a.go"},{"Start":0,"End":5,"Kind":0,"Text":"@a.go"}]}`,
	} {
		if _, err := RestoreDraftChecked([]byte(raw)); err == nil {
			t.Fatal("invalid saved occurrence accepted", raw)
		}
	}
}

func TestInlineIMECommandsAndMiddleTypingKeepNativeOwner(t *testing.T) {
	d := NewDraftState("suffix")
	tt := inlineTester(d, 600, false, false, nil)
	d.Client.SetSelectionUTF16(0, 0)
	tt.Type("@new")
	if len(d.Client.references) != 0 {
		t.Fatal("middle typing froze a partially typed trigger")
	}
	tt.Compose("中文", 2)
	before, selection := d.Text, d.Client.Selection()
	for _, command := range []string{"copy", "cut", "delete", "selectAll", "undo"} {
		tt.Command(command)
		if d.Text != before || d.Client.Selection() != selection {
			t.Fatalf("IME %s mutated preedit", command)
		}
	}
	tt.Key(0, ui.KeyBackspace)
	if d.Text != before {
		t.Fatal("Backspace stole preedit")
	}
	tt.Type("中文")
	if _, active := d.Client.MarkedRange(); active {
		t.Fatal("composition failed to commit")
	}
}

func TestInlineDragStartsItsOwnUndoAfterTyping(t *testing.T) {
	d := NewDraftState("@a.go x")
	tt := inlineTester(d, 600, false, false, nil)
	tt.Type(" ")
	before := d.Text
	chip, _ := tt.Find("file @a.go")
	input, _ := tt.Find("Composer draft")
	last, _, _ := d.Client.BoundsForRange(ui.TextInputRange{Start: ui.UTF16Len(d.Text), End: ui.UTF16Len(d.Text)})
	tt.Press(chip.X+chip.W/2, chip.Y+chip.H/2)
	tt.Move(input.X+last.X, input.Y+last.Y+last.H/2)
	tt.Release(input.X+last.X, input.Y+last.Y+last.H/2)
	moved := d.Text
	tt.Type("!")
	tt.Command("undo")
	if d.Text != moved {
		t.Fatal("typing after drag merged into move undo", d.Text, moved)
	}
	tt.Command("undo")
	if d.Text != before {
		t.Fatal("drag undo did not restore pre-move text", d.Text, before)
	}
}

func TestInlineUnmarkBeforeExplicitCommitKeepsTriggerPrefix(t *testing.T) {
	for _, unmarkFirst := range []bool{false, true} {
		d := NewDraftState("")
		tt := inlineTester(d, 600, false, false, nil)
		tt.Type("@")
		native := tt.NativeTextInputClient()
		native.SetMarkedText(nil, "候选", ui.TextInputRange{Start: 2, End: 2})
		marked, _ := native.MarkedRange()
		if unmarkFirst {
			native.UnmarkText()
			native.ReplaceText(&marked, "最终")
		} else {
			native.ReplaceText(nil, "最终")
		}
		if d.Text != "@最终" {
			t.Fatalf("unmark=%v lost trigger prefix: %q", unmarkFirst, d.Text)
		}
		tt.Frame()
		tt.Command("undo")
		if d.Text != "@" || len(d.Client.references) != 0 {
			t.Fatalf("unmark=%v undo did not restore unfinished trigger: %q", unmarkFirst, d.Text)
		}
	}
}

func TestInlineRemoveKeepsOtherOccurrencesAndUnfinishedTrigger(t *testing.T) {
	d := NewDraftState("@a.go ")
	tt := inlineTester(d, 600, false, false, nil)
	tt.Type("@editing")
	if len(d.Client.references) != 1 {
		t.Fatal("fixture trigger already frozen")
	}
	if err := tt.Click("Remove reference @a.go"); err != nil {
		t.Fatal(err)
	}
	if d.Text != "@editing" || len(d.Client.references) != 0 {
		t.Fatalf("remove froze unrelated trigger: %q %+v", d.Text, d.Client.references)
	}
	tt.Command("undo")
	if d.Text != "@a.go @editing" || len(d.Client.references) != 1 {
		t.Fatal("remove undo lost occurrence state", d.Text, d.Client.references)
	}
}

func TestInlineSafeParsingKeepsMalformedTextVisible(t *testing.T) {
	for _, text := range []string{`@"unfinished`, "@[chat](https://example.com)", "@bad\x00file", "/bad\x01name", `@""`} {
		d := NewDraftState(text)
		if len(d.Client.references) != 0 || d.Client.displayText != text {
			t.Fatalf("malformed input hidden as chip: %q -> %q", text, d.Client.displayText)
		}
	}
	d := NewDraftState("one\u2003@file.go\r\n/review")
	if len(d.Client.references) != 2 {
		t.Fatal("Unicode whitespace and CRLF lost reference boundaries", d.Client.references)
	}
	before := "@same.go @same.go"
	stale := DraftMention{Text: "@same.go", Start: 1, End: 9}
	if RemoveDraftMention(before, stale) != before {
		t.Fatal("stale occurrence removed a different duplicate")
	}
}
