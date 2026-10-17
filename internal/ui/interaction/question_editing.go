package interaction

import (
	"slices"
	"strconv"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/interaction"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

type questionEditorFlow struct{ focusID, feedback string }
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

func questionsBody(c *ui.Context, p interaction.Pending, card interaction.QuestionCardView, store *interaction.Store, fg, muted ui.Color) bool {
	area := ui.Column(c.Key("question-answer:" + p.AgentID + ":" + p.EventID)).FillWidth().Disabled(p.Sending)
	flow := ui.Local(area, "question-editor-flow", func() questionEditorFlow { return questionEditorFlow{} })
	area.Children(func() {
		for rowIndex, row := range card.Rows {
			ui.Column(c.Key("question-row:" + strconv.Itoa(rowIndex) + ":" + row.ID)).FillWidth().Gap(4).Children(func() {
				label := row.Question
				if label == "" {
					label = row.Header
				}
				if label != "" {
					ui.Text(c, label).FontSize(theme.InterfaceSize(12)).TextColor(fg)
				}
				if row.Detail != "" {
					ui.Text(c, row.Detail).FontSize(theme.InterfaceSize(12)).TextColor(muted)
				}
				ui.Row(c).Gap(8).Children(func() {
					if row.Skipped {
						ui.Text(c, "已跳过 / Skipped").FontSize(theme.InterfaceSize(11)).TextColor(muted)
						if ui.Button(c, "撤销跳过 / Undo skip").Clicked() {
							store.SetQuestionSkipped(p.EventID, row.ID, false)
						}
					} else if ui.Button(c, "跳过此题 / Skip question").Clicked() {
						store.SetQuestionSkipped(p.EventID, row.ID, true)
					}
				})
				if row.MultiSelect || len(row.OptionLabels) <= 4 {
					for index, opt := range row.OptionLabels {
						checked := slices.Contains(row.Selected, opt)
						ui.Checkbox(c.Key("question-option:"+strconv.Itoa(index)+":"+opt), &checked, opt).OnChange(func() {
							if row.MultiSelect || checked {
								store.SetQuestionOptionChecked(p.EventID, row.ID, opt, checked)
								flow.feedback = ""
							}
						})
						if index < len(row.OptionDescriptions) && row.OptionDescriptions[index] != "" {
							ui.Text(c, row.OptionDescriptions[index]).FontSize(theme.InterfaceSize(11)).TextColor(muted)
						}
					}
				} else {
					selected := ""
					if len(row.Selected) > 0 {
						selected = row.Selected[0]
					}
					ui.Select(c.Key("question-select"), &selected, row.OptionLabels).OnChange(func() { store.SetQuestionSingleSelect(p.EventID, row.ID, selected); flow.feedback = "" })
				}
				// Bind a frame-local copy, never a pointer into the store. Even a queued
				// input from before Sending flips cannot mutate the retained response.
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
		}
		if flow.feedback != "" {
			message := "Answer this question before continuing."
			if flow.feedback == "incomplete" {
				message = "Complete or skip the remaining questions before submitting."
			}
			if i18n.Locale() == i18n.LocaleZH {
				message = "请先回答此题再继续。"
				if flow.feedback == "incomplete" {
					message = "请回答或跳过其余问题后提交。"
				}
			}
			ui.Text(c, message).FontSize(theme.InterfaceSize(11)).TextColor(muted)
		}
	})
	return area.FocusWithin()
}
