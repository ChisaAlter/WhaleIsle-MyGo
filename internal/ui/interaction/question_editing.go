package interaction

import (
	"slices"
	"strconv"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/interaction"
)

type questionEditorFlow struct {
	focusID, feedback  string
	minimized, summary bool
}
type questionEditorKeys struct{ composing, compositionEnded bool }

// Native TextArea routes marked-text events directly to its text adapter, not
// HandleInput. Observe its state during builds; an Element cannot be queried
// from the later input callback because build-pass handles have expired.
func (k *questionEditorKeys) observeComposition(composing bool) {
	if k.composing && !composing {
		k.compositionEnded = true
	}
	k.composing = composing
}

// This hook is attached only to the custom answer field. Shift+Enter stays a
// native newline; input-method composition and its committing Enter cannot
// advance or submit the request. It never installs a page/window shortcut.
func (k *questionEditorKeys) handle(ev ui.InputEvent, composing bool, advance func()) bool {
	switch ev.Kind {
	case ui.InputCompose:
		k.composing = ev.Text != ""
		k.compositionEnded = !k.composing
		return false
	case ui.InputText:
		if k.composing {
			k.composing = false
			k.compositionEnded = true
		}
		return false
	case ui.InputKeyUp:
		k.compositionEnded = false
		return false
	}
	if ev.Kind != ui.InputKeyDown || ev.Key != ui.KeyEnter {
		return false
	}
	if composing || k.composing || k.compositionEnded {
		return true
	}
	if ev.Mods&ui.Shift != 0 {
		return false
	}
	if !ev.Repeat {
		advance()
	}
	return true
}

func questionAnswerBody(c *ui.Context, p interaction.Pending, row interaction.QuestionRowView, rowIndex int, store CardStore, flow *questionEditorFlow, fg, muted ui.Color) bool {
	_, height := c.Size()
	area := ui.Scroll(c.Key("question-answer:" + p.AgentID + ":" + p.EventID + ":" + strconv.Itoa(rowIndex))).
		Label("Question content").FillWidth().MinWidth(0).MaxHeight(min(520, height/2)).Disabled(p.Sending)
	area.Children(func() {
		label := row.Question
		if label == "" {
			label = row.Header
		}
		questionDetail(c, row.Detail, fg, muted)
		for index, opt := range row.OptionLabels {
			checked := slices.Contains(row.Selected, opt)
			choice := ui.Checkbox(c.Key("question-option:"+strconv.Itoa(index)), &checked, "").Label(opt).FillWidth().MinWidth(0).OnChange(func() {
				if row.MultiSelect || checked {
					store.SetQuestionOptionChecked(p.EventID, row.ID, opt, checked)
					if !row.MultiSelect {
						store.AdvanceQuestionChoice(p.AgentID, p.EventID, row.ID)
					}
					flow.feedback = ""
				}
			})
			// The styled checkbox's implicit label has an automatic min-content
			// width. A shrinkable label also wraps long unbroken identifiers.
			choice.Children(func() { ui.Text(c, opt).MinWidth(0).Grow(1).TextColor(fg) })
			choice.HandleInput(func(ev ui.InputEvent) bool {
				if ev.Kind != ui.InputKeyDown || ev.Key != ui.KeyEnter {
					return false
				}
				if !ev.Repeat {
					flow.focusID, flow.feedback = store.SubmitQuestionBatch(p.AgentID, p.EventID)
				}
				return true
			})
			if index < len(row.OptionDescriptions) && row.OptionDescriptions[index] != "" {
				questionText(c, row.OptionDescriptions[index], 11, muted)
			}
		}
		// Frame-local binding prevents pre-Sending events from mutating the draft.
		custom := row.Custom
		input := ui.TextArea(c.Key("question-custom"), &custom).FillWidth().Lines(2, 5).Label("Question answer: " + label)
		keys := ui.Local(input, "question-enter", func() questionEditorKeys { return questionEditorKeys{} })
		input.OnChange(func() { store.SetQuestionCustom(p.EventID, row.ID, custom); flow.feedback = "" })
		composing := input.Composing()
		keys.observeComposition(composing)
		input.HandleInput(func(ev ui.InputEvent) bool {
			return keys.handle(ev, composing, func() { flow.focusID, flow.feedback = store.ContinueQuestion(p.AgentID, p.EventID, row.ID) })
		})
		if flow.focusID == row.ID && !p.Sending {
			input.Focus()
			flow.focusID = ""
		}
	})
	return area.FocusWithin()
}
